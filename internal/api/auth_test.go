package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/keychain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Task 1.3: PKCE flow tests ---

// TestGenerateCodeVerifier_Length verifies the verifier is exactly 128 characters.
func TestGenerateCodeVerifier_Length(t *testing.T) {
	verifier, err := api.GenerateCodeVerifier()
	require.NoError(t, err)
	assert.Equal(t, 128, len(verifier))
}

// TestGenerateCodeVerifier_Base64URLSafe verifies only URL-safe base64 characters are used.
func TestGenerateCodeVerifier_Base64URLSafe(t *testing.T) {
	verifier, err := api.GenerateCodeVerifier()
	require.NoError(t, err)
	matched, err := regexp.MatchString(`^[A-Za-z0-9_\-]+$`, verifier)
	require.NoError(t, err)
	assert.True(t, matched, "verifier should only contain base64url-safe characters, got: %q", verifier)
}

// TestGenerateCodeVerifier_Unique verifies that two calls produce different values.
func TestGenerateCodeVerifier_Unique(t *testing.T) {
	v1, err := api.GenerateCodeVerifier()
	require.NoError(t, err)
	v2, err := api.GenerateCodeVerifier()
	require.NoError(t, err)
	assert.NotEqual(t, v1, v2)
}

// TestComputeCodeChallenge_KnownVector verifies the challenge against a known input/output.
// The expected value is SHA256 of the verifier, base64url-encoded, no padding.
// Computed: base64url(SHA256("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"))
func TestComputeCodeChallenge_KnownVector(t *testing.T) {
	// Test vector derived from the verifier string.
	// SHA256 of this verifier, base64url-encoded without padding.
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	expected := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	challenge := api.ComputeCodeChallenge(verifier)
	assert.Equal(t, expected, challenge)
}

// TestBuildAuthURL_ContainsAllParams verifies the authorization URL contains
// all required OAuth parameters.
func TestBuildAuthURL_ContainsAllParams(t *testing.T) {
	url := api.BuildAuthURL(
		"test-client-id",
		"http://localhost:8080/callback",
		"test-challenge",
		"user-read-playback-state user-modify-playback-state",
	)

	assert.Contains(t, url, "client_id=test-client-id")
	assert.Contains(t, url, "response_type=code")
	assert.Contains(t, url, "redirect_uri=")
	assert.Contains(t, url, "localhost%3A8080")
	assert.Contains(t, url, "code_challenge=test-challenge")
	assert.Contains(t, url, "code_challenge_method=S256")
	assert.Contains(t, url, "scope=")
}

// --- Task 1.4: Local callback server tests ---

// TestCallbackServer_ExtractsCode verifies that a GET to /callback?code=abc
// sends the code on the returned channel.
func TestCallbackServer_ExtractsCode(t *testing.T) {
	srv, codeCh, err := api.StartCallbackServer()
	require.NoError(t, err)
	defer srv.Close()

	// Send the callback request.
	resp, err := http.Get(fmt.Sprintf("%s/callback?code=myauthcode", srv.URL))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Expect code on channel.
	select {
	case result := <-codeCh:
		require.NoError(t, result.Err)
		assert.Equal(t, "myauthcode", result.Code)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for code on channel")
	}
}

// TestCallbackServer_HandlesError verifies that error=access_denied sends an error on the channel.
func TestCallbackServer_HandlesError(t *testing.T) {
	srv, codeCh, err := api.StartCallbackServer()
	require.NoError(t, err)
	defer srv.Close()

	resp, err := http.Get(fmt.Sprintf("%s/callback?error=access_denied", srv.URL))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	select {
	case result := <-codeCh:
		require.Error(t, result.Err)
		assert.Contains(t, result.Err.Error(), "access_denied")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for error on channel")
	}
}

// TestCallbackServer_RandomPort verifies the server binds to a port > 0.
func TestCallbackServer_RandomPort(t *testing.T) {
	srv, _, err := api.StartCallbackServer()
	require.NoError(t, err)
	defer srv.Close()
	assert.NotEmpty(t, srv.URL)
	assert.NotContains(t, srv.URL, ":0")
}

// --- Task 1.5: Token exchange tests ---

