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
	// Pre-populate stats data so the range is within TTL.
	a.Store().SetTopTracks("short_term", []domain.Track{{ID: "t1", Name: "Song"}})
	a.Store().SetTopArtists("short_term", []domain.FullArtist{{ID: "a1", Name: "Artist"}})

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

// TestFetchPlaylistsRequest_WhenFresh_SkipsFetch verifies playlist fetch is skipped when
// the store already has fresh playlist data.
func TestFetchPlaylistsRequest_WhenFresh_SkipsFetch(t *testing.T) {
	a := newTestApp()
	// Pre-populate fresh playlists.
	a.Store().SetPlaylists([]domain.SimplePlaylist{{ID: "pl1", Name: "My Playlist"}})

	_, cmd := a.Update(panes.FetchPlaylistsRequestMsg{Offset: 0})
	assert.Nil(t, cmd, "FetchPlaylistsRequestMsg when fresh should return nil command (skip fetch)")
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

// TestFetchAlbumsRequest_WhenFresh_SkipsFetch verifies album fetch is skipped when
// the store already has fresh album data.
func TestFetchAlbumsRequest_WhenFresh_SkipsFetch(t *testing.T) {
	a := newTestApp()
	a.Store().SetSavedAlbums([]domain.SavedAlbum{{AddedAt: "2024-01-01", Album: domain.FullAlbum{ID: "a1"}}})

	_, cmd := a.Update(panes.FetchAlbumsRequestMsg{Offset: 0})
	assert.Nil(t, cmd, "FetchAlbumsRequestMsg when fresh should return nil command (skip fetch)")
}

// --- Library fetch staleness: FetchLikedTracksRequestMsg ---

// TestFetchLikedTracksRequest_WhenStale_DispatchesFetch verifies liked tracks fetch is
// dispatched when liked tracks have never been fetched.
func TestFetchLikedTracksRequest_WhenStale_DispatchesFetch(t *testing.T) {
	a := newTestApp()
	_, cmd := a.Update(panes.FetchLikedTracksRequestMsg{Offset: 0})
	assert.NotNil(t, cmd, "FetchLikedTracksRequestMsg when stale should dispatch a fetch command")
}

// TestFetchLikedTracksRequest_WhenFresh_SkipsFetch verifies liked tracks fetch is skipped
// when the store already has fresh liked tracks data.
func TestFetchLikedTracksRequest_WhenFresh_SkipsFetch(t *testing.T) {
	a := newTestApp()
	a.Store().SetLikedTracks([]domain.SavedTrack{{AddedAt: "2024-01-01", Track: domain.Track{ID: "t1"}}})

	_, cmd := a.Update(panes.FetchLikedTracksRequestMsg{Offset: 0})
	assert.Nil(t, cmd, "FetchLikedTracksRequestMsg when fresh should return nil command (skip fetch)")
}

// --- Library fetch staleness: FetchRecentlyPlayedRequestMsg ---

// TestFetchRecentlyPlayedRequest_WhenStale_DispatchesFetch verifies recently played fetch is
// dispatched when data has never been fetched.
func TestFetchRecentlyPlayedRequest_WhenStale_DispatchesFetch(t *testing.T) {
	a := newTestApp()
	_, cmd := a.Update(panes.FetchRecentlyPlayedRequestMsg{})
	assert.NotNil(t, cmd, "FetchRecentlyPlayedRequestMsg when stale should dispatch a fetch command")
}

// TestFetchRecentlyPlayedRequest_WhenFresh_SkipsFetch verifies recently played fetch is
// skipped when the store already has fresh data.
func TestFetchRecentlyPlayedRequest_WhenFresh_SkipsFetch(t *testing.T) {
	a := newTestApp()
	a.Store().SetRecentlyPlayed([]domain.PlayHistory{{PlayedAt: "2024-01-01", Track: domain.Track{ID: "t1"}}})

	_, cmd := a.Update(panes.FetchRecentlyPlayedRequestMsg{})
	assert.Nil(t, cmd, "FetchRecentlyPlayedRequestMsg when fresh should return nil command (skip fetch)")
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
