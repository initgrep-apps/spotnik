package app_test

// staleness_test.go — Tests for Feature 32: Staleness-gated data fetching
//
// Verifies that Update() skips fetch dispatches when store data is fresh
// and fires them when data is stale or never fetched.

import (
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
)

// newTestApp creates a fresh App for staleness tests.
func newTestApp() *app.App {
	return app.New(&config.Config{}, app.AppOptions{})
}

// --- Stats staleness ---

// TestFetchStatsMsg_WhenStale_DispatchesFetch verifies that a FetchStatsMsg triggers
// buildFetchStatsCmd when the store's stats for that range are stale (never fetched).
func TestFetchStatsMsg_WhenStale_DispatchesFetch(t *testing.T) {
	a := newTestApp()
	// Stats never fetched → stale → should dispatch a fetch command.
	_, cmd := a.Update(panes.FetchStatsMsg{TimeRange: "short_term"})
	assert.NotNil(t, cmd, "FetchStatsMsg when stale should dispatch a fetch command")
}

// TestFetchStatsMsg_WhenFresh_SkipsFetch verifies that a FetchStatsMsg is skipped when
// the store already has fresh stats for that time range.
func TestFetchStatsMsg_WhenFresh_SkipsFetch(t *testing.T) {
	a := newTestApp()
	// Pre-populate stats data AND stamp so the range is within TTL.
	// SetTopTracks/SetTopArtists no longer stamp statsFetchedAt (Task 4);
	// StampStatsFetchedAt must be called explicitly after both setters.
	a.Store().SetTopTracks("short_term", []domain.Track{{ID: "t1", Name: "Song"}})
	a.Store().SetTopArtists("short_term", []domain.FullArtist{{ID: "a1", Name: "Artist"}})
	a.Store().StampStatsFetchedAt("short_term")

	_, cmd := a.Update(panes.FetchStatsMsg{TimeRange: "short_term"})
	assert.Nil(t, cmd, "FetchStatsMsg when fresh should return nil command (skip fetch)")
}

// TestFetchStatsMsg_DifferentRange_AlwaysFetches verifies that a different time range
// (not yet fetched) is still fetched even when another range is fresh.
func TestFetchStatsMsg_DifferentRange_AlwaysFetches(t *testing.T) {
	a := newTestApp()
	// Populate short_term — medium_term is still stale.
	a.Store().SetTopTracks("short_term", []domain.Track{{ID: "t1", Name: "Song"}})
	a.Store().SetTopArtists("short_term", []domain.FullArtist{{ID: "a1", Name: "Artist"}})

	_, cmd := a.Update(panes.FetchStatsMsg{TimeRange: "medium_term"})
	assert.NotNil(t, cmd, "FetchStatsMsg for unstale range should dispatch a fetch command")
}

// --- Library fetch staleness: FetchPlaylistsRequestMsg ---

// TestFetchPlaylistsRequest_WhenStale_DispatchesFetch verifies playlist fetch is dispatched
// when playlists have never been fetched.
func TestFetchPlaylistsRequest_WhenStale_DispatchesFetch(t *testing.T) {
	a := newTestApp()
	_, cmd := a.Update(panes.FetchPlaylistsRequestMsg{Offset: 0})
	assert.NotNil(t, cmd, "FetchPlaylistsRequestMsg when stale should dispatch a fetch command")
}

// TestFetchPlaylistsRequest_WhenFresh_ReturnsSyntheticCmd verifies that when playlists
// are fresh, a FetchPlaylistsRequestMsg returns a synthetic command with cached data
// (Task 6: cached data delivery so pane can initialize without a redundant API call).
func TestFetchPlaylistsRequest_WhenFresh_ReturnsSyntheticCmd(t *testing.T) {
	a := newTestApp()
	// Pre-populate fresh playlists.
	a.Store().SetPlaylists([]domain.SimplePlaylist{{ID: "pl1", Name: "My Playlist"}})

	_, cmd := a.Update(panes.FetchPlaylistsRequestMsg{Offset: 0})
	// Task 6: now returns a synthetic cmd delivering cached data, not nil.
	assert.NotNil(t, cmd, "FetchPlaylistsRequestMsg when fresh should return a synthetic cmd (not nil)")
	msg := cmd()
	loaded, ok := msg.(panes.LibraryLoadedMsg)
	assert.True(t, ok, "synthetic cmd should produce LibraryLoadedMsg")
	assert.Nil(t, loaded.Err, "synthetic LibraryLoadedMsg should have nil Err")
}

