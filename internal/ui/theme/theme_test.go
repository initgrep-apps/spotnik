package theme_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- Task 0b.1 tests: registry and loader ----

func TestLoad_KnownID(t *testing.T) {
	got := theme.Load("monokai")
	require.NotNil(t, got)
	assert.Equal(t, "monokai", got.ID())
}

func TestLoad_UnknownID_FallsBackToDefault(t *testing.T) {
	got := theme.Load("does-not-exist")
	require.NotNil(t, got)
	assert.Equal(t, theme.DefaultThemeID, got.ID())
}

func TestLoad_DefaultTheme(t *testing.T) {
	got := theme.Load("black")
	require.NotNil(t, got)
	assert.Equal(t, "black", got.ID())
}

func TestAvailable_Returns13Entries(t *testing.T) {
	entries := theme.Available()
	// Available() returns sorted IDs — order is alphabetical.
	assert.Equal(t, []string{
		"black", "catppuccin", "dracula", "gruvbox", "light",
		"mono-dark", "mono-light", "monokai", "nord", "rosepine",
		"solarized", "synthwave", "tokyonight",
	}, entries)
}

func TestAvailable_StableOrder(t *testing.T) {
	first := theme.Available()
	second := theme.Available()
	assert.Equal(t, first, second)
}

func TestDefaultThemeID_IsBlack(t *testing.T) {
	assert.Equal(t, "black", theme.DefaultThemeID)
	assert.NotEmpty(t, theme.DefaultThemeID)
}

// ---- Task 0b.2 tests: all five themes ----

// allMethodsReturnNonEmpty verifies that every method on a Theme returns a non-empty value.
// This covers all 43 methods: 23 original + 16 new tokens (Feature 40) + 4 column tokens (Feature 70).
func allMethodsReturnNonEmpty(t *testing.T, th theme.Theme) {
	t.Helper()
	// Original 26 tokens
	assert.NotEmpty(t, string(th.Base()), "Base()")
	assert.NotEmpty(t, string(th.Surface()), "Surface()")
	assert.NotEmpty(t, string(th.SurfaceAlt()), "SurfaceAlt()")
	assert.NotEmpty(t, string(th.ActiveBorder()), "ActiveBorder()")
	assert.NotEmpty(t, string(th.InactiveBorder()), "InactiveBorder()")
	assert.NotEmpty(t, string(th.TextPrimary()), "TextPrimary()")
	assert.NotEmpty(t, string(th.TextSecondary()), "TextSecondary()")
	assert.NotEmpty(t, string(th.TextMuted()), "TextMuted()")
	assert.NotEmpty(t, string(th.SelectedBg()), "SelectedBg()")
	assert.NotEmpty(t, string(th.SelectedFg()), "SelectedFg()")
	assert.NotEmpty(t, string(th.SectionHeader()), "SectionHeader()")
	assert.NotEmpty(t, string(th.PlayingIndicator()), "PlayingIndicator()")
	assert.NotEmpty(t, string(th.SeekBar()), "SeekBar()")
	assert.NotEmpty(t, string(th.VolumeBar()), "VolumeBar()")
	assert.NotEmpty(t, string(th.Success()), "Success()")
	assert.NotEmpty(t, string(th.Warning()), "Warning()")
	assert.NotEmpty(t, string(th.Error()), "Error()")
	assert.NotEmpty(t, string(th.HeaderChipFg()), "HeaderChipFg()")
	assert.NotEmpty(t, string(th.StatusBarBg()), "StatusBarBg()")
	assert.NotEmpty(t, string(th.StatusBarFg()), "StatusBarFg()")
	assert.NotEmpty(t, string(th.KeyHint()), "KeyHint()")
	assert.NotEmpty(t, th.ID(), "ID()")
	assert.NotEmpty(t, th.Name(), "Name()")
	// New 16 tokens (Feature 40)
	assert.NotEmpty(t, string(th.Gradient1()), "Gradient1()")
	assert.NotEmpty(t, string(th.Gradient2()), "Gradient2()")
	assert.NotEmpty(t, string(th.Gradient3()), "Gradient3()")
	assert.NotEmpty(t, string(th.VisualizerFg()), "VisualizerFg()")
	assert.NotEmpty(t, string(th.TableHeader()), "TableHeader()")
	assert.NotEmpty(t, string(th.PresetIndicator()), "PresetIndicator()")
	assert.NotEmpty(t, string(th.PaneBorderNowPlaying()), "PaneBorderNowPlaying()")
	assert.NotEmpty(t, string(th.PaneBorderQueue()), "PaneBorderQueue()")
	assert.NotEmpty(t, string(th.PaneBorderPlaylists()), "PaneBorderPlaylists()")
	assert.NotEmpty(t, string(th.PaneBorderAlbums()), "PaneBorderAlbums()")
	assert.NotEmpty(t, string(th.PaneBorderLikedSongs()), "PaneBorderLikedSongs()")
	assert.NotEmpty(t, string(th.PaneBorderRecentlyPlayed()), "PaneBorderRecentlyPlayed()")
	assert.NotEmpty(t, string(th.PaneBorderTopTracks()), "PaneBorderTopTracks()")
	assert.NotEmpty(t, string(th.PaneBorderTopArtists()), "PaneBorderTopArtists()")
	assert.NotEmpty(t, string(th.PaneBorderRequestFlow()), "PaneBorderRequestFlow()")
	assert.NotEmpty(t, string(th.PaneBorderNetworkLog()), "PaneBorderNetworkLog()")
	// New 4 column tokens (Feature 70)
	assert.NotEmpty(t, string(th.ColumnIndex()), "ColumnIndex()")
	assert.NotEmpty(t, string(th.ColumnPrimary()), "ColumnPrimary()")
	assert.NotEmpty(t, string(th.ColumnSecondary()), "ColumnSecondary()")
	assert.NotEmpty(t, string(th.ColumnTertiary()), "ColumnTertiary()")
	// Story 79: Info token
	assert.NotEmpty(t, string(th.Info()), "Info()")
	// Story 146: Accent token (optional in TOML; falls back to SeekBar so always non-empty)
	assert.NotEmpty(t, string(th.Accent()), "Accent()")
	// Story 221: OverlayBackground token (aliased to Base in built-in themes)
	assert.NotEmpty(t, string(th.OverlayBackground()), "OverlayBackground()")
}

