package app_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/domain"
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

// TestSearchResult_EpisodeOnMusicPreset_SwitchesToPodcast verifies that selecting an
// episode from search while on a music preset auto-switches to the Podcast preset,
// closes search, and dispatches a play command.
func TestSearchResult_EpisodeOnMusicPreset_SwitchesToPodcast(t *testing.T) {
	a := newSearchTestApp(t)
	require.Equal(t, layout.PresetNameDashboard, a.Layout().ActivePresetName())

	msg := panes.SearchResultSelectedMsg{
		URI:       "spotify:episode:ep1",
		IsEpisode: true,
	}
	_, cmd := a.Update(msg)

	// Should switch to Podcast preset
	assert.Equal(t, layout.PresetNamePodcast, a.Layout().ActivePresetName())
	assert.Equal(t, layout.PagePlayer, a.Layout().ActivePage())
	// Should close search
	assert.False(t, a.SearchOpen(), "search should be closed after episode selection")
	// Should dispatch a batched command
	assert.NotNil(t, cmd, "episode selection must return a batched command")
}

// TestSearchResult_ShowOnMusicPreset_SwitchesToPodcastAndLoadsEpisodes verifies that
// selecting a show from search while on a music preset auto-switches to the Podcast
// preset, sets selected show ID, closes search, and dispatches a fetch command.
func TestSearchResult_ShowOnMusicPreset_SwitchesToPodcastAndLoadsEpisodes(t *testing.T) {
	a := newSearchTestApp(t)
	require.Equal(t, layout.PresetNameDashboard, a.Layout().ActivePresetName())

	msg := panes.SearchResultSelectedMsg{
		URI:    "spotify:show:abc123",
		IsShow: true,
	}
	_, cmd := a.Update(msg)

	// Should switch to Podcast preset
	assert.Equal(t, layout.PresetNamePodcast, a.Layout().ActivePresetName())
	assert.Equal(t, layout.PagePlayer, a.Layout().ActivePage())
	// Should close search
	assert.False(t, a.SearchOpen(), "search should be closed after show selection")
	// Should set the selected show ID
	assert.Equal(t, "abc123", a.Store().SelectedShowID(),
		"selected show ID should be set")
	// Should dispatch a batched command
	assert.NotNil(t, cmd, "show selection must return a batched command")
}

// TestSearchResult_ShowOnPodcastPreset_NoSwitchJustLoads verifies that selecting a
// show from search while already on a podcast preset does not change the preset,
// but still sets the show ID, closes search, and dispatches a fetch command.
func TestSearchResult_ShowOnPodcastPreset_NoSwitchJustLoads(t *testing.T) {
	a := newSearchTestApp(t)
	a.Layout().SetPreset(2) // Podcast preset
	require.Equal(t, layout.PresetNamePodcast, a.Layout().ActivePresetName())

	msg := panes.SearchResultSelectedMsg{
		URI:    "spotify:show:abc123",
		IsShow: true,
	}
	_, cmd := a.Update(msg)

	// Should stay on Podcast preset
	assert.Equal(t, layout.PresetNamePodcast, a.Layout().ActivePresetName(),
		"should stay on podcast preset")
	// Should close search
	assert.False(t, a.SearchOpen(), "search should be closed after show selection")
	// Should set the selected show ID
	assert.Equal(t, "abc123", a.Store().SelectedShowID(),
		"selected show ID should be set")
	// Should dispatch a batched command
	assert.NotNil(t, cmd, "show selection must return a batched command")
}

// TestSearchResult_TrackOnPodcastPreset_SwitchesToListening verifies that the
// auto-switch "track" path in SearchResultSelectedMsg switches from Podcast
// to Listening preset when the message type is not show or episode.
func TestSearchResult_TrackOnPodcastPreset_SwitchesToListening(t *testing.T) {
	a := newSearchTestApp(t)
	a.Layout().SetPreset(2) // Podcast preset
	require.Equal(t, layout.PresetNamePodcast, a.Layout().ActivePresetName())

	msg := panes.SearchResultSelectedMsg{
		URI: "spotify:track:abc123",
	}
	_, cmd := a.Update(msg)

	assert.Equal(t, layout.PresetNameListening, a.Layout().ActivePresetName(),
		"should switch from Podcast to Listening for track-type results")
	assert.Equal(t, layout.PagePlayer, a.Layout().ActivePage())
	assert.NotNil(t, cmd, "autoSwitchPreset must return a stale-check command")
}

// TestSearchResult_TrackOnMusicPreset_NoSwitch verifies that selecting a non-show/
// non-episode result while on a music preset does not change the preset.
func TestSearchResult_TrackOnMusicPreset_NoSwitch(t *testing.T) {
	a := newSearchTestApp(t)
	require.Equal(t, layout.PresetNameDashboard, a.Layout().ActivePresetName())

	msg := panes.SearchResultSelectedMsg{
		URI: "spotify:track:abc123",
	}
	a.Update(msg)

	assert.Equal(t, layout.PresetNameDashboard, a.Layout().ActivePresetName(),
		"should not switch when already on music preset")
	assert.Equal(t, layout.PagePlayer, a.Layout().ActivePage())
}

