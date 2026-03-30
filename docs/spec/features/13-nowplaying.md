---
title: "NowPlaying Pane"
description: "A rich terminal music player pane with btop-inspired split layout, braille and block character visualizer engine with per-row color gradients, Unicode transport controls, and adaptive rendering from compact strips to expanded centered layouts."
status: done
stories: [45, 58, "58b", 59, 60]
---

# NowPlaying Pane

## Background

The NowPlaying pane is the centerpiece of Spotnik's terminal UI, displaying the currently playing track with transport controls, audio visualization, and playback progress. It evolved from a simple `PlayerPane` with monochrome seek and volume bars into a sophisticated btop-inspired split layout featuring a Track Info sub-pane on the left and an animated braille/block visualizer on the right, with a gradient seek bar embedded between visualization rows in the right panel.

The visualizer system was extracted into a dedicated `viz/` package providing a `Renderer` interface with two implementations (braille and block characters), 7 animation patterns (4 braille, 3 block), per-row color gradients using theme tokens (Gradient1/2/3), and a precomputed frame table for smooth 200ms tick animation. The engine accepts any height with no hard cap, supports pattern cycling via the `v` key, and replaces the original single-color 4-line-max visualizer.

The pane adapts to its allocated space: in expanded presets it vertically centers its content block via `lipgloss.Place`; in compact presets (height < 8) it embeds track info directly in the border title. Transport controls were upgraded to Unicode glyphs, the volume bar switched to discrete block characters with a music note icon, and border actions were expanded to surface all relevant keyboard shortcuts.

---

## Story: NowPlaying Pane (spec 45)

### Background
The original `PlayerPane` rendered the currently playing track with transport controls, a monochrome seek bar, and volume bar. It did not implement the `layout.Pane` interface and had no visualizer or gradient effects. This story renamed the pane to `NowPlayingPane`, implemented the `layout.Pane` interface, embedded the braille visualizer and gradient bars, and added compact mode for presets where NowPlaying occupies a small row.

### Acceptance Criteria
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

### Tasks
1. **Rename PlayerPane to NowPlayingPane** — Rename files, struct, constructor, and all references.
   - Files: `internal/ui/panes/player.go` -> `internal/ui/panes/nowplaying.go`, `internal/ui/panes/player_test.go` -> `internal/ui/panes/nowplaying_test.go`, `internal/app/app.go`, `internal/app/render.go`, `internal/app/routing.go`
   - Tests: Existing player tests pass under new name, build succeeds with no broken references
   - Commit: `refactor(ui): rename PlayerPane to NowPlayingPane`

2. **Implement layout.Pane interface** — Add `ID()` returning `layout.PaneNowPlaying`, `Title()` returning "Now Playing", `ToggleKey()` returning 1, `Actions()` returning shuffle + repeat. Ensure `SetSize`, `SetFocused`, `IsFocused` match interface signatures.
   - Files: `internal/ui/panes/nowplaying.go`
   - Tests: Unit tests for `ID()`, `Title()`, `ToggleKey()`, `Actions()`, compile-time check `var _ layout.Pane = &NowPlayingPane{}`
   - Commit: `feat(ui): NowPlayingPane implements layout.Pane interface`

3. **Embed visualizer component** — Add `visualizer *components.Visualizer` field, initialize in constructor, handle `VisualizerTickMsg` in `Update()`, set playing state from `PlaybackStateFetchedMsg`, compute `vizHeight` based on mode (1 compact, 2 full, 4 expanded), render in full mode `View()` above track info.
   - Files: `internal/ui/panes/nowplaying.go`
   - Tests: Visualizer starts ticking after Init(), PlaybackStateFetchedMsg controls animation, VisualizerTickMsg advances frame, View() in full mode contains braille characters when playing
   - Commit: `feat(ui): embed braille visualizer in NowPlayingPane`

