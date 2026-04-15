# Playback Control Correctness — Request-Aware Dedup + Remove Gateway Debounce

**Date:** 2026-04-14
**Status:** Approved — ready for implementation planning
**Features:** `11-api-gateway`, `03-playback`

---

## 1. Goal

**Ensure the reconcile fetch after a playback command always returns fresh Spotify state, and
ensure all playback commands fire without a gateway-level hold.**

Two bugs in the current gateway pipeline produce incorrect behaviour after user-triggered
playback actions:

1. The reconcile GET can silently join a stale in-flight Background poll and receive
   pre-command data — reverting the UI to the old state for up to 3 seconds.
2. A 100ms gateway debounce kills all-but-last in any rapid command burst, silently
   dropping intermediate presses on the same API path.

---

## 2. Problems

### Problem 1 — Incorrect Dedup (Stale Reconcile Fetch)

`fetchPlaybackStateCmd` uses `context.Background()` — no priority set — so the gateway
defaults to Background priority. This means the reconcile GET that fires after a command
completes can join an in-flight Background poll and share its response body.

That poll was initiated **before the command was sent**. It carries pre-command Spotify state.

```
t=0ms:   Background poll tick fires GET /v1/me/player → registered in inflight map
t=100ms: PUT /v1/me/player/volume returns 204 → PlaybackCmdSentMsg
          → reconcile fetchPlaybackStateCmd dispatched (Background priority)
          → gateway: key {GET, /v1/me/player} already in inflight map
          → reconcile JOINS as waiter — no new HTTP call fires
t=250ms: Polling GET HTTP response arrives (vol=20, pre-command)
          → both poll and reconcile receive vol=20
          → store.vol=20   ← WRONG, command was applied but state reverted
t=3000ms: next regular poll returns vol=21 → finally correct
```

The user sees: press volume → UI reverts to old value for ~3 seconds → then corrects.

**Affects:** all playback controls — volume, shuffle, repeat, play, pause, next, previous.

### Problem 2 — Gateway Debounce Kills Rapid Commands

The gateway applies a 100ms hold window (`interactiveDebounce`) to all Interactive requests
keyed by path. A new arrival for the same path cancels the previous one and starts a fresh
100ms timer.

For volume up/down, both share path `/v1/me/player/volume`. Three presses at 60ms intervals:
- t=0ms: PUT #1 enters debounce hold
- t=60ms: PUT #2 cancels PUT #1, starts fresh 100ms hold
- t=120ms: PUT #3 cancels PUT #2, starts fresh 100ms hold
- t=220ms: PUT #3 fires (the only one that survives)

Only the last press survives. The first two are silently dropped.

The debounce was designed for search (last query wins, intermediate queries are obsolete).
For playback controls, each press is a semantically independent command that must fire.

---

## 3. API Reference

### 3.1 Interactive Playback Commands — User-Triggered (PUT / POST)

These bypass the token bucket. The 100ms debounce hold is being removed (see §5.2).

| Key | Method | Endpoint | Query Params | Request Body | Success Response |
|-----|--------|----------|--------------|--------------|-----------------|
| Space (playing) | PUT | `/v1/me/player/pause` | — | — | 204 No Content |
| Space (paused) | PUT | `/v1/me/player/play` | — | JSON (see below) | 204 No Content |
| → | POST | `/v1/me/player/next` | — | — | 204 No Content |
| ← or `p` | POST | `/v1/me/player/previous` | — | — | 204 No Content |
| `+` | PUT | `/v1/me/player/volume` | `volume_percent=21` | — | 204 No Content |
| `-` | PUT | `/v1/me/player/volume` | `volume_percent=19` | — | 204 No Content |
| `s` | PUT | `/v1/me/player/shuffle` | `state=true` | — | 204 No Content |
| `r` | PUT | `/v1/me/player/repeat` | `state=context` | — | 204 No Content |

**Play body variants:**

```json
// Resume current context (no options):
{}

// Play a playlist or album (context_uri):
{
  "context_uri": "spotify:playlist:37i9dQZF1DXcBWIGoYBM5M",
  "offset": { "uri": "spotify:track:4uLU6hMCjMI75M1A2tKUQC" }
}

// Play an ordered list of tracks (top tracks, search results, recently played):
{
  "uris": ["spotify:track:4uLU6hMCjMI75M1A2tKUQC", "spotify:track:2TpxZ7JUBn3uw46aR7qd6V"]
}
```

**Repeat state cycle:** `off` → `context` → `track` → `off`

**Error responses (all controls):**

