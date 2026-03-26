# Feature 41 — Layout Infrastructure

> **Feature:** Create the `internal/ui/layout/` package with the LayoutManager,
> Pane interface, grid system, space distribution algorithm, preset definitions,
> page toggling, and pane toggling. This is the skeleton that all panes hang on.

## Context

The current UI uses a hardcoded 3-column layout (`libraryView | playerView | queueView`)
assembled with `lipgloss.JoinHorizontal()` in `render.go`. There is no concept of
pages, presets, pane toggling, or responsive grid reflow.

The new DESIGN.md specifies a btop-inspired responsive grid system where:
- 10 panes are organized across 2 pages (Page A: Music, Page B: Nerd Status)
- Page A has 4 presets (Full Dashboard, Listening, Library, Discovery)
- Keys `1`-`8` toggle individual pane visibility
- Hidden panes redistribute space to visible siblings
- The grid recomputes on terminal resize and preset/toggle changes

This feature builds the layout engine only — no pane implementations, no rendering.
Subsequent features (F42-F49) build on this foundation.

**Design reference:** `docs/DESIGN.md` §2 (Pane Definitions), §3 (Layout Grid System),
§4 (Pages, Pane Toggling, and Preset Layouts), §16 (Focus & Navigation), §22 (Architecture — LayoutManager)

**Depends on:** Nothing — can run in parallel with F40.

---

## Design Diagram

```
Grid System
═══════════

Grid = []Row
Row  = {HeightWeight int, Cells []Cell}
Cell = {PaneID, WidthWeight int}

Space Distribution:
  1. Filter hidden cells → remove from row
  2. Filter empty rows → remove entirely
  3. Distribute height among active rows by HeightWeight
  4. Distribute width per row among visible cells by WidthWeight
  5. Last cell/row absorbs rounding remainder

Reserved Space:
  Total height = terminal rows
    - Header:     1 line
    - Status bar: 1 line
    - Content:    terminal rows - 2

Page A — Preset 0 (Full Dashboard):
╭─ NowPlaying (weight 1) ────────────────────────────────╮  Row 1 (weight 2)
╰─────────────────────────────────────────────────────────╯
╭─ Playlists ──╮╭─ Albums ─────╮╭─ LikedSongs ───────────╮  Row 2 (weight 3)
╰──────────────╯╰──────────────╯╰─────────────────────────╯
╭─ Queue ──╮╭─ Recent ──╮╭─ TopTracks ╮╭─ TopArtists ────╮  Row 3 (weight 3)
╰──────────╯╰───────────╯╰────────────╯╰─────────────────╯

Page B:
╭─ NowPlaying (weight 1) ────────────────────────────────╮  Row 1 (weight 1)
╰─────────────────────────────────────────────────────────╯
╭─ RequestFlow (weight 1) ───────────────────────────────╮  Row 2 (weight 3)
╰─────────────────────────────────────────────────────────╯
╭─ NetworkLog (weight 1) ────────────────────────────────╮  Row 3 (weight 2)
╰─────────────────────────────────────────────────────────╯
```

---

## Task 1: Define PaneID, PageID enums and Rect struct

**Problem:** No type-safe identifiers for panes and pages exist.

**Fix:**

Create `internal/ui/layout/pane.go`:

```go
package layout

import tea "github.com/charmbracelet/bubbletea"

// PaneID uniquely identifies a pane slot in the grid.
type PaneID int

const (
    PaneNowPlaying PaneID = iota
    PaneQueue
    PanePlaylists
    PaneAlbums
    PaneLikedSongs
    PaneRecentlyPlayed
    PaneTopTracks
    PaneTopArtists
    PaneRequestFlow
    PaneNetworkLog
)

// PageID identifies a page (group of panes).
type PageID int

const (
    PageA PageID = iota // Music
    PageB               // Nerd Status
)

// Rect describes a pane's position and size in terminal cells.
type Rect struct {
    X, Y          int // Top-left corner (relative to content area)
    Width, Height int // Dimensions including borders
}

// ContentWidth returns the usable width inside borders.
func (r Rect) ContentWidth() int {
    if r.Width < 2 { return 0 }
    return r.Width - 2
}

// ContentHeight returns the usable height inside borders.
func (r Rect) ContentHeight() int {
    if r.Height < 2 { return 0 }
    return r.Height - 2
}

// Action describes a pane-specific shortcut shown in the border.
type Action struct {
    Key   string // e.g., "f"
    Label string // e.g., "filter"
}

// Pane is the interface every grid pane must implement.
type Pane interface {
    tea.Model
    SetSize(width, height int)
    SetFocused(focused bool)
    IsFocused() bool
    ID() PaneID
    Title() string
    ToggleKey() int       // 1-8 for Page A panes, 0 if not toggleable
    Actions() []Action    // Pane-specific shortcuts for border display
}
```

