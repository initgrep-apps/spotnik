package app_test

// elm_purity_test.go — Tests for Feature 29: Elm Purity Data-Carrying Messages
//
// Task 1: PlaybackStateFetchedMsg and QueueLoadedMsg carry data payloads.
//         Store writes move to Update() — commands do NOT touch the store.
// Task 2: Library messages carry data payloads.
// Task 3: Stats/Search/Devices/Playlist messages carry data payloads.
// Task 4: fetchPlaybackStateCmd and fetchQueueCmd have no store parameter.

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// playbackServer returns a server that responds with a playback state JSON body.
func playbackServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"is_playing": true,
			"progress_ms": 30000,
			"shuffle_state": false,
			"repeat_state": "off",
			"item": {"id": "t1", "name": "Test Track", "uri": "spotify:track:t1", "duration_ms": 200000, "artists": [{"name": "Test Artist"}]},
			"device": {"id": "d1", "name": "MacBook", "type": "Computer", "volume_percent": 65}
		}`))
	}))
}

// queueServer returns a server that responds with a queue JSON body.
func queueServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"currently_playing": {"id": "t1", "name": "Current", "uri": "spotify:track:t1"},
			"queue": [
				{"id": "t2", "name": "Next Song", "uri": "spotify:track:t2"},
				{"id": "t3", "name": "Song After", "uri": "spotify:track:t3"}
			]
		}`))
	}))
}

// --- Task 1: PlaybackStateFetchedMsg carries data ---

// TestFetchPlaybackStateCmd_ReturnsPlaybackInMsg verifies that the playback command
// returns the fetched state in the PlaybackStateFetchedMsg payload (not via store write).
func TestFetchPlaybackStateCmd_ReturnsPlaybackInMsg(t *testing.T) {
	srv := playbackServer()
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetPlayer(api.NewPlayer(srv.URL, "test-token"))

	// Trigger a tick to fire fetchPlaybackStateCmd
	_, cmd := a.Update(panes.TickMsg{})
	require.NotNil(t, cmd)

	// Execute all commands in the batch, find PlaybackStateFetchedMsg
	msg := cmd()
	var fetchedMsg panes.PlaybackStateFetchedMsg
	found := false
	// cmd() may return a BatchMsg — try executing it
	if batchCmd, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batchCmd {
			m := c()
			if fm, ok := m.(panes.PlaybackStateFetchedMsg); ok {
				fetchedMsg = fm
				found = true
				break
			}
		}
	} else if fm, ok := msg.(panes.PlaybackStateFetchedMsg); ok {
		fetchedMsg = fm
		found = true
	}

	// Not finding it in a batch is OK — just run the full Update chain
	if !found {
		// Use a direct app with the real player to get the cmd
		a2 := app.New(cfg, app.AppOptions{})
		a2.SetPlayer(api.NewPlayer(srv.URL, "test-token"))
		m, err := runFetchPlaybackCmd(a2)
		require.NoError(t, err)
		fetchedMsg = m
	}

	assert.NotNil(t, fetchedMsg.State, "PlaybackStateFetchedMsg.State should be populated by command")
	assert.Equal(t, "t1", fetchedMsg.State.Item.ID)
}

// TestUpdate_PlaybackStateFetchedMsg_WritesStore verifies that when Update() receives
// a PlaybackStateFetchedMsg with a State payload, it writes it to the store.
func TestUpdate_PlaybackStateFetchedMsg_WritesStore(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	state := &domain.PlaybackState{
		IsPlaying:  true,
		ProgressMs: 45000,
		Item:       &domain.Track{ID: "t1", Name: "Song", DurationMs: 200000},
		Device:     &domain.Device{VolumePercent: 80},
	}

	// Send data-carrying message to Update()
	msg := panes.PlaybackStateFetchedMsg{State: state}
	_, _ = a.Update(msg)

	got := a.Store().PlaybackState()
	require.NotNil(t, got, "store should be updated by Update() when PlaybackStateFetchedMsg carries State")
	assert.Equal(t, "t1", got.Item.ID)
	assert.Equal(t, 45000, got.ProgressMs)
}

