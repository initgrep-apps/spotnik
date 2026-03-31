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
	assert.Equal(t, "black", cfg.Preferences.Theme)
}

// TestLoad_MissingFile_ReturnsDefaults verifies that when no config file exists,
// Load returns the default config without error.
func TestLoad_MissingFile_ReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Load(filepath.Join(dir, "nonexistent.toml"))
	require.NoError(t, err)
	assert.Equal(t, "black", cfg.Preferences.Theme)
}

// TestLoad_ValidFile verifies that a valid TOML file with all fields parses correctly.
func TestLoad_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[spotify]
client_id = "my-client-id"

[preferences]
theme = "nord"
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, "my-client-id", cfg.ClientID)
	assert.Equal(t, "nord", cfg.Preferences.Theme)
}

// TestLoad_EmptyClientID_NoError verifies that a config file without client_id
// returns a valid Config with empty ClientID — no error. The caller handles the
// embedded fallback.
func TestLoad_EmptyClientID_NoError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[spotify]

[preferences]
theme = "black"
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, "", cfg.ClientID)
}

// TestLoad_WithClientID_StillWorks verifies that a config with client_id continues
// to work correctly after the validation change.
func TestLoad_WithClientID_StillWorks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[spotify]
client_id = "abc123"

[preferences]
theme = "dracula"
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, "abc123", cfg.ClientID)
	assert.Equal(t, "dracula", cfg.Preferences.Theme)
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
	assert.Equal(t, "black", cfg.Preferences.Theme)
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
	// ClientID specified → must be set.
	assert.Equal(t, "another-id", cfg.ClientID)
	// Theme not specified → must be default.
	assert.Equal(t, "black", cfg.Preferences.Theme)
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
	assert.Equal(t, 5, cfg.Preferences.VolumeStep, "default volume step should be 5")
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

[preferences]
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
	assert.Equal(t, "nord", cfg.Preferences.Theme, "theme should be updated")
	assert.Equal(t, "my-client-id", cfg.ClientID, "client_id should be preserved")
	assert.Equal(t, 10, cfg.Preferences.VolumeStep, "volume_step should be preserved")
}

// TestPersistTheme_OutputUsesPreferencesSection verifies that PersistThemeTo writes
// the [preferences] section (not the old [ui] section).
func TestPersistTheme_OutputUsesPreferencesSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	err := config.PersistThemeTo(path, "black")
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "[preferences]", "output should use [preferences] not [ui]")
}

// TestLoad_NegativePreset_ClampsToZero verifies that a negative preset value is
// clamped to 0 on load.
func TestLoad_NegativePreset_ClampsToZero(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[preferences]
preset = -3
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, 0, cfg.Preferences.Preset, "negative preset should be clamped to 0")
}

// TestLoad_NegativeVisualizer_ClampsToZero verifies that a negative visualizer value
// is clamped to 0 on load.
func TestLoad_NegativeVisualizer_ClampsToZero(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[preferences]
visualizer = -1
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, 0, cfg.Preferences.Visualizer, "negative visualizer should be clamped to 0")
}

// TestLoad_ValidPreferences_Preserved verifies that valid non-default preference
// values are preserved as-is.
func TestLoad_ValidPreferences_Preserved(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[preferences]
theme = "monokai"
volume_step = 10
preset = 2
visualizer = 3
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, "monokai", cfg.Preferences.Theme)
	assert.Equal(t, 10, cfg.Preferences.VolumeStep)
	assert.Equal(t, 2, cfg.Preferences.Preset)
	assert.Equal(t, 3, cfg.Preferences.Visualizer)
}

// TestBootstrap_CreatesFileWhenMissing verifies that Bootstrap creates a config
// file when none exists.
func TestBootstrap_CreatesFileWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	err := config.Bootstrap(path)
	require.NoError(t, err)

	_, err = os.Stat(path)
	require.NoError(t, err, "config file should exist after Bootstrap")
}

// TestBootstrap_NoopWhenExists verifies that Bootstrap does not modify an existing
// config file.
func TestBootstrap_NoopWhenExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	// Write a known initial file.
	original := "# my config\n"
	require.NoError(t, os.WriteFile(path, []byte(original), 0o600))

	err := config.Bootstrap(path)
	require.NoError(t, err)

	// File should be unchanged.
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, original, string(data), "Bootstrap should not modify an existing file")
}

// TestBootstrap_CreatesDirectory verifies that Bootstrap creates parent directories
// when they do not exist.
func TestBootstrap_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	// Use a nested path that doesn't exist.
	path := filepath.Join(dir, "nested", "spotnik", "config.toml")

	err := config.Bootstrap(path)
	require.NoError(t, err)

	_, err = os.Stat(path)
	require.NoError(t, err, "config file should exist in created directory")
}

// TestBootstrap_TemplateContent verifies that the bootstrapped file contains
// the expected default content.
func TestBootstrap_TemplateContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	err := config.Bootstrap(path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)

	assert.Contains(t, content, "[spotify]", "template should have [spotify] section")
	assert.Contains(t, content, "[preferences]", "template should have [preferences] section")
	assert.Contains(t, content, "theme = \"black\"", "template should set default theme")
	assert.Contains(t, content, "volume_step = 5", "template should set default volume_step")
	assert.Contains(t, content, "client_id", "template should mention client_id as a comment")
}

// TestBootstrap_FilePermissions verifies that the created config file has
// owner-only read/write permissions (0600).
func TestBootstrap_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	err := config.Bootstrap(path)
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	// Mask to permission bits only (lower 9 bits).
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "config file should have 0600 permissions")
}

// TestPersistTheme_WithBootstrappedConfig verifies that PersistTheme works correctly
// after Bootstrap has created the config file.
func TestPersistTheme_WithBootstrappedConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	// Bootstrap creates the template file.
	require.NoError(t, config.Bootstrap(path))

	// Persist a theme to the bootstrapped file.
	err := config.PersistThemeTo(path, "monokai")
	require.NoError(t, err)

	// Load and verify the round-trip.
	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, "monokai", cfg.Preferences.Theme, "theme should persist through bootstrap round-trip")
}
