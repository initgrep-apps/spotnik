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
