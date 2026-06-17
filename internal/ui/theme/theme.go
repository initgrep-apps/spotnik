// Package theme defines the Theme interface and the config-driven loader used
// to resolve a config theme ID to a concrete theme implementation.
//
// Components call Theme methods to obtain colours — they never use raw hex
// strings. Themes are loaded from embedded TOML files (built-in) and from
// ~/.config/spotnik/themes/ (user overrides). The active theme is loaded once
// at startup and injected into every pane constructor.
package theme

import (
	"fmt"
	"os"
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
	// OverlayBackground is the solid background used for floating panels that
	// sit on top of dynamic content (e.g., the NowPlaying InfoBox overlaid on
	// the visualizer). Falls back to Base() in all built-in themes.
	OverlayBackground() lipgloss.Color

	// Borders
	ActiveBorder() lipgloss.Color   // Focused pane border
	InactiveBorder() lipgloss.Color // Unfocused pane borders

	// Text hierarchy
	TextPrimary() lipgloss.Color   // Main content — track names, body text
	TextSecondary() lipgloss.Color // Supporting — artist names, subtitles
	TextMuted() lipgloss.Color     // Dim — timestamps, counts, hints

	// Selection
	SelectedBg() lipgloss.Color // Selected list item background (retained for backward compat)
	// SelectedFg is the selection accent applied to item text (name + subtitle lines)
	// when a list item is focused. Used WITHOUT a background fill — must be visually
	// distinct from ColumnSecondary, ColumnTertiary, and TextPrimary in every theme.
	SelectedFg() lipgloss.Color

	// Semantic colours
	SectionHeader() lipgloss.Color    // Section labels: LIBRARY, QUEUE, NOW PLAYING
	PlayingIndicator() lipgloss.Color // playing symbol (GlyphPlaying role)
	SeekBar() lipgloss.Color          // Seek bar fill
	VolumeBar() lipgloss.Color        // Volume bar fill
	Success() lipgloss.Color          // Success states
	Warning() lipgloss.Color          // Caution notices
	Error() lipgloss.Color            // Error messages
	Info() lipgloss.Color             // Informational notices (info toasts)
	HeaderChipFg() lipgloss.Color     // header chip foreground (device chip, profile chip)

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

	// VizGradient1–7 are the seven-stage gradient palette used by the
	// visualizer renderer for per-segment / per-column colour interpolation
	// (Story 223).  They form a continuous ramp from dark base → mid cool →
	// bright accent → white hot, chosen so that every pattern looks good on
	// every theme without hard-coding hex values in the renderer.
	VizGradient1() lipgloss.Color // Darkest background stage
	VizGradient2() lipgloss.Color // Low-mid stage
	VizGradient3() lipgloss.Color // Mid stage
	VizGradient4() lipgloss.Color // Bright cool accent
	VizGradient5() lipgloss.Color // Bright warm accent
	VizGradient6() lipgloss.Color // Near-white hot stage
	VizGradient7() lipgloss.Color // White peak / highlight

	// Tables — dense column header text (Feature 43+)
	TableHeader() lipgloss.Color // Column header text

	// Status — preset label in the header bar
	PresetIndicator() lipgloss.Color // Preset label in header

	// Column colors — distinct foreground for each table column semantic (Feature 70)
	ColumnIndex() lipgloss.Color     // # column (muted but colorful)
	ColumnPrimary() lipgloss.Color   // Main data: track name, playlist name
	ColumnSecondary() lipgloss.Color // Supporting: artist, genre
	ColumnTertiary() lipgloss.Color  // Metadata: duration, year, played time

	// CLI accent — used by cliout palette resolution in "theme" mode (Story 146).
	// Optional in TOML; implementations fall back to SeekBar() when "accent" is unset.
	Accent() lipgloss.Color

	// Metadata
	ID() string   // Config key matching the TOML id field (e.g. "black", "dracula")
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
	fmt.Fprintf(os.Stderr, "spotnik: CRITICAL — no themes loaded, using empty fallback\n")
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