4. **Replace bars with gradient versions** — Replace `ProgressBar` with `GradientSeekBar`, replace `VolumeBar` with `GradientVolumeBar`, update `SetSize()` and `View()` to use new API (`seekBar.Render(progressMs, durationMs)`, `volumeBar.Render(volume)`).
   - Files: `internal/ui/panes/nowplaying.go`
   - Tests: Seek bar renders gradient colors, volume bar shows correct color band, bars resize correctly with `SetSize()`
   - Commit: `feat(ui): gradient seek bar and volume bar in NowPlayingPane`

5. **Implement compact mode** — Add `compact bool` field, detect in `SetSize()` when `height <= 3`, render single content line with controls + volume, override `Title()` to include track name and progress when compact, no visualizer in compact mode.
   - Files: `internal/ui/panes/nowplaying.go`
   - Tests: `SetSize(80, 3)` enables compact, `SetSize(80, 10)` disables, compact `Title()` includes track info, compact `View()` has 1 content line, full `View()` has visualizer + track info + bars, no visualizer in compact
   - Commit: `feat(ui): compact mode for NowPlaying in small presets`

6. **Comprehensive tests** — Full lifecycle, layout integration, mode transitions, playing/paused state, shuffle/repeat, volume changes, progress changes, interface satisfaction, nil state edge case, zero duration edge case.
   - Files: `internal/ui/panes/nowplaying_test.go`
   - Tests: Integration tests for full lifecycle, full mode layout, compact mode layout, transition full/compact, playing/paused visualizer, shuffle/repeat controls, volume gradient, progress gradient, interface satisfaction, nil playback state, zero duration
   - Commit: `test(ui): comprehensive NowPlayingPane tests`

---

## Story: NowPlaying Split Layout (spec 58)

### Background
The NowPlayingPane's vertical stack layout was replaced with a btop-inspired horizontal split: an InfoBox sub-pane on the left (~1/4 width) containing track info, controls, and volume bar, and the braille visualizer on the right (~3/4 width), with a gradient seek bar spanning full width at the bottom. The `compact` boolean field and `renderCompact()` method were removed in favor of a height < 8 check in `Title()` for inline track info. Several compact-mode helper functions (`interpolateHexCompact`, `parseHexParts`, `lerpByte`) were also cleaned up.

### Acceptance Criteria
- [ ] NowPlayingPane has `infoBox` field initialized in constructor
- [ ] SetSize computes split dimensions for InfoBox, Visualizer, seek bar, volume bar
- [ ] `compact` field and `renderCompact()` removed
- [ ] `interpolateHexCompact`, `parseHexParts`, `lerpByte` removed
- [ ] View() renders horizontal split: InfoBox left, Visualizer right, seek bar bottom
- [ ] Title() uses height < 8 check instead of compact flag
- [ ] All old compact-mode tests deleted
- [ ] New split layout tests added and passing
- [ ] `make ci` passes

