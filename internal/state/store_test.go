package state

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_SetGetPlaybackState(t *testing.T) {
	s := New()

	// Initially nil.
	assert.Nil(t, s.PlaybackState())

	track := &api.Track{
		ID:         "track-1",
		Name:       "Test Track",
		DurationMs: 200000,
	}
	state := &api.PlaybackState{
		IsPlaying:  true,
		ProgressMs: 5000,
		Item:       track,
	}

	s.SetPlaybackState(state)
	got := s.PlaybackState()

	assert.NotNil(t, got)
	assert.True(t, got.IsPlaying)
	assert.Equal(t, 5000, got.ProgressMs)
	assert.Equal(t, "Test Track", got.Item.Name)
}

func TestStore_SetPlaybackState_Nil(t *testing.T) {
	s := New()

	// Set something first.
	s.SetPlaybackState(&api.PlaybackState{IsPlaying: true})
	assert.NotNil(t, s.PlaybackState())

	// Clear it.
	s.SetPlaybackState(nil)
	assert.Nil(t, s.PlaybackState())
}

func TestStore_SetGetActiveDevice(t *testing.T) {
	s := New()

	assert.Nil(t, s.ActiveDevice())

	device := &api.Device{
		ID:            "dev-1",
		Name:          "MacBook Pro",
		VolumePercent: 70,
	}

	s.SetActiveDevice(device)
	got := s.ActiveDevice()

	assert.NotNil(t, got)
	assert.Equal(t, "MacBook Pro", got.Name)
	assert.Equal(t, 70, got.VolumePercent)
}

func TestStore_SetGetPlaylists(t *testing.T) {
	s := New()

	assert.Empty(t, s.Playlists())
	assert.Equal(t, 0, s.PlaylistsTotal())

	playlists := []api.SimplePlaylist{
		{ID: "pl1", Name: "Chill Vibes", URI: "spotify:playlist:pl1"},
		{ID: "pl2", Name: "Workout Mix", URI: "spotify:playlist:pl2"},
	}
	s.SetPlaylists(playlists)
	s.SetPlaylistsTotal(42)

	got := s.Playlists()
	assert.Len(t, got, 2)
	assert.Equal(t, "pl1", got[0].ID)
	assert.Equal(t, "Chill Vibes", got[0].Name)
	assert.Equal(t, 42, s.PlaylistsTotal())
}

func TestStore_SetGetSavedAlbums(t *testing.T) {
	s := New()

	assert.Empty(t, s.SavedAlbums())
	assert.False(t, s.AlbumsLoaded())

	albums := []api.SavedAlbum{
		{AddedAt: "2024-01-15T10:30:00Z", Album: api.FullAlbum{ID: "album-1", Name: "After Hours"}},
	}
	s.SetSavedAlbums(albums)

	got := s.SavedAlbums()
	assert.Len(t, got, 1)
	assert.Equal(t, "album-1", got[0].Album.ID)
	assert.True(t, s.AlbumsLoaded(), "AlbumsLoaded should be true after SetSavedAlbums")
}

func TestStore_SetGetLikedTracks(t *testing.T) {
	s := New()

	assert.Empty(t, s.LikedTracks())
	assert.False(t, s.LikedLoaded())
	assert.Equal(t, 0, s.LikedTotal())

	tracks := []api.SavedTrack{
		{AddedAt: "2024-02-20T14:00:00Z", Track: api.Track{ID: "track-1", Name: "Blinding Lights"}},
	}
	s.SetLikedTracks(tracks)
	s.SetLikedTotal(287)

	got := s.LikedTracks()
	assert.Len(t, got, 1)
	assert.Equal(t, "track-1", got[0].Track.ID)
	assert.True(t, s.LikedLoaded(), "LikedLoaded should be true after SetLikedTracks")
	assert.Equal(t, 287, s.LikedTotal())
}

func TestStore_SetGetRecentlyPlayed(t *testing.T) {
	s := New()

	assert.Empty(t, s.RecentlyPlayed())

	items := []api.PlayHistory{
		{PlayedAt: "2024-03-01T22:15:00Z", Track: api.Track{ID: "track-xyz", Name: "Blinding Lights"}},
		{PlayedAt: "2024-03-01T22:10:00Z", Track: api.Track{ID: "track-abc", Name: "Save Your Tears"}},
	}
	s.SetRecentlyPlayed(items)

	got := s.RecentlyPlayed()
	assert.Len(t, got, 2)
	assert.Equal(t, "track-xyz", got[0].Track.ID)
	assert.Equal(t, "2024-03-01T22:15:00Z", got[0].PlayedAt)
}

func TestStore_SearchResults(t *testing.T) {
	s := New()

	// Initially nil.
	assert.Nil(t, s.SearchResults())

	results := &api.SearchResult{
		Tracks: api.SearchTracksResult{
			Items: []api.Track{{ID: "t1", Name: "Blinding Lights"}},
			Total: 1,
		},
	}
	s.SetSearchResults(results)

	got := s.SearchResults()
	require.NotNil(t, got)
	require.Len(t, got.Tracks.Items, 1)
	assert.Equal(t, "Blinding Lights", got.Tracks.Items[0].Name)

	// Clear results.
	s.SetSearchResults(nil)
	assert.Nil(t, s.SearchResults())
}

func TestStore_SearchQuery(t *testing.T) {
	s := New()

	assert.Equal(t, "", s.SearchQuery())

	s.SetSearchQuery("blinding lights")
	assert.Equal(t, "blinding lights", s.SearchQuery())
}

func TestStore_SearchLoading(t *testing.T) {
	s := New()

	assert.False(t, s.SearchLoading())

	s.SetSearchLoading(true)
	assert.True(t, s.SearchLoading())

	s.SetSearchLoading(false)
	assert.False(t, s.SearchLoading())
}

func TestStore_SetGetQueue(t *testing.T) {
	s := New()

	// Initially empty.
	assert.Empty(t, s.Queue())

	tracks := []api.Track{
		{ID: "q1", Name: "Save Your Tears", URI: "spotify:track:q1"},
		{ID: "q2", Name: "Starboy", URI: "spotify:track:q2"},
	}
	s.SetQueue(tracks)

	got := s.Queue()
	require.Len(t, got, 2)
	assert.Equal(t, "Save Your Tears", got[0].Name)
	assert.Equal(t, "Starboy", got[1].Name)

	// Clear queue.
	s.SetQueue(nil)
	assert.Empty(t, s.Queue())
}
