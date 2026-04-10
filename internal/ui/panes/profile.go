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
)

// maxProfileNameLen is the maximum number of runes shown for the display name.
const maxProfileNameLen = 20

// ProfileOverlay renders the authenticated user's profile as a floating overlay.
// Reads directly from the Store — no local state beyond width/height.
// Triggered by the 'u' key; closed by Esc.
type ProfileOverlay struct {
	store  state.StateReader
	theme  theme.Theme
	width  int
	height int
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
// Only tea.KeyEsc is handled — it emits ProfileOverlayClosedMsg.
// All other messages are ignored so the overlay does not interfere with
// background message processing while it is visible.
func (p *ProfileOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyEsc {
		return p, func() tea.Msg { return ProfileOverlayClosedMsg{} }
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

// truncateRunes truncates s to at most max runes, appending … if truncated.
// Used for display name capping so the profile card stays within its fixed width.
func truncateRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max-1]) + "…"
}
