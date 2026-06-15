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

// newAutoSwitchTestApp creates an App with a reasonable window size,
// sets terminal size and initializes layout, and returns the app in grid view.
func newAutoSwitchTestApp(t *testing.T) *app.App {
	t.Helper()
	a := app.New(&config.Config{}, app.AppOptions{})
	model, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a = model.(*app.App)
	return a
}

// =============================================================================
// autoSwitchPreset unit tests (direct method calls)
// =============================================================================

func TestAutoSwitchPreset_TrackOnPodcastPreset_SwitchesToListening(t *testing.T) {
	a := newAutoSwitchTestApp(t)
	// Set to Podcast preset (podcast-oriented)
	a.Layout().SetPreset(2)
	require.Equal(t, layout.PresetNamePodcast, a.Layout().ActivePresetName())

	a.AutoSwitchPreset("track")

	assert.Equal(t, layout.PresetNameListening, a.Layout().ActivePresetName(),
		"should switch from Podcast to Listening when playing a track")
}

func TestAutoSwitchPreset_EpisodeOnMusicPreset_SwitchesToPodcast(t *testing.T) {
	a := newAutoSwitchTestApp(t)
	// Default is Dashboard (music preset)
	require.Equal(t, layout.PresetNameDashboard, a.Layout().ActivePresetName())
	require.False(t, a.IsCurrentPresetPodcastOriented())

	a.AutoSwitchPreset("episode")

	assert.Equal(t, layout.PresetNamePodcast, a.Layout().ActivePresetName(),
		"should switch from music preset to Podcast when playing an episode")
}

func TestAutoSwitchPreset_TrackOnMusicPreset_NoSwitch(t *testing.T) {
	a := newAutoSwitchTestApp(t)
	// Default is Dashboard (music preset)
	require.Equal(t, layout.PresetNameDashboard, a.Layout().ActivePresetName())

	a.AutoSwitchPreset("track")

	assert.Equal(t, layout.PresetNameDashboard, a.Layout().ActivePresetName(),
		"should not switch when playing a track on a music preset")
}

func TestAutoSwitchPreset_EpisodeOnPodcastPreset_NoSwitch(t *testing.T) {
	a := newAutoSwitchTestApp(t)
	// Set to Podcast Dashboard (podcast-oriented, index 5)
	a.Layout().SetPreset(5)
	require.Equal(t, layout.PresetNamePodcastDashboard, a.Layout().ActivePresetName())

	a.AutoSwitchPreset("episode")

	assert.Equal(t, layout.PresetNamePodcastDashboard, a.Layout().ActivePresetName(),
		"should not switch when playing an episode on a podcast preset")
}

func TestAutoSwitchPreset_TrackOnPodcastDashboard_SwitchesToListening(t *testing.T) {
	a := newAutoSwitchTestApp(t)
	// Podcast Dashboard is also podcast-oriented
	a.Layout().SetPreset(5)
	require.Equal(t, layout.PresetNamePodcastDashboard, a.Layout().ActivePresetName())

	a.AutoSwitchPreset("track")

	assert.Equal(t, layout.PresetNameListening, a.Layout().ActivePresetName(),
		"should switch from Podcast Dashboard to Listening when playing a track")
}

func TestAutoSwitchPreset_EpisodeOnDashboard_SwitchesToPodcast(t *testing.T) {
	a := newAutoSwitchTestApp(t)
	// Set to Library (music preset, index 3)
	a.Layout().SetPreset(3)
	require.Equal(t, layout.PresetNameLibrary, a.Layout().ActivePresetName())

	a.AutoSwitchPreset("episode")

	assert.Equal(t, layout.PresetNamePodcast, a.Layout().ActivePresetName(),
		"should switch from Library to Podcast when playing an episode")
}

// =============================================================================
// isCurrentPresetPodcastOriented unit tests
// =============================================================================

func TestIsCurrentPresetPodcastOriented_PodcastPreset(t *testing.T) {
	a := newAutoSwitchTestApp(t)
	a.Layout().SetPreset(2) // Podcast
	assert.True(t, a.IsCurrentPresetPodcastOriented(),
		"Podcast preset should be podcast-oriented")
}

func TestIsCurrentPresetPodcastOriented_PodcastDashboardPreset(t *testing.T) {
	a := newAutoSwitchTestApp(t)
	a.Layout().SetPreset(5) // Podcast Dashboard
	assert.True(t, a.IsCurrentPresetPodcastOriented(),
		"Podcast Dashboard preset should be podcast-oriented")
}

