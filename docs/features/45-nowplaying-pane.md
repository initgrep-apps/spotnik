# Feature 45 — NowPlaying Pane

> **Feature:** Rename `PlayerPane` to `NowPlayingPane`, implement the `layout.Pane`
> interface, embed the braille visualizer and gradient bars, and support compact mode
> for presets where NowPlaying is a single-line strip.

## Context

The current `PlayerPane` (`internal/ui/panes/player.go`, ~7.1KB) renders the currently
playing track with transport controls, a monochrome seek bar, and volume bar. It does
not implement the `layout.Pane` interface and has no visualizer or gradient effects.

The new DESIGN.md (§2, §4, §11) specifies:
- NowPlaying is pane `1` on Page A with toggle key `1`
- Full mode: braille visualizer + track info + seek bar (gradient) + controls + volume bar (gradient)
- Compact mode (presets 1-3 where row weight=1): single-line strip showing track + progress inline
- Actions in border: shuffle, repeat

**Design reference:** `docs/DESIGN.md` §2 (Pane Definitions — NowPlaying), §4 (Preset Layouts —
compact strip in presets 1/2/3), §11 (Visual Components — Visualizer + Gradient Bars),
§23 (Migration — PlayerPane → NowPlayingPane)

**Depends on:** Feature 41 (Pane interface), Feature 42 (border renderer), Feature 44 (visualizer + gradient bars)

---

## Design Diagram

```
Full Mode (Preset 0 — Dashboard, row height weight 2):

╭─ ¹Now Playing ────────────────────────── ᐅs shuffle ─ ᐅr repeat ╮
│  ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿                     │
│                                                                  │
│  Martbaan · Samar Mehdi, June, Sarah Mehdi                       │
│  Martbaan (Album)                                                │
│                                                                  │
│  ▶  1:41  ████████████████████████░░░░░░░░░░░░░░░░  5:30         │
│  |<   ||   >|        ~   =>           VOL  ████████░░  65%       │
╰──────────────────────────────────────────────────────────────────╯

Expanded Mode (Preset 1 — Listening, row height weight 3):

Same as full mode but with more visualizer rows (3-4 lines) due to extra height.

Compact Mode (Presets 2/3 — Library/Discovery, row height weight 1):

╭─ ¹Now Playing ── Martbaan · Samar Mehdi ── ▶ 1:41/5:30 ─────────╮
│  ████████████░░░░░░░  |<  ||  >|  ~  =>   VOL ████░░ 65%         │
╰──────────────────────────────────────────────────────────────────╯
```

---

## Task 1: Rename PlayerPane to NowPlayingPane

**Problem:** The pane name doesn't match the new design terminology.

**Fix:**

1. Rename `internal/ui/panes/player.go` → `internal/ui/panes/nowplaying.go`
2. Rename struct `PlayerPane` → `NowPlayingPane`
3. Rename constructor `NewPlayerPane` → `NewNowPlayingPane`
4. Update all references in `internal/app/app.go`, `render.go`, `routing.go`
5. Rename test file: `player_test.go` → `nowplaying_test.go`

**Files:**
- Rename: `internal/ui/panes/player.go` → `internal/ui/panes/nowplaying.go`
- Rename: `internal/ui/panes/player_test.go` → `internal/ui/panes/nowplaying_test.go`
- Modify: `internal/app/app.go` — update field name and constructor call
- Modify: `internal/app/render.go` — update references
- Modify: `internal/app/routing.go` — update references

**Tests:**
- Existing player tests pass under new name
- Build succeeds with no broken references

**Commit:** `refactor(ui): rename PlayerPane to NowPlayingPane`

---

## Task 2: Implement layout.Pane interface

**Problem:** NowPlayingPane doesn't satisfy the `layout.Pane` interface.

**Fix:**

Add methods to `NowPlayingPane`:

```go
func (p *NowPlayingPane) ID() layout.PaneID       { return layout.PaneNowPlaying }
func (p *NowPlayingPane) Title() string            { return "Now Playing" }
func (p *NowPlayingPane) ToggleKey() int           { return 1 }
func (p *NowPlayingPane) Actions() []layout.Action {
    return []layout.Action{
        {Key: "s", Label: "shuffle"},
        {Key: "r", Label: "repeat"},
    }
}
```

Ensure `SetSize(width, height int)`, `SetFocused(bool)`, and `IsFocused() bool` already
exist (they do in the current PlayerPane) and match the interface signatures.

**Files:**
- Modify: `internal/ui/panes/nowplaying.go`

**Tests:**
- Unit: `ID()` returns `PaneNowPlaying`
- Unit: `Title()` returns "Now Playing"
- Unit: `ToggleKey()` returns 1
- Unit: `Actions()` returns shuffle + repeat
- Unit: Compile-time check: `var _ layout.Pane = &NowPlayingPane{}`

**Commit:** `feat(ui): NowPlayingPane implements layout.Pane interface`

---

## Task 3: Embed visualizer component

**Problem:** NowPlaying has no audio visualizer.

**Fix:**

1. Add `visualizer *components.Visualizer` field to `NowPlayingPane`
2. Initialize in constructor: `visualizer: components.NewVisualizer(theme)`
3. In `Update()`:
   - Handle `VisualizerTickMsg` → forward to `visualizer.Update(msg)`
   - On `PlaybackStateFetchedMsg` → call `visualizer.SetPlaying(state.IsPlaying)`
4. In `SetSize()` → call `visualizer.SetSize(contentWidth, vizHeight)` where:
   - `vizHeight` = 1 if compact mode, 2 if full mode, 4 if expanded mode (Preset 1)
