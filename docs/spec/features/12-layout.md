---
title: "Layout System"
description: "Provides the responsive grid layout engine, btop-style pane borders, reusable table/filter/visualizer components, app migration to the LayoutManager, restyled header/statusbar/overlays, and mouse scroll support — the full visual infrastructure that makes Spotnik a multi-pane, keyboard-and-mouse-driven terminal dashboard."
status: done
stories: [41, 42, 43, 44, 49, 50, 52]
---

# Layout System

## Background

Spotnik's original UI was a hardcoded 3-column layout (`libraryView | playerView | queueView`) assembled with `lipgloss.JoinHorizontal()`. There was no concept of pages, presets, pane toggling, or responsive grid reflow. The redesign replaces this with a btop-inspired responsive grid system where 10 panes are organized across 2 pages (Page A: Music, Page B: Nerd Status), Page A offers 4 presets (Full Dashboard, Listening, Library, Discovery), keys `1`-`8` toggle individual pane visibility, hidden panes redistribute space to visible siblings, and the grid recomputes on terminal resize and preset/toggle changes.

This feature encompasses the entire layout infrastructure: the LayoutManager that computes pane positions from grid definitions (spec 41), a custom border renderer that draws btop-style pane borders with embedded titles, toggle key superscripts, and action shortcuts (spec 42), reusable components including a dense Table wrapper around bubble-table, an in-pane Filter using bubbles/textinput, and text truncation utilities (spec 43), a braille-dot audio visualizer and gradient-colored seek/volume bars (spec 44), the full app migration from the old viewMode/focusedPane system to the LayoutManager-based grid (spec 49), restyled header, status bar, and overlays using bubbletea-overlay compositing (spec 50), and mouse scroll support with responsive terminal size handling (spec 52).

Together these stories transform Spotnik from a fixed 3-pane terminal app into a fully responsive, keyboard-and-mouse-driven multi-page dashboard with btop-style visual polish, dense data tables, animated visualizers, and overlay compositing.

---

## Story: Layout Infrastructure (spec 41)

### Background

The current UI uses a hardcoded 3-column layout assembled with `lipgloss.JoinHorizontal()` in `render.go`. There is no concept of pages, presets, pane toggling, or responsive grid reflow. The new DESIGN.md specifies a btop-inspired responsive grid system where 10 panes are organized across 2 pages, Page A has 4 presets, keys `1`-`8` toggle individual pane visibility, hidden panes redistribute space to visible siblings, and the grid recomputes on terminal resize and preset/toggle changes. This story builds the layout engine only — no pane implementations, no rendering. Subsequent stories build on this foundation.

Design reference: `docs/DESIGN.md` §2 (Pane Definitions), §3 (Layout Grid System), §4 (Pages, Pane Toggling, and Preset Layouts), §16 (Focus & Navigation), §22 (Architecture — LayoutManager)

### Acceptance Criteria
- [ ] `internal/ui/layout/` package compiles independently (no imports from `app/`, `api/`, `state/`)
- [ ] `PaneID` enum has 10 values (8 music + 2 nerd status)
- [ ] `Pane` interface defines all 8 methods (Init, Update, View, SetSize, SetFocused, IsFocused, ID, Title, ToggleKey, Actions)
- [ ] 4 Page A presets match DESIGN.md §4 exactly (pane sets and grid weights)
- [ ] 1 Page B preset matches DESIGN.md §4 (NowPlaying + RequestFlow + NetworkLog)
- [ ] Space distribution algorithm produces rects that tile the content area perfectly (no gaps, no overlap)
- [ ] Pane toggle redistributes space to siblings
- [ ] Row collapse works when all panes in a row are hidden
- [ ] Focus rotation skips hidden panes and wraps correctly
- [ ] Preset switch resets manual toggles
- [ ] `PaneAt()` correctly identifies pane from terminal coordinates
- [ ] `make ci` passes

### Tasks

1. **Define PaneID, PageID enums and Rect struct** — Create type-safe identifiers for panes and pages, plus the Rect struct for pane positioning.
   - Files: `internal/ui/layout/pane.go`
   - Tests:
     - Unit: `Rect.ContentWidth()` and `ContentHeight()` return correct values
     - Unit: `Rect.ContentWidth()` returns 0 for width < 2
     - Unit: PaneID constants have expected iota values (0-9)
     - Unit: PageID constants (PageA=0, PageB=1)
   - Key types: `PaneID` (10 values: PaneNowPlaying through PaneNetworkLog), `PageID` (PageA, PageB), `Rect` (X, Y, Width, Height with ContentWidth/ContentHeight methods), `Action` (Key, Label), `Pane` interface (tea.Model + SetSize, SetFocused, IsFocused, ID, Title, ToggleKey, Actions)

2. **Define grid model and preset data structures** — Create data structures representing the row-based grid layout and all preset configurations.
   - Files: `internal/ui/layout/presets.go`
   - Tests:
     - Unit: Each preset's Visible map matches its Grid cells (no pane in Grid that isn't in Visible)
     - Unit: PresetDashboard has 8 visible panes, 3 rows
     - Unit: PresetListening has 3 visible panes, 2 rows
     - Unit: PresetLibrary has 4 visible panes, 2 rows
     - Unit: PresetDiscovery has 4 visible panes, 3 rows
     - Unit: PresetNerdStatus has 3 visible panes, 3 rows
     - Unit: PageAPresets has 4 entries, PageBPresets has 1
   - Key types: `Cell` (PaneID, WidthWeight), `Row` (HeightWeight, Cells), `Preset` (Name, Visible map, Grid []Row)
   - Presets: PresetDashboard (8 panes, 3 rows with weights 2:3:3), PresetListening (3 panes, 2 rows with weights 3:2), PresetLibrary (4 panes, 2 rows with weights 1:4), PresetDiscovery (4 panes, 3 rows with weights 1:2:2), PresetNerdStatus (3 panes, 3 rows with weights 1:3:2)

