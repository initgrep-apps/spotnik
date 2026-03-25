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

const priorityKey contextKey = 0

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

		// Wait for the next token or context cancellation.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitFor):
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

// Gateway is the central control point for all outbound Spotify API requests.
// It enforces:
//   - Token-bucket rate limiting (10 req/s burst of 10)
//   - Concurrency cap of 5 simultaneous in-flight requests
//   - In-flight request deduplication (same key → only one HTTP call)
//   - 429 backoff with priority bypass for Interactive requests
type Gateway struct {
	mu           sync.Mutex
	bucket       *tokenBucket
	semaphore    chan struct{} // concurrency limiter, buffered to size 5
	inflight     map[RequestKey]*inflightEntry
	backoffUntil time.Time
	retryAfter   int
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
func (g *Gateway) Do(ctx context.Context, priority Priority, key RequestKey,
	fn func() (*http.Response, error)) (*http.Response, error) {

	// Phase 1: rate limiting policy based on priority.
	if priority == Interactive {
		// Interactive: wait for backoff to expire, then proceed immediately.
		if err := g.waitForBackoff(ctx); err != nil {
			return nil, err
		}
	} else {
		// Background: reject immediately if throttled.
		g.mu.Lock()
		throttled := time.Now().Before(g.backoffUntil)
		retryAfter := g.retryAfter
		g.mu.Unlock()
		if throttled {
			return nil, &RateLimitError{RetryAfter: retryAfter}
		}
		// Apply token-bucket throttle.
		if err := g.bucket.wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit wait: %w", err)
		}
	}

	// Phase 2: concurrency semaphore.
	select {
	case g.semaphore <- struct{}{}:
		defer func() { <-g.semaphore }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Phase 3: in-flight deduplication.
	g.mu.Lock()
	if entry, ok := g.inflight[key]; ok {
		// A matching request is already in flight — wait for it.
		g.mu.Unlock()
		select {
		case <-entry.done:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		if entry.err != nil {
			return nil, entry.err
		}
		// Clone the buffered body so each caller gets their own reader.
		resp := cloneResponseWithBody(entry.resp, entry.body)
		return resp, nil
	}

	// No matching request in flight — register this one.
	entry := &inflightEntry{done: make(chan struct{})}
	g.inflight[key] = entry
	g.mu.Unlock()

	// Execute the actual HTTP call.
	resp, err := fn()

	// Buffer the response body so waiters can read it.
	if err == nil {
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			err = fmt.Errorf("buffering response body: %w", readErr)
		} else {
			entry.body = body
			// Check for 429 from the real response.
			if resp.StatusCode == http.StatusTooManyRequests {
				retryAfter := 5
				if ra := resp.Header.Get("Retry-After"); ra != "" {
					if v, err2 := strconv.Atoi(ra); err2 == nil {
						retryAfter = v
					}
				}
				g.mu.Lock()
				g.retryAfter = retryAfter
				g.backoffUntil = time.Now().Add(time.Duration(retryAfter) * time.Second)
				g.mu.Unlock()
				err = &RateLimitError{RetryAfter: retryAfter}
			} else {
				// Replace body with a readable clone for the primary caller.
				resp.Body = io.NopCloser(bytes.NewReader(body))
			}
		}
	}

	entry.resp = resp
	entry.err = err
	close(entry.done)

	// Clean up the in-flight entry.
	g.mu.Lock()
	delete(g.inflight, key)
	g.mu.Unlock()

	if err != nil {
		return nil, err
	}
	return resp, nil
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

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(remaining):
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
