---
title: "Interactive Seek Bar"
feature: 03-playback
status: open
---

## Background

The NowPlaying pane renders a gradient seek bar showing track progress, but users cannot interact with it. The Spotify Web API's `PUT /v1/me/player/seek` endpoint (with `position_ms`) is already available through `Player.Seek()` in `internal/api/player.go:100-111`. No UI integration connects key presses to that API call.

The `GradientVolumeBar` already implements the exact debounce pattern this feature needs: local state → 300ms debounce tick → intent message → API command → confirm/cancel. This story extracts that pattern into a reusable `DebounceTracker` and mirrors it for seek.

## Design

### Keybinding changes

| Key | Old Action | New Action |
|-----|-----------|------------|
| `←` | Previous track | Seek back 5s |
| `→` | Next track | Seek forward 5s |
| `Shift+←` | — | Previous track |
| `Shift+→` | — | Next track |
| `p` | Previous track | Previous track (unchanged) |

**Bubble Tea key detection:** Shift+arrows are separate `KeyType` constants (`tea.KeyShiftLeft`, `tea.KeyShiftRight`) — NOT modifier flags on `tea.KeyLeft`/`tea.KeyRight`. The `handleKey` switch must check `tea.KeyShiftLeft`/`tea.KeyShiftRight` before `tea.KeyLeft`/`tea.KeyRight` (more specific first).

All three keybinding docs must update in the same commit per CLAUDE.md rule 15:
- `README.md` Keybindings section
- `docs/system/design.md` §17
- `internal/ui/panes/help_overlay.go` `helpContent`

### Architecture

#### DebounceTracker — shared state machine

Extract the identical debounce logic from `GradientVolumeBar` into a reusable struct in `internal/ui/components/debounce.go`:

```go
type DebounceTracker struct {
    current    int  // displayed value: pending or last confirmed
    hasPending bool // true while a debounce tick is in flight
    seq        int  // monotonically increasing; stale ticks have lower seq
}

func (d *DebounceTracker) HandleKey(delta, confirmed, min, max int) int
func (d *DebounceTracker) HandleDebounce(tickSeq int) (matched bool, targetVal int, intentSeq int)
func (d *DebounceTracker) ConfirmFromAPI(intentSeq, val int)
func (d *DebounceTracker) CancelPending(intentSeq, confirmed int)
func (d *DebounceTracker) SetConfirmed(val int)
func (d *DebounceTracker) Current() int
func (d *DebounceTracker) HasPending() bool
```

`GradientVolumeBar` removes `currentVol`, `hasPending`, `seq` and embeds `DebounceTracker` instead. All volume methods delegate to it. This is a pure refactor — no volume behavior changes.

#### GradientSeekBar — add interactive state

`GradientSeekBar` gains `DebounceTracker` embed + `trackDuration int`. Methods mirror the volume bar pattern:

| Method | Purpose |
|--------|---------|
| `HandleKey(deltaMs, confirmedMs, durationMs int) tea.Cmd` | DebounceTracker.HandleKey(delta, confirmed, 0, durationMs) → returns SeekDebounceTickMsg cmd |
| `HandleDebounce(msg SeekDebounceTickMsg) (matched, targetMs, intentSeq int)` | Delegates to DebounceTracker.HandleDebounce |
| `ConfirmFromAPI(intentSeq, posMs int)` | Delegates to DebounceTracker.ConfirmFromAPI |
| `CancelPending(intentSeq, confirmedMs int)` | Delegates to DebounceTracker.CancelPending |
| `SetPositionConfirmed(posMs int)` | Delegates to DebounceTracker.SetConfirmed |
| `SetTrackDuration(ms int)` | Sets clamping bound |
| `Render(progressMs, durationMs int) string` | Uses `Current()` when pending, parameter otherwise |

#### New message types

`internal/ui/components/gradient.go`:
```go
type SeekDebounceTickMsg struct {
    TargetMs int
    Seq      int
}
```

`internal/ui/panes/messages.go`:
```go
type SeekIntentMsg struct {
    TargetMs int
    Seq      int
}

type SeekAppliedMsg struct {
    PosMs int
    Seq   int
    Err   error
}
```

