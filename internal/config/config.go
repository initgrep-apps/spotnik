// Package config handles loading and providing application configuration.
// Config is read from ~/.config/spotnik/config.toml at startup.
package config

import (
	"encoding/hex"
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

	// Preset is the Music page layout preset index (0-based).
	// Negative values are clamped to 0 on load.
	Preset int `toml:"preset"`

	// Visualizer is the visualizer pattern index (0-5).
	// Negative values are clamped to 0 on load.
	Visualizer int `toml:"visualizer"`
}

// spotifyConfig holds Spotify-specific configuration.
type spotifyConfig struct {
	ClientID     string `toml:"client_id"`
	CallbackPort int    `toml:"callback_port"`
}

// CLIConfig holds CLI-specific settings.
type CLIConfig struct {
	// Palette controls how CLI output colours are resolved.
	// Valid values: "auto", "fixed", "theme".
	// Empty or unrecognised values are clamped to "auto" on load.
	Palette string `toml:"palette"`
}

// UIConfig holds TUI rendering preferences.
type UIConfig struct {
	// Glyphs controls whether unicode or ASCII glyphs are used in the TUI.
	// Valid values: "auto" (default), "unicode", "ascii".
	// "auto" inspects LC_ALL/LC_CTYPE/LANG for a UTF-8 marker.
	Glyphs string `toml:"glyphs"`
}

// Validate returns an error if any UIConfig field contains an invalid value.
func (c *UIConfig) Validate() error {
	switch strings.ToLower(strings.TrimSpace(c.Glyphs)) {
	case "", "auto", "unicode", "ascii":
		// OK; empty defaults to auto at resolution time.
	default:
		return fmt.Errorf("ui.glyphs must be one of auto|unicode|ascii, got %q", c.Glyphs)
	}
	return nil
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
	CLI         CLIConfig         `toml:"cli"`
	UI          UIConfig          `toml:"ui"`
}

// validPalettes is the set of accepted palette mode strings.
var validPalettes = map[string]bool{
	"auto":  true,
	"fixed": true,
	"theme": true,
}

// Default returns a Config populated with sensible defaults.
func Default() *Config {
	return &Config{
		CallbackPort: 8888,
		Preferences: PreferencesConfig{
			Theme: "black",
		},
		CLI: CLIConfig{Palette: "auto"},
		UI:  UIConfig{Glyphs: "auto"},
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
		CLI         CLIConfig         `toml:"cli"`
		UI          UIConfig          `toml:"ui"`
	}{
		// Pre-populate fields with defaults so unset fields keep their defaults.
		Preferences: cfg.Preferences,
		CLI:         cfg.CLI,
		UI:          cfg.UI,
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
	cfg.CLI = raw.CLI
	cfg.UI = raw.UI

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

	// Clamp invalid CLI palette to "auto". Empty string (field absent) is silently
	// set to "auto"; non-empty invalid values produce a stderr warning.
	if cfg.CLI.Palette == "" {
		cfg.CLI.Palette = "auto"
	} else if !validPalettes[cfg.CLI.Palette] {
		fmt.Fprintf(os.Stderr, "config: invalid cli.palette %q — using \"auto\"\n", cfg.CLI.Palette)
		cfg.CLI.Palette = "auto"
	}

	// Validate UI config — invalid glyphs value is a hard error (not silently
	// clamped) so users discover typos immediately.
	if err := cfg.UI.Validate(); err != nil {
		return nil, err
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

// ValidateClientID enforces the Spotify client ID shape: exactly 32 hexadecimal
// characters (after trimming whitespace). Used by both the CLI prompt and the
// TUI onboarding step to reject malformed IDs before writing to config.
func ValidateClientID(s string) error {
	s = strings.TrimSpace(s)
	if len(s) != 32 {
		return fmt.Errorf("client ID must be 32 characters (got %d)", len(s))
	}
	if _, err := hex.DecodeString(s); err != nil {
		return fmt.Errorf("client ID must be hexadecimal")
	}
	return nil
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
# preset = 0          # Music page layout preset index (0-based)
# visualizer = 0      # Visualizer pattern index (0-5)

[cli]
# CLI palette: "auto" (default), "fixed", or "theme"
# - auto:  theme colours on dark-bg terminals, fixed elsewhere
# - fixed: always the built-in Spotnik palette
# - theme: inherit the TUI theme (may be unreadable on light terminals)
palette = "auto"

[ui]
# Glyph rendering mode: "auto" (default), "unicode", or "ascii"
# - auto:    use unicode glyphs when LC_ALL/LANG contains UTF-8, else ASCII
# - unicode: always use unicode glyphs (requires a UTF-8 capable terminal)
# - ascii:   always use ASCII fallback glyphs (safe for all terminals)
glyphs = "auto"
`

// sectionPatches lists sections that Bootstrap ensures are present in the config.
// When the file exists but is missing a section, Bootstrap appends the patch block.
// Add new entries here whenever a future feature introduces a new config section.
var sectionPatches = []struct {
	header string
	block  string
}{
	{
		header: "[ui]",
		block: `
[ui]
# Glyph rendering mode: "auto" (default), "unicode", or "ascii"
# - auto:    use unicode glyphs when LC_ALL/LANG contains UTF-8, else ASCII
# - unicode: always use unicode glyphs (requires a UTF-8 capable terminal)
# - ascii:   always use ASCII fallback glyphs (safe for all terminals)
glyphs = "auto"
`,
	},
}

// Bootstrap creates the config file at path with a default template if it does
// not already exist. Creates the parent directory if needed.
// If the file already exists, Bootstrap appends any missing config sections so
// that pre-feature-13 users always have every required section after upgrade.
func Bootstrap(path string) error {
	_, statErr := os.Stat(path)
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("checking config file: %w", statErr)
	}

	if errors.Is(statErr, os.ErrNotExist) {
		// New file — write the full template (existing behaviour).
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			return fmt.Errorf("creating config directory: %w", err)
		}
		if err := os.WriteFile(path, []byte(defaultTemplate), 0o600); err != nil {
			return fmt.Errorf("writing config template: %w", err)
		}
		return nil
	}

	// File exists — append any missing sections.
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading config for patch: %w", err)
	}
	content := string(data)

	var additions strings.Builder
	for _, patch := range sectionPatches {
		if !strings.Contains(content, patch.header) {
			additions.WriteString(patch.block)
		}
	}
	if additions.Len() == 0 {
		return nil // nothing missing
	}

	patched := content
	if len(patched) > 0 && patched[len(patched)-1] != '\n' {
		patched += "\n"
	}
	patched += additions.String()
	return os.WriteFile(path, []byte(patched), 0o600)
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
			if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o600); err != nil {
				return fmt.Errorf("writing config after client ID update (replace): %w", err)
			}
			return nil
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
			if err := os.WriteFile(path, []byte(strings.Join(updated, "\n")), 0o600); err != nil {
				return fmt.Errorf("writing config after client ID update (insert): %w", err)
			}
			return nil
		}
	}

	// Case 3: no [spotify] section — append it.
	appended := strings.TrimRight(string(data), "\n") + "\n\n[spotify]\n" + newLine + "\n"
	if err := os.WriteFile(path, []byte(appended), 0o600); err != nil {
		return fmt.Errorf("writing config after client ID update (append): %w", err)
	}
	return nil
}
