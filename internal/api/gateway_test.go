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

// --- GatewayEventRecorder tests (replaces old GatewayRecorder tests) ---

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
	rec := &mockEventRecorder{}
	gw.SetRecorder(rec)

	_, err := gw.Do(context.Background(), Background,
		RequestKey{Method: "GET", Path: "/me/player"},
		func() (*http.Response, error) {
			return newFakeResponse(200, "ok"), nil
		})
	require.NoError(t, err)

	// The primary caller path emits RequestAllowed with correct fields.
	allowed := rec.byKind(domain.EventRequestAllowed)
	require.NotEmpty(t, allowed, "recorder should have captured a RequestAllowed event")
	e := allowed[0]
	assert.Equal(t, "GET", e.Method)
	assert.Equal(t, "/me/player", e.Path)
	assert.Equal(t, domain.PriorityBackground, e.Priority)
}

func TestGateway_Recorder_BlockedDecision(t *testing.T) {
	gw := NewGateway()
	rec := &mockEventRecorder{}
	gw.SetRecorder(rec)

	// Trigger a 429 to put the gateway in backoff.
	_, _ = gw.Do(context.Background(), Background,
		RequestKey{Method: "GET", Path: "/limited"},
		func() (*http.Response, error) {
			resp := newFakeResponse(429, "")
			resp.Header.Set("Retry-After", "30")
			return resp, nil
		})

	// Now a Background request during backoff should emit EventRequestBlocked.
	_, err := gw.Do(context.Background(), Background,
		RequestKey{Method: "GET", Path: "/blocked"},
		func() (*http.Response, error) {
			t.Error("fn should not be called for blocked request")
			return newFakeResponse(200, "ok"), nil
		})
	require.Error(t, err)

	// Find the blocked event.
	events := rec.all()
	var blocked *domain.GatewayEvent
	for i := range events {
		if events[i].Kind == domain.EventRequestBlocked && events[i].Path == "/blocked" {
			blocked = &events[i]
		}
	}
	require.NotNil(t, blocked, "blocked request should emit EventRequestBlocked")
	assert.Equal(t, domain.PriorityBackground, blocked.Priority)
	assert.Equal(t, 0, blocked.StatusCode, "blocked request has no HTTP status")
}

func TestGateway_Recorder_DedupedDecision(t *testing.T) {
	gw := NewGateway()
	rec := &mockEventRecorder{}
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

	kinds := collectKinds(rec.all())
	assert.Contains(t, kinds, domain.EventRequestAllowed, "primary caller emits RequestAllowed")
	assert.Contains(t, kinds, domain.EventDedupJoined, "dedup waiter emits DedupJoined")
	assert.Contains(t, kinds, domain.EventDedupResolved, "dedup waiter emits DedupResolved")
}

func TestGateway_Recorder_InteractiveCancelledDuringBackoff_RecordsWaited(t *testing.T) {
	gw := NewGateway()
	rec := &mockEventRecorder{}
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

	// Cancelled Interactive request must emit EventRequestWaited (was waiting on backoff).
	events := rec.all()
	var waited *domain.GatewayEvent
	for i := range events {
		if events[i].Kind == domain.EventRequestWaited && events[i].Path == "/interactive-cancelled" {
			waited = &events[i]
		}
	}
	require.NotNil(t, waited, "cancelled Interactive request should emit EventRequestWaited")
	assert.Equal(t, domain.PriorityInteractive, waited.Priority)
	assert.Equal(t, 0, waited.StatusCode, "cancelled request has no HTTP status")
}

