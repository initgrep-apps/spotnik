// Package panes — ProfileOverlay is the floating user profile overlay.
// It displays the authenticated user's display name, subscription tier,
// and country. It reads directly from the Store and emits ProfileOverlayClosedMsg
// on Esc. Triggered by the 'u' key; never imports api/ directly.
package panes

import (
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// maxProfileNameLen is the maximum number of runes shown for the display name.
const maxProfileNameLen = 20

// profileAction represents the pending confirmation action in the profile overlay.
type profileAction int

const (
	// profileActionNone means no action is pending — idle state.
	profileActionNone profileAction = iota
	// profileActionLogout means the user pressed 'l' once and confirmation is pending.
	profileActionLogout
	// profileActionForget means the user pressed 'f' once and confirmation is pending.
	profileActionForget
)

// ProfileOverlay renders the authenticated user's profile as a floating overlay.
// Reads directly from the Store. Tracks pending confirmation state for logout/forget.
// Triggered by the 'u' key; closed by Esc.
type ProfileOverlay struct {
	store         state.StateReader
	theme         theme.Theme
	width         int
	height        int
	pendingAction profileAction
}

// NewProfileOverlay constructs a ProfileOverlay wired to the given store and theme.
func NewProfileOverlay(store state.StateReader, t theme.Theme) *ProfileOverlay {
	return &ProfileOverlay{
		store: store,
		theme: t,
	}
}

// Init returns nil — data is already in the store before the user can press 'u'.
func (p *ProfileOverlay) Init() tea.Cmd {
	return nil
}

// Update handles messages for the ProfileOverlay.
// Esc closes the overlay and resets any pending action.
// 'l' and 'f' use double-key confirmation: first press arms the action (pendingAction is set),
// second press of the same key executes it. Any other key resets the pending action and arms
// the new key if it is 'l' or 'f'.
func (p *ProfileOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return p, nil
	}

	if key.Type == tea.KeyEsc {
		p.pendingAction = profileActionNone
		return p, func() tea.Msg { return ProfileOverlayClosedMsg{} }
	}

	if key.Type == tea.KeyRunes && len(key.Runes) == 1 {
		switch key.Runes[0] {
		case 'l':
			if p.pendingAction == profileActionLogout {
				// Second 'l' press — confirm logout.
				p.pendingAction = profileActionNone
				return p, func() tea.Msg { return ProfileLogoutMsg{} }
			}
			// First 'l' press — arm confirmation and emit a warning toast.
			p.pendingAction = profileActionLogout
			return p, func() tea.Msg {
				return ProfileConfirmToastMsg{Text: "Press l again to confirm logout"}
			}
		case 'f':
			if p.pendingAction == profileActionForget {
				// Second 'f' press — confirm forget.
				p.pendingAction = profileActionNone
				return p, func() tea.Msg { return ProfileForgetMsg{} }
			}
			// First 'f' press — arm confirmation and emit a warning toast.
			p.pendingAction = profileActionForget
			return p, func() tea.Msg {
				return ProfileConfirmToastMsg{Text: "Press f again to confirm forget"}
			}
		default:
			// Any other key cancels the pending action.
			p.pendingAction = profileActionNone
		}
	}

	return p, nil
}

// View renders the profile overlay content.
// Pure function — reads store state, returns a string, performs no I/O.
func (p *ProfileOverlay) View() string {
	profile := p.store.UserProfile()
	isPremium := p.store.IsPremium()

	const innerWidth = 34 // fixed card inner content width

	var lines []string

	if profile.ID == "" {
		// Profile not yet loaded — show a loading placeholder.
		loadingStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())
		lines = append(lines, loadingStyle.Render("Loading profile..."))
	} else {
		// Display name — bold, truncated to maxProfileNameLen runes.
		nameStyle := lipgloss.NewStyle().
			Foreground(p.theme.TextPrimary()).
			Bold(true)
		name := truncateRunes(profile.DisplayName, maxProfileNameLen)
		lines = append(lines, nameStyle.Render(name))

		// Separator line.
		sepStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())
		lines = append(lines, sepStyle.Render("────────────────────"))

		// Subscription badge.
		if isPremium {
			badgeStyle := lipgloss.NewStyle().Foreground(p.theme.Info())
			lines = append(lines, badgeStyle.Render("♛  Premium"))
		} else {
			badgeStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())
			lines = append(lines, badgeStyle.Render("○  Free"))
		}

		// Country code.
		if profile.Country != "" {
			iconStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())
			codeStyle := lipgloss.NewStyle().Foreground(p.theme.TextPrimary())
			lines = append(lines, iconStyle.Render("◎  ")+codeStyle.Render(profile.Country))
		}

		// Separator before action section.
		sepStyle2 := lipgloss.NewStyle().Foreground(p.theme.TextMuted())
		lines = append(lines, "")
		lines = append(lines, sepStyle2.Render("────────────────────"))
		lines = append(lines, "")

		// Action section: logout and forget with double-key confirmation.
		lines = append(lines, p.renderActions()...)
	}

	inner := strings.Join(lines, "\n")
	inner = lipgloss.NewStyle().
		Width(innerWidth).MaxWidth(innerWidth).
		Render(inner)

	cfg := layout.BorderConfig{
		Width:       innerWidth + 2, // +2 for left/right border
		Height:      strings.Count(inner, "\n") + 3,
		Title:       "Profile",
		AccentColor: p.theme.ActiveBorder(),
		Focused:     true, // overlays are always focused
		Theme:       p.theme,
	}

	return layout.RenderPaneBorder(inner, cfg)
}

// SetSize updates the render dimensions for the overlay.
// The profile card is a fixed-size card; width/height are stored but not
// used for dynamic sizing — they are available for future resizable variants.
func (p *ProfileOverlay) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// SetTheme updates the theme reference for runtime theme switching.
func (p *ProfileOverlay) SetTheme(t theme.Theme) {
	p.theme = t
}

// renderActions returns lines for the logout/forget action section of the overlay.
// When pendingAction is set, the armed action shows a warning prompt instead of
// the normal label; the other action is shown as normal.
// Each normal action collapses to a single ListRow (label + caption on one line).
func (p *ProfileOverlay) renderActions() []string {
	const actionWidth = 34 // matches the inner content width used in View()
	warnStyle := lipgloss.NewStyle().Foreground(p.theme.Warning())

	var lines []string

	// Logout row.
	if p.pendingAction == profileActionLogout {
		lines = append(lines, warnStyle.Render("  Press l again to confirm logout"))
	} else {
		row := uikit.ListRow{
			Label:   "l  Logout",
			Caption: "ends session · keeps Client ID",
			Intent:  uikit.RolePlain,
			Theme:   p.theme,
		}
		lines = append(lines, row.Render(actionWidth))
	}

	lines = append(lines, "")

	// Forget row.
	if p.pendingAction == profileActionForget {
		lines = append(lines, warnStyle.Render("  Press f again to confirm forget"))
	} else {
		row := uikit.ListRow{
			Label:   "f  Forget",
			Caption: "removes session + Client ID",
			Intent:  uikit.RolePlain,
			Theme:   p.theme,
		}
		lines = append(lines, row.Render(actionWidth))
	}

	return lines
}

// truncateRunes truncates s to at most max runes, appending … if truncated.
// Used for display name capping so the profile card stays within its fixed width.
func truncateRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max-1]) + "…"
}
