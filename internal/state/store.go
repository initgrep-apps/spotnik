// Package state provides the central Store — the single source of truth for all
// application data. Panes read from the store via accessor methods. Only the root
// app.Update() writes to the store via data-carrying Msg payloads, never Commands,
// pane Update(), or View() directly.
package state

import (
	"sync"
	"time"

	"github.com/initgrep-apps/spotnik/internal/domain"
)

// Staleness classification:
//
// Stable data (staleness-gated, 5-10min TTL): playlists, albums, liked tracks, stats.
//   These only change when the user acts within Spotnik or very slowly.
//   TTL prevents redundant API calls on repeated pane navigation.
//
// Volatile data (staleness-gated, short TTL): devices, recently played.
//   These change externally (devices appear/disappear, playback advances).
//   Short TTL balances freshness with API efficiency.
//
// Real-time data (polled on tick, no TTL): playback state, queue.
//   Overwritten every tick cycle with adaptive polling intervals.

// TTL constants define how long each data domain is considered fresh.
// After the TTL expires, Update() should trigger a re-fetch from the Spotify API.
const (
	// PlaylistsTTL is the cache lifetime for the user's playlist list.
	PlaylistsTTL = 5 * time.Minute
	// AlbumsTTL is the cache lifetime for the user's saved albums.
	AlbumsTTL = 5 * time.Minute
	// LikedTracksTTL is the cache lifetime for the user's liked tracks.
	LikedTracksTTL = 5 * time.Minute
	// RecentlyPlayedTTL is the cache lifetime for recently played tracks.
	// Shorter than library data because it changes with every playback event.
	RecentlyPlayedTTL = 2 * time.Minute
	// StatsTTL is the cache lifetime for user stats (top tracks/artists).
	// Long because Spotify updates these slowly.
	StatsTTL = 10 * time.Minute
	// DevicesTTL is the cache lifetime for the available device list.
	// Short cooldown prevents rapid-fire API calls while ensuring fresh data on user request.
	DevicesTTL = 5 * time.Second
	// FollowedShowsTTL is the cache lifetime for the user's followed podcast shows.
	FollowedShowsTTL = 5 * time.Minute
	// SavedEpisodesTTL is the cache lifetime for the user's saved podcast episodes.
	SavedEpisodesTTL = 5 * time.Minute
	// ShowEpisodesTTL is the cache lifetime for a specific show's episode list.
	ShowEpisodesTTL = 5 * time.Minute
)

// IsStale returns true if fetchedAt is zero (never fetched) or older than ttl.
// Use this to decide whether to re-fetch cached data from the Spotify API.
func IsStale(fetchedAt time.Time, ttl time.Duration) bool {
	return fetchedAt.IsZero() || time.Since(fetchedAt) > ttl
}