5. In `Init()` → batch `visualizer.Init()` with existing init commands
6. In `View()` full mode → render `visualizer.View()` above track info

**Files:**
- Modify: `internal/ui/panes/nowplaying.go`

**Tests:**
- Unit: Visualizer starts ticking after Init()
- Unit: PlaybackStateFetchedMsg with IsPlaying=true → visualizer animates
- Unit: PlaybackStateFetchedMsg with IsPlaying=false → visualizer paused
- Unit: VisualizerTickMsg → visualizer frame advances
- Unit: View() in full mode contains braille characters when playing

**Commit:** `feat(ui): embed braille visualizer in NowPlayingPane`

---

## Task 4: Replace bars with gradient versions

**Problem:** Seek bar and volume bar use monochrome colors.

**Fix:**

1. Replace `ProgressBar` with `GradientSeekBar` (from Feature 44)
2. Replace `VolumeBar` with `GradientVolumeBar` (from Feature 44)
3. Update `SetSize()` to call `seekBar.SetWidth()` and `volumeBar.SetWidth()`
4. Update `View()` to use `seekBar.Render(progressMs, durationMs)` and `volumeBar.Render(volume)`

**Files:**
- Modify: `internal/ui/panes/nowplaying.go`

**Tests:**
- Unit: Seek bar renders gradient colors (verify styled output differs from monochrome)
- Unit: Volume bar shows correct color band for volume level
- Unit: Bars resize correctly with `SetSize()`

**Commit:** `feat(ui): gradient seek bar and volume bar in NowPlayingPane`

---

## Task 5: Implement compact mode

**Problem:** No compact rendering mode for presets where NowPlaying is in a small row.

**Fix:**

1. Add `compact bool` field to `NowPlayingPane`
2. Determine compact mode in `SetSize()`: if `height <= 3` (border + 1 content line), enable compact
3. Compact `View()` renders a single content line:
   ```
   ████████████░░░░░░░  |<  ||  >|  ~  =>   VOL ████░░ 65%
   ```
4. Compact border title includes track info:
   ```
   ╭─ ¹Now Playing ── Martbaan · Samar Mehdi ── ▶ 1:41/5:30 ─────╮
   ```
   Override `Title()` to return dynamic title when compact:
   ```go
   func (p *NowPlayingPane) Title() string {
       if p.compact {
           return fmt.Sprintf("Now Playing ── %s · %s ── %s %s/%s",
               trackName, artistName, playSymbol, currentTime, totalTime)
       }
       return "Now Playing"
   }
   ```
5. Full `View()` (non-compact): render visualizer + track info + seek bar + controls + volume
6. No visualizer in compact mode (not enough vertical space)

**Files:**
- Modify: `internal/ui/panes/nowplaying.go`

**Tests:**
- Unit: `SetSize(80, 3)` → compact mode enabled
- Unit: `SetSize(80, 10)` → compact mode disabled
- Unit: Compact `Title()` includes track name and progress
- Unit: Compact `View()` has exactly 1 content line (controls + volume)
- Unit: Full `View()` has visualizer + track info + bars
- Unit: No visualizer rendered in compact mode

**Commit:** `feat(ui): compact mode for NowPlaying in small presets`

---

## Task 6: Comprehensive tests

**Files:**
- Modify: `internal/ui/panes/nowplaying_test.go`

**Tests:**
- Integration: Full lifecycle — construct → Init → resize → playback state → tick → verify View
- Integration: Full mode layout — visualizer + track + seek + controls + volume all present
- Integration: Compact mode layout — single-line content, dynamic title
- Integration: Transition: full → compact (resize smaller) → full (resize larger)
- Integration: Playing/paused state affects visualizer animation
- Integration: Shuffle/repeat state reflected in controls
- Integration: Volume changes reflected in gradient volume bar
- Integration: Progress changes reflected in gradient seek bar
- Integration: Interface satisfaction: `var _ layout.Pane = &NowPlayingPane{}`
- Edge: No playback state (nil) → safe rendering (empty/placeholder)
- Edge: Zero duration → seek bar handles gracefully

**Commit:** `test(ui): comprehensive NowPlayingPane tests`

---

## Acceptance Criteria

- [ ] `PlayerPane` renamed to `NowPlayingPane` everywhere
- [ ] `NowPlayingPane` satisfies `layout.Pane` interface
- [ ] Braille visualizer animates when playing, stops when paused
- [ ] Gradient seek bar shows color transition from Gradient1 to Gradient2
- [ ] Volume bar shows correct color band (green/yellow/red)
- [ ] Compact mode renders single-line strip for small height
- [ ] Dynamic title in compact mode shows track + progress
- [ ] All playback keys (Space, >, <, +, -, s, r) still work
- [ ] Old `ProgressBar` and `VolumeBar` imports removed from this pane
- [ ] `make ci` passes

---

## Notes

- The old `ProgressBar` and `VolumeBar` components in `internal/ui/components/` are NOT deleted
  in this feature. They may be used by other parts. If they become unused after all pane
  migrations, they'll be cleaned up in Feature 53 (Cleanup).
- Compact mode detection uses height threshold. The LayoutManager assigns heights based on
  preset row weights — NowPlaying gets weight=1 in presets 2/3, which results in ~3-4 rows
  (including border), triggering compact mode.
- The visualizer tick (200ms) is shared conceptually with the NowPlaying pane but the
  VisualizerTickMsg is a distinct type. The app's root Update must forward it to NowPlaying.
  This wiring happens in Feature 49 (App Migration).