func TestIsCurrentPresetPodcastOriented_MusicPreset_Dashboard(t *testing.T) {
	a := newAutoSwitchTestApp(t)
	// Default is Dashboard (music preset)
	assert.False(t, a.IsCurrentPresetPodcastOriented(),
		"Dashboard preset should not be podcast-oriented")
}

func TestIsCurrentPresetPodcastOriented_MusicPreset_Listening(t *testing.T) {
	a := newAutoSwitchTestApp(t)
	a.Layout().SetPreset(1) // Listening
	assert.False(t, a.IsCurrentPresetPodcastOriented(),
		"Listening preset should not be podcast-oriented")
}

func TestIsCurrentPresetPodcastOriented_MusicPreset_Library(t *testing.T) {
	a := newAutoSwitchTestApp(t)
	a.Layout().SetPreset(3) // Library
	assert.False(t, a.IsCurrentPresetPodcastOriented(),
		"Library preset should not be podcast-oriented")
}

func TestIsCurrentPresetPodcastOriented_MusicPreset_Discovery(t *testing.T) {
	a := newAutoSwitchTestApp(t)
	a.Layout().SetPreset(4) // Discovery
	assert.False(t, a.IsCurrentPresetPodcastOriented(),
		"Discovery preset should not be podcast-oriented")
}

// =============================================================================
// Handler integration tests (message-driven)
// =============================================================================

func TestPlayTrackMsg_AutoSwitches_WhenOnPodcastPreset(t *testing.T) {
	a := newAutoSwitchTestApp(t)
	a.Layout().SetPreset(2) // Podcast preset
	require.Equal(t, layout.PresetNamePodcast, a.Layout().ActivePresetName())

	msg := panes.PlayTrackMsg{TrackURI: "spotify:track:abc123"}
	_, cmd := a.Update(msg)

	assert.Equal(t, layout.PresetNameListening, a.Layout().ActivePresetName(),
		"PlayTrackMsg on podcast preset should switch to Listening")
	assert.NotNil(t, cmd, "PlayTrackMsg must batch preset stale-check with play command")
}

func TestPlayTrackMsg_NoSwitch_WhenOnMusicPreset(t *testing.T) {
	a := newAutoSwitchTestApp(t)
	// Default is Dashboard (music preset)
	require.Equal(t, layout.PresetNameDashboard, a.Layout().ActivePresetName())

	msg := panes.PlayTrackMsg{TrackURI: "spotify:track:abc123"}
	a.Update(msg)

	assert.Equal(t, layout.PresetNameDashboard, a.Layout().ActivePresetName(),
		"PlayTrackMsg on music preset should not switch")
}

func TestPlayTrackListMsg_AutoSwitches_WhenOnPodcastPreset(t *testing.T) {
	a := newAutoSwitchTestApp(t)
	a.Layout().SetPreset(2) // Podcast preset
	require.Equal(t, layout.PresetNamePodcast, a.Layout().ActivePresetName())

	msg := panes.PlayTrackListMsg{URIs: []string{"spotify:track:abc123"}}
	_, cmd := a.Update(msg)

	assert.Equal(t, layout.PresetNameListening, a.Layout().ActivePresetName(),
		"PlayTrackListMsg on podcast preset should switch to Listening")
	assert.NotNil(t, cmd, "PlayTrackListMsg must batch preset stale-check with play command")
}

func TestPlayTrackListMsg_NoSwitch_WhenOnMusicPreset(t *testing.T) {
	a := newAutoSwitchTestApp(t)
	require.Equal(t, layout.PresetNameDashboard, a.Layout().ActivePresetName())

	msg := panes.PlayTrackListMsg{URIs: []string{"spotify:track:abc123"}}
	a.Update(msg)

	assert.Equal(t, layout.PresetNameDashboard, a.Layout().ActivePresetName(),
		"PlayTrackListMsg on music preset should not switch")
}

func TestPlayEpisodeMsg_AutoSwitches_WhenOnMusicPreset(t *testing.T) {
	a := newAutoSwitchTestApp(t)
	// Default is Dashboard (music preset)
	require.Equal(t, layout.PresetNameDashboard, a.Layout().ActivePresetName())

	msg := panes.PlayEpisodeMsg{EpisodeURI: "spotify:episode:abc123"}
	_, cmd := a.Update(msg)

	assert.Equal(t, layout.PresetNamePodcast, a.Layout().ActivePresetName(),
		"PlayEpisodeMsg on music preset should switch to Podcast")
	assert.NotNil(t, cmd, "PlayEpisodeMsg must batch preset stale-check with play command")
}

