package viz

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// AsciiBarsRenderer tests
// ---------------------------------------------------------------------------

// Compile-time interface check.
var _ Renderer = (*AsciiBarsRenderer)(nil)

// TestAsciiBars_MaxLevels confirms MaxHeight() returns 4.
func TestAsciiBars_MaxLevels(t *testing.T) {
	r := NewAsciiBarsRenderer()
	assert.Equal(t, 4, r.MaxHeight(1))
	assert.Equal(t, 4, r.MaxHeight(4))
	assert.Equal(t, 4, r.MaxHeight(8))
	assert.Equal(t, 4, r.MaxHeight(20))
}

// TestAsciiBars_AllAscii confirms Render output contains no braille, block,
// or half-block characters.
func TestAsciiBars_AllAscii(t *testing.T) {
	bannedRanges := []struct {
		lo, hi rune
		desc   string
	}{
		{'▀', '▟', "block-elements (half-blocks etc.)"},
		{'⠀', '⣿', "braille patterns"},
	}
	bannedChars := []rune{
		'█',                               // U+2588 FULL BLOCK
		'▉', '▊', '▋', '▌', '▍', '▎', '▏', // partial blocks
	}

	r := NewAsciiBarsRenderer()
	colors := makeColors(4)
	colHeights := []int{0, 1, 2, 3, 4, 0, 4, 2}
	frame := r.RenderFrame(8, 4, colHeights, colors)

	for rowIdx, line := range frame {
		// Strip ANSI sequences before checking runes.
		// We only inspect the rendered content without lipgloss styling.
		// Use the raw text field which is already stripped by design for tests.
		for _, ch := range line.Text {
			for _, b := range bannedChars {
				assert.NotEqual(t, b, ch,
					"row %d: banned character %U (%c) found in output", rowIdx, ch, ch)
			}
			for _, br := range bannedRanges {
				assert.False(t, ch >= br.lo && ch <= br.hi,
					"row %d: character %U (%c) is in banned range %s", rowIdx, ch, ch, br.desc)
			}
		}
	}
}

// TestAsciiBars_OnlyPermittedChars confirms output only contains #, =, ., space.
func TestAsciiBars_OnlyPermittedChars(t *testing.T) {
	r := NewAsciiBarsRenderer()
	colors := makeColors(4)
	colHeights := []int{0, 1, 2, 3, 4, 0, 4, 2}
	frame := r.RenderFrame(8, 4, colHeights, colors)

	for rowIdx, line := range frame {
		for _, ch := range line.Text {
			assert.True(t, ch == '#' || ch == '=' || ch == '.' || ch == ' ',
				"row %d: unexpected character %U (%c)", rowIdx, ch, ch)
		}
	}
}

// TestAsciiBars_FrameHeight confirms the number of rows matches displayHeight.
func TestAsciiBars_FrameHeight(t *testing.T) {
	r := NewAsciiBarsRenderer()
	colHeights := makeColHeights(10, 4)

	for _, h := range []int{1, 4, 8, 12} {
		frame := r.RenderFrame(10, h, colHeights, makeColors(h))
		assert.Len(t, frame, h, "expected %d rows for displayHeight=%d", h, h)
	}
}

// TestAsciiBars_FullHeight_HasFilledChars confirms that when all columns are at
// max height, there are filled (#) characters in the output.
func TestAsciiBars_FullHeight_HasFilledChars(t *testing.T) {
	r := NewAsciiBarsRenderer()
	width := 6
	height := 4
	colHeights := make([]int, width)
	for i := range colHeights {
		colHeights[i] = 4 // max height
	}
	colors := makeColors(height)
	frame := r.RenderFrame(width, height, colHeights, colors)

	// At least the bottom row should be fully filled with '#'
	bottomRow := frame[height-1].Text
	require.NotEmpty(t, bottomRow)
	for _, ch := range bottomRow {
		assert.Equal(t, '#', ch, "bottom row with max height should be all '#'")
	}
}