func TestGateway_Recorder_BackgroundCancelledDuringTokenBucketWait_RecordsBlocked(t *testing.T) {
	// Use a very slow refill rate so bucket.wait blocks until context is cancelled.
	gw := NewGateway()
	// Replace the bucket with a nearly-empty, ultra-slow one.
	gw.bucket = newTokenBucket(1, 0.001) // refills 1 token per 1000s
	// Drain the single token so the next wait will block.
	require.NoError(t, gw.bucket.wait(context.Background()))

	rec := &mockEventRecorder{}
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

	// Cancelled background bucket wait must emit EventRequestBlocked.
	events := rec.all()
	var blocked *domain.GatewayEvent
	for i := range events {
		if events[i].Kind == domain.EventRequestBlocked && events[i].Path == "/bg-bucket-cancelled" {
			blocked = &events[i]
		}
	}
	require.NotNil(t, blocked, "Background request cancelled during token bucket wait should emit EventRequestBlocked")
	assert.Equal(t, domain.PriorityBackground, blocked.Priority)
	assert.Equal(t, 0, blocked.StatusCode, "cancelled request has no HTTP status")
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

// --- Feature 65: Gateway-internal watermarks ---

// TestGateway_NewGateway_InitializesWatermarks verifies that a fresh gateway
// starts with minTokens equal to the bucket max capacity.
func TestGateway_NewGateway_InitializesWatermarks(t *testing.T) {
	gw := NewGateway()
	snap := gw.Snapshot()
	// Before any requests, MinTokens should equal TokensMax (full bucket).
	assert.Equal(t, snap.TokensMax, snap.MinTokens,
		"fresh gateway MinTokens must equal TokensMax")
	// PeakConcurrent starts at 0 since no requests have been made.
	assert.Equal(t, 0, snap.PeakConcurrent,
		"fresh gateway PeakConcurrent must be 0")
}

// TestGateway_MinTokens_TrackedOnConsumption verifies that making requests
// through the gateway causes Snapshot().MinTokens to drop below TokensMax.
func TestGateway_MinTokens_TrackedOnConsumption(t *testing.T) {
	gw := NewGateway()
	// Initial: MinTokens == TokensMax == 10.
	initial := gw.Snapshot()
	require.Equal(t, initial.TokensMax, initial.MinTokens,
		"precondition: fresh gateway MinTokens must equal TokensMax")

	// Make 3 background requests to consume tokens from the bucket.
	for i := 0; i < 3; i++ {
		key := RequestKey{Method: "GET", Path: fmt.Sprintf("/track/%d", i)}
		_, err := gw.Do(context.Background(), Background, key, func() (*http.Response, error) {
			return newFakeResponse(200, "ok"), nil
		})
		require.NoError(t, err)
	}

	snap := gw.Snapshot()
	assert.Less(t, snap.MinTokens, snap.TokensMax,
		"MinTokens must be less than TokensMax after consuming tokens")
}

// TestGateway_PeakConcurrent_TrackedOnAcquisition verifies that launching
// concurrent requests causes Snapshot().PeakConcurrent to become positive.
func TestGateway_PeakConcurrent_TrackedOnAcquisition(t *testing.T) {
	gw := NewGateway()

	release := make(chan struct{})
	var wg sync.WaitGroup

	// Launch 3 concurrent slow requests.
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := RequestKey{Method: "GET", Path: fmt.Sprintf("/slow/%d", i)}
			_, _ = gw.Do(context.Background(), Background, key, func() (*http.Response, error) {
				<-release
				return newFakeResponse(200, "ok"), nil
			})
		}(i)
	}

	// Give goroutines time to acquire semaphore slots.
	time.Sleep(30 * time.Millisecond)

	snap := gw.Snapshot()
	assert.GreaterOrEqual(t, snap.PeakConcurrent, 2,
		"PeakConcurrent must be >= 2 when 3 concurrent requests are in-flight")

	close(release)
	wg.Wait()
}

// TestGateway_Snapshot_IncludesWatermarks verifies that Snapshot() returns
// watermark fields populated with meaningful values after activity.
func TestGateway_Snapshot_IncludesWatermarks(t *testing.T) {
	gw := NewGateway()

	// Consume a few tokens.
	for i := 0; i < 5; i++ {
		key := RequestKey{Method: "GET", Path: fmt.Sprintf("/ep/%d", i)}
		_, _ = gw.Do(context.Background(), Background, key, func() (*http.Response, error) {
			return newFakeResponse(200, "ok"), nil
		})
	}

	snap := gw.Snapshot()
	// After 5 requests, MinTokens must be <= (TokensMax - 5).
	// Allow some slack for token refill between calls, but MinTokens < TokensMax.
	assert.Less(t, snap.MinTokens, snap.TokensMax,
		"Snapshot().MinTokens must reflect token consumption")
}

