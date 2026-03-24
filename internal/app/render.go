// Package app — rendering extracted from app.go.
// This file contains View() and all render* helper methods on *App.
// No state mutation or command dispatch lives here — only pure string rendering.
package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the full terminal UI.
func (a *App) View() string {
	// DESIGN.md: minimum terminal size check.
	if a.width > 0 && a.height > 0 && (a.width < 100 || a.height < 24) {
		return a.renderTooSmall()
	}

	// Splash screen on startup (only when terminal size is known).
	if a.currentView == viewSplash {
		if a.width > 0 && a.height > 0 {
			return a.renderSplash()
		}
		// No size yet — fall through to main view for tests.
	}

	// Auth panel shown when the user needs to authenticate.
	if a.currentView == viewAuth {
		return renderAuthPanel(a.theme, a.width, a.height, a.authURL, a.authStatus)
	}

	// Stats view replaces the three-pane layout when active.
	if a.currentView == viewStats && a.statsPane != nil {
		header := a.renderHeader("[STATS]")
		statsContent := a.statsPane.View()
		statusBar := a.renderStatusBar(a.statsHints())
		return strings.Join([]string{header, statsContent, statusBar}, "\n")
	}

	// Playlist Manager replaces the three-pane layout when active.
	if a.currentView == viewPlaylists && a.playlistPane != nil {
		header := a.renderHeader("[PLAYLISTS]")
		playlistContent := a.playlistPane.View()
		statusBar := a.renderStatusBar(a.playlistsHints())
		return strings.Join([]string{header, playlistContent, statusBar}, "\n")
	}

	header := a.renderHeader("")
	statusBar := a.renderStatusBar(a.mainHints())

	libraryView := a.renderPaneWithBorder(a.libraryPane.View(), a.focus == focusLibrary)
	playerView := a.renderPaneWithBorder(a.playerPane.View(), a.focus == focusPlayer)
	queueView := a.renderPaneWithBorder(a.queuePane.View(), a.focus == focusQueue)

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, libraryView, playerView, queueView)
	body := strings.Join([]string{header, mainContent, statusBar}, "\n")

	if a.deviceOverlayOpen {
		return a.renderWithDeviceOverlay(body)
	}

	if a.searchOpen {
		return a.renderWithSearchOverlay(body)
	}

	return body
}

// renderWithDeviceOverlay renders the three-pane view dimmed and places the
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
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#000000")),
	)
	return centered
}

// renderWithSearchOverlay renders the three-pane view dimmed and places the
// search overlay centered on top using lipgloss.Place() per the DESIGN.md spec.
func (a *App) renderWithSearchOverlay(background string) string {
	overlay := a.searchPane.View()
	dimmed := lipgloss.NewStyle().Faint(true).Render(background)
	if a.width <= 0 || a.height <= 0 {
		return dimmed + "\n" + overlay
	}

	// Center the overlay on a consistent black background so the dimmed
	// three-pane view is replaced with a uniform dark surface behind the modal.
	centered := lipgloss.Place(
		a.width, a.height,
		lipgloss.Center, lipgloss.Center,
		overlay,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#000000")),
	)
	return centered
}

// renderSplash renders the startup splash screen with go-figure ASCII art.
func (a *App) renderSplash() string {
	return renderSplashView(a.theme, a.width, a.height)
}

// renderTooSmall renders the "terminal too small" message.
func (a *App) renderTooSmall() string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(a.theme.ActiveBorder()).
		Padding(1, 2)

	msg := fmt.Sprintf(
		"Spotnik needs more space\n\nCurrent:  %d × %d\nRequired: 100 × 24\n\nPlease resize your terminal and retry.",
		a.width, a.height,
	)
	return style.Render(msg)
}

// renderPaneWithBorder wraps a pane's view with a rounded border per DESIGN.md.
func (a *App) renderPaneWithBorder(content string, focused bool) string {
	borderColor := a.theme.InactiveBorder()
	if focused {
		borderColor = a.theme.ActiveBorder()
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Render(content)
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

// renderStatusBar renders the bottom status bar. If a.statusMsg is set it takes
// priority over hints. hints is a pre-built slice of rendered key-hint strings;
// use mainHints(), statsHints(), or playlistsHints() to obtain the right set.
func (a *App) renderStatusBar(hints []string) string {
	if a.statusMsg != "" {
		return lipgloss.NewStyle().
			Background(a.theme.StatusBarBg()).
			Foreground(a.theme.Error()).
			Render("  " + a.statusMsg)
	}

	style := lipgloss.NewStyle().
		Background(a.theme.StatusBarBg()).
		Foreground(a.theme.StatusBarFg())

	return style.Render("  " + strings.Join(hints, "  "))
}

// mainHints returns the context-sensitive key hints for the three-pane main view.
func (a *App) mainHints() []string {
	keyStyle := lipgloss.NewStyle().
		Foreground(a.theme.KeyHint()).
		Bold(true)

	switch a.focus {
	case focusLibrary:
		return []string{
			keyStyle.Render("/") + " search",
			keyStyle.Render("Enter") + " play",
			keyStyle.Render("a") + " queue",
			keyStyle.Render("l") + " like",
			keyStyle.Render("d") + " devices",
			keyStyle.Render("2") + " stats",
			keyStyle.Render("3") + " lists",
			keyStyle.Render("Tab") + " pane",
			keyStyle.Render("q") + " quit",
		}
	case focusQueue:
		return []string{
			keyStyle.Render("/") + " search",
			keyStyle.Render("j/k") + " navigate",
			keyStyle.Render("Enter") + " play",
			keyStyle.Render("d") + " devices",
			keyStyle.Render("2") + " stats",
			keyStyle.Render("3") + " lists",
			keyStyle.Render("Tab") + " pane",
			keyStyle.Render("q") + " quit",
		}
	default:
		return []string{
			keyStyle.Render("/") + " search",
			keyStyle.Render("Space") + " play",
			keyStyle.Render("n/p") + " skip",
			keyStyle.Render("+/-") + " vol",
			keyStyle.Render("s") + " shuffle",
			keyStyle.Render("r") + " repeat",
			keyStyle.Render("d") + " devices",
			keyStyle.Render("2") + " stats",
			keyStyle.Render("3") + " lists",
			keyStyle.Render("Tab") + " pane",
			keyStyle.Render("q") + " quit",
		}
	}
}

// statsHints returns the key hints for the stats view status bar.
func (a *App) statsHints() []string {
	keyStyle := lipgloss.NewStyle().
		Foreground(a.theme.KeyHint()).
		Bold(true)

	return []string{
		keyStyle.Render("Tab") + " next section",
		keyStyle.Render("j/k") + " move",
		keyStyle.Render("Enter") + " play",
		keyStyle.Render("f") + " cycle range",
		keyStyle.Render("1") + " library",
		keyStyle.Render("q") + " quit",
	}
}

// playlistsHints returns the key hints for the playlist manager view status bar.
func (a *App) playlistsHints() []string {
	keyStyle := lipgloss.NewStyle().
		Foreground(a.theme.KeyHint()).
		Bold(true)

	return []string{
		keyStyle.Render("Enter") + " play",
		keyStyle.Render("r") + " rename",
		keyStyle.Render("n") + " new playlist",
		keyStyle.Render("x") + " remove track",
		keyStyle.Render("Shift+↑↓") + " reorder",
		keyStyle.Render("Tab") + " switch pane",
		keyStyle.Render("1") + " library",
		keyStyle.Render("q") + " quit",
	}
}
