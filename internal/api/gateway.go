// Package api provides the Spotify HTTP client and gateway infrastructure.
package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/initgrep-apps/spotnik/internal/domain"
)

// Priority classifies a request so the gateway can apply different policies.
// Interactive requests come from user key presses and should feel instant.
// Background requests come from polling loops and can be throttled or dropped.
type Priority int

const (
	// Background is for polling, pre-fetch, and any non-user-initiated request.
	Background Priority = iota
	// Interactive is for requests triggered by user key presses.
	Interactive
)

// contextKey is a package-private type to avoid collisions with other packages
// using context.WithValue.
type contextKey int

const (
	priorityKey contextKey = 0
)

// WithPriority returns a new context that carries the given Priority.
// BaseClient reads this via PriorityFromContext when routing through the Gateway.
func WithPriority(ctx context.Context, p Priority) context.Context {
	return context.WithValue(ctx, priorityKey, p)
}

// PriorityFromContext extracts the Priority from ctx, defaulting to Background
// if none is set.
func PriorityFromContext(ctx context.Context) Priority {
	if v, ok := ctx.Value(priorityKey).(Priority); ok {
		return v
	}
	return Background
}

// tokenBucket implements a token-bucket rate limiter.
// Tokens are refilled continuously at `rate` per second up to `max`.
// Callers call wait() to consume a token, blocking until one is available.
type tokenBucket struct {
	mu       sync.Mutex
	tokens   float64
	max      float64
	rate     float64 // tokens per second
	lastFill time.Time
}

// newTokenBucket creates a full token bucket with the given max capacity and
// refill rate (tokens per second).
func newTokenBucket(max, rate float64) *tokenBucket {
	return &tokenBucket{
		tokens:   max,
		max:      max,
		rate:     rate,
		lastFill: time.Now(),
	}
}

// wait blocks until a token is available or ctx is cancelled.
// It refills tokens based on elapsed time before checking availability.
func (tb *tokenBucket) wait(ctx context.Context) error {
	for {
		tb.mu.Lock()
		// Refill: add tokens proportional to elapsed time since last fill.
		now := time.Now()
		elapsed := now.Sub(tb.lastFill).Seconds()
		tb.tokens += elapsed * tb.rate
		if tb.tokens > tb.max {
			tb.tokens = tb.max
		}
		tb.lastFill = now

		if tb.tokens >= 1 {
			tb.tokens--
			tb.mu.Unlock()
			return nil
		}

		// Calculate how long until the next token is available.
		waitFor := time.Duration((1.0-tb.tokens)/tb.rate*1000) * time.Millisecond
		tb.mu.Unlock()

		// Use time.NewTimer instead of time.After to prevent timer leaks
		// when ctx is cancelled before the timer fires.
		timer := time.NewTimer(waitFor)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			// Loop back and try again.
		}
	}
}

// --- Gateway ---

// RequestKey uniquely identifies a request for deduplication purposes.
// Two requests with the same Method and Path are considered identical.
type RequestKey struct {
	Method string
	Path   string
}

// inflightEntry tracks an in-flight HTTP request for deduplication.
// All goroutines waiting on the same key share this entry.
type inflightEntry struct {
	done chan struct{}
	resp *http.Response
	body []byte
	err  error
}

// interactiveDebounceEntry holds the cancel function and ready channel for one
// pending Interactive request waiting in the 100ms debounce hold window.
// cancel stops the hold; ready is closed when the entry exits (used by the
// replacement to wait before registering itself).
type interactiveDebounceEntry struct {
	cancel context.CancelFunc
	ready  chan struct{}
}

// Gateway is the central control point for all outbound Spotify API requests.
// It enforces:
//   - Token-bucket rate limiting (10 req/s burst of 10)
//   - Concurrency cap of 5 simultaneous in-flight requests
//   - In-flight request deduplication (same key → only one HTTP call)
//   - 429 backoff with priority bypass for Interactive requests
//   - 100ms transport-layer debounce for Interactive requests (path-keyed)
type Gateway struct {
	mu            sync.Mutex
	bucket        *tokenBucket
	semaphore     chan struct{} // concurrency limiter, buffered to size 5
	inflight      map[RequestKey]*inflightEntry
	backoffUntil  time.Time
	retryAfter    int
	recorder      domain.GatewayEventRecorder // optional; nil means no recording
	nextRequestID atomic.Uint64               // monotonically incrementing request ID

	// lastEmittedTokens tracks the token level of the last TokenRefilled event.
	// Used to avoid emitting duplicate refill events when tokens haven't changed.
	lastEmittedTokens int
	// lastBackoffActive tracks whether backoff was active at the last check.
	// Used to detect the backoff→clear transition for BackoffExpired events.
	lastBackoffActive bool

	// debounceMu protects debounceEntries.
	debounceMu sync.Mutex
	// debounceEntries maps API path → pending Interactive debounce entry.
	// Only one entry per path exists at a time; a new arrival cancels the prior.
	debounceEntries map[string]*interactiveDebounceEntry
}

