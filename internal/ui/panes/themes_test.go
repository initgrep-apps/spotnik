package panes

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestThemeOverlay creates a ThemeOverlay with all available themes.
func newTestThemeOverlay() *ThemeOverlay {
	themes := theme.AllThemes()
	current := theme.Load("black")
	return NewThemeOverlay(themes, "black", current)
}

// TestThemeOverlay_View_ShowsThemeNames verifies that all theme names appear in the view.
func TestThemeOverlay_View_ShowsThemeNames(t *testing.T) {
	overlay := newTestThemeOverlay()
	overlay.width = 60
	overlay.height = 20

	view := overlay.View()
	require.NotEmpty(t, view)

	themes := theme.AllThemes()
	for _, th := range themes {
		assert.Contains(t, view, th.Name(), "theme name %q should appear in overlay", th.Name())
	}
}

// TestThemeOverlay_View_CurrentThemeMarked verifies that the current theme is marked with ◉.
func TestThemeOverlay_CurrentThemeMarked(t *testing.T) {
	themes := theme.AllThemes()
	require.NotEmpty(t, themes, "must have at least one theme loaded")

	currentID := "black"
	overlay := NewThemeOverlay(themes, currentID, theme.Load(currentID))
	overlay.width = 60
	overlay.height = 20

	view := overlay.View()
	assert.Contains(t, view, "◉", "current theme should be marked with ◉")
}

// TestThemeOverlay_KeyNavigation_J verifies pressing j moves the cursor down.
func TestThemeOverlay_KeyNavigation(t *testing.T) {
	overlay := newTestThemeOverlay()
	initialCursor := overlay.cursor

	// Press j to move down.
	m, _ := overlay.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated, ok := m.(*ThemeOverlay)
	require.True(t, ok)

	// If there are multiple themes, cursor should have advanced.
	if len(overlay.themes) > 1 {
		assert.Equal(t, initialCursor+1, updated.cursor, "j key should move cursor down")
	}

	// Press k to move back up.
	m2, _ := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	updated2, ok := m2.(*ThemeOverlay)
	require.True(t, ok)

	if len(overlay.themes) > 1 {
		assert.Equal(t, initialCursor, updated2.cursor, "k key should move cursor up")
	}
}

// TestThemeOverlay_Enter_EmitsThemeSwitchMsg verifies that pressing Enter emits ThemeSwitchMsg.
func TestThemeOverlay_Enter_EmitsThemeSwitchMsg(t *testing.T) {
	overlay := newTestThemeOverlay()

	_, cmd := overlay.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd, "Enter should produce a command")

	msg := cmd()
	switchMsg, ok := msg.(ThemeSwitchMsg)
	require.True(t, ok, "Enter command should produce ThemeSwitchMsg, got %T", msg)
	assert.NotEmpty(t, switchMsg.ThemeID, "ThemeSwitchMsg should carry a theme ID")
}

// TestThemeOverlay_Esc_NoMsg verifies that pressing Esc emits ThemeOverlayClosedMsg.
func TestThemeOverlay_Esc_NoMsg(t *testing.T) {
	overlay := newTestThemeOverlay()

	_, cmd := overlay.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd, "Esc should produce a close command")

	msg := cmd()
	_, ok := msg.(ThemeOverlayClosedMsg)
	assert.True(t, ok, "Esc command should produce ThemeOverlayClosedMsg")
}

// TestThemeOverlay_View_BorderTitle verifies the overlay has a "Themes" border title.
func TestThemeOverlay_View_BorderTitle(t *testing.T) {
	overlay := newTestThemeOverlay()
	overlay.width = 60
	overlay.height = 20

	view := overlay.View()
	assert.Contains(t, view, "Themes", "overlay should show 'Themes' in its border title")
}

