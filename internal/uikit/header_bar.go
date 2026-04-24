package uikit

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// HeaderBar renders the top app bar for Spotnik:
//
//	"spotnik ─ Page A ─ preset N"  [fill]  [chip1][chip2]...
//
// The left segment always contains AppName and Page. When Preset >= 0 the
// preset index is shown as well; set Preset to -1 to hide it (Page B layout).
// RightChips are pre-rendered strings from Chip.Render(); they are joined and
// placed flush-right with at least one space gap between left and right.
// The full bar is exactly Width terminal columns wide when Width > 0.
type HeaderBar struct {
	Width      int
	AppName    string
	Page       string   // "A" or "B"
	Preset     int      // -1 hides the preset segment (Page B)
	RightChips []string // pre-rendered chip strings from Chip.Render()
	Theme      theme.Theme
}

// Render produces the ANSI-styled header bar string.
func (h HeaderBar) Render() string {
	t := h.Theme
	bg := lipgloss.NewStyle().
		Background(t.StatusBarBg()).
		Foreground(t.StatusBarFg())

	appName := lipgloss.NewStyle().
		Background(t.StatusBarBg()).
		Foreground(t.TextPrimary()).
		Bold(true).
		Render(" " + h.AppName + " ")

	// Page label uses Accent colour per design §7.1 role table, preceded by "Page " in Muted.
	muted := lipgloss.NewStyle().
		Background(t.StatusBarBg()).
		Foreground(t.TextMuted())
	key := lipgloss.NewStyle().
		Background(t.StatusBarBg()).
		Foreground(ColourFor(RoleAccent, t)).
		Bold(true).
		Render(h.Page)

	sep := muted.Render(" ─ ")
	left := appName + sep + muted.Render("Page ") + key

	// Append preset segment only when Preset >= 0 (Page A).
	if h.Preset >= 0 {
		left += sep + muted.Render(fmt.Sprintf("preset %d", h.Preset))
	}

	right := strings.Join(h.RightChips, "")

	if h.Width > 0 {
		gap := h.Width - lipgloss.Width(left) - lipgloss.Width(right)
		if gap < 1 {
			gap = 1
		}
		return left + bg.Render(strings.Repeat(" ", gap)) + right
	}
	// Fallback when no width is known (tests without terminal size).
	return left + "  " + right
}