// TestAsciiBars_ZeroHeight_AllEmpty confirms that zero-height columns produce
// only spaces.
func TestAsciiBars_ZeroHeight_AllEmpty(t *testing.T) {
	r := NewAsciiBarsRenderer()
	width := 5
	height := 4
	colHeights := make([]int, width) // all zero
	colors := makeColors(height)
	frame := r.RenderFrame(width, height, colHeights, colors)

	for rowIdx, line := range frame {
		for _, ch := range line.Text {
			assert.Equal(t, ' ', ch,
				"row %d: zero-height columns should produce only spaces", rowIdx)
		}
	}
}

// TestAsciiBars_ColorsAssigned confirms that colors from the input slice are
// assigned to each row in the frame.
func TestAsciiBars_ColorsAssigned(t *testing.T) {
	r := NewAsciiBarsRenderer()
	colors := []lipgloss.Color{"#ff0000", "#00ff00", "#0000ff", "#ffffff"}
	colHeights := makeColHeights(6, 2)
	frame := r.RenderFrame(6, 4, colHeights, colors)
	for i, line := range frame {
		assert.Equal(t, colors[i], line.Color,
			"row %d: expected color %v, got %v", i, colors[i], line.Color)
	}
}

// TestAsciiBars_ZeroWidth returns an empty frame.
func TestAsciiBars_ZeroWidth(t *testing.T) {
	r := NewAsciiBarsRenderer()
	frame := r.RenderFrame(0, 4, nil, makeColors(4))
	assert.Empty(t, frame)
}

// TestAsciiBars_ZeroDisplayHeight returns an empty frame.
func TestAsciiBars_ZeroDisplayHeight(t *testing.T) {
	r := NewAsciiBarsRenderer()
	frame := r.RenderFrame(10, 0, makeColHeights(10, 2), makeColors(0))
	assert.Empty(t, frame)
}

// TestAsciiBars_HalfHeight_FourLevelRamp confirms the 4-level density ramp at the
// canonical height=4 case. With colHeight=2 (half of MaxHeight=4), the new
// integer-row mapping must produce all four glyph levels:
//
//	row 0 (top):    rowFromBottom=3, diff=-1 → '.'  (cap above bar)
//	row 1:          rowFromBottom=2, diff= 0 → '='  (top edge of bar)
//	row 2:          rowFromBottom=1, diff= 1 → '#'  (inside bar)
//	row 3 (bottom): rowFromBottom=0, diff= 2 → '#'  (inside bar)
func TestAsciiBars_HalfHeight_FourLevelRamp(t *testing.T) {
	r := NewAsciiBarsRenderer()
	width := 4
	height := 4
	// Half-height: colHeight = 2 (out of MaxHeight=4)
	colHeights := []int{2, 2, 2, 2}
	colors := makeColors(height)
	frame := r.RenderFrame(width, height, colHeights, colors)

	for _, ch := range frame[0].Text {
		assert.Equal(t, '.', ch, "row 0 (top): one step above bar top should be '.'")
	}
	for _, ch := range frame[1].Text {
		assert.Equal(t, '=', ch, "row 1: top edge of bar should be '='")
	}
	for _, ch := range frame[2].Text {
		assert.Equal(t, '#', ch, "row 2: inside bar should be '#'")
	}
	for _, ch := range frame[3].Text {
		assert.Equal(t, '#', ch, "row 3 (bottom): inside bar should be '#'")
	}
}