func TestAllThemes_ImplementInterface(t *testing.T) {
	for _, id := range theme.Available() {
		id := id // capture
		t.Run(id, func(t *testing.T) {
			th := theme.Load(id)
			allMethodsReturnNonEmpty(t, th)
		})
	}
}

func TestAllThemes_IDMatchesRegistryKey(t *testing.T) {
	for _, id := range theme.Available() {
		id := id
		t.Run(id, func(t *testing.T) {
			th := theme.Load(id)
			assert.Equal(t, id, th.ID())
		})
	}
}

func TestBlackTheme_Base_IsPureBlack(t *testing.T) {
	th := theme.Load("black")
	assert.Equal(t, "#000000", string(th.Base()))
}

func TestMonokaiTheme_Base(t *testing.T) {
	th := theme.Load("monokai")
	assert.Equal(t, "#272822", string(th.Base()))
}

func TestCatppuccinTheme_Base(t *testing.T) {
	th := theme.Load("catppuccin")
	assert.Equal(t, "#1e1e2e", string(th.Base()))
}

func TestNordTheme_Base(t *testing.T) {
	th := theme.Load("nord")
	assert.Equal(t, "#2e3440", string(th.Base()))
}

func TestLightTheme_Base(t *testing.T) {
	th := theme.Load("light")
	assert.Equal(t, "#eff1f5", string(th.Base()))
}

// ---- Feature 70: interface satisfaction compile-time check ----

// This blank-identifier assignment verifies at compile time that ConfigTheme
// fully implements the Theme interface. If any method is missing, the build
// fails here with a clear "does not implement" error.
var _ theme.Theme = &theme.ConfigTheme{}

// ---- Feature 40: exact hex value tests for all 16 new tokens ----

// newTokenWant holds expected hex values for one theme's new tokens.
type newTokenWant struct {
	gradient1                string
	gradient2                string
	gradient3                string
	visualizerFg             string
	tableHeader              string
	presetIndicator          string
	paneBorderNowPlaying     string
	paneBorderQueue          string
	paneBorderPlaylists      string
	paneBorderAlbums         string
	paneBorderLikedSongs     string
	paneBorderRecentlyPlayed string
	paneBorderTopTracks      string
	paneBorderTopArtists     string
	paneBorderRequestFlow    string
	paneBorderNetworkLog     string
}

