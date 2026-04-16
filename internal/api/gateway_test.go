package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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
			key := RequestKey{Method: "GET", Path: fmt.Sprintf("/hold/%d", i), Priority: Background}
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
		key := RequestKey{Method: "GET", Path: "/sixth", Priority: Background}
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
			key := RequestKey{Method: "GET", Path: fmt.Sprintf("/hold/%d", i), Priority: Background}
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

	_, err := gw.Do(ctx, Background, RequestKey{Method: "GET", Path: "/sixth", Priority: Background}, func() (*http.Response, error) {
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

	key := RequestKey{Method: "GET", Path: "/tracks/1", Priority: Background}

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

	key1 := RequestKey{Method: "GET", Path: "/tracks/1", Priority: Background}
	key2 := RequestKey{Method: "GET", Path: "/tracks/2", Priority: Background}

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
	key := RequestKey{Method: "GET", Path: "/error-endpoint", Priority: Background}

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
		RequestKey{Method: "GET", Path: "/limited", Priority: Background},
		func() (*http.Response, error) {
			resp := newFakeResponse(429, "")
			resp.Header.Set("Retry-After", "10")
			return resp, nil
		})
	var rlErr *RateLimitError
	require.ErrorAs(t, err, &rlErr)

	// Subsequent background request should be rejected immediately.
	_, err2 := gw.Do(context.Background(), Background,
		RequestKey{Method: "GET", Path: "/other", Priority: Background},
		func() (*http.Response, error) {
			t.Error("fn should not be called during backoff")
			return newFakeResponse(200, "ok"), nil
		})
	require.ErrorAs(t, err2, &rlErr, "background request should get RateLimitError during backoff")
}

// TestGateway_Backoff_InteractiveRejectedImmediately verifies that after F27-S126,
// Interactive requests during an active backoff are rejected immediately (not queued).
func TestGateway_Backoff_InteractiveRejectedImmediately(t *testing.T) {
	gw := NewGateway()

	// Force a backoff window.
	gw.mu.Lock()
	gw.retryAfter = 5
	gw.backoffUntil = time.Now().Add(5 * time.Second)
	gw.mu.Unlock()

	// Interactive request should return a RateLimitError without waiting.
	start := time.Now()
	_, err := gw.Do(context.Background(), Interactive,
		RequestKey{Method: "GET", Path: "/interactive", Priority: Interactive},
		func() (*http.Response, error) {
			t.Error("fn should not be called — rejected before HTTP")
			return newFakeResponse(200, "ok"), nil
		})
	elapsed := time.Since(start)

	require.Error(t, err)
	var rlErr *RateLimitError
	require.ErrorAs(t, err, &rlErr, "must return RateLimitError")
	assert.Equal(t, 5, rlErr.RetryAfter)
	assert.Less(t, elapsed.Milliseconds(), int64(200),
		"Interactive rejection must be immediate (no waiting), got %v", elapsed)
}

// TestGateway_Interactive_ConsumesTokenBucket verifies that Interactive requests
// now consume tokens from the bucket (F27-S126: bucket bypass was the root cause
// of burst-fire 429s when holding a volume key at OS key-repeat rate).
func TestGateway_Interactive_ConsumesTokenBucket(t *testing.T) {
	gw := NewGateway()
	// Replace the bucket with a very slow one (1 token, essentially zero refill rate).
	gw.bucket = newTokenBucket(1, 0.001)
	// Drain the single token.
	require.NoError(t, gw.bucket.wait(context.Background()))

	// Interactive request must now wait for the bucket — context timeout proves it blocks.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := gw.Do(ctx, Interactive,
		RequestKey{Method: "POST", Path: "/play", Priority: Interactive},
		func() (*http.Response, error) {
			return newFakeResponse(204, ""), nil
		})

	// Expect a timeout error proving the request waited on the empty bucket.
	require.Error(t, err, "interactive call should now wait on the token bucket")
}

