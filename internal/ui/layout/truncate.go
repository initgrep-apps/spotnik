package layout

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

// ellipsis is the single-character Unicode ellipsis (U+2026), 1 terminal column wide.
const ellipsis = "…"

// ellipsisWidth is the terminal column width of the ellipsis character.
const ellipsisWidth = 1

// Truncate truncates s to at most maxWidth terminal columns.
// If s is wider than maxWidth, it is truncated and "…" (U+2026) is appended.
// Uses lipgloss.Width() for accurate rendered-width measurement so that
// CJK characters (2 columns), emoji, combining marks, and ANSI escape
// sequences are all handled correctly.
func Truncate(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	w := lipgloss.Width(s)
	if w <= maxWidth {
		return s
	}

	// maxWidth == 1: we can only fit the ellipsis itself.
	if maxWidth <= ellipsisWidth {
		return ellipsis
	}

	// Walk the string rune-by-rune, tracking rendered width with lipgloss.
	// NOTE: lipgloss.Width measures full strings including ANSI sequences; we
	// therefore pass prefix slices so that ANSI state is included in each
	// measurement.  The byte position is used to slice the string efficiently.
	limit := maxWidth - ellipsisWidth
	bytePos := 0
	for bytePos < len(s) {
		_, size := utf8.DecodeRuneInString(s[bytePos:])
		candidate := s[:bytePos+size]
		if lipgloss.Width(candidate) > limit {
			break
		}
		bytePos += size
	}

	return s[:bytePos] + ellipsis
}

// PadRight pads s with trailing spaces so that its terminal column width
// equals width. If s is already at least width columns wide, it is returned
// unchanged (use Truncate first if you need to guarantee the output fits).
func PadRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

// TruncateOrPad ensures s is exactly width terminal columns wide.
// Long strings are truncated (with "…" appended); short strings are
// padded with trailing spaces. Equivalent to PadRight(Truncate(s, width), width).
func TruncateOrPad(s string, width int) string {
	return PadRight(Truncate(s, width), width)
}
