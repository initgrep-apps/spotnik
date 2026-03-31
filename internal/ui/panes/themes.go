// Package panes — ThemeOverlay is the floating theme switcher overlay.
// It lists all available themes with color swatches and lets the user switch
// the active theme at runtime. Selecting a theme emits ThemeSwitchMsg.
package panes

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// ThemeSwitchMsg is emitted when the user selects a theme in the ThemeOverlay.
// The root app handles this by loading the new theme and propagating it to all panes.
type ThemeSwitchMsg struct {
	ThemeID string
}

// ThemeOverlayClosedMsg is emitted when the user presses Esc in the ThemeOverlay.
// The root app handles this by closing the overlay without changing the theme.
type ThemeOverlayClosedMsg struct{}

// ThemeOverlay is the floating theme-switcher overlay model.
// It renders a navigable list of all available themes with color swatches.
// Pressing Enter emits ThemeSwitchMsg; pressing Esc closes without change.
type ThemeOverlay struct {
	// themes is the full sorted list of available themes.
	themes []*theme.ConfigTheme
	// cursor is the highlighted row index.
	cursor int
	// currentID is the ID of the currently active theme (marked with ◉).
	currentID string
	// theme is the active theme used for rendering the overlay's own chrome.
	theme  theme.Theme
	width  int
	height int
}

// NewThemeOverlay creates a ThemeOverlay with the cursor positioned on currentID.
func NewThemeOverlay(themes []*theme.ConfigTheme, currentID string, th theme.Theme) *ThemeOverlay {
	return &ThemeOverlay{
		themes:    themes,
		cursor:    findThemeIndex(themes, currentID),
		currentID: currentID,
		theme:     th,
	}
}

// findThemeIndex returns the index of the theme with the given ID in themes.
// Returns 0 if the ID is not found.
func findThemeIndex(themes []*theme.ConfigTheme, id string) int {
	for i, t := range themes {
		if t.ID() == id {
			return i
		}
	}
	return 0
}

// SetSize updates the render dimensions for the overlay.
func (o *ThemeOverlay) SetSize(width, height int) {
	o.width = width
	o.height = height
}

// SetTheme updates the overlay's own theme reference for runtime theme switching.
func (o *ThemeOverlay) SetTheme(th theme.Theme) {
	o.theme = th
}

// Init satisfies tea.Model; no startup command needed.
func (o *ThemeOverlay) Init() tea.Cmd { return nil }

// Update handles keyboard input for the theme overlay.
func (o *ThemeOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return o, nil
	}
	return o.handleKey(keyMsg)
}

// handleKey processes keyboard input for the theme overlay.
func (o *ThemeOverlay) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyRunes && string(msg.Runes) == "j",
		msg.Type == tea.KeyDown:
		if o.cursor < len(o.themes)-1 {
			o.cursor++
		}
		return o, nil

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "k",
		msg.Type == tea.KeyUp:
		if o.cursor > 0 {
			o.cursor--
		}
		return o, nil

	case msg.Type == tea.KeyEnter:
		if len(o.themes) == 0 {
			return o, nil
		}
		selected := o.themes[o.cursor]
		id := selected.ID()
		return o, func() tea.Msg {
			return ThemeSwitchMsg{ThemeID: id}
		}

	case msg.Type == tea.KeyEsc:
		return o, func() tea.Msg { return ThemeOverlayClosedMsg{} }
	}
	return o, nil
}

