package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// RenderError returns a styled error box centered within the given width/height.
// It uses rounded corners and theme tokens for the error symbol, message, and hint.
//
//	╭──────────────────────────────╮
//	│  ✗ Failed to load devices    │
//	│                              │
//	│  Press d to retry            │
//	╰──────────────────────────────╯
func RenderError(t theme.Theme, width, height int, message, retryHint string) string {
	errorSymbol := lipgloss.NewStyle().
		Foreground(t.Error()).
		Bold(true).
		Render("✗")

	msgText := lipgloss.NewStyle().
		Foreground(t.TextPrimary()).
		Render(message)

	hintText := lipgloss.NewStyle().
		Foreground(t.TextMuted()).
		Render(retryHint)

	content := errorSymbol + " " + msgText + "\n\n" + hintText

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Error()).
		Padding(1, 2).
		Align(lipgloss.Center)

	box := boxStyle.Render(content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
