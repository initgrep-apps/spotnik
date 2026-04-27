package panes_test

import (
	"testing"
	"time"

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

// TestPollingTrafficPane_Update_Running asserts that a non-idle snapshot with
// TickIntervalMs=1000 renders "1s · running" in the Playback row.
func TestPollingTrafficPane_Update_Running(t *testing.T) {
	p := newTestPollingTrafficPane(t)
	p.SetSize(80, 10)

	model, cmd := p.Update(panes.PollingSnapshotMsg{
		TickIntervalMs: 1000,
		IsIdle:         false,
	})
	assert.Nil(t, cmd)
	view := model.(*panes.PollingTrafficPane).View()
	assert.Contains(t, view, "1s · running",
		"running snapshot with 1000ms interval must render '1s · running'")
}

// TestPollingTrafficPane_Update_Idle asserts that an idle snapshot with
// TickIntervalMs=10000 renders "idle · 10s" in the Playback row.
func TestPollingTrafficPane_Update_Idle(t *testing.T) {
	p := newTestPollingTrafficPane(t)
	p.SetSize(80, 10)

	model, _ := p.Update(panes.PollingSnapshotMsg{
		TickIntervalMs: 10000,
		IsIdle:         true,
		IdleSecs:       90,
	})
	view := model.(*panes.PollingTrafficPane).View()
	assert.Contains(t, view, "idle · 10s",
		"idle snapshot with 10000ms interval must render 'idle · 10s'")
}

// TestPollingTrafficPane_CacheStale_WarningVsError asserts that a cache age
// below 1 hour (Warning color) produces different ANSI output than a cache
// age at or above 1 hour (Error color).
func TestPollingTrafficPane_CacheStale_WarningVsError(t *testing.T) {
	makeView := func(age time.Duration) string {
		st := state.New()
		// Force playlists to be stale by setting fetchedAt far in the past.
		// state.PlaylistsTTL is typically 5 minutes; setting fetchedAt to
		// (now - age) ensures it is well past TTL.
		st.SetPlaylistsFetchedAt(time.Now().Add(-age))
		p := panes.NewPollingTrafficPane(st, theme.Load("black"))
		p.SetSize(80, 10)
		return p.View()
	}

	// Under 1 hour → Warning color.
	viewWarn := makeView(30 * time.Minute)
	// At or above 1 hour → Error color.
	viewError := makeView(2 * time.Hour)

	assert.NotEqual(t, viewWarn, viewError,
		"stale<1h (Warning) and stale≥1h (Error) must produce different ANSI-colored output")
}

// TestCacheAge covers all 4 buckets of the cacheAge helper and their boundaries.
func TestCacheAge(t *testing.T) {
	tests := []struct {
		name string
		age  time.Duration
		want string
	}{
		// Bucket 1: < 1 minute → "just now"
		{"30 seconds", 30 * time.Second, "just now"},
		{"59 seconds", 59 * time.Second, "just now"},
		// Boundary: exactly 60s → first second of "minutes" bucket.
		{"60 seconds boundary", 60 * time.Second, "1m"},
		// Bucket 2: >= 1m < 1h → "Xm"
		{"5 minutes", 5 * time.Minute, "5m"},
		{"59 minutes", 59 * time.Minute, "59m"},
		// Boundary: exactly 60m → first second of "hours" bucket.
		{"60 minutes boundary", 60 * time.Minute, "1h"},
		// Bucket 3: >= 1h, 0 leftover minutes → "Xh"
		{"2 hours exact", 2 * time.Hour, "2h"},
		// Bucket 4: >= 1h, non-zero leftover minutes → "Xh Ym"
		{"2h 15m", 2*time.Hour + 15*time.Minute, "2h 15m"},
		{"1h 1m", 1*time.Hour + 1*time.Minute, "1h 1m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// cacheAge(t) computes time.Since(t), so pass (now - age).
			got := panes.CacheAge(time.Now().Add(-tt.age))
			assert.Equal(t, tt.want, got)
		})
	}
}