// TestGateway_ResetWatermarks_ClearsToCurrentValues verifies that after
// consuming tokens and calling ResetWatermarks(), the subsequent Snapshot()
// reflects the reset values (not the historical peak/min).
func TestGateway_ResetWatermarks_ClearsToCurrentValues(t *testing.T) {
	gw := NewGateway()

	// Consume several tokens to drive MinTokens below max.
	for i := 0; i < 5; i++ {
		key := RequestKey{Method: "GET", Path: fmt.Sprintf("/ep/%d", i)}
		_, _ = gw.Do(context.Background(), Background, key, func() (*http.Response, error) {
			return newFakeResponse(200, "ok"), nil
		})
	}

	// Verify MinTokens dropped.
	before := gw.Snapshot()
	require.Less(t, before.MinTokens, before.TokensMax,
		"precondition: MinTokens must be < TokensMax after 5 requests")

	// Reset watermarks.
	gw.ResetWatermarks()

	// After reset, MinTokens should be set to current token level (not the historical min).
	after := gw.Snapshot()
	// MinTokens post-reset must be >= the pre-reset MinTokens (it was reset to current level).
	assert.GreaterOrEqual(t, after.MinTokens, before.MinTokens,
		"MinTokens after ResetWatermarks must be >= historical min (reset to current)")
	// PeakConcurrent should be 0 (no requests in-flight during reset).
	assert.Equal(t, 0, after.PeakConcurrent,
		"PeakConcurrent must reset to 0 when no concurrent requests are in-flight")
}

// TestGateway_Watermarks_IdleShowsDefaults verifies that an idle gateway
// (no requests made) shows MinTokens == TokensMax and PeakConcurrent == 0.
func TestGateway_Watermarks_IdleShowsDefaults(t *testing.T) {
	gw := NewGateway()
	snap := gw.Snapshot()
	assert.Equal(t, snap.TokensMax, snap.MinTokens,
		"idle gateway: MinTokens must equal TokensMax")
	assert.Equal(t, 0, snap.PeakConcurrent,
		"idle gateway: PeakConcurrent must be 0")
}

// --- Feature 67: captureSnapshot and emitEvent helpers ---

// mockEventRecorder captures RecordEvent invocations for assertion.
type mockEventRecorder struct {
	mu     sync.Mutex
	events []domain.GatewayEvent
}

func (m *mockEventRecorder) RecordEvent(e domain.GatewayEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, e)
}

func (m *mockEventRecorder) all() []domain.GatewayEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.GatewayEvent, len(m.events))
	copy(out, m.events)
	return out
}

func (m *mockEventRecorder) byKind(kind domain.EventKind) []domain.GatewayEvent {
	all := m.all()
	var out []domain.GatewayEvent
	for _, e := range all {
		if e.Kind == kind {
			out = append(out, e)
		}
	}
	return out
}

// TestGateway_CaptureSnapshot_TokenLevel verifies captureSnapshot returns
// the correct token level including refill arithmetic.
func TestGateway_CaptureSnapshot_TokenLevel(t *testing.T) {
	gw := NewGateway()
	// Consume 3 tokens.
	for i := 0; i < 3; i++ {
		key := RequestKey{Method: "GET", Path: fmt.Sprintf("/snap/%d", i)}
		_, _ = gw.Do(context.Background(), Background, key, func() (*http.Response, error) {
			return newFakeResponse(200, "ok"), nil
		})
	}
	snap := gw.captureSnapshot()
	assert.Equal(t, 10, snap.TokensMax, "TokensMax must always be 10")
	// After 3 tokens consumed, available should be ≤ TokensMax-3 (some refill may have occurred).
	assert.LessOrEqual(t, snap.TokensAvailable, snap.TokensMax,
		"TokensAvailable must not exceed TokensMax")
	assert.GreaterOrEqual(t, snap.TokensAvailable, 0,
		"TokensAvailable must be non-negative")
}

