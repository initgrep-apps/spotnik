// Package state — StateReader is a read-only view of the central Store.
// Pane constructors accept StateReader so they cannot inadvertently call write
// methods (Set*, Clear*, Stamp*). The root app holds *Store directly and
// remains the sole writer via Update() handlers.
package state

import (
	"time"

	"github.com/initgrep-apps/spotnik/internal/domain"
)

// StateReader is the read-only subset of *Store that panes are permitted to call.
// It contains every accessor method that pane View() and Update() bodies invoke,
// grouped by domain area.
//
// Write-only methods (Set*, Clear*, Stamp*, SetDevicesFetchedAt, etc.) are intentionally
// omitted — only the root App.Update() may call those on the concrete *Store.
type StateReader interface {
	// --- Playback ---

	// PlaybackState returns the current playback state, or nil if nothing is playing.
	PlaybackState() *domain.PlaybackState
	// ActiveDevice returns the currently active Spotify device, or nil if unknown.
	ActiveDevice() *domain.Device
	// UserID returns the Spotify user ID of the authenticated user.
	UserID() string

	// --- User Profile ---

	// UserProfile returns the full authenticated user profile.
	// Returns a zero-value UserProfile before profile is loaded.
	UserProfile() domain.UserProfile
	// IsPremium returns true only when the authenticated user's Product is "premium".
	// Returns false for free users, unknown tier, or when profile not yet loaded.
	IsPremium() bool

	// --- Queue ---

	// Queue returns the upcoming tracks in the user's play queue.
	Queue() []domain.Track

	// --- Library: Playlists ---

	// Playlists returns the user's saved playlists.
	Playlists() []domain.SimplePlaylist
	// PlaylistsTotal returns the total number of playlists (for pagination).
	PlaylistsTotal() int
	// PlaylistTracks returns cached tracks for a given playlist ID, or nil if not loaded.
	PlaylistTracks(playlistID string) []domain.Track
	// PlayingPlaylistID returns the Spotify playlist ID that is currently playing.
	PlayingPlaylistID() string

	// --- Library: Albums ---

	// SavedAlbums returns the user's saved albums.
	SavedAlbums() []domain.SavedAlbum
	// AlbumsLoaded returns true if saved albums have been fetched at least once.
	AlbumsLoaded() bool

	// --- Library: Liked Tracks ---

	// LikedTracks returns the user's liked tracks.
	LikedTracks() []domain.SavedTrack
	// LikedTotal returns the total number of liked tracks (for pagination).
	LikedTotal() int
	// LikedLoaded returns true if liked tracks have been fetched at least once.
	LikedLoaded() bool

	// --- Library: Recently Played ---

	// RecentlyPlayed returns the recently played track history.
	RecentlyPlayed() []domain.PlayHistory

	// --- Stats ---

	// TopTracks returns cached top tracks for the given time range.
	TopTracks(timeRange string) []domain.Track
	// TopArtists returns cached top artists for the given time range.
	TopArtists(timeRange string) []domain.FullArtist
	// StatsFetchedAt returns the time stats for the given time range were last
	// successfully fetched. Returns zero time if never fetched.
	StatsFetchedAt(timeRange string) time.Time

	// --- Devices ---

	// Devices returns the most recently fetched list of Spotify Connect devices.
	Devices() []domain.Device

	// --- Staleness ---

	// StatsStale returns true if stats for the given time range should be re-fetched.
	// Other staleness methods (PlaylistsStale, AlbumsStale, LikedTracksStale,
	// RecentlyPlayedStale, DevicesStale) are omitted because only handlers.go
	// calls them on the concrete *Store — no pane reads them via StateReader.
	StatsStale(timeRange string) bool

	// --- Fetching sentinels (read-only) ---

	// PlaylistsFetching returns true while a playlists fetch is in-flight.
	PlaylistsFetching() bool
	// AlbumsFetching returns true while a saved-albums fetch is in-flight.
	AlbumsFetching() bool
	// LikedFetching returns true while a liked-tracks fetch is in-flight.
	LikedFetching() bool
	// RecentFetching returns true while a recently-played fetch is in-flight.
	RecentFetching() bool

	// --- Fetched-at timestamps (read-only) ---

	// PlaylistsFetchedAt returns the time playlists were last successfully fetched.
	PlaylistsFetchedAt() time.Time
	// AlbumsFetchedAt returns the time saved albums were last successfully fetched.
	AlbumsFetchedAt() time.Time
	// LikedTracksFetchedAt returns the time liked tracks were last successfully fetched.
	LikedTracksFetchedAt() time.Time
	// RecentPlayedFetchedAt returns the time recently played was last successfully fetched.
	RecentPlayedFetchedAt() time.Time

	// --- Gateway event journal ---

	// ReadEventsFrom returns gateway events added since the given cursor.
	// Pass cursor=0 on the first call.
	ReadEventsFrom(cursor uint64) (uint64, []domain.GatewayEvent)

	// --- Podcasts ---

	// FollowedShows returns the user's followed podcast shows.
	FollowedShows() []domain.SavedShow
	// SavedEpisodes returns the user's saved podcast episodes.
	SavedEpisodes() []domain.SavedEpisode
	// ShowEpisodes returns the cached episodes for the selected show.
	ShowEpisodes() []domain.Episode
	// SelectedShowID returns the Spotify ID of the currently selected show.
	SelectedShowID() string
	// SelectedShow returns the full show data for the currently selected show, or nil.
	SelectedShow() *domain.Show

	// --- Throttle observability ---

	// IsThrottled returns true if the gateway is currently in a 429 backoff period.
	IsThrottled() bool
	// ThrottleRetryAfterSecs returns the Retry-After seconds from the last 429 response.
	ThrottleRetryAfterSecs() int
}

// Compile-time assertion: *Store must implement StateReader.
// This fails to compile if any method listed above is missing from *Store.
var _ StateReader = (*Store)(nil)
