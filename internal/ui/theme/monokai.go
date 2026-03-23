package theme

import "github.com/charmbracelet/lipgloss"

// MonokaiTheme implements Theme using the Monokai colour palette.
type MonokaiTheme struct{}

// ID returns the config key for this theme.
func (t *MonokaiTheme) ID() string { return "monokai" }

// Name returns the human-readable display name.
func (t *MonokaiTheme) Name() string { return "Monokai" }

// Backgrounds
func (t *MonokaiTheme) Base() lipgloss.Color       { return "#272822" }
func (t *MonokaiTheme) Surface() lipgloss.Color    { return "#3e3d32" }
func (t *MonokaiTheme) SurfaceAlt() lipgloss.Color { return "#49483e" }

// Borders
func (t *MonokaiTheme) ActiveBorder() lipgloss.Color   { return "#66d9ef" }
func (t *MonokaiTheme) InactiveBorder() lipgloss.Color { return "#3e3d32" }

// Text hierarchy
func (t *MonokaiTheme) TextPrimary() lipgloss.Color   { return "#f8f8f2" }
func (t *MonokaiTheme) TextSecondary() lipgloss.Color { return "#cfcfc2" }
func (t *MonokaiTheme) TextMuted() lipgloss.Color     { return "#75715e" }

// Selection
func (t *MonokaiTheme) SelectedBg() lipgloss.Color { return "#49483e" }
func (t *MonokaiTheme) SelectedFg() lipgloss.Color { return "#f8f8f2" }

// Semantic colours
func (t *MonokaiTheme) SectionHeader() lipgloss.Color    { return "#66d9ef" }
func (t *MonokaiTheme) PlayingIndicator() lipgloss.Color { return "#a6e22e" }
func (t *MonokaiTheme) SeekBar() lipgloss.Color          { return "#fd971f" }
func (t *MonokaiTheme) VolumeBar() lipgloss.Color        { return "#fd971f" }
func (t *MonokaiTheme) Success() lipgloss.Color          { return "#a6e22e" }
func (t *MonokaiTheme) Warning() lipgloss.Color          { return "#e6db74" }
func (t *MonokaiTheme) Error() lipgloss.Color            { return "#f92672" }
func (t *MonokaiTheme) DeviceActive() lipgloss.Color     { return "#66d9ef" }

// Status bar
func (t *MonokaiTheme) StatusBarBg() lipgloss.Color { return "#1e1f1c" }
func (t *MonokaiTheme) StatusBarFg() lipgloss.Color { return "#75715e" }
func (t *MonokaiTheme) KeyHint() lipgloss.Color     { return "#66d9ef" }