// TestFetchPlaylistsRequest_PaginatedOffset_AlwaysFetches verifies that paginated
// playlist requests (offset > 0) always fetch, bypassing staleness checks.
func TestFetchPlaylistsRequest_PaginatedOffset_AlwaysFetches(t *testing.T) {
	a := newTestApp()
	// Even with fresh data, offset > 0 should always fetch (pagination).
	a.Store().SetPlaylists([]domain.SimplePlaylist{{ID: "pl1", Name: "My Playlist"}})

	_, cmd := a.Update(panes.FetchPlaylistsRequestMsg{Offset: 20})
	assert.NotNil(t, cmd, "FetchPlaylistsRequestMsg with offset > 0 should always dispatch fetch")
}

// --- Library fetch staleness: FetchAlbumsRequestMsg ---

// TestFetchAlbumsRequest_WhenStale_DispatchesFetch verifies album fetch is dispatched
// when albums have never been fetched.
func TestFetchAlbumsRequest_WhenStale_DispatchesFetch(t *testing.T) {
	a := newTestApp()
	_, cmd := a.Update(panes.FetchAlbumsRequestMsg{Offset: 0})
	assert.NotNil(t, cmd, "FetchAlbumsRequestMsg when stale should dispatch a fetch command")
}

// TestFetchAlbumsRequest_WhenFresh_ReturnsSyntheticCmd verifies that when albums are fresh,
// a FetchAlbumsRequestMsg returns a synthetic command with cached data (Task 6).
func TestFetchAlbumsRequest_WhenFresh_ReturnsSyntheticCmd(t *testing.T) {
	a := newTestApp()
	a.Store().SetSavedAlbums([]domain.SavedAlbum{{AddedAt: "2024-01-01", Album: domain.FullAlbum{ID: "a1"}}})

	_, cmd := a.Update(panes.FetchAlbumsRequestMsg{Offset: 0})
	assert.NotNil(t, cmd, "FetchAlbumsRequestMsg when fresh should return a synthetic cmd")
	msg := cmd()
	loaded, ok := msg.(panes.AlbumsLoadedMsg)
	assert.True(t, ok, "synthetic cmd should produce AlbumsLoadedMsg")
	assert.Nil(t, loaded.Err)
}

// --- Library fetch staleness: FetchLikedTracksRequestMsg ---

// TestFetchLikedTracksRequest_WhenStale_DispatchesFetch verifies liked tracks fetch is
// dispatched when liked tracks have never been fetched.
func TestFetchLikedTracksRequest_WhenStale_DispatchesFetch(t *testing.T) {
	a := newTestApp()
	_, cmd := a.Update(panes.FetchLikedTracksRequestMsg{Offset: 0})
	assert.NotNil(t, cmd, "FetchLikedTracksRequestMsg when stale should dispatch a fetch command")
}

// TestFetchLikedTracksRequest_WhenFresh_ReturnsSyntheticCmd verifies that when liked tracks
// are fresh, a FetchLikedTracksRequestMsg returns a synthetic command with cached data (Task 6).
func TestFetchLikedTracksRequest_WhenFresh_ReturnsSyntheticCmd(t *testing.T) {
	a := newTestApp()
	a.Store().SetLikedTracks([]domain.SavedTrack{{AddedAt: "2024-01-01", Track: domain.Track{ID: "t1"}}})

	_, cmd := a.Update(panes.FetchLikedTracksRequestMsg{Offset: 0})
	assert.NotNil(t, cmd, "FetchLikedTracksRequestMsg when fresh should return a synthetic cmd")
	msg := cmd()
	loaded, ok := msg.(panes.LikedTracksLoadedMsg)
	assert.True(t, ok, "synthetic cmd should produce LikedTracksLoadedMsg")
	assert.Nil(t, loaded.Err)
}

// --- Library fetch staleness: FetchRecentlyPlayedRequestMsg ---

