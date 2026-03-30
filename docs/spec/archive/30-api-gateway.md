# Feature 30 — API Gateway

> **Feature:** Introduce a centralized API gateway that controls all outbound HTTP
> traffic to Spotify. Provides rate limiting, concurrency capping, request dedup,
> priority classification, and 429 backoff.

## Context

All API requests currently fire directly to Spotify through `BaseClient.doJSON` /
`doNoContent` (internal/api/base.go lines 73-107) with no throttling, dedup,
concurrency cap, or priority. A burst of user actions + polling can trigger rate
limiting. There's no single control point for all HTTP traffic.

**Gap reference:** G2, G8, G9 in `docs/superpowers/plans/2026-03-24-architecture-baseline.md`

**Depends on:** Nothing — can run in parallel with Feature 29.

---

## Task 1: Token bucket rate limiter

**Problem:** No global rate cap on outbound requests.

**Fix:**

1. Create `internal/api/gateway.go` with a `tokenBucket` struct:
   ```go
   type tokenBucket struct {
       mu       sync.Mutex
       tokens   float64
       max      float64
       rate     float64     // tokens per second
       lastFill time.Time
   }

   func (tb *tokenBucket) wait(ctx context.Context) error
   ```

2. `wait()` refills tokens based on elapsed time, then either returns immediately
   (tokens available) or blocks until a token is available or ctx is cancelled.

3. Default: 10 tokens/second, burst of 10.

**Files:**
- Create: `internal/api/gateway.go`
- Create: `internal/api/gateway_test.go`

**Tests:**
- Unit: token bucket allows burst up to max
- Unit: token bucket blocks when empty, unblocks after refill interval
- Unit: token bucket respects context cancellation

**Commit:** `feat(api): token bucket rate limiter for gateway`

---

## Task 2: Concurrency limiter + Gateway struct

**Problem:** Unlimited parallel in-flight requests can overwhelm the API.

**Fix:**

1. Add `Gateway` struct to `gateway.go`:
   ```go
   type Gateway struct {
       mu           sync.Mutex
       bucket       *tokenBucket
       semaphore    chan struct{}     // concurrency limiter (buffered channel, size 5)
       inflight     map[RequestKey]*inflightEntry
       backoffUntil time.Time
       retryAfter   int
   }

   type RequestKey struct {
       Method string
       Path   string
   }

   func NewGateway() *Gateway
   ```

2. Semaphore: buffered channel of size 5. Acquire before request, release after.

3. `Do()` method signature:
   ```go
   func (g *Gateway) Do(ctx context.Context, priority Priority, key RequestKey,
       fn func() (*http.Response, error)) (*http.Response, error)
   ```

**Tests:**
- Unit: max 5 concurrent requests; 6th blocks until one completes
- Unit: semaphore respects context cancellation

**Commit:** `feat(api): concurrency limiter and Gateway struct`

---

## Task 3: In-flight request dedup

**Problem:** Duplicate in-flight requests are possible (tick fires while a previous
fetch is still pending).

**Fix:**

1. Add `inflightEntry` struct:
   ```go
   type inflightEntry struct {
       done chan struct{}
       resp *http.Response
       body []byte
       err  error
   }
   ```

2. In `Do()`: before executing `fn`, check if `key` exists in `inflight` map.
   If yes, wait on its `done` channel and return the cached result (clone the
   response body). If no, add entry, execute `fn`, buffer the response body,
   broadcast to waiters, clean up.

3. Response body buffering: read body into `[]byte`, store in entry, create
   new `io.ReadCloser` for each waiter from the buffer.

**Tests:**
- Unit: two concurrent requests with same key → only one HTTP call, both get result
- Unit: different keys execute independently
- Unit: error result is shared with waiters too

**Commit:** `feat(api): in-flight request dedup in gateway`

---

## Task 4: 429 backoff + priority bypass

**Problem:** 429 backoff is currently handled per-command in `app.go`. No centralized
backoff. Interactive and background requests are treated identically.

**Fix:**

1. Add `Priority` type:
   ```go
   type Priority int

   const (
       Background  Priority = iota // polling, prefetch
       Interactive                  // user-initiated actions
   )
   ```

2. In `Do()`:
   - Check `backoffUntil`: if `time.Now().Before(g.backoffUntil)`, block background
     requests (return a `RateLimitError`); interactive requests wait until backoff expires.
   - On 429 response: parse `Retry-After`, set `g.backoffUntil`, return error.
   - Interactive requests skip the token bucket wait (acquire immediately).
   - Background requests go through normal token bucket flow.

3. Expose `IsThrottled() bool` and `RetryAfterSecs() int` for UI observability.

**Tests:**
- Unit: after 429, background requests are rejected until backoff expires
- Unit: interactive requests wait during backoff but eventually proceed
- Unit: interactive requests bypass token bucket
- Unit: `IsThrottled()` returns correct state

**Commit:** `feat(api): 429 backoff and priority bypass in gateway`

---

## Task 5: Integration into BaseClient + Store + docs

**Problem:** Gateway exists but nothing uses it yet.

**Fix:**

1. Modify `BaseClient` (internal/api/base.go):
   - Add optional `gateway *Gateway` field
   - `NewBaseClientWithProvider` accepts optional `Gateway`
   - `doJSON` and `doNoContent` route through `gateway.Do()` when gateway is set
   - Construct `RequestKey` from the request method + path
   - Default priority: `Background` (callers can override via context or param)

2. Add priority passing mechanism:
   - Use `context.WithValue` with a package-private key to carry `Priority`
   - Command builders set `Interactive` for user-triggered actions, `Background` for polling

3. Modify `internal/state/store.go`:
   - Add throttle observability fields: `IsThrottled bool`, `RetryAfterSecs int`, `Last429At time.Time`
   - Gateway updates these via a callback or direct reference

4. Modify `internal/app/app.go`:
   - Create `Gateway` in `New()`, pass to `BaseClient` constructors
   - Remove duplicate 429/backoff handling from `parse429RetryAfter` where gateway handles it
   - Keep `RateLimitedMsg` for cases where gateway surfaces the error to the Cmd

5. Update docs:
   - **`docs/ARCHITECTURE.md`**: New section "API Gateway" documenting rate limiting, dedup, priority, backoff
   - **`CLAUDE.md`** → "API Rules": Add "All requests go through the API Gateway"

**Files:**
- Modify: `internal/api/base.go` — add gateway integration
- Modify: `internal/api/gateway.go` — add priority context, observability
- Modify: `internal/state/store.go` — add throttle fields
- Modify: `internal/app/app.go` — create gateway, pass to clients
- Modify: `internal/app/commands.go` — set priority on contexts
- Modify: `docs/ARCHITECTURE.md` — add Gateway section
- Modify: `CLAUDE.md` — add Gateway rule

**Tests:**
- Integration: all API calls go through gateway (verify with httptest server + request counting)
- Unit: BaseClient with gateway routes through Do()
- Unit: BaseClient without gateway works as before (backwards compat)

**Commit 1:** `feat(api): integrate gateway into BaseClient and all API calls`
**Commit 2:** `docs: add API Gateway documentation`

---

## Verification

```bash
# All API calls should go through the gateway
grep -r 'b.http.Do(' internal/api/base.go
# Expected: only inside gateway.Do's fn callback or when gateway is nil

make ci
# Expected: Full pass
```

---

*Depends on: None*
*Blocked by: Nothing*
*Can run in parallel with: Feature 29*
