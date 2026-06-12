package viz

import (
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/uikit"
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
	require.Len(t, frame, 3)
	// Color remapping: center gets G1 (high index), edges get G7 (low index)
	// center=(3-1)/2=1.0: row0 dist=1 zoneIdx=(2-1*2)=0 -> colors[0]
	//                      row1 dist=0 zoneIdx=(2-0)=2 -> colors[2]
	//                      row2 dist=1 zoneIdx=(2-1*2)=0 -> colors[0]
	assert.Equal(t, colors[0], frame[0].Color)
	assert.Equal(t, colors[2], frame[1].Color)
	assert.Equal(t, colors[0], frame[2].Color)
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

func TestBrailleMirrorRenderer_DiscreteDistanceMapping(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	r := BrailleMirrorRenderer{}
	width := 3
	height := 11 // center at row 5
	colHeights := []int{6, 6, 6} // large lobe thickness
	colors := makeColors(height)
	frame := r.RenderFrame(width, height, colHeights, colors)
	require.Len(t, frame, height)

	// Center row (dist=0): should have ⣿
	centerRow := frame[5]
	assert.Contains(t, centerRow.Text, "⣿",
		"center row should contain full solid braille character")

	// Row at dist=1 (row 4 or 6): should have ⢸ (not ⣿)
	rowDist1 := frame[4]
	assert.NotContains(t, rowDist1.Text, "⣿",
		"row at dist=1 should NOT contain full solid character")
	assert.Contains(t, rowDist1.Text, "⢸",
		"row at dist=1 should contain dense braille character")
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
