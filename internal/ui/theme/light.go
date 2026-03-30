package theme

import "github.com/charmbracelet/lipgloss"

// LightTheme implements Theme using the Catppuccin Latte (light) colour palette.
type LightTheme struct{}

// ID returns the config key for this theme.
func (t *LightTheme) ID() string { return "light" }

// Name returns the human-readable display name.
func (t *LightTheme) Name() string { return "Light (Catppuccin Latte)" }

// Backgrounds
func (t *LightTheme) Base() lipgloss.Color       { return "#eff1f5" }
func (t *LightTheme) Surface() lipgloss.Color    { return "#e6e9ef" }
func (t *LightTheme) SurfaceAlt() lipgloss.Color { return "#dce0e8" }

// Borders
func (t *LightTheme) ActiveBorder() lipgloss.Color   { return "#1e66f5" }
func (t *LightTheme) InactiveBorder() lipgloss.Color { return "#ccd0da" }

// Text hierarchy
func (t *LightTheme) TextPrimary() lipgloss.Color   { return "#4c4f69" }
func (t *LightTheme) TextSecondary() lipgloss.Color { return "#6c6f85" }
func (t *LightTheme) TextMuted() lipgloss.Color     { return "#9ca0b0" }

// Selection
func (t *LightTheme) SelectedBg() lipgloss.Color { return "#c6d0f5" }
func (t *LightTheme) SelectedFg() lipgloss.Color { return "#4c4f69" }

// Semantic colours
func (t *LightTheme) SectionHeader() lipgloss.Color    { return "#1e66f5" }
func (t *LightTheme) PlayingIndicator() lipgloss.Color { return "#40a02b" }
func (t *LightTheme) SeekBar() lipgloss.Color          { return "#fe640b" }
func (t *LightTheme) VolumeBar() lipgloss.Color        { return "#fe640b" }
func (t *LightTheme) Success() lipgloss.Color          { return "#40a02b" }
func (t *LightTheme) Warning() lipgloss.Color          { return "#df8e1d" }
func (t *LightTheme) Error() lipgloss.Color            { return "#d20f39" }
func (t *LightTheme) DeviceActive() lipgloss.Color     { return "#179299" }

// Status bar
func (t *LightTheme) StatusBarBg() lipgloss.Color { return "#dce0e8" }
func (t *LightTheme) StatusBarFg() lipgloss.Color { return "#6c6f85" }
func (t *LightTheme) KeyHint() lipgloss.Color     { return "#1e66f5" }

// Gradient bars
func (t *LightTheme) Gradient1() lipgloss.Color { return "#40a02b" }
func (t *LightTheme) Gradient2() lipgloss.Color { return "#df8e1d" }
func (t *LightTheme) Gradient3() lipgloss.Color { return "#d20f39" }

// Visualizer
func (t *LightTheme) VisualizerFg() lipgloss.Color { return "#1e66f5" }

// Tables
func (t *LightTheme) TableHeader() lipgloss.Color { return "#9ca0b0" }

// Status
func (t *LightTheme) PresetIndicator() lipgloss.Color { return "#1e66f5" }

// Per-pane borders
func (t *LightTheme) PaneBorderNowPlaying() lipgloss.Color     { return "#40a02b" }
func (t *LightTheme) PaneBorderQueue() lipgloss.Color          { return "#df8e1d" }
func (t *LightTheme) PaneBorderPlaylists() lipgloss.Color      { return "#1e66f5" }
func (t *LightTheme) PaneBorderAlbums() lipgloss.Color         { return "#179299" }
func (t *LightTheme) PaneBorderLikedSongs() lipgloss.Color     { return "#40a02b" }
func (t *LightTheme) PaneBorderRecentlyPlayed() lipgloss.Color { return "#179299" }
func (t *LightTheme) PaneBorderTopTracks() lipgloss.Color      { return "#8839ef" }
func (t *LightTheme) PaneBorderTopArtists() lipgloss.Color     { return "#d20f39" }
func (t *LightTheme) PaneBorderRequestFlow() lipgloss.Color    { return "#fe640b" }
func (t *LightTheme) PaneBorderNetworkLog() lipgloss.Color     { return "#9ca0b0" }

// Column colors
func (t *LightTheme) ColumnIndex() lipgloss.Color     { return "#9ca0b0" }
func (t *LightTheme) ColumnPrimary() lipgloss.Color   { return "#40a02b" }
func (t *LightTheme) ColumnSecondary() lipgloss.Color { return "#1e66f5" }
func (t *LightTheme) ColumnTertiary() lipgloss.Color  { return "#fe640b" }
