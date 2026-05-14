// Package panes — ProfileOverlay is the floating user profile overlay.
// It displays the authenticated user's display name, subscription tier,
// and country. It reads directly from the Store and emits ProfileOverlayClosedMsg
// on Esc. Triggered by the 'u' key; never imports api/ directly.
package panes

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// maxProfileNameLen is the maximum runes shown for the display name.
// Sized to fit innerWidth=22 with the widest ASCII glyph (3 cols) + 2-space gap.
const maxProfileNameLen = 17

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
	err           error // set when a self-triggered fetch fails; cleared on success
}

// NewProfileOverlay constructs a ProfileOverlay wired to the given store and theme.
func NewProfileOverlay(store state.StateReader, t theme.Theme) *ProfileOverlay {
	return &ProfileOverlay{
		store: store,
		theme: t,
	}
}

// Init returns a FetchCurrentUserRequestMsg command when the store has no user
// profile loaded, triggering a fetch from the app layer. Returns nil when the
// profile is already available.
func (p *ProfileOverlay) Init() tea.Cmd {
	if p.store.UserProfile().ID == "" {
		return func() tea.Msg { return FetchCurrentUserRequestMsg{} }
	}
	return nil
}

// Update handles messages for the ProfileOverlay.
// Esc closes the overlay and resets any pending action.
// 'l' and 'f' use double-key confirmation: first press arms the action (pendingAction is set),
// second press of the same key executes it. Any other key resets the pending action and arms
// the new key if it is 'l' or 'f'.
// UserProfileLoadedMsg stores or clears the error field for the error/loading state.
func (p *ProfileOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle UserProfileLoadedMsg — stores error or clears it on success.
	if m, ok := msg.(UserProfileLoadedMsg); ok {
		p.err = m.Err
		return p, nil
	}

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

	const innerWidth = 22 // narrow card: 3 short rows + 1 keybar line

	mode := uikit.ActiveMode()

	var lines []string

	if p.err != nil {
		// Error state — distinct from loading and from legitimate empty profile.
		errStyle := lipgloss.NewStyle().Foreground(p.theme.TextPrimary())
		hintStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())
		lines = append(lines, errStyle.Render("Profile unavailable"))
		lines = append(lines, hintStyle.Render("Check your connection."))
	} else if profile.ID == "" {
		loadingStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())
		lines = append(lines, loadingStyle.Render("Loading profile..."))
	} else {
		// Row 1 — Name.
		name := truncateRunes(profile.DisplayName, maxProfileNameLen)
		lines = append(lines, p.renderRow(
			uikit.GlyphFor(uikit.GlyphActive, mode),
			p.theme.Info(),
			p.theme.TextPrimary(),
			name,
		))

		// Row 2 — Plan.
		var planGlyph string
		var planValue string
		var planColor lipgloss.Color
		if isPremium {
			planGlyph = uikit.GlyphFor(uikit.GlyphPremium, mode)
			planValue = "Premium"
			planColor = p.theme.Info()
		} else {
			planGlyph = uikit.GlyphFor(uikit.GlyphFreeTier, mode)
			planValue = "Free"
			planColor = p.theme.TextMuted()
		}
		lines = append(lines, p.renderRow(
			planGlyph,
			planColor,
			p.theme.TextPrimary(),
			planValue,
		))

		// Row 3 — Region (only if known).
		if profile.Country != "" {
			lines = append(lines, p.renderRow(
				uikit.GlyphFor(uikit.GlyphInactive, mode),
				p.theme.TextMuted(),
				p.theme.TextPrimary(),
				profile.Country,
			))
		}

		// Spacer + KeyBar action line.
		lines = append(lines, "")
		lines = append(lines, p.renderActions())
	}

	inner := strings.Join(lines, "\n")
	inner = lipgloss.NewStyle().
		Width(innerWidth).MaxWidth(innerWidth).
		Render(inner)

	chrome := uikit.PaneChrome{
		Width:       innerWidth + 2,
		Height:      strings.Count(inner, "\n") + 3,
		Title:       "Profile",
		AccentColor: p.theme.ActiveBorder(),
		Focused:     true,
		Theme:       p.theme,
	}

	return chrome.Render(inner)
}

// SetSize updates the render dimensions for the overlay.
// The profile card is a fixed-size card; width/height are stored but not
// used for dynamic sizing — they are available for future resizable variants.
func (p *ProfileOverlay) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// Err returns the last fetch error, or nil if the profile loaded successfully.
// Exported for test helpers; the overlay uses this to decide whether to show
// the error state in View().
func (p *ProfileOverlay) Err() error {
	return p.err
}

// SetTheme updates the theme reference for runtime theme switching.
func (p *ProfileOverlay) SetTheme(t theme.Theme) {
	p.theme = t
}

// renderRow renders a single icon+value line: "<glyph>  <value>".
// glyph and value get independent foreground colors so the glyph can carry
// intent (Info for premium, TextMuted for region) while the value uses the
// pane's primary text color.
func (p *ProfileOverlay) renderRow(glyph string, glyphColor, valueColor lipgloss.Color, value string) string {
	g := lipgloss.NewStyle().Foreground(glyphColor).Render(glyph)
	v := lipgloss.NewStyle().Foreground(valueColor).Render(value)
	return g + "  " + v
}

// renderActions returns the single-line KeyBar advertising logout/forget.
// Real key handling lives in Update() — these key.Bindings are display-only.
// Confirmation feedback is delivered via toast (ProfileConfirmToastMsg);
// the rendered hint stays static regardless of pendingAction.
func (p *ProfileOverlay) renderActions() string {
	bindings := []key.Binding{
		key.NewBinding(key.WithHelp("l", "logout")),
		key.NewBinding(key.WithHelp("f", "forget")),
	}
	return uikit.KeyBar{Bindings: bindings, Theme: p.theme}.Render()
}

// truncateRunes truncates s to at most max runes, appending the ellipsis glyph if truncated.
// Used for display name capping so the profile card stays within its fixed width.
func truncateRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	ell := uikit.GlyphFor(uikit.GlyphEllipsis, uikit.ActiveMode())
	ellRunes := utf8.RuneCountInString(ell)
	runes := []rune(s)
	return string(runes[:max-ellRunes]) + ell
}
