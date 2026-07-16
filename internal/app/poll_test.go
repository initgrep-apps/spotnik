package app_test

// poll_test.go — Behavioural tests for Story 199: universal polling infrastructure.
// These tests verify that TickMsg drives library pane and devices overlay polling.

import (
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tea "github.com/charmbracelet/bubbletea"
)

// execCmdFast runs cmd with a short timeout. Fetch commands in tests resolve
// immediately; the periodic tea.Tick command blocks for 1s and is ignored.
func execCmdFast(cmd tea.Cmd, timeout time.Duration) tea.Msg {
	if cmd == nil {
		return nil
	}
	type result struct {
		msg tea.Msg
	}
	ch := make(chan result, 1)
	go func() {
		ch <- result{msg: cmd()}
	}()
	select {
	case r := <-ch:
		return r.msg
	case <-time.After(timeout):
		return nil
	}
}

// collectAllMsgs executes a tea.Cmd (which may be a BatchMsg) and recursively
// collects all resulting tea.Msg values by executing each sub-command recursively. This mirrors
// the collectInitMsgs pattern in app_test.go so nested batches are resolved
// regardless of depth.
func collectAllMsgs(cmd tea.Cmd) []tea.Msg {
	return collectAllMsgsTimeout(cmd, 10*time.Millisecond)
}

func collectAllMsgsTimeout(cmd tea.Cmd, timeout time.Duration) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := execCmdFast(cmd, timeout)
	if msg == nil {
		return nil
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, c := range batch {
			if c != nil {
				msgs = append(msgs, collectAllMsgsTimeout(c, timeout)...)
			}
		}
		return msgs
	}
	return []tea.Msg{msg}
}

// collectImmediateMsgs resolves a tea.Cmd recursively, but skips the periodic
// tea.Tick command that produces the next panes.TickMsg. This keeps tests fast
// while still expanding batches of immediate fetch commands.
func collectImmediateMsgs(cmd tea.Cmd) []tea.Msg {
	return collectAllMsgsTimeout(cmd, 10*time.Millisecond)
}

// countLibraryMsgs returns how many distinct library fetch messages are in msgs.
func countLibraryMsgs(msgs []tea.Msg) int {
	count := 0
	for _, m := range msgs {
		switch m.(type) {
		case panes.LibraryLoadedMsg, panes.AlbumsLoadedMsg, panes.LikedTracksLoadedMsg,
			panes.RecentlyPlayedLoadedMsg, panes.StatsLoadedMsg,
			panes.FollowedShowsLoadedMsg, panes.SavedEpisodesLoadedMsg:
			count++
		}
	}
	return count
}

// TestApp_TickMsg_LibraryPollDispatchesAtMostOnePane verifies the scheduler
// dispatches exactly one library pane per tick and spreads visible panes across
// consecutive ticks.
func TestApp_TickMsg_LibraryPollDispatchesAtMostOnePane(t *testing.T) {
	a := app.New(&config.Config{}, app.AppOptions{})
	model, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a = model.(*app.App)

	// Tick 0: scheduler should pick exactly one visible library pane.
	_, cmd := a.Update(panes.TickMsg{})
	require.NotNil(t, cmd, "TickMsg at tick 0 must return a command")
	msgs := collectImmediateMsgs(cmd)
	assert.Equal(t, 1, countLibraryMsgs(msgs), "scheduler must dispatch exactly one library pane per tick")
}

// TestApp_TickMsg_LibrarySchedulerCyclesVisiblePanes verifies that sending
// several consecutive ticks eventually dispatches each visible library pane type.
func TestApp_TickMsg_LibrarySchedulerCyclesVisiblePanes(t *testing.T) {
	a := app.New(&config.Config{}, app.AppOptions{})
	model, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a = model.(*app.App)

	seen := map[string]bool{}
	for i := 0; i < 20; i++ {
		m, cmd := a.Update(panes.TickMsg{})
		a = m.(*app.App)
		if cmd != nil {
			for _, msg := range collectImmediateMsgs(cmd) {
				switch msg.(type) {
				case panes.LibraryLoadedMsg:
					seen["playlists"] = true
				case panes.AlbumsLoadedMsg:
					seen["albums"] = true
				case panes.LikedTracksLoadedMsg:
					seen["liked"] = true
				case panes.RecentlyPlayedLoadedMsg:
					seen["recent"] = true
				case panes.StatsLoadedMsg:
					seen["stats"] = true
				}
			}
		}
	}

	for _, name := range []string{"playlists", "albums", "liked", "recent", "stats"} {
		assert.True(t, seen[name], "scheduler should eventually dispatch %s pane", name)
	}
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
