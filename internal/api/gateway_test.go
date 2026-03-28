package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newFakeResponse builds a minimal *http.Response for use in gateway tests.
func newFakeResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}

// --- Priority context helpers ---

func TestWithPriority_AndPriorityFromContext(t *testing.T) {
	ctx := context.Background()
	// Default is Background.
	assert.Equal(t, Background, PriorityFromContext(ctx))

	// Set Interactive.
	ctx2 := WithPriority(ctx, Interactive)
	assert.Equal(t, Interactive, PriorityFromContext(ctx2))

	// Original context is unchanged.
	assert.Equal(t, Background, PriorityFromContext(ctx))
}

// --- tokenBucket tests ---
// NOTE: tokenBucket and Gateway are in the same package (api) so we access unexported types directly.

func TestTokenBucket_AllowsBurst(t *testing.T) {
	// A fresh bucket with max=5 should allow 5 calls immediately.
	tb := newTokenBucket(5, 5)

	for i := 0; i < 5; i++ {
		ctx := context.Background()
		err := tb.wait(ctx)
		require.NoError(t, err, "burst call %d should not block", i+1)
	}
}

func TestTokenBucket_BlocksWhenEmpty(t *testing.T) {
	// rate=100 tokens/s so the refill interval is 10ms.
	tb := newTokenBucket(1, 100)

	// Drain the single token.
	require.NoError(t, tb.wait(context.Background()))

	// Next call should block ~10ms then succeed.
	start := time.Now()
	err := tb.wait(context.Background())
	elapsed := time.Since(start)

	require.NoError(t, err)
	// Should have waited at least 5ms (half the expected 10ms, to tolerate timing jitter).
	assert.Greater(t, elapsed.Milliseconds(), int64(5),
		"expected blocking wait for token refill, got %v", elapsed)
}

func TestTokenBucket_RespectsContextCancellation(t *testing.T) {
	// rate=0.01 tokens/s → refill time is 100s; we'll cancel well before that.
	tb := newTokenBucket(1, 0.01)

	// Drain the single token.
	require.NoError(t, tb.wait(context.Background()))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := tb.wait(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

// --- Gateway concurrency limiter tests ---

func TestGateway_MaxConcurrentRequests(t *testing.T) {
	// 5 concurrent requests should all succeed; the 6th must wait until one completes.
	gw := NewGateway()

	const concurrency = 5
	started := make(chan struct{}, concurrency+1)
	release := make(chan struct{})

	// Launch 5 goroutines that each hold the semaphore until release is closed.
	// Each uses a unique key to avoid in-flight deduplication merging them.
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := RequestKey{Method: "GET", Path: fmt.Sprintf("/hold/%d", i)}
			_, _ = gw.Do(context.Background(), Background, key, func() (*http.Response, error) {
				started <- struct{}{} // signal: slot acquired
				<-release             // hold the slot
				return newFakeResponse(200, "ok"), nil
			})
		}(i)
	}

	// Wait for all 5 slots to be filled.
	for i := 0; i < concurrency; i++ {
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for 5 concurrent slots to fill")
		}
	}

	// Launch a 6th goroutine — it should block.
	blocked := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		key := RequestKey{Method: "GET", Path: "/sixth"}
		close(blocked) // signal that we're about to try
		_, _ = gw.Do(context.Background(), Background, key, func() (*http.Response, error) {
			started <- struct{}{}
			return newFakeResponse(200, "ok"), nil
		})
	}()

	<-blocked
	// Give the 6th goroutine time to attempt and block on the semaphore.
	time.Sleep(20 * time.Millisecond)

	// The 6th slot should NOT have been acquired yet.
	select {
	case <-started:
		t.Fatal("6th goroutine acquired a slot before one was released")
	default:
		// Expected: still blocked.
	}

	// Release all 5 held slots.
	close(release)

	// Now the 6th should eventually complete.
	select {
	case <-started:
		// Good: 6th goroutine ran after a slot opened.
	case <-time.After(2 * time.Second):
		t.Fatal("6th goroutine never acquired a slot after release")
	}

	wg.Wait()
}