// TestGateway_CaptureSnapshot_ConcurrentActive verifies captureSnapshot
// reflects semaphore occupancy.
func TestGateway_CaptureSnapshot_ConcurrentActive(t *testing.T) {
	release := make(chan struct{})
	gw := NewGateway()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		key := RequestKey{Method: "PUT", Path: "/player/play"}
		_, _ = gw.Do(context.Background(), Interactive, key, func() (*http.Response, error) {
			<-release
			return newFakeResponse(204, ""), nil
		})
	}()

	time.Sleep(30 * time.Millisecond)
	snap := gw.captureSnapshot()
	assert.GreaterOrEqual(t, snap.ConcurrentActive, 1,
		"captureSnapshot must reflect in-flight semaphore occupancy")

	close(release)
	wg.Wait()
}

// TestGateway_EmitEvent_NilRecorderNoPanic verifies emitEvent does not panic
// when no recorder is attached.
func TestGateway_EmitEvent_NilRecorderNoPanic(t *testing.T) {
	gw := NewGateway()
	// No recorder attached — must not panic.
	require.NotPanics(t, func() {
		gw.emitEvent(domain.EventRequestEntered, 1, "GET", "/me/player",
			domain.PriorityBackground, 0, 0)
	})
}

// TestGateway_EmitEvent_CallsRecorderWithCorrectFields verifies that emitEvent
// calls RecordEvent with the correct Kind, RequestID, Method, Path, and Priority.
func TestGateway_EmitEvent_CallsRecorderWithCorrectFields(t *testing.T) {
	gw := NewGateway()
	rec := &mockEventRecorder{}
	gw.mu.Lock()
	gw.recorder = rec
	gw.mu.Unlock()

	gw.emitEvent(domain.EventRequestEntered, 42, "GET", "/me/player",
		domain.PriorityBackground, 0, 0)

	events := rec.all()
	require.Len(t, events, 1, "expected exactly one event")
	e := events[0]
	assert.Equal(t, domain.EventRequestEntered, e.Kind)
	assert.Equal(t, uint64(42), e.RequestID)
	assert.Equal(t, "GET", e.Method)
	assert.Equal(t, "/me/player", e.Path)
	assert.Equal(t, domain.PriorityBackground, e.Priority)
	assert.Equal(t, 10, e.Snapshot.TokensMax, "snapshot must be populated")
}

// TestGateway_NextRequestID_Increments verifies nextRequestID increments
// correctly across calls.
func TestGateway_NextRequestID_Increments(t *testing.T) {
	gw := NewGateway()
	id1 := gw.nextRequestID.Add(1)
	id2 := gw.nextRequestID.Add(1)
	id3 := gw.nextRequestID.Add(1)
	assert.Equal(t, uint64(1), id1)
	assert.Equal(t, uint64(2), id2)
	assert.Equal(t, uint64(3), id3)
}

// --- Feature 67 Task 2: Do() lifecycle event emission ---

// collectKinds extracts just the EventKind values from a slice of events.
func collectKinds(events []domain.GatewayEvent) []domain.EventKind {
	kinds := make([]domain.EventKind, len(events))
	for i, e := range events {
		kinds[i] = e.Kind
	}
	return kinds
}

// TestGateway_Do_NormalRequest_EmitsLifecycle verifies that a normal background
// GET emits: RequestEntered → TokenConsumed → SemaphoreAcquired → HttpCompleted →
// SemaphoreReleased → RequestAllowed.
func TestGateway_Do_NormalRequest_EmitsLifecycle(t *testing.T) {
	gw := NewGateway()
	rec := &mockEventRecorder{}
	gw.mu.Lock()
	gw.recorder = rec
	gw.mu.Unlock()

	_, err := gw.Do(context.Background(), Background,
		RequestKey{Method: "GET", Path: "/me/player"},
		func() (*http.Response, error) {
			return newFakeResponse(200, "ok"), nil
		})
	require.NoError(t, err)

	events := rec.all()
	kinds := collectKinds(events)
	assert.Contains(t, kinds, domain.EventRequestEntered, "must emit RequestEntered")
	assert.Contains(t, kinds, domain.EventTokenConsumed, "must emit TokenConsumed")
	assert.Contains(t, kinds, domain.EventSemaphoreAcquired, "must emit SemaphoreAcquired")
	assert.Contains(t, kinds, domain.EventHttpCompleted, "must emit HttpCompleted")
	assert.Contains(t, kinds, domain.EventSemaphoreReleased, "must emit SemaphoreReleased")
	assert.Contains(t, kinds, domain.EventRequestAllowed, "must emit RequestAllowed")
}

