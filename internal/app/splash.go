package app

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/common-nighthawk/go-figure"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

const appVersion = "v1.1.0"

// renderSplashView renders the splash screen with go-figure ASCII art.
// Extracted as a free function for testability without an App instance.
func renderSplashView(t theme.Theme, width, height int) string {
	fig := figure.NewFigure("SPOTNIK", "doom", false)
	banner := fig.String()

	bannerStyle := lipgloss.NewStyle().
		Foreground(t.ActiveBorder()).
		Bold(true)

	tagline := lipgloss.NewStyle().
		Foreground(t.TextMuted()).
		Render("A terminal Spotify client for developers")

	version := lipgloss.NewStyle().
		Foreground(t.TextMuted()).
		Render(appVersion)

	content := lipgloss.JoinVertical(lipgloss.Center,
		bannerStyle.Render(banner),
		"",
		tagline,
		version,
	)

	return lipgloss.Place(width, height,
		lipgloss.Center, lipgloss.Center, content)
}