#### Data flow

```
User taps ←/→
  → handleKey() calls seekBar.HandleKey(±5000, confirmed, duration)
    → adjusts current, increments seq
    → returns tea.Tick(300ms) → SeekDebounceTickMsg

300ms later
  → SeekDebounceTickMsg arrives at NowPlayingPane.Update()
    → seekBar.HandleDebounce(msg)
      → matched: return SeekIntentMsg{TargetMs, Seq}
      → stale: no-op

SeekIntentMsg arrives at App.Update() (handlers.go)
  → premium check
  → buildSeekCmd(targetMs, seq) → Player.Seek(ctx, targetMs)
  → returns SeekAppliedMsg{PosMs, Seq, Err}

SeekAppliedMsg arrives at App.Update()
  → on success: seekBar.ConfirmFromAPI(seq, posMs), reset localProgressMs, dispatch interactive poll
  → on error: seekBar.CancelPending(seq, confirmedProgress), toast notification (429/401/generic)

Each 1s poll
  → PlaybackStateFetchedMsg → seekBar.SetPositionConfirmed(ps.ProgressMs)
    → if hasPending && posMs == current: clear pending
    → if !hasPending: update current to server value
```

#### App command builder

`buildSeekCmd` in `internal/app/commands.go` mirrors `buildSetVolumeCmd`:
```go
func (a *App) buildSeekCmd(targetMs, intentSeq int) tea.Cmd {
    player := a.player
    return func() tea.Msg {
        if player == nil {
            return panes.SeekAppliedMsg{Seq: intentSeq, Err: errNilClient}
        }
        ctx := api.WithPriority(context.Background(), api.Interactive)
        err := player.Seek(ctx, targetMs)
        if err != nil {
            return panes.SeekAppliedMsg{Seq: intentSeq, Err: err}
        }
        return panes.SeekAppliedMsg{PosMs: targetMs, Seq: intentSeq}
    }
}
```

#### NowPlayingPane changes

**`handleKey()`** — rebind arrows. Switch cases must check `tea.KeyShiftLeft`/`tea.KeyShiftRight` BEFORE `tea.KeyLeft`/`tea.KeyRight`:
- `tea.KeyShiftLeft` → `emitPlaybackRequest(ActionPrevious)`
- `tea.KeyShiftRight` → `emitPlaybackRequest(ActionNext)`
- `tea.KeyLeft` → seek back 5s (if track duration > 0)
- `tea.KeyRight` → seek forward 5s (if track duration > 0)
- `"p"` → `emitPlaybackRequest(ActionPrevious)` (unchanged, now separate from Left arrow)

**`Update()`** — add message handlers:
- `components.SeekDebounceTickMsg` → delegate to seekBar.HandleDebounce, emit SeekIntentMsg if matched
- `SeekAppliedMsg` → confirm or cancel seek bar, reset `localProgressMs` on success

**`handlePlaybackFetched()`** — add seek bar sync after volume bar sync:
```go
if ps.Item != nil {
    p.seekBar.SetTrackDuration(ps.Item.DurationMs)
}
p.seekBar.SetPositionConfirmed(ps.ProgressMs)
```

**`handleTick()`** — when `seekBar.HasPending()`, skip incrementing `localProgressMs` and use `seekBar.Current()` for display instead:

```go
func (p *NowPlayingPane) handleTick() (*NowPlayingPane, tea.Cmd) {
    ps := p.store.PlaybackState()
    if ps != nil && ps.IsPlaying {
        if !p.seekBar.HasPending() {
            p.localProgressMs += 1000
            if ps.Item != nil && p.localProgressMs > ps.Item.DurationMs {
                p.localProgressMs = ps.Item.DurationMs
            }
        }
    }
    return p, nil
}
```

**`SetTheme()`** — restore seek bar state after theme change:
```go
if ps := p.store.PlaybackState(); ps != nil {
    p.seekBar.SetPositionConfirmed(ps.ProgressMs)
    if ps.Item != nil {
        p.seekBar.SetTrackDuration(ps.Item.DurationMs)
    }
}
```