### Tasks
1. **Rewrite NowPlayingPane View() with split layout** — Add `infoBox *components.InfoBox` field initialized via `components.NewInfoBox(t)`. Update `SetSize()` to compute split dimensions: `infoWidth = paneMax(contentWidth/4, 28)`, `vizWidth = contentWidth - infoWidth - 1`, `bodyHeight = paneMax(height-4, 4) - progressHeight`. Remove `compact bool` field, delete `renderCompact()` method, delete `interpolateHexCompact()`, `parseHexParts()`, `lerpByte()` helpers. Rewrite `View()` to render InfoBox left (track name, artist, album, controls, volume bar) and Visualizer right via `lipgloss.JoinHorizontal`, seek bar at bottom via `lipgloss.JoinVertical`. Update `Title()` to use `height < 8` check instead of compact flag for inline track info.
   - Files: `internal/ui/panes/nowplaying.go`, `internal/ui/panes/nowplaying_test.go`
   - Tests:
     - DELETE: `TestNowPlayingPane_CompactMode_EnabledAtHeight3`, `TestNowPlayingPane_CompactMode_DisabledAtHeight10`, `TestNowPlayingPane_CompactMode_DisabledAtHeight4`, `TestNowPlayingPane_CompactView_SingleContentLine`, `TestNowPlayingPane_CompactView_ContainsVol`, `TestNowPlayingPane_NoVisualizerInCompactMode`, `TestNowPlayingPane_Transition_FullToCompact`, `TestNowPlayingPane_CompactView_NilState`
     - UPDATE: `TestNowPlayingPane_CompactTitle_IncludesTrackInfo` — test Title() when height < 8
     - ADD: `TestNowPlayingPane_SplitLayout_ContainsInfoBoxBorders` (View() at 80x24 contains rounded border chars), `TestNowPlayingPane_SplitLayout_ContainsBraille` (braille characters present), `TestNowPlayingPane_SplitLayout_ContainsSeekBar` (time stamps from seek bar), `TestNowPlayingPane_SplitLayout_ContainsVolumeInInfoBox` (contains "VOL"), `TestNowPlayingPane_SplitLayout_ContainsControls` (control characters), `TestNowPlayingPane_Title_ShowsTrackInfoWhenSmall` (Title() at height 6 includes track name), `TestNowPlayingPane_Title_DefaultWhenTall` (Title() at height 24 is "Now Playing"), `TestNowPlayingPane_SplitLayout_AdaptsToDifferentSizes` (different sizes produce different output)
   - Commit: Tasks combined into single commit for this story

---

## Story: NowPlaying Design Docs Update (spec 58b)

### Background
The design documentation needed updating to reflect the new btop-inspired NowPlaying pane split layout, the 3 animation patterns and `v` key cycling behavior, and the responsive proportions of the InfoBox/Visualizer split. All preset diagrams in DESIGN.md were updated to show the new rendering style.

### Acceptance Criteria
- [ ] DESIGN.md §2 (Pane Definitions) updated with NowPlaying split layout note
- [ ] DESIGN.md §4 (Presets) — all preset diagrams updated for split layout (InfoBox left, Visualizer right, seek bar bottom)
- [ ] DESIGN.md §11 (Visual Components) documents 3 animation patterns and `v` key cycling
- [ ] DESIGN.md §11 new "NowPlaying Split Layout" subsection added describing proportions and responsive behavior
- [ ] All preset diagrams (0, 1, 2, 3) show the new NowPlaying rendering style
- [ ] `make ci` passes

### Tasks
1. **Update DESIGN.md with NowPlaying split layout documentation** — Update §2 Pane Definitions Key Notes to add split layout note. Update §4 Presets diagrams for all presets (0-3) to show InfoBox + Visualizer side-by-side layout. Update §11 Visual Components to document 3 animation patterns (Dual Sine Wave, Standing Wave, Pulse/Ripple) with `v` key cycling. Add new "NowPlaying Split Layout" subsection to §11 documenting:
   - Layout proportions: Left ~1/4 width (min 28 chars) InfoBox with rounded border, Track Info title, track name (bold TextPrimary), artist names (TextSecondary), album name (TextMuted), controls row, volume bar, vertically centered content. Right ~3/4 width animated braille visualizer (full body height). Bottom full-width 1-line gradient seek bar with time labels.
   - Responsive behavior: `infoWidth = max(contentWidth/4, 28)`, `vizWidth = contentWidth - infoWidth - 1`, `bodyHeight = paneHeight - borders - progressBar`. Height < 8 embeds track info in title bar. No separate compact mode.
   - InfoBox border: standard rounded corners, color follows `ActiveBorder()`/`InactiveBorder()` based on focus.
   - Pattern state local to pane (not in Store), `v` key routes via `isPlaybackKey()`.
   - Files: `docs/DESIGN.md`
   - Tests: Docs change, no code tests needed, `make ci` passes

---

## Story: Visualizer Engine (spec 59)