3. **Implement LayoutManager with space distribution** — Build the Manager that computes pane positions from grid definitions and terminal size.
   - Files: `internal/ui/layout/layout.go`
   - Tests:
     - Unit: `NewManager()` starts on PageA, preset 0 (Dashboard), no hidden panes
     - Unit: `Resize(120, 30)` computes rects for all 8 Page A panes
     - Unit: Space distribution — all rects sum to content area (no gaps, no overflow)
     - Unit: Height weight 2:3:3 distributes correctly for 28 content rows (header+status=2)
     - Unit: Width weights 1:1:1 in a row divide evenly, last absorbs remainder
     - Unit: Single-cell row gets full width
     - Unit: `VisiblePanes()` returns 8 panes for Dashboard preset
     - Unit: `PaneRect()` returns zero Rect for hidden pane
     - Unit: Rects don't overlap (no pane covers another's area)
   - Key methods: `NewManager()`, `Resize(width, height int)`, `recompute()`, `PaneRect(id PaneID) Rect`, `VisiblePanes() []PaneID`, `ActivePage() PageID`, `ActivePresetIndex() int`, `ActivePresetName() string`
   - Space distribution algorithm: (1) Get current preset's Grid, (2) Build activeGrid by filtering hidden cells/empty rows, (3) Compute contentHeight = height - headerHeight - statusHeight, (4) Distribute height by HeightWeight (last row absorbs remainder), (5) Per row distribute width by WidthWeight (last cell absorbs remainder), (6) Compute Rect for each visible pane, (7) Build focusOrder from visible panes in grid order

4. **Implement page toggle, preset cycling, and pane toggling** — Add mechanisms to switch pages, cycle presets, and toggle individual pane visibility.
   - Files: `internal/ui/layout/layout.go`
   - Tests:
     - Unit: `TogglePage()` switches PageA<->PageB, returns correct `ActivePage()`
     - Unit: `TogglePage()` clears hidden state
     - Unit: `CyclePreset()` cycles 0->1->2->3->0 on Page A
     - Unit: `CyclePreset()` resets manual toggles
     - Unit: `TogglePane(PanePlaylists)` hides playlists, siblings expand
     - Unit: `TogglePane()` again restores playlists to original position
     - Unit: Hiding all panes in a row collapses the row, other rows expand
     - Unit: Cannot hide the last visible pane (toggle rejected)
     - Unit: `TogglePane()` does nothing on Page B (no toggle keys)
     - Unit: `IsPaneVisible()` reflects toggle state
   - Key methods: `TogglePage()`, `CyclePreset()`, `SetPreset(index int)`, `TogglePane(id PaneID)`, `IsPaneVisible(id PaneID) bool`

5. **Implement focus rotation** — Add keyboard focus rotation among visible panes using focusOrder built during recompute().
   - Files: `internal/ui/layout/layout.go`
   - Tests:
     - Unit: `RotateFocus(true)` cycles through all visible panes in order
     - Unit: `RotateFocus(false)` cycles in reverse
     - Unit: Focus wraps from last to first and vice versa
     - Unit: After `TogglePane()` hides focused pane, focus moves to first visible
     - Unit: After `CyclePreset()`, focus resets to first visible pane
     - Unit: `SetFocus(id)` changes focus to specified pane
     - Unit: `SetFocus(id)` no-op for hidden pane
     - Unit: `FocusedPane()` returns the correct PaneID
   - Key methods: `RotateFocus(forward bool)`, `FocusedPane() PaneID`, `SetFocus(id PaneID)`

6. **Add PaneAt hit-test for mouse support** — Implement hit-testing to identify which pane is under given terminal coordinates.
   - Files: `internal/ui/layout/layout.go`
   - Tests:
     - Unit: Click in center of NowPlaying rect returns `PaneNowPlaying`
     - Unit: Click in header area returns -1 (no pane)
     - Unit: Click in status bar area returns -1
     - Unit: Click on border between two panes returns the left/top pane (or either — just consistent)
     - Unit: Click outside all panes returns -1
   - Key method: `PaneAt(x, y int) PaneID` — iterates rects map, checks if (x, y) falls within any Rect's bounds accounting for header height offset

7. **Comprehensive layout tests** — Integration-level tests verifying the full layout lifecycle.
   - Files: `internal/ui/layout/layout_test.go`, `internal/ui/layout/pane_test.go`, `internal/ui/layout/presets_test.go`
   - Tests:
     - Integration: Full lifecycle: NewManager -> Resize -> CyclePreset -> TogglePane -> TogglePage -> verify rects
     - Integration: Resize from 120x30 to 80x24 -> all rects shrink proportionally
     - Integration: Dashboard preset with terminal 120x30 -> verify each pane gets expected rect
     - Integration: Hide 3 panes in row 3 -> row 3 collapses, rows 1+2 expand
     - Integration: Hide all panes in row 2 -> row disappears, rows 1+3 split remaining height
     - Integration: Toggle all 8 panes off one by one -> last toggle rejected, 1 pane remains
     - Integration: Preset cycle full loop (0->1->2->3->0) -> correct visible panes each time
     - Integration: Page B has 3 panes, no toggle keys work
     - Integration: Focus rotation after hiding panes -> skips hidden, wraps correctly
     - Edge: Zero-size terminal -> graceful handling (no panics, empty rects)
     - Edge: Very small terminal (1x1) -> no panics

---

## Story: Custom Border Renderer (spec 42)

### Background

The current border rendering (`internal/app/render.go:139-149`) uses `lipgloss.RoundedBorder()` with a single accent color. It has no title, no toggle key indicator, and no action shortcuts. The new DESIGN.md (§5 "Embedded Shortcut Borders") specifies btop-style borders where the top border line contains the pane title, toggle key superscript, and action shortcuts. Each pane has a distinct accent color (per-pane border tokens from the theme system). Focused panes show full accent color; unfocused panes use dimmed/faint borders. Filter mode replaces action shortcuts with the filter query.

Design reference: `docs/DESIGN.md` §5 (Embedded Shortcut Borders), §10 (Per-Pane Border Colors)

