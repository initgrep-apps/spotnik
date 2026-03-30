// Package theme defines the Theme interface and the config-driven loader.
// This file contains ConfigTheme — the sole concrete implementation used for
// all built-in and user-provided themes loaded from TOML files.
package theme

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/lipgloss"
)

// themeFile is the top-level structure matching the TOML schema.
type themeFile struct {
	ID          string           `toml:"id"`
	Name        string           `toml:"name"`
	Colors      themeColors      `toml:"colors"`
	PaneBorders paneBorderColors `toml:"pane_borders"`
}

// themeColors holds all color token values parsed from the [colors] section.
type themeColors struct {
	Base             string `toml:"base"`
	Surface          string `toml:"surface"`
	SurfaceAlt       string `toml:"surface_alt"`
	ActiveBorder     string `toml:"active_border"`
	InactiveBorder   string `toml:"inactive_border"`
	TextPrimary      string `toml:"text_primary"`
	TextSecondary    string `toml:"text_secondary"`
	TextMuted        string `toml:"text_muted"`
	SelectedBg       string `toml:"selected_bg"`
	SelectedFg       string `toml:"selected_fg"`
	SectionHeader    string `toml:"section_header"`
	PlayingIndicator string `toml:"playing_indicator"`
	SeekBar          string `toml:"seek_bar"`
	VolumeBar        string `toml:"volume_bar"`
	Success          string `toml:"success"`
	Warning          string `toml:"warning"`
	Error            string `toml:"error"`
	DeviceActive     string `toml:"device_active"`
	StatusBarBg      string `toml:"status_bar_bg"`
	StatusBarFg      string `toml:"status_bar_fg"`
	KeyHint          string `toml:"key_hint"`
	Gradient1        string `toml:"gradient1"`
	Gradient2        string `toml:"gradient2"`
	Gradient3        string `toml:"gradient3"`
	VisualizerFg     string `toml:"visualizer_fg"`
	TableHeader      string `toml:"table_header"`
	PresetIndicator  string `toml:"preset_indicator"`
	ColumnIndex      string `toml:"column_index"`
	ColumnPrimary    string `toml:"column_primary"`
	ColumnSecondary  string `toml:"column_secondary"`
	ColumnTertiary   string `toml:"column_tertiary"`
}

// paneBorderColors holds per-pane border accent values from [pane_borders].
type paneBorderColors struct {
	NowPlaying     string `toml:"now_playing"`
	Queue          string `toml:"queue"`
	Playlists      string `toml:"playlists"`
	Albums         string `toml:"albums"`
	LikedSongs     string `toml:"liked_songs"`
	RecentlyPlayed string `toml:"recently_played"`
	TopTracks      string `toml:"top_tracks"`
	TopArtists     string `toml:"top_artists"`
	RequestFlow    string `toml:"request_flow"`
	NetworkLog     string `toml:"network_log"`
}

// ConfigTheme implements Theme by loading color values from a parsed TOML file.
// This is the sole concrete implementation used for all themes (built-in and user-provided).
type ConfigTheme struct {
	id   string
	name string
	c    themeColors
	pb   paneBorderColors
}

// ParseTheme decodes TOML bytes into a ConfigTheme.
// Returns an error if the TOML is malformed or if the id field is missing.
// This is exported so tests and tooling can parse TOML snippets directly.
func ParseTheme(data []byte) (*ConfigTheme, error) {
	var f themeFile
	if err := toml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing theme: %w", err)
	}
	if f.ID == "" {
		return nil, fmt.Errorf("theme missing id field")
	}
	return &ConfigTheme{id: f.ID, name: f.Name, c: f.Colors, pb: f.PaneBorders}, nil
}

// Metadata

// ID returns the config key for this theme (e.g. "black", "dracula").
func (t *ConfigTheme) ID() string { return t.id }

// Name returns the human-readable display name (e.g. "True Black", "Dracula").
func (t *ConfigTheme) Name() string { return t.name }

// Backgrounds

// Base returns the app canvas background color.
func (t *ConfigTheme) Base() lipgloss.Color { return lipgloss.Color(t.c.Base) }

// Surface returns the pane interior background color.
func (t *ConfigTheme) Surface() lipgloss.Color { return lipgloss.Color(t.c.Surface) }

// SurfaceAlt returns the overlay background color.
func (t *ConfigTheme) SurfaceAlt() lipgloss.Color { return lipgloss.Color(t.c.SurfaceAlt) }

// Borders

// ActiveBorder returns the focused pane border color.
func (t *ConfigTheme) ActiveBorder() lipgloss.Color { return lipgloss.Color(t.c.ActiveBorder) }

// InactiveBorder returns the unfocused pane border color.
func (t *ConfigTheme) InactiveBorder() lipgloss.Color { return lipgloss.Color(t.c.InactiveBorder) }

