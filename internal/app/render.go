// Package app — rendering extracted from app.go.
// This file contains View() and all render* helper methods on *App.
// No state mutation or command dispatch lives here — only pure string rendering.
package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	btoverlay "github.com/rmhubbert/bubbletea-overlay"

	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/uikit"
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
// StatusBar.Render() uses bubbles/help in short-help mode (ShowAll stays false)
// so this method drives the single-row output seen at the bottom of the app.
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

// onboardingTitle renders the shared header for all onboarding screens.
// Both lines are centered so they appear naturally in the centered panel.
func (a *App) onboardingTitle() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(a.theme.TextPrimary()).
		Bold(true)
	subtitleStyle := lipgloss.NewStyle().
		Foreground(a.theme.TextMuted())

	return lipgloss.JoinVertical(lipgloss.Center,
		titleStyle.Render("♪  spotnik"),
		subtitleStyle.Render("A terminal Spotify client"),
	)
}

// renderOnboarding dispatches to the active sub-step renderer and centers the
// result when terminal dimensions are known.
func (a *App) renderOnboarding() string {
	var body string
	switch a.onboardingStep {
	case stepOAuth:
		body = a.renderOnboardingOAuth()
	case stepError:
		body = a.renderOnboardingError()
	default:
		body = a.renderOnboardingRegister()
	}
	if a.width > 0 && a.height > 0 {
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, body)
	}
	return body
}

// renderOnboardingRegister renders Step 1: instructions, redirect URI box, and
// the client ID FormField. No network I/O — reads only app.onboardingPort and
// app.onboardingField.
func (a *App) renderOnboardingRegister() string {
	t := a.theme

	textStyle := lipgloss.NewStyle().Foreground(t.TextPrimary())

	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", a.onboardingPort)

	panelW, panelH := uikit.PanelSize(a.width, a.height)

	// URLBox wraps the redirect URI inside a muted rounded box.
	// Minimum box width of 50 ensures the URI is readable even at zero terminal width (tests).
	uriBoxW := panelW - 12
	if uriBoxW < 50 {
		uriBoxW = 50
	}
	uriBox := uikit.URLBox{URL: redirectURI, Width: uriBoxW, Theme: t}.Render()

	instructions := lipgloss.JoinVertical(lipgloss.Left,
		textStyle.Render("1. Go to https://developer.spotify.com/dashboard and create an app."),
		textStyle.Render("2. In the app settings, set the Redirect URI exactly as shown below:"),
		"",
		uriBox,
		"",
		uikit.StatusGlyph{Role: uikit.RoleWarning, Text: "Spotify Premium is required for playback controls", Theme: t, Gap: 1}.Render(),
		uikit.StatusGlyph{Role: uikit.RoleSuccess, Text: "Your Client ID will be saved to ~/.config/spotnik/config.toml", Theme: t, Gap: 1}.Render(),
	)

	// Center the title block within the panel inner area.
	panelInnerWidth := panelW - 8
	if panelInnerWidth < 72 {
		panelInnerWidth = 72
	}
	centeredTitle := lipgloss.NewStyle().
		Width(panelInnerWidth).
		Align(lipgloss.Center).
		Render(a.onboardingTitle())

	// KeyBar hints: copy-URI hint when empty, confirm-only when typing.
	// Copying is confirmed via toast notification — no inline flash needed.
	var hintBar string
	if a.onboardingField.Value() == "" {
		copyBinding := key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy URI"))
		enterBinding := key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "confirm"))
		quitBinding := key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit"))
		hintBar = uikit.KeyBar{Bindings: []key.Binding{copyBinding, enterBinding, quitBinding}, Theme: t}.Render()
	} else {
		enterBinding := key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "confirm"))
		quitBinding := key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit"))
		hintBar = uikit.KeyBar{Bindings: []key.Binding{enterBinding, quitBinding}, Theme: t}.Render()
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		centeredTitle,
		"",
		instructions,
		"",
		a.onboardingField.Render(),
		"",
		hintBar,
	)

	// Panel wraps body with a titled border; padding is added by caller before passing body.
	paddedBody := lipgloss.NewStyle().Padding(1, 2).Render(body)
	return uikit.Panel{
		Width:  panelW,
		Height: panelH,
		Title:  "Step 1 of 2 — Set up your Spotify Developer App",
		Intent: uikit.PanelIntentDefault,
		Theme:  t,
	}.Render(paddedBody)
}

