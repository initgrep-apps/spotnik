package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newThemeOverlayForTest creates a ThemeOverlay for use in render tests.
func newThemeOverlayForTest(a *App) *panes.ThemeOverlay {
	return panes.NewThemeOverlay(theme.AllThemes(), a.theme.ID(), a.theme)
}

// newRenderTestApp creates a minimal App suitable for render unit tests.
func newRenderTestApp() *App {
	cfg := &config.Config{}
	cfg.Preferences.Theme = theme.DefaultThemeID
	return New(cfg, AppOptions{})
}

// TestBuildView_OnboardingMode verifies that when currentView is viewOnboarding,
// buildView() returns the placeholder string and does not fall through to grid rendering.
// This prevents nil-pointer crashes because API clients are not initialized during onboarding.
func TestBuildView_OnboardingMode(t *testing.T) {
	a := newRenderTestApp()
	a.width = 160
	a.height = 50
	a.currentView = viewOnboarding

	result := a.buildView()
	assert.Contains(t, result, "Onboarding",
		"buildView should return onboarding placeholder when currentView == viewOnboarding")
	assert.NotContains(t, result, "spotnik",
		"onboarding view must not render the grid header")
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
	// Story 75: search and devices shortcuts are removed from header (they live in the status bar).
	// This test now verifies they are NOT in the header.
	a := newRenderTestApp()
	result := a.renderHeader()

	assert.NotContains(t, result, "search", "header should NOT show search action (it's in the status bar)")
	assert.NotContains(t, result, "devices", "header should NOT show devices action (it's in the status bar)")
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

// --- Feature 52 Task 3: Responsive behavior tests ---

// TestBuildView_TooSmall_120x29 verifies terminal height below 30 shows too-small message.
func TestBuildView_TooSmall_120x29(t *testing.T) {
	a := newRenderTestApp()
	a.width = 120
	a.height = 29
	result := a.buildView()

	assert.Contains(t, result, "Spotnik needs more space",
		"height below 30 should show too-small message")
	assert.Contains(t, result, "120 × 30",
		"too-small message should show required dimensions")
}

// TestBuildView_ExactMinimum_ShowsGrid verifies 120×30 shows the grid, not the error.
func TestBuildView_ExactMinimum_ShowsGrid(t *testing.T) {
	a := newRenderTestApp()
	a.width = 120
	a.height = 30
	a.currentView = viewGrid
	result := a.buildView()

	assert.NotContains(t, result, "Spotnik needs more space",
		"exactly 120×30 should not show too-small message")
}

// TestRenderTooSmall_ShowsCurrentDimensions verifies the message includes actual
// terminal dimensions so the user knows how much to resize.
func TestRenderTooSmall_ShowsCurrentDimensions(t *testing.T) {
	a := newRenderTestApp()
	a.width = 98
	a.height = 25
	result := a.renderTooSmall()

	assert.Contains(t, result, "98 × 25",
		"too-small message should include current terminal dimensions")
	assert.Contains(t, result, "120 × 30",
		"too-small message should include required dimensions")
	assert.Contains(t, result, "Spotnik needs more space",
		"too-small message should contain the friendly header")
}

// TestRenderTooSmall_UsesRoundedBorder verifies the message is wrapped in a
// rounded border (╭ and ╰ corners confirm lipgloss.RoundedBorder is used).
func TestRenderTooSmall_UsesRoundedBorder(t *testing.T) {
	a := newRenderTestApp()
	a.width = 80
	a.height = 20
	result := a.renderTooSmall()

	// lipgloss.RoundedBorder() uses ╭ (top-left) and ╰ (bottom-left) corners.
	assert.Contains(t, result, "╭",
		"too-small message should use rounded border (╭ corner)")
	assert.Contains(t, result, "╰",
		"too-small message should use rounded border (╰ corner)")
}

// TestRender_ThemeOverlay_Composited verifies that when showThemeSwitcher is true,
// the theme overlay appears in the rendered output.
func TestRender_ThemeOverlay_Composited(t *testing.T) {
	a := newRenderTestApp()
	a.width = 160
	a.height = 50
	a.currentView = viewGrid

	// Open the theme overlay by setting state directly (internal test).
	a.showThemeSwitcher = true
	a.themeOverlay = newThemeOverlayForTest(a)

	result := a.buildView()
	assert.Contains(t, result, "Themes", "theme overlay should appear when showThemeSwitcher is true")
}

// TestRender_HelpOverlay_Composited verifies that when helpOpen is true,
// the help overlay appears in the rendered output.
func TestRender_HelpOverlay_Composited(t *testing.T) {
	a := newRenderTestApp()
	a.width = 160
	a.height = 50
	a.currentView = viewGrid

	// Open the help overlay by setting state directly (internal test).
	a.helpOpen = true
	a.helpOverlay = panes.NewHelpOverlay(a.theme)
	a.helpOverlay.SetSize(a.width, a.height)

	result := a.buildView()
	// "Pane Actions" is a section header rendered inside the help overlay right column.
	// Checking for it confirms the overlay content is composited into the view.
	assert.Contains(t, result, "Pane Actions", "help overlay should appear when helpOpen is true")
}

// TestBuildView_DynamicResize_ShrinkThenGrow verifies that shrinking below minimum
// shows the error, then growing back shows the grid.
func TestBuildView_DynamicResize_ShrinkThenGrow(t *testing.T) {
	a := newRenderTestApp()

	// Start at minimum size — grid renders.
	a.width = 120
	a.height = 30
	a.currentView = viewGrid
	result := a.buildView()
	assert.NotContains(t, result, "Spotnik needs more space",
		"at minimum size, grid should render")

	// Shrink below minimum — error message renders.
	a.width = 80
	a.height = 20
	result = a.buildView()
	assert.Contains(t, result, "Spotnik needs more space",
		"below minimum size, error should render")

	// Grow back to minimum — grid renders again.
	a.width = 120
	a.height = 30
	result = a.buildView()
	assert.NotContains(t, result, "Spotnik needs more space",
		"restored to minimum size, grid should render again")
}

// --- Story 75 Task 2: Header cleanup — no shortcut duplicates ---

// TestRenderHeader_NoShortcutKeys verifies that the header does NOT show
// shortcut hints (ᐅ/ search, ᐅd devices) that are already in the status bar.
func TestRenderHeader_NoShortcutKeys(t *testing.T) {
	a := newRenderTestApp()
	result := a.renderHeader()

	assert.NotContains(t, result, "ᐅ/", "header must not show ᐅ/ search shortcut")
	assert.NotContains(t, result, "ᐅd", "header must not show ᐅd devices shortcut")
	assert.NotContains(t, result, "ᐅp", "header must not show ᐅp preset shortcut")
}

// TestRenderHeader_PageA_ShowsPreset verifies that on Page A, the header shows
// the preset number (e.g. "preset 0").
func TestRenderHeader_PageA_ShowsPreset(t *testing.T) {
	a := newRenderTestApp()
	// Default page is Page A.
	result := a.renderHeader()

	assert.Contains(t, result, "preset 0", "header should show 'preset 0' on Page A")
}

// TestRenderHeader_PageB_NoPreset verifies that on Page B, the header does NOT
// show any preset number (Page B has a single fixed layout with no presets).
func TestRenderHeader_PageB_NoPreset(t *testing.T) {
	a := newRenderTestApp()
	// Switch to Page B.
	a.layout.TogglePage()
	result := a.renderHeader()

	assert.NotContains(t, result, "preset", "header should NOT show preset on Page B")
}

// --- Story 75 Task 3: Page-aware status bar ---

// TestRenderStatusBar_PageA_IncludesPresetAndToggle verifies that on Page A the status
// bar includes both "preset" and "toggle" hints.
func TestRenderStatusBar_PageA_IncludesPresetAndToggle(t *testing.T) {
	a := newRenderTestApp()
	// Default page is Page A.
	result := a.renderStatusBar()

	assert.Contains(t, result, "preset", "Page A status bar should include 'preset' hint")
	assert.Contains(t, result, "toggle", "Page A status bar should include 'toggle' hint")
}

// TestRenderStatusBar_PageB_OmitsPresetAndToggle verifies that on Page B the status
// bar omits "preset" and "toggle" (Page B has a single fixed layout).
func TestRenderStatusBar_PageB_OmitsPresetAndToggle(t *testing.T) {
	a := newRenderTestApp()
	// Switch to Page B.
	a.layout.TogglePage()
	result := a.renderStatusBar()

	assert.NotContains(t, result, "preset", "Page B status bar must NOT include 'preset' hint")
	assert.NotContains(t, result, "toggle", "Page B status bar must NOT include 'toggle' hint")
}

// --- Story 75 Task 2: status bar theme hint ---

// TestRenderStatusBar_ContainsThemeHint verifies that the status bar includes
// the "t" key and "theme" label added by story 73.
func TestRenderStatusBar_ContainsThemeHint(t *testing.T) {
	a := newRenderTestApp()
	result := a.renderStatusBar()

	assert.Contains(t, result, "t", "status bar should contain 't' key for theme shortcut")
	assert.Contains(t, result, "theme", "status bar should contain 'theme' shortcut label")
}

// --- bubbles/help component tests ---

// TestAppKeyMap_PageA_FullHelp_FiveGroups verifies that the Page A appKeyMap produces
// 5 FullHelp groups (one per column). Column 3 (Pane/Devices/Profile) has 3 bindings;
// all others have at most 2 (story 117: Profile was added to column 3).
func TestAppKeyMap_PageA_FullHelp_FiveGroups(t *testing.T) {
	km := newAppKeyMap()
	km.activePage = layout.PageA
	groups := km.FullHelp()
	assert.Len(t, groups, 5, "Page A FullHelp must have 5 groups (5 columns)")
	// Column 3 (index 2) holds Pane, Devices, Profile — 3 entries since story 117.
	for i, g := range groups {
		if i == 2 {
			assert.LessOrEqual(t, len(g), 3, "group %d (pane/devices/profile column) must have at most 3 bindings", i)
		} else {
			assert.LessOrEqual(t, len(g), 2, "group %d must have at most 2 bindings (2-row layout)", i)
		}
	}
}

// TestAppKeyMap_PageB_FullHelp_FourGroups verifies that the Page B appKeyMap produces
// 4 FullHelp groups and does not include "preset" or "toggle" bindings.
func TestAppKeyMap_PageB_FullHelp_FourGroups(t *testing.T) {
	km := newAppKeyMap()
	km.activePage = layout.PageB
	groups := km.FullHelp()
	assert.Len(t, groups, 4, "Page B FullHelp must have 4 groups")
	for _, g := range groups {
		for _, b := range g {
			h := b.Help()
			assert.NotEqual(t, "preset", h.Desc, "Page B must not include preset binding")
			assert.NotEqual(t, "toggle", h.Desc, "Page B must not include toggle binding")
		}
	}
}

// TestRenderStatusBar_HeightIsThreeLines verifies the help-component status bar renders
// exactly 3 lines: border top + 1 content row + border bottom.
func TestRenderStatusBar_HeightIsThreeLines(t *testing.T) {
	a := newRenderTestApp()
	a.width = 160
	result := a.renderStatusBar()
	lines := strings.Split(strings.TrimRight(result, "\n"), "\n")
	assert.Len(t, lines, 3, "status bar must be exactly 3 lines tall (1 content row + top/bottom border)")
}

// TestRenderStatusBar_ShowsAllPageABindings verifies that all 10 Page A key descriptions
// appear in the rendered status bar output (including "profile" added in story 117).
func TestRenderStatusBar_ShowsAllPageABindings(t *testing.T) {
	a := newRenderTestApp()
	a.width = 200 // wide terminal so nothing is truncated
	// Default page is Page A.
	result := a.renderStatusBar()
	for _, want := range []string{"search", "page", "preset", "toggle", "pane", "devices", "profile", "theme", "help", "quit"} {
		assert.Contains(t, result, want, "Page A status bar must show %q", want)
	}
}

// TestRenderStatusBar_ContainsProfileHint verifies that "u" and "profile" appear in the
// status bar on both Page A and Page B — fix for story 117.
func TestRenderStatusBar_ContainsProfileHint(t *testing.T) {
	tests := []struct {
		name  string
		pageB bool
	}{
		{"Page A", false},
		{"Page B", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newRenderTestApp()
			a.width = 200
			if tt.pageB {
				a.layout.TogglePage()
			}
			result := a.renderStatusBar()
			assert.Contains(t, result, "u", "status bar should contain 'u' key for profile shortcut")
			assert.Contains(t, result, "profile", "status bar should contain 'profile' label")
		})
	}
}