**Files:**
- Create: `internal/ui/layout/pane.go`

**Tests:**
- Unit: `Rect.ContentWidth()` and `ContentHeight()` return correct values
- Unit: `Rect.ContentWidth()` returns 0 for width < 2
- Unit: PaneID constants have expected iota values (0-9)
- Unit: PageID constants (PageA=0, PageB=1)

**Commit:** `feat(layout): define PaneID, PageID, Rect, Pane interface`

---

## Task 2: Define grid model and preset data structures

**Problem:** No data structures represent the row-based grid layout.

**Fix:**

Create `internal/ui/layout/presets.go`:

```go
package layout

// Cell represents a pane slot in a row.
type Cell struct {
    PaneID      PaneID
    WidthWeight int
}

// Row represents a horizontal strip of cells in the grid.
type Row struct {
    HeightWeight int
    Cells        []Cell
}

// Preset is a named grid configuration — a bitmask of visible panes + grid layout.
type Preset struct {
    Name    string
    Visible map[PaneID]bool
    Grid    []Row
}

// Page A presets (DESIGN.md §4)

var PresetDashboard = Preset{
    Name: "Full Dashboard",
    Visible: map[PaneID]bool{
        PaneNowPlaying: true, PaneQueue: true, PanePlaylists: true,
        PaneAlbums: true, PaneLikedSongs: true, PaneRecentlyPlayed: true,
        PaneTopTracks: true, PaneTopArtists: true,
    },
    Grid: []Row{
        {HeightWeight: 2, Cells: []Cell{{PaneNowPlaying, 1}}},
        {HeightWeight: 3, Cells: []Cell{{PanePlaylists, 1}, {PaneAlbums, 1}, {PaneLikedSongs, 1}}},
        {HeightWeight: 3, Cells: []Cell{{PaneQueue, 1}, {PaneRecentlyPlayed, 1}, {PaneTopTracks, 1}, {PaneTopArtists, 1}}},
    },
}

var PresetListening = Preset{
    Name: "Listening",
    Visible: map[PaneID]bool{
        PaneNowPlaying: true, PaneQueue: true, PaneRecentlyPlayed: true,
    },
    Grid: []Row{
        {HeightWeight: 3, Cells: []Cell{{PaneNowPlaying, 1}}},
        {HeightWeight: 2, Cells: []Cell{{PaneQueue, 1}, {PaneRecentlyPlayed, 1}}},
    },
}

var PresetLibrary = Preset{
    Name: "Library",
    Visible: map[PaneID]bool{
        PaneNowPlaying: true, PanePlaylists: true, PaneAlbums: true, PaneLikedSongs: true,
    },
    Grid: []Row{
        {HeightWeight: 1, Cells: []Cell{{PaneNowPlaying, 1}}},
        {HeightWeight: 4, Cells: []Cell{{PanePlaylists, 1}, {PaneAlbums, 1}, {PaneLikedSongs, 1}}},
    },
}

var PresetDiscovery = Preset{
    Name: "Discovery",
    Visible: map[PaneID]bool{
        PaneNowPlaying: true, PaneTopTracks: true, PaneTopArtists: true, PaneRecentlyPlayed: true,
    },
    Grid: []Row{
        {HeightWeight: 1, Cells: []Cell{{PaneNowPlaying, 1}}},
        {HeightWeight: 2, Cells: []Cell{{PaneTopTracks, 1}, {PaneTopArtists, 1}}},
        {HeightWeight: 2, Cells: []Cell{{PaneRecentlyPlayed, 1}}},
    },
}

// Page B preset

var PresetNerdStatus = Preset{
    Name: "Nerd Status",
    Visible: map[PaneID]bool{
        PaneNowPlaying: true, PaneRequestFlow: true, PaneNetworkLog: true,
    },
    Grid: []Row{
        {HeightWeight: 1, Cells: []Cell{{PaneNowPlaying, 1}}},
        {HeightWeight: 3, Cells: []Cell{{PaneRequestFlow, 1}}},
        {HeightWeight: 2, Cells: []Cell{{PaneNetworkLog, 1}}},
    },
}

// PageAPresets is the ordered list of presets for Page A.
var PageAPresets = []Preset{PresetDashboard, PresetListening, PresetLibrary, PresetDiscovery}

// PageBPresets is the ordered list of presets for Page B.
var PageBPresets = []Preset{PresetNerdStatus}
```

