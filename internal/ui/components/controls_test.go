package components

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func newTestControls(isPlaying, shuffleOn bool, repeatMode string) Controls {
	t := theme.Load("black")
	return NewControls(t, isPlaying, shuffleOn, repeatMode)
}

func TestControls_Playing_ShowsPause(t *testing.T) {
	c := newTestControls(true, false, "off")
	out := c.Render()
	assert.Contains(t, out, "⏸", "playing state should show pause symbol")
	assert.NotContains(t, out, "▷", "playing state should not show play symbol")
}

func TestControls_Paused_ShowsPlay(t *testing.T) {
	c := newTestControls(false, false, "off")
	out := c.Render()
	assert.Contains(t, out, "▷", "paused state should show play symbol")
	assert.NotContains(t, out, "⏸", "paused state should not show pause symbol")
}

func TestControls_ShuffleOn(t *testing.T) {
	c := newTestControls(false, true, "off")
	out := c.Render()
	assert.Contains(t, out, "⇄")
}

func TestControls_ShuffleOff(t *testing.T) {
	c := newTestControls(false, false, "off")
	out := c.Render()
	assert.Contains(t, out, "⇄")
}

func TestControls_RepeatOff(t *testing.T) {
	c := newTestControls(false, false, "off")
	out := c.Render()
	assert.Contains(t, out, "↻")
	assert.NotContains(t, out, "↻1")
}

func TestControls_RepeatContext(t *testing.T) {
	c := newTestControls(false, false, "context")
	out := c.Render()
	assert.Contains(t, out, "↻")
}

func TestControls_RepeatTrack(t *testing.T) {
	c := newTestControls(false, false, "track")
	out := c.Render()
	assert.Contains(t, out, "↻1")
}

func TestControls_QueueIcon(t *testing.T) {
	c := newTestControls(false, false, "off")
	out := c.Render()
	assert.Contains(t, out, "≡")
}

func TestControls_NoPrevNext(t *testing.T) {
	c := newTestControls(true, true, "context")
	out := c.Render()
	assert.NotContains(t, out, "|<")
	assert.NotContains(t, out, ">|")
}

func TestControls_NoOldSymbols(t *testing.T) {
	c := newTestControls(true, true, "context")
	out := c.Render()
	assert.NotContains(t, out, "~")
	assert.NotContains(t, out, "=>")
}
