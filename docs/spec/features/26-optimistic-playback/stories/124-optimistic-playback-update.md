---
title: "Optimistic Playback Update on Key Press"
feature: 26-optimistic-playback
status: open
---

## Background

Playback control actions have visible UI lag of ~500ms–1s between key press and the UI
reflecting the new state. The physical device responds immediately (Spotify SDK), but
the UI bar waits for a full roundtrip: debounce → HTTP PUT → fetch → HTTP GET → store
write → render.

For actions where the new state is fully predictable from local state (volume ±1, play,
pause, shuffle toggle, repeat cycle), the UI can write the predicted state immediately
and let the API response overwrite it with authoritative data when it arrives.

`ActionNext` and `ActionPrevious` are explicit no-ops — the next track is determined by
Spotify, not local state.

**Depends on:** feature 25 (volume 1% steps, `volumeStep: 1` on `*App`) — must be merged first.

## Design

### Data Flow

**Before:**
```
PlaybackRequestMsg → buildPlaybackAPICmd(action)
```

**After:**
```
PlaybackRequestMsg → applyOptimisticUpdate(action)   ← store written, UI renders next frame
                   → buildPlaybackAPICmd(action)     ← API fires async, unchanged
```

### New method: `applyOptimisticUpdate`

Location: `internal/app/optimistic.go` (new file, `package app`)

```go
func (a *App) applyOptimisticUpdate(action panes.PlaybackAction) {
    ps := a.store.PlaybackState()
    if ps == nil {
        return
    }

    // Deep-copy: copy the struct value and any pointer fields to avoid aliasing.
    updated := *ps
    if ps.Device != nil {
        dev := *ps.Device
        updated.Device = &dev
    }

    switch action {
    case panes.ActionVolumeUp:
        if updated.Device != nil {
            v := updated.Device.VolumePercent + a.volumeStep
            if v > 100 { v = 100 }
            updated.Device.VolumePercent = v
        }
    case panes.ActionVolumeDown:
        if updated.Device != nil {
            v := updated.Device.VolumePercent - a.volumeStep
            if v < 0 { v = 0 }
            updated.Device.VolumePercent = v
        }
    case panes.ActionPause:
        updated.IsPlaying = false
    case panes.ActionPlay:
        updated.IsPlaying = true
    case panes.ActionToggleShuffle:
        updated.ShuffleState = !updated.ShuffleState
    case panes.ActionCycleRepeat:
        updated.RepeatState = nextRepeatMode(updated.RepeatState)
    case panes.ActionNext, panes.ActionPrevious:
        // no-op: next track is determined by Spotify, not local state
    }

    a.store.SetPlaybackState(&updated)
}
```

`nextRepeatMode` is already defined in `internal/app/commands.go` (same package).

### Wire-up in `handlers.go`

The `PlaybackRequestMsg` case currently reads:
```go
case panes.PlaybackRequestMsg:
    return a, a.buildPlaybackAPICmd(m.Action)
```

Replace with:
```go
case panes.PlaybackRequestMsg:
    a.applyOptimisticUpdate(m.Action)
    return a, a.buildPlaybackAPICmd(m.Action)
```

### Error handling and reconciliation

**Success path (no change):** `PlaybackCmdSentMsg{Err: nil}` → `fetchPlaybackStateCmd` →
`PlaybackStateFetchedMsg` → `store.SetPlaybackState(ps)` overwrites the optimistic value.

**Error path (no change):** `PlaybackCmdSentMsg{Err: non-nil}` → toast fires +
`fetchPlaybackStateCmd` fires → authoritative state overwrites optimistic value within
~200–400ms. Acceptable.

**Rapid keypresses (hold `+`):** each press reads the current (already-optimistic) store
value and applies another step. The 100ms gateway debounce means only the latest value
fires to the API. UI tracks every step. Correct behaviour.

### ARCHITECTURE.md fixes

**Fix 1 — Remove stale `n` key (two locations):**

Line ~209, key event flow diagram:
```
# before
├── Playback keys (Space, n, +, -, s, r, v, ←, →) → always NowPlayingPane
# after
├── Playback keys (Space, +, -, s, r, v, ←, →) → always NowPlayingPane
```

Line ~245, overlay routing precedence table:
```
# before
| 8 | Playback keys | `Space`, `n`, `+`, `-`, `s`, `r`, `v`, `←`, `→` → always NowPlayingPane |
# after
| 8 | Playback keys | `Space`, `+`, `-`, `s`, `r`, `v`, `←`, `→` → always NowPlayingPane |
```

**Fix 2 — Add Optimistic Update Pattern section** after the "Data-Carrying Messages"
paragraph (before the `---` separator at line ~298):

```markdown
### Optimistic Updates

For user-triggered actions where the new state is **fully predictable from local state**,
`Update()` may write an optimistic value to the store immediately — before the API cmd
fires — to give instant UI feedback.

**When to use:** volume up/down, play/pause, shuffle toggle, repeat cycle.
**When NOT to use:** actions whose outcome depends on server data (Next, Previous, any fetch).

**Pattern:**
\`\`\`go
case panes.PlaybackRequestMsg:
    a.applyOptimisticUpdate(m.Action) // sync: store written, UI renders next frame
    return a, a.buildPlaybackAPICmd(m.Action) // async: API call, result overwrites store
\`\`\`

The optimistic write happens in `Update()` — consistent with the Elm contract. Commands
still never write to the store. When the API response arrives via `PlaybackStateFetchedMsg`,
`store.SetPlaybackState()` overwrites the optimistic value with the authoritative one. On
API error, the `fetchPlaybackStateCmd` fired from the `PlaybackCmdSentMsg` error handler
corrects the store automatically.
```

