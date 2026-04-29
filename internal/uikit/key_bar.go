package uikit

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// KeyBar renders a one-row keybinding strip: "c copy · q quit".
// Used by StatusBar (global) and by overlays/panels for local hints.
// The "·" middot separator becomes "|" in ascii mode.
//
// Roles: Key → theme.KeyHint(), Desc → theme.TextMuted(), Separator → theme.TextMuted().
type KeyBar struct {
	Bindings []key.Binding
	Theme    theme.Theme
}

// Render returns the styled keybinding strip as a single-line ANSI string.
// An empty Bindings slice returns "".
func (k KeyBar) Render() string {
	if len(k.Bindings) == 0 {
		return ""
	}
	t := k.Theme
	keyStyle := lipgloss.NewStyle().Foreground(t.KeyHint())
	descStyle := lipgloss.NewStyle().Foreground(t.TextMuted())

	sep := " " + GlyphFor(GlyphSeparator, ActiveMode()) + " "

	parts := make([]string, 0, len(k.Bindings))
	for _, b := range k.Bindings {
		h := b.Help()
		parts = append(parts, keyStyle.Render(h.Key)+" "+descStyle.Render(h.Desc))
	}
	return strings.Join(parts, descStyle.Render(sep))
}
