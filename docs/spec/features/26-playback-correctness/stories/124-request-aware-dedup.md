---
title: "Request-Aware Dedup"
feature: 26-playback-correctness
status: open
---

## Background

`fetchPlaybackStateCmd` uses `context.Background()` — no priority set — so the
gateway defaults to Background priority. The reconcile GET that fires after a
command can therefore join an in-flight Background poll and share its response
body.

That poll was initiated *before* the command was sent. It carries pre-command
Spotify state.

```
t=0ms:   Background poll tick fires GET /v1/me/player → registered in inflight map
t=100ms: PUT /v1/me/player/volume?volume_percent=21 returns 204 → PlaybackCmdSentMsg
          → reconcile fetchPlaybackStateCmd dispatched (Background priority)
          → gateway: key {GET, /v1/me/player} already in inflight map
          → reconcile JOINS as waiter — no new HTTP call fires
t=250ms: Polling GET HTTP response arrives (vol=20, pre-command)
          → both poll and reconcile receive vol=20
          → store.vol=20   ← WRONG
t=3000ms: next regular poll returns vol=21 → finally correct
```

The fix is two-pronged:

1. **Gateway**: add `Priority` to `RequestKey` and gate Phase 2 (inflight check)
   and Phase 4 (inflight registration) on `priority == Background`. Interactive
   GETs skip the inflight map entirely — they never check for an existing entry
   and never register themselves.

2. **App**: `fetchPlaybackStateCmd` accepts a `priority api.Priority` argument.
   Call sites on the success path pass `api.Interactive`; all other sites pass
   `api.Background`.

**Depends on:** nothing — self-contained within `internal/api/` and
`internal/app/`.

## Design

### `internal/api/gateway.go` — `RequestKey` gains `Priority`

```go
// Before
type RequestKey struct {
    Method string
    Path   string
}

// After
type RequestKey struct {
    Method   string
    Path     string
    Priority Priority
}
```

The `Priority` field makes the key semantically complete. However, Interactive
requests also skip the inflight map entirely — so only
`{GET, path, Background}` entries ever exist in practice.

### `internal/api/gateway.go` — gate Phase 2 + Phase 4 on Background

Phase 2 (dedup check) and Phase 4 (inflight registration) both gain a priority
guard. The existing join-as-waiter and register-and-defer logic is unchanged
inside the guard.

```go
// Phase 2: dedup check — Background only
if key.Method == http.MethodGet && priority == Background {
    g.mu.Lock()
    if entry, ok := g.inflight[key]; ok {
        // join as waiter — existing logic unchanged
        ...
    }
    g.mu.Unlock()
}

// Phase 4: inflight registration — Background only
if key.Method == http.MethodGet && priority == Background {
    // double-check + register — existing logic unchanged
    ...
}
```

Interactive GETs fall through both guards and proceed directly to the
semaphore and HTTP call.

### `internal/api/base.go` — populate `Priority` in all three `RequestKey` constructions

`doJSON`, `doJSONOptional`, and `doNoContent` each build a `RequestKey`.
Add the `Priority` field in all three:

```go
// Before
key := RequestKey{Method: req.Method, Path: req.URL.Path}

// After
key := RequestKey{
    Method:   req.Method,
    Path:     req.URL.Path,
    Priority: PriorityFromContext(req.Context()),
}
```

### `internal/app/commands.go` — `fetchPlaybackStateCmd` accepts priority

```go
// Before
func fetchPlaybackStateCmd(player api.PlayerAPI) tea.Cmd {
    return func() tea.Msg {
        ...
        ps, err := player.PlaybackState(context.Background())
        ...
    }
}

// After
func fetchPlaybackStateCmd(player api.PlayerAPI, priority api.Priority) tea.Cmd {
    return func() tea.Msg {
        ...
        ctx := api.WithPriority(context.Background(), priority)
        ps, err := player.PlaybackState(ctx)
        ...
    }
}
```

### `internal/app/handlers.go` — call-site priorities

Every existing call to `fetchPlaybackStateCmd(a.player)` gains a priority
argument. The table below shows all call sites:

| Handler | Location | Priority | Reason |
|---------|----------|----------|--------|
| `PlaybackCmdSentMsg` success | `handlers.go:517` | `api.Interactive` | User command — needs fresh response |
| `PlaybackCmdSentMsg` ForbiddenError | `handlers.go:508` | `api.Background` | Error recovery — not user-visible fresh state needed |
| `PlaybackCmdSentMsg` generic error | `handlers.go:513` | `api.Background` | Error recovery |
| `DeviceTransferredMsg` success | `handlers.go:885` | `api.Interactive` | User action — needs fresh device state |
| `DeviceTransferredMsg` error | `handlers.go:880` | `api.Background` | Error recovery |
| TickMsg polling (backoff expired) | `handlers.go:347` | `api.Background` | Regular poll |
| TickMsg polling (interval tick) | `handlers.go:360` | `api.Background` | Regular poll |
| AuthMsg (initial fetch) | `handlers.go:83` | `api.Background` | Startup fetch — no prior command |
| `app.Init()` | `app.go:759` | `api.Background` | App startup |

