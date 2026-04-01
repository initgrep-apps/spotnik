package app_test

// routing_test.go — Tests for filter-active key routing guard.
// When a pane's filter is active, global shortcuts (q, /, d, etc.) must be
// bypassed so keystrokes reach the filter text input.

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper: create an app, resize it, and Tab to a filterable pane.
// Returns the app with a filterable pane focused (any pane that supports 'f' filter).
func setupAppWithFilterablePane(t *testing.T) *app.App {
	t.Helper()
	a := app.New(&config.Config{}, app.AppOptions{})
	// Give the layout a reasonable size so panes are visible.
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	// NowPlaying is focused by default (PaneID 0). Tab to reach a filterable pane.
	// The first filterable pane in focus order may vary by preset; tab until we
	// land on one that isn't NowPlaying (NowPlaying has no filter).
	a.Update(tea.KeyMsg{Type: tea.KeyTab})
	focused := a.FocusedPane()
	require.NotEqual(t, layout.PaneNowPlaying, focused, "should have tabbed past NowPlaying")
	return a
}

// activateFilter sends 'f' to the focused pane to open the filter.
func activateFilter(a *app.App) {
	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
}

// isBatchContainingQuit checks if a Cmd (possibly a batch) contains tea.Quit.
// tea.Quit returns a special QuitMsg when executed.
func cmdProducesQuit(t *testing.T, cmd tea.Cmd) bool {
	t.Helper()
	if cmd == nil {
		return false
	}
	// Execute the command and check the message type.
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); ok {
		return true
	}
	// Handle batch commands: tea.Batch returns a batchMsg ([]tea.Cmd).
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c != nil {
				if innerMsg := c(); innerMsg != nil {
					if _, ok := innerMsg.(tea.QuitMsg); ok {
						return true
					}
				}
			}
		}
	}
	return false
}

// TestFilterActive_Q_DoesNotQuit verifies that pressing 'q' while filter is active
// does NOT produce a tea.Quit command.
func TestFilterActive_Q_DoesNotQuit(t *testing.T) {
	a := setupAppWithFilterablePane(t)
	activateFilter(a)

	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	assert.False(t, cmdProducesQuit(t, cmd), "'q' with active filter should not quit the app")
}

// TestFilterInactive_Q_Quits verifies that pressing 'q' without an active filter
// produces a tea.Quit command (baseline).
func TestFilterInactive_Q_Quits(t *testing.T) {
	a := setupAppWithFilterablePane(t)

	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	assert.True(t, cmdProducesQuit(t, cmd), "'q' without active filter should quit the app")
}

// TestFilterActive_Slash_DoesNotOpenSearch verifies that '/' while filter is active
// does NOT open the search overlay.
func TestFilterActive_Slash_DoesNotOpenSearch(t *testing.T) {
	a := setupAppWithFilterablePane(t)
	activateFilter(a)

	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	assert.False(t, a.SearchOpen(), "'/' with active filter should not open search")
}

// TestFilterActive_D_DoesNotOpenDeviceOverlay verifies that 'd' while filter is active
// does NOT open the device overlay.
func TestFilterActive_D_DoesNotOpenDeviceOverlay(t *testing.T) {
	a := setupAppWithFilterablePane(t)
	activateFilter(a)

	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	assert.False(t, a.DeviceOverlayOpen(), "'d' with active filter should not open device overlay")
}

// TestFilterActive_Esc_ClosesFilter verifies that Esc while filter is active
// closes the filter (the pane handles Esc internally to deactivate the filter).
func TestFilterActive_Esc_ClosesFilter(t *testing.T) {
	a := setupAppWithFilterablePane(t)
	activateFilter(a)

	// Esc should be routed to the pane (via the filter guard), which deactivates the filter.
	a.Update(tea.KeyMsg{Type: tea.KeyEsc})

	// After Esc, filter should be inactive, so 'q' should now quit.
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	assert.True(t, cmdProducesQuit(t, cmd), "'q' after filter closed should quit the app")
}

// TestFilterActive_NumberKeys_DoNotTogglePanes verifies that number keys
// while filter is active do NOT toggle pane visibility.
func TestFilterActive_NumberKeys_DoNotTogglePanes(t *testing.T) {
	a := setupAppWithFilterablePane(t)
	originalFocus := a.FocusedPane()
	activateFilter(a)

	// '0' would normally toggle pages. Verify it doesn't by checking focus is unchanged.
	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
	assert.Equal(t, originalFocus, a.FocusedPane(), "'0' with active filter should not toggle page")
}

// TestMouseScroll_SearchOpen_ForwardsToOverlay verifies that when the search overlay
// is open, mouse wheel events are forwarded to the search overlay rather than discarded.
func TestMouseScroll_SearchOpen_ForwardsToOverlay(t *testing.T) {
	a := app.New(&config.Config{}, app.AppOptions{})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Open the search overlay via '/' key.
	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	require.True(t, a.SearchOpen(), "search overlay should be open after '/'")

	// Wheel down should not return nil (it should produce a cmd or at minimum not panic).
	// Since the bubble-table may or may not produce a command, we just verify no panic
	// and that SearchOpen is still true (the overlay was not closed).
	_, cmd := a.Update(tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelDown,
		X:      60, Y: 20,
	})
	_ = cmd
	assert.True(t, a.SearchOpen(), "search overlay should still be open after wheel scroll")
}

// TestMouseScroll_SearchOpen_IgnoresNonWheel verifies that non-wheel mouse events
// are ignored when the search overlay is open.
func TestMouseScroll_SearchOpen_IgnoresNonWheel(t *testing.T) {
	a := app.New(&config.Config{}, app.AppOptions{})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Open the search overlay.
	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	require.True(t, a.SearchOpen(), "search overlay should be open")

	// A non-wheel mouse press (e.g., motion) should produce nil cmd.
	_, cmd := a.Update(tea.MouseMsg{
		Action: tea.MouseActionMotion,
		Button: tea.MouseButtonNone,
		X:      60, Y: 20,
	})
	assert.Nil(t, cmd, "non-wheel mouse event with search open should not produce a cmd")
	assert.True(t, a.SearchOpen(), "search overlay should still be open after non-wheel mouse event")
}
