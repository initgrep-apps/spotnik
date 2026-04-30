package app

import (
	"github.com/charmbracelet/lipgloss"
	figure "github.com/common-nighthawk/go-figure"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// renderSplashView builds the splash screen using go-figure ASCII art.
// version is injected at build time via LDFLAGS (-X main.version=...) and
// forwarded through AppOptions; it falls back to "dev" for local builds.
// It is a standalone function so it can be tested without an App instance.
func renderSplashView(t theme.Theme, version string, width, height int) string {
	// NOTE: dotmatrix was the preferred font but requires ~144 columns to render
	// without wrapping; banner3-D fits cleanly at 120 columns.
	fig := figure.NewFigure("SPOTNIK", "banner3-D", false)
	banner := fig.String()

	bannerStyle := lipgloss.NewStyle().
		Foreground(t.ActiveBorder()).
		Bold(true)

	tagline := lipgloss.NewStyle().
		Foreground(t.TextMuted()).
		Render("A terminal Spotify client")

	versionLine := lipgloss.NewStyle().
		Foreground(t.TextPrimary()).
		Render("Version " + version)

	premiumLine := uikit.StatusGlyph{
		Role:  uikit.RoleWarning,
		Text:  "Playback controls require Spotify Premium",
		Theme: t,
		Gap:   1,
	}.Render()

	warningPanel := lipgloss.NewStyle().
		Border(uikit.RoundedBorder()).
		BorderForeground(t.TextMuted()).
		Padding(0, 2).
		Render(premiumLine)

	content := lipgloss.JoinVertical(lipgloss.Center,
		bannerStyle.Render(banner),
		"",
		tagline,
		"",
		versionLine,
		"",
		warningPanel,
	)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}