### Background
The existing visualizer component rendered braille-dot patterns in a single color with a hard 4-line height cap and 3 animation patterns baked into a monolithic file. The NowPlaying redesign required per-row color gradients (green base, yellow mid, red peaks), block character rendering mode alongside braille, 7 animation patterns, no height cap, and an extensible `Renderer` interface. This story created the `viz/` package as a clean extraction, to be integrated into NowPlaying by Feature 60.

### Acceptance Criteria
- [ ] `viz/` package compiles independently with no imports from `components/` or `panes/`
- [ ] `Renderer` interface has two implementations: `BrailleRenderer` and `BlockRenderer`
- [ ] 7 patterns registered: 4 braille, 3 block
- [ ] Per-row color gradient uses `Gradient1/2/3` theme tokens (no hardcoded hex)
- [ ] Engine has no height cap — accepts any height via `SetSize()`
- [ ] Frame table precomputed (40 frames) on `SetSize()` and `CyclePattern()`
- [ ] `CurrentFrame()` returns blank frame when paused
- [ ] `CyclePattern()` cycles through all 7 patterns and wraps
- [ ] `TickMsg` type defined in `viz` package (replaces `components.VisualizerTickMsg`)
- [ ] All height functions are deterministic (no `math/rand`)
- [ ] 80%+ test coverage on the `viz/` package
- [ ] `make ci` passes

### Tasks
1. **Create Frame and StyledLine types** — Create `internal/ui/components/viz/frame.go` with `StyledLine` struct (Text string, Color lipgloss.Color) and `Frame` type (slice of StyledLine).
   - Files: `internal/ui/components/viz/frame.go`
   - Tests: StyledLine stores text and color correctly, Frame slice can be created and indexed, package compiles
   - Commit: `feat(viz): Frame and StyledLine types`

2. **Create Renderer interface and braille renderer** — Create `internal/ui/components/viz/braille.go` with `Renderer` interface (`RenderFrame(width, height int, colHeights []int, colors []lipgloss.Color) Frame`) and `BrailleRenderer` struct implementing it. Port existing `renderFrame` and `brailleChar` logic from `visualizer.go`. Each braille column maps to a height value (0 to `height*4` dot rows), iterate rows top-to-bottom, assign `colors[rowIndex]` to each StyledLine. Braille encoding: 0 dots `U+2800`, 1 dot `U+2840`, 2 dots `U+2860`, 3 dots `U+2870`, 4 dots `U+28F0`.
   - Files: `internal/ui/components/viz/braille.go`
   - Tests: Compile-time interface check, RenderFrame returns correct number of StyledLines, text contains only braille runes (U+2800-U+28FF), colors match input, full/zero column heights, frame width matches input, zero width/height edge cases
   - Commit: `feat(viz): Renderer interface and braille renderer`

3. **Create block renderer** — Create `internal/ui/components/viz/block.go` with `BlockRenderer` struct implementing `Renderer`. Each column is 1 character wide, heights map to filled rows bottom-up: full = `U+2588` (`█`), empty = space. Assign `colors[rowIndex]` to each StyledLine.
   - Files: `internal/ui/components/viz/block.go`
   - Tests: Compile-time interface check, RenderFrame returns correct StyledLines, text contains only block chars or spaces, colors match input, full/zero heights, frame width matches, zero width/height edge cases
   - Commit: `feat(viz): block character renderer`

