---
title: "Apply Token Bucket to Interactive Requests"
feature: 27-gateway-rate-protection
status: done
---

## Background

### What F26 left behind

Story 125 removed the 100ms `interactiveDebounce` from the gateway — correct,
because the debounce was silently discarding semantically independent commands.
But the debounce had a secondary effect: it accidentally rate-limited Interactive
requests to at most 1 per 100ms (10/s). Removing it left Interactive requests
with **no local rate gate at all**.

The token-bucket check in `Do()` has always been in the Background-only `else`
branch:

```go
// internal/api/gateway.go  Do()  lines 345–367  (CURRENT — BROKEN)

if priority == Interactive {
    // ... backoff wait (fixed in Story 126) ...
} else {
    // Background only:
    if err := g.bucket.wait(ctx); err != nil {   // ← Interactive never reaches this
        return nil, fmt.Errorf("rate limit wait: %w", err)
    }
    g.emitEvent(domain.EventTokenConsumed, ...)
}
```

This means every Interactive request reaches Spotify with **zero local rate
checking**. The only protection is Spotify's own server-side rate limit — which
triggers a 429.

### Why bucket bypass was considered safe before F26

Before F25 the gateway had `interactiveDebounce` (100ms hold window, last-wins
per path). It was designed for search ("last query wins") but incidentally
capped Interactive throughput to 10/s as a side effect. Story 125 correctly
identified the debounce as wrong for playback (each press is independent) and
removed it. The token-bucket bypass was not re-examined at the same time.

### Proof — OS key-repeat rate vs. bucket capacity

A modern macOS key-repeat at the default "fast" setting fires at **~15 events/s**
(66ms between events). The gateway bucket holds 10 tokens and refills at 10/s.

```
Bucket state: 10 tokens (full)
User holds '-' key.  OS fires ~15 events/s.

Event  1 (t=0ms):    PUT dispatched. Bucket bypassed. Spotify: 200
Event  2 (t=66ms):   PUT dispatched. Bucket bypassed. Spotify: 200
Event  3 (t=133ms):  PUT dispatched. Bucket bypassed. Spotify: 200
...
Event 10 (t=600ms):  PUT dispatched. Bucket bypassed. Spotify: 200
Event 11 (t=666ms):  PUT dispatched. Bucket bypassed. Spotify: 200  ← bucket would
Event 12 (t=733ms):  PUT dispatched. Bucket bypassed. Spotify: 200    have been empty
Event 13 (t=800ms):  PUT dispatched. Bucket bypassed. Spotify: 200    for events 11+
Event 14 (t=866ms):  PUT dispatched. Bucket bypassed. Spotify: 200    if enforced
Event 15 (t=933ms):  PUT dispatched. Bucket bypassed. Spotify: 429  ← burst limit hit
```

At 15 events/s for 1 second, 15 PUTs fire. The bucket (10 tokens, 10/s refill)
would have held 10 of them; 5 would have waited in the bucket ~200ms each.
With the bypass, all 15 fire instantly. Spotify returns 429 on the 15th.

### Why enforcing the bucket for Interactive is safe for user experience

A single Interactive press on a warm bucket (tokens available) consumes one
token and proceeds immediately. `bucket.wait` only blocks if the bucket is
empty. In normal usage — occasional keypresses, not a held key — the bucket
stays comfortably full and Interactive requests feel instant.

The only scenario where `bucket.wait` adds latency is a sustained burst (held
key). In that case, latency is desirable: it prevents a 429.

Background polling consumes one token per second (1s tick interval). That
leaves 9 tokens/s for Interactive — plenty for any realistic single-key usage.

---

## Design

### `internal/api/gateway.go` — Phase 1 restructure

Move `bucket.wait` out of the `else` branch so both priorities consume a token.
The `if/else` separating Interactive from Background in Phase 1 becomes two
sequential checks: first the priority-specific check (backoff), then the shared
check (token bucket).

