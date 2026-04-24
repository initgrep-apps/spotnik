package uikit

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// Chip renders a small inline pill: " <glyph> <label> " on the status-bar
// background. Used for header chips (device, profile) and wherever a pill is
// appropriate. The glyph colour is driven by Intent role; the label colour is
// always theme.HeaderChipFg(). The entire chip sits on theme.StatusBarBg().
type Chip struct {
	Glyph  GlyphRole
	Label  string
	Intent Role
	Theme  theme.Theme
}

// Render returns the ANSI-styled chip string in the form " <glyph> <label> ".
func (c Chip) Render() string {
	bg := lipgloss.NewStyle().Background(c.Theme.StatusBarBg())
	glyph := lipgloss.NewStyle().
		Foreground(ColourFor(c.Intent, c.Theme)).
		Background(c.Theme.StatusBarBg()).
		Render(GlyphFor(c.Glyph, ActiveMode()))
	label := lipgloss.NewStyle().
		Foreground(c.Theme.HeaderChipFg()).
		Background(c.Theme.StatusBarBg()).
		Render(c.Label)
	return bg.Render(" ") + glyph + bg.Render(" ") + label + bg.Render(" ")
}
