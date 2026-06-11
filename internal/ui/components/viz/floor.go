package viz

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

type FloorRenderer struct{}

func (r FloorRenderer) MaxHeight(displayHeight int) int { return displayHeight }

func (r FloorRenderer) RenderFrame(width, height int, colHeights []int, colors []lipgloss.Color) Frame {
	if width <= 0 || height <= 0 {
		return Frame{}
	}
	m := uikit.ActiveMode()
	fillGlyph := uikit.GlyphFor(uikit.GlyphBarFull, m)

	floorHeight := height / 4
	if floorHeight < 1 {
		floorHeight = 1
	}

	frame := make(Frame, height)
	for rowIdx := 0; rowIdx < height; rowIdx++ {
		rowFromBottom := height - 1 - rowIdx
		var sb strings.Builder
		for col := 0; col < width; col++ {
			var h int
			if col < len(colHeights) {
				h = colHeights[col]
			}
			if rowFromBottom < floorHeight {
				sb.WriteString(fillGlyph)
			} else if rowFromBottom < h {
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

var _ Renderer = FloorRenderer{}
