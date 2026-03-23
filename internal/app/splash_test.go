package app

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func TestRenderSplash_ContainsBranding(t *testing.T) {
	th := theme.Load("black")
	output := renderSplashView(th, 120, 30)
	assert.Contains(t, output, "terminal Spotify client", "should show tagline")
	assert.Contains(t, output, "v1.1.0", "should show version")
	assert.NotEmpty(t, output)
}

func TestRenderSplash_SmallTerminal(t *testing.T) {
	th := theme.Load("black")
	output := renderSplashView(th, 40, 10)
	assert.NotEmpty(t, output, "should render even in small terminals")
}
