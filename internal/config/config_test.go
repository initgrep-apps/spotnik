package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefault_ReturnsNonNil(t *testing.T) {
	cfg := config.Default()
	require.NotNil(t, cfg)
}

func TestDefault_ThemeIsBlack(t *testing.T) {
	cfg := config.Default()
	assert.Equal(t, "black", cfg.UI.Theme)
}

// TestLoad_MissingFile_ReturnsDefaults verifies that when no config file exists,
// Load returns the default config without error.
func TestLoad_MissingFile_ReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Load(filepath.Join(dir, "nonexistent.toml"))
	require.NoError(t, err)
	assert.Equal(t, "black", cfg.UI.Theme)
}

// TestLoad_ValidFile verifies that a valid TOML file with all fields parses correctly.
func TestLoad_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[spotify]
client_id = "my-client-id"

[ui]
theme = "nord"
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, "my-client-id", cfg.ClientID)
	assert.Equal(t, "nord", cfg.UI.Theme)
}

// TestLoad_MissingClientID_ReturnsError verifies that a config file without
// client_id returns a descriptive error.
func TestLoad_MissingClientID_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[spotify]

[ui]
theme = "black"
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	_, err := config.Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client_id")
}

// TestLoad_InvalidTOML_ReturnsError verifies that a malformed TOML file
// returns a parse error containing the file path for context.
func TestLoad_InvalidTOML_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte("not valid toml ][[["), 0o600))
	_, err := config.Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), path)
}

// TestLoad_DefaultTheme verifies that when theme is not set, it defaults to "black".
func TestLoad_DefaultTheme(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[spotify]
client_id = "my-client-id"
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, "black", cfg.UI.Theme)
}

// TestLoad_PartialConfig_MergesWithDefaults verifies that only specified fields
// override defaults, leaving unspecified fields at their default values.
func TestLoad_PartialConfig_MergesWithDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[spotify]
client_id = "another-id"
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	cfg, err := config.Load(path)
	require.NoError(t, err)
	// ClientID specified → must be set
	assert.Equal(t, "another-id", cfg.ClientID)
	// Theme not specified → must be default
	assert.Equal(t, "black", cfg.UI.Theme)
}

// TestDefaultConfigPath_ContainsSpotnik verifies that the default config path
// includes "spotnik" and ends with "config.toml".
func TestDefaultConfigPath_ContainsSpotnik(t *testing.T) {
	path := config.DefaultConfigPath()
	assert.Contains(t, path, "spotnik")
	assert.True(t, len(path) > 0, "path should not be empty")
	// Path should end with config.toml.
	assert.True(t,
		path == "config.toml" || // fallback
			filepath.Base(path) == "config.toml",
		"path should end with config.toml, got: %s", path)
}

func TestDefault_VolumeStep(t *testing.T) {
	cfg := config.Default()
	assert.Equal(t, 5, cfg.UI.VolumeStep, "default volume step should be 5")
}

// TestPersistTheme_WritesFile verifies that PersistThemeTo creates a config file
// with the given theme ID at the specified path.
func TestPersistTheme_WritesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	err := config.PersistThemeTo(path, "dracula")
	require.NoError(t, err)

	// Read back and verify the theme was written.
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "dracula", "config file should contain the theme ID")
}

// TestPersistTheme_CreatesFileIfMissing verifies that PersistThemeTo creates the file
// and any required parent directories when they don't exist yet.
func TestPersistTheme_CreatesFileIfMissing(t *testing.T) {
	dir := t.TempDir()
	// Use a nested path that doesn't exist yet.
	path := filepath.Join(dir, "spotnik", "config.toml")

	err := config.PersistThemeTo(path, "monokai")
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "monokai")
}

// TestPersistTheme_PreservesOtherConfig verifies that PersistThemeTo only changes
// the theme field and preserves other settings (client_id, volume_step, etc.).
func TestPersistTheme_PreservesOtherConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	// Write an initial config with extra fields.
	initial := `[spotify]
client_id = "my-client-id"

[ui]
theme = "black"
volume_step = 10
`
	require.NoError(t, os.WriteFile(path, []byte(initial), 0o600))

	// Persist a new theme.
	err := config.PersistThemeTo(path, "nord")
	require.NoError(t, err)

	// Read back and check that only the theme changed.
	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, "nord", cfg.UI.Theme, "theme should be updated")
	assert.Equal(t, "my-client-id", cfg.ClientID, "client_id should be preserved")
	assert.Equal(t, 10, cfg.UI.VolumeStep, "volume_step should be preserved")
}
