package uikit

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// URLBox wraps a URL or code snippet in a muted-border rounded rectangle. The
// content is coloured in the Accent role. URLs longer than the box width are
// broken at '&' boundaries (preferred) or hard-wrapped when no '&' appears in
// the first half of the remaining segment.
//
// Roles:
//   - URLBox.Border → Muted (TextMuted colour token)
//   - URLBox.Content → Accent
type URLBox struct {
	// URL is the URL or code snippet to display.
	URL string
	// Width is the total column width of the rendered box including borders.
	Width int
	// Theme provides colour tokens.
	Theme theme.Theme
}

// Render returns the URL inside a styled box. In unicode mode a rounded border
// (╭╮╰╯─│) is used; in ASCII mode the border uses +/-/| characters.
// Long URLs are wrapped at '&' boundaries or hard-wrapped at the inner width.
func (b URLBox) Render() string {
	// border (1 each side) + padding (1 each side) = 4 chars overhead.
	innerW := b.Width - 4
	if innerW < 1 {
		innerW = 1
	}
	wrapped := wrapAtAmpersand(b.URL, innerW)

	mode := ActiveMode()
	var border lipgloss.Border
	if mode == GlyphASCII {
		border = lipgloss.Border{
			Top:         GlyphFor(GlyphHRule, mode),
			Bottom:      GlyphFor(GlyphHRule, mode),
			Left:        GlyphFor(GlyphVRule, mode),
			Right:       GlyphFor(GlyphVRule, mode),
			TopLeft:     GlyphFor(GlyphCornerTL, mode),
			TopRight:    GlyphFor(GlyphCornerTR, mode),
			BottomLeft:  GlyphFor(GlyphCornerBL, mode),
			BottomRight: GlyphFor(GlyphCornerBR, mode),
		}
	} else {
		border = lipgloss.RoundedBorder()
	}

	style := lipgloss.NewStyle().
		Border(border).
		BorderForeground(b.Theme.TextMuted()).
		Foreground(b.Theme.Accent()).
		Padding(0, 1).
		Width(b.Width - 2)
	return style.Render(wrapped)
}

// wrapAtAmpersand breaks u into lines of at most width runes. When a segment
// exceeds width, it scans back from width for the last '&' in the first half of
// the remaining text. If found, the cut is made at that position (keeping '&'
// at the end of the current line). Otherwise the segment is hard-wrapped at
// width.
func wrapAtAmpersand(u string, width int) string {
	if len(u) <= width {
		return u
	}
	var lines []string
	for len(u) > width {
		cut := width
		if i := strings.LastIndex(u[:width], "&"); i > width/2 {
			// Cut after '&' so '&' stays at the end of the current line.
			cut = i + 1
		}
		lines = append(lines, u[:cut])
		u = u[cut:]
	}
	if u != "" {
		lines = append(lines, u)
	}
	return strings.Join(lines, "\n")
}
