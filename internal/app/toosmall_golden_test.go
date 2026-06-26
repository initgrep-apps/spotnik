package app

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/goldentest"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// TestTooSmallView_Default verifies the golden snapshot of the "terminal too small"
// warning at 40×10, centered with current and required dimensions.
func TestTooSmallView_Default(t *testing.T) {
	cfg := &config.Config{}
	cfg.Preferences.Theme = theme.DefaultThemeID
	a := New(cfg, AppOptions{})
	a.width = 40
	a.height = 10

	got := a.renderTooSmall()
	goldentest.AssertGolden(t, got)
}

// TestTooSmallView_BareMinimum verifies the golden snapshot of the "terminal too small"
// warning at dimensions just below the minimum threshold (119×29, where min is 120×30).
func TestTooSmallView_BareMinimum(t *testing.T) {
	cfg := &config.Config{}
	cfg.Preferences.Theme = theme.DefaultThemeID
	a := New(cfg, AppOptions{})
	a.width = 119
	a.height = 29

	got := a.renderTooSmall()
	goldentest.AssertGolden(t, got)
}
