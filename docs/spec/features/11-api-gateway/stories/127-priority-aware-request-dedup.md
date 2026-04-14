---
title: "Gateway: priority-aware request deduplication"
feature: 11-api-gateway
status: open
---

## Background

The gateway deduplicates in-flight GET requests using a `RequestKey{Method, Path}` key.
When two goroutines issue a `GET /v1/me/player` simultaneously, the second joins the
first as a waiter and both receive the same HTTP response — one HTTP call instead of two.

This was designed to prevent duplicate polling requests from wasting rate-limit budget.
It is correct for two Background polls racing each other, but it breaks down when a
user-triggered (Interactive) state fetch is coerced into sharing a stale Background poll
response.

---

## Problem Statement

### The Stale-Dedup Bug

When a playback command is executed (volume, pause/play, shuffle, repeat), the handler
fires a reconcile fetch — a `GET /v1/me/player` — to update the store with the
authoritative post-command Spotify state.

That reconcile fetch uses `context.Background()` → priority `Background`. It goes
through the gateway with key `{GET, /v1/me/player, <no priority distinction>}`.

If a Background polling GET for the same path is already in-flight at that moment, the
reconcile joins it as a dedup waiter. They share one HTTP response. That response was
initiated **before the command was sent** — it carries pre-command Spotify state
(e.g. vol=70 before a vol-down command).

Result: the reconcile fetch, which was supposed to confirm the command succeeded,
instead writes the pre-command state back to the store. The UI shows the wrong value
until the next regular poll corrects it ~1–3 seconds later.

### Exact Failure Timeline

```
T=0ms    User presses - (volume 70 → 69)
         PlaybackRequestMsg → buildPlaybackAPICmd dispatched
         PUT /v1/me/player/volume goroutine starts
         (Interactive priority, 100ms debounce hold)

T=200ms  TickMsg fires → fetchPlaybackStateCmd dispatched
         Background GET /v1/me/player registers in gateway inflight map
         HTTP GET in flight (captures vol=70 — command not yet applied)

T=100ms  PUT debounce expires → HTTP PUT fires
T=300ms  HTTP PUT returns 204 → PlaybackCmdSentMsg
         → reconcile fetchPlaybackStateCmd dispatched (Background)
         → gateway: GET /v1/me/player already in inflight map
         → reconcile JOINS as dedup waiter — NO new HTTP call fired

T=400ms  Polling GET HTTP response arrives (vol=70, pre-command)
         Both polling goroutine and reconcile goroutine receive vol=70
         → PlaybackStateFetchedMsg{vol=70} × 2 queued in Bubble Tea channel
         → Both write vol=70 to store
         → UI shows vol=70   ← WRONG — command succeeded but state reverted

T=3000ms Next regular poll → GET /v1/me/player → vol=69 (finally correct)
         → UI shows vol=69
```

The user sees: press `-` → (no optimistic update) → store shows 70 for ~3s → corrects to 69.

### Why Optimistic Updates Do Not Help Here

Optimistic updates (feature 26, now reverted) attempted to mask this by writing the
predicted state immediately and suppressing polling overwrites via a `playbackCmdPending`
counter. That counter was insufficient because:

1. A polling GET goroutine started while `pending=1` can complete at the same instant as
   the PUT, queuing its `PlaybackStateFetchedMsg` just after `PlaybackCmdSentMsg`. The
   counter drops to 0 in `PlaybackCmdSentMsg` processing, and the already-queued stale
   message writes through.

2. The reconcile fetch deduplicates with that same stale polling GET, so it also returns
   stale data and provides no correction.

The correct fix is at the gateway level: prevent Interactive and Background requests from
sharing dedup buckets, so a user-triggered reconcile fetch always fires a fresh HTTP call.

---

## Design

### Core Change: Priority Field in `RequestKey`

Add a `Priority` field to `RequestKey`. Dedup buckets are now keyed on
`{Method, Path, Priority}`. A Background `GET /v1/me/player` and an Interactive
`GET /v1/me/player` are separate entries in the inflight map and never share a response.

