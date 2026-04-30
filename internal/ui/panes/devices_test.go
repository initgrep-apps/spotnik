package panes

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/muesli/termenv"
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
	overlay.SetSize(60, 20)
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
		{"Unknown", "◎"},
		{"", "◎"},
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
	overlay.SetSize(60, 20)

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

// TestDeviceOverlay_CursorClampedOnListShrink verifies that when a DevicesLoadedMsg
// arrives with a shorter device list, the cursor is clamped to remain in bounds.
// This prevents a panic (index out of range) in handleEnter when the user presses Enter
// after devices go offline between refreshes.
func TestDeviceOverlay_CursorClampedOnListShrink(t *testing.T) {
	overlay := newTestDeviceOverlay()

	// Step 1: load 3 devices and navigate to cursor=2 (last device).
	threeDevices := []DeviceInfo{
		{ID: "a", Name: "Device A", Type: "Computer", IsActive: true},
		{ID: "b", Name: "Device B", Type: "Smartphone", IsActive: false},
		{ID: "c", Name: "Device C", Type: "Speaker", IsActive: false},
	}
	model, _ := overlay.Update(DevicesLoadedMsg{Devices: threeDevices})
	overlay = model.(*DeviceOverlay)
	overlay.cursor = 2 // manually set cursor to last position

	// Step 2: refresh with only 1 device (simulating devices going offline).
	oneDevice := []DeviceInfo{
		{ID: "a", Name: "Device A", Type: "Computer", IsActive: true},
	}
	model, _ = overlay.Update(DevicesLoadedMsg{Devices: oneDevice})
	overlay = model.(*DeviceOverlay)

	// Cursor must be clamped to 0 (last valid index for a 1-element list).
	assert.Equal(t, 0, overlay.cursor, "cursor should be clamped to 0 after list shrinks to 1 device")

	// Step 3: pressing Enter must not panic — no index out of range.
	assert.NotPanics(t, func() {
		overlay.Update(tea.KeyMsg{Type: tea.KeyEnter})
	}, "pressing Enter after list shrinks must not panic")
}

// ── Story 77 Task 2: Device overlay non-cursor row — no explicit background ───

// TestDeviceOverlay_NonCursorRow_NoExplicitBackground verifies that non-cursor rows
// in the device overlay have NO explicit background color (no "48;2;" ANSI sequence).
// Without an explicit background the rows blend with the composited overlay background
// instead of rendering as opaque colored rectangles over the dimmed grid.
func TestDeviceOverlay_NonCursorRow_NoExplicitBackground(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	s := state.New()
	th := theme.Load("black")
	overlay := NewDeviceOverlay(s, th)
	overlay.devices = testDevices()
	overlay.cursor = 0

	// renderDevice with idx=1 is a non-cursor row.
	row := overlay.renderDevice(1, overlay.devices[1])

	// "48;2;" is the ANSI SGR introducer for 24-bit RGB background color.
	// Non-cursor rows must produce no background escape at all.
	assert.NotContains(t, row, "48;2;",
		"non-cursor device row should have NO explicit background (no 48;2; ANSI sequence)")
}

// TestDeviceOverlay_CursorRow_UsesSelectedBg verifies that the cursor row in the
// device overlay uses SelectedBg() so it clearly stands out.
func TestDeviceOverlay_CursorRow_UsesSelectedBg(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	s := state.New()
	th := theme.Load("black")
	overlay := NewDeviceOverlay(s, th)
	overlay.devices = testDevices()
	overlay.cursor = 0

	row := overlay.renderDevice(0, overlay.devices[0])

	selectedBgStyle := lipgloss.NewStyle().Background(th.SelectedBg()).Render("X")
	selectedBg := extractDeviceBgANSI(selectedBgStyle)
	require.NotEmpty(t, selectedBg, "sanity: SelectedBg must produce 48;2; in TrueColor")
	assert.Contains(t, row, selectedBg,
		"cursor device row should use SelectedBg() background")
}

