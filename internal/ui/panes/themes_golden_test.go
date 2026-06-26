package panes_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/goldentest"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// TestThemeOverlay_View_ThemeList verifies the golden snapshot of ThemeOverlay
// with all 13 themes listed, current theme marked with ✓, at 80×24.
func TestThemeOverlay_View_ThemeList(t *testing.T) {
	th := theme.Load("black")
	allThemes := theme.AllThemes()
	overlay := panes.NewThemeOverlay(allThemes, "black", th)
	overlay.SetSize(80, 24)

	tm := goldentest.NewPaneTest(t, overlay, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestThemeOverlay_View_Narrow verifies the golden snapshot of ThemeOverlay
// at narrow terminal width (40×24).
func TestThemeOverlay_View_Narrow(t *testing.T) {
	th := theme.Load("black")
	allThemes := theme.AllThemes()
	overlay := panes.NewThemeOverlay(allThemes, "black", th)
	overlay.SetSize(40, 24)

	tm := goldentest.NewPaneTest(t, overlay, 40, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}
