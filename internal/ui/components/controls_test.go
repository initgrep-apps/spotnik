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
	output := c.Render()
	assert.Contains(t, output, "||", "playing state should show pause symbol")
}

func TestControls_Paused_ShowsPlay(t *testing.T) {
	c := newTestControls(false, false, "off")
	output := c.Render()
	assert.Contains(t, output, ">", "paused state should show play symbol")
}

func TestControls_ShuffleOn(t *testing.T) {
	c := newTestControls(false, true, "off")
	output := c.Render()
	assert.Contains(t, output, "~", "should contain shuffle icon")
}

func TestControls_ShuffleOff(t *testing.T) {
	c := newTestControls(false, false, "off")
	output := c.Render()
	assert.Contains(t, output, "~")
}

func TestControls_RepeatOff(t *testing.T) {
	c := newTestControls(false, false, "off")
	output := c.Render()
	assert.Contains(t, output, "=>")
}

func TestControls_RepeatContext(t *testing.T) {
	c := newTestControls(false, false, "context")
	output := c.Render()
	assert.Contains(t, output, "=>")
}

func TestControls_RepeatTrack(t *testing.T) {
	c := newTestControls(false, false, "track")
	output := c.Render()
	assert.Contains(t, output, "=>1", "repeat track should show =>1 symbol")
}

func TestControls_AlwaysHasSkipIcons(t *testing.T) {
	c := newTestControls(true, true, "context")
	output := c.Render()
	assert.Contains(t, output, "|<")
	assert.Contains(t, output, ">|")
}
