---
name: project_spotnik_feature30_complete
description: Feature 30 (API Gateway): Gateway struct, token bucket, dedup, 429 backoff, BaseClient integration, Store throttle observability
type: project
---

## Feature 30 — API Gateway

**Built:**
- `internal/api/gateway.go` — Gateway infra (tokenBucket, Gateway struct, Priority type, context helpers)
- `internal/api/gateway_test.go` — 19 tests, all gateway behaviors
- BaseClient: optional `*Gateway` field, `SetGateway()` method, `doJSON`/`doNoContent` route thru it
- 6 API clients get `SetGateway(a.gateway)` in `initAPIClients()`
- `App.New()` creates single shared `*Gateway`
- Store throttle observability: `SetThrottle()`, `IsThrottled()`, `ThrottleRetryAfterSecs()`, `ThrottleLast429At()`
- `throttleExpiredMsg` in app.go clears store throttle post-backoff
- `RateLimitedMsg` handler also calls `store.SetThrottle()`
- Interactive priority on user cmds via `api.WithPriority(ctx, api.Interactive)`

**Key files:**
- `internal/api/gateway.go` — tokenBucket, RequestKey, inflightEntry, Gateway, Priority, WithPriority, PriorityFromContext
- `internal/api/base.go` — SetGateway() method, gateway routing in doJSON/doNoContent
- `internal/app/auth.go` — initAPIClients() calls player.SetGateway(a.gateway) etc.
- `internal/app/app.go` — gateway field, New() creates Gateway, throttleExpiredMsg type+handler
- `internal/app/commands.go` — Interactive priority on play/pause/search/queue/like/playlist cmds
- `internal/state/store.go` — throttle struct field + 4 accessor methods

**Patterns:**
- Gateway created once in `App.New()`, shared across clients
- `SetGateway()` pointer method on `*BaseClient`, promoted via embedding to `*Player`, `*LibraryClient`, etc. — no explicit forwarders
- Priority context: `api.WithPriority(context.Background(), api.Interactive)` in cmd closures
- Store throttle state set in app.go `RateLimitedMsg` handler + cleared by `throttleExpiredMsg`
- 429 from gateway: returned as `*RateLimitError` from `gateway.Do()`, wrapped "sending request: %w" by doJSON/doNoContent — `errors.As` in `parse429RetryAfter` unwraps fine

**Gotchas:**
- Double body read worry: gateway buffers body, swaps resp.Body w/ NopCloser(bytes.NewReader(body)). doJSON reads NopCloser — no double-read
- Semaphore + dedup: dedup-waiting goroutine holds semaphore slot. Bounded (resolves when primary done), no deadlock
- `WithPriority` 0% coverage initially — self-review caught, test added
- Store throttle methods 0% coverage — self-review caught, tests added (state 93.7% → 99.6%)
- `time.Sleep(20ms)` for goroutine scheduling needed for dedup/semaphore concurrency tests — tolerable, short

**Testing:**
- 19 new tests gateway_test.go + 3 base_test.go + 3 store_test.go
- Coverage: 82.1% (api 83.9%, state 99.6%)
- Concurrency tests: channels + release patterns — no shared mutable callCount sans wg.Wait sync
- `TestGateway_MaxConcurrentRequests` bug initially: 5 goroutines shared key "/hold", dedup merged them — fixed w/ unique "/hold/i" keys