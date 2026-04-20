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

// TestLoad_UnknownTheme_ClampsToBlack verifies that an unrecognised theme ID is
// clamped to the default "black" theme when a ThemeValidator is registered.
func TestLoad_UnknownTheme_ClampsToBlack(t *testing.T) {
	// Register a simple validator that only accepts known themes.
	original := config.ThemeValidator
	config.ThemeValidator = func(id string) bool {
		known := []string{"black", "nord", "monokai", "dracula"}
		for _, k := range known {
			if k == id {
				return true
			}
		}
		return false
	}
	defer func() { config.ThemeValidator = original }()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[preferences]
theme = "absolutely-not-a-theme"
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, "black", cfg.Preferences.Theme, "unknown theme should be clamped to 'black'")
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
preset = 2
visualizer = 3
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, "monokai", cfg.Preferences.Theme)
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
	assert.Contains(t, content, "client_id", "template should mention client_id as a comment")
}

// TestBootstrap_FilePermissions verifies that the created config file has
// owner-only read/write permissions (0600) and that the created directory
// has owner-writable, group-readable permissions (0750).
func TestBootstrap_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	// Use a nested subdirectory so we can check Bootstrap's directory creation.
	subDir := filepath.Join(dir, "spotnik")
	path := filepath.Join(subDir, "config.toml")

	err := config.Bootstrap(path)
	require.NoError(t, err)

	// Verify file permissions (0600: owner read/write only).
	fileInfo, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), fileInfo.Mode().Perm(),
		"config file should have 0600 permissions")

	// Verify directory permissions (0750: owner rwx, group rx, others none).
	dirInfo, err := os.Stat(subDir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o750), dirInfo.Mode().Perm(),
		"config directory should have 0750 permissions")
}

// ---- CallbackPort tests ----

// TestLoad_CallbackPort_defaultsTo8888 verifies that when callback_port is absent
// from the config file, the default of 8888 is used.
func TestLoad_CallbackPort_defaultsTo8888(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[spotify]
client_id = "my-client-id"
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, 8888, cfg.CallbackPort, "callback_port should default to 8888 when absent")
}

// TestLoad_CallbackPort_fromFile verifies that when callback_port is set in the
// config file, it overrides the default.
func TestLoad_CallbackPort_fromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[spotify]
client_id = "my-client-id"
callback_port = 9999
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, 9999, cfg.CallbackPort, "callback_port from file should be 9999")
}

// ---- ClearClientID tests ----

// TestClearClientID_removesClientIDPreservesPreferences verifies that ClearClientID
// removes only the client_id line while preserving other settings.
func TestClearClientID_removesClientIDPreservesPreferences(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `[spotify]
client_id = "abc123"
callback_port = 9000

[preferences]
theme = "nord"
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	err := config.ClearClientID(path)
	require.NoError(t, err)

	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, "", cfg.ClientID, "client_id should be removed")
	assert.Equal(t, 9000, cfg.CallbackPort, "callback_port should be preserved")
	assert.Equal(t, "nord", cfg.Preferences.Theme, "theme should be preserved")
}

// TestClearClientID_noClientID_isNoOp verifies that ClearClientID on a file with
// no client_id returns nil and leaves the file unchanged.
func TestClearClientID_noClientID_isNoOp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	original := `[spotify]
callback_port = 9000

[preferences]
theme = "black"
`
	require.NoError(t, os.WriteFile(path, []byte(original), 0o600))

	err := config.ClearClientID(path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, original, string(data), "file should be unchanged when no client_id present")
}

// TestClearClientID_missingFile_returnsError verifies that ClearClientID returns
// an error when the file does not exist.
func TestClearClientID_missingFile_returnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.toml")

	err := config.ClearClientID(path)
	require.Error(t, err, "ClearClientID should return an error for missing file")
}

// ---- SetClientID tests ----

// TestSetClientID_writesNewValue verifies that SetClientID inserts client_id when
// the [spotify] section exists but has no client_id key.
func TestSetClientID_writesNewValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `[spotify]
callback_port = 9000

[preferences]
theme = "nord"
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	err := config.SetClientID(path, "new-client-id")
	require.NoError(t, err)

	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, "new-client-id", cfg.ClientID, "client_id should be set to new value")
	assert.Equal(t, 9000, cfg.CallbackPort, "callback_port should be preserved")
	assert.Equal(t, "nord", cfg.Preferences.Theme, "theme should be preserved")
}

// TestSetClientID_replacesExistingValue verifies that SetClientID replaces an
// existing client_id value in-place.
func TestSetClientID_replacesExistingValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `[spotify]
client_id = "old-client-id"
callback_port = 9000
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	err := config.SetClientID(path, "new-client-id")
	require.NoError(t, err)

	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, "new-client-id", cfg.ClientID, "client_id should be replaced with new value")
	assert.Equal(t, 9000, cfg.CallbackPort, "callback_port should be preserved after replace")
}

// TestBootstrap_StatErrorPropagated verifies that Bootstrap returns an error when
// os.Stat fails for a reason other than the file not existing (e.g. the parent
// path component is a file, not a directory, which yields ENOTDIR).
func TestBootstrap_StatErrorPropagated(t *testing.T) {
	dir := t.TempDir()

	// Create a regular file where Bootstrap will expect a directory.
	// Using it as a path component forces os.Stat to fail with ENOTDIR,
	// which is not os.ErrNotExist.
	filePath := filepath.Join(dir, "notadir")
	require.NoError(t, os.WriteFile(filePath, []byte("x"), 0o600))

	// Stat-ing a child inside a regular file returns ENOTDIR, not ErrNotExist.
	path := filepath.Join(filePath, "config.toml")
	err := config.Bootstrap(path)
	require.Error(t, err, "Bootstrap should return an error when stat fails for non-ErrNotExist reason")
	assert.Contains(t, err.Error(), "checking config file")
}