// Text hierarchy

// TextPrimary returns the main content text color.
func (t *ConfigTheme) TextPrimary() lipgloss.Color { return lipgloss.Color(t.c.TextPrimary) }

// TextSecondary returns the supporting text color.
func (t *ConfigTheme) TextSecondary() lipgloss.Color { return lipgloss.Color(t.c.TextSecondary) }

// TextMuted returns the dim text color for timestamps, counts, hints.
func (t *ConfigTheme) TextMuted() lipgloss.Color { return lipgloss.Color(t.c.TextMuted) }

// Selection

// SelectedBg returns the selected list item background color.
func (t *ConfigTheme) SelectedBg() lipgloss.Color { return lipgloss.Color(t.c.SelectedBg) }

// SelectedFg returns the selected list item foreground color.
func (t *ConfigTheme) SelectedFg() lipgloss.Color { return lipgloss.Color(t.c.SelectedFg) }

// Semantic colours

// SectionHeader returns the section label color.
func (t *ConfigTheme) SectionHeader() lipgloss.Color { return lipgloss.Color(t.c.SectionHeader) }

// PlayingIndicator returns the currently-playing indicator color.
func (t *ConfigTheme) PlayingIndicator() lipgloss.Color {
	return lipgloss.Color(t.c.PlayingIndicator)
}

// SeekBar returns the seek bar fill color.
func (t *ConfigTheme) SeekBar() lipgloss.Color { return lipgloss.Color(t.c.SeekBar) }

// VolumeBar returns the volume bar fill color.
func (t *ConfigTheme) VolumeBar() lipgloss.Color { return lipgloss.Color(t.c.VolumeBar) }

// Success returns the success state color.
func (t *ConfigTheme) Success() lipgloss.Color { return lipgloss.Color(t.c.Success) }

// Warning returns the caution notice color.
func (t *ConfigTheme) Warning() lipgloss.Color { return lipgloss.Color(t.c.Warning) }

// Error returns the error message color.
func (t *ConfigTheme) Error() lipgloss.Color { return lipgloss.Color(t.c.Error) }

// DeviceActive returns the active device indicator color.
func (t *ConfigTheme) DeviceActive() lipgloss.Color { return lipgloss.Color(t.c.DeviceActive) }

// Status bar

// StatusBarBg returns the status bar background color.
func (t *ConfigTheme) StatusBarBg() lipgloss.Color { return lipgloss.Color(t.c.StatusBarBg) }

// StatusBarFg returns the status bar body text color.
func (t *ConfigTheme) StatusBarFg() lipgloss.Color { return lipgloss.Color(t.c.StatusBarFg) }

// KeyHint returns the keybinding label color.
func (t *ConfigTheme) KeyHint() lipgloss.Color { return lipgloss.Color(t.c.KeyHint) }

// Gradient bars

// Gradient1 returns the seek bar start / low volume color.
func (t *ConfigTheme) Gradient1() lipgloss.Color { return lipgloss.Color(t.c.Gradient1) }

// Gradient2 returns the seek bar end / mid volume color.
func (t *ConfigTheme) Gradient2() lipgloss.Color { return lipgloss.Color(t.c.Gradient2) }

// Gradient3 returns the high volume color.
func (t *ConfigTheme) Gradient3() lipgloss.Color { return lipgloss.Color(t.c.Gradient3) }

// Visualizer

// VisualizerFg returns the braille dot foreground color.
func (t *ConfigTheme) VisualizerFg() lipgloss.Color { return lipgloss.Color(t.c.VisualizerFg) }

// Tables

// TableHeader returns the column header text color.
func (t *ConfigTheme) TableHeader() lipgloss.Color { return lipgloss.Color(t.c.TableHeader) }

// Status

// PresetIndicator returns the preset label color.
func (t *ConfigTheme) PresetIndicator() lipgloss.Color { return lipgloss.Color(t.c.PresetIndicator) }

// Per-pane borders

// PaneBorderNowPlaying returns the now-playing pane border accent color.
func (t *ConfigTheme) PaneBorderNowPlaying() lipgloss.Color {
	return lipgloss.Color(t.pb.NowPlaying)
}

// PaneBorderQueue returns the queue pane border accent color.
func (t *ConfigTheme) PaneBorderQueue() lipgloss.Color { return lipgloss.Color(t.pb.Queue) }

// PaneBorderPlaylists returns the playlists pane border accent color.
func (t *ConfigTheme) PaneBorderPlaylists() lipgloss.Color { return lipgloss.Color(t.pb.Playlists) }

// PaneBorderAlbums returns the albums pane border accent color.
func (t *ConfigTheme) PaneBorderAlbums() lipgloss.Color { return lipgloss.Color(t.pb.Albums) }