// TestThemeOverlay_View_ShowsSwatches verifies that swatch blocks (█) appear.
func TestThemeOverlay_View_ShowsSwatches(t *testing.T) {
	overlay := newTestThemeOverlay()
	overlay.width = 60
	overlay.height = 20

	view := overlay.View()
	assert.Contains(t, view, "█", "view should contain swatch blocks")
}

// TestThemeOverlay_CursorClamped verifies j does not go past the last theme.
func TestThemeOverlay_CursorClamped(t *testing.T) {
	overlay := newTestThemeOverlay()
	// Move cursor to the last item.
	overlay.cursor = len(overlay.themes) - 1

	// Press j again — should stay at last item.
	m, _ := overlay.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated, ok := m.(*ThemeOverlay)
	require.True(t, ok)
	assert.Equal(t, len(overlay.themes)-1, updated.cursor, "cursor should not exceed last item")
}

// TestThemeOverlay_CursorClampedAtTop verifies k does not go above 0.
func TestThemeOverlay_CursorClampedAtTop(t *testing.T) {
	overlay := newTestThemeOverlay()
	overlay.cursor = 0

	m, _ := overlay.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	updated, ok := m.(*ThemeOverlay)
	require.True(t, ok)
	assert.Equal(t, 0, updated.cursor, "cursor should not go below 0")
}

// TestThemeOverlay_SetTheme verifies that SetTheme updates the overlay's own theme.
func TestThemeOverlay_SetTheme(t *testing.T) {
	overlay := newTestThemeOverlay()
	th2 := theme.Load("dracula")
	overlay.SetTheme(th2)
	assert.Equal(t, th2, overlay.theme)
}

// TestNewThemeOverlay_CursorAtCurrentTheme verifies that the overlay starts with
// the cursor positioned on the currently active theme.
func TestNewThemeOverlay_CursorAtCurrentTheme(t *testing.T) {
	themes := theme.AllThemes()
	require.NotEmpty(t, themes)

	// Pick a theme that isn't the first in sorted order.
	var targetID string
	for _, th := range themes {
		if th.ID() != themes[0].ID() {
			targetID = th.ID()
			break
		}
	}
	if targetID == "" {
		t.Skip("only one theme loaded, skip cursor positioning test")
	}

	overlay := NewThemeOverlay(themes, targetID, theme.Load(targetID))
	assert.Equal(t, targetID, overlay.themes[overlay.cursor].ID(),
		"cursor should point at the current theme")
}

// TestThemeOverlay_View_DownArrow verifies the down arrow key also moves the cursor.
func TestThemeOverlay_DownArrow(t *testing.T) {
	overlay := newTestThemeOverlay()
	initial := overlay.cursor

	m, _ := overlay.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, ok := m.(*ThemeOverlay)
	require.True(t, ok)

	if len(overlay.themes) > 1 {
		assert.Equal(t, initial+1, updated.cursor, "down arrow should move cursor down")
	}
}

// TestThemeOverlay_View_UpArrow verifies the up arrow key also moves the cursor.
func TestThemeOverlay_UpArrow(t *testing.T) {
	overlay := newTestThemeOverlay()
	// Set cursor to non-zero position first.
	if len(overlay.themes) > 1 {
		overlay.cursor = 1
	}

	m, _ := overlay.Update(tea.KeyMsg{Type: tea.KeyUp})
	updated, ok := m.(*ThemeOverlay)
	require.True(t, ok)

	if len(overlay.themes) > 1 {
		assert.Equal(t, 0, updated.cursor, "up arrow should move cursor up")
	}
}

// TestRenderSwatches_Contains5Blocks verifies that renderSwatches produces 5 block chars.
func TestRenderSwatches_Contains5Blocks(t *testing.T) {
	th := theme.Load("black")
	ct, ok := th.(*theme.ConfigTheme)
	require.True(t, ok, "Load should return *ConfigTheme")

	result := renderSwatches(ct)
	count := strings.Count(result, "█")
	assert.Equal(t, 5, count, "renderSwatches should produce exactly 5 block chars")
}
