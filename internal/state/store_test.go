package state

import (
	"fmt"
	"sync"
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

func TestStore_SetGetQueue(t *testing.T) {
	s := New()

	// Initially empty.
	assert.Empty(t, s.Queue())

	items := []domain.QueueItem{
		{Type: domain.QueueItemTypeTrack, Track: &domain.Track{ID: "q1", Name: "Save Your Tears", URI: "spotify:track:q1"}},
		{Type: domain.QueueItemTypeTrack, Track: &domain.Track{ID: "q2", Name: "Starboy", URI: "spotify:track:q2"}},
	}
	s.SetQueue(items)

	got := s.Queue()
	require.Len(t, got, 2)
	assert.Equal(t, domain.QueueItemTypeTrack, got[0].Type)
	assert.Equal(t, "Save Your Tears", got[0].Track.Name)
	assert.Equal(t, "Starboy", got[1].Track.Name)

	// Clear queue.
	s.SetQueue(nil)
	assert.Empty(t, s.Queue())
}

// TestStore_SetGetQueue_QueueItemType verifies Queue() returns QueueItem for tracks.
func TestStore_SetGetQueue_QueueItemType(t *testing.T) {
	s := New()

	items := []domain.QueueItem{
		{Type: domain.QueueItemTypeTrack, Track: &domain.Track{ID: "t1", Name: "Track One", URI: "spotify:track:t1"}},
	}
	s.SetQueue(items)

	got := s.Queue()
	require.Len(t, got, 1)
	assert.Equal(t, domain.QueueItemTypeTrack, got[0].Type)
	assert.NotNil(t, got[0].Track)
	assert.Nil(t, got[0].Episode)
}

// TestStore_Queue_TrackAndEpisode verifies Queue() returns both tracks and episodes.
func TestStore_Queue_TrackAndEpisode(t *testing.T) {
	s := New()

	items := []domain.QueueItem{
		{Type: domain.QueueItemTypeTrack, Track: &domain.Track{ID: "t1", Name: "Track", URI: "spotify:track:t1"}},
		{Type: domain.QueueItemTypeEpisode, Episode: &domain.Episode{ID: "e1", Name: "Episode", URI: "spotify:episode:e1", Show: &domain.Show{Name: "Show"}}},
	}
	s.SetQueue(items)

	got := s.Queue()
	require.Len(t, got, 2)
	assert.Equal(t, domain.QueueItemTypeTrack, got[0].Type)
	assert.Equal(t, "Track", got[0].Track.Name)
	assert.Equal(t, domain.QueueItemTypeEpisode, got[1].Type)
	assert.Equal(t, "Episode", got[1].Episode.Name)
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

// --- Podcast data accessors ---

func TestStore_SetGetFollowedShows(t *testing.T) {
	s := New()

	assert.Empty(t, s.FollowedShows())
	assert.False(t, s.FollowedShowsLoaded())

	shows := []domain.SavedShow{
		{AddedAt: "2024-01-01", Show: domain.Show{ID: "show-1", Name: "Podcast One", Publisher: "Pub A"}},
		{AddedAt: "2024-02-01", Show: domain.Show{ID: "show-2", Name: "Podcast Two", Publisher: "Pub B"}},
	}
	s.SetFollowedShows(shows)

	got := s.FollowedShows()
	assert.Len(t, got, 2)
	assert.Equal(t, "show-1", got[0].Show.ID)
	assert.Equal(t, "Podcast One", got[0].Show.Name)
	assert.True(t, s.FollowedShowsLoaded(), "FollowedShowsLoaded should be true after SetFollowedShows")
}

func TestStore_SetGetSavedEpisodes(t *testing.T) {
	s := New()

	assert.Empty(t, s.SavedEpisodes())
	assert.False(t, s.SavedEpisodesLoaded())

	episodes := []domain.SavedEpisode{
		{AddedAt: "2024-03-01", Episode: domain.Episode{ID: "ep-1", Name: "Episode One", DurationMs: 1800000}},
		{AddedAt: "2024-03-15", Episode: domain.Episode{ID: "ep-2", Name: "Episode Two", DurationMs: 2400000}},
	}
	s.SetSavedEpisodes(episodes)

	got := s.SavedEpisodes()
	assert.Len(t, got, 2)
	assert.Equal(t, "ep-1", got[0].Episode.ID)
	assert.Equal(t, "Episode One", got[0].Episode.Name)
	assert.True(t, s.SavedEpisodesLoaded(), "SavedEpisodesLoaded should be true after SetSavedEpisodes")
}

func TestStore_SetGetShowEpisodes(t *testing.T) {
	s := New()

	assert.Empty(t, s.ShowEpisodes())
	assert.False(t, s.ShowEpisodesLoaded())
	assert.Equal(t, 0, s.ShowEpisodesTotal())

	episodes := []domain.Episode{
		{ID: "ep-1", Name: "Episode One", DurationMs: 1800000, URI: "spotify:episode:ep-1"},
		{ID: "ep-2", Name: "Episode Two", DurationMs: 2400000, URI: "spotify:episode:ep-2"},
	}
	s.SetShowEpisodes(episodes)
	s.SetShowEpisodesTotal(42)

	got := s.ShowEpisodes()
	assert.Len(t, got, 2)
	assert.Equal(t, "ep-1", got[0].ID)
	assert.Equal(t, "Episode One", got[0].Name)
	assert.Equal(t, 42, s.ShowEpisodesTotal())
	assert.True(t, s.ShowEpisodesLoaded(), "ShowEpisodesLoaded should be true after SetShowEpisodes")
}

func TestStore_SetGetSelectedShowID(t *testing.T) {
	s := New()

	assert.Equal(t, "", s.SelectedShowID(), "initial selected show ID should be empty")

	s.SetSelectedShowID("show-abc")
	assert.Equal(t, "show-abc", s.SelectedShowID())

	s.SetSelectedShowID("show-xyz")
	assert.Equal(t, "show-xyz", s.SelectedShowID())
}

func TestStore_SetGetSelectedShow(t *testing.T) {
	s := New()

	assert.Nil(t, s.SelectedShow(), "initial selected show should be nil")

	show := &domain.Show{ID: "show-1", Name: "My Podcast", Publisher: "Pub A", TotalEpisodes: 20}
	s.SetSelectedShow(show)

	got := s.SelectedShow()
	assert.NotNil(t, got)
	assert.Equal(t, "show-1", got.ID)
	assert.Equal(t, "My Podcast", got.Name)
	assert.Equal(t, 20, got.TotalEpisodes)

	s.SetSelectedShow(nil)
	assert.Nil(t, s.SelectedShow(), "selected show should be nil after setting nil")
}

func TestStore_UserID_Default(t *testing.T) {
	s := New()
	assert.Equal(t, "", s.UserID(), "initial user ID should be empty")
}

func TestStore_SetGetUserID(t *testing.T) {
	s := New()
	s.SetUserProfile(domain.UserProfile{ID: "spotify-user-123"})
	assert.Equal(t, "spotify-user-123", s.UserID())
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
		{
			name:  "FollowedShowsFetchError",
			set:   func(s *Store) { s.SetFollowedShowsFetchError(testErr) },
			get:   func(s *Store) error { return s.FollowedShowsFetchError() },
			clear: func(s *Store) { s.ClearFollowedShowsFetchError() },
		},
		{
			name:  "SavedEpisodesFetchError",
			set:   func(s *Store) { s.SetSavedEpisodesFetchError(testErr) },
			get:   func(s *Store) error { return s.SavedEpisodesFetchError() },
			clear: func(s *Store) { s.ClearSavedEpisodesFetchError() },
		},
		{
			name:  "ShowEpisodesFetchError",
			set:   func(s *Store) { s.SetShowEpisodesFetchError(testErr) },
			get:   func(s *Store) error { return s.ShowEpisodesFetchError() },
			clear: func(s *Store) { s.ClearShowEpisodesFetchError() },
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

	// Podcast domains: initially zero for all.
	assert.True(t, s.FollowedShowsFetchedAt().IsZero(), "FollowedShowsFetchedAt initially zero")
	assert.True(t, s.SavedEpisodesFetchedAt().IsZero(), "SavedEpisodesFetchedAt initially zero")
	assert.True(t, s.ShowEpisodesFetchedAt().IsZero(), "ShowEpisodesFetchedAt initially zero")

	// Nil/empty sets do NOT stamp fetchedAt.
	s.SetFollowedShows(nil)
	assert.True(t, s.FollowedShowsFetchedAt().IsZero(), "SetFollowedShows(nil) must not stamp fetchedAt")

	s.SetSavedEpisodes(nil)
	assert.True(t, s.SavedEpisodesFetchedAt().IsZero(), "SetSavedEpisodes(nil) must not stamp fetchedAt")

	s.SetShowEpisodes(nil)
	assert.True(t, s.ShowEpisodesFetchedAt().IsZero(), "SetShowEpisodes(nil) must not stamp fetchedAt")

	// Non-empty sets DO stamp fetchedAt.
	s.SetFollowedShows([]domain.SavedShow{{AddedAt: "2024-01-01", Show: domain.Show{ID: "s1"}}})
	assert.False(t, s.FollowedShowsFetchedAt().IsZero(), "SetFollowedShows(nonEmpty) must stamp fetchedAt")

	s.SetSavedEpisodes([]domain.SavedEpisode{{AddedAt: "2024-01-01", Episode: domain.Episode{ID: "e1"}}})
	assert.False(t, s.SavedEpisodesFetchedAt().IsZero(), "SetSavedEpisodes(nonEmpty) must stamp fetchedAt")

	s.SetShowEpisodes([]domain.Episode{{ID: "e1", Name: "Ep One"}})
	assert.False(t, s.ShowEpisodesFetchedAt().IsZero(), "SetShowEpisodes(nonEmpty) must stamp fetchedAt")
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

// --- Podcast staleness ---

func TestStore_FollowedShowsStale_NeverFetched(t *testing.T) {
	s := New()
	assert.True(t, s.FollowedShowsStale(), "FollowedShowsStale should be true when never fetched")
}

func TestStore_FollowedShowsStale_WithinTTL(t *testing.T) {
	s := New()
	s.SetFollowedShows([]domain.SavedShow{{AddedAt: "2024-01-01", Show: domain.Show{ID: "s1", Name: "Test Show"}}})
	assert.False(t, s.FollowedShowsStale(), "FollowedShowsStale should be false within TTL")
}

func TestStore_FollowedShowsStale_ExpiredTTL(t *testing.T) {
	s := New()
	s.mu.Lock()
	s.followedShowsFetchedAt = time.Now().Add(-(FollowedShowsTTL + time.Second))
	s.mu.Unlock()
	assert.True(t, s.FollowedShowsStale(), "FollowedShowsStale should be true after TTL")
}

func TestStore_SavedEpisodesStale_NeverFetched(t *testing.T) {
	s := New()
	assert.True(t, s.SavedEpisodesStale(), "SavedEpisodesStale should be true when never fetched")
}

func TestStore_SavedEpisodesStale_WithinTTL(t *testing.T) {
	s := New()
	s.SetSavedEpisodes([]domain.SavedEpisode{{AddedAt: "2024-01-01", Episode: domain.Episode{ID: "e1", Name: "Test Ep"}}})
	assert.False(t, s.SavedEpisodesStale(), "SavedEpisodesStale should be false within TTL")
}

func TestStore_SavedEpisodesStale_ExpiredTTL(t *testing.T) {
	s := New()
	s.mu.Lock()
	s.savedEpisodesFetchedAt = time.Now().Add(-(SavedEpisodesTTL + time.Second))
	s.mu.Unlock()
	assert.True(t, s.SavedEpisodesStale(), "SavedEpisodesStale should be true after TTL")
}

func TestStore_ShowEpisodesStale_NeverFetched(t *testing.T) {
	s := New()
	assert.True(t, s.ShowEpisodesStale(), "ShowEpisodesStale should be true when never fetched")
}

func TestStore_ShowEpisodesStale_WithinTTL(t *testing.T) {
	s := New()
	s.SetShowEpisodes([]domain.Episode{{ID: "e1", Name: "Test Ep"}})
	assert.False(t, s.ShowEpisodesStale(), "ShowEpisodesStale should be false within TTL")
}

func TestStore_ShowEpisodesStale_ExpiredTTL(t *testing.T) {
	s := New()
	s.mu.Lock()
	s.showEpisodesFetchedAt = time.Now().Add(-(ShowEpisodesTTL + time.Second))
	s.mu.Unlock()
	assert.True(t, s.ShowEpisodesStale(), "ShowEpisodesStale should be true after TTL")
}

// --- Podcast TTL constants ---

func TestStore_PodcastTTL_Constants(t *testing.T) {
	// All podcast TTLs must be positive.
	assert.Positive(t, FollowedShowsTTL, "FollowedShowsTTL must be positive")
	assert.Positive(t, SavedEpisodesTTL, "SavedEpisodesTTL must be positive")
	assert.Positive(t, ShowEpisodesTTL, "ShowEpisodesTTL must be positive")

	// All podcast TTLs must be distinct from each other and from existing TTLs.
	ttls := map[string]time.Duration{
		"FollowedShowsTTL": FollowedShowsTTL,
		"SavedEpisodesTTL": SavedEpisodesTTL,
		"ShowEpisodesTTL":  ShowEpisodesTTL,
	}
	for name, ttl := range ttls {
		assert.Equal(t, 5*time.Minute, ttl, "%s should be 5 minutes", name)
	}
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

// --- Podcast fetching sentinels ---

func TestStore_FollowedShowsFetching_DefaultFalse(t *testing.T) {
	s := New()
	assert.False(t, s.FollowedShowsFetching(), "FollowedShowsFetching should be false initially")
}

func TestStore_SetFollowedShowsFetching_SetsAndClears(t *testing.T) {
	s := New()
	s.SetFollowedShowsFetching(true)
	assert.True(t, s.FollowedShowsFetching(), "FollowedShowsFetching should be true after Set(true)")
	s.SetFollowedShowsFetching(false)
	assert.False(t, s.FollowedShowsFetching(), "FollowedShowsFetching should be false after Set(false)")
}

func TestStore_SavedEpisodesFetching_DefaultFalse(t *testing.T) {
	s := New()
	assert.False(t, s.SavedEpisodesFetching(), "SavedEpisodesFetching should be false initially")
}

func TestStore_SetSavedEpisodesFetching_SetsAndClears(t *testing.T) {
	s := New()
	s.SetSavedEpisodesFetching(true)
	assert.True(t, s.SavedEpisodesFetching())
	s.SetSavedEpisodesFetching(false)
	assert.False(t, s.SavedEpisodesFetching())
}

func TestStore_ShowEpisodesFetching_DefaultFalse(t *testing.T) {
	s := New()
	assert.False(t, s.ShowEpisodesFetching(), "ShowEpisodesFetching should be false initially")
}

func TestStore_SetShowEpisodesFetching_SetsAndClears(t *testing.T) {
	s := New()
	s.SetShowEpisodesFetching(true)
	assert.True(t, s.ShowEpisodesFetching())
	s.SetShowEpisodesFetching(false)
	assert.False(t, s.ShowEpisodesFetching())
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

// TestStore_SetGetUserProfile verifies the round-trip for SetUserProfile / UserProfile
// and that UserID() delegates to userProfile.ID for call-site compatibility.
func TestStore_SetGetUserProfile(t *testing.T) {
	s := New()

	// Zero state: UserID and UserProfile must return safe defaults.
	assert.Equal(t, "", s.UserID(), "UserID should be empty before profile is loaded")
	assert.Equal(t, domain.UserProfile{}, s.UserProfile(), "UserProfile should be zero-value before set")

	p := domain.UserProfile{
		ID:          "user123",
		DisplayName: "Test User",
		Product:     "premium",
		Country:     "DE",
	}
	s.SetUserProfile(p)

	assert.Equal(t, "user123", s.UserID(), "UserID must delegate to userProfile.ID")
	got := s.UserProfile()
	assert.Equal(t, "user123", got.ID)
	assert.Equal(t, "Test User", got.DisplayName)
	assert.Equal(t, "premium", got.Product)
	assert.Equal(t, "DE", got.Country)
}

// TestStore_IsPremium verifies that IsPremium returns true only for the "premium" product tier.
func TestStore_IsPremium(t *testing.T) {
	tests := []struct {
		name    string
		product string
		want    bool
	}{
		{name: "premium user", product: "premium", want: true},
		{name: "free user", product: "free", want: false},
		{name: "empty product (not yet loaded)", product: "", want: false},
		{name: "unexpected tier", product: "open", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New()
			s.SetUserProfile(domain.UserProfile{ID: "u1", Product: tt.product})
			assert.Equal(t, tt.want, s.IsPremium())
		})
	}
}

// TestStore_UserProfile_ConcurrentAccess runs SetUserProfile and IsPremium/UserProfile
// concurrently under the race detector to catch any missing mutex coverage.
// Run with: go test -race ./internal/state/...
func TestStore_UserProfile_ConcurrentAccess(t *testing.T) {
	s := New()
	var wg sync.WaitGroup

	// Writer: repeatedly set the user profile.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 100 {
			s.SetUserProfile(domain.UserProfile{ID: "u1", Product: "premium"})
		}
	}()

	// Reader A: read IsPremium while writes are in-flight.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 100 {
			_ = s.IsPremium()
		}
	}()

	// Reader B: read UserProfile while writes are in-flight.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 100 {
			_ = s.UserProfile()
		}
	}()

	wg.Wait()
	// No assertion needed — the race detector catches data races if locking is broken.
}