// TestFetchRecentlyPlayedRequest_WhenStale_DispatchesFetch verifies recently played fetch is
// dispatched when data has never been fetched.
func TestFetchRecentlyPlayedRequest_WhenStale_DispatchesFetch(t *testing.T) {
	a := newTestApp()
	_, cmd := a.Update(panes.FetchRecentlyPlayedRequestMsg{})
	assert.NotNil(t, cmd, "FetchRecentlyPlayedRequestMsg when stale should dispatch a fetch command")
}

// TestFetchRecentlyPlayedRequest_WhenFresh_ReturnsSyntheticCmd verifies that when recently
// played data is fresh, a FetchRecentlyPlayedRequestMsg returns a synthetic cmd (Task 6).
func TestFetchRecentlyPlayedRequest_WhenFresh_ReturnsSyntheticCmd(t *testing.T) {
	a := newTestApp()
	a.Store().SetRecentlyPlayed([]domain.PlayHistory{{PlayedAt: "2024-01-01", Track: domain.Track{ID: "t1"}}})

	_, cmd := a.Update(panes.FetchRecentlyPlayedRequestMsg{})
	assert.NotNil(t, cmd, "FetchRecentlyPlayedRequestMsg when fresh should return a synthetic cmd")
	msg := cmd()
	loaded, ok := msg.(panes.RecentlyPlayedLoadedMsg)
	assert.True(t, ok, "synthetic cmd should produce RecentlyPlayedLoadedMsg")
	assert.Nil(t, loaded.Err)
}

// --- Cross-domain isolation ---

// TestFetchAlbumsRequest_IndependentOfPlaylists verifies that albums and playlists
// are tracked independently — a fresh playlists store does not affect album staleness.
func TestFetchAlbumsRequest_IndependentOfPlaylists(t *testing.T) {
	a := newTestApp()
	// Fresh playlists should not affect album staleness.
	a.Store().SetPlaylists([]domain.SimplePlaylist{{ID: "pl1", Name: "My Playlist"}})

	// Albums are still stale (never fetched) — fetch should proceed.
	_, cmd := a.Update(panes.FetchAlbumsRequestMsg{Offset: 0})
	assert.NotNil(t, cmd, "albums should still be stale even when playlists are fresh")
}

// --- Device fetch staleness: FetchDevicesRequestMsg ---

// TestFetchDevicesRequest_WhenStale_DispatchesFetch verifies device fetch is dispatched
// when the device list has never been fetched (store fetchedAt is zero).
func TestFetchDevicesRequest_WhenStale_DispatchesFetch(t *testing.T) {
	a := newTestApp()
	_, cmd := a.Update(panes.FetchDevicesRequestMsg{})
	assert.NotNil(t, cmd, "FetchDevicesRequestMsg when stale should dispatch a fetch command")
}

// TestFetchDevicesRequest_WhenFresh_SkipsFetch verifies device fetch is skipped when
// the store already has a recent devicesFetchedAt timestamp (within DevicesTTL).
func TestFetchDevicesRequest_WhenFresh_SkipsFetch(t *testing.T) {
	a := newTestApp()
	// Stamp as recently fetched so the store is within DevicesTTL.
	a.Store().SetDevicesFetchedAt(time.Now())

	_, cmd := a.Update(panes.FetchDevicesRequestMsg{})
	assert.Nil(t, cmd, "FetchDevicesRequestMsg when fresh should return nil command (skip fetch)")
}

// TestStalenessTTLConstants verifies that the exported TTL constants match the spec values.
// This acts as a regression guard against accidental constant changes.
func TestStalenessTTLConstants(t *testing.T) {
	assert.Equal(t, 5*time.Minute, state.PlaylistsTTL, "PlaylistsTTL should be 5 minutes")
	assert.Equal(t, 5*time.Minute, state.AlbumsTTL, "AlbumsTTL should be 5 minutes")
	assert.Equal(t, 5*time.Minute, state.LikedTracksTTL, "LikedTracksTTL should be 5 minutes")
	assert.Equal(t, 2*time.Minute, state.RecentlyPlayedTTL, "RecentlyPlayedTTL should be 2 minutes")
	assert.Equal(t, 10*time.Minute, state.StatsTTL, "StatsTTL should be 10 minutes")
	assert.Equal(t, 30*time.Second, state.DevicesTTL, "DevicesTTL should be 30 seconds")
}

