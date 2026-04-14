# Playback Control Correctness — Request-Aware Dedup + Pane-Local Optimistic State

**Date:** 2026-04-14
**Status:** Approved — ready for implementation planning
**Features:** `11-api-gateway`, `03-playback`

---

## 1. Goal

**Eventual operational consistency for all playback controls.**

If the user presses volume up 3 times from vol=20, the UI must eventually show vol=23. If the
user presses up 3 times then down 2 times, the UI must eventually show vol=21. The same applies
to shuffle (on → off → on is a net on) and repeat (cycles must be applied in order).

Slowness is acceptable. Incorrect final values are not.

---

## 2. Two Compounding Problems

### Problem 1 — Stale Snapshot (Incorrect Target Values)

`buildPlaybackAPICmd` in `commands.go` reads `store.PlaybackState()` at the moment the command
is built. If the user presses `+` three times at 60ms intervals and the network round-trip
is 300ms, the store has not changed by the time presses 2 and 3 are dispatched.

```
t=0ms:   press +1 → store vol=20 → cmd snapshots 20 → PUT vol=21
t=60ms:  press +2 → store STILL vol=20 → cmd snapshots 20 → PUT vol=21 (again)
t=120ms: press +3 → store STILL vol=20 → cmd snapshots 20 → PUT vol=21 (again)
```

All three commands target vol=21. Vol=22 and vol=23 are never computed. Even if the gateway
fired all three commands, they would all write the same value.

**Affects:** volume up/down, shuffle toggle (double-toggle reads stale IsShuffled),
repeat cycle (double-cycle reads stale RepeatMode).

### Problem 2 — Incorrect Dedup (Stale Reconcile Fetch)

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

### Problem 3 — Gateway Debounce Kills Rapid Commands

The gateway applies a 100ms hold window (`interactiveDebounce`) to all Interactive requests
keyed by path. A new arrival for the same path cancels the previous one and starts a fresh
100ms timer.

For volume up/down, both share path `/v1/me/player/volume`. Three presses at 60ms intervals:
- t=0ms: PUT #1 enters debounce hold
- t=60ms: PUT #2 cancels PUT #1, starts fresh 100ms hold
- t=120ms: PUT #3 cancels PUT #2, starts fresh 100ms hold
- t=220ms: PUT #3 fires (the only one that survives)

Combined with Problem 1 (all three snapshotted vol=20), the only PUT that fires targets vol=21.
Vol=22 and vol=23 never exist.

The debounce was designed for search (last query wins, intermediate queries are obsolete). For
volume, each press is a semantically independent increment that must fire.

---

## 3. API Reference

### 3.1 Interactive Playback Commands — User-Triggered (PUT / POST)

These bypass the token bucket and go through the 100ms debounce hold (to be removed).

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
| 204 | Success | Fire reconcile GET |
| 401 | Token expired | Refresh token, retry once |
| 403 | Not Premium | Toast "Spotify Premium required" |
| 429 | Rate limited | Backoff for `Retry-After` seconds, toast |
| 5xx | Spotify down | Toast error |

### 3.2 Reconcile State Fetch — Post-Command (GET)

Fired by `fetchPlaybackStateCmd` after every `PlaybackCmdSentMsg` (success or failure).

**Currently:** Background priority → can dedup with stale poll
**After fix:** Interactive priority → always fires fresh

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
- `gateway_debounce_test.go` — deleted (tests for removed feature)

### 5.3 Fix 3 — Pane-Local Optimistic State (NowPlayingPane)

**Precedent:** `localProgressMs` already lives in `NowPlayingPane` and uses this exact pattern —
a locally-incremented value that resets to server truth on each `PlaybackStateFetchedMsg`. This
design extends the same pattern to volume, shuffle, and repeat.

**New fields in `NowPlayingPane`:**

```go
// Optimistic local state — pane intent between keypress and command acknowledgement.
// Sentinel: localVolume=-1, localShuffle=nil, localRepeat=nil → no pending state; use store.
localVolume  int     // -1 = no pending
localShuffle *bool   // nil = no pending
localRepeat  *string // nil = no pending

// In-flight counters: how many commands of each type are awaiting 204 or error.
// When a counter reaches 0, the corresponding local field is cleared.
// The store value then owns the display on the next PlaybackStateFetchedMsg.
volInFlight     int
shuffleInFlight int
repeatInFlight  int
```

