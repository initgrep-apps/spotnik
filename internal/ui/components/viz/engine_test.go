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

// ---------------------------------------------------------------------------
// Task 1: StyledLine and Frame types
// ---------------------------------------------------------------------------

func TestStyledLine(t *testing.T) {
	line := StyledLine{Text: "hello", Color: lipgloss.Color("#ff0000")}
	assert.Equal(t, "hello", line.Text)
	assert.Equal(t, lipgloss.Color("#ff0000"), line.Color)
}

func TestFrame(t *testing.T) {
	f := Frame{
		{Text: "row0", Color: lipgloss.Color("#ff0000")},
		{Text: "row1", Color: lipgloss.Color("#00ff00")},
	}
	assert.Len(t, f, 2)
	assert.Equal(t, "row0", f[0].Text)
	assert.Equal(t, "row1", f[1].Text)
}

// ---------------------------------------------------------------------------
// Task 2: BrailleRenderer
// ---------------------------------------------------------------------------

// Compile-time interface check.
var _ Renderer = BrailleRenderer{}

func TestBrailleRenderer_FrameHeight(t *testing.T) {
	r := BrailleRenderer{}
	colors := makeColors(3)
	frame := r.RenderFrame(10, 3, makeColHeights(10, 6), colors)
	assert.Len(t, frame, 3)
}

func TestBrailleRenderer_ColorsAssigned(t *testing.T) {
	r := BrailleRenderer{}
	colors := []lipgloss.Color{"#ff0000", "#00ff00", "#0000ff"}
	frame := r.RenderFrame(10, 3, makeColHeights(10, 6), colors)
	for i, line := range frame {
		assert.Equal(t, colors[i], line.Color)
	}
}

func TestBrailleRenderer_OnlyBrailleRunes(t *testing.T) {
	r := BrailleRenderer{}
	colors := makeColors(4)
	colHeights := makeColHeights(20, 8)
	frame := r.RenderFrame(20, 4, colHeights, colors)
	for _, line := range frame {
		for _, ch := range line.Text {
			assert.True(t, ch >= '\u2800' && ch <= '\u28FF',
				"expected braille rune, got %U", ch)
		}
	}
}

func TestBrailleRenderer_FullHeight_TopFilled(t *testing.T) {
	r := BrailleRenderer{}
	width := 5
	height := 2
	// maxDotRows = height*4 = 8; fill all columns to max
	colHeights := make([]int, width)
	for i := range colHeights {
		colHeights[i] = height * 4
	}
	colors := makeColors(height)
	frame := r.RenderFrame(width, height, colHeights, colors)
	// Top row (index 0) should all be non-blank braille
	for _, ch := range frame[0].Text {
		assert.NotEqual(t, '\u2800', ch, "top row should be filled")
	}
}

func TestBrailleRenderer_ZeroHeights_AllBlank(t *testing.T) {
	r := BrailleRenderer{}
	colHeights := make([]int, 10)
	colors := makeColors(3)
	frame := r.RenderFrame(10, 3, colHeights, colors)
	for _, line := range frame {
		for _, ch := range line.Text {
			assert.Equal(t, '\u2800', ch, "zero height should be blank braille")
		}
	}
}

func TestBrailleRenderer_Width(t *testing.T) {
	r := BrailleRenderer{}
	width := 15
	colors := makeColors(2)
	colHeights := makeColHeights(width, 4)
	frame := r.RenderFrame(width, 2, colHeights, colors)
	for _, line := range frame {
		assert.Equal(t, width, utf8.RuneCountInString(line.Text))
	}
}

func TestBrailleRenderer_ZeroWidth(t *testing.T) {
	r := BrailleRenderer{}
	frame := r.RenderFrame(0, 3, nil, makeColors(3))
	assert.Empty(t, frame)
}

func TestBrailleRenderer_ZeroHeight(t *testing.T) {
	r := BrailleRenderer{}
	frame := r.RenderFrame(10, 0, makeColHeights(10, 4), makeColors(0))
	assert.Empty(t, frame)
}

// ---------------------------------------------------------------------------
// Task 3: BlockRenderer
// ---------------------------------------------------------------------------

// Compile-time interface check.
var _ Renderer = BlockRenderer{}

func TestBlockRenderer_FrameHeight(t *testing.T) {
	r := BlockRenderer{}
	colors := makeColors(4)
	frame := r.RenderFrame(10, 4, makeColHeights(10, 4), colors)
	assert.Len(t, frame, 4)
}

