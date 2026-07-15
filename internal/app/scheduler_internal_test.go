package app

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newSchedulerTestApp creates an App in grid view with a deterministic layout.
func newSchedulerTestApp(t *testing.T) *App {
	t.Helper()
	a := newTestAppInternal()
	model, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a = model.(*App)
	a.currentView = viewGrid
	a.store.SetPlaybackState(&domain.PlaybackState{
		IsPlaying: false,
		Item:      &domain.Track{ID: "t1", Name: "Test"},
	})
	a.store.SetQueue([]domain.QueueItem{})
	return a
}

func setAllLibraryPollsFresh(a *App) {
	fresh := []*pollState{
		&a.playlistsPoll, &a.albumsPoll, &a.likedSongsPoll,
		&a.recentPlayedPoll, &a.statsPoll, &a.followedShowsPoll, &a.savedEpisodesPoll,
	}
	for _, p := range fresh {
		p.lastSuccessTick = a.tickCount
		p.hasData = true
		p.hasSuccess = true
	}
}

func TestApp_PickMostOverdue_OverdueWins(t *testing.T) {
	a := newSchedulerTestApp(t)
	a.tickCount = 1000
	setAllLibraryPollsFresh(a)

	// Make Playlists far overdue; others are fresh.
	a.playlistsPoll.lastSuccessTick = 0

	best := a.pickMostOverdueLibraryPane()
	require.NotNil(t, best)
	assert.Equal(t, layout.PanePlaylists, best.paneID)
}

func TestApp_PickMostOverdue_SkipsHidden(t *testing.T) {
	a := newSchedulerTestApp(t)
	// Listening preset hides Playlists, Albums, LikedSongs.
	a.layout.SetPreset(1)
	a.tickCount = 1000
	setAllLibraryPollsFresh(a)
	// Make Playlists overdue but hidden; make RecentlyPlayed overdue and visible.
	a.playlistsPoll.lastSuccessTick = 0
	a.recentPlayedPoll.lastSuccessTick = 0

	best := a.pickMostOverdueLibraryPane()
	require.NotNil(t, best)
	assert.Equal(t, layout.PaneRecentlyPlayed, best.paneID)
	assert.NotEqual(t, layout.PanePlaylists, best.paneID)
}

func TestApp_PickMostOverdue_SkipsBackoff(t *testing.T) {
	a := newSchedulerTestApp(t)
	a.tickCount = 1000
	setAllLibraryPollsFresh(a)
	// Playlists is in backoff and overdue; Albums is overdue and not in backoff.
	a.playlistsPoll.lastSuccessTick = 0
	a.playlistsPoll.backoffTicks = 5
	a.albumsPoll.lastSuccessTick = 0

	best := a.pickMostOverdueLibraryPane()
	require.NotNil(t, best)
	assert.Equal(t, layout.PaneAlbums, best.paneID)
	assert.NotEqual(t, layout.PanePlaylists, best.paneID)
	assert.Equal(t, 4, a.playlistsPoll.backoffTicks, "backoff should be decremented")
}

func TestApp_PickMostOverdue_UsesSuccessTick(t *testing.T) {
	a := newSchedulerTestApp(t)
	a.tickCount = 1000
	setAllLibraryPollsFresh(a)
	// Albums was dispatched recently but its last successful fetch was long ago.
	a.albumsPoll.lastDispatchedTick = 999
	a.albumsPoll.lastSuccessTick = 10

	a.playlistsPoll.lastDispatchedTick = 500
	a.playlistsPoll.lastSuccessTick = 950

	best := a.pickMostOverdueLibraryPane()
	require.NotNil(t, best)
	assert.Equal(t, layout.PaneAlbums, best.paneID)
}

func TestApp_Tick_DispatchesAtMostOneLibraryPane(t *testing.T) {
	a := newSchedulerTestApp(t)
	_, cmd := a.Update(panes.TickMsg{})
	require.NotNil(t, cmd)
	msgs := collectAllTestMsgs(cmd)
	count := 0
	for _, m := range msgs {
		switch m.(type) {
		case panes.LibraryLoadedMsg, panes.AlbumsLoadedMsg, panes.LikedTracksLoadedMsg,
			panes.RecentlyPlayedLoadedMsg, panes.StatsLoadedMsg,
			panes.FollowedShowsLoadedMsg, panes.SavedEpisodesLoadedMsg:
			count++
		}
	}
	assert.LessOrEqual(t, count, 1, "TickMsg must dispatch at most one library pane")
}

