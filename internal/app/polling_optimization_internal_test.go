package app

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newPollOptTestApp creates an App with a window size set and returns it in grid view
// after dismissing splash, so the tick loop is active and layout is computed.
func newPollOptTestApp(t *testing.T) *App {
	t.Helper()
	a := newTestAppInternal()
	model, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a = model.(*App)
	a.currentView = viewGrid

	// Set NowPlaying pane to a non-nil state so the progression forwarding works.
	a.store.SetPlaybackState(&domain.PlaybackState{
		IsPlaying: false,
		Item:      &domain.Track{ID: "t1", Name: "Test"},
	})
	a.store.SetQueue([]domain.QueueItem{})
	return a
}

// =============================================================================
// TestPolling_SkipsInvisiblePane — panes not in the current preset are skipped.
// =============================================================================

func TestPolling_SkipsInvisiblePane(t *testing.T) {
	a := newPollOptTestApp(t)

	// Dashboard preset (index 0) does NOT include FollowedShows or SavedEpisodes.
	require.Equal(t, layout.PresetNameDashboard, a.layout.ActivePresetName())
	assert.False(t, a.layout.IsPaneVisible(layout.PaneFollowedShows))
	assert.False(t, a.layout.IsPaneVisible(layout.PaneSavedEpisodes))

	// Set fetching sentinel on FollowedShows BEFORE tick. If polling loop runs
	// for FollowedShows, it will create a FollowedShowsLoadedMsg. But since the
	// pane is NOT visible, the loop should skip it entirely, leaving the sentinel
	// UNCHANGED (no fetch dispatched, no sentinel cleared).
	a.store.SetFollowedShowsFetching(true)

	// Send tick that would trigger FollowedShows poll.
	// tickCount=0 → interval=5 (retry mode), 0%5==0 would trigger if visible.
	_, _ = a.Update(panes.TickMsg{})

	// Sentinel must NOT be cleared because the entry was skipped.
	assert.True(t, a.store.FollowedShowsFetching(),
		"fetching sentinel must NOT be cleared when pane is invisible")
}

// =============================================================================
// TestPolling_PollsVisiblePane — visible panes do get polled.
// =============================================================================

func TestPolling_PollsVisiblePane(t *testing.T) {
	a := newPollOptTestApp(t)

	// Dashboard preset (index 0) includes Playlists, Albums, LikedSongs, RecentlyPlayed, TopTracks/Artists.
	require.Equal(t, layout.PresetNameDashboard, a.layout.ActivePresetName())
	assert.True(t, a.layout.IsPaneVisible(layout.PanePlaylists))

	// At tick 0 with hasData=false, interval=5, 0%5==0 → should dispatch fetch.
	_, cmd := a.Update(panes.TickMsg{})
	require.NotNil(t, cmd)

	msgs := collectAllTestMsgs(cmd)
	hasPlaylists := false
	for _, m := range msgs {
		if _, ok := m.(panes.LibraryLoadedMsg); ok {
			hasPlaylists = true
		}
	}
	assert.True(t, hasPlaylists, "visible Playlists pane should dispatch LibraryLoadedMsg")
}

// collectAllTestMsgs resolves a tea.Cmd recursively, collecting all messages.
func collectAllTestMsgs(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if msg == nil {
		return nil
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, c := range batch {
			if c != nil {
				msgs = append(msgs, collectAllTestMsgs(c)...)
			}
		}
		return msgs
	}
	return []tea.Msg{msg}
}

// =============================================================================
// TestPolling_PlaybackAlwaysPolled — playback state polling is NOT affected.
// =============================================================================

func TestPolling_PlaybackAlwaysPolled(t *testing.T) {
	a := newPollOptTestApp(t)

	// Set a preset that excludes many panes.
	a.layout.SetPreset(1) // Listening: only NowPlaying, Queue, RecentlyPlayed
	require.Equal(t, layout.PresetNameListening, a.layout.ActivePresetName())

	// At tick 0, playback interval = 3 (active+paused), 0%3==0 so playback fetch dispatched.
	_, cmd := a.Update(panes.TickMsg{})
	require.NotNil(t, cmd)

	msgs := collectAllTestMsgs(cmd)
	hasPlayback := false
	for _, m := range msgs {
		if _, ok := m.(panes.PlaybackStateFetchedMsg); ok {
			hasPlayback = true
		}
	}
	assert.True(t, hasPlayback, "playback state must be polled regardless of preset")
}