func TestBlockRenderer_ColorsAssigned(t *testing.T) {
	r := BlockRenderer{}
	colors := []lipgloss.Color{"#ff0000", "#00ff00", "#0000ff", "#ffffff"}
	frame := r.RenderFrame(10, 4, makeColHeights(10, 4), colors)
	for i, line := range frame {
		assert.Equal(t, colors[i], line.Color)
	}
}

func TestBlockRenderer_OnlyBlockOrSpace(t *testing.T) {
	// No mode pin — let LANG-driven ActiveMode() determine the fill glyph.
	// fillGlyph is '█' in unicode mode and '#' in ASCII mode; both pass.
	r := BlockRenderer{}
	colors := makeColors(4)
	colHeights := makeColHeights(20, 4)
	frame := r.RenderFrame(20, 4, colHeights, colors)
	fillGlyph := []rune(uikit.GlyphFor(uikit.GlyphBarFull, uikit.ActiveMode()))[0]
	for _, line := range frame {
		for _, ch := range line.Text {
			assert.True(t, ch == fillGlyph || ch == ' ',
				"expected block char or space, got %U (%c)", ch, ch)
		}
	}
}

func TestBlockRenderer_FullHeight_AllFilled(t *testing.T) {
	// No mode pin — let LANG-driven ActiveMode() determine the fill glyph.
	// fillGlyph is '█' in unicode mode and '#' in ASCII mode; both pass.
	r := BlockRenderer{}
	width := 5
	height := 4
	colHeights := make([]int, width)
	for i := range colHeights {
		colHeights[i] = height // full height
	}
	colors := makeColors(height)
	frame := r.RenderFrame(width, height, colHeights, colors)
	fillGlyph := []rune(uikit.GlyphFor(uikit.GlyphBarFull, uikit.ActiveMode()))[0]
	for _, line := range frame {
		for _, ch := range line.Text {
			assert.Equal(t, fillGlyph, ch)
		}
	}
}

func TestBlockRenderer_ZeroHeights_AllSpaces(t *testing.T) {
	r := BlockRenderer{}
	colHeights := make([]int, 10)
	colors := makeColors(4)
	frame := r.RenderFrame(10, 4, colHeights, colors)
	for _, line := range frame {
		for _, ch := range line.Text {
			assert.Equal(t, ' ', ch)
		}
	}
}

func TestBlockRenderer_Width(t *testing.T) {
	r := BlockRenderer{}
	width := 12
	colors := makeColors(3)
	colHeights := makeColHeights(width, 3)
	frame := r.RenderFrame(width, 3, colHeights, colors)
	for _, line := range frame {
		assert.Equal(t, width, utf8.RuneCountInString(line.Text))
	}
}

func TestBlockRenderer_ZeroWidth(t *testing.T) {
	r := BlockRenderer{}
	frame := r.RenderFrame(0, 4, nil, makeColors(4))
	assert.Empty(t, frame)
}

func TestBlockRenderer_ZeroHeight(t *testing.T) {
	r := BlockRenderer{}
	frame := r.RenderFrame(10, 0, makeColHeights(10, 4), makeColors(0))
	assert.Empty(t, frame)
}

// ---------------------------------------------------------------------------
// Task 4: Pattern type and registry
// ---------------------------------------------------------------------------

func TestPatterns_Count(t *testing.T) {
	assert.Len(t, Patterns(), 6)
}

func TestPatterns_Names(t *testing.T) {
	for i, p := range Patterns() {
		assert.NotEmpty(t, p.Name, "pattern %d has empty name", i)
	}
}

func TestPatterns_NonNilRenderer(t *testing.T) {
	for i, p := range Patterns() {
		assert.NotNil(t, p.Renderer, "pattern %d has nil renderer", i)
	}
}

func TestPatterns_NonNilHeightFunc(t *testing.T) {
	for i, p := range Patterns() {
		assert.NotNil(t, p.HeightFunc, "pattern %d has nil HeightFunc", i)
	}
}

func TestPatterns_BrailleRenderers(t *testing.T) {
	ps := Patterns()
	for _, idx := range []int{0, 1} {
		_, ok := ps[idx].Renderer.(BrailleRenderer)
		assert.True(t, ok, "pattern %d should use BrailleRenderer", idx)
	}
}

func TestPatterns_BlockRenderers(t *testing.T) {
	ps := Patterns()
	for _, idx := range []int{2} {
		_, ok := ps[idx].Renderer.(BlockRenderer)
		assert.True(t, ok, "pattern %d should use BlockRenderer", idx)
	}
}

func TestPatterns_DensityRenderer(t *testing.T) {
	ps := Patterns()
	_, ok := ps[3].Renderer.(DensityRenderer)
	assert.True(t, ok, "pattern 3 should use DensityRenderer")
}