func TestNewTokens_ExactValues(t *testing.T) {
	tests := []struct {
		themeID string
		want    newTokenWant
	}{
		{
			themeID: "black",
			want: newTokenWant{
				gradient1:                "#00ff88",
				gradient2:                "#ffcc00",
				gradient3:                "#ff5555",
				visualizerFg:             "#00afff",
				tableHeader:              "#666666",
				presetIndicator:          "#00afff",
				paneBorderNowPlaying:     "#00ff88",
				paneBorderQueue:          "#ffcc00",
				paneBorderPlaylists:      "#00afff",
				paneBorderAlbums:         "#00e5cc",
				paneBorderLikedSongs:     "#00ff88",
				paneBorderRecentlyPlayed: "#00ccaa",
				paneBorderTopTracks:      "#bd93f9",
				paneBorderTopArtists:     "#ff79c6",
				paneBorderRequestFlow:    "#ffb86c",
				paneBorderNetworkLog:     "#ff6ac1",
			},
		},
		{
			themeID: "monokai",
			want: newTokenWant{
				gradient1:                "#a6e22e",
				gradient2:                "#e6db74",
				gradient3:                "#f92672",
				visualizerFg:             "#66d9ef",
				tableHeader:              "#75715e",
				presetIndicator:          "#66d9ef",
				paneBorderNowPlaying:     "#a6e22e",
				paneBorderQueue:          "#fd971f",
				paneBorderPlaylists:      "#66d9ef",
				paneBorderAlbums:         "#e6db74",
				paneBorderLikedSongs:     "#a6e22e",
				paneBorderRecentlyPlayed: "#4dc9b0",
				paneBorderTopTracks:      "#ae81ff",
				paneBorderTopArtists:     "#f92672",
				paneBorderRequestFlow:    "#fd971f",
				paneBorderNetworkLog:     "#f4a261",
			},
		},
		{
			themeID: "catppuccin",
			want: newTokenWant{
				gradient1:                "#a6e3a1",
				gradient2:                "#f9e2af",
				gradient3:                "#f38ba8",
				visualizerFg:             "#89b4fa",
				tableHeader:              "#6c7086",
				presetIndicator:          "#89b4fa",
				paneBorderNowPlaying:     "#a6e3a1",
				paneBorderQueue:          "#f9e2af",
				paneBorderPlaylists:      "#89b4fa",
				paneBorderAlbums:         "#94e2d5",
				paneBorderLikedSongs:     "#a6e3a1",
				paneBorderRecentlyPlayed: "#94e2d5",
				paneBorderTopTracks:      "#cba6f7",
				paneBorderTopArtists:     "#f38ba8",
				paneBorderRequestFlow:    "#fab387",
				paneBorderNetworkLog:     "#eba0ac",
			},
		},
		{
			themeID: "nord",
			want: newTokenWant{
				gradient1:                "#a3be8c",
				gradient2:                "#ebcb8b",
				gradient3:                "#bf616a",
				visualizerFg:             "#88c0d0",
				tableHeader:              "#4c566a",
				presetIndicator:          "#88c0d0",
				paneBorderNowPlaying:     "#a3be8c",
				paneBorderQueue:          "#ebcb8b",
				paneBorderPlaylists:      "#88c0d0",
				paneBorderAlbums:         "#8fbcbb",
				paneBorderLikedSongs:     "#a3be8c",
				paneBorderRecentlyPlayed: "#8fbcbb",
				paneBorderTopTracks:      "#b48ead",
				paneBorderTopArtists:     "#bf616a",
				paneBorderRequestFlow:    "#d08770",
				paneBorderNetworkLog:     "#5e81ac",
			},
		},
		{
			themeID: "light",
			want: newTokenWant{
				gradient1:                "#40a02b",
				gradient2:                "#df8e1d",
				gradient3:                "#d20f39",
				visualizerFg:             "#1e66f5",
				tableHeader:              "#9ca0b0",
				presetIndicator:          "#1e66f5",
				paneBorderNowPlaying:     "#40a02b",
				paneBorderQueue:          "#df8e1d",
				paneBorderPlaylists:      "#1e66f5",
				paneBorderAlbums:         "#179299",
				paneBorderLikedSongs:     "#40a02b",
				paneBorderRecentlyPlayed: "#179299",
				paneBorderTopTracks:      "#8839ef",
				paneBorderTopArtists:     "#d20f39",
				paneBorderRequestFlow:    "#fe640b",
				paneBorderNetworkLog:     "#ea76cb",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.themeID, func(t *testing.T) {
			th := theme.Load(tt.themeID)
			require.NotNil(t, th)

			assert.Equal(t, tt.want.gradient1, string(th.Gradient1()), "Gradient1")
			assert.Equal(t, tt.want.gradient2, string(th.Gradient2()), "Gradient2")
			assert.Equal(t, tt.want.gradient3, string(th.Gradient3()), "Gradient3")
			assert.Equal(t, tt.want.visualizerFg, string(th.VisualizerFg()), "VisualizerFg")
			assert.Equal(t, tt.want.tableHeader, string(th.TableHeader()), "TableHeader")
			assert.Equal(t, tt.want.presetIndicator, string(th.PresetIndicator()), "PresetIndicator")
			assert.Equal(t, tt.want.paneBorderNowPlaying, string(th.PaneBorderNowPlaying()), "PaneBorderNowPlaying")
			assert.Equal(t, tt.want.paneBorderQueue, string(th.PaneBorderQueue()), "PaneBorderQueue")
			assert.Equal(t, tt.want.paneBorderPlaylists, string(th.PaneBorderPlaylists()), "PaneBorderPlaylists")
			assert.Equal(t, tt.want.paneBorderAlbums, string(th.PaneBorderAlbums()), "PaneBorderAlbums")
			assert.Equal(t, tt.want.paneBorderLikedSongs, string(th.PaneBorderLikedSongs()), "PaneBorderLikedSongs")
			assert.Equal(t, tt.want.paneBorderRecentlyPlayed, string(th.PaneBorderRecentlyPlayed()), "PaneBorderRecentlyPlayed")
			assert.Equal(t, tt.want.paneBorderTopTracks, string(th.PaneBorderTopTracks()), "PaneBorderTopTracks")
			assert.Equal(t, tt.want.paneBorderTopArtists, string(th.PaneBorderTopArtists()), "PaneBorderTopArtists")
			assert.Equal(t, tt.want.paneBorderRequestFlow, string(th.PaneBorderRequestFlow()), "PaneBorderRequestFlow")
			assert.Equal(t, tt.want.paneBorderNetworkLog, string(th.PaneBorderNetworkLog()), "PaneBorderNetworkLog")
		})
	}
}

func TestLoad_UnknownID_HasAllNewTokens(t *testing.T) {
	th := theme.Load("unknown-theme-id")
	require.NotNil(t, th)
	// Fallback to black — verify new tokens are present
	assert.Equal(t, "#00ff88", string(th.Gradient1()), "Gradient1 on fallback theme")
	assert.NotEmpty(t, string(th.VisualizerFg()), "VisualizerFg on fallback theme")
	assert.NotEmpty(t, string(th.PaneBorderNowPlaying()), "PaneBorderNowPlaying on fallback theme")
}

// ── Story 77 Task 4: Network Log border vibrant color assertion ───────────────

// TestAllThemes_NetworkLogBorderIsVibrant verifies that every theme's network_log
// pane border color has a saturation > 20% (i.e. not a desaturated grey).
// Saturation is approximated as: max(r,g,b) - min(r,g,b) > 50 on a 0-255 scale.
// Mono themes are exempt — they are intentionally grayscale.
// This prevents regression back to the grey values (#8a8a8a etc.) that existed before.
func TestAllThemes_NetworkLogBorderIsVibrant(t *testing.T) {
	for _, id := range theme.Available() {
		id := id
		t.Run(id, func(t *testing.T) {
			th := theme.Load(id)
			hex := strings.TrimPrefix(string(th.PaneBorderNetworkLog()), "#")
			require.Len(t, hex, 6, "network log color for %s should be a 6-char hex: %q", id, hex)

			var r, g, b uint8
			_, err := fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
			require.NoError(t, err, "could not parse hex color %q for theme %s", hex, id)

			maxC := max3(r, g, b)
			minC := min3(r, g, b)
			saturation := int(maxC) - int(minC)
			if strings.HasPrefix(id, "mono-") {
				assert.LessOrEqual(t, saturation, 50,
					"mono theme network_log border for %s should be grayscale (saturation %d)",
					id, saturation)
				return
			}
			assert.Greater(t, saturation, 50,
				"network_log border for %s (%s) is too grey (saturation %d ≤ 50); use a vibrant color",
				id, "#"+hex, saturation)
		})
	}
}

// max3 returns the largest of three uint8 values.
func max3(a, b, c uint8) uint8 {
	if a >= b && a >= c {
		return a
	}
	if b >= c {
		return b
	}
	return c
}

// min3 returns the smallest of three uint8 values.
func min3(a, b, c uint8) uint8 {
	if a <= b && a <= c {
		return a
	}
	if b <= c {
		return b
	}
	return c
}

// ---- Feature 70: parseTheme tests ----

// minimalValidTOML is the smallest valid theme TOML for testing parseTheme.
const minimalValidTOML = `
id = "test-theme"
name = "Test Theme"

[colors]
base             = "#111111"
surface          = "#222222"
surface_alt      = "#333333"
active_border    = "#444444"
inactive_border  = "#555555"
text_primary     = "#666666"
text_secondary   = "#777777"
text_muted       = "#888888"
selected_bg      = "#999999"
selected_fg      = "#aaaaaa"
section_header   = "#bbbbbb"
playing_indicator = "#cccccc"
seek_bar         = "#dddddd"
volume_bar       = "#eeeeee"
success          = "#ff0000"
warning          = "#00ff00"
error            = "#0000ff"
info             = "#00aaff"
header_chip_fg    = "#112233"
status_bar_bg    = "#223344"
status_bar_fg    = "#334455"
key_hint         = "#445566"
gradient1        = "#556677"
gradient2        = "#667788"
gradient3        = "#778899"
visualizer_fg    = "#889900"
table_header     = "#990011"
preset_indicator = "#001122"
column_index     = "#aa1122"
column_primary   = "#bb2233"
column_secondary = "#cc3344"
column_tertiary  = "#dd4455"

[pane_borders]
now_playing     = "#ee5566"
queue           = "#ff6677"
playlists       = "#006677"
albums          = "#007788"
liked_songs     = "#008899"
recently_played = "#009900"
top_tracks      = "#00aa11"
top_artists     = "#00bb22"
request_flow    = "#00cc33"
network_log     = "#00dd44"
`

func TestParseTheme_ValidTOML(t *testing.T) {
	th, err := theme.ParseTheme([]byte(minimalValidTOML))
	require.NoError(t, err)
	require.NotNil(t, th)
	assert.Equal(t, "test-theme", th.ID())
	assert.Equal(t, "Test Theme", th.Name())
	assert.Equal(t, "#111111", string(th.Base()))
	assert.Equal(t, "#aa1122", string(th.ColumnIndex()))
	assert.Equal(t, "#bb2233", string(th.ColumnPrimary()))
	assert.Equal(t, "#cc3344", string(th.ColumnSecondary()))
	assert.Equal(t, "#dd4455", string(th.ColumnTertiary()))
	assert.Equal(t, "#ee5566", string(th.PaneBorderNowPlaying()))
}

func TestParseTheme_MissingID_ReturnsError(t *testing.T) {
	noID := `
name = "No ID"
[colors]
base = "#000000"
`
	_, err := theme.ParseTheme([]byte(noID))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "id")
}