// TestExchangeCode_Success verifies a successful token exchange stores tokens correctly.
func TestExchangeCode_Success(t *testing.T) {
	tokenResp := map[string]interface{}{
		"access_token":  "new-access-token",
		"refresh_token": "new-refresh-token",
		"expires_in":    3600,
		"token_type":    "Bearer",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/token", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		body, _ := io.ReadAll(r.Body)
		assert.Contains(t, string(body), "grant_type=authorization_code")
		assert.Contains(t, string(body), "code=mycode")

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tokenResp)
	}))
	defer srv.Close()

	store := keychain.NewInMemoryTokenStore()
	pair, err := api.ExchangeCode(
		context.Background(),
		http.DefaultClient
		srv.URL,
		"mycode",
		"myverifier",
		"http://localhost:9999/callback",
		"test-client-id",
		store,
	)
	require.NoError(t, err)
	assert.Equal(t, "new-access-token", pair.AccessToken)
	assert.Equal(t, "new-refresh-token", pair.RefreshToken)

	// Verify tokens stored in keychain.
	access, err := store.Get(keychain.KeyAccessToken)
	require.NoError(t, err)
	assert.Equal(t, "new-access-token", access)

	refresh, err := store.Get(keychain.KeyRefreshToken)
	require.NoError(t, err)
	assert.Equal(t, "new-refresh-token", refresh)
}

// TestExchangeCode_ServerError verifies a 500 response returns a descriptive error.
func TestExchangeCode_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	store := keychain.NewInMemoryTokenStore()
	_, err := api.ExchangeCode(
		context.Background(),
		http.DefaultClient
		srv.URL,
		"code", "verifier", "http://localhost/callback", "client-id",
		store,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

// TestExchangeCode_InvalidJSON verifies that garbage JSON returns a parse error.
func TestExchangeCode_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-json{{{"))
	}))
	defer srv.Close()

	store := keychain.NewInMemoryTokenStore()
	_, err := api.ExchangeCode(
		context.Background(),
		http.DefaultClient
		srv.URL,
		"code", "verifier", "http://localhost/callback", "client-id",
		store,
	)
	require.Error(t, err)
}

// TestExchangeCode_MissingFields verifies that a partial JSON response returns an error.
func TestExchangeCode_MissingFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Missing refresh_token and expires_in.
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "only-access"})
	}))
	defer srv.Close()

	store := keychain.NewInMemoryTokenStore()
	_, err := api.ExchangeCode(
		context.Background(),
		http.DefaultClient
		srv.URL,
		"code", "verifier", "http://localhost/callback", "client-id",
		store,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}

// --- Task 1.6: Token refresh tests ---

// TestRefresh_Success verifies that new tokens are stored in the keychain.
func TestRefresh_Success(t *testing.T) {
	tokenResp := map[string]interface{}{
		"access_token": "refreshed-access-token",
		"expires_in":   3600,
		"token_type":   "Bearer",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/token", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		assert.Contains(t, string(body), "grant_type=refresh_token")
		assert.Contains(t, string(body), "refresh_token=old-refresh")

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tokenResp)
	}))
	defer srv.Close()

	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyRefreshToken, "old-refresh"))

	err := api.Refresh(context.Background(), http.DefaultClient, srv.URL, "old-refresh", "test-client-id", store)
	require.NoError(t, err)

	access, err := store.Get(keychain.KeyAccessToken)
	require.NoError(t, err)
	assert.Equal(t, "refreshed-access-token", access)
}

// TestRefresh_InvalidGrant verifies that a 400 response triggers re-auth by
// returning an ErrInvalidGrant error.
func TestRefresh_InvalidGrant(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
	}))
	defer srv.Close()

	store := keychain.NewInMemoryTokenStore()
	err := api.Refresh(context.Background(), http.DefaultClient, srv.URL, "bad-refresh", "test-client-id", store)
	require.Error(t, err)
	assert.ErrorIs(t, err, api.ErrInvalidGrant)
}

// TestRefresh_NetworkError verifies a network error returns a wrapped error.
func TestRefresh_NetworkError(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()
	// Use a non-existent server address.
	err := api.Refresh(context.Background(), http.DefaultClient, "http://localhost:1", "refresh", "client-id", store)
	require.Error(t, err)
	// Should wrap the network error with context.
	assert.True(t, strings.Contains(err.Error(), "refreshing token") || strings.Contains(err.Error(), "connection refused"),
		"expected wrapped network error, got: %v", err)
}