### Acceptance Criteria
- [ ] `RenderPaneBorder()` produces btop-style borders matching DESIGN.md §5 exactly
- [ ] Top border contains: corner, toggle key superscript, title, dashes, action shortcuts, corner
- [ ] Superscript digits (`¹`-`⁸`) render correctly for keys 1-8
- [ ] `ᐅ` prefix used for action labels with key in `KeyHint()` and label in `TextMuted()`
- [ ] Focused borders use full accent color, unfocused use `Faint(true)`
- [ ] Filter mode replaces actions with `filtering: "query"` text
- [ ] Output dimensions exactly match requested Width x Height
- [ ] Content is truncated/padded to fit inside border perfectly
- [ ] Unicode-safe width measurement using `lipgloss.Width()`
- [ ] No hardcoded hex colors — all colors come from Theme interface
- [ ] `make ci` passes

### Tasks

1. **Implement RenderPaneBorder function** — Build the btop-style border renderer with embedded title, toggle key, and action shortcuts.
   - Files: `internal/ui/layout/border.go`
   - Tests:
     - Unit: Basic border with title only — correct corner characters
     - Unit: Border with toggle key — superscript appears after `╭─ `
     - Unit: Border with 2 actions — both rendered right-aligned in top border
     - Unit: Border width matches requested width exactly
     - Unit: Border height matches requested height exactly
     - Unit: Content lines are padded/truncated to fit inside border
     - Unit: Filter mode — actions replaced with filter query text
     - Unit: ToggleKey=0 — no superscript rendered
     - Unit: Empty actions — only title and dashes in top border
     - Unit: Focused=true — accent color applied (verify styled output contains color)
     - Unit: Focused=false — faint style applied
   - Key types: `BorderConfig` (Width, Height, Title, ToggleKey, Actions, AccentColor, Focused, FilterQuery, Theme), function `RenderPaneBorder(content string, cfg BorderConfig) string`
   - Top border anatomy: `╭─ ` + superscript toggle key (`¹²³⁴⁵⁶⁷⁸`) in KeyHint() color + title in accent color (bold when focused) + dash fill + actions right-aligned (`ᐅ` + key in KeyHint() + label in TextMuted(), separated by ` ─── `) + ` ╮`
   - Side borders: `│` + content line padded to Width-2 + `│`
   - Bottom border: `╰` + `─` x (Width-2) + `╯`
   - Filter mode: `filtering: "query" ─── ᐅEsc close`
   - Superscript mapping: `var superscripts = map[int]string{1: "¹", 2: "²", 3: "³", 4: "⁴", 5: "⁵", 6: "⁶", 7: "⁷", 8: "⁸"}`

2. **Handle edge cases and content truncation** — Ensure borders handle narrow widths, content overflow, and Unicode safely.
   - Files: `internal/ui/layout/border.go`
   - Tests:
     - Unit: Very narrow border (width=15) — title truncated, no actions
     - Unit: Minimum border (width=10) — still renders valid border
     - Unit: Content shorter than height — padded with empty lines
     - Unit: Content wider than width — truncated with `…`
     - Unit: Unicode content (CJK characters, emoji) — width measured correctly
     - Unit: `ᐅ` character width measurement
   - Narrow border handling: First drop actions (show only title), if still too narrow truncate title with `…`, minimum width 10 characters
   - Content line handling: Each content line truncated to Width-2 using `lipgloss.Width()`, lines shorter than Width-2 padded with spaces, fewer lines than Height-2 padded with empty lines

3. **Border integration tests** — Verify borders work end-to-end with realistic pane configurations.
   - Files: `internal/ui/layout/border_test.go`
   - Tests:
     - Integration: NowPlaying border — title "Now Playing", key=1, actions=[shuffle, repeat], green accent
     - Integration: Playlists border — title "Playlists", key=3, actions=[filter, new, rename, delete], blue accent
     - Integration: Queue border with active filter — shows `filtering: "rock"` instead of actions
     - Integration: Page B RequestFlow border — title "Request Flow", key=0 (no superscript), orange accent
     - Integration: Border output is exactly Width x Height characters (measure each line)
     - Integration: Multiple borders side-by-side with `lipgloss.JoinHorizontal` — no overlap
     - Integration: Border with all 5 themes — accent colors change correctly

---

## Story: Reusable Components (spec 43)

### Background

The new DESIGN.md specifies that all 7 list panes (Queue, Playlists, Albums, LikedSongs, RecentlyPlayed, TopTracks, TopArtists) plus the Network Log pane render data in dense aligned columns with per-column colors, scrolling, and real-time filtering. Currently, panes render lists manually with `fmt.Sprintf` and fixed-width padding. There are no reusable table or filter components. This story creates three components: a Table wrapper around `github.com/evertras/bubble-table` with Spotnik conventions, an in-pane Filter using `bubbles/textinput`, and rune-aware truncation utilities.

Design reference: `docs/DESIGN.md` §6 (Content Containment), §8 (In-Pane Filtering), §9 (Dense Table Formatting)

### Acceptance Criteria
- [ ] `bubble-table` dependency added to go.mod
- [ ] `Truncate()`, `PadRight()`, `TruncateOrPad()` in `layout/truncate.go` use `lipgloss.Width()` for measurement
- [ ] `Table` wraps `bubble-table` with: flex columns, per-column colors, selected row highlighting, playing indicator, borderless mode
- [ ] `Filter` wraps `textinput` with: toggle, case-insensitive matching, border label, Esc/Enter handling
- [ ] `Table.SetSize()` recalculates column widths
- [ ] `Filter.Matches()` returns true when filter is inactive
- [ ] All 3 components have independent tests
- [ ] No imports from `app/`, `api/`, `state/` in these components
- [ ] `make ci` passes

### Tasks

1. **Add bubble-table dependency** — Add `github.com/evertras/bubble-table` to the project.
   - Files: `go.mod`, `go.sum`
   - Tests: Build: `go build ./...` succeeds

