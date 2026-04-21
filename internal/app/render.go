// Package app — rendering extracted from app.go.
// This file contains View() and all render* helper methods on *App.
// No state mutation or command dispatch lives here — only pure string rendering.
package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	btoverlay "github.com/rmhubbert/bubbletea-overlay"

	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// appKeyMap implements help.KeyMap for the app-level status bar.
// activePage is set via a copy per render call so renderStatusBar() stays pure.
//
// Page A (10 bindings, 5 columns × 2 rows):
//
//	/ search   p preset   Tab pane    t theme    q quit
//	0 page     1-8 toggle  d devices   u profile  ? help
//
// Page B (8 bindings, 4 columns × 2 rows):
//
//	/ search   Tab pane    t theme    q quit
//	0 page     d devices   u profile  ? help
type appKeyMap struct {
	activePage                                                              layout.PageID
	Search, Page, Preset, Toggle, Pane, Devices, Profile, Theme, Help, Quit key.Binding
}

// ShortHelp returns all applicable bindings for the active page.
// Bypassed at runtime because the help model uses ShowAll=true, but kept
// complete so callers inspecting the KeyMap programmatically see the full set.
func (k appKeyMap) ShortHelp() []key.Binding {
	if k.activePage == layout.PageA {
		return []key.Binding{k.Search, k.Page, k.Preset, k.Toggle, k.Pane, k.Devices, k.Profile, k.Theme, k.Help, k.Quit}
	}
	return []key.Binding{k.Search, k.Page, k.Pane, k.Devices, k.Profile, k.Theme, k.Help, k.Quit}
}

// FullHelp returns groups of bindings for the column layout.
// Each inner slice is one column rendered vertically; columns are joined horizontally.
// Column 3 (Pane/Devices/Profile) has 3 entries — Devices and Profile are overlay shortcuts.
func (k appKeyMap) FullHelp() [][]key.Binding {
	if k.activePage == layout.PageA {
		return [][]key.Binding{
			{k.Search, k.Page},
			{k.Preset, k.Toggle},
			{k.Pane, k.Devices, k.Profile},
			{k.Theme, k.Help},
			{k.Quit},
		}
	}
	return [][]key.Binding{
		{k.Search, k.Page},
		{k.Pane, k.Devices, k.Profile},
		{k.Theme, k.Help},
		{k.Quit},
	}
}

// newAppKeyMap creates the default keybindings for the app-level status bar.
func newAppKeyMap() appKeyMap {
	return appKeyMap{
		Search:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		Page:    key.NewBinding(key.WithKeys("0"), key.WithHelp("0", "page")),
		Preset:  key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "preset")),
		Toggle:  key.NewBinding(key.WithKeys("1", "2", "3", "4", "5", "6", "7", "8"), key.WithHelp("1-8", "toggle")),
		Pane:    key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "pane")),
		Devices: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "devices")),
		Profile: key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "profile")),
		Theme:   key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "theme")),
		Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:    key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	}
}

// newStatusHelp creates and styles a help.Model for the app status bar.
// Uses ShortHelp (single-row) mode — ShowAll stays false so all bindings render
// on one line, matching the flat single-row status bar aesthetic.
// Keys use Info() color; descriptions and separators use TextMuted().
func newStatusHelp(t theme.Theme) help.Model {
	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(t.Info())
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(t.TextMuted())
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(t.TextMuted())
	h.Styles.FullKey = lipgloss.NewStyle().Foreground(t.Info())
	h.Styles.FullDesc = lipgloss.NewStyle().Foreground(t.TextMuted())
	h.Styles.FullSeparator = lipgloss.NewStyle().Foreground(t.TextMuted())
	return h
}

// View renders the full terminal UI.
// IMPORTANT: The final step calls a.alerts.Render(view) to overlay any active
// toast notification on top of the complete rendered view. Never call alerts.View()
// — BubbleUp's View() returns empty string by design.
func (a *App) View() string {
	return a.alerts.Render(a.buildView())
}

