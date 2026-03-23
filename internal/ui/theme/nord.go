package theme

import "github.com/charmbracelet/lipgloss"

// NordTheme implements Theme using the Nord colour palette.
type NordTheme struct{}

// ID returns the config key for this theme.
func (t *NordTheme) ID() string { return "nord" }

// Name returns the human-readable display name.
func (t *NordTheme) Name() string { return "Nord" }

// Backgrounds
func (t *NordTheme) Base() lipgloss.Color       { return "#2e3440" }
func (t *NordTheme) Surface() lipgloss.Color    { return "#3b4252" }
func (t *NordTheme) SurfaceAlt() lipgloss.Color { return "#434c5e" }

// Borders
func (t *NordTheme) ActiveBorder() lipgloss.Color   { return "#88c0d0" }
func (t *NordTheme) InactiveBorder() lipgloss.Color { return "#3b4252" }

// Text hierarchy
func (t *NordTheme) TextPrimary() lipgloss.Color   { return "#eceff4" }
func (t *NordTheme) TextSecondary() lipgloss.Color { return "#d8dee9" }
func (t *NordTheme) TextMuted() lipgloss.Color     { return "#4c566a" }

// Selection
func (t *NordTheme) SelectedBg() lipgloss.Color { return "#4c566a" }
func (t *NordTheme) SelectedFg() lipgloss.Color { return "#eceff4" }

// Semantic colours
func (t *NordTheme) SectionHeader() lipgloss.Color    { return "#88c0d0" }
func (t *NordTheme) PlayingIndicator() lipgloss.Color { return "#a3be8c" }
func (t *NordTheme) SeekBar() lipgloss.Color          { return "#81a1c1" }
func (t *NordTheme) VolumeBar() lipgloss.Color        { return "#81a1c1" }
func (t *NordTheme) Success() lipgloss.Color          { return "#a3be8c" }
func (t *NordTheme) Warning() lipgloss.Color          { return "#ebcb8b" }
func (t *NordTheme) Error() lipgloss.Color            { return "#bf616a" }
func (t *NordTheme) DeviceActive() lipgloss.Color     { return "#8fbcbb" }

// Status bar
func (t *NordTheme) StatusBarBg() lipgloss.Color { return "#242831" }
func (t *NordTheme) StatusBarFg() lipgloss.Color { return "#4c566a" }
func (t *NordTheme) KeyHint() lipgloss.Color     { return "#88c0d0" }