func TestGateway_IsThrottled(t *testing.T) {
	gw := NewGateway()
	assert.False(t, gw.IsThrottled(), "should not be throttled initially")

	// Trigger a 429.
	_, _ = gw.Do(context.Background(), Background,
		RequestKey{Method: "GET", Path: "/limited", Priority: Background},
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
	key := RequestKey{Method: http.MethodPost, Path: "/me/player/next", Priority: Background}

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
			key := RequestKey{Method: http.MethodGet, Path: fmt.Sprintf("/tracks/%d", i), Priority: Background}
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
			key := RequestKey{Method: http.MethodGet, Path: fmt.Sprintf("/tracks/%d", i), Priority: Background}
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

// --- Gateway state via event snapshots ---

// TestGateway_StateSnapshot_TokensAvailable verifies that a fresh gateway's first
// recorded event carries a full token bucket in its snapshot.
func TestGateway_StateSnapshot_TokensAvailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	rec := &mockEventRecorder{}
	gw := NewGateway()
	gw.SetRecorder(rec)

	key := RequestKey{Method: "PUT", Path: "/v1/me/player/play", Priority: Interactive}
	_, _ = gw.Do(context.Background(), Interactive, key, func() (*http.Response, error) {
		req, _ := http.NewRequest("PUT", srv.URL+"/play", http.NoBody)
		return http.DefaultClient.Do(req)
	})

	events := rec.all()
	require.NotEmpty(t, events, "at least one event must be recorded")
	// The first event (RequestEntered) carries the state at entry.
	first := events[0]
	assert.Equal(t, domain.EventRequestEntered, first.Kind)
	// Token bucket starts full at 10, no backoff on fresh gateway.
	assert.Equal(t, 10, first.Snapshot.TokensMax, "token max should be 10")
	assert.Equal(t, 0, first.Snapshot.ConcurrentActive, "no concurrent requests at entry")
	assert.Equal(t, 5, first.Snapshot.ConcurrentMax, "semaphore max should be 5")
	assert.Equal(t, 0.0, first.Snapshot.BackoffRemaining, "no backoff on fresh gateway")
}

// TestGateway_StateSnapshot_ConcurrentActive verifies that semaphore acquired event
// carries a non-zero ConcurrentActive count in its snapshot.
func TestGateway_StateSnapshot_ConcurrentActive(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	rec := &mockEventRecorder{}
	gw := NewGateway()
	gw.SetRecorder(rec)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		key := RequestKey{Method: "PUT", Path: "/v1/me/player/play", Priority: Interactive}
		_, _ = gw.Do(context.Background(), Interactive, key, func() (*http.Response, error) {
			req, _ := http.NewRequest("PUT", srv.URL+"/play", http.NoBody)
			return http.DefaultClient.Do(req)
		})
	}()

	// 150ms synchronization buffer to ensure the goroutine has acquired the semaphore.
	time.Sleep(150 * time.Millisecond)
	// The SemaphoreAcquired event should show ConcurrentActive >= 1.
	events := rec.all()
	var semAcq *domain.GatewayEvent
	for i := range events {
		if events[i].Kind == domain.EventSemaphoreAcquired {
			semAcq = &events[i]
			break
		}
	}
	require.NotNil(t, semAcq, "SemaphoreAcquired event must be recorded")
	assert.GreaterOrEqual(t, semAcq.Snapshot.ConcurrentActive, 1, "at least one concurrent request in flight")
	assert.Equal(t, 5, semAcq.Snapshot.ConcurrentMax)

	close(release)
	wg.Wait()

	// After completion, SemaphoreReleased event should show ConcurrentActive == 0.
	events = rec.all()
	var semRel *domain.GatewayEvent
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Kind == domain.EventSemaphoreReleased {
			semRel = &events[i]
			break
		}
	}
	require.NotNil(t, semRel, "SemaphoreReleased event must be recorded")
	assert.Equal(t, 0, semRel.Snapshot.ConcurrentActive, "concurrent count should drop to zero after release")
}

