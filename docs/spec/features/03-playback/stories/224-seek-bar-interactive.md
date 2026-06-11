---
title: "Interactive Seek Bar"
feature: 03-playback
status: done
---

## Background

The NowPlaying pane renders a gradient seek bar showing track progress, but users
cannot interact with it. The Spotify Web API's `PUT /v1/me/player/seek` endpoint
(with `position_ms`) is already available through `Player.Seek()` in
`internal/api/player.go:100-111`. No UI integration connects key presses to that
API call.

The `GradientVolumeBar` already implements the exact debounce pattern this feature
needs: local state → 300 ms debounce tick → intent message → API command →
confirm/cancel. This story extracts that pattern into a reusable `DebounceTracker`
and mirrors it for seek.

**Keybinding rebinding:** `←` and `→` currently skip to previous/next track. This
story rebinds them to seek back/forward 5 s and moves previous/next to `Shift+←` /
`Shift+→`. The `"p"` case in `handleKey()` that combined with `tea.KeyLeft` for
`ActionPrevious` is dead code (routing.go intercepts `"p"` for preset cycling before
the pane ever sees it) and is removed.

**Bubble Tea key detection:** Shift+arrows are separate `KeyType` constants
(`tea.KeyShiftLeft`, `tea.KeyShiftRight`) — NOT modifier flags on `tea.KeyLeft` /
`tea.KeyRight`. The `handleKey` switch must check shift variants before plain arrows
(more specific first).

**Depends on:** nothing — self-contained.

---

## Design

### Change 1 — DebounceTracker shared component

**Files:** `internal/ui/components/debounce.go`, `internal/ui/components/debounce_test.go`

Extract the debounce state machine from `GradientVolumeBar` into a standalone struct
so both `GradientVolumeBar` and `GradientSeekBar` can embed it.

```go
// internal/ui/components/debounce.go
type DebounceTracker struct {
    current    int  // displayed value: pending or last confirmed
    hasPending bool // true while a debounce tick is in flight
    seq        int  // monotonically increasing; stale ticks have a lower seq
}

func (d *DebounceTracker) Current() int
func (d *DebounceTracker) HasPending() bool
func (d *DebounceTracker) HandleKey(delta, confirmed, min, max int) int
func (d *DebounceTracker) HandleDebounce(tickSeq int) (matched bool, targetVal int, intentSeq int)
func (d *DebounceTracker) ConfirmFromAPI(intentSeq, val int)
func (d *DebounceTracker) CancelPending(intentSeq, confirmed int)
func (d *DebounceTracker) SetConfirmed(val int)
```

`HandleKey` returns the new seq number (or -1 if min ≥ max — caller creates the
tick cmd). `HandleDebounce` increments seq as a double-fire guard. `ConfirmFromAPI`
and `CancelPending` guard on `seq == intentSeq + 1` so concurrent bursts don't
clobber each other. `SetConfirmed` clears `hasPending` only when the poll value
matches `current` — blocking stale polls from snapping the bar back.

Unit tests cover: HandleKey clamping, accumulation from pending state, no-op when
min ≥ max, HandleDebounce stale rejection, double-fire guard, SetConfirmed
pending/confirmed transitions, ConfirmFromAPI seq match/mismatch, CancelPending
seq match/mismatch.

---

### Change 2 — Refactor GradientVolumeBar to embed DebounceTracker

**File:** `internal/ui/components/gradient.go`

Replace `currentVol`, `hasPending`, `seq` fields with `DebounceTracker` embed.
All methods delegate:

- `HandleKey(delta, confirmedVol int) tea.Cmd` — calls
  `b.DebounceTracker.HandleKey(delta, confirmedVol, 0, 100)`, wraps result into
  `VolumeDebounceTickMsg`
- `HandleDebounce(m VolumeDebounceTickMsg)` — delegates to
  `b.DebounceTracker.HandleDebounce(m.Seq)`
- `ConfirmFromAPI`, `CancelPending`, `SetConfirmed` — delegate to embedded tracker
- `Render()` — reads `b.Current()` instead of `b.currentVol`

No behavior change — pure refactor. Existing volume bar tests must pass unchanged.

---

### Change 3 — GradientSeekBar interactive state + seek messages

