// Package config handles loading and providing application configuration.
// Config is read from ~/.config/spotnik/config.toml at startup.
package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// UIConfig holds UI-related configuration settings.
type UIConfig struct {
	// Theme is the config key for the active colour theme.
	// Valid values: "black", "monokai", "catppuccin", "nord", "light".
	// Defaults to "black" if unset or unknown.
	Theme string `toml:"theme"`

	// VolumeStep is the percentage change per volume up/down keypress.
	// Defaults to 5 if unset or zero.
	VolumeStep int `toml:"volume_step"`
}

// spotifyConfig holds Spotify-specific configuration.
type spotifyConfig struct {
	ClientID string `toml:"client_id"`
}

// Config holds all application configuration.
type Config struct {
	// ClientID is the Spotify application client ID, required for auth.
	ClientID string
	UI       UIConfig `toml:"ui"`
}

// Default returns a Config populated with sensible defaults.
func Default() *Config {
	return &Config{
		UI: UIConfig{
			Theme:      "black",
			VolumeStep: 5,
		},
	}
}

// Load reads the config file at the given path and returns a populated Config.
// If the file does not exist, Load returns Default() with no error.
// If client_id is missing, Load returns a descriptive error.
// If the TOML is malformed, Load returns a parse error with file path context.
func Load(path string) (*Config, error) {
	cfg := Default()

	// Use a raw struct for TOML decoding so we can extract the nested spotify section.
	raw := struct {
		Spotify spotifyConfig `toml:"spotify"`
		UI      UIConfig      `toml:"ui"`
	}{
		// Pre-populate UI with defaults so unset fields keep their defaults.
		UI: cfg.UI,
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
	cfg.UI = raw.UI

	// Ensure theme default is preserved if not set.
	if cfg.UI.Theme == "" {
		cfg.UI.Theme = "black"
	}

	// Validate required fields.
	if cfg.ClientID == "" {
		return nil, fmt.Errorf("missing client_id in [spotify] section of %s — "+
			"create an app at https://developer.spotify.com/dashboard and add it to your config", path)
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