// renderOnboardingOAuth renders Step 2: the full auth URL, spinner, and copy hint.
// The URL is never truncated — URLBox handles wrapping at '&' boundaries.
func (a *App) renderOnboardingOAuth() string {
	t := a.theme

	textStyle := lipgloss.NewStyle().Foreground(t.TextPrimary())

	panelW, panelH := uikit.PanelSize(a.width, a.height)

	// URLBox wraps the auth URL inside a muted rounded box.
	// Minimum box width of 50 ensures the URL is readable even at zero terminal width (tests).
	urlBoxW := panelW - 12
	if urlBoxW < 50 {
		urlBoxW = 50
	}
	urlBox := uikit.URLBox{URL: a.onboardingAuthURL, Width: urlBoxW, Theme: t}.Render()

	// uikit.Spinner.View() already renders "frame  muted(text)" — no manual join needed.
	spinnerText := a.onboardingSpinner.View()

	// Center the title block within the panel inner area.
	panelInnerWidth := panelW - 8
	if panelInnerWidth < 72 {
		panelInnerWidth = 72
	}
	centeredTitle := lipgloss.NewStyle().
		Width(panelInnerWidth).
		Align(lipgloss.Center).
		Render(a.onboardingTitle())

	copyBinding := key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy URL"))
	quitBinding := key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit"))
	hintBar := uikit.KeyBar{Bindings: []key.Binding{copyBinding, quitBinding}, Theme: t}.Render()

	body := lipgloss.JoinVertical(lipgloss.Left,
		centeredTitle,
		"",
		textStyle.Render("A browser window has been opened. Log in and click Agree."),
		"",
		textStyle.Render("On a headless server or browser didn't open? Visit this URL:"),
		urlBox,
		"",
		spinnerText,
		"",
		hintBar,
	)

	paddedBody := lipgloss.NewStyle().Padding(1, 2).Render(body)
	return uikit.Panel{
		Width:  panelW,
		Height: panelH,
		Title:  "Step 2 of 2 — Authorize Spotnik with Spotify",
		Intent: uikit.PanelIntentDefault,
		Theme:  t,
	}.Render(paddedBody)
}

// renderOnboardingError renders the Step 2 error screen with common causes and
// retry/quit options. The Panel uses PanelIntentError for a red border.
func (a *App) renderOnboardingError() string {
	t := a.theme

	textStyle := lipgloss.NewStyle().Foreground(t.TextPrimary())

	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", a.onboardingPort)

	panelW, panelH := uikit.PanelSize(a.width, a.height)

	// Center the title block within the panel inner area.
	panelInnerWidth := panelW - 8
	if panelInnerWidth < 72 {
		panelInnerWidth = 72
	}
	centeredTitle := lipgloss.NewStyle().
		Width(panelInnerWidth).
		Align(lipgloss.Center).
		Render(a.onboardingTitle())

	retryBinding := key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "re-enter Client ID"))
	retryOAuthBinding := key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "try again"))
	quitBinding := key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit"))
	hintBar := uikit.KeyBar{Bindings: []key.Binding{retryBinding, retryOAuthBinding, quitBinding}, Theme: t}.Render()

	body := lipgloss.JoinVertical(lipgloss.Left,
		centeredTitle,
		"",
		uikit.StatusGlyph{Role: uikit.RoleError, Text: "Authorization failed", Theme: t, Gap: 1}.Render(),
		uikit.StatusGlyph{Role: uikit.RoleError, Text: "Error: " + a.onboardingError, Theme: t, Gap: 1}.Render(),
		"",
		textStyle.Render("Common causes:"),
		textStyle.Render("  •  Client ID mistyped or truncated"),
		textStyle.Render("  •  Redirect URI does not match: "+redirectURI),
		textStyle.Render("  •  Spotify app deleted or suspended"),
		"",
		hintBar,
	)

	paddedBody := lipgloss.NewStyle().Padding(1, 2).Render(body)
	return uikit.Panel{
		Width:  panelW,
		Height: panelH,
		Title:  "Step 2 of 2 — Authorization Failed",
		Intent: uikit.PanelIntentError,
		Theme:  t,
	}.Render(paddedBody)
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

	// Onboarding flow shown on first launch when no client_id is configured.
	// Must be checked before viewAuth — onboarding is a superset of the auth step.
	if a.currentView == viewOnboarding {
		return a.renderOnboarding()
	}

	// Auth panel shown when the user needs to re-authenticate (returning user).
	if a.currentView == viewAuth {
		return renderAuthPanel(a.theme, a.width, a.height, a.authURL, a.authStatus)
	}

	// Grid view: header + grid content + status bar.
	header := a.renderHeader()
	gridContent := a.renderGrid()
	statusBar := a.renderStatusBar()
	body := strings.Join([]string{header, gridContent, statusBar}, "\n")

	// Compact corner overlays — positioned at top-right.
	if a.showThemeSwitcher && a.themeOverlay != nil {
		return a.renderWithOverlayChrome(body, a.themeOverlay.View(), btoverlay.Right, btoverlay.Top)
	}

	if a.deviceOverlayOpen {
		return a.renderWithOverlayChrome(body, a.devicePane.View(), btoverlay.Right, btoverlay.Top)
	}

	if a.profileOverlayOpen {
		return a.renderWithOverlayChrome(body, a.profilePane.View(), btoverlay.Right, btoverlay.Top)
	}

	// Full-screen overlays — centered.
	if a.searchOpen {
		return a.renderWithOverlayChrome(body, a.searchPane.View(), btoverlay.Center, btoverlay.Center)
	}

	if a.helpOpen && a.helpOverlay != nil {
		return a.renderWithOverlayChrome(body, a.helpOverlay.View(), btoverlay.Center, btoverlay.Center)
	}

	return body
}

