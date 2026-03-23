package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/keychain"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// authPreparedMsg carries PKCE state after the callback server is started.
type authPreparedMsg struct {
	authURL     string
	codeCh      <-chan api.CallbackResult
	verifier    string
	redirectURI string
	serverClose func()
	browserErr  error
}

// authSuccessMsg signals OAuth callback completed and tokens are stored.
type authSuccessMsg struct {
	accessToken string
}

// authErrorMsg signals an auth flow failure.
type authErrorMsg struct {
	err error
}

// prepareAuthCmd starts PKCE flow: generates credentials, starts callback server,
// builds auth URL, opens browser. Does NOT close the server.
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
		// Do NOT defer callbackSrv.Close() — server must stay alive for callback.
		redirectURI := callbackSrv.URL + "/callback"
		authURL := api.BuildAuthURL(clientID, redirectURI, challenge, api.SpotifyScopes)
		browserErr := api.OpenBrowser(authURL)
		return authPreparedMsg{
			authURL: authURL, codeCh: codeCh, verifier: verifier,
			redirectURI: redirectURI, serverClose: callbackSrv.Close,
			browserErr: browserErr,
		}
	}
}

// waitForCallbackCmd blocks on the callback channel, exchanges code, closes server.
func waitForCallbackCmd(clientID string, store keychain.TokenStore,
	verifier, redirectURI string, codeCh <-chan api.CallbackResult, serverClose func()) tea.Cmd {
	return func() tea.Msg {
		defer serverClose()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		select {
		case result := <-codeCh:
			if result.Err != nil {
				return authErrorMsg{err: fmt.Errorf("authorization failed: %w", result.Err)}
			}
			tok, err := api.ExchangeCode(ctx, "", result.Code, verifier, redirectURI, clientID, store)
			if err != nil {
				return authErrorMsg{err: fmt.Errorf("exchanging code: %w", err)}
			}
			return authSuccessMsg{accessToken: tok.AccessToken}
		case <-ctx.Done():
			return authErrorMsg{err: fmt.Errorf("authorization timed out")}
		}
	}
}

// renderAuthPanel renders the auth panel box centered in the terminal.
func renderAuthPanel(t theme.Theme, width, height int, authURL, status string) string {
	titleStyle := lipgloss.NewStyle().Foreground(t.TextPrimary()).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(t.TextMuted())
	urlStyle := lipgloss.NewStyle().Foreground(t.ActiveBorder())

	var lines []string
	lines = append(lines, "")
	lines = append(lines, titleStyle.Render("  Authentication Required"))
	lines = append(lines, "")
	if authURL != "" {
		lines = append(lines, mutedStyle.Render("  If browser didn't open, visit:"))
		displayURL := authURL
		if len(displayURL) > 60 {
			displayURL = displayURL[:57] + "..."
		}
		lines = append(lines, urlStyle.Render("  "+displayURL))
		lines = append(lines, "")
	}
	if status != "" {
		lines = append(lines, mutedStyle.Render("  "+status))
	}
	lines = append(lines, "")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.ActiveBorder()).
		Padding(0, 1).Width(66)
	box := boxStyle.Render(strings.Join(lines, "\n"))
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