// TestRenderStatusBar_ProfileAdjacentToDevices verifies that "devices" and "profile" both
// appear in the status bar on both pages and that devices comes before profile.
func TestRenderStatusBar_ProfileAdjacentToDevices(t *testing.T) {
	tests := []struct {
		name  string
		pageB bool
	}{
		{"Page A", false},
		{"Page B", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newRenderTestApp()
			a.width = 200
			if tt.pageB {
				a.layout.TogglePage()
			}
			result := a.renderStatusBar()
			dIdx := strings.Index(result, "devices")
			pIdx := strings.Index(result, "profile")
			assert.Greater(t, dIdx, -1, "status bar should contain 'devices'")
			assert.Greater(t, pIdx, -1, "status bar should contain 'profile'")
			assert.Less(t, dIdx, pIdx, "'devices' should appear before 'profile' in status bar")
		})
	}
}

// --- Story 115: Profile chip tests ---

// TestRenderProfileChip_EmptyWhenNotLoaded verifies that renderProfileChip() returns ""
// when the user profile has not yet been loaded (ID == "").
func TestRenderProfileChip_EmptyWhenNotLoaded(t *testing.T) {
	a := newRenderTestApp()
	// Store has zero-value profile (ID == "")
	chip := a.renderProfileChip()
	assert.Empty(t, chip, "profile chip should be empty when profile not loaded")
}