**New helpers:**
- `confirmedProgress(s state.StateReader) int` — returns `PlaybackState.ProgressMs` or `0`
- `confirmedDuration(s state.StateReader) int` — returns `PlaybackState.Item.DurationMs` or `0`

#### App routing changes

`internal/app/routing.go`:
- `isPlaybackKey()`: Add `tea.KeyShiftLeft`, `tea.KeyShiftRight` (so Shift+arrows also route to NowPlaying)
- `isPremiumOnlyPlaybackKey()`: Add `tea.KeyShiftLeft`, `tea.KeyShiftRight`, `tea.KeyLeft`, `tea.KeyRight` (all arrow keys are premium-gated for seek/prev/next)

**Important:** `tea.KeyLeft` and `tea.KeyRight` are already in `isPremiumOnlyPlaybackKey`. The new additions are `tea.KeyShiftLeft` and `tea.KeyShiftRight`.

#### Error handling

Same pattern as volume:
- **429 Rate Limited** → back off for `Retry-After` seconds, emit `"ratelimit"` toast via `a.alerts.NewAlertCmd`
- **401 Unauthorized** → refresh token, retry once
- **403 Forbidden** → emit `"warning"` toast "Spotify Premium required"
- **Other errors** → emit error toast via `a.errorMapper.Map(uikit.OpSeek, m.Err)` with `fetchPlaybackStateCmd(Interactive)` to reconcile

#### Clamping

- Seek below 0 → clamp to 0
- Seek past `trackDurationMs` → clamp to `trackDurationMs`
- If `trackDurationMs` is 0 or unknown → do not seek (no-op, HandleKey returns nil cmd)

#### localProgressMs reset

On successful seek, `localProgressMs` must be set to the target position immediately so the seek bar jumps visually. The next 1s poll reconciles any drift.

### Comparison with volume implementation

| Aspect | Volume | Seek |
|--------|--------|------|
| Component | GradientVolumeBar | GradientSeekBar |
| Local state | DebounceTracker embed | DebounceTracker embed |
| Debounce msg | VolumeDebounceTickMsg | SeekDebounceTickMsg |
| Intent msg | VolumeIntentMsg | SeekIntentMsg |
| Applied msg | VolumeAppliedMsg | SeekAppliedMsg |
| API call | Player.SetVolume(ctx, vol) | Player.Seek(ctx, positionMs) |
| Debounce delay | 300ms | 300ms |
| Step size | ±1% | ±5000ms (5 seconds) |
| Clamping | 0–100 | 0–trackDurationMs |
| Premium check | Yes | Yes |
| Error handling | Toast on 429/401/generic | Same pattern |
| Error op type | uikit.OpVolume | uikit.OpSeek (new) |
| Extra param | None | trackDurationMs for clamping |

## Files to add

| File | Purpose |
|------|---------|
| `internal/ui/components/debounce.go` | DebounceTracker struct with HandleKey, HandleDebounce, ConfirmFromAPI, CancelPending, SetConfirmed |
| `internal/ui/components/debounce_test.go` | Unit tests for DebounceTracker |
| `internal/app/seek_test.go` | Handler routing tests for SeekIntentMsg/SeekAppliedMsg |

## Files to modify

| File | Change |
|------|--------|
| `internal/ui/components/gradient.go` | Refactor GradientVolumeBar to embed DebounceTracker; add SeekDebounceTickMsg; add debounce fields/methods to GradientSeekBar |
| `internal/ui/components/gradient_test.go` | Update volume bar tests to work with DebounceTracker embed; add GradientSeekBar debounce tests |
| `internal/ui/panes/messages.go` | Add SeekIntentMsg, SeekAppliedMsg |
| `internal/ui/panes/nowplaying.go` | Rebind ←/→ to seek, add Shift+←/Shift+→ for prev/next, add SeekDebounceTickMsg/SeekAppliedMsg handlers, sync seekBar in handlePlaybackFetched, guard handleTick |
| `internal/app/commands.go` | Add buildSeekCmd |
| `internal/app/handlers.go` | Add SeekDebounceTickMsg forwarding, SeekIntentMsg/SeekAppliedMsg handlers |
| `internal/app/routing.go` | Update isPlaybackKey/isPremiumOnlyPlaybackKey for Shift+arrow keybindings |
| `internal/uikit/error_mapper.go` | Add OpSeek to Operation const, opTitle map, opForbiddenBody map |
| `README.md` | Update keybindings table |
| `docs/system/design.md` | Update §16 and §17 keybinding tables |
| `internal/ui/panes/help_overlay.go` | Update helpContent keybinding display |

