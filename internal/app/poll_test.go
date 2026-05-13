package app_test

// poll_test.go — Behavioural tests for Story 199: universal polling infrastructure.
// These tests verify that TickMsg drives library pane and devices overlay polling.

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tea "github.com/charmbracelet/bubbletea"
)

// collectAllMsgs executes a tea.Cmd (possibly a batch) and collects every
// resulting tea.Msg by executing each sub-command in the batch.
// Only one level of nesting is resolved (batch-within-batch is not expected here).
func collectAllMsgs(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if msg == nil {
		return nil
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, c := range batch {
			if c != nil {
				m := c()
				if m != nil {
					msgs = append(msgs, m)
				}
			}
		}
		return msgs
	}
	return []tea.Msg{msg}
}

// hasLibraryMsg returns true if msgs contains any of the library-loaded message types
// (LibraryLoadedMsg, AlbumsLoadedMsg, LikedTracksLoadedMsg, RecentlyPlayedLoadedMsg,
// StatsLoadedMsg). These are the messages returned by the fetch commands when the
// API client is nil (errNilClient path), which confirms the command was dispatched.
func hasLibraryMsg(msgs []tea.Msg) bool {
	for _, m := range msgs {
		switch m.(type) {
		case panes.LibraryLoadedMsg,
			panes.AlbumsLoadedMsg,
			panes.LikedTracksLoadedMsg,
			panes.RecentlyPlayedLoadedMsg,
			panes.StatsLoadedMsg:
			return true
		}
	}
	return false
}

// TestApp_TickMsg_LibraryPollDispatchesAtTick0 verifies that the TickMsg handler
// dispatches library fetch commands at tick 0 (retry-mode interval = 5s, and
// 0 % 5 == 0 triggers the first fetch immediately).
func TestApp_TickMsg_LibraryPollDispatchesAtTick0(t *testing.T) {
	a := app.New(&config.Config{}, app.AppOptions{})

	// Tick 0: all library panes are in retry mode (hasData=false) so interval=5.
	// 0 % 5 == 0 → all five library panes should dispatch fetch commands.
	_, cmd := a.Update(panes.TickMsg{})
	require.NotNil(t, cmd, "TickMsg at tick 0 must return a batch command")

	msgs := collectAllMsgs(cmd)
	assert.True(t, hasLibraryMsg(msgs),
		"TickMsg at tick 0 must dispatch at least one library fetch cmd; got msgs: %T", msgs)
}

// TestApp_TickMsg_DevicesPollWhileOverlayOpen verifies that the devices overlay is
// polled every 10 ticks while deviceOverlayOpen is true. Since tickCount starts at 0
// when the app is created, the very first TickMsg (tickCount == 0, 0 % 10 == 0)
// triggers a device fetch — so we only need to send one tick to observe it.
func TestApp_TickMsg_DevicesPollWhileOverlayOpen(t *testing.T) {
	a := app.New(&config.Config{}, app.AppOptions{})

	// Open the device overlay by pressing 'd'. Key handlers do not advance tickCount,
	// so tickCount remains 0 after this.
	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	a = m.(*app.App)
	require.True(t, a.DeviceOverlayOpen(), "device overlay should be open after 'd' key")

	// Send one TickMsg. At tickCount=0, 0 % 10 == 0 → device fetch dispatched.
	_, cmd := a.Update(panes.TickMsg{})
	require.NotNil(t, cmd, "TickMsg at tick 0 with overlay open must return a command")

	msgs := collectAllMsgs(cmd)
	hasDevices := false
	for _, msg := range msgs {
		if _, ok := msg.(panes.DevicesLoadedMsg); ok {
			hasDevices = true
			break
		}
	}
	assert.True(t, hasDevices,
		"device poll must dispatch DevicesLoadedMsg when overlay is open and tickCount%%10==0; got msgs: %v", msgs)
}

// TestApp_TickMsg_DevicesNotPolledWhenOverlayClosed verifies that no device fetch
// is dispatched when the device overlay is closed.
func TestApp_TickMsg_DevicesNotPolledWhenOverlayClosed(t *testing.T) {
	a := app.New(&config.Config{}, app.AppOptions{})
	require.False(t, a.DeviceOverlayOpen(), "device overlay should be closed initially")

	// Send 10 ticks; device overlay is closed, so DevicesLoadedMsg should never appear.
	for i := 0; i < 10; i++ {
		m, cmd := a.Update(panes.TickMsg{})
		a = m.(*app.App)

		if cmd != nil {
			for _, msg := range collectAllMsgs(cmd) {
				if _, ok := msg.(panes.DevicesLoadedMsg); ok {
					t.Errorf("DevicesLoadedMsg dispatched at tick %d with overlay closed", i)
				}
			}
		}
	}
}
