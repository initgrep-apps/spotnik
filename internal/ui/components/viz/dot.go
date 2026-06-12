package viz

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// DotRenderer renders column heights as a standing wave of dot characters
// (separator, bullet, filled dot). Density varies by vertical profile and
// horizontal phase position.
type DotRenderer struct{}

func (r DotRenderer) MaxHeight(displayHeight int) int { return displayHeight }

// RenderFrameAt computes per-cell density from the standing wave formula
// and maps to dot characters. Implements FrameAwareRenderer.
func (r DotRenderer) RenderFrameAt(width, height, frameIdx int, colors []lipgloss.Color) Frame {
	if width <= 0 || height <= 0 {
		return Frame{}
	}

	m := uikit.ActiveMode()
	light := uikit.GlyphFor(uikit.GlyphSeparator, m)
	medium := uikit.GlyphFor(uikit.GlyphBullet, m)
	heavy := uikit.GlyphFor(uikit.GlyphFilledDot, m)

	phase := phaseFor(frameIdx)

	frame := make(Frame, height)
	for rowIdx := 0; rowIdx < height; rowIdx++ {
		vProfile := math.Abs(math.Sin(2 * math.Pi * float64(rowIdx) / float64(height)))
		var sb strings.Builder
		for col := 0; col < width; col++ {
			x := float64(col) / float64(width) * 2 * math.Pi
			hProfile := 0.5 + 0.5*math.Sin(x+phase)
			density := vProfile * hProfile

			switch {
			case density < 0.15:
				sb.WriteRune(' ')
			case density < 0.35:
				sb.WriteString(light)
			case density < 0.65:
				sb.WriteString(medium)
			default:
				sb.WriteString(heavy)
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

// RenderFrame is required by the Renderer interface but delegates to
// RenderFrameAt for consistency (frameIdx=0 as default).
func (r DotRenderer) RenderFrame(width, height int, colHeights []int, colors []lipgloss.Color) Frame {
	return r.RenderFrameAt(width, height, 0, colors)
}

var _ Renderer = DotRenderer{}
var _ FrameAwareRenderer = DotRenderer{}
