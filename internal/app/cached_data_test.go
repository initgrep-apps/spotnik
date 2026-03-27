package app_test

// cached_data_test.go — Tests for Task 6: Send cached data when staleness gate blocks
//
// Verifies that when a pane's Init() fires a fetch request and the data is already
// fresh (not stale), the staleness gate returns a synthetic loaded message with cached
// data instead of returning nil. This allows panes to initialize their display without
// waiting for a redundant API round-trip.

import (
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFetchPlaylistsRequest_WhenFresh_ReturnsCachedData verifies that when playlists are
// fresh (not stale), a FetchPlaylistsRequestMsg{Offset:0} returns a command that
// delivers cached playlists via a synthetic LibraryLoadedMsg.
func TestFetchPlaylistsRequest_WhenFresh_ReturnsCachedData(t *testing.T) {
	a := newTestApp()
	playlist := domain.SimplePlaylist{ID: "pl1", Name: "My Playlist"}
	a.Store().SetPlaylists([]domain.SimplePlaylist{playlist})
	// Playlists are now fresh (fetchedAt stamped by non-empty SetPlaylists).

	_, cmd := a.Update(panes.FetchPlaylistsRequestMsg{Offset: 0})
	require.NotNil(t, cmd, "FetchPlaylistsRequestMsg when fresh should return a synthetic cmd (not nil)")

	// Execute the command to get the synthetic message.
	msg := cmd()
	loaded, ok := msg.(panes.LibraryLoadedMsg)
	require.True(t, ok, "synthetic cmd should produce a LibraryLoadedMsg, got %T", msg)
	assert.Equal(t, 0, loaded.Offset, "synthetic LibraryLoadedMsg Offset should be 0")
	assert.Len(t, loaded.Items, 1, "synthetic LibraryLoadedMsg should carry cached playlists")
	assert.Equal(t, "pl1", loaded.Items[0].ID)
	assert.Nil(t, loaded.Err, "synthetic LibraryLoadedMsg should have nil Err")
}

// TestFetchPlaylistsRequest_WhenStale_DispatchesRealFetch verifies that when playlists
// are stale, FetchPlaylistsRequestMsg dispatches a real API fetch command (not synthetic).
func TestFetchPlaylistsRequest_WhenStale_DispatchesRealFetch(t *testing.T) {
	a := newTestApp()
	// Playlists have never been fetched → stale.

	_, cmd := a.Update(panes.FetchPlaylistsRequestMsg{Offset: 0})
	require.NotNil(t, cmd, "stale playlists should dispatch a fetch command")

	// The command should NOT return a LibraryLoadedMsg immediately (it would need a real API call).
	// We verify that it does NOT return a synthetic cached message.
	msg := cmd()
	_, isCached := msg.(panes.LibraryLoadedMsg)
	// A real API call returns LibraryLoadedMsg too, but with errNilClient since no client is set.
	// So we check that the result has an error (errNilClient), not a zero-error cached message.
	if isCached {
		loaded := msg.(panes.LibraryLoadedMsg)
		assert.NotNil(t, loaded.Err, "real fetch should produce a LibraryLoadedMsg with Err (no API client in test)")
	}
}

// TestFetchAlbumsRequest_WhenFresh_ReturnsCachedData verifies that albums staleness gate
// returns cached data when albums are fresh.
func TestFetchAlbumsRequest_WhenFresh_ReturnsCachedData(t *testing.T) {
	a := newTestApp()
	album := domain.SavedAlbum{AddedAt: "2024-01-01", Album: domain.FullAlbum{ID: "al1"}}
	a.Store().SetSavedAlbums([]domain.SavedAlbum{album})

	_, cmd := a.Update(panes.FetchAlbumsRequestMsg{Offset: 0})
	require.NotNil(t, cmd, "FetchAlbumsRequestMsg when fresh should return a synthetic cmd")

	msg := cmd()
	loaded, ok := msg.(panes.AlbumsLoadedMsg)
	require.True(t, ok, "synthetic cmd should produce an AlbumsLoadedMsg, got %T", msg)
	assert.Equal(t, 0, loaded.Offset)
	assert.Len(t, loaded.Items, 1)
	assert.Equal(t, "al1", loaded.Items[0].Album.ID)
	assert.Nil(t, loaded.Err)
}

// TestFetchLikedTracksRequest_WhenFresh_ReturnsCachedData verifies that liked tracks
// staleness gate returns cached data when tracks are fresh.
func TestFetchLikedTracksRequest_WhenFresh_ReturnsCachedData(t *testing.T) {
	a := newTestApp()
	track := domain.SavedTrack{AddedAt: "2024-01-01", Track: domain.Track{ID: "t1", Name: "Song"}}
	a.Store().SetLikedTracks([]domain.SavedTrack{track})

	_, cmd := a.Update(panes.FetchLikedTracksRequestMsg{Offset: 0})
	require.NotNil(t, cmd, "FetchLikedTracksRequestMsg when fresh should return a synthetic cmd")

	msg := cmd()
	loaded, ok := msg.(panes.LikedTracksLoadedMsg)
	require.True(t, ok, "synthetic cmd should produce a LikedTracksLoadedMsg, got %T", msg)
	assert.Equal(t, 0, loaded.Offset)
	assert.Len(t, loaded.Items, 1)
	assert.Equal(t, "t1", loaded.Items[0].Track.ID)
	assert.Nil(t, loaded.Err)
}

// TestFetchRecentlyPlayed_WhenFresh_ReturnsCachedData verifies that recently played
// staleness gate returns cached data when data is fresh.
func TestFetchRecentlyPlayed_WhenFresh_ReturnsCachedData(t *testing.T) {
	a := newTestApp()
	item := domain.PlayHistory{PlayedAt: "2024-01-01", Track: domain.Track{ID: "t1"}}
	a.Store().SetRecentlyPlayed([]domain.PlayHistory{item})

	_, cmd := a.Update(panes.FetchRecentlyPlayedRequestMsg{})
	require.NotNil(t, cmd, "FetchRecentlyPlayedRequestMsg when fresh should return a synthetic cmd")

	msg := cmd()
	loaded, ok := msg.(panes.RecentlyPlayedLoadedMsg)
	require.True(t, ok, "synthetic cmd should produce a RecentlyPlayedLoadedMsg, got %T", msg)
	assert.Len(t, loaded.Items, 1)
	assert.Equal(t, "t1", loaded.Items[0].Track.ID)
	assert.Nil(t, loaded.Err)
}

// TestFetchStatsMsg_WhenFresh_ReturnsCachedData verifies that when stats are fresh,
// a FetchStatsMsg returns a synthetic command delivering cached data so the stats
// view can initialize without a redundant API round-trip.
func TestFetchStatsMsg_WhenFresh_ReturnsCachedData(t *testing.T) {
	a := newTestApp()
	a.Store().SetTopTracks("short_term", []domain.Track{{ID: "t1", Name: "Song A"}})
	a.Store().SetTopArtists("short_term", []domain.FullArtist{{ID: "ar1", Name: "Artist A"}})
	a.Store().StampStatsFetchedAt("short_term")
	// Stats are now fresh (within StatsTTL).

	_, cmd := a.Update(panes.FetchStatsMsg{TimeRange: "short_term"})
	require.NotNil(t, cmd, "FetchStatsMsg when fresh should return a synthetic cmd (not nil)")

	msg := cmd()
	loaded, ok := msg.(panes.StatsLoadedMsg)
	require.True(t, ok, "synthetic cmd should produce a StatsLoadedMsg, got %T", msg)
	assert.Equal(t, "short_term", loaded.TimeRange)
	assert.Nil(t, loaded.Err, "synthetic StatsLoadedMsg should have nil Err")
	require.Len(t, loaded.TopTracks, 1, "synthetic StatsLoadedMsg should carry cached tracks")
	assert.Equal(t, "t1", loaded.TopTracks[0].ID)
	require.Len(t, loaded.TopArtists, 1, "synthetic StatsLoadedMsg should carry cached artists")
	assert.Equal(t, "ar1", loaded.TopArtists[0].ID)
}

// TestFetchStatsMsg_WhenFetching_ReturnsNil verifies that when a stats fetch is
// already in-flight, FetchStatsMsg returns nil (prevents TOCTOU duplicates).
func TestFetchStatsMsg_WhenFetching_ReturnsNil(t *testing.T) {
	a := newTestApp()
	a.Store().SetStatsFetching("short_term", true)

	_, cmd := a.Update(panes.FetchStatsMsg{TimeRange: "short_term"})
	assert.Nil(t, cmd, "FetchStatsMsg with in-flight fetch should return nil (no duplicate)")
}

// TestFetchDevicesRequest_WhenFresh_ReturnsCachedData verifies that when device data is
// fresh (within the 5s DevicesTTL), FetchDevicesRequestMsg returns a synthetic cmd
// delivering cached devices — it does NOT dispatch a new API fetch.
// The short 5s TTL prevents rapid-fire API calls while keeping the data fresh.
func TestFetchDevicesRequest_WhenFresh_ReturnsCachedData(t *testing.T) {
	a := newTestApp()
	a.Store().SetDevices([]domain.Device{
		{ID: "d1", Name: "MacBook Pro", Type: "Computer", IsActive: true},
		{ID: "d2", Name: "iPhone 15", Type: "Smartphone", IsActive: false},
	})
	a.Store().SetDevicesFetchedAt(time.Now())
	// Devices are fresh (within 5s DevicesTTL) — handler must return cached data.

	_, cmd := a.Update(panes.FetchDevicesRequestMsg{})
	require.NotNil(t, cmd, "FetchDevicesRequestMsg when fresh should return a synthetic cmd (not nil)")
	// The fetching sentinel must NOT be set — no real API fetch was dispatched.
	assert.False(t, a.Store().DevicesFetching(), "DevicesFetching sentinel must NOT be set when returning cached data")

	// Execute the synthetic command and verify it delivers cached devices.
	msg := cmd()
	loaded, ok := msg.(panes.DevicesLoadedMsg)
	require.True(t, ok, "synthetic cmd must produce a DevicesLoadedMsg, got %T", msg)
	assert.Nil(t, loaded.Err, "synthetic DevicesLoadedMsg should have nil Err")
	require.Len(t, loaded.Devices, 2, "synthetic cmd should carry both cached devices")
	assert.Equal(t, "d1", loaded.Devices[0].ID)
	assert.Equal(t, "d2", loaded.Devices[1].ID)
}

// TestFetchDevicesRequest_WhenFetching_ReturnsNil verifies that when a devices fetch is
// already in-flight, FetchDevicesRequestMsg returns nil (prevents TOCTOU duplicates).
func TestFetchDevicesRequest_WhenFetching_ReturnsNil(t *testing.T) {
	a := newTestApp()
	a.Store().SetDevicesFetching(true)

	_, cmd := a.Update(panes.FetchDevicesRequestMsg{})
	assert.Nil(t, cmd, "FetchDevicesRequestMsg with in-flight fetch should return nil (no duplicate)")
}

// TestDevicesLoadedMsg_StoresCachedDevices verifies that after a successful
// DevicesLoadedMsg is processed, the store contains the raw device list for
// subsequent cached delivery.
func TestDevicesLoadedMsg_StoresCachedDevices(t *testing.T) {
	a := newTestApp()
	msg := panes.DevicesLoadedMsg{
		Devices: []panes.DeviceInfo{
			{ID: "d1", Name: "Laptop", Type: "Computer", IsActive: true},
		},
	}

	a.Update(msg)

	devices := a.Store().Devices()
	require.Len(t, devices, 1, "store should hold the device list after DevicesLoadedMsg")
	assert.Equal(t, "d1", devices[0].ID)
	assert.Equal(t, "Laptop", devices[0].Name)
	assert.Equal(t, "Computer", devices[0].Type)
	assert.True(t, devices[0].IsActive)
}