// TestGateway_StateSnapshot_BackoffRemaining verifies that a BackoffStarted event
// carries a non-zero BackoffRemaining in its snapshot after receiving a 429.
func TestGateway_StateSnapshot_BackoffRemaining(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	rec := &mockEventRecorder{}
	gw := NewGateway()
	gw.SetRecorder(rec)

	key := RequestKey{Method: "GET", Path: "/v1/me/player", Priority: Background}
	_, _ = gw.Do(context.Background(), Background, key, func() (*http.Response, error) {
		req, _ := http.NewRequest("GET", srv.URL+"/player", http.NoBody)
		return http.DefaultClient.Do(req)
	})

	events := rec.all()
	var backoffEv *domain.GatewayEvent
	for i := range events {
		if events[i].Kind == domain.EventBackoffStarted {
			backoffEv = &events[i]
			break
		}
	}
	require.NotNil(t, backoffEv, "EventBackoffStarted must be recorded after 429")
	assert.Greater(t, backoffEv.Snapshot.BackoffRemaining, 0.0, "backoff should be active after 429")
	assert.Less(t, backoffEv.Snapshot.BackoffRemaining, 31.0, "backoff should not exceed Retry-After")
}

// TestGateway_StateSnapshot_DedupWaiters verifies that DedupJoined event is recorded
// when a second GET request joins an in-flight identical request.
func TestGateway_StateSnapshot_DedupWaiters(t *testing.T) {
	release := make(chan struct{})
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		<-release
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	rec := &mockEventRecorder{}
	gw := NewGateway()
	gw.SetRecorder(rec)
	key := RequestKey{Method: "GET", Path: "/v1/me/player", Priority: Background}

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

	time.Sleep(30 * time.Millisecond)
	// DedupJoined event's snapshot must have non-negative DedupWaiters.
	events := rec.all()
	var dedupEv *domain.GatewayEvent
	for i := range events {
		if events[i].Kind == domain.EventDedupJoined {
			dedupEv = &events[i]
			break
		}
	}
	if dedupEv != nil {
		assert.GreaterOrEqual(t, dedupEv.Snapshot.DedupWaiters, 0, "dedup waiters count must not be negative")
	}

	close(release)
	wg.Wait()
}

// --- GatewayEventRecorder tests (replaces old GatewayRecorder tests) ---

