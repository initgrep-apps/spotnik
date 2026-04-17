---
title: "Centralized API Gateway"
feature: 11-api-gateway
status: done
---

## Background
All API requests fired directly to Spotify through `BaseClient.doJSON` / `doNoContent` with no throttling, dedup, concurrency cap, or priority. A burst of user actions plus polling could trigger rate limiting. There was no single control point for all HTTP traffic. This story introduced a centralized API gateway that controls all outbound HTTP traffic with token-bucket rate limiting, concurrency capping, in-flight request deduplication, priority classification (Interactive vs Background), and 429 backoff.

Gap reference: G2, G8, G9 in `docs/superpowers/plans/2026-03-24-architecture-baseline.md`

## Design

### Token Bucket Rate Limiter
`tokenBucket` struct: 10 tokens/second, burst of 10. `wait()` refills tokens based on elapsed time, then either returns immediately or blocks until a token is available or ctx is cancelled.

### Gateway Struct
```go
type Gateway struct {
    mu           sync.Mutex
    bucket       *tokenBucket
    semaphore    chan struct{}     // concurrency limiter (buffered channel, size 5)
    inflight     map[RequestKey]*inflightEntry
    backoffUntil time.Time
    retryAfter   int
}
```

### In-Flight Request Dedup
Before executing `fn` in `Do()`, check if `key` exists in `inflight` map. If yes, wait on its `done` channel and return the cached result (clone the response body). Response body buffering: read body into `[]byte`, store in entry, create new `io.ReadCloser` for each waiter from the buffer.

### Priority and 429 Backoff
```go
type Priority int
const (
    Background  Priority = iota
    Interactive
)
```
After 429: background requests rejected (return RateLimitError), interactive requests wait until backoff expires. Interactive requests skip the token bucket wait.

### Integration into BaseClient
BaseClient gets optional `gateway *Gateway` field. `doJSON`/`doNoContent` route through `gateway.Do()` when attached. Priority passed via `context.WithValue` with a package-private key.

### Verification
```bash
grep -r 'b.http.Do(' internal/api/base.go
# Expected: only inside gateway.Do's fn callback or when gateway is nil
make ci
```

## Acceptance Criteria
- [ ] All API calls route through the gateway (or bypass only when gateway is nil for backwards compat)
- [ ] Token bucket limits requests to 10/second with burst of 10
- [ ] Max 5 concurrent in-flight requests
- [ ] Duplicate in-flight requests are deduplicated (same method+path -> one HTTP call)
- [ ] 429 responses trigger global backoff; background requests rejected, interactive requests wait
- [ ] Interactive requests bypass token bucket
- [ ] `IsThrottled()` and `RetryAfterSecs()` expose gateway state for UI observability
- [ ] `make ci` passes

## Tasks
- [ ] Token bucket rate limiter -- create tokenBucket struct in internal/api/gateway.go
      - test: token bucket allows burst up to max; blocks when empty; respects context cancellation
- [ ] Concurrency limiter + Gateway struct -- add semaphore-based concurrency limiting
      - test: max 5 concurrent requests; 6th blocks until one completes; context cancellation
- [ ] In-flight request dedup -- check inflight map before executing fn
      - test: two concurrent requests with same key -> only one HTTP call; different keys execute independently; error result shared
- [ ] 429 backoff + priority bypass -- add Priority type and backoff/priority logic
      - test: after 429, background requests rejected; interactive requests wait; interactive bypass token bucket; IsThrottled() correct
- [ ] Integration into BaseClient + Store + docs -- wire gateway into all existing API infrastructure
      - test: all API calls go through gateway; BaseClient with gateway routes through Do(); BaseClient without gateway works as before