2. **Create truncation utilities** — Build rune-aware text truncation/padding functions using `lipgloss.Width()` for accurate rendered-width measurement.
   - Files: `internal/ui/layout/truncate.go`
   - Tests:
     - Unit: ASCII string shorter than maxWidth -> unchanged
     - Unit: ASCII string equal to maxWidth -> unchanged
     - Unit: ASCII string longer than maxWidth -> truncated with `…`
     - Unit: Unicode string with CJK characters -> correct width measurement
     - Unit: String with ANSI escape codes -> escapes ignored in width measurement
     - Unit: Empty string -> empty result
     - Unit: maxWidth=0 -> empty result
     - Unit: maxWidth=1 -> `…` only
     - Unit: `PadRight` shorter string -> padded with spaces to exact width
     - Unit: `PadRight` exact-width string -> unchanged
     - Unit: `TruncateOrPad` combines both — long strings truncated, short strings padded
   - Key functions: `Truncate(s string, maxWidth int) string`, `PadRight(s string, width int) string`, `TruncateOrPad(s string, width int) string`
   - Implementation: Uses `lipgloss.Width()` for measurement (handles CJK 2-cell, emoji, combining marks, ANSI escapes). Truncation iterates runes and sums widths to find cut point. `…` (U+2026) is 1 cell wide.

3. **Create Table component wrapper** — Build a reusable dense table wrapping `bubble-table` with Spotnik styling conventions.
   - Files: `internal/ui/components/table.go`
   - Tests:
     - Unit: `NewTable` with 4 columns creates correct bubble-table columns
     - Unit: `SetSize(80, 20)` -> columns scale proportionally
     - Unit: `SetRows` with 5 rows -> table shows 5 rows
     - Unit: `SelectedIndex()` returns 0 initially
     - Unit: Keyboard navigation (j/k or up/down) changes selected index
     - Unit: `SetPlayingIndex(2)` -> row 2 shows `▶` indicator
     - Unit: Column headers rendered in `TableHeader()` color
     - Unit: Per-column colors applied correctly
     - Unit: Selected row overrides column colors with selection colors
     - Unit: Empty rows -> renders headers only (or empty state)
     - Unit: Width recalculation after `SetSize` -> columns resize correctly
   - Key types: `ColumnDef` (Key, Header, FlexFactor, Color), `TableConfig` (Columns, Theme, PlayingIndex, ShowHeader), `Table` (inner btable.Model, config, width, height)
   - Key methods: `NewTable(cfg TableConfig) Table`, `SetSize(width, height int)`, `SetRows(rows []map[string]string)`, `SetPlayingIndex(index int)`, `SelectedIndex() int`, `Update(msg tea.Msg) tea.Cmd`, `View() string`
   - Implementation: Flex columns via `btable.NewFlexColumn()`, borderless mode via `btable.Border(customEmptyBox)`, row styling via `WithRowStyleFunc` (selected: SelectedBg/SelectedFg, playing: `▶` in PlayingIndicator color), header styling in TableHeader() color, `WithTargetWidth(width)`, `WithPageSize(height - headerLines)`

4. **Create Filter component** — Build an in-pane text filter using `bubbles/textinput`.
   - Files: `internal/ui/components/filter.go`
   - Tests:
     - Unit: `NewFilter()` starts inactive
     - Unit: `Toggle()` activates, second `Toggle()` deactivates
     - Unit: `Query()` returns current text
     - Unit: `Matches("Blinding Lights")` with query "blind" -> true
     - Unit: `Matches("Blinding Lights")` with query "xyz" -> false
     - Unit: `Matches("Blinding Lights")` with inactive filter -> true (always matches)
     - Unit: Matching is case-insensitive
     - Unit: `MatchesAny("Track", "Artist")` with query matching artist -> true
     - Unit: `BorderLabel()` returns `filtering: "rock"` when active with query "rock"
     - Unit: `BorderLabel()` returns "" when inactive
     - Unit: `View()` returns empty string when inactive
     - Unit: `View(40)` returns styled input bar when active
     - Unit: Esc key deactivates filter
     - Unit: Enter key deactivates filter but preserves query
   - Key types: `Filter` (input textinput.Model, active bool, query string, theme theme.Theme)
   - Key methods: `NewFilter(t theme.Theme) *Filter`, `Toggle()`, `IsActive() bool`, `Query() string`, `Matches(text string) bool`, `MatchesAny(texts ...string) bool`, `Update(msg tea.Msg) tea.Cmd`, `View(width int) string`, `BorderLabel() string`
   - Implementation: Placeholder "type to filter...", CharLimit 50, Width paneWidth-4, text in TextPrimary(), background in SurfaceAlt(). Esc deactivates and clears, Enter deactivates but keeps query. Matching: `strings.Contains(strings.ToLower(text), strings.ToLower(f.query))`

5. **Component integration tests** — Verify Table and Filter work together as they will in panes.
   - Files: `internal/ui/components/table_test.go`, `internal/ui/components/filter_test.go`, `internal/ui/layout/truncate_test.go`
   - Tests:
     - Integration: Table with Filter — set rows, activate filter, filter rows externally using `Filter.Matches()`, update table with filtered rows
     - Integration: Table resize cycle — set rows, resize smaller, verify no overflow
     - Integration: Filter activation -> pane border shows filter label, deactivation -> border shows actions
     - Integration: Truncate used on table cell values -> no overflow past column width
     - Integration: Table with 0 rows -> renders cleanly (no panic, shows headers or empty state)
     - Integration: Filter with empty query -> matches everything

---

## Story: Visualizer + Gradient Bars (spec 44)

### Background

The current player uses monochrome bars: `ProgressBar` with `SeekBar()` color and `VolumeBar` with `VolumeBar()` color (both in `internal/ui/components/`). The new DESIGN.md (§11) specifies a braille-dot audio visualizer that animates when music plays using Unicode braille characters (U+2800-U+28FF) with a precomputed frame table, a gradient seek bar where fill transitions from `Gradient1()` to `Gradient2()` left-to-right, and a volume bar with color bands: green (0-33%), yellow (34-66%), red (67-100%). These components are embedded in the NowPlaying pane.

Design reference: `docs/DESIGN.md` §11 (Visual Components — Braille-Dot Audio Visualizer, Gradient-Colored Bars)

### Acceptance Criteria
- [ ] Visualizer animates braille characters on 200ms tick when playing
- [ ] Visualizer shows flat-line when paused
- [ ] Frame table has 40 deterministic patterns (no randomness in View)
- [ ] Visualizer adapts to width/height via `SetSize()`
- [ ] Gradient seek bar interpolates color from `Gradient1()` to `Gradient2()`
- [ ] Volume bar uses 3 color bands: green (0-33%), yellow (34-66%), red (67-100%)
- [ ] All colors come from Theme interface tokens
- [ ] `interpolateHex` correctly handles RGB color interpolation
- [ ] No panics on edge cases (zero width, zero duration, extreme volumes)
- [ ] `make ci` passes

