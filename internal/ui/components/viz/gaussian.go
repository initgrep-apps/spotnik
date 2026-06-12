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
				switch {
				case distFromCenter < 1:
					sb.WriteString(fillGlyph)
				case distFromCenter < 2:
					sb.WriteString(heavyGlyph)
				case distFromCenter < 3:
					sb.WriteString(mediumGlyph)
				case distFromCenter < 4:
					sb.WriteString(emptyGlyph)
				default:
					sb.WriteRune(' ')
				}
			} else {
				sb.WriteRune(' ')
			}
		}
		// Build Segments for per-column hue shift
		var segments []StyledSegment
		var segBuf strings.Builder
		var prevColor lipgloss.Color
		lineText := sb.String()
		runes := []rune(lineText)
		for col := 0; col < width; col++ {
			zoneStep := 2
			shift := (col * zoneStep) % height
			shiftedRow := (rowIdx + shift) % height
			var colColor lipgloss.Color
			if shiftedRow < len(colors) {
				colColor = colors[shiftedRow]
			}
			if colColor != prevColor && segBuf.Len() > 0 {
				segments = append(segments, StyledSegment{Text: segBuf.String(), Color: prevColor})
				segBuf.Reset()
			}
			prevColor = colColor
			segBuf.WriteString(string(runes[col]))
		}
		if segBuf.Len() > 0 {
			segments = append(segments, StyledSegment{Text: segBuf.String(), Color: prevColor})
		}

		var lineColor lipgloss.Color
		if rowIdx < len(colors) {
			lineColor = colors[rowIdx]
		}
		frame[rowIdx] = StyledLine{Text: lineText, Color: lineColor, Segments: segments}
	}
	return frame
}

var _ Renderer = GaussianRenderer{}
