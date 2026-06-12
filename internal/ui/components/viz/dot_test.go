package viz

import (
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ Renderer = DotRenderer{}

func TestDotRenderer_FrameHeight(t *testing.T) {
	r := DotRenderer{}
	colors := makeColors(3)
	frame := r.RenderFrame(10, 3, makeColHeights(10, 6), colors)
	assert.Len(t, frame, 3)
}

func TestDotRenderer_Width(t *testing.T) {
	r := DotRenderer{}
	width := 15
	colors := makeColors(2)
	colHeights := makeColHeights(width, 4)
	frame := r.RenderFrame(width, 2, colHeights, colors)
	for _, line := range frame {
		assert.Equal(t, width, utf8.RuneCountInString(line.Text))
	}
}

func TestDotRenderer_ZeroWidth(t *testing.T) {
	r := DotRenderer{}
	frame := r.RenderFrame(0, 3, nil, makeColors(3))
	assert.Empty(t, frame)
}

func TestDotRenderer_ZeroHeight(t *testing.T) {
	r := DotRenderer{}
	frame := r.RenderFrame(10, 0, makeColHeights(10, 4), makeColors(0))
	assert.Empty(t, frame)
}

func TestDotRenderer_ColorsAssigned(t *testing.T) {
	r := DotRenderer{}
	colors := []lipgloss.Color{"#ff0000", "#00ff00", "#0000ff"}
	frame := r.RenderFrame(10, 3, makeColHeights(10, 6), colors)
	for i, line := range frame {
		assert.Equal(t, colors[i], line.Color)
	}
}

func TestDotRenderer_MaxHeight(t *testing.T) {
	r := DotRenderer{}
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
			"DotRenderer.MaxHeight(%d) should be %d", tt.displayHeight, tt.want)
	}
}

func TestDotRenderer_DensityMapping(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	r := DotRenderer{}
	frame := r.RenderFrameAt(10, 4, 0, makeColors(4))
	require.Len(t, frame, 4)

	for _, line := range frame {
		for _, ch := range line.Text {
			assert.True(t, ch == ' ' || ch == '·' || ch == '•' || ch == '●',
				"unexpected rune %U (%c)", ch, ch)
		}
	}
}

func TestDotRenderer_RenderFrameAt_MaxDensityCenter(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	r := DotRenderer{}
	frame := r.RenderFrameAt(10, 4, 0, makeColors(4))
	require.Len(t, frame, 4)
	for _, line := range frame {
		assert.Equal(t, 10, utf8.RuneCountInString(line.Text))
	}
}

func TestDotRenderer_ImplementsFrameAwareRenderer(t *testing.T) {
	assert.Implements(t, (*FrameAwareRenderer)(nil), DotRenderer{})
}
