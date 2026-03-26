# Feature 43 — Reusable Components

> **Feature:** Build three reusable components required by all list panes:
> a dense Table wrapper around bubble-table, an in-pane Filter using bubbles/textinput,
> and text truncation utilities. These are the building blocks every list pane depends on.

## Context

The new DESIGN.md specifies that all 7 list panes (Queue, Playlists, Albums, LikedSongs,
RecentlyPlayed, TopTracks, TopArtists) plus the Network Log pane render data in dense
aligned columns with per-column colors, scrolling, and real-time filtering.

Currently, panes render lists manually with `fmt.Sprintf` and fixed-width padding. There
are no reusable table or filter components.

This feature creates three components:

1. **Table** — Wraps `github.com/evertras/bubble-table` with Spotnik conventions
2. **Filter** — In-pane text filter using `bubbles/textinput`
3. **Truncation utilities** — Rune-aware `Truncate`, `PadRight`, `TruncateOrPad`

**Design reference:** `docs/DESIGN.md` §6 (Content Containment), §8 (In-Pane Filtering),
§9 (Dense Table Formatting)

**Depends on:** Feature 40 (theme tokens for TableHeader, TextMuted, etc.)

---

## Design Diagram

```
Dense Table (DESIGN.md §9):

 #   Track                    Artist              Duration
 1   Lil Boo Thang            Paul Russell        3:12
 2   Street Fighter           Kamasi Washington   5:44
▶3   BIRDS OF A FEATHER       Billie Eilish       3:30    ← currently playing
 4   Peaches                  Justin Bieber       3:18    ← selected (highlight bg)

Column Colors:
  # index    → TextMuted()
  Track name → TextPrimary()
  Artist     → TextSecondary()
  Duration   → TextMuted()
  Selected   → SelectedBg() + SelectedFg() (overrides column colors)
  Playing    → ▶ in PlayingIndicator() color

Filter (DESIGN.md §8):

╭─ ¹Queue ────────── filtering: "rock" ─── ᐅEsc close ╮
│  > rock█                                             │  ← filter input bar
│  ──────────────────────────────────────────────────── │
│  3   Rocket Man              Elton John     4:52     │  ← filtered results
│  7   Rock and Roll           Led Zeppelin   3:40     │
╰──────────────────────────────────────────────────────╯

Truncation:
  "Kamasi Washington" at width 12 → "Kamasi Wash…"
  "OK"               at width 12 → "OK          " (padded)
```

---

## Task 1: Add bubble-table dependency

**Problem:** `github.com/evertras/bubble-table` is not in go.mod.

**Fix:**

1. Run `go get github.com/evertras/bubble-table/table`
2. Run `go mod tidy`
3. Verify it compiles

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

**Tests:**
- Build: `go build ./...` succeeds

**Commit:** `chore(deps): add bubble-table dependency`

---

## Task 2: Create truncation utilities

**Problem:** No rune-aware text truncation/padding functions exist. Current panes use
`len()` or `utf8.RuneCountInString()` which mishandle wide characters.

**Fix:**

Create `internal/ui/layout/truncate.go`:

```go
package layout

import "github.com/charmbracelet/lipgloss"

// Truncate truncates s to maxWidth terminal columns.
// If s is wider than maxWidth, it is truncated and "…" is appended.
// Uses lipgloss.Width() for accurate rendered-width measurement.
func Truncate(s string, maxWidth int) string

// PadRight pads s with spaces to exactly width terminal columns.
// If s is already wider, it is returned unchanged (use Truncate first).
func PadRight(s string, width int) string

// TruncateOrPad truncates or pads s to exactly width terminal columns.
// Equivalent to PadRight(Truncate(s, width), width).
func TruncateOrPad(s string, width int) string
```

**Implementation notes:**
- Use `lipgloss.Width()` for measurement, NOT `len()` or `utf8.RuneCountInString()`
- `lipgloss.Width()` correctly handles CJK (2 cells), emoji, combining marks, ANSI escapes
- Truncation must iterate runes and sum widths to find the cut point
- `…` (U+2026) is 1 cell wide

**Files:**
- Create: `internal/ui/layout/truncate.go`

**Tests:**
- Unit: ASCII string shorter than maxWidth → unchanged
- Unit: ASCII string equal to maxWidth → unchanged
- Unit: ASCII string longer than maxWidth → truncated with `…`
- Unit: Unicode string with CJK characters → correct width measurement
- Unit: String with ANSI escape codes → escapes ignored in width measurement
- Unit: Empty string → empty result
- Unit: maxWidth=0 → empty result
- Unit: maxWidth=1 → `…` only
- Unit: `PadRight` shorter string → padded with spaces to exact width
- Unit: `PadRight` exact-width string → unchanged
- Unit: `TruncateOrPad` combines both — long strings truncated, short strings padded

