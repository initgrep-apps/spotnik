package panes_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/goldentest"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// TestHelpOverlay_View_Keybindings verifies the golden snapshot of HelpOverlay
// with all keybinding categories rendered at 80×24.
func TestHelpOverlay_View_Keybindings(t *testing.T) {
	th := theme.Load("black")
	overlay := panes.NewHelpOverlay(th)
	overlay.SetSize(80, 24)

	tm := goldentest.NewPaneTest(t, overlay, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestHelpOverlay_View_Narrow verifies the golden snapshot of HelpOverlay
// at narrow terminal width (40×24).
func TestHelpOverlay_View_Narrow(t *testing.T) {
	th := theme.Load("black")
	overlay := panes.NewHelpOverlay(th)
	overlay.SetSize(40, 24)

	tm := goldentest.NewPaneTest(t, overlay, 40, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}