// Store is the central application state. All panes read from here; only
// the root app.Update() writes to it via Msg payloads. Fields are never accessed
// directly — use the accessor methods to ensure safe concurrent access.
type Store struct {
	mu            sync.RWMutex
	playbackState *domain.PlaybackState
	activeDevice  *domain.Device

	// Device list — the most recent list of Spotify Connect devices returned by the API.
	// Populated by the DevicesLoadedMsg handler so the staleness gate can return cached
	// data on subsequent FetchDevicesRequestMsg calls without a redundant API round-trip.
	devices []domain.Device

	// Library data
	playlists      []domain.SimplePlaylist
	playlistsTotal int
	savedAlbums    []domain.SavedAlbum
	likedTracks    []domain.SavedTrack
	likedTotal     int
	// likedSet is an O(1) lookup map of trackID → liked. It is rebuilt in
	// SetLikedTracks and kept in sync by AddLikedTrack/RemoveLikedTrack so
	// panes can render the ♥ indicator without scanning the slice.
	likedSet       map[string]bool
	recentlyPlayed []domain.PlayHistory

	// Staleness tracking — set to time.Now() on successful data write.
	playlistsFetchedAt    time.Time
	albumsFetchedAt       time.Time
	likedTracksFetchedAt  time.Time
	recentPlayedFetchedAt time.Time
	statsFetchedAt        map[string]time.Time // keyed by time range
	devicesFetchedAt      time.Time

	// Fetching sentinels — true while an in-flight fetch is outstanding.
	// Checked in Update() staleness gates to prevent TOCTOU duplicate fetches:
	// between the staleness check and fetch completion a second identical request
	// could also pass the gate. Set to true just before dispatch, cleared in the
	// corresponding loaded-message handler. Paginated requests (Offset > 0) bypass
	// these guards since they are explicitly page-by-page continuation fetches.
	playlistsFetching bool
	albumsFetching    bool
	likedFetching     bool
	recentFetching    bool
	statsFetching     map[string]bool // keyed by time range
	devicesFetching   bool

	// Queue data
	queue []domain.QueueItem

	// Stats data: top tracks and top artists keyed by time range.
	// Ranges: "short_term", "medium_term", "long_term".
	// NOTE: cached per range and re-fetched after StatsTTL (10 min) via staleness check in Update().
	topTracks  map[string][]domain.Track
	topArtists map[string][]domain.FullArtist

	// Playlist Manager data: tracks for each playlist keyed by playlist ID.
	playlistTracks map[string][]domain.Track

	// userProfile is the authenticated user's full Spotify profile.
	// Set once at startup after GET /v1/me succeeds.
	userProfile domain.UserProfile

	// playingPlaylistID is the Spotify playlist ID that is currently playing.
	// Used by PlaylistsPane to render the ▶ indicator next to the active playlist.
	playingPlaylistID string

	// eventLog records gateway lifecycle events for the event journal.
	eventLog *GatewayEventLog

	// throttle holds rate-limit observability state updated by the API Gateway.
	// The UI status bar reads these to show a "Rate limited" indicator.
	throttle struct {
		isThrottled    bool
		retryAfterSecs int
		last429At      time.Time
	}

	// Error state — one per data-fetching feature.
	// Set by Update() handlers on failure, cleared on successful retry.
	statsError           error
	devicesError         error
	queueError           error
	playlistsFetchErr    error // playlists list fetch
	albumsFetchErr       error // saved albums fetch
	likedTracksFetchErr  error // liked tracks fetch
	recentPlayedFetchErr error // recently played fetch
	playlistsError       error // playlist manager (tracks, mutations)

	// Podcast data
	followedShows     []domain.SavedShow
	savedEpisodes     []domain.SavedEpisode
	showEpisodes      []domain.Episode
	showEpisodesTotal int
	selectedShowID    string
	selectedShow      *domain.Show

	// Podcast staleness tracking
	followedShowsFetchedAt time.Time
	savedEpisodesFetchedAt time.Time
	showEpisodesFetchedAt  time.Time

	// Podcast fetching sentinels
	followedShowsFetching bool
	savedEpisodesFetching bool
	showEpisodesFetching  bool

	// Podcast error state
	followedShowsFetchErr error
	savedEpisodesFetchErr error
	showEpisodesFetchErr  error
}

// New returns an empty Store with no playback state.
// statsFetchedAt is pre-allocated so callers never encounter a nil map panic
// when reading stats staleness before any fetch has completed.
func New() *Store {
	return &Store{
		eventLog:       NewGatewayEventLog(defaultEventLogCapacity),
		statsFetchedAt: make(map[string]time.Time),
		statsFetching:  make(map[string]bool),
		likedSet:       make(map[string]bool),
	}
}

// PlaybackState returns the current playback state, or nil if nothing is playing.
func (s *Store) PlaybackState() *domain.PlaybackState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.playbackState
}

// UserID returns the Spotify user ID. Returns "" before profile is loaded.
// Preserved for call-site compatibility — delegates to userProfile.ID.
func (s *Store) UserID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.userProfile.ID
}

// UserProfile returns the full authenticated user profile.
// Returns a zero-value UserProfile before profile is loaded.
func (s *Store) UserProfile() domain.UserProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.userProfile
}

// SetUserProfile stores the authenticated user's full Spotify profile.
// Called once at startup after GET /v1/me succeeds.
func (s *Store) SetUserProfile(p domain.UserProfile) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.userProfile = p
}

// IsPremium returns true only when Product == "premium".
// Returns false for free users, unknown tier, or when profile not yet loaded.
func (s *Store) IsPremium() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.userProfile.Product == "premium"
}

// SetPlaybackState updates the playback state. Pass nil to clear (204 response).
// Also updates the active device from the state's Device field.
func (s *Store) SetPlaybackState(state *domain.PlaybackState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.playbackState = state
	if state != nil {
		s.activeDevice = state.Device
	}
}

// ActiveDevice returns the currently active Spotify device, or nil if unknown.
func (s *Store) ActiveDevice() *domain.Device {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeDevice
}

// SetActiveDevice updates the active device independently of playback state.
func (s *Store) SetActiveDevice(device *domain.Device) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeDevice = device
}

