package app_test

// routing_test.go — Tests for filter-active key routing guard.
// When a pane's filter is active, global shortcuts (q, /, d, etc.) must be
// bypassed so keystrokes reach the filter text input.

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
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

// TestFilterActive_Tab_DoesNotRotateFocus verifies that Tab while filter is
// active does NOT rotate focus to the next pane. Tab is consumed by the
// textinput inside the filter.
//
// Story 181: this pins the focus invariant — no code path moves focus away
// from a pane with an active filter input.
func TestFilterActive_Tab_DoesNotRotateFocus(t *testing.T) {
	a := setupAppWithFilterablePane(t)
	originalFocus := a.FocusedPane()
	activateFilter(a)

	a.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, originalFocus, a.FocusedPane(),
		"Tab with active filter must not rotate focus")
}

// TestFilterActive_P_DoesNotCyclePreset verifies that 'p' while filter is
// active does NOT cycle the preset. The keystroke is consumed by the filter
// input as a literal character.
//
// Story 181: pins the focus invariant for preset cycling.
func TestFilterActive_P_DoesNotCyclePreset(t *testing.T) {
	a := setupAppWithFilterablePane(t)
	originalFocus := a.FocusedPane()
	activateFilter(a)

	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	assert.Equal(t, originalFocus, a.FocusedPane(),
		"'p' with active filter must not cycle preset")
}

// --- Profile overlay routing tests ---

// newProfileTestApp creates a minimal App for profile overlay routing tests.
func newProfileTestApp(t *testing.T) *app.App {
	t.Helper()
	a := app.New(&config.Config{}, app.AppOptions{})
	// Resize so the app is in grid view (splash needs a size to dismiss).
	a.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	return a
}

// TestProfileOverlay_UKey_OpensOverlay verifies that pressing 'u' opens the profile overlay.
func TestProfileOverlay_UKey_OpensOverlay(t *testing.T) {
	a := newProfileTestApp(t)

	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	assert.True(t, a.ProfileOverlayOpen(), "'u' should open the profile overlay")
}

// TestProfileOverlay_EscKey_ClosesOverlay verifies that pressing Esc closes the profile overlay.
// In Bubble Tea, Esc routes to the pane which returns a ProfileOverlayClosedMsg command.
// The test simulates the runtime loop by executing the command and feeding the message back.
func TestProfileOverlay_EscKey_ClosesOverlay(t *testing.T) {
	a := newProfileTestApp(t)

	// Open the overlay.
	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	require.True(t, a.ProfileOverlayOpen(), "profile overlay should be open after 'u'")

	// Esc returns a command that emits ProfileOverlayClosedMsg.
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd, "Esc should return a close command")

	// Execute the command and deliver the resulting message.
	msg := cmd()
	a.Update(msg)
	assert.False(t, a.ProfileOverlayOpen(), "ProfileOverlayClosedMsg should close the profile overlay")
}

// TestProfileOverlay_KeysIntercepted_WhenOpen verifies that other keys (e.g. 'q') do not
// pass through to global handlers while the profile overlay is open.
func TestProfileOverlay_KeysIntercepted_WhenOpen(t *testing.T) {
	a := newProfileTestApp(t)

	// Open the overlay.
	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	require.True(t, a.ProfileOverlayOpen(), "profile overlay should be open")

	// 'q' normally quits but must be captured by the overlay, not global handler.
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	assert.False(t, cmdProducesQuit(t, cmd), "'q' must not quit while profile overlay is open")
}

// TestFilterActive_U_DoesNotOpenProfileOverlay verifies that 'u' while filter is
// active does NOT open the profile overlay.
func TestFilterActive_U_DoesNotOpenProfileOverlay(t *testing.T) {
	a := setupAppWithFilterablePane(t)
	activateFilter(a)

	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	assert.False(t, a.ProfileOverlayOpen(), "'u' with active filter should not open profile overlay")
}

// --- Profile self-sufficiency routing tests (Story 202) ---

// TestApp_FetchCurrentUserRequestMsg_Dispatches verifies that when the app receives
// a FetchCurrentUserRequestMsg, it dispatches buildFetchCurrentUserCmd.
func TestApp_FetchCurrentUserRequestMsg_Dispatches(t *testing.T) {
	a := newProfileTestApp(t)

	// Send the FetchCurrentUserRequestMsg that ProfileOverlay.Init() would emit.
	_, cmd := a.Update(panes.FetchCurrentUserRequestMsg{})

	// The command must be non-nil — it should produce a userProfileLoadedMsg.
	require.NotNil(t, cmd, "FetchCurrentUserRequestMsg should dispatch buildFetchCurrentUserCmd")
	msg := cmd()
	// The command must produce a non-nil message (even if it's an error like errNilClient).
	assert.NotNil(t, msg, "command result must be a non-nil message")
}