// SetRecorder sets the gateway event recorder. Pass nil to disable recording.
// Thread-safe.
func (g *Gateway) SetRecorder(r domain.GatewayEventRecorder) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.recorder = r
}

// NewGateway creates a Gateway with default limits:
// 10 requests/second token bucket, burst of 10, max 5 concurrent in-flight.
func NewGateway() *Gateway {
	g := &Gateway{
		bucket:          newTokenBucket(10, 10),
		semaphore:       make(chan struct{}, 5),
		inflight:        make(map[RequestKey]*inflightEntry),
		debounceEntries: make(map[string]*interactiveDebounceEntry),
	}
	// Initialize lastEmittedTokens to max so the first CheckAndEmitRefill
	// only fires when the level actually changes from the initial full state.
	g.lastEmittedTokens = int(g.bucket.max)
	return g
}

// captureSnapshot reads the gateway's current state under locks.
// Returns a GatewayStateSnapshot suitable for embedding in a GatewayEvent.
//
// Lock ordering: acquires bucket.mu first, then g.mu. The caller must NOT
// hold g.mu when calling this method. If the caller already holds g.mu,
// use captureSnapshotLocked() instead.
func (g *Gateway) captureSnapshot() domain.GatewayStateSnapshot {
	g.bucket.mu.Lock()
	now := time.Now()
	elapsed := now.Sub(g.bucket.lastFill).Seconds()
	tokens := g.bucket.tokens + elapsed*g.bucket.rate
	if tokens > g.bucket.max {
		tokens = g.bucket.max
	}
	tokenMax := int(g.bucket.max)
	g.bucket.mu.Unlock()

	g.mu.Lock()
	backoffRemaining := time.Until(g.backoffUntil).Seconds()
	if backoffRemaining < 0 {
		backoffRemaining = 0
	}
	dedupWaiters := len(g.inflight)
	inFlightKeys := make([]string, 0, len(g.inflight))
	for k := range g.inflight {
		inFlightKeys = append(inFlightKeys, fmt.Sprintf("%s %s", k.Method, k.Path))
	}
	g.mu.Unlock()

	return domain.GatewayStateSnapshot{
		TokensAvailable:  int(tokens),
		TokensMax:        tokenMax,
		ConcurrentActive: len(g.semaphore),
		ConcurrentMax:    cap(g.semaphore),
		BackoffRemaining: backoffRemaining,
		DedupWaiters:     dedupWaiters,
		InFlightKeys:     inFlightKeys,
	}
}

// captureSnapshotLocked reads gateway state when g.mu is already held.
// Only acquires bucket.mu (safe — bucket.mu is never held when g.mu is acquired).
// Reads semaphore length without a lock (channel len is always safe).
// Reads inflight/backoff from g.mu-protected fields without re-acquiring.
func (g *Gateway) captureSnapshotLocked() domain.GatewayStateSnapshot {
	g.bucket.mu.Lock()
	now := time.Now()
	elapsed := now.Sub(g.bucket.lastFill).Seconds()
	tokens := g.bucket.tokens + elapsed*g.bucket.rate
	if tokens > g.bucket.max {
		tokens = g.bucket.max
	}
	tokenMax := int(g.bucket.max)
	g.bucket.mu.Unlock()

	// g.mu is already held by caller — read fields directly.
	backoffRemaining := time.Until(g.backoffUntil).Seconds()
	if backoffRemaining < 0 {
		backoffRemaining = 0
	}
	dedupWaiters := len(g.inflight)
	inFlightKeys := make([]string, 0, len(g.inflight))
	for k := range g.inflight {
		inFlightKeys = append(inFlightKeys, fmt.Sprintf("%s %s", k.Method, k.Path))
	}

	return domain.GatewayStateSnapshot{
		TokensAvailable:  int(tokens),
		TokensMax:        tokenMax,
		ConcurrentActive: len(g.semaphore),
		ConcurrentMax:    cap(g.semaphore),
		BackoffRemaining: backoffRemaining,
		DedupWaiters:     dedupWaiters,
		InFlightKeys:     inFlightKeys,
	}
}

