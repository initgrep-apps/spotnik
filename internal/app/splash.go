package app

import (
	"github.com/charmbracelet/lipgloss"
	figure "github.com/common-nighthawk/go-figure"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// renderSplashView builds the splash screen using go-figure ASCII art.
// version is injected at build time via LDFLAGS (-X main.version=...) and
// forwarded through AppOptions; it falls back to "dev" for local builds.
// It is a standalone function so it can be tested without an App instance.
func renderSplashView(t theme.Theme, version string, width, height int) string {
	fig := figure.NewFigure("SPOTNIK", "serifcap", false)
	banner := fig.String()

	bannerStyle := lipgloss.NewStyle().
		Foreground(t.ActiveBorder()).
		Bold(true)

	tagline := lipgloss.NewStyle().
		Foreground(t.TextMuted()).
		Render("A terminal Spotify client for developers")

	versionStr := lipgloss.NewStyle().
		Foreground(t.TextMuted()).
		Render(version)

	notice := lipgloss.NewStyle().
		Foreground(t.TextMuted()).
		Render("Playback controls require Spotify Premium")

	content := lipgloss.JoinVertical(lipgloss.Center,
		bannerStyle.Render(banner),
		"",
		tagline,
		versionStr,
		"",
		notice,
	)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}
