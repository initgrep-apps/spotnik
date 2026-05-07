// Package panes — OnboardingPermissionsOverlay is the modal that appears when
// the user presses 'v' on Step 2 of the onboarding flow. It explains, in
// generic terms, what spotnik reads and what it can write, and points users to
// Spotify for full scope details and revoke. Modal: Esc emits the close
// message; all other keys are consumed.
package panes

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// OnboardingPermissionsOverlayClosedMsg is emitted when the user presses Esc
// in the OnboardingPermissionsOverlay. The root app handles this by clearing
// the overlay state.
type OnboardingPermissionsOverlayClosedMsg struct{}

// permissionsContent returns the body rendered inside the overlay border.
// Generic by design — no per-scope semantics, no internals — so it remains
// accurate as the underlying scope set evolves. Users who need scope details
// are pointed to spotify.com/account/apps in the tail line.
//
// Bullets are sourced from the uikit glyph catalogue so ASCII mode renders
// them as `*` and unicode mode renders them as `•`.
func permissionsContent() string {
	b := uikit.GlyphFor(uikit.GlyphBullet, uikit.ActiveMode())
	return "Read access\n" +
		" " + b + " Playback state, queue, and devices.\n" +
		" " + b + " Saved tracks, albums, and playlists.\n" +
		" " + b + " Profile, listening history, and followed artists.\n" +
		"\n" +
		"Write access\n" +
		" " + b + " Playback control — play, pause, skip, seek, volume, transfer.\n" +
		" " + b + " Library and playlist edits — reserved for upcoming features.\n" +
		"\n" +
		"For full scope details and revoke: spotify.com/account/apps"
}

// OnboardingPermissionsOverlay is the floating permissions reference shown on
// Step 2 of onboarding. Pressing Esc emits OnboardingPermissionsOverlayClosedMsg;
// all other keys are consumed (modal).
type OnboardingPermissionsOverlay struct {
	theme  theme.Theme
	width  int
	height int
}

// NewOnboardingPermissionsOverlay creates an overlay using the given theme.
func NewOnboardingPermissionsOverlay(th theme.Theme) *OnboardingPermissionsOverlay {
	return &OnboardingPermissionsOverlay{theme: th}
}

// SetSize updates the render dimensions used for the overlay.
func (o *OnboardingPermissionsOverlay) SetSize(width, height int) {
	o.width = width
	o.height = height
}

// SetTheme updates the overlay's theme reference for runtime theme switching.
func (o *OnboardingPermissionsOverlay) SetTheme(th theme.Theme) {
	o.theme = th
}

// Init satisfies tea.Model; no startup command needed.
func (o *OnboardingPermissionsOverlay) Init() tea.Cmd { return nil }

// Update routes keyboard input. Esc closes the overlay; all other keys are
// consumed with a nil cmd.
func (o *OnboardingPermissionsOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return o, nil
	}
	if keyMsg.Type == tea.KeyEsc {
		return o, func() tea.Msg { return OnboardingPermissionsOverlayClosedMsg{} }
	}
	return o, nil
}

// overlayWidth returns the fixed overlay width (76), capped to the terminal
// width when the terminal is narrower.
func (o *OnboardingPermissionsOverlay) overlayWidth() int {
	const fixedWidth = 76
	if o.width > 0 && fixedWidth > o.width {
		return o.width
	}
	return fixedWidth
}

// View renders the overlay inside an OverlayChrome border.
func (o *OnboardingPermissionsOverlay) View() string {
	totalW := o.overlayWidth()
	innerW := totalW - 2
	if innerW < 2 {
		innerW = 2
	}

	bodyStyle := lipgloss.NewStyle().
		Foreground(o.theme.TextPrimary()).
		Width(innerW).
		MaxWidth(innerW).
		Padding(1, 2)
	body := bodyStyle.Render(permissionsContent())

	escBinding := key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "close"))
	keyBar := uikit.KeyBar{Bindings: []key.Binding{escBinding}, Theme: o.theme}.Render()
	keyBarStyled := lipgloss.NewStyle().
		Width(innerW).
		MaxWidth(innerW).
		Padding(0, 2, 1, 2).
		Render(keyBar)

	inner := lipgloss.JoinVertical(lipgloss.Left, body, keyBarStyled)
	height := strings.Count(inner, "\n") + 3 // +1 for last line, +2 for borders

	chrome := uikit.OverlayChrome{
		Width:  totalW,
		Height: height,
		Title:  "Permissions Spotnik requests",
		Theme:  o.theme,
	}
	return chrome.Render(inner)
}