4. **Create Pattern type and all 7 pattern definitions** — Create `internal/ui/components/viz/pattern.go` with `HeightFunc` type (`func(width, maxHeight, frameIdx int) []int`), `Pattern` struct (Name, Renderer, HeightFunc), and `Patterns()` function returning 7 patterns:
   - Pattern 0: Braille Dual Sine Wave (ported from old pattern 0, BrailleRenderer)
   - Pattern 1: Braille Standing Wave (ported from old pattern 1, BrailleRenderer)
   - Pattern 2: Braille Pulse/Ripple (ported from old pattern 2, BrailleRenderer)
   - Pattern 3: Block Dense Equalizer (full-height bars with slight variation, BlockRenderer)
   - Pattern 4: Block Waveform/Sine (smooth sine-based heights, BlockRenderer)
   - Pattern 5: Block Sparse/Low Amplitude (low height with occasional spikes, BlockRenderer)
   - Pattern 6: Braille Mid-density Organic (multi-frequency sine composition, BrailleRenderer)
   - All height functions use deterministic math (sine, cosine, Gaussian) — no `math/rand`. Frame count is 40 per pattern.
   - Files: `internal/ui/components/viz/pattern.go`
   - Tests: `Patterns()` returns exactly 7, each has non-empty Name, non-nil Renderer and HeightFunc, patterns 0-2 and 6 use BrailleRenderer, patterns 3-5 use BlockRenderer, HeightFunc returns slice of length `width` with values in [0, maxHeight], deterministic output, different patterns produce different profiles
   - Commit: `feat(viz): 7 animation patterns with height functions`

5. **Create Engine with frame orchestration** — Create `internal/ui/components/viz/engine.go` with `TickMsg` type (replaces `components.VisualizerTickMsg`), `Engine` struct (theme, patterns, patternIdx, frames []Frame, frameIdx, playing, width, height, interval). Constructor `NewEngine(th theme.Theme)` initializes with `Patterns()` list, pattern 0, 200ms interval. `SetSize(width, height)` regenerates 40 precomputed frames calling HeightFunc then Renderer.RenderFrame with per-row colors (top 1/3 Gradient3, middle 1/3 Gradient2, bottom 1/3 Gradient1). `Advance()` increments frameIdx modulo frame count when playing. `CurrentFrame()` returns blank frame when paused. `CyclePattern()` advances patternIdx and regenerates. `Init()` and `Update(msg)` manage tea.Tick loop.
   - Files: `internal/ui/components/viz/engine.go`
   - Tests: NewEngine creates engine with 7 patterns, PatternCount() returns 7, Pattern() returns 0 initially, SetSize produces correct frame dimensions, Advance increments/stays based on playing, CurrentFrame blank when paused/non-empty when playing, frame wraps after 40 advances, CyclePattern advances and wraps, CyclePattern regenerates frames, Init/Update return tick commands, per-row colors match gradient assignments, height=1/0 edge cases, Advance before SetSize no panic, CurrentFrame before SetSize returns empty Frame
   - Commit: `feat(viz): Engine with frame precomputation and pattern cycling`

6. **Comprehensive engine tests** — Table-driven tests across all 7 patterns and both renderers.
   - Files: `internal/ui/components/viz/engine_test.go`
   - Tests: For each of 7 patterns: frame dimensions match, non-empty when playing, blank when paused, braille patterns produce only braille runes, block patterns produce only block chars/spaces, color gradient assignment, deterministic output, different frames differ. Integration: full lifecycle, pattern cycling through all 7, resize mid-animation. Edge: width=1, height=1, large dimensions (200x20).
   - Commit: `test(viz): comprehensive engine and pattern tests`

---

## Story: NowPlaying Pane Redesign (spec 60)

### Background
After the split layout and visualizer engine were built, the NowPlaying pane needed final restructuring: moving the seek bar from full-width bottom into the right panel between top and bottom visualization rows, upgrading transport icons from ASCII to Unicode glyphs (`⇄ ▷ ⏸ ≡ ↻`), updating the volume bar to discrete small blocks (`■□`) with a music note icon (`♪`), expanding border actions to 5 shortcuts, adding vertical centering for expanded mode via `lipgloss.Place`, migrating from `*components.Visualizer` to `*viz.Engine`, and deleting the old visualizer code.

### Acceptance Criteria
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