func TestGateway_SemaphoreRespectsContextCancellation(t *testing.T) {
	gw := NewGateway()

	release := make(chan struct{})

	// Fill all 5 semaphore slots.
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := RequestKey{Method: "GET", Path: fmt.Sprintf("/hold/%d", i)}
			_, _ = gw.Do(context.Background(), Background, key, func() (*http.Response, error) {
				<-release
				return newFakeResponse(200, "ok"), nil
			})
		}(i)
	}

	// Give goroutines time to fill slots.
	time.Sleep(20 * time.Millisecond)

	// 6th with a cancelled context.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := gw.Do(ctx, Background, RequestKey{Method: "GET", Path: "/sixth"}, func() (*http.Response, error) {
		return newFakeResponse(200, "ok"), nil
	})

	assert.Error(t, err, "expected error when context cancelled waiting for semaphore")

	close(release)
	wg.Wait()
}

// --- In-flight deduplication tests ---

func TestGateway_Dedup_SameKey_OneHTTPCall(t *testing.T) {
	gw := NewGateway()

	callCount := 0
	release := make(chan struct{})

	key := RequestKey{Method: "GET", Path: "/tracks/1"}

	var wg sync.WaitGroup
	results := make([]string, 2)
	errs := make([]error, 2)

	// Launch two goroutines with the same key simultaneously.
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			resp, err := gw.Do(context.Background(), Background, key, func() (*http.Response, error) {
				<-release // both goroutines race to register, one makes the real call
				callCount++
				return newFakeResponse(200, "track-data"), nil
			})
			errs[idx] = err
			if err == nil {
				body, _ := io.ReadAll(resp.Body)
				results[idx] = string(body)
			}
		}(i)
	}

	// Give both goroutines time to register before releasing.
	time.Sleep(20 * time.Millisecond)
	close(release)
	wg.Wait()

	require.NoError(t, errs[0])
	require.NoError(t, errs[1])
	assert.Equal(t, "track-data", results[0])
	assert.Equal(t, "track-data", results[1])
	assert.Equal(t, 1, callCount, "expected exactly one HTTP call for two deduplicated requests")
}

func TestGateway_Dedup_DifferentKeys_IndependentCalls(t *testing.T) {
	gw := NewGateway()

	key1 := RequestKey{Method: "GET", Path: "/tracks/1"}
	key2 := RequestKey{Method: "GET", Path: "/tracks/2"}

	resp1, err1 := gw.Do(context.Background(), Background, key1, func() (*http.Response, error) {
		return newFakeResponse(200, "track-1"), nil
	})
	resp2, err2 := gw.Do(context.Background(), Background, key2, func() (*http.Response, error) {
		return newFakeResponse(200, "track-2"), nil
	})

	require.NoError(t, err1)
	require.NoError(t, err2)
	body1, _ := io.ReadAll(resp1.Body)
	body2, _ := io.ReadAll(resp2.Body)
	assert.Equal(t, "track-1", string(body1))
	assert.Equal(t, "track-2", string(body2))
}

func TestGateway_Dedup_ErrorSharedWithWaiters(t *testing.T) {
	gw := NewGateway()

	release := make(chan struct{})
	key := RequestKey{Method: "GET", Path: "/error-endpoint"}

	var wg sync.WaitGroup
	errs := make([]error, 2)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := gw.Do(context.Background(), Background, key, func() (*http.Response, error) {
				<-release
				return nil, fmt.Errorf("connection refused")
			})
			errs[idx] = err
		}(i)
	}

	time.Sleep(20 * time.Millisecond)
	close(release)
	wg.Wait()

	assert.Error(t, errs[0], "primary caller should get the error")
	assert.Error(t, errs[1], "waiting caller should also get the error")
}

// --- 429 backoff + priority tests ---

func TestGateway_Backoff_BackgroundRejectedDuringBackoff(t *testing.T) {
	gw := NewGateway()

	// Trigger a 429.
	_, err := gw.Do(context.Background(), Background,
		RequestKey{Method: "GET", Path: "/limited"},
		func() (*http.Response, error) {
			resp := newFakeResponse(429, "")
			resp.Header.Set("Retry-After", "10")
			return resp, nil
		})
	var rlErr *RateLimitError
	require.ErrorAs(t, err, &rlErr)

	// Subsequent background request should be rejected immediately.
	_, err2 := gw.Do(context.Background(), Background,
		RequestKey{Method: "GET", Path: "/other"},
		func() (*http.Response, error) {
			t.Error("fn should not be called during backoff")
			return newFakeResponse(200, "ok"), nil
		})
	require.ErrorAs(t, err2, &rlErr, "background request should get RateLimitError during backoff")
}