// View renders the theme overlay with a btop-style border.
// Each row shows the theme name with a "◉" or "○" indicator and 5 color swatches.
func (o *ThemeOverlay) View() string {
	if len(o.themes) == 0 {
		return ""
	}

	totalWidth := o.overlayWidth()
	innerWidth := totalWidth - 2
	if innerWidth < 4 {
		innerWidth = 4
	}

	var lines []string
	for i, th := range o.themes {
		lines = append(lines, o.renderRow(i, th, innerWidth))
	}

	inner := strings.Join(lines, "\n")
	inner = lipgloss.NewStyle().
		Width(innerWidth).MaxWidth(innerWidth).
		Render(inner)

	renderedLines := strings.Split(inner, "\n")
	totalHeight := len(renderedLines) + 2 // +2 for top and bottom border rows
	if totalHeight < 4 {
		totalHeight = 4
	}

	cfg := layout.BorderConfig{
		Width:  totalWidth,
		Height: totalHeight,
		Title:  "Themes",
		Actions: []layout.Action{
			{Key: "Enter", Label: "select"},
		},
		AccentColor: o.theme.ActiveBorder(),
		Focused:     true, // overlays are always focused
		Theme:       o.theme,
	}

	return layout.RenderPaneBorder(inner, cfg)
}

// renderRow renders a single theme row with indicator and color swatches.
func (o *ThemeOverlay) renderRow(idx int, th *theme.ConfigTheme, innerWidth int) string {
	isCursor := idx == o.cursor
	isCurrent := th.ID() == o.currentID

	// Background color for the row. Non-cursor rows use Base() (effectively transparent
	// against the dimmed grid behind the overlay) so the cursor row with SelectedBg()
	// clearly stands out. Surface() would create an opaque block that obscures the grid.
	bg := o.theme.Base()
	if isCursor {
		bg = o.theme.SelectedBg()
	}

	// Indicator: ◉ for current theme (in Success color), ○ for others (in TextMuted).
	var indicator string
	var indicatorStyle lipgloss.Style
	if isCurrent {
		indicatorStyle = lipgloss.NewStyle().
			Foreground(o.theme.Success()).
			Background(bg)
		indicator = "◉"
	} else {
		indicatorStyle = lipgloss.NewStyle().
			Foreground(o.theme.TextMuted()).
			Background(bg)
		indicator = "○"
	}

	// Theme name.
	var nameStyle lipgloss.Style
	if isCursor {
		nameStyle = lipgloss.NewStyle().
			Foreground(o.theme.SelectedFg()).
			Background(bg)
	} else {
		nameStyle = lipgloss.NewStyle().
			Foreground(o.theme.TextPrimary()).
			Background(bg)
	}

	// Color swatches — 5 colored █ chars using the target theme's colors (not current).
	swatches := renderSwatches(th)

	row := indicatorStyle.Render(indicator) + " " +
		nameStyle.Render(th.Name()) +
		"  " +
		swatches

	// Pad entire row to inner width with the row's background color.
	rowStyle := lipgloss.NewStyle().Background(bg)
	return rowStyle.Render(lipgloss.NewStyle().
		Width(innerWidth).MaxWidth(innerWidth).
		Background(bg).
		Render(row))
}

// overlayWidth computes the overlay width based on the longest theme name plus swatch space.
// Minimum 40 columns.
func (o *ThemeOverlay) overlayWidth() int {
	minWidth := 40
	for _, th := range o.themes {
		// indicator (1) + space (1) + name + padding (2) + 5 swatches + space (2) each
		needed := 1 + 1 + len(th.Name()) + 2 + 5*2 + 4
		if needed > minWidth {
			minWidth = needed
		}
	}
	if o.width > 0 && minWidth > o.width {
		minWidth = o.width
	}
	return minWidth
}

// renderSwatches renders 5 colored █ characters using the target theme's colors.
// The swatches preview the theme's palette before the user switches to it.
func renderSwatches(t *theme.ConfigTheme) string {
	colors := []lipgloss.Color{
		t.ColumnPrimary(),
		t.ColumnSecondary(),
		t.ColumnTertiary(),
		t.PaneBorderNowPlaying(),
		t.ActiveBorder(),
	}
	var b strings.Builder
	for _, c := range colors {
		b.WriteString(lipgloss.NewStyle().Foreground(c).Render("█"))
		b.WriteString(" ")
	}
	return b.String()
}
