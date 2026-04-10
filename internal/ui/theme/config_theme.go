// This file contains ConfigTheme — the sole concrete implementation of Theme
// used for all built-in and user-provided themes loaded from TOML files.
// It also holds the embed directive, lazy registry, and export helpers.
package theme

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
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
	Info             string `toml:"info"`
	HeaderChipFg     string `toml:"header_chip_fg"`
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
// Returns an error if the TOML is malformed, the id field is missing, or any
// color field is empty. This is exported so tests and tooling can parse TOML
// snippets directly.
func ParseTheme(data []byte) (*ConfigTheme, error) {
	var f themeFile
	if err := toml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing theme: %w", err)
	}
	if f.ID == "" {
		return nil, fmt.Errorf("theme missing id field")
	}
	// Validate that all color fields are non-empty.
	if err := validateColorFields(f.ID, f.Colors, f.PaneBorders); err != nil {
		return nil, err
	}
	return &ConfigTheme{id: f.ID, name: f.Name, c: f.Colors, pb: f.PaneBorders}, nil
}

// validateColorFields checks that every string field in themeColors and
// paneBorderColors is non-empty. Returns an error identifying the first
// missing field found.
func validateColorFields(themeID string, c themeColors, pb paneBorderColors) error {
	cv := reflect.ValueOf(c)
	ct := cv.Type()
	for i := 0; i < cv.NumField(); i++ {
		if cv.Field(i).String() == "" {
			return fmt.Errorf("theme %q: missing or empty color field %q", themeID, ct.Field(i).Tag.Get("toml"))
		}
	}
	pv := reflect.ValueOf(pb)
	pt := pv.Type()
	for i := 0; i < pv.NumField(); i++ {
		if pv.Field(i).String() == "" {
			return fmt.Errorf("theme %q: missing or empty color field %q", themeID, pt.Field(i).Tag.Get("toml"))
		}
	}
	return nil
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

// Info returns the informational notice color (used for info toasts).
func (t *ConfigTheme) Info() lipgloss.Color { return lipgloss.Color(t.c.Info) }

// HeaderChipFg returns the foreground color for header chips (device chip, profile chip).
func (t *ConfigTheme) HeaderChipFg() lipgloss.Color { return lipgloss.Color(t.c.HeaderChipFg) }

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
		if err != nil {
			fmt.Fprintf(os.Stderr, "spotnik: failed to load themes: %v\n", err)
			loaded = make(map[string]*ConfigTheme)
		} else if len(loaded) == 0 {
			fmt.Fprintf(os.Stderr, "spotnik: failed to load themes: no themes found\n")
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
			fmt.Fprintf(os.Stderr, "spotnik: skipping built-in theme %s: %v\n", e.Name(), err)
			continue
		}
		t, err := ParseTheme(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "spotnik: skipping built-in theme %s: %v\n", e.Name(), err)
			continue
		}
		themes[t.id] = t
	}

	// 2. Load user themes — override built-in themes by ID.
	if userDir != "" {
		if entries, err := os.ReadDir(userDir); err == nil {
			for _, e := range entries {
				if !strings.HasSuffix(e.Name(), ".toml") {
					continue
				}
				data, err := os.ReadFile(filepath.Join(userDir, e.Name()))
				if err != nil {
					fmt.Fprintf(os.Stderr, "spotnik: skipping user theme %s: %v\n", e.Name(), err)
					continue
				}
				t, err := ParseTheme(data)
				if err != nil {
					fmt.Fprintf(os.Stderr, "spotnik: skipping user theme %s: %v\n", e.Name(), err)
					continue
				}
				themes[t.id] = t // override built-in if same ID
			}
		}
	}

	return themes, nil
}

// userThemeDir returns the user theme directory path (~/.config/spotnik/themes/).
// Returns an empty string if the user config directory cannot be determined,
// which causes the caller to skip user theme loading.
func userThemeDir() string {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
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