// Devices returns the most recently fetched list of Spotify Connect devices,
// or nil if the list has not been fetched yet.
func (s *Store) Devices() []domain.Device {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.devices
}

// SetDevices replaces the cached device list. Called by app.Update() after a
// successful DevicesLoadedMsg.
func (s *Store) SetDevices(devices []domain.Device) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.devices = devices
}

// Playlists returns the user's saved playlists.
func (s *Store) Playlists() []domain.SimplePlaylist {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.playlists
}

// SetPlaylists updates the saved playlists in the store.
// fetchedAt is only stamped when the slice is non-empty, preventing an empty
// (nil-client or error-fallback) response from resetting the TTL and blocking
// retries for the full cache duration.
func (s *Store) SetPlaylists(playlists []domain.SimplePlaylist) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.playlists = playlists
	if len(playlists) > 0 {
		s.playlistsFetchedAt = time.Now()
	}
}

// PlaylistsTotal returns the total number of playlists (for pagination).
func (s *Store) PlaylistsTotal() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.playlistsTotal
}

// SetPlaylistsTotal updates the total playlists count for pagination.
func (s *Store) SetPlaylistsTotal(total int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.playlistsTotal = total
}

// SavedAlbums returns the user's saved albums.
func (s *Store) SavedAlbums() []domain.SavedAlbum {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.savedAlbums
}

// SetSavedAlbums updates the saved albums in the store.
// fetchedAt is only stamped when the slice is non-empty to avoid resetting
// the TTL on empty/error responses and blocking retries prematurely.
func (s *Store) SetSavedAlbums(albums []domain.SavedAlbum) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.savedAlbums = albums
	if len(albums) > 0 {
		s.albumsFetchedAt = time.Now()
	}
}

// AlbumsLoaded returns true if saved albums have been fetched at least once.
// Replaces the former albumsLoaded boolean sentinel — derived from albumsFetchedAt.
func (s *Store) AlbumsLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return !s.albumsFetchedAt.IsZero()
}

// LikedTracks returns the user's liked tracks.
func (s *Store) LikedTracks() []domain.SavedTrack {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.likedTracks
}

// SetLikedTracks updates the liked tracks in the store.
// fetchedAt is only stamped when the slice is non-empty to avoid resetting
// the TTL on empty/error responses and blocking retries prematurely.
// It also rebuilds likedSet from the incoming tracks so IsTrackLiked reflects
// the new state immediately (including removals when the slice shrinks).
func (s *Store) SetLikedTracks(tracks []domain.SavedTrack) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.likedTracks = tracks
	if s.likedSet == nil {
		s.likedSet = make(map[string]bool, len(tracks))
	} else {
		// Clear and re-use the underlying map to avoid allocation churn.
		for k := range s.likedSet {
			delete(s.likedSet, k)
		}
	}
	for _, st := range tracks {
		if st.Track.ID != "" {
			s.likedSet[st.Track.ID] = true
		}
	}
	if len(tracks) > 0 {
		s.likedTracksFetchedAt = time.Now()
	}
}

// IsTrackLiked returns true if the given track ID is present in the user's
// liked tracks. This is an O(1) lookup against likedSet. Returns false for
// an empty trackID or when no liked tracks have been loaded.
func (s *Store) IsTrackLiked(trackID string) bool {
	if trackID == "" {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.likedSet[trackID]
}

// AddLikedTrack optimistically prepends a track to likedTracks and marks it as
// liked in likedSet. Used by the like/unlike toggle flow to update the UI
// before the API call confirms. The track is wrapped in a SavedTrack with
// AddedAt set to the current time in RFC3339 format.
func (s *Store) AddLikedTrack(track domain.Track) {
	if track.ID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.likedSet == nil {
		s.likedSet = make(map[string]bool)
	}
	// Idempotent: if already liked, do not duplicate the slice entry.
	if s.likedSet[track.ID] {
		return
	}
	s.likedSet[track.ID] = true
	s.likedTracks = append([]domain.SavedTrack{{
		AddedAt: time.Now().Format(time.RFC3339),
		Track:   track,
	}}, s.likedTracks...)
	s.likedTotal++
}

// RemoveLikedTrack optimistically removes a track from likedTracks and likedSet.
// Used by the like/unlike toggle flow to update the UI before the API call
// confirms. If the track is not present this is a no-op (safe to call on a
// non-liked trackID, e.g. during rollback of an optimistic update).
func (s *Store) RemoveLikedTrack(trackID string) {
	if trackID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.likedSet == nil {
		return
	}
	if _, ok := s.likedSet[trackID]; !ok {
		return
	}
	delete(s.likedSet, trackID)
	// Remove from the likedTracks slice.
	for i, st := range s.likedTracks {
		if st.Track.ID == trackID {
			s.likedTracks = append(s.likedTracks[:i], s.likedTracks[i+1:]...)
			break
		}
	}
	if s.likedTotal > 0 {
		s.likedTotal--
	}
}

// LikedTotal returns the total number of liked tracks (for pagination).
func (s *Store) LikedTotal() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.likedTotal
}