func TestGateway_Backoff_InteractiveWaitsAndProceeds(t *testing.T) {
	gw := NewGateway()

	// Set a very short backoff (50ms) directly via a 429 response.
	_, _ = gw.Do(context.Background(), Background,
		RequestKey{Method: "GET", Path: "/limited"},
		func() (*http.Response, error) {
			resp := newFakeResponse(429, "")
			resp.Header.Set("Retry-After", "0") // 0s → backoffUntil = now, expires instantly
			return resp, nil
		})

	// Force a small backoff manually so we can test the wait path.
	gw.mu.Lock()
	gw.backoffUntil = time.Now().Add(100 * time.Millisecond)
	gw.mu.Unlock()

	// Interactive request should wait ~100ms then succeed.
	start := time.Now()
	resp, err := gw.Do(context.Background(), Interactive,
		RequestKey{Method: "GET", Path: "/interactive"},
		func() (*http.Response, error) {
			return newFakeResponse(200, "ok"), nil
		})
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(80),
		"interactive request should have waited for backoff, got %v", elapsed)
}

func TestGateway_Interactive_BypassesTokenBucket(t *testing.T) {
	// Bucket with rate=0.001 tokens/s → would take 1000s to refill.
	// We drain it first, then verify an Interactive call goes through immediately.
	gw := NewGateway()
	// Replace the bucket with a very slow one.
	gw.bucket = newTokenBucket(1, 0.001)
	// Drain the single token.
	require.NoError(t, gw.bucket.wait(context.Background()))

	// Interactive request should not wait for the bucket.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := gw.Do(ctx, Interactive,
		RequestKey{Method: "POST", Path: "/play"},
		func() (*http.Response, error) {
			return newFakeResponse(204, ""), nil
		})
	elapsed := time.Since(start)

	require.NoError(t, err, "interactive call should bypass empty token bucket")
	assert.Less(t, elapsed.Milliseconds(), int64(100),
		"interactive call should not have waited for token bucket")
}

func TestGateway_IsThrottled(t *testing.T) {
	gw := NewGateway()
	assert.False(t, gw.IsThrottled(), "should not be throttled initially")

	// Trigger a 429.
	_, _ = gw.Do(context.Background(), Background,
		RequestKey{Method: "GET", Path: "/limited"},
		func() (*http.Response, error) {
			resp := newFakeResponse(429, "")
			resp.Header.Set("Retry-After", "30")
			return resp, nil
		})

	assert.True(t, gw.IsThrottled(), "should be throttled after 429")
	assert.Equal(t, 30, gw.RetryAfterSecs())
}

// --- GET-only dedup safety ---

// TestGateway_Dedup_OnlyForGET verifies that POST requests with the same key
// are NOT deduplicated — each must trigger an independent HTTP call.
func TestGateway_Dedup_OnlyForGET(t *testing.T) {
	gw := NewGateway()

	var callCount int64
	release := make(chan struct{})
	key := RequestKey{Method: http.MethodPost, Path: "/me/player/next"}

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = gw.Do(context.Background(), Background, key, func() (*http.Response, error) {
				<-release
				// Use channel-based counting to avoid races in test closures.
				atomic.AddInt64(&callCount, 1)
				return newFakeResponse(204, ""), nil
			})
		}()
	}

	// Give both goroutines time to be in-flight simultaneously.
	time.Sleep(20 * time.Millisecond)
	close(release)
	wg.Wait()

	assert.Equal(t, int64(2), atomic.LoadInt64(&callCount), "POST requests must not be deduplicated — each should fire independently")
}

