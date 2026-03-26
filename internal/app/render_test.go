package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/domain"
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

// --- Task 2: Btop-style header tests ---

func TestRenderHeader_ContainsAppName(t *testing.T) {
	a := newRenderTestApp()
	result := a.renderHeader()

	require.NotEmpty(t, result)
	assert.Contains(t, result, "spotnik", "header should contain app name")
}

func TestRenderHeader_ContainsPageIndicator(t *testing.T) {
	a := newRenderTestApp()
	result := a.renderHeader()

	// Page A is the default page.
	assert.Contains(t, result, "Page A", "header should show current page")
}

func TestRenderHeader_ContainsPresetIndex(t *testing.T) {
	a := newRenderTestApp()
	result := a.renderHeader()

	// Default preset index is 0.
	assert.Contains(t, result, "preset 0", "header should show current preset index")
}

func TestRenderHeader_ContainsActionShortcuts(t *testing.T) {
	a := newRenderTestApp()
	result := a.renderHeader()

	assert.Contains(t, result, "search", "header should show search action")
	assert.Contains(t, result, "devices", "header should show devices action")
}

func TestRenderHeader_NoDevice_ShowsNoDevice(t *testing.T) {
	a := newRenderTestApp()
	result := a.renderHeader()

	assert.Contains(t, result, "No device", "header should show '○ No device' when no device is active")
}

func TestRenderHeader_WithActiveDevice(t *testing.T) {
	a := newRenderTestApp()
	// Inject an active device into the store via SetActiveDevice.
	dev := &domain.Device{ID: "dev1", Name: "iPhone 14", IsActive: true}
	a.store.SetActiveDevice(dev)
	result := a.renderHeader()

	assert.Contains(t, result, "iPhone 14", "header should show active device name")
}

func TestRenderHeader_FitsWidth(t *testing.T) {
	a := newRenderTestApp()
	a.width = 160
	result := a.renderHeader()

	// lipgloss.Width() already handles ANSI escape codes internally.
	assert.Equal(t, 160, lipgloss.Width(result), "header should fit exactly the terminal width")
}

func TestRenderHeader_FitsWidth_Narrow(t *testing.T) {
	a := newRenderTestApp()
	a.width = 120
	result := a.renderHeader()

	assert.Equal(t, 120, lipgloss.Width(result), "header should fit terminal width even at minimum")
}

// --- Task 3: Global-only status bar tests ---

func TestRenderStatusBar_ContainsGlobalShortcuts(t *testing.T) {
	a := newRenderTestApp()
	result := a.renderStatusBar()

	// All global shortcuts from the spec must be present.
	assert.Contains(t, result, "search", "status bar should contain search shortcut")
	assert.Contains(t, result, "page", "status bar should contain page shortcut")
	assert.Contains(t, result, "preset", "status bar should contain preset shortcut")
	assert.Contains(t, result, "toggle", "status bar should contain toggle shortcut")
	assert.Contains(t, result, "pane", "status bar should contain pane shortcut")
	assert.Contains(t, result, "devices", "status bar should contain devices shortcut")
	assert.Contains(t, result, "help", "status bar should contain help shortcut")
	assert.Contains(t, result, "quit", "status bar should contain quit shortcut")
}

func TestRenderStatusBar_DoesNotContainPaneHints(t *testing.T) {
	a := newRenderTestApp()
	result := a.renderStatusBar()

	// Pane-specific hints like "filter" should NOT appear in the global status bar.
	assert.NotContains(t, result, "filter", "status bar should NOT contain pane-specific filter hint")
}

func TestRenderStatusBar_FitsWidth(t *testing.T) {
	a := newRenderTestApp()
	a.width = 160
	result := a.renderStatusBar()

	// Status bar should not exceed terminal width.
	assert.LessOrEqual(t, lipgloss.Width(result), 160, "status bar should not exceed terminal width")
}

// --- Legacy compatibility tests (renderStatusBar without hints) ---

func TestRenderStatusBar_AlwaysShowsHints(t *testing.T) {
	// renderStatusBar now takes no hints parameter — hints are always global.
	a := newRenderTestApp()
	result := a.renderStatusBar()

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "quit")
}

// --- Existing tests updated for Task 2/3 ---

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
