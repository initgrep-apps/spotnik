// Package app — rendering extracted from app.go.
// This file contains View() and all render* helper methods on *App.
// No state mutation or command dispatch lives here — only pure string rendering.
package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
)

// View renders the full terminal UI.
// IMPORTANT: The final step calls a.alerts.Render(view) to overlay any active
// toast notification on top of the complete rendered view. Never call alerts.View()
// — BubbleUp's View() returns empty string by design.
func (a *App) View() string {
	return a.alerts.Render(a.buildView())
}

// buildView renders the full terminal UI content without the alert overlay.
// Called by View() which applies alerts.Render() as the final step.
func (a *App) buildView() string {
	// DESIGN.md: minimum terminal size check (updated to 120×30 per F49).
	if a.width > 0 && a.height > 0 && (a.width < 120 || a.height < 30) {
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

	// Grid view: header + grid content + status bar.
	header := a.renderHeader("")
	gridContent := a.renderGrid()
	statusBar := a.renderStatusBar(a.gridHints())
	body := strings.Join([]string{header, gridContent, statusBar}, "\n")

	if a.deviceOverlayOpen {
		return a.renderWithDeviceOverlay(body)
	}

	if a.searchOpen {
		return a.renderWithSearchOverlay(body)
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

// renderWithDeviceOverlay renders the grid dimmed and places the
// device switcher overlay in the top-right area per the DESIGN.md spec.
func (a *App) renderWithDeviceOverlay(background string) string {
	overlay := a.devicePane.View()
	dimmed := lipgloss.NewStyle().Faint(true).Render(background)
	if a.width <= 0 || a.height <= 0 {
		return dimmed + "\n" + overlay
	}

	// Position overlay in the top-right area (below the header/device indicator).
	centered := lipgloss.Place(
		a.width, a.height,
		lipgloss.Right, lipgloss.Top,
		overlay,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(a.theme.Base()),
	)
	return centered
}

// renderWithSearchOverlay renders the grid dimmed and places the
// search overlay centered on top using lipgloss.Place() per the DESIGN.md spec.
func (a *App) renderWithSearchOverlay(background string) string {
	overlay := a.searchPane.View()
	dimmed := lipgloss.NewStyle().Faint(true).Render(background)
	if a.width <= 0 || a.height <= 0 {
		return dimmed + "\n" + overlay
	}

	// Center the overlay on a consistent black background so the dimmed
	// grid is replaced with a uniform dark surface behind the modal.
	centered := lipgloss.Place(
		a.width, a.height,
		lipgloss.Center, lipgloss.Center,
		overlay,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(a.theme.Base()),
	)
	return centered
}

// renderSplash renders the startup splash screen with go-figure ASCII art.
func (a *App) renderSplash() string {
	return renderSplashView(a.theme, a.width, a.height)
}

// renderTooSmall renders the "terminal too small" message.
// Updated minimum: 120×30 per F49 spec.
func (a *App) renderTooSmall() string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(a.theme.ActiveBorder()).
		Padding(1, 2)

	msg := fmt.Sprintf(
		"Spotnik needs more space\n\nCurrent:  %d × %d\nRequired: 120 × 30\n\nPlease resize your terminal and retry.",
		a.width, a.height,
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

// renderHeader renders the top bar with the app name left-aligned and the device
// indicator right-aligned. label is an optional suffix shown after "spotnik"
// (e.g. "[STATS]", "[PLAYLISTS]"); pass "" for the default main-view header.
func (a *App) renderHeader(label string) string {
	appNameStyle := lipgloss.NewStyle().
		Background(a.theme.SurfaceAlt()).
		Foreground(a.theme.TextPrimary()).
		Bold(true)

	device := a.store.ActiveDevice()
	var deviceStr string
	if device != nil {
		name := truncateDeviceName(device.Name)
		deviceStr = lipgloss.NewStyle().
			Foreground(a.theme.DeviceActive()).
			Render(fmt.Sprintf("◉ %s", name))
	} else {
		deviceStr = lipgloss.NewStyle().
			Foreground(a.theme.TextMuted()).
			Render("○ No device")
	}

	appName := appNameStyle.Render(" spotnik ")
	if label != "" {
		labelStyle := lipgloss.NewStyle().
			Foreground(a.theme.SectionHeader()).
			Bold(true)
		appName = appName + " " + labelStyle.Render(label)
	}

	if a.width > 0 {
		gap := a.width - lipgloss.Width(appName) - lipgloss.Width(deviceStr)
		if gap < 1 {
			gap = 1
		}
		return appName + strings.Repeat(" ", gap) + deviceStr
	}
	return appName + "  " + deviceStr
}

// renderStatusBar renders the bottom status bar with keybinding hints.
// Toast notifications are shown as overlays via alerts.Render() — they no longer
// appear in the status bar. The status bar is now always hints-only.
// hints is a pre-built slice of rendered key-hint strings;
// use gridHints() to obtain the right set.
func (a *App) renderStatusBar(hints []string) string {
	style := lipgloss.NewStyle().
		Background(a.theme.StatusBarBg()).
		Foreground(a.theme.StatusBarFg())

	return style.Render("  " + strings.Join(hints, "  "))
}

// gridHints returns the context-sensitive key hints for the grid view status bar.
func (a *App) gridHints() []string {
	keyStyle := lipgloss.NewStyle().
		Foreground(a.theme.KeyHint()).
		Bold(true)

	return []string{
		keyStyle.Render("/") + " search",
		keyStyle.Render("Space") + " play",
		keyStyle.Render("n") + " next",
		keyStyle.Render("+/-") + " vol",
		keyStyle.Render("0") + " page",
		keyStyle.Render("p") + " preset",
		keyStyle.Render("1-8") + " toggle",
		keyStyle.Render("Tab") + " focus",
		keyStyle.Render("d") + " devices",
		keyStyle.Render("q") + " quit",
	}
}