// TestSearchResult_EpisodeOnPodcastPreset_NoSwitch verifies that selecting an episode
// from search while already on a podcast preset does not change the preset.
func TestSearchResult_EpisodeOnPodcastPreset_NoSwitch(t *testing.T) {
	a := newSearchTestApp(t)
	a.Layout().SetPreset(2) // Podcast preset
	require.Equal(t, layout.PresetNamePodcast, a.Layout().ActivePresetName())

	msg := panes.SearchResultSelectedMsg{
		URI:       "spotify:episode:ep1",
		IsEpisode: true,
	}
	_, cmd := a.Update(msg)

	// Should stay on Podcast preset
	assert.Equal(t, layout.PresetNamePodcast, a.Layout().ActivePresetName(),
		"should stay on podcast preset")
	// Should close search
	assert.False(t, a.SearchOpen(), "search should be closed after episode selection")
	assert.NotNil(t, cmd, "episode selection must return a batched command")
}

// TestSearchResult_ShowOnStatsPage_SwitchesToPlayerAndPodcast verifies that selecting
// a show while on the Stats page switches to the Player page with Podcast preset.
func TestSearchResult_ShowOnStatsPage_SwitchesToPlayerAndPodcast(t *testing.T) {
	a := newSearchTestApp(t)
	a.Layout().TogglePage()
	require.Equal(t, layout.PageStats, a.Layout().ActivePage())

	msg := panes.SearchResultSelectedMsg{
		URI:    "spotify:show:abc123",
		IsShow: true,
	}
	a.Update(msg)

	// Should switch to Player page with Podcast preset
	assert.Equal(t, layout.PagePlayer, a.Layout().ActivePage(),
		"should switch from Stats to Player page")
	assert.Equal(t, layout.PresetNamePodcast, a.Layout().ActivePresetName(),
		"should be on Podcast preset")
	// Search should be closed
	assert.False(t, a.SearchOpen(), "search should be closed after show selection")
}

// TestSearchResult_EpisodeOnStatsPage_SwitchesToPlayerAndPodcast verifies that
// selecting an episode while on the Stats page switches to the Player page with
// Podcast preset.
func TestSearchResult_EpisodeOnStatsPage_SwitchesToPlayerAndPodcast(t *testing.T) {
	a := newSearchTestApp(t)
	a.Layout().TogglePage()
	require.Equal(t, layout.PageStats, a.Layout().ActivePage())

	msg := panes.SearchResultSelectedMsg{
		URI:       "spotify:episode:ep1",
		IsEpisode: true,
	}
	_, cmd := a.Update(msg)

	// Should switch to Player page with Podcast preset
	assert.Equal(t, layout.PagePlayer, a.Layout().ActivePage(),
		"should switch from Stats to Player page")
	assert.Equal(t, layout.PresetNamePodcast, a.Layout().ActivePresetName(),
		"should be on Podcast preset")
	assert.False(t, a.SearchOpen(), "search should be closed after episode selection")
	assert.NotNil(t, cmd, "episode selection must return a batched command")
}

// TestSearchResultSelectedMsg_Show_SetsShowEpisodesID verifies that selecting a show
// from search results sets the showEpisodesID staleness key so that the subsequent
// ShowEpisodesLoadedMsg response is not discarded.
func TestSearchResultSelectedMsg_Show_SetsShowEpisodesID(t *testing.T) {
	a := newSearchTestApp(t)

	msg := panes.SearchResultSelectedMsg{
		URI:    "spotify:show:abc123",
		IsShow: true,
	}
	_, cmd := a.Update(msg)

	assert.Equal(t, "abc123", a.ShowEpisodesID(),
		"showEpisodesID must be set when a show is selected from search")
	assert.NotNil(t, cmd, "show selection must return a batched cmd with fetch")

	// Verify that a ShowEpisodesLoadedMsg with the matching ShowID is NOT
	// discarded (it would be discarded if showEpisodesID were still "").
	loadedMsg := panes.ShowEpisodesLoadedMsg{
		ShowID: "abc123",
		Items:  []domain.Episode{{ID: "e1", Name: "Ep 1", URI: "spotify:episode:e1"}},
		Total:  1,
		Offset: 0,
	}
	model, _ := a.Update(loadedMsg)
	a = model.(*app.App)

	assert.False(t, a.Store().ShowEpisodesFetching(),
		"ShowEpisodesFetching sentinel must be cleared on matching response")
	assert.Len(t, a.Store().ShowEpisodes(), 1,
		"episodes must be written to store (response not discarded)")
	assert.Equal(t, 1, a.Store().ShowEpisodesTotal(),
		"total must be written to store")
}
