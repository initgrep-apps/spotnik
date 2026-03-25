package app

import (
	"strings"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newRenderTestApp creates a minimal App suitable for render unit tests.
func newRenderTestApp() *App {
	cfg := &config.Config{}
	cfg.UI.Theme = theme.DefaultThemeID
	return New(cfg, AppOptions{})
}

func TestRenderHeader_NoLabel(t *testing.T) {
	a := newRenderTestApp()
	result := a.renderHeader("")

	require.NotEmpty(t, result)
	assert.Contains(t, result, "spotnik", "main header should contain 'spotnik'")
	assert.NotContains(t, result, "[STATS]")
	assert.NotContains(t, result, "[PLAYLISTS]")
}

func TestRenderHeader_StatsLabel(t *testing.T) {
	a := newRenderTestApp()
	result := a.renderHeader("[STATS]")

	require.NotEmpty(t, result)
	assert.Contains(t, result, "spotnik")
	assert.Contains(t, result, "[STATS]")
	assert.NotContains(t, result, "[PLAYLISTS]")
}

func TestRenderHeader_PlaylistsLabel(t *testing.T) {
	a := newRenderTestApp()
	result := a.renderHeader("[PLAYLISTS]")

	require.NotEmpty(t, result)
	assert.Contains(t, result, "spotnik")
	assert.Contains(t, result, "[PLAYLISTS]")
	assert.NotContains(t, result, "[STATS]")
}

func TestRenderHeader_DifferentLabelsProduceDifferentOutput(t *testing.T) {
	a := newRenderTestApp()
	main := a.renderHeader("")
	stats := a.renderHeader("[STATS]")
	playlists := a.renderHeader("[PLAYLISTS]")

	assert.NotEqual(t, main, stats, "different labels must produce different headers")
	assert.NotEqual(t, main, playlists, "different labels must produce different headers")
	assert.NotEqual(t, stats, playlists, "different labels must produce different headers")
}

func TestRenderStatusBar_AlwaysShowsHints(t *testing.T) {
	// After removing statusMsg, renderStatusBar always shows hints.
	// Toast notifications appear via alerts.Render() overlay, not in the status bar.
	a := newRenderTestApp()
	hints := []string{"/ search", "q quit"}
	result := a.renderStatusBar(hints)

	assert.Contains(t, result, "/ search")
	assert.Contains(t, result, "q quit")
}

func TestRenderStatusBar_ShowsHints(t *testing.T) {
	a := newRenderTestApp()
	hints := []string{"/ search", "q quit"}
	result := a.renderStatusBar(hints)

	assert.Contains(t, result, "/ search")
	assert.Contains(t, result, "q quit")
}

func TestRenderStatusBar_DifferentHintsProduceDifferentOutput(t *testing.T) {
	a := newRenderTestApp()
	statsResult := a.renderStatusBar(a.statsHints())
	playlistsResult := a.renderStatusBar(a.playlistsHints())

	assert.NotEqual(t, statsResult, playlistsResult,
		"different hint sets must produce different status bars")
}

func TestMainHints_FocusDependent(t *testing.T) {
	tests := []struct {
		name         string
		focus        focusedPane
		wantContains string
	}{
		{name: "library focus", focus: focusLibrary, wantContains: "like"},
		{name: "queue focus", focus: focusQueue, wantContains: "navigate"},
		{name: "player focus (default)", focus: focusPlayer, wantContains: "Space"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newRenderTestApp()
			a.focus = tt.focus
			hints := a.mainHints()
			combined := strings.Join(hints, " ")
			assert.Contains(t, combined, tt.wantContains,
				"main hints for %s should contain %q", tt.name, tt.wantContains)
		})
	}
}

func TestStatsHints_ContainsExpectedKeys(t *testing.T) {
	a := newRenderTestApp()
	combined := strings.Join(a.statsHints(), " ")

	assert.Contains(t, combined, "cycle range")
	assert.Contains(t, combined, "library")
}

func TestPlaylistsHints_ContainsExpectedKeys(t *testing.T) {
	a := newRenderTestApp()
	combined := strings.Join(a.playlistsHints(), " ")

	assert.Contains(t, combined, "rename")
	assert.Contains(t, combined, "reorder")
}

func TestTruncateDeviceName_ShortName(t *testing.T) {
	assert.Equal(t, "My Speaker", truncateDeviceName("My Speaker"))
}

func TestTruncateDeviceName_ExactLength(t *testing.T) {
	name := strings.Repeat("a", maxDeviceNameLen)
	assert.Equal(t, name, truncateDeviceName(name))
}

func TestTruncateDeviceName_LongName(t *testing.T) {
	name := strings.Repeat("a", maxDeviceNameLen+5)
	result := truncateDeviceName(name)
	assert.True(t, len([]rune(result)) <= maxDeviceNameLen,
		"truncated name should not exceed maxDeviceNameLen")
	assert.True(t, strings.HasSuffix(result, "…"), "truncated name should end with ellipsis")
}
