# Feature 60 — NowPlaying Pane Redesign

> **Feature:** Restructure the NowPlaying pane layout to a two-column split with seek bar
> inside the right panel, upgrade transport icons to Unicode glyphs, update volume bar
> characters/icon, add border actions, support vertical centering in expanded mode, and
> migrate from the old Visualizer to the new `viz.Engine`.

## Context

The NowPlaying pane has diverged from the intended design through multiple iterations.
Key issues to fix:

- **Progress bar placement**: Currently spans full width at the pane bottom. Should be
  inside the right panel, vertically centered between visualization rows.
- **Transport icons**: ASCII symbols (`|< || >| ~ =>`) look crude. Should use cleaner
  Unicode glyphs (`⇄ ▷ ⏸ ≡ ↻`). Previous/Next removed from controls row.
- **Volume bar**: Full-block characters (`█░`) are too heavy. Should use discrete small
  blocks (`■□`) with a music note icon (`♪`) instead of `VOL` text.
- **Border actions**: Missing shortcuts for play, volume, and visualization toggle.
- **Expanded state**: Content stretches to fill height. Should remain fixed-height and
  vertically centered using `lipgloss.Place`.
- **Visualizer integration**: Old single-color `*components.Visualizer` replaced by
  `*viz.Engine` with per-row color gradient.

**Design reference:** `docs/superpowers/specs/2026-03-27-nowplaying-redesign.md` §2, §4, §5, §6

**Depends on:** Feature 59 (Visualizer Engine — `viz/` package)

---

## Design Diagram

```
╭─ Now Playing ──────────────────────── actions...  ─╮
│                                                    │
│  ╭─ Track Info ─╮  ┌────────────────────────────┐  │
│  │ SONG NAME    │  │  ▲ Visualization (top)     │  │
│  │ Album        │  │    per-row color gradient  │  │
│  │ Artist       │  ├────────────────────────────┤  │
│  │              │  │ 2:09 ███████░░░░░░░░  4:13 │  │
│  │ ⇄  ▷  ≡  ↻  │  ├────────────────────────────┤  │
│  │ ♪ ■■■□□□□   │  │  ▼ Visualization (bottom)  │  │
│  ╰──────────────╯  │    mirror / continuation   │  │
│                    └────────────────────────────┘  │
│    ~1/3 width             ~2/3 width               │
╰────────────────────────────────────────────────────╯

Border actions:
  ᐅs shfl — ᐅr rpt — ᐅspace play — ᐅ+/- vol — ᐅv viz

Controls row (Previous/Next removed — they have keyboard shortcuts):
  ⇄  ▷  ≡  ↻

Volume bar:
  ♪ ■■■■□□□□□□ 31%    (volume > 0: ♪ in Gradient1 color)
  ♪ □□□□□□□□□□  0%    (volume = 0: ♪ in TextMuted color)

Expanded state: same content block, vertically centered via lipgloss.Place.
Compact state (height < 8): inline track info in border title (existing behavior).

Visualization row split:
  topRows = (availableHeight - 1) / 2
  bottomRows = availableHeight - 1 - topRows
  Seek bar occupies 1 row between top and bottom viz rows.
```

---

## Task 1: Update transport controls to Unicode glyphs

**Problem:** Controls use ASCII symbols (`|< || >| ~ =>`) that look crude. Previous
and Next icons clutter the row when keyboard shortcuts already exist.

**Fix:**

Modify `internal/ui/components/controls.go`:

1. Remove Previous (`|<`) and Next (`>|`) from the controls row
2. Replace symbols:
   - Shuffle: `~` → `⇄` (U+21C4)
   - Play: `>` → `▷` (U+25B7)
   - Pause: `||` → `⏸` (U+23F8)
   - Repeat off: `=>` → `↻` (U+21BB, inactive color)
   - Repeat context: `=>` → `↻` (active color)
   - Repeat track: `=>1` → `↻1` (active color)
3. Add Queue icon: `≡` (U+2261) — purely decorative, always `TextSecondary` color
4. New row format: `⇄  ▷  ≡  ↻`

Active/inactive coloring logic stays the same: `PlayingIndicator()` for active,
`TextSecondary()` for inactive.

