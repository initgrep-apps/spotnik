package uikit

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// SectionLabel renders a caps label followed by a horizontal rule. It is used
// for sub-section headings in the Request Flow pane (GATEWAY, APP, GATEWAY LOG,
// SPOTIFY, AUTO-TRAFFIC). The accent colour is the parent pane's border token
// so each sub-section visually belongs to its pane.
//
// Render() returns exactly two newline-separated lines:
//
//	Line 1: " <Label> "  (bold, AccentColor)
//	Line 2: Width repetitions of ─  (AccentColor; '-' in ascii mode)
type SectionLabel struct {
	// Label is the section heading text (typically ALL-CAPS).
	Label string
	// Width is the total column width of the horizontal rule (line 2).
	Width int
	// AccentColor is the parent pane's border token used for both lines.
	AccentColor lipgloss.Color
	// Theme provides colour tokens (retained for future role-based extensions).
	Theme theme.Theme
}

// Render returns the two-line string: padded bold label + horizontal rule.
// The result always contains exactly one newline (joining the two lines).
func (s SectionLabel) Render() string {
	mode := ActiveMode()
	accentStyle := lipgloss.NewStyle().Foreground(s.AccentColor).Bold(true)

	// Line 1: " <Label> " styled in bold accent.
	labelLine := accentStyle.Render(" " + s.Label + " ")

	// Line 2: horizontal rule of Width characters using the canonical HRule glyph
	// (─ in unicode mode, - in ascii mode).
	ruleChar := GlyphFor(GlyphHRule, mode)

	w := s.Width
	if w <= 0 {
		w = 0
	}
	ruleStr := strings.Repeat(ruleChar, w)
	// Remove bold for the rule — just foreground.
	ruleLine := lipgloss.NewStyle().Foreground(s.AccentColor).Render(ruleStr)

	return labelLine + "\n" + ruleLine
}
