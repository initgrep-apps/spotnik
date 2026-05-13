package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/keychain"
)

// authPreparedMsg is sent after PKCE setup and callback server are ready.
type authPreparedMsg struct {
	authURL     string
	codeCh      <-chan api.CallbackResult
	verifier    string
	redirectURI string
	browserErr  error
}

// onboardingClientIDSavedMsg is sent when client_id has been written to config.toml.
type onboardingClientIDSavedMsg struct {
	clientID string
}

// onboardingRetryMsg is sent when the user presses 'r' on the onboarding error screen.
// Produced by handleOnboardingKey and consumed by the onboardingRetryMsg handler in handlers.go.
type onboardingRetryMsg struct{}

// authSuccessMsg is sent when the OAuth code exchange succeeds.
type authSuccessMsg struct {
	accessToken string
}

// authErrorMsg is sent when the OAuth flow fails.
type authErrorMsg struct {
	err error
}

// saveClientIDCmd writes clientID to the config file at path, then returns
// onboardingClientIDSavedMsg on success or authErrorMsg on failure.
func saveClientIDCmd(path, clientID string) tea.Cmd {
	return func() tea.Msg {
		if err := config.SetClientID(path, clientID); err != nil {
			return authErrorMsg{err: fmt.Errorf("saving client ID: %w", err)}
		}
		return onboardingClientIDSavedMsg{clientID: clientID}
	}
}

// prepareOAuthCmd generates PKCE credentials, builds the Spotify auth URL, and opens
// the browser. The callback server must already be running (started by cmd/ before
// the app is created); this command does NOT start or stop the server.
func prepareOAuthCmd(clientID string, port int, codeCh <-chan api.CallbackResult) tea.Cmd {
	return func() tea.Msg {
		verifier, err := api.GenerateCodeVerifier()
		if err != nil {
			return authErrorMsg{err: fmt.Errorf("generating PKCE verifier: %w", err)}
		}
		challenge := api.ComputeCodeChallenge(verifier)
		redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", port)
		authURL := api.BuildAuthURL(clientID, redirectURI, challenge, api.SpotifyScopes)
		browserErr := api.OpenBrowser(authURL)
		return authPreparedMsg{
			authURL:     authURL,
			codeCh:      codeCh,
			verifier:    verifier,
			redirectURI: redirectURI,
			browserErr:  browserErr,
		}
	}
}

// waitForCallbackCmd blocks on the callback channel and exchanges the code for tokens.
// The caller is responsible for closing the callback server — this function intentionally
// does NOT close it so the server remains alive across retries (e.g. user presses 'r' or 'l').
func waitForCallbackCmd(clientID string, store keychain.TokenStore, verifier, redirectURI string, codeCh <-chan api.CallbackResult) tea.Cmd {
	return func() tea.Msg {
		if codeCh == nil {
			// Programming error: callback server was not started before the auth flow.
			return authErrorMsg{err: fmt.Errorf("internal error: OAuth callback channel is nil — callback server may not have started")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		select {
		case result := <-codeCh:
			if result.Err != nil {
				return authErrorMsg{err: fmt.Errorf("authorization failed: %w", result.Err)}
			}
			pair, err := api.ExchangeCode(
				context.Background(),
				http.DefaultClient,
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

// InitAPIClients constructs and wires all Spotify API clients with the gateway
// and event recorder. Called from cmd/ for pre-authenticated startup.
func (a *App) InitAPIClients(token string) {
	a.initAPIClients(token)
}

// initAPIClients constructs all Spotify API clients with the centralized API
// gateway, then injects them into the app. All request recording is handled by
// the gateway event journal (GatewayEventLog) — no separate logging transport
// is needed. Called after a successful auth flow or token refresh.
func (a *App) initAPIClients(token string) {
	// Wire the store as a GatewayEventRecorder so Gateway.Do() records
	// per-request lifecycle events (allowed/waited/deduped/blocked/completed)
	// into the GatewayEventLog. Both the Request Flow pane and the Network Log
	// pane read from this single authoritative event source.
	a.gateway.SetRecorder(a.store)

	httpClient := &http.Client{Timeout: 30 * time.Second}

	player := api.NewPlayer("", token)
	player.SetHTTPClient(httpClient)
	player.SetGateway(a.gateway)
	a.player = player

	library := api.NewLibraryClient("", token)
	library.SetHTTPClient(httpClient)
	library.SetGateway(a.gateway)
	a.library = library

	search := api.NewSearchClient("", token)
	search.SetHTTPClient(httpClient)
	search.SetGateway(a.gateway)
	a.search = search

	devices := api.NewDevicesClient("", token)
	devices.SetHTTPClient(httpClient)
	devices.SetGateway(a.gateway)
	a.devices = devices

	userAPI := api.NewUserClient("", token)
	userAPI.SetHTTPClient(httpClient)
	userAPI.SetGateway(a.gateway)
	a.userAPI = userAPI

	playlistsAPI := api.NewPlaylistsClient("", token)
	playlistsAPI.SetHTTPClient(httpClient)
	playlistsAPI.SetGateway(a.gateway)
	a.playlistsAPI = playlistsAPI
}
