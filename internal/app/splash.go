package app

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// bannerUnicode is the SPOTNIK splash art for unicode terminals.
// ANSI Shadow style: full-block █ + box-drawing corners form filled block letters.
const bannerUnicode = `███████╗██████╗  ██████╗ ████████╗███╗   ██╗██╗██╗  ██╗
██╔════╝██╔══██╗██╔═══██╗╚══██╔══╝████╗  ██║██║██║ ██╔╝
███████╗██████╔╝██║   ██║   ██║   ██╔██╗ ██║██║█████╔╝
╚════██║██╔═══╝ ██║   ██║   ██║   ██║╚██╗██║██║██╔═██╗
███████║██║     ╚██████╔╝   ██║   ██║ ╚████║██║██║  ██╗
╚══════╝╚═╝      ╚═════╝    ╚═╝   ╚═╝  ╚═══╝╚═╝╚═╝  ╚═╝`

// bannerASCII is the SPOTNIK splash art for ascii terminals (ui.glyphs = "ascii").
// figlet "big" font style — 6-line open-outline matching unicode banner height.
// Backtick in N replaced with apostrophe for raw-string-literal compatibility.
const bannerASCII = `   _____   _____     ____    _______   _   _   _____   _  __
  / ____| |  __ \   / __ \  |__   __| | \ | | |_   _| | |/ /
 | (___   | |__) | | |  | |    | |    |  \| |   | |   | ' /
  \___ \  |  ___/  | |  | |    | |    | . ' |   | |   |  <
  ____) | | |      | |__| |    | |    | |\  |  _| |_  | . \
 |_____/  |_|       \____/     |_|    |_| \_| |_____| |_|\_\`

// renderSplashView builds the splash screen.
// version is injected at build time via LDFLAGS (-X main.version=...) and
// forwarded through AppOptions; it falls back to "dev" for local builds.
// It is a standalone function so it can be tested without an App instance.
func renderSplashView(t theme.Theme, version string, width, height int) string {
	banner := bannerUnicode
	if uikit.ActiveMode() == uikit.GlyphASCII {
		banner = bannerASCII
	}

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