func triggerBackoff(g *api.Gateway) {
	_, _ = g.Do(context.Background(), api.Background,
		api.RequestKey{Method: "GET", Path: "/test-backoff", Priority: api.Background},
		func() (*http.Response, error) {
			resp := &http.Response{StatusCode: 429, Header: make(http.Header)}
			resp.Header.Set("Retry-After", "60")
			resp.Body = io.NopCloser(bytes.NewReader(nil))
			return resp, nil
		})
}

func TestApp_Tick_SchedulerRespectsCanAdmit(t *testing.T) {
	a := newSchedulerTestApp(t)
	triggerBackoff(a.gateway)
	require.True(t, a.gateway.IsThrottled())

	_, cmd := a.Update(panes.TickMsg{})
	msgs := collectAllTestMsgs(cmd)
	for _, m := range msgs {
		switch m.(type) {
		case panes.LibraryLoadedMsg, panes.AlbumsLoadedMsg, panes.LikedTracksLoadedMsg,
			panes.RecentlyPlayedLoadedMsg, panes.StatsLoadedMsg,
			panes.FollowedShowsLoadedMsg, panes.SavedEpisodesLoadedMsg:
			t.Fatal("scheduler must not dispatch library pane when CanAdmit is false")
		}
	}
}

func TestApp_Tick_PausesLibraryOnLongIdle(t *testing.T) {
	a := newSchedulerTestApp(t)
	a.lastInteraction = time.Now().Add(-2 * api.LongIdleThreshold)

	_, cmd := a.Update(panes.TickMsg{})
	msgs := collectAllTestMsgs(cmd)
	for _, m := range msgs {
		switch m.(type) {
		case panes.LibraryLoadedMsg, panes.AlbumsLoadedMsg, panes.LikedTracksLoadedMsg,
			panes.RecentlyPlayedLoadedMsg, panes.StatsLoadedMsg,
			panes.FollowedShowsLoadedMsg, panes.SavedEpisodesLoadedMsg:
			t.Fatal("library polling must pause when isLongIdle is true")
		}
	}
}

func TestApp_IsLongIdle(t *testing.T) {
	a := newSchedulerTestApp(t)
	a.lastInteraction = time.Now()
	assert.False(t, a.isLongIdle())

	a.lastInteraction = time.Now().Add(-2 * api.LongIdleThreshold)
	assert.True(t, a.isLongIdle())
}

func TestApp_LongIdlePausesLibraryPolling(t *testing.T) {
	a := newSchedulerTestApp(t)
	a.lastInteraction = time.Now().Add(-2 * api.LongIdleThreshold)
	assert.True(t, a.isLongIdle())

	_, cmd := a.Update(panes.TickMsg{})
	require.NotNil(t, cmd)
	msgs := collectAllTestMsgs(cmd)
	for _, m := range msgs {
		switch m.(type) {
		case panes.LibraryLoadedMsg, panes.AlbumsLoadedMsg, panes.LikedTracksLoadedMsg,
			panes.RecentlyPlayedLoadedMsg, panes.StatsLoadedMsg,
			panes.FollowedShowsLoadedMsg, panes.SavedEpisodesLoadedMsg:
			t.Fatal("library polling must not run during long idle")
		}
	}
}

func TestApp_CheckNewlyVisiblePanes_StopsWhenCanAdmitFalse(t *testing.T) {
	a := newSchedulerTestApp(t)
	// Switch to a preset with many visible library panes.
	a.layout.SetPreset(0) // Dashboard
	a.propagateSizes()
	a.syncFocus()

	triggerBackoff(a.gateway)
	require.True(t, a.gateway.IsThrottled())

	oldVisible := map[layout.PaneID]bool{}
	cmd := a.checkNewlyVisiblePanes(oldVisible)
	assert.Nil(t, cmd, "checkNewlyVisiblePanes must stop when CanAdmit returns false")
}
