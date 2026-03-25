package app_test

// cached_data_test.go — Tests for Task 6: Send cached data when staleness gate blocks
//
// Verifies that when a pane's Init() fires a fetch request and the data is already
// fresh (not stale), the staleness gate returns a synthetic loaded message with cached
// data instead of returning nil. This allows panes to initialize their display without
// waiting for a redundant API round-trip.

import (
	"testing"

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