**Files:**
- Create: `internal/ui/layout/presets.go`

**Tests:**
- Unit: Each preset's Visible map matches its Grid cells (no pane in Grid that isn't in Visible)
- Unit: PresetDashboard has 8 visible panes, 3 rows
- Unit: PresetListening has 3 visible panes, 2 rows
- Unit: PresetLibrary has 4 visible panes, 2 rows
- Unit: PresetDiscovery has 4 visible panes, 3 rows
- Unit: PresetNerdStatus has 3 visible panes, 3 rows
- Unit: PageAPresets has 4 entries, PageBPresets has 1

**Commit:** `feat(layout): define grid model and preset configurations`

---

## Task 3: Implement LayoutManager with space distribution

**Problem:** No component computes pane positions from grid definitions.

**Fix:**

Create `internal/ui/layout/layout.go`:

```go
package layout

// Manager computes pane positions from a grid definition and terminal size.
type Manager struct {
    activePage   PageID
    presets      map[PageID][]Preset
    activePreset map[PageID]int       // index into presets slice
    hidden       map[PaneID]bool      // manual toggles (Page A only)
    rects        map[PaneID]Rect      // computed positions
    focusOrder   []PaneID             // visible panes in layout order
    focusIndex   int
    width        int
    height       int
    headerHeight int                  // 1
    statusHeight int                  // 1
}

// NewManager creates a Manager with default presets and Page A active.
func NewManager() *Manager

// Resize updates terminal dimensions and recomputes all pane rects.
func (m *Manager) Resize(width, height int)

// recompute recalculates all Rects from the active preset + hidden state.
// Called after Resize, SetPreset, CyclePreset, TogglePage, TogglePane.
func (m *Manager) recompute()

// PaneRect returns the computed Rect for a pane. Returns zero Rect if hidden.
func (m *Manager) PaneRect(id PaneID) Rect

// VisiblePanes returns all visible PaneIDs in layout order (top-left to bottom-right).
func (m *Manager) VisiblePanes() []PaneID

// ActivePage returns the current page.
func (m *Manager) ActivePage() PageID

// ActivePresetIndex returns the current preset index for the active page.
func (m *Manager) ActivePresetIndex() int

// ActivePresetName returns the name of the current preset.
func (m *Manager) ActivePresetName() string
```

**Space distribution algorithm** (in `recompute()`):

1. Get the current preset's Grid definition
2. Build `activeGrid []Row` by filtering:
   - Remove cells whose PaneID is in `m.hidden`
   - Remove rows where all cells were removed
3. Compute content area: `contentHeight = m.height - m.headerHeight - m.statusHeight`
4. Distribute height: sum all active row HeightWeights, divide `contentHeight` proportionally.
   Last row absorbs rounding remainder.
5. Per row, distribute width: sum cell WidthWeights, divide row width proportionally.
   Last cell absorbs remainder.
