package theme_test

import (
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

func TestAvailable_Returns5Entries(t *testing.T) {
	entries := theme.Available()
	assert.Equal(t, []string{"black", "monokai", "catppuccin", "nord", "light"}, entries)
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
// This covers all 42 methods: 26 original + 16 new tokens added in Feature 40.
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
	assert.NotEmpty(t, string(th.DeviceActive()), "DeviceActive()")
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

// ---- Feature 40: interface satisfaction compile-time checks ----

// These blank-identifier assignments verify at compile time that each theme
// struct fully implements the Theme interface. If any method is missing, the
// build fails here with a clear "does not implement" error.
var (
	_ theme.Theme = &theme.BlackTheme{}
	_ theme.Theme = &theme.MonokaiTheme{}
	_ theme.Theme = &theme.CatppuccinTheme{}
	_ theme.Theme = &theme.NordTheme{}
	_ theme.Theme = &theme.LightTheme{}
)

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
				paneBorderNetworkLog:     "#8a8a8a",
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
				paneBorderNetworkLog:     "#75715e",
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
				paneBorderNetworkLog:     "#6c7086",
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
				paneBorderNetworkLog:     "#4c566a",
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
				paneBorderNetworkLog:     "#9ca0b0",
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