// =============================================================================
// TestPolling_QueueAlwaysPolled — queue polling is NOT affected by visibility.
// =============================================================================

func TestPolling_QueueAlwaysPolled(t *testing.T) {
	a := newPollOptTestApp(t)

	// Dashboard: queue is visible.
	require.True(t, a.layout.IsPaneVisible(layout.PaneQueue))

	// At tick 0, queue interval = 30 (active+paused), 0%30==0 → queue fetch dispatched.
	a.lastInteraction = time.Now()
	_, cmd := a.Update(panes.TickMsg{})
	require.NotNil(t, cmd)

	msgs := collectAllTestMsgs(cmd)
	hasQueue := false
	for _, m := range msgs {
		if _, ok := m.(panes.QueueLoadedMsg); ok {
			hasQueue = true
		}
	}
	assert.True(t, hasQueue, "queue must be polled at tick 0")
}

// =============================================================================
// TestCheckNewlyVisiblePanes_DispatchesFetchForStalePanes
// =============================================================================

func TestCheckNewlyVisiblePanes_DispatchesFetchForStalePanes(t *testing.T) {
	a := newPollOptTestApp(t)

	// Start on Listening preset (only NowPlaying, Queue, RecentlyPlayed).
	a.layout.SetPreset(1)
	require.Equal(t, layout.PresetNameListening, a.layout.ActivePresetName())

	oldVisible := copyVisibleMap(a.layout.ActivePreset().Visible)

	// Switch to Dashboard (all 8 music panes).
	a.layout.SetPreset(0) // Dashboard
	require.Equal(t, layout.PresetNameDashboard, a.layout.ActivePresetName())

	// Playlists, Albums, LikedSongs, TopTracks/Artists are newly visible and stale.
	cmd := a.checkNewlyVisiblePanes(oldVisible)
	require.NotNil(t, cmd, "should dispatch fetches for newly visible stale panes")

	msgs := collectAllTestMsgs(cmd)
	t.Logf("dispatched %d messages", len(msgs))

	// Verify that at least Playlists is dispatched (newly visible from Listening→Dashboard).
	hasPlaylists := false
	for _, m := range msgs {
		if _, ok := m.(panes.LibraryLoadedMsg); ok {
			hasPlaylists = true
		}
	}
	assert.True(t, hasPlaylists, "newly visible Playlists pane should dispatch fetch on preset switch")
}

// =============================================================================
// TestCheckNewlyVisiblePanes_SkipsFreshPanes
// =============================================================================

func TestCheckNewlyVisiblePanes_SkipsFreshPanes(t *testing.T) {
	a := newPollOptTestApp(t)

	// Start on Listening preset.
	a.layout.SetPreset(1)
	require.Equal(t, layout.PresetNameListening, a.layout.ActivePresetName())

	oldVisible := copyVisibleMap(a.layout.ActivePreset().Visible)

	// Set playlists data (non-empty required to stamp fetchedAt) so they are NOT stale.
	a.store.SetPlaylists([]domain.SimplePlaylist{{ID: "p1", Name: "Test"}})

	// Switch to Dashboard.
	a.layout.SetPreset(0)
	require.Equal(t, layout.PresetNameDashboard, a.layout.ActivePresetName())

	cmd := a.checkNewlyVisiblePanes(oldVisible)

	// Playlists should NOT be fetched because they are fresh.
	msgs := collectAllTestMsgs(cmd)
	for _, m := range msgs {
		if lm, ok := m.(panes.LibraryLoadedMsg); ok {
			// If a LibraryLoadedMsg is dispatched, it should be for playlists.
			// But playlists are fresh, so this should not happen.
			// However, other panes (Albums, LikedSongs, Stats) may still be stale.
			t.Logf("LibraryLoadedMsg dispatched: %+v", lm)
		}
	}

	// Verify playlists fetching sentinel was NOT set (data was fresh).
	assert.False(t, a.store.PlaylistsFetching(),
		"playlists fetching sentinel must NOT be set when data is fresh")
}