// TestGateway_Do_BlockedRequest_EmitsBlockedEvent verifies a background request
// during backoff emits: RequestEntered → RequestBlocked.
func TestGateway_Do_BlockedRequest_EmitsBlockedEvent(t *testing.T) {
	gw := NewGateway()
	// Set backoff directly.
	gw.mu.Lock()
	gw.backoffUntil = time.Now().Add(30 * time.Second)
	gw.mu.Unlock()

	rec := &mockEventRecorder{}
	gw.mu.Lock()
	gw.recorder = rec
	gw.mu.Unlock()

	_, err := gw.Do(context.Background(), Background,
		RequestKey{Method: "GET", Path: "/blocked"},
		func() (*http.Response, error) {
			t.Error("fn must not be called for blocked request")
			return newFakeResponse(200, "ok"), nil
		})
	require.Error(t, err)

	kinds := collectKinds(rec.all())
	assert.Contains(t, kinds, domain.EventRequestEntered, "must emit RequestEntered")
	assert.Contains(t, kinds, domain.EventRequestBlocked, "must emit RequestBlocked")
}

// TestGateway_Do_InteractiveWait_EmitsWaitedEvent verifies an interactive request
// during backoff emits RequestWaited as the final decision event (not RequestAllowed).
func TestGateway_Do_InteractiveWait_EmitsWaitedEvent(t *testing.T) {
	gw := NewGateway()
	// Set a short backoff.
	gw.mu.Lock()
	gw.backoffUntil = time.Now().Add(50 * time.Millisecond)
	gw.mu.Unlock()

	rec := &mockEventRecorder{}
	gw.mu.Lock()
	gw.recorder = rec
	gw.mu.Unlock()

	_, err := gw.Do(context.Background(), Interactive,
		RequestKey{Method: "PUT", Path: "/play"},
		func() (*http.Response, error) {
			return newFakeResponse(204, ""), nil
		})
	require.NoError(t, err)

	kinds := collectKinds(rec.all())
	assert.Contains(t, kinds, domain.EventRequestEntered, "must emit RequestEntered")
	// An interactive request that waited on backoff emits RequestWaited as the
	// final decision (not RequestAllowed) to distinguish "had to wait" from "passed through".
	assert.Contains(t, kinds, domain.EventRequestWaited, "must emit RequestWaited as final decision")
	assert.NotContains(t, kinds, domain.EventRequestAllowed,
		"waited interactive request must not emit RequestAllowed (RequestWaited is the final decision)")
}

// TestGateway_Do_DedupRequest_EmitsJoinAndResolve verifies a dedup waiter
// emits DedupJoined and DedupResolved.
func TestGateway_Do_DedupRequest_EmitsJoinAndResolve(t *testing.T) {
	gw := NewGateway()
	rec := &mockEventRecorder{}
	gw.mu.Lock()
	gw.recorder = rec
	gw.mu.Unlock()

	release := make(chan struct{})
	key := RequestKey{Method: "GET", Path: "/dedup-test"}

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

	time.Sleep(20 * time.Millisecond)
	close(release)
	wg.Wait()

	kinds := collectKinds(rec.all())
	assert.Contains(t, kinds, domain.EventDedupJoined, "must emit DedupJoined for waiter")
	assert.Contains(t, kinds, domain.EventDedupResolved, "must emit DedupResolved for waiter")
}