**Why in-flight counters are needed:**

Without counters, there is no reliable way to know when all pending commands for a control
have been acknowledged. Without that signal:
- If the server returns vol=20 (Spotify lag) before all PUTs return, and we clear local
  on every `PlaybackStateFetchedMsg`, the UI flashes to 20 before eventually settling at 23.
- If all commands fail, we must know to clear and show the server's confirmed state.

The counters are simple: one increment per keypress, one decrement per ack. When zero: clear.

**`PlaybackRequestMsg` extended:**

```go
// messages.go — before
type PlaybackRequestMsg struct {
    Action PlaybackAction
}

// messages.go — after
type PlaybackRequestMsg struct {
    Action        PlaybackAction
    TargetVolume  int    // pre-computed by pane for VolumeUp/Down
    TargetShuffle bool   // pre-computed by pane for ToggleShuffle
    TargetRepeat  string // pre-computed by pane for CycleRepeat
}
```

The pane pre-computes the target using its `localVolume`/`localShuffle`/`localRepeat` fields
and embeds it in the message. `buildPlaybackAPICmd` uses the message's target directly instead
of reading from the store — eliminating the stale snapshot problem.

**`PlaybackCmdSentMsg` extended:**

```go
// messages.go — before
type PlaybackCmdSentMsg struct {
    Err error
}

// messages.go — after
type PlaybackCmdSentMsg struct {
    Action PlaybackAction // which control completed
    Err    error
}
```

**New `PlaybackCmdAckMsg`:**

```go
// messages.go — new type
// PlaybackCmdAckMsg is forwarded by App.Update to NowPlayingPane after a
// PlaybackCmdSentMsg is received. It allows the pane to decrement its in-flight
// counter and clear local optimistic state when all commands have been acknowledged.
type PlaybackCmdAckMsg struct {
    Action PlaybackAction
    Err    error
}
```

**Keypress handler in `NowPlayingPane`:**

```go
case "+":
    if p.localVolume < 0 {
        p.localVolume = storeVolume // anchor to server value at burst start
    }
    p.localVolume = min(100, p.localVolume+1)
    p.volInFlight++
    return p, func() tea.Msg {
        return PlaybackRequestMsg{Action: ActionVolumeUp, TargetVolume: p.localVolume}
    }

case "-":
    if p.localVolume < 0 {
        p.localVolume = storeVolume
    }
    p.localVolume = max(0, p.localVolume-1)
    p.volInFlight++
    return p, func() tea.Msg {
        return PlaybackRequestMsg{Action: ActionVolumeDown, TargetVolume: p.localVolume}
    }

case "s":
    current := ps.ShuffleState
    if p.localShuffle != nil {
        current = *p.localShuffle
    }
    next := !current
    p.localShuffle = &next
    p.shuffleInFlight++
    return p, func() tea.Msg {
        return PlaybackRequestMsg{Action: ActionToggleShuffle, TargetShuffle: next}
    }

case "r":
    current := ps.RepeatState
    if p.localRepeat != nil {
        current = *p.localRepeat
    }
    next := nextRepeatMode(current)
    p.localRepeat = &next
    p.repeatInFlight++
    return p, func() tea.Msg {
        return PlaybackRequestMsg{Action: ActionCycleRepeat, TargetRepeat: next}
    }
```

**Ack handler in `NowPlayingPane`:**

```go
case PlaybackCmdAckMsg:
    switch m.Action {
    case ActionVolumeUp, ActionVolumeDown:
        p.volInFlight--
        if p.volInFlight == 0 {
            p.localVolume = -1
        }
    case ActionToggleShuffle:
        p.shuffleInFlight--
        if p.shuffleInFlight == 0 {
            p.localShuffle = nil
        }
    case ActionCycleRepeat:
        p.repeatInFlight--
        if p.repeatInFlight == 0 {
            p.localRepeat = nil
        }
    }
    return p, nil
```

**`ClearOptimisticState()` method (called on 429 and 401):**

```go
func (p *NowPlayingPane) ClearOptimisticState() {
    p.localVolume = -1
    p.localShuffle = nil
    p.localRepeat = nil
    p.volInFlight = 0
    p.shuffleInFlight = 0
    p.repeatInFlight = 0
}
```

**`View()` changes:**

