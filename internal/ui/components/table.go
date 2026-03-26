package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	btable "github.com/evertras/bubble-table/table"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// playingSymbol is the Unicode playing indicator shown in the index column.
const playingSymbol = "▶"

// emptyBorder is a bubble-table Border with space characters for all positions,
// effectively hiding the table's built-in border. The outer pane border
// (rendered by internal/ui/layout.RenderPaneBorder) handles the visible border.
var emptyBorder = btable.Border{
	Top:            " ",
	Left:           "",
	Right:          "",
	Bottom:         " ",
	TopRight:       " ",
	TopLeft:        " ",
	BottomRight:    " ",
	BottomLeft:     " ",
	TopJunction:    " ",
	LeftJunction:   "",
	RightJunction:  "",
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
	// Color is the lipgloss foreground color applied to data cells in this column.
	Color lipgloss.Color
}

// TableConfig holds configuration for creating a Table.
type TableConfig struct {
	// Columns defines the column layout and per-column colors.
	Columns []ColumnDef
	// Theme provides color tokens for header, selection, and playing indicator.
	Theme theme.Theme
	// PlayingIndex is the row index that shows the ▶ indicator (-1 = none).
	PlayingIndex int
	// ShowHeader controls whether the column header row is rendered.
	ShowHeader bool
}

// Table wraps bubble-table with Spotnik styling conventions: borderless mode,
// per-column colors, selected row highlighting, and a playing indicator.
type Table struct {
	inner        btable.Model
	config       TableConfig
	rows         []map[string]string
	playingIndex int
	width        int
	height       int
}

// NewTable creates a Table with the given configuration.
// Call SetSize before calling View to set dimensions.
func NewTable(cfg TableConfig) *Table {
	t := &Table{
		config:       cfg,
		playingIndex: cfg.PlayingIndex,
	}
	t.rebuild()
	return t
}

// rebuild reconstructs the inner bubble-table Model with current config and size.
// Called after config changes or size changes.
func (t *Table) rebuild() {
	th := t.config.Theme

	// Build bubble-table flex columns from ColumnDef slice.
	btCols := make([]btable.Column, len(t.config.Columns))
	for i, col := range t.config.Columns {
		btCols[i] = btable.NewFlexColumn(col.Key, col.Header, col.FlexFactor).
			WithStyle(lipgloss.NewStyle().Foreground(col.Color))
	}

	inner := btable.New(btCols).
		Border(emptyBorder).
		HeaderStyle(lipgloss.NewStyle().Foreground(th.TableHeader()).Bold(false)).
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

	pageSize := t.height - 1 // reserve 1 line for header when shown
	if !t.config.ShowHeader {
		pageSize = t.height
	}
	if pageSize > 0 {
		inner = inner.WithPageSize(pageSize)
	}

	t.inner = inner
	t.applyRows()
}

// applyRows converts the stored []map[string]string into bubble-table Row values
// and applies them to the inner model. The playing indicator replaces the index
// column value for the currently playing row.
func (t *Table) applyRows() {
	if len(t.rows) == 0 {
		t.inner = t.inner.WithRows(nil)
		return
	}

	th := t.config.Theme
	btRows := make([]btable.Row, len(t.rows))

	for i, rowData := range t.rows {
		data := btable.RowData{}
		for k, v := range rowData {
			data[k] = v
		}

		if i == t.playingIndex {
			// Replace the first column value with a styled playing indicator.
			if len(t.config.Columns) > 0 {
				firstKey := t.config.Columns[0].Key
				data[firstKey] = btable.NewStyledCell(
					playingSymbol,
					lipgloss.NewStyle().Foreground(th.PlayingIndicator()),
				)
			}
		}

		btRows[i] = btable.NewRow(data)
	}

	t.inner = t.inner.WithRows(btRows)
}

// SetSize updates the table dimensions. Recalculates column widths and page size.
func (t *Table) SetSize(width, height int) {
	t.width = width
	t.height = height
	t.inner = t.inner.WithTargetWidth(width)

	pageSize := height - 1
	if !t.config.ShowHeader {
		pageSize = height
	}
	if pageSize > 0 {
		t.inner = t.inner.WithPageSize(pageSize)
	}
}

// SetRows updates the table data. Each row is a map[string]string keyed by
// the column Key values defined in ColumnDef. Rows are re-styled immediately.
func (t *Table) SetRows(rows []map[string]string) {
	t.rows = rows
	t.applyRows()
}

// SetPlayingIndex marks which row index shows the ▶ indicator.
// Pass -1 to clear the indicator. The rows are re-applied immediately.
func (t *Table) SetPlayingIndex(index int) {
	t.playingIndex = index
	t.applyRows()
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

// View renders the table to a string. Call SetSize before first render.
func (t *Table) View() string {
	return t.inner.View()
}