func TestPatterns_CharRenderer(t *testing.T) {
	ps := Patterns()
	cr, ok := ps[4].Renderer.(CharRenderer)
	assert.True(t, ok, "pattern 4 should use CharRenderer")
	assert.Equal(t, []rune{' ', '0', '1'}, cr.Chars)
	assert.Equal(t, 1, cr.Scale)
}

func TestPatterns_SpectrumRenderer(t *testing.T) {
	ps := Patterns()
	_, ok := ps[5].Renderer.(SpectrumRenderer)
	assert.True(t, ok, "pattern 5 should use SpectrumRenderer")
}

func TestPatterns_HeightFunc_Length(t *testing.T) {
	for i, p := range Patterns() {
		out := p.HeightFunc(20, 16, 0)
		assert.Len(t, out, 20, "pattern %d HeightFunc should return width-length slice", i)
	}
}

func TestPatterns_HeightFunc_Range(t *testing.T) {
	for i, p := range Patterns() {
		maxH := 16
		out := p.HeightFunc(20, maxH, 5)
		for j, h := range out {
			assert.True(t, h >= 0 && h <= maxH,
				"pattern %d col %d: height %d out of range [0,%d]", i, j, h, maxH)
		}
	}
}

func TestPatterns_HeightFunc_Deterministic(t *testing.T) {
	for i, p := range Patterns() {
		a := p.HeightFunc(20, 16, 7)
		b := p.HeightFunc(20, 16, 7)
		assert.Equal(t, a, b, "pattern %d HeightFunc should be deterministic", i)
	}
}

func TestPatterns_DifferentProfiles(t *testing.T) {
	// All patterns should produce different height profiles for the same input
	seen := make([][]int, 0, 6)
	for _, p := range Patterns() {
		out := p.HeightFunc(40, 32, 10)
		seen = append(seen, out)
	}
	// At least two distinct profiles must exist
	allSame := true
	for i := 1; i < len(seen); i++ {
		if !equalSlices(seen[0], seen[i]) {
			allSame = false
			break
		}
	}
	assert.False(t, allSame, "all patterns produce identical height profiles")
}

// ---------------------------------------------------------------------------
// Task 5: Engine
// ---------------------------------------------------------------------------

func TestEngine_NewEngine_PatternCount(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	assert.Equal(t, 6, e.PatternCount())
}

func TestEngine_NewEngine_DefaultPattern(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	assert.Equal(t, 0, e.Pattern())
}

func TestEngine_SetSize_FrameCount(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	e.SetSize(40, 6)
	e.SetPlaying(true)
	f := e.CurrentFrame()
	assert.Len(t, f, 6)
}

func TestEngine_Advance_Playing(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	e.SetSize(20, 4)
	e.SetPlaying(true)
	assert.Equal(t, 0, e.FrameIndex())
	e.Advance()
	assert.Equal(t, 1, e.FrameIndex())
}

func TestEngine_Advance_Paused(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	e.SetSize(20, 4)
	e.SetPlaying(false)
	e.Advance()
	assert.Equal(t, 0, e.FrameIndex())
}

func TestEngine_CurrentFrame_Paused_FreezesLastFrame(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	e.SetSize(20, 4)
	e.SetPlaying(true)
	e.Advance()
	playingFrame := e.CurrentFrame()
	require.NotEmpty(t, playingFrame)

	// Pause — frame should freeze at the same content as when playing
	e.SetPlaying(false)
	pausedFrame := e.CurrentFrame()
	require.Len(t, pausedFrame, len(playingFrame))
	for i := range pausedFrame {
		assert.Equal(t, playingFrame[i].Text, pausedFrame[i].Text,
			"paused frame row %d should match playing frame", i)
	}
}

func TestEngine_CurrentFrame_Playing_NotEmpty(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	e.SetSize(20, 4)
	e.SetPlaying(true)
	f := e.CurrentFrame()
	require.NotEmpty(t, f)
	// At least one line should have non-empty text
	hasContent := false
	for _, line := range f {
		if line.Text != "" {
			hasContent = true
			break
		}
	}
	assert.True(t, hasContent, "playing frame should have some content")
}

func TestEngine_FrameWrap(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	e.SetSize(20, 4)
	e.SetPlaying(true)
	// Advance 40 times to wrap around
	for i := 0; i < 40; i++ {
		e.Advance()
	}
	assert.Equal(t, 0, e.FrameIndex())
}

func TestEngine_CyclePattern(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	e.SetSize(20, 4)
	e.SetPlaying(true)
	e.CyclePattern()
	assert.Equal(t, 1, e.Pattern())
}