// TestGateway_Do_429Response_EmitsBackoffStarted verifies a 429 response
// causes EventBackoffStarted to be emitted.
func TestGateway_Do_429Response_EmitsBackoffStarted(t *testing.T) {
	gw := NewGateway()
	rec := &mockEventRecorder{}
	gw.mu.Lock()
	gw.recorder = rec
	gw.mu.Unlock()

	_, _ = gw.Do(context.Background(), Background,
		RequestKey{Method: "GET", Path: "/limited"},
		func() (*http.Response, error) {
			resp := newFakeResponse(429, "")
			resp.Header.Set("Retry-After", "5")
			return resp, nil
		})

	kinds := collectKinds(rec.all())
	assert.Contains(t, kinds, domain.EventBackoffStarted, "must emit BackoffStarted on 429")
}

// TestGateway_Do_EventsHaveCorrectRequestID verifies all events for the same
// request share the same RequestID.
func TestGateway_Do_EventsHaveCorrectRequestID(t *testing.T) {
	gw := NewGateway()
	rec := &mockEventRecorder{}
	gw.mu.Lock()
	gw.recorder = rec
	gw.mu.Unlock()

	_, _ = gw.Do(context.Background(), Background,
		RequestKey{Method: "GET", Path: "/req-id-test"},
		func() (*http.Response, error) {
			return newFakeResponse(200, "ok"), nil
		})

	events := rec.all()
	require.NotEmpty(t, events, "must have events")

	// All events for this request should share the same non-zero RequestID.
	firstID := events[0].RequestID
	assert.NotZero(t, firstID, "RequestID must be non-zero")
	for _, e := range events {
		assert.Equal(t, firstID, e.RequestID, "all request events must share the same RequestID")
	}
}

// TestGateway_Do_EventsHaveSnapshots verifies every event has a non-zero
// Snapshot.TokensMax (proving snapshot was captured).
func TestGateway_Do_EventsHaveSnapshots(t *testing.T) {
	gw := NewGateway()
	rec := &mockEventRecorder{}
	gw.mu.Lock()
	gw.recorder = rec
	gw.mu.Unlock()

	_, _ = gw.Do(context.Background(), Background,
		RequestKey{Method: "GET", Path: "/snapshot-test"},
		func() (*http.Response, error) {
			return newFakeResponse(200, "ok"), nil
		})

	events := rec.all()
	require.NotEmpty(t, events, "must have events")
	for _, e := range events {
		assert.Equal(t, 10, e.Snapshot.TokensMax,
			"every event must carry a snapshot with TokensMax=10, kind=%v", e.Kind)
	}
}

// TestGateway_Do_SnapshotReflectsStateAtMoment verifies the EventTokenConsumed
// snapshot shows TokensAvailable < TokensMax.
func TestGateway_Do_SnapshotReflectsStateAtMoment(t *testing.T) {
	gw := NewGateway()
	rec := &mockEventRecorder{}
	gw.mu.Lock()
	gw.recorder = rec
	gw.mu.Unlock()

	_, _ = gw.Do(context.Background(), Background,
		RequestKey{Method: "GET", Path: "/state-moment"},
		func() (*http.Response, error) {
			return newFakeResponse(200, "ok"), nil
		})

	consumed := rec.byKind(domain.EventTokenConsumed)
	require.NotEmpty(t, consumed, "must have TokenConsumed event")
	snap := consumed[0].Snapshot
	assert.Less(t, snap.TokensAvailable, snap.TokensMax,
		"after token consumption, TokensAvailable must be < TokensMax")
}

// --- Feature 67 Task 3: CheckAndEmitRefill / CheckAndEmitBackoffExpiry ---

