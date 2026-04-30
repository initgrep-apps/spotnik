package uikit

import "github.com/charmbracelet/lipgloss"

// RoundedBorder returns a lipgloss.Border whose glyphs honour the active
// GlyphMode. In unicode mode this is the standard rounded border (╭╮╰╯─│);
// in ASCII mode corners become "+" and rules become "-"/"|".
//
// Use this anywhere a small inline bordered box is needed (warning panels,
// auth chips, form-field outlines, URL boxes, status indicators) — i.e.
// surfaces that are NOT routed through PaneChrome / OverlayChrome / Panel.
//
// Direct calls to lipgloss.RoundedBorder() (and friends) outside this package
// leak unicode glyphs into ASCII mode and are flagged by
// scripts/check-banned-glyphs.sh.
func RoundedBorder() lipgloss.Border {
	m := ActiveMode()
	if m == GlyphUnicode {
		return lipgloss.RoundedBorder()
	}
	return lipgloss.Border{
		Top:         GlyphFor(GlyphHRule, m),
		Bottom:      GlyphFor(GlyphHRule, m),
		Left:        GlyphFor(GlyphVRule, m),
		Right:       GlyphFor(GlyphVRule, m),
		TopLeft:     GlyphFor(GlyphCornerTL, m),
		TopRight:    GlyphFor(GlyphCornerTR, m),
		BottomLeft:  GlyphFor(GlyphCornerBL, m),
		BottomRight: GlyphFor(GlyphCornerBR, m),
	}
}
