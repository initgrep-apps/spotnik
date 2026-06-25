package components

import (
	"maps"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	btable "github.com/evertras/bubble-table/table"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// emptyBorder is a bubble-table Border with space characters for all positions,
// effectively hiding the table's built-in border. The outer pane border
// (rendered by internal/ui/layout.RenderPaneBorder) handles the visible border.
//
// Using " " (space) instead of "" (empty) for all border fields ensures consistent
// cell heights across all header and row styles. With empty chars, bubble-table's
// generateMultiStyles creates first-column styles without top/bottom borders
// (height=1) while other columns get .BorderTop(true)/.BorderBottom(true)
// (height=3). This height mismatch causes JoinHorizontal to stagger columns
// onto separate lines at the full table width. Space chars auto-enable all
// borders via BorderStyle, giving every cell identical height.
var emptyBorder = btable.Border{
	Top:            " ",
	Left:           " ",
	Right:          " ",
	Bottom:         " ",
	TopRight:       " ",
	TopLeft:        " ",
	BottomRight:    " ",
	BottomLeft:     " ",
	TopJunction:    " ",
	LeftJunction:   " ",
	RightJunction:  " ",
	BottomJunction: " ",
	InnerJunction:  " ",
	InnerDivider:   " ",
}

// ColumnDef defines a table column with its display properties.
type ColumnDef struct {
	// Key is the data key used as a map lookup in each row.
	Key string
	// Header is the text shown in the column header row.
	Header string
	// FlexFactor controls the column's relative share of available width.
	// A column with FlexFactor 2 gets twice the space of one with FlexFactor 1.
	FlexFactor int
	// Priority determines column visibility at different pane widths.
	// 0/1 = always visible, 2 = visible at >=40 cols, 3 = visible at >=60 cols.
	// Zero-value (0) is treated as Priority 1 for backward compatibility.
	Priority int
	// Color is the lipgloss foreground color applied to data cells in this column.
	Color lipgloss.Color
}

// TableConfig holds configuration for creating a Table.
type TableConfig struct {
	// Columns defines the column layout and per-column colors.
	Columns []ColumnDef
	// Theme provides color tokens for header and selection.
	Theme theme.Theme
	// ShowHeader controls whether the column header row is rendered.
	ShowHeader bool
}

// Table wraps bubble-table with Spotnik styling conventions: borderless mode,
// per-column colors and selected row highlighting.
type Table struct {
	inner    btable.Model
	config   TableConfig
	rows     []map[string]string
	richRows []map[string]any // set via SetRichRows; nil when SetRows was used last
	width    int
	height   int
}

// NewTable creates a Table with the given configuration.
// Call SetSize before calling View to set dimensions.
func NewTable(cfg TableConfig) *Table {
	t := &Table{
		config: cfg,
	}
	t.rebuild()
	return t
}

// filterColumnsByPriority returns the column subset that should be visible at the
// given pane width. Priority 0/1 are always visible; Priority 2 requires >=40 cols;
// Priority 3 requires >=60 cols. At least one column is always returned.
func filterColumnsByPriority(cols []ColumnDef, width int) []ColumnDef {
	filtered := make([]ColumnDef, 0, len(cols))
	for _, c := range cols {
		switch c.Priority {
		case 0, 1:
			filtered = append(filtered, c)
		case 2:
			if width >= 40 {
				filtered = append(filtered, c)
			}
		case 3:
			if width >= 60 {
				filtered = append(filtered, c)
			}
		default:
			filtered = append(filtered, c)
		}
	}
	if len(filtered) == 0 && len(cols) > 0 {
		filtered = append(filtered, cols[0])
	}
	return filtered
}

// rebuild reconstructs the inner bubble-table Model with current config and size.
// Called after config changes or size changes.
func (t *Table) rebuild() {
	th := t.config.Theme

	activeCols := filterColumnsByPriority(t.config.Columns, t.width)

	// Build bubble-table flex columns from the active (priority-filtered) columns.
	btCols := make([]btable.Column, len(activeCols))
	for i, col := range activeCols {
		btCols[i] = btable.NewFlexColumn(col.Key, col.Header, col.FlexFactor).
			WithStyle(lipgloss.NewStyle().Foreground(col.Color).Align(lipgloss.Left))
	}

	inner := btable.New(btCols).
		Border(emptyBorder).
		HeaderStyle(lipgloss.NewStyle().Foreground(th.TableHeader()).Bold(false).Align(lipgloss.Left)).
		WithTargetWidth(t.width)

	if !t.config.ShowHeader {
		inner = inner.WithHeaderVisibility(false)
	}

	// Use WithRowStyleFunc to drive selection and playing indicator styling.
	// NOTE: HighlightStyle must NOT be set alongside WithRowStyleFunc per bubble-table docs.
	inner = inner.WithRowStyleFunc(func(in btable.RowStyleFuncInput) lipgloss.Style {
		if in.IsHighlighted {
			return lipgloss.NewStyle().
				Background(th.SelectedBg()).
				Foreground(th.SelectedFg())
		}
		// Playing row gets a subtle indicator via a dedicated column but no special row bg.
		return lipgloss.NewStyle()
	})

	// Overhead: top border(1) + header(1) + header-data divider(1) +
	//   data-bottom divider(1) + pagination footer(1) + bottom border(1) = 6.
	// pageSize = height - 6 ensures footer always fits (verified: h=10→PS=4→10 lines).
	pageSize := t.height - 6 // header visible
	if !t.config.ShowHeader {
		pageSize = t.height - 4 // no header
	}
	if pageSize < 1 {
		pageSize = 1
	}
	inner = inner.WithPageSize(pageSize)

	t.inner = inner
	t.applyRows()
}

// applyRows converts the stored row data into bubble-table Row values and applies
// them to the inner model. When richRows is set (via SetRichRows), each cell value
// may be a plain string or a btable.StyledCell — both are passed directly to
// bubble-table. When richRows is nil, the plain []map[string]string path is used.
func (t *Table) applyRows() {
	// Rich-rows path: accept string or btable.StyledCell per cell.
	if t.richRows != nil {
		if len(t.richRows) == 0 {
			t.inner = t.inner.WithRows(nil)
			return
		}
		btRows := make([]btable.Row, len(t.richRows))
		for i, rowData := range t.richRows {
			data := make(btable.RowData, len(rowData))
			maps.Copy(data, rowData)
			btRows[i] = btable.NewRow(data)
		}
		t.inner = t.inner.WithRows(btRows)
		return
	}

	// Plain string-rows path (existing behaviour, unmodified).
	if len(t.rows) == 0 {
		t.inner = t.inner.WithRows(nil)
		return
	}

	btRows := make([]btable.Row, len(t.rows))

	for i, rowData := range t.rows {
		data := btable.RowData{}
		for k, v := range rowData {
			data[k] = v
		}

		btRows[i] = btable.NewRow(data)
	}

	t.inner = t.inner.WithRows(btRows)
}

// SetSize updates the table dimensions. Recalculates column widths and page size.
// If the width crosses a column-priority threshold (40 or 60 cols), the table is
// rebuilt so the visible column set adapts.
func (t *Table) SetSize(width, height int) {
	oldWidth := t.width
	t.width = width
	t.height = height

	if crossesThreshold(oldWidth, width) {
		wasFocused := (&t.inner).GetFocused()
		t.rebuild()
		t.inner = t.inner.Focused(wasFocused)
		return
	}

	t.inner = t.inner.WithTargetWidth(width)

	pageSize := height - 6
	if !t.config.ShowHeader {
		pageSize = height - 4
	}
	if pageSize < 1 {
		pageSize = 1
	}
	t.inner = t.inner.WithPageSize(pageSize)
}

// crossesThreshold reports whether oldW and newW fall on opposite sides of a
// column-priority width threshold (40 or 60 terminal columns).
func crossesThreshold(oldW, newW int) bool {
	if (oldW < 40 && newW >= 40) || (oldW >= 40 && newW < 40) {
		return true
	}
	if (oldW < 60 && newW >= 60) || (oldW >= 60 && newW < 60) {
		return true
	}
	return false
}

// SetRows updates the table data. Each row is a map[string]string keyed by
// the column Key values defined in ColumnDef. Rows are re-styled immediately.
// Calling SetRows clears any previously set rich rows.
func (t *Table) SetRows(rows []map[string]string) {
	t.rows = rows
	t.richRows = nil
	t.applyRows()
}

// SetRichRows updates the table data with rows whose cell values may be either
// plain strings (rendered with the column's foreground colour) or
// btable.StyledCell instances (rendered with a per-cell foreground while still
// inheriting the row-level highlight background). Used by panes that need
// per-row colour variation that single-value column Color cannot express
// (e.g. GatewayLivePane's per-event-kind glyph colours).
//
// Existing SetRows([]map[string]string) callers are unaffected.
func (t *Table) SetRichRows(rows []map[string]any) {
	// Stored separately from t.rows so that Rows() (used by RebuildTableTheme)
	// keeps its existing string-only return contract for callers that do not
	// need rich values.
	t.richRows = rows
	t.rows = nil
	t.applyRows()
}

// Rows returns the current table data as a slice of row maps. Used by
// RebuildTableTheme to copy existing data into a freshly themed table.
func (t *Table) Rows() []map[string]string {
	return t.rows
}

// SetFocused enables or disables keyboard navigation. When unfocused the
// highlight cursor is hidden and key events are not processed.
func (t *Table) SetFocused(focused bool) {
	t.inner = t.inner.Focused(focused)
}

// SelectedIndex returns the currently highlighted row index (0-based).
func (t *Table) SelectedIndex() int {
	return (&t.inner).GetHighlightedRowIndex()
}

// Update forwards tea.Msg events to the inner bubble-table model and returns
// any resulting command. Key events only take effect when the table is focused.
func (t *Table) Update(msg tea.Msg) tea.Cmd {
	updated, cmd := t.inner.Update(msg)
	t.inner = updated
	return cmd
}

// Columns returns the currently visible column definitions after applying
// priority-based filtering at the current pane width. Tests use this to verify
// that the correct columns (and only those columns) are active after SetSize.
func (t *Table) Columns() []ColumnDef {
	return filterColumnsByPriority(t.config.Columns, t.width)
}

// View renders the table to a string. Call SetSize before first render.
func (t *Table) View() string {
	return t.inner.View()
}

// GotoTop resets the table scroll position to the first page.
// Used by panes to implement the universal Esc scroll-reset behaviour.
func (t *Table) GotoTop() {
	t.inner = t.inner.PageFirst()
}

// CurrentPage returns the current page number (1-indexed).
// Delegates to the inner model's pointer-receiver method.
func (t *Table) CurrentPage() int {
	return (&t.inner).CurrentPage()
}

// GotoPage navigates to the given page number (1-indexed). Used in tests to
// seed a non-first-page state before verifying that GotoTop resets it.
func (t *Table) GotoPage(page int) {
	t.inner = t.inner.WithCurrentPage(page)
}
