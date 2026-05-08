---
title: "Volume Debounce & Queue Icon Cleanup"
feature: 03-playback
status: open
---

## Background

Two independent problems in the `NowPlayingPane` / transport strip:

### 1. Dead queue icon

`PlaybackControls.Render()` in `internal/uikit/playback_controls.go` emits a `≡`
glyph (GlyphQueue) between play/pause and repeat:

```go
queue := inactiveStyle.Render(GlyphFor(GlyphQueue, m))
return shuffle + "  " + playPause + "  " + queue + "  " + repeat
```

This icon has no keybinding, no action, and no interactive state. It is pure dead UI.

`GlyphQueue` itself is still valid — `polling_traffic_pane.go:114` uses it as a
playlists indicator in the network log. Only the render site in `PlaybackControls`
is removed; the glyph constant and its `tui.md` entry stay.

### 2. Per-keystroke volume API calls — compound stale-read bug

Every `+`/`-` keypress fires an immediate Spotify `PUT /v1/me/player/volume` call
via `emitPlaybackRequest(ActionVolumeUp/Down)` → `buildPlaybackAPICmd` → the API.

Two failure modes compound:

**Rate limiting.** A rapid burst sends many HTTP requests in ~300 ms. With the
token bucket in place this is less likely to 429, but it is still wasteful and
risks transient failures.

**Stale-read accumulation.** `buildPlaybackAPICmd` snapshots
`store.PlaybackState().Device.VolumePercent` in the background cmd closure. The
store only refreshes every 1 s. Five rapid `+` presses each snapshot the same
base volume — all five calls set the same target:

```
User presses + × 5 from vol=49.
Expected: Spotify receives SetVolume(54).
Actual:   Spotify receives SetVolume(50) five times, settles at 50.
```

**Fix:** promote `GradientVolumeBar` from a pure render struct to a **smart
component** that owns its own visual volume state and a debounce sequence. The
parent `NowPlayingPane` delegates `+`/`-` keypresses to the bar. After 300 ms of
silence the bar emits `VolumeIntentMsg`; the root app builds exactly one
`SetVolume` API call per burst. The store's next poll reconciles the bar via
`SetConfirmed`.

**Depends on:** nothing — self-contained.

---

## Design

### Change 1 — Remove queue icon from transport strip

**`internal/uikit/playback_controls.go`**

- Delete the `queue` local variable and remove it from the return string.
- Strip shrinks from four positions to three: `shuffle  play/pause  repeat`.
- Update the struct doc comment (line ~22) and `Render()` doc comment (line ~40)
  to reference three positions.

**`internal/uikit/playback_controls_test.go`**

- Update assertions to expect no `≡` / `Q` in rendered output.
- Add `assert.NotContains(t, out, "≡")` and `assert.NotContains(t, out, "Q")`.

---

### Change 2 — GradientVolumeBar smart component

#### New message type — `internal/ui/components/gradient.go`

`VolumeDebounceTickMsg` is defined in `components` (not `panes`) to avoid
introducing a circular import. `NowPlayingPane` (in `panes`) imports `components`,
so `components` cannot import `panes`.

```go
// VolumeDebounceTickMsg is the timer payload fired by GradientVolumeBar.HandleKey
// after the 300 ms debounce window. Seq is a monotonically increasing counter;
// ticks with a lower seq than the bar's current seq are stale and discarded.
type VolumeDebounceTickMsg struct {
    TargetVol int
    Seq       int
}
```

#### New `GradientVolumeBar` fields

```go
type GradientVolumeBar struct {
    th         theme.Theme
    width      int
    currentVol int  // pending optimistic value, or last confirmed value
    hasPending bool // true while a debounce tick is in flight
    seq        int  // monotonically increasing; stale ticks have a lower seq
}
```

#### New methods (added before existing `Render`)

**`Render()` signature change:** `Render(volume int) string` → `Render() string`

The volume is now read from `b.currentVol` internally; callers no longer pass an
argument.

**`SetConfirmed(vol int)`**

Updates `currentVol` from the Spotify poll. No-op when `hasPending == true` —
the optimistic pending value is displayed until the debounce resolves.