// TestUpdate_PlaybackStateFetchedMsg_NilState_NoStoreMutation verifies that a
// PlaybackStateFetchedMsg with nil State and nil Err does not crash or clear the store.
func TestUpdate_PlaybackStateFetchedMsg_NilState_NoStoreMutation(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Pre-populate store
	a.Store().SetPlaybackState(&domain.PlaybackState{IsPlaying: true, ProgressMs: 10000})

	// Empty message (nil State, nil Err) — should not change store
	msg := panes.PlaybackStateFetchedMsg{}
	_, _ = a.Update(msg)

	// Store still has the previous value
	got := a.Store().PlaybackState()
	require.NotNil(t, got)
	assert.Equal(t, 10000, got.ProgressMs)
}

// TestUpdate_PlaybackStateFetchedMsg_WithErr_NoStoreMutation verifies that a
// PlaybackStateFetchedMsg with an error does not crash and does not modify the store.
func TestUpdate_PlaybackStateFetchedMsg_WithErr_NoStoreMutation(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	a.Store().SetPlaybackState(&domain.PlaybackState{IsPlaying: true})

	msg := panes.PlaybackStateFetchedMsg{Err: errors.New("network error")}
	_, _ = a.Update(msg)

	// Store should still have original value
	assert.NotNil(t, a.Store().PlaybackState())
}

// TestUpdate_QueueLoadedMsg_WritesStore verifies that QueueLoadedMsg with items
// causes Update() to write to the store.
func TestUpdate_QueueLoadedMsg_WritesStore(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	items := []domain.QueueItem{
		{Type: domain.QueueItemTypeTrack, Track: &domain.Track{ID: "q1", Name: "First in Queue"}},
		{Type: domain.QueueItemTypeTrack, Track: &domain.Track{ID: "q2", Name: "Second in Queue"}},
	}

	msg := panes.QueueLoadedMsg{Items: items}
	_, _ = a.Update(msg)

	got := a.Store().Queue()
	require.Len(t, got, 2)
	assert.Equal(t, "q1", got[0].Track.ID)
	assert.Equal(t, "First in Queue", got[0].Track.Name)
}

// TestUpdate_QueueLoadedMsg_WithErr_WritesQueueError verifies that QueueLoadedMsg
// with an error causes Update() to write the error to the store.
func TestUpdate_QueueLoadedMsg_WithErr_WritesQueueError(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	qErr := errors.New("queue fetch failed")
	msg := panes.QueueLoadedMsg{Err: qErr}
	_, _ = a.Update(msg)

	assert.Equal(t, qErr, a.Store().QueueError())
}

// TestUpdate_QueueLoadedMsg_Success_ClearsQueueError verifies that a successful
// QueueLoadedMsg (no error) clears any previous queue error.
func TestUpdate_QueueLoadedMsg_Success_ClearsQueueError(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Set an error first
	a.Store().SetQueueError(errors.New("previous error"))

	items := []domain.QueueItem{
		{Type: domain.QueueItemTypeTrack, Track: &domain.Track{ID: "q1", Name: "Track"}},
	}
	msg := panes.QueueLoadedMsg{Items: items}
	_, _ = a.Update(msg)

	assert.Nil(t, a.Store().QueueError(), "QueueError should be cleared on successful load")
}

// TestFetchQueueCmd_ReturnsTracksInMsg verifies that fetchQueueCmd returns
// queue items in the QueueLoadedMsg payload.
func TestFetchQueueCmd_ReturnsTracksInMsg(t *testing.T) {
	srv := queueServer()
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetPlayer(api.NewPlayer(srv.URL, "test-token"))

	m, err := runFetchQueueCmd(a)
	require.NoError(t, err, "should be able to extract QueueLoadedMsg from command chain")
	assert.Len(t, m.Items, 2, "QueueLoadedMsg.Items should contain the fetched items")
	assert.Equal(t, "t2", m.Items[0].Track.ID)
	assert.Equal(t, "Next Song", m.Items[0].Track.Name)
}

// --- Task 2: Library messages carry data ---

// TestUpdate_LibraryLoadedMsg_WritesPlaylists verifies that LibraryLoadedMsg with
// items causes Update() to write to the store.
func TestUpdate_LibraryLoadedMsg_WritesPlaylists(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	items := []domain.SimplePlaylist{
		{ID: "pl1", Name: "Chill Vibes"},
		{ID: "pl2", Name: "Workout Mix"},
	}

	// Offset 0: full replace
	msg := panes.LibraryLoadedMsg{Items: items, Offset: 0}
	_, _ = a.Update(msg)

	got := a.Store().Playlists()
	require.Len(t, got, 2)
	assert.Equal(t, "pl1", got[0].ID)
}

