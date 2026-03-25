package state

import (
	"fmt"
	"testing"
	"time"

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

func TestStore_TopTracks_InitiallyNil(t *testing.T) {
	s := New()
	assert.Nil(t, s.TopTracks("short_term"), "before any set, TopTracks should return nil")
}

func TestStore_SetGetTopTracks(t *testing.T) {
	s := New()

	tracks := []api.Track{
		{ID: "t1", Name: "Blinding Lights"},
		{ID: "t2", Name: "Levitating"},
	}
	s.SetTopTracks("short_term", tracks)

	got := s.TopTracks("short_term")
	require.Len(t, got, 2)
	assert.Equal(t, "Blinding Lights", got[0].Name)
	assert.Equal(t, "Levitating", got[1].Name)

	// Different range should still be nil.
	assert.Nil(t, s.TopTracks("medium_term"))
}

func TestStore_TopArtists_InitiallyNil(t *testing.T) {
	s := New()
	assert.Nil(t, s.TopArtists("short_term"), "before any set, TopArtists should return nil")
}

func TestStore_SetGetTopArtists(t *testing.T) {
	s := New()

	artists := []api.FullArtist{
		{ID: "a1", Name: "The Weeknd", Genres: []string{"pop"}, Popularity: 95},
		{ID: "a2", Name: "Dua Lipa", Genres: []string{"dance pop"}, Popularity: 92},
	}
	s.SetTopArtists("short_term", artists)

	got := s.TopArtists("short_term")
	require.Len(t, got, 2)
	assert.Equal(t, "The Weeknd", got[0].Name)
	assert.Equal(t, 95, got[0].Popularity)

	// Different range should still be nil.
	assert.Nil(t, s.TopArtists("long_term"))
}

func TestStore_PlaylistTracks_InitiallyNil(t *testing.T) {
	s := New()
	assert.Nil(t, s.PlaylistTracks("pl-1"), "before any set, PlaylistTracks should return nil")
}

func TestStore_SetGetPlaylistTracks(t *testing.T) {
	s := New()

	tracks := []api.Track{
		{ID: "t1", Name: "Blinding Lights", URI: "spotify:track:t1"},
		{ID: "t2", Name: "Levitating", URI: "spotify:track:t2"},
	}
	s.SetPlaylistTracks("pl-1", tracks)

	got := s.PlaylistTracks("pl-1")
	require.Len(t, got, 2)
	assert.Equal(t, "Blinding Lights", got[0].Name)
	assert.Equal(t, "Levitating", got[1].Name)

	// Different playlist should still be nil.
	assert.Nil(t, s.PlaylistTracks("pl-99"))
}

func TestStore_PlayingPlaylistID_Default(t *testing.T) {
	s := New()
	assert.Equal(t, "", s.PlayingPlaylistID(), "initial playing playlist ID should be empty")
}

func TestStore_SetGetPlayingPlaylistID(t *testing.T) {
	s := New()
	s.SetPlayingPlaylistID("pl-abc")
	assert.Equal(t, "pl-abc", s.PlayingPlaylistID())
}

func TestStore_ErrorState(t *testing.T) {
	testErr := fmt.Errorf("api error")

	tests := []struct {
		name  string
		set   func(s *Store)
		get   func(s *Store) error
		clear func(s *Store)
	}{
		{
			name:  "SearchError",
			set:   func(s *Store) { s.SetSearchError(testErr) },
			get:   func(s *Store) error { return s.SearchError() },
			clear: func(s *Store) { s.ClearSearchError() },
		},
		{
			name:  "StatsError",
			set:   func(s *Store) { s.SetStatsError(testErr) },
			get:   func(s *Store) error { return s.StatsError() },
			clear: func(s *Store) { s.ClearStatsError() },
		},
		{
			name:  "DevicesError",
			set:   func(s *Store) { s.SetDevicesError(testErr) },
			get:   func(s *Store) error { return s.DevicesError() },
			clear: func(s *Store) { s.ClearDevicesError() },
		},
		{
			name:  "QueueError",
			set:   func(s *Store) { s.SetQueueError(testErr) },
			get:   func(s *Store) error { return s.QueueError() },
			clear: func(s *Store) { s.ClearQueueError() },
		},
		{
			name:  "PlaylistsFetchError",
			set:   func(s *Store) { s.SetPlaylistsFetchError(testErr) },
			get:   func(s *Store) error { return s.PlaylistsFetchError() },
			clear: func(s *Store) { s.ClearPlaylistsFetchError() },
		},
		{
			name:  "AlbumsFetchError",
			set:   func(s *Store) { s.SetAlbumsFetchError(testErr) },
			get:   func(s *Store) error { return s.AlbumsFetchError() },
			clear: func(s *Store) { s.ClearAlbumsFetchError() },
		},
		{
			name:  "LikedTracksFetchError",
			set:   func(s *Store) { s.SetLikedTracksFetchError(testErr) },
			get:   func(s *Store) error { return s.LikedTracksFetchError() },
			clear: func(s *Store) { s.ClearLikedTracksFetchError() },
		},
		{
			name:  "RecentPlayedFetchError",
			set:   func(s *Store) { s.SetRecentPlayedFetchError(testErr) },
			get:   func(s *Store) error { return s.RecentPlayedFetchError() },
			clear: func(s *Store) { s.ClearRecentPlayedFetchError() },
		},
		{
			name:  "PlaylistsError",
			set:   func(s *Store) { s.SetPlaylistsError(testErr) },
			get:   func(s *Store) error { return s.PlaylistsError() },
			clear: func(s *Store) { s.ClearPlaylistsError() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New()

			// Initially nil.
			assert.Nil(t, tt.get(s), "initial error should be nil")

			// Set error.
			tt.set(s)
			require.Error(t, tt.get(s))
			assert.Equal(t, testErr, tt.get(s))

			// Clear error.
			tt.clear(s)
			assert.Nil(t, tt.get(s), "error should be nil after clear")
		})
	}
}

// --- Throttle observability ---

func TestStore_Throttle_InitiallyFalse(t *testing.T) {
	s := New()
	assert.False(t, s.IsThrottled(), "store should not be throttled initially")
	assert.Equal(t, 0, s.ThrottleRetryAfterSecs(), "retry-after should be 0 initially")
	assert.True(t, s.ThrottleLast429At().IsZero(), "last 429 time should be zero initially")
}

func TestStore_SetThrottle_SetsAllFields(t *testing.T) {
	s := New()

	now := time.Now()
	s.SetThrottle(true, 30, now)

	assert.True(t, s.IsThrottled())
	assert.Equal(t, 30, s.ThrottleRetryAfterSecs())
	assert.Equal(t, now.Unix(), s.ThrottleLast429At().Unix())
}

func TestStore_SetThrottle_ClearsState(t *testing.T) {
	s := New()

	// Set throttle.
	s.SetThrottle(true, 30, time.Now())
	assert.True(t, s.IsThrottled())

	// Clear by setting isThrottled=false.
	s.SetThrottle(false, 0, time.Time{})
	assert.False(t, s.IsThrottled())
	assert.Equal(t, 0, s.ThrottleRetryAfterSecs())
}