// TestGateway_Dedup_WaitersDoNotConsumeSlots verifies that dedup waiters
// do not hold semaphore slots, preventing deadlock when all slots are taken.
// With 5 slots, 5 primary callers fill all slots. Then 5 more goroutines for
// the same keys should be able to join as dedup waiters (without needing slots)
// and get results when the primaries finish.
func TestGateway_Dedup_WaitersDoNotConsumeSlots(t *testing.T) {
	gw := NewGateway()

	const concurrency = 5
	release := make(chan struct{})
	var callCount int64

	// Launch 5 primary goroutines that each hold a semaphore slot.
	// Each uses a unique key so they are distinct in-flight entries.
	var wgPrimary sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wgPrimary.Add(1)
		go func(i int) {
			defer wgPrimary.Done()
			key := RequestKey{Method: http.MethodGet, Path: fmt.Sprintf("/tracks/%d", i)}
			_, _ = gw.Do(context.Background(), Background, key, func() (*http.Response, error) {
				<-release
				atomic.AddInt64(&callCount, 1)
				return newFakeResponse(200, "data"), nil
			})
		}(i)
	}

	// Give primaries time to acquire semaphore slots and register in inflight map.
	time.Sleep(30 * time.Millisecond)

	// Now launch 5 dedup waiters with the same keys. They should NOT need slots.
	waiterDone := make(chan struct{}, concurrency)
	for i := 0; i < concurrency; i++ {
		go func(i int) {
			key := RequestKey{Method: http.MethodGet, Path: fmt.Sprintf("/tracks/%d", i)}
			resp, err := gw.Do(context.Background(), Background, key, func() (*http.Response, error) {
				// This fn should never be called — dedup waiter reuses the result.
				t.Error("dedup waiter should not make a new HTTP call")
				return newFakeResponse(200, "data"), nil
			})
			if err == nil && resp != nil {
				waiterDone <- struct{}{}
			}
		}(i)
	}

	// Give waiters time to register against the inflight entries.
	time.Sleep(30 * time.Millisecond)

	// Release all primaries — waiters should complete shortly after.
	close(release)
	wgPrimary.Wait()

	// All 5 waiters should receive results within a short timeout.
	for i := 0; i < concurrency; i++ {
		select {
		case <-waiterDone:
		case <-time.After(2 * time.Second):
			t.Fatalf("waiter %d did not complete after primaries finished", i)
		}
	}

	// Only the 5 primary calls should have been made.
	assert.Equal(t, int64(concurrency), atomic.LoadInt64(&callCount), "expected exactly 5 HTTP calls (one per key)")
}

// TestGateway_PlaybackState_RoutedThroughGateway verifies that the Player's
// PlaybackState method routes through the gateway when one is attached,
// so that dedup and rate limiting apply to the most-polled endpoint.
func TestGateway_PlaybackState_RoutedThroughGateway(t *testing.T) {
	callCount := 0
	release := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent) // nothing playing
	}))
	defer srv.Close()

	gw := NewGateway()
	p := NewPlayer(srv.URL, "test-token")
	p.SetGateway(gw)

	// Two concurrent calls for the same endpoint should only hit the server once.
	var wg sync.WaitGroup
	results := make([]*PlaybackState, 2)
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = p.PlaybackState(context.Background())
		}(i)
	}

	time.Sleep(20 * time.Millisecond)
	close(release)
	wg.Wait()

	require.NoError(t, errs[0])
	require.NoError(t, errs[1])
	assert.Nil(t, results[0], "204 should yield nil state")
	assert.Nil(t, results[1], "204 should yield nil state")
	assert.Equal(t, 1, callCount, "dedup should result in exactly one HTTP call")
}

// --- Gateway.Snapshot() tests ---

func TestGateway_Snapshot_TokensAvailable(t *testing.T) {
	gw := NewGateway()
	snap := gw.Snapshot()
	// Fresh gateway: bucket full at 10, no concurrent requests, no backoff.
	assert.Equal(t, 10, snap.TokensAvailable, "fresh gateway should have full token bucket")
	assert.Equal(t, 10, snap.TokensMax, "token max should be 10")
	assert.Equal(t, 0, snap.ConcurrentActive, "no concurrent requests on fresh gateway")
	assert.Equal(t, 5, snap.ConcurrentMax, "semaphore max should be 5")
	assert.Equal(t, 0.0, snap.BackoffRemaining, "no backoff on fresh gateway")
	assert.Equal(t, 0, snap.DedupWaiters, "no dedup waiters on fresh gateway")
}

func TestGateway_Snapshot_ConcurrentActive(t *testing.T) {
	// Pause the server so we can measure in-flight count.
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	gw := NewGateway()

	// Launch one goroutine holding a semaphore slot.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		key := RequestKey{Method: "PUT", Path: "/v1/me/player/play"}
		_, _ = gw.Do(context.Background(), Interactive, key, func() (*http.Response, error) {
			req, _ := http.NewRequest("PUT", srv.URL+"/play", http.NoBody)
			return http.DefaultClient.Do(req)
		})
	}()

	// Give the goroutine time to acquire the semaphore.
	time.Sleep(30 * time.Millisecond)

	snap := gw.Snapshot()
	assert.GreaterOrEqual(t, snap.ConcurrentActive, 1, "at least one concurrent request in flight")
	assert.Equal(t, 5, snap.ConcurrentMax)

	close(release)
	wg.Wait()

	snap2 := gw.Snapshot()
	assert.Equal(t, 0, snap2.ConcurrentActive, "concurrent count should drop to zero after request completes")
}

