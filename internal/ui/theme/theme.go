// Package theme defines the Theme interface and the registry/loader used
// to resolve a config theme ID to a concrete theme implementation.
//
// Components call Theme methods to obtain colours — they never use raw hex
// strings. The active theme is loaded once at startup and injected into
// every pane constructor.
package theme

import "github.com/charmbracelet/lipgloss"

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

	// Metadata
	ID() string   // Config key: "black", "monokai", "catppuccin", "nord", "light"
	Name() string // Display name: "True Black", "Monokai", etc.
}

// registry maps config IDs to theme constructors.
// Add new themes here — nowhere else needs to change.
var registry = map[string]func() Theme{
	"black":      func() Theme { return &BlackTheme{} },
	"monokai":    func() Theme { return &MonokaiTheme{} },
	"catppuccin": func() Theme { return &CatppuccinTheme{} },
	"nord":       func() Theme { return &NordTheme{} },
	"light":      func() Theme { return &LightTheme{} },
}

// DefaultThemeID is the fallback theme ID used when an unknown ID is provided.
// It is always "black" and can never be empty.
const DefaultThemeID = "black"

// Load returns the theme for the given config ID.
// Falls back to DefaultThemeID if the ID is unknown — never panics.
func Load(id string) Theme {
	if constructor, ok := registry[id]; ok {
		return constructor()
	}
	// NOTE: unknown theme ID — fall back to default rather than panic.
	return registry[DefaultThemeID]()
}

// Available returns all registered theme IDs in a stable order.
// The order is intentional: default first, then alphabetical-ish by familiarity.
func Available() []string {
	return []string{"black", "monokai", "catppuccin", "nord", "light"}
}