// --- Fetching sentinels: TOCTOU prevention (Task 5) ---

// TestFetchPlaylistsRequest_WhenFetching_SkipsDuplicateDispatch verifies that a second
// FetchPlaylistsRequestMsg is not dispatched when a fetch is already in-flight.
func TestFetchPlaylistsRequest_WhenFetching_SkipsDuplicateDispatch(t *testing.T) {
	a := newTestApp()
	// Playlists are stale (never fetched), but fetching sentinel is set.
	a.Store().SetPlaylistsFetching(true)

	_, cmd := a.Update(panes.FetchPlaylistsRequestMsg{Offset: 0})
	assert.Nil(t, cmd, "FetchPlaylistsRequestMsg should be skipped when already fetching")
}

// TestFetchPlaylistsRequest_PaginatedOffset_IgnoresFetchingSentinel verifies that
// paginated requests (offset > 0) bypass the fetching sentinel.
func TestFetchPlaylistsRequest_PaginatedOffset_IgnoresFetchingSentinel(t *testing.T) {
	a := newTestApp()
	// Even with fetching sentinel set, offset > 0 should always dispatch.
	a.Store().SetPlaylistsFetching(true)

	_, cmd := a.Update(panes.FetchPlaylistsRequestMsg{Offset: 20})
	assert.NotNil(t, cmd, "paginated FetchPlaylistsRequestMsg must bypass fetching sentinel")
}

// TestFetchAlbumsRequest_WhenFetching_SkipsDuplicateDispatch verifies the albums
// fetching sentinel prevents duplicate dispatches.
func TestFetchAlbumsRequest_WhenFetching_SkipsDuplicateDispatch(t *testing.T) {
	a := newTestApp()
	a.Store().SetAlbumsFetching(true)

	_, cmd := a.Update(panes.FetchAlbumsRequestMsg{Offset: 0})
	assert.Nil(t, cmd, "FetchAlbumsRequestMsg should be skipped when already fetching")
}

// TestFetchLikedTracksRequest_WhenFetching_SkipsDuplicateDispatch verifies the liked
// tracks fetching sentinel prevents duplicate dispatches.
func TestFetchLikedTracksRequest_WhenFetching_SkipsDuplicateDispatch(t *testing.T) {
	a := newTestApp()
	a.Store().SetLikedFetching(true)

	_, cmd := a.Update(panes.FetchLikedTracksRequestMsg{Offset: 0})
	assert.Nil(t, cmd, "FetchLikedTracksRequestMsg should be skipped when already fetching")
}

// TestFetchRecentlyPlayed_WhenFetching_SkipsDuplicateDispatch verifies the recently-played
// fetching sentinel prevents duplicate dispatches.
func TestFetchRecentlyPlayed_WhenFetching_SkipsDuplicateDispatch(t *testing.T) {
	a := newTestApp()
	a.Store().SetRecentFetching(true)

	_, cmd := a.Update(panes.FetchRecentlyPlayedRequestMsg{})
	assert.Nil(t, cmd, "FetchRecentlyPlayedRequestMsg should be skipped when already fetching")
}

// TestFetchStats_WhenFetching_SkipsDuplicateDispatch verifies the stats fetching
// sentinel prevents duplicate dispatches.
func TestFetchStats_WhenFetching_SkipsDuplicateDispatch(t *testing.T) {
	a := newTestApp()
	a.Store().SetStatsFetching("short_term", true)

	_, cmd := a.Update(panes.FetchStatsMsg{TimeRange: "short_term"})
	assert.Nil(t, cmd, "FetchStatsMsg should be skipped when already fetching for that range")
}

// TestFetchDevicesRequest_WhenFetching_SkipsDuplicateDispatch verifies the devices
// fetching sentinel prevents duplicate dispatches.
func TestFetchDevicesRequest_WhenFetching_SkipsDuplicateDispatch(t *testing.T) {
	a := newTestApp()
	a.Store().SetDevicesFetching(true)

	_, cmd := a.Update(panes.FetchDevicesRequestMsg{})
	assert.Nil(t, cmd, "FetchDevicesRequestMsg should be skipped when already fetching")
}