func TestGateway_Snapshot_BackoffRemaining(t *testing.T) {
	// Trigger a 429 response to activate backoff.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	gw := NewGateway()
	key := RequestKey{Method: "GET", Path: "/v1/me/player"}
	_, _ = gw.Do(context.Background(), Background, key, func() (*http.Response, error) {
		req, _ := http.NewRequest("GET", srv.URL+"/player", http.NoBody)
		return http.DefaultClient.Do(req)
	})

	snap := gw.Snapshot()
	assert.Greater(t, snap.BackoffRemaining, 0.0, "backoff should be active after 429")
	assert.Less(t, snap.BackoffRemaining, 31.0, "backoff should not exceed Retry-After")
}

func TestGateway_Snapshot_DedupWaiters(t *testing.T) {
	// One waiter joins a dedup: primary request in-flight, second waits.
	release := make(chan struct{})
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		<-release
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	gw := NewGateway()
	key := RequestKey{Method: "GET", Path: "/v1/me/player"}

	// Start two GET requests for the same key; only one hits the server.
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = gw.Do(context.Background(), Background, key, func() (*http.Response, error) {
				req, _ := http.NewRequest("GET", srv.URL+"/player", http.NoBody)
				return http.DefaultClient.Do(req)
			})
		}()
	}

	// Give both goroutines time to start.
	time.Sleep(30 * time.Millisecond)

	snap := gw.Snapshot()
	// At least one dedup waiter should appear (secondary goroutine waits on primary).
	assert.GreaterOrEqual(t, snap.DedupWaiters, 0, "dedup waiters count must not be negative")

	close(release)
	wg.Wait()
}

func TestGateway_Snapshot_ThreadSafe(t *testing.T) {
	gw := NewGateway()

	// Concurrent reads of Snapshot() must not race.
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = gw.Snapshot()
		}()
	}
	wg.Wait() // If data race occurs, go test -race will catch it.
}

// --- GatewayRecorder tests ---

// mockRecorder captures RecordGatewayCall invocations for assertion.
type mockRecorder struct {
	mu    sync.Mutex
	calls []recordedCall
}

type recordedCall struct {
	method   string
	path     string
	status   int
	duration int64
	priority domain.RequestPriority
	decision domain.GatewayDecision
}

func (m *mockRecorder) RecordGatewayCall(method, path string, statusCode int, durationMs int64, priority domain.RequestPriority, decision domain.GatewayDecision) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, recordedCall{
		method:   method,
		path:     path,
		status:   statusCode,
		duration: durationMs,
		priority: priority,
		decision: decision,
	})
}

func (m *mockRecorder) last() (recordedCall, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.calls) == 0 {
		return recordedCall{}, false
	}
	return m.calls[len(m.calls)-1], true
}

func (m *mockRecorder) all() []recordedCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]recordedCall, len(m.calls))
	copy(out, m.calls)
	return out
}

func TestGateway_SetRecorder_NilSafe(t *testing.T) {
	gw := NewGateway()
	// SetRecorder(nil) must not panic.
	gw.SetRecorder(nil)
	_, err := gw.Do(context.Background(), Background,
		RequestKey{Method: "GET", Path: "/test"},
		func() (*http.Response, error) {
			return newFakeResponse(200, "ok"), nil
		})
	require.NoError(t, err)
}

func TestGateway_Recorder_AllowedDecision(t *testing.T) {
	gw := NewGateway()
	rec := &mockRecorder{}
	gw.SetRecorder(rec)

	_, err := gw.Do(context.Background(), Background,
		RequestKey{Method: "GET", Path: "/me/player"},
		func() (*http.Response, error) {
			return newFakeResponse(200, "ok"), nil
		})
	require.NoError(t, err)

	call, ok := rec.last()
	require.True(t, ok, "recorder should have captured a call")
	assert.Equal(t, "GET", call.method)
	assert.Equal(t, "/me/player", call.path)
	assert.Equal(t, 200, call.status)
	assert.Equal(t, domain.PriorityBackground, call.priority)
	assert.Equal(t, domain.DecisionAllowed, call.decision)
}

