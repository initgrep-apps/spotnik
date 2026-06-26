package app_test

// navigation_flow_test.go — Story 264: Cross-cutting focus-rotation integration tests.
//
// Verifies that Tab/Shift+Tab cycles through visible panes and that keys route only
// to the focused pane (unfocused panes do not react).

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newNavTestApp creates an App with the grid visible (splash dismissed) ready for
// focus-rotation tests. The Dashboard preset is the default and exposes 3 visible
// panes (NowPlaying, Playlists, Queue).
func newNavTestApp(t *testing.T) *app.App {
	t.Helper()
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	// Initialize layout so focus rotation works.
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)
	// Dismiss the splash screen so the grid view is active.
	a.Update(app.SplashDismissMsgForTest{})
	return a
}

// visiblePaneCount returns the number of panes visible in the current preset.
func visiblePaneCount(a *app.App) int {
	count := 0
	for _, id := range []layout.PaneID{
		layout.PaneNowPlaying, layout.PaneQueue, layout.PanePlaylists,
		layout.PaneAlbums, layout.PaneLikedSongs, layout.PaneRecentlyPlayed,
		layout.PaneTopTracks, layout.PaneTopArtists,
		layout.PaneFollowedShows, layout.PaneSavedEpisodes,
	} {
		if a.Layout().IsPaneVisible(id) {
			count++
		}
	}
	return count
}

// TestFocusRotation_TabCyclesThroughVisiblePanes verifies that Tab advances focus
// forward through every visible pane and wraps back to the first pane after the
// last. Shift+Tab moves focus backward.
func TestFocusRotation_TabCyclesThroughVisiblePanes(t *testing.T) {
	a := newNavTestApp(t)

	// Dashboard preset starts with NowPlaying focused.
	require.True(t, a.NowPlayingFocused(), "NowPlaying should be focused initially")
	start := a.FocusedPane()

	visible := visiblePaneCount(a)
	require.Greater(t, visible, 1, "preset must expose more than one pane for rotation")

	// Tab through every visible pane — after N tabs we must return to the start.
	for i := 0; i < visible; i++ {
		m, _ := a.Update(tea.KeyMsg{Type: tea.KeyTab})
		a = m.(*app.App)
	}
	assert.Equal(t, start, a.FocusedPane(),
		"Tab cycling through all %d visible panes should wrap to the starting pane", visible)

	// Shift+Tab moves focus backward (to the last pane in focus order).
	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	a = m.(*app.App)
	assert.NotEqual(t, start, a.FocusedPane(),
		"Shift+Tab should move focus off the start pane")

	// One more Shift+Tab should wrap forward back to the start (full backward cycle
	// over `visible` panes returns to start).
	for i := 1; i < visible; i++ {
		m, _ = a.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
		a = m.(*app.App)
	}
	assert.Equal(t, start, a.FocusedPane(),
		"Shift+Tab cycling through all visible panes should wrap to the starting pane")
}

// TestFocusRotation_KeysRoutedToFocusedPaneOnly verifies that a key is delivered to
// the focused pane and NOT to an unfocused one. We use the Queue pane's filter key
// 'f' as the probe: 'f' on the focused Queue pane activates the filter; 'f' while
// NowPlaying is focused must not activate the Queue filter.
func TestFocusRotation_KeysRoutedToFocusedPaneOnly(t *testing.T) {
	a := newNavTestApp(t)
	// Seed the store with queue data so the Queue pane can accept filter input.
	a.Store().SetQueue([]domain.QueueItem{
		{Type: domain.QueueItemTypeTrack, Track: &domain.Track{ID: "q1", Name: "Track A", URI: "spotify:track:q1"}},
	})

	// NowPlaying is focused first. Send 'f' — NowPlaying has no filter, so the
	// key routes to NowPlaying (a no-op) and the Queue filter must remain inactive.
	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	a = m.(*app.App)
	assert.True(t, a.NowPlayingFocused(), "NowPlaying remains focused after 'f'")
	qp := a.QueuePane()
	require.NotNil(t, qp)
	assert.Equal(t, "", qp.ActiveFilterQuery(),
		"'f' routed to NowPlaying must not activate the Queue filter")

	// Now Tab to the Queue pane and press 'f' — the filter should activate.
	for !a.QueueFocused() {
		mm, _ := a.Update(tea.KeyMsg{Type: tea.KeyTab})
		a = mm.(*app.App)
	}
	require.True(t, a.QueueFocused(), "Tab should eventually focus the Queue pane")

	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	a = m.(*app.App)
	// Filter is now active but query is empty — type a rune to populate it.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	a = m.(*app.App)
	assert.Equal(t, "r", a.QueuePane().ActiveFilterQuery(),
		"'f' then 'r' on the focused Queue pane should activate the filter with query 'r'")
}