**Post-fix dedup matrix (for reference):**

| In-flight priority \ Arriving priority | Background | Interactive |
|---------------------------------------|------------|-------------|
| **Background** | Join (correct — existing) | Don't join — Interactive fires fresh |
| **Interactive** | Both fire independently | Both fire independently |

## Acceptance Criteria

- [ ] `GET /v1/me/player` fired from `PlaybackCmdSentMsg` success path does not join an in-flight Background poll for the same path.
- [ ] `GET /v1/me/player` fired from `DeviceTransferredMsg` success path does not join an in-flight Background poll.
- [ ] Background polls still deduplicate with other Background polls (existing behaviour preserved).
- [ ] Interactive GET requests never join any in-flight request — Background or Interactive.
- [ ] `make ci` passes.

## Tasks

- [ ] Add `Priority Priority` field to `RequestKey` in `internal/api/gateway.go`
  - Compile check: `go build ./internal/api/...` → missing-field errors on all `RequestKey` literals
- [ ] Populate `Priority` in all three `RequestKey` constructions in `internal/api/base.go`
  (`doJSON`, `doJSONOptional`, `doNoContent`)
  - test: `go build ./internal/api/...` → compiles cleanly
- [ ] Gate Phase 2 (dedup check) on `priority == Background` in `gateway.go Do()`
  - Move the existing `if key.Method == http.MethodGet { ... }` check inside
    `if key.Method == http.MethodGet && priority == Background { ... }`
  - test: `go test ./internal/api/... -run TestDedup` → FAIL (new tests not yet written)
- [ ] Gate Phase 4 (inflight registration) on `priority == Background` in `gateway.go Do()`
  - Same guard: `if key.Method == http.MethodGet && priority == Background { ... }`
  - test: `go build ./internal/api/...` → compiles cleanly
- [ ] Update all `RequestKey` literals in `internal/api/gateway_test.go` —
  add `Priority: Background` to all existing Background test keys. Interactive
  test keys that exist (e.g. lines ~340, ~826) add `Priority: Interactive`.
  - test: `go test ./internal/api/... -run TestGateway` → PASS (existing tests unaffected)
- [ ] Update all `RequestKey` literals in `internal/api/gateway_hardening_test.go`
  — add `Priority: Background` to all existing keys.
  - test: `go test ./internal/api/... -run TestGatewayHardening` → PASS
- [ ] Add `TestDedup_InteractiveDoesNotJoinBackground` to `gateway_test.go`:
  - Register a Background GET in the inflight map (hold HTTP with a `chan struct{}`).
  - Fire an Interactive GET to the same path concurrently.
  - Assert Interactive GET fires its own HTTP call independently (use a counter or channel).
  - Release the hold; assert both receive independent responses.
  - test: `go test ./internal/api/... -run TestDedup_InteractiveDoesNotJoinBackground` → PASS
- [ ] Add `TestDedup_InteractiveDoesNotJoinInteractive` to `gateway_test.go`:
  - Fire two Interactive GETs to the same path concurrently.
  - Assert both fire independent HTTP calls (counter == 2).
  - test: `go test ./internal/api/... -run TestDedup_InteractiveDoesNotJoinInteractive` → PASS
- [ ] Add `TestDedup_BackgroundJoinsBackground` to `gateway_test.go` (preserves existing behaviour):
  - Fire two Background GETs to the same path concurrently.
  - Assert only one HTTP call fires (counter == 1); both receive the same body.
  - test: `go test ./internal/api/... -run TestDedup_BackgroundJoinsBackground` → PASS
- [ ] Update `fetchPlaybackStateCmd` signature in `internal/app/commands.go`
  to accept `priority api.Priority`; use `api.WithPriority(context.Background(), priority)`
  as the context passed to `player.PlaybackState`.
  - test: `go build ./internal/app/...` → compile errors on all call sites
- [ ] Update all `fetchPlaybackStateCmd` call sites in `internal/app/handlers.go`
  and `internal/app/app.go` with the priorities from the table above.
  - test: `go build ./internal/app/...` → compiles cleanly
- [ ] `make ci` passes