// PaneBorderLikedSongs returns the liked songs pane border accent color.
func (t *ConfigTheme) PaneBorderLikedSongs() lipgloss.Color { return lipgloss.Color(t.pb.LikedSongs) }

// PaneBorderRecentlyPlayed returns the recently played pane border accent color.
func (t *ConfigTheme) PaneBorderRecentlyPlayed() lipgloss.Color {
	return lipgloss.Color(t.pb.RecentlyPlayed)
}

// PaneBorderTopTracks returns the top tracks pane border accent color.
func (t *ConfigTheme) PaneBorderTopTracks() lipgloss.Color { return lipgloss.Color(t.pb.TopTracks) }

// PaneBorderTopArtists returns the top artists pane border accent color.
func (t *ConfigTheme) PaneBorderTopArtists() lipgloss.Color { return lipgloss.Color(t.pb.TopArtists) }

// PaneBorderRequestFlow returns the request flow pane border accent color.
func (t *ConfigTheme) PaneBorderRequestFlow() lipgloss.Color {
	return lipgloss.Color(t.pb.RequestFlow)
}

// PaneBorderNetworkLog returns the network log pane border accent color.
func (t *ConfigTheme) PaneBorderNetworkLog() lipgloss.Color { return lipgloss.Color(t.pb.NetworkLog) }

// Column colors

// ColumnIndex returns the # column foreground color (muted but colorful).
func (t *ConfigTheme) ColumnIndex() lipgloss.Color { return lipgloss.Color(t.c.ColumnIndex) }

// ColumnPrimary returns the main data column color (track name, playlist name).
func (t *ConfigTheme) ColumnPrimary() lipgloss.Color { return lipgloss.Color(t.c.ColumnPrimary) }

// ColumnSecondary returns the supporting column color (artist, genre).
func (t *ConfigTheme) ColumnSecondary() lipgloss.Color { return lipgloss.Color(t.c.ColumnSecondary) }

// ColumnTertiary returns the metadata column color (duration, year, played time).
func (t *ConfigTheme) ColumnTertiary() lipgloss.Color { return lipgloss.Color(t.c.ColumnTertiary) }

// ---- Registry (lazy-loaded from embedded TOML files) ----

//go:embed themes/*.toml
var builtinThemes embed.FS

var (
	loaded   map[string]*ConfigTheme
	loadOnce sync.Once
)

// ensureLoaded lazily loads all themes on first access using sync.Once.
func ensureLoaded() {
	loadOnce.Do(func() {
		var err error
		loaded, err = loadAll()
		if err != nil || len(loaded) == 0 {
			// NOTE: should never happen when built-in themes are embedded correctly,
			// but guard to avoid a nil map on method calls.
			loaded = make(map[string]*ConfigTheme)
		}
	})
}

// loadAll discovers and loads all built-in embedded themes plus user themes from
// userThemeDir(). Built-in themes are loaded first; user themes override by ID.
func loadAll() (map[string]*ConfigTheme, error) {
	return LoadAllWithUserDir(userThemeDir())
}

// LoadAllWithUserDir loads all built-in themes and then applies user themes from
// the given directory, with user themes overriding built-ins by ID. It is
// exported so tests can inject a custom user theme directory without touching
// the real filesystem.
func LoadAllWithUserDir(userDir string) (map[string]*ConfigTheme, error) {
	themes := make(map[string]*ConfigTheme)

	// 1. Load built-in embedded themes.
	entries, _ := fs.ReadDir(builtinThemes, "themes")
	for _, e := range entries {
		data, err := builtinThemes.ReadFile("themes/" + e.Name())
		if err != nil {
			continue // skip unreadable entries (shouldn't happen with embed)
		}
		t, err := ParseTheme(data)
		if err != nil {
			continue // skip malformed built-in themes
		}
		themes[t.id] = t
	}

	// 2. Load user themes — override built-in themes by ID.
	if entries, err := os.ReadDir(userDir); err == nil {
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".toml") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(userDir, e.Name()))
			if err != nil {
				continue
			}
			t, err := ParseTheme(data)
			if err != nil {
				continue // skip malformed user themes without aborting
			}
			themes[t.id] = t // override built-in if same ID
		}
	}

	return themes, nil
}

// userThemeDir returns the user theme directory path (~/.config/spotnik/themes/).
func userThemeDir() string {
	cfgDir, _ := os.UserConfigDir()
	return filepath.Join(cfgDir, "spotnik", "themes")
}

// AllThemes returns all loaded ConfigTheme instances, sorted by ID.
// Used by the theme switcher overlay to display names and preview colors.
func AllThemes() []*ConfigTheme {
	ensureLoaded()
	themes := make([]*ConfigTheme, 0, len(loaded))
	for _, t := range loaded {
		themes = append(themes, t)
	}
	sort.Slice(themes, func(i, j int) bool { return themes[i].id < themes[j].id })
	return themes
}