**Files:** `internal/ui/components/gradient.go`, `internal/ui/panes/messages.go`

Add `SeekDebounceTickMsg` in `gradient.go` (alongside `VolumeDebounceTickMsg`):

```go
type SeekDebounceTickMsg struct {
    TargetMs int
    Seq      int
}
```

Add `DebounceTracker` embed and `trackDuration int` to `GradientSeekBar`:

```go
type GradientSeekBar struct {
    DebounceTracker
    th            theme.Theme
    width         int
    trackDuration int
}
```

Add seek methods mirroring the volume pattern:

| Method | Purpose |
|--------|---------|
| `HandleKey(deltaMs, confirmedMs, durationMs int) tea.Cmd` | DebounceTracker.HandleKey(delta, confirmed, 0, durationMs) → SeekDebounceTickMsg cmd |
| `HandleDebounce(m SeekDebounceTickMsg) (matched, targetMs, intentSeq int)` | Delegates to DebounceTracker.HandleDebounce |
| `ConfirmFromAPI(intentSeq, posMs int)` | Delegates to DebounceTracker.ConfirmFromAPI |
| `CancelPending(intentSeq, confirmedMs int)` | Delegates to DebounceTracker.CancelPending |
| `SetPositionConfirmed(posMs int)` | Delegates to DebounceTracker.SetConfirmed |
| `SetTrackDuration(ms int)` | Sets clamping bound |
| `Render(progressMs, durationMs int) string` | Uses `Current()` when `HasPending()`, parameter otherwise |

`HandleKey` returns nil when `durationMs` is 0 (no valid range — no-op).

Add `SeekIntentMsg` and `SeekAppliedMsg` to `messages.go`:

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

Unit tests for GradientSeekBar: HandleKey updates immediately, seeks backward,
clamps at zero, clamps at duration, no-op when zero duration, HandleDebounce
stale rejection, HandleDebounce current acceptance, ConfirmFromAPI seq match,
CancelPending seq match, SetConfirmed clears pending on match, Render uses
pending when active, Render uses parameter when not pending.

---

### Change 4 — App layer: buildSeekCmd + handlers

**Files:** `internal/app/commands.go`, `internal/app/handlers.go`, `internal/uikit/error_mapper.go`

Add `buildSeekCmd` in `commands.go` (mirrors `buildSetVolumeCmd`):

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

Add message routing in `handlers.go`:

- `components.SeekDebounceTickMsg` — forward to `NowPlayingPane.Update(m)`, return
  pane command (same pattern as `VolumeDebounceTickMsg`)
- `panes.SeekIntentMsg` — premium check → `buildSeekCmd(m.TargetMs, m.Seq)`
- `panes.SeekAppliedMsg` — forward to `NowPlayingPane.Update(m)`, then: on error
  route through existing error handlers (429 → `RateLimitedMsg`, 401 →
  `unauthorizedMsg`, generic → error mapper + interactive poll); on success batch
  pane command with `fetchPlaybackStateCmd(Interactive)`

Add `OpSeek` to `uikit.Operation`, `opTitle`, and `opForbiddenBody`:

```go
OpSeek Operation = "seek"

// opTitle:
OpSeek: "Seek failed",

// opForbiddenBody:
OpSeek: "Premium required for seek control.",
```

---

### Change 5 — NowPlayingPane: rebind arrows, add seek handlers, guard handleTick

**Files:** `internal/ui/panes/nowplaying.go`, `internal/app/routing.go`

**handleKey() — rebind arrows, remove dead `"p"` case:**

```go
// Before:
case msg.Type == tea.KeyRight:
    return p, emitPlaybackRequest(ActionNext)
case msg.Type == tea.KeyRunes && string(msg.Runes) == "p",
    msg.Type == tea.KeyLeft:
    return p, emitPlaybackRequest(ActionPrevious)

// After:
case msg.Type == tea.KeyShiftLeft:
    return p, emitPlaybackRequest(ActionPrevious)
case msg.Type == tea.KeyShiftRight:
    return p, emitPlaybackRequest(ActionNext)
case msg.Type == tea.KeyLeft:
    if ps := p.store.PlaybackState(); ps != nil && ps.Item != nil && ps.Item.DurationMs > 0 {
        confirmed := ps.ProgressMs
        if p.seekBar.HasPending() {
            confirmed = p.seekBar.Current()
        }
        cmd := p.seekBar.HandleKey(-5000, confirmed, ps.Item.DurationMs)
        return p, cmd
    }
    return p, nil
case msg.Type == tea.KeyRight:
    if ps := p.store.PlaybackState(); ps != nil && ps.Item != nil && ps.Item.DurationMs > 0 {
        confirmed := ps.ProgressMs
        if p.seekBar.HasPending() {
            confirmed = p.seekBar.Current()
        }
        cmd := p.seekBar.HandleKey(+5000, confirmed, ps.Item.DurationMs)
        return p, cmd
    }
    return p, nil
```

