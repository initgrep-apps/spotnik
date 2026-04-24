package uikit

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// PanelIntent selects the border colour applied to a Panel.
type PanelIntent int

const (
	// PanelIntentDefault renders the panel border with the theme Accent colour.
	// Use for normal full-screen panels (onboarding, auth, splash).
	PanelIntentDefault PanelIntent = iota
	// PanelIntentError renders the panel border with the theme Error colour.
	// Use for failure screens (e.g. onboarding failure panel).
	PanelIntentError
)

// Panel is a full-screen bordered container whose title lives inside the top
// border line — there is no separate step-header row above the body. It
// delegates to layout.RenderPaneBorder, supplying AccentColor from the intent.
//
// Use PaneChrome for grid panes and OverlayChrome for floating overlays.
// Panel is reserved for full-viewport surfaces such as onboarding, auth, and
// the too-small screen.
type Panel struct {
	// Width is the total border width in terminal columns (includes 2 border columns).
	Width int
	// Height is the total border height in terminal rows (includes 2 border rows).
	Height int
	// Title is shown in the top border line (absorbs the step-header role).
	Title string
	// Intent selects the border colour: Default → Accent, Error → Error.
	Intent PanelIntent
	// Theme provides token colours for the border and title.
	Theme theme.Theme
}

// accentColor returns the lipgloss.Color selected by the Panel's Intent.
func (p Panel) accentColor() lipgloss.Color {
	if p.Intent == PanelIntentError {
		return p.Theme.Error()
	}
	return p.Theme.Accent()
}

// Render produces the full-screen bordered panel with content placed inside.
// The content argument is pre-sized to (Width-2, Height-2) by the caller; this
// method only composes the border around it.
//
// Mode (unicode/ascii) is taken from uikit.ActiveMode() and forwarded to
// layout.RenderPaneBorder via the BorderConfig glyph fields. This avoids an
// import cycle: layout must not import uikit, so glyph resolution happens here
// and the resolved strings are passed down.
func (p Panel) Render(content string) string {
	m := ActiveMode()
	cfg := layout.BorderConfig{
		Width:       p.Width,
		Height:      p.Height,
		Title:       p.Title,
		AccentColor: p.accentColor(),
		Focused:     true,
		Theme:       p.Theme,
		// Resolve glyph mode here so layout/border.go does not need to import
		// internal/uikit (which would create a cycle).
		CornerTL: GlyphFor(GlyphCornerTL, m),
		CornerTR: GlyphFor(GlyphCornerTR, m),
		CornerBL: GlyphFor(GlyphCornerBL, m),
		CornerBR: GlyphFor(GlyphCornerBR, m),
		HRule:    GlyphFor(GlyphHRule, m),
		VRule:    GlyphFor(GlyphVRule, m),
	}
	return layout.RenderPaneBorder(content, cfg)
}