func TestPlayEpisodeMsg_NoSwitch_WhenOnPodcastPreset(t *testing.T) {
	a := newAutoSwitchTestApp(t)
	a.Layout().SetPreset(2) // Podcast preset
	require.Equal(t, layout.PresetNamePodcast, a.Layout().ActivePresetName())

	msg := panes.PlayEpisodeMsg{EpisodeURI: "spotify:episode:abc123"}
	a.Update(msg)

	assert.Equal(t, layout.PresetNamePodcast, a.Layout().ActivePresetName(),
		"PlayEpisodeMsg on podcast preset should not switch")
}

func TestPlayContextMsg_ShowURI_AutoSwitchesToPodcast(t *testing.T) {
	a := newAutoSwitchTestApp(t)
	// Default is Dashboard (music preset)
	require.Equal(t, layout.PresetNameDashboard, a.Layout().ActivePresetName())

	msg := panes.PlayContextMsg{ContextURI: "spotify:show:xyz789"}
	_, cmd := a.Update(msg)

	assert.Equal(t, layout.PresetNamePodcast, a.Layout().ActivePresetName(),
		"PlayContextMsg with show URI on music preset should switch to Podcast")
	assert.NotNil(t, cmd, "PlayContextMsg must batch preset stale-check with play command")
}

func TestPlayContextMsg_AlbumURI_AutoSwitchesToListening(t *testing.T) {
	a := newAutoSwitchTestApp(t)
	a.Layout().SetPreset(2) // Podcast preset (podcast-oriented)
	require.Equal(t, layout.PresetNamePodcast, a.Layout().ActivePresetName())

	msg := panes.PlayContextMsg{ContextURI: "spotify:album:xyz789"}
	a.Update(msg)

	assert.Equal(t, layout.PresetNameListening, a.Layout().ActivePresetName(),
		"PlayContextMsg with album URI on podcast preset should switch to Listening")
}

func TestPlayContextMsg_PlaylistURI_AutoSwitchesToListening(t *testing.T) {
	a := newAutoSwitchTestApp(t)
	a.Layout().SetPreset(5) // Podcast Dashboard (podcast-oriented)
	require.Equal(t, layout.PresetNamePodcastDashboard, a.Layout().ActivePresetName())

	msg := panes.PlayContextMsg{ContextURI: "spotify:playlist:xyz789"}
	a.Update(msg)

	assert.Equal(t, layout.PresetNameListening, a.Layout().ActivePresetName(),
		"PlayContextMsg with playlist URI on podcast preset should switch to Listening")
}

func TestPlayContextMsg_ShowURI_OnPodcastPreset_NoSwitch(t *testing.T) {
	a := newAutoSwitchTestApp(t)
	a.Layout().SetPreset(2) // Already on Podcast preset
	require.Equal(t, layout.PresetNamePodcast, a.Layout().ActivePresetName())

	msg := panes.PlayContextMsg{ContextURI: "spotify:show:xyz789"}
	a.Update(msg)

	assert.Equal(t, layout.PresetNamePodcast, a.Layout().ActivePresetName(),
		"PlayContextMsg with show URI on podcast preset should not switch")
}

func TestPlayContextMsg_AlbumURI_OnMusicPreset_NoSwitch(t *testing.T) {
	a := newAutoSwitchTestApp(t)
	// Default is Dashboard (music preset)
	require.Equal(t, layout.PresetNameDashboard, a.Layout().ActivePresetName())

	msg := panes.PlayContextMsg{ContextURI: "spotify:album:xyz789"}
	a.Update(msg)

	assert.Equal(t, layout.PresetNameDashboard, a.Layout().ActivePresetName(),
		"PlayContextMsg with album URI on music preset should not switch")
}

func TestCyclePreset_DispatchesCheckNewlyVisiblePanes(t *testing.T) {
	a := newAutoSwitchTestApp(t)

	// Set to Listening preset (index 1) — only NowPlaying, Queue, RecentlyPlayed.
	a.Layout().SetPreset(1)
	require.Equal(t, layout.PresetNameListening, a.Layout().ActivePresetName())

	// Press 'p' to cycle to Podcast preset (index 2) — FollowedShows becomes newly visible.
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})

	assert.Equal(t, layout.PresetNamePodcast, a.Layout().ActivePresetName(),
		"'p' should cycle from Listening to Podcast")
	assert.NotNil(t, cmd, "'p' CyclePreset must dispatch checkNewlyVisiblePanes batch")
}

