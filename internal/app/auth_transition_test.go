package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/keychain"
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

func TestSplash_TransitionsToAuth_WhenNeedsAuth(t *testing.T) {
	a := newTestApp(true)
	require.Equal(t, viewSplash, a.currentView)

	model, cmd := a.Update(splashDismissMsg{})
	updated := model.(*App)

	assert.Equal(t, viewAuth, updated.currentView)
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
	a.currentView = viewAuth

	model, cmd := a.Update(authSuccessMsg{accessToken: "test-token"})
	updated := model.(*App)

	assert.Equal(t, viewGrid, updated.currentView)
	assert.False(t, updated.needsAuth)
	assert.NotNil(t, updated.player, "player should be injected")
	assert.NotNil(t, updated.library, "library should be injected")
	assert.NotNil(t, updated.search, "search should be injected")
	assert.NotNil(t, updated.devices, "devices should be injected")
	assert.NotNil(t, updated.userAPI, "userAPI should be injected")
	assert.NotNil(t, updated.playlistsAPI, "playlistsAPI should be injected")
	assert.NotNil(t, cmd, "should start data fetching batch")
}

func TestAuthError_ShowsMessage(t *testing.T) {
	a := newTestApp(true)
	a.currentView = viewAuth

	model, _ := a.Update(authErrorMsg{err: assert.AnError})
	updated := model.(*App)

	assert.Equal(t, viewAuth, updated.currentView, "should stay on auth view")
	assert.Contains(t, updated.authStatus, "Error")
}

func TestAuthPrepared_SetsURLAndStatus(t *testing.T) {
	a := newTestApp(true)
	a.currentView = viewAuth

	model, cmd := a.Update(authPreparedMsg{
		authURL:     "https://accounts.spotify.com/authorize?...",
		codeCh:      make(chan api.CallbackResult),
		verifier:    "test-verifier",
		redirectURI: "http://localhost:1234/callback",
		serverClose: func() {},
		browserErr:  nil,
	})
	updated := model.(*App)

	assert.Equal(t, "https://accounts.spotify.com/authorize?...", updated.authURL)
	assert.Contains(t, updated.authStatus, "Waiting")
	assert.NotNil(t, cmd, "should dispatch waitForCallbackCmd")
}

func TestAuthPrepared_BrowserFailed_ShowsURLPrompt(t *testing.T) {
	a := newTestApp(true)
	a.currentView = viewAuth

	model, _ := a.Update(authPreparedMsg{
		authURL:     "https://accounts.spotify.com/authorize?...",
		codeCh:      make(chan api.CallbackResult),
		verifier:    "test-verifier",
		redirectURI: "http://localhost:1234/callback",
		serverClose: func() {},
		browserErr:  assert.AnError,
	})
	updated := model.(*App)

	assert.Contains(t, updated.authStatus, "Could not open browser")
}

func TestQuitDuringAuth(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyMsg
	}{
		{"q key", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}},
		{"Esc key", tea.KeyMsg{Type: tea.KeyEsc}},
		{"Ctrl+C", tea.KeyMsg{Type: tea.KeyCtrlC}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newTestApp(true)
			a.currentView = viewAuth

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

func TestNonQuitKeysDuringAuth_Ignored(t *testing.T) {
	a := newTestApp(true)
	a.currentView = viewAuth

	// Pressing '/' during auth should not open search
	model, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	updated := model.(*App)

	assert.Equal(t, viewAuth, updated.currentView)
	assert.Nil(t, cmd, "non-quit keys should be ignored during auth")
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

	// Pressing '/' during onboarding should not open search (API clients are nil)
	model, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	updated := model.(*App)

	assert.Equal(t, viewOnboarding, updated.currentView)
	assert.Nil(t, cmd, "non-quit keys should be ignored during onboarding")
	assert.False(t, updated.searchOpen)
}