| Status | Meaning | Handler |
|--------|---------|---------|
| 204 | Success | Fire reconcile GET (Interactive priority) |
| 401 | Token expired | Refresh token, retry once |
| 403 | Not Premium | Toast "Spotify Premium required" |
| 429 | Rate limited | Backoff for `Retry-After` seconds, toast |
| 5xx | Spotify down | Toast error |

### 3.2 Reconcile State Fetch — Post-Command (GET)

Fired by `fetchPlaybackStateCmd` after every `PlaybackCmdSentMsg` (success or failure).

**Currently:** Background priority → can dedup with stale poll
**After fix:** Interactive priority on success path → always fires fresh

| Trigger | Method | Endpoint | Query Params | Response |
|---------|--------|----------|--------------|----------|
| After any playback command | GET | `/v1/me/player` | — | 200 OK (JSON) or 204 No Content (nothing playing) |

**200 response shape (relevant fields):**

```json
{
  "is_playing": true,
  "shuffle_state": false,
  "repeat_state": "context",
  "progress_ms": 45000,
  "item": {
    "id": "4uLU6hMCjMI75M1A2tKUQC",
    "name": "Somebody That I Used to Know",
    "duration_ms": 244960,
    "artists": [{ "name": "Gotye" }],
    "album": { "name": "Making Mirrors" }
  },
  "device": {
    "id": "abc123",
    "name": "MacBook Pro",
    "type": "Computer",
    "volume_percent": 21
  }
}
```

**204 response:** Nothing is playing — `PlaybackStateFetchedMsg{State: nil}`.

### 3.3 Background Polling Schedule

These run on `tea.Tick` at intervals determined by playback state.

| Condition | Playback poll interval | Queue poll interval |
|-----------|----------------------|---------------------|
| Active + playing | every 3s | every 9s |
| Active + paused | every 10s | every 30s |
| Idle + playing | every 10s | every 30s |
| Idle + paused | every 30s | every 60s |

Both use Background priority, both go through the inflight dedup map.

### 3.4 Other Interactive GET Requests (for dedup context)

| Request | Method | Endpoint | Priority | Notes |
|---------|--------|----------|----------|-------|
| Search | GET | `/v1/search?q=queen&type=track,artist&limit=10&offset=0` | Interactive | Own 300ms UI debounce + searchCancel() |
| Devices | GET | `/v1/me/devices` | Interactive | User opens device picker |
| User profile | GET | `/v1/me` | Interactive | On auth completion |
| Playlist tracks | GET | `/v1/playlists/{id}/items?limit=100&offset=0` | Interactive | User opens playlist |
| Album tracks | GET | `/v1/albums/{id}/tracks?limit=50&offset=0` | Interactive | User opens album |

---

## 4. Current Gateway Architecture

### 4.1 `Do()` Pipeline — Phase Order

```
Phase 0:  emit EventRequestEntered

Phase 1:  rate-limiting (priority split)
          ├─ Interactive → waitForBackoff() (blocks until 429 expires; no token)
          └─ Background  → reject immediately if throttled
                         → bucket.wait() — consume one token

Phase 1b: interactiveDebounce() — Interactive only ← BEING REMOVED
          └─ 100ms hold per path; new arrival cancels prior

Phase 2:  in-flight dedup — GET only, priority-blind ← BEING FIXED
          └─ if {Method, Path} exists in inflight map → join as waiter

Phase 3:  concurrency semaphore (max 5 slots)

Phase 4:  register in inflight map (GET) → execute fn() → 429 handling
```

### 4.2 `RequestKey` — Current vs After Fix

```go
// Current — priority-blind, Background and Interactive share a bucket
type RequestKey struct {
    Method string
    Path   string
}

// After fix — Priority added; only Background requests use the map
type RequestKey struct {
    Method   string
    Path     string
    Priority Priority
}
```

The `Priority` field makes the key semantically complete. However, Interactive requests
also skip the inflight map entirely (neither check nor register) — so only
`{GET, path, Background}` entries ever exist in practice.

### 4.3 Token Bucket

- 10 tokens/second, burst of 10
- Lazy refill: tokens computed on demand from elapsed time
- Interactive bypasses entirely
- Background consumes one token per request

### 4.4 Debounce (current — being removed)

`interactiveDebounce` in `gateway_dedup.go`: 100ms hold per API path for Interactive requests.
Uses `wrappedCtx` to cancel the hold without cancelling the downstream HTTP context.
Uses `ready` channel to prevent race when a new arrival registers before the prior has cleaned up.