// =============================================================================
// PlaybackStateFetchedMsg must NOT trigger auto-switch
// =============================================================================

func TestPlaybackStateFetchedMsg_NoAutoSwitch(t *testing.T) {
	a := newAutoSwitchTestApp(t)
	a.Layout().SetPreset(2) // Podcast preset
	require.Equal(t, layout.PresetNamePodcast, a.Layout().ActivePresetName())

	// Simulate background poll returning a track (episode playback changed by someone else)
	state := &domain.PlaybackState{
		CurrentlyPlayingType: "track",
		Item: &domain.Track{
			ID:   "track123",
			Name: "Test Track",
		},
	}
	msg := panes.PlaybackStateFetchedMsg{
		State: state,
		Err:   nil,
	}
	a.Update(msg)

	// Must stay on Podcast preset — background changes don't trigger auto-switch
	assert.Equal(t, layout.PresetNamePodcast, a.Layout().ActivePresetName(),
		"PlaybackStateFetchedMsg must NOT trigger auto-switch")
}

func TestTogglePage_DispatchesCheckNewlyVisiblePanes(t *testing.T) {
	a := newAutoSwitchTestApp(t)

	// Toggle to Stats page first so that toggling back to Player makes library
	// panes newly visible and stale.
	a.Layout().TogglePage()
	require.Equal(t, layout.PageStats, a.Layout().ActivePage())

	// Press '0' to toggle back to Player page.
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})

	assert.Equal(t, layout.PagePlayer, a.Layout().ActivePage(),
		"'0' should toggle from Stats back to Player page")
	assert.NotNil(t, cmd, "'0' TogglePage must dispatch checkNewlyVisiblePanes batch")
}

func TestPlaybackStateFetchedMsg_EpisodeOnMusicPreset_NoSwitch(t *testing.T) {
	a := newAutoSwitchTestApp(t)
	// Default is Dashboard (music preset)
	require.Equal(t, layout.PresetNameDashboard, a.Layout().ActivePresetName())

	state := &domain.PlaybackState{
		CurrentlyPlayingType: "episode",
		Episode: &domain.Episode{
			ID:   "ep123",
			Name: "Test Episode",
		},
	}
	msg := panes.PlaybackStateFetchedMsg{
		State: state,
		Err:   nil,
	}
	a.Update(msg)

	assert.Equal(t, layout.PresetNameDashboard, a.Layout().ActivePresetName(),
		"PlaybackStateFetchedMsg must NOT trigger auto-switch even when content type changes")
}

// =============================================================================
// Preset stays on Player page after switch
// =============================================================================

func TestAutoSwitchPreset_StaysOnPlayerPage(t *testing.T) {
	a := newAutoSwitchTestApp(t)
	require.Equal(t, layout.PagePlayer, a.Layout().ActivePage())

	// Switch from music to podcast
	a.AutoSwitchPreset("episode")
	assert.Equal(t, layout.PagePlayer, a.Layout().ActivePage(),
		"should stay on Player page after auto-switch")

	// Switch from podcast to music
	a.AutoSwitchPreset("track")
	assert.Equal(t, layout.PagePlayer, a.Layout().ActivePage(),
		"should stay on Player page after auto-switch")
}

// =============================================================================
// Auto-switch from Stats page: preset switches on Player page, not Stats
// =============================================================================

func TestAutoSwitchPreset_FromStatsPage_SwitchesPlayerPreset(t *testing.T) {
	a := newAutoSwitchTestApp(t)
	a.Layout().TogglePage() // Switch to Stats page
	require.Equal(t, layout.PageStats, a.Layout().ActivePage())

	// autoSwitchPreset doesn't change the page, only the preset on the active page.
	// But the active page is Stats, so SetPreset(2) would set the Stats page preset.
	// This is expected behavior — auto-switch operates on the active page.

	// Switch back to Player page first (as SearchResultSelectedMsg does)
	a.Layout().SwitchToPage(layout.PagePlayer)
	require.Equal(t, layout.PagePlayer, a.Layout().ActivePage())
	require.Equal(t, layout.PresetNameDashboard, a.Layout().ActivePresetName())

	a.AutoSwitchPreset("episode")
	assert.Equal(t, layout.PresetNamePodcast, a.Layout().ActivePresetName())
}