// SetLikedTotal updates the total liked tracks count for pagination.
func (s *Store) SetLikedTotal(total int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.likedTotal = total
}

// LikedLoaded returns true if liked tracks have been fetched at least once.
// Replaces the former likedLoaded boolean sentinel — derived from likedTracksFetchedAt.
func (s *Store) LikedLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return !s.likedTracksFetchedAt.IsZero()
}

// RecentlyPlayed returns the recently played track history.
func (s *Store) RecentlyPlayed() []domain.PlayHistory {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.recentlyPlayed
}

// SetRecentlyPlayed updates the recently played history in the store.
// fetchedAt is only stamped when the slice is non-empty to avoid resetting
// the TTL on empty/error responses and blocking retries prematurely.
func (s *Store) SetRecentlyPlayed(items []domain.PlayHistory) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recentlyPlayed = items
	if len(items) > 0 {
		s.recentPlayedFetchedAt = time.Now()
	}
}

// Queue returns the upcoming items in the user's play queue.
// Each item is either a track or an episode, determined by its Type field.
func (s *Store) Queue() []domain.QueueItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.queue
}

// SetQueue updates the queue items in the store.
func (s *Store) SetQueue(items []domain.QueueItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queue = items
}

// TopTracks returns the cached top tracks for the given time range,
// or nil if that range has not been fetched yet.
// timeRange should be "short_term", "medium_term", or "long_term".
func (s *Store) TopTracks(timeRange string) []domain.Track {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.topTracks == nil {
		return nil
	}
	return s.topTracks[timeRange]
}

// SetTopTracks caches top tracks for a specific time range in the store.
// It does NOT stamp statsFetchedAt — call StampStatsFetchedAt after both
// SetTopTracks and SetTopArtists succeed so the range is only marked fresh
// when both datasets are written (avoids partial-data false-fresh state).
func (s *Store) SetTopTracks(timeRange string, tracks []domain.Track) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.topTracks == nil {
		s.topTracks = make(map[string][]domain.Track)
	}
	s.topTracks[timeRange] = tracks
}

// TopArtists returns the cached top artists for the given time range,
// or nil if that range has not been fetched yet.
// timeRange should be "short_term", "medium_term", or "long_term".
func (s *Store) TopArtists(timeRange string) []domain.FullArtist {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.topArtists == nil {
		return nil
	}
	return s.topArtists[timeRange]
}

// SetTopArtists caches top artists for a specific time range in the store.
// It does NOT stamp statsFetchedAt — call StampStatsFetchedAt after both
// SetTopTracks and SetTopArtists succeed so the range is only marked fresh
// when both datasets are written (avoids partial-data false-fresh state).
func (s *Store) SetTopArtists(timeRange string, artists []domain.FullArtist) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.topArtists == nil {
		s.topArtists = make(map[string][]domain.FullArtist)
	}
	s.topArtists[timeRange] = artists
}

// StampStatsFetchedAt records the time when stats for a time range were fully
// loaded. Call this once after both SetTopTracks and SetTopArtists succeed so
// that StatsStale only returns false when both datasets are present.
func (s *Store) StampStatsFetchedAt(timeRange string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statsFetchedAt[timeRange] = time.Now()
}

// PlaylistTracks returns the cached tracks for a given playlist ID,
// or nil if the playlist has not been loaded yet.
func (s *Store) PlaylistTracks(playlistID string) []domain.Track {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.playlistTracks == nil {
		return nil
	}
	return s.playlistTracks[playlistID]
}

// SetPlaylistTracks caches the tracks for a specific playlist in the store.
func (s *Store) SetPlaylistTracks(playlistID string, tracks []domain.Track) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.playlistTracks == nil {
		s.playlistTracks = make(map[string][]domain.Track)
	}
	s.playlistTracks[playlistID] = tracks
}

// PlayingPlaylistID returns the Spotify playlist ID that is currently playing.
// Returns "" if no playlist is active or the ID is unknown.
func (s *Store) PlayingPlaylistID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.playingPlaylistID
}