func TestParseTheme_MalformedTOML_ReturnsError(t *testing.T) {
	bad := `id = "ok" name = [[[ not valid toml`
	_, err := theme.ParseTheme([]byte(bad))
	require.Error(t, err)
}

func TestParseTheme_EmptyColorField_ReturnsError(t *testing.T) {
	// Theme with id but missing most color fields — should fail validation.
	incomplete := `
id = "incomplete"
name = "Incomplete Theme"
[colors]
base = "#000000"
`
	_, err := theme.ParseTheme([]byte(incomplete))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing or empty color field")
	assert.Contains(t, err.Error(), "incomplete")
}

// ---- Feature 70: ConfigTheme interface compliance ----

func TestConfigTheme_ImplementsInterface(t *testing.T) {
	th, err := theme.ParseTheme([]byte(minimalValidTOML))
	require.NoError(t, err)
	// allMethodsReturnNonEmpty already checks all 47 color-returning methods.
	allMethodsReturnNonEmpty(t, th)
}

func TestConfigTheme_ReturnsCorrectColors(t *testing.T) {
	th, err := theme.ParseTheme([]byte(minimalValidTOML))
	require.NoError(t, err)
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"Base", string(th.Base()), "#111111"},
		{"Surface", string(th.Surface()), "#222222"},
		{"SurfaceAlt", string(th.SurfaceAlt()), "#333333"},
		{"ActiveBorder", string(th.ActiveBorder()), "#444444"},
		{"InactiveBorder", string(th.InactiveBorder()), "#555555"},
		{"TextPrimary", string(th.TextPrimary()), "#666666"},
		{"TextSecondary", string(th.TextSecondary()), "#777777"},
		{"TextMuted", string(th.TextMuted()), "#888888"},
		{"SelectedBg", string(th.SelectedBg()), "#999999"},
		{"SelectedFg", string(th.SelectedFg()), "#aaaaaa"},
		{"SectionHeader", string(th.SectionHeader()), "#bbbbbb"},
		{"PlayingIndicator", string(th.PlayingIndicator()), "#cccccc"},
		{"SeekBar", string(th.SeekBar()), "#dddddd"},
		{"VolumeBar", string(th.VolumeBar()), "#eeeeee"},
		{"Success", string(th.Success()), "#ff0000"},
		{"Warning", string(th.Warning()), "#00ff00"},
		{"Error", string(th.Error()), "#0000ff"},
		{"Info", string(th.Info()), "#00aaff"},
		{"HeaderChipFg", string(th.HeaderChipFg()), "#112233"},
		{"StatusBarBg", string(th.StatusBarBg()), "#223344"},
		{"StatusBarFg", string(th.StatusBarFg()), "#334455"},
		{"KeyHint", string(th.KeyHint()), "#445566"},
		{"Gradient1", string(th.Gradient1()), "#556677"},
		{"Gradient2", string(th.Gradient2()), "#667788"},
		{"Gradient3", string(th.Gradient3()), "#778899"},
		{"VisualizerFg", string(th.VisualizerFg()), "#889900"},
		{"TableHeader", string(th.TableHeader()), "#990011"},
		{"PresetIndicator", string(th.PresetIndicator()), "#001122"},
		{"ColumnIndex", string(th.ColumnIndex()), "#aa1122"},
		{"ColumnPrimary", string(th.ColumnPrimary()), "#bb2233"},
		{"ColumnSecondary", string(th.ColumnSecondary()), "#cc3344"},
		{"ColumnTertiary", string(th.ColumnTertiary()), "#dd4455"},
		{"PaneBorderNowPlaying", string(th.PaneBorderNowPlaying()), "#ee5566"},
		{"PaneBorderQueue", string(th.PaneBorderQueue()), "#ff6677"},
		{"PaneBorderPlaylists", string(th.PaneBorderPlaylists()), "#006677"},
		{"PaneBorderAlbums", string(th.PaneBorderAlbums()), "#007788"},
		{"PaneBorderLikedSongs", string(th.PaneBorderLikedSongs()), "#008899"},
		{"PaneBorderRecentlyPlayed", string(th.PaneBorderRecentlyPlayed()), "#009900"},
		{"PaneBorderTopTracks", string(th.PaneBorderTopTracks()), "#00aa11"},
		{"PaneBorderTopArtists", string(th.PaneBorderTopArtists()), "#00bb22"},
		{"PaneBorderRequestFlow", string(th.PaneBorderRequestFlow()), "#00cc33"},
		{"PaneBorderNetworkLog", string(th.PaneBorderNetworkLog()), "#00dd44"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.got)
		})
	}
}

