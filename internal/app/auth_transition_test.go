package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/keychain"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestApp(needsAuth bool) *App {
	cfg := &config.Config{ClientID: "test-client"}
	opts := AppOptions{
		NeedsAuth:  needsAuth,
		ClientID:   "test-client",
		TokenStore: keychain.NewInMemoryTokenStore(),
	}
	return New(cfg, opts)
}

func TestSplash_TransitionsToStepOAuth_WhenNeedsAuth(t *testing.T) {
	a := newTestApp(true)
	require.Equal(t, viewSplash, a.currentView)

	model, cmd := a.Update(splashDismissMsg{})
	updated := model.(*App)

	assert.Equal(t, viewOnboarding, updated.currentView)
	assert.Equal(t, stepOAuth, updated.OnboardingStep())
	assert.NotNil(t, cmd, "should dispatch prepareAuthCmd")
}

func TestSplash_TransitionsToMain_WhenAuthenticated(t *testing.T) {
	a := newTestApp(false)
	require.Equal(t, viewSplash, a.currentView)

	model, cmd := a.Update(splashDismissMsg{})
	updated := model.(*App)

	assert.Equal(t, viewGrid, updated.currentView)
	assert.Nil(t, cmd, "no auth cmd needed")
}

func TestAuthSuccess_TransitionsToMain(t *testing.T) {
	a := newTestApp(true)
	a.currentView = viewOnboarding
	a.onboardingStep = stepOAuth

	// Step 1: authSuccessMsg wires up clients and resolves the spinner to Done.
	// The view stays at viewOnboarding/stepOAuth during the 1.2 s hold; only
	// SpinnerDoneMsg triggers the grid transition.
	model, cmd := a.Update(authSuccessMsg{accessToken: "test-token"})
	updated := model.(*App)

	assert.False(t, updated.needsAuth, "needsAuth must be cleared immediately")
	assert.NotNil(t, updated.player, "player should be injected")
	assert.NotNil(t, updated.library, "library should be injected")
	assert.NotNil(t, updated.search, "search should be injected")
	assert.NotNil(t, updated.devices, "devices should be injected")
	assert.NotNil(t, updated.userAPI, "userAPI should be injected")
	assert.NotNil(t, updated.playlistsAPI, "playlistsAPI should be injected")
	require.NotNil(t, cmd, "should return SpinnerDone tick cmd")

	// Step 2: execute the hold-timer cmd — it must produce SpinnerDoneMsg.
	doneMsg := cmd()
	require.IsType(t, uikit.SpinnerDoneMsg{}, doneMsg, "cmd must fire SpinnerDoneMsg")

	// Step 3: SpinnerDoneMsg triggers the grid transition and data-fetch batch.
	model2, cmd2 := updated.Update(doneMsg)
	final := model2.(*App)

	assert.Equal(t, viewGrid, final.currentView, "SpinnerDoneMsg must transition to viewGrid")
	assert.NotNil(t, cmd2, "should start data fetching batch")
}

func TestAuthError_ResolvesSpinnerToFail(t *testing.T) {
	a := newTestApp(true)
	a.currentView = viewOnboarding
	a.onboardingStep = stepOAuth

	model, _ := a.Update(authErrorMsg{err: assert.AnError})
	updated := model.(*App)

	assert.Equal(t, viewOnboarding, updated.currentView, "should stay on onboarding view")
	assert.NotEmpty(t, updated.onboardingError, "auth error must populate onboardingError")
}

func TestAuthPrepared_SetsOnboardingAuthURL(t *testing.T) {
	a := newTestApp(true)
	a.currentView = viewOnboarding
	a.onboardingStep = stepOAuth

	model, cmd := a.Update(authPreparedMsg{
		authURL:     "https://accounts.spotify.com/authorize?...",
		codeCh:      make(chan api.CallbackResult),
		verifier:    "test-verifier",
		redirectURI: "http://localhost:1234/callback",
		browserErr:  nil,
	})
	updated := model.(*App)

	assert.Equal(t, "https://accounts.spotify.com/authorize?...", updated.onboardingAuthURL)
	assert.NotNil(t, cmd, "should dispatch waitForCallbackCmd")
}

func TestQuitDuringStepOAuth(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyMsg
	}{
		{"q key", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}},
		{"Ctrl+C", tea.KeyMsg{Type: tea.KeyCtrlC}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newTestApp(true)
			a.currentView = viewOnboarding
			a.onboardingStep = stepOAuth

			_, cmd := a.Update(tt.key)
			// Execute the cmd and check it's tea.Quit
			assert.NotNil(t, cmd)
			msg := cmd()
			_, isQuit := msg.(tea.QuitMsg)
			assert.True(t, isQuit, "should quit")
		})
	}
}

func TestSplash_TransitionsToOnboarding_WhenNeedsRegister(t *testing.T) {
	cfg := &config.Config{}
	opts := AppOptions{NeedsRegister: true, TokenStore: keychain.NewInMemoryTokenStore()}
	a := New(cfg, opts)
	require.Equal(t, viewSplash, a.currentView)

	model, cmd := a.Update(splashDismissMsg{})
	updated := model.(*App)

	assert.Equal(t, viewOnboarding, updated.currentView)
	assert.Equal(t, stepRegister, updated.OnboardingStep())
	assert.Nil(t, cmd, "no command needed for stepRegister")
}