### Tasks

1. **Create braille-dot visualizer component** — Build an animated braille-dot audio spectrum visualizer.
   - Files: `internal/ui/components/visualizer.go`
   - Tests:
     - Unit: `NewVisualizer` starts with frameIndex=0
     - Unit: `SetPlaying(true)` + Update with VisualizerTickMsg -> frameIndex increments
     - Unit: `SetPlaying(false)` + Update with VisualizerTickMsg -> frameIndex stays same
     - Unit: `View()` when playing returns non-empty braille string
     - Unit: `View()` when paused returns flat-line pattern
     - Unit: Frame wraps after reaching end of frame table
     - Unit: `SetSize(40, 2)` -> View output is 2 lines, each 40 chars wide
     - Unit: `SetSize(80, 4)` -> View output is 4 lines, each 80 chars wide
     - Unit: Frame table has deterministic output (same frameIndex -> same output)
     - Unit: `Init()` returns a tick command
   - Key types: `VisualizerTickMsg time.Time`, `Visualizer` (theme, playing, frameIndex, width, height, interval, frames [][]string)
   - Key methods: `NewVisualizer(t theme.Theme) *Visualizer`, `SetSize(width, height int)`, `SetPlaying(playing bool)`, `Init() tea.Cmd`, `Update(msg tea.Msg) tea.Cmd`, `View() string`, `tickCmd() tea.Cmd`
   - Frame table: 40 frames generated deterministically using sine waves with phase offsets. Each frame is `[]string` (one per line of height). Braille chars encode 2x4 dot matrix. Regenerated on SetSize() width change. Color: `VisualizerFg()` token.
   - Animation: On VisualizerTickMsg, if playing increment frameIndex (wrap at len), re-arm tick at 200ms. If paused, still tick but don't increment.

2. **Create gradient seek bar component** — Build a seek bar with gradient fill transitioning from `Gradient1()` to `Gradient2()`.
   - Files: `internal/ui/components/gradient.go`
   - Tests:
     - Unit: `Render(0, 300000)` -> all empty characters
     - Unit: `Render(150000, 300000)` -> 50% filled
     - Unit: `Render(300000, 300000)` -> all filled
     - Unit: First filled char uses `Gradient1()` color
     - Unit: Last filled char uses `Gradient2()` color
     - Unit: Time labels correct: `"2:30"` format for 150000ms
     - Unit: Width changes -> bar length changes proportionally
     - Unit: `durationMs=0` -> safe handling (no division by zero)
   - Key types: `GradientSeekBar` (theme, width)
   - Key methods: `NewGradientSeekBar(t theme.Theme) *GradientSeekBar`, `SetWidth(width int)`, `Render(progressMs, durationMs int) string`
   - Implementation: Fill ratio calculation, linear RGB interpolation between Gradient1() and Gradient2() per filled position, each `█` with interpolated color, empty positions `░` in Surface() color. Format: `"1:41  ████████████████░░░░░░░░░░░░░░  5:30"`
   - Helper: `interpolateHex(hex1, hex2 string, t float64) lipgloss.Color` — parses hex strings like `#00ff88` into RGB, interpolates, returns new lipgloss.Color

3. **Create gradient volume bar component** — Build a volume bar with color bands based on volume level.
   - Files: `internal/ui/components/gradient.go`
   - Tests:
     - Unit: `Render(0)` -> no filled chars, "VOL ... 0%"
     - Unit: `Render(25)` -> green-colored fill (Gradient1)
     - Unit: `Render(50)` -> yellow-colored fill (Gradient2)
     - Unit: `Render(80)` -> red-colored fill (Gradient3)
     - Unit: `Render(100)` -> all filled, red
     - Unit: Volume clamped: `Render(150)` -> treated as 100
     - Unit: Volume clamped: `Render(-5)` -> treated as 0
     - Unit: Width changes -> bar length adjusts
   - Key types: `GradientVolumeBar` (theme, width)
   - Key methods: `NewGradientVolumeBar(t theme.Theme) *GradientVolumeBar`, `SetWidth(width int)`, `Render(volume int) string`
   - Implementation: Color bands: 0-33% Gradient1() (green/cool), 34-66% Gradient2() (yellow/warm), 67-100% Gradient3() (red/hot). Format: `"VOL  ████████░░░░░░  65%"`. Clamp volume to [0, 100].

4. **Integration tests** — Verify visualizer and gradient bars work end-to-end.
   - Files: `internal/ui/components/visualizer_test.go`, `internal/ui/components/gradient_test.go`
   - Tests:
     - Integration: Visualizer lifecycle — Init -> multiple VisualizerTickMsg updates -> View changes each frame
     - Integration: Visualizer play/pause cycle — play->tick->frame advances, pause->tick->frame freezes, play->tick->frame resumes
     - Integration: Seek bar at various progress points -> gradient visible in output
     - Integration: Volume bar threshold transitions: 33->34 (green->yellow), 66->67 (yellow->red)
     - Integration: All components render within specified width (no overflow)
     - Integration: All components use theme tokens (verify no hardcoded hex in output)

---

## Story: App Migration (spec 49)

### Background

The current `app.go` (~41KB) uses a `viewMode` enum (`viewMain`, `viewStats`, `viewPlaylists` plus `viewSplash`, `viewAuth`), a `focusedPane` enum (`focusPlayer`, `focusLibrary`, `focusQueue`), hardcoded 3-pane `JoinHorizontal` in `render.go`, individual pane fields (`playerPane`, `libraryPane`, `queuePane`, `statsPane`, `playlistPane`), and fixed pane widths (22%/50%/28%). This story replaces all of this with `layout.Manager` for grid computation, presets, page/pane toggling; a `panes map[PaneID]layout.Pane` for all 10 panes; `renderGrid()` that assembles panes into the grid using Manager.PaneRect(); `p` for preset cycling, `0` for page toggle, `1-8` for pane toggle; and `Tab/Shift+Tab` for focus rotation via Manager.RotateFocus().

