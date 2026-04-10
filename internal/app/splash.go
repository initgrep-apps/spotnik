package app

import (
	"github.com/charmbracelet/lipgloss"
	figure "github.com/common-nighthawk/go-figure"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// appVersion is displayed on the splash screen.
const appVersion = "v1.1.0"

// renderSplashView builds the splash screen using go-figure ASCII art.
// It is a standalone function so it can be tested without an App instance.
func renderSplashView(t theme.Theme, width, height int) string {
	fig := figure.NewFigure("SPOTNIK", "serifcap", false)
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

	notice := lipgloss.NewStyle().
		Foreground(t.TextMuted()).
		Render("Playback controls require Spotify Premium")

	content := lipgloss.JoinVertical(lipgloss.Center,
		bannerStyle.Render(banner),
		"",
		tagline,
		version,
		"",
		notice,
	)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}