// renderGrid assembles all visible panes into the full grid using LayoutManager.
// Each pane is rendered independently and placed at its absolute Rect position,
// composing the grid line by line. This handles RowSpan geometry correctly.
func (a *App) renderGrid() string {
	visiblePanes := a.layout.VisiblePanes()
	if len(visiblePanes) == 0 {
		return ""
	}

	// Render each pane to a bordered, size-capped string and split into lines.
	type renderedPane struct {
		rect  layout.Rect
		lines []string
	}
	rendered := make([]renderedPane, 0, len(visiblePanes))
	maxBottom := 0
	for _, paneID := range visiblePanes {
		rect := a.layout.PaneRect(paneID)
		pane, ok := a.panes[paneID]
		if !ok || rect.Width == 0 || rect.Height == 0 {
			continue
		}
		chrome := uikit.PaneChrome{
			Width:       rect.Width,
			Height:      rect.Height,
			Title:       pane.Title(),
			ToggleKey:   pane.ToggleKey(),
			Actions:     pane.Actions(),
			AccentColor: layout.PaneBorderColor(paneID, a.theme),
			Focused:     pane.IsFocused(),
			Theme:       a.theme,
		}
		if fqp, ok := pane.(layout.FilterQueryPane); ok {
			chrome.FilterQuery = fqp.ActiveFilterQuery()
		}
		bordered := chrome.Render(pane.View())
		capped := lipgloss.NewStyle().
			Width(rect.Width).MaxWidth(rect.Width).
			Height(rect.Height).MaxHeight(rect.Height).
			Render(bordered)
		lines := strings.Split(capped, "\n")
		rendered = append(rendered, renderedPane{rect: rect, lines: lines})
		if bottom := rect.Y + rect.Height; bottom > maxBottom {
			maxBottom = bottom
		}
	}

	// Compose grid line by line using absolute Rect positions.
	// Non-overlapping panes are concatenated left-to-right within each terminal row.
	outputLines := make([]string, maxBottom)
	for y := 0; y < maxBottom; y++ {
		type segment struct {
			x     int
			width int
			line  string
		}
		var segs []segment
		for _, rp := range rendered {
			lineIdx := y - rp.rect.Y
			if lineIdx < 0 || lineIdx >= len(rp.lines) {
				continue
			}
			segs = append(segs, segment{x: rp.rect.X, width: rp.rect.Width, line: rp.lines[lineIdx]})
		}
		// Sort segments left-to-right by X for deterministic composition.
		for i := 1; i < len(segs); i++ {
			for j := i; j > 0 && segs[j].x < segs[j-1].x; j-- {
				segs[j], segs[j-1] = segs[j-1], segs[j]
			}
		}
		var sb strings.Builder
		curX := 0
		for _, seg := range segs {
			if seg.x > curX {
				sb.WriteString(strings.Repeat(" ", seg.x-curX))
				curX = seg.x
			}
			sb.WriteString(seg.line)
			curX += seg.width
		}
		outputLines[y] = sb.String()
	}

	return strings.Join(outputLines, "\n")
}

