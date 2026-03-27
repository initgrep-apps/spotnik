package viz

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Renderer produces a Frame from column heights, dimensions, and per-row colors.
// Implementations may use different character sets (braille, block, etc.).
type Renderer interface {
	// RenderFrame converts colHeights (one value per column, range [0, height*4] for
	// braille or [0, height] for block) into a Frame of height StyledLines.
	// colors must have at least height elements; colors[i] applies to row i.
	// Returns an empty Frame when width or height is zero.
	RenderFrame(width, height int, colHeights []int, colors []lipgloss.Color) Frame
}

// BrailleRenderer renders column heights as Unicode braille characters (U+2800 block).
// Each character cell represents a 2×4 dot grid.
// Column heights are expected in the range [0, height*4].
type BrailleRenderer struct{}

// RenderFrame converts column heights to braille display lines with per-row coloring.
// Each row represents 4 dot-rows of the braille grid.
// Row 0 is the top; row (height-1) is the bottom.
func (r BrailleRenderer) RenderFrame(width, height int, colHeights []int, colors []lipgloss.Color) Frame {
	if width <= 0 || height <= 0 {
		return Frame{}
	}

	frame := make(Frame, height)
	for lineIdx := 0; lineIdx < height; lineIdx++ {
		// Bottom dot-row of this display line.
		// lineIdx 0 is the top display row, so its bottom dot-row is at the highest position.
		lineBottom := (height - 1 - lineIdx) * 4
		var sb strings.Builder
		for col := 0; col < width; col++ {
			var h int
			if col < len(colHeights) {
				h = colHeights[col]
			}
			filledInChar := h - lineBottom
			if filledInChar < 0 {
				filledInChar = 0
			}
			if filledInChar > 4 {
				filledInChar = 4
			}
			sb.WriteRune(brailleChar(filledInChar))
		}
		var color lipgloss.Color
		if lineIdx < len(colors) {
			color = colors[lineIdx]
		}
		frame[lineIdx] = StyledLine{Text: sb.String(), Color: color}
	}
	return frame
}

// brailleChar returns a Unicode braille character for a given fill level (0-4).
// The characters match the spec's height mapping exactly and produce a bar
// appearance when rendered in a terminal at each fill level.
//
// Braille dot layout: 1 4 / 2 5 / 3 6 / 7 8 (left|right per row).
// Bits: dot1=0x01, dot2=0x02, dot3=0x04, dot4=0x08, dot5=0x10, dot6=0x20, dot7=0x40, dot8=0x80.
//
//	0 filled: ⠀ (U+2800, offset 0x00) — blank
//	1 filled: ⡀ (U+2840, offset 0x40) — dot7 (left col, row 4)
//	2 filled: ⡠ (U+2860, offset 0x60) — dot6+dot7 (right col row 3 + left col row 4)
//	3 filled: ⡰ (U+2870, offset 0x70) — dot5+dot6+dot7 (right col rows 2-3 + left col row 4)
//	4 filled: ⣰ (U+28F0, offset 0xF0) — dot5+dot6+dot7+dot8 (bottom two rows filled)
//
// NOTE: These codepoints match the spec exactly. The fill pattern spans both
// columns for a wider visual bar effect rather than a single-column fill.
func brailleChar(filledDots int) rune {
	switch filledDots {
	case 0:
		return '\u2800' // ⠀ blank
	case 1:
		return '\u2840' // ⡀ dot7 (0x40)
	case 2:
		return '\u2860' // ⡠ dot6+dot7 (0x20|0x40=0x60)
	case 3:
		return '\u2870' // ⡰ dot5+dot6+dot7 (0x10|0x20|0x40=0x70)
	default: // 4+
		return '\u28F0' // ⣰ dot5+dot6+dot7+dot8 (0x10|0x20|0x40|0x80=0xF0)
	}
}
