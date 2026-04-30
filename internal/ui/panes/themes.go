// Package panes — ThemeOverlay is the floating theme switcher overlay.
// It lists all available themes with color swatches and lets the user switch
// the active theme at runtime. Selecting a theme emits ThemeSwitchMsg.
package panes

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
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
		if i > 0 {
			lines = append(lines, strings.Repeat(" ", innerWidth))
		}
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

	chrome := uikit.PaneChrome{
		Width:       totalWidth,
		Height:      totalHeight,
		Title:       "Themes",
		AccentColor: o.theme.ActiveBorder(),
		Focused:     true, // overlays are always focused
		Theme:       o.theme,
	}

	return chrome.Render(inner)
}

// renderRow renders a single theme row with indicator and color swatches.
// ListRow handles the indicator glyph, theme name, and optional "active" caption.
// Swatches and cursor-highlight background are applied outside ListRow so they
// remain independent of the primitive's internal layout.
func (o *ThemeOverlay) renderRow(idx int, th *theme.ConfigTheme, innerWidth int) string {
	isCursor := idx == o.cursor
	isCurrent := th.ID() == o.currentID

	// Swatches take fixed space; reserve that to give ListRow the remaining width.
	swatches := renderSwatches(th)
	swatchWidth := lipgloss.Width(swatches) + 2 // +2 for the leading "  " gap
	rowWidth := innerWidth - swatchWidth
	if rowWidth < 1 {
		rowWidth = 1
	}

	// Choose glyph and role based on whether this is the current theme.
	// Glyph alone signals active state — no caption needed.
	glyph := uikit.GlyphAvailable
	intent := uikit.RoleMuted
	if isCurrent {
		glyph = uikit.GlyphActive
		intent = uikit.RoleAccent
	}

	// Apply cursor background to ListRow theme when the row is highlighted.
	// Only override intent for non-active rows; the active row keeps RoleAccent
	// so the ◉ glyph retains its accent colour even when the cursor is on it.
	rowTheme := o.theme
	if isCursor && !isCurrent {
		intent = uikit.RoleSelection
	}

	// For the cursor row, propagate SelectedBg into each ListRow segment so the
	// highlight is visually continuous. Each segment closes with \x1b[0m; without
	// the bg baked into the segment style an outer wrapper shows gaps between the
	// glyph, label, and caption.
	var rowBg lipgloss.TerminalColor
	if isCursor {
		rowBg = o.theme.SelectedBg()
	}

	listRow := uikit.ListRow{
		Glyph:         glyph,
		Label:         th.Name(),
		Intent:        intent,
		Theme:         rowTheme,
		RowBackground: rowBg,
	}
	rowContent := listRow.Render(rowWidth) + "  " + swatches

	// Cursor row: pad to innerWidth with SelectedBg so trailing space is also
	// highlighted. The "  " + swatches area after ListRow still has no explicit
	// bg; the width-enforcing wrapper fills the remaining columns with the bg.
	if isCursor {
		bg := o.theme.SelectedBg()
		return lipgloss.NewStyle().
			Width(innerWidth).MaxWidth(innerWidth).
			Background(bg).
			Render(rowContent)
	}
	return lipgloss.NewStyle().
		Width(innerWidth).MaxWidth(innerWidth).
		Render(rowContent)
}

// overlayWidth computes the overlay width based on the longest theme name plus swatch space.
// Minimum 40 columns.
func (o *ThemeOverlay) overlayWidth() int {
	minWidth := 40
	for _, th := range o.themes {
		// indicator (1) + space (1) + name + padding (2) + 5 swatches × ("██" + space) (15) + leading gap (2)
		needed := 1 + 1 + len(th.Name()) + 2 + 5*3 + 2
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
	swatch := uikit.GlyphFor(uikit.GlyphBarFull, uikit.ActiveMode())
	var b strings.Builder
	for _, c := range colors {
		b.WriteString(lipgloss.NewStyle().Foreground(c).Render(swatch + swatch))
		b.WriteString(" ")
	}
	return b.String()
}