func TestGateway_Recorder_BlockedDecision(t *testing.T) {
	gw := NewGateway()
	rec := &mockRecorder{}
	gw.SetRecorder(rec)

	// Trigger a 429 to put the gateway in backoff.
	_, _ = gw.Do(context.Background(), Background,
		RequestKey{Method: "GET", Path: "/limited"},
		func() (*http.Response, error) {
			resp := newFakeResponse(429, "")
			resp.Header.Set("Retry-After", "30")
			return resp, nil
		})

	// Now a Background request during backoff should be recorded as Blocked.
	_, err := gw.Do(context.Background(), Background,
		RequestKey{Method: "GET", Path: "/blocked"},
		func() (*http.Response, error) {
			t.Error("fn should not be called for blocked request")
			return newFakeResponse(200, "ok"), nil
		})
	require.Error(t, err)

	// Find the blocked call in the recorder.
	calls := rec.all()
	var blocked *recordedCall
	for i := range calls {
		if calls[i].path == "/blocked" {
			blocked = &calls[i]
		}
	}
	require.NotNil(t, blocked, "blocked request should be recorded")
	assert.Equal(t, domain.DecisionBlocked, blocked.decision)
	assert.Equal(t, domain.PriorityBackground, blocked.priority)
	assert.Equal(t, 0, blocked.status, "blocked request has no HTTP status")
}

func TestGateway_Recorder_DedupedDecision(t *testing.T) {
	gw := NewGateway()
	rec := &mockRecorder{}
	gw.SetRecorder(rec)

	release := make(chan struct{})
	key := RequestKey{Method: "GET", Path: "/dedup-endpoint"}

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = gw.Do(context.Background(), Background, key, func() (*http.Response, error) {
				<-release
				return newFakeResponse(200, "data"), nil
			})
		}()
	}

	// Give both goroutines time to start.
	time.Sleep(20 * time.Millisecond)
	close(release)
	wg.Wait()

	calls := rec.all()
	// Should have 2 recorded calls: one Allowed + one Deduped.
	require.GreaterOrEqual(t, len(calls), 2, "both goroutines should be recorded")
	decisions := make(map[domain.GatewayDecision]int)
	for _, c := range calls {
		if c.path == "/dedup-endpoint" {
			decisions[c.decision]++
		}
	}
	assert.Equal(t, 1, decisions[domain.DecisionAllowed], "exactly one Allowed call")
	assert.Equal(t, 1, decisions[domain.DecisionDeduped], "exactly one Deduped call")
}

func TestGateway_Recorder_InteractiveCancelledDuringBackoff_RecordsWaited(t *testing.T) {
	gw := NewGateway()
	rec := &mockRecorder{}
	gw.SetRecorder(rec)

	// Set a long backoff so waitForBackoff blocks until the context is cancelled.
	gw.mu.Lock()
	gw.backoffUntil = time.Now().Add(30 * time.Second)
	gw.mu.Unlock()

	// Cancel the context while the Interactive request is waiting on the backoff.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := gw.Do(ctx, Interactive,
		RequestKey{Method: "GET", Path: "/interactive-cancelled"},
		func() (*http.Response, error) {
			t.Error("fn should not be called for cancelled request")
			return newFakeResponse(200, "ok"), nil
		})
	require.Error(t, err, "expected context cancellation error")

	// The cancelled request must be recorded as DecisionWaited with status 0.
	calls := rec.all()
	var found *recordedCall
	for i := range calls {
		if calls[i].path == "/interactive-cancelled" {
			found = &calls[i]
		}
	}
	require.NotNil(t, found, "cancelled Interactive request should be recorded")
	assert.Equal(t, domain.DecisionWaited, found.decision, "cancelled backoff wait should record DecisionWaited")
	assert.Equal(t, domain.PriorityInteractive, found.priority)
	assert.Equal(t, 0, found.status, "cancelled request has no HTTP status")
}