func TestEngine_CyclePattern_Wraps(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	e.SetSize(20, 4)
	for i := 0; i < 6; i++ {
		e.CyclePattern()
	}
	assert.Equal(t, 0, e.Pattern())
}

func TestEngine_CyclePattern_RegeneratesFrames(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	e.SetSize(20, 4)
	e.SetPlaying(true)
	e.CyclePattern()
	// After cycling, pattern should be 1
	assert.Equal(t, 1, e.Pattern())
}

func TestEngine_Init_NonNil(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	cmd := e.Init()
	assert.NotNil(t, cmd)
}

func TestEngine_Update_TickMsg_NonNil(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	e.SetSize(20, 4)
	e.SetPlaying(true)
	cmd := e.Update(TickMsg{})
	assert.NotNil(t, cmd)
}

func TestEngine_PerRowColors_GradientAssignment(t *testing.T) {
	th := theme.Load("black")
	e := NewEngine(th)
	height := 6
	e.SetSize(20, height)
	e.SetPlaying(true)
	f := e.CurrentFrame()
	require.Len(t, f, height)

	// Top third (rows 0..1) should use Gradient3
	for i := 0; i < 2; i++ {
		assert.Equal(t, th.Gradient3(), f[i].Color,
			"row %d should use Gradient3 (peaks)", i)
	}
	// Middle third (rows 2..3) should use Gradient2
	for i := 2; i < 4; i++ {
		assert.Equal(t, th.Gradient2(), f[i].Color,
			"row %d should use Gradient2 (mid)", i)
	}
	// Bottom third (rows 4..5) should use Gradient1
	for i := 4; i < 6; i++ {
		assert.Equal(t, th.Gradient1(), f[i].Color,
			"row %d should use Gradient1 (base)", i)
	}
}

func TestEngine_SetSize_Height1_GradientColor(t *testing.T) {
	th := theme.Load("black")
	e := NewEngine(th)
	e.SetSize(10, 1)
	e.SetPlaying(true)
	f := e.CurrentFrame()
	require.Len(t, f, 1)
	// Single row should use Gradient1 (bottom/base)
	assert.Equal(t, th.Gradient1(), f[0].Color)
}

func TestEngine_SetSize_Height0_NoFrame(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	e.SetSize(10, 0)
	f := e.CurrentFrame()
	assert.Empty(t, f)
}

func TestEngine_SetSize_RegeneratesFrames(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	e.SetSize(20, 4)
	e.SetPlaying(true)
	e.Advance()
	e.SetSize(40, 6) // resize — should reset frameIdx to 0
	assert.Equal(t, 0, e.FrameIndex())
}

func TestEngine_Advance_BeforeSetSize_NoPanic(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	assert.NotPanics(t, func() { e.Advance() })
}

func TestEngine_CurrentFrame_BeforeSetSize_EmptyFrame(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	f := e.CurrentFrame()
	assert.Empty(t, f)
}

// ---------------------------------------------------------------------------
// Task 6: Comprehensive per-pattern tests
// ---------------------------------------------------------------------------

func TestAllPatterns_FrameDimensions(t *testing.T) {
	th := theme.Load("black")
	ps := Patterns()
	width, height := 30, 6

	for i, p := range ps {
		t.Run(p.Name, func(t *testing.T) {
			e := NewEngine(th)
			// Cycle to target pattern
			for e.Pattern() != i {
				e.CyclePattern()
			}
			e.SetSize(width, height)
			e.SetPlaying(true)
			f := e.CurrentFrame()
			assert.Len(t, f, height, "pattern %d (%s): wrong frame height", i, p.Name)
			for _, line := range f {
				assert.Equal(t, width, utf8.RuneCountInString(line.Text),
					"pattern %d (%s): wrong line width", i, p.Name)
			}
		})
	}
}

func TestAllPatterns_PlayingNonEmpty(t *testing.T) {
	th := theme.Load("black")
	ps := Patterns()

	for i, p := range ps {
		t.Run(p.Name, func(t *testing.T) {
			e := NewEngine(th)
			for e.Pattern() != i {
				e.CyclePattern()
			}
			e.SetSize(20, 4)
			e.SetPlaying(true)
			f := e.CurrentFrame()
			assert.NotEmpty(t, f, "pattern %d (%s): playing frame should not be empty", i, p.Name)
		})
	}
}

func TestEngine_CurrentFrame_Paused_CyclePattern(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	e.SetSize(20, 4)
	e.SetPlaying(false)
	// Should not panic and should return a valid frame
	e.CyclePattern()
	f := e.CurrentFrame()
	assert.Len(t, f, 4, "paused cycle should still produce a frame")
}