// SetPlayingPlaylistID records the currently playing playlist ID.
// This is set by the root app when a PlayContextMsg plays a playlist.
func (s *Store) SetPlayingPlaylistID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.playingPlaylistID = id
}

// --- Staleness tracking accessors ---

// PlaylistsFetchedAt returns the time when playlists were last successfully fetched.
// Returns the zero time if playlists have never been fetched.
func (s *Store) PlaylistsFetchedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.playlistsFetchedAt
}

// AlbumsFetchedAt returns the time when saved albums were last successfully fetched.
// Returns the zero time if albums have never been fetched.
func (s *Store) AlbumsFetchedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.albumsFetchedAt
}

// LikedTracksFetchedAt returns the time when liked tracks were last successfully fetched.
// Returns the zero time if liked tracks have never been fetched.
func (s *Store) LikedTracksFetchedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.likedTracksFetchedAt
}

// RecentPlayedFetchedAt returns the time when recently played was last successfully fetched.
// Returns the zero time if recently played has never been fetched.
func (s *Store) RecentPlayedFetchedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.recentPlayedFetchedAt
}

// StatsFetchedAt returns the time when stats for the given time range were last fetched.
// Returns the zero time if that range has never been fetched.
func (s *Store) StatsFetchedAt(timeRange string) time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.statsFetchedAt[timeRange]
}

// DevicesFetchedAt returns the time when the device list was last successfully fetched.
// Returns the zero time if devices have never been fetched.
func (s *Store) DevicesFetchedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.devicesFetchedAt
}

// SetDevicesFetchedAt stamps the time when devices were last successfully loaded.
// Called by root app.Update() after a successful DevicesLoadedMsg.
func (s *Store) SetDevicesFetchedAt(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.devicesFetchedAt = t
}

// SetPlaylistsFetchedAt stamps the time when playlists were last successfully loaded.
// Used by tests and import flows that need precise staleness control.
func (s *Store) SetPlaylistsFetchedAt(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.playlistsFetchedAt = t
}

// SetAlbumsFetchedAt stamps the time when saved albums were last successfully loaded.
func (s *Store) SetAlbumsFetchedAt(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.albumsFetchedAt = t
}

// SetLikedTracksFetchedAt stamps the time when liked tracks were last successfully loaded.
func (s *Store) SetLikedTracksFetchedAt(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.likedTracksFetchedAt = t
}

// SetRecentPlayedFetchedAt stamps the time when recently played was last successfully loaded.
func (s *Store) SetRecentPlayedFetchedAt(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recentPlayedFetchedAt = t
}

// --- TTL-based staleness convenience methods ---

// PlaylistsStale returns true if the playlists list is stale and should be re-fetched.
func (s *Store) PlaylistsStale() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return IsStale(s.playlistsFetchedAt, PlaylistsTTL)
}

// AlbumsStale returns true if the saved albums are stale and should be re-fetched.
func (s *Store) AlbumsStale() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return IsStale(s.albumsFetchedAt, AlbumsTTL)
}

// LikedTracksStale returns true if the liked tracks are stale and should be re-fetched.
func (s *Store) LikedTracksStale() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return IsStale(s.likedTracksFetchedAt, LikedTracksTTL)
}

// RecentlyPlayedStale returns true if the recently played list is stale and should be re-fetched.
func (s *Store) RecentlyPlayedStale() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return IsStale(s.recentPlayedFetchedAt, RecentlyPlayedTTL)
}

// StatsStale returns true if stats for the given time range are stale and should be re-fetched.
func (s *Store) StatsStale(timeRange string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return IsStale(s.statsFetchedAt[timeRange], StatsTTL)
}

// DevicesStale returns true if the device list is stale and should be re-fetched.
func (s *Store) DevicesStale() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return IsStale(s.devicesFetchedAt, DevicesTTL)
}

// --- Fetching sentinel accessors ---

// PlaylistsFetching returns true while a playlists fetch is in-flight.
func (s *Store) PlaylistsFetching() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.playlistsFetching
}

// SetPlaylistsFetching sets or clears the in-flight playlists fetch sentinel.
func (s *Store) SetPlaylistsFetching(f bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.playlistsFetching = f
}

// AlbumsFetching returns true while a saved-albums fetch is in-flight.
func (s *Store) AlbumsFetching() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.albumsFetching
}

// SetAlbumsFetching sets or clears the in-flight albums fetch sentinel.
func (s *Store) SetAlbumsFetching(f bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.albumsFetching = f
}

