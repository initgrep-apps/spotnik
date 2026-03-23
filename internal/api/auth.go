// Package api provides the Spotify HTTP client, OAuth authentication flow,
// and token management. It never imports ui/ — data flows via messages and store.
package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/initgrep-apps/spotnik/internal/keychain"
)

// SpotifyScopes defines the OAuth scopes requested at initial authorization.
// All scopes are requested at once so users are not prompted again later.
const SpotifyScopes = "user-read-playback-state user-modify-playback-state " +
	"user-read-currently-playing playlist-read-private playlist-read-collaborative " +
	"playlist-modify-public playlist-modify-private user-library-read " +
	"user-library-modify user-read-private user-read-email " +
	"user-top-read user-follow-read"

// spotifyAuthURL is the Spotify authorization endpoint.
const spotifyAuthURL = "https://accounts.spotify.com/authorize"

// spotifyTokenURL is the Spotify token endpoint.
const spotifyTokenURL = "https://accounts.spotify.com/api/token"

// ErrInvalidGrant is returned by Refresh when Spotify rejects the refresh token
// (HTTP 400), indicating the user must re-authenticate.
var ErrInvalidGrant = errors.New("invalid grant: refresh token rejected, re-authentication required")

// TokenPair holds an access token and optional refresh token returned from
// token exchange or refresh operations.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

// CallbackResult holds the result of a local callback server waiting for the
// OAuth redirect from Spotify.
type CallbackResult struct {
	Code string
	Err  error
}

// tokenResponse is the JSON shape returned by the Spotify token endpoint.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// GenerateCodeVerifier generates a cryptographically random PKCE code verifier.
// It produces 96 random bytes, base64url-encodes them to exactly 128 characters.
// 96 bytes × (4/3) = 128 base64 chars exactly.
// The output contains only [A-Za-z0-9_-].
func GenerateCodeVerifier() (string, error) {
	// NOTE: 96 bytes × (4/3) = 128 base64url chars — exactly the required length.
	b := make([]byte, 96)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating code verifier: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(b)
	return encoded[:128], nil
}

// ComputeCodeChallenge computes the PKCE code challenge for a given verifier.
// It returns the base64url-encoded SHA256 hash of the verifier, without padding.
func ComputeCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// BuildAuthURL constructs the Spotify authorization URL with all required
// PKCE and OAuth parameters.
func BuildAuthURL(clientID, redirectURI, challenge, scopes string) string {
	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("response_type", "code")
	params.Set("redirect_uri", redirectURI)
	params.Set("code_challenge_method", "S256")
	params.Set("code_challenge", challenge)
	params.Set("scope", scopes)
	return spotifyAuthURL + "?" + params.Encode()
}

// callbackServer wraps an httptest-compatible server for the OAuth callback.
// It binds to a random available port and exposes the callback URL.
type callbackServer struct {
	server   *http.Server
	listener net.Listener
	URL      string
}

// Close shuts down the callback server.
func (s *callbackServer) Close() {
	_ = s.server.Shutdown(context.Background())
}

// StartCallbackServer starts a local HTTP server on a random available port
// to receive the OAuth callback from Spotify.
// It returns the server (for URL and Close), and a channel that receives
// the authorization code or error once the callback arrives.
func StartCallbackServer() (*callbackServer, <-chan CallbackResult, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, nil, fmt.Errorf("starting callback server: %w", err)
	}

	resultCh := make(chan CallbackResult, 1)

	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux}

	cs := &callbackServer{
		server:   srv,
		listener: ln,
		URL:      "http://" + ln.Addr().String(),
	}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		errParam := r.URL.Query().Get("error")
		if errParam != "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprintf(w, "Authorization failed: %s", errParam)
			resultCh <- CallbackResult{Err: fmt.Errorf("authorization denied: %s", errParam)}
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprintf(w, "Missing code parameter")
			resultCh <- CallbackResult{Err: fmt.Errorf("callback missing code parameter")}
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "<html><body><h1>Authorization successful!</h1><p>You can close this tab.</p></body></html>")
		resultCh <- CallbackResult{Code: code}
	})

	// Start serving in background.
	go func() {
		_ = srv.Serve(ln)
	}()

	return cs, resultCh, nil
}

// ExchangeCode exchanges an authorization code for access and refresh tokens.
// It POSTs to the Spotify token endpoint and stores the resulting tokens in the store.
// The tokenBaseURL parameter allows overriding for tests (use "" for production).
func ExchangeCode(
	ctx context.Context,
	tokenBaseURL string, // base URL for the token endpoint; production uses spotifyTokenURL
	code, verifier, redirectURI, clientID string,
	store keychain.TokenStore,
) (TokenPair, error) {
	endpoint := buildTokenEndpoint(tokenBaseURL)

	formData := url.Values{}
	formData.Set("grant_type", "authorization_code")
	formData.Set("code", code)
	formData.Set("redirect_uri", redirectURI)
	formData.Set("client_id", clientID)
	formData.Set("code_verifier", verifier)

	pair, err := postTokenRequest(ctx, endpoint, formData)
	if err != nil {
		return TokenPair{}, fmt.Errorf("exchanging code: %w", err)
	}

	// Validate that both tokens are present.
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		return TokenPair{}, fmt.Errorf("missing access_token or refresh_token in response")
	}

	// Store tokens in keychain.
	if err := storeTokens(store, pair); err != nil {
		return TokenPair{}, err
	}

	return pair, nil
}

