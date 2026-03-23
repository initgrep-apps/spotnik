package panes_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
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

func TestNetLogView_NewestFirst(t *testing.T) {
	v, s := newNetLogView()

	s.RecordNetCall("GET", "/v1/first", 200, 10)
	s.RecordNetCall("GET", "/v1/second", 200, 20)

	output := v.View()
	// "second" should appear before "first" in the output (newest first).
	secondIdx := len(output) // fallback
	firstIdx := len(output)
	for i := range output {
		if i+len("/v1/second") <= len(output) && output[i:i+len("/v1/second")] == "/v1/second" {
			secondIdx = i
			break
		}
	}
	for i := range output {
		if i+len("/v1/first") <= len(output) && output[i:i+len("/v1/first")] == "/v1/first" {
			firstIdx = i
			break
		}
	}
	assert.Less(t, secondIdx, firstIdx, "newest entry should appear first")
}

func TestNetLogView_ScrollDown(t *testing.T) {
	v, s := newNetLogView()
	v.SetSize(80, 3) // header + 2 visible rows

	for i := 0; i < 10; i++ {
		s.RecordNetCall("GET", "/v1/test", 200, int64(i))
	}

	assert.Equal(t, 0, v.Scroll())
	v.ScrollDown()
	assert.Equal(t, 1, v.Scroll())
}

func TestNetLogView_ScrollUp_AtZero(t *testing.T) {
	v, _ := newNetLogView()
	v.ScrollUp()
	assert.Equal(t, 0, v.Scroll(), "scroll should not go below 0")
}