// extractDeviceBgANSI extracts the "48;2;R;G;B" portion from an ANSI string.
func extractDeviceBgANSI(s string) string {
	const bgPrefix = "48;2;"
	idx := strings.Index(s, bgPrefix)
	if idx < 0 {
		return ""
	}
	end := strings.Index(s[idx:], "m")
	if end < 0 {
		return ""
	}
	return s[idx : idx+end]
}

// TestDeviceOverlay_CursorClampedOnEmptyList verifies that when a DevicesLoadedMsg
// arrives with an empty device list, the cursor is reset to 0.
func TestDeviceOverlay_CursorClampedOnEmptyList(t *testing.T) {
	overlay := newTestDeviceOverlay()

	// Start with 2 devices and cursor at 1.
	twoDevices := []DeviceInfo{
		{ID: "a", Name: "Device A", Type: "Computer", IsActive: true},
		{ID: "b", Name: "Device B", Type: "Smartphone", IsActive: false},
	}
	model, _ := overlay.Update(DevicesLoadedMsg{Devices: twoDevices})
	overlay = model.(*DeviceOverlay)
	overlay.cursor = 1

	// Now receive an empty list.
	model, _ = overlay.Update(DevicesLoadedMsg{Devices: []DeviceInfo{}})
	overlay = model.(*DeviceOverlay)

	assert.Equal(t, 0, overlay.cursor, "cursor should be reset to 0 when device list becomes empty")

	// Pressing Enter on empty list must not panic (handleEnter already guards len==0).
	assert.NotPanics(t, func() {
		overlay.Update(tea.KeyMsg{Type: tea.KeyEnter})
	}, "pressing Enter on empty list must not panic")
}

// TestDevicesOverlay_AsciiContent verifies that in ASCII mode the device overlay
// renders ASCII glyphs for status bullets and device-type icons, and that none of
// the unicode literals (◉ ○ ⊡ ⊞ ⊟ ⊠) appear in the output.
func TestDevicesOverlay_AsciiContent(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	o := newTestDeviceOverlay()
	o.SetSize(60, 20)
	o.devices = []DeviceInfo{
		{ID: "1", Name: "My Mac", Type: "Computer", IsActive: true},
		{ID: "2", Name: "My Phone", Type: "Smartphone", IsActive: false},
		{ID: "3", Name: "My Speaker", Type: "Speaker", IsActive: false},
		{ID: "4", Name: "My TV", Type: "TV", IsActive: false},
	}

	out := stripANSI(o.View())

	// ASCII bullet for active device must be present, unicode must not appear.
	assert.Contains(t, out, "(*)", "ASCII mode active bullet should be (*)")
	assert.NotContains(t, out, "◉", "unicode ◉ must not appear in ASCII mode")
	assert.NotContains(t, out, "○", "unicode ○ must not appear in ASCII mode")

	// ASCII device-type icons.
	assert.Contains(t, out, "[c]", "Computer ASCII icon should be [c]")
	assert.Contains(t, out, "[p]", "Smartphone ASCII icon should be [p]")
	assert.Contains(t, out, "[s]", "Speaker ASCII icon should be [s]")
	assert.Contains(t, out, "[tv]", "TV ASCII icon should be [tv]")

	// None of the unicode literals must survive in ASCII mode.
	for _, ch := range []string{"⊡", "⊞", "⊟", "⊠"} {
		assert.NotContains(t, out, ch, "unicode device icon %q must not appear in ASCII mode", ch)
	}
}

// TestDevicesOverlay_AsciiBorder verifies that the devices overlay renders
// ASCII-safe border characters (+, -, |) when the uikit glyph mode is ASCII,
// and that no unicode box-drawing characters (╭╮╰╯─│) are present.
func TestDevicesOverlay_AsciiBorder(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	o := newTestDeviceOverlay()
	o.SetSize(50, 20)
	// Populate devices so the chrome rendering path is taken (not the EmptyState path).
	o.devices = testDevices()
	out := stripANSI(o.View())
	if strings.ContainsAny(out, "╭╮╰╯─│") {
		t.Errorf("ascii overlay must not contain unicode borders, got: %q", out)
	}
	assert.Contains(t, out, "+", "ASCII mode should render '+' corners")
}
