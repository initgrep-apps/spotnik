package viz

import (
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface check.
var _ Renderer = CharRenderer{}

func TestCharRenderer_BinaryChars(t *testing.T) {
	r := CharRenderer{Chars: []rune{' ', '0', '1'}, Scale: 1}
	colors := makeColors(4)
	colHeights := makeColHeights(10, 4)
	frame := r.RenderFrame(10, 4, colHeights, colors)

	for rowIdx, line := range frame {
		for _, ch := range line.Text {
			assert.True(t, ch == ' ' || ch == '0' || ch == '1',
				"row %d: unexpected character %U (%c), expected ' ', '0', or '1'", rowIdx, ch, ch)
		}
	}
}

func TestCharRenderer_FrameHeight(t *testing.T) {
	r := CharRenderer{Chars: []rune{' ', '0', '1'}, Scale: 1}
	colors := makeColors(5)
	frame := r.RenderFrame(10, 5, makeColHeights(10, 5), colors)
	assert.Len(t, frame, 5)
}

func TestCharRenderer_Width(t *testing.T) {
	r := CharRenderer{Chars: []rune{' ', '0', '1'}, Scale: 1}
	width := 12
	colors := makeColors(3)
	colHeights := makeColHeights(width, 3)
	frame := r.RenderFrame(width, 3, colHeights, colors)
	for _, line := range frame {
		assert.Equal(t, width, utf8.RuneCountInString(line.Text))
	}
}

func TestCharRenderer_ZeroWidth(t *testing.T) {
	r := CharRenderer{Chars: []rune{' ', '0', '1'}, Scale: 1}
	frame := r.RenderFrame(0, 3, nil, makeColors(3))
	assert.Empty(t, frame)
}

func TestCharRenderer_ZeroHeight(t *testing.T) {
	r := CharRenderer{Chars: []rune{' ', '0', '1'}, Scale: 1}
	frame := r.RenderFrame(10, 0, makeColHeights(10, 4), makeColors(0))
	assert.Empty(t, frame)
}

func TestCharRenderer_MaxHeight(t *testing.T) {
	tests := []struct {
		name          string
		renderer      CharRenderer
		displayHeight int
		want          int
	}{
		{name: "scale 1", renderer: CharRenderer{Chars: []rune{' ', '0', '1'}, Scale: 1}, displayHeight: 5, want: 5},
		{name: "scale 2", renderer: CharRenderer{Chars: []rune{' ', '0', '1'}, Scale: 2}, displayHeight: 5, want: 10},
		{name: "scale 0 defaults to 1", renderer: CharRenderer{Chars: []rune{' ', '0', '1'}, Scale: 0}, displayHeight: 5, want: 5},
		{name: "negative scale defaults to 1", renderer: CharRenderer{Chars: []rune{' ', '0', '1'}, Scale: -1}, displayHeight: 5, want: 5},
		{name: "zero display height", renderer: CharRenderer{Chars: []rune{' ', '0', '1'}, Scale: 1}, displayHeight: 0, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.renderer.MaxHeight(tt.displayHeight)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCharRenderer_ZeroHeights_AllSpaces(t *testing.T) {
	r := CharRenderer{Chars: []rune{' ', '0', '1'}, Scale: 1}
	width := 5
	height := 4
	colHeights := make([]int, width) // all zeros
	colors := makeColors(height)
	frame := r.RenderFrame(width, height, colHeights, colors)

	for rowIdx, line := range frame {
		for _, ch := range line.Text {
			assert.Equal(t, ' ', ch,
				"row %d: zero-height columns should produce only spaces", rowIdx)
		}
	}
}

func TestCharRenderer_ColorsAssigned(t *testing.T) {
	r := CharRenderer{Chars: []rune{' ', '0', '1'}, Scale: 1}
	colors := []lipgloss.Color{"#ff0000", "#00ff00", "#0000ff", "#ffffff"}
	colHeights := makeColHeights(6, 4)
	frame := r.RenderFrame(6, 4, colHeights, colors)
	for i, line := range frame {
		assert.Equal(t, colors[i], line.Color,
			"row %d: expected color %v, got %v", i, colors[i], line.Color)
	}
}

func TestCharRenderer_CharAlternation(t *testing.T) {
	// With Chars = {' ', '0', '1'}, Scale = 1, numVisible = 2.
	// Characters alternate: charIdx = ((rowIdx + col) % 2) + 1
	// At position (0,0): charIdx = 1 → '0'
	// At position (0,1): charIdx = 2 → '1'
	// At position (1,0): charIdx = 2 → '1'
	// At position (1,1): charIdx = 1 → '0'
	r := CharRenderer{Chars: []rune{' ', '0', '1'}, Scale: 1}
	width := 4
	height := 3
	colHeights := make([]int, width)
	for i := range colHeights {
		colHeights[i] = height // full height so all cells are filled
	}
	colors := makeColors(height)
	frame := r.RenderFrame(width, height, colHeights, colors)
	require.Len(t, frame, height)

	// Verify alternation: (rowIdx + col) % 2 == 0 → '0', (rowIdx + col) % 2 == 1 → '1'
	for rowIdx, line := range frame {
		runes := []rune(line.Text)
		for colIdx, ch := range runes {
			if (rowIdx+colIdx)%2 == 0 {
				assert.Equal(t, '0', ch,
					"row %d col %d: expected '0' for (row+col)%%2==0", rowIdx, colIdx)
			} else {
				assert.Equal(t, '1', ch,
					"row %d col %d: expected '1' for (row+col)%%2==1", rowIdx, colIdx)
			}
		}
	}
}

func TestCharRenderer_CharAlternation_TwoVisible(t *testing.T) {
	// With Chars = {' ', '0', '1', '2'}, Scale = 1, numVisible = 3.
	// Characters alternate: charIdx = ((rowIdx + col) % 3) + 1
	// So position (0,0) → charIdx = 1 → '0'
	//    position (0,1) → charIdx = 2 → '1'
	//    position (0,2) → charIdx = 3 → '2'
	//    position (0,3) → charIdx = 1 → '0'  (wraps)
	//    position (1,0) → charIdx = 2 → '1'
	r := CharRenderer{Chars: []rune{' ', '0', '1', '2'}, Scale: 1}
	width := 4
	height := 3
	colHeights := make([]int, width)
	for i := range colHeights {
		colHeights[i] = height // full height so all cells are filled
	}
	colors := makeColors(height)
	frame := r.RenderFrame(width, height, colHeights, colors)
	require.Len(t, frame, height)

	// Verify alternation pattern at specific positions
	// rowIdx=0, col=0 → ((0+0) % 3) + 1 = 1 → '0'
	assert.Equal(t, '0', []rune(frame[0].Text)[0])
	// rowIdx=0, col=1 → ((0+1) % 3) + 1 = 2 → '1'
	assert.Equal(t, '1', []rune(frame[0].Text)[1])
	// rowIdx=0, col=2 → ((0+2) % 3) + 1 = 3 → '2'
	assert.Equal(t, '2', []rune(frame[0].Text)[2])
	// rowIdx=0, col=3 → ((0+3) % 3) + 1 = 1 → '0' (wraps)
	assert.Equal(t, '0', []rune(frame[0].Text)[3])
	// rowIdx=1, col=0 → ((1+0) % 3) + 1 = 2 → '1'
	assert.Equal(t, '1', []rune(frame[1].Text)[0])
}

func TestCharRenderer_FullHeight_NoSpaces(t *testing.T) {
	r := CharRenderer{Chars: []rune{' ', '0', '1'}, Scale: 1}
	width := 5
	height := 4
	colHeights := make([]int, width)
	for i := range colHeights {
		colHeights[i] = height // full height, Scale=1
	}
	colors := makeColors(height)
	frame := r.RenderFrame(width, height, colHeights, colors)

	for rowIdx, line := range frame {
		for colIdx, ch := range line.Text {
			assert.NotEqual(t, ' ', ch,
				"row %d col %d: full height should have no spaces", rowIdx, colIdx)
		}
	}
}