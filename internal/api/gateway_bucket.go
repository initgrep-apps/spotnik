// token bucket rate limiter for the API gateway — 10 tokens/second,
// burst 10; background requests drain the bucket; interactive requests bypass it.
package api

import (
	"context"
	"sync"
	"time"
)

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
