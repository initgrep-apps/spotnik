package uikit_test

import (
	"strings"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
)

// TestProgressBar_Unicode_HalfFilled verifies that a 50%-filled bar with width 20
// emits exactly 20 bar-character cells (full █ + empty ░), confirming total width.
func TestProgressBar_Unicode_HalfFilled(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	th := theme.Load("black")
	pb := uikit.ProgressBar{Width: 20, Progress: 0.5, Theme: th}
	bar := pb.Render()
	// Progress=0.5 is exactly 10 filled cells, no partial — 10 full + 10 empty = 20.
	assert.Equal(t, 20, strings.Count(bar, "█")+strings.Count(bar, "░"),
		"unicode half-filled bar must sum to exactly Width cells")
}

// TestProgressBar_ASCII_HalfFilled verifies that in ASCII mode a 50%-filled
// bar with width 20 emits 10 fill chars ("#") and 10 empty chars (".").
func TestProgressBar_ASCII_HalfFilled(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	th := theme.Load("black")
	pb := uikit.ProgressBar{Width: 20, Progress: 0.5, Theme: th}
	bar := pb.Render()
	assert.Equal(t, 10, strings.Count(bar, "#"),
		"ascii half-filled bar must have 10 fill chars (#)")
	assert.Equal(t, 10, strings.Count(bar, "."),
		"ascii half-filled bar must have 10 empty chars (.)")
}

// TestProgressBar_ClampsProgress verifies that progress values outside [0,1] are
// clamped: > 1.0 renders identically to 1.0, < 0.0 renders identically to 0.0.
func TestProgressBar_ClampsProgress(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	th := theme.Load("black")
	assert.Equal(t,
		uikit.ProgressBar{Width: 10, Progress: 2.0, Theme: th}.Render(),
		uikit.ProgressBar{Width: 10, Progress: 1.0, Theme: th}.Render(),
		"progress > 1.0 must clamp to 1.0")
	assert.Equal(t,
		uikit.ProgressBar{Width: 10, Progress: -0.5, Theme: th}.Render(),
		uikit.ProgressBar{Width: 10, Progress: 0.0, Theme: th}.Render(),
		"progress < 0.0 must clamp to 0.0")
}

// TestProgressBar_PartialBlock verifies that a progress producing a fractional
// cell emits exactly one partial-block glyph.
func TestProgressBar_PartialBlock(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	th := theme.Load("black")
	// Width=8, Progress=0.3125 → filledFloat=2.5 → filled=2, remainder=0.5 → ▌ (≥4/8).
	pb := uikit.ProgressBar{Width: 8, Progress: 0.3125, Theme: th}
	bar := pb.Render()
	stripped := stripANSI(bar)
	assert.Contains(t, stripped, "▌", "remainder == 4/8 must emit ▌")
}

// TestProgressBar_AllEmpty verifies a zero-progress bar has Width empty cells.
func TestProgressBar_AllEmpty(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	th := theme.Load("black")
	pb := uikit.ProgressBar{Width: 12, Progress: 0.0, Theme: th}
	bar := pb.Render()
	stripped := stripANSI(bar)
	assert.Equal(t, 12, strings.Count(stripped, "░"),
		"zero-progress bar must have Width empty cells")
}

// TestProgressBar_AllFull verifies a fully-filled bar has Width full cells.
func TestProgressBar_AllFull(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	th := theme.Load("black")
	pb := uikit.ProgressBar{Width: 12, Progress: 1.0, Theme: th}
	bar := pb.Render()
	stripped := stripANSI(bar)
	assert.Equal(t, 12, strings.Count(stripped, "█"),
		"fully-filled bar must have Width full cells")
}

// TestPartialGlyph_Thresholds exercises every threshold step in PartialGlyph.
func TestPartialGlyph_Thresholds(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	tests := []struct {
		name      string
		remainder float64
		want      string
	}{
		{"7/8", 7.0 / 8.0, "▉"},
		{"6/8", 6.0 / 8.0, "▊"},
		{"5/8", 5.0 / 8.0, "▋"},
		{"4/8", 4.0 / 8.0, "▌"},
		{"3/8", 3.0 / 8.0, "▍"},
		{"2/8", 2.0 / 8.0, "▎"},
		{"1/8_below", 1.0 / 8.0, "▏"},
		{"tiny", 0.01, "▏"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uikit.PartialGlyph(tt.remainder, uikit.GlyphUnicode)
			assert.Equal(t, tt.want, got,
				"PartialGlyph(%.4f, Unicode) should be %q", tt.remainder, tt.want)
		})
	}
}

// TestPartialGlyph_ASCIIThresholds verifies ASCII mode returns correct chars.
func TestPartialGlyph_ASCIIThresholds(t *testing.T) {
	tests := []struct {
		name      string
		remainder float64
		want      string
	}{
		{"≥7/8 → #", 7.0 / 8.0, "#"},
		{"≥6/8 → #", 6.0 / 8.0, "#"},
		{"≥5/8 → =", 5.0 / 8.0, "="},
		{"≥4/8 → =", 4.0 / 8.0, "="},
		{"≥3/8 → -", 3.0 / 8.0, "-"},
		{"≥2/8 → -", 2.0 / 8.0, "-"},
		{"tiny → .", 0.01, "."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uikit.PartialGlyph(tt.remainder, uikit.GlyphASCII)
			assert.Equal(t, tt.want, got,
				"PartialGlyph(%.4f, ASCII) should be %q", tt.remainder, tt.want)
		})
	}
}
