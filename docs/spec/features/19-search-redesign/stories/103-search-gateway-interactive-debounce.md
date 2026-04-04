---
title: "Search Redesign: Gateway Interactive debounce (100ms path-keyed)"
feature: 19-search-redesign
status: open
---

## Background

The BubbleTea 300ms debounce (story 99) is the primary guard. However, it operates inside
the Elm Update loop and can be bypassed by rapid page flips that each fire exactly one tick
300ms apart. The Gateway needs an independent transport-layer backstop.

The requirement: **all `api.Interactive` requests for the same API path are debounced at
the gateway — only the last request within a 100ms window proceeds.** This applies to
search, device switching, playback controls — any user-triggered call. `Background`
requests (polling, prefetch) are unaffected.

The gateway already has in-flight dedup (`inflight map[RequestKey]*inflightEntry`). The
new debounce phase is separate and comes before dedup in the `Do()` pipeline.

## Architecture Context

### Layer: Gateway — transport backstop (independent of BubbleTea layer)

The two debounce layers are independent. They do not coordinate. Each enforces last-wins
on its own:

```
User action → scheduleDebounce (300ms, BubbleTea) → SearchRequestMsg
  │
  ▼
app.go → buildSearchPageCmd(ctx, ...) → Gateway.Do(ctx, Interactive, path, fn)
                                          │
                                          ▼
                              interactiveDebounce(ctx, path)    ← THIS STORY
                                │
                                ├── hold 100ms
                                │   new Interactive request for same path?
                                │   → cancel this hold, return error
                                │
                                └── 100ms elapsed → proceed to HTTP
                                          │
                                          ▼
                                    in-flight dedup (existing)
                                          │
                                          ▼
                                    Spotify API
```

### Why two independent layers?

| Scenario | BubbleTea layer handles | Gateway layer handles |
|---|---|---|
| Normal typing | ✓ 300ms debounce | backup, usually no-op |
| Rapid page flips (one per 300ms) | ✗ each fires | ✓ gateway cancels all but last |
| Test environments (fast ticks) | may be bypassed | ✓ always enforced |
| Direct API calls (no Update loop) | ✗ not applicable | ✓ enforced |

### Scope and keying

- **Applies to:** all requests with `key.Priority == Interactive`
- **Key:** `key.Path` only — query parameters are ignored. All `/v1/search` requests
  (regardless of `q=jazz` vs `q=rock`) share one debounce slot.
- **`Background` requests:** pass through `Do()` without touching `debounceEntries`

### `wrappedCtx` design note

`interactiveDebounce` creates a `wrappedCtx` from the incoming `ctx`. This is used solely
to make the 100ms hold cancellable by a competing request (via `prev.cancel()`). The
`wrappedCtx` is **not** threaded into the HTTP call — after the hold completes, `Do()`
uses the original caller `ctx` for the actual HTTP request. This is correct because:
1. The caller's `ctx` already propagates cancellation from `app.go` (story 100)
2. Threading `wrappedCtx` into the HTTP call would not add cancellation coverage that
   `ctx` doesn't already provide
3. Keeping the gate as a separate function (`interactiveDebounce`) makes it easy to test
   in isolation

## Design

### New types in `internal/api/gateway.go`

```go
// interactiveDebounceEntry holds the cancel and ready channel for one pending
// Interactive request. cancel stops the 100ms wait; ready is closed when the
// entry exits (used by the replacement to wait before registering itself).
type interactiveDebounceEntry struct {
    cancel context.CancelFunc
    ready  chan struct{}
}
```

### New fields on `Gateway`

```go
debounceMu      sync.Mutex
debounceEntries map[string]*interactiveDebounceEntry
```

Initialize in `NewGateway()` (or equivalent constructor):
```go
g.debounceEntries = make(map[string]*interactiveDebounceEntry)
```

### New phase in `Gateway.Do()` — for Interactive only

Insert between the rate-limit policy phase and the in-flight dedup phase:

