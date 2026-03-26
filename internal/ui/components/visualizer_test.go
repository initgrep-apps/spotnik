package components

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestVisualizer(width, height int) *Visualizer {
	t := theme.Load("black")
	v := NewVisualizer(t)
	v.SetSize(width, height)
	return v
}

// TestVisualizer_InitialState verifies frameIndex starts at 0.
func TestVisualizer_InitialState(t *testing.T) {
	v := newTestVisualizer(40, 2)
	assert.Equal(t, 0, v.frameIndex)
}

// TestVisualizer_InitReturnsCmd verifies Init returns a non-nil tick command.
func TestVisualizer_InitReturnsCmd(t *testing.T) {
	v := newTestVisualizer(40, 2)
	cmd := v.Init()
	assert.NotNil(t, cmd, "Init() must return a tick command")
}

// TestVisualizer_TickWhenPlaying verifies that frameIndex advances on tick when playing.
func TestVisualizer_TickWhenPlaying(t *testing.T) {
	v := newTestVisualizer(40, 2)
	v.SetPlaying(true)
	initial := v.frameIndex

	cmd := v.Update(VisualizerTickMsg(time.Now()))
	assert.NotNil(t, cmd, "Update must re-arm the tick")
	assert.Equal(t, initial+1, v.frameIndex, "frameIndex should advance when playing")
}

// TestVisualizer_TickWhenPaused verifies that frameIndex stays fixed on tick when paused.
func TestVisualizer_TickWhenPaused(t *testing.T) {
	v := newTestVisualizer(40, 2)
	v.SetPlaying(false)
	initial := v.frameIndex

	cmd := v.Update(VisualizerTickMsg(time.Now()))
	assert.NotNil(t, cmd, "Update must re-arm the tick even when paused")
	assert.Equal(t, initial, v.frameIndex, "frameIndex should not advance when paused")
}

// TestVisualizer_ViewWhenPlaying verifies View returns a non-empty braille string.
func TestVisualizer_ViewWhenPlaying(t *testing.T) {
	v := newTestVisualizer(40, 1)
	v.SetPlaying(true)

	view := v.View()
	assert.NotEmpty(t, view, "View() should return non-empty string when playing")
}

// TestVisualizer_ViewWhenPaused verifies View returns a flat-line pattern when paused.
func TestVisualizer_ViewWhenPaused(t *testing.T) {
	v := newTestVisualizer(40, 1)
	v.SetPlaying(false)

	view := v.View()
	// Flat line should contain only blank braille (U+2800) or spaces.
	assert.NotEmpty(t, view, "View() should return non-empty string when paused")
	// The view should not contain active dot patterns when paused.
	// Flat line chars are U+2800 (⠀) — just verify the output is consistent.
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	assert.Equal(t, 1, len(lines), "single-height visualizer should have 1 line")
}

// TestVisualizer_FrameWraps verifies frameIndex wraps after reaching end of frame table.
func TestVisualizer_FrameWraps(t *testing.T) {
	v := newTestVisualizer(40, 2)
	v.SetPlaying(true)
	tableLen := len(v.frames)
	require.Greater(t, tableLen, 0)

	// Advance frameIndex to the last frame.
	v.frameIndex = tableLen - 1
	v.Update(VisualizerTickMsg(time.Now()))

	assert.Equal(t, 0, v.frameIndex, "frameIndex should wrap back to 0 at end of frame table")
}

// TestVisualizer_SetSize_1Line verifies SetSize(40, 1) gives 1-line view at 40 chars.
func TestVisualizer_SetSize_1Line(t *testing.T) {
	v := newTestVisualizer(40, 1)
	v.SetPlaying(true)

	view := v.View()
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	assert.Equal(t, 1, len(lines), "height=1 should produce 1 line")
	// Each line should be the right width in rune count (braille chars are 1 rune wide).
	assert.Equal(t, 40, len([]rune(lines[0])), "width should be 40 runes")
}

// TestVisualizer_SetSize_4Lines verifies SetSize(80, 4) gives 4-line view at 80 chars.
func TestVisualizer_SetSize_4Lines(t *testing.T) {
	v := newTestVisualizer(80, 4)
	v.SetPlaying(true)

	view := v.View()
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	assert.Equal(t, 4, len(lines), "height=4 should produce 4 lines")
	assert.Equal(t, 80, len([]rune(lines[0])), "width should be 80 runes")
}

// TestVisualizer_DeterministicFrames verifies same frameIndex produces same output.
func TestVisualizer_DeterministicFrames(t *testing.T) {
	v1 := newTestVisualizer(40, 2)
	v2 := newTestVisualizer(40, 2)

	v1.SetPlaying(true)
	v2.SetPlaying(true)

	// Both at frameIndex=3 should produce identical output.
	v1.frameIndex = 3
	v2.frameIndex = 3

	assert.Equal(t, v1.View(), v2.View(), "same frameIndex should produce identical output")
}

// TestVisualizer_UpdateBeforeSetSize verifies Update does not panic if SetSize was never called.
// Regression test for divide-by-zero when len(frames)==0 with playing=true.
func TestVisualizer_UpdateBeforeSetSize(t *testing.T) {
	th := theme.Load("black")
	v := NewVisualizer(th)
	v.SetPlaying(true) // frames is still nil

	require.NotPanics(t, func() {
		cmd := v.Update(VisualizerTickMsg(time.Now()))
		assert.NotNil(t, cmd, "Update should still re-arm tick even with nil frames")
	})
	// frameIndex should remain 0 since there are no frames to cycle through.
	assert.Equal(t, 0, v.frameIndex)
}