6. Compute `Rect` for each visible pane: `{X, Y, Width, Height}`
7. Build `focusOrder` from visible panes in grid order (row by row, left to right)

**Files:**
- Create: `internal/ui/layout/layout.go`

**Tests:**
- Unit: `NewManager()` starts on PageA, preset 0 (Dashboard), no hidden panes
- Unit: `Resize(120, 30)` computes rects for all 8 Page A panes
- Unit: Space distribution — all rects sum to content area (no gaps, no overflow)
- Unit: Height weight 2:3:3 distributes correctly for 28 content rows (header+status=2)
- Unit: Width weights 1:1:1 in a row divide evenly, last absorbs remainder
- Unit: Single-cell row gets full width
- Unit: `VisiblePanes()` returns 8 panes for Dashboard preset
- Unit: `PaneRect()` returns zero Rect for hidden pane
- Unit: Rects don't overlap (no pane covers another's area)

**Commit:** `feat(layout): implement Manager with space distribution algorithm`

---

## Task 4: Implement page toggle, preset cycling, and pane toggling

**Problem:** No mechanisms to switch pages, cycle presets, or toggle panes.

**Fix:**

Add to `Manager`:

```go
// TogglePage switches between PageA and PageB.
// Resets hidden map and recomputes layout.
func (m *Manager) TogglePage()

// CyclePreset advances to the next preset on the active page.
// Wraps to first preset after the last. Resets manual toggles.
func (m *Manager) CyclePreset()

// SetPreset sets a specific preset index. Resets manual toggles.
func (m *Manager) SetPreset(index int)

// TogglePane toggles visibility of a pane (Page A only, keys 1-8).
// Does nothing if the pane is not part of the current page.
func (m *Manager) TogglePane(id PaneID)

// IsPaneVisible returns whether a pane is currently visible.
func (m *Manager) IsPaneVisible(id PaneID) bool
```

**Behavior (DESIGN.md §4):**
- `TogglePage()`: switches activePage, clears hidden map, calls `recompute()`
- `CyclePreset()`: increments `activePreset[page]`, wraps to 0, clears hidden map, calls `recompute()`
- `TogglePane(id)`: flips `hidden[id]`, calls `recompute()`. Only works for Page A panes (1-8). If toggling would hide ALL panes in the grid, the toggle is rejected (at least 1 pane must remain visible).
- Preset switch resets all manual toggles

**Files:**
- Modify: `internal/ui/layout/layout.go`

**Tests:**
- Unit: `TogglePage()` switches PageA↔PageB, returns correct `ActivePage()`
- Unit: `TogglePage()` clears hidden state
- Unit: `CyclePreset()` cycles 0→1→2→3→0 on Page A
- Unit: `CyclePreset()` resets manual toggles
- Unit: `TogglePane(PanePlaylists)` hides playlists, siblings expand
- Unit: `TogglePane()` again restores playlists to original position
- Unit: Hiding all panes in a row collapses the row, other rows expand
- Unit: Cannot hide the last visible pane (toggle rejected)
- Unit: `TogglePane()` does nothing on Page B (no toggle keys)
- Unit: `IsPaneVisible()` reflects toggle state

**Commit:** `feat(layout): page toggle, preset cycling, pane toggle`

---

## Task 5: Implement focus rotation

**Problem:** No mechanism to rotate keyboard focus among visible panes.

**Fix:**

Add to `Manager`:

```go
// RotateFocus moves focus to the next (forward=true) or previous visible pane.
// Wraps around. Uses focusOrder built during recompute().
func (m *Manager) RotateFocus(forward bool)

// FocusedPane returns the PaneID that currently has keyboard focus.
func (m *Manager) FocusedPane() PaneID

// SetFocus sets focus to a specific pane. No-op if pane is not visible.
func (m *Manager) SetFocus(id PaneID)
```

**Behavior (DESIGN.md §16):**
- `focusOrder` is built in `recompute()`: visible panes in grid order (row by row, left to right)
- `RotateFocus(true)`: `focusIndex = (focusIndex + 1) % len(focusOrder)`
- `RotateFocus(false)`: `focusIndex = (focusIndex - 1 + len(focusOrder)) % len(focusOrder)`
- When focused pane becomes hidden (via toggle/preset), focus moves to the first visible pane
- `SetFocus(id)` finds the pane in focusOrder and sets focusIndex

