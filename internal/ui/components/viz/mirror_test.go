package viz

import (
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ Renderer = BrailleMirrorRenderer{}

func TestBrailleMirrorRenderer_FrameHeight(t *testing.T) {
	r := BrailleMirrorRenderer{}
	colors := makeColors(3)
	frame := r.RenderFrame(10, 3, makeColHeights(10, 6), colors)
	assert.Len(t, frame, 3)
}

func TestBrailleMirrorRenderer_Width(t *testing.T) {
	r := BrailleMirrorRenderer{}
	width := 15
	colors := makeColors(2)
	colHeights := makeColHeights(width, 4)
	frame := r.RenderFrame(width, 2, colHeights, colors)
	for _, line := range frame {
		assert.Equal(t, width, utf8.RuneCountInString(line.Text))
	}
}

func TestBrailleMirrorRenderer_ZeroWidth(t *testing.T) {
	r := BrailleMirrorRenderer{}
	frame := r.RenderFrame(0, 3, nil, makeColors(3))
	assert.Empty(t, frame)
}

func TestBrailleMirrorRenderer_ZeroHeight(t *testing.T) {
	r := BrailleMirrorRenderer{}
	frame := r.RenderFrame(10, 0, makeColHeights(10, 4), makeColors(0))
	assert.Empty(t, frame)
}

func TestBrailleMirrorRenderer_ColorsAssigned(t *testing.T) {
	r := BrailleMirrorRenderer{}
	colors := []lipgloss.Color{"#ff0000", "#00ff00", "#0000ff"}
	frame := r.RenderFrame(10, 3, makeColHeights(10, 6), colors)
	for i, line := range frame {
		assert.Equal(t, colors[i], line.Color)
	}
}

func TestBrailleMirrorRenderer_MaxHeight(t *testing.T) {
	r := BrailleMirrorRenderer{}
	tests := []struct {
		displayHeight int
		want          int
	}{
		{displayHeight: 0, want: 0},
		{displayHeight: 1, want: 1},
		{displayHeight: 4, want: 4},
		{displayHeight: 10, want: 10},
	}
	for _, tt := range tests {
		got := r.MaxHeight(tt.displayHeight)
		assert.Equal(t, tt.want, got,
			"BrailleMirrorRenderer.MaxHeight(%d) should be %d", tt.displayHeight, tt.want)
	}
}

func TestBrailleMirrorRenderer_DensityFalloff(t *testing.T) {
	// With symmetric colHeights, verify that center rows are denser than edge rows.
	r := BrailleMirrorRenderer{}
	width := 10
	height := 9
	// Large lobe thickness so every row is inside a lobe
	colHeights := make([]int, width)
	for i := range colHeights {
		colHeights[i] = height
	}
	colors := makeColors(height)
	frame := r.RenderFrame(width, height, colHeights, colors)
	require.Len(t, frame, height)

	// Count non-space characters per row
	countNonSpace := func(text string) int {
		count := 0
		for _, ch := range text {
			if ch != ' ' {
				count++
			}
		}
		return count
	}

	center := height / 2
	centerCount := countNonSpace(frame[center].Text)
	edgeCount := countNonSpace(frame[0].Text)

	assert.GreaterOrEqual(t, centerCount, edgeCount,
		"center row should have at least as many non-space chars as edge row")
}
