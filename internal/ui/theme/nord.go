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

// Gradient bars
func (t *NordTheme) Gradient1() lipgloss.Color { return "#a3be8c" }
func (t *NordTheme) Gradient2() lipgloss.Color { return "#ebcb8b" }
func (t *NordTheme) Gradient3() lipgloss.Color { return "#bf616a" }

// Visualizer
func (t *NordTheme) VisualizerFg() lipgloss.Color { return "#88c0d0" }

// Tables
func (t *NordTheme) TableHeader() lipgloss.Color { return "#4c566a" }

// Status
func (t *NordTheme) PresetIndicator() lipgloss.Color { return "#88c0d0" }

// Per-pane borders
func (t *NordTheme) PaneBorderNowPlaying() lipgloss.Color     { return "#a3be8c" }
func (t *NordTheme) PaneBorderQueue() lipgloss.Color          { return "#ebcb8b" }
func (t *NordTheme) PaneBorderPlaylists() lipgloss.Color      { return "#88c0d0" }
func (t *NordTheme) PaneBorderAlbums() lipgloss.Color         { return "#8fbcbb" }
func (t *NordTheme) PaneBorderLikedSongs() lipgloss.Color     { return "#a3be8c" }
func (t *NordTheme) PaneBorderRecentlyPlayed() lipgloss.Color { return "#8fbcbb" }
func (t *NordTheme) PaneBorderTopTracks() lipgloss.Color      { return "#b48ead" }
func (t *NordTheme) PaneBorderTopArtists() lipgloss.Color     { return "#bf616a" }
func (t *NordTheme) PaneBorderRequestFlow() lipgloss.Color    { return "#d08770" }
func (t *NordTheme) PaneBorderNetworkLog() lipgloss.Color     { return "#4c566a" }

// Column colors
func (t *NordTheme) ColumnIndex() lipgloss.Color     { return "#4c566a" }
func (t *NordTheme) ColumnPrimary() lipgloss.Color   { return "#a3be8c" }
func (t *NordTheme) ColumnSecondary() lipgloss.Color { return "#88c0d0" }
func (t *NordTheme) ColumnTertiary() lipgloss.Color  { return "#d08770" }