```go
if key.Priority == Interactive {
    if err := g.interactiveDebounce(ctx, key.Path); err != nil {
        return err
    }
}
```

### `interactiveDebounce` method

```go
// interactiveDebounce implements a 100ms hold window for Interactive requests
// keyed by API path (query params ignored). If a newer request for the same
// path arrives within 100ms, the older one is cancelled and returns an error.
// Only the last request in any burst window proceeds.
func (g *Gateway) interactiveDebounce(ctx context.Context, path string) error {
    // wrappedCtx allows a competing request to cancel this hold without
    // cancelling the caller's ctx (which would affect the HTTP call too).
    wrappedCtx, wrappedCancel := context.WithCancel(ctx)

    g.debounceMu.Lock()
    if prev, ok := g.debounceEntries[path]; ok {
        // Cancel the prior request's hold and wait for it to finish unregistering.
        prev.cancel()
        g.debounceMu.Unlock()
        <-prev.ready
        g.debounceMu.Lock()
    }
    entry := &interactiveDebounceEntry{
        cancel: wrappedCancel,
        ready:  make(chan struct{}),
    }
    g.debounceEntries[path] = entry
    g.debounceMu.Unlock()

    // Cleanup: remove from map and signal ready when we exit.
    defer func() {
        wrappedCancel()
        g.debounceMu.Lock()
        if g.debounceEntries[path] == entry {
            delete(g.debounceEntries, path)
        }
        g.debounceMu.Unlock()
        close(entry.ready)
    }()

    // Hold for 100ms. The first request to survive the full hold proceeds.
    select {
    case <-time.After(100 * time.Millisecond):
        return nil // proceed to HTTP
    case <-wrappedCtx.Done():
        return wrappedCtx.Err() // cancelled by newer request for same path
    case <-ctx.Done():
        return ctx.Err() // cancelled by caller (Esc, new query from app.go)
    }
}
```

### Interaction with existing in-flight dedup

The debounce phase runs **before** in-flight dedup. After the 100ms hold, the request
enters the existing dedup logic unchanged. The two mechanisms are independent and do not
coordinate.

## Acceptance Criteria

- [ ] `Gateway` has `debounceMu` and `debounceEntries` fields; initialized in constructor
- [ ] `interactiveDebounce(ctx, path)` implements 100ms hold with cancel-and-wait on prior entry
- [ ] Second `Interactive` request for same path within 100ms cancels first; first returns error
- [ ] Requests for different paths are independent (different debounce slots)
- [ ] After 100ms with no replacement, request proceeds normally
- [ ] `Background` requests bypass the debounce phase entirely
- [ ] Caller-cancelled context (`ctx.Done()`) returns immediately without blocking
- [ ] No goroutine leaks: entry removed from map after exit; ready channel closed
- [ ] `make ci` passes; no data races under `-race`

## Tasks

- [ ] Add `interactiveDebounceEntry` type and `debounceMu`/`debounceEntries` fields to `Gateway`;
      initialize in constructor
      - test: `NewGateway()` initializes `debounceEntries` as non-nil map

- [ ] Implement `interactiveDebounce(ctx context.Context, path string) error`
      - test: single request → waits 100ms → returns nil
      - test: two requests same path within 100ms → first returns error; second proceeds after 100ms
      - test: two requests different paths → both proceed independently (no interference)
      - test: caller ctx cancelled during hold → returns immediately with ctx error
      - test: no goroutine leak — entry removed from map after exit; ready channel closed

- [ ] Integrate `interactiveDebounce` into `Gateway.Do()` — Interactive only, before in-flight dedup
      - test: Background priority request passes through without debounce delay (< 10ms overhead)
      - test: Interactive request experiences ~100ms hold before HTTP call

- [ ] Verify no data races: run `go test -race ./internal/api/...`
      - test: rapid concurrent Interactive requests for same path produce no race condition

- [ ] `make ci` passes
