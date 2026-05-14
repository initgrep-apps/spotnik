// Package panes — BasePane is an embedded struct providing the five fields and four
// trivial Pane interface methods shared by all 8 table-based Music page panes.
package panes

import (
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// BasePane holds the fields and trivial Pane interface methods shared by all 8
// table-based Music page panes. Embed this struct in each pane to eliminate repeated
// declarations.
//
// What BasePane provides: store, theme, focused, width, height fields;
// IsFocused(), SetFocused(), SetSize(), HasActiveFilter() implementations.
//
// What BasePane does NOT provide: Table or Filter fields (column layouts differ
// per pane); SetTheme() (table rebuild is pane-specific); ID(), Title(),
// ToggleKey(), Actions(); Init(), Update(), View().
//
// Override policy: SetFocused and SetSize are base implementations. Panes that
// also need to forward focus/size to their table must override these (calling
// b.BasePane.SetFocused(f) or b.BasePane.SetSize(w, h) first).
// HasActiveFilter returns false by default; panes with a filter override it.
type BasePane struct {
	store   state.StateReader
	theme   theme.Theme
	focused bool
	width   int
	height  int
}

// IsFocused returns true when the pane has keyboard focus.
func (b *BasePane) IsFocused() bool { return b.focused }

// SetFocused sets the keyboard focus state.
// Panes that forward focus to a table must override this method.
func (b *BasePane) SetFocused(f bool) { b.focused = f }

// SetSize updates the render dimensions.
// Panes that forward size to a table must override this method.
func (b *BasePane) SetSize(w, h int) { b.width = w; b.height = h }

// HasActiveFilter returns false by default. Panes with an in-pane filter
// must override this to return p.filter.IsActive().
func (b *BasePane) HasActiveFilter() bool { return false }
