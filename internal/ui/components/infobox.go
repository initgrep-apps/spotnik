package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// InfoBox renders a bordered sub-pane with a title in the top border and
// vertically-centered content lines inside.  It uses rounded corners (╭╮╰╯)
// per project design rules and draws the border manually so that the border
// color can be swapped without relying on lipgloss box model constraints.
type InfoBox struct {
	th     theme.Theme
	width  int
	height int
}

// NewInfoBox creates an InfoBox using the given theme.
func NewInfoBox(th theme.Theme) *InfoBox {
	return &InfoBox{th: th}
}

// SetSize updates the total width and height of the box (including the border
// rows and columns themselves).
func (b *InfoBox) SetSize(w, h int) {
	b.width = w
	b.height = h
}

// Render returns the InfoBox as a multi-line string.
//
// title is rendered in the top border: ╭─ Title ─────────────╮
// lines contains the content to vertically-center inside the box.
// When focused is true the border is drawn in ActiveBorder(); otherwise
// InactiveBorder() is used.
//
// Content behaviour:
//   - Each line is truncated (with "…") to the inner width (width-2).
//   - If len(lines) exceeds the inner height (height-2) the excess is
//     truncated from the bottom — the top lines (track name, artist) are
//     always shown first.
//   - Remaining vertical space is distributed as topPad above and bottom
//     padding below to centre the block.
func (b *InfoBox) Render(title string, lines []string, focused bool) string {
	w := b.width
	h := b.height

	// Enforce a sane minimum so we never index out-of-range.
	if w < 4 {
		w = 4
	}
	if h < 2 {
		h = 2
	}

	innerW := w - 2 // subtract left and right border chars
	innerH := h - 2 // subtract top and bottom border rows

	// Choose border color based on focus state.
	var borderColor lipgloss.Color
	if focused {
		borderColor = b.th.ActiveBorder()
	} else {
		borderColor = b.th.InactiveBorder()
	}
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	// -----------------------------------------------------------------------
	// Top border: ╭─ Title ──────────────────────╮
	// -----------------------------------------------------------------------
	topBorder := buildTopBorder(title, w, borderStyle)

	// -----------------------------------------------------------------------
	// Bottom border: ╰──────────────────────────────╯
	// -----------------------------------------------------------------------
	bottomBorder := buildBottomBorder(w, borderStyle)

	// -----------------------------------------------------------------------
	// Interior rows
	// -----------------------------------------------------------------------

	// Truncate content to available inner height.
	content := lines
	if len(content) > innerH {
		content = content[:innerH]
	}

	// Vertical centering: spread remaining rows above and below.
	remaining := innerH - len(content)
	topPad := 0
	if remaining > 0 {
		topPad = remaining / 2
	}
	bottomPad := remaining - topPad

	borderChar := borderStyle.Render("│")

	var sb strings.Builder

	// Top padding rows.
	for i := 0; i < topPad; i++ {
		sb.WriteString(borderChar)
		sb.WriteString(strings.Repeat(" ", innerW))
		sb.WriteString(borderChar)
		sb.WriteString("\n")
	}

	// Content rows.
	for _, line := range content {
		// Truncate then pad to exactly innerW columns.
		cell := layout.TruncateOrPad(line, innerW)
		sb.WriteString(borderChar)
		sb.WriteString(cell)
		sb.WriteString(borderChar)
		sb.WriteString("\n")
	}

	// Bottom padding rows.
	for i := 0; i < bottomPad; i++ {
		sb.WriteString(borderChar)
		sb.WriteString(strings.Repeat(" ", innerW))
		sb.WriteString(borderChar)
		sb.WriteString("\n")
	}

	interiorRows := strings.TrimRight(sb.String(), "\n")

	return topBorder + "\n" + interiorRows + "\n" + bottomBorder
}

// buildTopBorder constructs the top border line with the title embedded.
//
// Format: ╭─ Title ──────────────────────╮
// The total visible width of the returned string equals w terminal columns.
func buildTopBorder(title string, w int, style lipgloss.Style) string {
	// The inner fill is w-2 columns (between ╭ and ╮).
	// We use: "─ " + title + " " + repeated "─" to fill the rest.
	innerW := w - 2
	if innerW < 0 {
		innerW = 0
	}

	// If title is non-empty we render "─ <title> " then pad with "─".
	var fill string
	if title == "" || innerW == 0 {
		fill = strings.Repeat("─", innerW)
	} else {
		prefix := "─ " + title + " "
		prefixWidth := lipgloss.Width(prefix)
		remaining := innerW - prefixWidth
		if remaining < 0 {
			// Title too long: truncate so the border still fits.
			// Reserve "─ " (2) + " " (1) + "─" (1) at minimum.
			maxTitleW := innerW - 4
			if maxTitleW < 0 {
				maxTitleW = 0
			}
			trimmed := layout.Truncate(title, maxTitleW)
			prefix = "─ " + trimmed + " "
			prefixWidth = lipgloss.Width(prefix)
			remaining = innerW - prefixWidth
			if remaining < 0 {
				remaining = 0
			}
		}
		fill = prefix + strings.Repeat("─", remaining)
	}

	return style.Render("╭" + fill + "╮")
}

// buildBottomBorder constructs the bottom border line.
//
// Format: ╰──────────────────────────────╯
func buildBottomBorder(w int, style lipgloss.Style) string {
	innerW := w - 2
	if innerW < 0 {
		innerW = 0
	}
	return style.Render("╰" + strings.Repeat("─", innerW) + "╯")
}