```go
volume := 0
if ps.Device != nil {
    volume = ps.Device.VolumePercent
}
if p.localVolume >= 0 {
    volume = p.localVolume // optimistic override
}

shuffle := ps.ShuffleState
if p.localShuffle != nil {
    shuffle = *p.localShuffle
}

repeat := ps.RepeatState
if p.localRepeat != nil {
    repeat = *p.localRepeat
}

ctrl := components.NewControls(p.theme, ps.IsPlaying, shuffle, repeat)
p.volumeBar.Render(volume)
```

**App forwarding in `handlers.go`:**

```go
case panes.PlaybackCmdSentMsg:
    var fetchCmd tea.Cmd
    if m.Err != nil {
        // existing error handling (toast, fetch with Background)...
        fetchCmd = fetchPlaybackStateCmd(a.player, api.Background)
    } else {
        fetchCmd = fetchPlaybackStateCmd(a.player, api.Interactive)
    }
    // Forward ack to NowPlayingPane so it can decrement its in-flight counter.
    ackMsg := panes.PlaybackCmdAckMsg{Action: m.Action, Err: m.Err}
    if np := a.nowPlayingPane(); np != nil {
        updatedPane, ackCmd := np.Update(ackMsg)
        if pp, ok := updatedPane.(*panes.NowPlayingPane); ok {
            a.panes[layout.PaneNowPlaying] = pp
        }
        return a, tea.Batch(fetchCmd, ackCmd)
    }
    return a, fetchCmd
```

**`buildPlaybackAPICmd` uses message target:**

```go
// commands.go — before
func (a *App) buildPlaybackAPICmd(action panes.PlaybackAction) tea.Cmd {
    ps := a.store.PlaybackState()
    currentVolume = ps.Device.VolumePercent  // stale snapshot

// commands.go — after
func (a *App) buildPlaybackAPICmd(msg panes.PlaybackRequestMsg) tea.Cmd {
    // For volume, shuffle, repeat: use pre-computed target from pane.
    // For play, pause, next, previous: no target needed, snapshot not used.
    return func() tea.Msg {
        switch msg.Action {
        case panes.ActionVolumeUp, panes.ActionVolumeDown:
            err = player.SetVolume(ctx, msg.TargetVolume)
        case panes.ActionToggleShuffle:
            err = player.SetShuffle(ctx, msg.TargetShuffle)
        case panes.ActionCycleRepeat:
            err = player.SetRepeat(ctx, msg.TargetRepeat)
        ...
        }
        return panes.PlaybackCmdSentMsg{Action: msg.Action, Err: err}
    }
}
```

**Elm architecture — why this is correct:**

- `View()` is pure — reads from model fields, no side effects
- `Update()` is the only place state changes — `localVolume`, `volInFlight` etc. set in `handleKey` and cleared in the ack handler, both within `Update()`
- Pane never calls the API — it emits `PlaybackRequestMsg` with a pre-computed target
- Store remains the server-confirmed truth — local fields are a shadow that exists only between keypress and acknowledgement
- Direct `np.Update(ackMsg)` from `App.Update()` is the established BubbleTea parent-child composition pattern; the returned `(Model, Cmd)` is stored and batched correctly

**On 429 rate limit:**

`buildPlaybackAPICmd` returns `RateLimitedMsg` (not `PlaybackCmdSentMsg`) when Spotify returns
429. Since `RateLimitedMsg` bypasses the normal ack flow, the pane's in-flight counter would
never decrement. The `RateLimitedMsg` handler in `App.Update()` calls
`np.ClearOptimisticState()` immediately — resetting all local fields and counters to zero.
This ensures the UI shows the server-confirmed state after the backoff period.

Same applies to `unauthorizedMsg` (401) — `ClearOptimisticState()` is called.

---

## 6. Use-Case Simulations

### Sim 1: Volume up × 3 from vol=20 (rapid presses, 60ms apart)

Background poll fires at t=0 (GET `/v1/me/player`, 250ms RTT, returns pre-command vol=20).