Design reference: `docs/DESIGN.md` §3 (Layout Grid System), §4 (Pages, Presets, Toggling), §16 (Focus & Navigation), §22 (Architecture — LayoutManager), §23 (Migration)

### Acceptance Criteria
- [ ] `viewMode` reduced to `viewSplash | viewAuth | viewGrid`
- [ ] `focusedPane` enum deleted, replaced by `layout.Manager.FocusedPane()`
- [ ] Individual pane fields replaced by `panes map[PaneID]layout.Pane`
- [ ] `renderGrid()` assembles panes using `LayoutManager.PaneRect()`
- [ ] Key `0` toggles Page A/B
- [ ] Key `p` cycles presets within current page
- [ ] Keys `1-8` toggle pane visibility on Page A
- [ ] `Tab/Shift+Tab` rotates focus among visible panes
- [ ] Playback keys always route to NowPlaying
- [ ] All data-loaded messages route to correct new panes
- [ ] Overlays (search, devices) still work
- [ ] Minimum terminal size updated to 120x30
- [ ] Old `renderPaneWithBorder()` deleted
- [ ] `make ci` passes

### Tasks

1. **Replace viewMode and focus enums** — Consolidate view modes and remove the old focus system.
   - Files: `internal/app/app.go`
   - Tests:
     - Unit: App starts with `viewSplash`, transitions to `viewGrid` (not `viewMain`)
     - Unit: `layout.FocusedPane()` returns valid PaneID
     - Unit: All 8 Page A panes registered in `panes` map
   - Changes: Keep `viewSplash`, `viewAuth`; add `viewGrid` (replaces `viewMain`, `viewStats`, `viewPlaylists`); delete `viewMain`, `viewStats`, `viewPlaylists`. Remove `focusedPane` enum entirely. Add `layout *layout.Manager` and `panes map[layout.PaneID]layout.Pane` fields. Remove individual pane fields; keep `searchPane` and `devicePane` as separate overlay fields.

2. **Implement renderGrid()** — Replace `buildView()` body for grid mode with LayoutManager-based pane assembly.
   - Files: `internal/app/render.go`
   - Tests:
     - Unit: `renderGrid()` produces output with correct total dimensions
     - Unit: Each pane's border appears in the output
     - Unit: Grid respects preset layout (Dashboard shows 8 panes in 3 rows)
     - Unit: Hidden panes don't appear in output
   - Implementation: Group visible panes by row (using Rect.Y), for each row join cells horizontally using `lipgloss.JoinHorizontal`, wrap each pane in btop border via `layout.RenderPaneBorder()` with `BorderConfig`, cap each cell to exact rect dimensions with `lipgloss.NewStyle().Width().MaxWidth().Height().MaxHeight()`, join rows vertically.
   - Helper: `paneBorderColor(id layout.PaneID, t theme.Theme) lipgloss.Color` — switch on PaneID to return per-pane border color from theme.

3. **Wire key routing for pages, presets, and toggles** — Connect keyboard shortcuts to LayoutManager methods.
   - Files: `internal/app/routing.go`, `internal/app/app.go`
   - Tests:
     - Unit: Key `0` toggles page A<->B
     - Unit: Key `p` cycles presets (0->1->2->3->0)
     - Unit: Key `1` toggles NowPlaying visibility
     - Unit: Key `5` toggles LikedSongs visibility
     - Unit: Tab moves focus forward through visible panes
     - Unit: Shift+Tab moves focus backward
     - Unit: After toggle, pane sizes update correctly
     - Unit: Playback keys (Space, >, <, etc.) still route to NowPlaying regardless of focus
   - Key routing: `0` -> `layout.TogglePage()`, `p` -> `layout.CyclePreset()`, `1-8` -> `layout.TogglePane()` via toggleMap (`'1'`->PaneNowPlaying, `'2'`->PaneQueue, ..., `'8'`->PaneTopArtists), `tab` -> `layout.RotateFocus(true)`, `shift+tab` -> `layout.RotateFocus(false)`
   - Helpers: `propagateSizes()` — calls SetSize on all visible panes with computed Rects; `syncFocus()` — calls SetFocused(true/false) on all panes based on layout.FocusedPane()

4. **Wire WindowSizeMsg propagation** — Propagate terminal resize through LayoutManager to all panes.
   - Files: `internal/app/app.go`
   - Tests:
     - Unit: WindowSizeMsg updates layout dimensions
     - Unit: All visible panes receive SetSize after resize
     - Unit: Hidden panes don't receive SetSize (or receive zero)
   - Implementation: On `tea.WindowSizeMsg`, call `a.layout.Resize(msg.Width, msg.Height)` then `a.propagateSizes()`. `propagateSizes()` iterates visible panes calling `SetSize(contentWidth, contentHeight)` using `layout.PaneRect(id).ContentWidth()` and `.ContentHeight()`.

5. **Wire message routing to new panes** — Route data-loaded messages to the correct new split panes.
   - Files: `internal/app/app.go`, `internal/app/routing.go`
   - Tests:
     - Unit: QueueLoadedMsg reaches QueuePane
     - Unit: AlbumsLoadedMsg reaches AlbumsPane
     - Unit: StatsLoadedMsg reaches both TopTracks and TopArtists panes
     - Unit: PlaylistTracksLoadedMsg reaches PlaylistsPane
     - Unit: VisualizerTickMsg reaches NowPlayingPane
     - Unit: TickMsg reaches all panes
   - Message routing table:
     - `PlaybackStateFetchedMsg` -> NowPlaying pane
     - `QueueLoadedMsg` -> Queue pane
     - `LibraryLoadedMsg` -> Playlists pane
     - `AlbumsLoadedMsg` -> Albums pane
     - `LikedTracksLoadedMsg` -> LikedSongs pane
     - `RecentlyPlayedLoadedMsg` -> RecentlyPlayed pane
     - `StatsLoadedMsg` -> TopTracks pane + TopArtists pane
     - `FetchStatsMsg` -> dispatch API call
     - `PlaylistTracksLoadedMsg` -> Playlists pane
     - `Playlist*Msg` (create/rename/remove/reorder) -> Playlists pane
     - `VisualizerTickMsg` -> NowPlaying pane
     - `TickMsg` -> all panes (broadcast)
     - `tea.WindowSizeMsg` -> all panes (broadcast)