```go
func (c Controls) Render() string {
    var shuffle string
    if c.shuffleOn {
        shuffle = c.activeStyle.Render("⇄")
    } else {
        shuffle = c.inactiveStyle.Render("⇄")
    }

    var playPause string
    if c.isPlaying {
        playPause = c.activeStyle.Render("⏸")
    } else {
        playPause = c.inactiveStyle.Render("▷")
    }

    queue := c.inactiveStyle.Render("≡")

    var repeat string
    switch c.repeatMode {
    case "track":
        repeat = c.activeStyle.Render("↻1")
    case "context":
        repeat = c.activeStyle.Render("↻")
    default:
        repeat = c.inactiveStyle.Render("↻")
    }

    return shuffle + "  " + playPause + "  " + queue + "  " + repeat
}
```

**Files:**
- Modify: `internal/ui/components/controls.go`
- Modify: `internal/ui/components/controls_test.go`

**Tests:**
- Unit: Playing state → output contains `⏸` (pause symbol)
- Unit: Paused state → output contains `▷` (play symbol)
- Unit: Shuffle on → `⇄` in active color
- Unit: Shuffle off → `⇄` in inactive color
- Unit: Repeat off → `↻` in inactive color
- Unit: Repeat context → `↻` in active color
- Unit: Repeat track → `↻1` in active color
- Unit: Queue icon `≡` always present
- Unit: Output does NOT contain `|<` or `>|` (Previous/Next removed)
- Unit: Output does NOT contain `~` or `=>` (old symbols gone)

**Commit:** `feat(ui): Unicode transport control glyphs`

---

## Task 2: Update volume bar characters and icon

**Problem:** Volume bar uses heavy full-block characters (`█░`) and `VOL` text prefix.

**Fix:**

Modify `internal/ui/components/gradient.go` (the `GradientVolumeBar` section):

1. Replace filled character: `█` (U+2588) → `■` (U+25A0)
2. Replace empty character: `░` (U+2591) → `□` (U+25A1)
3. Replace `VOL` prefix with `♪` (U+266A, music note):
   - Volume > 0: `♪` in `Gradient1()` color (green)
   - Volume = 0 / muted: `♪` in `TextMuted()` color
4. Format: `♪ ■■■■□□□□□□ 31%`

Update the `Render` method:

```go
func (b *GradientVolumeBar) Render(volume int) string {
    // ... (existing clamping + bar width logic unchanged) ...

    bar := fillStyle.Render(strings.Repeat("■", filled)) +
        emptyStyle.Render(strings.Repeat("□", empty))

    var icon string
    if volume > 0 {
        iconStyle := lipgloss.NewStyle().Foreground(b.th.Gradient1())
        icon = iconStyle.Render("♪")
    } else {
        iconStyle := lipgloss.NewStyle().Foreground(b.th.TextMuted())
        icon = iconStyle.Render("♪")
    }

    return fmt.Sprintf("%s %s  %d%%", icon, bar, volume)
}
```

Adjust the reserved width calculation: `♪ ` = 2 chars (was `VOL  ` = 5 chars), so
`reserved` drops from 10 to 7.

**Files:**
- Modify: `internal/ui/components/gradient.go`
- Modify: `internal/ui/components/gradient_test.go`

**Tests:**
- Unit: Output contains `■` for filled portion (not `█`)
- Unit: Output contains `□` for empty portion (not `░`)
- Unit: Output contains `♪` icon (not `VOL`)
- Unit: Volume > 0 → `♪` uses Gradient1 color
- Unit: Volume = 0 → `♪` uses TextMuted color
- Unit: Volume clamping still works (negative → 0, >100 → 100)
- Unit: Color bands unchanged: 0-33% green, 34-66% yellow, 67-100% red
- Unit: Width changes → bar length adjusts correctly with new reserved width

**Commit:** `feat(ui): volume bar with ■□ blocks and ♪ icon`

---

## Task 3: Update border actions

**Problem:** Border actions only show shuffle and repeat. Missing play, volume, and
visualization toggle shortcuts.

**Fix:**

Modify `internal/ui/panes/nowplaying.go` — update `Actions()`:

```go
func (p *NowPlayingPane) Actions() []layout.Action {
    return []layout.Action{
        {Key: "s", Label: "shfl"},
        {Key: "r", Label: "rpt"},
        {Key: "space", Label: "play"},
        {Key: "+/-", Label: "vol"},
        {Key: "v", Label: "viz"},
    }
}
```

Labels are abbreviated to fit in the border without overflow on typical terminal widths.

**Files:**
- Modify: `internal/ui/panes/nowplaying.go`
- Modify: `internal/ui/panes/nowplaying_test.go`

