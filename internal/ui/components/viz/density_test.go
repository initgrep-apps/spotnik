package viz

import (
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface check.
var _ Renderer = DensityRenderer{}

func TestDensityRenderer_DensityHaloAboveBars(t *testing.T) {
	// Pin to unicode mode for deterministic glyph output.
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	r := DensityRenderer{}
	// width=1, height=6, colHeights={3}
	// Bar fills rows 3,4,5 from bottom (0-indexed from bottom).
	// Row 0 (top)    → rowFromBottom=5 → distAbove=5-3=2 → ▒ (medium)
	// Row 1          → rowFromBottom=4 → distAbove=4-3=1 → ▓ (heavy)
	// Row 2          → rowFromBottom=3 → distAbove=3-3=0 → █ (full, at bar top)
	// Row 3          → rowFromBottom=2 → distAbove=2-3=-1 → █ (fill, below bar top)
	// Row 4          → rowFromBottom=1 → distAbove=1-3=-2 → █ (fill, below bar top)
	// Row 5 (bottom) → rowFromBottom=0 → distAbove=0-3=-3 → █ (fill, below bar top)
	//
	// Wait — let me recalculate. height=6, colHeights={3}:
	// rowIdx=0 → rowFromBottom=5 → distAbove = 5-3 = 2 → medium (▒)
	// rowIdx=1 → rowFromBottom=4 → distAbove = 4-3 = 1 → heavy (▓)
	// rowIdx=2 → rowFromBottom=3 → distAbove = 3-3 = 0 → full (█)
	// rowIdx=3 → rowFromBottom=2 → distAbove = 2-3 = -1 → full (█, below bar top)
	// rowIdx=4 → rowFromBottom=1 → distAbove = 1-3 = -2 → full (█, below bar top)
	// rowIdx=5 → rowFromBottom=0 → distAbove = 0-3 = -3 → full (█, below bar top)
	//
	// But we need 3 halo rows above the bar top. With height=6 and colHeights=3,
	// the bar fills rows from bottom 0..2 (3 rows). Row from bottom 3 = bar top.
	// distAbove at rowFromBottom=3 is 0 (at bar top → █).
	// Above that: rowFromBottom=4 → distAbove=1 (▓), rowFromBottom=5 → distAbove=2 (▒).
	// That's only 2 halo rows above. We need 3.
	// Let's use height=7, colHeights={3} to get 3 halo rows:
	// rowIdx=0 → rowFromBottom=6 → distAbove=6-3=3 → ░ (empty)
	// rowIdx=1 → rowFromBottom=5 → distAbove=5-3=2 → ▒ (medium)
	// rowIdx=2 → rowFromBottom=4 → distAbove=4-3=1 → ▓ (heavy)
	// rowIdx=3 → rowFromBottom=3 → distAbove=3-3=0 → █ (at bar top)
	// rowIdx=4 → rowFromBottom=2 → distAbove=2-3=-1 → █ (below bar top)
	// rowIdx=5 → rowFromBottom=1 → distAbove=1-3=-2 → █ (below bar top)
	// rowIdx=6 → rowFromBottom=0 → distAbove=0-3=-3 → █ (below bar top)

	width := 1
	height := 7
	colHeights := []int{3}
	colors := makeColors(height)
	frame := r.RenderFrame(width, height, colHeights, colors)
	require.Len(t, frame, height)

	expectedRunes := []rune{
		[]rune(uikit.GlyphFor(uikit.GlyphBarEmpty, uikit.GlyphUnicode))[0],  // ░  distAbove=3
		[]rune(uikit.GlyphFor(uikit.GlyphBarMedium, uikit.GlyphUnicode))[0], // ▒  distAbove=2
		[]rune(uikit.GlyphFor(uikit.GlyphBarHeavy, uikit.GlyphUnicode))[0],  // ▓  distAbove=1
		[]rune(uikit.GlyphFor(uikit.GlyphBarFull, uikit.GlyphUnicode))[0],   // █  distAbove=0 (at bar top)
		[]rune(uikit.GlyphFor(uikit.GlyphBarFull, uikit.GlyphUnicode))[0],   // █  distAbove=-1 (below bar top)
		[]rune(uikit.GlyphFor(uikit.GlyphBarFull, uikit.GlyphUnicode))[0],   // █  distAbove=-2
		[]rune(uikit.GlyphFor(uikit.GlyphBarFull, uikit.GlyphUnicode))[0],   // █  distAbove=-3
	}

	for rowIdx, line := range frame {
		got := []rune(line.Text)
		assert.Equal(t, 1, len(got), "row %d should have 1 rune", rowIdx)
		if len(got) >= 1 {
			assert.Equal(t, expectedRunes[rowIdx], got[0],
				"row %d (rowFromBottom=%d): expected %U, got %U",
				rowIdx, height-1-rowIdx, expectedRunes[rowIdx], got[0])
		}
	}
}

func TestDensityRenderer_FrameHeight(t *testing.T) {
	r := DensityRenderer{}
	colors := makeColors(4)
	frame := r.RenderFrame(10, 4, makeColHeights(10, 4), colors)
	assert.Len(t, frame, 4)
}

func TestDensityRenderer_ColorsAssigned(t *testing.T) {
	r := DensityRenderer{}
	colors := []lipgloss.Color{"#ff0000", "#00ff00", "#0000ff", "#ffffff"}
	frame := r.RenderFrame(10, 4, makeColHeights(10, 4), colors)
	for i, line := range frame {
		assert.Equal(t, colors[i], line.Color, "row %d color mismatch", i)
	}
}

func TestDensityRenderer_Width(t *testing.T) {
	r := DensityRenderer{}
	width := 12
	colors := makeColors(3)
	colHeights := makeColHeights(width, 3)
	frame := r.RenderFrame(width, 3, colHeights, colors)
	for _, line := range frame {
		assert.Equal(t, width, utf8.RuneCountInString(line.Text))
	}
}

func TestDensityRenderer_ZeroWidth(t *testing.T) {
	r := DensityRenderer{}
	frame := r.RenderFrame(0, 3, nil, makeColors(3))
	assert.Empty(t, frame)
}

func TestDensityRenderer_ZeroHeight(t *testing.T) {
	r := DensityRenderer{}
	frame := r.RenderFrame(10, 0, makeColHeights(10, 4), makeColors(0))
	assert.Empty(t, frame)
}

func TestDensityRenderer_FullHeight_AllFilled(t *testing.T) {
	r := DensityRenderer{}
	width := 5
	height := 4
	colHeights := make([]int, width)
	for i := range colHeights {
		colHeights[i] = height
	}
	colors := makeColors(height)
	frame := r.RenderFrame(width, height, colHeights, colors)
	fillGlyph := []rune(uikit.GlyphFor(uikit.GlyphBarFull, uikit.ActiveMode()))[0]
	for _, line := range frame {
		for _, ch := range line.Text {
			assert.Equal(t, fillGlyph, ch, "full height bar should fill all rows")
		}
	}
}

func TestDensityRenderer_ZeroHeights_AllSpaces(t *testing.T) {
	r := DensityRenderer{}
	colHeights := make([]int, 10)
	colors := makeColors(4)
	frame := r.RenderFrame(10, 4, colHeights, colors)
	for _, line := range frame {
		for _, ch := range line.Text {
			assert.Equal(t, ' ', ch, "zero height should be all spaces")
		}
	}
}

func TestDensityRenderer_MaxHeight(t *testing.T) {
	r := DensityRenderer{}
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
			"DensityRenderer.MaxHeight(%d) should be %d", tt.displayHeight, tt.want)
	}
}

