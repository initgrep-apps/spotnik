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
	// Short because the user may switch devices at any time.
	DevicesTTL = 30 * time.Second
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

	// Library data
	playlists      []domain.SimplePlaylist
	playlistsTotal int
	savedAlbums    []domain.SavedAlbum
	likedTracks    []domain.SavedTrack
	likedTotal     int
	recentlyPlayed []domain.PlayHistory

	// Staleness tracking — set to time.Now() on successful data write.
	playlistsFetchedAt    time.Time
	albumsFetchedAt       time.Time
	likedTracksFetchedAt  time.Time
	recentPlayedFetchedAt time.Time
	statsFetchedAt        map[string]time.Time // keyed by time range
	devicesFetchedAt      time.Time

	// Queue data
	queue []domain.Track

	// Search data — searchResults holds the raw search response for the SearchClearedMsg handler.
	searchResults *domain.SearchResult
	searchQuery   string
	searchLoading bool

	// Stats data: top tracks and top artists keyed by time range.
	// Ranges: "short_term", "medium_term", "long_term".
	// NOTE: cached per range and re-fetched after StatsTTL (10 min) via staleness check in Update().
	topTracks  map[string][]domain.Track
	topArtists map[string][]domain.FullArtist

	// Playlist Manager data: tracks for each playlist keyed by playlist ID.
	playlistTracks map[string][]domain.Track

	// playingPlaylistID is the Spotify playlist ID that is currently playing.
	// Used by PlaylistManager to render the ▶ indicator next to the active playlist.
	playingPlaylistID string

	// netLog records all API calls for the network log panel.
	netLog *NetLog

	// throttle holds rate-limit observability state updated by the API Gateway.
	// The UI status bar reads these to show a "Rate limited" indicator.
	throttle struct {
		isThrottled    bool
		retryAfterSecs int
		last429At      time.Time
	}

	// Error state — one per data-fetching feature.
	// Set by Update() handlers on failure, cleared on successful retry.
	searchError          error
	statsError           error
	devicesError         error
	queueError           error
	playlistsFetchErr    error // playlists list fetch
	albumsFetchErr       error // saved albums fetch
	likedTracksFetchErr  error // liked tracks fetch
	recentPlayedFetchErr error // recently played fetch
	playlistsError       error // playlist manager (tracks, mutations)

}

// New returns an empty Store with no playback state.
// statsFetchedAt is pre-allocated so callers never encounter a nil map panic
// when reading stats staleness before any fetch has completed.
func New() *Store {
	return &Store{
		netLog:         NewNetLog(),
		statsFetchedAt: make(map[string]time.Time),
	}
}

// PlaybackState returns the current playback state, or nil if nothing is playing.
func (s *Store) PlaybackState() *domain.PlaybackState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.playbackState
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

// Playlists returns the user's saved playlists.
func (s *Store) Playlists() []domain.SimplePlaylist {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.playlists
}

// SetPlaylists updates the saved playlists in the store and stamps the fetch time.
func (s *Store) SetPlaylists(playlists []domain.SimplePlaylist) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.playlists = playlists
	s.playlistsFetchedAt = time.Now()
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

// SetSavedAlbums updates the saved albums in the store and stamps the fetch time.
func (s *Store) SetSavedAlbums(albums []domain.SavedAlbum) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.savedAlbums = albums
	s.albumsFetchedAt = time.Now()
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

// SetLikedTracks updates the liked tracks in the store and stamps the fetch time.
func (s *Store) SetLikedTracks(tracks []domain.SavedTrack) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.likedTracks = tracks
	s.likedTracksFetchedAt = time.Now()
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

// SetRecentlyPlayed updates the recently played history in the store and stamps the fetch time.
func (s *Store) SetRecentlyPlayed(items []domain.PlayHistory) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recentlyPlayed = items
	s.recentPlayedFetchedAt = time.Now()
}

// Queue returns the upcoming tracks in the user's play queue.
func (s *Store) Queue() []domain.Track {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.queue
}

// SetQueue updates the queue tracks in the store.
func (s *Store) SetQueue(tracks []domain.Track) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queue = tracks
}

// SearchResults returns the most recent search result, or nil if no search has completed.
func (s *Store) SearchResults() *domain.SearchResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.searchResults
}

// SetSearchResults updates the search results in the store.
func (s *Store) SetSearchResults(results *domain.SearchResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.searchResults = results
}

// SearchQuery returns the most recent search query string.
func (s *Store) SearchQuery() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.searchQuery
}

// SetSearchQuery updates the current search query string in the store.
func (s *Store) SetSearchQuery(query string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.searchQuery = query
}

// SearchLoading returns true while a search API call is in flight.
func (s *Store) SearchLoading() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.searchLoading
}

// SetSearchLoading sets the search loading flag.
func (s *Store) SetSearchLoading(loading bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.searchLoading = loading
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

// SetTopTracks caches top tracks for a specific time range in the store
// and stamps the fetch time for that range.
func (s *Store) SetTopTracks(timeRange string, tracks []domain.Track) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.topTracks == nil {
		s.topTracks = make(map[string][]domain.Track)
	}
	s.topTracks[timeRange] = tracks
	s.statsFetchedAt[timeRange] = time.Now()
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

// SetTopArtists caches top artists for a specific time range in the store
// and stamps the fetch time for that range.
func (s *Store) SetTopArtists(timeRange string, artists []domain.FullArtist) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.topArtists == nil {
		s.topArtists = make(map[string][]domain.FullArtist)
	}
	s.topArtists[timeRange] = artists
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

// --- Error state accessors ---

// SearchError returns the last search error, or nil if the last search succeeded.
func (s *Store) SearchError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.searchError
}

// SetSearchError records a search failure.
func (s *Store) SetSearchError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.searchError = err
}

// ClearSearchError clears the search error state on successful retry.
func (s *Store) ClearSearchError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.searchError = nil
}

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

// --- Network log accessors ---

// RecordNetCall adds an API call record to the network log.
// Implements api.NetLogRecorder.
func (s *Store) RecordNetCall(method, path string, statusCode int, durationMs int64) {
	s.netLog.Add(NetLogEntry{
		Timestamp:  time.Now(),
		Method:     method,
		Path:       path,
		StatusCode: statusCode,
		DurationMs: durationMs,
	})
}

// NetLogEntries returns all network log entries in oldest-first order.
func (s *Store) NetLogEntries() []NetLogEntry {
	return s.netLog.Entries()
}

// NetLog returns the underlying NetLog ring buffer.
func (s *Store) NetLog() *NetLog {
	return s.netLog
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