func TestNonQuitKeysDuringStepOAuth_Ignored(t *testing.T) {
	a := newTestApp(true)
	a.currentView = viewOnboarding
	a.onboardingStep = stepOAuth

	// Pressing '/' during stepOAuth should not open search
	model, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	updated := model.(*App)

	assert.Equal(t, viewOnboarding, updated.currentView)
	assert.Nil(t, cmd, "non-quit keys should be ignored during stepOAuth")
	assert.False(t, updated.searchOpen)
}

func TestCtrlC_QuitsOnboarding(t *testing.T) {
	cfg := &config.Config{}
	opts := AppOptions{NeedsRegister: true, TokenStore: keychain.NewInMemoryTokenStore()}
	a := New(cfg, opts)
	a.currentView = viewOnboarding

	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	assert.NotNil(t, cmd)
	msg := cmd()
	_, isQuit := msg.(tea.QuitMsg)
	assert.True(t, isQuit, "Ctrl+C should quit during onboarding")
}

func TestNonQuitKeysDuringOnboarding_Ignored(t *testing.T) {
	cfg := &config.Config{}
	opts := AppOptions{NeedsRegister: true, TokenStore: keychain.NewInMemoryTokenStore()}
	a := New(cfg, opts)
	a.currentView = viewOnboarding

	// Pressing '/' during stepRegister is forwarded to the text input (not the search overlay).
	// The textinput component may return a cursor-blink cmd — that is expected.
	// The critical assertion is that the search overlay was NOT opened.
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	updated := model.(*App)

	assert.Equal(t, viewOnboarding, updated.currentView)
	assert.False(t, updated.searchOpen, "search overlay must not open during onboarding")
}

// TestQuitDuringStepOAuth_ClosesCallbackServer verifies that the callback server is shut down
// when the user quits from stepOAuth (q or Ctrl+C).
func TestQuitDuringStepOAuth_ClosesCallbackServer(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyMsg
	}{
		{"q key", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}},
		{"Ctrl+C", tea.KeyMsg{Type: tea.KeyCtrlC}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			closed := false
			cfg := &config.Config{ClientID: "test-client"}
			opts := AppOptions{
				NeedsAuth:     true,
				ClientID:      "test-client",
				TokenStore:    keychain.NewInMemoryTokenStore(),
				CallbackClose: func() { closed = true },
			}
			a := New(cfg, opts)
			a.currentView = viewOnboarding
			a.onboardingStep = stepOAuth

			a.Update(tt.key)

			assert.True(t, closed, "callback server must be closed on quit during stepOAuth (%s)", tt.name)
		})
	}
}

// TestQuitDuringOnboarding_ClosesCallbackServer verifies that the callback server is shut
// down when the user quits from any onboarding step.
func TestQuitDuringOnboarding_ClosesCallbackServer(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyMsg
		step int
	}{
		{"q from stepRegister", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}, stepRegister},
		{"Ctrl+C from stepOAuth", tea.KeyMsg{Type: tea.KeyCtrlC}, stepOAuth},
		{"q from stepError", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}, stepError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			closed := false
			cfg := &config.Config{}
			opts := AppOptions{
				NeedsRegister: true,
				TokenStore:    keychain.NewInMemoryTokenStore(),
				CallbackClose: func() { closed = true },
			}
			a := New(cfg, opts)
			a.currentView = viewOnboarding
			a.onboardingStep = tt.step

			a.Update(tt.key)

			assert.True(t, closed, "callback server must be closed on quit during onboarding (%s)", tt.name)
		})
	}
}

// TestAuthSuccess_ClosesCallbackServer verifies that authSuccessMsg closes the callback
// server so it does not leak after a successful OAuth flow.
func TestAuthSuccess_ClosesCallbackServer(t *testing.T) {
	closed := false
	cfg := &config.Config{ClientID: "test-client"}
	opts := AppOptions{
		NeedsAuth:     true,
		ClientID:      "test-client",
		TokenStore:    keychain.NewInMemoryTokenStore(),
		CallbackClose: func() { closed = true },
	}
	a := New(cfg, opts)
	a.currentView = viewOnboarding
	a.onboardingStep = stepOAuth

	a.Update(authSuccessMsg{accessToken: "tok"})

	assert.True(t, closed, "callback server must be closed on auth success")
}

// TestOnboardingRetry_ServerStaysAlive verifies that pressing 'r' on the error step does
// NOT close the callback server — it must remain alive for the user to retry OAuth.
func TestOnboardingRetry_ServerStaysAlive(t *testing.T) {
	closed := false
	cfg := &config.Config{}
	opts := AppOptions{
		NeedsRegister: true,
		TokenStore:    keychain.NewInMemoryTokenStore(),
		CallbackClose: func() { closed = true },
	}
	a := New(cfg, opts)
	a.currentView = viewOnboarding
	a.onboardingStep = stepError
	a.clientID = "test-client"

	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})

	assert.False(t, closed, "callback server must stay alive when user retries (r)")
}

// TestOnboardingRelaunch_ServerStaysAlive verifies that pressing 'l' on the error step
// does NOT close the callback server — the re-launched OAuth needs it alive.
func TestOnboardingRelaunch_ServerStaysAlive(t *testing.T) {
	closed := false
	cfg := &config.Config{}
	opts := AppOptions{
		NeedsRegister: true,
		TokenStore:    keychain.NewInMemoryTokenStore(),
		CallbackClose: func() { closed = true },
	}
	a := New(cfg, opts)
	a.currentView = viewOnboarding
	a.onboardingStep = stepError
	a.clientID = "test-client"

	a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})

	assert.False(t, closed, "callback server must stay alive when user re-launches OAuth (l)")
}