// ---- Feature 70: built-in TOML themes ----

func TestBuiltinThemes_AllLoad(t *testing.T) {
	ids := theme.Available()
	assert.GreaterOrEqual(t, len(ids), 5, "expect at least 5 built-in themes")
	for _, id := range ids {
		id := id
		t.Run(id, func(t *testing.T) {
			th := theme.Load(id)
			require.NotNil(t, th)
			assert.Equal(t, id, th.ID())
			allMethodsReturnNonEmpty(t, th)
		})
	}
}

func TestBuiltinThemes_HexValuesMatch(t *testing.T) {
	// Spot-check key hex values per theme to verify TOML matches the old Go structs.
	tests := []struct {
		id   string
		base string
	}{
		{"black", "#000000"},
		{"monokai", "#272822"},
		{"catppuccin", "#1e1e2e"},
		{"nord", "#2e3440"},
		{"light", "#eff1f5"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.id, func(t *testing.T) {
			th := theme.Load(tt.id)
			require.NotNil(t, th)
			assert.Equal(t, tt.base, string(th.Base()), "Base color mismatch for %s", tt.id)
		})
	}
}

// ---- Feature 70: registry (loadAll, Load, Available, AllThemes) ----

func TestLoad_UnknownID_FallsBack(t *testing.T) {
	th := theme.Load("does-not-exist-xyz")
	require.NotNil(t, th)
	assert.Equal(t, theme.DefaultThemeID, th.ID())
}

