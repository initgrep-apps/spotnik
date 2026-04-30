package viz

import "github.com/charmbracelet/lipgloss"

// AsciiBarsRenderer renders column heights as ASCII bar characters.
// It uses four levels of visual fill density:
//
//	'#' — fully filled (bar reaches this row)
//	'=' — heavily filled (bar just below this row)
//	'.' — lightly filled (bar well below this row)
//	' ' — empty
//
// MaxHeight always returns 4 so HeightFunc values are in [0, 4].
// The renderer maps those four integer levels to the four character levels
// across any display height, making it suitable for low-resolution ASCII
// terminals where braille and block-element characters are unavailable.
type AsciiBarsRenderer struct{}

// NewAsciiBarsRenderer creates a new AsciiBarsRenderer.
func NewAsciiBarsRenderer() *AsciiBarsRenderer { return &AsciiBarsRenderer{} }

// MaxHeight returns the fixed maximum height value for the ASCII renderer.
// Unlike BrailleRenderer (height×4) or BlockRenderer (height), AsciiBarsRenderer
// always returns 4 regardless of display height. This gives HeightFunc a
// stable [0, 4] integer range while the renderer itself handles the mapping
// from those four levels to any number of display rows.
func (r *AsciiBarsRenderer) MaxHeight(_ int) int { return 4 }

// RenderFrame converts column heights (range [0, 4]) to ASCII bar display lines
// with per-row coloring. Row 0 is the top; row (height-1) is the bottom.
//
// Because MaxHeight always returns 4, column heights are integers in [0, 4].
// The mapping uses direct integer arithmetic (no fractional bands) so all four
// glyph levels are reachable at every display height:
//
//   - r = height - 1 - rowIdx  (0 = bottom row, height-1 = top row)
//   - h - r >= 1          → '#'  (row is inside the filled bar)
//   - h - r == 0, h > 0   → '='  (row is exactly at the top edge of bar)
//   - h - r == -1, h > 0  → '.'  (row is one step above the bar top)
//   - otherwise            → ' '  (empty)
//
// The h > 0 guard ensures zero-height columns remain blank.
func (r *AsciiBarsRenderer) RenderFrame(width, height int, colHeights []int, colors []lipgloss.Color) Frame {
	if width <= 0 || height <= 0 {
		return Frame{}
	}

	frame := make(Frame, height)

	for rowIdx := 0; rowIdx < height; rowIdx++ {
		// r is the bottom-up index: bottom row is 0, top row is height-1.
		rowFromBottom := height - 1 - rowIdx

		row := make([]byte, width)
		for col := 0; col < width; col++ {
			var h int
			if col < len(colHeights) {
				h = colHeights[col]
			}
			diff := h - rowFromBottom

			switch {
			case diff >= 1:
				row[col] = '#'
			case diff == 0 && h > 0:
				row[col] = '='
			case diff == -1 && h > 0:
				row[col] = '.'
			default:
				row[col] = ' '
			}
		}

		var color lipgloss.Color
		if rowIdx < len(colors) {
			color = colors[rowIdx]
		}
		frame[rowIdx] = StyledLine{Text: string(row), Color: color}
	}
	return frame
}
