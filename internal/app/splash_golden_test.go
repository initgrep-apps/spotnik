package app

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/goldentest"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// TestSplashView_Default verifies the golden snapshot of the splash screen
// at 120×40 with the unicode banner, version string, and premium warning.
func TestSplashView_Default(t *testing.T) {
	th := theme.Load("black")
	got := renderSplashView(th, "v0.1.0", 120, 40)
	goldentest.AssertGolden(t, got)
}

// TestSplashView_Narrow verifies the golden snapshot of the splash screen
// at 40×10 — scaled down, still centered.
func TestSplashView_Narrow(t *testing.T) {
	th := theme.Load("black")
	got := renderSplashView(th, "v0.1.0", 40, 10)
	goldentest.AssertGolden(t, got)
}