// --- Story 267: Like/Unlike core infrastructure ---

// TestStore_IsTrackLiked verifies IsTrackLiked returns true for tracks in
// likedTracks and false otherwise (including for an empty store).
func TestStore_IsTrackLiked(t *testing.T) {
	t.Parallel()
	s := New()

	// Empty store: nothing liked.
	assert.False(t, s.IsTrackLiked("track-1"), "IsTrackLiked on empty store should be false")
	assert.False(t, s.IsTrackLiked(""), "IsTrackLiked on empty ID should be false")

	// Populate liked tracks.
	s.SetLikedTracks([]api.SavedTrack{
		{AddedAt: "2024-02-20T14:00:00Z", Track: api.Track{ID: "track-1", Name: "Blinding Lights"}},
		{AddedAt: "2024-02-21T14:00:00Z", Track: api.Track{ID: "track-2", Name: "Save Your Tears"}},
	})

	assert.True(t, s.IsTrackLiked("track-1"), "IsTrackLiked should be true for a liked track")
	assert.True(t, s.IsTrackLiked("track-2"), "IsTrackLiked should be true for another liked track")
	assert.False(t, s.IsTrackLiked("track-999"), "IsTrackLiked should be false for a non-liked track")
}

// TestStore_SetLikedTracksRebuildsLikedSet verifies that SetLikedTracks rebuilds
// the O(1) likedSet from the incoming tracks. This prevents stale liked status
// when the store is re-populated after a re-fetch.
func TestStore_SetLikedTracksRebuildsLikedSet(t *testing.T) {
	t.Parallel()
	s := New()

	// Initial population.
	s.SetLikedTracks([]api.SavedTrack{
		{Track: api.Track{ID: "track-1"}},
		{Track: api.Track{ID: "track-2"}},
	})
	assert.True(t, s.IsTrackLiked("track-1"))
	assert.True(t, s.IsTrackLiked("track-2"))

	// Re-set with a different set — track-2 should no longer be liked.
	s.SetLikedTracks([]api.SavedTrack{
		{Track: api.Track{ID: "track-3"}},
	})
	assert.True(t, s.IsTrackLiked("track-3"), "newly-set track should be liked")
	assert.False(t, s.IsTrackLiked("track-1"), "track-1 should no longer be liked after rebuild")
	assert.False(t, s.IsTrackLiked("track-2"), "track-2 should no longer be liked after rebuild")
}

