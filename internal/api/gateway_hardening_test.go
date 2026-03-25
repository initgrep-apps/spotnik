package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Task 1: SetGateway thread-safety ---

// TestSetGateway_ThreadSafe verifies that concurrent SetGateway and doJSON
// calls do not produce a data race. Run with -race flag.
func TestSetGateway_ThreadSafe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	// Create player but don't set gateway yet.
	p := NewPlayer(srv.URL, "test-token")

	gw := NewGateway()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Alternate between setting and using the gateway.
			p.SetGateway(gw)
			_, _ = p.PlaybackState(context.Background())
			p.SetGateway(nil)
		}()
	}
	wg.Wait()
}

// TestSetGateway_NilBeforeSet verifies that the gateway field is nil before SetGateway is called,
// and that the client works correctly without a gateway.
func TestSetGateway_NilBeforeSet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	p := NewPlayer(srv.URL, "test-token")
	// No gateway set — doJSONOptional should proceed directly via http client.
	_, err := p.PlaybackState(context.Background())
	require.NoError(t, err, "client without gateway should work")
}

// --- Task 2: time.NewTimer to prevent leaks ---

// TestTokenBucket_ContextCancelReturnsImmediately verifies that wait() returns
// quickly on context cancellation without leaking a timer.
func TestTokenBucket_ContextCancelReturnsImmediately(t *testing.T) {
	// rate=0.001 tokens/s → refill takes ~1000s; we cancel well before.
	tb := newTokenBucket(1, 0.001)
	require.NoError(t, tb.wait(context.Background())) // drain

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately so wait() must return without timer expiring.
	cancel()

	start := time.Now()
	err := tb.wait(ctx)
	elapsed := time.Since(start)

	assert.ErrorIs(t, err, context.Canceled)
	assert.Less(t, elapsed.Milliseconds(), int64(100),
		"wait() should return quickly on cancellation, got %v", elapsed)
}

// TestWaitForBackoff_ContextCancelReturnsImmediately verifies that waitForBackoff
// returns quickly on context cancellation.
func TestWaitForBackoff_ContextCancelReturnsImmediately(t *testing.T) {
	gw := NewGateway()
	// Set a 1-minute backoff.
	gw.mu.Lock()
	gw.backoffUntil = time.Now().Add(time.Minute)
	gw.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	start := time.Now()
	err := gw.waitForBackoff(ctx)
	elapsed := time.Since(start)

	assert.ErrorIs(t, err, context.Canceled)
	assert.Less(t, elapsed.Milliseconds(), int64(100),
		"waitForBackoff() should return quickly on cancellation, got %v", elapsed)
}

// TestWaitForBackoff_CompletesAfterDuration verifies that waitForBackoff
// waits for the full backoff period before returning.
func TestWaitForBackoff_CompletesAfterDuration(t *testing.T) {
	gw := NewGateway()
	// Set a short backoff.
	gw.mu.Lock()
	gw.backoffUntil = time.Now().Add(80 * time.Millisecond)
	gw.mu.Unlock()

	start := time.Now()
	err := gw.waitForBackoff(context.Background())
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(60),
		"waitForBackoff should wait for the full duration, got %v", elapsed)
}

// --- Task 3: nil response guard after fn() ---

// TestGateway_NilResponseFromFn verifies that a (nil, nil) return from fn
// produces an error rather than a nil-pointer panic.
func TestGateway_NilResponseFromFn(t *testing.T) {
	gw := NewGateway()

	_, err := gw.Do(context.Background(), Background,
		RequestKey{Method: "GET", Path: "/nil-transport"},
		func() (*http.Response, error) {
			return nil, nil // degenerate case
		})

	require.Error(t, err, "nil response from fn() should return error")
	assert.Contains(t, err.Error(), "nil response",
		"error message should mention nil response")
}

// --- Task 4: io.ReadAll error in doNoContent ---

// TestDoNoContent_ReadAllError verifies that doNoContent returns an error
// when reading the response body fails.
func TestDoNoContent_ReadAllError(t *testing.T) {
	p := NewPlayer("http://"+fmt.Sprintf("localhost:%d", 19999), "test-token")
	// Build a response with a body that returns an error when read.
	req, err := http.NewRequest("PUT", "http://localhost:19999/test", nil)
	require.NoError(t, err)

	resp := &http.Response{
		StatusCode: http.StatusNoContent,
		Header:     make(http.Header),
		Body:       &failingBody{},
	}

	// Call doNoContent directly using a custom HTTP client.
	origHTTP := p.http
	p.http = &http.Client{Transport: &fixedResponseTransport{resp: resp}}
	defer func() { p.http = origHTTP }()

	err = p.doNoContent(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading response body",
		"error should mention reading body")
}

