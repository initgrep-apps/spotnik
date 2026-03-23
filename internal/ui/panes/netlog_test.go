package panes_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newNetLogView() (*panes.NetLogView, *state.Store) {
	t := theme.Load("black")
	s := state.New()
	v := panes.NewNetLogView(s, t)
	v.SetSize(80, 10)
	return v, s
}

func TestNetLogView_Empty(t *testing.T) {
	v, _ := newNetLogView()
	output := v.View()
	assert.Contains(t, output, "No API calls recorded")
}

func TestNetLogView_ShowsEntries(t *testing.T) {
	v, s := newNetLogView()

	s.RecordNetCall("GET", "/v1/me/player", 200, 42)
	s.RecordNetCall("GET", "/v1/me/player/queue", 429, 5)

	output := v.View()
	assert.Contains(t, output, "TIME")
	assert.Contains(t, output, "/v1/me/player")
	assert.Contains(t, output, "429")
}

func TestNetLogView_ChronologicalOrder(t *testing.T) {
	v, s := newNetLogView()

	s.RecordNetCall("GET", "/v1/first", 200, 10)
	s.RecordNetCall("GET", "/v1/second", 200, 20)

	output := v.View()
	// Entries are oldest-first (chronological), so "first" appears before "second".
	firstIdx := -1
	secondIdx := -1
	for i := range output {
		if i+len("/v1/first") <= len(output) && output[i:i+len("/v1/first")] == "/v1/first" && firstIdx == -1 {
			firstIdx = i
		}
		if i+len("/v1/second") <= len(output) && output[i:i+len("/v1/second")] == "/v1/second" && secondIdx == -1 {
			secondIdx = i
		}
	}
	require.NotEqual(t, -1, firstIdx, "should find /v1/first")
	require.NotEqual(t, -1, secondIdx, "should find /v1/second")
	assert.Less(t, firstIdx, secondIdx, "oldest entry should appear first (chronological)")
}

func TestNetLogView_AutoScroll_PinnedByDefault(t *testing.T) {
	v, s := newNetLogView()
	v.SetSize(80, 4) // header + 3 visible rows

	assert.True(t, v.Pinned(), "should start pinned")

	// Add 10 entries — more than visible.
	for i := 0; i < 10; i++ {
		s.RecordNetCall("GET", "/v1/test", 200, int64(i))
	}

	// Render — auto-scroll should pin to the bottom.
	_ = v.View()
	// scroll should be at the end: 10 entries - 3 visible = 7
	assert.Equal(t, 7, v.Scroll())
	assert.True(t, v.Pinned())
}

func TestNetLogView_ScrollUp_Unpins(t *testing.T) {
	v, s := newNetLogView()
	v.SetSize(80, 4) // header + 3 visible rows

	for i := 0; i < 10; i++ {
		s.RecordNetCall("GET", "/v1/test", 200, int64(i))
	}

	// Render to set scroll position.
	_ = v.View()
	assert.True(t, v.Pinned())

	// Scroll up — should unpin.
	v.ScrollUp()
	assert.False(t, v.Pinned())
	assert.Equal(t, 6, v.Scroll())
}

func TestNetLogView_ScrollDown_RepinsAtBottom(t *testing.T) {
	v, s := newNetLogView()
	v.SetSize(80, 4) // header + 3 visible rows

	for i := 0; i < 10; i++ {
		s.RecordNetCall("GET", "/v1/test", 200, int64(i))
	}

	_ = v.View()
	v.ScrollUp() // scroll=6, unpinned
	assert.False(t, v.Pinned())

	v.ScrollDown() // scroll=7, back at bottom
	assert.True(t, v.Pinned())
}

func TestNetLogView_ScrollUp_AtZero(t *testing.T) {
	v, _ := newNetLogView()
	v.ScrollUp()
	assert.Equal(t, 0, v.Scroll(), "scroll should not go below 0")
}