```go
func (b *GradientVolumeBar) SetConfirmed(vol int) {
    if !b.hasPending {
        b.currentVol = vol
    }
}
```

**`HandleKey(delta, confirmedVol int) tea.Cmd`**

`delta` is `+1` or `-1`. `confirmedVol` is the store's current value, used as
the starting base only when `hasPending == false`. When `hasPending == true`, the
pending `currentVol` is the base (accumulation across rapid keypresses).

```go
func (b *GradientVolumeBar) HandleKey(delta, confirmedVol int) tea.Cmd {
    base := confirmedVol
    if b.hasPending {
        base = b.currentVol
    }
    newVol := base + delta
    if newVol > 100 { newVol = 100 }
    if newVol < 0   { newVol = 0   }
    b.currentVol = newVol
    b.hasPending = true
    b.seq++
    target, seq := newVol, b.seq
    return tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg {
        return VolumeDebounceTickMsg{TargetVol: target, Seq: seq}
    })
}
```

**`HandleDebounce(m VolumeDebounceTickMsg) (matched bool, targetVol int)`**

> **Architectural note:** the design spec shows `HandleDebounce` returning `tea.Cmd`.
> The implementation plan deviates to `(bool, int)`. This is intentional: the bar
> must not import `panes` (where `VolumeIntentMsg` lives), so `NowPlayingPane`
> constructs the `VolumeIntentMsg` cmd itself after receiving `(true, vol)`.

```go
func (b *GradientVolumeBar) HandleDebounce(m VolumeDebounceTickMsg) (matched bool, targetVol int) {
    if m.Seq != b.seq {
        return false, 0
    }
    b.hasPending = false
    return true, m.TargetVol
}
```

#### Required import additions in `gradient.go`

```go
import (
    "fmt"
    "math"
    "strconv"
    "strings"
    "time"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/initgrep-apps/spotnik/internal/ui/theme"
    "github.com/initgrep-apps/spotnik/internal/uikit"
)
```

---

### Change 3 — `NowPlayingPane` wiring

**`internal/ui/panes/messages.go`**

Add `VolumeIntentMsg`:

```go
// VolumeIntentMsg is emitted by NowPlayingPane after the volume debounce resolves.
// TargetVol is the exact percentage to set — the app does not read the store.
// Handled by App.Update() → buildSetVolumeCmd.
type VolumeIntentMsg struct {
    TargetVol int
}
```

Remove `ActionVolumeUp` and `ActionVolumeDown` constants (and their doc comments)
from the `PlaybackAction` iota block.

**`internal/ui/panes/nowplaying.go`**

`handleKey` — replace the two volume cases:

```go
// Before:
case msg.Type == tea.KeyRunes && string(msg.Runes) == "+":
    return p, emitPlaybackRequest(ActionVolumeUp)
case msg.Type == tea.KeyRunes && string(msg.Runes) == "-":
    return p, emitPlaybackRequest(ActionVolumeDown)

// After:
case msg.Type == tea.KeyRunes && string(msg.Runes) == "+":
    return p, p.volumeBar.HandleKey(+1, confirmedVolume(p.store))
case msg.Type == tea.KeyRunes && string(msg.Runes) == "-":
    return p, p.volumeBar.HandleKey(-1, confirmedVolume(p.store))
```

New package-level helper (add near other helpers like `formatDurationMs`):

```go
// confirmedVolume reads the active device's volume from the store.
// Returns 0 when playback state or device info is unavailable.
func confirmedVolume(s state.StateReader) int {
    if ps := s.PlaybackState(); ps != nil && ps.Device != nil {
        return ps.Device.VolumePercent
    }
    return 0
}
```

`Update()` — add `VolumeDebounceTickMsg` case (before the `tea.KeyMsg` case):

```go
case components.VolumeDebounceTickMsg:
    if matched, vol := p.volumeBar.HandleDebounce(m); matched {
        return p, func() tea.Msg { return VolumeIntentMsg{TargetVol: vol} }
    }
    return p, nil
```

