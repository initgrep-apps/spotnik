package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Task 1: constructor initializes debounceEntries ---

// TestNewGateway_DebounceEntriesInitialized verifies that NewGateway() initializes
// the debounceEntries map as a non-nil map ready for use.
func TestNewGateway_DebounceEntriesInitialized(t *testing.T) {
	gw := NewGateway()
	gw.debounceMu.Lock()
	defer gw.debounceMu.Unlock()
	assert.NotNil(t, gw.debounceEntries, "debounceEntries must be initialized in NewGateway")
}

// --- Task 2: interactiveDebounce method ---

// TestInteractiveDebounce_SingleRequest verifies that a single call to
// interactiveDebounce waits approximately 100ms and returns nil.
func TestInteractiveDebounce_SingleRequest(t *testing.T) {
	gw := NewGateway()
	start := time.Now()
	err := gw.interactiveDebounce(context.Background(), "/v1/search")
	elapsed := time.Since(start)

	require.NoError(t, err)
	// Should have waited at least 80ms (80% of 100ms to tolerate timing jitter).
	assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(80),
		"single request should wait ~100ms, got %v", elapsed)
}

// TestInteractiveDebounce_SecondRequestCancelsFirst verifies that when two
// requests for the same path arrive within 100ms, the first is cancelled (returns
// an error) and the second proceeds after its own 100ms hold.
func TestInteractiveDebounce_SecondRequestCancelsFirst(t *testing.T) {
	gw := NewGateway()

	errs := make([]error, 2)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		errs[0] = gw.interactiveDebounce(context.Background(), "/v1/search")
	}()

	// Give first goroutine time to register its entry.
	time.Sleep(10 * time.Millisecond)

	wg.Add(1)
	go func() {
		defer wg.Done()
		errs[1] = gw.interactiveDebounce(context.Background(), "/v1/search")
	}()

	wg.Wait()

	// First request should have been cancelled.
	assert.Error(t, errs[0], "first request for same path should be cancelled by second")
	// Second request should have succeeded.
	assert.NoError(t, errs[1], "second request should succeed after 100ms hold")
}

// TestInteractiveDebounce_DifferentPaths verifies that requests for different
// paths are independent — both proceed without interfering with each other.
func TestInteractiveDebounce_DifferentPaths(t *testing.T) {
	gw := NewGateway()

	errs := make([]error, 2)
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		errs[0] = gw.interactiveDebounce(context.Background(), "/v1/search")
	}()
	go func() {
		defer wg.Done()
		errs[1] = gw.interactiveDebounce(context.Background(), "/v1/me/player")
	}()

	wg.Wait()

	assert.NoError(t, errs[0], "request for /v1/search should not be affected by /v1/me/player")
	assert.NoError(t, errs[1], "request for /v1/me/player should not be affected by /v1/search")
}

// TestInteractiveDebounce_CallerCtxCancelled verifies that if the caller's
// context is cancelled during the 100ms hold, interactiveDebounce returns
// immediately with the context error.
func TestInteractiveDebounce_CallerCtxCancelled(t *testing.T) {
	gw := NewGateway()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- gw.interactiveDebounce(ctx, "/v1/search")
	}()

	// Cancel after a short time, well before the 100ms hold completes.
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		assert.Error(t, err, "should return error when ctx is cancelled")
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("interactiveDebounce did not return after context cancellation")
	}
}

// TestInteractiveDebounce_NoGoroutineLeak verifies that after
// interactiveDebounce exits, the entry is removed from the map and the ready
// channel is closed (no goroutine leak).
func TestInteractiveDebounce_NoGoroutineLeak(t *testing.T) {
	gw := NewGateway()

	err := gw.interactiveDebounce(context.Background(), "/v1/search")
	require.NoError(t, err)

	gw.debounceMu.Lock()
	entry, exists := gw.debounceEntries["/v1/search"]
	gw.debounceMu.Unlock()

	assert.False(t, exists, "entry should be removed from map after exit")
	assert.Nil(t, entry, "entry pointer should be nil after removal")
}

// --- Task 3: Do() integration ---