## Acceptance Criteria

- [ ] `applyOptimisticUpdate` called in `PlaybackRequestMsg` handler before `buildPlaybackAPICmd`
- [ ] Volume up: store reflects `vol + 1` (clamped at 100) immediately after `Update()` returns
- [ ] Volume down: store reflects `vol - 1` (clamped at 0) immediately after `Update()` returns
- [ ] Volume actions with `ps.Device == nil` do not panic
- [ ] `nil` playback state in store → `applyOptimisticUpdate` returns without panic, store stays nil
- [ ] `ActionNext` and `ActionPrevious` are no-ops (store value unchanged)
- [ ] Play, pause, shuffle toggle, repeat cycle mutations all correct per test table
- [ ] Stale `n` removed from ARCHITECTURE.md key event diagram and routing table
- [ ] Optimistic Update Pattern section present in ARCHITECTURE.md
- [ ] `make ci` passes (lint + tests + ≥ 80% coverage)

## Tasks

### ARCHITECTURE.md cleanup

- [ ] In `docs/ARCHITECTURE.md` line ~209, remove `n,` from the playback keys list in the key event flow diagram
  - verify: `grep -n "Playback keys" docs/ARCHITECTURE.md` shows `n` is absent
- [ ] In `docs/ARCHITECTURE.md` line ~245, remove `` `n`, `` from the overlay routing table row
  - verify: same grep
- [ ] Commit: `git add docs/ARCHITECTURE.md && git commit -m "docs(arch): remove stale n key from playback routing table"`

### Write failing tests

- [ ] Create `internal/app/optimistic_test.go` (`package app_test`) with `TestApplyOptimisticUpdate`
      table-driven test covering all 13 cases below. Use `newTestApp()` from `staleness_test.go` —
      do not redefine it. Inject initial state via `a.Store().SetPlaybackState(&initial)`, then
      call `a.Update(panes.PlaybackRequestMsg{Action: tt.action})`, then assert `a.Store().PlaybackState()`.

  Test cases:
  | Name | Initial | Action | Expected |
  |------|---------|--------|----------|
  | volume up | vol=65, device non-nil | ActionVolumeUp | vol=66 |
  | volume up at max | vol=100 | ActionVolumeUp | vol=100 (clamped) |
  | volume down | vol=65 | ActionVolumeDown | vol=64 |
  | volume down at floor | vol=0 | ActionVolumeDown | vol=0 (clamped) |
  | pause | IsPlaying=true | ActionPause | IsPlaying=false |
  | play | IsPlaying=false | ActionPlay | IsPlaying=true |
  | shuffle on | ShuffleState=false | ActionToggleShuffle | ShuffleState=true |
  | shuffle off | ShuffleState=true | ActionToggleShuffle | ShuffleState=false |
  | repeat cycle off→context | RepeatState="off" | ActionCycleRepeat | RepeatState="context" |
  | nil device no panic | device=nil | ActionVolumeUp | no panic, device still nil |
  | next no-op | vol=65 | ActionNext | vol=65 unchanged |
  | previous no-op | vol=65 | ActionPrevious | vol=65 unchanged |

- [ ] Add `TestApplyOptimisticUpdate_NilPlaybackState_DoesNotPanic` — store has no playback
      state; call `a.Update(panes.PlaybackRequestMsg{Action: panes.ActionVolumeUp})`; assert
      no panic and `a.Store().PlaybackState()` is still nil. This is a separate named test
      (not a table row) because no initial state is injected — the store starts nil by default.

- [ ] Run: `go test ./internal/app/ -run TestApplyOptimisticUpdate -v` → **FAIL** (expected —
      `applyOptimisticUpdate` does not exist yet, store value unchanged)

### Implement `applyOptimisticUpdate`

- [ ] Create `internal/app/optimistic.go` (`package app`) with the method body shown in Design above.
      Export comment required. All `PlaybackAction` values must be handled (exhaustiveness linters).
  - test: `go test ./internal/app/ -run TestApplyOptimisticUpdate -v` → **PASS** (all 13 cases)
  - test: `go test ./internal/app/ -run TestApplyOptimisticUpdate_NilPlaybackState -v` → **PASS**

### Wire into handlers.go

- [ ] In `internal/app/handlers.go`, find the `PlaybackRequestMsg` case and add
      `a.applyOptimisticUpdate(m.Action)` on the line before the `return` statement.
  - test: `go test ./internal/app/ -v` → all existing tests pass

- [ ] Commit:
  ```
  git add internal/app/optimistic.go internal/app/optimistic_test.go internal/app/handlers.go
  git commit -m "feat(playback): add optimistic store update on key press"
  ```

### ARCHITECTURE.md — Optimistic Update Pattern section

- [ ] In `docs/ARCHITECTURE.md`, insert the `### Optimistic Updates` subsection after the
      "Data-Carrying Messages" paragraph and before the next `---` separator (see Design above for
      exact content).
  - verify: `grep -n "Optimistic" docs/ARCHITECTURE.md` → two hits (heading + "When NOT to use" line)
- [ ] Commit: `git add docs/ARCHITECTURE.md && git commit -m "docs(arch): document optimistic update pattern"`

### CI gate

- [ ] `make ci` passes (lint + tests + coverage ≥ 80%)
  - If `exhaustive` linter complains on the `switch action`: all `PlaybackAction` values have
    explicit cases — `ActionNext` and `ActionPrevious` share an explicit no-op case. Should be satisfied.
  - If coverage drops: run `make test-coverage`, add missing cases to `optimistic_test.go`.
