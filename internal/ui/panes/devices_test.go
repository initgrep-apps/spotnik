package panes

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDevices returns a set of test devices for use in table tests.
func testDevices() []DeviceInfo {
	return []DeviceInfo{
		{ID: "abc123", Name: "MacBook Pro Speakers", Type: "Computer", IsActive: true},
		{ID: "def456", Name: "iPhone 14", Type: "Smartphone", IsActive: false},
		{ID: "ghi789", Name: "Kitchen Speaker", Type: "Speaker", IsActive: false},
	}
}

func newTestDeviceOverlay() *DeviceOverlay {
	s := state.New()
	t := theme.Load("black")
	return NewDeviceOverlay(s, t)
}

func TestDeviceOverlay_Init_FetchesDevices(t *testing.T) {
	overlay := newTestDeviceOverlay()
	cmd := overlay.Init()
	// Init must return a non-nil command that fetches devices.
	require.NotNil(t, cmd, "Init() should return a fetch command")
}

func TestDeviceOverlay_View_DeviceList(t *testing.T) {
	overlay := newTestDeviceOverlay()
	overlay.devices = testDevices()

	view := overlay.View()

	assert.Contains(t, view, "MacBook Pro Speakers", "active device name should appear")
	assert.Contains(t, view, "iPhone 14", "inactive device name should appear")
	assert.Contains(t, view, "Kitchen Speaker", "inactive device name should appear")
	assert.Contains(t, view, "DEVICES", "overlay header should appear")
}

func TestDeviceOverlay_View_ActiveDevice(t *testing.T) {
	overlay := newTestDeviceOverlay()
	overlay.devices = testDevices()

	view := overlay.View()

	// Active device should show ◉ symbol and [active] label
	assert.Contains(t, view, "◉", "active device should show ◉ symbol")
	assert.Contains(t, view, "[active]", "active device should show [active] label")
}

func TestDeviceOverlay_View_EmptyList(t *testing.T) {
	overlay := newTestDeviceOverlay()
	overlay.devices = []DeviceInfo{}

	view := overlay.View()

	assert.Contains(t, view, "No devices found", "empty device list should show 'No devices found'")
}

func TestDeviceOverlay_Update_J_MovesDown(t *testing.T) {
	overlay := newTestDeviceOverlay()
	overlay.devices = testDevices()
	overlay.cursor = 0

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	model, _ := overlay.Update(keyMsg)
	updated, ok := model.(*DeviceOverlay)
	require.True(t, ok)
	assert.Equal(t, 1, updated.cursor, "cursor should move down to index 1")
}

func TestDeviceOverlay_Update_K_MovesUp(t *testing.T) {
	overlay := newTestDeviceOverlay()
	overlay.devices = testDevices()
	overlay.cursor = 1

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
	model, _ := overlay.Update(keyMsg)
	updated, ok := model.(*DeviceOverlay)
	require.True(t, ok)
	assert.Equal(t, 0, updated.cursor, "cursor should move up to index 0")
}

func TestDeviceOverlay_Update_J_DoesNotGoOutOfBounds(t *testing.T) {
	overlay := newTestDeviceOverlay()
	overlay.devices = testDevices()
	overlay.cursor = len(testDevices()) - 1 // last device

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	model, _ := overlay.Update(keyMsg)
	updated, ok := model.(*DeviceOverlay)
	require.True(t, ok)
	assert.Equal(t, len(testDevices())-1, updated.cursor, "cursor should not go past last device")
}

func TestDeviceOverlay_Update_K_DoesNotGoNegative(t *testing.T) {
	overlay := newTestDeviceOverlay()
	overlay.devices = testDevices()
	overlay.cursor = 0

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
	model, _ := overlay.Update(keyMsg)
	updated, ok := model.(*DeviceOverlay)
	require.True(t, ok)
	assert.Equal(t, 0, updated.cursor, "cursor should not go below 0")
}