This is being removed. See §5.2.

---

## 5. Solution Design

### 5.1 Fix 1 — Request-Aware Dedup (Gateway)

**New dedup matrix:**

| In-flight priority \ Arriving priority | Background | Interactive |
|---------------------------------------|------------|-------------|
| **Background** | Join (existing — correct) | Don't join — Interactive fires fresh |
| **Interactive** | Both fire independently | Both fire independently |

**Implementation rule (simpler than the matrix):**

Interactive GET requests skip the inflight map entirely — they never check for an existing
entry (Phase 2) and never register themselves (Phase 4). Background GET requests behave
exactly as before.

```go
// Phase 2: dedup check — Background only
if key.Method == http.MethodGet && priority == Background {
    g.mu.Lock()
    if entry, ok := g.inflight[key]; ok {
        // join as waiter (existing logic unchanged)
        ...
    }
    g.mu.Unlock()
}

// Phase 4: inflight registration — Background only
if key.Method == http.MethodGet && priority == Background {
    // double-check + register (existing logic unchanged)
    ...
}
```

**`fetchPlaybackStateCmd` priority change:**

```go
// commands.go — before
func fetchPlaybackStateCmd(player api.PlayerAPI) tea.Cmd {
    ps, err := player.PlaybackState(context.Background())

// commands.go — after
func fetchPlaybackStateCmd(player api.PlayerAPI, priority api.Priority) tea.Cmd {
    ctx := api.WithPriority(context.Background(), priority)
    ps, err := player.PlaybackState(ctx)
```

**Call site priorities:**

| Call site | Priority | Reason |
|-----------|----------|--------|
| `PlaybackCmdSentMsg` success path | `api.Interactive` | User command — needs fresh response |
| `PlaybackCmdSentMsg` error paths | `api.Background` | Recovery — not user-triggered |
| Polling tick handler | `api.Background` | Regular poll — dedup with other polls is correct |
| Any other recovery sites | `api.Background` | Non-user-triggered |

**`base.go` — `RequestKey` construction:**

All three functions (`doJSON`, `doNoContent`, `doJSONOptional`) build `RequestKey`. Add `Priority`:

```go
// before
key := RequestKey{Method: req.Method, Path: req.URL.Path}

// after
key := RequestKey{
    Method:   req.Method,
    Path:     req.URL.Path,
    Priority: PriorityFromContext(req.Context()),
}
```

**`handlers.go` — `PlaybackCmdSentMsg` handler (no other changes to this handler):**

```go
// before
case panes.PlaybackCmdSentMsg:
    if m.Err != nil {
        ...
        return a, tea.Batch(
            fetchPlaybackStateCmd(a.player),
            a.alerts.NewAlertCmd(...),
        )
    }
    return a, fetchPlaybackStateCmd(a.player)

// after
case panes.PlaybackCmdSentMsg:
    if m.Err != nil {
        ...
        return a, tea.Batch(
            fetchPlaybackStateCmd(a.player, api.Background),
            a.alerts.NewAlertCmd(...),
        )
    }
    return a, fetchPlaybackStateCmd(a.player, api.Interactive)
```

### 5.2 Fix 2 — Remove Gateway Debounce Entirely

The transport-layer debounce (`interactiveDebounce`, `debounceMu`, `debounceEntries`) is removed.

**Why it is safe to remove:**

Search (GET `/v1/search`) — the only request that needed last-wins behaviour — has two
independent protection layers that exist before the gateway:

1. `scheduleDebounce` in `SearchPane` (300ms hold) — rate-limits keystroke-to-message emission.
   Only one `SearchRequestMsg` fires after 300ms of typing silence.
2. `searchCancel()` in `handlers.go` — called at the moment a new `SearchRequestMsg` is handled,
   cancels the previous in-flight HTTP context. Even if two search GETs reach the gateway
   simultaneously, the first has a cancelled context and returns `context.Canceled`, which the
   command turns into a nil message that BubbleTea drops silently.

The gateway debounce was a third redundant layer for search. For playback controls (PUT/POST),
the debounce applied to the path (e.g. `/v1/me/player/volume`) and killed all-but-last in any
100ms burst.

**Code removed:**

- `func (g *Gateway) interactiveDebounce(...)` — deleted
- `debounceMu sync.Mutex` — removed from `Gateway` struct
- `debounceEntries map[string]*interactiveDebounceEntry` — removed from `Gateway` struct
- `interactiveDebounceEntry` type — deleted
- Phase 1b block in `Do()` — deleted
- `debounceEntries: make(...)` in `NewGateway()` — removed
- `internal/api/gateway_debounce_test.go` — deleted (tests for removed feature)

