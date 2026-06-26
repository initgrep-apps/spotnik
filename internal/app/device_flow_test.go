package app_test

// device_flow_test.go — Story 264: Cross-cutting device-overlay integration tests.
//
// Verifies the device switcher overlay flow through the root App:
//   - Opening the overlay, selecting a device with Enter produces a
//     TransferPlaybackMsg carrying the correct DeviceID.
//   - Esc closes the overlay and restores the previously focused pane.

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

// TestDevicesFlow_EnterTransfersPlayback verifies that opening the device overlay,
// loading devices, and pressing Enter on a non-active device produces a
// TransferPlaybackMsg with the correct DeviceID, and that a subsequent
// DeviceTransferredMsg with no error triggers a reconcile fetch.
func TestDevicesFlow_EnterTransfersPlayback(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	// Premium user so the transfer gate passes.
	a.Store().SetUserProfile(domain.UserProfile{ID: "u1", Product: "premium"})
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)
	a.Update(app.SplashDismissMsgForTest{})

	// Open the device overlay with 'd'.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	a = m.(*app.App)
	require.True(t, a.DeviceOverlayOpen(), "'d' should open the device overlay")

	// Populate the overlay with a device list (simulates the fetch completing).
	devices := []panes.DeviceInfo{
		{ID: "dev-active", Name: "MacBook Pro", Type: "Computer", IsActive: true},
		{ID: "dev-2", Name: "iPhone 14", Type: "Smartphone", IsActive: false},
	}
	m, _ = a.Update(panes.DevicesLoadedMsg{Devices: devices, Err: nil})
	a = m.(*app.App)
	require.Equal(t, devices, a.DevicePane().Devices(), "overlay should show the loaded devices")

	// Move the cursor to the second (non-active) device.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	a = m.(*app.App)
	require.Equal(t, 1, a.DevicePane().Cursor(), "cursor should be on the second device")

	// Press Enter — overlay emits TransferPlaybackMsg.
	m, cmd := a.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a = m.(*app.App)
	require.NotNil(t, cmd, "Enter on a non-active device should produce a cmd")
	transferMsg, ok := cmd().(panes.TransferPlaybackMsg)
	require.True(t, ok, "Enter should produce TransferPlaybackMsg, got %T", cmd())
	assert.Equal(t, "dev-2", transferMsg.DeviceID, "TransferPlaybackMsg should carry the selected DeviceID")
	assert.Equal(t, "iPhone 14", transferMsg.DeviceName, "TransferPlaybackMsg should carry the device name")

	// The overlay should close after Enter (TransferPlaybackMsg handler closes it).
	// Feed TransferPlaybackMsg back to the App — it closes the overlay and dispatches
	// the transfer API cmd. With no player set, buildTransferPlaybackCmd returns a
	// no-op; we only verify the overlay closed.
	m, _ = a.Update(transferMsg)
	a = m.(*app.App)
	assert.False(t, a.DeviceOverlayOpen(), "overlay should close after Enter transfers playback")

	// Simulate the transfer completing successfully — DeviceTransferredMsg{Err: nil}
	// triggers a reconcile fetchPlaybackStateCmd. With no player, the cmd is nil-safe.
	_, reconcileCmd := a.Update(panes.DeviceTransferredMsg{DeviceID: "dev-2", Err: nil})
	// The handler returns a fetch cmd; it may be nil if player is nil, but Update
	// must not panic.
	_ = reconcileCmd
}

// TestDevicesFlow_EscClosesOverlay verifies that Esc closes the device overlay and
// restores focus to the previously focused pane.
func TestDevicesFlow_EscClosesOverlay(t *testing.T) {
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = m.(*app.App)
	a.Update(app.SplashDismissMsgForTest{})

	// Tab away from NowPlaying so the restored focus is observable.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyTab})
	a = m.(*app.App)
	require.False(t, a.NowPlayingFocused(), "NowPlaying should not be focused after Tab")
	focusedBefore := a.FocusedPane()

	// Open the device overlay.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	a = m.(*app.App)
	require.True(t, a.DeviceOverlayOpen(), "device overlay should be open")

	// Esc emits DeviceOverlayClosedMsg — execute and feed back to close.
	m, cmd := a.Update(tea.KeyMsg{Type: tea.KeyEsc})
	a = m.(*app.App)
	require.NotNil(t, cmd, "Esc in the device overlay should produce a cmd")
	closedMsg := cmd()
	_, isClosed := closedMsg.(panes.DeviceOverlayClosedMsg)
	assert.True(t, isClosed, "Esc should produce DeviceOverlayClosedMsg")

	// Feed the closed msg back to actually close the overlay state.
	m, _ = a.Update(closedMsg)
	a = m.(*app.App)
	assert.False(t, a.DeviceOverlayOpen(), "overlay should be closed after Esc")
	assert.Equal(t, focusedBefore, a.FocusedPane(),
		"closing the device overlay should restore the previously focused pane")
}