`handlePlaybackFetched` — add `SetConfirmed` after existing sync:

```go
if ps := p.store.PlaybackState(); ps != nil {
    p.localProgressMs = ps.ProgressMs
    p.engine.SetPlaying(ps.IsPlaying)
    if ps.Device != nil {
        p.volumeBar.SetConfirmed(ps.Device.VolumePercent)
    }
}
```

`NewNowPlayingPane` — seed the bar at construction (inside the existing
`if ps := s.PlaybackState(); ps != nil` block):

```go
if ps.Device != nil {
    p.volumeBar.SetConfirmed(ps.Device.VolumePercent)
}
```

`SetTheme` — seed the bar after reconstruction:

```go
p.volumeBar = components.NewGradientVolumeBar(th)
p.volumeBar.SetConfirmed(confirmedVolume(p.store))
```

`View()` — remove the `volume` snapshot variable; the three `p.volumeBar.Render(volume)`
calls (for compact/medium/full track layouts) all become `p.volumeBar.Render()`.

Ensure `components` is imported:

```go
"github.com/initgrep-apps/spotnik/internal/ui/components"
```

---

### Change 4 — App layer

**`internal/app/commands.go`**

Add `buildSetVolumeCmd`:

```go
// buildSetVolumeCmd creates a command that calls player.SetVolume with the
// exact target volume delivered via VolumeIntentMsg after debounce resolves.
func (a *App) buildSetVolumeCmd(targetVol int) tea.Cmd {
    player := a.player
    return func() tea.Msg {
        if player == nil {
            return panes.PlaybackCmdSentMsg{Err: errNilClient}
        }
        ctx := api.WithPriority(context.Background(), api.Interactive)
        err := player.SetVolume(ctx, targetVol)
        if err != nil {
            if secs := parse429RetryAfter(err); secs > 0 {
                return panes.RateLimitedMsg{RetryAfterSecs: secs}
            }
            if isUnauthorizedError(err) {
                return unauthorizedMsg{}
            }
        }
        return panes.PlaybackCmdSentMsg{Err: err}
    }
}
```

Remove the `ActionVolumeUp` and `ActionVolumeDown` case blocks from
`buildPlaybackAPICmd`. After removal, `volStep` and `currentVolume` become unused
within the function — delete them:

```go
// DELETE from buildPlaybackAPICmd:
volStep := a.volumeStep
// DELETE:
currentVolume := 65
if ps != nil {
    if ps.Device != nil {
        currentVolume = ps.Device.VolumePercent
    }
}
```

Also delete the `volumeStep` field from the `App` struct in `internal/app/app.go`
(lines ~179–180) and its initialization at line ~384. It is only used in the now-
removed volume cases.

**`internal/app/handlers.go`**

Forward `VolumeDebounceTickMsg` to `NowPlayingPane` (same pattern as `viz.TickMsg`
at line ~470):

```go
case components.VolumeDebounceTickMsg:
    if np := a.nowPlayingPane(); np != nil {
        updated, cmd := np.Update(m)
        if pp, ok := updated.(*panes.NowPlayingPane); ok {
            a.panes[layout.PaneNowPlaying] = pp
        }
        return a, cmd
    }
    return a, nil
```

Add `VolumeIntentMsg` handler (near the `PlaybackRequestMsg` case):

```go
case panes.VolumeIntentMsg:
    return a, a.buildSetVolumeCmd(m.TargetVol)
```

`components` is already imported in `handlers.go` — no new import needed.

**`internal/api/apitest/mock.go`**

Add `LastSetVolume int` to `MockPlayer` and update `SetVolume` to record the arg:

```go
// in MockPlayer struct:
LastSetVolume int

// updated SetVolume:
func (m *MockPlayer) SetVolume(_ context.Context, vol int) error {
    m.SetVolumeCalled = true
    m.LastSetVolume = vol
    return m.SetVolumeErr
}
```

---

## Data Flow (after this change)