// TestExchangeCode_NetworkError verifies a network error in ExchangeCode returns a wrapped error.
func TestExchangeCode_NetworkError(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()
	_, err := api.ExchangeCode(
		context.Background(),
		http.DefaultClient
		"http://localhost:1",
		"code", "verifier", "http://localhost/callback", "client-id",
		store,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exchanging code")
}

// TestCallbackServer_MissingCode verifies that a callback with neither code nor error
// sends an error on the channel.
func TestCallbackServer_MissingCode(t *testing.T) {
	srv, codeCh, err := api.StartCallbackServer()
	require.NoError(t, err)
	defer srv.Close()

	resp, err := http.Get(fmt.Sprintf("%s/callback", srv.URL))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	select {
	case result := <-codeCh:
		require.Error(t, result.Err)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for error on channel")
	}
}

// TestErrInvalidGrant_IsSentinel verifies ErrInvalidGrant is a well-typed sentinel error.
func TestErrInvalidGrant_IsSentinel(t *testing.T) {
	require.NotNil(t, api.ErrInvalidGrant)
	assert.Contains(t, api.ErrInvalidGrant.Error(), "invalid grant")
}

// TestTokenPair_Fields verifies that a TokenPair struct holds the expected fields.
func TestTokenPair_Fields(t *testing.T) {
	pair := api.TokenPair{
		AccessToken:  "at",
		RefreshToken: "rt",
		ExpiresIn:    3600,
	}
	assert.Equal(t, "at", pair.AccessToken)
	assert.Equal(t, "rt", pair.RefreshToken)
	assert.Equal(t, 3600, pair.ExpiresIn)
}

// TestExchangeCode_ZeroExpiresIn verifies exchange succeeds when expires_in is 0.
// This tests the storeTokens path where no expiry is stored.
func TestExchangeCode_ZeroExpiresIn(t *testing.T) {
	tokenResp := map[string]interface{}{
		"access_token":  "access-only",
		"refresh_token": "refresh-only",
		"expires_in":    0,
		"token_type":    "Bearer",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tokenResp)
	}))
	defer srv.Close()

	store := keychain.NewInMemoryTokenStore()
	pair, err := api.ExchangeCode(
		context.Background(),
		http.DefaultClient
		srv.URL,
		"code", "verifier", "http://localhost/callback", "client-id",
		store,
	)
	require.NoError(t, err)
	assert.Equal(t, "access-only", pair.AccessToken)

	// Access and refresh tokens stored.
	access, err := store.Get(keychain.KeyAccessToken)
	require.NoError(t, err)
	assert.Equal(t, "access-only", access)
}

// TestRefresh_NoRefreshTokenInResponse verifies that when the refresh response
// omits refresh_token (as Spotify sometimes does), only the access token is stored.
func TestRefresh_NoRefreshTokenInResponse(t *testing.T) {
	tokenResp := map[string]interface{}{
		"access_token": "new-access",
		"expires_in":   3600,
		"token_type":   "Bearer",
		// NOTE: no refresh_token — Spotify may omit it on refresh.
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tokenResp)
	}))
	defer srv.Close()

	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyRefreshToken, "old-refresh"))

	err := api.Refresh(context.Background(), http.DefaultClient, srv.URL, "old-refresh", "test-client", store)
	require.NoError(t, err)

	// Access token updated.
	access, err := store.Get(keychain.KeyAccessToken)
	require.NoError(t, err)
	assert.Equal(t, "new-access", access)

	// Old refresh token still intact (no new one provided).
	refresh, err := store.Get(keychain.KeyRefreshToken)
	require.NoError(t, err)
	assert.Equal(t, "old-refresh", refresh)
}

// TestSpotifyScopes_NotEmpty verifies that the scopes constant is defined and non-empty.
func TestSpotifyScopes_NotEmpty(t *testing.T) {
	assert.NotEmpty(t, api.SpotifyScopes)
	assert.Contains(t, api.SpotifyScopes, "user-read-playback-state")
	assert.Contains(t, api.SpotifyScopes, "user-read-recently-played")
}

// TestBuildTokenEndpoint_EmptyUsesProduction verifies that empty baseURL
// returns the production Spotify token endpoint.
func TestBuildTokenEndpoint_EmptyUsesProduction(t *testing.T) {
	endpoint := api.BuildTokenEndpoint("")
	assert.Contains(t, endpoint, "accounts.spotify.com")
	assert.Contains(t, endpoint, "/api/token")
}

// TestBuildTokenEndpoint_CustomBaseURL verifies that a custom base URL
// gets /api/token appended correctly.
func TestBuildTokenEndpoint_CustomBaseURL(t *testing.T) {
	endpoint := api.BuildTokenEndpoint("http://localhost:9090")
	assert.Equal(t, "http://localhost:9090/api/token", endpoint)
}

// TestBuildTokenEndpoint_TrailingSlash verifies trailing slashes are trimmed.
func TestBuildTokenEndpoint_TrailingSlash(t *testing.T) {
	endpoint := api.BuildTokenEndpoint("http://localhost:9090/")
	assert.Equal(t, "http://localhost:9090/api/token", endpoint)
}
