package panes

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/muesli/termenv"
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

// ── Story 77 Task 1: Theme overlay non-cursor row — no explicit background ────

// TestThemeOverlay_NonCursorRow_NoExplicitBackground verifies that non-cursor rows
// in the theme overlay have NO explicit background color set (no "48;2;" ANSI
// background sequence). Without an explicit background the row blends with the
// dimmed overlay background produced by btoverlay.Composite(), rather than
// showing an opaque colored rectangle.
func TestThemeOverlay_NonCursorRow_NoExplicitBackground(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	themes := theme.AllThemes()
	require.GreaterOrEqual(t, len(themes), 2, "need at least 2 themes to test non-cursor row")

	th := theme.Load("black")
	overlay := NewThemeOverlay(themes, "black", th)
	// Set cursor to 0 so row 1 is a non-cursor row.
	overlay.cursor = 0

	// renderRow with idx=1 is a non-cursor row.
	row := overlay.renderRow(1, themes[1], 40)

	// "48;2;" is the ANSI SGR introducer for 24-bit RGB background color.
	// Non-cursor rows must produce no background escape at all.
	assert.NotContains(t, row, "48;2;",
		"non-cursor row should have NO explicit background (no 48;2; ANSI sequence)")
}

// TestThemeOverlay_CursorRow_UsesSelectedBg verifies that the cursor row uses
// SelectedBg() so it clearly stands out from non-cursor rows, and that the bg
// appears in the same SGR run that opens immediately before the theme label text
// (not merely somewhere in the row). This ensures the cursor highlight is
// visually continuous rather than a discontiguous rectangle.
func TestThemeOverlay_CursorRow_UsesSelectedBg(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	themes := theme.AllThemes()
	require.NotEmpty(t, themes)

	th := theme.Load("black")
	overlay := NewThemeOverlay(themes, "black", th)
	overlay.cursor = 0

	// renderRow with idx=0 is the cursor row.
	row := overlay.renderRow(0, themes[0], 40)

	// The cursor row must include the SelectedBg() color as a background.
	selectedBgStyle := lipgloss.NewStyle().Background(th.SelectedBg()).Render("X")
	selectedBg := extractBackgroundANSI(selectedBgStyle)
	require.NotEmpty(t, selectedBg, "sanity: SelectedBg must produce 48;2; in TrueColor")
	assert.Contains(t, row, selectedBg,
		"cursor row should use SelectedBg() background")

	// Stronger assertion: the bg MUST appear in the same SGR run immediately
	// before the theme label text — not just somewhere in the row. This verifies
	// the highlight is continuous across the label, not just on trailing padding.
	themeName := themes[0].Name()
	labelPos := strings.Index(row, themeName)
	require.GreaterOrEqual(t, labelPos, 0, "theme name %q not found in row output", themeName)

	preLabel := row[:labelPos]
	lastEsc := strings.LastIndex(preLabel, "\x1b[")
	require.GreaterOrEqual(t, lastEsc, 0, "no ANSI escape found before label in cursor row")

	mPos := strings.Index(row[lastEsc:], "m")
	require.GreaterOrEqual(t, mPos, 0, "malformed ANSI escape before label in cursor row")

	sgrBeforeLabel := row[lastEsc : lastEsc+mPos+1]
	assert.Contains(t, sgrBeforeLabel, selectedBg,
		"SelectedBg must appear in the SGR run immediately before theme label %q; got: %q\nfull row: %q",
		themeName, sgrBeforeLabel, row)
}

// extractBackgroundANSI extracts the "48;2;R;G;B" portion of an ANSI string if present.
// Returns empty string if no background sequence is found.
func extractBackgroundANSI(s string) string {
	const bgPrefix = "48;2;"
	idx := strings.Index(s, bgPrefix)
	if idx < 0 {
		return ""
	}
	// Find end of sequence (terminated by 'm').
	end := strings.Index(s[idx:], "m")
	if end < 0 {
		return ""
	}
	return s[idx : idx+end]
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