```go
// gateway.go — before
type RequestKey struct {
    Method string
    Path   string
}

// gateway.go — after
type RequestKey struct {
    Method   string
    Path     string
    Priority Priority
}
```

Background polls dedup with other Background polls (correct, unchanged behaviour).
Interactive fetches dedup with other Interactive fetches (correct for rapid commands).
Background and Interactive never dedup (the new invariant this story establishes).

### Reconcile Fetch Uses Interactive Priority

`fetchPlaybackStateCmd` currently calls `player.PlaybackState(context.Background())`.
The context carries no priority, so the gateway sees `Background`. Change the function
signature to accept a context (or a priority flag) so callers can pass
`api.WithPriority(ctx, api.Interactive)`.

The `PlaybackCmdSentMsg` success handler (the only reconcile call site) passes
`Interactive`. All other callers (tick handler, error recovery) continue to pass
`Background` — no behaviour change for polling.

### Debounce Behaviour for Interactive State Fetches

Interactive requests go through the 100ms `interactiveDebounce` hold (gateway_dedup.go).
This is desirable for state fetches after commands:

- **Single command**: request enters debounce, waits 100ms, fires. Total reconcile
  latency ≈ 100ms + HTTP roundtrip ≈ 300ms. Acceptable.
- **Rapid commands** (e.g. volume pressed 5× quickly): each command's
  `PlaybackCmdSentMsg` fires a new Interactive GET. Each new arrival cancels the
  prior debounce entry. Only the last one survives and fires. Natural batching of
  back-to-back reconcile fetches — one HTTP call for a burst of commands. Correct.

No changes to debounce logic are needed.

### Updated Flow After Fix

```
T=0ms    User presses - (volume 70 → 69)
         PlaybackRequestMsg → buildPlaybackAPICmd dispatched
         PUT /v1/me/player/volume (Interactive, 100ms debounce)

T=200ms  TickMsg → Background GET /v1/me/player fires
         inflight key: {GET, /v1/me/player, Background} → registered

T=100ms  PUT debounce expires → HTTP PUT fires
T=300ms  HTTP PUT returns 204 → PlaybackCmdSentMsg
         → reconcile fetch dispatched with Interactive priority
         → gateway: key = {GET, /v1/me/player, Interactive}
         → NOT in inflight map (different key from Background poll)
         → 100ms debounce hold starts, then fresh HTTP GET fires

T=400ms  Background polling GET returns vol=70 → PlaybackStateFetchedMsg
         → written to store → UI shows vol=70 (briefly)

T=400ms  Reconcile debounce expires → fresh HTTP GET fires
T=600ms  Reconcile GET returns vol=69 (correct, command applied)
         → PlaybackStateFetchedMsg → store writes vol=69
         → UI shows vol=69   ✓

Total time from keypress to correct UI: ~600ms.
No stale dedup. No revert-then-correct cycle.
```

---

## Tasks

### Task 1 — Add Priority to RequestKey

**File:** `internal/api/gateway.go`

Change `RequestKey`:

```go
// RequestKey uniquely identifies a request for deduplication purposes.
// Two requests with the same Method, Path, and Priority are considered identical
// for dedup. Interactive and Background requests to the same endpoint are kept
// in separate dedup buckets so a user-triggered fetch never shares a response
// with a stale background poll.
type RequestKey struct {
    Method   string
    Path     string
    Priority Priority
}
```

No other changes to gateway.go — the inflight map is already `map[RequestKey]*inflightEntry`.
The new field participates in map equality automatically.

### Task 2 — Populate Priority in RequestKey construction sites

**File:** `internal/api/base.go`

Three functions build `RequestKey` from a request: `doJSON`, `doNoContent`, and
`doJSONOptional` (or equivalent). Each uses:

```go
key := RequestKey{Method: req.Method, Path: req.URL.Path}
```

Change each to:

```go
key := RequestKey{
    Method:   req.Method,
    Path:     req.URL.Path,
    Priority: PriorityFromContext(req.Context()),
}
```

