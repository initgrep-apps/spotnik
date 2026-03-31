// Package config handles loading and providing application configuration.
// Config is read from ~/.config/spotnik/config.toml at startup.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// ThemeValidator is an optional function that reports whether a theme ID is
// valid. It is set by the caller (cmd/root.go) at startup to avoid an import
// cycle between config and ui/theme. When nil, only the empty-string check is
// applied.
//
// Swappable for testing: set to a function that knows the valid IDs.
var ThemeValidator func(id string) bool

// PreferencesConfig holds user-facing preference settings.
type PreferencesConfig struct {
	// Theme is the config key for the active colour theme.
	// Valid values: "black", "monokai", "catppuccin", "nord", "light", etc.
	// Defaults to "black" if unset or unknown.
	Theme string `toml:"theme"`

	// VolumeStep is the percentage change per volume up/down keypress.
	// Defaults to 5 if unset or zero.
	VolumeStep int `toml:"volume_step"`

	// Preset is the Page A layout preset index (0-based).
	// Negative values are clamped to 0 on load.
	Preset int `toml:"preset"`

	// Visualizer is the visualizer pattern index (0-6).
	// Negative values are clamped to 0 on load.
	Visualizer int `toml:"visualizer"`
}

// spotifyConfig holds Spotify-specific configuration.
type spotifyConfig struct {
	ClientID string `toml:"client_id"`
}

// Config holds all application configuration.
type Config struct {
	// ClientID is the Spotify application client ID.
	// May be empty if not set in config — caller provides embedded fallback.
	ClientID    string
	Preferences PreferencesConfig `toml:"preferences"`
}

// Default returns a Config populated with sensible defaults.
func Default() *Config {
	return &Config{
		Preferences: PreferencesConfig{
			Theme:      "black",
			VolumeStep: 5,
		},
	}
}

// Load reads the config file at the given path and returns a populated Config.
// If the file does not exist, Load returns Default() with no error.
// Empty ClientID is not an error — the caller handles the embedded fallback.
// If the TOML is malformed, Load returns a parse error with file path context.
// Invalid preference values are clamped: negative Preset/Visualizer → 0,
// unknown theme → "black".
func Load(path string) (*Config, error) {
	cfg := Default()

	// Use a raw struct for TOML decoding so we can extract the nested spotify section.
	raw := struct {
		Spotify     spotifyConfig     `toml:"spotify"`
		Preferences PreferencesConfig `toml:"preferences"`
	}{
		// Pre-populate Preferences with defaults so unset fields keep their defaults.
		Preferences: cfg.Preferences,
	}

	_, err := toml.DecodeFile(path, &raw)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Missing config file is not an error — use defaults.
			return cfg, nil
		}
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	// Apply parsed fields.
	cfg.ClientID = raw.Spotify.ClientID
	cfg.Preferences = raw.Preferences

	// Clamp theme: if empty or not recognised by the registry, fall back to default.
	// ThemeValidator is registered by cmd/root.go at startup to avoid an import
	// cycle between config and ui/theme. When unset, only the empty-string case
	// is handled here.
	if cfg.Preferences.Theme == "" {
		cfg.Preferences.Theme = "black"
	} else if ThemeValidator != nil && !ThemeValidator(cfg.Preferences.Theme) {
		cfg.Preferences.Theme = "black"
	}

	// Clamp negative Preset to 0. Out-of-range positive values are handled by
	// layout.SetPreset() which ignores invalid indices.
	if cfg.Preferences.Preset < 0 {
		cfg.Preferences.Preset = 0
	}

	// Clamp negative Visualizer to 0. The viz engine also guards out-of-range values.
	if cfg.Preferences.Visualizer < 0 {
		cfg.Preferences.Visualizer = 0
	}

	return cfg, nil
}

// DefaultConfigPath returns the default config file path.
// It is ~/.config/spotnik/config.toml on all platforms.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home is unavailable.
		return "config.toml"
	}
	return fmt.Sprintf("%s/.config/spotnik/config.toml", home)
}

// defaultTemplate is the config file template written on first launch.
// It documents available options as comments and sets sensible defaults.
const defaultTemplate = `# Spotnik configuration
# https://github.com/initgrep-apps/spotnik

[spotify]
# To use your own Spotify app credentials, uncomment and set:
# client_id = "your-client-id-from-spotify-developer-dashboard"

[preferences]
theme = "black"
volume_step = 5
# preset = 0          # Page A layout preset index (0-based)
# visualizer = 0      # Visualizer pattern index (0-6)
`

// Bootstrap creates the config file at path with a default template if it does
// not already exist. Creates the parent directory if needed. If the file already
// exists, Bootstrap is a no-op.
func Bootstrap(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil // file exists, nothing to do
	}

	// Create directory with permissive owner-only access.
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Write template with owner-only read/write permissions.
	if err := os.WriteFile(path, []byte(defaultTemplate), 0o600); err != nil {
		return fmt.Errorf("writing config template: %w", err)
	}
	return nil
}

// PersistTheme updates the [preferences] theme field in the config file at the default path.
// If the file does not exist it is created (with only the theme field).
// Only the theme field is modified — all other settings are preserved.
func PersistTheme(themeID string) error {
	return persistThemeToPath(DefaultConfigPath(), themeID)
}

// PersistThemeTo updates the [preferences] theme field in the config file at the given path.
// It is the path-parameterised variant of PersistTheme, intended for tests.
func PersistThemeTo(cfgPath string, themeID string) error {
	return persistThemeToPath(cfgPath, themeID)
}

// persistThemeToPath is the path-parameterised implementation of PersistTheme,
// used by tests to write to a temporary directory instead of the real config path.
func persistThemeToPath(cfgPath string, themeID string) error {
	// Read existing config (or start fresh if the file doesn't exist).
	cfg := Default()

	// Use the raw struct matching the full TOML schema so we preserve all fields.
	raw := struct {
		Spotify struct {
			ClientID string `toml:"client_id,omitempty"`
		} `toml:"spotify"`
		Preferences PreferencesConfig `toml:"preferences"`
	}{
		Preferences: cfg.Preferences,
	}

	if _, err := toml.DecodeFile(cfgPath, &raw); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("reading config for theme update: %w", err)
	}

	// Update only the theme field; other fields remain as read (or default).
	raw.Preferences.Theme = themeID

	// Ensure the config directory exists.
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o750); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	f, err := os.OpenFile(cfgPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("opening config file for write: %w", err)
	}

	enc := toml.NewEncoder(f)
	encErr := enc.Encode(raw)
	if closeErr := f.Close(); closeErr != nil && encErr == nil {
		return fmt.Errorf("closing config file: %w", closeErr)
	}
	if encErr != nil {
		return fmt.Errorf("writing config: %w", encErr)
	}
	return nil
}
