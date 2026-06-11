package viz

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// SpectrumRenderer renders block bars with per-column gradient shifting.
// It produces StyledLines with Segments populated so that each column zone
// can have a different color, creating a diagonal G3→G2→G1 sweep across
// the visualizer width.
//
// The color shift per column is determined by:
//
//	shift = (col * 2) % height
//
// This means column 0 uses standard G3/G2/G1 row boundaries, column 1 shifts
// them down by 2 rows, column 2 by 4, etc. The result is a diagonal color
// sweep from upper-right to lower-left.
type SpectrumRenderer struct{}

// MaxHeight returns the maximum height value for a given display height.
// SpectrumRenderer uses one unit per display row (same as BlockRenderer).
func (r SpectrumRenderer) MaxHeight(displayHeight int) int {
	return displayHeight
}

// RenderFrame converts column heights to block display lines with per-column
// color shifting. Each row is broken into segments of contiguous columns that
// share the same gradient zone after applying the column shift.
func (r SpectrumRenderer) RenderFrame(width, height int, colHeights []int, colors []lipgloss.Color) Frame {
	if width <= 0 || height <= 0 {
		return Frame{}
	}

	m := uikit.ActiveMode()
	fillGlyph := uikit.GlyphFor(uikit.GlyphBarFull, m)

	frame := make(Frame, height)
	for rowIdx := 0; rowIdx < height; rowIdx++ {
		rowFromBottom := height - 1 - rowIdx

		// Build segments: group contiguous columns that share the same shifted color
		var segments []StyledSegment
		var segBuf strings.Builder
		var prevColor lipgloss.Color

		for col := 0; col < width; col++ {
			// Compute shifted color for this column
			shift := (col * 2) % height
			effectiveRow := (rowIdx + shift) % height
			var colColor lipgloss.Color
			if effectiveRow < len(colors) {
				colColor = colors[effectiveRow]
			} else if len(colors) > 0 {
				colColor = colors[len(colors)-1]
			}

			// Determine character for this column
			var h int
			if col < len(colHeights) {
				h = colHeights[col]
			}
			ch := ' '
			if h > rowFromBottom {
				ch = []rune(fillGlyph)[0]
			}

			// Start new segment if color changed
			if segBuf.Len() == 0 {
				prevColor = colColor
				segBuf.WriteRune(ch)
			} else if colColor != prevColor {
				segments = append(segments, StyledSegment{Text: segBuf.String(), Color: prevColor})
				segBuf.Reset()
				prevColor = colColor
				segBuf.WriteRune(ch)
			} else {
				segBuf.WriteRune(ch)
			}
		}

		// Flush last segment
		if segBuf.Len() > 0 {
			segments = append(segments, StyledSegment{Text: segBuf.String(), Color: prevColor})
		}

		// Build full text from segments for backward compat
		var fullText strings.Builder
		for _, seg := range segments {
			fullText.WriteString(seg.Text)
		}

		var lineColor lipgloss.Color
		if rowIdx < len(colors) {
			lineColor = colors[rowIdx]
		}
		frame[rowIdx] = StyledLine{Text: fullText.String(), Color: lineColor, Segments: segments}
	}
	return frame
}
