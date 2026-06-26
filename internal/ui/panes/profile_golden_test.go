package panes_test

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/goldentest"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// TestProfileOverlay_View_Premium verifies the golden snapshot of ProfileOverlay
// for a premium user with display name and country at 80×24.
func TestProfileOverlay_View_Premium(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetUserProfile(domain.UserProfile{
		ID:          "user123",
		DisplayName: "Alice Johnson",
		Product:     "premium",
		Country:     "US",
	})

	overlay := panes.NewProfileOverlay(s, th)
	overlay.SetSize(80, 24)

	tm := goldentest.NewPaneTest(t, overlay, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestProfileOverlay_View_Free verifies the golden snapshot of ProfileOverlay
// for a free tier user with Free badge at 80×24.
func TestProfileOverlay_View_Free(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetUserProfile(domain.UserProfile{
		ID:          "user456",
		DisplayName: "Bob Smith",
		Product:     "free",
		Country:     "GB",
	})

	overlay := panes.NewProfileOverlay(s, th)
	overlay.SetSize(80, 24)

	tm := goldentest.NewPaneTest(t, overlay, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestProfileOverlay_View_Loading verifies the golden snapshot of ProfileOverlay
// when the profile has not yet been fetched (loading state) at 80×24.
func TestProfileOverlay_View_Loading(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	overlay := panes.NewProfileOverlay(s, th)
	overlay.SetSize(80, 24)

	tm := goldentest.NewPaneTest(t, overlay, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestProfileOverlay_View_Error verifies the golden snapshot of ProfileOverlay
// when the profile fetch returned an error at 80×24.
func TestProfileOverlay_View_Error(t *testing.T) {
	s := state.New()
	th := theme.Load("black")

	overlay := panes.NewProfileOverlay(s, th)
	overlay.SetSize(80, 24)

	// Simulate a failed profile fetch by sending UserProfileLoadedMsg with an error.
	overlay.Update(panes.UserProfileLoadedMsg{Err: errors.New("network timeout")})

	tm := goldentest.NewPaneTest(t, overlay, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestProfileOverlay_View_LogoutConfirmation verifies the golden snapshot of
// ProfileOverlay after 'l' is pressed once (logout confirmation pending) at 80×24.
func TestProfileOverlay_View_LogoutConfirmation(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetUserProfile(domain.UserProfile{
		ID:          "user123",
		DisplayName: "Alice Johnson",
		Product:     "premium",
		Country:     "US",
	})

	overlay := panes.NewProfileOverlay(s, th)
	overlay.SetSize(80, 24)

	// Press 'l' once to arm logout confirmation.
	overlay.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	tm := goldentest.NewPaneTest(t, overlay, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}

// TestProfileOverlay_View_ForgetConfirmation verifies the golden snapshot of
// ProfileOverlay after 'f' is pressed once (forget confirmation pending) at 80×24.
func TestProfileOverlay_View_ForgetConfirmation(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	s.SetUserProfile(domain.UserProfile{
		ID:          "user123",
		DisplayName: "Alice Johnson",
		Product:     "premium",
		Country:     "US",
	})

	overlay := panes.NewProfileOverlay(s, th)
	overlay.SetSize(80, 24)

	// Press 'f' once to arm forget confirmation.
	overlay.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})

	tm := goldentest.NewPaneTest(t, overlay, 80, 24)
	goldentest.AssertGolden(t, goldentest.WaitAndReadOutput(t, tm))
}
