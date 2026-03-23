// Package config handles loading and providing application configuration.
// Config is read from ~/.config/spotnik/config.toml at startup.
package config

// UIConfig holds UI-related configuration settings.
type UIConfig struct {
	// Theme is the config key for the active colour theme.
	// Valid values: "black", "monokai", "catppuccin", "nord", "light".
	// Defaults to "black" if unset or unknown.
	Theme string `toml:"theme"`
}

// Config holds all application configuration.
type Config struct {
	UI UIConfig `toml:"ui"`
}

// Default returns a Config populated with sensible defaults.
func Default() *Config {
	return &Config{
		UI: UIConfig{
			Theme: "black",
		},
	}
}
