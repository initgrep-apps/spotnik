---
title: "Remove Gateway Debounce"
feature: 26-playback-correctness
status: done
---

## Background

`interactiveDebounce` in `gateway_dedup.go` applies a 100ms hold window to all
Interactive requests, keyed by API path. A new arrival for the same path
cancels the previous one and starts a fresh 100ms timer. Only the last request
in any burst window proceeds.

For volume up/down, both share path `/v1/me/player/volume`. Three presses at
60ms intervals:

```
t=0ms:   PUT #1 enters debounce hold
t=60ms:  PUT #2 cancels PUT #1, starts fresh 100ms hold
t=120ms: PUT #3 cancels PUT #2, starts fresh 100ms hold
t=220ms: PUT #3 fires (the only one that survives)
```

Only PUT #3 fires. PUT #1 and #2 are silently dropped. For playback controls
each press is a semantically independent command that must fire.

**Why it is safe to remove:**

Search (`GET /v1/search`) — the only request that needed last-wins behaviour —
has two independent upstream protection layers:

1. `scheduleDebounce` in `SearchPane` (300ms UI hold): only one
   `SearchRequestMsg` fires after 300ms of typing silence.
2. `searchCancel()` in `handlers.go`: cancels the previous in-flight HTTP
   context when a new `SearchRequestMsg` is handled. Even if two search GETs
   reach the gateway simultaneously, the first has a cancelled context and
   returns `context.Canceled`, which the command turns into a nil message that
   BubbleTea drops silently.

The gateway debounce was a third redundant layer for search. Its removal has
zero impact on search correctness.

**Depends on:** nothing — self-contained deletion within `internal/api/`.

## Design

### Code removed

**`internal/api/gateway_dedup.go`:**
- Delete `interactiveDebounceEntry` type (with `cancel` and `ready` fields)
- Delete `func (g *Gateway) interactiveDebounce(ctx context.Context, path string) error`

**`internal/api/gateway.go`:**
- Remove `debounceMu sync.Mutex` from `Gateway` struct
- Remove `debounceEntries map[string]*interactiveDebounceEntry` from `Gateway` struct
- Remove `debounceEntries: make(map[string]*interactiveDebounceEntry)` from `NewGateway()`
- Delete Phase 1b block from `Do()`:
  ```go
  // DELETE this entire block:
  if priority == Interactive {
      if err := g.interactiveDebounce(ctx, key.Path); err != nil {
          return nil, err
      }
  }
  ```
- Update the `Gateway` doc comment: remove the bullet "100ms transport-layer
  debounce for Interactive requests (path-keyed)"

**`internal/api/gateway_debounce_test.go`:**
- Delete the entire file (tests for removed feature)

### What is NOT changed

- `scheduleDebounce` in `SearchPane` — unchanged
- `searchCancel()` in `handlers.go` — unchanged
- Token bucket, 429 backoff, concurrency semaphore — unchanged
- Background dedup — unchanged

## Acceptance Criteria

- [ ] All playback command PUTs/POSTs (`/v1/me/player/pause`, `/v1/me/player/play`,
  `/v1/me/player/next`, `/v1/me/player/previous`, `/v1/me/player/volume`,
  `/v1/me/player/shuffle`, `/v1/me/player/repeat`) fire immediately with no
  100ms gateway hold.
- [ ] Search behaviour is unchanged — last query wins, stale results are dropped.
- [ ] `make ci` passes.

## Tasks

- [ ] Delete `internal/api/gateway_debounce_test.go`
  - test: `go test ./internal/api/...` → some debounce tests gone; remaining pass
- [ ] Delete `interactiveDebounceEntry` type from `internal/api/gateway_dedup.go`
  - test: `go build ./internal/api/...` → compile error from `gateway.go` use of the type
- [ ] Delete `func (g *Gateway) interactiveDebounce(...)` from `internal/api/gateway_dedup.go`
  - test: `go build ./internal/api/...` → compile error from Phase 1b call site in `gateway.go`
- [ ] Remove `debounceMu`, `debounceEntries` fields from `Gateway` struct in `gateway.go`;
  remove `debounceEntries: make(...)` from `NewGateway()`; delete Phase 1b block from `Do()`;
  update `Gateway` doc comment
  - test: `go build ./internal/api/...` → compiles cleanly
- [ ] `make ci` passes