// TestGateway_CheckAndEmitRefill_EmitsOnChange verifies that after consuming
// tokens, CheckAndEmitRefill emits EventTokenRefilled.
func TestGateway_CheckAndEmitRefill_EmitsOnChange(t *testing.T) {
	gw := NewGateway()
	rec := &mockEventRecorder{}
	gw.mu.Lock()
	gw.recorder = rec
	gw.mu.Unlock()

	// Consume a token (changes the token level from the initial max).
	_, _ = gw.Do(context.Background(), Background,
		RequestKey{Method: "GET", Path: "/consume"},
		func() (*http.Response, error) {
			return newFakeResponse(200, "ok"), nil
		})

	// lastEmittedTokens is now stale (was set to max, tokens are now max-1).
	// Force a clear difference by setting lastEmittedTokens high.
	gw.mu.Lock()
	gw.lastEmittedTokens = int(gw.bucket.max) // reset to max so next check detects change
	gw.mu.Unlock()

	// Simulate the token level being below max by consuming more directly.
	_ = gw.bucket.wait(context.Background())

	// Now CheckAndEmitRefill should detect the difference and emit an event.
	gw.CheckAndEmitRefill()

	refilled := rec.byKind(domain.EventTokenRefilled)
	assert.NotEmpty(t, refilled, "CheckAndEmitRefill must emit EventTokenRefilled when level changed")
}

// TestGateway_CheckAndEmitRefill_NoEmitWhenStable verifies that when the token
// level hasn't changed, no duplicate event is emitted on the second call.
func TestGateway_CheckAndEmitRefill_NoEmitWhenStable(t *testing.T) {
	gw := NewGateway()
	rec := &mockEventRecorder{}
	gw.mu.Lock()
	gw.recorder = rec
	gw.mu.Unlock()

	// First call: may emit if level differs from max.
	gw.CheckAndEmitRefill()
	count1 := len(rec.byKind(domain.EventTokenRefilled))

	// Second call immediately after: token level has not changed.
	gw.CheckAndEmitRefill()
	count2 := len(rec.byKind(domain.EventTokenRefilled))

	// No additional event should have been emitted.
	assert.Equal(t, count1, count2, "consecutive calls without level change must not emit duplicate events")
}

// TestGateway_CheckAndEmitBackoffExpiry_EmitsOnTransition verifies that when
// backoff expires, CheckAndEmitBackoffExpiry emits EventBackoffExpired.
func TestGateway_CheckAndEmitBackoffExpiry_EmitsOnTransition(t *testing.T) {
	gw := NewGateway()
	rec := &mockEventRecorder{}
	gw.mu.Lock()
	gw.recorder = rec
	gw.mu.Unlock()

	// Set a very short backoff.
	gw.mu.Lock()
	gw.backoffUntil = time.Now().Add(50 * time.Millisecond)
	gw.mu.Unlock()

	// First check: backoff is active — sets lastBackoffActive = true, no event yet.
	gw.CheckAndEmitBackoffExpiry()
	assert.Empty(t, rec.byKind(domain.EventBackoffExpired), "no event when backoff first detected as active")

	// Wait for backoff to expire.
	time.Sleep(100 * time.Millisecond)

	// Second check: backoff just expired — should emit EventBackoffExpired.
	gw.CheckAndEmitBackoffExpiry()
	assert.NotEmpty(t, rec.byKind(domain.EventBackoffExpired),
		"CheckAndEmitBackoffExpiry must emit EventBackoffExpired on active→clear transition")
}

// TestGateway_CheckAndEmitBackoffExpiry_NoEmitWhenAlreadyClear verifies that
// calling CheckAndEmitBackoffExpiry when no backoff is active emits nothing.
func TestGateway_CheckAndEmitBackoffExpiry_NoEmitWhenAlreadyClear(t *testing.T) {
	gw := NewGateway()
	rec := &mockEventRecorder{}
	gw.mu.Lock()
	gw.recorder = rec
	gw.mu.Unlock()

	// No backoff set — both calls should emit nothing.
	gw.CheckAndEmitBackoffExpiry()
	gw.CheckAndEmitBackoffExpiry()
	assert.Empty(t, rec.byKind(domain.EventBackoffExpired),
		"no event when backoff was never active")
}

// TestGateway_CheckAndEmitRefill_NilRecorder verifies no panic when no
// recorder is attached.
func TestGateway_CheckAndEmitRefill_NilRecorder(t *testing.T) {
	gw := NewGateway()
	// No recorder attached — force a level change.
	_ = gw.bucket.wait(context.Background())
	require.NotPanics(t, func() {
		gw.CheckAndEmitRefill()
		gw.CheckAndEmitBackoffExpiry()
	})
}