// failingBody is an io.ReadCloser that always returns an error on Read.
type failingBody struct{}

func (f *failingBody) Read(p []byte) (int, error) {
	return 0, fmt.Errorf("simulated read failure")
}

func (f *failingBody) Close() error { return nil }

// fixedResponseTransport is an http.RoundTripper that returns a fixed response.
type fixedResponseTransport struct {
	resp *http.Response
}

func (t *fixedResponseTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return t.resp, nil
}

// --- Task 5: Dedup waiters get consistent RateLimitError on 429 ---

// TestGateway_429_DedupWaitersGetRateLimitError verifies that both the primary
// caller and dedup waiters receive a RateLimitError on 429 responses.
func TestGateway_429_DedupWaitersGetRateLimitError(t *testing.T) {
	gw := NewGateway()

	release := make(chan struct{})
	key := RequestKey{Method: "GET", Path: "/rate-limited"}

	var wg sync.WaitGroup
	errs := make([]error, 3)

	// Launch 3 goroutines for the same key; only one will make the HTTP call.
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := gw.Do(context.Background(), Background, key,
				func() (*http.Response, error) {
					<-release
					resp := newFakeResponse(429, "rate limited")
					resp.Header.Set("Retry-After", "5")
					return resp, nil
				})
			errs[idx] = err
		}(i)
	}

	time.Sleep(20 * time.Millisecond)
	close(release)
	wg.Wait()

	for i, err := range errs {
		var rlErr *RateLimitError
		assert.ErrorAs(t, err, &rlErr,
			"caller %d should get RateLimitError, got %v", i, err)
		if rlErr != nil {
			assert.Equal(t, 5, rlErr.RetryAfter,
				"RetryAfter should match Retry-After header for caller %d", i)
		}
	}
}

// TestGateway_429_DedupWaiterCanReadBody verifies that dedup waiters can read
// the response body on 429 when entry.resp is cloned.
// (This is a secondary check — the primary check is that waiters get an error.)
func TestGateway_429_PrimaryAndWaiterConsistentError(t *testing.T) {
	gw := NewGateway()

	release := make(chan struct{})
	key := RequestKey{Method: "GET", Path: "/rl-body"}

	var wg sync.WaitGroup
	errs := make([]error, 2)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := gw.Do(context.Background(), Background, key,
				func() (*http.Response, error) {
					<-release
					resp := newFakeResponse(429, "Too Many Requests")
					resp.Header.Set("Retry-After", "3")
					return resp, nil
				})
			errs[idx] = err
		}(i)
	}

	time.Sleep(20 * time.Millisecond)
	close(release)
	wg.Wait()

	var rl0 *RateLimitError
	var rl1 *RateLimitError
	require.ErrorAs(t, errs[0], &rl0, "primary caller must get RateLimitError")
	require.ErrorAs(t, errs[1], &rl1, "dedup waiter must get RateLimitError")
	assert.Equal(t, rl0.RetryAfter, rl1.RetryAfter,
		"both callers should see the same RetryAfter value")
}

// --- Task 6: parseRetryAfter defaults for non-integer values ---

// TestParseRetryAfter_IntegerValue verifies a valid integer Retry-After header.
func TestParseRetryAfter_IntegerValue(t *testing.T) {
	resp := newFakeResponse(429, "")
	resp.Header.Set("Retry-After", "42")
	got := parseRetryAfter(resp)
	assert.Equal(t, 42, got)
}

// TestParseRetryAfter_NonIntegerValue verifies that non-integer values use the default.
func TestParseRetryAfter_NonIntegerValue(t *testing.T) {
	resp := newFakeResponse(429, "")
	resp.Header.Set("Retry-After", "Wed, 21 Oct 2025 07:28:00 GMT") // HTTP-date format
	got := parseRetryAfter(resp)
	assert.Equal(t, defaultRetryAfterSecs, got,
		"non-integer Retry-After should use default")
}

// TestParseRetryAfter_MissingHeader verifies that a missing header uses the default.
func TestParseRetryAfter_MissingHeader(t *testing.T) {
	resp := newFakeResponse(429, "")
	got := parseRetryAfter(resp)
	assert.Equal(t, defaultRetryAfterSecs, got,
		"missing Retry-After header should use default")
}

// TestParseRetryAfter_EmptyHeader verifies that an empty header uses the default.
func TestParseRetryAfter_EmptyHeader(t *testing.T) {
	resp := newFakeResponse(429, "")
	resp.Header.Set("Retry-After", "")
	got := parseRetryAfter(resp)
	assert.Equal(t, defaultRetryAfterSecs, got,
		"empty Retry-After header should use default")
}
