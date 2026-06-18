package components

import "github.com/initgrep-apps/spotnik/internal/ui/theme"

// RebuildTableTheme creates a new Table with updated theme colors and re-applies
// the existing rows from the old table. Called by pane SetTheme() to avoid
// repeating the same 10-line pattern across all 8 panes.
//
// Usage:
//
//	func (p *MyPane) SetTheme(th theme.Theme) {
//	    p.theme = th
//	    cols := []ColumnDef{
//	        {Key: "track", Header: "Track", FlexFactor: 9, Color: th.ColumnPrimary()},
//	    }
//	    p.table, p.filter = components.RebuildTableTheme(th, cols, p.table.Rows(), p.focused && !p.filter.IsActive())
//	}
func RebuildTableTheme(
	th theme.Theme,
	cols []ColumnDef,
	rows []map[string]string,
	focused bool,
) (*Table, *Filter) {
	t := NewTable(TableConfig{Columns: cols, Theme: th, ShowHeader: true})
	t.SetRows(rows)
	t.SetFocused(focused)
	return t, NewFilter(th)
}