// TestApp_UserProfileLoaded_ForwardsToOverlayWhenOpen verifies that when the profile
// overlay is open and a userProfileLoadedMsg arrives with an error, the error is
// forwarded to the overlay so it can display the error state.
func TestApp_UserProfileLoaded_ForwardsToOverlayWhenOpen(t *testing.T) {
	a := newProfileTestApp(t)

	// Open the profile overlay.
	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	require.True(t, a.ProfileOverlayOpen(), "profile overlay should be open")

	// Simulate a failed user profile load forwarded to the overlay.
	a.InjectUserProfileLoadedErr(fmt.Errorf("network error"))

	// Verify the overlay received the error.
	assert.NotNil(t, a.ProfilePaneErr(), "overlay should have a non-nil error after failed load")
}

// TestApp_UserProfileLoaded_ClearsErrorOnSuccess verifies that when a UserProfileLoadedMsg
// with nil error arrives, any previous error on the overlay is cleared.
func TestApp_UserProfileLoaded_ClearsErrorOnSuccess(t *testing.T) {
	a := newProfileTestApp(t)

	// Open the profile overlay.
	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	require.True(t, a.ProfileOverlayOpen(), "profile overlay should be open")

	// First: inject an error.
	a.InjectUserProfileLoadedErr(fmt.Errorf("network error"))
	assert.NotNil(t, a.ProfilePaneErr(), "overlay should have error after failed load")

	// Now: inject success (nil error).
	a.InjectUserProfileLoadedErr(nil)
	assert.Nil(t, a.ProfilePaneErr(), "overlay error should be cleared after successful load")
}

// --- Playback key routing tests ---

// newPremiumPlaybackTestApp creates an App pre-configured as a Premium user with a reasonable
// window size so playback keys pass the premium gate and reach NowPlayingPane.
func newPremiumPlaybackTestApp(t *testing.T) *app.App {
	t.Helper()
	a := app.New(&config.Config{}, app.AppOptions{})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	// Set premium profile so playback keys are not blocked by the subscription gate.
	a.Store().SetUserProfile(domain.UserProfile{ID: "u1", Product: "premium"})
	return a
}

// cmdProducesPlaybackRequestMsg returns true if cmd (possibly a batch) produces a
// panes.PlaybackRequestMsg. Used to verify that a key is routed to NowPlayingPane.
func cmdProducesPlaybackRequestMsg(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	msg := cmd()
	if _, ok := msg.(panes.PlaybackRequestMsg); ok {
		return true
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c == nil {
				continue
			}
			if _, ok := c().(panes.PlaybackRequestMsg); ok {
				return true
			}
		}
	}
	return false
}

// TestIsPlaybackKey_Space verifies that tea.KeySpace is treated as a playback key
// and routes to NowPlayingPane, producing a PlaybackRequestMsg.
func TestIsPlaybackKey_Space(t *testing.T) {
	a := newPremiumPlaybackTestApp(t)

	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeySpace})
	assert.True(t, cmdProducesPlaybackRequestMsg(cmd),
		"tea.KeySpace must route to NowPlayingPane and return PlaybackRequestMsg")
}

// TestIsPlaybackKey_N_NotPlaybackKey verifies that the "n" rune is no longer treated as a
// global playback key. Even for a Premium user, pressing "n" when NowPlayingPane is not
// focused must NOT produce a PlaybackRequestMsg — it must fall through to the focused pane.
func TestIsPlaybackKey_N_NotPlaybackKey(t *testing.T) {
	a := newPremiumPlaybackTestApp(t)
	// Tab away from NowPlayingPane so the focused pane is not NowPlaying.
	a.Update(tea.KeyMsg{Type: tea.KeyTab})
	require.NotEqual(t, layout.PaneNowPlaying, a.FocusedPane(), "should have tabbed past NowPlaying")

	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	assert.False(t, cmdProducesPlaybackRequestMsg(cmd),
		"'n' must not produce PlaybackRequestMsg — it must fall through to the focused pane")
}

// TestStatsPageNumberKeys_ToggleStatsPanes verifies that keys '2'-'5' on Stats page toggle
// the correct Stats page pane, and that a transpose in statsToggleKeyMap would be caught.
func TestStatsPageNumberKeys_ToggleStatsPanes(t *testing.T) {
	cases := []struct {
		key    rune
		paneID layout.PaneID
		name   string
	}{
		{'2', layout.PaneGatewayHealth, "GatewayHealth"},
		{'3', layout.PanePollingTraffic, "PollingTraffic"},
		{'4', layout.PaneGatewayLive, "GatewayLive"},
		{'5', layout.PaneNetworkLog, "NetworkLog"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := app.New(&config.Config{}, app.AppOptions{})
			a.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
			a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("0")}) // Player → Stats
			require.True(t, a.Layout().IsPaneVisible(tc.paneID),
				"%s must be visible on Stats page before toggle", tc.name)
			a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{tc.key}})
			assert.False(t, a.Layout().IsPaneVisible(tc.paneID),
				"%s must be hidden after pressing '%c' on Stats page", tc.name, tc.key)
		})
	}
}