| Time | Event | localVolume | volInFlight | Store.vol | Displayed |
|------|-------|-------------|-------------|-----------|-----------|
| t=0 | poll fires Background GET | -1 | 0 | 20 | 20 |
| t=20 | press +1 | 21 | 1 | 20 | **21** |
| t=80 | press +2 | 22 | 2 | 20 | **22** |
| t=140 | press +3 | 23 | 3 | 20 | **23** |
| t=250 | Background GET returns vol=20 | 23 | 3 | 20 | **23** (local overrides) |
| t=280 | PUT #1 (vol=21) returns 204 → ack | 23 | 2 | 20 | **23** |
| t=280 | Interactive GET #1 fires (no Background in-flight) | | | | |
| t=340 | PUT #2 (vol=22) returns 204 → ack | 23 | 1 | 20 | **23** |
| t=400 | PUT #3 (vol=23) returns 204 → ack | **-1** | **0** | 20 | 20 (briefly) |
| t=430 | Interactive GET #1 returns vol=21 | -1 | 0 | 21 | 21 |
| t=490 | Interactive GET #2 returns vol=22 | -1 | 0 | 22 | 22 |
| t=530 | Interactive GET #3 returns vol=23 | -1 | 0 | **23** | **23** ✓ |

**Final: vol=23. Correct.**

