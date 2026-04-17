package app

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func TestRenderSplash_ContainsBranding(t *testing.T) {
	th := theme.Load("black")
	view := renderSplashView(th, "v0.1.0", 120, 40)

	// go-figure "doom" font renders letters as ASCII art, so we check for
	// recognizable fragments from the rendered output.
	assert.Contains(t, view, "___", "splash should contain go-figure ASCII art")
	assert.Contains(t, view, "v0.1.0", "splash should contain the injected version")
	assert.Contains(t, view, "terminal Spotify client", "splash should contain the tagline")
}

func TestRenderSplash_SmallTerminal(t *testing.T) {
	th := theme.Load("black")
	// Even with a small terminal, renderSplashView should not panic.
	view := renderSplashView(th, "dev", 40, 10)
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "dev")
}

func TestRenderSplash_ContainsPremiumNotice(t *testing.T) {
	th := theme.Load("black")
	view := renderSplashView(th, "v0.1.0", 120, 40)

	assert.Contains(t, view, "Playback controls require",
		"splash should contain the static Premium notice line")
	assert.Contains(t, view, "Spotify Premium",
		"splash should mention Spotify Premium in the notice")
}
