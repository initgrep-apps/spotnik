// Package panes — HelpOverlay is the floating keybinding reference overlay.
// It displays all app keybindings grouped into four sections (Global, Navigation,
// Playback, Pane Actions) in a two-column layout inside a btop-style border.
package panes

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// HelpOverlayClosedMsg is emitted when the user presses Esc in the HelpOverlay.
// The root app handles this by closing the overlay without any state change.
type HelpOverlayClosedMsg struct{}

// helpBinding is a single key → label pair displayed in the help overlay.
type helpBinding struct{ key, label string }

// helpSection groups related bindings under a titled header.
type helpSection struct {
	title    string
	bindings []helpBinding
}

// keyColWidth is the fixed width of the key sub-column in each binding row.
// Wide enough for "Shift+Tab" and "Shift+↑/↓" (9 visible chars + padding).
const keyColWidth = 12

// helpContent is the static two-column keybinding reference.
// [0] = left column (Global, Navigation), [1] = right column (Playback, Pane Actions).
// NOTE: When changing any keybinding, also update docs/keybinding.md and docs/DESIGN.md §17.
var helpContent = [2][]helpSection{
	{
		{title: "Global", bindings: []helpBinding{
			{"/", "search"}, {"d", "devices"}, {"t", "theme"}, {"?", "help"},
			{"q", "quit"}, {"0", "toggle page"}, {"1-8", "toggle pane"}, {"p", "preset"},
		}},
		{title: "Navigation", bindings: []helpBinding{
			{"Tab", "next pane"}, {"Shift+Tab", "prev pane"},
			{"j / k", "scroll"}, {"Esc", "close overlay"},
		}},
	},
	{
		{title: "Playback", bindings: []helpBinding{
			{"Space", "play / pause"}, {"n", "next track"}, {"← / →", "prev / next"},
			{"+  / -", "volume"}, {"s", "shuffle"}, {"r", "repeat"}, {"v", "visualizer"},
		}},
		{title: "Pane Actions", bindings: []helpBinding{
			{"Enter", "select / play"}, {"f", "filter"}, {"A", "add to queue"},
			{"i", "like / unlike"}, {"x", "remove track"}, {"Shift+↑/↓", "reorder (playlists)"},
		}},
	},
}

// HelpOverlay is the floating keybinding reference overlay model.
// Pressing Esc emits HelpOverlayClosedMsg; all other keys are consumed (modal).
type HelpOverlay struct {
	theme  theme.Theme
	width  int
	height int
}

// NewHelpOverlay creates a HelpOverlay using the given theme.
func NewHelpOverlay(th theme.Theme) *HelpOverlay {
	return &HelpOverlay{theme: th}
}

// SetSize updates the render dimensions for the overlay.
func (o *HelpOverlay) SetSize(width, height int) {
	o.width = width
	o.height = height
}

// SetTheme updates the overlay's own theme reference for runtime theme switching.
func (o *HelpOverlay) SetTheme(th theme.Theme) {
	o.theme = th
}

// Init satisfies tea.Model; no startup command needed.
func (o *HelpOverlay) Init() tea.Cmd { return nil }

// Update handles keyboard input for the help overlay.
// Esc emits HelpOverlayClosedMsg; all other keys are consumed with nil cmd (modal).
func (o *HelpOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return o, nil
	}
	if keyMsg.Type == tea.KeyEsc {
		return o, func() tea.Msg { return HelpOverlayClosedMsg{} }
	}
	// Consume all other keys — overlay is modal.
	return o, nil
}

// overlayWidth returns the fixed overlay width (78), capped to the terminal width
// when the terminal is narrower than 78 columns.
func (o *HelpOverlay) overlayWidth() int {
	const fixedWidth = 78
	if o.width > 0 && fixedWidth > o.width {
		return o.width
	}
	return fixedWidth
}

// View renders the help overlay as a two-column keybinding reference inside
// a btop-style border. Centered via btoverlay.Composite() at the call site.
func (o *HelpOverlay) View() string {
	totalW := o.overlayWidth()
	// innerW excludes the two border columns (│ on each side).
	innerW := totalW - 2
	if innerW < 2 {
		innerW = 2
	}

	// Split inner width: divider takes 1 col, left and right share the rest.
	// leftW = floor((innerW-1)/2), rightW = innerW-1-leftW.
	leftW := (innerW - 1) / 2
	rightW := innerW - 1 - leftW

	leftLines := strings.Split(o.renderColumn(helpContent[0], leftW), "\n")
	rightLines := strings.Split(o.renderColumn(helpContent[1], rightW), "\n")

	// Pad the shorter column to match heights so the divider is aligned.
	for len(leftLines) < len(rightLines) {
		leftLines = append(leftLines, strings.Repeat(" ", leftW))
	}
	for len(rightLines) < len(leftLines) {
		rightLines = append(rightLines, strings.Repeat(" ", rightW))
	}

	divider := lipgloss.NewStyle().Foreground(o.theme.TextMuted()).Render("│")

	var rows []string
	for i := range leftLines {
		rows = append(rows, leftLines[i]+divider+rightLines[i])
	}

	inner := strings.Join(rows, "\n")
	inner = lipgloss.NewStyle().
		Width(innerW).MaxWidth(innerW).
		Render(inner)

	cfg := layout.BorderConfig{
		Width:  totalW,
		Height: len(rows) + 2, // +2 for top and bottom border rows
		Title:  "Help",
		Actions: []layout.Action{
			{Key: "Esc", Label: "close"},
		},
		AccentColor: o.theme.ActiveBorder(),
		Focused:     true, // overlays are always focused
		Theme:       o.theme,
	}

	return layout.RenderPaneBorder(inner, cfg)
}

// renderColumn renders one side of the two-column layout.
// Each section title is rendered with Info()+bold; each binding row uses a fixed
// keyColWidth sub-column for the key name and the remainder for the label.
func (o *HelpOverlay) renderColumn(sections []helpSection, width int) string {
	headerStyle := lipgloss.NewStyle().
		Foreground(o.theme.Info()).
		Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(o.theme.TextPrimary())
	labelStyle := lipgloss.NewStyle().Foreground(o.theme.TextMuted())

	labelW := width - keyColWidth
	if labelW < 1 {
		labelW = 1
	}

	var lines []string
	for i, sec := range sections {
		// Blank separator between sections (not before the first one).
		if i > 0 {
			lines = append(lines, strings.Repeat(" ", width))
		}
		header := lipgloss.NewStyle().Width(width).MaxWidth(width).
			Render(headerStyle.Render(sec.title))
		lines = append(lines, header)

		for _, b := range sec.bindings {
			keyPart := lipgloss.NewStyle().
				Width(keyColWidth).MaxWidth(keyColWidth).
				Render(keyStyle.Render(b.key))
			labelPart := lipgloss.NewStyle().
				Width(labelW).MaxWidth(labelW).
				Render(labelStyle.Render(b.label))
			row := lipgloss.NewStyle().Width(width).MaxWidth(width).
				Render(keyPart + labelPart)
			lines = append(lines, row)
		}
	}

	return strings.Join(lines, "\n")
}
