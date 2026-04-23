package cliout

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func TestResolve_NoColor_returnsFixed(t *testing.T) {
	// Any mode + noColor=true must return Fixed regardless of TTY or theme.
	th := theme.Load("black")
	for _, m := range []PaletteMode{ModeAuto, ModeFixed, ModeTheme} {
		got := resolve(m, true, true, th)
		assert.Equal(t, Fixed, got, "mode %v with NO_COLOR should return Fixed", m)
	}
}

func TestResolve_ModeFixed_alwaysFixed(t *testing.T) {
	th := theme.Load("black")
	got := resolve(ModeFixed, true, false, th)
	assert.Equal(t, Fixed, got)
}

func TestResolve_ModeTheme_withTheme_returnsThemeTokens(t *testing.T) {
	th := theme.Load("black")
	got := resolve(ModeTheme, true, false, th)
	// Accent and Muted should come from the theme, not Fixed.
	assert.Equal(t, th.Accent(), got.Accent)
	assert.Equal(t, th.TextMuted(), got.Muted)
}

func TestResolve_ModeTheme_nilTheme_fallsBackToFixed(t *testing.T) {
	got := resolve(ModeTheme, true, false, nil)
	assert.Equal(t, Fixed, got)
}

func TestResolve_ModeAuto_nonTTY_returnsFixed(t *testing.T) {
	th := theme.Load("black")
	got := resolve(ModeAuto, false, false, th)
	assert.Equal(t, Fixed, got)
}

// Note: TestResolve_ModeAuto_TTY_dark is omitted because termenv.HasDarkBackground()
// is not mockable in unit tests. That path is covered by integration in Story 148.

func TestUse_replacesActive(t *testing.T) {
	prev := current()
	t.Cleanup(func() { Use(prev) })

	custom := Fixed
	custom.Accent = lipgloss.AdaptiveColor{Dark: "#ff00ff"}
	Use(custom)
	assert.Equal(t, custom, current())
}