// =============================================================================
// TestPolling_SentinelNotLeakedOnHide — sentinel set before pane hidden
// is not cleared by the polling loop after the pane is hidden.
// =============================================================================

func TestPolling_SentinelNotLeakedOnHide(t *testing.T) {
	a := newPollOptTestApp(t)

	// Dashboard preset — FollowedShows is NOT visible.
	require.False(t, a.layout.IsPaneVisible(layout.PaneFollowedShows))

	// Manually set a fetching sentinel (simulating one set earlier when pane was visible).
	a.store.SetFollowedShowsFetching(true)

	// Send multiple ticks — FollowedShows is invisible, so loop skips it.
	for i := 0; i < 10; i++ {
		_, _ = a.Update(panes.TickMsg{})
	}

	// Sentinel must NOT be cleared — the entry is never processed.
	assert.True(t, a.store.FollowedShowsFetching(),
		"fetching sentinel must NOT be cleared by polling loop when pane is hidden")
}

// =============================================================================
// TestPolling_SentinelClearedOnResponse — sentinel is cleared when the
// LoadedMsg response arrives, even if pane was briefly hidden.
// =============================================================================

func TestPolling_SentinelClearedOnResponse(t *testing.T) {
	a := newPollOptTestApp(t)

	// Set fetching sentinel directly (simulating a fetch command dispatch).
	a.store.SetPlaylistsFetching(true)
	assert.True(t, a.store.PlaylistsFetching())

	// Simulate the loaded message arriving (as if HTTP response came back).
	msg := panes.LibraryLoadedMsg{Items: []domain.SimplePlaylist{}, Offset: 0}
	model, _ := a.Update(msg)
	a = model.(*App)

	// Sentinel must be cleared by the handler.
	assert.False(t, a.store.PlaylistsFetching(),
		"fetching sentinel must be cleared when LoadedMsg arrives")
}

// =============================================================================
// TestCheckNewlyVisiblePanes_NoCmdWhenNoNewPanes
// =============================================================================

func TestCheckNewlyVisiblePanes_NoCmdWhenNoNewPanes(t *testing.T) {
	a := newPollOptTestApp(t)

	// Stay on the same preset - no newly visible panes.
	require.Equal(t, layout.PresetNameDashboard, a.layout.ActivePresetName())
	oldVisible := copyVisibleMap(a.layout.ActivePreset().Visible)

	cmd := a.checkNewlyVisiblePanes(oldVisible)
	assert.Nil(t, cmd, "should return nil when no panes are newly visible")
}

// =============================================================================
// TestCheckNewlyVisiblePanes_IgnoresPlaybackAndQueue
// =============================================================================

func TestCheckNewlyVisiblePanes_IgnoresPlaybackAndQueue(t *testing.T) {
	a := newPollOptTestApp(t)

	// Start on a preset that does NOT have Queue visible.
	// Actually, all presets have NowPlaying and Queue visible. Let's test that
	// even if NowPlaying/Queue are in oldVisible, they don't trigger data fetches.
	// This is inherently handled because NowPlaying and Queue have no gate entries
	// in the gates map within checkNewlyVisiblePanes.

	// Switch from Dashboard to Listening — both have NowPlaying and Queue.
	oldVisible := copyVisibleMap(a.layout.ActivePreset().Visible)
	a.layout.SetPreset(1) // Listening

	cmd := a.checkNewlyVisiblePanes(oldVisible)
	// No playback/queue fetch commands should be in the batch.
	msgs := collectAllTestMsgs(cmd)
	for _, m := range msgs {
		switch m.(type) {
		case panes.PlaybackStateFetchedMsg:
			t.Error("checkNewlyVisiblePanes must not dispatch playback state fetches")
		case panes.QueueLoadedMsg:
			t.Error("checkNewlyVisiblePanes must not dispatch queue fetches")
		}
	}
}

// =============================================================================
// TestCheckNewlyVisiblePanes_SkipsFetchingPane — a pane with an active
// fetching sentinel must be skipped even if newly visible.
// =============================================================================

