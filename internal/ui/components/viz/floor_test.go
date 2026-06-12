package viz

import (
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ Renderer = FloorRenderer{}

func TestFloorRenderer_FrameHeight(t *testing.T) {
	r := FloorRenderer{}
	colors := makeColors(3)
	frame := r.RenderFrame(10, 3, makeColHeights(10, 6), colors)
	assert.Len(t, frame, 3)
}

func TestFloorRenderer_Width(t *testing.T) {
	r := FloorRenderer{}
	width := 15
	colors := makeColors(2)
	colHeights := makeColHeights(width, 4)
	frame := r.RenderFrame(width, 2, colHeights, colors)
	for _, line := range frame {
		assert.Equal(t, width, utf8.RuneCountInString(line.Text))
	}
}

func TestFloorRenderer_ZeroWidth(t *testing.T) {
	r := FloorRenderer{}
	frame := r.RenderFrame(0, 3, nil, makeColors(3))
	assert.Empty(t, frame)
}

func TestFloorRenderer_ZeroHeight(t *testing.T) {
	r := FloorRenderer{}
	frame := r.RenderFrame(10, 0, makeColHeights(10, 4), makeColors(0))
	assert.Empty(t, frame)
}

func TestFloorRenderer_ColorsAssigned(t *testing.T) {
	r := FloorRenderer{}
	colors := []lipgloss.Color{"#ff0000", "#00ff00", "#0000ff"}
	frame := r.RenderFrame(10, 3, makeColHeights(10, 6), colors)
	for i, line := range frame {
		assert.Equal(t, colors[i], line.Color)
	}
}

func TestFloorRenderer_MaxHeight(t *testing.T) {
	r := FloorRenderer{}
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
			"FloorRenderer.MaxHeight(%d) should be %d", tt.displayHeight, tt.want)
	}
}

func TestFloorRenderer_ColorSegments(t *testing.T) {
	r := FloorRenderer{}
	width := 10
	height := 6
	colHeights := makeColHeights(width, 6)
	// Use varied colors so hue shift creates segment boundaries
	colors := []lipgloss.Color{
		"#ff0000", "#00ff00", "#0000ff", "#ffff00", "#ff00ff", "#00ffff",
	}
	frame := r.RenderFrame(width, height, colHeights, colors)
	require.Len(t, frame, height)

	hasMultiSegment := false
	for _, line := range frame {
		if len(line.Segments) > 1 {
			hasMultiSegment = true
			break
		}
	}
	assert.True(t, hasMultiSegment,
		"at least one row should have multiple color segments due to per-column hue shift")
}

func TestFloorRenderer_FloorPresence(t *testing.T) {
	// Verify that bottom rows are always filled (floor) regardless of colHeights.
	r := FloorRenderer{}
	width := 10
	height := 8
	// Zero colHeights means only floor should be filled
	colHeights := make([]int, width)
	colors := makeColors(height)
	frame := r.RenderFrame(width, height, colHeights, colors)
	require.Len(t, frame, height)

	floorHeight := height / 4
	if floorHeight < 1 {
		floorHeight = 1
	}

	fillGlyph := uikit.GlyphFor(uikit.GlyphBarFull, uikit.ActiveMode())

	// Bottom floorHeight rows should be entirely filled
	for rowIdx := height - floorHeight; rowIdx < height; rowIdx++ {
		for _, ch := range frame[rowIdx].Text {
			assert.Equal(t, []rune(fillGlyph)[0], ch,
				"floor row %d should be filled", rowIdx)
		}
	}
}
