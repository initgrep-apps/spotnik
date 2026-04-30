// Package panes — HelpOverlay is the floating keybinding reference overlay.
// It displays all app keybindings grouped into four sections (Global, Navigation,
// Playback, Pane Actions) in a two-column layout inside a btop-style border.
package panes

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
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
// Wide enough for "Shift+Tab" with comfortable gap before the label.
const keyColWidth = 16

// colPadLeft is the left-indent applied inside each column so content doesn't
// sit flush against the outer border or the centre divider.
const colPadLeft = 2

// buildHelpContent returns the two-column keybinding reference with glyphs
// resolved via uikit.ActiveMode() so arrow keys render as ASCII alternatives
// when running in ASCII mode. Call this in View(), not at package init.
// [0] = left column (Global, Navigation), [1] = right column (Playback, Pane Actions).
// NOTE: When changing any keybinding, also update docs/keybinding.md and docs/DESIGN.md §17.
func buildHelpContent() [2][]helpSection {
	m := uikit.ActiveMode()
	al := uikit.GlyphFor(uikit.GlyphArrowLeft, m)
	ar := uikit.GlyphFor(uikit.GlyphArrowRight, m)
	au := uikit.GlyphFor(uikit.GlyphArrowUp, m)
	ad := uikit.GlyphFor(uikit.GlyphArrowDown, m)
	return [2][]helpSection{
		{
			{title: "Global", bindings: []helpBinding{
				{"/", "Search"}, {"d", "Devices"}, {"u", "Profile"}, {"t", "Theme"}, {"?", "Help"},
				{"q", "Quit"}, {"0", "Toggle page"}, {"1-8 / 1-5", "Toggle pane (A/B)"}, {"p", "Preset"},
			}},
			{title: "Navigation", bindings: []helpBinding{
				{"Tab", "Next pane"}, {"Shift+Tab", "Prev pane"},
				{"Esc", "Close / clear"},
			}},
		},
		{
			{title: "Playback", bindings: []helpBinding{
				{"Space", "Play / Pause"}, {al + " / " + ar, "Prev / Next"},
				{"+  / -", "Volume"}, {"s", "Shuffle"}, {"r", "Repeat"}, {"v", "Visualizer"},
			}},
			{title: "Pane Actions", bindings: []helpBinding{
				{"Enter", "Select / Play"}, {"f", "Filter"}, {"g", "Cycle time range"},
				{au + " / k", "Scroll up"}, {ad + " / j", "Scroll down"},
			}},
			{title: "Profile Overlay", bindings: []helpBinding{
				{"l", "Logout"},
				{"f", "Forget"},
			}},
		},
	}
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

// overlayWidth returns the fixed overlay width (100), capped to the terminal width
// when the terminal is narrower than 100 columns.
func (o *HelpOverlay) overlayWidth() int {
	const fixedWidth = 100
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
	innerW := max(totalW-2, 2)

	// Split inner width: divider takes 1 col, left and right share the rest.
	// leftW = floor((innerW-1)/2), rightW = innerW-1-leftW.
	leftW := (innerW - 1) / 2
	rightW := innerW - 1 - leftW

	content := buildHelpContent()
	leftLines := strings.Split(o.renderColumn(content[0], leftW), "\n")
	rightLines := strings.Split(o.renderColumn(content[1], rightW), "\n")

	// Pad the shorter column to match heights so the divider is aligned.
	for len(leftLines) < len(rightLines) {
		leftLines = append(leftLines, strings.Repeat(" ", leftW))
	}
	for len(rightLines) < len(leftLines) {
		rightLines = append(rightLines, strings.Repeat(" ", rightW))
	}

	divider := lipgloss.NewStyle().Foreground(o.theme.TextMuted()).Render(uikit.GlyphFor(uikit.GlyphVRule, uikit.ActiveMode()))

	var rows []string
	for i := range leftLines {
		rows = append(rows, leftLines[i]+divider+rightLines[i])
	}

	inner := strings.Join(rows, "\n")
	inner = lipgloss.NewStyle().
		Width(innerW).MaxWidth(innerW).
		Render(inner)

	chrome := uikit.OverlayChrome{
		Width:  totalW,
		Height: len(rows) + 2, // +2 for top and bottom border rows
		Title:  "Help",
		Theme:  o.theme,
	}
	return chrome.Render(inner)
}

// renderColumn renders one side of the two-column layout.
// Each section title is rendered with Info()+bold; each binding row uses a fixed
// keyColWidth sub-column for the key name and the remainder for the label.
// colPadLeft spaces of left indent keep content away from the border/divider.
func (o *HelpOverlay) renderColumn(sections []helpSection, width int) string {
	headerStyle := lipgloss.NewStyle().
		Foreground(o.theme.Info()).
		Bold(true)
	// Keys use KeyHint — the same token as status-bar key labels — so they stand
	// out from the muted label descriptions while remaining on-theme.
	keyStyle := lipgloss.NewStyle().Foreground(o.theme.KeyHint()).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(o.theme.TextMuted())

	pad := strings.Repeat(" ", colPadLeft)
	// contentW is the usable width after left padding; labelW fills what keyCol doesn't use.
	contentW := max(width-colPadLeft, 1)
	labelW := max(contentW-keyColWidth, 1)

	// Top padding — one blank line before the first section.
	var lines []string
	lines = append(lines, strings.Repeat(" ", width))

	for i, sec := range sections {
		// Two blank separator lines between sections for visual breathing room.
		if i > 0 {
			lines = append(lines, strings.Repeat(" ", width))
			lines = append(lines, strings.Repeat(" ", width))
		}
		header := lipgloss.NewStyle().Width(width).MaxWidth(width).
			Render(pad + headerStyle.Render(sec.title))
		lines = append(lines, header)

		for _, b := range sec.bindings {
			keyPart := lipgloss.NewStyle().
				Width(keyColWidth).MaxWidth(keyColWidth).
				Render(keyStyle.Render(b.key))
			labelPart := lipgloss.NewStyle().
				Width(labelW).MaxWidth(labelW).
				Render(labelStyle.Render(b.label))
			row := lipgloss.NewStyle().Width(width).MaxWidth(width).
				Render(pad + keyPart + labelPart)
			lines = append(lines, row)
		}
	}

	// Bottom padding — one blank line after the last section.
	lines = append(lines, strings.Repeat(" ", width))

	return strings.Join(lines, "\n")
}