// TestAsciiBars_FourLevelDensityRamp confirms that all four glyph levels —
// ' ', '.', '=', '#' — appear in a height=4 frame when column heights span
// the full [1,4] range. This is the canonical acceptance test for the 4-level
// density contract: the old band-based implementation never produced '=' or '.'
// at integer column heights.
func TestAsciiBars_FourLevelDensityRamp(t *testing.T) {
	r := NewAsciiBarsRenderer()
	// colHeights {1,2,3,4} at height=4 exercises every glyph:
	//   h=1: rows get  ' ' ' ' '=' '#'  (bottom-up: diff=1,'=','-1','.')
	//   h=2: rows get  '.' '=' '#' '#'
	//   h=3: rows get  '=' '#' '#' '#'
	//   h=4: rows get  '#' '#' '#' '#'
	frame := r.RenderFrame(4, 4, []int{1, 2, 3, 4}, makeColors(4))
	require.Len(t, frame, 4)

	all := strings.Join(func() []string {
		s := make([]string, len(frame))
		for i, l := range frame {
			s[i] = l.Text
		}
		return s
	}(), "")

	assert.Contains(t, all, "#", "frame should contain '#'")
	assert.Contains(t, all, "=", "frame should contain '=' (top-edge glyph)")
	assert.Contains(t, all, ".", "frame should contain '.' (cap-above-bar glyph)")
}

// TestAsciiBars_RowCount_MatchesDisplay confirms row count in the frame equals
// the displayHeight argument.
func TestAsciiBars_RowCount_MatchesDisplay(t *testing.T) {
	r := NewAsciiBarsRenderer()
	for _, h := range []int{1, 2, 5, 10} {
		frame := r.RenderFrame(5, h, makeColHeights(5, 2), makeColors(h))
		assert.Len(t, frame, h)
	}
}

// TestAsciiBars_NewAsciiBarsRenderer confirms the constructor returns a non-nil value.
func TestAsciiBars_NewAsciiBarsRenderer(t *testing.T) {
	r := NewAsciiBarsRenderer()
	assert.NotNil(t, r)
}

// TestAsciiBars_ThemeColorApplied confirms the frame rows carry the theme
// gradient color from the Engine's buildColors — we pass explicit colors and
// verify they are preserved.
func TestAsciiBars_ThemeColorApplied(t *testing.T) {
	r := NewAsciiBarsRenderer()
	th := theme.Load("black")
	colors := []lipgloss.Color{th.Gradient3(), th.Gradient2(), th.Gradient1(), th.Gradient1()}
	frame := r.RenderFrame(5, 4, []int{4, 3, 2, 1, 0}, colors)
	require.Len(t, frame, 4)
	assert.Equal(t, th.Gradient3(), frame[0].Color)
	assert.Equal(t, th.Gradient2(), frame[1].Color)
	assert.Equal(t, th.Gradient1(), frame[2].Color)
}

// TestAsciiBars_WidthPreserved confirms each row has exactly `width` rune columns
// (ignoring ANSI escape sequences in the Text field — Text holds raw chars).
func TestAsciiBars_WidthPreserved(t *testing.T) {
	r := NewAsciiBarsRenderer()
	width := 10
	frame := r.RenderFrame(width, 4, makeColHeights(width, 2), makeColors(4))
	for rowIdx, line := range frame {
		// line.Text contains the raw chars (before lipgloss wraps with color).
		assert.Equal(t, width, len([]rune(line.Text)),
			"row %d should have %d runes", rowIdx, width)
	}
}

// TestAsciiBars_LevelOrdering confirms that filled columns produce more '#'
// characters in lower rows than in upper rows.
func TestAsciiBars_LevelOrdering(t *testing.T) {
	r := NewAsciiBarsRenderer()
	height := 4
	colHeights := []int{4, 4, 4, 4} // fully filled
	colors := makeColors(height)
	frame := r.RenderFrame(4, height, colHeights, colors)

	// Count '#' in top row vs bottom row; bottom should have >= top.
	countHash := func(s string) int {
		return strings.Count(s, "#")
	}
	topHashes := countHash(frame[0].Text)
	bottomHashes := countHash(frame[height-1].Text)
	assert.GreaterOrEqual(t, bottomHashes, topHashes,
		"bottom row should have >= filled chars as top row")
}
