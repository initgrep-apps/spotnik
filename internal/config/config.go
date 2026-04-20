// Package config handles loading and providing application configuration.
// Config is read from ~/.config/spotnik/config.toml at startup.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

	// Preset is the Page A layout preset index (0-based).
	// Negative values are clamped to 0 on load.
	Preset int `toml:"preset"`

	// Visualizer is the visualizer pattern index (0-6).
	// Negative values are clamped to 0 on load.
	Visualizer int `toml:"visualizer"`
}

// spotifyConfig holds Spotify-specific configuration.
type spotifyConfig struct {
	ClientID     string `toml:"client_id"`
	CallbackPort int    `toml:"callback_port"`
}

// Config holds all application configuration.
type Config struct {
	// ClientID is the Spotify application client ID.
	// May be empty if not set in config — user must provide it via onboarding.
	ClientID string

	// CallbackPort is the port the OAuth callback server listens on.
	// Defaults to 8888. Register http://127.0.0.1:<port>/callback in the
	// Spotify Developer Dashboard exactly once — it never changes between launches.
	CallbackPort int

	Preferences PreferencesConfig `toml:"preferences"`
}

// Default returns a Config populated with sensible defaults.
func Default() *Config {
	return &Config{
		CallbackPort: 8888,
		Preferences: PreferencesConfig{
			Theme: "black",
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
	// Only override the default (8888) when an explicit non-zero port is set.
	if raw.Spotify.CallbackPort > 0 {
		cfg.CallbackPort = raw.Spotify.CallbackPort
	}
	cfg.Preferences = raw.Preferences

	// Clamp theme: if empty or not recognised by the registry, fall back to default.
	// ThemeValidator is registered by cmd/root.go at startup to avoid an import
	// cycle between config and ui/theme. When unset, only the empty-string case
	// is handled here.
	if cfg.Preferences.Theme == "" {
		cfg.Preferences.Theme = "black"
	} else if ThemeValidator != nil && !ThemeValidator(cfg.Preferences.Theme) {
		fmt.Fprintf(os.Stderr, "spotnik: warning: unknown theme %q, falling back to \"black\"\n", cfg.Preferences.Theme)
		cfg.Preferences.Theme = "black"
	}

	// Clamp negative Preset to 0. Out-of-range positive values are handled by
	// layout.SetPreset() which ignores invalid indices.
	if cfg.Preferences.Preset < 0 {
		fmt.Fprintf(os.Stderr, "spotnik: warning: negative preset %d clamped to 0\n", cfg.Preferences.Preset)
		cfg.Preferences.Preset = 0
	}

	// Clamp negative Visualizer to 0. The viz engine also guards out-of-range values.
	if cfg.Preferences.Visualizer < 0 {
		fmt.Fprintf(os.Stderr, "spotnik: warning: negative visualizer %d clamped to 0\n", cfg.Preferences.Visualizer)
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
# preset = 0          # Page A layout preset index (0-based)
# visualizer = 0      # Visualizer pattern index (0-6)
`

// Bootstrap creates the config file at path with a default template if it does
// not already exist. Creates the parent directory if needed. If the file already
// exists, Bootstrap is a no-op.
func Bootstrap(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil // file exists, nothing to do
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("checking config file: %w", err)
	}

	// Create directory: owner gets rwx, group gets r-x, others none.
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Write template with owner-only read/write permissions (no group access).
	if err := os.WriteFile(path, []byte(defaultTemplate), 0o600); err != nil {
		return fmt.Errorf("writing config template: %w", err)
	}
	return nil
}

// ClearClientID removes any line whose trimmed content starts with "client_id"
// from the config file at path, then writes the result back. The [spotify]
// section header and all other keys (callback_port, preferences, etc.) are
// preserved. Returns an error if the file cannot be read or written.
func ClearClientID(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading config for client ID removal: %w", err)
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "client_id") {
			continue
		}
		out = append(out, line)
	}
	if err := os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o600); err != nil {
		return fmt.Errorf("writing config after client ID removal: %w", err)
	}
	return nil
}

// SetClientID writes or updates the client_id key in the config file at path.
// Three cases are handled:
//
//   - client_id line already exists → replace it in-place
//   - [spotify] section exists but no client_id → insert after the [spotify] header
//   - Neither exists → append "\n[spotify]\nclient_id = "..."\n"
//
// If the file does not exist, it is created (along with its parent directory).
func SetClientID(path string, clientID string) error {
	newLine := fmt.Sprintf(`client_id = "%s"`, clientID)

	// Ensure the config directory exists before attempting to read or write.
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("reading config for client ID update: %w", err)
	}

	lines := strings.Split(string(data), "\n")

	// Case 1: replace an existing client_id line.
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "client_id") {
			lines[i] = newLine
			return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o600)
		}
	}

	// Case 2: [spotify] section exists but no client_id — insert after the header.
	for i, line := range lines {
		if strings.TrimSpace(line) == "[spotify]" {
			// Insert the new line immediately after the section header.
			updated := make([]string, 0, len(lines)+1)
			updated = append(updated, lines[:i+1]...)
			updated = append(updated, newLine)
			updated = append(updated, lines[i+1:]...)
			return os.WriteFile(path, []byte(strings.Join(updated, "\n")), 0o600)
		}
	}

	// Case 3: no [spotify] section — append it.
	appended := strings.TrimRight(string(data), "\n") + "\n\n[spotify]\n" + newLine + "\n"
	return os.WriteFile(path, []byte(appended), 0o600)
}