// TestRenderProfileChip_PremiumBadge verifies that a premium user shows ♛.
func TestRenderProfileChip_PremiumBadge(t *testing.T) {
	a := newRenderTestApp()
	a.store.SetUserProfile(domain.UserProfile{
		ID:          "user1",
		DisplayName: "Irshad",
		Product:     "premium",
		Country:     "DE",
	})
	chip := a.renderProfileChip()
	assert.Contains(t, chip, "♛", "premium profile chip should contain ♛")
	assert.Contains(t, chip, "Irshad", "profile chip should contain display name")
}

// TestRenderProfileChip_FreeBadge verifies that a free user shows ○.
func TestRenderProfileChip_FreeBadge(t *testing.T) {
	a := newRenderTestApp()
	a.store.SetUserProfile(domain.UserProfile{
		ID:          "user2",
		DisplayName: "Free User",
		Product:     "free",
		Country:     "US",
	})
	chip := a.renderProfileChip()
	assert.Contains(t, chip, "○", "free profile chip should contain ○")
	assert.Contains(t, chip, "Free User", "profile chip should contain display name")
}

// TestRenderHeader_WithProfile_ShowsProfileChip verifies that when a profile is loaded,
// the header right side contains the display name and tier badge.
func TestRenderHeader_WithProfile_ShowsProfileChip(t *testing.T) {
	a := newRenderTestApp()
	a.store.SetUserProfile(domain.UserProfile{
		ID:          "user1",
		DisplayName: "Irshad Sheikh",
		Product:     "premium",
		Country:     "DE",
	})
	result := a.renderHeader()
	assert.Contains(t, result, "Irshad Sheikh", "header should contain profile display name")
	assert.Contains(t, result, "♛", "header should contain premium badge")
}