**Tests:**
- Unit: `Actions()` returns exactly 5 actions
- Unit: Actions contain keys: `s`, `r`, `space`, `+/-`, `v`
- Unit: Actions contain labels: `shfl`, `rpt`, `play`, `vol`, `viz`

**Commit:** `feat(ui): border actions for play, volume, viz toggle`

---

## Task 4: Migrate NowPlayingPane from Visualizer to viz.Engine

**Problem:** The pane uses `*components.Visualizer` which has a 4-line height cap,
single color, and only braille rendering. Need to switch to `*viz.Engine`.

**Fix:**

Modify `internal/ui/panes/nowplaying.go`:

1. Replace field: `visualizer *components.Visualizer` → `engine *viz.Engine`
2. Update constructor:
   ```go
   engine: viz.NewEngine(t),
   ```
3. Update `Init()`:
   ```go
   return p.engine.Init()
   ```
4. Update `Update()` — change `components.VisualizerTickMsg` to `viz.TickMsg`:
   ```go
   case viz.TickMsg:
       cmd := p.engine.Update(m)
       return p, cmd
   ```
5. Update `handlePlaybackFetched()`:
   ```go
   p.engine.SetPlaying(ps.IsPlaying)
   ```
6. Update `handleKey()` for `v` key:
   ```go
   p.engine.CyclePattern()
   ```
7. Update import: add `"github.com/initgrep-apps/spotnik/internal/ui/components/viz"`,
   remove the `components.VisualizerTickMsg` usage.

**Files:**
- Modify: `internal/ui/panes/nowplaying.go`
- Modify: `internal/app/app.go` — update `components.VisualizerTickMsg` to `viz.TickMsg`
  in the message routing switch case (lines ~911-929)
- Modify: `internal/ui/panes/requestflow_pane.go` — update `components.VisualizerTickMsg`
  case to `viz.TickMsg` in the `Update()` switch
- Modify: `internal/ui/panes/requestflow_pane_test.go` — update `VisualizerTickMsg`
  references to `viz.TickMsg`

**Tests:**
- Unit: `NewNowPlayingPane` creates pane with engine (no panic)
- Unit: `Init()` returns tick command from engine
- Unit: `viz.TickMsg` → engine frame advances
- Unit: `PlaybackStateFetchedMsg` with IsPlaying=true → engine is playing
- Unit: `PlaybackStateFetchedMsg` with IsPlaying=false → engine is paused
- Unit: `v` key → engine pattern cycles
- Build: `components.VisualizerTickMsg` no longer referenced in `nowplaying.go`
- Build: `app.go` compiles with `viz.TickMsg` import
- Build: `requestflow_pane.go` compiles with `viz.TickMsg` import

**Commit:** `refactor(ui): migrate NowPlayingPane to viz.Engine`

---

## Task 5: Restructure View to two-column split with seek bar in right panel

**Problem:** Current layout puts the seek bar at the bottom spanning full width.
The new design puts it inside the right panel, vertically centered between
visualization rows.

**Fix:**

Modify `internal/ui/panes/nowplaying.go` — rewrite `View()` and `SetSize()`:

**SetSize changes:**

```go
func (p *NowPlayingPane) SetSize(width, height int) {
    p.width = width
    p.height = height

    contentWidth := paneMax(width-4, 10)

    // Two-column split: ~1/3 left, ~2/3 right (min 28 for controls).
    infoWidth := paneMax(contentWidth/3, 28)
    vizWidth := contentWidth - infoWidth - 1

    bodyHeight := paneMax(height-4, 4)

    p.infoBox.SetSize(infoWidth, bodyHeight)

    // Viz rows split around the seek bar (1 row).
    // Engine gets the full bodyHeight for rendering; we'll split
    // the frame into top/bottom portions in View().
    vizHeight := paneMax(bodyHeight-1, 1) // -1 for seek bar row
    p.engine.SetSize(vizWidth, vizHeight)

    // Seek bar now sized to right panel width (not full content width).
    p.seekBar.SetWidth(vizWidth)
    p.volumeBar.SetWidth(infoWidth - 4)
}
```

**View changes:**