// minTermWidth and minTermHeight define the minimum terminal dimensions required
// for Spotnik to render the full grid layout correctly.
// Below these dimensions, renderTooSmall() is shown instead.
const (
	minTermWidth  = 120
	minTermHeight = 30
)

// wrapURL breaks a long URL into multiple lines each at most width characters wide.
// It prefers to break at '&' boundaries found in the second half of each width window
// so that query parameters land at the start of a new line, which is easier to read.
// When no '&' is present in the target window it falls back to a hard break at width.
// A URL that already fits within width is returned unchanged (no newlines added).
func wrapURL(rawURL string, width int) string {
	if len(rawURL) <= width {
		return rawURL
	}
	var lines []string
	for len(rawURL) > width {
		breakAt := width
		if idx := strings.LastIndex(rawURL[:width], "&"); idx > width/2 {
			breakAt = idx
		}
		lines = append(lines, rawURL[:breakAt])
		rawURL = rawURL[breakAt:]
	}
	if rawURL != "" {
		lines = append(lines, rawURL)
	}
	return strings.Join(lines, "\n")
}

// buildView renders the full terminal UI content without the alert overlay.
// Called by View() which applies alerts.Render() as the final step.
func (a *App) buildView() string {
	// DESIGN.md §21: minimum terminal size check.
	if a.width > 0 && a.height > 0 && (a.width < minTermWidth || a.height < minTermHeight) {
		return a.renderTooSmall()
	}

	// Splash screen on startup (only when terminal size is known).
	if a.currentView == viewSplash {
		if a.width > 0 && a.height > 0 {
			return a.renderSplash()
		}
		// No size yet — fall through to grid view for tests.
	}

	// Auth panel shown when the user needs to authenticate.
	if a.currentView == viewAuth {
		return renderAuthPanel(a.theme, a.width, a.height, a.authURL, a.authStatus)
	}

	// Onboarding flow shown on first launch when no client_id is configured.
	// Story 139 will deliver the real onboarding renderer; for now return a
	// placeholder so the app does not fall through to grid rendering (which
	// would crash because API clients are nil at this point).
	if a.currentView == viewOnboarding {
		return "Onboarding — setting up your Spotify connection…"
	}

	// Grid view: header + grid content + status bar.
	header := a.renderHeader()
	gridContent := a.renderGrid()
	statusBar := a.renderStatusBar()
	body := strings.Join([]string{header, gridContent, statusBar}, "\n")

	if a.showThemeSwitcher && a.themeOverlay != nil {
		return a.renderWithThemeOverlay(body)
	}

	if a.deviceOverlayOpen {
		return a.renderWithDeviceOverlay(body)
	}

	if a.profileOverlayOpen {
		return a.renderWithProfileOverlay(body)
	}

	if a.searchOpen {
		return a.renderWithSearchOverlay(body)
	}

	if a.helpOpen && a.helpOverlay != nil {
		return a.renderWithHelpOverlay(body)
	}

	return body
}

