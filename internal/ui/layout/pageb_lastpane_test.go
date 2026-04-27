package layout_test

import (
	"testing"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/stretchr/testify/assert"
)

func TestTogglePane_PageB_CannotHideLastPane(t *testing.T) {
	m := layout.NewManager()
	m.Resize(200, 50)
	m.TogglePage() // Page B — PresetNerdStatus has 5 panes

	// Hide 4 of 5 panes
	m.TogglePane(layout.PaneGatewayHealth)
	m.TogglePane(layout.PanePollingTraffic)
	m.TogglePane(layout.PaneGatewayLive)
	m.TogglePane(layout.PaneNetworkLog)

	// Only NowPlaying remains visible
	assert.True(t, m.IsPaneVisible(layout.PaneNowPlaying), "NowPlaying must be visible")
	// Attempt to hide the last pane must be rejected
	m.TogglePane(layout.PaneNowPlaying)
	assert.True(t, m.IsPaneVisible(layout.PaneNowPlaying), "cannot-hide-last guard must reject on Page B")
}