```go
func (p *NowPlayingPane) View() string {
    ps := p.store.PlaybackState()
    if ps == nil || ps.Item == nil {
        return p.renderEmpty()
    }

    t := ps.Item

    // ... (existing style + artist + volume + controls code) ...

    infoView := p.infoBox.Render("Track Info", infoLines, p.focused)

    // Right panel: viz top rows + seek bar + viz bottom rows.
    frame := p.engine.CurrentFrame()
    topRows, bottomRows := splitFrame(frame)
    topView := renderStyledLines(topRows)
    bottomView := renderStyledLines(bottomRows)
    seekBar := p.seekBar.Render(p.localProgressMs, t.DurationMs)

    rightPanel := lipgloss.JoinVertical(lipgloss.Left, topView, seekBar, bottomView)

    return lipgloss.JoinHorizontal(lipgloss.Top, infoView, " ", rightPanel)
}
```

**New helper functions:**

```go
// splitFrame divides a frame into top and bottom halves around a center row.
// topRows = (len(frame)) / 2, bottomRows = len(frame) - topRows.
func splitFrame(f viz.Frame) (top, bottom viz.Frame) {
    if len(f) == 0 {
        return nil, nil
    }
    mid := len(f) / 2
    return f[:mid], f[mid:]
}

// renderStyledLines joins StyledLines into a single string with per-line coloring.
func renderStyledLines(lines viz.Frame) string {
    if len(lines) == 0 {
        return ""
    }
    rows := make([]string, len(lines))
    for i, line := range lines {
        style := lipgloss.NewStyle().Foreground(line.Color)
        rows[i] = style.Render(line.Text)
    }
    return strings.Join(rows, "\n")
}
```

**Files:**
- Modify: `internal/ui/panes/nowplaying.go`
- Modify: `internal/ui/panes/nowplaying_test.go`

**Tests:**
- Unit: `SetSize` sets seek bar width to `vizWidth` (not full `contentWidth`)
- Unit: View contains seek bar text (time labels) between viz rows
- Unit: View contains braille/block characters (from engine) above and below seek bar
- Unit: InfoBox is on the left (~1/3 width), viz+seekbar on the right (~2/3 width)
- Unit: `splitFrame` with 6 lines → 3 top, 3 bottom
- Unit: `splitFrame` with 5 lines → 2 top, 3 bottom
- Unit: `splitFrame` with 0 lines → nil, nil
- Unit: `renderStyledLines` applies color from each StyledLine
- Edge: Height too small for visualization → graceful degradation (empty viz area)

**Commit:** `feat(ui): two-column layout with seek bar in right panel`

---

## Task 6: Vertical centering in expanded mode

**Problem:** In expanded state (large height from preset 1), content stretches to
fill the available space. Should remain fixed-height and vertically centered.

**Fix:**

Modify `internal/ui/panes/nowplaying.go` — update `View()` to wrap the composite
output with `lipgloss.Place` when there is excess vertical space:

```go
func (p *NowPlayingPane) View() string {
    // ... (existing View code that builds the composite) ...

    composite := lipgloss.JoinHorizontal(lipgloss.Top, infoView, " ", rightPanel)

    // If pane is taller than the content, vertically center the block.
    contentHeight := lipgloss.Height(composite)
    availableHeight := paneMax(p.height-2, 1) // subtract border chrome
    if contentHeight < availableHeight {
        contentWidth := paneMax(p.width-4, 10)
        composite = lipgloss.Place(contentWidth, availableHeight,
            lipgloss.Center, lipgloss.Center, composite)
    }

    return composite
}
```

The `lipgloss.Place` call wraps the **entire composite** (left panel + right panel
joined horizontally). The dimensions passed are `(pane content width, pane content height)`
and the content is placed at `(Center, Center)`. This centers the whole content block
as a unit — the internal layout of left/right panels is not affected by centering.

**Files:**
- Modify: `internal/ui/panes/nowplaying.go`
- Modify: `internal/ui/panes/nowplaying_test.go`

**Tests:**
- Unit: `SetSize(80, 30)` (expanded) → View output height = 30 lines (padded)
- Unit: Expanded mode → content block is surrounded by blank lines (centered)
- Unit: `SetSize(80, 10)` (compact) → View output has no extra centering padding
- Unit: Compact and expanded produce same core content (track, viz, seekbar)
- Edge: Content exactly fills available height → no centering applied

**Commit:** `feat(ui): vertical centering in expanded NowPlaying`

---

## Task 7: Delete old visualizer and update tests

**Problem:** The old `internal/ui/components/visualizer.go` is now unused after
the migration to `viz.Engine`. Tests need updating for new glyphs and layout.

**Fix:**