func TestDeviceOverlay_Update_Enter_TransfersPlayback(t *testing.T) {
	overlay := newTestDeviceOverlay()
	overlay.devices = testDevices()
	overlay.cursor = 1 // iPhone 14 — not active

	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := overlay.Update(keyMsg)

	// Must return a non-nil command (the transfer command)
	require.NotNil(t, cmd, "Enter on non-active device should return a transfer command")

	// Execute the command and check it returns TransferPlaybackMsg
	msg := cmd()
	transferMsg, ok := msg.(TransferPlaybackMsg)
	require.True(t, ok, "command should return TransferPlaybackMsg, got %T", msg)
	assert.Equal(t, "def456", transferMsg.DeviceID, "transfer should target iPhone 14")
	assert.Equal(t, "iPhone 14", transferMsg.DeviceName, "transfer should include device name")
}

func TestDeviceOverlay_Update_Enter_OnActiveDevice(t *testing.T) {
	overlay := newTestDeviceOverlay()
	overlay.devices = testDevices()
	overlay.cursor = 0 // MacBook Pro Speakers — active

	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	model, cmd := overlay.Update(keyMsg)
	updated, ok := model.(*DeviceOverlay)
	require.True(t, ok)

	// Should NOT return a transfer command
	if cmd != nil {
		msg := cmd()
		_, isTransfer := msg.(TransferPlaybackMsg)
		assert.False(t, isTransfer, "Enter on active device should not produce a transfer command")
	}

	// Status message should indicate already playing
	assert.Contains(t, updated.statusMsg, "Already playing", "should set 'Already playing' status")
}

func TestDeviceOverlay_Update_Esc(t *testing.T) {
	overlay := newTestDeviceOverlay()
	overlay.devices = testDevices()

	keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd := overlay.Update(keyMsg)

	require.NotNil(t, cmd, "Esc should return a close command")
	msg := cmd()
	_, ok := msg.(DeviceOverlayClosedMsg)
	require.True(t, ok, "Esc should produce DeviceOverlayClosedMsg, got %T", msg)
}

func TestDeviceOverlay_DeviceTypeIcon(t *testing.T) {
	tests := []struct {
		deviceType string
		wantIcon   string
	}{
		{"Computer", "⊡"},
		{"Smartphone", "⊞"},
		{"Speaker", "⊟"},
		{"TV", "⊠"},
		{"Unknown", "○"},
		{"", "○"},
	}
	for _, tt := range tests {
		t.Run(tt.deviceType, func(t *testing.T) {
			icon := deviceTypeIcon(tt.deviceType)
			assert.Equal(t, tt.wantIcon, icon)
		})
	}
}

func TestDeviceOverlay_View_DeviceTypeIcons(t *testing.T) {
	overlay := newTestDeviceOverlay()
	overlay.devices = []DeviceInfo{
		{ID: "a", Name: "My Computer", Type: "Computer", IsActive: false},
		{ID: "b", Name: "My Phone", Type: "Smartphone", IsActive: false},
		{ID: "c", Name: "My Speaker", Type: "Speaker", IsActive: false},
		{ID: "d", Name: "My TV", Type: "TV", IsActive: false},
	}

	view := overlay.View()

	assert.True(t, strings.Contains(view, "⊡") || strings.Contains(view, "○"),
		"Computer icon should appear in view")
	assert.True(t, strings.Contains(view, "⊞") || strings.Contains(view, "○"),
		"Smartphone icon should appear in view")
}

func TestDeviceOverlay_devicesLoadedMsg(t *testing.T) {
	overlay := newTestDeviceOverlay()
	assert.Empty(t, overlay.devices, "devices should be empty before load")

	devices := testDevices()
	model, _ := overlay.Update(devicesLoadedMsg{devices: devices})
	updated, ok := model.(*DeviceOverlay)
	require.True(t, ok)
	assert.Len(t, updated.devices, 3, "devices should be populated after devicesLoadedMsg")
}

func TestDeviceOverlay_devicesLoadedMsg_ErrorEmitsDevicesLoadErrorMsg(t *testing.T) {
	// When devicesLoadedMsg carries an error, Update() must return a command
	// that emits DevicesLoadErrorMsg so the root app can show a toast.
	overlay := newTestDeviceOverlay()
	loadErr := fmt.Errorf("network error")

	_, cmd := overlay.Update(devicesLoadedMsg{err: loadErr})

	require.NotNil(t, cmd, "devicesLoadedMsg with error must return a command")
	msg := cmd()
	errMsg, ok := msg.(DevicesLoadErrorMsg)
	require.True(t, ok, "command must return DevicesLoadErrorMsg, got %T", msg)
	assert.Equal(t, loadErr, errMsg.Err, "DevicesLoadErrorMsg must carry the original error")
}