// TestUpdate_LibraryLoadedMsg_Pagination_AppendsBeyondOffset0 verifies that
// LibraryLoadedMsg with Offset > 0 appends to existing playlists.
func TestUpdate_LibraryLoadedMsg_Pagination_AppendsBeyondOffset0(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// First page
	first := []domain.SimplePlaylist{{ID: "pl1", Name: "First"}}
	_, _ = a.Update(panes.LibraryLoadedMsg{Items: first, Offset: 0})
	require.Len(t, a.Store().Playlists(), 1)

	// Second page
	second := []domain.SimplePlaylist{{ID: "pl2", Name: "Second"}}
	_, _ = a.Update(panes.LibraryLoadedMsg{Items: second, Offset: 1})

	got := a.Store().Playlists()
	require.Len(t, got, 2, "second page should be appended")
	assert.Equal(t, "pl1", got[0].ID)
	assert.Equal(t, "pl2", got[1].ID)
}

// TestUpdate_LibraryLoadedMsg_WithErr_WritesPlaylistsFetchError verifies error handling.
func TestUpdate_LibraryLoadedMsg_WithErr_WritesPlaylistsFetchError(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	libErr := errors.New("playlists fetch failed")
	_, _ = a.Update(panes.LibraryLoadedMsg{Err: libErr})

	assert.Equal(t, libErr, a.Store().PlaylistsFetchError())
}

// TestUpdate_AlbumsLoadedMsg_WritesAlbums verifies AlbumsLoadedMsg causes store write.
func TestUpdate_AlbumsLoadedMsg_WritesAlbums(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	albums := []domain.SavedAlbum{
		{AddedAt: "2024-01-01", Album: domain.FullAlbum{ID: "al1", Name: "After Hours"}},
	}

	_, _ = a.Update(panes.AlbumsLoadedMsg{Items: albums})

	got := a.Store().SavedAlbums()
	require.Len(t, got, 1)
	assert.Equal(t, "al1", got[0].Album.ID)
}

// TestUpdate_AlbumsLoadedMsg_WithErr_WritesError verifies error handling for albums.
func TestUpdate_AlbumsLoadedMsg_WithErr_WritesError(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	albumErr := errors.New("albums error")
	_, _ = a.Update(panes.AlbumsLoadedMsg{Err: albumErr})

	assert.Equal(t, albumErr, a.Store().AlbumsFetchError())
}

// TestUpdate_LikedTracksLoadedMsg_WritesLikedTracks verifies LikedTracksLoadedMsg.
func TestUpdate_LikedTracksLoadedMsg_WritesLikedTracks(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	tracks := []domain.SavedTrack{
		{AddedAt: "2024-02-01", Track: domain.Track{ID: "t1", Name: "Liked Track"}},
	}

	_, _ = a.Update(panes.LikedTracksLoadedMsg{Items: tracks, Offset: 0})

	got := a.Store().LikedTracks()
	require.Len(t, got, 1)
	assert.Equal(t, "t1", got[0].Track.ID)
}

// TestUpdate_RecentlyPlayedLoadedMsg_WritesRecentlyPlayed verifies the message.
func TestUpdate_RecentlyPlayedLoadedMsg_WritesRecentlyPlayed(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	items := []domain.PlayHistory{
		{PlayedAt: "2024-03-01", Track: domain.Track{ID: "t1", Name: "Recent Track"}},
	}

	_, _ = a.Update(panes.RecentlyPlayedLoadedMsg{Items: items})

	got := a.Store().RecentlyPlayed()
	require.Len(t, got, 1)
	assert.Equal(t, "t1", got[0].Track.ID)
}

// --- Task 3: Stats/Search/Devices/Playlist messages carry data ---