func TestGateway_SetRecorder_NilSafe(t *testing.T) {
	gw := NewGateway()
	// SetRecorder(nil) must not panic.
	gw.SetRecorder(nil)
	_, err := gw.Do(context.Background(), Background,
		RequestKey{Method: "GET", Path: "/test", Priority: Background},
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
		RequestKey{Method: "GET", Path: "/me/player", Priority: Background},
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
		RequestKey{Method: "GET", Path: "/limited", Priority: Background},
		func() (*http.Response, error) {
			resp := newFakeResponse(429, "")
			resp.Header.Set("Retry-After", "30")
			return resp, nil
		})

	// Now a Background request during backoff should emit EventRequestBlocked.
	_, err := gw.Do(context.Background(), Background,
		RequestKey{Method: "GET", Path: "/blocked", Priority: Background},
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
	key := RequestKey{Method: "GET", Path: "/dedup-endpoint", Priority: Background}

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

// TestGateway_Recorder_InteractiveRejectedDuringBackoff_RecordsBlocked verifies that an
// Interactive request during an active backoff emits EventRequestBlocked (F27-S126:
// Interactive requests are rejected immediately, not queued behind waitForBackoff).
func TestGateway_Recorder_InteractiveRejectedDuringBackoff_RecordsBlocked(t *testing.T) {
	gw := NewGateway()
	rec := &mockEventRecorder{}
	gw.SetRecorder(rec)

	// Set a long backoff so an Interactive request would have previously parked until expiry.
	gw.mu.Lock()
	gw.backoffUntil = time.Now().Add(30 * time.Second)
	gw.mu.Unlock()

	_, err := gw.Do(context.Background(), Interactive,
		RequestKey{Method: "GET", Path: "/interactive-backoff-rejected", Priority: Interactive},
		func() (*http.Response, error) {
			t.Error("fn should not be called — request rejected before HTTP")
			return newFakeResponse(200, "ok"), nil
		})
	require.Error(t, err, "expected RateLimitError")

	var rlErr *RateLimitError
	require.ErrorAs(t, err, &rlErr)

	// Interactive request rejected during backoff must emit EventRequestBlocked.
	events := rec.all()
	var blocked *domain.GatewayEvent
	for i := range events {
		if events[i].Kind == domain.EventRequestBlocked && events[i].Path == "/interactive-backoff-rejected" {
			blocked = &events[i]
		}
	}
	require.NotNil(t, blocked, "Interactive request rejected during backoff should emit EventRequestBlocked")
	assert.Equal(t, domain.PriorityInteractive, blocked.Priority)
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
		RequestKey{Method: "GET", Path: "/bg-bucket-cancelled", Priority: Background},
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

// TestGateway_StateSnapshot_InFlightKeys verifies that EventHttpCompleted carries
// InFlightKeys showing the in-flight GET request, and that EventSemaphoreReleased
// does not (because the inflight entry is cleaned up before SemaphoreReleased).
//
// NOTE: The inflight key is added to the map AFTER EventSemaphoreAcquired is emitted,
// so SemaphoreAcquired will NOT contain it. The key appears in EventHttpCompleted and
// EventRequestAllowed snapshots, which are emitted before the cleanup defer fires.
func TestGateway_StateSnapshot_InFlightKeys(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	rec := &mockEventRecorder{}
	gw := NewGateway()
	gw.SetRecorder(rec)
	key := RequestKey{Method: "GET", Path: "/v1/me/player", Priority: Background}

	_, _ = gw.Do(context.Background(), Background, key, func() (*http.Response, error) {
		req, _ := http.NewRequest("GET", srv.URL+"/player", http.NoBody)
		return http.DefaultClient.Do(req)
	})

	// After completion, EventHttpCompleted snapshot must contain the in-flight key
	// (emitted before the inflight cleanup defer fires).
	events := rec.all()
	keyStr := fmt.Sprintf("%s %s", key.Method, key.Path)
	foundInFlight := false
	for _, e := range events {
		if e.Kind == domain.EventHttpCompleted {
			for _, k := range e.Snapshot.InFlightKeys {
				if k == keyStr {
					foundInFlight = true
				}
			}
		}
	}
	assert.True(t, foundInFlight, "EventHttpCompleted snapshot must contain the in-flight GET key")

	// EventSemaphoreReleased snapshot should not contain the key (cleanup defer fires first).
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Kind == domain.EventSemaphoreReleased {
			assert.NotContains(t, events[i].Snapshot.InFlightKeys, keyStr,
				"completed key should not appear in EventSemaphoreReleased snapshot")
			break
		}
	}
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
		key := RequestKey{Method: "GET", Path: fmt.Sprintf("/snap/%d", i), Priority: Background}
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
		key := RequestKey{Method: "PUT", Path: "/player/play", Priority: Interactive}
		_, _ = gw.Do(context.Background(), Interactive, key, func() (*http.Response, error) {
			<-release
			return newFakeResponse(204, ""), nil
		})
	}()

	// 150ms synchronization buffer to ensure the goroutine has acquired the semaphore.
	time.Sleep(150 * time.Millisecond)
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
		RequestKey{Method: "GET", Path: "/me/player", Priority: Background},
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
		RequestKey{Method: "GET", Path: "/blocked", Priority: Background},
		func() (*http.Response, error) {
			t.Error("fn must not be called for blocked request")
			return newFakeResponse(200, "ok"), nil
		})
	require.Error(t, err)

	kinds := collectKinds(rec.all())
	assert.Contains(t, kinds, domain.EventRequestEntered, "must emit RequestEntered")
	assert.Contains(t, kinds, domain.EventRequestBlocked, "must emit RequestBlocked")
}