## Tasks

### Task 1: Create DebounceTracker shared component

**Files:** Create `internal/ui/components/debounce.go`, `internal/ui/components/debounce_test.go`

1. Write failing tests for DebounceTracker (HandleKey clamping, seq increment, stale-tick rejection, SetConfirmed pending/confirmed transitions, ConfirmFromAPI/CancelPending guards, no-op when min≥max)
2. Verify tests fail (compilation error — type not yet defined)
3. Implement DebounceTracker with all methods per the design above
4. Verify tests pass
5. Commit: `feat(seek): add DebounceTracker shared state machine for debounce pattern`

### Task 2: Refactor GradientVolumeBar to embed DebounceTracker

**Files:** Modify `internal/ui/components/gradient.go`

1. Replace `currentVol`, `hasPending`, `seq` fields with `DebounceTracker` embed
2. Delegate all methods (HandleKey, HandleDebounce, ConfirmFromAPI, CancelPending, SetConfirmed) to `DebounceTracker`
3. Update `Render()` to use `b.Current()` instead of `b.currentVol`
4. HandleKey returns `tea.Cmd` (wraps DebounceTracker.HandleKey return value into tick cmd)
5. Verify existing volume bar tests still pass
6. Commit: `refactor(volume): embed DebounceTracker in GradientVolumeBar`

### Task 3: Add seek debounce messages and GradientSeekBar interactive state

**Files:** Modify `internal/ui/components/gradient.go`, `internal/ui/panes/messages.go`

1. Add `SeekDebounceTickMsg` after `VolumeDebounceTickMsg` in gradient.go
2. Add `DebounceTracker` embed and `trackDuration int` to `GradientSeekBar`
3. Add seek methods: HandleKey, HandleDebounce, ConfirmFromAPI, CancelPending, SetPositionConfirmed, SetTrackDuration
4. Update `GradientSeekBar.Render()` to use `b.Current()` when pending
5. Add `SeekIntentMsg`, `SeekAppliedMsg` to messages.go
6. Write GradientSeekBar debounce unit tests in gradient_test.go
7. Commit: `feat(seek): add SeekDebounceTickMsg, seek state to GradientSeekBar, SeekIntentMsg/SeekAppliedMsg`

### Task 4: Wire up seek in App layer (handlers + commands)

**Files:** Modify `internal/app/commands.go`, `internal/app/handlers.go`, `internal/uikit/error_mapper.go`

1. Add `buildSeekCmd` to commands.go (mirrors buildSetVolumeCmd)
2. Add `SeekDebounceTickMsg` forwarding in handlers.go (mirrors VolumeDebounceTickMsg)
3. Add `SeekIntentMsg` handler: premium check → buildSeekCmd
4. Add `SeekAppliedMsg` handler: confirm/cancel bar → error mapping (mirrors VolumeAppliedMsg)
5. Add `OpSeek` to `uikit.Operation`, `opTitle`, and `opForbiddenBody` maps
6. Compile check: `go build ./...`
7. Commit: `feat(seek): add buildSeekCmd and SeekIntent/SeekApplied handlers in app layer`

### Task 5: Wire up seek in NowPlayingPane (keybindings + message handlers)

**Files:** Modify `internal/ui/panes/nowplaying.go`, `internal/app/routing.go`