// TestUpdate_StatsLoadedMsg_WritesTopTracksAndArtists verifies that StatsLoadedMsg
// with data causes Update() to write to the store.
func TestUpdate_StatsLoadedMsg_WritesTopTracksAndArtists(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	tracks := []domain.Track{{ID: "t1", Name: "Top Track"}}
	artists := []domain.FullArtist{{ID: "a1", Name: "Top Artist"}}

	msg := panes.StatsLoadedMsg{
		TimeRange:  "short_term",
		TopTracks:  tracks,
		TopArtists: artists,
	}
	// Need to open stats pane for the message to be handled
	_, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	_, _ = a.Update(msg)

	gotTracks := a.Store().TopTracks("short_term")
	gotArtists := a.Store().TopArtists("short_term")
	require.Len(t, gotTracks, 1)
	assert.Equal(t, "t1", gotTracks[0].ID)
	require.Len(t, gotArtists, 1)
	assert.Equal(t, "a1", gotArtists[0].ID)
}

// TestUpdate_StatsLoadedMsg_WithErr_WritesStatsError verifies error handling.
func TestUpdate_StatsLoadedMsg_WithErr_WritesStatsError(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	statsErr := errors.New("stats error")
	msg := panes.StatsLoadedMsg{TimeRange: "short_term", Err: statsErr}
	_, _ = a.Update(msg)

	assert.Equal(t, statsErr, a.Store().StatsError())
}

// TestUpdate_PlaylistTracksLoadedMsg_ForwardsToPaneAndNotStore verifies that
// PlaylistTracksLoadedMsg is forwarded to the PlaylistsPane (pane-owned data)
// and NOT written to the global store (new architecture from story 106).
func TestUpdate_PlaylistTracksLoadedMsg_ForwardsToPaneAndNotStore(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	tracks := []domain.Track{
		{ID: "t1", Name: "Track One"},
		{ID: "t2", Name: "Track Two"},
	}

	// Set the staleness key so the message is not discarded.
	a.SetPlaylistTracksID("pl1")

	msg := panes.PlaylistTracksLoadedMsg{PlaylistID: "pl1", Tracks: tracks, Total: 2, Offset: 0}
	_, _ = a.Update(msg)

	// Data must NOT be written to the global store (pane owns it now).
	got := a.Store().PlaylistTracks("pl1")
	assert.Empty(t, got, "playlist tracks must no longer be written to the store")
}

// TestUpdate_PlaylistTracksLoadedMsg_StaleMsg_Discarded verifies that a stale
// PlaylistTracksLoadedMsg (mismatched PlaylistID) is silently discarded.
func TestUpdate_PlaylistTracksLoadedMsg_StaleMsg_Discarded(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	a.SetPlaylistTracksID("current-playlist")

	// Deliver a message for a different playlist.
	msg := panes.PlaylistTracksLoadedMsg{PlaylistID: "old-playlist", Tracks: []domain.Track{{ID: "t1"}}}
	_, cmd := a.Update(msg)
	assert.Nil(t, cmd, "stale PlaylistTracksLoadedMsg must be silently discarded")
}

// TestUpdate_PlaylistTracksLoadedMsg_WithErr_EmitsToast verifies that an error in
// PlaylistTracksLoadedMsg triggers a toast notification (not a store write).
func TestUpdate_PlaylistTracksLoadedMsg_WithErr_EmitsToast(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	a.SetPlaylistTracksID("pl1")

	plErr := errors.New("playlist tracks error")
	msg := panes.PlaylistTracksLoadedMsg{PlaylistID: "pl1", Err: plErr}
	_, cmd := a.Update(msg)

	// Must emit a toast command, not write to store.
	assert.NotNil(t, cmd, "error in PlaylistTracksLoadedMsg must emit a toast cmd")
	assert.Equal(t, nil, a.Store().PlaylistsError(), "error must NOT be written to the store")
}

// --- Task 4: Store param removed from package-level functions ---

// TestBuildStatsCmd_RunsConcurrently verifies that buildFetchStatsCmd fetches
// both tracks and artists and returns them in a single StatsLoadedMsg.
func TestBuildStatsCmd_RunsConcurrently(t *testing.T) {
	srv := successServer()
	defer srv.Close()

	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.SetUserAPI(api.NewUserClient(srv.URL, "test-token"))

	// Send FetchStatsMsg to trigger buildFetchStatsCmd
	_, cmd := a.Update(panes.FetchStatsMsg{TimeRange: "short_term"})
	require.NotNil(t, cmd)

	msg := cmd()
	statsMsg, ok := msg.(panes.StatsLoadedMsg)
	require.True(t, ok, "buildFetchStatsCmd should return StatsLoadedMsg, got %T", msg)
	assert.Equal(t, "short_term", statsMsg.TimeRange)
	// TopTracks and TopArtists are empty arrays (success server returns empty)
	assert.NotNil(t, statsMsg.TopTracks)
	assert.NotNil(t, statsMsg.TopArtists)
}

