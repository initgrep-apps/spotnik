package viz

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// CharRenderer renders column heights using a configurable character sequence.
// Characters are selected based on (row + col) position for visual variety.
// The first character in Chars must be a space (' ') — it represents the
// "empty" state when a column's height does not reach a given row.
//
// For Matrix Rain: Chars = []rune{' ', '0', '1'}
// The renderer alternates between '0' and '1' based on (row + col) % 2,
// creating a binary rain effect.
//
// Column heights are expected in the range [0, height * Scale].
type CharRenderer struct {
	Chars []rune
	Scale int
}

// MaxHeight returns the maximum height value for a given display height.
// CharRenderer uses Scale units per display row (1 by default).
func (r CharRenderer) MaxHeight(displayHeight int) int {
	if r.Scale <= 0 {
		return displayHeight
	}
	return displayHeight * r.Scale
}

// RenderFrame converts column heights to character display lines with per-row coloring.
// For each cell (row, col):
//   - If the column height does not reach this row, render space (Chars[0])
//   - Otherwise, render Chars[((row + col) % (len(Chars) - 1)) + 1]
//
// The +1 offset skips Chars[0] (space) which is used for empty cells.
// This produces alternating characters based on position for visual variety.
func (r CharRenderer) RenderFrame(width, height int, colHeights []int, colors []lipgloss.Color) Frame {
	if width <= 0 || height <= 0 {
		return Frame{}
	}
	if len(r.Chars) == 0 {
		return Frame{}
	}

	scale := r.Scale
	if scale <= 0 {
		scale = 1
	}
	// Number of non-space characters to cycle through
	numVisible := len(r.Chars) - 1
	if numVisible < 1 {
		numVisible = 1
	}

	frame := make(Frame, height)
	for rowIdx := 0; rowIdx < height; rowIdx++ {
		rowFromBottom := height - 1 - rowIdx
		var sb strings.Builder
		for col := 0; col < width; col++ {
			var h int
			if col < len(colHeights) {
				h = colHeights[col]
			}
			if h > rowFromBottom*scale && h > 0 {
				// Within the column's rain streak — pick alternating char
				charIdx := ((rowIdx + col) % numVisible) + 1
				if charIdx >= len(r.Chars) {
					charIdx = len(r.Chars) - 1
				}
				sb.WriteRune(r.Chars[charIdx])
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
