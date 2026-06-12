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

	// Remap colors: center gets G1, edges get G7
	// colors is built top-to-bottom (index 0=VizGradient7, index height-1=VizGradient1)
	// so invert the index: center→high index(G1), edges→low index(G7)
	remapped := make([]lipgloss.Color, height)
	for i := 0; i < height; i++ {
		dist := int(math.Abs(float64(i) - center))
		zoneIdx := (height - 1) - dist*2
		if zoneIdx < 0 {
			zoneIdx = 0
		}
		if zoneIdx < len(colors) {
			remapped[i] = colors[zoneIdx]
		} else if len(colors) > 0 {
			remapped[i] = colors[len(colors)-1]
		}
	}

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

			if centerDist <= lobeThickness {
				sb.WriteRune(brailleCharForDist(centerDist))
			} else {
				sb.WriteRune(' ')
			}
		}
		frame[rowIdx] = StyledLine{Text: sb.String(), Color: remapped[rowIdx]}
	}
	return frame
}

// brailleCharForDist returns a braille character based on absolute
// distance from center, matching the spec's discrete mapping table.
//
// dist=0: full solid (⣿)
// dist=1: dense (⢸⡇, 3 dots)
// dist=2: medium (⢰⡆, 3 dots, different pattern)
// dist=3: light (⠄⠄, 1-2 dots)
// dist=4: very light (⠂ or ⠄, isolated dots)
// dist>=5: sparse edge (⢠⡄ or ⠠⠤, 2 dots decorative)
func brailleCharForDist(dist float64) rune {
	d := int(math.Round(dist))
	switch {
	case d == 0:
		return '⣿' // full solid — center peak
	case d == 1:
		return '⢸' // dense — 3 dots
	case d == 2:
		return '⢰' // medium — 2-3 dots
	case d == 3:
		return '⠄' // light — 1 dot
	case d == 4:
		return '⠂' // very light — isolated dot
	default:
		return '⢀' // sparse edge — 1 dot decorative
	}
}

var _ Renderer = BrailleMirrorRenderer{}
