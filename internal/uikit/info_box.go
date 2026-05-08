package uikit

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// InfoBox renders a short titled bordered block intended for inline use inside
// a larger panel — for example, an "About these permissions" notice above a
// URL box on the OAuth onboarding screen.
//
// It is visually similar to URLBox (rounded muted border, padding 0,1) but
// shows Title as an emphasized first row above the body. For full-viewport
// panels use Panel; for floating overlays use OverlayChrome.
//
// Roles:
//   - InfoBox.Border → Muted (TextMuted colour token)
//   - InfoBox.Title  → Accent + bold
//   - InfoBox.Body   → TextPrimary
type InfoBox struct {
	// Title is the emphasized first line inside the box.
	Title string
	// Body is the wrapped content displayed below the title.
	Body string
	// Width is the total column width of the rendered box including borders.
	Width int
	// Theme provides colour tokens.
	Theme theme.Theme
}

// Render returns the titled box with the body inside. In unicode mode a
// rounded border (╭╮╰╯─│) is used; in ASCII mode the border uses +/-/|.
func (b InfoBox) Render() string {
	// innerW: border (1 each side) + padding (1 each side) = 4 chars overhead.
	// The outer style uses Width - 2 (border columns only) so lipgloss draws
	// the border at the intended width while content flows inside the padding.
	innerW := b.Width - 4
	if innerW < 1 {
		innerW = 1
	}

	titleRow := lipgloss.NewStyle().
		Foreground(b.Theme.Accent()).
		Bold(true).
		Width(innerW).
		MaxWidth(innerW).
		Render(b.Title)

	bodyRow := lipgloss.NewStyle().
		Foreground(b.Theme.TextPrimary()).
		Width(innerW).
		MaxWidth(innerW).
		Render(b.Body)

	content := lipgloss.JoinVertical(lipgloss.Left, titleRow, bodyRow)

	style := lipgloss.NewStyle().
		Border(RoundedBorder()).
		BorderForeground(b.Theme.TextMuted()).
		Padding(0, 1).
		Width(b.Width - 2)
	return style.Render(content)
}
