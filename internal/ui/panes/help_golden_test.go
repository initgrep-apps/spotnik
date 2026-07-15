package panes_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/goldentest"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// TestHelpOverlay_View_Keybindings verifies the golden snapshot of HelpOverlay
// with all keybinding categories rendered at 80×40 so the full overlay fits.
func TestHelpOverlay_View_Keybindings(t *testing.T) {
	th := theme.Load("black")
	overlay := panes.NewHelpOverlay(th)
	overlay.SetSize(80, 40)

	tm := goldentest.NewPaneTest(t, overlay, 80, 40)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestHelpOverlay_View_Narrow verifies the golden snapshot of HelpOverlay
// at narrow terminal width (40×40 so content is not clipped by height).
func TestHelpOverlay_View_Narrow(t *testing.T) {
	th := theme.Load("black")
	overlay := panes.NewHelpOverlay(th)
	overlay.SetSize(40, 40)

	tm := goldentest.NewPaneTest(t, overlay, 40, 40)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}
