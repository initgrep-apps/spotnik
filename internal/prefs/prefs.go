// Package prefs provides a coalescing preference writer for Spotnik.
// It batches in-memory preference changes and flushes them to disk in a
// single debounced write via the Bubble Tea Cmd/Msg pattern.
package prefs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/BurntSushi/toml"
	"github.com/initgrep-apps/spotnik/internal/config"
)

// PreferenceStore manages in-memory preference state and coalesces writes to disk.
// It is the single point of truth for runtime preference changes. Thread-safe.
type PreferenceStore struct {
	mu      sync.Mutex
	path    string         // config file path
	pending map[string]any // dirty preferences not yet flushed
}

// New creates a PreferenceStore targeting the given config file path.
func New(path string) *PreferenceStore {
	return &PreferenceStore{
		path:    path,
		pending: make(map[string]any),
	}
}

// Set marks a preference as dirty. The value is held in memory until FlushCmd
// writes it to disk. Thread-safe — can be called from Update() safely.
func (s *PreferenceStore) Set(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending[key] = value
}

// HasPending returns true if there are unsaved preference changes.
func (s *PreferenceStore) HasPending() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.pending) > 0
}

// FlushedMsg is emitted after a flush attempt completes.
// Err is nil on success.
type FlushedMsg struct {
	Err error
}

// FlushCmd returns a tea.Cmd that writes all pending preferences to disk.
// It reads the existing config file, applies pending changes to the
// [preferences] section, and writes it back. Pending map is
// cleared after a successful write. If nothing is pending, returns a no-op
// FlushedMsg immediately.
func (s *PreferenceStore) FlushCmd() tea.Cmd {
	return func() tea.Msg {
		s.mu.Lock()
		if len(s.pending) == 0 {
			s.mu.Unlock()
			return FlushedMsg{}
		}
		// Snapshot pending and clear.
		snapshot := make(map[string]any, len(s.pending))
		for k, v := range s.pending {
			snapshot[k] = v
		}
		s.pending = make(map[string]any)
		s.mu.Unlock()

		err := s.writeToDisk(snapshot)
		if err != nil {
			// Re-queue failed changes so they retry on next flush.
			// Only re-queue keys that have not been superseded by a newer Set().
			s.mu.Lock()
			for k, v := range snapshot {
				if _, exists := s.pending[k]; !exists {
					s.pending[k] = v
				}
			}
			s.mu.Unlock()
		}
		return FlushedMsg{Err: err}
	}
}

// writeToDisk reads the existing config TOML, applies the snapshot of
// preference changes, and writes the result back. Creates the file and
// parent directory if needed.
func (s *PreferenceStore) writeToDisk(snapshot map[string]any) error {
	// Use the same raw TOML struct pattern as the existing config package
	// to preserve the [spotify] section and any unknown fields.
	raw := struct {
		Spotify struct {
			ClientID string `toml:"client_id,omitempty"`
		} `toml:"spotify"`
		Preferences config.PreferencesConfig `toml:"preferences"`
	}{
		Preferences: config.PreferencesConfig{
			Theme:      "black",
			VolumeStep: 5,
		},
	}

	// Read existing file (ignore missing).
	if _, err := toml.DecodeFile(s.path, &raw); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("reading config for preference update: %w", err)
	}

	// Apply snapshot fields to the preferences section.
	for key, val := range snapshot {
		switch key {
		case "theme":
			raw.Preferences.Theme = val.(string)
		case "preset":
			raw.Preferences.Preset = val.(int)
		case "visualizer":
			raw.Preferences.Visualizer = val.(int)
		}
	}

	// Ensure the config directory exists.
	if err := os.MkdirAll(filepath.Dir(s.path), 0o750); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	f, err := os.OpenFile(s.path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("opening config for write: %w", err)
	}

	enc := toml.NewEncoder(f)
	encErr := enc.Encode(raw)
	if closeErr := f.Close(); closeErr != nil && encErr == nil {
		return fmt.Errorf("closing config: %w", closeErr)
	}
	if encErr != nil {
		return fmt.Errorf("writing config: %w", encErr)
	}
	return nil
}
