package viz

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// BrailleMirrorRenderer renders column heights as a double-lobe standing wave
// using braille density characters (⣿⢸⢰⠄⠂). The center band is always
// filled, with lobe thickness determined by column heights.
type BrailleMirrorRenderer struct{}

func (r BrailleMirrorRenderer) MaxHeight(displayHeight int) int { return displayHeight }

func (r BrailleMirrorRenderer) RenderFrame(width, height int, colHeights []int, colors []lipgloss.Color) Frame {
	if width <= 0 || height <= 0 {
		return Frame{}
	}

	center := float64(height-1) / 2.0
	centerBand := 1

	frame := make(Frame, height)
	for rowIdx := 0; rowIdx < height; rowIdx++ {
		var sb strings.Builder
		for col := 0; col < width; col++ {
			var h int
			if col < len(colHeights) {
				h = colHeights[col]
			}
			lobeThickness := float64(h)
			centerDist := math.Abs(float64(rowIdx) - center)

			if centerDist <= float64(centerBand) {
				sb.WriteRune(brailleForDensity(1.0))
			} else if centerDist <= lobeThickness {
				denom := lobeThickness - float64(centerBand)
				var relativeDist float64
				if denom > 0 {
					relativeDist = (centerDist - float64(centerBand)) / denom
				} else {
					relativeDist = 0
				}
				sb.WriteRune(brailleForDensity(1.0 - relativeDist))
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

func brailleForDensity(d float64) rune {
	if math.IsNaN(d) || math.IsInf(d, 0) {
		d = 0
	}
	switch {
	case d >= 0.85:
		return '⣿'
	case d >= 0.65:
		return '⢸'
	case d >= 0.45:
		return '⢰'
	case d >= 0.25:
		return '⠄'
	case d >= 0.10:
		return '⠂'
	default:
		return '⠀'
	}
}

var _ Renderer = BrailleMirrorRenderer{}