// renderGrid assembles all visible panes into the full grid using LayoutManager.
// Panes are grouped by row (using Rect.Y), rendered with btop-style borders,
// and joined horizontally per row, then vertically across rows.
func (a *App) renderGrid() string {
	visiblePanes := a.layout.VisiblePanes()
	if len(visiblePanes) == 0 {
		return ""
	}

	// Group panes by row (panes with the same Rect.Y belong to the same row).
	rows := groupPanesByRow(visiblePanes, a.layout)

	var rowStrings []string
	for _, row := range rows {
		var cellStrings []string
		for _, paneID := range row {
			rect := a.layout.PaneRect(paneID)
			pane, ok := a.panes[paneID]
			if !ok {
				continue
			}

			// Get pane content (sized to content area).
			content := pane.View()

			// Wrap in btop-style border.
			cfg := layout.BorderConfig{
				Width:       rect.Width,
				Height:      rect.Height,
				Title:       pane.Title(),
				ToggleKey:   pane.ToggleKey(),
				Actions:     pane.Actions(),
				AccentColor: layout.PaneBorderColor(paneID, a.theme),
				Focused:     pane.IsFocused(),
				Theme:       a.theme,
			}
			bordered := layout.RenderPaneBorder(content, cfg)

			// Ensure exact width/height via lipgloss (safety cap against oversized pane output).
			capped := lipgloss.NewStyle().
				Width(rect.Width).MaxWidth(rect.Width).
				Height(rect.Height).MaxHeight(rect.Height).
				Render(bordered)
			cellStrings = append(cellStrings, capped)
		}
		if len(cellStrings) > 0 {
			rowStr := lipgloss.JoinHorizontal(lipgloss.Top, cellStrings...)
			rowStrings = append(rowStrings, rowStr)
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, rowStrings...)
}

// groupPanesByRow groups visible PaneIDs into rows based on their Y coordinate.
// Returns a slice of rows, each row being a slice of PaneIDs in left-to-right order.
func groupPanesByRow(paneIDs []layout.PaneID, mgr *layout.Manager) [][]layout.PaneID {
	if len(paneIDs) == 0 {
		return nil
	}

	// Track which Y values we've seen, in order.
	seen := make(map[int]bool)
	var yOrder []int
	rowMap := make(map[int][]layout.PaneID)

	for _, id := range paneIDs {
		rect := mgr.PaneRect(id)
		if !seen[rect.Y] {
			seen[rect.Y] = true
			yOrder = append(yOrder, rect.Y)
		}
		rowMap[rect.Y] = append(rowMap[rect.Y], id)
	}

	rows := make([][]layout.PaneID, len(yOrder))
	for i, y := range yOrder {
		rows[i] = rowMap[y]
	}
	return rows
}

// renderWithThemeOverlay renders the grid dimmed and places the theme switcher
// overlay in the top-right area using bubbletea-overlay Composite().
func (a *App) renderWithThemeOverlay(background string) string {
	fg := a.themeOverlay.View()
	dimmed := lipgloss.NewStyle().Faint(true).Render(background)
	if a.width <= 0 || a.height <= 0 {
		return dimmed + "\n" + fg
	}
	return btoverlay.Composite(fg, dimmed, btoverlay.Right, btoverlay.Top, 0, 0)
}

// renderWithProfileOverlay renders the grid dimmed and places the profile overlay
// in the top-right area using bubbletea-overlay Composite().
func (a *App) renderWithProfileOverlay(background string) string {
	fg := a.profilePane.View()
	dimmed := lipgloss.NewStyle().Faint(true).Render(background)
	if a.width <= 0 || a.height <= 0 {
		return dimmed + "\n" + fg
	}
	return btoverlay.Composite(fg, dimmed, btoverlay.Right, btoverlay.Top, 0, 0)
}

// renderWithDeviceOverlay renders the grid dimmed and places the
// device switcher overlay in the top-right area using bubbletea-overlay Composite().
func (a *App) renderWithDeviceOverlay(background string) string {
	fg := a.devicePane.View()
	dimmed := lipgloss.NewStyle().Faint(true).Render(background)
	if a.width <= 0 || a.height <= 0 {
		return dimmed + "\n" + fg
	}

	// Position overlay in the top-right using bubbletea-overlay string-level compositing.
	return btoverlay.Composite(fg, dimmed, btoverlay.Right, btoverlay.Top, 0, 0)
}

// renderWithSearchOverlay renders the grid dimmed and places the
// search overlay centered on top using bubbletea-overlay Composite().
func (a *App) renderWithSearchOverlay(background string) string {
	fg := a.searchPane.View()
	dimmed := lipgloss.NewStyle().Faint(true).Render(background)
	if a.width <= 0 || a.height <= 0 {
		return dimmed + "\n" + fg
	}

	// Center the search overlay using bubbletea-overlay string-level compositing.
	return btoverlay.Composite(fg, dimmed, btoverlay.Center, btoverlay.Center, 0, 0)
}

// renderWithHelpOverlay renders the grid dimmed and places the help overlay
// centered on screen using bubbletea-overlay Composite().
func (a *App) renderWithHelpOverlay(background string) string {
	fg := a.helpOverlay.View()
	dimmed := lipgloss.NewStyle().Faint(true).Render(background)
	if a.width <= 0 || a.height <= 0 {
		return dimmed + "\n" + fg
	}
	return btoverlay.Composite(fg, dimmed, btoverlay.Center, btoverlay.Center, 0, 0)
}

// renderSplash renders the startup splash screen with go-figure ASCII art.
func (a *App) renderSplash() string {
	return renderSplashView(a.theme, a.version, a.width, a.height)
}

// renderTooSmall renders the "terminal too small" message.
// Shows the current and required dimensions so the user knows how much to resize.
func (a *App) renderTooSmall() string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(a.theme.ActiveBorder()).
		Padding(1, 2)

	msg := fmt.Sprintf(
		"Spotnik needs more space\n\nCurrent:  %d × %d\nRequired: %d × %d\n\nPlease resize your terminal and retry.",
		a.width, a.height, minTermWidth, minTermHeight,
	)
	return style.Render(msg)
}