// Refresh exchanges a refresh token for a new access token via the Spotify token endpoint.
// On success, the new access token is stored in the keychain.
// On HTTP 400 (invalid grant), it returns ErrInvalidGrant so the caller can trigger re-auth.
// The tokenBaseURL parameter allows overriding for tests (use "" for production).
func Refresh(
	ctx context.Context,
	tokenBaseURL string, // base URL for the token endpoint; production uses spotifyTokenURL
	refreshToken, clientID string,
	store keychain.TokenStore,
) error {
	endpoint := buildTokenEndpoint(tokenBaseURL)

	formData := url.Values{}
	formData.Set("grant_type", "refresh_token")
	formData.Set("refresh_token", refreshToken)
	formData.Set("client_id", clientID)

	pair, err := postTokenRequest(ctx, endpoint, formData)
	if err != nil {
		// Check if this is an invalid_grant HTTP 400 — must wrap with ErrInvalidGrant.
		var httpErr *httpStatusError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusBadRequest {
			return fmt.Errorf("%w: %v", ErrInvalidGrant, err)
		}
		return fmt.Errorf("refreshing token: %w", err)
	}

	// Store the new access token (refresh_token may not be returned on refresh).
	if err := store.Set(keychain.KeyAccessToken, pair.AccessToken); err != nil {
		return fmt.Errorf("storing refreshed access token: %w", err)
	}
	if pair.RefreshToken != "" {
		if err := store.Set(keychain.KeyRefreshToken, pair.RefreshToken); err != nil {
			return fmt.Errorf("storing refreshed refresh token: %w", err)
		}
	}
	if pair.ExpiresIn > 0 {
		expiry := time.Now().Add(time.Duration(pair.ExpiresIn) * time.Second)
		if err := store.Set(keychain.KeyTokenExpiry, fmt.Sprintf("%d", expiry.Unix())); err != nil {
			return fmt.Errorf("storing token expiry: %w", err)
		}
	}

	return nil
}

// httpStatusError is returned when the token endpoint returns a non-2xx status code.
type httpStatusError struct {
	StatusCode int
	Body       string
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("token endpoint returned %d: %s", e.StatusCode, e.Body)
}

// BuildTokenEndpoint constructs the token endpoint URL.
// If baseURL is empty, it returns the production Spotify token URL.
// Otherwise it appends /api/token to the base URL (for test mocks).
// Exported for testing.
func BuildTokenEndpoint(baseURL string) string {
	return buildTokenEndpoint(baseURL)
}

// buildTokenEndpoint is the internal implementation of BuildTokenEndpoint.
func buildTokenEndpoint(baseURL string) string {
	if baseURL == "" {
		return spotifyTokenURL
	}
	return strings.TrimRight(baseURL, "/") + "/api/token"
}

// postTokenRequest sends a POST request to the token endpoint and parses the response.
func postTokenRequest(ctx context.Context, endpoint string, formData url.Values) (TokenPair, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint,
		strings.NewReader(formData.Encode()))
	if err != nil {
		return TokenPair{}, fmt.Errorf("creating token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return TokenPair{}, fmt.Errorf("posting to token endpoint: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TokenPair{}, fmt.Errorf("reading token response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return TokenPair{}, &httpStatusError{
			StatusCode: resp.StatusCode,
			Body:       string(body),
		}
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return TokenPair{}, fmt.Errorf("parsing token response: %w", err)
	}

	return TokenPair{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		ExpiresIn:    tr.ExpiresIn,
	}, nil
}

// storeTokens saves the token pair and its expiry in the token store.
func storeTokens(store keychain.TokenStore, pair TokenPair) error {
	if err := store.Set(keychain.KeyAccessToken, pair.AccessToken); err != nil {
		return fmt.Errorf("storing access token: %w", err)
	}
	if pair.RefreshToken != "" {
		if err := store.Set(keychain.KeyRefreshToken, pair.RefreshToken); err != nil {
			return fmt.Errorf("storing refresh token: %w", err)
		}
	}
	if pair.ExpiresIn > 0 {
		expiry := time.Now().Add(time.Duration(pair.ExpiresIn) * time.Second)
		if err := store.Set(keychain.KeyTokenExpiry, fmt.Sprintf("%d", expiry.Unix())); err != nil {
			return fmt.Errorf("storing token expiry: %w", err)
		}
	}
	return nil
}

// OpenBrowser opens the given URL in the default browser.
// On macOS it uses `open`, on Linux `xdg-open`. Failure is not fatal.
func OpenBrowser(urlStr string) error {
	// NOTE: This uses os/exec which is acceptable for the browser-open UX helper.
	// In tests, this function is not called.
	return openBrowserPlatform(urlStr)
}