**Files:**
- Modify: `internal/ui/layout/layout.go`

**Tests:**
- Unit: `RotateFocus(true)` cycles through all visible panes in order
- Unit: `RotateFocus(false)` cycles in reverse
- Unit: Focus wraps from last to first and vice versa
- Unit: After `TogglePane()` hides focused pane, focus moves to first visible
- Unit: After `CyclePreset()`, focus resets to first visible pane
- Unit: `SetFocus(id)` changes focus to specified pane
- Unit: `SetFocus(id)` no-op for hidden pane
- Unit: `FocusedPane()` returns the correct PaneID

**Commit:** `feat(layout): focus rotation among visible panes`

---

## Task 6: Add PaneAt hit-test for mouse support

**Problem:** Mouse scroll needs to identify which pane is under the cursor.

**Fix:**

Add to `Manager`:

```go
// PaneAt returns the PaneID at terminal coordinates (x, y).
// Returns -1 (invalid PaneID) if no pane is at that position.
// Coordinates are 0-based from top-left of terminal.
// Accounts for header offset.
func (m *Manager) PaneAt(x, y int) PaneID
```

**Implementation:** Iterate `rects` map, check if `(x, y)` falls within any Rect's bounds (accounting for header height offset).

**Files:**
- Modify: `internal/ui/layout/layout.go`

**Tests:**
- Unit: Click in center of NowPlaying rect returns `PaneNowPlaying`
- Unit: Click in header area returns -1 (no pane)
- Unit: Click in status bar area returns -1
- Unit: Click on border between two panes returns the left/top pane (or either — just consistent)
- Unit: Click outside all panes returns -1

**Commit:** `feat(layout): PaneAt hit-test for mouse scroll routing`

---

## Task 7: Comprehensive layout tests

**Problem:** Need integration-level tests verifying the full layout lifecycle.

**Fix:**

Add thorough table-driven tests in `internal/ui/layout/layout_test.go`:

**Files:**
- Create: `internal/ui/layout/layout_test.go`
- Create: `internal/ui/layout/pane_test.go`
- Create: `internal/ui/layout/presets_test.go`

**Tests:**
- Integration: Full lifecycle: NewManager → Resize → CyclePreset → TogglePane → TogglePage → verify rects
- Integration: Resize from 120×30 to 80×24 → all rects shrink proportionally
- Integration: Dashboard preset with terminal 120×30 → verify each pane gets expected rect
- Integration: Hide 3 panes in row 3 → row 3 collapses, rows 1+2 expand
- Integration: Hide all panes in row 2 → row disappears, rows 1+3 split remaining height
- Integration: Toggle all 8 panes off one by one → last toggle rejected, 1 pane remains
- Integration: Preset cycle full loop (0→1→2→3→0) → correct visible panes each time
- Integration: Page B has 3 panes, no toggle keys work
- Integration: Focus rotation after hiding panes → skips hidden, wraps correctly
- Edge: Zero-size terminal → graceful handling (no panics, empty rects)
- Edge: Very small terminal (1×1) → no panics

**Commit:** `test(layout): comprehensive layout manager tests`

---

## Acceptance Criteria

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

---

## Notes

- The `Pane` interface is defined here but no panes implement it yet. Features 45-48 add implementations.
- The `Manager` does not render anything — it only computes `Rect` positions. Feature 42 (border renderer) and Feature 49 (app migration) handle rendering.
- `focusOrder` is deterministic: row-by-row, left-to-right within each row. This matches btop's Tab behavior.
- Page B panes (`RequestFlow`, `NetworkLog`) are not toggleable via number keys (DESIGN.md §2). The `TogglePane` method enforces this by checking if the pane belongs to Page A.
- NowPlaying appears on both pages (it's in every preset). The same pane instance is shared — the Manager just includes it in both page layouts.