// maxDeviceNameLen is the maximum number of characters for the device name in the header.
const maxDeviceNameLen = 25

// truncateDeviceName truncates a device name to maxDeviceNameLen chars, appending … if needed.
func truncateDeviceName(name string) string {
	runes := []rune(name)
	if len(runes) > maxDeviceNameLen {
		return string(runes[:maxDeviceNameLen-1]) + "…"
	}
	return name
}

// pageLabel converts a layout.PageID to its display label ("A" or "B").
func pageLabel(page layout.PageID) string {
	switch page {
	case layout.PageA:
		return "A"
	case layout.PageB:
		return "B"
	default:
		return "?"
	}
}

// maxProfileDisplayNameLen is the maximum runes shown for the display name in the header chip.
const maxProfileDisplayNameLen = 20

// truncateProfileName truncates a display name to maxProfileDisplayNameLen runes,
// appending … if truncated. Mirrors truncateDeviceName for display name capping.
func truncateProfileName(name string) string {
	runes := []rune(name)
	if len(runes) > maxProfileDisplayNameLen {
		return string(runes[:maxProfileDisplayNameLen-1]) + "…"
	}
	return name
}

// renderProfileChip renders the profile chip shown in the header right side.
// Returns "" if the profile has not yet been loaded (graceful startup).
// Format: "♛ DisplayName " (Premium) or "○ DisplayName " (Free) — badge precedes name.
func (a *App) renderProfileChip() string {
	profile := a.store.UserProfile()
	if profile.ID == "" {
		// Profile not yet loaded — render nothing so header is clean on startup.
		return ""
	}

	bgStyle := lipgloss.NewStyle().Background(a.theme.StatusBarBg())

	name := truncateProfileName(profile.DisplayName)

	nameStyle := lipgloss.NewStyle().
		Background(a.theme.StatusBarBg()).
		Foreground(a.theme.HeaderChipFg())

	var badge string
	if a.store.IsPremium() {
		badgeStyle := lipgloss.NewStyle().
			Background(a.theme.StatusBarBg()).
			Foreground(a.theme.Info())
		badge = badgeStyle.Render("♛")
	} else {
		badgeStyle := lipgloss.NewStyle().
			Background(a.theme.StatusBarBg()).
			Foreground(a.theme.TextMuted())
		badge = badgeStyle.Render("○")
	}

	return bgStyle.Render(" ") + badge + bgStyle.Render(" ") + nameStyle.Render(name) + bgStyle.Render(" ")
}

