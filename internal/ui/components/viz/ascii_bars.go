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
// Mapping from column height to character at each display row:
//   - A column of height h fills from the bottom upward. The fractional fill
//     of row rowIdx from the top is: filledFrac = h/4.0, and the row's threshold
//     from the bottom is: rowFrac = (height - rowIdx) / height.
//   - If filledFrac >= rowFrac          → '#' (fully covered)
//   - If filledFrac >= rowFrac - 1/4H   → '=' (partially covered, upper band)
//   - If filledFrac >= rowFrac - 2/4H   → '.' (lightly covered, lower band)
//   - Otherwise                          → ' ' (empty)
func (r *AsciiBarsRenderer) RenderFrame(width, height int, colHeights []int, colors []lipgloss.Color) Frame {
	if width <= 0 || height <= 0 {
		return Frame{}
	}

	frame := make(Frame, height)
	hf := float64(height)
	// band is the fractional height of one sub-level step, constant for this render.
	band := 1.0 / (4.0 * hf)

	for rowIdx := 0; rowIdx < height; rowIdx++ {
		// rowFrac is the normalised threshold from the bottom for this row.
		// Row 0 (top) has the highest threshold; row height-1 (bottom) has
		// threshold = 1/height (just above zero).
		rowFrac := float64(height-rowIdx) / hf

		row := make([]byte, width)
		for col := 0; col < width; col++ {
			var h int
			if col < len(colHeights) {
				h = colHeights[col]
			}
			filledFrac := float64(h) / 4.0

			switch {
			case filledFrac >= rowFrac:
				row[col] = '#'
			case filledFrac >= rowFrac-band:
				row[col] = '='
			case filledFrac >= rowFrac-2*band:
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