func TestDensityRenderer_OnlyDensityOrSpace(t *testing.T) {
	// Pin to unicode mode for deterministic glyph output.
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	r := DensityRenderer{}
	colors := makeColors(6)
	colHeights := makeColHeights(20, 6)
	frame := r.RenderFrame(20, 6, colHeights, colors)

	fillGlyph := []rune(uikit.GlyphFor(uikit.GlyphBarFull, uikit.GlyphUnicode))[0]
	heavyGlyph := []rune(uikit.GlyphFor(uikit.GlyphBarHeavy, uikit.GlyphUnicode))[0]
	mediumGlyph := []rune(uikit.GlyphFor(uikit.GlyphBarMedium, uikit.GlyphUnicode))[0]
	emptyGlyph := []rune(uikit.GlyphFor(uikit.GlyphBarEmpty, uikit.GlyphUnicode))[0]

	for _, line := range frame {
		for _, ch := range line.Text {
			assert.True(t,
				ch == fillGlyph || ch == heavyGlyph || ch == mediumGlyph || ch == emptyGlyph || ch == ' ',
				"expected density glyph or space, got %U (%c)", ch, ch)
		}
	}
}

func TestDensityRenderer_DensestGlyphWins(t *testing.T) {
	// Pin to unicode mode for deterministic glyph output.
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	r := DensityRenderer{}
	// Two columns: col 0 height=5, col 1 height=2
	// height=6. For row 2 (rowFromBottom=3):
	//   col 0: distAbove = 3-5 = -2 → █ (fill)
	//   col 1: distAbove = 3-2 = 1 → ▓ (heavy)
	// Each cell is independent — "densest wins" is per-column, which
	// naturally happens since we compute per column.
	width := 2
	height := 6
	colHeights := []int{5, 2}
	colors := makeColors(height)
	frame := r.RenderFrame(width, height, colHeights, colors)
	require.Len(t, frame, height)

	fillGlyph := []rune(uikit.GlyphFor(uikit.GlyphBarFull, uikit.GlyphUnicode))[0]
	heavyGlyph := []rune(uikit.GlyphFor(uikit.GlyphBarHeavy, uikit.GlyphUnicode))[0]

	// Row 2 (rowFromBottom=3): col 0 should be fill, col 1 should be heavy
	row := frame[2]
	got := []rune(row.Text)
	assert.Equal(t, fillGlyph, got[0], "col 0 at row 2 should be fill glyph")
	assert.Equal(t, heavyGlyph, got[1], "col 1 at row 2 should be heavy glyph")
}