**Commit:** `feat(layout): rune-aware truncation utilities`

---

## Task 3: Create Table component wrapper

**Problem:** No reusable table component exists for dense pane rendering.

**Fix:**

Create `internal/ui/components/table.go`:

```go
package components

import (
    "github.com/charmbracelet/lipgloss"
    btable "github.com/evertras/bubble-table/table"
    "github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// ColumnDef defines a table column with its display properties.
type ColumnDef struct {
    Key       string         // Data key for bubble-table
    Header    string         // Display header text
    FlexFactor int           // Relative width (flex column)
    Color     lipgloss.Color // Text color for this column
}

// TableConfig holds configuration for creating a Table.
type TableConfig struct {
    Columns       []ColumnDef
    Theme         theme.Theme
    PlayingIndex  int  // Row index of currently playing track (-1 if none)
    ShowHeader    bool // Whether to show column headers
}

// Table wraps bubble-table with Spotnik styling conventions.
type Table struct {
    inner  btable.Model
    config TableConfig
    width  int
    height int
}

// NewTable creates a Table with the given configuration.
func NewTable(cfg TableConfig) Table

// SetSize updates the table dimensions. Recalculates column widths.
func (t *Table) SetSize(width, height int)

// SetRows updates the table data. Each row is a map[string]string keyed by ColumnDef.Key.
func (t *Table) SetRows(rows []map[string]string)

// SetPlayingIndex marks which row shows the ▶ indicator.
func (t *Table) SetPlayingIndex(index int)

// SelectedIndex returns the currently highlighted row index.
func (t *Table) SelectedIndex() int

// Update forwards messages to the inner bubble-table model.
func (t *Table) Update(msg tea.Msg) tea.Cmd

// View renders the table.
func (t *Table) View() string
```

**Implementation details:**

1. **Column setup:** Convert `[]ColumnDef` to `[]btable.Column` using `btable.NewFlexColumn(key, header, flexFactor)`
2. **Borderless mode:** Use `btable.Border(customEmptyBox)` where `customEmptyBox` has space characters for all border positions, effectively hiding borders. The outer pane border (Feature 42) handles the visible border.
3. **Row styling via `WithRowStyleFunc`:**
   - Selected/highlighted row: `SelectedBg()` + `SelectedFg()`
   - Currently playing row: `▶` in `PlayingIndicator()` color replaces the `#` index
   - Per-column colors from `ColumnDef.Color`
4. **Header styling:** Column headers in `TableHeader()` color
5. **Width:** `WithTargetWidth(width)` for flex column calculation
6. **Height:** `WithPageSize(height - headerLines)` for internal scrolling

**Files:**
- Create: `internal/ui/components/table.go`

**Tests:**
- Unit: `NewTable` with 4 columns creates correct bubble-table columns
- Unit: `SetSize(80, 20)` → columns scale proportionally
- Unit: `SetRows` with 5 rows → table shows 5 rows
- Unit: `SelectedIndex()` returns 0 initially
- Unit: Keyboard navigation (j/k or up/down) changes selected index
- Unit: `SetPlayingIndex(2)` → row 2 shows ▶ indicator
- Unit: Column headers rendered in `TableHeader()` color
- Unit: Per-column colors applied correctly
- Unit: Selected row overrides column colors with selection colors
- Unit: Empty rows → renders headers only (or empty state)
- Unit: Width recalculation after `SetSize` → columns resize correctly

**Commit:** `feat(ui): dense Table component wrapping bubble-table`

---

## Task 4: Create Filter component

**Problem:** No reusable in-pane filter component exists.

**Fix:**

Create `internal/ui/components/filter.go`:

```go
package components

import (
    "strings"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/bubbles/textinput"
    "github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// Filter provides in-pane text filtering using bubbles/textinput.
type Filter struct {
    input  textinput.Model
    active bool
    query  string
    theme  theme.Theme
}

// NewFilter creates a Filter with the given theme.
func NewFilter(t theme.Theme) *Filter

// Toggle activates or deactivates the filter.
// When activated, focuses the text input. When deactivated, clears the query.
func (f *Filter) Toggle()

// IsActive returns whether the filter is currently active.
func (f *Filter) IsActive() bool

// Query returns the current filter text.
func (f *Filter) Query() string

// Matches returns true if text contains the filter query (case-insensitive substring).
// Always returns true if filter is inactive.
func (f *Filter) Matches(text string) bool

// MatchesAny returns true if any of the provided strings match the filter.
func (f *Filter) MatchesAny(texts ...string) bool

// Update handles input events when filter is active.
// Returns a tea.Cmd if the textinput needs to blink cursor, etc.
func (f *Filter) Update(msg tea.Msg) tea.Cmd

// View renders the filter input bar at the given width.
// Returns empty string if filter is not active.
func (f *Filter) View(width int) string

// BorderLabel returns the text to show in the pane border when filter is active.
// e.g., `filtering: "rock"`. Returns "" if inactive.
func (f *Filter) BorderLabel() string
```

