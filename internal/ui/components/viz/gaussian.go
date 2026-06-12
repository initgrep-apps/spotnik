package viz

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// GaussianRenderer renders column heights as a centered Gaussian density wave
// using block character density bands. Bars are thickest at center and taper
// toward edges using fill/heavy/medium/empty glyph levels.
type GaussianRenderer struct{}

func (r GaussianRenderer) MaxHeight(displayHeight int) int { return displayHeight }

func (r GaussianRenderer) RenderFrame(width, height int, colHeights []int, colors []lipgloss.Color) Frame {
	if width <= 0 || height <= 0 {
		return Frame{}
	}
	m := uikit.ActiveMode()
	fillGlyph := uikit.GlyphFor(uikit.GlyphBarFull, m)
	heavyGlyph := uikit.GlyphFor(uikit.GlyphBarHeavy, m)
	mediumGlyph := uikit.GlyphFor(uikit.GlyphBarMedium, m)
	emptyGlyph := uikit.GlyphFor(uikit.GlyphBarEmpty, m)

	center := float64(height-1) / 2.0
	frame := make(Frame, height)

	for rowIdx := 0; rowIdx < height; rowIdx++ {
		distFromCenter := math.Abs(float64(rowIdx) - center)
		var sb strings.Builder
		for col := 0; col < width; col++ {
			var h int
			if col < len(colHeights) {
				h = colHeights[col]
			}
			waveReach := float64(h) / 2.0
			if distFromCenter <= waveReach {
				relativeDist := distFromCenter / waveReach
				switch {
				case relativeDist < 0.25:
					sb.WriteString(fillGlyph)
				case relativeDist < 0.5:
					sb.WriteString(heavyGlyph)
				case relativeDist < 0.75:
					sb.WriteString(mediumGlyph)
				default:
					sb.WriteString(emptyGlyph)
				}
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

var _ Renderer = GaussianRenderer{}
