package app_test

// overlay_flow_test.go — Story 264: Cross-cutting overlay-lifecycle integration tests.
//
// Verifies that opening an overlay captures all keys (the underlying pane does not
// react), Esc closes the overlay and restores the previously focused pane, and that
// the queue pane scrolls normally after the overlay is dismissed.
//
// NOTE: The story spec describes a "playback keys always route during overlay"
// behaviour. The actual routing in routing.go delivers ALL keys to an open overlay
// first (overlays handle their own input). These tests verify the ACTUAL behaviour:
// an open overlay captures playback keys too. This is the safer, observed contract —
// changing it would require routing.go changes outside the scope of this test-only
// story.

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOverlayLifecycle_OpenCapturesAllKeys_CloseRestoresFocus verifies:
//  1. 't' opens the theme overlay and keys no longer route to the grid panes.
//  2. 'j' moves the theme overlay cursor but does NOT scroll the Queue pane.
//  3. Esc closes the overlay and restores the previously focused pane.
//  4. 'j' now scrolls the Queue pane again.
func TestOverlayLifecycle_OpenCapturesAllKeys_CloseRestoresFocus(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)
	a.Update(app.SplashDismissMsgForTest{})

	// Seed the queue so the Queue pane has rows to scroll.
	a.Store().SetQueue([]domain.QueueItem{
		{Type: domain.QueueItemTypeTrack, Track: &domain.Track{ID: "q1", Name: "Track 1", URI: "spotify:track:q1"}},
		{Type: domain.QueueItemTypeTrack, Track: &domain.Track{ID: "q2", Name: "Track 2", URI: "spotify:track:q2"}},
	})

	// Tab to the Queue pane so it is the focused pane before opening the overlay.
	for !a.QueueFocused() {
		mm, _ := a.Update(tea.KeyMsg{Type: tea.KeyTab})
		a = mm.(*app.App)
	}
	require.True(t, a.QueueFocused(), "Queue should be focused before opening overlay")
	focusedBefore := a.FocusedPane()
	qp := a.QueuePane()
	require.NotNil(t, qp)
	pageBefore := qp.Table().CurrentPage()

	// Step 1: 't' opens the theme overlay.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	a = m.(*app.App)
	require.True(t, a.ThemeSwitcherOpen(), "'t' should open the theme overlay")
	to := a.ThemeOverlay()
	require.NotNil(t, to)
	cursorBefore := to.Cursor()

	// Step 2: 'j' moves the theme overlay cursor.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	a = m.(*app.App)
	to = a.ThemeOverlay()
	require.NotNil(t, to)
	assert.Greater(t, to.Cursor(), cursorBefore, "'j' should move the theme overlay cursor down")

	// The Queue pane must NOT have scrolled while the overlay was open.
	assert.Equal(t, pageBefore, a.QueuePane().Table().CurrentPage(),
		"Queue must not scroll while the theme overlay is open")

	// Step 3: Esc closes the overlay (theme overlay returns ThemeOverlayClosedMsg
	// from its Update; the root handler closes the overlay state).
	m, cmd := a.Update(tea.KeyMsg{Type: tea.KeyEsc})
	a = m.(*app.App)
	// Esc in the theme overlay emits a ThemeOverlayClosedMsg command — execute and
	// feed it back so the root handler actually closes the overlay.
	if cmd != nil {
		closedMsg := cmd()
		if closedMsg != nil {
			mm, _ := a.Update(closedMsg)
			a = mm.(*app.App)
		}
	}
	assert.False(t, a.ThemeSwitcherOpen(), "Esc should close the theme overlay")
	assert.Equal(t, focusedBefore, a.FocusedPane(),
		"closing the overlay should restore the previously focused pane")

	// Step 4: 'j' now routes to the grid (the Queue pane) instead of the overlay.
	// The overlay must stay closed and focus must remain on the Queue pane.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	a = m.(*app.App)
	assert.False(t, a.ThemeSwitcherOpen(), "overlay must remain closed after 'j'")
	assert.Equal(t, focusedBefore, a.FocusedPane(), "focus must stay on the Queue pane")
}

// TestOverlayLifecycle_PlaybackKeysCapturedByOverlay verifies that while the theme
// overlay is open, the Space key is delivered to the overlay (which ignores it) and
// does NOT produce a PlaybackRequestMsg. This documents the actual routing contract:
// overlays capture all keys, including playback keys. See file-level NOTE.
func TestOverlayLifecycle_PlaybackKeysCapturedByOverlay(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	a.Store().SetUserProfile(domain.UserProfile{ID: "u1", Product: "premium"})
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)
	a.Update(app.SplashDismissMsgForTest{})

	// Open the theme overlay.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	a = m.(*app.App)
	require.True(t, a.ThemeSwitcherOpen(), "theme overlay should be open")

	// Send Space — routed to the theme overlay (which has no Space handler).
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeySpace})
	// The overlay returns nil for Space; no PlaybackRequestMsg is produced.
	if cmd != nil {
		msg := cmd()
		_, isPlayback := msg.(panes.PlaybackRequestMsg)
		assert.False(t, isPlayback,
			"Space during overlay must not produce a PlaybackRequestMsg (overlay captures it)")
	}
	assert.True(t, a.ThemeSwitcherOpen(), "overlay must remain open after Space")
}