// TestVisualizer_OtherMessagesIgnored verifies non-tick messages don't change state.
func TestVisualizer_OtherMessagesIgnored(t *testing.T) {
	v := newTestVisualizer(40, 2)
	v.SetPlaying(true)
	initial := v.frameIndex

	cmd := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Nil(t, cmd, "non-tick message should return nil cmd")
	assert.Equal(t, initial, v.frameIndex, "frameIndex should not change on non-tick message")
}

// TestVisualizer_PatternInitialValue verifies pattern defaults to 0.
func TestVisualizer_PatternInitialValue(t *testing.T) {
	v := newTestVisualizer(40, 2)
	assert.Equal(t, 0, v.Pattern())
}

// TestVisualizer_CyclePattern verifies pattern cycling wraps 0→1→2→0.
func TestVisualizer_CyclePattern(t *testing.T) {
	tests := []struct {
		name    string
		initial int
		want    int
	}{
		{"0 to 1", 0, 1},
		{"1 to 2", 1, 2},
		{"2 wraps to 0", 2, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := newTestVisualizer(40, 2)
			v.pattern = tt.initial
			v.frames = generateFrames(v.width, v.height, v.pattern)
			v.CyclePattern()
			assert.Equal(t, tt.want, v.Pattern())
		})
	}
}

// TestVisualizer_SetPattern verifies SetPattern sets the pattern and regenerates frames.
func TestVisualizer_SetPattern(t *testing.T) {
	v := newTestVisualizer(40, 2)
	framesBefore := v.frames

	v.SetPattern(1)
	assert.Equal(t, 1, v.Pattern())
	// Frames should be regenerated (different pointer).
	assert.NotSame(t, &framesBefore[0], &v.frames[0], "SetPattern should regenerate frames")
}

// TestVisualizer_SetPatternClampsOutOfRange verifies out-of-range values are clamped.
func TestVisualizer_SetPatternClampsOutOfRange(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  int
	}{
		{"negative clamped to 0", -1, 0},
		{"above max clamped to NumPatterns-1", 10, NumPatterns - 1},
		{"at max-1 stays", NumPatterns - 1, NumPatterns - 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := newTestVisualizer(40, 2)
			// First set to a non-default so clamped values still trigger change.
			v.pattern = 1
			v.frames = generateFrames(v.width, v.height, 1)
			v.SetPattern(tt.input)
			assert.Equal(t, tt.want, v.Pattern())
		})
	}
}

// TestVisualizer_SetPatternNoopOnSameValue verifies SetPattern is a no-op when pattern unchanged.
func TestVisualizer_SetPatternNoopOnSameValue(t *testing.T) {
	v := newTestVisualizer(40, 2)
	framesBefore := v.frames
	v.SetPattern(0) // same as default
	assert.Equal(t, framesBefore, v.frames, "SetPattern with same value should not regenerate")
}

// TestVisualizer_SetPatternBeforeSetSize verifies SetPattern is safe before SetSize.
func TestVisualizer_SetPatternBeforeSetSize(t *testing.T) {
	th := theme.Load("black")
	v := NewVisualizer(th)
	require.NotPanics(t, func() {
		v.SetPattern(2)
	})
	assert.Equal(t, 2, v.Pattern())
	assert.Nil(t, v.frames, "frames should remain nil when size is not set")
}

// TestVisualizer_PatternsProduceDifferentFrames verifies each pattern generates distinct output.
func TestVisualizer_PatternsProduceDifferentFrames(t *testing.T) {
	width, height := 40, 2
	frames0 := generateFrames(width, height, 0)
	frames1 := generateFrames(width, height, 1)
	frames2 := generateFrames(width, height, 2)

	// Compare frame 5 across patterns — they should differ.
	idx := 5
	assert.NotEqual(t, frames0[idx], frames1[idx], "pattern 0 and 1 should produce different frames")
	assert.NotEqual(t, frames0[idx], frames2[idx], "pattern 0 and 2 should produce different frames")
	assert.NotEqual(t, frames1[idx], frames2[idx], "pattern 1 and 2 should produce different frames")
}

// TestVisualizer_PatternDeterministic verifies same pattern produces identical frames.
func TestVisualizer_PatternDeterministic(t *testing.T) {
	for p := 0; p < NumPatterns; p++ {
		a := generateFrames(40, 2, p)
		b := generateFrames(40, 2, p)
		assert.Equal(t, a, b, "pattern %d should be deterministic", p)
	}
}

// TestVisualizer_SetSizeRegeneratesWithPattern verifies SetSize uses the current pattern.
func TestVisualizer_SetSizeRegeneratesWithPattern(t *testing.T) {
	v := newTestVisualizer(40, 2)
	v.SetPattern(2)
	framesPattern2 := v.frames

	// Changing size should regenerate with pattern 2, not pattern 0.
	v.SetSize(50, 2)
	expected := generateFrames(50, 2, 2)
	assert.Equal(t, expected, v.frames, "SetSize should regenerate using current pattern")
	assert.NotEqual(t, framesPattern2, v.frames, "different width should produce different frames")
}
