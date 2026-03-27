package viz

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// BlockRenderer renders column heights as block characters (█ and space).
// Visually heavier and coarser than braille.
// Column heights are expected in the range [0, height].
//
// NOTE: Block character `█` (U+2588) may render as 2 columns in some East Asian
// terminal configurations. This is a known limitation, not something we solve for.
type BlockRenderer struct{}

// RenderFrame converts column heights to block display lines with per-row coloring.
// Row 0 is the top; row (height-1) is the bottom.
// A column is filled (█) for all rows at or below its height from the bottom.
func (r BlockRenderer) RenderFrame(width, height int, colHeights []int, colors []lipgloss.Color) Frame {
	if width <= 0 || height <= 0 {
		return Frame{}
	}

	frame := make(Frame, height)
	for rowIdx := 0; rowIdx < height; rowIdx++ {
		// rowIdx 0 = top row; rowIdx (height-1) = bottom row.
		// A column fills rows from the bottom up to its height.
		// The display row's distance from the bottom is (height - 1 - rowIdx).
		rowFromBottom := height - 1 - rowIdx
		var sb strings.Builder
		for col := 0; col < width; col++ {
			var h int
			if col < len(colHeights) {
				h = colHeights[col]
			}
			// Fill this cell if the column height reaches above this row.
			if h > rowFromBottom {
				sb.WriteRune('█')
			} else {
				sb.WriteRune(' ')
			}
		}
		var color lipgloss.Color
		if rowIdx < len(colors) {
			color = colors[rowIdx]
		}
		frame[rowIdx] = StyledLine{Text: sb.String(), Color: color}
	}
	return frame
}
