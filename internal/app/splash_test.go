package app

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
)

func TestRenderSplashView_containsTagline(t *testing.T) {
	th := theme.Load("black")
	view := renderSplashView(th, "v0.1.0", 120, 40)
	assert.Contains(t, view, "A terminal Spotify client")
	assert.NotContains(t, view, "for developers")
}

func TestRenderSplashView_containsVersion(t *testing.T) {
	th := theme.Load("black")
	view := renderSplashView(th, "v1.2.3", 120, 40)
	assert.Contains(t, view, "v1.2.3")
}

func TestRenderSplashView_smallTerminal_noPanic(t *testing.T) {
	th := theme.Load("black")
	// Should not panic even at small sizes.
	view := renderSplashView(th, "dev", 40, 10)
	assert.NotEmpty(t, view)
}

func TestRenderSplashView_containsPremiumNotice(t *testing.T) {
	th := theme.Load("black")
	view := renderSplashView(th, "v0.1.0", 120, 40)

	assert.Contains(t, view, "Playback controls require",
		"splash should contain the static Premium notice line")
	assert.Contains(t, view, "Spotify Premium",
		"splash should mention Spotify Premium in the notice")
}

// TestRenderSplashView_AsciiMode asserts that ascii mode selects the plain-ASCII
// banner and that the warning panel border also honours the mode (no rounded glyphs).
func TestRenderSplashView_AsciiMode(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	th := theme.Load("black")
	view := renderSplashView(th, "v0.1.0", 120, 40)

	for _, banned := range []string{"╭", "╮", "╰", "╯", "─", "│", "█"} {
		assert.NotContains(t, view, banned,
			"ASCII mode must not contain unicode glyph %q", banned)
	}
	assert.Contains(t, view, "+", "ASCII mode warning panel should render '+' corners")
	assert.Contains(t, view, "|", "ASCII mode warning panel should render '|' vertical rules")
}

// TestRenderSplashView_UnicodeMode verifies the block-character banner and rounded
// panel border are both present in unicode mode.
func TestRenderSplashView_UnicodeMode(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	th := theme.Load("black")
	view := renderSplashView(th, "v0.1.0", 120, 40)

	assert.Contains(t, view, "█", "unicode mode splash should contain block-character banner")

	hasRounded := false
	for _, want := range []string{"╭", "╮", "╰", "╯"} {
		if assert.Contains(t, view, want) {
			hasRounded = true
		}
	}
	assert.True(t, hasRounded, "unicode mode splash should contain rounded border corners")
}
