package components

import (
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// TableChrome wraps Table. The primitive's role is to standardise
// construction — column tokens, header colour, playing-indicator colour — so
// that panes no longer build TableConfig literals inline.
//
// Call sites are not changed in this story; panes continue to call
// NewTable directly. TableChrome is the canonical wrapping pattern
// for future migrations.
type TableChrome struct {
	// Columns defines the column layout and per-column colour tokens.
	Columns []ColumnDef
	// Theme provides colour tokens for header, selection, and playing indicator.
	Theme theme.Theme
	inner *Table
}

// Inner returns the wrapped *Table, constructing it on first call.
// The inner table owns all interactive state (scroll position, selection, etc.);
// TableChrome is effectively stateless from the caller's perspective.
func (t *TableChrome) Inner() *Table {
	if t.inner == nil {
		t.inner = NewTable(TableConfig{
			Columns:      t.Columns,
			Theme:        t.Theme,
			PlayingIndex: -1,
			ShowHeader:   true,
		})
	}
	return t.inner
}
