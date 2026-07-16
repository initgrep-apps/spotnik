// token bucket rate limiter for the API gateway — burst 5, 2 req/s default
// Background rate (adaptive). All requests consume tokens; Interactive is not
// bypassed but is only gated by 429 backoff at the CanAdmit pre-flight layer.
package api

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// tokenBucket implements a token-bucket rate limiter.
// Tokens are refilled continuously at `rate` per second up to `max`.
// Callers call wait() to consume a token, blocking until one is available.
type tokenBucket struct {
	mu       sync.Mutex
	tokens   float64
	max      float64 // burst capacity (independent of refill rate)
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

// setRate updates the refill rate without changing the burst capacity.
// Returns an error if rate is non-positive to prevent div-by-zero in wait().
// Existing tokens are preserved, capped to the unchanged max.
func (tb *tokenBucket) setRate(rate float64) error {
	if rate <= 0 {
		return fmt.Errorf("token bucket rate must be positive, got %f", rate)
	}
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.rate = rate
	if tb.tokens > tb.max {
		tb.tokens = tb.max
	}
	return nil
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
