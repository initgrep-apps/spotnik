package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
)

// OpenProfileOverlay opens the profile overlay and runs its Init command.
// Returns the App and any command from the overlay's Init(). Used in tests
// to simulate the 'u' key path without having to send key events.
func (a *App) OpenProfileOverlay() (*App, tea.Cmd) {
	a.profileOverlayOpen = true
	cmd := a.profilePane.Init()
	return a, cmd
}

// ProfilePaneErr returns the current error on the profile overlay, or nil if none.
// Used in tests to verify that UserProfileLoadedMsg forwarding works correctly.
func (a *App) ProfilePaneErr() error {
	if a.profilePane == nil {
		return nil
	}
	return a.profilePane.Err()
}

// InjectUserProfileLoadedErr sends a UserProfileLoadedMsg with the given error
// to the profile overlay, simulating a failed fetch forwarded from the app layer.
func (a *App) InjectUserProfileLoadedErr(err error) {
	updated, _ := a.profilePane.Update(panes.UserProfileLoadedMsg{Err: err})
	if p, ok := updated.(*panes.ProfileOverlay); ok {
		a.profilePane = p
	}
}