// TestStore_AddLikedTrack verifies AddLikedTrack prepends to likedTracks and
// updates likedSet so IsTrackLiked returns true immediately.
func TestStore_AddLikedTrack(t *testing.T) {
	t.Parallel()
	s := New()

	// Start with one liked track.
	s.SetLikedTracks([]api.SavedTrack{
		{AddedAt: "2024-01-01T00:00:00Z", Track: api.Track{ID: "track-1", Name: "First"}},
	})
	// likedTotal is set by the Loaded handler (API total), not by SetLikedTracks.
	s.SetLikedTotal(1)

	// Add a new track.
	newTrack := api.Track{ID: "track-2", Name: "Second", URI: "spotify:track:track-2"}
	s.AddLikedTrack(newTrack)

	// IsTrackLiked should reflect the new state immediately (O(1) lookup).
	assert.True(t, s.IsTrackLiked("track-2"), "newly-added track should be liked")

	// The new track should be prepended to likedTracks.
	tracks := s.LikedTracks()
	require.Len(t, tracks, 2, "AddLikedTrack should prepend to likedTracks")
	assert.Equal(t, "track-2", tracks[0].Track.ID, "newly-added track should be at index 0 (prepend)")
	assert.Equal(t, "track-1", tracks[1].Track.ID, "existing track should shift down")

	// likedTotal should be updated.
	assert.Equal(t, 2, s.LikedTotal(), "AddLikedTrack should update likedTotal")

	// AddedAt should be set in RFC3339 format.
	assert.NotEmpty(t, tracks[0].AddedAt, "AddLikedTrack should set AddedAt")

	// Adding a track that is already liked should be idempotent (set membership).
	s.AddLikedTrack(newTrack)
	tracks = s.LikedTracks()
	assert.Len(t, tracks, 2, "AddLikedTrack of already-liked track should not duplicate the entry")
}

