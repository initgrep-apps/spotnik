---
title: "NowPlaying Pane Redesign"
feature: 13-nowplaying
status: done
---

## Background
After the split layout and visualizer engine were built, the NowPlaying pane needed final restructuring: moving the seek bar from full-width bottom into the right panel between top and bottom visualization rows, upgrading transport icons from ASCII to Unicode glyphs, updating the volume bar to discrete small blocks with a music note icon, expanding border actions to 5 shortcuts, adding vertical centering for expanded mode, migrating from *components.Visualizer to *viz.Engine, and deleting the old visualizer code.

## Design

### Unicode Transport Controls
Remove Previous/Next from controls row. Replace: Shuffle ~ -> ⇄, Play > -> ▷, Pause || -> ⏸, Repeat off => -> ↻ (inactive), Repeat context -> ↻ (active), Repeat track -> ↻1 (active). Add Queue icon ≡ (always TextSecondary). Format: `⇄  ▷  ≡  ↻`.

### Volume Bar Update
Replace █ with ■, ░ with □, VOL with ♪. Volume > 0: ♪ in Gradient1(). Volume = 0: ♪ in TextMuted(). Format: `♪ ■■■■□□□□□□ 31%`.

### Border Actions
5 actions: {s, shfl}, {r, rpt}, {space, play}, {+/-, vol}, {v, viz}.

### viz.Engine Migration
Replace `visualizer *components.Visualizer` with `engine *viz.Engine`. Update Init(), Update() for viz.TickMsg, handlePlaybackFetched() for SetPlaying(), v key for CyclePattern(). Update app.go and requestflow_pane.go imports.

### Two-Column Split with Seek Bar in Right Panel
`infoWidth = paneMax(contentWidth/3, 28)`, `vizWidth = contentWidth - infoWidth - 1`, `vizHeight = paneMax(bodyHeight-1, 1)`. Seek bar width = vizWidth. splitFrame() into top/bottom halves, render topView + seekBar + bottomView as right panel.

### Vertical Centering
In View(), if contentHeight < availableHeight, wrap with lipgloss.Place(contentWidth, availableHeight, lipgloss.Center, lipgloss.Center, composite).

### Old Visualizer Cleanup
Delete visualizer.go and visualizer_test.go. Update integration tests for new assertions.

## Acceptance Criteria
- [ ] Controls row shows Unicode glyphs (⇄ ▷ ⏸ ≡ ↻)
- [ ] Volume bar shows ♪ ■■■□□□ 31% format
- [ ] Border actions include all 5 shortcuts
- [ ] NowPlayingPane uses *viz.Engine
- [ ] app.go routes viz.TickMsg
- [ ] Seek bar inside right panel between viz rows
- [ ] Info panel ~1/3 width, viz+seekbar ~2/3 width
- [ ] Expanded mode: vertically centered
- [ ] Compact mode: inline track info unchanged
- [ ] Old visualizer.go deleted
- [ ] v key cycles through 7 patterns
- [ ] No hardcoded hex values
- [ ] make ci passes

## Tasks
- [ ] Update transport controls to Unicode glyphs in internal/ui/components/controls.go
      - test: playing ⏸; paused ▷; shuffle ⇄; repeat ↻/↻1; queue ≡; no old ASCII symbols
- [ ] Update volume bar characters and icon in internal/ui/components/gradient.go
      - test: ■ not █; □ not ░; ♪ not VOL; correct icon coloring; width adjustment
- [ ] Update border actions in nowplaying.go -- 5 actions
      - test: Actions() returns exactly 5 with correct keys and labels
- [ ] Migrate NowPlayingPane from Visualizer to viz.Engine
      - test: constructor creates with engine; Init returns tick; viz.TickMsg advances; v cycles pattern; no VisualizerTickMsg references
- [ ] Restructure View to two-column split with seek bar in right panel
      - test: seek bar width = vizWidth; seek bar between viz rows; InfoBox ~1/3; splitFrame helper; height edge cases
- [ ] Vertical centering in expanded mode
      - test: SetSize(80, 30) padded to 30 lines; SetSize(80, 10) no extra centering; exact fill no centering
- [ ] Delete old visualizer and update tests
      - test: no remaining imports of components.Visualizer; all tests pass with updated assertions; make ci passes
- [ ] Update DESIGN.md keybindings table -- verify border action keys match
      - test: border action keys match keybindings table entries
