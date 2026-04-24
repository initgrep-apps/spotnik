package uikit

import (
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// TableChrome wraps components.Table. The primitive's role is to standardise
// construction — column tokens, header colour, playing-indicator colour — so
// that panes no longer build TableConfig literals inline.
//
// Call sites are not changed in this story; panes continue to call
// components.NewTable directly. TableChrome is the canonical wrapping pattern
// for future migrations.
type TableChrome struct {
	// Columns defines the column layout and per-column colour tokens.
	Columns []components.ColumnDef
	// Theme provides colour tokens for header, selection, and playing indicator.
	Theme theme.Theme
	inner *components.Table
}

// Inner returns the wrapped *components.Table, constructing it on first call.
// The inner table owns all interactive state (scroll position, selection, etc.);
// TableChrome is effectively stateless from the caller's perspective.
func (t *TableChrome) Inner() *components.Table {
	if t.inner == nil {
		t.inner = components.NewTable(components.TableConfig{
			Columns:      t.Columns,
			Theme:        t.Theme,
			PlayingIndex: -1,
			ShowHeader:   true,
		})
	}
	return t.inner
}