func TestAvailable_ReturnsSortedIDs(t *testing.T) {
	ids := theme.Available()
	require.NotEmpty(t, ids)
	for i := 1; i < len(ids); i++ {
		assert.LessOrEqual(t, ids[i-1], ids[i], "Available() must be sorted")
	}
}

func TestAllThemes_ReturnsAll(t *testing.T) {
	all := theme.AllThemes()
	available := theme.Available()
	assert.Len(t, all, len(available), "AllThemes() count must match Available()")
	for _, th := range all {
		assert.NotNil(t, th)
		assert.NotEmpty(t, th.ID())
	}
}

// ---- Feature 70: user theme override ----

func TestUserTheme_OverridesBuiltin(t *testing.T) {
	// Write a temp TOML to a temp dir that mimics the user theme dir.
	// We exercise ParseTheme + loadAll override logic via ParseTheme directly here,
	// since userThemeDir() points at the real filesystem. The full integration
	// (loadAll picking up files from a custom dir) is covered by TestUserThemeDir_Integration.
	userTOML := `
id = "black"
name = "User Black Override"
[colors]
base             = "#123456"
surface          = "#234567"
surface_alt      = "#345678"
active_border    = "#456789"
inactive_border  = "#56789a"
text_primary     = "#6789ab"
text_secondary   = "#789abc"
text_muted       = "#89abcd"
selected_bg      = "#9abcde"
selected_fg      = "#abcdef"
section_header   = "#bcdef0"
playing_indicator = "#cdef01"
seek_bar         = "#def012"
volume_bar       = "#ef0123"
success          = "#f01234"
warning          = "#012345"
error            = "#123450"
info             = "#1188ee"
header_chip_fg    = "#234561"
status_bar_bg    = "#345672"
status_bar_fg    = "#456783"
key_hint         = "#567894"
gradient1        = "#6789a5"
gradient2        = "#789ab6"
gradient3        = "#89abc7"
visualizer_fg    = "#9abcd8"
table_header     = "#abcde9"
preset_indicator = "#bcdef0"
column_index     = "#aabbcc"
column_primary   = "#bbccdd"
column_secondary = "#ccddee"
column_tertiary  = "#ddeeff"
[pane_borders]
now_playing     = "#112233"
queue           = "#223344"
playlists       = "#334455"
albums          = "#445566"
liked_songs     = "#556677"
recently_played = "#667788"
top_tracks      = "#778899"
top_artists     = "#8899aa"
request_flow    = "#99aabb"
network_log     = "#aabbcc"
`
	th, err := theme.ParseTheme([]byte(userTOML))
	require.NoError(t, err)
	assert.Equal(t, "black", th.ID())
	assert.Equal(t, "User Black Override", th.Name())
	assert.Equal(t, "#123456", string(th.Base()))
}

func TestUserThemeDir_NoDir_NoError(t *testing.T) {
	// Verify that LoadAllWithUserDir with a non-existent dir does not panic or error.
	themes, err := theme.LoadAllWithUserDir("/tmp/spotnik-test-nonexistent-dir-xyz")
	require.NoError(t, err)
	// Built-in themes should still load.
	assert.GreaterOrEqual(t, len(themes), 5)
}

func TestUserThemeDir_Integration(t *testing.T) {
	// Write a user theme TOML to a temp dir and verify LoadAllWithUserDir loads it.
	dir := t.TempDir()
	userTOML := `
id = "my-custom-theme"
name = "My Custom Theme"
[colors]
base             = "#abcdef"
surface          = "#bcdef0"
surface_alt      = "#cdef01"
active_border    = "#def012"
inactive_border  = "#ef0123"
text_primary     = "#f01234"
text_secondary   = "#012345"
text_muted       = "#123450"
selected_bg      = "#234561"
selected_fg      = "#345672"
section_header   = "#456783"
playing_indicator = "#567894"
seek_bar         = "#6789a5"
volume_bar       = "#789ab6"
success          = "#89abc7"
warning          = "#9abcd8"
error            = "#abcde9"
info             = "#22bbff"
header_chip_fg    = "#bcdef0"
status_bar_bg    = "#cdef01"
status_bar_fg    = "#def012"
key_hint         = "#ef0123"
gradient1        = "#f01234"
gradient2        = "#012345"
gradient3        = "#123450"
visualizer_fg    = "#234561"
table_header     = "#345672"
preset_indicator = "#456783"
column_index     = "#aabbcc"
column_primary   = "#bbccdd"
column_secondary = "#ccddee"
column_tertiary  = "#ddeeff"
[pane_borders]
now_playing     = "#112233"
queue           = "#223344"
playlists       = "#334455"
albums          = "#445566"
liked_songs     = "#556677"
recently_played = "#667788"
top_tracks      = "#778899"
top_artists     = "#8899aa"
request_flow    = "#99aabb"
network_log     = "#aabbcc"
`
	err := os.WriteFile(filepath.Join(dir, "my-custom-theme.toml"), []byte(userTOML), 0o644)
	require.NoError(t, err)

	themes, err := theme.LoadAllWithUserDir(dir)
	require.NoError(t, err)
	th, ok := themes["my-custom-theme"]
	require.True(t, ok, "custom theme should be present")
	assert.Equal(t, "My Custom Theme", th.Name())
	assert.Equal(t, "#abcdef", string(th.Base()))
}

