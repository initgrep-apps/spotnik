package app

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestSplash_TransitionsToAuth_WhenNeedsAuth(t *testing.T) {
	cfg := &config.Config{}
	a := New(cfg, AppOptions{NeedsAuth: true})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	// Send splashDismissMsg — should transition to viewAuth.
	model, cmd := a.Update(splashDismissMsg{})
	a = model.(*App)

	assert.Equal(t, viewAuth, a.currentView)
	assert.NotNil(t, cmd, "should return prepareAuthCmd")

	output := a.View()
	assert.Contains(t, output, "Authentication Required")
}

func TestSplash_TransitionsToMain_WhenAuthenticated(t *testing.T) {
	cfg := &config.Config{}
	a := New(cfg, AppOptions{NeedsAuth: false})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	model, _ := a.Update(splashDismissMsg{})
	a = model.(*App)

	assert.Equal(t, viewMain, a.currentView)
}

func TestAuthSuccess_TransitionsToMain(t *testing.T) {
	cfg := &config.Config{}
	a := New(cfg, AppOptions{NeedsAuth: true})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	// Transition to auth view first.
	model, _ := a.Update(splashDismissMsg{})
	a = model.(*App)

	// Simulate auth success.
	model, cmd := a.Update(authSuccessMsg{accessToken: "test-token"})
	a = model.(*App)

	assert.Equal(t, viewMain, a.currentView)
	assert.False(t, a.needsAuth)
	assert.NotNil(t, cmd, "should start data fetching commands")
}

func TestAuthError_ShowsMessage(t *testing.T) {
	cfg := &config.Config{}
	a := New(cfg, AppOptions{NeedsAuth: true})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	// Transition to auth view.
	model, _ := a.Update(splashDismissMsg{})
	a = model.(*App)

	// Simulate auth error.
	model, _ = a.Update(authErrorMsg{err: fmt.Errorf("token exchange failed")})
	a = model.(*App)

	assert.Equal(t, viewAuth, a.currentView, "should stay on auth view")
	output := a.View()
	assert.Contains(t, output, "Auth failed")
}

func TestAuthPrepared_SetsURLAndStatus(t *testing.T) {
	cfg := &config.Config{}
	a := New(cfg, AppOptions{NeedsAuth: true})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	// Transition to auth view.
	model, _ := a.Update(splashDismissMsg{})
	a = model.(*App)

	// Simulate auth prepared (browser opened successfully).
	model, cmd := a.Update(authPreparedMsg{
		authURL:     "https://accounts.spotify.com/authorize?client_id=test",
		browserErr:  nil,
		codeCh:      make(chan api.CallbackResult),
		verifier:    "test-verifier",
		redirectURI: "http://localhost:12345/callback",
		serverClose: func() {},
	})
	a = model.(*App)

	assert.Contains(t, a.authURL, "accounts.spotify.com")
	assert.Contains(t, a.authStatus, "Waiting for authorization")
	assert.NotNil(t, cmd, "should start waitForCallbackCmd")

	output := a.View()
	assert.Contains(t, output, "accounts.spotify.com")
}

func TestAuthPrepared_BrowserFailed_ShowsURLPrompt(t *testing.T) {
	cfg := &config.Config{}
	a := New(cfg, AppOptions{NeedsAuth: true})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	model, _ := a.Update(splashDismissMsg{})
	a = model.(*App)

	model, _ = a.Update(authPreparedMsg{
		authURL:     "https://accounts.spotify.com/authorize?client_id=test",
		browserErr:  fmt.Errorf("no browser"),
		codeCh:      make(chan api.CallbackResult),
		verifier:    "test-verifier",
		redirectURI: "http://localhost:12345/callback",
		serverClose: func() {},
	})
	a = model.(*App)

	assert.Contains(t, a.authStatus, "Open this URL")
}

func TestQuitDuringAuth(t *testing.T) {
	cfg := &config.Config{}
	a := New(cfg, AppOptions{NeedsAuth: true})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	model, _ := a.Update(splashDismissMsg{})
	a = model.(*App)

	// Press q during auth — should quit.
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	assert.NotNil(t, cmd, "q during auth should produce a quit command")

	// Verify the command produces a QuitMsg.
	if cmd != nil {
		msg := cmd()
		_, isQuit := msg.(tea.QuitMsg)
		assert.True(t, isQuit, "q during auth should produce QuitMsg")
	}
}

func TestEscDuringAuth(t *testing.T) {
	cfg := &config.Config{}
	a := New(cfg, AppOptions{NeedsAuth: true})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	model, _ := a.Update(splashDismissMsg{})
	a = model.(*App)

	// Press Esc during auth — should quit.
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.NotNil(t, cmd, "Esc during auth should produce a quit command")

	if cmd != nil {
		msg := cmd()
		_, isQuit := msg.(tea.QuitMsg)
		assert.True(t, isQuit, "Esc during auth should produce QuitMsg")
	}
}

func TestSplashDismiss_IgnoredWhenNotInSplash(t *testing.T) {
	cfg := &config.Config{}
	a := New(cfg, AppOptions{NeedsAuth: false})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	// First dismiss moves to viewMain.
	model, _ := a.Update(splashDismissMsg{})
	a = model.(*App)
	assert.Equal(t, viewMain, a.currentView)

	// Second dismiss should be a no-op (already in main).
	model, _ = a.Update(splashDismissMsg{})
	a = model.(*App)
	assert.Equal(t, viewMain, a.currentView, "should remain in main view")
}
