package viz

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// DensityRenderer renders column heights as block characters with a density
// halo above each bar top. The halo uses ░ ▒ ▓ characters (uikit GlyphBarEmpty,
// GlyphBarMedium, GlyphBarHeavy) at 3, 2, and 1 rows above the bar respectively.
// The bar itself and everything below uses █ (GlyphBarFull).
//
// This creates the classic 90s Winamp equalizer look where bar tops have a
// soft glow fade rather than a hard edge.
//
// Column heights are expected in the range [0, height] (same as BlockRenderer).
type DensityRenderer struct{}

// MaxHeight returns the maximum height value for a given display height.
// DensityRenderer uses one unit per display row (same scale as BlockRenderer).
func (r DensityRenderer) MaxHeight(displayHeight int) int {
	return displayHeight
}

// RenderFrame converts column heights to density-halo display lines with per-row coloring.
// Row 0 is the top; row (height-1) is the bottom.
//
// The rendering uses a halo depth of 3 rows above each bar top:
//
//	3 rows above bar → ░ (GlyphBarEmpty)
//	2 rows above bar → ▒ (GlyphBarMedium)
//	1 row above bar  → ▓ (GlyphBarHeavy)
//	At or below bar  → █ (GlyphBarFull)
//	More than 3 above → space
//
// When multiple bars overlap, the densest glyph wins (█ > ▓ > ▒ > ░ > space).
func (r DensityRenderer) RenderFrame(width, height int, colHeights []int, colors []lipgloss.Color) Frame {
	if width <= 0 || height <= 0 {
		return Frame{}
	}

	m := uikit.ActiveMode()
	fillGlyph := uikit.GlyphFor(uikit.GlyphBarFull, m)
	heavyGlyph := uikit.GlyphFor(uikit.GlyphBarHeavy, m)
	mediumGlyph := uikit.GlyphFor(uikit.GlyphBarMedium, m)
	emptyGlyph := uikit.GlyphFor(uikit.GlyphBarEmpty, m)

	frame := make(Frame, height)
	for rowIdx := 0; rowIdx < height; rowIdx++ {
		rowFromBottom := height - 1 - rowIdx
		var sb strings.Builder
		for col := 0; col < width; col++ {
			var h int
			if col < len(colHeights) {
				h = colHeights[col]
			}
			// Zero-height columns are entirely spaces — no bar, no halo.
			if h == 0 {
				sb.WriteRune(' ')
				continue
			}
			distAbove := rowFromBottom - h
			switch {
			case distAbove < 0:
				// Below bar top: fill
				sb.WriteString(fillGlyph)
			case distAbove == 0:
				// At bar top: fill
				sb.WriteString(fillGlyph)
			case distAbove == 1:
				sb.WriteString(heavyGlyph)
			case distAbove == 2:
				sb.WriteString(mediumGlyph)
			case distAbove == 3:
				sb.WriteString(emptyGlyph)
			default:
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