// renderWithOverlayChrome renders the grid dimmed and composites an overlay view on top
// using bubbletea-overlay Composite(). hPos and vPos control placement — callers use
// btoverlay.Right/Top for compact corner overlays (theme, profile, device) and
// btoverlay.Center/Center for full-screen overlays (search, help).
//
// This is the single overlay helper — it replaces the five former per-overlay helpers
// (renderWithThemeOverlay, renderWithProfileOverlay, renderWithDeviceOverlay,
// renderWithSearchOverlay, renderWithHelpOverlay). Callers pass a pre-rendered
// overlayView string (e.g. a.searchPane.View()) so that the rendering strategy remains
// uniform across all overlay states.
func (a *App) renderWithOverlayChrome(background, overlayView string, hPos, vPos btoverlay.Position) string {
	dimmed := lipgloss.NewStyle().Faint(true).Render(background)
	if a.width <= 0 || a.height <= 0 {
		return dimmed + "\n" + overlayView
	}
	return btoverlay.Composite(overlayView, dimmed, hPos, vPos, 0, 0)
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

// renderHeader renders the btop-style header bar containing:
// Page A — Left: spotnik ─ Page A ─ preset 0    Right: ◉ DeviceName   ♛ DisplayName
// Page B — Left: spotnik ─ Page B               Right: ◉ DeviceName   ♛ DisplayName
//
// The profile chip (name + tier badge) appears to the right of the device chip.
// The profile chip is absent when the profile has not yet been loaded.
//
// Shortcut hints (search, devices, preset key) are omitted from the header because
// they already appear in the bottom status bar — the header is for contextual info only.
// Rendering is delegated to uikit.HeaderBar + uikit.Chip.
func (a *App) renderHeader() string {
	preset := a.layout.ActivePresetIndex()
	if a.layout.ActivePage() == layout.PageB {
		// Page B has no user-selectable presets — hide the segment.
		preset = -1
	}

	// Build right-side chips: device chip first, then profile chip.
	var chips []string
	if dev := a.store.ActiveDevice(); dev != nil {
		chips = append(chips, uikit.Chip{
			Glyph:  uikit.GlyphActive,
			Label:  truncateDeviceName(dev.Name),
			Intent: uikit.RoleInfo,
			Theme:  a.theme,
		}.Render())
	} else {
		chips = append(chips, uikit.Chip{
			Glyph:  uikit.GlyphAvailable,
			Label:  "No device",
			Intent: uikit.RoleMuted,
			Theme:  a.theme,
		}.Render())
	}
	if profile := a.store.UserProfile(); profile.ID != "" {
		g := uikit.GlyphAvailable
		intent := uikit.RoleMuted
		if a.store.IsPremium() {
			g = uikit.GlyphPremium
			intent = uikit.RoleInfo
		}
		chips = append(chips, uikit.Chip{
			Glyph:  g,
			Label:  truncateProfileName(profile.DisplayName),
			Intent: intent,
			Theme:  a.theme,
		}.Render())
	}

	return uikit.HeaderBar{
		Width:      a.width,
		AppName:    "spotnik",
		Page:       pageLabel(a.layout.ActivePage()),
		Preset:     preset,
		RightChips: chips,
		Theme:      a.theme,
	}.Render()
}

// renderStatusBar renders the global bottom status bar as a bubbles/help panel.
// Delegates to uikit.StatusBar which owns the layout: 3 lines tall (top border +
// 1 content row + bottom border). Page A shows all 10 bindings; Page B omits preset/toggle.
func (a *App) renderStatusBar() string {
	// Copy the keymap so we can set activePage without mutating App state.
	// renderStatusBar() must remain a pure render function.
	km := a.statusKeyMap
	km.activePage = a.layout.ActivePage()
	return uikit.StatusBar{
		Width:    a.width,
		Bindings: km,
		Theme:    a.theme,
	}.Render()
}
