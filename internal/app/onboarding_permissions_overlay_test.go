package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApp_V_OnStepOAuth_OpensPermissionsOverlay verifies that pressing 'v'
// while the onboarding step is stepOAuth opens the permissions overlay.
func TestApp_V_OnStepOAuth_OpensPermissionsOverlay(t *testing.T) {
	a := newRenderTestApp()
	a.currentView = viewOnboarding
	a.onboardingStep = stepOAuth
	require.Nil(t, a.onboardingPermissionsOverlay, "overlay must start closed")

	updated, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	app, ok := updated.(*App)
	require.True(t, ok)

	assert.NotNil(t, app.onboardingPermissionsOverlay,
		"pressing v on stepOAuth must open the permissions overlay")
}

// TestApp_V_OnStepRegister_DoesNotOpenOverlay verifies that 'v' while
// onboardingStep is stepRegister does not open the overlay (it should pass
// through to the FormField, which treats 'v' as a hex character).
func TestApp_V_OnStepRegister_DoesNotOpenOverlay(t *testing.T) {
	a := newRenderTestApp()
	a.currentView = viewOnboarding
	a.onboardingStep = stepRegister

	updated, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	app, ok := updated.(*App)
	require.True(t, ok)

	assert.Nil(t, app.onboardingPermissionsOverlay,
		"v on stepRegister must not open the permissions overlay")
}

// TestApp_PermissionsOverlay_EscClosesOverlay verifies that the overlay's
// close message clears the overlay state.
func TestApp_PermissionsOverlay_EscClosesOverlay(t *testing.T) {
	a := newRenderTestApp()
	a.currentView = viewOnboarding
	a.onboardingStep = stepOAuth
	a.openOnboardingPermissions()
	require.NotNil(t, a.onboardingPermissionsOverlay)

	updated, _ := a.Update(panes.OnboardingPermissionsOverlayClosedMsg{})
	app, ok := updated.(*App)
	require.True(t, ok)

	assert.Nil(t, app.onboardingPermissionsOverlay,
		"OnboardingPermissionsOverlayClosedMsg must clear the overlay")
}

// TestApp_PermissionsOverlay_KeysRoutedThroughOverlayWhenOpen verifies that
// while the overlay is open, key events go to it (modal) — pressing Esc
// triggers the close round-trip.
func TestApp_PermissionsOverlay_KeysRoutedThroughOverlayWhenOpen(t *testing.T) {
	a := newRenderTestApp()
	a.currentView = viewOnboarding
	a.onboardingStep = stepOAuth
	a.openOnboardingPermissions()
	require.NotNil(t, a.onboardingPermissionsOverlay)

	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd, "Esc while overlay open must produce a close cmd")

	closeMsg := cmd()
	_, ok := closeMsg.(panes.OnboardingPermissionsOverlayClosedMsg)
	require.True(t, ok)

	updated, _ := a.Update(closeMsg)
	app, ok := updated.(*App)
	require.True(t, ok)
	assert.Nil(t, app.onboardingPermissionsOverlay)
}
