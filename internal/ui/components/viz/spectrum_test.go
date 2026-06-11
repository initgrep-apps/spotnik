package viz

import (
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface check.
var _ Renderer = SpectrumRenderer{}

// buildColorsFromTheme constructs per-row colors using the theme gradient.
// Top 1/3: Gradient3 (hot/peaks), middle 1/3: Gradient2 (warm),
// bottom 1/3: Gradient1 (cool/base). Extra rows go to the bottom.
func buildColorsFromTheme(th theme.Theme, height int) []lipgloss.Color {
	colors := make([]lipgloss.Color, height)
	third := height / 3
	for i := 0; i < height; i++ {
		switch {
		case i < third:
			colors[i] = th.Gradient3()
		case i < 2*third:
			colors[i] = th.Gradient2()
		default:
			colors[i] = th.Gradient1()
		}
	}
	return colors
}

func TestSpectrumRenderer_FrameHeight(t *testing.T) {
	r := SpectrumRenderer{}
	colors := makeColors(4)
	frame := r.RenderFrame(10, 4, makeColHeights(10, 4), colors)
	assert.Len(t, frame, 4)
}

func TestSpectrumRenderer_Width(t *testing.T) {
	r := SpectrumRenderer{}
	width := 12
	colors := makeColors(3)
	colHeights := makeColHeights(width, 3)
	frame := r.RenderFrame(width, 3, colHeights, colors)
	for _, line := range frame {
		assert.Equal(t, width, utf8.RuneCountInString(line.Text))
	}
}

func TestSpectrumRenderer_ZeroWidth(t *testing.T) {
	r := SpectrumRenderer{}
	frame := r.RenderFrame(0, 3, nil, makeColors(3))
	assert.Empty(t, frame)
}

func TestSpectrumRenderer_ZeroHeight(t *testing.T) {
	r := SpectrumRenderer{}
	frame := r.RenderFrame(10, 0, makeColHeights(10, 4), makeColors(0))
	assert.Empty(t, frame)
}

func TestSpectrumRenderer_MaxHeight(t *testing.T) {
	r := SpectrumRenderer{}
	tests := []struct {
		displayHeight int
		want          int
	}{
		{displayHeight: 0, want: 0},
		{displayHeight: 1, want: 1},
		{displayHeight: 4, want: 4},
		{displayHeight: 10, want: 10},
	}
	for _, tt := range tests {
		got := r.MaxHeight(tt.displayHeight)
		assert.Equal(t, tt.want, got,
			"SpectrumRenderer.MaxHeight(%d) should be %d", tt.displayHeight, tt.want)
	}
}

func TestSpectrumRenderer_SegmentsPopulated(t *testing.T) {
	r := SpectrumRenderer{}
	th := theme.Load("black")
	colors := buildColorsFromTheme(th, 6)
	colHeights := makeColHeights(10, 6)
	frame := r.RenderFrame(10, 6, colHeights, colors)
	require.Len(t, frame, 6, "frame should have height rows")
	for rowIdx, line := range frame {
		assert.NotEmpty(t, line.Segments,
			"row %d should have non-empty Segments", rowIdx)
	}
}

func TestSpectrumRenderer_DifferentColorsAcrossColumns(t *testing.T) {
	r := SpectrumRenderer{}
	th := theme.Load("black")
	width := 20
	height := 6
	colors := buildColorsFromTheme(th, height)
	colHeights := makeColHeights(width, height)
	frame := r.RenderFrame(width, height, colHeights, colors)

	// With shift = (col * 2) % height and 3 gradient zones, multiple segments
	// should appear on at least one row.
	hasMultiSegmentRow := false
	for _, line := range frame {
		if len(line.Segments) > 1 {
			hasMultiSegmentRow = true
			break
		}
	}
	assert.True(t, hasMultiSegmentRow,
		"at least one row should have multiple segments due to per-column color shift")
}

func TestSpectrumRenderer_FullHeight_SingleGlyph(t *testing.T) {
	// Pin to unicode mode for deterministic glyph output.
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	r := SpectrumRenderer{}
	th := theme.Load("black")
	width := 5
	height := 4
	colors := buildColorsFromTheme(th, height)
	// All columns at full height
	colHeights := make([]int, width)
	for i := range colHeights {
		colHeights[i] = height
	}

	frame := r.RenderFrame(width, height, colHeights, colors)
	require.Len(t, frame, height)

	fillGlyph := []rune(uikit.GlyphFor(uikit.GlyphBarFull, uikit.GlyphUnicode))[0]
	for rowIdx, line := range frame {
		got := []rune(line.Text)
		for colIdx, ch := range got {
			assert.Equal(t, fillGlyph, ch,
				"row %d col %d: full-height column should be fill glyph", rowIdx, colIdx)
		}
	}
}