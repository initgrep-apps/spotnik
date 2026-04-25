package uikit

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// EmptyState is shown when a pane has nothing to display. Text is centered
// vertically and horizontally in the provided rectangle; an optional Hint
// renders below Text. Both Text and Hint are rendered in the Muted role.
//
// Render() returns exactly Height newline-separated lines.
type EmptyState struct {
	// Text is the primary no-data message (e.g. "Empty queue").
	Text string
	// Hint is the optional secondary help text rendered below Text.
	Hint string
	// Width is the column width of the rendered output.
	Width int
	// Height is the number of lines in the rendered output.
	Height int
	// Theme provides colour tokens.
	Theme theme.Theme
}

// Render centers Text (and Hint below it) both horizontally and vertically
// within the Height×Width rectangle. Both are styled in the Muted role.
// Returns exactly Height newline-joined lines.
func (e EmptyState) Render() string {
	if e.Height <= 0 {
		return ""
	}

	mutedStyle := lipgloss.NewStyle().Foreground(e.Theme.TextMuted())

	// Build the body lines (text + optional hint).
	body := mutedStyle.Render(e.Text)
	if e.Hint != "" {
		body = body + "\n" + mutedStyle.Render(e.Hint)
	}

	bodyLines := strings.Split(body, "\n")
	bodyHeight := len(bodyLines)

	// Calculate top padding for vertical centering.
	// Clamp to zero so a body larger than Height still renders without blank top lines.
	topPad := max(0, (e.Height-bodyHeight)/2)

	lines := make([]string, 0, e.Height)

	// Pad above.
	blankLine := strings.Repeat(" ", e.Width)
	for i := 0; i < topPad; i++ {
		lines = append(lines, blankLine)
	}

	// Append body lines, each horizontally centered.
	// Clamp to Height so overflowing body lines do not exceed the rectangle.
	for _, bl := range bodyLines {
		if len(lines) >= e.Height {
			break
		}
		lines = append(lines, centerLine(bl, e.Width))
	}

	// Pad below to reach exactly Height.
	for len(lines) < e.Height {
		lines = append(lines, blankLine)
	}

	return strings.Join(lines, "\n")
}

// centerLine pads a single rendered line with spaces so the visible content
// is horizontally centered in a column of width w. ANSI escape codes are
// not counted toward the visible width (lipgloss.Width handles this).
func centerLine(s string, w int) string {
	cur := lipgloss.Width(s)
	if cur >= w {
		return s
	}
	left := (w - cur) / 2
	right := w - cur - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}