```go
// BEFORE — lines 334–367 (after Story 126's backoff fix, but before this story)

if priority == Interactive {
    g.mu.Lock()
    throttled := time.Now().Before(g.backoffUntil)
    retryAfter := g.retryAfter
    if throttled {
        g.emitEventLocked(domain.EventRequestBlocked, ...)
    }
    g.mu.Unlock()
    if throttled {
        return nil, &RateLimitError{RetryAfter: retryAfter}
    }
} else {
    // Background: reject immediately if throttled.
    g.mu.Lock()
    throttled := time.Now().Before(g.backoffUntil)
    retryAfter := g.retryAfter
    if throttled {
        g.emitEventLocked(domain.EventRequestBlocked, ...)
    }
    g.mu.Unlock()
    if throttled {
        return nil, &RateLimitError{RetryAfter: retryAfter}
    }
    // Apply token-bucket throttle.
    if err := g.bucket.wait(ctx); err != nil {
        g.emitEvent(domain.EventRequestBlocked, ...)
        return nil, fmt.Errorf("rate limit wait: %w", err)
    }
    g.emitEvent(domain.EventTokenConsumed, ...)
}

// AFTER — Phase 1 becomes two shared blocks

// Phase 1a: backoff check — both priorities reject immediately.
g.mu.Lock()
throttled := time.Now().Before(g.backoffUntil)
retryAfter := g.retryAfter
if throttled {
    g.emitEventLocked(domain.EventRequestBlocked, reqID, key.Method, key.Path, domainPriority, 0, 0)
}
g.mu.Unlock()
if throttled {
    return nil, &RateLimitError{RetryAfter: retryAfter}
}

// Phase 1b: token bucket — both priorities consume a token.
if err := g.bucket.wait(ctx); err != nil {
    g.emitEvent(domain.EventRequestBlocked, reqID, key.Method, key.Path, domainPriority, 0, 0)
    return nil, fmt.Errorf("rate limit wait: %w", err)
}
g.emitEvent(domain.EventTokenConsumed, reqID, key.Method, key.Path, domainPriority, 0, 0)
```

The dedup Phase 2 guard (`if key.Method == http.MethodGet && priority == Background`)
and semaphore Phase 3 are unchanged — Interactive still skips dedup.

### Update `Gateway` doc comment

```go
// BEFORE:
//   - Token-bucket rate limiting (10 req/s burst of 10)
//   - 429 backoff with priority bypass for Interactive requests

// AFTER:
//   - Token-bucket rate limiting (10 req/s burst of 10, both priorities)
//   - 429 backoff: both priorities are rejected immediately; Interactive
//     requests are not queued (see Story 126)
```

### Simulation — AFTER both fixes (Story 126 + Story 127)

Same scenario: user holds `-` for 3 seconds at 15 events/s.

```
Bucket state: 10 tokens full.  No active backoff.

t=0ms:    PUT #1 → Phase 1a: no backoff. Phase 1b: token consumed (9 left). Spotify: 200
t=66ms:   PUT #2 → bucket: 8 tokens. Spotify: 200
t=133ms:  PUT #3 → bucket: 7 tokens. Spotify: 200
t=200ms:  PUT #4 → bucket: 6 tokens. Spotify: 200
t=266ms:  PUT #5 → bucket: 5 tokens. Spotify: 200
t=333ms:  PUT #6 → bucket: 4 tokens. Spotify: 200
t=400ms:  PUT #7 → bucket: 3 tokens. Spotify: 200
t=466ms:  PUT #8 → bucket: 2 tokens. Spotify: 200
t=533ms:  PUT #9 → bucket: 1 token.  Spotify: 200
t=600ms:  PUT #10 → bucket: 0 tokens. Spotify: 200
                    (bucket exhausted — refill rate 10/s = 1 token per 100ms)

t=666ms:  PUT #11 → bucket.wait() BLOCKS until ~t=700ms when 1 token refills.
          PUT fires at t=700ms. bucket: 0 tokens.  Spotify: 200
t=733ms:  PUT #12 → bucket.wait() blocks until ~t=800ms. Spotify: 200
...
          From this point, requests are naturally throttled to ~10/s.
          No 429. No cascade. Volume changes smoothly ~10 steps/second.

t=3000ms: User releases key. No goroutines parked. No backlog.
```

