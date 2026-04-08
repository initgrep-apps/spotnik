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

// newHelpTestApp creates an App in grid view, sized, ready for help overlay tests.
func newHelpTestApp(t *testing.T) *app.App {
	t.Helper()
	cfg := &config.Config{}
	cfg.Preferences.Theme = "black"
	a := app.New(cfg, app.AppOptions{})
	a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	return a
}

// TestApp_QuestionMark_OpensHelpOverlay verifies pressing '?' opens the help overlay.
func TestApp_QuestionMark_OpensHelpOverlay(t *testing.T) {
	a := newHelpTestApp(t)

	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})

	assert.True(t, a.HelpOpen(), "pressing '?' should open the help overlay")
}

// TestApp_QuestionMark_DoesNotOpenWhenSearchOpen verifies '?' is a no-op when search overlay is open.
func TestApp_QuestionMark_DoesNotOpenWhenSearchOpen(t *testing.T) {
	a := newHelpTestApp(t)
	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	require.True(t, a.SearchOpen(), "search overlay should be open")

	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})

	assert.False(t, a.HelpOpen(), "'?' should not open help overlay when search is open")
}

// TestApp_QuestionMark_DoesNotOpenWhenDeviceOverlayOpen verifies '?' is a no-op when device overlay is open.
func TestApp_QuestionMark_DoesNotOpenWhenDeviceOverlayOpen(t *testing.T) {
	a := newHelpTestApp(t)
	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	require.True(t, a.DeviceOverlayOpen(), "device overlay should be open")

	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})

	assert.False(t, a.HelpOpen(), "'?' should not open help overlay when device overlay is open")
}

// TestApp_QuestionMark_DoesNotOpenWhenThemeOverlayOpen verifies '?' is a no-op when theme overlay is open.
func TestApp_QuestionMark_DoesNotOpenWhenThemeOverlayOpen(t *testing.T) {
	a := newHelpTestApp(t)
	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	require.True(t, a.ThemeSwitcherOpen(), "theme switcher should be open")

	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})

	assert.False(t, a.HelpOpen(), "'?' should not open help overlay when theme switcher is open")
}

// TestApp_HelpOverlayClosedMsg_ClosesOverlay verifies HelpOverlayClosedMsg closes the overlay.
func TestApp_HelpOverlayClosedMsg_ClosesOverlay(t *testing.T) {
	a := newHelpTestApp(t)

	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	require.True(t, a.HelpOpen(), "help overlay should be open")

	a.Update(panes.HelpOverlayClosedMsg{})

	assert.False(t, a.HelpOpen(), "HelpOverlayClosedMsg should close the help overlay")
}

// TestApp_HelpOverlay_EscKey_Closes verifies that pressing Esc while the help overlay
// is open sends the HelpOverlayClosedMsg and closes the overlay.
func TestApp_HelpOverlay_EscKey_Closes(t *testing.T) {
	a := newHelpTestApp(t)

	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	require.True(t, a.HelpOpen(), "help overlay should be open")

	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd, "Esc in help overlay should produce a command")
	msg := cmd()
	_, ok := msg.(panes.HelpOverlayClosedMsg)
	assert.True(t, ok, "Esc command should produce HelpOverlayClosedMsg")
}