1. Update `handleKey()`: add `tea.KeyShiftLeft` → previous, `tea.KeyShiftRight` → next, `tea.KeyLeft` → seek back 5s, `tea.KeyRight` → seek forward 5s. Check shift variants BEFORE plain arrows. Separate `"p"` from Left arrow.
2. Add `SeekDebounceTickMsg` and `SeekAppliedMsg` handlers in `Update()`
3. Add `confirmedProgress()` and `confirmedDuration()` helpers
4. Sync seekBar in `handlePlaybackFetched()` (SetTrackDuration + SetPositionConfirmed)
5. Restore seek bar state in `SetTheme()`
6. Update `handleTick()`: skip `localProgressMs` increment when `seekBar.HasPending()`
7. Update `routing.go`: add `tea.KeyShiftLeft`, `tea.KeyShiftRight` to `isPlaybackKey()`; add them to `isPremiumOnlyPlaybackKey()` (alongside existing `tea.KeyLeft`/`tea.KeyRight`)
8. Compile check
9. Commit: `feat(seek): wire seek keybindings and handlers in NowPlayingPane`

### Task 6: Add seek handler tests

**Files:** Create `internal/app/seek_test.go`

1. Write seek handler tests mirroring volume test pattern:
   - SeekIntentMsg calls Seek API with correct position
   - SeekIntentMsg nil player → errNilClient
   - SeekIntentMsg non-premium → blocked
   - SeekDebounceTickMsg forwards to pane
   - SeekAppliedMsg success → dispatches interactive poll
   - SeekAppliedMsg 429 → rate limit handling
2. Commit: `test(seek): add seek handler tests mirroring volume pattern`

### Task 7: Update keybinding documentation (all 3 locations)

**Files:** Modify `README.md`, `docs/system/design.md`, `internal/ui/panes/help_overlay.go`

Changes in all 3 locations:
- `←` → "Seek back 5s" (was: "Previous track")
- `→` → "Seek forward 5s" (was: "Next track")
- Add `Shift+←` → "Previous track"
- Add `Shift+→` → "Next track"
- Keep `p` → "Cycle preset" in global keys (unchanged — `p` was never "previous track" in the keybinding table; it was only in the help overlay's playback section)

**Important:** The help overlay currently shows `{al + " / " + ar, "Prev / Next"}` for arrows. This must change to show seek and Shift+arrows. Update to separate entries.

1. Update README.md keybindings section
2. Update docs/system/design.md §16 and §17
3. Update internal/ui/panes/help_overlay.go helpContent
4. Commit: `docs(keybindings): rebind arrows to seek, Shift+arrows to prev/next track`

### Task 8: Run full CI suite

1. `make ci` — lint + tests + 80% coverage
2. Fix any issues
3. Final commit if needed

## Testing plan

1. **DebounceTracker unit tests** — HandleKey clamping, seq increment, stale-tick rejection, SetConfirmed pending/confirmed transitions, ConfirmFromAPI/CancelPending guards
2. **GradientSeekBar unit tests** — HandleKey, HandleDebounce, ConfirmFromAPI, CancelPending, SetConfirmed, stale-tick rejection, clamping (below 0, past duration), zero-duration no-op, render uses pending when active
3. **GradientVolumeBar regression tests** — verify existing volume behavior unchanged after refactor to embed DebounceTracker
4. **SeekIntentMsg/SeekAppliedMsg routing** — handler tests in app/handlers
5. **Key routing** — ←/→ emit seek, Shift+←/Shift+→ emit prev/next, premium gate
6. **Premium check** — SeekIntentMsg blocked when not premium, toast shown
7. **Error paths** — 429, 401, 403, nil client
8. **Poll reconciliation** — SetPositionConfirmed clears pending when server matches local
9. **handleTick guard** — localProgressMs not incremented when seekBar.HasPending()

## Verification

- `make ci` passes (lint + tests + 80% coverage)
- Left arrow seeks back 5s in the seek bar visually (optimistic update)
- Right arrow seeks forward 5s visually
- Shift+Left goes to previous track
- Shift+Right goes to next track
- `p` key still cycles presets (global) — NOT previous track
- Volume bar (+/-) behavior is unchanged (regression test)
- Non-premium users see "Spotify Premium required" toast on seek
- Rapid key presses debounce to one API call