### Tasks
1. **Update transport controls to Unicode glyphs** — Remove Previous (`|<`) and Next (`>|`) from controls row. Replace: Shuffle `~` -> `⇄` (U+21C4), Play `>` -> `▷` (U+25B7), Pause `||` -> `⏸` (U+23F8), Repeat off `=>` -> `↻` (U+21BB, inactive), Repeat context `=>` -> `↻` (active), Repeat track `=>1` -> `↻1` (active). Add Queue icon `≡` (U+2261, always TextSecondary). New format: `⇄  ▷  ≡  ↻`. Active/inactive coloring: `PlayingIndicator()` for active, `TextSecondary()` for inactive.
   - Files: `internal/ui/components/controls.go`, `internal/ui/components/controls_test.go`
   - Tests: Playing state contains `⏸`, paused contains `▷`, shuffle on/off `⇄` coloring, repeat off/context/track `↻`/`↻1` coloring, queue `≡` always present, output does NOT contain `|<`, `>|`, `~`, or `=>`
   - Commit: `feat(ui): Unicode transport control glyphs`

2. **Update volume bar characters and icon** — In `GradientVolumeBar.Render()`: replace `█` (U+2588) with `■` (U+25A0), `░` (U+2591) with `□` (U+25A1), `VOL` prefix with `♪` (U+266A). Volume > 0: `♪` in `Gradient1()` color. Volume = 0: `♪` in `TextMuted()` color. Format: `♪ ■■■■□□□□□□ 31%`. Adjust reserved width from 10 to 7 (`♪ ` = 2 chars vs `VOL  ` = 5 chars).
   - Files: `internal/ui/components/gradient.go`, `internal/ui/components/gradient_test.go`
   - Tests: Output contains `■` not `█`, `□` not `░`, `♪` not `VOL`, volume > 0 uses Gradient1 color for icon, volume = 0 uses TextMuted, clamping works, color bands unchanged, width adjustment correct with new reserved width
   - Commit: `feat(ui): volume bar with ■□ blocks and ♪ icon`

3. **Update border actions** — Update `Actions()` to return 5 actions: `{Key: "s", Label: "shfl"}`, `{Key: "r", Label: "rpt"}`, `{Key: "space", Label: "play"}`, `{Key: "+/-", Label: "vol"}`, `{Key: "v", Label: "viz"}`. Labels abbreviated to fit border.
   - Files: `internal/ui/panes/nowplaying.go`, `internal/ui/panes/nowplaying_test.go`
   - Tests: `Actions()` returns exactly 5 actions, correct keys and labels
   - Commit: `feat(ui): border actions for play, volume, viz toggle`

4. **Migrate NowPlayingPane from Visualizer to viz.Engine** — Replace `visualizer *components.Visualizer` field with `engine *viz.Engine`. Update constructor to `viz.NewEngine(t)`. Update `Init()` to `p.engine.Init()`. Change `components.VisualizerTickMsg` to `viz.TickMsg` in Update(). Update `handlePlaybackFetched()` to call `p.engine.SetPlaying(ps.IsPlaying)`. Update `handleKey()` for `v` key to `p.engine.CyclePattern()`. Update imports in `app.go` (message routing switch case), `requestflow_pane.go` (Update switch), and `requestflow_pane_test.go`.
   - Files: `internal/ui/panes/nowplaying.go`, `internal/app/app.go`, `internal/ui/panes/requestflow_pane.go`, `internal/ui/panes/requestflow_pane_test.go`
   - Tests: Constructor creates pane with engine, Init returns tick command, viz.TickMsg advances engine frame, PlaybackStateFetchedMsg controls engine playing state, `v` key cycles pattern, no remaining references to `components.VisualizerTickMsg`, app.go and requestflow_pane.go compile with viz.TickMsg
   - Commit: `refactor(ui): migrate NowPlayingPane to viz.Engine`