// ---- Story 72: six new built-in themes ----

func TestDraculaTheme_Loads(t *testing.T) {
	th := theme.Load("dracula")
	require.NotNil(t, th)
	assert.Equal(t, "dracula", th.ID())
	assert.Equal(t, "Dracula", th.Name())
	allMethodsReturnNonEmpty(t, th)
}

func TestDraculaTheme_Base(t *testing.T) {
	th := theme.Load("dracula")
	assert.Equal(t, "#282A36", string(th.Base()))
}

func TestGruvboxTheme_Loads(t *testing.T) {
	th := theme.Load("gruvbox")
	require.NotNil(t, th)
	assert.Equal(t, "gruvbox", th.ID())
	assert.Equal(t, "Gruvbox Dark", th.Name())
	allMethodsReturnNonEmpty(t, th)
}

func TestGruvboxTheme_Base(t *testing.T) {
	th := theme.Load("gruvbox")
	assert.Equal(t, "#282828", string(th.Base()))
}

func TestTokyoNightTheme_Loads(t *testing.T) {
	th := theme.Load("tokyonight")
	require.NotNil(t, th)
	assert.Equal(t, "tokyonight", th.ID())
	assert.Equal(t, "Tokyo Night", th.Name())
	allMethodsReturnNonEmpty(t, th)
}

func TestTokyoNightTheme_Base(t *testing.T) {
	th := theme.Load("tokyonight")
	assert.Equal(t, "#1a1b26", string(th.Base()))
}

func TestRosePineTheme_Loads(t *testing.T) {
	th := theme.Load("rosepine")
	require.NotNil(t, th)
	assert.Equal(t, "rosepine", th.ID())
	assert.Equal(t, "Rose Pine", th.Name())
	allMethodsReturnNonEmpty(t, th)
}

func TestRosePineTheme_Base(t *testing.T) {
	th := theme.Load("rosepine")
	assert.Equal(t, "#191724", string(th.Base()))
}

// ---- Story 146: Accent() token ----

// TestConfigTheme_Accent_fallsBackToSeekBarWhenUnset verifies that when a theme
// has no explicit "accent" TOML field, Accent() returns the same value as SeekBar().
func TestConfigTheme_Accent_fallsBackToSeekBarWhenUnset(t *testing.T) {
	// minimalValidTOML has no "accent" field — fallback must kick in.
	th, err := theme.ParseTheme([]byte(minimalValidTOML))
	require.NoError(t, err)
	// SeekBar is "#dddddd" in minimalValidTOML.
	assert.Equal(t, th.SeekBar(), th.Accent(), "Accent() must fall back to SeekBar() when unset")
}

// TestConfigTheme_Accent_usesExplicitValueWhenSet verifies that when a theme
// sets an explicit "accent" TOML field, Accent() returns that value, not SeekBar().
func TestConfigTheme_Accent_usesExplicitValueWhenSet(t *testing.T) {
	// accent must be inside [colors] section — insert before [pane_borders].
	withAccent := `
id = "test-accent"
name = "Test Accent"

[colors]
base             = "#111111"
surface          = "#222222"
surface_alt      = "#333333"
active_border    = "#444444"
inactive_border  = "#555555"
text_primary     = "#666666"
text_secondary   = "#777777"
text_muted       = "#888888"
selected_bg      = "#999999"
selected_fg      = "#aaaaaa"
section_header   = "#bbbbbb"
playing_indicator = "#cccccc"
seek_bar         = "#dddddd"
volume_bar       = "#eeeeee"
success          = "#ff0000"
warning          = "#00ff00"
error            = "#0000ff"
info             = "#00aaff"
header_chip_fg   = "#112233"
status_bar_bg    = "#223344"
status_bar_fg    = "#334455"
key_hint         = "#445566"
gradient1        = "#556677"
gradient2        = "#667788"
gradient3        = "#778899"
visualizer_fg    = "#889900"
table_header     = "#990011"
preset_indicator = "#001122"
column_index     = "#aa1122"
column_primary   = "#bb2233"
column_secondary = "#cc3344"
column_tertiary  = "#dd4455"
accent           = "#00ff00"

[pane_borders]
now_playing     = "#ee5566"
queue           = "#ff6677"
playlists       = "#006677"
albums          = "#007788"
liked_songs     = "#008899"
recently_played = "#009900"
top_tracks      = "#00aa11"
top_artists     = "#00bb22"
request_flow    = "#00cc33"
network_log     = "#00dd44"
`
	th, err := theme.ParseTheme([]byte(withAccent))
	require.NoError(t, err)
	assert.Equal(t, "#00ff00", string(th.Accent()), "Accent() must return explicit value when set")
	// seek_bar is "#dddddd", accent is "#00ff00" — they must differ.
	assert.NotEqual(t, th.SeekBar(), th.Accent(), "Accent() must not equal SeekBar() when explicitly set")
}

