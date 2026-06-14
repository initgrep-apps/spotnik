package app_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newSearchTestApp creates an App with a reasonable window size,
// opens the search overlay, and returns the app in the search-open state.
func newSearchTestApp(t *testing.T) *app.App {
	t.Helper()
	a := app.New(&config.Config{}, app.AppOptions{})
	model, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a = model.(*app.App)
	return a
}

// TestSearchResultSelectedMsg_Show_SwitchesToPodcastPreset verifies that selecting a show
// on a music preset auto-switches to the Podcast preset.
func TestSearchResultSelectedMsg_Show_SwitchesToPodcastPreset(t *testing.T) {
	a := newSearchTestApp(t)
	// Default is Dashboard (music preset)
	require.Equal(t, layout.PresetNameDashboard, a.Layout().ActivePresetName())

	msg := panes.SearchResultSelectedMsg{
		URI:    "spotify:show:abc123",
		IsShow: true,
	}
	a.Update(msg)

	// Should have switched to Podcast preset
	assert.Equal(t, layout.PresetNamePodcast, a.Layout().ActivePresetName())
	assert.Equal(t, layout.PagePlayer, a.Layout().ActivePage())
}

// TestSearchResultSelectedMsg_Show_OnStatsPage_SwitchesToPlayer verifies that selecting
// a show while on the Stats page switches to the Player page and Podcast preset.
func TestSearchResultSelectedMsg_Show_OnStatsPage_SwitchesToPlayer(t *testing.T) {
	a := newSearchTestApp(t)
	// Switch to Stats page
	a.Layout().TogglePage()
	require.Equal(t, layout.PageStats, a.Layout().ActivePage())

	msg := panes.SearchResultSelectedMsg{
		URI:    "spotify:show:abc123",
		IsShow: true,
	}
	a.Update(msg)

	// Should have switched to Player page with Podcast preset
	assert.Equal(t, layout.PagePlayer, a.Layout().ActivePage())
	assert.Equal(t, layout.PresetNamePodcast, a.Layout().ActivePresetName())
}

// TestSearchResultSelectedMsg_Show_AlreadyOnPodcastPreset_NoDoubleSwitch verifies that
// selecting a show when already on a podcast preset does not change the preset.
func TestSearchResultSelectedMsg_Show_AlreadyOnPodcastPreset_NoDoubleSwitch(t *testing.T) {
	a := newSearchTestApp(t)
	// Switch to Podcast Dashboard preset (index 5)
	a.Layout().SetPreset(5)
	require.Equal(t, layout.PresetNamePodcastDashboard, a.Layout().ActivePresetName())

	presetBefore := a.Layout().ActivePresetIndex()

	msg := panes.SearchResultSelectedMsg{
		URI:    "spotify:show:abc123",
		IsShow: true,
	}
	a.Update(msg)

	// Should stay on Podcast Dashboard — no unnecessary switch to Podcast
	assert.Equal(t, layout.PresetNamePodcastDashboard, a.Layout().ActivePresetName())
	assert.Equal(t, presetBefore, a.Layout().ActivePresetIndex(),
		"preset index should not change when already on a podcast preset")
}