// TestStore_RemoveLikedTrack verifies RemoveLikedTrack removes from both
// likedTracks slice and likedSet so IsTrackLiked returns false.
func TestStore_RemoveLikedTrack(t *testing.T) {
	t.Parallel()
	s := New()

	s.SetLikedTracks([]api.SavedTrack{
		{Track: api.Track{ID: "track-1", Name: "First"}},
		{Track: api.Track{ID: "track-2", Name: "Second"}},
		{Track: api.Track{ID: "track-3", Name: "Third"}},
	})
	s.SetLikedTotal(3)
	require.True(t, s.IsTrackLiked("track-2"))

	// Remove the middle track.
	s.RemoveLikedTrack("track-2")

	// likedSet should reflect removal immediately.
	assert.False(t, s.IsTrackLiked("track-2"), "removed track should not be liked")

	// likedTracks should no longer contain the removed track.
	tracks := s.LikedTracks()
	require.Len(t, tracks, 2, "RemoveLikedTrack should remove from likedTracks")
	assert.Equal(t, "track-1", tracks[0].Track.ID)
	assert.Equal(t, "track-3", tracks[1].Track.ID)

	// likedTotal should be updated.
	assert.Equal(t, 2, s.LikedTotal(), "RemoveLikedTrack should update likedTotal")

	// Removing a track that is not liked should be a no-op (no panic).
	s.RemoveLikedTrack("track-999")
	tracks = s.LikedTracks()
	assert.Len(t, tracks, 2, "removing a non-liked track should be a no-op")

	// Removing from empty store should not panic.
	s2 := New()
	s2.RemoveLikedTrack("track-1")
	assert.False(t, s2.IsTrackLiked("track-1"))
}
