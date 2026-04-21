package app

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
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
