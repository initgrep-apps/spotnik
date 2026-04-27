package panes_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func newTestPollingTrafficPane(t *testing.T) *panes.PollingTrafficPane {
	t.Helper()
	return panes.NewPollingTrafficPane(state.New(), theme.Load("black"))
}

func TestPollingTrafficPane_ImplementsLayoutPane(t *testing.T) {
	var _ layout.Pane = newTestPollingTrafficPane(t)
}

func TestPollingTrafficPane_ID(t *testing.T) {
	assert.Equal(t, layout.PanePollingTraffic, newTestPollingTrafficPane(t).ID())
}

func TestPollingTrafficPane_Title(t *testing.T) {
	assert.Equal(t, "Polling Traffic", newTestPollingTrafficPane(t).Title())
}

func TestPollingTrafficPane_ToggleKey(t *testing.T) {
	assert.Equal(t, 3, newTestPollingTrafficPane(t).ToggleKey())
}

func TestPollingTrafficPane_View_EmptyBeforeResize(t *testing.T) {
	assert.Equal(t, "", newTestPollingTrafficPane(t).View())
}

func TestPollingTrafficPane_View_ContainsAllRows(t *testing.T) {
	p := newTestPollingTrafficPane(t)
	p.SetSize(50, 10)
	view := p.View()
	assert.Contains(t, view, "Playback")
	assert.Contains(t, view, "Playlists")
	assert.Contains(t, view, "Albums")
	assert.Contains(t, view, "Liked")
	assert.Contains(t, view, "Recent")
}

func TestPollingTrafficPane_View_NoBorder(t *testing.T) {
	p := newTestPollingTrafficPane(t)
	p.SetSize(50, 10)
	view := p.View()
	// render.go adds the outer border; View() must return raw content only.
	assert.NotContains(t, view, "╭")
	assert.NotContains(t, view, "╰")
}

func TestPollingTrafficPane_Update_PollingSnapshotMsg(t *testing.T) {
	p := newTestPollingTrafficPane(t)
	p.SetSize(50, 10)

	model, cmd := p.Update(panes.PollingSnapshotMsg{
		TickIntervalMs: 1000,
		IsIdle:         false,
	})
	assert.Nil(t, cmd)
	view := model.(*panes.PollingTrafficPane).View()
	// Playback row reflects "running" state when not idle.
	assert.Contains(t, view, "running")
}

func TestPollingTrafficPane_Update_IdleSnapshot(t *testing.T) {
	p := newTestPollingTrafficPane(t)
	p.SetSize(50, 10)

	model, _ := p.Update(panes.PollingSnapshotMsg{
		TickIntervalMs: 10000,
		IsIdle:         true,
		IdleSecs:       90,
	})
	view := model.(*panes.PollingTrafficPane).View()
	assert.Contains(t, view, "idle")
}
