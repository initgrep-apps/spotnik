package app

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/keychain"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// authPreparedMsg is sent after PKCE setup and callback server are ready.
type authPreparedMsg struct {
	authURL     string
	codeCh      <-chan api.CallbackResult
	verifier    string
	redirectURI string
	serverClose func()
	browserErr  error
}

// authSuccessMsg is sent when the OAuth code exchange succeeds.
type authSuccessMsg struct {
	accessToken string
}

// authErrorMsg is sent when the OAuth flow fails.
type authErrorMsg struct {
	err error
}

// prepareAuthCmd performs PKCE setup, starts the callback server, and opens the browser.
// It does NOT defer-close the server — the caller (waitForCallbackCmd) handles that.
func prepareAuthCmd(clientID string) tea.Cmd {
	return func() tea.Msg {
		verifier, err := api.GenerateCodeVerifier()
		if err != nil {
			return authErrorMsg{err: fmt.Errorf("generating PKCE verifier: %w", err)}
		}
		challenge := api.ComputeCodeChallenge(verifier)

		callbackSrv, codeCh, err := api.StartCallbackServer()
		if err != nil {
			return authErrorMsg{err: fmt.Errorf("starting callback server: %w", err)}
		}

		redirectURI := callbackSrv.URL + "/callback"
		authURL := api.BuildAuthURL(clientID, redirectURI, challenge, api.SpotifyScopes)

		browserErr := api.OpenBrowser(authURL)

		return authPreparedMsg{
			authURL:     authURL,
			codeCh:      codeCh,
			verifier:    verifier,
			redirectURI: redirectURI,
			serverClose: callbackSrv.Close,
			browserErr:  browserErr,
		}
	}
}

// waitForCallbackCmd blocks on the callback channel, exchanges the code for tokens,
// and closes the callback server when done.
func waitForCallbackCmd(clientID string, store keychain.TokenStore, verifier, redirectURI string, codeCh <-chan api.CallbackResult, serverClose func()) tea.Cmd {
	return func() tea.Msg {
		defer serverClose()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		select {
		case result := <-codeCh:
			if result.Err != nil {
				return authErrorMsg{err: fmt.Errorf("authorization failed: %w", result.Err)}
			}
			pair, err := api.ExchangeCode(
				context.Background(),
				"", // production token endpoint
				result.Code,
				verifier,
				redirectURI,
				clientID,
				store,
			)
			if err != nil {
				return authErrorMsg{err: fmt.Errorf("exchanging authorization code: %w", err)}
			}
			return authSuccessMsg{accessToken: pair.AccessToken}

		case <-ctx.Done():
			return authErrorMsg{err: fmt.Errorf("authorization timed out after 5 minutes")}
		}
	}
}

// renderAuthPanel renders a centered auth prompt box.
func renderAuthPanel(t theme.Theme, width, height int, authURL, status string) string {
	// Truncate URL for display.
	displayURL := authURL
	if len(displayURL) > 60 {
		displayURL = displayURL[:57] + "..."
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.ActiveBorder()).
		Padding(1, 2)

	titleStyle := lipgloss.NewStyle().
		Foreground(t.TextPrimary()).
		Bold(true)

	urlStyle := lipgloss.NewStyle().
		Foreground(t.ActiveBorder())

	statusStyle := lipgloss.NewStyle().
		Foreground(t.TextMuted())

	content := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("Authentication Required"),
		"",
		"Visit this URL to authorize:",
		urlStyle.Render(displayURL),
		"",
		statusStyle.Render(status),
	)

	box := boxStyle.Render(content)

	if width <= 0 || height <= 0 {
		return box
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