// emitEvent records a gateway event if a recorder is attached.
// Captures a state snapshot at the current moment.
// The caller must NOT hold g.mu — use emitEventLocked() if g.mu is held.
func (g *Gateway) emitEvent(kind domain.EventKind, reqID uint64, method, path string,
	priority domain.RequestPriority, statusCode int, durationMs int64) {
	g.mu.Lock()
	rec := g.recorder
	g.mu.Unlock()
	if rec == nil {
		return
	}
	rec.RecordEvent(domain.GatewayEvent{
		Timestamp:  time.Now(),
		Kind:       kind,
		RequestID:  reqID,
		Method:     method,
		Path:       path,
		Priority:   priority,
		StatusCode: statusCode,
		DurationMs: durationMs,
		Snapshot:   g.captureSnapshot(),
	})
}

// emitEventLocked is like emitEvent but for use when g.mu is already held.
// Reads recorder from the locked state and uses captureSnapshotLocked().
func (g *Gateway) emitEventLocked(kind domain.EventKind, reqID uint64, method, path string,
	priority domain.RequestPriority, statusCode int, durationMs int64) {
	rec := g.recorder
	if rec == nil {
		return
	}
	rec.RecordEvent(domain.GatewayEvent{
		Timestamp:  time.Now(),
		Kind:       kind,
		RequestID:  reqID,
		Method:     method,
		Path:       path,
		Priority:   priority,
		StatusCode: statusCode,
		DurationMs: durationMs,
		Snapshot:   g.captureSnapshotLocked(),
	})
}

// CheckAndEmitRefill checks if the token bucket level has changed since the
// last emission and emits EventTokenRefilled if so. Called by the app on
// viz.TickMsg (every 200ms). Does NOT mutate bucket.tokens — the lazy refill
// stays as-is for the hot path.
func (g *Gateway) CheckAndEmitRefill() {
	g.bucket.mu.Lock()
	now := time.Now()
	elapsed := now.Sub(g.bucket.lastFill).Seconds()
	tokens := g.bucket.tokens + elapsed*g.bucket.rate
	if tokens > g.bucket.max {
		tokens = g.bucket.max
	}
	currentLevel := int(tokens)
	g.bucket.mu.Unlock()

	g.mu.Lock()
	changed := currentLevel != g.lastEmittedTokens
	rec := g.recorder
	g.lastEmittedTokens = currentLevel
	g.mu.Unlock()

	if changed && rec != nil {
		rec.RecordEvent(domain.GatewayEvent{
			Timestamp: time.Now(),
			Kind:      domain.EventTokenRefilled,
			Snapshot:  g.captureSnapshot(),
		})
	}
}

// CheckAndEmitBackoffExpiry checks if the 429 backoff period has just expired
// and emits EventBackoffExpired on the active→cleared transition. Called by the
// app on viz.TickMsg (every 200ms).
func (g *Gateway) CheckAndEmitBackoffExpiry() {
	g.mu.Lock()
	nowActive := time.Now().Before(g.backoffUntil)
	wasActive := g.lastBackoffActive
	g.lastBackoffActive = nowActive
	rec := g.recorder
	g.mu.Unlock()

	if wasActive && !nowActive && rec != nil {
		rec.RecordEvent(domain.GatewayEvent{
			Timestamp: time.Now(),
			Kind:      domain.EventBackoffExpired,
			Snapshot:  g.captureSnapshot(),
		})
	}
}

// IsThrottled returns true when the gateway is in a 429 backoff period.
func (g *Gateway) IsThrottled() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return time.Now().Before(g.backoffUntil)
}

// RetryAfterSecs returns the Retry-After duration in seconds from the last 429.
func (g *Gateway) RetryAfterSecs() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.retryAfter
}

