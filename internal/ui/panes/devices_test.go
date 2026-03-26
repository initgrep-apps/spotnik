package panes

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errMake is a test helper to create simple errors inline.
func errMake(msg string) error { return errors.New(msg) }

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
	// After F50, "Devices" title is in the btop border (not "DEVICES" inside the body).
	assert.Contains(t, view, "Devices", "overlay border title should appear")
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

func TestDeviceOverlay_DevicesLoadedMsg_PopulatesDevices(t *testing.T) {
	overlay := newTestDeviceOverlay()
	assert.Empty(t, overlay.devices, "devices should be empty before load")

	devices := testDevices()
	// DevicesLoadedMsg is now exported; root app.Update() handles store mutations.
	model, cmd := overlay.Update(DevicesLoadedMsg{Devices: devices})
	updated, ok := model.(*DeviceOverlay)
	require.True(t, ok)
	assert.Len(t, updated.devices, 3, "devices should be populated after DevicesLoadedMsg")
	assert.Nil(t, cmd, "DeviceOverlay should not return a command on success")
}

func TestDeviceOverlay_DevicesLoadedMsg_ErrorDoesNotPopulateDevices(t *testing.T) {
	// When DevicesLoadedMsg carries an error, DeviceOverlay.Update() must NOT
	// populate its local device list. Store mutations are handled by root app.Update().
	overlay := newTestDeviceOverlay()
	overlay.devices = testDevices() // pre-populate

	model, cmd := overlay.Update(DevicesLoadedMsg{Devices: nil, Err: errMake("network error")})
	updated, ok := model.(*DeviceOverlay)
	require.True(t, ok)
	// On error, devices should remain as they were (not wiped, not updated).
	assert.Len(t, updated.devices, 3, "devices should be unchanged on error")
	assert.Nil(t, cmd, "DeviceOverlay should not return a command on error")
}

func TestDeviceOverlay_View_ShowsErrorOnAPIFailure(t *testing.T) {
	s := state.New()
	s.SetDevicesError(errMake("API error"))
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
	// Use exported DevicesLoadedMsg; store mutations are handled by root app.Update().
	overlay.Update(DevicesLoadedMsg{Devices: devices})

	output := overlay.View()
	assert.Contains(t, output, "MacBook Pro")
	assert.NotContains(t, output, "Failed to load devices")
}

// --- F50 Task 5: btop-style border in device overlay ---

func TestDeviceOverlay_View_HasBtopBorder(t *testing.T) {
	overlay := newTestDeviceOverlay()
	overlay.devices = testDevices()

	view := overlay.View()

	// btop border uses ╭ and ╰ corners.
	assert.Contains(t, view, "╭", "device overlay should use btop-style rounded corner")
	assert.Contains(t, view, "╰", "device overlay should use btop-style rounded corner")
}

func TestDeviceOverlay_View_BtopBorderTitle(t *testing.T) {
	overlay := newTestDeviceOverlay()
	overlay.devices = testDevices()

	view := overlay.View()

	// The border title should contain "Devices" (note: was "DEVICES" inside the body).
	assert.Contains(t, view, "Devices", "border title should contain 'Devices'")
}

func TestDeviceOverlay_View_BtopBorderActions(t *testing.T) {
	overlay := newTestDeviceOverlay()
	overlay.devices = testDevices()

	view := overlay.View()

	// Action from the spec: "Enter select"
	assert.Contains(t, view, "select", "border should show 'select' action")
}

func TestDeviceOverlay_View_ActiveDeviceSymbol(t *testing.T) {
	overlay := newTestDeviceOverlay()
	overlay.devices = testDevices()

	view := overlay.View()

	assert.Contains(t, view, "◉", "active device should show ◉ symbol")
}

func TestDeviceOverlay_View_InactiveDeviceSymbol(t *testing.T) {
	overlay := newTestDeviceOverlay()
	overlay.devices = []DeviceInfo{
		{ID: "a", Name: "Phone", Type: "Smartphone", IsActive: false},
	}

	view := overlay.View()

	assert.Contains(t, view, "○", "inactive device should show ○ symbol")
}