`PriorityFromContext` already exists in `gateway.go` and reads the value set by
`api.WithPriority`. No new code needed — just thread the context through.

### Task 3 — Add priority parameter to fetchPlaybackStateCmd

**File:** `internal/app/commands.go`

Current signature:

```go
func fetchPlaybackStateCmd(player api.PlayerAPI) tea.Cmd {
    return func() tea.Msg {
        ps, err := player.PlaybackState(context.Background())
        ...
    }
}
```

Add a `priority` parameter:

```go
func fetchPlaybackStateCmd(player api.PlayerAPI, priority api.Priority) tea.Cmd {
    return func() tea.Msg {
        ctx := api.WithPriority(context.Background(), priority)
        ps, err := player.PlaybackState(ctx)
        ...
    }
}
```

### Task 4 — Update all call sites of fetchPlaybackStateCmd

**File:** `internal/app/handlers.go`

| Call site | Current | After |
|---|---|---|
| `PlaybackCmdSentMsg` success path | `fetchPlaybackStateCmd(a.player)` | `fetchPlaybackStateCmd(a.player, api.Interactive)` |
| `PlaybackCmdSentMsg` error path (both branches) | `fetchPlaybackStateCmd(a.player)` | `fetchPlaybackStateCmd(a.player, api.Background)` — error recovery is not user-triggered, use Background |
| `TickMsg` polling dispatch | `fetchPlaybackStateCmd(a.player)` | `fetchPlaybackStateCmd(a.player, api.Background)` |
| Any other recovery/retry fetch sites | `fetchPlaybackStateCmd(a.player)` | `fetchPlaybackStateCmd(a.player, api.Background)` |

Only the `PlaybackCmdSentMsg` success path uses `Interactive`. Everything else stays `Background`.

### Task 5 — Update gateway tests

**File:** `internal/api/gateway_test.go`, `internal/api/gateway_debounce_test.go`

All existing `RequestKey` literals need a `Priority` field. Most tests exercise
Background behaviour — add `Priority: Background` to existing keys. Tests that
specifically test Interactive paths already use `Interactive` priority in context;
update their keys accordingly.

Verify the existing dedup test at `gateway_test.go:642` (key `{GET, /v1/me/player}`)
still passes with `Priority: Background`. Add a new test:

```
TestDedup_InteractiveDoesNotJoinBackgroundInflight
  - Register a Background GET /v1/me/player (hold HTTP, don't complete yet)
  - Fire an Interactive GET /v1/me/player
  - Assert Interactive request fires its own HTTP call (not a dedup waiter)
  - Complete both; assert both receive their own independent responses
```

### Task 6 — Update commands tests

**File:** `internal/app/command_safety_test.go` (or equivalent snapshot test)

Any test that calls `fetchPlaybackStateCmd` directly needs the new `priority` argument.
Pass `api.Background` in all existing tests — no behaviour change, just signature update.

---

## Acceptance Criteria

- `GET /v1/me/player` fired from `PlaybackCmdSentMsg` success path uses
  `api.Interactive` priority and does **not** join an in-flight Background poll
  for the same path.
- All other `fetchPlaybackStateCmd` call sites continue to use `Background` priority
  and behave identically to today.
- Background polls still dedup with other Background polls (existing behaviour preserved).
- Interactive reconcile fetches dedup with other Interactive reconcile fetches
  (correct for rapid command bursts — debounce naturally batches them anyway).
- `make ci` passes (lint + tests + 80% coverage).

---

## What This Does Not Change

- Token bucket behaviour — unchanged for Background; Interactive still bypasses it.
- Debounce — unchanged; Interactive still holds 100ms. No new debounce logic.
- Polling intervals — unchanged; adaptive tick matrix is untouched.
- 429 backoff — unchanged; both priorities respect backoff as before.
- Concurrency semaphore — unchanged; both priorities share the 5-slot semaphore.
- Any pane, store, or app-level logic — this story touches only `api/gateway.go`,
  `api/base.go`, and `app/commands.go` / `app/handlers.go`.