func TestDensityRenderer_ShortColHeights_PadsWithZero(t *testing.T) {
	r := DensityRenderer{}
	// colHeights shorter than width — missing columns default to height 0
	width := 5
	height := 4
	colHeights := []int{3, 2} // only 2 of 5 columns have heights
	colors := makeColors(height)
	frame := r.RenderFrame(width, height, colHeights, colors)
	require.Len(t, frame, height)

	// All rows should have width=5
	for _, line := range frame {
		assert.Equal(t, width, utf8.RuneCountInString(line.Text))
	}

	// Column 2-4 should be spaces (height 0 → all rows above)
	for _, line := range frame {
		got := []rune(line.Text)
		assert.Equal(t, ' ', got[2], "missing colHeights should default to 0")
		assert.Equal(t, ' ', got[3], "missing colHeights should default to 0")
		assert.Equal(t, ' ', got[4], "missing colHeights should default to 0")
	}
}

func TestDensityRenderer_SpacesBeyondHaloRange(t *testing.T) {
	// Pin to unicode mode for deterministic glyph output.
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	r := DensityRenderer{}
	// With colHeights=1 and height=8, the halo extends 3 rows above bar top,
	// and rows beyond 3 above the bar should be spaces.
	// rowIdx=0 → rowFromBottom=7 → distAbove=7-1=6 → space (>3)
	// rowIdx=1 → rowFromBottom=6 → distAbove=6-1=5 → space (>3)
	// rowIdx=2 → rowFromBottom=5 → distAbove=5-1=4 → space (>3)
	// rowIdx=3 → rowFromBottom=4 → distAbove=4-1=3 → ░ (empty)
	// rowIdx=4 → rowFromBottom=3 → distAbove=3-1=2 → ▒ (medium)
	// rowIdx=5 → rowFromBottom=2 → distAbove=2-1=1 → ▓ (heavy)
	// rowIdx=6 → rowFromBottom=1 → distAbove=1-1=0 → █ (at bar top)
	// rowIdx=7 → rowFromBottom=0 → distAbove=0-1=-1 → █ (below bar top)

	width := 1
	height := 8
	colHeights := []int{1}
	colors := makeColors(height)
	frame := r.RenderFrame(width, height, colHeights, colors)
	require.Len(t, frame, 8)

	space := ' '
	emptyGlyph := []rune(uikit.GlyphFor(uikit.GlyphBarEmpty, uikit.GlyphUnicode))[0]
	mediumGlyph := []rune(uikit.GlyphFor(uikit.GlyphBarMedium, uikit.GlyphUnicode))[0]
	heavyGlyph := []rune(uikit.GlyphFor(uikit.GlyphBarHeavy, uikit.GlyphUnicode))[0]
	fillGlyph := []rune(uikit.GlyphFor(uikit.GlyphBarFull, uikit.GlyphUnicode))[0]

	expected := []rune{space, space, space, emptyGlyph, mediumGlyph, heavyGlyph, fillGlyph, fillGlyph}
	for rowIdx, line := range frame {
		got := []rune(line.Text)
		require.Len(t, got, 1, "row %d should have 1 rune", rowIdx)
		assert.Equal(t, expected[rowIdx], got[0],
			"row %d: expected %U, got %U", rowIdx, expected[rowIdx], got[0])
	}
}
