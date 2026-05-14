package uikit

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// PaneChrome is the standard bordered pane primitive. It wraps
// layout.RenderPaneBorder (which is the internal implementation), applying
// the design-system role matrix for title, toggle key, and action hints.
//
// Every pane composes PaneChrome to render its outer border. Overlay-style
// panels should use OverlayChrome instead; full-screen panels should use
// Panel.
type PaneChrome struct {
	// Width is the total border width in terminal columns (includes the 2 border columns).
	Width int
	// Height is the total border height in terminal rows (includes the 2 border rows).
	Height int
	// Title is the pane title shown in the top border (e.g., "Playlists").
	Title string
	// ToggleKey is the number key (1-8) shown as a superscript before the title.
	// Pass 0 for panes that have no toggle key (e.g. Stats page panes).
	ToggleKey int
	// Actions are pane-specific shortcuts shown in the top-right of the border.
	// Displayed in corner-notch format: ╮ key label ╭, separated by ─.
	Actions []layout.Action
	// AccentColor is the per-pane border accent color (from Theme.PaneBorder*()).
	AccentColor lipgloss.Color
	// Focused controls whether the pane has keyboard focus.
	// Focused: full AccentColor + Bold title; unfocused: AccentColor + Faint (dimmed but colored).
	Focused bool
	// FilterQuery is non-empty when filter mode is active.
	// When set, replaces the action shortcuts with: filtering: "query" ─╮ Esc close ╭
	FilterQuery string
	// Theme provides token colours (KeyHint, TextMuted, etc.) for action rendering.
	Theme theme.Theme
}

// Render produces the bordered pane with content placed inside. The content
// argument is pre-sized to (Width-2, Height-2) by the caller; this method
// only composes the border around it.
//
// Mode (unicode/ascii) is taken from uikit.ActiveMode() and forwarded to
// layout.RenderPaneBorder via the BorderConfig glyph fields. This avoids an
// import cycle: layout must not import uikit, so glyph resolution happens here
// and the resolved strings are passed down.
func (p PaneChrome) Render(content string) string {
	m := ActiveMode()
	cfg := layout.BorderConfig{
		Width:       p.Width,
		Height:      p.Height,
		Title:       p.Title,
		ToggleKey:   p.ToggleKey,
		Actions:     p.Actions,
		AccentColor: p.AccentColor,
		Focused:     p.Focused,
		FilterQuery: p.FilterQuery,
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
	// In ascii mode, substitute a plain "N " (digit + space) for the unicode
	// superscript so the rendered prefix reads "+ - N Title" instead of using
	// a multi-byte unicode codepoint that may not render correctly.
	if m == GlyphASCII && p.ToggleKey > 0 {
		cfg.ToggleKeyStr = fmt.Sprintf("%d ", p.ToggleKey)
	}
	return layout.RenderPaneBorder(content, cfg)
}
