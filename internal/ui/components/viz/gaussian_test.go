package viz

import (
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ Renderer = GaussianRenderer{}

func TestGaussianRenderer_FrameHeight(t *testing.T) {
	r := GaussianRenderer{}
	colors := makeColors(3)
	frame := r.RenderFrame(10, 3, makeColHeights(10, 6), colors)
	assert.Len(t, frame, 3)
}

func TestGaussianRenderer_Width(t *testing.T) {
	r := GaussianRenderer{}
	width := 15
	colors := makeColors(2)
	colHeights := makeColHeights(width, 4)
	frame := r.RenderFrame(width, 2, colHeights, colors)
	for _, line := range frame {
		assert.Equal(t, width, utf8.RuneCountInString(line.Text))
	}
}

func TestGaussianRenderer_ZeroWidth(t *testing.T) {
	r := GaussianRenderer{}
	frame := r.RenderFrame(0, 3, nil, makeColors(3))
	assert.Empty(t, frame)
}

func TestGaussianRenderer_ZeroHeight(t *testing.T) {
	r := GaussianRenderer{}
	frame := r.RenderFrame(10, 0, makeColHeights(10, 4), makeColors(0))
	assert.Empty(t, frame)
}

func TestGaussianRenderer_ColorsAssigned(t *testing.T) {
	r := GaussianRenderer{}
	colors := []lipgloss.Color{"#ff0000", "#00ff00", "#0000ff"}
	frame := r.RenderFrame(10, 3, makeColHeights(10, 6), colors)
	for i, line := range frame {
		assert.Equal(t, colors[i], line.Color)
	}
}

func TestGaussianRenderer_ColorSegments(t *testing.T) {
	r := GaussianRenderer{}
	width := 10
	height := 6
	colHeights := makeColHeights(width, 6)
	// Use varied colors so hue shift creates segment boundaries
	colors := []lipgloss.Color{
		"#ff0000", "#00ff00", "#0000ff", "#ffff00", "#ff00ff", "#00ffff",
	}
	frame := r.RenderFrame(width, height, colHeights, colors)
	require.Len(t, frame, height)

	hasMultiSegment := false
	for _, line := range frame {
		if len(line.Segments) > 1 {
			hasMultiSegment = true
			break
		}
	}
	assert.True(t, hasMultiSegment,
		"at least one row should have multiple color segments due to per-column hue shift")
}

func TestGaussianRenderer_AbsoluteDensityThresholds(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	r := GaussianRenderer{}
	width := 5
	height := 9
	// All columns at full wave reach so all rows are within wave
	colHeights := []int{18, 18, 18, 18, 18}
	colors := makeColors(height)
	frame := r.RenderFrame(width, height, colHeights, colors)
	require.Len(t, frame, height)

	center := 4 // (height-1)/2
	fillGlyph := "█"

	// dist < 1: only center row (dist=0) gets fill
	assert.Equal(t, fillGlyph+fillGlyph+fillGlyph+fillGlyph+fillGlyph,
		frame[center].Text, "center row (dist=0) should be fill")
	// dist < 2: row at dist=1 should be heavy glyph
	assert.NotContains(t, frame[center+1].Text, " ",
		"row at dist=1 should not be empty")
}

func TestGaussianRenderer_MaxHeight(t *testing.T) {
	r := GaussianRenderer{}
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
			"GaussianRenderer.MaxHeight(%d) should be %d", tt.displayHeight, tt.want)
	}
}

func TestGaussianRenderer_CenterDense_EdgeSparse(t *testing.T) {
	// With full-height columns (max wave reach), center rows should be
	// denser (more fill glyphs) than edge rows.
	r := GaussianRenderer{}
	width := 10
	height := 7
	colHeights := make([]int, width)
	for i := range colHeights {
		colHeights[i] = height * 2 // large wave reach so every row is in wave
	}
	colors := makeColors(height)
	frame := r.RenderFrame(width, height, colHeights, colors)
	require.Len(t, frame, height)

	// Count non-space characters per row
	countNonSpace := func(text string) int {
		count := 0
		for _, ch := range text {
			if ch != ' ' {
				count++
			}
		}
		return count
	}

	center := height / 2
	centerCount := countNonSpace(frame[center].Text)
	edgeCount := countNonSpace(frame[0].Text)

	// With full wave reach every row gets characters, so counts are equal.
	// Instead, verify center uses denser glyphs (fillGlyph) vs edges using lighter glyphs.
	assert.GreaterOrEqual(t, centerCount, edgeCount,
		"center row should have at least as many non-space chars as edge row")

	// Verify center has the densest glyph (first in the falloff priority)
	fillGlyph := uikit.GlyphFor(uikit.GlyphBarFull, uikit.ActiveMode())
	centerFirstRune := []rune(frame[center].Text)[0]
	assert.Equal(t, []rune(fillGlyph)[0], centerFirstRune,
		"center row should use the densest glyph")
}
