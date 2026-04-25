package uikit

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// ProgressBar renders a horizontal fill bar using partial-block glyphs for 1/8
// sub-character resolution. Used for the seek bar and volume bar.
//
// Unicode mode: fill █, partial ▏▎▍▌▋▊▉, empty ░.
// ASCII mode:   fill #, partial ./-/=/#, empty .
//
// The Fill colour is theme.Gradient1(); Empty is theme.TextMuted().
// For per-position gradient colouring (seek/volume bars), callers may build their
// own character loop using GlyphFor + PartialGlyph and apply lipgloss styles directly.
type ProgressBar struct {
	// Width is the number of terminal columns the bar should occupy.
	Width int
	// Progress is the fill fraction in [0.0, 1.0]. Values outside this range
	// are clamped.
	Progress float64
	// Theme provides colour tokens.
	Theme theme.Theme
}

// Render returns the ANSI-styled bar string.
// The returned string is exactly Width terminal columns wide (before ANSI escapes).
func (p ProgressBar) Render() string {
	if p.Progress < 0 {
		p.Progress = 0
	}
	if p.Progress > 1 {
		p.Progress = 1
	}
	m := ActiveMode()
	fillStyle := lipgloss.NewStyle().Foreground(p.Theme.Gradient1())
	emptyStyle := lipgloss.NewStyle().Foreground(p.Theme.TextMuted())
	full := GlyphFor(GlyphBarFull, m)
	empty := GlyphFor(GlyphBarEmpty, m)

	totalCells := p.Width
	filledFloat := p.Progress * float64(totalCells)
	filled := int(filledFloat)
	remainder := filledFloat - float64(filled)

	var sb strings.Builder
	// Full filled blocks.
	if filled > 0 {
		sb.WriteString(fillStyle.Render(strings.Repeat(full, filled)))
	}
	// Partial block at the boundary (only when there is a fractional cell).
	if remainder > 0 && filled < totalCells {
		sb.WriteString(fillStyle.Render(PartialGlyph(remainder, m)))
		filled++
	}
	// Empty cells.
	if rem := totalCells - filled; rem > 0 {
		sb.WriteString(emptyStyle.Render(strings.Repeat(empty, rem)))
	}
	return sb.String()
}

// PartialGlyph returns the bar glyph for a fractional remainder in (0, 1).
// The threshold mapping follows §5.7 of the TUI design system spec:
//
//	remainder ≥ 7/8 → ▉ / #
//	remainder ≥ 6/8 → ▊ / #
//	remainder ≥ 5/8 → ▋ / =
//	remainder ≥ 4/8 → ▌ / =
//	remainder ≥ 3/8 → ▍ / -
//	remainder ≥ 2/8 → ▎ / -
//	any            → ▏ / .
//
// GradientSeekBar and GradientVolumeBar use this helper to compose partial
// characters without duplicating threshold logic.
func PartialGlyph(remainder float64, m GlyphMode) string {
	switch {
	case remainder >= 7.0/8.0:
		return GlyphFor(GlyphBarSevenEighths, m)
	case remainder >= 6.0/8.0:
		return GlyphFor(GlyphBarThreeQuarters, m)
	case remainder >= 5.0/8.0:
		return GlyphFor(GlyphBarFiveEighths, m)
	case remainder >= 4.0/8.0:
		return GlyphFor(GlyphBarHalf, m)
	case remainder >= 3.0/8.0:
		return GlyphFor(GlyphBarThreeEighths, m)
	case remainder >= 2.0/8.0:
		return GlyphFor(GlyphBarQuarter, m)
	default:
		return GlyphFor(GlyphBarOneEighth, m)
	}
}