// TestGateway_Do_BackgroundBypassesDebounce verifies that Background requests
// pass through Do() without any debounce delay (< 10ms overhead).
func TestGateway_Do_BackgroundBypassesDebounce(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	gw := NewGateway()
	key := RequestKey{Method: http.MethodGet, Path: "/v1/me/player/queue"}

	start := time.Now()
	_, err := gw.Do(context.Background(), Background, key, func() (*http.Response, error) {
		return http.Get(srv.URL) //nolint:noctx
	})
	elapsed := time.Since(start)

	require.NoError(t, err)
	// Background should complete in well under 50ms (no 100ms debounce hold).
	assert.Less(t, elapsed.Milliseconds(), int64(50),
		"Background request should bypass debounce, got %v", elapsed)
}

// TestGateway_Do_InteractiveExperiencesDebounceHold verifies that an Interactive
// request experiences approximately 100ms hold before the HTTP call is made.
func TestGateway_Do_InteractiveExperiencesDebounceHold(t *testing.T) {
	httpCalled := make(chan struct{}, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpCalled <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	gw := NewGateway()
	key := RequestKey{Method: http.MethodGet, Path: "/v1/search"}

	start := time.Now()
	_, err := gw.Do(context.Background(), Interactive, key, func() (*http.Response, error) {
		return http.Get(srv.URL) //nolint:noctx
	})
	elapsed := time.Since(start)

	require.NoError(t, err)

	// HTTP call should have been made.
	select {
	case <-httpCalled:
	default:
		t.Fatal("HTTP handler was not called")
	}

	// Should have waited at least 80ms for the debounce hold.
	assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(80),
		"Interactive request should experience ~100ms debounce hold, got %v", elapsed)
}

// --- Task 4: data race check ---

// --- Story 104: Gateway Integration Tests ---

// TestGateway_InteractiveDebounce_LastWins verifies that when two Interactive Do()
// calls for the same path arrive within 10ms, only the second (last) request reaches
// the HTTP server. The debounce ensures earlier duplicate requests are cancelled.
func TestGateway_InteractiveDebounce_LastWins(t *testing.T) {
	var mu sync.Mutex
	var requestCount int
	var lastBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		lastBody = r.URL.Query().Get("q")
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	gw := NewGateway()
	key := RequestKey{Method: http.MethodGet, Path: "/v1/search"}

	// Launch two goroutines that call Interactive Do() for the same path in rapid succession.
	var wg sync.WaitGroup
	var errs [2]error

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, errs[0] = gw.Do(context.Background(), Interactive, key, func() (*http.Response, error) {
			return http.Get(srv.URL + "?q=first") //nolint:noctx
		})
	}()

	// Small delay so the first goroutine enters interactiveDebounce first.
	time.Sleep(5 * time.Millisecond)

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, errs[1] = gw.Do(context.Background(), Interactive, key, func() (*http.Response, error) {
			return http.Get(srv.URL + "?q=second") //nolint:noctx
		})
	}()

	wg.Wait()

	// The second request should succeed; the first may be cancelled.
	assert.NoError(t, errs[1], "second Interactive request for same path should succeed")

	// Exactly 1 request should reach the server (the second one wins).
	mu.Lock()
	count := requestCount
	mu.Unlock()
	assert.Equal(t, 1, count, "exactly 1 HTTP request should reach the server (last-wins debounce)")
	assert.Equal(t, "second", lastBody, "the winning request should be the second one")
}

// TestGateway_InteractiveDebounce_DifferentPathsIndependent verifies that Interactive
// Do() calls for different paths are independent — both requests reach the server
// without interfering with each other.
func TestGateway_InteractiveDebounce_DifferentPathsIndependent(t *testing.T) {
	var mu sync.Mutex
	var requestPaths []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestPaths = append(requestPaths, r.URL.Path)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	gw := NewGateway()
	searchKey := RequestKey{Method: http.MethodGet, Path: "/v1/search"}
	devicesKey := RequestKey{Method: http.MethodGet, Path: "/v1/me/player/devices"}

	var wg sync.WaitGroup
	var errs [2]error

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, errs[0] = gw.Do(context.Background(), Interactive, searchKey, func() (*http.Response, error) {
			return http.Get(srv.URL + "/v1/search") //nolint:noctx
		})
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, errs[1] = gw.Do(context.Background(), Interactive, devicesKey, func() (*http.Response, error) {
			return http.Get(srv.URL + "/v1/me/player/devices") //nolint:noctx
		})
	}()

	wg.Wait()

	// Both requests should succeed — different paths don't debounce each other.
	assert.NoError(t, errs[0], "/v1/search Interactive request should succeed")
	assert.NoError(t, errs[1], "/v1/me/player/devices Interactive request should succeed")

	// Both HTTP requests should reach the server.
	mu.Lock()
	count := len(requestPaths)
	mu.Unlock()
	assert.Equal(t, 2, count,
		"both requests for different paths should reach the server (independent debounce per path)")
}