// LikedFetching returns true while a liked-tracks fetch is in-flight.
func (s *Store) LikedFetching() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.likedFetching
}

// SetLikedFetching sets or clears the in-flight liked-tracks fetch sentinel.
func (s *Store) SetLikedFetching(f bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.likedFetching = f
}

// RecentFetching returns true while a recently-played fetch is in-flight.
func (s *Store) RecentFetching() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.recentFetching
}

// SetRecentFetching sets or clears the in-flight recently-played fetch sentinel.
func (s *Store) SetRecentFetching(f bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recentFetching = f
}

// StatsFetching returns true while a stats fetch for the given time range is in-flight.
func (s *Store) StatsFetching(timeRange string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.statsFetching[timeRange]
}

// SetStatsFetching sets or clears the in-flight stats fetch sentinel for a time range.
func (s *Store) SetStatsFetching(timeRange string, f bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statsFetching[timeRange] = f
}

// DevicesFetching returns true while a devices fetch is in-flight.
func (s *Store) DevicesFetching() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.devicesFetching
}

// SetDevicesFetching sets or clears the in-flight devices fetch sentinel.
func (s *Store) SetDevicesFetching(f bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.devicesFetching = f
}

// --- Error state accessors ---

// StatsError returns the last stats fetch error, or nil if successful.
func (s *Store) StatsError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.statsError
}

// SetStatsError records a stats fetch failure.
func (s *Store) SetStatsError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statsError = err
}

// ClearStatsError clears the stats error state on successful retry.
func (s *Store) ClearStatsError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statsError = nil
}

// DevicesError returns the last devices fetch error, or nil if successful.
func (s *Store) DevicesError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.devicesError
}

// SetDevicesError records a devices fetch failure.
func (s *Store) SetDevicesError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.devicesError = err
}

// ClearDevicesError clears the devices error state on successful retry.
func (s *Store) ClearDevicesError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.devicesError = nil
}

// QueueError returns the last queue fetch error, or nil if successful.
func (s *Store) QueueError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.queueError
}

// SetQueueError records a queue fetch failure.
func (s *Store) SetQueueError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queueError = err
}

// ClearQueueError clears the queue error state on successful retry.
func (s *Store) ClearQueueError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queueError = nil
}

// PlaylistsFetchError returns the last playlists list fetch error, or nil.
func (s *Store) PlaylistsFetchError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.playlistsFetchErr
}

// SetPlaylistsFetchError records a playlists list fetch failure.
func (s *Store) SetPlaylistsFetchError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.playlistsFetchErr = err
}

// ClearPlaylistsFetchError clears the playlists list fetch error on successful retry.
func (s *Store) ClearPlaylistsFetchError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.playlistsFetchErr = nil
}

// AlbumsFetchError returns the last saved albums fetch error, or nil.
func (s *Store) AlbumsFetchError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.albumsFetchErr
}

// SetAlbumsFetchError records a saved albums fetch failure.
func (s *Store) SetAlbumsFetchError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.albumsFetchErr = err
}

// ClearAlbumsFetchError clears the saved albums fetch error on successful retry.
func (s *Store) ClearAlbumsFetchError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.albumsFetchErr = nil
}

// LikedTracksFetchError returns the last liked tracks fetch error, or nil.
func (s *Store) LikedTracksFetchError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.likedTracksFetchErr
}

// SetLikedTracksFetchError records a liked tracks fetch failure.
func (s *Store) SetLikedTracksFetchError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.likedTracksFetchErr = err
}

// ClearLikedTracksFetchError clears the liked tracks fetch error on successful retry.
func (s *Store) ClearLikedTracksFetchError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.likedTracksFetchErr = nil
}

// RecentPlayedFetchError returns the last recently played fetch error, or nil.
func (s *Store) RecentPlayedFetchError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.recentPlayedFetchErr
}

// SetRecentPlayedFetchError records a recently played fetch failure.
func (s *Store) SetRecentPlayedFetchError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recentPlayedFetchErr = err
}

// ClearRecentPlayedFetchError clears the recently played fetch error on successful retry.
func (s *Store) ClearRecentPlayedFetchError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recentPlayedFetchErr = nil
}

// PlaylistsError returns the last playlists fetch error, or nil if successful.
func (s *Store) PlaylistsError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.playlistsError
}

// SetPlaylistsError records a playlists fetch failure.
func (s *Store) SetPlaylistsError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.playlistsError = err
}

// ClearPlaylistsError clears the playlists error state on successful retry.
func (s *Store) ClearPlaylistsError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.playlistsError = nil
}