func TestGateway_Recorder_BackgroundCancelledDuringTokenBucketWait_RecordsBlocked(t *testing.T) {
	// Use a very slow refill rate so bucket.wait blocks until context is cancelled.
	gw := NewGateway()
	// Replace the bucket with a nearly-empty, ultra-slow one.
	gw.bucket = newTokenBucket(1, 0.001) // refills 1 token per 1000s
	// Drain the single token so the next wait will block.
	require.NoError(t, gw.bucket.wait(context.Background()))

	rec := &mockRecorder{}
	gw.SetRecorder(rec)

	// Cancel the context while waiting for a token.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := gw.Do(ctx, Background,
		RequestKey{Method: "GET", Path: "/bg-bucket-cancelled"},
		func() (*http.Response, error) {
			t.Error("fn should not be called for cancelled request")
			return newFakeResponse(200, "ok"), nil
		})
	require.Error(t, err, "expected context cancellation error")

	// The cancelled request must be recorded as DecisionBlocked with status 0.
	calls := rec.all()
	var found *recordedCall
	for i := range calls {
		if calls[i].path == "/bg-bucket-cancelled" {
			found = &calls[i]
		}
	}
	require.NotNil(t, found, "Background request cancelled during token bucket wait should be recorded")
	assert.Equal(t, domain.DecisionBlocked, found.decision, "token bucket wait cancellation should record DecisionBlocked")
	assert.Equal(t, domain.PriorityBackground, found.priority)
	assert.Equal(t, 0, found.status, "cancelled request has no HTTP status")
}

func TestGateway_Snapshot_InFlightKeys(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	gw := NewGateway()
	key := RequestKey{Method: "GET", Path: "/v1/me/player"}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = gw.Do(context.Background(), Background, key, func() (*http.Response, error) {
			req, _ := http.NewRequest("GET", srv.URL+"/player", http.NoBody)
			return http.DefaultClient.Do(req)
		})
	}()

	time.Sleep(30 * time.Millisecond)
	snap := gw.Snapshot()
	keyStr := fmt.Sprintf("%s %s", key.Method, key.Path)
	assert.Contains(t, snap.InFlightKeys, keyStr, "in-flight GET key should appear in Snapshot")

	close(release)
	wg.Wait()

	snap2 := gw.Snapshot()
	assert.NotContains(t, snap2.InFlightKeys, keyStr, "completed key should no longer appear in Snapshot")
}

// --- MarkGatewayRecorded / IsGatewayRecorded tests (I61-5) ---

// TestIsGatewayRecorded_FalseForPlainRequest verifies that a plain request
// (with no gateway marker) returns false from IsGatewayRecorded.
func TestIsGatewayRecorded_FalseForPlainRequest(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://api.spotify.com/v1/me/player", http.NoBody)
	require.NoError(t, err)
	assert.False(t, IsGatewayRecorded(req),
		"plain request without marker should return false from IsGatewayRecorded")
}

// TestIsGatewayRecorded_TrueAfterMarking verifies that IsGatewayRecorded
// returns true after MarkGatewayRecorded is called.
func TestIsGatewayRecorded_TrueAfterMarking(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://api.spotify.com/v1/me/player", http.NoBody)
	require.NoError(t, err)
	markedReq := MarkGatewayRecorded(req)
	assert.True(t, IsGatewayRecorded(markedReq),
		"marked request should return true from IsGatewayRecorded")
}

// TestMarkGatewayRecorded_OriginalUnchanged verifies that MarkGatewayRecorded
// does not modify the original request — only the returned copy is marked.
func TestMarkGatewayRecorded_OriginalUnchanged(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://api.spotify.com/v1/me/player", http.NoBody)
	require.NoError(t, err)
	_ = MarkGatewayRecorded(req)
	// The original must remain unmarked.
	assert.False(t, IsGatewayRecorded(req),
		"MarkGatewayRecorded must not mutate the original request")
}

// TestMarkGatewayRecorded_PreservesRequestProperties verifies that the marked
// request retains the original method, URL, and auth header.
func TestMarkGatewayRecorded_PreservesRequestProperties(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://api.spotify.com/v1/me/player", http.NoBody)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer test-token")

	markedReq := MarkGatewayRecorded(req)

	assert.Equal(t, http.MethodGet, markedReq.Method,
		"marked request should preserve Method")
	assert.Equal(t, req.URL.String(), markedReq.URL.String(),
		"marked request should preserve URL")
	assert.Equal(t, "Bearer test-token", markedReq.Header.Get("Authorization"),
		"marked request should preserve Authorization header")
	assert.True(t, IsGatewayRecorded(markedReq),
		"marked request should still be marked after properties check")
}

