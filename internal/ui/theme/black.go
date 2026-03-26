package theme

import "github.com/charmbracelet/lipgloss"

// BlackTheme implements Theme using a true-black terminal colour palette.
// This is the default theme for Spotnik.
type BlackTheme struct{}

// ID returns the config key for this theme.
func (t *BlackTheme) ID() string { return "black" }

// Name returns the human-readable display name.
func (t *BlackTheme) Name() string { return "True Black" }

// Backgrounds
func (t *BlackTheme) Base() lipgloss.Color       { return "#000000" }
func (t *BlackTheme) Surface() lipgloss.Color    { return "#0f0f0f" }
func (t *BlackTheme) SurfaceAlt() lipgloss.Color { return "#1a1a1a" }

// Borders
func (t *BlackTheme) ActiveBorder() lipgloss.Color   { return "#00afff" }
func (t *BlackTheme) InactiveBorder() lipgloss.Color { return "#1e1e1e" }

// Text hierarchy
func (t *BlackTheme) TextPrimary() lipgloss.Color   { return "#f0f0f0" }
func (t *BlackTheme) TextSecondary() lipgloss.Color { return "#888888" }
func (t *BlackTheme) TextMuted() lipgloss.Color     { return "#444444" }

// Selection
func (t *BlackTheme) SelectedBg() lipgloss.Color { return "#1c3a5e" }
func (t *BlackTheme) SelectedFg() lipgloss.Color { return "#f0f0f0" }

// Semantic colours
func (t *BlackTheme) SectionHeader() lipgloss.Color    { return "#00afff" }
func (t *BlackTheme) PlayingIndicator() lipgloss.Color { return "#00ff88" }
func (t *BlackTheme) SeekBar() lipgloss.Color          { return "#00afff" }
func (t *BlackTheme) VolumeBar() lipgloss.Color        { return "#00afff" }
func (t *BlackTheme) Success() lipgloss.Color          { return "#00ff88" }
func (t *BlackTheme) Warning() lipgloss.Color          { return "#ffcc00" }
func (t *BlackTheme) Error() lipgloss.Color            { return "#ff5555" }
func (t *BlackTheme) DeviceActive() lipgloss.Color     { return "#00e5cc" }

// Status bar
func (t *BlackTheme) StatusBarBg() lipgloss.Color { return "#000000" }
func (t *BlackTheme) StatusBarFg() lipgloss.Color { return "#444444" }
func (t *BlackTheme) KeyHint() lipgloss.Color     { return "#00afff" }

// Gradient bars
func (t *BlackTheme) Gradient1() lipgloss.Color { return "#00ff88" }
func (t *BlackTheme) Gradient2() lipgloss.Color { return "#ffcc00" }
func (t *BlackTheme) Gradient3() lipgloss.Color { return "#ff5555" }

// Visualizer
func (t *BlackTheme) VisualizerFg() lipgloss.Color { return "#00afff" }

// Tables
func (t *BlackTheme) TableHeader() lipgloss.Color { return "#666666" }

// Status
func (t *BlackTheme) PresetIndicator() lipgloss.Color { return "#00afff" }

// Per-pane borders
func (t *BlackTheme) PaneBorderNowPlaying() lipgloss.Color     { return "#00ff88" }
func (t *BlackTheme) PaneBorderQueue() lipgloss.Color          { return "#ffcc00" }
func (t *BlackTheme) PaneBorderPlaylists() lipgloss.Color      { return "#00afff" }
func (t *BlackTheme) PaneBorderAlbums() lipgloss.Color         { return "#00e5cc" }
func (t *BlackTheme) PaneBorderLikedSongs() lipgloss.Color     { return "#00ff88" }
func (t *BlackTheme) PaneBorderRecentlyPlayed() lipgloss.Color { return "#00ccaa" }
func (t *BlackTheme) PaneBorderTopTracks() lipgloss.Color      { return "#bd93f9" }
func (t *BlackTheme) PaneBorderTopArtists() lipgloss.Color     { return "#ff79c6" }
func (t *BlackTheme) PaneBorderRequestFlow() lipgloss.Color    { return "#ffb86c" }
func (t *BlackTheme) PaneBorderNetworkLog() lipgloss.Color     { return "#8a8a8a" }