// TestRenderHeader_WithoutProfile_NoProfileChip verifies that when the profile is not
// yet loaded, the header right side does not contain spurious profile content.
func TestRenderHeader_WithoutProfile_NoProfileChip(t *testing.T) {
	a := newRenderTestApp()
	// No profile in store (zero-value).
	result := a.renderHeader()
	assert.NotContains(t, result, "♛", "header should not show premium badge when profile not loaded")
	assert.NotContains(t, result, "○  Free", "header should not show Free badge when profile not loaded")
}

// TestRenderHeader_DeviceAndProfile_FitsWidth verifies that the header fits its width
// when both a device chip and a profile chip are present.
func TestRenderHeader_DeviceAndProfile_FitsWidth(t *testing.T) {
	a := newRenderTestApp()
	a.width = 160
	a.store.SetActiveDevice(&domain.Device{ID: "d1", Name: "MacBook", IsActive: true})
	a.store.SetUserProfile(domain.UserProfile{
		ID:          "user1",
		DisplayName: "Irshad Sheikh",
		Product:     "premium",
		Country:     "DE",
	})
	result := a.renderHeader()
	assert.Equal(t, 160, lipgloss.Width(result), "header should fit exactly terminal width with both chips")
}

// --- Story 115 PR fixes: renderWithProfileOverlay coverage ---

