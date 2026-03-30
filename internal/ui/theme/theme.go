// Package theme defines the Theme interface and the config-driven loader used
// to resolve a config theme ID to a concrete theme implementation.
//
// Components call Theme methods to obtain colours — they never use raw hex
// strings. Themes are loaded from embedded TOML files (built-in) and from
// ~/.config/spotnik/themes/ (user overrides). The active theme is loaded once
// at startup and injected into every pane constructor.
package theme

import (
	"sort"

	"github.com/charmbracelet/lipgloss"
)

// Theme defines all colour tokens used across the UI.
// Components call these methods — they never use raw hex strings.
type Theme interface {
	// Backgrounds
	Base() lipgloss.Color       // App canvas background
	Surface() lipgloss.Color    // Pane interior background
	SurfaceAlt() lipgloss.Color // Overlay backgrounds (search, devices, help)

	// Borders
	ActiveBorder() lipgloss.Color   // Focused pane border
	InactiveBorder() lipgloss.Color // Unfocused pane borders

	// Text hierarchy
	TextPrimary() lipgloss.Color   // Main content — track names, body text
	TextSecondary() lipgloss.Color // Supporting — artist names, subtitles
	TextMuted() lipgloss.Color     // Dim — timestamps, counts, hints

	// Selection
	SelectedBg() lipgloss.Color // Selected list item background
	SelectedFg() lipgloss.Color // Selected list item foreground

	// Semantic colours
	SectionHeader() lipgloss.Color    // Section labels: LIBRARY, QUEUE, NOW PLAYING
	PlayingIndicator() lipgloss.Color // ▶ currently playing symbol
	SeekBar() lipgloss.Color          // Seek bar fill
	VolumeBar() lipgloss.Color        // Volume bar fill
	Success() lipgloss.Color          // Success states
	Warning() lipgloss.Color          // Caution notices
	Error() lipgloss.Color            // Error messages
	DeviceActive() lipgloss.Color     // ◉ active device indicator

	// Status bar
	StatusBarBg() lipgloss.Color // Status bar background
	StatusBarFg() lipgloss.Color // Status bar body text
	KeyHint() lipgloss.Color     // Keybinding key labels (Space, Tab, etc.)

	// Gradient bars — seek bar fill stages and volume bands (Feature 44)
	Gradient1() lipgloss.Color // Seek bar start / low volume (cool)
	Gradient2() lipgloss.Color // Seek bar end / mid volume
	Gradient3() lipgloss.Color // High volume (hot)

	// Visualizer — braille-dot audio spectrum (Feature 44)
	VisualizerFg() lipgloss.Color // Braille dot foreground

	// Tables — dense column header text (Feature 43+)
	TableHeader() lipgloss.Color // Column header text

	// Status — preset label in the header bar
	PresetIndicator() lipgloss.Color // Preset label in header

	// Per-pane borders — distinct accent colour per pane, btop-style identity (Feature 42)
	PaneBorderNowPlaying() lipgloss.Color     // Green accent
	PaneBorderQueue() lipgloss.Color          // Yellow accent
	PaneBorderPlaylists() lipgloss.Color      // Blue accent
	PaneBorderAlbums() lipgloss.Color         // Cyan accent
	PaneBorderLikedSongs() lipgloss.Color     // Green accent
	PaneBorderRecentlyPlayed() lipgloss.Color // Teal accent
	PaneBorderTopTracks() lipgloss.Color      // Purple accent
	PaneBorderTopArtists() lipgloss.Color     // Pink/red accent
	PaneBorderRequestFlow() lipgloss.Color    // Orange/amber accent
	PaneBorderNetworkLog() lipgloss.Color     // Warm grey accent

	// Column colors — distinct foreground for each table column semantic (Feature 70)
	ColumnIndex() lipgloss.Color     // # column (muted but colorful)
	ColumnPrimary() lipgloss.Color   // Main data: track name, playlist name
	ColumnSecondary() lipgloss.Color // Supporting: artist, genre
	ColumnTertiary() lipgloss.Color  // Metadata: duration, year, played time

	// Metadata
	ID() string   // Config key: "black", "monokai", "catppuccin", "nord", "light"
	Name() string // Display name: "True Black", "Monokai", etc.
}

// DefaultThemeID is the fallback theme ID used when an unknown ID is provided.
// It is always "black" and can never be empty.
const DefaultThemeID = "black"

// Load returns the theme for the given config ID.
// Falls back to DefaultThemeID if the ID is unknown — never panics.
// Themes are loaded lazily from embedded TOML files on first call.
func Load(id string) Theme {
	ensureLoaded()
	if t, ok := loaded[id]; ok {
		return t
	}
	// NOTE: unknown theme ID — fall back to default rather than panic.
	if t, ok := loaded[DefaultThemeID]; ok {
		return t
	}
	// Should never reach here if built-in themes are embedded correctly.
	return &ConfigTheme{id: DefaultThemeID, name: "True Black"}
}

// Available returns all registered theme IDs in sorted order.
// The list grows automatically as new TOML files are embedded or dropped
// into the user theme directory.
func Available() []string {
	ensureLoaded()
	ids := make([]string, 0, len(loaded))
	for id := range loaded {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