// --- Feature 39: Additional Elm purity and coverage gap tests ---

// TestStatsLoadedMsg_PartialFailure verifies that when a StatsLoadedMsg carries
// an error, the store does not update with stale data and an error toast is emitted.
// This covers the partial-failure case: TopTracks may be populated but if Err is set,
// the store should not be stamped and the pane should receive an error signal.
func TestStatsLoadedMsg_PartialFailure(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Open the stats pane so the StatsLoadedMsg is forwarded to it.
	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	a = m.(*app.App)

	tracks := []domain.Track{{ID: "t1", Name: "Top Track"}}
	// Partial failure: TopTracks populated but Err set (TopArtists fetch failed).
	msg := panes.StatsLoadedMsg{
		TimeRange: "short_term",
		TopTracks: tracks,
		Err:       errors.New("artists fetch failed"),
	}
	_, cmd := a.Update(msg)

	// Store should NOT be updated when Err is set.
	assert.Nil(t, a.Store().TopTracks("short_term"),
		"TopTracks must not be written to store when StatsLoadedMsg.Err is set")
	assert.Nil(t, a.Store().TopArtists("short_term"),
		"TopArtists must not be written to store when StatsLoadedMsg.Err is set")

	// An error toast must be emitted.
	require.NotNil(t, cmd, "partial failure should emit an error toast cmd")
	alertMsg := cmd()
	_, _ = a.Update(alertMsg)
	assert.Contains(t, a.View(), "Failed to load stats", "error toast should mention stats failure")
}

// --- Helpers ---

// runFetchPlaybackCmd runs the app tick loop and extracts the PlaybackStateFetchedMsg.
func runFetchPlaybackCmd(a *app.App) (panes.PlaybackStateFetchedMsg, error) {
	_, cmd := a.Update(panes.TickMsg{})
	if cmd == nil {
		return panes.PlaybackStateFetchedMsg{}, errors.New("no command returned from tick")
	}
	return extractPlaybackMsg(cmd())
}

func extractPlaybackMsg(msg tea.Msg) (panes.PlaybackStateFetchedMsg, error) {
	switch m := msg.(type) {
	case panes.PlaybackStateFetchedMsg:
		return m, nil
	case tea.BatchMsg:
		for _, c := range m {
			result := c()
			if fm, ok := result.(panes.PlaybackStateFetchedMsg); ok {
				return fm, nil
			}
		}
	}
	return panes.PlaybackStateFetchedMsg{}, errors.New("PlaybackStateFetchedMsg not found in command output")
}

// runFetchQueueCmd executes the queue fetch command and returns the QueueLoadedMsg.
func runFetchQueueCmd(a *app.App) (panes.QueueLoadedMsg, error) {
	// Trigger queue fetch by simulating tick count at queue interval
	// We need to send enough ticks to hit the queue fetch interval (9 ticks)
	// OR we can use the app's Update to dispatch and extract directly.
	// The simpler approach: we know tick 0 fires both playback and queue.
	// But the first tick at count=0 fires both. Let's just run the tick.
	_, cmd := a.Update(panes.TickMsg{})
	if cmd == nil {
		return panes.QueueLoadedMsg{}, errors.New("no command from tick")
	}
	return extractQueueMsg(cmd())
}

func extractQueueMsg(msg tea.Msg) (panes.QueueLoadedMsg, error) {
	switch m := msg.(type) {
	case panes.QueueLoadedMsg:
		return m, nil
	case tea.BatchMsg:
		for _, c := range m {
			result := c()
			if qm, ok := result.(panes.QueueLoadedMsg); ok {
				return qm, nil
			}
		}
	}
	return panes.QueueLoadedMsg{}, errors.New("QueueLoadedMsg not found in command output")
}
