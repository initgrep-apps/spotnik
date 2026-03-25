---
name: project_spotnik_feature30_complete
description: Feature 30 (API Gateway): Gateway struct, token bucket, dedup, 429 backoff, BaseClient integration, Store throttle observability
type: project
---

## Feature 30 — API Gateway

**What was built:**
- `internal/api/gateway.go` — complete Gateway infrastructure (tokenBucket, Gateway struct, Priority type, context helpers)
- `internal/api/gateway_test.go` — 19 tests covering all gateway behaviors
- BaseClient gains optional `*Gateway` field, `SetGateway()` method, `doJSON`/`doNoContent` route through it
- All 6 API clients get `SetGateway(a.gateway)` in `initAPIClients()`
- `App.New()` creates a single shared `*Gateway`
- Store gains throttle observability: `SetThrottle()`, `IsThrottled()`, `ThrottleRetryAfterSecs()`, `ThrottleLast429At()`
- `throttleExpiredMsg` in app.go clears store throttle state after backoff expires
- `RateLimitedMsg` handler updated to also call `store.SetThrottle()`
- Interactive priority set on all user-triggered commands via `api.WithPriority(ctx, api.Interactive)`

**Key files:**
- `internal/api/gateway.go` — tokenBucket, RequestKey, inflightEntry, Gateway, Priority, WithPriority, PriorityFromContext
- `internal/api/base.go` — SetGateway() method, gateway routing in doJSON/doNoContent
- `internal/app/auth.go` — initAPIClients() calls player.SetGateway(a.gateway) etc.
- `internal/app/app.go` — gateway field, New() creates Gateway, throttleExpiredMsg type+handler
- `internal/app/commands.go` — Interactive priority on play/pause/search/queue/like/playlist cmds
- `internal/state/store.go` — throttle struct field + 4 accessor methods

**Patterns established:**
- Gateway is created once in `App.New()`, shared across all clients
- `SetGateway()` is a pointer method on `*BaseClient`, promoted automatically through embedding to `*Player`, `*LibraryClient`, etc. — no need for explicit forwarding methods
- Priority context: `api.WithPriority(context.Background(), api.Interactive)` in command closures
- Store throttle state updated in app.go `RateLimitedMsg` handler + cleared by `throttleExpiredMsg`
- 429 from gateway: returned as `*RateLimitError` from `gateway.Do()`, wrapped in "sending request: %w" by doJSON/doNoContent — `errors.As` in `parse429RetryAfter` unwraps correctly

**Gotchas:**
- Double body reading concern: gateway buffers body, replaces resp.Body with NopCloser(bytes.NewReader(body)). doJSON then reads from the NopCloser — no double-read problem
- Semaphore + dedup interaction: a dedup-waiting goroutine holds a semaphore slot. This is bounded (resolves when primary completes) and doesn't deadlock
- `WithPriority` had 0% coverage in initial implementation — caught in self-review, added test
- Store throttle methods had 0% coverage — caught in self-review, added tests (state coverage 93.7% → 99.6%)
- Tests using `time.Sleep(20ms)` for goroutine scheduling are necessary for dedup and semaphore concurrency tests — tolerable given the short duration

**Testing notes:**
- 19 new tests in gateway_test.go + 3 in base_test.go + 3 in store_test.go
- Total coverage: 82.1% (api 83.9%, state 99.6%)
- Concurrency tests use channels and release patterns — avoid shared mutable callCount without wg.Wait synchronization
- `TestGateway_MaxConcurrentRequests` had a bug initially: all 5 goroutines used same key "/hold" causing dedup to merge them — fixed by using unique "/hold/i" keys
