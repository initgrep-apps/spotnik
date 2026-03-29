package state

import (
	"fmt"
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/domain"
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

// --- Staleness tracking ---

func TestIsStale_ZeroTime(t *testing.T) {
	// Zero time (never fetched) is always stale.
	var zero time.Time
	assert.True(t, IsStale(zero, 5*time.Minute), "zero fetchedAt should be stale")
}

func TestIsStale_ExpiredTTL(t *testing.T) {
	// fetchedAt older than TTL → stale.
	fetchedAt := time.Now().Add(-6 * time.Minute)
	assert.True(t, IsStale(fetchedAt, 5*time.Minute), "fetchedAt older than TTL should be stale")
}

func TestIsStale_WithinTTL(t *testing.T) {
	// fetchedAt within TTL → not stale.
	fetchedAt := time.Now().Add(-1 * time.Minute)
	assert.False(t, IsStale(fetchedAt, 5*time.Minute), "recent fetchedAt should not be stale")
}

func TestStore_SetPlaylists_UpdatesFetchedAt(t *testing.T) {
	s := New()
	before := time.Now()
	s.SetPlaylists([]api.SimplePlaylist{{ID: "pl1", Name: "Test"}})
	after := time.Now()

	ts := s.PlaylistsFetchedAt()
	assert.False(t, ts.IsZero(), "PlaylistsFetchedAt should be set after SetPlaylists")
	assert.True(t, ts.After(before) || ts.Equal(before), "fetchedAt should be >= before")
	assert.True(t, ts.Before(after) || ts.Equal(after), "fetchedAt should be <= after")
}

func TestStore_SetSavedAlbums_UpdatesFetchedAt(t *testing.T) {
	s := New()
	before := time.Now()
	s.SetSavedAlbums([]api.SavedAlbum{{AddedAt: "2024-01-01", Album: api.FullAlbum{ID: "a1"}}})
	after := time.Now()

	ts := s.AlbumsFetchedAt()
	assert.False(t, ts.IsZero(), "AlbumsFetchedAt should be set after SetSavedAlbums")
	assert.True(t, ts.After(before) || ts.Equal(before))
	assert.True(t, ts.Before(after) || ts.Equal(after))
}

func TestStore_SetTopTracks_DoesNotStampStatsFetchedAt(t *testing.T) {
	// Task 4: SetTopTracks must NOT stamp statsFetchedAt anymore.
	// Only StampStatsFetchedAt (called once after both setters) stamps it.
	s := New()
	s.SetTopTracks("short_term", []api.Track{{ID: "t1", Name: "Track 1"}})

	ts := s.StatsFetchedAt("short_term")
	assert.True(t, ts.IsZero(), "SetTopTracks must NOT stamp statsFetchedAt (use StampStatsFetchedAt)")
}

func TestStore_SetTopArtists_DoesNotStampStatsFetchedAt(t *testing.T) {
	// Task 4: SetTopArtists must NOT stamp statsFetchedAt anymore.
	s := New()
	s.SetTopArtists("long_term", []api.FullArtist{{ID: "a1", Name: "Artist 1"}})

	ts := s.StatsFetchedAt("long_term")
	assert.True(t, ts.IsZero(), "SetTopArtists must NOT stamp statsFetchedAt (use StampStatsFetchedAt)")
}

func TestStore_StampStatsFetchedAt_UpdatesTimestamp(t *testing.T) {
	// Task 4: StampStatsFetchedAt must stamp the timestamp for the given range.
	s := New()
	s.SetTopTracks("short_term", []api.Track{{ID: "t1"}})
	s.SetTopArtists("short_term", []api.FullArtist{{ID: "a1"}})

	// Before stamping — stale.
	assert.True(t, s.StatsStale("short_term"), "StatsStale must return true before StampStatsFetchedAt")

	before := time.Now()
	s.StampStatsFetchedAt("short_term")
	after := time.Now()

	ts := s.StatsFetchedAt("short_term")
	assert.False(t, ts.IsZero(), "StampStatsFetchedAt must stamp statsFetchedAt")
	assert.True(t, ts.After(before) || ts.Equal(before))
	assert.True(t, ts.Before(after) || ts.Equal(after))

	// Other ranges are unaffected.
	assert.True(t, s.StatsFetchedAt("medium_term").IsZero(), "other ranges should remain zero")

	// After stamping — not stale.
	assert.False(t, s.StatsStale("short_term"), "StatsStale must return false after StampStatsFetchedAt")
}

func TestStore_FetchedAt_Accessors(t *testing.T) {
	s := New()

	// Initially zero for all domains.
	assert.True(t, s.PlaylistsFetchedAt().IsZero(), "PlaylistsFetchedAt initially zero")
	assert.True(t, s.AlbumsFetchedAt().IsZero(), "AlbumsFetchedAt initially zero")
	assert.True(t, s.LikedTracksFetchedAt().IsZero(), "LikedTracksFetchedAt initially zero")
	assert.True(t, s.RecentPlayedFetchedAt().IsZero(), "RecentPlayedFetchedAt initially zero")
	assert.True(t, s.DevicesFetchedAt().IsZero(), "DevicesFetchedAt initially zero")
	assert.True(t, s.StatsFetchedAt("short_term").IsZero(), "StatsFetchedAt initially zero")

	// Nil/empty sets do NOT stamp fetchedAt (F38: guard against empty data resetting TTL).
	s.SetPlaylists(nil)
	assert.True(t, s.PlaylistsFetchedAt().IsZero(), "SetPlaylists(nil) must not stamp fetchedAt")

	s.SetLikedTracks(nil)
	assert.True(t, s.LikedTracksFetchedAt().IsZero(), "SetLikedTracks(nil) must not stamp fetchedAt")

	s.SetRecentlyPlayed(nil)
	assert.True(t, s.RecentPlayedFetchedAt().IsZero(), "SetRecentlyPlayed(nil) must not stamp fetchedAt")

	// Non-empty sets DO stamp fetchedAt.
	s.SetPlaylists([]domain.SimplePlaylist{{ID: "pl1", Name: "My Playlist"}})
	assert.False(t, s.PlaylistsFetchedAt().IsZero(), "SetPlaylists(nonEmpty) must stamp fetchedAt")

	s.SetLikedTracks([]domain.SavedTrack{{AddedAt: "2024-01-01", Track: domain.Track{ID: "t1"}}})
	assert.False(t, s.LikedTracksFetchedAt().IsZero(), "SetLikedTracks(nonEmpty) must stamp fetchedAt")

	s.SetRecentlyPlayed([]domain.PlayHistory{{PlayedAt: "2024-01-01", Track: domain.Track{ID: "t1"}}})
	assert.False(t, s.RecentPlayedFetchedAt().IsZero(), "SetRecentlyPlayed(nonEmpty) must stamp fetchedAt")
}

// --- TTL-based staleness convenience methods ---

func TestStore_PlaylistsStale_NeverFetched(t *testing.T) {
	s := New()
	assert.True(t, s.PlaylistsStale(), "PlaylistsStale should be true when never fetched")
}

func TestStore_PlaylistsStale_AfterTTL(t *testing.T) {
	s := New()
	// Manually set a past fetchedAt to simulate TTL expiry.
	s.mu.Lock()
	s.playlistsFetchedAt = time.Now().Add(-(PlaylistsTTL + time.Second))
	s.mu.Unlock()
	assert.True(t, s.PlaylistsStale(), "PlaylistsStale should be true after TTL")
}

func TestStore_PlaylistsStale_WithinTTL(t *testing.T) {
	s := New()
	// Non-empty slice required to stamp fetchedAt (nil does not stamp per Task 3 guard).
	s.SetPlaylists([]domain.SimplePlaylist{{ID: "pl1", Name: "Test"}})
	assert.False(t, s.PlaylistsStale(), "PlaylistsStale should be false within TTL")
}

func TestStore_AlbumsStale_NeverFetched(t *testing.T) {
	s := New()
	assert.True(t, s.AlbumsStale(), "AlbumsStale should be true when never fetched")
}

func TestStore_AlbumsStale_WithinTTL(t *testing.T) {
	s := New()
	// Non-empty slice required to stamp fetchedAt (nil does not stamp per Task 3 guard).
	s.SetSavedAlbums([]domain.SavedAlbum{{AddedAt: "2024-01-01", Album: domain.FullAlbum{ID: "a1"}}})
	assert.False(t, s.AlbumsStale(), "AlbumsStale should be false within TTL")
}

func TestStore_LikedTracksStale_NeverFetched(t *testing.T) {
	s := New()
	assert.True(t, s.LikedTracksStale(), "LikedTracksStale should be true when never fetched")
}

func TestStore_LikedTracksStale_WithinTTL(t *testing.T) {
	s := New()
	// Non-empty slice required to stamp fetchedAt (nil does not stamp per Task 3 guard).
	s.SetLikedTracks([]domain.SavedTrack{{AddedAt: "2024-01-01", Track: domain.Track{ID: "t1"}}})
	assert.False(t, s.LikedTracksStale(), "LikedTracksStale should be false within TTL")
}

func TestStore_RecentlyPlayedStale_NeverFetched(t *testing.T) {
	s := New()
	assert.True(t, s.RecentlyPlayedStale(), "RecentlyPlayedStale should be true when never fetched")
}

func TestStore_RecentlyPlayedStale_WithinTTL(t *testing.T) {
	s := New()
	// Non-empty slice required to stamp fetchedAt (nil does not stamp per Task 3 guard).
	s.SetRecentlyPlayed([]domain.PlayHistory{{PlayedAt: "2024-01-01", Track: domain.Track{ID: "t1"}}})
	assert.False(t, s.RecentlyPlayedStale(), "RecentlyPlayedStale should be false within TTL")
}

func TestStore_StatsStale_NeverFetched(t *testing.T) {
	s := New()
	assert.True(t, s.StatsStale("short_term"), "StatsStale should be true when never fetched")
}

func TestStore_StatsStale_WithinTTL(t *testing.T) {
	s := New()
	// StampStatsFetchedAt is now the only way to mark stats as fresh (Task 4).
	s.StampStatsFetchedAt("short_term")
	assert.False(t, s.StatsStale("short_term"), "StatsStale should be false within TTL")
}

func TestStore_DevicesStale_NeverFetched(t *testing.T) {
	s := New()
	assert.True(t, s.DevicesStale(), "DevicesStale should be true when never fetched")
}

func TestStore_DevicesStale_WithinTTL(t *testing.T) {
	s := New()
	s.SetDevicesFetchedAt(time.Now()) // stamp now
	assert.False(t, s.DevicesStale(), "DevicesStale should be false within TTL")
}

func TestStore_DevicesStale_AfterTTL(t *testing.T) {
	s := New()
	s.SetDevicesFetchedAt(time.Now().Add(-(DevicesTTL + time.Second)))
	assert.True(t, s.DevicesStale(), "DevicesStale should be true after TTL")
}

// --- statsFetchedAt initialization ---

// TestStore_New_HasInitializedStatsFetchedAt verifies that New() initializes the
// statsFetchedAt map so callers do not need to guard against nil before reading.
func TestStore_New_HasInitializedStatsFetchedAt(t *testing.T) {
	s := New()
	// Access the unexported field directly — we are in package state.
	assert.NotNil(t, s.statsFetchedAt, "New() must initialize statsFetchedAt to a non-nil map")
}

// TestStore_StatsStale_OnFreshStore_NoPanic verifies that StatsStale does not panic
// on a brand-new Store before any Set* call has been made.
func TestStore_StatsStale_OnFreshStore_NoPanic(t *testing.T) {
	s := New()
	// Should not panic; fresh store has no fetched-at stamp so result must be true.
	assert.True(t, s.StatsStale("short_term"), "fresh store should report stats as stale")
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

// --- Fetching sentinels (Task 5) ---

func TestStore_PlaylistsFetching_DefaultFalse(t *testing.T) {
	s := New()
	assert.False(t, s.PlaylistsFetching(), "playlistsFetching should be false initially")
}

func TestStore_SetPlaylistsFetching_SetsAndClears(t *testing.T) {
	s := New()
	s.SetPlaylistsFetching(true)
	assert.True(t, s.PlaylistsFetching(), "PlaylistsFetching should be true after Set(true)")
	s.SetPlaylistsFetching(false)
	assert.False(t, s.PlaylistsFetching(), "PlaylistsFetching should be false after Set(false)")
}

func TestStore_AlbumsFetching_DefaultFalse(t *testing.T) {
	s := New()
	assert.False(t, s.AlbumsFetching(), "albumsFetching should be false initially")
}

func TestStore_SetAlbumsFetching_SetsAndClears(t *testing.T) {
	s := New()
	s.SetAlbumsFetching(true)
	assert.True(t, s.AlbumsFetching())
	s.SetAlbumsFetching(false)
	assert.False(t, s.AlbumsFetching())
}

func TestStore_LikedFetching_DefaultFalse(t *testing.T) {
	s := New()
	assert.False(t, s.LikedFetching(), "likedFetching should be false initially")
}

func TestStore_SetLikedFetching_SetsAndClears(t *testing.T) {
	s := New()
	s.SetLikedFetching(true)
	assert.True(t, s.LikedFetching())
	s.SetLikedFetching(false)
	assert.False(t, s.LikedFetching())
}

func TestStore_RecentFetching_DefaultFalse(t *testing.T) {
	s := New()
	assert.False(t, s.RecentFetching(), "recentFetching should be false initially")
}

func TestStore_SetRecentFetching_SetsAndClears(t *testing.T) {
	s := New()
	s.SetRecentFetching(true)
	assert.True(t, s.RecentFetching())
	s.SetRecentFetching(false)
	assert.False(t, s.RecentFetching())
}

func TestStore_StatsFetching_DefaultFalse(t *testing.T) {
	s := New()
	assert.False(t, s.StatsFetching("short_term"), "statsFetching should be false initially")
	assert.False(t, s.StatsFetching("medium_term"), "statsFetching should be false for all ranges")
}

func TestStore_SetStatsFetching_SetsAndClears(t *testing.T) {
	s := New()
	s.SetStatsFetching("short_term", true)
	assert.True(t, s.StatsFetching("short_term"))
	assert.False(t, s.StatsFetching("medium_term"), "other ranges should be unaffected")
	s.SetStatsFetching("short_term", false)
	assert.False(t, s.StatsFetching("short_term"))
}

func TestStore_DevicesFetching_DefaultFalse(t *testing.T) {
	s := New()
	assert.False(t, s.DevicesFetching(), "devicesFetching should be false initially")
}

func TestStore_SetDevicesFetching_SetsAndClears(t *testing.T) {
	s := New()
	s.SetDevicesFetching(true)
	assert.True(t, s.DevicesFetching())
	s.SetDevicesFetching(false)
	assert.False(t, s.DevicesFetching())
}

func TestStore_Devices_InitiallyNil(t *testing.T) {
	s := New()
	assert.Nil(t, s.Devices(), "Devices should return nil before any SetDevices call")
}

func TestStore_SetDevices_StoresAndRetrieves(t *testing.T) {
	s := New()
	devices := []domain.Device{
		{ID: "dev1", Name: "MacBook Pro", Type: "Computer", IsActive: true},
		{ID: "dev2", Name: "iPhone 15", Type: "Smartphone", IsActive: false},
	}
	s.SetDevices(devices)
	got := s.Devices()
	assert.Len(t, got, 2)
	assert.Equal(t, "dev1", got[0].ID)
	assert.Equal(t, "dev2", got[1].ID)
}

func TestStore_SetDevices_ReplacesExistingList(t *testing.T) {
	s := New()
	s.SetDevices([]domain.Device{{ID: "old", Name: "Old Device", Type: "Computer"}})
	s.SetDevices([]domain.Device{{ID: "new", Name: "New Device", Type: "Smartphone"}})
	got := s.Devices()
	assert.Len(t, got, 1, "SetDevices should replace the old list")
	assert.Equal(t, "new", got[0].ID)
}

// --- GatewayEventRecorder interface ---

// Compile-time check: *Store must implement domain.GatewayEventRecorder.
var _ domain.GatewayEventRecorder = &Store{}

func TestStore_RecordEvent_StoreAndRetrieve(t *testing.T) {
	s := New()

	ev := domain.GatewayEvent{
		Timestamp:  time.Now(),
		Kind:       domain.EventHttpCompleted,
		RequestID:  99,
		Method:     "GET",
		Path:       "/me/player",
		Priority:   domain.PriorityInteractive,
		StatusCode: 200,
		DurationMs: 125,
		Snapshot: domain.GatewayStateSnapshot{
			TokensAvailable: 8,
			TokensMax:       10,
		},
	}

	s.RecordEvent(ev)

	cursor, events := s.ReadEventsFrom(0)
	require.Len(t, events, 1, "ReadEventsFrom should return the recorded event")
	assert.Equal(t, uint64(1), cursor)
	assert.Equal(t, domain.EventHttpCompleted, events[0].Kind)
	assert.Equal(t, uint64(99), events[0].RequestID)
	assert.Equal(t, 200, events[0].StatusCode)
}

func TestStore_ReadEventsFrom_IncrementalCursor(t *testing.T) {
	s := New()

	s.RecordEvent(domain.GatewayEvent{Kind: domain.EventRequestEntered, RequestID: 1})
	s.RecordEvent(domain.GatewayEvent{Kind: domain.EventTokenConsumed, RequestID: 2})
	s.RecordEvent(domain.GatewayEvent{Kind: domain.EventHttpCompleted, RequestID: 3})

	// First read.
	cursor, events := s.ReadEventsFrom(0)
	assert.Len(t, events, 3)

	// Add one more.
	s.RecordEvent(domain.GatewayEvent{Kind: domain.EventSemaphoreAcquired, RequestID: 4})

	// Incremental read from saved cursor.
	_, events2 := s.ReadEventsFrom(cursor)
	require.Len(t, events2, 1)
	assert.Equal(t, domain.EventSemaphoreAcquired, events2[0].Kind)
}