// --- Podcast data accessors ---

// FollowedShows returns the user's followed podcast shows.
func (s *Store) FollowedShows() []domain.SavedShow {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.followedShows
}

// SetFollowedShows updates the followed shows in the store.
// fetchedAt is only stamped when the slice is non-empty to avoid resetting
// the TTL on empty/error responses and blocking retries prematurely.
func (s *Store) SetFollowedShows(items []domain.SavedShow) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.followedShows = items
	if len(items) > 0 {
		s.followedShowsFetchedAt = time.Now()
	}
}

// FollowedShowsLoaded returns true if followed shows have been fetched at least once.
func (s *Store) FollowedShowsLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return !s.followedShowsFetchedAt.IsZero()
}

// FollowedShowsFetchedAt returns the time when followed shows were last successfully fetched.
func (s *Store) FollowedShowsFetchedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.followedShowsFetchedAt
}

// SetFollowedShowsFetchedAt stamps the time when followed shows were last successfully loaded.
func (s *Store) SetFollowedShowsFetchedAt(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.followedShowsFetchedAt = t
}

// FollowedShowsStale returns true if the followed shows are stale and should be re-fetched.
func (s *Store) FollowedShowsStale() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return IsStale(s.followedShowsFetchedAt, FollowedShowsTTL)
}

// FollowedShowsFetching returns true while a followed-shows fetch is in-flight.
func (s *Store) FollowedShowsFetching() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.followedShowsFetching
}

// SetFollowedShowsFetching sets or clears the in-flight followed-shows fetch sentinel.
func (s *Store) SetFollowedShowsFetching(f bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.followedShowsFetching = f
}

// FollowedShowsFetchError returns the last followed shows fetch error, or nil.
func (s *Store) FollowedShowsFetchError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.followedShowsFetchErr
}

// SetFollowedShowsFetchError records a followed shows fetch failure.
func (s *Store) SetFollowedShowsFetchError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.followedShowsFetchErr = err
}

// ClearFollowedShowsFetchError clears the followed shows fetch error on successful retry.
func (s *Store) ClearFollowedShowsFetchError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.followedShowsFetchErr = nil
}

// SavedEpisodes returns the user's saved podcast episodes.
func (s *Store) SavedEpisodes() []domain.SavedEpisode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.savedEpisodes
}

// SetSavedEpisodes updates the saved episodes in the store.
// fetchedAt is only stamped when the slice is non-empty to avoid resetting
// the TTL on empty/error responses and blocking retries prematurely.
func (s *Store) SetSavedEpisodes(items []domain.SavedEpisode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.savedEpisodes = items
	if len(items) > 0 {
		s.savedEpisodesFetchedAt = time.Now()
	}
}

// SavedEpisodesLoaded returns true if saved episodes have been fetched at least once.
func (s *Store) SavedEpisodesLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return !s.savedEpisodesFetchedAt.IsZero()
}

// SavedEpisodesFetchedAt returns the time when saved episodes were last successfully fetched.
func (s *Store) SavedEpisodesFetchedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.savedEpisodesFetchedAt
}

// SetSavedEpisodesFetchedAt stamps the time when saved episodes were last successfully loaded.
func (s *Store) SetSavedEpisodesFetchedAt(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.savedEpisodesFetchedAt = t
}

// SavedEpisodesStale returns true if the saved episodes are stale and should be re-fetched.
func (s *Store) SavedEpisodesStale() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return IsStale(s.savedEpisodesFetchedAt, SavedEpisodesTTL)
}

// SavedEpisodesFetching returns true while a saved-episodes fetch is in-flight.
func (s *Store) SavedEpisodesFetching() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.savedEpisodesFetching
}

// SetSavedEpisodesFetching sets or clears the in-flight saved-episodes fetch sentinel.
func (s *Store) SetSavedEpisodesFetching(f bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.savedEpisodesFetching = f
}

// SavedEpisodesFetchError returns the last saved episodes fetch error, or nil.
func (s *Store) SavedEpisodesFetchError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.savedEpisodesFetchErr
}

// SetSavedEpisodesFetchError records a saved episodes fetch failure.
func (s *Store) SetSavedEpisodesFetchError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.savedEpisodesFetchErr = err
}

// ClearSavedEpisodesFetchError clears the saved episodes fetch error on successful retry.
func (s *Store) ClearSavedEpisodesFetchError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.savedEpisodesFetchErr = nil
}

