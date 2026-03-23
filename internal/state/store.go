// Package state provides the central Store — the single source of truth for all
// application data. Panes read from the store via accessor methods. Only Commands
// write to the store, never pane Update() or View() directly.
package state

import (
	"sync"

	"github.com/initgrep-apps/spotnik/internal/api"
)

// Store is the central application state. All panes read from here; only
// tea.Cmd callbacks write to it. Fields are never accessed directly — use the
// accessor methods to ensure safe concurrent access.
type Store struct {
	mu            sync.RWMutex
	playbackState *api.PlaybackState
	activeDevice  *api.Device

	// Library data
	playlists      []api.SimplePlaylist
	playlistsTotal int
	savedAlbums    []api.SavedAlbum
	albumsLoaded   bool
	likedTracks    []api.SavedTrack
	likedTotal     int
	likedLoaded    bool
	recentlyPlayed []api.PlayHistory

	// Queue data
	queue []api.Track

	// Search data
	searchResults *api.SearchResult
	searchQuery   string
	searchLoading bool
}

// New returns an empty Store with no playback state.
func New() *Store {
	return &Store{}
}

// PlaybackState returns the current playback state, or nil if nothing is playing.
func (s *Store) PlaybackState() *api.PlaybackState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.playbackState
}

// SetPlaybackState updates the playback state. Pass nil to clear (204 response).
// Also updates the active device from the state's Device field.
func (s *Store) SetPlaybackState(state *api.PlaybackState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.playbackState = state
	if state != nil {
		s.activeDevice = state.Device
	}
}

// ActiveDevice returns the currently active Spotify device, or nil if unknown.
func (s *Store) ActiveDevice() *api.Device {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeDevice
}

// SetActiveDevice updates the active device independently of playback state.
func (s *Store) SetActiveDevice(device *api.Device) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeDevice = device
}

// Playlists returns the user's saved playlists.
func (s *Store) Playlists() []api.SimplePlaylist {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.playlists
}

// SetPlaylists updates the saved playlists in the store.
func (s *Store) SetPlaylists(playlists []api.SimplePlaylist) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.playlists = playlists
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
func (s *Store) SavedAlbums() []api.SavedAlbum {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.savedAlbums
}

// SetSavedAlbums updates the saved albums in the store.
// Also marks albumsLoaded = true.
func (s *Store) SetSavedAlbums(albums []api.SavedAlbum) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.savedAlbums = albums
	s.albumsLoaded = true
}

// AlbumsLoaded returns true if saved albums have been fetched at least once.
func (s *Store) AlbumsLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.albumsLoaded
}

// LikedTracks returns the user's liked tracks.
func (s *Store) LikedTracks() []api.SavedTrack {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.likedTracks
}

// SetLikedTracks updates the liked tracks in the store.
// Also marks likedLoaded = true.
func (s *Store) SetLikedTracks(tracks []api.SavedTrack) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.likedTracks = tracks
	s.likedLoaded = true
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
func (s *Store) LikedLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.likedLoaded
}

// RecentlyPlayed returns the recently played track history.
func (s *Store) RecentlyPlayed() []api.PlayHistory {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.recentlyPlayed
}

// SetRecentlyPlayed updates the recently played history in the store.
func (s *Store) SetRecentlyPlayed(items []api.PlayHistory) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recentlyPlayed = items
}

// Queue returns the upcoming tracks in the user's play queue.
func (s *Store) Queue() []api.Track {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.queue
}

// SetQueue updates the queue tracks in the store.
func (s *Store) SetQueue(tracks []api.Track) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queue = tracks
}

// SearchResults returns the most recent search result, or nil if no search has completed.
func (s *Store) SearchResults() *api.SearchResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.searchResults
}

// SetSearchResults updates the search results in the store.
func (s *Store) SetSearchResults(results *api.SearchResult) {
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