func TestBraillePatterns_OnlyBrailleRunes(t *testing.T) {
	// Pin to unicode mode: the engine falls back to AsciiBarsRenderer in ASCII mode,
	// so braille-pattern assertions are only meaningful in unicode mode.
	prev := uikit.ActiveMode()
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(prev)

	th := theme.Load("black")
	braillePatterns := []int{0, 1}

	for _, idx := range braillePatterns {
		ps := Patterns()
		p := ps[idx]
		t.Run(p.Name, func(t *testing.T) {
			e := NewEngine(th)
			for e.Pattern() != idx {
				e.CyclePattern()
			}
			e.SetSize(20, 4)
			e.SetPlaying(true)
			f := e.CurrentFrame()
			for _, line := range f {
				for _, ch := range line.Text {
					assert.True(t, ch >= '\u2800' && ch <= '\u28FF',
						"braille pattern %d: unexpected rune %U", idx, ch)
				}
			}
		})
	}
}

func TestBlockPatterns_OnlyBlockOrSpace(t *testing.T) {
	// Pin to unicode mode: the engine falls back to AsciiBarsRenderer in ASCII mode,
	// so block-pattern assertions are only meaningful in unicode mode.
	prev := uikit.ActiveMode()
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(prev)

	th := theme.Load("black")
	blockPatterns := []int{2}
	fillGlyph := []rune(uikit.GlyphFor(uikit.GlyphBarFull, uikit.ActiveMode()))[0]

	for _, idx := range blockPatterns {
		ps := Patterns()
		p := ps[idx]
		t.Run(p.Name, func(t *testing.T) {
			e := NewEngine(th)
			for e.Pattern() != idx {
				e.CyclePattern()
			}
			e.SetSize(20, 4)
			e.SetPlaying(true)
			f := e.CurrentFrame()
			for _, line := range f {
				for _, ch := range line.Text {
					assert.True(t, ch == fillGlyph || ch == ' ',
						"block pattern %d: unexpected rune %U (%c)", idx, ch, ch)
				}
			}
		})
	}
}

// TestEngine_ASCIIMode_BlockPatterns_OnlyASCIIChars confirms that in ASCII mode
// the engine routes all patterns to AsciiBarsRenderer and emits only
// { '#', '=', '-', '.', ' ' } — the ASCII block chars.
func TestEngine_ASCIIMode_BlockPatterns_OnlyASCIIChars(t *testing.T) {
	prev := uikit.ActiveMode()
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(prev)

	th := theme.Load("black")
	// Use block-pattern index (2) as the representative set; in ASCII
	// mode the engine falls back to AsciiBarsRenderer for ALL patterns.
	blockPatterns := []int{2}
	for _, idx := range blockPatterns {
		ps := Patterns()
		p := ps[idx]
		t.Run(p.Name, func(t *testing.T) {
			e := NewEngine(th)
			for e.Pattern() != idx {
				e.CyclePattern()
			}
			e.SetSize(20, 4)
			e.SetPlaying(true)
			f := e.CurrentFrame()
			for _, line := range f {
				for _, ch := range line.Text {
					assert.True(t,
						ch == '#' || ch == '=' || ch == '-' || ch == '.' || ch == ' ',
						"ASCII mode block pattern %d: unexpected rune %U (%c)", idx, ch, ch)
				}
			}
		})
	}
}

func TestAllPatterns_ColorGradient(t *testing.T) {
	th := theme.Load("black")
	ps := Patterns()
	height := 6

	for i, p := range ps {
		t.Run(p.Name, func(t *testing.T) {
			e := NewEngine(th)
			for e.Pattern() != i {
				e.CyclePattern()
			}
			e.SetSize(20, height)
			e.SetPlaying(true)
			f := e.CurrentFrame()
			require.Len(t, f, height)
			// Top rows: Gradient3
			assert.Equal(t, th.Gradient3(), f[0].Color)
			// Bottom rows: Gradient1
			assert.Equal(t, th.Gradient1(), f[height-1].Color)
		})
	}
}

func TestAllPatterns_Deterministic(t *testing.T) {
	th := theme.Load("black")
	ps := Patterns()

	for i, p := range ps {
		t.Run(p.Name, func(t *testing.T) {
			// Create two engines with same pattern at same frame
			e1 := NewEngine(th)
			e2 := NewEngine(th)
			for e1.Pattern() != i {
				e1.CyclePattern()
			}
			for e2.Pattern() != i {
				e2.CyclePattern()
			}
			e1.SetSize(20, 4)
			e2.SetSize(20, 4)
			e1.SetPlaying(true)
			e2.SetPlaying(true)
			f1 := e1.CurrentFrame()
			f2 := e2.CurrentFrame()
			assert.Equal(t, f1, f2, "pattern %d should be deterministic", i)
		})
	}
}