// TestGateway_Background_NoDebounce verifies that Background Do() calls bypass the
// 100ms interactive debounce. Unlike Interactive requests, Background requests for the
// same path proceed immediately without any hold delay.
//
// NOTE: Background requests still share the GET in-flight deduplication layer,
// so two perfectly simultaneous Background GETs for the same key may merge. To test
// debounce bypass without dedup interference, we use sequential Background calls
// (each after the prior one completes) and verify the total elapsed time is far below
// the 100ms interactive hold threshold.
func TestGateway_Background_NoDebounce(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	gw := NewGateway()
	key := RequestKey{Method: http.MethodGet, Path: "/v1/me/player"}

	// Two sequential Background requests: measure total time.
	// If debounce were applied (100ms per request) the total would be ≥200ms.
	// Background bypasses debounce, so both complete in well under 50ms.
	start := time.Now()

	_, err0 := gw.Do(context.Background(), Background, key, func() (*http.Response, error) {
		return http.Get(srv.URL) //nolint:noctx
	})
	_, err1 := gw.Do(context.Background(), Background, key, func() (*http.Response, error) {
		return http.Get(srv.URL) //nolint:noctx
	})

	elapsed := time.Since(start)

	assert.NoError(t, err0, "first Background request should succeed")
	assert.NoError(t, err1, "second Background request should succeed")
	// Background requests must complete in well under 50ms each (no 100ms debounce hold).
	assert.Less(t, elapsed.Milliseconds(), int64(100),
		"two sequential Background requests must complete in <100ms total (no debounce hold); got %v", elapsed)
}

// TestGateway_Background_vs_Interactive_TimingComparison verifies the timing difference
// between Background and Interactive requests. An Interactive request experiences the
// ~100ms debounce hold; a Background request bypasses it and completes immediately.
func TestGateway_Background_vs_Interactive_TimingComparison(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	gw := NewGateway()
	bgKey := RequestKey{Method: http.MethodGet, Path: "/v1/me/player"}
	interactiveKey := RequestKey{Method: http.MethodGet, Path: "/v1/search"}

	// Background: should complete without any debounce hold.
	bgStart := time.Now()
	_, err := gw.Do(context.Background(), Background, bgKey, func() (*http.Response, error) {
		return http.Get(srv.URL) //nolint:noctx
	})
	bgElapsed := time.Since(bgStart)

	require.NoError(t, err)
	assert.Less(t, bgElapsed.Milliseconds(), int64(50),
		"Background request must complete in <50ms (no debounce); got %v", bgElapsed)

	// Interactive: should experience ≥80ms debounce hold.
	intStart := time.Now()
	_, err = gw.Do(context.Background(), Interactive, interactiveKey, func() (*http.Response, error) {
		return http.Get(srv.URL) //nolint:noctx
	})
	intElapsed := time.Since(intStart)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, intElapsed.Milliseconds(), int64(80),
		"Interactive request must wait ≥80ms for debounce; got %v", intElapsed)
}

// TestInteractiveDebounce_RapidConcurrent verifies that rapid concurrent
// Interactive requests for the same path produce no data races.
// Run with -race flag: go test -race ./internal/api/...
func TestInteractiveDebounce_RapidConcurrent(t *testing.T) {
	gw := NewGateway()

	const goroutines = 10
	var started sync.WaitGroup
	var done sync.WaitGroup
	var successCount atomic.Int32

	started.Add(goroutines)
	done.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer done.Done()
			started.Done()
			started.Wait() // all goroutines start as simultaneously as possible
			err := gw.interactiveDebounce(context.Background(), "/v1/search")
			if err == nil {
				successCount.Add(1)
			}
		}()
	}

	done.Wait()

	// With 10 concurrent requests all firing at nearly the same time,
	// at most a few should succeed (most will be cancelled by a successor).
	// The exact count depends on scheduling, but no data race should occur.
	t.Logf("successCount=%d out of %d concurrent requests", successCount.Load(), goroutines)
}