```
keypress(+/-)
    │
    ▼
nowplaying.handleKey()
    │  confirmedVol := confirmedVolume(store)
    │  calls volumeBar.HandleKey(±1, confirmedVol)
    │  currentVol updates → bar renders new value immediately
    │  returns tea.Tick(300ms, VolumeDebounceTickMsg{target, seq})
    │
    ▼ [300 ms later — or stale tick discarded by seq check]
VolumeDebounceTickMsg arrives at App.Update()
    │
    ▼
handlers.go → np.Update(m) → volumeBar.HandleDebounce(m)
    │  seq matches? → hasPending=false, return VolumeIntentMsg{TargetVol}
    │  stale?       → return nil (discard)
    │
    ▼
handlers.go: VolumeIntentMsg → buildSetVolumeCmd(targetVol)
    │
    ▼
HTTP PUT /v1/me/player/volume?volume_percent=N   ← 1 call per burst
    │
    ▼
PlaybackCmdSentMsg{Err}  ← err → toast
    │
    ▼
[next poll — PlaybackStateFetchedMsg]
    │
    ▼
volumeBar.SetConfirmed(ps.Device.VolumePercent)  ← reconcile
```

---

## Acceptance Criteria

- [ ] Transport strip renders `shuffle  play/pause  repeat` — no queue `≡` glyph
- [ ] `assert.NotContains(t, out, "≡")` and `NotContains(t, out, "Q")` in playback controls tests
- [ ] Pressing `+` five times rapidly from vol=49 sends exactly one
      `SetVolume(54)` call (not five `SetVolume(50)` calls)
- [ ] Bar renders the optimistic pending value instantly on each keypress;
      no wait for Spotify poll
- [ ] `SetConfirmed` called from `handlePlaybackFetched` reconciles bar after each
      1s poll when no pending intent is in flight
- [ ] Theme switch via `SetTheme` seeds the new bar's `currentVol` from the store
      so it does not start at 0
- [ ] Constructor `NewNowPlayingPane` seeds bar from store at startup
- [ ] Stale `VolumeDebounceTickMsg` (lower seq) returns nil cmd and is silently discarded
- [ ] `buildSetVolumeCmd` with nil player returns `PlaybackCmdSentMsg{Err: errNilClient}`
- [ ] `App.volumeStep` field removed from `App` struct; `volStep` local removed from `buildPlaybackAPICmd`
- [ ] `ActionVolumeUp` and `ActionVolumeDown` removed from `PlaybackAction` constants
- [ ] `make ci` passes (lint + tests + ≥ 80% coverage)

---

## Tasks

> **Implementation guide:** `docs/superpowers/plans/2026-05-08-volume-debounce-controls-cleanup.md`
> contains the TDD step-by-step sequence (write failing test → implement → verify → commit).
> Follow that plan task-by-task.

### Task 1 — Remove queue icon from transport strip

**Files:** `internal/uikit/playback_controls.go`, `internal/uikit/playback_controls_test.go`

- [ ] Update `TestPlaybackControls_RenderUnicode_Playing` and
      `TestPlaybackControls_RenderASCII_Playing` to assert `NotContains "≡"` / `NotContains "Q"`;
      remove any assertion that expects the queue glyph to be present
  - test: `go test ./internal/uikit/... -run TestPlaybackControls -v` → FAIL (queue still present)
- [ ] Remove `queue` variable and its position from `Render()` return in
      `playback_controls.go`; update struct doc and `Render()` doc comment to say three positions
  - test: same run → PASS
- [ ] `make ci` passes

### Task 2 — GradientVolumeBar: smart component fields and methods

**Files:** `internal/ui/components/gradient.go`, `internal/ui/components/gradient_test.go`

- [ ] Add table-driven tests for `HandleKey`, `HandleDebounce`, `SetConfirmed` (see Design §
      for exact test names) to `gradient_test.go` — `Render(vol)` signature unchanged in this task
  - test: `go test ./internal/ui/components/... -run "TestVolumeBar_Handle|TestVolumeBar_SetConfirmed" -v` → compile error (undefined)
