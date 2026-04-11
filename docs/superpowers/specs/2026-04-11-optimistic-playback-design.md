# Optimistic Playback Updates

**Date:** 2026-04-11  
**Status:** Approved  
**Scope:** `internal/app/` only — no changes to Store interface, panes, or theme

---

## Problem

Playback control actions (volume, play/pause, shuffle, repeat) have visible UI lag of
~500ms–1s between key press and the UI reflecting the new state. The physical device
responds immediately (Spotify SDK), but the UI bar waits for a full roundtrip:

```
key press
  → 100ms gateway debounce
  → ~200-400ms HTTP PUT (set volume)
  → PlaybackCmdSentMsg received
  → ~200-400ms HTTP GET (fetch state)
  → PlaybackStateFetchedMsg received
  → store.SetPlaybackState()
  → UI renders updated bar
```

---

## Solution

Write the predicted state to the store immediately when the key is pressed, before
the API call fires. The existing fetch-on-completion path overwrites with authoritative
state. No new types, no new store fields, no new messages.

---

## Data Flow

**Before:**
```
PlaybackRequestMsg
  → buildPlaybackAPICmd(action)
```

**After:**
```
PlaybackRequestMsg
  → applyOptimisticUpdate(action)   ← store written immediately, UI renders next frame
  → buildPlaybackAPICmd(action)     ← API fires async, unchanged
```

---

## Implementation

### New method: `applyOptimisticUpdate`

Location: `internal/app/handlers.go` (or `internal/app/optimistic.go`)

1. Read `store.PlaybackState()` — returns nil if no state yet
2. If nil, return early (guard — nothing to mutate)
3. Deep-copy: copy the struct value, copy pointer fields (`Device`)
4. Apply the predicted mutation for the given action
5. Write back via `store.SetPlaybackState(&updated)`

### Mutations per action

| Action | Mutation |
|--------|---------|
| `ActionVolumeUp` | `device.VolumePercent = min(v + volumeStep, 100)` |
| `ActionVolumeDown` | `device.VolumePercent = max(v - volumeStep, 0)` |
| `ActionPause` | `ps.IsPlaying = false` |
| `ActionPlay` | `ps.IsPlaying = true` |
| `ActionToggleShuffle` | `ps.ShuffleState = !ps.ShuffleState` |
| `ActionCycleRepeat` | `ps.RepeatState = nextRepeatMode(ps.RepeatState)` |
| `ActionNext` | explicit no-op case — next track is determined by Spotify |
| `ActionPrevious` | explicit no-op case — next track is determined by Spotify |

Guard for volume actions: `ps.Device != nil` required before touching `VolumePercent`.

### Change point in handlers.go

```go
case panes.PlaybackRequestMsg:
    a.applyOptimisticUpdate(m.Action)       // NEW
    return a, a.buildPlaybackAPICmd(m.Action)
```

---

## Error Handling & Reconciliation

**Success path (no change):**  
`PlaybackCmdSentMsg{Err: nil}` → `fetchPlaybackStateCmd` → `PlaybackStateFetchedMsg`
→ `store.SetPlaybackState(ps)` overwrites optimistic value with authoritative state.

**Error path (no change):**  
`PlaybackCmdSentMsg{Err: non-nil}` → toast fires + `fetchPlaybackStateCmd` fires
→ authoritative state overwrites optimistic value (~200–400ms after the error).  
Optimistic value stays visible only for the duration of the correcting fetch — acceptable.

**Rapid keypresses (hold `+`):**  
Each press reads the current (already-optimistic) store value and applies another step.
Gateway 100ms debounce means only the latest value fires to the API. UI tracks every step.
Correct behaviour.

---

## Testing

New file: `internal/app/optimistic_test.go`

Table-driven unit tests:

| Test case | Input | Action | Expected |
|-----------|-------|--------|----------|
| volume up | vol=65, device non-nil | ActionVolumeUp | vol=66 |
| volume up at max | vol=100 | ActionVolumeUp | vol=100 (clamped) |
| volume down | vol=65 | ActionVolumeDown | vol=64 |
| volume down at floor | vol=0 | ActionVolumeDown | vol=0 (clamped) |
| pause | IsPlaying=true | ActionPause | IsPlaying=false |
| play | IsPlaying=false | ActionPlay | IsPlaying=true |
| shuffle on | ShuffleState=false | ActionToggleShuffle | ShuffleState=true |
| shuffle off | ShuffleState=true | ActionToggleShuffle | ShuffleState=false |
| repeat cycle | RepeatState="off" | ActionCycleRepeat | RepeatState="context" |
| nil playback state | store nil | ActionVolumeUp | store still nil, no panic |
| nil device | ps non-nil, device nil | ActionVolumeUp | no panic |
| next — no-op | vol=65 | ActionNext | vol=65 unchanged |
| previous — no-op | vol=65 | ActionPrevious | vol=65 unchanged |

Integration test: send `PlaybackRequestMsg` to app model, assert store state changes
before async cmd resolves.

---

## docs/ARCHITECTURE.md Updates Required

Two stale entries found during design review — must be fixed in the same PR as the implementation:

### 1. Remove stale `n` key from playback routing table

`n` (next track) was removed in story 118 but still appears in ARCHITECTURE.md in two places:

- The key event flow diagram (`Playback keys (Space, n, +, -, s, r, v, ←, →)`)
- The overlay routing precedence table (same list)

Remove `n` from both.

### 2. Add Optimistic Update Pattern section

Add a new subsection under "Data-Carrying Messages" (or after it) documenting the optimistic
update pattern. Key points to cover:

- **When to use it**: user-triggered actions where the new state is fully predictable
  (volume, play/pause, shuffle, repeat). Do NOT use for actions with unpredictable outcomes
  (Next, Previous, any fetch that depends on server data).
- **Where it happens**: inside `app.Update()` — the same place all other Store writes live.
  This is consistent with the Elm contract: commands still never write to the store.
- **Pattern**: read current store state → deep-copy → apply predicted mutation → write back
  → return the API Cmd. The API response overwrites with authoritative state on completion.
- **Revert**: on API error, the existing `fetchPlaybackStateCmd` fired from the error handler
  corrects the store. No separate revert mechanism needed.

Example skeleton to include in the doc:

```go
// In app.Update(), BEFORE returning the API cmd:
case panes.PlaybackRequestMsg:
    a.applyOptimisticUpdate(m.Action)        // sync: store written, UI renders next frame
    return a, a.buildPlaybackAPICmd(m.Action) // async: API call, result overwrites store
```

---

## Out of Scope

- Next / Previous loading state (unpredictable — deferred to future story)
- Instant revert on error (Option B — deferred, simple error path is sufficient)
- Changes to Store interface, pane structs, or theme