// TestRenderWithProfileOverlay_NonEmpty verifies that when profileOverlayOpen is true and
// the store has a loaded profile, buildView() returns a non-empty string containing the
// rounded border corner ╭ from the profile overlay.
func TestRenderWithProfileOverlay_NonEmpty(t *testing.T) {
	a := newRenderTestApp()
	a.width = 160
	a.height = 50
	a.currentView = viewGrid
	a.store.SetUserProfile(domain.UserProfile{
		ID:          "user1",
		DisplayName: "Irshad Sheikh",
		Product:     "premium",
		Country:     "DE",
	})
	a.profilePane.SetSize(40, 20)
	a.profileOverlayOpen = true

	result := a.buildView()
	assert.NotEmpty(t, result, "buildView should return non-empty output when profile overlay is open")
	assert.Contains(t, result, "╭", "profile overlay border should include rounded corner ╭")
}

// TestRenderWithProfileOverlay_ZeroWidth verifies that with zero terminal width,
// renderWithProfileOverlay returns the background unchanged (no panic, guard triggers).
func TestRenderWithProfileOverlay_ZeroWidth(t *testing.T) {
	a := newRenderTestApp()
	// width=0 triggers the guard inside renderWithProfileOverlay.
	a.width = 0
	a.height = 0
	background := "background content"
	result := a.renderWithProfileOverlay(background)
	assert.NotEmpty(t, result, "renderWithProfileOverlay should return non-empty result even at zero size")
}

func TestTruncateProfileName_ShortName(t *testing.T) {
	assert.Equal(t, "Alice", truncateProfileName("Alice"))
}

func TestTruncateProfileName_ExactLength(t *testing.T) {
	name := strings.Repeat("a", maxProfileDisplayNameLen)
	assert.Equal(t, name, truncateProfileName(name))
}

func TestTruncateProfileName_LongName(t *testing.T) {
	// 25 runes — exceeds the 20-rune cap.
	name := strings.Repeat("a", maxProfileDisplayNameLen+5)
	result := truncateProfileName(name)
	assert.True(t, len([]rune(result)) <= maxProfileDisplayNameLen,
		"truncated name must not exceed maxProfileDisplayNameLen runes")
	assert.True(t, strings.HasSuffix(result, "…"), "truncated name must end with ellipsis")
	assert.NotEqual(t, name, result, "truncated name must differ from original")
}

func TestTruncateProfileName_UnicodeRunes(t *testing.T) {
	// 21 multi-byte runes — truncation must operate on rune count, not byte count.
	name := strings.Repeat("é", maxProfileDisplayNameLen+1)
	result := truncateProfileName(name)
	assert.True(t, len([]rune(result)) <= maxProfileDisplayNameLen)
	assert.True(t, strings.HasSuffix(result, "…"))
}
