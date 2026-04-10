package panes

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestProfileOverlay creates a ProfileOverlay wired to a fresh store and black theme.
func newTestProfileOverlay() (*ProfileOverlay, *state.Store) {
	s := state.New()
	t := theme.Load("black")
	return NewProfileOverlay(s, t), s
}

// TestProfileOverlay_View_ShowsDisplayName verifies that View() renders the user's display name.
func TestProfileOverlay_View_ShowsDisplayName(t *testing.T) {
	overlay, store := newTestProfileOverlay()
	overlay.SetSize(40, 12)
	store.SetUserProfile(domain.UserProfile{
		ID:          "user123",
		DisplayName: "Irshad Sheikh",
		Product:     "premium",
		Country:     "DE",
	})

	view := overlay.View()

	assert.Contains(t, view, "Irshad Sheikh", "View should render the user's display name")
}

// TestProfileOverlay_View_PremiumBadge verifies that a premium user sees the ♛ Premium badge.
func TestProfileOverlay_View_PremiumBadge(t *testing.T) {
	overlay, store := newTestProfileOverlay()
	overlay.SetSize(40, 12)
	store.SetUserProfile(domain.UserProfile{
		ID:          "user123",
		DisplayName: "Irshad Sheikh",
		Product:     "premium",
		Country:     "DE",
	})

	view := overlay.View()

	assert.Contains(t, view, "♛", "View should render ♛ for Premium users")
	assert.Contains(t, view, "Premium", "View should render 'Premium' text for premium users")
}

// TestProfileOverlay_View_FreeBadge verifies that a free-tier user sees the ○ Free badge.
func TestProfileOverlay_View_FreeBadge(t *testing.T) {
	overlay, store := newTestProfileOverlay()
	overlay.SetSize(40, 12)
	store.SetUserProfile(domain.UserProfile{
		ID:          "user456",
		DisplayName: "Free User",
		Product:     "free",
		Country:     "US",
	})

	view := overlay.View()

	assert.Contains(t, view, "○", "View should render ○ for Free users")
	assert.Contains(t, view, "Free", "View should render 'Free' text for free users")
}

// TestProfileOverlay_View_ShowsCountry verifies that View() renders the country code.
func TestProfileOverlay_View_ShowsCountry(t *testing.T) {
	overlay, store := newTestProfileOverlay()
	overlay.SetSize(40, 12)
	store.SetUserProfile(domain.UserProfile{
		ID:          "user123",
		DisplayName: "Irshad Sheikh",
		Product:     "premium",
		Country:     "DE",
	})

	view := overlay.View()

	assert.Contains(t, view, "DE", "View should render the country code")
	assert.Contains(t, view, "◎", "View should render ◎ before country code")
}

// TestProfileOverlay_View_LoadingState verifies that View() renders "Loading profile..."
// when the profile has not yet been loaded (ID is empty).
func TestProfileOverlay_View_LoadingState(t *testing.T) {
	overlay, _ := newTestProfileOverlay()
	overlay.SetSize(40, 12)
	// Store has zero-value UserProfile (ID == "")

	view := overlay.View()

	assert.Contains(t, view, "Loading profile...", "View should show loading placeholder when profile not loaded")
}

// TestProfileOverlay_View_NoEscHint verifies that View() does NOT contain any "esc" or
// "close" text. Esc-to-close is a universal convention; the hint is redundant noise.
func TestProfileOverlay_View_NoEscHint(t *testing.T) {
	overlay, store := newTestProfileOverlay()
	overlay.SetSize(40, 12)
	store.SetUserProfile(domain.UserProfile{
		ID:          "user123",
		DisplayName: "Irshad Sheikh",
		Product:     "premium",
		Country:     "DE",
	})

	view := overlay.View()

	assert.NotContains(t, view, "esc", "View should NOT contain 'esc' hint — universal convention")
	assert.NotContains(t, view, "close", "View should NOT contain 'close' hint — redundant noise")
}

// TestProfileOverlay_View_HasBorderCorners verifies that the overlay uses rounded border corners.
func TestProfileOverlay_View_HasBorderCorners(t *testing.T) {
	overlay, store := newTestProfileOverlay()
	overlay.SetSize(40, 12)
	store.SetUserProfile(domain.UserProfile{
		ID:          "user123",
		DisplayName: "Irshad Sheikh",
		Product:     "premium",
		Country:     "DE",
	})

	view := overlay.View()

	assert.Contains(t, view, "╭", "overlay should use rounded corner ╭")
	assert.Contains(t, view, "╰", "overlay should use rounded corner ╰")
}

// TestProfileOverlay_EscEmitsClosedMsg verifies that Esc key produces ProfileOverlayClosedMsg.
func TestProfileOverlay_EscEmitsClosedMsg(t *testing.T) {
	overlay, _ := newTestProfileOverlay()

	_, cmd := overlay.Update(tea.KeyMsg{Type: tea.KeyEsc})

	require.NotNil(t, cmd, "Esc should return a command")
	msg := cmd()
	_, ok := msg.(ProfileOverlayClosedMsg)
	require.True(t, ok, "Esc should produce ProfileOverlayClosedMsg, got %T", msg)
}

// TestProfileOverlay_OtherKeysIgnored verifies that non-Esc keys do not close the overlay.
func TestProfileOverlay_OtherKeysIgnored(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyMsg
	}{
		{"rune q", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}},
		{"enter", tea.KeyMsg{Type: tea.KeyEnter}},
		{"tab", tea.KeyMsg{Type: tea.KeyTab}},
		{"rune j", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overlay, _ := newTestProfileOverlay()
			_, cmd := overlay.Update(tt.key)

			// Either cmd is nil, or the command does NOT produce ProfileOverlayClosedMsg.
			if cmd != nil {
				msg := cmd()
				_, isClose := msg.(ProfileOverlayClosedMsg)
				assert.False(t, isClose, "key %q should not produce ProfileOverlayClosedMsg", tt.name)
			}
		})
	}
}

// TestProfileOverlay_Init_ReturnsNil verifies that Init() returns nil.
func TestProfileOverlay_Init_ReturnsNil(t *testing.T) {
	overlay, _ := newTestProfileOverlay()
	cmd := overlay.Init()
	assert.Nil(t, cmd, "Init() should return nil — data is already in the store")
}

// TestProfileOverlay_SetSize verifies that SetSize stores the dimensions.
func TestProfileOverlay_SetSize(t *testing.T) {
	overlay, _ := newTestProfileOverlay()
	overlay.SetSize(40, 12)
	assert.Equal(t, 40, overlay.width, "SetSize should store width")
	assert.Equal(t, 12, overlay.height, "SetSize should store height")
}

// TestProfileOverlay_SetTheme verifies that after calling SetTheme the overlay renders
// without panicking and reflects the new theme (smoke test).
func TestProfileOverlay_SetTheme(t *testing.T) {
	overlay, store := newTestProfileOverlay()
	overlay.SetSize(40, 12)
	store.SetUserProfile(domain.UserProfile{
		ID:          "user1",
		DisplayName: "Theme Switcher",
		Product:     "premium",
		Country:     "US",
	})

	// Switch to a different theme.
	newTheme := theme.Load("dracula")
	overlay.SetTheme(newTheme)

	// View must not panic and must still render content.
	view := overlay.View()
	assert.NotEmpty(t, view, "View should return non-empty content after SetTheme")
	assert.Contains(t, view, "Theme Switcher", "overlay should render profile name after SetTheme")
}