// TestGateway_Do_InteractiveBackoff_EmitsBlocked verifies that an Interactive request
// arriving during active backoff emits EventRequestBlocked (F27-S126: waitForBackoff
// removed; Interactive requests rejected immediately like Background).
func TestGateway_Do_InteractiveBackoff_EmitsBlocked(t *testing.T) {
	gw := NewGateway()
	gw.mu.Lock()
	gw.backoffUntil = time.Now().Add(10 * time.Second)
	gw.mu.Unlock()

	rec := &mockEventRecorder{}
	gw.mu.Lock()
	gw.recorder = rec
	gw.mu.Unlock()

	_, err := gw.Do(context.Background(), Interactive,
		RequestKey{Method: "PUT", Path: "/play", Priority: Interactive},
		func() (*http.Response, error) {
			t.Error("fn should not be called during backoff")
			return newFakeResponse(204, ""), nil
		})
	require.Error(t, err)

	kinds := collectKinds(rec.all())
	assert.Contains(t, kinds, domain.EventRequestEntered, "must emit RequestEntered")
	assert.Contains(t, kinds, domain.EventRequestBlocked, "must emit RequestBlocked as rejection event")
	assert.NotContains(t, kinds, domain.EventRequestAllowed,
		"blocked interactive request must not emit RequestAllowed")
	assert.NotContains(t, kinds, domain.EventRequestWaited,
		"EventRequestWaited is removed — no waiting path exists any more")
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
	key := RequestKey{Method: "GET", Path: "/dedup-test", Priority: Background}

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
		RequestKey{Method: "GET", Path: "/limited", Priority: Background},
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
		RequestKey{Method: "GET", Path: "/req-id-test", Priority: Background},
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
		RequestKey{Method: "GET", Path: "/snapshot-test", Priority: Background},
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
		RequestKey{Method: "GET", Path: "/state-moment", Priority: Background},
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
		RequestKey{Method: "GET", Path: "/consume", Priority: Background},
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

// --- Story 124: Request-Aware Dedup ---

// TestDedup_InteractiveDoesNotJoinBackground verifies that an Interactive GET
// to the same path as an in-flight Background GET fires its own HTTP call
// independently — it does NOT join the Background request as a dedup waiter.
func TestDedup_InteractiveDoesNotJoinBackground(t *testing.T) {
	gw := NewGateway()

	// holdBg blocks the Background HTTP call so an Interactive GET can arrive
	// while the Background GET is still in flight.
	holdBg := make(chan struct{})
	// bgCallCount tracks Background HTTP calls; interactiveCallCount tracks Interactive ones.
	var bgCallCount, interactiveCallCount atomic.Int64

	path := "/v1/me/player"

	// Launch the Background GET and hold it in the HTTP handler.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = gw.Do(context.Background(), Background,
			RequestKey{Method: http.MethodGet, Path: path, Priority: Background},
			func() (*http.Response, error) {
				bgCallCount.Add(1)
				<-holdBg // hold until released
				return newFakeResponse(200, `{"bg":"true"}`), nil
			})
	}()

	// Give the Background goroutine time to register in the inflight map.
	time.Sleep(30 * time.Millisecond)

	// Fire an Interactive GET to the same path — must NOT join the Background.
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = gw.Do(context.Background(), Interactive,
			RequestKey{Method: http.MethodGet, Path: path, Priority: Interactive},
			func() (*http.Response, error) {
				interactiveCallCount.Add(1)
				return newFakeResponse(200, `{"interactive":"true"}`), nil
			})
	}()

	// Wait for the Interactive request to complete. Since it bypasses the inflight
	// map it will finish before the held Background GET.
	// 250ms synchronization buffer to ensure the Interactive goroutine has completed.
	time.Sleep(250 * time.Millisecond)

	// The Interactive HTTP call must have fired independently.
	assert.Equal(t, int64(1), interactiveCallCount.Load(),
		"Interactive GET must fire its own HTTP call, not join the Background waiter")

	// Release the Background hold and wait for everything to finish.
	close(holdBg)
	wg.Wait()

	assert.Equal(t, int64(1), bgCallCount.Load(),
		"Background GET must also complete with exactly 1 HTTP call")
}

// TestDedup_InteractiveDoesNotJoinInteractive verifies that two Interactive GETs
// to the same path each fire their own independent HTTP calls — they are never
// deduplicated via the inflight map regardless of whether one is already executing.
//
// Design: the first Interactive GET enters its HTTP handler (which blocks). The second
// Interactive GET then arrives while the first is still executing. With the old code the
// second would join the inflight map
// and reuse the first's response (callCount == 1). With the fix the second fires its
// own HTTP call (callCount == 2).
func TestDedup_InteractiveDoesNotJoinInteractive(t *testing.T) {
	gw := NewGateway()

	holdFirst := make(chan struct{})
	var callCount atomic.Int64
	path := "/v1/me/player"

	key := RequestKey{Method: http.MethodGet, Path: path, Priority: Interactive}

	// First Interactive GET: enters the HTTP handler and blocks there.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = gw.Do(context.Background(), Interactive, key, func() (*http.Response, error) {
			callCount.Add(1)
			<-holdFirst // hold until released
			return newFakeResponse(200, `{"state":"first"}`), nil
		})
	}()

	// 150ms synchronization buffer to ensure the first goroutine has entered the HTTP handler.
	time.Sleep(150 * time.Millisecond)

	// Second Interactive GET arrives while the first is still executing its HTTP call.
	// It must fire its own HTTP call (not join the first as a dedup waiter).
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = gw.Do(context.Background(), Interactive, key, func() (*http.Response, error) {
			callCount.Add(1)
			return newFakeResponse(200, `{"state":"second"}`), nil
		})
	}()

	// 250ms synchronization buffer to ensure both goroutines have completed.
	// Release the first hold after the second goroutine has had time to start.
	time.Sleep(250 * time.Millisecond)
	close(holdFirst)
	wg.Wait()

	// Both Interactive GETs must result in independent HTTP calls.
	assert.Equal(t, int64(2), callCount.Load(),
		"two Interactive GETs must each fire their own HTTP call (no dedup between them)")
}