1. Delete `internal/ui/components/visualizer.go`
2. Delete `internal/ui/components/visualizer_test.go`
3. Update `internal/ui/components/visualizer_gradient_integration_test.go`:
   - Delete `TestIntegration_Visualizer_Lifecycle` (uses deleted `NewVisualizer`)
   - Delete `TestIntegration_Visualizer_PlayPauseCycle` (uses deleted `VisualizerTickMsg`)
   - Update `TestIntegration_AllComponentsRenderWithinWidth`: remove `NewVisualizer` section
   - Update `TestIntegration_NoHardcodedHexInComponents`: remove `NewVisualizer` section
   - Update volume bar assertion: `VOL` → `♪`
4. Update `internal/ui/panes/nowplaying_test.go`:
   - All `components.VisualizerTickMsg` references → `viz.TickMsg`
   - Control assertions: `|<` → removed, `||` → `⏸`, `>` → `▷`, `~` → `⇄`, `=>` → `↻`
   - Volume assertions: `VOL` → `♪`, `█` → `■`, `░` → `□`
   - Actions assertions: 2 actions → 5 actions
   - Layout assertions: seek bar in right panel, not at bottom
   - Add expanded-mode centering test

**Files:**
- Delete: `internal/ui/components/visualizer.go`
- Delete: `internal/ui/components/visualizer_test.go`
- Modify: `internal/ui/components/visualizer_gradient_integration_test.go`
- Modify: `internal/ui/panes/nowplaying_test.go`

**Tests:**
- Build: No remaining imports of `components.Visualizer` or `components.VisualizerTickMsg`
- Unit: All existing NowPlaying tests pass with updated assertions
- Unit: Integration test passes with `♪` and `■` volume bar assertions
- Unit: Controls tests pass with Unicode glyph assertions
- Integration: Full `make ci` passes with no lint or coverage failures

**Commit:** `refactor(ui): delete old visualizer, update test assertions`

---

## Task 8: Update DESIGN.md keybindings table

**Problem:** The keybindings table in `docs/DESIGN.md` §17 needs to reflect that
border actions now surface `space`, `+/-`, and `v` shortcuts.

**Fix:**

1. Cross-check `docs/DESIGN.md` §17 keybindings table
2. Verify the keys (`space`, `+/-`, `v`, `s`, `r`) already exist as keybindings —
   they are now surfaced in the border, not new bindings
3. If any discrepancy, update the table to match

**Files:**
- Modify: `docs/DESIGN.md` (if needed — may be a no-op if already correct)

**Tests:**
- Docs: Border action keys match keybindings table entries

**Commit:** `docs(design): verify NowPlaying border actions in keybindings table`

---

## Acceptance Criteria

- [ ] Controls row shows `⇄  ▷  ≡  ↻` (Unicode glyphs, no Previous/Next)
- [ ] Volume bar shows `♪ ■■■□□□ 31%` format with correct mute/unmute coloring
- [ ] Border actions include all 5 shortcuts: `s`, `r`, `space`, `+/-`, `v`
- [ ] NowPlayingPane uses `*viz.Engine` instead of `*components.Visualizer`
- [ ] `app.go` routes `viz.TickMsg` instead of `components.VisualizerTickMsg`
- [ ] Seek bar is inside the right panel between top and bottom viz rows
- [ ] Seek bar width = right panel width (not full pane width)
- [ ] Info panel ~1/3 width, viz+seekbar panel ~2/3 width
- [ ] Expanded mode: content block vertically centered via `lipgloss.Place`
- [ ] Compact mode (height < 8): inline track info in border title (unchanged)
- [ ] Old `visualizer.go` and `visualizer_test.go` deleted
- [ ] All `v` key presses cycle through 7 patterns (via engine)
- [ ] No hardcoded hex values — all colors from Theme interface
- [ ] `make ci` passes (lint + tests + 80% coverage)

---

## Notes

- The `GradientSeekBar` component itself is unchanged — only its width input and
  position in the View composition change. No modifications to the seek bar's
  rendering logic.
- The `InfoBox` component (`internal/ui/components/infobox.go`) is unchanged
  structurally. The info lines passed to it update (new control glyphs, new volume
  format) but the component's render logic stays the same.
- Layout presets (`internal/ui/layout/presets.go`) are NOT modified — preset weights
  stay the same. The NowPlaying pane adapts its internal layout to the height it receives.
- The Store (`internal/state/store.go`) has no changes — all data flows remain the same.
- The `renderStyledLines` helper and `splitFrame` helper are private to the pane file.
  They don't need to be in a shared package since only NowPlaying uses them.
- The existing compact title mode (height < 8) continues to work as-is. The height
  threshold is unchanged.