func TestAllPatterns_DifferentFramesVary(t *testing.T) {
	th := theme.Load("black")
	ps := Patterns()

	for i, p := range ps {
		t.Run(p.Name, func(t *testing.T) {
			e := NewEngine(th)
			for e.Pattern() != i {
				e.CyclePattern()
			}
			e.SetSize(20, 4)
			e.SetPlaying(true)
			frames := make([]Frame, 0, 10)
			for j := 0; j < 10; j++ {
				frames = append(frames, cloneFrame(e.CurrentFrame()))
				e.Advance()
			}
			// At least one frame should differ from frame 0
			varied := false
			for j := 1; j < len(frames); j++ {
				if !framesEqual(frames[0], frames[j]) {
					varied = true
					break
				}
			}
			assert.True(t, varied, "pattern %d (%s): all 10 frames are identical", i, p.Name)
		})
	}
}

func TestFullLifecycle(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	assert.Equal(t, 0, e.FrameIndex())

	e.SetSize(30, 5)
	e.SetPlaying(true)

	for i := 0; i < 15; i++ {
		e.Advance()
	}
	assert.Equal(t, 15, e.FrameIndex())

	f := e.CurrentFrame()
	assert.Len(t, f, 5)
}

func TestPatternCycleAll(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	e.SetSize(20, 4)
	seen := make([]int, 0, 7)
	seen = append(seen, e.Pattern())
	for i := 0; i < 6; i++ {
		e.CyclePattern()
		seen = append(seen, e.Pattern())
	}
	assert.Equal(t, 0, seen[6], "after 6 cycles should wrap to 0")
}

func TestResizeMidAnimation(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	e.SetSize(20, 4)
	e.SetPlaying(true)
	e.Advance()
	e.Advance()
	e.Advance()
	e.SetSize(40, 6)
	// After resize, frameIdx should reset
	assert.Equal(t, 0, e.FrameIndex())
	f := e.CurrentFrame()
	assert.Len(t, f, 6)
}

func TestEdge_Width1(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	e.SetSize(1, 4)
	e.SetPlaying(true)
	assert.NotPanics(t, func() {
		f := e.CurrentFrame()
		assert.Len(t, f, 4)
		for _, line := range f {
			assert.Equal(t, 1, utf8.RuneCountInString(line.Text))
		}
	})
}

func TestEdge_Height1(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	e.SetSize(20, 1)
	e.SetPlaying(true)
	assert.NotPanics(t, func() {
		f := e.CurrentFrame()
		assert.Len(t, f, 1)
	})
}

func TestEdge_LargeDimensions(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	assert.NotPanics(t, func() {
		e.SetSize(200, 20)
		e.SetPlaying(true)
		e.Advance()
		f := e.CurrentFrame()
		assert.Len(t, f, 20)
	})
}

func TestEngine_Update_NonTickMsg_ReturnsNil(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	e.SetSize(20, 4)
	// A non-TickMsg should return nil (no re-arm)
	cmd := e.Update("some-other-message")
	assert.Nil(t, cmd)
}

func TestEngine_SetSize_SameDimensions_NoReset(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	e.SetSize(20, 4)
	e.SetPlaying(true)
	e.Advance()
	e.Advance()
	// Calling SetSize with identical dimensions should not reset frameIdx
	e.SetSize(20, 4)
	assert.Equal(t, 2, e.FrameIndex(), "same-dimensions SetSize must not reset frameIdx")
}

// ---------------------------------------------------------------------------
// Story 80: SetPattern
// ---------------------------------------------------------------------------

func TestSetPattern_ValidIndex(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	e.SetSize(20, 4)
	e.SetPattern(2)
	assert.Equal(t, 2, e.Pattern())
}

func TestSetPattern_OutOfRange_Wraps(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	// 6 patterns → index 8 wraps to 8 % 6 = 2
	e.SetPattern(8)
	assert.Equal(t, 2, e.Pattern())
}

func TestSetPattern_Negative_ClampsToZero(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	e.SetPattern(-1)
	assert.Equal(t, 0, e.Pattern())
}

func TestSetPattern_RegeneratesFrames(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	e.SetSize(20, 4)
	e.SetPlaying(true)
	e.Advance()
	// After SetPattern the frame index resets to 0 and frames are regenerated.
	e.SetPattern(2)
	assert.Equal(t, 0, e.FrameIndex(), "SetPattern should reset frameIdx to 0")
	f := e.CurrentFrame()
	assert.Len(t, f, 4, "SetPattern should regenerate frames with correct height")
}