6. **Update buildView and remove old rendering** — Replace old view mode branches with grid rendering and delete legacy rendering code.
   - Files: `internal/app/render.go`
   - Tests:
     - Unit: buildView() in viewGrid mode renders grid with header + content + statusbar
     - Unit: Minimum size check uses 120x30
     - Unit: Splash and Auth modes still render correctly
     - Unit: Old viewStats/viewPlaylists code paths removed
   - Changes: `viewSplash` -> renderSplash() (unchanged), `viewAuth` -> renderAuthPanel() (unchanged), `viewGrid` -> header + renderGrid() + statusBar. Remove `viewStats` branch, `viewPlaylists` branch, old 3-pane JoinHorizontal. Update `renderTooSmall()` minimum from 100x24 to 120x30. Delete `renderPaneWithBorder()`.

7. **Initialize panes in App constructor** — Update `New()` to create all 10 panes and register them in the panes map.
   - Files: `internal/app/app.go`
   - Tests:
     - Unit: All 8 panes initialized and registered
     - Unit: App.Init() batches all pane Init() commands
     - Unit: App compiles and runs with new pane structure
   - Implementation: Create panes map with `layout.PaneNowPlaying` -> `panes.NewNowPlayingPane(store, theme)`, `layout.PaneQueue` -> `panes.NewQueuePane(store, theme)`, `layout.PanePlaylists` -> `panes.NewPlaylistsPane(store, theme)`, `layout.PaneAlbums` -> `panes.NewAlbumsPane(store, theme)`, `layout.PaneLikedSongs` -> `panes.NewLikedSongsPane(store, theme)`, `layout.PaneRecentlyPlayed` -> `panes.NewRecentlyPlayedPane(store, theme)`, `layout.PaneTopTracks` -> `panes.NewTopTracksPane(store, theme)`, `layout.PaneTopArtists` -> `panes.NewTopArtistsPane(store, theme)`. PaneRequestFlow and PaneNetworkLog added in Feature 51. Remove old field assignments.

8. **Comprehensive integration tests** — Full app lifecycle and migration verification tests.
   - Files: `internal/app/app_test.go`, `internal/app/render_test.go`, `internal/app/routing_test.go`
   - Tests:
     - Integration: Full app lifecycle — init -> resize -> load data -> render -> verify grid output
     - Integration: Preset cycling — p key -> layout changes, all panes resize
     - Integration: Page toggle — 0 key -> switches to Page B (empty for now), back to Page A
     - Integration: Pane toggle — hide pane 3 (Playlists) -> row reflows
     - Integration: Focus rotation — Tab cycles through all 8 visible panes
     - Integration: Playback keys work regardless of focus
     - Integration: Search overlay still opens and closes correctly
     - Integration: Device overlay still opens and closes correctly
     - Integration: Data flow — simulate API responses -> panes show data
     - Integration: Resize -> all panes adjust
     - Edge: Very small terminal -> "too small" message
     - Edge: Toggle all panes except one -> still renders

---

## Story: Header + Status Bar + Overlay Restyle (spec 50)

### Background

The current header shows `spotnik` left-aligned and device indicator right-aligned. The status bar shows context-sensitive keybinding hints (different per focused pane). Overlays use manual `lipgloss.Place()` for positioning. The new DESIGN.md (§12, §13, §14, §15) specifies a btop-style header with page indicator, preset name, and global action shortcuts; a status bar with global-only shortcuts (pane hints now live in borders); search/device overlays with `RenderPaneBorder()` borders; `bubbletea-overlay` for overlay compositing; and toast notifications repositioned to bottom-right.

Design reference: `docs/DESIGN.md` §12 (Notifications), §13 (Search Overlay), §14 (Device Switcher Overlay), §15 (Global Header & Status Bar)

### Acceptance Criteria
- [ ] `bubbletea-overlay` dependency added
- [ ] Header shows: spotnik, page, preset index, global action shortcuts, device
- [ ] Status bar shows global-only shortcuts (no pane-specific hints)
- [ ] Search overlay uses `RenderPaneBorder()` with btop-style border
- [ ] Device overlay uses `RenderPaneBorder()` with btop-style border
- [ ] Both overlays use `bubbletea-overlay` for compositing
- [ ] Search overlay centered, device overlay top-right
- [ ] Toast notifications positioned bottom-right
- [ ] All `ᐅ` action prefixes consistent across header, borders, overlays
- [ ] `make ci` passes

### Tasks

1. **Add bubbletea-overlay dependency** — Add `github.com/rmhubbert/bubbletea-overlay` to the project.
   - Files: `go.mod`, `go.sum`
   - Tests: Build: `go build ./...` succeeds

2. **Restyle header bar** — Rewrite `renderHeader()` to btop-style format with page, preset, and global action shortcuts.
   - Files: `internal/app/render.go`
   - Tests:
     - Unit: Header contains "spotnik"
     - Unit: Header shows current page (A or B)
     - Unit: Header shows current preset index
     - Unit: Header shows device name when active
     - Unit: Header shows "○ No device" when no device
     - Unit: Header fits exactly terminal width (no overflow, no underflow)
   - Implementation: Left side: `spotnik` (bold, TextPrimary) + `Page A/B` + `ᐅp preset N` + `ᐅ/ search` + `ᐅd devices`, joined with ` ─ `. Right side: `◉ DeviceName` or `○ No device`. Middle filled with `─`. Key labels in KeyHint() color, descriptions in StatusBarFg(), page/preset in PresetIndicator(), background in SurfaceAlt()/StatusBarBg().

3. **Restyle status bar to global-only** — Replace context-sensitive hints with a single global hint set.
   - Files: `internal/app/render.go`
   - Tests:
     - Unit: Status bar contains all global shortcuts
     - Unit: Status bar does NOT contain pane-specific hints (filter, etc.)
     - Unit: Status bar fits terminal width
   - Implementation: Fixed hints: `/` search, `0` page, `p` preset, `1-8` toggle, `Tab` pane, `d` devices, `?` help, `q` quit. Key in KeyHint(), label in StatusBarFg(), background StatusBarBg(). Remove `mainHints()`, `statsHints()`, `playlistsHints()`, status bar parameter.

