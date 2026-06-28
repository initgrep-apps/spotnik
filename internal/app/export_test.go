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

// EpisodeDetailsOpen returns true if the episode details overlay is currently open.
func (a *App) EpisodeDetailsOpen() bool {
	return a.episodeDetailsOpen
}

// SplashDismissMsgForTest is exported for use in app_test package tests.
type SplashDismissMsgForTest = splashDismissMsg

// AutoSwitchPreset wraps autoSwitchPreset for test access.
func (a *App) AutoSwitchPreset(forContentType string) tea.Cmd {
	return a.autoSwitchPreset(forContentType)
}

// IsCurrentPresetPodcastOriented wraps isCurrentPresetPodcastOriented for test access.
func (a *App) IsCurrentPresetPodcastOriented() bool {
	return a.isCurrentPresetPodcastOriented()
}

// ErrNilClientForTest exposes the unexported errNilClient sentinel so external
// test packages (app_test) can construct ToggleLikeResultMsg values that
// exercise the errNilClient rollback branch of the routing handler.
var ErrNilClientForTest = errNilClient

// UnauthorizedMsgForTest exposes the unexported unauthorizedMsg type so
// external test packages (app_test) can type-assert on the secondary message
// dispatched by the 401 rollback branch of the routing handler.
type UnauthorizedMsgForTest = unauthorizedMsg
