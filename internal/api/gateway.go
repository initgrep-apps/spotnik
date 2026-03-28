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
	priorityKey        contextKey = 0
	gatewayRecordedKey contextKey = 1
)

// MarkGatewayRecorded returns a new request with a context marker indicating
// that the gateway has already recorded this call. LoggingTransport checks
// this marker and skips its own RecordNetCall to prevent double-recording.
func MarkGatewayRecorded(req *http.Request) *http.Request {
	ctx := context.WithValue(req.Context(), gatewayRecordedKey, true)
	return req.WithContext(ctx)
}

// IsGatewayRecorded reports whether the request context carries the gateway-recorded marker.
func IsGatewayRecorded(req *http.Request) bool {
	v, _ := req.Context().Value(gatewayRecordedKey).(bool)
	return v
}

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
	mu        sync.Mutex
	tokens    float64
	max       float64
	rate      float64 // tokens per second
	lastFill  time.Time
	minTokens float64 // lowest token level observed since last reset (watermark)
}

// newTokenBucket creates a full token bucket with the given max capacity and
// refill rate (tokens per second).
func newTokenBucket(max, rate float64) *tokenBucket {
	return &tokenBucket{
		tokens:    max,
		max:       max,
		rate:      rate,
		lastFill:  time.Now(),
		minTokens: max, // start at max — no consumption observed yet
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
			// Track the minimum token level at the exact moment of consumption,
			// before any refill can hide the dip. This is the key fix over
			// Feature 64's UI-side sampling approach.
			if tb.tokens < tb.minTokens {
				tb.minTokens = tb.tokens
			}
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

// GatewayRecorder records per-request gateway decisions for visualization.
// Implemented by *state.Store; defined here to avoid an import cycle
// (api/ cannot import state/).
type GatewayRecorder interface {
	RecordGatewayCall(method, path string, statusCode int, durationMs int64,
		priority domain.RequestPriority, decision domain.GatewayDecision)
}

// Gateway is the central control point for all outbound Spotify API requests.
// It enforces:
//   - Token-bucket rate limiting (10 req/s burst of 10)
//   - Concurrency cap of 5 simultaneous in-flight requests
//   - In-flight request deduplication (same key → only one HTTP call)
//   - 429 backoff with priority bypass for Interactive requests
type Gateway struct {
	mu             sync.Mutex
	bucket         *tokenBucket
	semaphore      chan struct{} // concurrency limiter, buffered to size 5
	inflight       map[RequestKey]*inflightEntry
	backoffUntil   time.Time
	retryAfter     int
	recorder       GatewayRecorder // optional; nil means no recording
	peakConcurrent int             // max semaphore occupancy observed since last reset (watermark)
}

// SetRecorder sets the gateway decision recorder. Pass nil to disable recording.
// Thread-safe.
func (g *Gateway) SetRecorder(r GatewayRecorder) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.recorder = r
}

// NewGateway creates a Gateway with default limits:
// 10 requests/second token bucket, burst of 10, max 5 concurrent in-flight.
func NewGateway() *Gateway {
	return &Gateway{
		bucket:    newTokenBucket(10, 10),
		semaphore: make(chan struct{}, 5),
		inflight:  make(map[RequestKey]*inflightEntry),
	}
}

// Snapshot returns a read-only snapshot of the gateway's current internal state.
// Thread-safe — acquires the gateway mutex and the token bucket mutex internally.
// Best-effort point-in-time: each field group is read under its own lock, so the
// snapshot is not guaranteed to be atomically consistent across all fields.
func (g *Gateway) Snapshot() domain.GatewayState {
	// Read the token bucket level and watermark without consuming a token.
	g.bucket.mu.Lock()
	// Apply any pending refill before reading so the value is current.
	now := time.Now()
	elapsed := now.Sub(g.bucket.lastFill).Seconds()
	tokens := g.bucket.tokens + elapsed*g.bucket.rate
	if tokens > g.bucket.max {
		tokens = g.bucket.max
	}
	tokenMax := int(g.bucket.max)
	minTokens := int(g.bucket.minTokens)
	g.bucket.mu.Unlock()

	// Read gateway fields under the gateway mutex.
	g.mu.Lock()
	backoffRemaining := time.Until(g.backoffUntil).Seconds()
	if backoffRemaining < 0 {
		backoffRemaining = 0
	}
	// DedupWaiters = number of in-flight GET requests in the dedup map.
	// Each entry represents one primary in-flight call. Secondary goroutines
	// that join as waiters are not separately tracked here.
	dedupWaiters := len(g.inflight)
	inFlightKeys := make([]string, 0, len(g.inflight))
	for k := range g.inflight {
		inFlightKeys = append(inFlightKeys, fmt.Sprintf("%s %s", k.Method, k.Path))
	}
	peakConcurrent := g.peakConcurrent
	g.mu.Unlock()

	// Concurrent active = semaphore slots currently occupied.
	concurrentMax := cap(g.semaphore)
	concurrentActive := len(g.semaphore)

	return domain.GatewayState{
		TokensAvailable:  int(tokens),
		TokensMax:        tokenMax,
		ConcurrentActive: concurrentActive,
		ConcurrentMax:    concurrentMax,
		BackoffRemaining: backoffRemaining,
		DedupWaiters:     dedupWaiters,
		InFlightKeys:     inFlightKeys,
		PeakConcurrent:   peakConcurrent,
		MinTokens:        minTokens,
	}
}

// ResetWatermarks resets peak activity watermarks to their current values.
// Called by the UI on each 1-second boundary so annotations reflect only
// activity in the most recent window.
func (g *Gateway) ResetWatermarks() {
	g.bucket.mu.Lock()
	// Apply pending refill before reading the current level, exactly as Snapshot()
	// does. Without this, minTokens could be set below the refilled TokensAvailable
	// returned by the next Snapshot(), causing a false (min: N) annotation.
	now := time.Now()
	elapsed := now.Sub(g.bucket.lastFill).Seconds()
	current := g.bucket.tokens + elapsed*g.bucket.rate
	if current > g.bucket.max {
		current = g.bucket.max
	}
	g.bucket.minTokens = current
	g.bucket.mu.Unlock()

	g.mu.Lock()
	// Reset peakConcurrent to current semaphore occupancy (usually 0 when idle).
	g.peakConcurrent = len(g.semaphore)
	g.mu.Unlock()
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
// When a GatewayRecorder is attached, Do() records the gateway decision
// (Allowed/Waited/Deduped/Blocked) for each request, including Background
// requests rejected by backoff that never reach the HTTP layer.
//
// IMPORTANT: Callers must call MarkGatewayRecorded on the *http.Request passed
// to fn when a GatewayRecorder is attached. This prevents LoggingTransport from
// double-recording the request. See BaseClient.doJSON for the canonical pattern.
func (g *Gateway) Do(ctx context.Context, priority Priority, key RequestKey,
	fn func() (*http.Response, error)) (*http.Response, error) {

	domainPriority := priorityToDomain(priority)

	// waited is set true when an Interactive request blocks on the backoff timer.
	// This causes the final recording to use DecisionWaited instead of DecisionAllowed,
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
			g.mu.Lock()
			rec := g.recorder
			g.mu.Unlock()
			if rec != nil {
				rec.RecordGatewayCall(key.Method, key.Path, 0, 0,
					domainPriority, domain.DecisionWaited)
			}
			return nil, err
		}
	} else {
		// Background: reject immediately if throttled.
		g.mu.Lock()
		throttled := time.Now().Before(g.backoffUntil)
		retryAfter := g.retryAfter
		rec := g.recorder
		g.mu.Unlock()
		if throttled {
			// Record the block decision before returning — blocked Background
			// requests never reach the HTTP layer so LoggingTransport misses them.
			if rec != nil {
				rec.RecordGatewayCall(key.Method, key.Path, 0, 0,
					domainPriority, domain.DecisionBlocked)
			}
			return nil, &RateLimitError{RetryAfter: retryAfter}
		}
		// Apply token-bucket throttle.
		if err := g.bucket.wait(ctx); err != nil {
			g.mu.Lock()
			rec := g.recorder
			g.mu.Unlock()
			if rec != nil {
				rec.RecordGatewayCall(key.Method, key.Path, 0, 0,
					domainPriority, domain.DecisionBlocked)
			}
			return nil, fmt.Errorf("rate limit wait: %w", err)
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
			// A matching GET is already in flight — wait without holding a slot.
			rec := g.recorder
			g.mu.Unlock()
			waitStart := time.Now()
			select {
			case <-entry.done:
			case <-ctx.Done():
				if rec != nil {
					rec.RecordGatewayCall(key.Method, key.Path, 0, 0,
						domainPriority, domain.DecisionDeduped)
				}
				return nil, ctx.Err()
			}
			// Record the dedup decision — the waiter re-uses the primary's result.
			if rec != nil {
				statusCode := 0
				if entry.resp != nil {
					statusCode = entry.resp.StatusCode
				}
				rec.RecordGatewayCall(key.Method, key.Path, statusCode,
					time.Since(waitStart).Milliseconds(), domainPriority, domain.DecisionDeduped)
			}
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
		// Track peak concurrent usage for watermark display. len(g.semaphore) is
		// read under the gateway mutex to avoid a race with ResetWatermarks().
		g.mu.Lock()
		active := len(g.semaphore)
		if active > g.peakConcurrent {
			g.peakConcurrent = active
		}
		g.mu.Unlock()
		defer func() { <-g.semaphore }()
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
			rec := g.recorder
			g.mu.Unlock()
			waitStart := time.Now()
			select {
			case <-existing.done:
			case <-ctx.Done():
				if rec != nil {
					rec.RecordGatewayCall(key.Method, key.Path, 0, 0,
						domainPriority, domain.DecisionDeduped)
				}
				return nil, ctx.Err()
			}
			if rec != nil {
				statusCode := 0
				if existing.resp != nil {
					statusCode = existing.resp.StatusCode
				}
				rec.RecordGatewayCall(key.Method, key.Path, statusCode,
					time.Since(waitStart).Milliseconds(), domainPriority, domain.DecisionDeduped)
			}
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
	// Mark the context so LoggingTransport knows the gateway will record this call,
	// preventing double-recording in the net log.
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

			// On 429: set the gateway backoff and create a RateLimitError.
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
				err = &RateLimitError{RetryAfter: retryAfter}
			}
		}
	}

	if entry != nil {
		entry.resp = resp
		entry.err = err
		// entry.done is closed by the deferred cleanup above.
	}

	// Record the final decision for the primary caller.
	// DecisionWaited is used when an Interactive request had to block on the backoff
	// timer before proceeding; DecisionAllowed covers all other primary-caller paths.
	finalDecision := domain.DecisionAllowed
	if waited {
		finalDecision = domain.DecisionWaited
	}
	g.mu.Lock()
	rec := g.recorder
	g.mu.Unlock()
	if rec != nil {
		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
		}
		rec.RecordGatewayCall(key.Method, key.Path, statusCode,
			time.Since(httpStart).Milliseconds(), domainPriority, finalDecision)
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