The `"p"` case is removed entirely — it was dead code because routing.go
intercepts `"p"` for preset cycling before the pane sees it.

**Update() — add message handlers:**

```go
case components.SeekDebounceTickMsg:
    if matched, targetMs, seq := p.seekBar.HandleDebounce(m); matched {
        return p, func() tea.Msg { return SeekIntentMsg{TargetMs: targetMs, Seq: seq} }
    }
    return p, nil

case SeekAppliedMsg:
    if m.Err != nil {
        p.seekBar.CancelPending(m.Seq, confirmedProgress(p.store))
    } else {
        p.seekBar.ConfirmFromAPI(m.Seq, m.PosMs)
        p.localProgressMs = m.PosMs
    }
    return p, nil
```

**New helpers:**

```go
func confirmedProgress(s state.StateReader) int {
    if ps := s.PlaybackState(); ps != nil {
        return ps.ProgressMs
    }
    return 0
}

func confirmedDuration(s state.StateReader) int {
    if ps := s.PlaybackState(); ps != nil && ps.Item != nil {
        return ps.Item.DurationMs
    }
    return 0
}
```

**handlePlaybackFetched — sync seek bar:**

```go
if ps.Item != nil {
    p.seekBar.SetTrackDuration(ps.Item.DurationMs)
}
p.seekBar.SetPositionConfirmed(ps.ProgressMs)
```

**SetTheme — restore seek bar state:**

```go
if ps := p.store.PlaybackState(); ps != nil {
    p.seekBar.SetPositionConfirmed(ps.ProgressMs)
    if ps.Item != nil {
        p.seekBar.SetTrackDuration(ps.Item.DurationMs)
    }
}
```

**handleTick — guard localProgressMs when seek is pending:**

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

**routing.go — add shift arrows to key checks:**

In `isPlaybackKey`: add `tea.KeyShiftLeft`, `tea.KeyShiftRight`.
In `isPremiumOnlyPlaybackKey`: add `tea.KeyShiftLeft`, `tea.KeyShiftRight`
(plain arrows are already there).

---

### Change 6 — Keybinding documentation (all 3 locations)

**Files:** `README.md`, `docs/system/design.md`, `internal/ui/panes/help_overlay.go`

Changes in all 3 locations:
- `←` → "Seek back 5s" (was: "Previous track")
- `→` → "Seek forward 5s" (was: "Next track")
- Add `Shift+←` → "Previous track"
- Add `Shift+→` → "Next track"

The help overlay currently shows `{al + " / " + ar, "Prev / Next"}` for arrows.
Update to separate entries showing seek and shift-arrow bindings.

---

## Data Flow

```
User taps ←/→
    │
    ▼
nowplaying.handleKey()
    │  confirmed := confirmedProgress(store)
    │  if seekBar.HasPending(): confirmed = seekBar.Current()
    │  calls seekBar.HandleKey(±5000, confirmed, duration)
    │  seek bar current updates → renders new position immediately
    │  returns tea.Tick(300ms, SeekDebounceTickMsg{target, seq})
    │
    ▼ [300 ms later — or stale tick discarded by seq check]
SeekDebounceTickMsg arrives at App.Update()
    │
    ▼
handlers.go → np.Update(m) → seekBar.HandleDebounce(m)
    │  seq matches? → emit SeekIntentMsg{TargetMs, Seq}
    │  stale?       → return nil (discard)
    │
    ▼
handlers.go: SeekIntentMsg → premium check → buildSeekCmd(targetMs, seq)
    │
    ▼
HTTP PUT /v1/me/player/seek?position_ms=N   ← 1 call per burst
    │
    ▼
SeekAppliedMsg{PosMs, Seq, Err}
    │
    ▼  success: seekBar.ConfirmFromAPI(seq, posMs), localProgressMs = posMs,
    │          fetchPlaybackStateCmd(Interactive)
    ▼  error:   seekBar.CancelPending(seq, confirmedProgress), toast
    │
    ▼ [next 1s poll — PlaybackStateFetchedMsg]
    │
    ▼
seekBar.SetPositionConfirmed(ps.ProgressMs)  ← reconcile
    │  if hasPending && posMs == current: clear pending
    │  if !hasPending: update current to server value
```