The brief dip to 20 at t=400ms (between local clearing and Interactive GET #3 returning) is
a small window (~130ms). Acceptable.

### Sim 2: Volume up × 3 then down × 2 (net +1 → vol=21 from 20)

| Press | localVolume | volInFlight |
|-------|-------------|-------------|
| +1 | 21 | 1 |
| +2 | 22 | 2 |
| +3 | 23 | 3 |
| -1 | 22 | 4 |
| -2 | 21 | 5 |

5 PUTs fire (no debounce). Each targets a different volume. Last PUT = vol=21.
Spotify last-write-wins → server ends at vol=21.
Acks: inFlight 5→4→3→2→1→0 → local=-1.
Interactive GET (from last ack) returns vol=21. Store=21. ✓

### Sim 3: Shuffle toggle × 2 rapidly (false → true → false)

- press s: `localShuffle=true`, shuffleInFlight=1, PUT shuffle=true
- press s: `localShuffle=false`, shuffleInFlight=2, PUT shuffle=false
- Both PUTs fire independently (no debounce, no dedup for PUT)
- Last PUT = shuffle=false → Spotify ends at false
- Acks: shuffleInFlight 2→1→0 → localShuffle=nil
- Interactive GET confirms shuffle=false ✓

### Sim 4: Repeat cycle × 3 (off → context → track → off)

- press r: localRepeat="context", repeatInFlight=1, PUT repeat=context
- press r: localRepeat="track", repeatInFlight=2, PUT repeat=track
- press r: localRepeat="off", repeatInFlight=3, PUT repeat=off
- Last PUT = repeat=off → Spotify ends at off
- Acks: inFlight 3→2→1→0 → localRepeat=nil
- Interactive GET confirms repeat=off ✓

### Sim 5: Post-command fetch vs Background poll (the original stale dedup bug)

- Background poll fires GET `/v1/me/player` at t=0 (Background)
- PUT vol=21 returns 204 at t=200ms
- `fetchPlaybackStateCmd(player, api.Interactive)` fires
- Gateway Phase 2: `priority == Interactive` → **skip inflight map check entirely**
- Interactive GET fires its own HTTP call
- t=280ms: Background GET returns vol=20 → store=20; pane: local=22 (inFlight>0) → displays 22 ✓
- t=350ms: Interactive GET returns vol=21 → store=21 ✓

No stale dedup. Correct.

### Sim 6: Search — two layers of protection, no gateway debounce needed

User types "queen" character by character:
1. `scheduleDebounce` (300ms UI hold) — only fires `SearchRequestMsg` after 300ms silence
2. `searchCancel()` — cancels in-flight HTTP context when a new `SearchRequestMsg` arrives
3. Even if two search GETs reach gateway simultaneously: first has cancelled context → returns
   `context.Canceled` → command returns nil → BubbleTea drops silently. Second fires normally.

Gateway debounce removal has zero impact on search correctness.

### Sim 7: Rapid search + debounce removal (belt-and-suspenders check)

- User types "q", "u", "e", "e", "n" at 40ms intervals
- `scheduleDebounce`: resets timer on each keystroke, fires SearchRequestMsg only after "n" + 300ms silence
- One `SearchRequestMsg` arrives at gateway → one Interactive GET fires
- Previous debounce held this at gateway anyway; now UI debounce does all the work
- No change in observable search behaviour ✓

### Sim 8: All commands fail (network error)

- press +3: localVolume=23, volInFlight=3
- PUT #1 fails → PlaybackCmdAckMsg{err} → volInFlight=2 (local stays 23)
- PUT #2 fails → PlaybackCmdAckMsg{err} → volInFlight=1 (local stays 23)
- PUT #3 fails → PlaybackCmdAckMsg{err} → volInFlight=0 → **localVolume=-1**
- `fetchPlaybackStateCmd(Background)` fires on each error (existing behaviour)
- GET returns vol=20 (nothing changed) → store=20 → displays 20 ✓
- Error toasts fired ✓

### Sim 9: Middle command fails, outer commands succeed

- press +3: localVolume=23, volInFlight=3
- PUT #1 (vol=21) returns 204 → ack → volInFlight=2
- PUT #2 (vol=22) fails → ack{err} → volInFlight=1 (local stays 23; #3 still in-flight)
- PUT #3 (vol=23) returns 204 → ack → volInFlight=0 → **localVolume=-1**
- Spotify applied #1 (21), #2 failed, #3 applied (23) → server=23
- Interactive GET returns vol=23 → store=23 → displays 23 ✓
- Error toast for #2 fired ✓

### Sim 10: 429 rate limit mid-burst

- press +3: localVolume=23, volInFlight=3; 3 PUTs in-flight
- PUT #2 returns 429 → `buildPlaybackAPICmd` returns `RateLimitedMsg`
- `App.Update(RateLimitedMsg)`: fires backoff toast; calls `np.ClearOptimisticState()`
- pane: localVolume=-1, volInFlight=0 — all optimistic state abandoned
- Gateway blocks remaining Interactive requests during backoff (`waitForBackoff`)
- After Retry-After expires, user can press again — fresh burst starts from confirmed server state ✓

### Sim 11: Device transfer (Interactive POST `/v1/me/player`)

- POST never enters inflight map (dedup is GET-only) — unchanged behaviour
- `DeviceTransferredMsg` → `fetchPlaybackStateCmd(player, api.Interactive)` fires fresh
- Interactive GET bypasses any in-flight Background poll → gets fresh device state ✓

### Sim 12: Play from playlist (Interactive PUT `/v1/me/player/play` with body)

- PUT not deduped — fires immediately (no debounce)
- `PlaybackCmdSentMsg{Action: ActionPlay}` → Interactive reconcile GET
- No pane-local optimistic update for play (can't predict next track, artist, progress)
- UI shows "loading" state until PlaybackStateFetchedMsg arrives with new track info ✓

---

## 7. What Is Not Changed

| Component | Why unchanged |
|-----------|---------------|
| Token bucket | Interactive still bypasses; Background still consumes. Unchanged. |
| 429 backoff | Both priorities respect backoff as before. |
| Concurrency semaphore | Both priorities share the 5-slot cap. Unchanged. |
| Polling intervals | Adaptive tick matrix untouched. |
| Play/pause local state | Single press — no rapid accumulation use case. Store truth is fast enough. |
| Next/previous local state | Cannot predict next track. Store truth required. |
| Background+Background dedup | Still correct. Two polls share one response. Unchanged. |
| Search debounce (UI layer) | 300ms hold in SearchPane unchanged. |
| searchCancel() | Context cancellation for in-flight search unchanged. |

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

### Task 7 — Extend messages

**File:** `internal/ui/panes/messages.go`

1. Add `TargetVolume int`, `TargetShuffle bool`, `TargetRepeat string` to `PlaybackRequestMsg`
2. Add `Action PlaybackAction` to `PlaybackCmdSentMsg`
3. Add new `PlaybackCmdAckMsg` type (see §5.3)

### Task 8 — `buildPlaybackAPICmd` uses message targets

**File:** `internal/app/commands.go`

- Change signature to accept `panes.PlaybackRequestMsg` instead of `panes.PlaybackAction`
- Use `msg.TargetVolume` for volume actions (no store read for volume)
- Use `msg.TargetShuffle` for shuffle
- Use `msg.TargetRepeat` for repeat
- Return `PlaybackCmdSentMsg{Action: msg.Action, Err: err}` to carry action forward
- Update call site in `handlers.go`: `a.buildPlaybackAPICmd(m)` where `m` is the full `PlaybackRequestMsg`

### Task 9 — Forward ack to NowPlayingPane

**File:** `internal/app/handlers.go`

In `PlaybackCmdSentMsg` handler, after dispatching fetch, forward ack to pane (see §5.3 for
exact code). Handle returned `(Model, Cmd)` correctly — store updated pane, batch cmd.

Add `np.ClearOptimisticState()` call in `RateLimitedMsg` and `unauthorizedMsg` handlers.

### Task 10 — NowPlayingPane local optimistic state

**File:** `internal/ui/panes/nowplaying.go`

1. Add `localVolume int`, `localShuffle *bool`, `localRepeat *string` fields
2. Add `volInFlight int`, `shuffleInFlight int`, `repeatInFlight int` fields
3. Update `handleKey`: set local fields + increment counters + embed target in `PlaybackRequestMsg`
4. Add `PlaybackCmdAckMsg` case to `Update()`: decrement counters, clear local on zero
5. Add `ClearOptimisticState()` method
6. Update `View()`: read local fields when set, else read store (see §5.3)
7. Init `localVolume = -1` in `NewNowPlayingPane`

### Task 11 — Tests

**Gateway:**
- All `RequestKey` literals in `gateway_test.go` and `gateway_hardening_test.go` need `Priority` field. Add `Priority: Background` to existing Background test keys.
- Delete `gateway_debounce_test.go`.
- Add `TestDedup_InteractiveDoesNotJoinBackground`: register Background GET in inflight map (hold HTTP); fire Interactive GET to same path; assert Interactive fires its own HTTP call; assert both receive independent responses.
- Add `TestDedup_InteractiveDoesNotJoinInteractive`: fire two Interactive GETs to same path concurrently; assert both fire independent HTTP calls.

**App commands:**
- Update `command_safety_test.go` and any test calling `fetchPlaybackStateCmd` to pass priority argument.
- Update any test calling `buildPlaybackAPICmd` to pass full `PlaybackRequestMsg`.

**NowPlayingPane:**
- Add table-driven test for `localVolume` increment logic (3× press from vol=20 → localVolume=23).
- Add test for ack decrement (volInFlight 3→2→1→0 → localVolume=-1).
- Add test for failure ack (error on last ack → localVolume cleared).
- Add test for `ClearOptimisticState` resetting all fields.

---

## 9. Acceptance Criteria

- Pressing volume up 3 times from vol=20 eventually shows vol=23 in the UI.
- Pressing +3 then -2 eventually shows vol=21.
- Shuffle toggle × 2 returns to original state.
- Repeat cycle × 3 returns to original mode.
- `GET /v1/me/player` fired from `PlaybackCmdSentMsg` success path does not join an in-flight Background poll for the same path.
- Background polls still deduplicate with other Background polls (existing behaviour preserved).
- Interactive requests never wait for or share a response with any other request.
- All playback command PUTs fire immediately (no 100ms gateway hold).
- Search behaviour is unchanged — last query wins, stale results are dropped.
- `make ci` passes (lint + tests + 80% coverage).

---

## 10. Files Changed

| File | Change |
|------|--------|
| `internal/api/gateway.go` | Add `Priority` to `RequestKey`; gate Phase 2 + Phase 4 on `Background`; remove Phase 1b debounce block |
| `internal/api/gateway_dedup.go` | Remove `interactiveDebounceEntry`, remove `interactiveDebounce()` |
| `internal/api/base.go` | Add `Priority` to all three `RequestKey` constructions |
| `internal/api/gateway_debounce_test.go` | Delete |
| `internal/api/gateway_test.go` | Add `Priority` to all `RequestKey` literals; add new dedup tests |
| `internal/api/gateway_hardening_test.go` | Add `Priority` to `RequestKey` literals |
| `internal/app/commands.go` | `fetchPlaybackStateCmd` accepts priority; `buildPlaybackAPICmd` uses `PlaybackRequestMsg` targets |
| `internal/app/handlers.go` | Update all `fetchPlaybackStateCmd` call sites; forward ack to pane; call `ClearOptimisticState` on 429/401 |
| `internal/ui/panes/messages.go` | Extend `PlaybackRequestMsg`, `PlaybackCmdSentMsg`; add `PlaybackCmdAckMsg` |
| `internal/ui/panes/nowplaying.go` | Add local optimistic fields, ack handler, `ClearOptimisticState`, updated `View()` |