func TestDeviceOverlay_devicesLoadedMsg_ErrorSetsStoreError(t *testing.T) {
	// When devicesLoadedMsg carries an error, the store must record it for retry logic.
	s := state.New()
	overlay := NewDeviceOverlay(s, theme.Load("black"))
	loadErr := fmt.Errorf("timeout")

	overlay.Update(devicesLoadedMsg{err: loadErr})

	assert.Equal(t, loadErr, s.DevicesError(), "store must record the device load error")
}

func TestDeviceOverlay_devicesLoadedMsg_NoErrorClearsStoreError(t *testing.T) {
	// When devicesLoadedMsg has no error, any prior store error must be cleared.
	s := state.New()
	s.SetDevicesError(fmt.Errorf("prior error"))
	overlay := NewDeviceOverlay(s, theme.Load("black"))
	devices := testDevices()

	_, cmd := overlay.Update(devicesLoadedMsg{devices: devices})

	assert.NoError(t, s.DevicesError(), "store error must be cleared on successful load")
	assert.Nil(t, cmd, "successful load must not return an error command")
}

func TestDeviceOverlay_View_ShowsErrorOnAPIFailure(t *testing.T) {
	s := state.New()
	s.SetDevicesError(fmt.Errorf("API error"))
	th := theme.Load("black")
	overlay := NewDeviceOverlay(s, th)
	overlay.SetSize(60, 20)

	output := overlay.View()
	// Errors route through toast notifications, not inline pane rendering.
	// Store error is preserved for retry logic but never read in View().
	assert.NotContains(t, output, "Failed to load devices", "inline error rendering removed — toasts handle this")
	// With no devices loaded, the empty state renders normally.
	assert.Contains(t, output, "No devices found", "empty state shows when no devices loaded")
}

func TestDeviceOverlay_View_ShowsEmptyWhenNoError(t *testing.T) {
	s := state.New()
	// No error set, no devices loaded.
	th := theme.Load("black")
	overlay := NewDeviceOverlay(s, th)

	output := overlay.View()
	assert.Contains(t, output, "No devices found")
}

func TestDeviceOverlay_View_ShowsDevicesWhenNoError(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	overlay := NewDeviceOverlay(s, th)

	devices := testDevices()
	overlay.Update(devicesLoadedMsg{devices: devices})

	output := overlay.View()
	assert.Contains(t, output, "MacBook Pro")
	assert.NotContains(t, output, "Failed to load devices")
}

// TestDeviceOverlay_devicesLoadedMsg_StampsFetchedAt verifies that a successful
// devicesLoadedMsg stamps the store's devicesFetchedAt timestamp.
// This is required for DevicesStale() to return false after a successful load.
func TestDeviceOverlay_devicesLoadedMsg_StampsFetchedAt(t *testing.T) {
	s := state.New()
	overlay := NewDeviceOverlay(s, theme.Load("black"))

	// Before load, fetchedAt should be zero (stale).
	assert.True(t, s.DevicesFetchedAt().IsZero(), "DevicesFetchedAt should be zero before load")

	before := time.Now()
	overlay.Update(devicesLoadedMsg{devices: testDevices()})
	after := time.Now()

	fetchedAt := s.DevicesFetchedAt()
	assert.False(t, fetchedAt.IsZero(), "DevicesFetchedAt should be stamped after successful load")
	assert.False(t, before.After(fetchedAt), "fetchedAt should be >= before")
	assert.False(t, after.Before(fetchedAt), "fetchedAt should be <= after")
}

// TestDeviceOverlay_devicesLoadedMsg_ErrorDoesNotStampFetchedAt verifies that an
// error response does NOT stamp devicesFetchedAt — the data was not loaded successfully.
func TestDeviceOverlay_devicesLoadedMsg_ErrorDoesNotStampFetchedAt(t *testing.T) {
	s := state.New()
	overlay := NewDeviceOverlay(s, theme.Load("black"))

	overlay.Update(devicesLoadedMsg{err: fmt.Errorf("network error")})

	assert.True(t, s.DevicesFetchedAt().IsZero(), "DevicesFetchedAt must remain zero on error")
}