---

## Acceptance Criteria

- [ ] Pressing `←` seeks back 5 s; bar updates instantly (optimistic)
- [ ] Pressing `→` seeks forward 5 s; bar updates instantly (optimistic)
- [ ] Pressing `Shift+←` skips to previous track
- [ ] Pressing `Shift+→` skips to next track
- [ ] Pressing `p` cycles preset (global) — NOT previous track; dead `"p"` case removed from nowplaying
- [ ] Rapid `←`/`→` presses debounce to exactly one API call per burst
- [ ] Seek bar stays at user-intended position during debounce; does not snap back
- [ ] Stale `SeekDebounceTickMsg` (lower seq) is silently discarded
- [ ] Seek past `trackDurationMs` clamps to `trackDurationMs`; below 0 clamps to 0
- [ ] Seek is a no-op when `trackDurationMs` is 0 (no track loaded)
- [ ] `localProgressMs` does not increment while `seekBar.HasPending()` is true
- [ ] Successful seek resets `localProgressMs` to target position immediately
- [ ] Non-premium users see "Spotify Premium required" toast on seek attempt
- [ ] 429 rate limit → back off + ratelimit toast; 401 → token refresh; generic → error toast
- [ ] Volume bar (`+`/`-`) behavior is unchanged (regression: existing volume tests pass)
- [ ] All three keybinding docs updated (README, design.md §17, help_overlay.go)
- [ ] `make ci` passes (lint + tests + ≥ 80% coverage)

---

## Tasks

### Task 1 — DebounceTracker shared component

**Files:** `internal/ui/components/debounce.go`, `internal/ui/components/debounce_test.go`

- [ ] Write failing tests for `DebounceTracker`: HandleKey clamping, accumulation
      from pending, no-op when min ≥ max, HandleDebounce stale rejection, double-fire
      guard, SetConfirmed pending/confirmed transitions, ConfirmFromAPI seq match/mismatch,
      CancelPending seq match/mismatch, Current/HasPending accessors
  - test: `go test ./internal/ui/components/ -run TestDebounceTracker -v` → compile error
- [ ] Implement `DebounceTracker` with all methods per Design §1
  - test: `go test ./internal/ui/components/ -run TestDebounceTracker -v` → PASS
- [ ] `make ci` passes

### Task 2 — Refactor GradientVolumeBar to embed DebounceTracker

**File:** `internal/ui/components/gradient.go`

- [ ] Replace `currentVol`, `hasPending`, `seq` fields with `DebounceTracker` embed
- [ ] Delegate `HandleKey`, `HandleDebounce`, `ConfirmFromAPI`, `CancelPending`,
      `SetConfirmed` to `DebounceTracker`; `HandleKey` wraps result into
      `VolumeDebounceTickMsg` cmd
- [ ] Update `Render()` to use `b.Current()` instead of `b.currentVol`
- [ ] Existing volume bar tests pass unchanged (pure refactor)
  - test: `go test ./internal/ui/components/ -run "TestGradientVolumeBar|TestVolumeBar" -v` → PASS
  - test: `go test ./internal/ui/components/ -v` → all PASS
- [ ] `make ci` passes

### Task 3 — GradientSeekBar interactive state + seek messages

**Files:** `internal/ui/components/gradient.go`, `internal/ui/panes/messages.go`

- [ ] Add `SeekDebounceTickMsg` struct after `VolumeDebounceTickMsg` in `gradient.go`
- [ ] Add `DebounceTracker` embed and `trackDuration int` to `GradientSeekBar`
- [ ] Add seek methods: `HandleKey`, `HandleDebounce`, `ConfirmFromAPI`,
      `CancelPending`, `SetPositionConfirmed`, `SetTrackDuration`
