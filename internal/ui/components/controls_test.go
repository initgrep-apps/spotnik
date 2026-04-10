package components

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func newTestControls(isPlaying, shuffleOn bool, repeatMode string) Controls {
	t := theme.Load("black")
	return NewControls(t, isPlaying, shuffleOn, repeatMode, domain.PlaybackActions{}, true)
}

func newTestControlsDisallows(isPlaying, shuffleOn bool, repeatMode string, disallows domain.PlaybackActions, supportsVolume bool) Controls {
	t := theme.Load("black")
	return NewControls(t, isPlaying, shuffleOn, repeatMode, disallows, supportsVolume)
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
	assert.NotContains(t, out, "↻¹", "off state should not show superscript one")
}

func TestControls_RepeatContext(t *testing.T) {
	c := newTestControls(false, false, "context")
	out := c.Render()
	assert.Contains(t, out, "↻")
}

func TestControls_RepeatTrack(t *testing.T) {
	c := newTestControls(false, false, "track")
	out := c.Render()
	assert.Contains(t, out, "↻¹") // was "↻1"
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

func TestControls_RepeatTrack_SuperscriptOne(t *testing.T) {
	c := newTestControls(false, false, "track")
	out := c.Render()
	assert.Contains(t, out, "↻¹", "repeat-track should use superscript one (U+00B9)")
	assert.NotContains(t, out, "↻1", "repeat-track should not use ASCII 1")
}

func TestControls_ShuffleDisabled(t *testing.T) {
	c := newTestControlsDisallows(false, false, "off",
		domain.PlaybackActions{TogglingShuffle: true}, true)
	out := c.Render()
	assert.Contains(t, out, "⇄", "disabled shuffle should still render the icon")
}

func TestControls_RepeatDisabled_BothModesDisallowed(t *testing.T) {
	c := newTestControlsDisallows(false, false, "off",
		domain.PlaybackActions{TogglingRepeatContext: true, TogglingRepeatTrack: true}, true)
	out := c.Render()
	assert.Contains(t, out, "↻", "disabled repeat should still render the icon")
}

func TestControls_RepeatNotDisabled_OnlyContextDisallowed(t *testing.T) {
	c := newTestControlsDisallows(false, false, "off",
		domain.PlaybackActions{TogglingRepeatContext: true, TogglingRepeatTrack: false}, true)
	out := c.Render()
	assert.Contains(t, out, "↻")
}

func TestControls_PlayDisabled_ResumingDisallowed(t *testing.T) {
	c := newTestControlsDisallows(false, false, "off",
		domain.PlaybackActions{Resuming: true}, true)
	out := c.Render()
	assert.Contains(t, out, "▷", "disabled play should still render the icon")
}
