package app

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/goldentest"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// newOnboardingTestApp creates a minimal App with onboarding state for golden tests.
func newOnboardingTestApp() *App {
	cfg := &config.Config{}
	cfg.Preferences.Theme = theme.DefaultThemeID
	return New(cfg, AppOptions{})
}

// TestOnboardingRegister_View_EmptyField verifies the golden snapshot of
// Step 1 (registration) with an empty client ID field at 120×40.
func TestOnboardingRegister_View_EmptyField(t *testing.T) {
	a := newOnboardingTestApp()
	a.width = 120
	a.height = 40
	a.onboardingPort = 8888
	a.onboardingStep = stepRegister

	got := a.renderOnboardingRegister()
	goldentest.AssertGolden(t, got)
}

// TestOnboardingRegister_View_WithInput verifies the golden snapshot of
// Step 1 (registration) with "abc123" typed in the client ID field at 120×40.
func TestOnboardingRegister_View_WithInput(t *testing.T) {
	a := newOnboardingTestApp()
	a.width = 120
	a.height = 40
	a.onboardingPort = 8888
	a.onboardingStep = stepRegister
	a.onboardingField.SetValue("abc123")

	got := a.renderOnboardingRegister()
	goldentest.AssertGolden(t, got)
}

// TestOnboardingRegister_View_ValidationError verifies the golden snapshot of
// Step 1 (registration) with an invalid input submitted, showing the error
// glyph and message below the FormField at 120×40.
func TestOnboardingRegister_View_ValidationError(t *testing.T) {
	a := newOnboardingTestApp()
	a.width = 120
	a.height = 40
	a.onboardingPort = 8888
	a.onboardingStep = stepRegister
	a.onboardingField.SetValue("bad")
	_ = a.onboardingField.Validate()

	got := a.renderOnboardingRegister()
	goldentest.AssertGolden(t, got)
}

// TestOnboardingOAuth_View_SpinnerRunning verifies the golden snapshot of
// Step 2 (OAuth) with the auth URL displayed, spinner running, and key hints
// at 120×40.
func TestOnboardingOAuth_View_SpinnerRunning(t *testing.T) {
	a := newOnboardingTestApp()
	a.width = 120
	a.height = 40
	a.onboardingStep = stepOAuth
	a.onboardingAuthURL = "https://accounts.spotify.com/authorize?client_id=abc123&scope=user-read-playback-state"

	got := a.renderOnboardingOAuth()
	goldentest.AssertGolden(t, got)
}

// TestOnboardingOAuth_View_PermissionsOverlay verifies the golden snapshot of
// Step 2 (OAuth) with the permissions overlay visible on top at 120×40.
func TestOnboardingOAuth_View_PermissionsOverlay(t *testing.T) {
	a := newOnboardingTestApp()
	a.width = 120
	a.height = 40
	a.currentView = viewOnboarding
	a.onboardingStep = stepOAuth
	a.onboardingAuthURL = "https://accounts.spotify.com/authorize?client_id=abc123&scope=user-read-playback-state"
	a.openOnboardingPermissions()

	got := a.renderOnboarding()
	goldentest.AssertGolden(t, got)
}

// TestOnboardingError_View_Default verifies the golden snapshot of
// Step 2 error screen with error message, common causes, and key hints
// at 120×40.
func TestOnboardingError_View_Default(t *testing.T) {
	a := newOnboardingTestApp()
	a.width = 120
	a.height = 40
	a.onboardingPort = 8888
	a.onboardingStep = stepError
	a.onboardingError = "authorization code not received"

	got := a.renderOnboardingError()
	goldentest.AssertGolden(t, got)
}

// TestOnboardingError_View_WithPermissionsOverlay verifies the golden snapshot
// of Step 2 error screen with the permissions overlay visible on top at 120×40.
func TestOnboardingError_View_WithPermissionsOverlay(t *testing.T) {
	a := newOnboardingTestApp()
	a.width = 120
	a.height = 40
	a.currentView = viewOnboarding
	a.onboardingStep = stepError
	a.onboardingError = "authorization code not received"
	a.openOnboardingPermissions()

	got := a.renderOnboarding()
	goldentest.AssertGolden(t, got)
}