// ShowEpisodes returns the cached episodes for a specific show.
func (s *Store) ShowEpisodes() []domain.Episode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.showEpisodes
}

// SetShowEpisodes updates the show episodes in the store.
// fetchedAt is only stamped when the slice is non-empty to avoid resetting
// the TTL on empty/error responses and blocking retries prematurely.
func (s *Store) SetShowEpisodes(items []domain.Episode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.showEpisodes = items
	if len(items) > 0 {
		s.showEpisodesFetchedAt = time.Now()
	}
}

// ShowEpisodesTotal returns the total number of episodes for the selected show.
func (s *Store) ShowEpisodesTotal() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.showEpisodesTotal
}

// SetShowEpisodesTotal updates the total episode count for the selected show.
func (s *Store) SetShowEpisodesTotal(total int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.showEpisodesTotal = total
}

// ShowEpisodesLoaded returns true if show episodes have been fetched at least once.
func (s *Store) ShowEpisodesLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return !s.showEpisodesFetchedAt.IsZero()
}

// ShowEpisodesFetchedAt returns the time when show episodes were last successfully fetched.
func (s *Store) ShowEpisodesFetchedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.showEpisodesFetchedAt
}

// SetShowEpisodesFetchedAt stamps the time when show episodes were last successfully loaded.
func (s *Store) SetShowEpisodesFetchedAt(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.showEpisodesFetchedAt = t
}

// ShowEpisodesStale returns true if the show episodes are stale and should be re-fetched.
func (s *Store) ShowEpisodesStale() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return IsStale(s.showEpisodesFetchedAt, ShowEpisodesTTL)
}

// ShowEpisodesFetching returns true while a show-episodes fetch is in-flight.
func (s *Store) ShowEpisodesFetching() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.showEpisodesFetching
}

// SetShowEpisodesFetching sets or clears the in-flight show-episodes fetch sentinel.
func (s *Store) SetShowEpisodesFetching(f bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.showEpisodesFetching = f
}

// ShowEpisodesFetchError returns the last show episodes fetch error, or nil.
func (s *Store) ShowEpisodesFetchError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.showEpisodesFetchErr
}

// SetShowEpisodesFetchError records a show episodes fetch failure.
func (s *Store) SetShowEpisodesFetchError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.showEpisodesFetchErr = err
}

// ClearShowEpisodesFetchError clears the show episodes fetch error on successful retry.
func (s *Store) ClearShowEpisodesFetchError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.showEpisodesFetchErr = nil
}

// SelectedShowID returns the Spotify ID of the currently selected show.
func (s *Store) SelectedShowID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.selectedShowID
}

// SetSelectedShowID sets the Spotify ID of the currently selected show.
func (s *Store) SetSelectedShowID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.selectedShowID = id
}

// SelectedShow returns the full show data for the currently selected show, or nil.
func (s *Store) SelectedShow() *domain.Show {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.selectedShow
}

// SetSelectedShow sets the full show data for the currently selected show.
func (s *Store) SetSelectedShow(show *domain.Show) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.selectedShow = show
}

// --- Gateway Event Journal ---

// Compile-time check: *Store must implement domain.GatewayEventRecorder.
var _ domain.GatewayEventRecorder = &Store{}

// RecordEvent records a gateway lifecycle event. Implements domain.GatewayEventRecorder.
func (s *Store) RecordEvent(event domain.GatewayEvent) {
	s.eventLog.Add(event)
}

// ReadEventsFrom returns gateway events added since the given cursor.
// Returns the new cursor and the slice of new events.
// Pass cursor=0 on the first call.
func (s *Store) ReadEventsFrom(cursor uint64) (uint64, []domain.GatewayEvent) {
	return s.eventLog.ReadFrom(cursor)
}

// --- Throttle observability ---

// SetThrottle records the current rate-limit state from the API Gateway.
// Called after a 429 response; cleared when the backoff period expires.
func (s *Store) SetThrottle(isThrottled bool, retryAfterSecs int, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.throttle.isThrottled = isThrottled
	s.throttle.retryAfterSecs = retryAfterSecs
	s.throttle.last429At = at
}

// IsThrottled returns true if the gateway is currently in a 429 backoff period.
func (s *Store) IsThrottled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.throttle.isThrottled
}

// ThrottleRetryAfterSecs returns the Retry-After seconds from the last 429 response.
func (s *Store) ThrottleRetryAfterSecs() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.throttle.retryAfterSecs
}

// ThrottleLast429At returns the time of the most recent 429 response.
func (s *Store) ThrottleLast429At() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.throttle.last429At
}
