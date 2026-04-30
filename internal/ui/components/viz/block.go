package viz

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// BlockRenderer renders column heights as block characters (█ and space).
// Visually heavier and coarser than braille.
// Column heights are expected in the range [0, height].
//
// NOTE: Block character `█` (U+2588) may render as 2 columns in some East Asian
// terminal configurations. This is a known limitation, not something we solve for.
type BlockRenderer struct{}

// MaxHeight returns the maximum height value for a given display height.
// Block renderer uses one unit per display row.
func (r BlockRenderer) MaxHeight(displayHeight int) int {
	return displayHeight
}

// RenderFrame converts column heights to block display lines with per-row coloring.
// Row 0 is the top; row (height-1) is the bottom.
// A column is filled (GlyphBarFull) for all rows at or below its height from the bottom.
// The fill glyph is resolved via uikit.ActiveMode() so ASCII mode uses '#' instead of '█'.
func (r BlockRenderer) RenderFrame(width, height int, colHeights []int, colors []lipgloss.Color) Frame {
	if width <= 0 || height <= 0 {
		return Frame{}
	}

	// Resolve the fill glyph once outside the inner loops to avoid redundant calls.
	m := uikit.ActiveMode()
	fillGlyph := uikit.GlyphFor(uikit.GlyphBarFull, m)

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
				sb.WriteString(fillGlyph)
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
