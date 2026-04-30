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
	// RepeatOff uses GlyphRepeatOff (⟳), not GlyphRepeatAll (↻)
	assert.Contains(t, out, "⟳")
	assert.NotContains(t, out, "↻1")
	assert.NotContains(t, out, "↻¹")
}

func TestControls_RepeatContext(t *testing.T) {
	c := newTestControls(false, false, "context")
	out := c.Render()
	assert.Contains(t, out, "↻")
	assert.NotContains(t, out, "↻¹")
}

func TestControls_RepeatTrack(t *testing.T) {
	c := newTestControls(false, false, "track")
	out := c.Render()
	// superscript one (U+00B9) not ASCII 1
	assert.Contains(t, out, "↻¹")
	assert.NotContains(t, out, "↻1")
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

// TestNewControls_RepeatModeTranslation verifies all four string inputs are
// correctly translated to uikit.RepeatMode values.
func TestNewControls_RepeatModeTranslation(t *testing.T) {
	tests := []struct {
		input       string
		wantContain string
		desc        string
	}{
		{"off", "⟳", "off → RepeatOff renders ⟳"},
		{"context", "↻", "context → RepeatAll renders ↻"},
		{"track", "↻¹", "track → RepeatOne renders ↻¹"},
		{"unknown", "⟳", "unknown → RepeatOff (fallback) renders ⟳"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			c := newTestControls(false, false, tt.input)
			out := c.Render()
			assert.Contains(t, out, tt.wantContain, tt.desc)
		})
	}
}