// Do executes fn as a controlled HTTP call through the gateway.
//
// For Background requests:
//   - Go through the token bucket.
//   - If in 429 backoff, return a RateLimitError immediately.
//
// For Interactive requests:
//   - Skip token bucket.
//   - If in 429 backoff, wait until the backoff expires before proceeding.
//
// Both priorities:
//   - Acquire the concurrency semaphore.
//   - Check the in-flight map; if a matching request is already running,
//     wait for it and return the cached result.
//   - On 429 response, set backoffUntil and return RateLimitError.
//
// When a GatewayEventRecorder is attached, Do() emits fine-grained lifecycle events
// at every decision point (entered, token consumed, semaphore acquired/released,
// request allowed/blocked/waited, dedup joined/resolved, HTTP completed, backoff started).
func (g *Gateway) Do(ctx context.Context, priority Priority, key RequestKey,
	fn func() (*http.Response, error)) (*http.Response, error) {

	domainPriority := priorityToDomain(priority)

	// Generate a unique request ID for correlating all events for this request.
	reqID := g.nextRequestID.Add(1)

	// Emit RequestEntered at the top of Do(), before any policy checks.
	g.emitEvent(domain.EventRequestEntered, reqID, key.Method, key.Path, domainPriority, 0, 0)

	// waited is set true when an Interactive request blocks on the backoff timer.
	// This causes the final event to use EventRequestWaited instead of EventRequestAllowed,
	// so the UI can distinguish "passed through immediately" from "had to wait at backoff".
	var waited bool

	// Phase 1: rate limiting policy based on priority.
	if priority == Interactive {
		// Interactive: wait for backoff to expire, then proceed immediately.
		// Check first whether any backoff is active — if so, the request waited.
		g.mu.Lock()
		waited = time.Now().Before(g.backoffUntil)
		g.mu.Unlock()
		if err := g.waitForBackoff(ctx); err != nil {
			// Context cancelled while waiting on backoff — emit RequestWaited (was blocked).
			g.emitEvent(domain.EventRequestWaited, reqID, key.Method, key.Path, domainPriority, 0, 0)
			return nil, err
		}
	} else {
		// Background: reject immediately if throttled.
		g.mu.Lock()
		throttled := time.Now().Before(g.backoffUntil)
		retryAfter := g.retryAfter
		// Emit blocked event under g.mu using emitEventLocked — g.mu is held.
		if throttled {
			g.emitEventLocked(domain.EventRequestBlocked, reqID, key.Method, key.Path, domainPriority, 0, 0)
		}
		g.mu.Unlock()
		if throttled {
			return nil, &RateLimitError{RetryAfter: retryAfter}
		}
		// Apply token-bucket throttle.
		if err := g.bucket.wait(ctx); err != nil {
			// Context cancelled while waiting for a token — emit blocked event.
			// g.mu is NOT held here (bucket.wait releases bucket.mu before returning).
			g.emitEvent(domain.EventRequestBlocked, reqID, key.Method, key.Path, domainPriority, 0, 0)
			return nil, fmt.Errorf("rate limit wait: %w", err)
		}
		// Token successfully consumed — emit TokenConsumed event.
		g.emitEvent(domain.EventTokenConsumed, reqID, key.Method, key.Path, domainPriority, 0, 0)
	}

	// Phase 1b: transport-layer debounce for Interactive requests only.
	// Keyed by path (query params ignored) — all /v1/search requests share one slot.
	// Background requests bypass this phase entirely.
	if priority == Interactive {
		if err := g.interactiveDebounce(ctx, key.Path); err != nil {
			return nil, err
		}
	}

	// Phase 2: in-flight deduplication for GET requests only.
	// Dedup waiters do NOT consume a semaphore slot — they wait for the primary
	// caller (which holds a slot) to finish, then reuse its result.
	// Mutating requests (POST/PUT/DELETE) are never deduplicated because each
	// side-effect must fire independently.
	if key.Method == http.MethodGet {
		g.mu.Lock()
		if entry, ok := g.inflight[key]; ok {
			// A matching GET is already in flight — join as a dedup waiter.
			g.mu.Unlock()
			// Emit DedupJoined before waiting on entry.done.
			g.emitEvent(domain.EventDedupJoined, reqID, key.Method, key.Path, domainPriority, 0, 0)
			waitStart := time.Now()
			select {
			case <-entry.done:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			// Emit DedupResolved — the waiter received the shared response.
			statusCode := 0
			if entry.resp != nil {
				statusCode = entry.resp.StatusCode
			}
			g.emitEvent(domain.EventDedupResolved, reqID, key.Method, key.Path,
				domainPriority, statusCode, time.Since(waitStart).Milliseconds())
			if entry.err != nil {
				return nil, entry.err
			}
			// Clone the buffered body so each caller gets their own reader.
			// Defensively guard against a nil response — should not happen in practice
			// since the primary caller always sets entry.resp before closing entry.done,
			// but the check prevents a nil-pointer dereference if the invariant is ever broken.
			if entry.resp == nil {
				return nil, fmt.Errorf("dedup: primary caller returned nil response")
			}
			return cloneResponseWithBody(entry.resp, entry.body), nil
		}
		g.mu.Unlock()
	}

	// Phase 3: concurrency semaphore (only primary callers reach here).
	select {
	case g.semaphore <- struct{}{}:
		// Emit SemaphoreAcquired after successfully acquiring the slot.
		g.emitEvent(domain.EventSemaphoreAcquired, reqID, key.Method, key.Path, domainPriority, 0, 0)
		// Emit SemaphoreReleased in the defer so it fires even on panic or early return.
		defer func() {
			<-g.semaphore
			g.emitEvent(domain.EventSemaphoreReleased, reqID, key.Method, key.Path, domainPriority, 0, 0)
		}()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Phase 4: register in inflight map for GET requests, then execute.
	// Double-check: another goroutine may have registered between our check
	// and acquiring the semaphore.
	var entry *inflightEntry
	if key.Method == http.MethodGet {
		g.mu.Lock()
		if existing, ok := g.inflight[key]; ok {
			// Lost the race — become a waiter (semaphore slot is held via defer).
			// We release the slot in the defer above and join as a waiter result.
			g.mu.Unlock()
			// Emit DedupJoined before waiting.
			g.emitEvent(domain.EventDedupJoined, reqID, key.Method, key.Path, domainPriority, 0, 0)
			waitStart := time.Now()
			select {
			case <-existing.done:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			// Emit DedupResolved.
			statusCode := 0
			if existing.resp != nil {
				statusCode = existing.resp.StatusCode
			}
			g.emitEvent(domain.EventDedupResolved, reqID, key.Method, key.Path,
				domainPriority, statusCode, time.Since(waitStart).Milliseconds())
			if existing.err != nil {
				return nil, existing.err
			}
			// Nil-guard: defensively check existing.resp before cloning.
			if existing.resp == nil {
				return nil, fmt.Errorf("dedup: primary caller returned nil response")
			}
			return cloneResponseWithBody(existing.resp, existing.body), nil
		}
		entry = &inflightEntry{done: make(chan struct{})}
		g.inflight[key] = entry
		g.mu.Unlock()

		// Ensure the entry is always closed and removed, even on panic.
		defer func() {
			close(entry.done)
			g.mu.Lock()
			delete(g.inflight, key)
			g.mu.Unlock()
		}()
	}

	// Execute the actual HTTP call.
	httpStart := time.Now()
	resp, err := fn()

	// Guard against the rare (nil, nil) return from a transport implementation.
	// Without this check, resp.Body would panic with a nil-pointer dereference.
	if resp == nil && err == nil {
		err = fmt.Errorf("HTTP transport returned nil response")
	}

	// Buffer the response body so waiters can read it.
	if err == nil {
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			err = fmt.Errorf("buffering response body: %w", readErr)
		} else {
			// Always replace the body with a readable clone so the primary
			// caller can read it regardless of status code.
			resp.Body = io.NopCloser(bytes.NewReader(body))

			if entry != nil {
				entry.body = body
			}

			// On 429: set the gateway backoff and emit BackoffStarted before creating the error.
			// We do NOT suppress the error here — checkResponseStatus in
			// doJSON/doNoContent would also create one, but dedup waiters
			// bypass checkResponseStatus. By creating the error here, all
			// callers (primary + dedup waiters) receive a consistent
			// RateLimitError with the correct RetryAfter value.
			if resp.StatusCode == http.StatusTooManyRequests {
				retryAfter := parseRetryAfter(resp)
				g.mu.Lock()
				g.retryAfter = retryAfter
				g.backoffUntil = time.Now().Add(time.Duration(retryAfter) * time.Second)
				g.mu.Unlock()
				// Emit BackoffStarted now that backoffUntil is set.
				g.emitEvent(domain.EventBackoffStarted, reqID, key.Method, key.Path, domainPriority, resp.StatusCode, 0)
				err = &RateLimitError{RetryAfter: retryAfter}
			}
		}
	}

	// Emit HttpCompleted with status code and latency.
	httpDuration := time.Since(httpStart).Milliseconds()
	httpStatus := 0
	if resp != nil {
		httpStatus = resp.StatusCode
	}
	g.emitEvent(domain.EventHttpCompleted, reqID, key.Method, key.Path,
		domainPriority, httpStatus, httpDuration)

	if entry != nil {
		entry.resp = resp
		entry.err = err
		// entry.done is closed by the deferred cleanup above.
	}

	// Emit the final decision event for the primary caller.
	// EventRequestWaited is used when an Interactive request had to block on the
	// backoff timer before proceeding; EventRequestAllowed covers all other primary paths.
	if waited {
		g.emitEvent(domain.EventRequestWaited, reqID, key.Method, key.Path, domainPriority, httpStatus, httpDuration)
	} else {
		g.emitEvent(domain.EventRequestAllowed, reqID, key.Method, key.Path, domainPriority, httpStatus, httpDuration)
	}

	if err != nil {
		return nil, err
	}
	return resp, nil
}

// priorityToDomain converts the api-layer Priority to the domain.RequestPriority
// that is shared with the state and UI layers.
func priorityToDomain(p Priority) domain.RequestPriority {
	if p == Interactive {
		return domain.PriorityInteractive
	}
	return domain.PriorityBackground
}

// interactiveDebounce implements a 100ms hold window for Interactive requests
// keyed by API path (query params ignored). If a newer request for the same
// path arrives within 100ms, the older one is cancelled and returns an error.
// Only the last request in any burst window proceeds.
//
// The wrappedCtx allows a competing request to cancel this hold without also
// cancelling the caller's ctx (which controls the downstream HTTP call).
func (g *Gateway) interactiveDebounce(ctx context.Context, path string) error {
	// wrappedCtx allows a competing request to cancel this hold without
	// cancelling the caller's ctx (which would affect the HTTP call too).
	wrappedCtx, wrappedCancel := context.WithCancel(ctx)

	g.debounceMu.Lock()
	if prev, ok := g.debounceEntries[path]; ok {
		// Cancel the prior request's hold and wait for it to finish unregistering.
		prev.cancel()
		g.debounceMu.Unlock()
		<-prev.ready
		g.debounceMu.Lock()
	}
	entry := &interactiveDebounceEntry{
		cancel: wrappedCancel,
		ready:  make(chan struct{}),
	}
	g.debounceEntries[path] = entry
	g.debounceMu.Unlock()

	// Cleanup: remove from map and signal ready when we exit.
	defer func() {
		wrappedCancel()
		g.debounceMu.Lock()
		if g.debounceEntries[path] == entry {
			delete(g.debounceEntries, path)
		}
		g.debounceMu.Unlock()
		close(entry.ready)
	}()

	// Hold for 100ms. The first request to survive the full hold proceeds.
	// Use time.NewTimer instead of time.After to prevent timer leaks when
	// ctx is cancelled before the 100ms window expires.
	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil // proceed to HTTP
	case <-wrappedCtx.Done():
		return wrappedCtx.Err() // cancelled by newer request for same path
	case <-ctx.Done():
		return ctx.Err() // cancelled by caller (Esc, new query from app.go)
	}
}

// waitForBackoff blocks until the current 429 backoff period expires or ctx is cancelled.
func (g *Gateway) waitForBackoff(ctx context.Context) error {
	for {
		g.mu.Lock()
		until := g.backoffUntil
		g.mu.Unlock()

		remaining := time.Until(until)
		if remaining <= 0 {
			return nil
		}

		// Use time.NewTimer instead of time.After to prevent timer leaks
		// when ctx is cancelled before the timer fires.
		timer := time.NewTimer(remaining)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			return nil
		}
	}
}

// cloneResponseWithBody creates a shallow copy of resp with the Body replaced
// by a new reader over body. Used so multiple waiters each get their own Body.
func cloneResponseWithBody(resp *http.Response, body []byte) *http.Response {
	clone := *resp
	clone.Body = io.NopCloser(bytes.NewReader(body))
	return &clone
}

// defaultRetryAfterSecs is the fallback wait duration when no parseable
// Retry-After header is present in a 429 response.
const defaultRetryAfterSecs = 5

// parseRetryAfter extracts the Retry-After value (in seconds) from a 429 response.
// If the header is missing or not a plain integer (e.g. HTTP-date format per RFC 7231),
// it returns defaultRetryAfterSecs.
// NOTE: Spotify always sends integer seconds; HTTP-date values are intentionally
// not supported — falling back to the default is the correct behaviour.
func parseRetryAfter(resp *http.Response) int {
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if v, err := strconv.Atoi(ra); err == nil {
			return v
		}
		// Non-integer Retry-After (possibly HTTP-date format per RFC 7231).
		// Fall through to default — we don't support date-based values.
	}
	return defaultRetryAfterSecs
}
