package uikit

import (
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// Action mirrors layout.Action for the primitive surface so call sites can
// use uikit.Action without importing internal/ui/layout directly.
type Action = layout.Action

// OverlayChrome renders a floating overlay panel. It is visually identical to
// a focused PaneChrome but always uses Theme.Accent() as the border colour —
// overlays always own input focus.
//
// Every overlay pane composes OverlayChrome to render its outer border.
// PaneChrome should be used for regular grid panes; Panel for full-screen panels.
type OverlayChrome struct {
	// Width is the total border width in terminal columns (includes 2 border columns).
	Width int
	// Height is the total border height in terminal rows (includes 2 border rows).
	Height int
	// Title is the overlay title shown in the top border (e.g., "Search").
	Title string
	// Actions are overlay-specific shortcuts shown in the top-right of the border.
	// Displayed in corner-notch format: ╮ key label ╭, separated by ─.
	Actions []Action
	// Theme provides the accent colour and token colours for action rendering.
	Theme theme.Theme
}

// Render produces the bordered overlay panel with content placed inside. The
// content argument is pre-sized to (Width-2, Height-2) by the caller; this
// method only composes the border around it.
//
// Mode (unicode/ascii) is taken from uikit.ActiveMode() and forwarded to
// layout.RenderPaneBorder via the BorderConfig glyph fields. This avoids an
// import cycle: layout must not import uikit, so glyph resolution happens here
// and the resolved strings are passed down.
func (o OverlayChrome) Render(content string) string {
	m := ActiveMode()
	cfg := layout.BorderConfig{
		Width:       o.Width,
		Height:      o.Height,
		Title:       o.Title,
		Actions:     o.Actions,
		AccentColor: o.Theme.Accent(),
		Focused:     true,
		Theme:       o.Theme,
		// Resolve glyph mode here so layout/border.go does not need to import
		// internal/uikit (which would create an import cycle).
		CornerTL: GlyphFor(GlyphCornerTL, m),
		CornerTR: GlyphFor(GlyphCornerTR, m),
		CornerBL: GlyphFor(GlyphCornerBL, m),
		CornerBR: GlyphFor(GlyphCornerBR, m),
		HRule:    GlyphFor(GlyphHRule, m),
		VRule:    GlyphFor(GlyphVRule, m),
	}
	return layout.RenderPaneBorder(content, cfg)
}