4. **Update search overlay with btop borders + bubbletea-overlay** — Restyle the search overlay and use proper compositing.
   - Files: `internal/ui/panes/search.go`, `internal/app/render.go`
   - Tests:
     - Unit: Search overlay has btop-style border with title and actions
     - Unit: Search overlay centered on screen
     - Unit: Background is dimmed when overlay is open
     - Unit: Overlay compositing produces valid output
   - Implementation: Use `layout.BorderConfig` with Title "Search", ToggleKey 0, Actions [{Key: "Enter", Label: "play"}, {Key: "Tab", Label: "section"}], AccentColor theme.ActiveBorder(), Focused true. Replace `lipgloss.Place()` with `btoverlay.Composite(fg, dimmed, btoverlay.Center, btoverlay.Center, 0, 0)`.

5. **Update device overlay with btop borders + bubbletea-overlay** — Same pattern as search overlay for the device switcher.
   - Files: `internal/ui/panes/devices.go`, `internal/app/render.go`
   - Tests:
     - Unit: Device overlay has btop-style border
     - Unit: Device overlay positioned top-right
     - Unit: Active device shows `◉`, inactive shows `○`
   - Implementation: BorderConfig with Title "Devices", Actions [{Key: "Enter", Label: "select"}], AccentColor DeviceActive()/ActiveBorder(). Position: `btoverlay.Right, btoverlay.Top`.

6. **Reposition toast notifications to bottom-right** — Move toasts from default position to bottom-right per DESIGN.md §12.
   - Files: `internal/ui/components/notifications.go`, `internal/app/render.go` (if manual repositioning needed)
   - Tests:
     - Unit: Toast appears in bottom-right area of output
     - Unit: Toast doesn't interfere with grid content
     - Unit: Toast auto-dismisses after 4 seconds
   - Implementation: Use `alert.WithPosition(bubbleup.BottomRightPosition)` if supported, otherwise manually reposition via `lipgloss.Place()` in View().

---

## Story: Mouse Scroll + Responsive Behavior (spec 52)

### Background

The current app does not handle mouse events. Scrolling requires keyboard focus (`j`/`k`) on the target pane. The minimum terminal size is 100x24. The new DESIGN.md (§20, §21) specifies mouse wheel scrolling on any pane without changing focus (btop behavior), hit-test via `LayoutManager.PaneAt(x, y)` to identify the pane under the cursor, minimum terminal size increased to 120x30, and a friendly "needs more space" message when below minimum.

Design reference: `docs/DESIGN.md` §20 (Mouse Scroll Support), §21 (Responsive Behavior)

### Acceptance Criteria
- [ ] `tea.WithMouseCellMotion()` enabled at app startup
- [ ] Mouse wheel up/down scrolls the pane under the cursor
- [ ] Mouse scroll does NOT change keyboard focus
- [ ] `PaneAt()` hit-test correctly identifies pane from mouse coordinates
- [ ] Mouse scroll ignored when overlay is open
- [ ] Minimum terminal size check uses 120x30
- [ ] "Needs more space" message shows current and required dimensions
- [ ] `make ci` passes

### Tasks

1. **Enable mouse support at startup** — Add `tea.WithMouseCellMotion()` to program options.
   - Files: `cmd/root.go`
   - Tests:
     - Integration: App starts with mouse support enabled (verify program option)
   - Implementation: Add `tea.WithMouseCellMotion()` to `tea.NewProgram()` options alongside `tea.WithAltScreen()`.

2. **Handle mouse scroll events** — Route mouse wheel events to the pane under the cursor without changing focus.
   - Files: `internal/app/app.go`
   - Tests:
     - Unit: Mouse wheel up on Playlists pane -> Playlists scrolls up, focus unchanged
     - Unit: Mouse wheel down on Queue pane -> Queue scrolls down, focus unchanged
     - Unit: Mouse on header area -> no action (PaneAt returns -1)
     - Unit: Mouse on status bar -> no action
     - Unit: Mouse on border between panes -> routes to one pane (consistent)
     - Unit: Mouse scroll when overlay is open -> ignored (overlay captures all input)
   - Implementation: On `tea.MouseMsg` with `MouseActionPress` and `MouseButtonWheelUp`/`MouseButtonWheelDown`, call `a.layout.PaneAt(msg.X, msg.Y)` to identify target pane, convert scroll to `j`/`k` key message, route to target pane via `target.Update(scrollMsg)` WITHOUT changing focus. Only handle WheelUp/WheelDown, not clicks or drags.

3. **Update minimum terminal size** — Increase minimum from 100x24 to 120x30 with improved error message.
   - Files: `internal/app/render.go`, `cmd/root.go` (if startup check exists)
   - Tests:
     - Unit: Terminal 119x30 -> shows "needs more space" message
     - Unit: Terminal 120x29 -> shows "needs more space" message
     - Unit: Terminal 120x30 -> shows normal grid
     - Unit: Message shows actual dimensions and required dimensions
     - Unit: Message uses rounded border
   - Implementation: Constants `minTermWidth = 120`, `minTermHeight = 30`. Message format: `"Spotnik needs more space\n\nCurrent:  %d x %d\nRequired: %d x %d\n\nPlease resize your terminal and retry."` centered with rounded border.

4. **Tests** — Integration and edge case tests for mouse scroll and responsive behavior.
   - Files: `internal/app/app_test.go`
   - Tests:
     - Integration: Mouse scroll lifecycle — scroll pane -> verify content scrolled, focus unchanged
     - Integration: Mouse scroll during overlay -> ignored
     - Integration: Resize below minimum -> error message, resize above -> grid renders
     - Integration: Dynamic resize: start at 120x30, shrink to 80x20 -> error, grow to 120x30 -> grid
     - Edge: Mouse at position (0,0) -> header area, no action
     - Edge: Mouse at last row -> status bar area, no action
