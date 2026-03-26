package theme

import "github.com/charmbracelet/lipgloss"

// CatppuccinTheme implements Theme using the Catppuccin Mocha colour palette.
type CatppuccinTheme struct{}

// ID returns the config key for this theme.
func (t *CatppuccinTheme) ID() string { return "catppuccin" }

// Name returns the human-readable display name.
func (t *CatppuccinTheme) Name() string { return "Catppuccin Mocha" }

// Backgrounds
func (t *CatppuccinTheme) Base() lipgloss.Color       { return "#1e1e2e" }
func (t *CatppuccinTheme) Surface() lipgloss.Color    { return "#313244" }
func (t *CatppuccinTheme) SurfaceAlt() lipgloss.Color { return "#45475a" }

// Borders
func (t *CatppuccinTheme) ActiveBorder() lipgloss.Color   { return "#89b4fa" }
func (t *CatppuccinTheme) InactiveBorder() lipgloss.Color { return "#313244" }

// Text hierarchy
func (t *CatppuccinTheme) TextPrimary() lipgloss.Color   { return "#cdd6f4" }
func (t *CatppuccinTheme) TextSecondary() lipgloss.Color { return "#bac2de" }
func (t *CatppuccinTheme) TextMuted() lipgloss.Color     { return "#6c7086" }

// Selection
func (t *CatppuccinTheme) SelectedBg() lipgloss.Color { return "#b4befe" }
func (t *CatppuccinTheme) SelectedFg() lipgloss.Color { return "#1e1e2e" }

// Semantic colours
func (t *CatppuccinTheme) SectionHeader() lipgloss.Color    { return "#cba6f7" }
func (t *CatppuccinTheme) PlayingIndicator() lipgloss.Color { return "#a6e3a1" }
func (t *CatppuccinTheme) SeekBar() lipgloss.Color          { return "#fab387" }
func (t *CatppuccinTheme) VolumeBar() lipgloss.Color        { return "#fab387" }
func (t *CatppuccinTheme) Success() lipgloss.Color          { return "#a6e3a1" }
func (t *CatppuccinTheme) Warning() lipgloss.Color          { return "#f9e2af" }
func (t *CatppuccinTheme) Error() lipgloss.Color            { return "#f38ba8" }
func (t *CatppuccinTheme) DeviceActive() lipgloss.Color     { return "#94e2d5" }

// Status bar
func (t *CatppuccinTheme) StatusBarBg() lipgloss.Color { return "#11111b" }
func (t *CatppuccinTheme) StatusBarFg() lipgloss.Color { return "#a6adc8" }
func (t *CatppuccinTheme) KeyHint() lipgloss.Color     { return "#89dceb" }

// Gradient bars
func (t *CatppuccinTheme) Gradient1() lipgloss.Color { return "#a6e3a1" }
func (t *CatppuccinTheme) Gradient2() lipgloss.Color { return "#f9e2af" }
func (t *CatppuccinTheme) Gradient3() lipgloss.Color { return "#f38ba8" }

// Visualizer
func (t *CatppuccinTheme) VisualizerFg() lipgloss.Color { return "#89b4fa" }

// Tables
func (t *CatppuccinTheme) TableHeader() lipgloss.Color { return "#6c7086" }

// Status
func (t *CatppuccinTheme) PresetIndicator() lipgloss.Color { return "#89b4fa" }

// Per-pane borders
func (t *CatppuccinTheme) PaneBorderNowPlaying() lipgloss.Color     { return "#a6e3a1" }
func (t *CatppuccinTheme) PaneBorderQueue() lipgloss.Color          { return "#f9e2af" }
func (t *CatppuccinTheme) PaneBorderPlaylists() lipgloss.Color      { return "#89b4fa" }
func (t *CatppuccinTheme) PaneBorderAlbums() lipgloss.Color         { return "#94e2d5" }
func (t *CatppuccinTheme) PaneBorderLikedSongs() lipgloss.Color     { return "#a6e3a1" }
func (t *CatppuccinTheme) PaneBorderRecentlyPlayed() lipgloss.Color { return "#94e2d5" }
func (t *CatppuccinTheme) PaneBorderTopTracks() lipgloss.Color      { return "#cba6f7" }
func (t *CatppuccinTheme) PaneBorderTopArtists() lipgloss.Color     { return "#f38ba8" }
func (t *CatppuccinTheme) PaneBorderRequestFlow() lipgloss.Color    { return "#fab387" }
func (t *CatppuccinTheme) PaneBorderNetworkLog() lipgloss.Color     { return "#6c7086" }