5. **Restructure View to two-column split with seek bar in right panel** — Update `SetSize()`: two-column split ~1/3 left, ~2/3 right (`infoWidth = paneMax(contentWidth/3, 28)`, `vizWidth = contentWidth - infoWidth - 1`), `vizHeight = paneMax(bodyHeight-1, 1)` (subtract 1 for seek bar row), engine gets vizHeight, seek bar width = vizWidth. Rewrite `View()`: get `CurrentFrame()` from engine, `splitFrame()` into top/bottom halves, render `topView` + seekBar + `bottomView` as right panel via `lipgloss.JoinVertical`, join with infoView via `lipgloss.JoinHorizontal`. Add private helpers `splitFrame(f viz.Frame) (top, bottom viz.Frame)` and `renderStyledLines(lines viz.Frame) string`.
   - Files: `internal/ui/panes/nowplaying.go`, `internal/ui/panes/nowplaying_test.go`
   - Tests: SetSize sets seek bar width to vizWidth, View contains seek bar between viz rows, View contains braille/block characters above and below seek bar, InfoBox left ~1/3, viz+seekbar right ~2/3, splitFrame with 6/5/0 lines, renderStyledLines applies per-line color, height too small edge case
   - Commit: `feat(ui): two-column layout with seek bar in right panel`

6. **Vertical centering in expanded mode** — In `View()`, after building composite, check if `contentHeight < availableHeight` (where `availableHeight = paneMax(p.height-2, 1)`). If so, wrap with `lipgloss.Place(contentWidth, availableHeight, lipgloss.Center, lipgloss.Center, composite)`. Centers entire content block as a unit without affecting internal left/right layout.
   - Files: `internal/ui/panes/nowplaying.go`, `internal/ui/panes/nowplaying_test.go`
   - Tests: SetSize(80, 30) expanded View output = 30 lines padded, content surrounded by blank lines, SetSize(80, 10) no extra centering, compact and expanded produce same core content, content exactly fills height = no centering applied
   - Commit: `feat(ui): vertical centering in expanded NowPlaying`

7. **Delete old visualizer and update tests** — Delete `internal/ui/components/visualizer.go` and `internal/ui/components/visualizer_test.go`. Update `internal/ui/components/visualizer_gradient_integration_test.go`: delete `TestIntegration_Visualizer_Lifecycle`, delete `TestIntegration_Visualizer_PlayPauseCycle`, update `TestIntegration_AllComponentsRenderWithinWidth` (remove NewVisualizer section), update `TestIntegration_NoHardcodedHexInComponents` (remove NewVisualizer section), update volume bar assertion `VOL` -> `♪`. Update `internal/ui/panes/nowplaying_test.go`: all `components.VisualizerTickMsg` -> `viz.TickMsg`, control assertions `|<` removed / `||` -> `⏸` / `>` -> `▷` / `~` -> `⇄` / `=>` -> `↻`, volume assertions `VOL` -> `♪` / `█` -> `■` / `░` -> `□`, actions assertions 2 -> 5, layout assertions for seek bar in right panel, add expanded-mode centering test.
   - Files: Delete `internal/ui/components/visualizer.go`, delete `internal/ui/components/visualizer_test.go`, modify `internal/ui/components/visualizer_gradient_integration_test.go`, modify `internal/ui/panes/nowplaying_test.go`
   - Tests: No remaining imports of `components.Visualizer` or `components.VisualizerTickMsg`, all NowPlaying tests pass with updated assertions, integration test passes with `♪` and `■`, controls tests pass with Unicode glyphs, full `make ci` passes
   - Commit: `refactor(ui): delete old visualizer, update test assertions`

8. **Update DESIGN.md keybindings table** — Cross-check `docs/DESIGN.md` §17 keybindings table, verify keys (`space`, `+/-`, `v`, `s`, `r`) exist as keybindings (now surfaced in border, not new bindings), update table if any discrepancy.
   - Files: `docs/DESIGN.md` (if needed — may be a no-op if already correct)
   - Tests: Border action keys match keybindings table entries
   - Commit: `docs(design): verify NowPlaying border actions in keybindings table`