**Implementation details:**

1. **textinput setup:** `Placeholder: "type to filter..."`, `CharLimit: 50`, `Width: paneWidth - 4`
2. **Styling:** Input text in `TextPrimary()`, background in `SurfaceAlt()`
3. **Activation:** `Toggle()` flips `active`, calls `input.Focus()` or `input.Blur()`
4. **Key handling in Update:**
   - `Esc` → deactivate filter, clear query
   - `Enter` → deactivate filter, keep query (select first filtered result)
   - All other keys → forward to textinput
5. **Matching:** `strings.Contains(strings.ToLower(text), strings.ToLower(f.query))`
6. **View:** Renders `> {query}█` bar with styled background, only when active

**Files:**
- Create: `internal/ui/components/filter.go`

**Tests:**
- Unit: `NewFilter()` starts inactive
- Unit: `Toggle()` activates, second `Toggle()` deactivates
- Unit: `Query()` returns current text
- Unit: `Matches("Blinding Lights")` with query "blind" → true
- Unit: `Matches("Blinding Lights")` with query "xyz" → false
- Unit: `Matches("Blinding Lights")` with inactive filter → true (always matches)
- Unit: Matching is case-insensitive
- Unit: `MatchesAny("Track", "Artist")` with query matching artist → true
- Unit: `BorderLabel()` returns `filtering: "rock"` when active with query "rock"
- Unit: `BorderLabel()` returns "" when inactive
- Unit: `View()` returns empty string when inactive
- Unit: `View(40)` returns styled input bar when active
- Unit: Esc key deactivates filter
- Unit: Enter key deactivates filter but preserves query

**Commit:** `feat(ui): in-pane Filter component with textinput`

---

## Task 5: Component integration tests

**Problem:** Need to verify Table and Filter work together as they will in panes.

**Fix:**

**Files:**
- Create: `internal/ui/components/table_test.go`
- Create: `internal/ui/components/filter_test.go`
- Create: `internal/ui/layout/truncate_test.go`

**Tests:**
- Integration: Table with Filter — set rows, activate filter, filter rows externally
  using `Filter.Matches()`, update table with filtered rows
- Integration: Table resize cycle — set rows, resize smaller, verify no overflow
- Integration: Filter activation → pane border shows filter label, deactivation → border shows actions
- Integration: Truncate used on table cell values → no overflow past column width
- Integration: Table with 0 rows → renders cleanly (no panic, shows headers or empty state)
- Integration: Filter with empty query → matches everything

**Commit:** `test(ui): table, filter, and truncation integration tests`

---

## Acceptance Criteria

- [ ] `bubble-table` dependency added to go.mod
- [ ] `Truncate()`, `PadRight()`, `TruncateOrPad()` in `layout/truncate.go` use `lipgloss.Width()` for measurement
- [ ] `Table` wraps `bubble-table` with: flex columns, per-column colors, selected row highlighting, playing indicator, borderless mode
- [ ] `Filter` wraps `textinput` with: toggle, case-insensitive matching, border label, Esc/Enter handling
- [ ] `Table.SetSize()` recalculates column widths
- [ ] `Filter.Matches()` returns true when filter is inactive
- [ ] All 3 components have independent tests
- [ ] No imports from `app/`, `api/`, `state/` in these components
- [ ] `make ci` passes

---

## Notes

- **Column width proportions** (DESIGN.md §9) are defined per-pane, not in the Table component.
  Each pane passes its own `[]ColumnDef` with appropriate flex factors. The Table component is generic.
- **Filtered data flow:** The Filter component does NOT directly filter the Table's rows. Instead,
  the pane's `Update()` method reads `filter.Query()`, filters its data slice, and calls
  `table.SetRows(filteredRows)`. This keeps data ownership in the pane.
- **bubble-table's built-in filtering** exists but we use our own Filter component for consistency
  across panes and to show the filter state in the btop-style border (via `BorderLabel()`).
- **Scroll indicators** (`▲`/`▼`) are handled by bubble-table internally. If its indicators
  don't match the design, we can add custom indicators in the pane's `View()` method.
- The `Table` component should set `bubble-table`'s `Focused(true/false)` based on pane focus
  to enable/disable keyboard navigation.