---

## 6. Use-Case Simulations

### Sim 1: Post-command fetch vs Background poll (the stale dedup bug — fixed)

```
t=0ms:   Background poll tick fires GET /v1/me/player → registered in inflight map
t=100ms: PUT /v1/me/player/volume?volume_percent=21 returns 204
          → PlaybackCmdSentMsg{Err: nil}
          → fetchPlaybackStateCmd(player, api.Interactive) dispatched
          → gateway Phase 2: priority == Interactive → skip inflight map check entirely
          → Interactive GET fires its own HTTP call immediately
t=250ms: Background GET response arrives (vol=20, pre-command)
          → store.vol=20 (stale, but...)
t=320ms: Interactive GET response arrives (vol=21, post-command)
          → store.vol=21 ✓

No stale dedup. UI shows 21 within ~320ms of keypress.
```

### Sim 2: Rapid volume presses with debounce removed

```
t=0ms:   + press → PUT vol=21 fires immediately (no debounce hold)
t=60ms:  + press → PUT vol=21 fires immediately (store still 20 → same target — note:
                   stale snapshot is a separate issue, not addressed by these two fixes)
t=120ms: + press → PUT vol=21 fires immediately

All three PUTs fire and complete. Each triggers an Interactive reconcile GET.
Last Interactive GET (from PUT #3 ack) returns confirmed state.
```

**Note:** Rapid presses still snapshot stale store state (all target same value). That is a
separate problem (pane-local optimistic state) deferred to a future story.

### Sim 3: Search — two layers of protection, no gateway debounce needed

User types "queen" character by character:
1. `scheduleDebounce` (300ms UI hold) — only fires `SearchRequestMsg` after 300ms silence
2. `searchCancel()` — cancels in-flight HTTP context when a new `SearchRequestMsg` arrives
3. Even if two search GETs reach gateway simultaneously: first has cancelled context → returns
   `context.Canceled` → command returns nil → BubbleTea drops silently. Second fires normally.

Gateway debounce removal has zero impact on search correctness.

### Sim 4: Rapid search + debounce removal (belt-and-suspenders check)

- User types "q", "u", "e", "e", "n" at 40ms intervals
- `scheduleDebounce`: resets timer on each keystroke, fires `SearchRequestMsg` only after "n" + 300ms silence
- One `SearchRequestMsg` arrives at gateway → one Interactive GET fires
- No change in observable search behaviour ✓

### Sim 5: Device transfer (Interactive POST `/v1/me/player`)

- POST never enters inflight map (dedup is GET-only) — unchanged behaviour
- `DeviceTransferredMsg` → `fetchPlaybackStateCmd(player, api.Interactive)` fires fresh
- Interactive GET bypasses any in-flight Background poll → gets fresh device state ✓

### Sim 6: Background + Background dedup (existing behaviour preserved)

```
t=0ms:   tick poll fires GET /v1/me/player (Background) → registered in inflight map
t=500ms: second tick fires GET /v1/me/player (Background)
          → gateway Phase 2: priority == Background → check inflight map
          → found → join as waiter — correct, efficient
t=250ms: HTTP response arrives → both receive same body ✓
```

Background dedup is untouched. Token bucket still applies to all Background requests.

---

## 7. What Is Not Changed

| Component | Why unchanged |
|-----------|---------------|
| Token bucket | Interactive still bypasses; Background still consumes. Unchanged. |
| 429 backoff | Both priorities respect backoff as before. |
| Concurrency semaphore | Both priorities share the 5-slot cap. Unchanged. |
| Polling intervals | Adaptive tick matrix untouched. |
| Background+Background dedup | Still correct. Two polls share one response. Unchanged. |
| Search debounce (UI layer) | 300ms hold in SearchPane unchanged. |
| `searchCancel()` | Context cancellation for in-flight search unchanged. |
| `PlaybackRequestMsg` | No new fields. `buildPlaybackAPICmd` signature unchanged. |
| `PlaybackCmdSentMsg` | No new fields. |
| NowPlayingPane | No changes. Pane-local optimistic state is a future story. |

---

## 8. Tasks

### Task 1 — `RequestKey` gains `Priority` field

**File:** `internal/api/gateway.go`

```go
type RequestKey struct {
    Method   string
    Path     string
    Priority Priority
}
```

### Task 2 — Populate `Priority` in all `RequestKey` construction sites

**File:** `internal/api/base.go`