- [ ] Add `VolumeDebounceTickMsg` type above `GradientVolumeBar` struct; add `currentVol`,
      `hasPending`, `seq` fields; add `SetConfirmed`, `HandleKey`, `HandleDebounce` methods;
      add `time` and `tea` imports
  - test: same run → PASS; full `./internal/ui/components/...` → PASS (Render still works with old sig)
- [ ] `make ci` passes

### Task 3 — Change `Render()` to no-arg; update all call sites atomically

**Files:** `internal/ui/components/gradient.go`, `internal/ui/components/gradient_test.go`,
`internal/ui/components/visualizer_gradient_integration_test.go`, `internal/ui/panes/nowplaying.go`

- [ ] Change `Render(volume int) string` → `Render() string` in `gradient.go`;
      `volume := b.currentVol` at top of body; remove old clamping that used the parameter
- [ ] Replace every `b.Render(N)` call in `gradient_test.go` with
      `b.SetConfirmed(N); b.Render()`
- [ ] Replace `vb.Render(50)` calls in `visualizer_gradient_integration_test.go` with
      `vb.SetConfirmed(50); vb.Render()`
- [ ] In `nowplaying.go` `View()`: remove `volume` snapshot variable; replace the three
      `p.volumeBar.Render(volume)` calls with `p.volumeBar.Render()`; add
      `confirmedVolume` package helper; seed bar in `NewNowPlayingPane` and `SetTheme`
  - test: `go build ./...` → no errors; `go test ./internal/ui/...` → PASS
- [ ] `make ci` passes

### Task 4 — NowPlayingPane: delegate volume keys + `VolumeIntentMsg`

**Files:** `internal/ui/panes/messages.go`, `internal/ui/panes/nowplaying.go`,
`internal/ui/panes/nowplaying_test.go`

- [ ] Add `VolumeIntentMsg` to `messages.go` (keep `ActionVolumeUp/Down` — removed in Task 5)
- [ ] Add tests: `TestNowPlayingPane_VolumeUp_ReturnsDebounceCmdNotPlaybackRequest`,
      `TestNowPlayingPane_VolumeDebounceMsg_EmitsVolumeIntent`,
      `TestNowPlayingPane_StaleVolumeDebounce_ReturnsNilCmd` — import `components` in test file;
      add `playingStateWithVolume` helper if `playingState()` doesn't accept a volume arg
  - test: `go test ./internal/ui/panes/... -run "TestNowPlayingPane_Volume" -v` → compile/FAIL
- [ ] Replace `+`/`-` cases in `handleKey` with `HandleKey` calls; add
      `VolumeDebounceTickMsg` case to `Update()`; add `SetConfirmed` call in
      `handlePlaybackFetched`
  - test: same run → PASS; full pane suite → PASS
- [ ] `make ci` passes

### Task 5 — App layer: forward tick, handle intent, delete dead code

**Files:** `internal/app/commands.go`, `internal/app/handlers.go`,
`internal/app/app.go`, `internal/ui/panes/messages.go`, `internal/api/apitest/mock.go`

- [ ] Add `LastSetVolume int` to `MockPlayer` in `apitest/mock.go`; update `SetVolume`
      to record `m.LastSetVolume = vol`
- [ ] Add test `TestApp_VolumeIntentMsg_CallsSetVolume` (and
      `TestBuildSetVolumeCmd_NilPlayer_ReturnsErrNilClient` for coverage)
  - test: `go test ./internal/app/... -run "TestApp_VolumeIntent|TestBuildSetVolumeCmd"` → compile/FAIL
- [ ] Add `buildSetVolumeCmd` to `commands.go`; remove `ActionVolumeUp/Down` cases from
      `buildPlaybackAPICmd`; remove `volStep` local variable and `currentVolume` snapshot block
- [ ] Remove `volumeStep` field from `App` struct in `app.go` (lines ~179–180) and its
      initialization (line ~384) — it is now dead
- [ ] Add `VolumeDebounceTickMsg` forwarder and `VolumeIntentMsg` handler in `handlers.go`
- [ ] Remove `ActionVolumeUp` and `ActionVolumeDown` from `messages.go`
  - test: `go build ./...` → clean; volume intent tests → PASS
- [ ] `make ci` passes
