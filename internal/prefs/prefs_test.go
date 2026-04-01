package prefs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/prefs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew_CreatesStore verifies that New returns a non-nil PreferenceStore
// targeting the given path.
func TestNew_CreatesStore(t *testing.T) {
	s := prefs.New("/tmp/test-config.toml")
	require.NotNil(t, s)
}

// TestSet_StoresInMemory verifies that Set marks a preference as pending
// without touching the disk.
func TestSet_StoresInMemory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	s := prefs.New(path)
	s.Set("theme", "monokai")

	// File should NOT exist — Set is in-memory only.
	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "Set should not write to disk")

	// HasPending should be true after a Set call.
	assert.True(t, s.HasPending(), "HasPending should return true after Set")
}

// TestSet_MultipleKeys verifies that Set handles multiple distinct keys.
func TestSet_MultipleKeys(t *testing.T) {
	s := prefs.New("/tmp/test-config.toml")
	s.Set("theme", "nord")
	s.Set("preset", 2)
	s.Set("visualizer", 3)

	assert.True(t, s.HasPending(), "HasPending should be true after multiple Set calls")
}

// TestHasPending_TrueAfterSet verifies HasPending returns true when there is
// at least one pending (unsaved) change.
func TestHasPending_TrueAfterSet(t *testing.T) {
	s := prefs.New("/tmp/test-config.toml")
	assert.False(t, s.HasPending(), "fresh store should have no pending changes")

	s.Set("theme", "dracula")
	assert.True(t, s.HasPending(), "HasPending should be true after Set")
}

// TestFlushCmd_WritesToDisk verifies that executing FlushCmd writes the pending
// preferences to the config file.
func TestFlushCmd_WritesToDisk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	s := prefs.New(path)
	s.Set("theme", "monokai")

	msg := s.FlushCmd()()
	flushed, ok := msg.(prefs.FlushedMsg)
	require.True(t, ok, "FlushCmd should return a FlushedMsg")
	require.NoError(t, flushed.Err, "flush should succeed")

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "monokai", "flushed file should contain the theme")
}

// TestFlushCmd_PreservesSpotifySection verifies that FlushCmd preserves the
// [spotify] section when writing preferences.
func TestFlushCmd_PreservesSpotifySection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	// Write an initial config with a spotify section.
	initial := `[spotify]
client_id = "my-client-id"

[preferences]
theme = "black"
volume_step = 5
`
	require.NoError(t, os.WriteFile(path, []byte(initial), 0o600))

	s := prefs.New(path)
	s.Set("theme", "nord")

	msg := s.FlushCmd()()
	flushed, ok := msg.(prefs.FlushedMsg)
	require.True(t, ok)
	require.NoError(t, flushed.Err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "my-client-id", "spotify section should be preserved")
	assert.Contains(t, content, "nord", "theme should be updated")
}

// TestFlushCmd_ClearsPendingOnSuccess verifies that FlushCmd clears the pending
// map after a successful write.
func TestFlushCmd_ClearsPendingOnSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	s := prefs.New(path)
	s.Set("theme", "monokai")

	require.True(t, s.HasPending())

	msg := s.FlushCmd()()
	flushed, ok := msg.(prefs.FlushedMsg)
	require.True(t, ok)
	require.NoError(t, flushed.Err)

	assert.False(t, s.HasPending(), "pending map should be cleared after successful flush")
}

// TestFlushCmd_RequeuesOnFailure verifies that when FlushCmd cannot write to disk,
// the pending changes are re-queued so the next flush can retry.
func TestFlushCmd_RequeuesOnFailure(t *testing.T) {
	// Use a path where writes will fail: a directory as the config file path.
	dir := t.TempDir()
	// Point the path to the directory itself — os.OpenFile will fail.
	path := dir

	s := prefs.New(path)
	s.Set("theme", "nord")

	msg := s.FlushCmd()()
	flushed, ok := msg.(prefs.FlushedMsg)
	require.True(t, ok)
	require.Error(t, flushed.Err, "write to directory should fail")

	// Changes should be re-queued so next flush can retry.
	assert.True(t, s.HasPending(), "pending map should be restored after failed flush")
}

// TestFlushCmd_NoopWhenEmpty verifies that FlushCmd is a no-op when there are
// no pending changes — it returns FlushedMsg with nil Err without writing.
func TestFlushCmd_NoopWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	s := prefs.New(path)
	// No Set calls — nothing pending.

	msg := s.FlushCmd()()
	flushed, ok := msg.(prefs.FlushedMsg)
	require.True(t, ok)
	require.NoError(t, flushed.Err)

	// File should not have been created.
	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "FlushCmd should not create file when nothing pending")
}

// TestFlushCmd_CoalescesMultipleChanges verifies that setting multiple preferences
// then calling FlushCmd once writes all of them in a single file write.
func TestFlushCmd_CoalescesMultipleChanges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	s := prefs.New(path)
	s.Set("theme", "dracula")
	s.Set("preset", 2)
	s.Set("visualizer", 5)

	msg := s.FlushCmd()()
	flushed, ok := msg.(prefs.FlushedMsg)
	require.True(t, ok)
	require.NoError(t, flushed.Err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)

	assert.Contains(t, content, "dracula", "theme should be written")
	assert.Contains(t, content, "2", "preset should be written")
	assert.Contains(t, content, "5", "visualizer should be written")
	assert.False(t, s.HasPending(), "no pending changes after successful flush")
}

// TestFlushCmd_UnknownKeyReturnsError verifies that an unknown preference key
// causes FlushCmd to return a FlushedMsg with a non-nil error (changes re-queued).
func TestFlushCmd_UnknownKeyReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	s := prefs.New(path)
	s.Set("typo_key", "oops")

	msg := s.FlushCmd()()
	flushed, ok := msg.(prefs.FlushedMsg)
	require.True(t, ok, "FlushCmd must return a FlushedMsg")
	require.Error(t, flushed.Err, "unknown key should produce an error")
	assert.Contains(t, flushed.Err.Error(), "typo_key", "error should name the bad key")

	// Changes must be re-queued so the caller can decide what to do.
	assert.True(t, s.HasPending(), "unknown-key error should re-queue pending changes")
}

// TestFlushCmd_WrongTypeReturnsError verifies that passing the wrong Go type for
// a known key causes FlushCmd to return an error rather than panicking.
func TestFlushCmd_WrongTypeReturnsError(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value any
	}{
		{name: "theme with int", key: "theme", value: 42},
		{name: "preset with string", key: "preset", value: "two"},
		{name: "visualizer with bool", key: "visualizer", value: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.toml")

			s := prefs.New(path)
			s.Set(tt.key, tt.value)

			msg := s.FlushCmd()()
			flushed, ok := msg.(prefs.FlushedMsg)
			require.True(t, ok, "FlushCmd must return a FlushedMsg")
			require.Error(t, flushed.Err, "wrong type for %q should produce an error", tt.key)
		})
	}
}