With Story 126 in place, if a 429 somehow still occurs (e.g. from a concurrent
Background burst), Interactive requests during the backoff window are rejected
immediately — they never pile up.

---

## Acceptance Criteria

- [ ] Interactive requests consume a token from the bucket before proceeding,
  exactly as Background requests do.
- [ ] A single Interactive request on a warm bucket (tokens available) returns
  in the same time as before — no added latency for normal usage.
- [ ] `make ci` passes.

---

## Tasks

- [ ] In `internal/api/gateway.go` `Do()`, replace the `if priority == Interactive { ... } else { ... }` Phase 1 block with two sequential shared blocks: Phase 1a (backoff check, same for both) and Phase 1b (token bucket, same for both), as shown in the Design section.
  - test: `go build ./internal/api/...` compiles cleanly

- [ ] Update the `Gateway` doc comment as shown in the Design section.

- [ ] Add `TestGateway_InteractiveConsumesToken` to
  `internal/api/gateway_test.go`:
  ```go
  func TestGateway_InteractiveConsumesToken(t *testing.T) {
      gw := NewGateway()
      // Drain bucket to exactly 1 token by consuming 9 of them via Background requests.
      for i := 0; i < 9; i++ {
          key := RequestKey{Method: http.MethodGet, Path: fmt.Sprintf("/bg/%d", i), Priority: Background}
          _, err := gw.Do(context.Background(), Background, key, func() (*http.Response, error) {
              return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("{}"))}, nil
          })
          require.NoError(t, err)
      }
      // 1 token remains.  One Interactive request must succeed immediately.
      key := RequestKey{Method: http.MethodPut, Path: "/v1/me/player/volume", Priority: Interactive}
      start := time.Now()
      resp, err := gw.Do(context.Background(), Interactive, key, func() (*http.Response, error) {
          return &http.Response{StatusCode: 204, Body: io.NopCloser(strings.NewReader(""))}, nil
      })
      elapsed := time.Since(start)

      require.NoError(t, err)
      assert.Equal(t, 204, resp.StatusCode)
      assert.Less(t, elapsed, 50*time.Millisecond, "single token available: should proceed instantly")
  }
  ```
  - test: `go test ./internal/api/... -run TestGateway_InteractiveConsumesToken` PASS

- [ ] Add `TestGateway_InteractiveThrottledByBucket` to
  `internal/api/gateway_test.go` — verifies that when bucket is empty, an
  Interactive request blocks until a token is available:
  ```go
  func TestGateway_InteractiveThrottledByBucket(t *testing.T) {
      // Create a bucket with rate=10/s but set tokens=0 by exhausting them first.
      gw := NewGateway()
      for i := 0; i < 10; i++ {
          key := RequestKey{Method: http.MethodGet, Path: fmt.Sprintf("/bg/%d", i), Priority: Background}
          _, _ = gw.Do(context.Background(), Background, key, func() (*http.Response, error) {
              return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("{}"))}, nil
          })
      }
      // Bucket is now empty.  Interactive request must wait for refill.
      key := RequestKey{Method: http.MethodPut, Path: "/v1/me/player/volume", Priority: Interactive}
      start := time.Now()
      resp, err := gw.Do(context.Background(), Interactive, key, func() (*http.Response, error) {
          return &http.Response{StatusCode: 204, Body: io.NopCloser(strings.NewReader(""))}, nil
      })
      elapsed := time.Since(start)

      require.NoError(t, err)
      assert.Equal(t, 204, resp.StatusCode)
      // Bucket refills at 10/s = 100ms per token.  Allow up to 200ms for scheduling jitter.
      assert.GreaterOrEqual(t, elapsed, 80*time.Millisecond, "empty bucket: must wait for refill")
      assert.Less(t, elapsed, 200*time.Millisecond, "refill is predictable: should not wait too long")
  }
  ```
  - test: `go test ./internal/api/... -run TestGateway_InteractiveThrottledByBucket` PASS

- [ ] Verify existing Background tests still pass — the Phase 1 restructure must
  not regress Background throttle or reject behaviour:
  - test: `go test ./internal/api/... -run TestGateway` PASS

- [ ] `make ci` passes