In `doJSON`, `doJSONOptional`, and `doNoContent`:

```go
key := RequestKey{
    Method:   req.Method,
    Path:     req.URL.Path,
    Priority: PriorityFromContext(req.Context()),
}
```

### Task 3 — Gate inflight map on Background priority

**File:** `internal/api/gateway.go`

Phase 2 (dedup check) and Phase 4 (registration) are gated:

```go
// Phase 2
if key.Method == http.MethodGet && priority == Background {
    // existing join-as-waiter logic — unchanged
}

// Phase 4
if key.Method == http.MethodGet && priority == Background {
    // existing register + defer cleanup logic — unchanged
}
```

### Task 4 — Remove gateway debounce

**File:** `internal/api/gateway.go`, `internal/api/gateway_dedup.go`

- Delete `func (g *Gateway) interactiveDebounce(...)`
- Delete `interactiveDebounceEntry` type
- Delete `debounceMu sync.Mutex` and `debounceEntries map[string]*interactiveDebounceEntry` from `Gateway` struct
- Delete `debounceEntries: make(...)` from `NewGateway()`
- Delete Phase 1b block from `Do()`:
  ```go
  // DELETE this entire block:
  if priority == Interactive {
      if err := g.interactiveDebounce(ctx, key.Path); err != nil {
          return nil, err
      }
  }
  ```
- Delete `internal/api/gateway_debounce_test.go`

### Task 5 — `fetchPlaybackStateCmd` accepts priority

**File:** `internal/app/commands.go`

```go
func fetchPlaybackStateCmd(player api.PlayerAPI, priority api.Priority) tea.Cmd {
    return func() tea.Msg {
        ctx := api.WithPriority(context.Background(), priority)
        ps, err := player.PlaybackState(ctx)
        ...
    }
}
```

### Task 6 — Update all `fetchPlaybackStateCmd` call sites

**File:** `internal/app/handlers.go`

| Call site | Priority |
|-----------|----------|
| `PlaybackCmdSentMsg` success path | `api.Interactive` |
| `PlaybackCmdSentMsg` error paths | `api.Background` |
| Polling tick (`TickMsg` handler) | `api.Background` |
| Any other recovery sites | `api.Background` |

### Task 7 — Tests

**Gateway:**
- All `RequestKey` literals in `gateway_test.go` and `gateway_hardening_test.go` need `Priority` field. Add `Priority: Background` to existing Background test keys.
- Delete `gateway_debounce_test.go`.
- Add `TestDedup_InteractiveDoesNotJoinBackground`: register Background GET in inflight map (hold HTTP with a channel); fire Interactive GET to same path; assert Interactive fires its own HTTP call independently; assert both receive independent responses.
- Add `TestDedup_InteractiveDoesNotJoinInteractive`: fire two Interactive GETs to same path concurrently; assert both fire independent HTTP calls.
- Add `TestDedup_BackgroundJoinsBackground`: two Background GETs to same path; assert only one HTTP call fires; both receive same body (existing behaviour preserved).

**App commands:**
- Update `command_safety_test.go` and any test calling `fetchPlaybackStateCmd` to pass priority argument.

---

## 9. Acceptance Criteria

- `GET /v1/me/player` fired from `PlaybackCmdSentMsg` success path does not join an in-flight Background poll for the same path.
- Background polls still deduplicate with other Background polls (existing behaviour preserved).
- Interactive GET requests never join any in-flight request — Background or Interactive.
- All playback command PUTs/POSTs fire immediately with no 100ms gateway hold.
- Search behaviour is unchanged — last query wins, stale results are dropped.
- `make ci` passes (lint + tests + 80% coverage).

---

## 10. Files Changed

| File | Change |
|------|--------|
| `internal/api/gateway.go` | Add `Priority` to `RequestKey`; gate Phase 2 + Phase 4 on `Background`; remove Phase 1b debounce block |
| `internal/api/gateway_dedup.go` | Remove `interactiveDebounceEntry` type; remove `interactiveDebounce()` function |
| `internal/api/base.go` | Add `Priority` to all three `RequestKey` constructions |
| `internal/api/gateway_debounce_test.go` | Delete |
| `internal/api/gateway_test.go` | Add `Priority` to all `RequestKey` literals; add new dedup tests |
| `internal/api/gateway_hardening_test.go` | Add `Priority` to `RequestKey` literals |
| `internal/app/commands.go` | `fetchPlaybackStateCmd` accepts priority argument |
| `internal/app/handlers.go` | Update all `fetchPlaybackStateCmd` call sites with correct priority |