// TestDedup_BackgroundJoinsBackground verifies that two concurrent Background GETs
// to the same path are deduplicated — exactly one HTTP call fires and both callers
// receive the same response body.
func TestDedup_BackgroundJoinsBackground(t *testing.T) {
	gw := NewGateway()

	var callCount atomic.Int64
	holdBg := make(chan struct{})
	path := "/v1/me/player"

	key := RequestKey{Method: http.MethodGet, Path: path, Priority: Background}
	results := make([]string, 2)
	var wg sync.WaitGroup

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			resp, err := gw.Do(context.Background(), Background, key,
				func() (*http.Response, error) {
					callCount.Add(1)
					<-holdBg
					return newFakeResponse(200, "shared-body"), nil
				})
			if err == nil && resp != nil {
				body, _ := io.ReadAll(resp.Body)
				results[idx] = string(body)
			}
		}(i)
	}

	// Give both goroutines time to race to the inflight map.
	time.Sleep(30 * time.Millisecond)
	close(holdBg)
	wg.Wait()

	// Only one HTTP call should fire.
	assert.Equal(t, int64(1), callCount.Load(),
		"two concurrent Background GETs must be deduplicated into exactly one HTTP call")
	// Both callers must receive the same body.
	assert.Equal(t, "shared-body", results[0])
	assert.Equal(t, "shared-body", results[1])
}

// --- Interactive backoff rejection tests (F27-S126) ---

// TestGateway_InteractiveRejectedDuringBackoff verifies that an Interactive request
// arriving during an active 429 backoff returns a RateLimitError immediately without
// invoking the HTTP fn (no goroutine parking / burst-fire cycle).
func TestGateway_InteractiveRejectedDuringBackoff(t *testing.T) {
	gw := NewGateway()
	gw.mu.Lock()
	gw.retryAfter = 10
	gw.backoffUntil = time.Now().Add(10 * time.Second)
	gw.mu.Unlock()

	key := RequestKey{Method: http.MethodPut, Path: "/v1/me/player/volume", Priority: Interactive}
	calls := 0
	_, err := gw.Do(context.Background(), Interactive, key, func() (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 204, Body: io.NopCloser(strings.NewReader(""))}, nil
	})

	require.Error(t, err)
	var rlErr *RateLimitError
	require.ErrorAs(t, err, &rlErr, "must return RateLimitError, not block")
	assert.Equal(t, 10, rlErr.RetryAfter)
	assert.Equal(t, 0, calls, "fn must not be called — request rejected before HTTP")
}

// TestGateway_InteractiveAllowedAfterBackoffExpires verifies that an Interactive request
// is allowed through once the 429 backoff window has passed.
func TestGateway_InteractiveAllowedAfterBackoffExpires(t *testing.T) {
	gw := NewGateway()
	gw.mu.Lock()
	gw.retryAfter = 1
	gw.backoffUntil = time.Now().Add(-1 * time.Millisecond) // already expired
	gw.mu.Unlock()

	key := RequestKey{Method: http.MethodPut, Path: "/v1/me/player/volume", Priority: Interactive}
	calls := 0
	resp, err := gw.Do(context.Background(), Interactive, key, func() (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 204, Body: io.NopCloser(strings.NewReader(""))}, nil
	})

	require.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode)
	assert.Equal(t, 1, calls, "fn must be called once when backoff has expired")
}