// renderHeader renders the btop-style header bar containing:
// Page A — Left: spotnik ─ Page A ─ preset 0    Right: ◉ DeviceName   ♛ DisplayName
// Page B — Left: spotnik ─ Page B               Right: ◉ DeviceName   ♛ DisplayName
//
// The profile chip (name + tier badge) appears to the right of the device chip.
// The profile chip is absent when the profile has not yet been loaded.
//
// Shortcut hints (search, devices, preset key) are omitted from the header because
// they already appear in the bottom status bar — the header is for contextual info only.
// All separator dashes use "─" (U+2500). The app name uses TextPrimary()+Bold.
// The background is StatusBarBg(). The line is padded to match a.width exactly.
func (a *App) renderHeader() string {
	bgStyle := lipgloss.NewStyle().
		Background(a.theme.StatusBarBg()).
		Foreground(a.theme.StatusBarFg())

	appNameStyle := lipgloss.NewStyle().
		Background(a.theme.StatusBarBg()).
		Foreground(a.theme.TextPrimary()).
		Bold(true)

	keyStyle := lipgloss.NewStyle().
		Background(a.theme.StatusBarBg()).
		Foreground(a.theme.KeyHint()).
		Bold(true)

	mutedStyle := lipgloss.NewStyle().
		Background(a.theme.StatusBarBg()).
		Foreground(a.theme.TextMuted())

	sepStyle := lipgloss.NewStyle().
		Background(a.theme.StatusBarBg()).
		Foreground(a.theme.TextMuted())

	sep := sepStyle.Render(" ─ ")

	// App name segment.
	appName := appNameStyle.Render(" spotnik ")

	// Page indicator: "Page A" or "Page B"
	page := mutedStyle.Render("Page ") + keyStyle.Render(pageLabel(a.layout.ActivePage()))

	// Build left segment — Page A shows the active preset index as contextual info;
	// Page B has a single fixed layout with no user-selectable presets, so it is omitted.
	var left string
	if a.layout.ActivePage() == layout.PageB {
		left = appName + sep + page
	} else {
		presetIdx := a.layout.ActivePresetIndex()
		preset := mutedStyle.Render(fmt.Sprintf("preset %d", presetIdx))
		left = appName + sep + page + sep + preset
	}

	// Right side: device chip then profile chip (profile is rightmost).
	device := a.store.ActiveDevice()
	var deviceChip string
	if device != nil {
		name := truncateDeviceName(device.Name)
		activeStyle := lipgloss.NewStyle().
			Background(a.theme.StatusBarBg()).
			Foreground(a.theme.HeaderChipFg())
		deviceChip = activeStyle.Render("◉ " + name + " ")
	} else {
		deviceChip = bgStyle.Render("○ No device ")
	}
	right := deviceChip + a.renderProfileChip()

	if a.width > 0 {
		leftW := lipgloss.Width(left)
		rightW := lipgloss.Width(right)
		gap := a.width - leftW - rightW
		if gap < 1 {
			gap = 1
		}
		fill := bgStyle.Render(strings.Repeat(" ", gap))
		return left + fill + right
	}
	return left + "  " + right
}

// renderStatusBar renders the global bottom status bar as a bubbles/help panel.
// Uses the same component and visual style as the search overlay's keybinding bar:
// keys in Info() color, descriptions in TextMuted(), wrapped in RenderPaneBorder.
//
// The panel is always 3 lines tall (border + 1 content row + border).
// Page A shows all 10 bindings in a single row; Page B omits preset/toggle.
func (a *App) renderStatusBar() string {
	const statusH = 3 // 1 content row + top/bottom border
	// Use a minimum rendering width of 160 so all bindings are visible on one row
	// even when no terminal size has been set (e.g. in unit tests that call
	// renderStatusBar directly without first sending a tea.WindowSizeMsg).
	w := a.width
	if w < 160 {
		w = 160
	}
	innerW := w - 2

	// Copy the keymap so we can set activePage without mutating App state.
	// renderStatusBar() must remain a pure render function.
	km := a.statusKeyMap
	km.activePage = a.layout.ActivePage()

	helpContent := a.statusHelp.View(km)
	inner := lipgloss.NewStyle().
		Width(innerW).MaxWidth(innerW).
		Height(statusH - 2).MaxHeight(statusH - 2).
		Render(helpContent)

	cfg := layout.BorderConfig{
		Width:       w,
		Height:      statusH,
		Title:       "",
		Actions:     []layout.Action{},
		AccentColor: a.theme.TextMuted(),
		Focused:     false,
		Theme:       a.theme,
	}
	return layout.RenderPaneBorder(inner, cfg)
}
