// Package app — rendering extracted from app.go.
// This file contains View() and all render* helper methods on *App.
// No state mutation or command dispatch lives here — only pure string rendering.
package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	btoverlay "github.com/rmhubbert/bubbletea-overlay"

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
	header := a.renderHeader()
	gridContent := a.renderGrid()
	statusBar := a.renderStatusBar()
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

// renderHeader renders the btop-style header bar containing:
// Left: spotnik ─ Page A ─ ᐅp preset 0 ─ ᐅ/ search ─ ᐅd devices
// Right: ◉ DeviceName  (or  ○ No device)
//
// All separator dashes use "─" (U+2500). Key labels are rendered in KeyHint() color,
// descriptions in TextMuted() color, and the app name in TextPrimary()+Bold.
// The background is StatusBarBg(). The line is padded/trimmed to match a.width exactly.
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

	// Page indicator: "Page A"
	page := mutedStyle.Render("Page ") + keyStyle.Render(pageLabel(a.layout.ActivePage()))

	// Preset indicator: "ᐅp preset 0"
	presetIdx := a.layout.ActivePresetIndex()
	preset := mutedStyle.Render("ᐅ") + keyStyle.Render("p") + mutedStyle.Render(fmt.Sprintf(" preset %d", presetIdx))

	// Action shortcuts: "ᐅ/ search"  "ᐅd devices"
	search := mutedStyle.Render("ᐅ") + keyStyle.Render("/") + mutedStyle.Render(" search")
	devices := mutedStyle.Render("ᐅ") + keyStyle.Render("d") + mutedStyle.Render(" devices")

	left := appName + sep + page + sep + preset + sep + search + sep + devices

	// Right side: device indicator.
	device := a.store.ActiveDevice()
	var right string
	if device != nil {
		name := truncateDeviceName(device.Name)
		right = bgStyle.Render("◉ " + name + " ")
	} else {
		right = bgStyle.Render("○ No device ")
	}

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

// renderStatusBar renders the global-only bottom status bar with fixed keybinding hints.
// Pane-specific hints (filter, add, etc.) now live in pane borders — never here.
// Toast notifications are shown as overlays via alerts.Render() — not in the status bar.
func (a *App) renderStatusBar() string {
	bgStyle := lipgloss.NewStyle().
		Background(a.theme.StatusBarBg()).
		Foreground(a.theme.StatusBarFg())

	keyStyle := lipgloss.NewStyle().
		Background(a.theme.StatusBarBg()).
		Foreground(a.theme.KeyHint()).
		Bold(true)

	// Fixed global hints per DESIGN.md §15 — these never change per pane focus.
	hints := []struct{ Key, Label string }{
		{"/", "search"},
		{"0", "page"},
		{"p", "preset"},
		{"1-8", "toggle"},
		{"Tab", "pane"},
		{"d", "devices"},
		{"?", "help"},
		{"q", "quit"},
	}

	var parts []string
	for _, h := range hints {
		parts = append(parts, keyStyle.Render(h.Key)+" "+bgStyle.Render(h.Label))
	}

	return bgStyle.Render("  " + strings.Join(parts, "   "))
}