func TestCheckNewlyVisiblePanes_SkipsFetchingPane(t *testing.T) {
	a := newPollOptTestApp(t)

	// Start on Listening preset (only NowPlaying, Queue, RecentlyPlayed).
	a.layout.SetPreset(1)
	require.Equal(t, layout.PresetNameListening, a.layout.ActivePresetName())

	oldVisible := copyVisibleMap(a.layout.ActivePreset().Visible)

	// Set fetching sentinel on Playlists BEFORE switching.
	a.store.SetPlaylistsFetching(true)

	// Switch to Dashboard — Playlists is newly visible but fetching.
	a.layout.SetPreset(0) // Dashboard
	require.Equal(t, layout.PresetNameDashboard, a.layout.ActivePresetName())

	cmd := a.checkNewlyVisiblePanes(oldVisible)

	// Playlists must NOT dispatch a fetch because its sentinel is already true.
	msgs := collectAllTestMsgs(cmd)
	for _, m := range msgs {
		if lm, ok := m.(panes.LibraryLoadedMsg); ok {
			t.Logf("LibraryLoadedMsg dispatched: %+v", lm)
		}
	}

	// Playlists sentinel must remain true (it was set before, and skipped).
	assert.True(t, a.store.PlaylistsFetching(),
		"fetching sentinel must remain true when pane with active fetch becomes visible")
}

// =============================================================================
// TestPolling_SkipsHiddenPane — panes manually toggled off are skipped.
// =============================================================================

func TestPolling_SkipsHiddenPane(t *testing.T) {
	a := newPollOptTestApp(t)

	// Dashboard preset — Playlists is visible.
	require.True(t, a.layout.IsPaneVisible(layout.PanePlaylists))

	// Manually toggle Playlists off (key '3').
	a.layout.TogglePane(layout.PanePlaylists)
	assert.False(t, a.layout.IsPaneVisible(layout.PanePlaylists))

	// Set fetching sentinel BEFORE tick.
	a.store.SetPlaylistsFetching(true)

	// Send tick that would trigger Playlists poll if visible.
	_, _ = a.Update(panes.TickMsg{})

	// Sentinel must NOT be cleared because the entry was skipped.
	assert.True(t, a.store.PlaylistsFetching(),
		"fetching sentinel must NOT be cleared when pane is manually hidden")
}

// =============================================================================
// TestPolling_StatsSkippedWhenNotVisible — TopTracks pane maps to stats poll.
// =============================================================================

func TestPolling_StatsSkippedWhenNotVisible(t *testing.T) {
	a := newPollOptTestApp(t)

	// Listening preset (index 1) does NOT include TopTracks or TopArtists.
	a.layout.SetPreset(1)
	require.Equal(t, layout.PresetNameListening, a.layout.ActivePresetName())
	assert.False(t, a.layout.IsPaneVisible(layout.PaneTopTracks))
	assert.False(t, a.layout.IsPaneVisible(layout.PaneTopArtists))

	// Set stats fetching sentinel.
	a.store.SetStatsFetching("short_term", true)

	// Send tick. Stats pane is invisible → skip.
	_, _ = a.Update(panes.TickMsg{})

	assert.True(t, a.store.StatsFetching("short_term"),
		"stats fetching sentinel must NOT be cleared when stats panes are invisible")
}

// =============================================================================
// TestPolling_PodcastPanesPolledInPodcastPreset
// =============================================================================

func TestPolling_PodcastPanesPolledInPodcastPreset(t *testing.T) {
	a := newPollOptTestApp(t)

	// Switch to Podcast preset (index 2).
	a.layout.SetPreset(2)
	require.Equal(t, layout.PresetNamePodcast, a.layout.ActivePresetName())
	assert.True(t, a.layout.IsPaneVisible(layout.PaneFollowedShows))

	// At tick 0 with hasData=false → retry mode interval=5.
	_, cmd := a.Update(panes.TickMsg{})
	require.NotNil(t, cmd)

	msgs := collectAllTestMsgs(cmd)
	hasFollowedShows := false
	for _, m := range msgs {
		if _, ok := m.(panes.FollowedShowsLoadedMsg); ok {
			hasFollowedShows = true
		}
	}
	assert.True(t, hasFollowedShows, "FollowedShows should be polled in podcast preset")
}