- [ ] Update `GradientSeekBar.Render()` to use `b.Current()` when `b.HasPending()`
- [ ] Add `SeekIntentMsg`, `SeekAppliedMsg` to `messages.go`
- [ ] Write `GradientSeekBar` unit tests: HandleKey updates immediately, backward,
      clamp at zero, clamp at duration, no-op on zero duration, HandleDebounce stale
      rejection, HandleDebounce current acceptance, ConfirmFromAPI seq match,
      CancelPending seq match, SetConfirmed clears pending, Render pending vs confirmed
  - test: `go test ./internal/ui/components/ -v` → all PASS
- [ ] `make ci` passes

### Task 4 — App layer: buildSeekCmd + handlers

**Files:** `internal/app/commands.go`, `internal/app/handlers.go`, `internal/uikit/error_mapper.go`

- [ ] Add `buildSeekCmd` in `commands.go` (mirrors `buildSetVolumeCmd`)
- [ ] Add `SeekDebounceTickMsg` forwarding in `handlers.go`
- [ ] Add `SeekIntentMsg` handler: premium check → `buildSeekCmd`
- [ ] Add `SeekAppliedMsg` handler: confirm/cancel bar → error mapping
- [ ] Add `OpSeek` to `uikit.Operation`, `opTitle`, `opForbiddenBody`
- [ ] `go build ./...` → compiles without errors
- [ ] `make ci` passes

### Task 5 — NowPlayingPane: rebind arrows, add seek handlers, guard handleTick

**Files:** `internal/ui/panes/nowplaying.go`, `internal/app/routing.go`

- [ ] Replace `tea.KeyRight` / `"p"` + `tea.KeyLeft` cases with:
      `tea.KeyShiftLeft` → previous, `tea.KeyShiftRight` → next,
      `tea.KeyLeft` → seek back 5s, `tea.KeyRight` → seek forward 5s;
      remove dead `"p"` case
- [ ] Add `SeekDebounceTickMsg` and `SeekAppliedMsg` cases to `Update()`
- [ ] Add `confirmedProgress()` and `confirmedDuration()` helpers
- [ ] Sync seekBar in `handlePlaybackFetched()`: `SetTrackDuration` + `SetPositionConfirmed`
- [ ] Restore seek bar state in `SetTheme()`
- [ ] Guard `handleTick()`: skip `localProgressMs` increment when `seekBar.HasPending()`
- [ ] Update `routing.go`: add `tea.KeyShiftLeft`, `tea.KeyShiftRight` to
      `isPlaybackKey()` and `isPremiumOnlyPlaybackKey()`
- [ ] `go build ./...` → compiles without errors
- [ ] `make ci` passes

### Task 6 — Seek handler tests

**File:** `internal/app/seek_test.go`

- [ ] Write tests mirroring volume test pattern:
      `TestApp_SeekIntentMsg_CallsSeek`,
      `TestApp_SeekIntentMsg_NilPlayer_ReturnsErrNilClient`,
      `TestApp_SeekIntentMsg_NonPremium_Blocked`,
      `TestApp_SeekDebounceTickMsg_ForwardsToPane`,
      `TestApp_SeekAppliedMsg_Success_DispatchesInteractivePoll`,
      `TestBuildSeekCmd_429_EmitsSeekAppliedMsgWithRateLimitError`
  - test: `go test ./internal/app/ -run TestApp_Seek -v` → PASS
  - test: `go test ./internal/app/ -v` → all app tests PASS
- [ ] `make ci` passes

### Task 7 — Keybinding documentation (all 3 locations)

**Files:** `README.md`, `docs/system/design.md`, `internal/ui/panes/help_overlay.go`

- [ ] Update `README.md` keybindings section: `←`/`→` → seek, add `Shift+←`/`Shift+→`
- [ ] Update `docs/system/design.md` §16 and §17: same changes
- [ ] Update `internal/ui/panes/help_overlay.go` helpContent: split arrow row into
      seek row and shift-arrow row
- [ ] `make ci` passes (keybinding consistency check)

### Task 8 — Full CI verification

- [ ] `make ci` → lint + tests + 80% coverage all PASS
- [ ] Fix any issues and re-run