func TestSetPattern_NoSizeYet_NoPanic(t *testing.T) {
	e := NewEngine(theme.Load("black"))
	assert.NotPanics(t, func() { e.SetPattern(2) })
	assert.Equal(t, 2, e.Pattern(), "pattern should be set even before SetSize")
}

// ---------------------------------------------------------------------------
// MaxHeight interface method (Issue 1 fix)
// ---------------------------------------------------------------------------

func TestBrailleRenderer_MaxHeight(t *testing.T) {
	r := BrailleRenderer{}
	tests := []struct {
		displayHeight int
		want          int
	}{
		{displayHeight: 0, want: 0},
		{displayHeight: 1, want: 4},
		{displayHeight: 4, want: 16},
		{displayHeight: 10, want: 40},
	}
	for _, tt := range tests {
		got := r.MaxHeight(tt.displayHeight)
		assert.Equal(t, tt.want, got,
			"BrailleRenderer.MaxHeight(%d) should be %d", tt.displayHeight, tt.want)
	}
}

func TestBlockRenderer_MaxHeight(t *testing.T) {
	r := BlockRenderer{}
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
			"BlockRenderer.MaxHeight(%d) should be %d", tt.displayHeight, tt.want)
	}
}

// ---------------------------------------------------------------------------
// NewEngine nil-theme guard (Issue 2 fix)
// ---------------------------------------------------------------------------

func TestNewEngine_NilTheme_UsesDefault(t *testing.T) {
	assert.NotPanics(t, func() {
		e := NewEngine(nil)
		// Engine should be functional with default theme.
		e.SetSize(20, 4)
		e.SetPlaying(true)
		f := e.CurrentFrame()
		assert.Len(t, f, 4, "nil-theme engine should produce correct frame height")
	})
}

// ---------------------------------------------------------------------------
// Story 189: Engine selects AsciiBarsRenderer in ASCII mode
// ---------------------------------------------------------------------------

// TestEngine_SelectsAsciiRendererInAsciiMode confirms that when
// uikit.ActiveMode() == uikit.GlyphASCII the engine uses AsciiBarsRenderer
// and the resulting frame contains only ASCII bar characters.
func TestEngine_SelectsAsciiRendererInAsciiMode(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	e := NewEngine(theme.Load("black"))
	e.SetSize(10, 4)
	e.SetPlaying(true)
	f := e.CurrentFrame()
	require.Len(t, f, 4)

	for rowIdx, line := range f {
		for _, ch := range line.Text {
			assert.True(t, ch == '#' || ch == '=' || ch == '.' || ch == ' ',
				"row %d: ASCII mode should only produce # = . or space, got %U (%c)",
				rowIdx, ch, ch)
		}
	}
}

// TestEngine_SelectsUnicodeRendererInUnicodeMode confirms that in unicode mode
// the engine does NOT use the ASCII renderer — frames should contain braille or
// block characters from the current pattern.
func TestEngine_SelectsUnicodeRendererInUnicodeMode(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	e := NewEngine(theme.Load("black"))
	e.SetSize(10, 4)
	e.SetPlaying(true)
	f := e.CurrentFrame()
	require.Len(t, f, 4)

	// Default pattern 0 is braille — should contain braille runes.
	for rowIdx, line := range f {
		for _, ch := range line.Text {
			assert.False(t, ch == '#' || ch == '=',
				"row %d: unicode mode should not contain ASCII bar chars, got %U (%c)",
				rowIdx, ch, ch)
		}
	}

	// Positive assertion: at least one braille rune (U+2800–U+28FF) must appear.
	// A regression that returned all-spaces would still satisfy the negative check
	// above; this assertion catches that.
	hasBraille := false
	for _, line := range f {
		for _, ch := range line.Text {
			if ch >= '⠀' && ch <= '⣿' {
				hasBraille = true
				break
			}
		}
		if hasBraille {
			break
		}
	}
	assert.True(t, hasBraille, "unicode mode should produce at least one braille rune from BrailleRenderer")
}

