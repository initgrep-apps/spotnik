package panes_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/goldentest"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// TestOnboardingPermissionsOverlay_View_Normal verifies the golden snapshot of
// the onboarding permissions overlay at 80×24 with all permission sections visible.
func TestOnboardingPermissionsOverlay_View_Normal(t *testing.T) {
	th := theme.Load("black")
	overlay := panes.NewOnboardingPermissionsOverlay(th)
	overlay.SetSize(80, 24)

	tm := goldentest.NewPaneTest(t, overlay, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestOnboardingPermissionsOverlay_View_Narrow verifies the golden snapshot of
// the onboarding permissions overlay at narrow terminal width (40×24).
func TestOnboardingPermissionsOverlay_View_Narrow(t *testing.T) {
	th := theme.Load("black")
	overlay := panes.NewOnboardingPermissionsOverlay(th)
	overlay.SetSize(40, 24)

	tm := goldentest.NewPaneTest(t, overlay, 40, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}
