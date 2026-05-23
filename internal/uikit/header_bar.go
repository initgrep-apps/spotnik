package uikit

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// HeaderBar renders the top app bar for Spotnik:
//
//	"spotnik ─ Music ─ preset N"  [fill]  [chip1][chip2]...
//
// The left segment always contains AppName and Page. When PresetName is non-empty
// the preset name is shown as well; set PresetName to "" to hide it (Stats page layout).
// RightChips are pre-rendered strings from Chip.Render(); they are joined and
// placed flush-right with at least one space gap between left and right.
// The full bar is exactly Width terminal columns wide when Width > 0.
type HeaderBar struct {
	Width      int
	AppName    string
	Page       string   // "Music" or "Stats"
	PresetName string   // empty string hides the preset segment (Stats page)
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

	// Page label uses Accent colour per design §7.1 role table.
	muted := lipgloss.NewStyle().
		Background(t.StatusBarBg()).
		Foreground(t.TextMuted())
	key := lipgloss.NewStyle().
		Background(t.StatusBarBg()).
		Foreground(ColourFor(RoleAccent, t)).
		Bold(true).
		Render(h.Page)

	sep := muted.Render(" " + GlyphFor(GlyphHRule, ActiveMode()) + " ")
	left := appName + sep + key

	// Append preset segment only when PresetName is non-empty (Music page).
	if h.PresetName != "" {
		left += sep + muted.Render(h.PresetName)
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
