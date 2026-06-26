package app_test

// page_flow_test.go — Story 264: Cross-cutting page/preset/pane-toggle integration tests.
//
// Verifies:
//   - '0' toggles between Player and Stats pages and back.
//   - 'p' cycles through all 6 Player presets.
//   - Toggle keys ('2') hide/show a pane and the grid reflows.

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPageToggle_CyclesPlayerToStats verifies that '0' switches from the Player page
// to the Stats page, and a second '0' returns to the Player page. The status bar
// reflects the active page ("Stats" vs "Player").
func TestPageToggle_CyclesPlayerToStats(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)
	a.Update(app.SplashDismissMsgForTest{})

	require.Equal(t, layout.PagePlayer, a.ActivePage(), "app starts on the Player page")
	view := a.View()
	assert.Contains(t, view, "Player", "status bar should show Player page")

	// '0' → Stats page.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
	a = m.(*app.App)
	assert.Equal(t, layout.PageStats, a.ActivePage(), "'0' should switch to the Stats page")
	assert.Contains(t, a.View(), "Stats", "status bar should show Stats page")

	// '0' again → back to Player page.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
	a = m.(*app.App)
	assert.Equal(t, layout.PagePlayer, a.ActivePage(), "second '0' should return to Player page")
	assert.Contains(t, a.View(), "Player", "status bar should show Player page again")
}

// TestPresetCycle_PKeyAdvancesThroughPlayerPresets verifies that pressing 'p' cycles
// through all 6 Player presets in order, wrapping back to the first after the last.
func TestPresetCycle_PKeyAdvancesThroughPlayerPresets(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)
	a.Update(app.SplashDismissMsgForTest{})

	expected := []string{
		layout.PresetNameDashboard,
		layout.PresetNameListening,
		layout.PresetNamePodcast,
		layout.PresetNameLibrary,
		layout.PresetNameDiscovery,
		layout.PresetNamePodcastDashboard,
	}
	require.Equal(t, expected[0], a.ActivePresetName(), "should start on the Dashboard preset")

	// Cycle through all 6 presets. After 6 'p' presses we wrap back to the first.
	for i := 0; i < len(expected); i++ {
		want := expected[(i+1)%len(expected)]
		m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
		a = m.(*app.App)
		assert.Equal(t, want, a.ActivePresetName(),
			"'p' press %d should advance to %s", i+1, want)
	}
	// After a full cycle we are back on the first preset.
	assert.Equal(t, expected[0], a.ActivePresetName(),
		"after cycling all presets, focus should wrap to the first")
}

// TestPaneToggle_ToggleKeyHidesShowsPane verifies that the '2' toggle key hides the
// Queue pane (grid reflows) and a second '2' shows it again.
func TestPaneToggle_ToggleKeyHidesShowsPane(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)
	a.Update(app.SplashDismissMsgForTest{})

	// Dashboard preset exposes the Queue pane (toggle key '2').
	require.True(t, a.Layout().IsPaneVisible(layout.PaneQueue),
		"Queue pane should be visible in the Dashboard preset")

	// '2' hides the Queue pane.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	a = m.(*app.App)
	assert.False(t, a.Layout().IsPaneVisible(layout.PaneQueue),
		"'2' should hide the Queue pane")

	// '2' again shows the Queue pane.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	a = m.(*app.App)
	assert.True(t, a.Layout().IsPaneVisible(layout.PaneQueue),
		"second '2' should show the Queue pane again")
}
