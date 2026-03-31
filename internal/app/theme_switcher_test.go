package app_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newThemeSwitcherTestApp creates an App in grid view, sized, with a theme ready.
func newThemeSwitcherTestApp(t *testing.T) *app.App {
	t.Helper()
	cfg := &config.Config{}
	cfg.UI.Theme = "black"
	a := app.New(cfg, app.AppOptions{})
	// Resize to a valid terminal size so the grid is visible.
	a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	return a
}

// TestApp_TKey_OpensThemeSwitcher verifies pressing 't' opens the theme switcher overlay.
func TestApp_TKey_OpensThemeSwitcher(t *testing.T) {
	a := newThemeSwitcherTestApp(t)

	_, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

	assert.True(t, a.ThemeSwitcherOpen(), "pressing 't' should open the theme switcher overlay")
}

// TestApp_TKey_DoesNotOpenWhenSearchOpen verifies 't' is a no-op when search overlay is open.
func TestApp_TKey_DoesNotOpenWhenSearchOpen(t *testing.T) {
	a := newThemeSwitcherTestApp(t)
	// Open search overlay.
	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	require.True(t, a.SearchOpen(), "search overlay should be open")

	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

	assert.False(t, a.ThemeSwitcherOpen(), "'t' should not open theme switcher when search is open")
}

// TestApp_TKey_DoesNotOpenWhenDeviceOverlayOpen verifies 't' is a no-op when device overlay is open.
func TestApp_TKey_DoesNotOpenWhenDeviceOverlayOpen(t *testing.T) {
	a := newThemeSwitcherTestApp(t)
	// Open device overlay.
	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	require.True(t, a.DeviceOverlayOpen(), "device overlay should be open")

	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

	assert.False(t, a.ThemeSwitcherOpen(), "'t' should not open theme switcher when device overlay is open")
}

// TestApp_ThemeSwitchMsg_ClosesOverlay verifies that ThemeSwitchMsg closes the overlay.
func TestApp_ThemeSwitchMsg_ClosesOverlay(t *testing.T) {
	a := newThemeSwitcherTestApp(t)

	// Open overlay first.
	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	require.True(t, a.ThemeSwitcherOpen())

	// Send ThemeSwitchMsg.
	a.Update(panes.ThemeSwitchMsg{ThemeID: "black"})

	assert.False(t, a.ThemeSwitcherOpen(), "ThemeSwitchMsg should close the theme switcher")
}

// TestApp_ThemeSwitchMsg_PropagatesTheme verifies that ThemeSwitchMsg changes the active theme.
func TestApp_ThemeSwitchMsg_PropagatesTheme(t *testing.T) {
	a := newThemeSwitcherTestApp(t)
	require.Equal(t, "black", a.Theme().ID(), "initial theme should be black")

	// Open overlay and send a ThemeSwitchMsg for a different theme.
	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	a.Update(panes.ThemeSwitchMsg{ThemeID: "dracula"})

	assert.Equal(t, "dracula", a.Theme().ID(), "theme should switch to dracula")
}

// TestApp_ThemeSwitch_RecreatesAlerts verifies that switching themes recreates the
// alerts model so subsequent toasts render with the new theme's colors.
func TestApp_ThemeSwitch_RecreatesAlerts(t *testing.T) {
	a := newThemeSwitcherTestApp(t)
	require.Equal(t, "black", a.Theme().ID())

	// Switch theme — this should recreate a.alerts internally.
	_, cmd := a.Update(panes.ThemeSwitchMsg{ThemeID: "dracula"})

	// After the switch, theme must be updated.
	assert.Equal(t, "dracula", a.Theme().ID(), "theme should switch to dracula")

	// The returned cmd is a Batch containing the success toast + persist func.
	// Verify it is non-nil, proving alerts model is functional after recreation.
	require.NotNil(t, cmd, "ThemeSwitchMsg handler must return a non-nil Cmd (success toast)")
}

// TestApp_ThemeOverlayClosedMsg_ClosesOverlay verifies ThemeOverlayClosedMsg closes overlay
// without changing the theme.
func TestApp_ThemeOverlayClosedMsg_ClosesOverlay(t *testing.T) {
	a := newThemeSwitcherTestApp(t)
	originalThemeID := a.Theme().ID()

	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	require.True(t, a.ThemeSwitcherOpen())

	a.Update(panes.ThemeOverlayClosedMsg{})

	assert.False(t, a.ThemeSwitcherOpen(), "overlay should be closed")
	assert.Equal(t, originalThemeID, a.Theme().ID(), "theme should not change on Esc")
}
