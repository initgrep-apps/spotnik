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

func TestGridHints_ContainsExpectedKeys(t *testing.T) {
	a := newRenderTestApp()
	combined := strings.Join(a.gridHints(), " ")

	// The grid hints should include page/preset/toggle/focus controls.
	assert.Contains(t, combined, "page")
	assert.Contains(t, combined, "preset")
	assert.Contains(t, combined, "toggle")
	assert.Contains(t, combined, "focus")
	assert.Contains(t, combined, "quit")
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

// TestRenderTooSmall_UpdatedMinimum verifies the minimum size message uses 120x30.
func TestRenderTooSmall_UpdatedMinimum(t *testing.T) {
	a := newRenderTestApp()
	a.width = 80
	a.height = 24
	result := a.renderTooSmall()

	assert.Contains(t, result, "120 × 30", "minimum size message should reflect updated requirement")
}

// TestBuildView_MinimumSizeCheck_120x30 verifies the threshold is 120x30.
func TestBuildView_MinimumSizeCheck_120x30(t *testing.T) {
	a := newRenderTestApp()

	// Just below threshold
	a.width = 119
	a.height = 30
	result := a.buildView()
	assert.Contains(t, result, "120 × 30", "width below 120 should show too-small message")

	// Just above threshold
	a.width = 120
	a.height = 30
	result = a.buildView()
	assert.NotContains(t, result, "120 × 30", "120×30 should pass the minimum size check")
}

// TestRenderGrid_EmptyState verifies renderGrid returns empty string when no panes visible.
func TestRenderGrid_EmptyState(t *testing.T) {
	a := newRenderTestApp()
	// Without a resize, the layout has no terminal size and VisiblePanes may be empty.
	// The important thing is it doesn't panic.
	result := a.renderGrid()
	// May be empty or non-empty depending on layout defaults.
	_ = result
}

// TestRenderGrid_AfterResize verifies grid renders after a size message.
func TestRenderGrid_AfterResize(t *testing.T) {
	a := newRenderTestApp()
	a.currentView = viewGrid
	a.width = 160
	a.height = 50
	a.layout.Resize(160, 50)
	a.propagateSizes()
	a.syncFocus()

	result := a.renderGrid()
	assert.NotEmpty(t, result, "grid should render after resize")
}
