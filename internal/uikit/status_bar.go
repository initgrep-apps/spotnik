package uikit

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// StatusBar is the bottom global key bar. Always 3 lines tall (top border +
// content row + bottom border). Uses bubbles/help for short-help single-row
// rendering. Minimum effective width is 160 columns so all bindings are visible
// on one row even when no terminal size has been set.
//
// Roles: Key → theme.Info(), Desc → theme.TextMuted(), border accent → theme.TextMuted().
type StatusBar struct {
	Width    int
	Bindings help.KeyMap
	Theme    theme.Theme
}

// Render produces the ANSI-styled 3-line status bar string.
func (s StatusBar) Render() string {
	const statusH = 3 // top border + 1 content row + bottom border
	w := s.Width
	if w < 160 {
		w = 160
	}

	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(s.Theme.Info())
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(s.Theme.TextMuted())
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(s.Theme.TextMuted())
	h.Styles.FullKey = lipgloss.NewStyle().Foreground(s.Theme.Info())
	h.Styles.FullDesc = lipgloss.NewStyle().Foreground(s.Theme.TextMuted())
	h.Styles.FullSeparator = lipgloss.NewStyle().Foreground(s.Theme.TextMuted())

	content := h.View(s.Bindings)
	inner := lipgloss.NewStyle().
		Width(w - 2).MaxWidth(w - 2).
		Height(statusH - 2).MaxHeight(statusH - 2).
		Render(content)

	cfg := layout.BorderConfig{
		Width:       w,
		Height:      statusH,
		Title:       "",
		Actions:     []layout.Action{},
		AccentColor: s.Theme.TextMuted(),
		Focused:     false,
		Theme:       s.Theme,
	}
	return layout.RenderPaneBorder(inner, cfg)
}