// TestEngine_SelectsRenderer_AfterModeFlipAndCyclePattern is the regression
// test for story 193 task 5. selectRenderer is read at regen-time (inside
// generateFrames) rather than frame-time, so the new renderer only takes
// effect on the next regeneration (SetSize / CyclePattern / SetPattern).
// This test verifies that a mode flip followed by CyclePattern causes the
// engine to switch renderers:
//   - Start in unicode mode → braille renderer active after SetSize.
//   - Flip to ASCII mode → renderer not yet switched (frames are cached).
//   - Call CyclePattern → triggers generateFrames → AsciiBarsRenderer wins.
//   - Resulting frames must contain only ASCII bar characters.
func TestEngine_SelectsRenderer_AfterModeFlipAndCyclePattern(t *testing.T) {
	// Start in unicode mode.
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	e := NewEngine(theme.Load("black"))
	e.SetSize(10, 4)
	e.SetPlaying(true)

	// Sanity: in unicode mode, the first pattern is braille.
	f := e.CurrentFrame()
	require.Len(t, f, 4)
	hasBraille := false
	for _, line := range f {
		for _, ch := range line.Text {
			if ch >= '⠀' && ch <= '⣿' {
				hasBraille = true
				break
			}
		}
	}
	assert.True(t, hasBraille, "precondition: unicode mode SetSize should produce braille frames")

	// Flip mode to ASCII without resizing — cached frames stay as braille.
	uikit.SetModeForTest(uikit.GlyphASCII)

	// Trigger regen via CyclePattern: generateFrames calls selectRenderer which
	// now reads GlyphASCII and returns AsciiBarsRenderer.
	e.CyclePattern()

	fAscii := e.CurrentFrame()
	require.Len(t, fAscii, 4)
	for rowIdx, line := range fAscii {
		for _, ch := range line.Text {
			assert.True(t,
				ch == '#' || ch == '=' || ch == '-' || ch == '.' || ch == ' ',
				"row %d: after mode-flip + CyclePattern expected ASCII bar chars, got %U (%c)",
				rowIdx, ch, ch)
		}
	}
}

// ---------------------------------------------------------------------------
// New HeightFunc tests
// ---------------------------------------------------------------------------

func TestHeightWinampEQ_Range(t *testing.T) {
	for f := 0; f < 40; f++ {
		out := heightWinampEQ(20, 16, f)
		assert.Len(t, out, 20)
		for _, h := range out {
			assert.True(t, h >= 0 && h <= 16, "height %d out of range", h)
		}
	}
}

func TestHeightWinampEQ_Deterministic(t *testing.T) {
	a := heightWinampEQ(20, 16, 7)
	b := heightWinampEQ(20, 16, 7)
	assert.Equal(t, a, b)
}

func TestHeightWinampEQ_DifferentFromBlockSparse(t *testing.T) {
	wq := heightWinampEQ(40, 32, 10)
	bs := heightBlockSparse(40, 32, 10)
	assert.NotEqual(t, wq, bs, "Winamp EQ should have different profile than Block Sparse")
}

func TestHeightMatrixRain_Range(t *testing.T) {
	for f := 0; f < 40; f++ {
		out := heightMatrixRain(20, 16, f)
		assert.Len(t, out, 20)
		for _, h := range out {
			assert.True(t, h >= 1 && h <= 16,
				"height %d out of range [1,16]", h)
		}
	}
}

func TestHeightMatrixRain_Deterministic(t *testing.T) {
	a := heightMatrixRain(20, 16, 7)
	b := heightMatrixRain(20, 16, 7)
	assert.Equal(t, a, b)
}

func TestHeightMatrixRain_ColumnStagger(t *testing.T) {
	// Adjacent columns should have different heights (staggered phase)
	out := heightMatrixRain(20, 16, 0)
	differences := 0
	for i := 1; i < len(out); i++ {
		if out[i] != out[i-1] {
			differences++
		}
	}
	assert.Greater(t, differences, 0,
		"adjacent columns should have different heights due to phase stagger")
}

func TestHeightSpectrumSweep_Range(t *testing.T) {
	for f := 0; f < 40; f++ {
		out := heightSpectrumSweep(20, 16, f)
		assert.Len(t, out, 20)
		for _, h := range out {
			assert.True(t, h >= 0 && h <= 16, "height %d out of range", h)
		}
	}
}

func TestHeightSpectrumSweep_Deterministic(t *testing.T) {
	a := heightSpectrumSweep(20, 16, 7)
	b := heightSpectrumSweep(20, 16, 7)
	assert.Equal(t, a, b)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeColors(n int) []lipgloss.Color {
	colors := make([]lipgloss.Color, n)
	for i := range colors {
		colors[i] = lipgloss.Color("#ffffff")
	}
	return colors
}

func makeColHeights(width, maxH int) []int {
	h := make([]int, width)
	for i := range h {
		h[i] = maxH / 2
	}
	return h
}

func equalSlices(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func cloneFrame(f Frame) Frame {
	cp := make(Frame, len(f))
	copy(cp, f)
	return cp
}

func framesEqual(a, b Frame) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Text != b[i].Text {
			return false
		}
	}
	return true
}