func TestSolarizedTheme_Loads(t *testing.T) {
	th := theme.Load("solarized")
	require.NotNil(t, th)
	assert.Equal(t, "solarized", th.ID())
	assert.Equal(t, "Solarized Dark", th.Name())
	allMethodsReturnNonEmpty(t, th)
}

func TestSolarizedTheme_Base(t *testing.T) {
	th := theme.Load("solarized")
	assert.Equal(t, "#002b36", string(th.Base()))
}

func TestSynthwaveTheme_Loads(t *testing.T) {
	th := theme.Load("synthwave")
	require.NotNil(t, th)
	assert.Equal(t, "synthwave", th.ID())
	assert.Equal(t, "Synthwave '84", th.Name())
	allMethodsReturnNonEmpty(t, th)
}

func TestSynthwaveTheme_Base(t *testing.T) {
	th := theme.Load("synthwave")
	assert.Equal(t, "#262335", string(th.Base()))
}

// TestAllThemes_HaveAllTokens iterates all 13 themes and verifies no color token
// is an empty string. This is 13 × 50 = 650 assertions in total.
func TestAllThemes_HaveAllTokens(t *testing.T) {
	ids := theme.Available()
	assert.Len(t, ids, 13, "expect exactly 13 built-in themes")
	for _, id := range ids {
		id := id
		t.Run(id, func(t *testing.T) {
			th := theme.Load(id)
			require.NotNil(t, th)
			allMethodsReturnNonEmpty(t, th)
		})
	}
}

func TestUserThemeDir_OverridesBuiltin_ViaLoadAllWithUserDir(t *testing.T) {
	// A user theme with id="black" should override the built-in black theme.
	dir := t.TempDir()
	overrideTOML := `
id = "black"
name = "Custom Black"
[colors]
base             = "#ff0000"
surface          = "#ee0000"
surface_alt      = "#dd0000"
active_border    = "#cc0000"
inactive_border  = "#bb0000"
text_primary     = "#aa0000"
text_secondary   = "#990000"
text_muted       = "#880000"
selected_bg      = "#770000"
selected_fg      = "#660000"
section_header   = "#550000"
playing_indicator = "#440000"
seek_bar         = "#330000"
volume_bar       = "#220000"
success          = "#110000"
warning          = "#100000"
error            = "#0f0000"
info             = "#0e00ff"
header_chip_fg    = "#0e0000"
status_bar_bg    = "#0d0000"
status_bar_fg    = "#0c0000"
key_hint         = "#0b0000"
gradient1        = "#0a0000"
gradient2        = "#090000"
gradient3        = "#080000"
visualizer_fg    = "#070000"
table_header     = "#060000"
preset_indicator = "#050000"
column_index     = "#040000"
column_primary   = "#030000"
column_secondary = "#020000"
column_tertiary  = "#010000"
[pane_borders]
now_playing     = "#110000"
queue           = "#220000"
playlists       = "#330000"
albums          = "#440000"
liked_songs     = "#550000"
recently_played = "#660000"
top_tracks      = "#770000"
top_artists     = "#880000"
request_flow    = "#990000"
network_log     = "#aa0000"
`
	err := os.WriteFile(filepath.Join(dir, "black.toml"), []byte(overrideTOML), 0o644)
	require.NoError(t, err)

	themes, err := theme.LoadAllWithUserDir(dir)
	require.NoError(t, err)
	th, ok := themes["black"]
	require.True(t, ok)
	assert.Equal(t, "Custom Black", th.Name())
	// The override should replace the built-in black base color.
	assert.Equal(t, "#ff0000", string(th.Base()))
}

// ---- Story 79: Info() token ----

// TestConfigTheme_Info verifies that the Info() token returns the expected canonical
// info color for the black theme (used as a known reference).
func TestConfigTheme_Info(t *testing.T) {
	th := theme.Load("black")
	require.NotNil(t, th)
	assert.Equal(t, "#00afff", string(th.Info()),
		"black theme Info() should return canonical info color #00afff")
}

// ---- Story 221: OverlayBackground() token ----

// TestTheme_OverlayBackground_EqualsBase verifies that every loaded theme
// resolves OverlayBackground() to its own Base() colour. Story 221 ships the
// token aliased to Base — a future story can split them per theme.
func TestTheme_OverlayBackground_EqualsBase(t *testing.T) {
	themes := theme.AllThemes()
	require.NotEmpty(t, themes, "no themes loaded")
	for _, th := range themes {
		t.Run(th.ID(), func(t *testing.T) {
			assert.Equal(t, th.Base(), th.OverlayBackground(),
				"OverlayBackground() must equal Base() for theme %q", th.ID())
		})
	}
}

// TestAllThemes_InfoTokenNonEmpty verifies every theme has a non-empty Info() token.
func TestAllThemes_InfoTokenNonEmpty(t *testing.T) {
	for _, id := range theme.Available() {
		id := id
		t.Run(id, func(t *testing.T) {
			th := theme.Load(id)
			require.NotNil(t, th)
			assert.NotEmpty(t, string(th.Info()), "Info() must not be empty for theme %s", id)
		})
	}
}
