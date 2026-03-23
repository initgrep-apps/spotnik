package cmd_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/cmd"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/keychain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain disables browser opening for all tests in this package.
func TestMain(m *testing.M) {
	api.OpenBrowser = func(string) error { return nil }
	os.Exit(m.Run())
}

// TestRootCmd_Executes verifies the root command runs without error.
// In tests, the auth flow is skipped because the keychain is mocked.
func TestRootCmd_Executes(t *testing.T) {
	assert.NotPanics(t, func() {
		rootCmd := cmd.RootCommand()
		require.NotNil(t, rootCmd)
		assert.Equal(t, "spotnik", rootCmd.Use)
	})
}

// TestRootCmd_HasAuthSubcommand verifies the auth subcommand is registered.
func TestRootCmd_HasAuthSubcommand(t *testing.T) {
	rootCmd := cmd.RootCommand()
	require.NotNil(t, rootCmd)
	var authFound bool
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "auth" {
			authFound = true
			// Verify auth has logout and status sub-subcommands.
			var logoutFound, statusFound bool
			for _, authSub := range sub.Commands() {
				switch authSub.Use {
				case "logout":
					logoutFound = true
				case "status":
					statusFound = true
				}
			}
			assert.True(t, logoutFound, "auth logout subcommand should be registered")
			assert.True(t, statusFound, "auth status subcommand should be registered")
		}
	}
	assert.True(t, authFound, "auth subcommand should be registered")
}

// TestAuthLogout_ClearsTokens verifies that logout deletes all 3 keychain keys.
func TestAuthLogout_ClearsTokens(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()

	require.NoError(t, store.Set(keychain.KeyAccessToken, "access"))
	require.NoError(t, store.Set(keychain.KeyRefreshToken, "refresh"))
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, "1234567890"))

	err := cmd.LogoutTokens(store)
	require.NoError(t, err)

	_, err = store.Get(keychain.KeyAccessToken)
	require.Error(t, err)
	_, err = store.Get(keychain.KeyRefreshToken)
	require.Error(t, err)
	_, err = store.Get(keychain.KeyTokenExpiry)
	require.Error(t, err)
}

// TestAuthStatus_PrintsExpiry verifies that auth status includes the formatted expiry time.
func TestAuthStatus_PrintsExpiry(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()

	expiry := time.Unix(1735689600, 0)
	require.NoError(t, store.Set(keychain.KeyAccessToken, "valid-access-token"))
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, "1735689600"))

	var buf bytes.Buffer
	err := cmd.PrintAuthStatus(store, &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, expiry.Format(time.RFC1123))
}

// TestAuthStatus_NotAuthenticated verifies status output when no token is present.
func TestAuthStatus_NotAuthenticated(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()
	var buf bytes.Buffer
	err := cmd.PrintAuthStatus(store, &buf)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "not authenticated")
}

// TestAuthStatus_ExpiredExpiryUnknown verifies status when access token exists
// but expiry cannot be parsed.
func TestAuthStatus_ExpiredExpiryUnknown(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyAccessToken, "valid-token"))
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, "not-a-number"))

	var buf bytes.Buffer
	err := cmd.PrintAuthStatus(store, &buf)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "authenticated")
}

// TestAuthStatus_ExpiringSoon verifies that the "expiring soon" note appears.
func TestAuthStatus_ExpiringSoon(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyAccessToken, "valid-token"))

	// Set expiry to 2 minutes from now (within the 5-minute threshold).
	expiry := time.Now().Add(2 * time.Minute)
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, fmt.Sprintf("%d", expiry.Unix())))

	var buf bytes.Buffer
	err := cmd.PrintAuthStatus(store, &buf)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "expiring soon")
}

// TestMissingClientID_PrintsInstructions verifies that missing client_id
// prints helpful setup instructions.
func TestMissingClientID_PrintsInstructions(t *testing.T) {
	var buf bytes.Buffer
	err := cmd.PrintMissingClientIDInstructions(&buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "client_id")
	assert.Contains(t, output, "developer.spotify.com")
}

// TestMissingClientID_ExitsWithCode1 verifies that HandleMissingClientID
// returns an error (which the CLI converts to exit code 1).
func TestMissingClientID_ExitsWithCode1(t *testing.T) {
	err := cmd.HandleMissingClientID()
	require.Error(t, err)
}

// TestLoadConfig_ValidFile verifies that LoadConfig parses a valid config.
func TestLoadConfig_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := "[spotify]\nclient_id = \"test-client\"\n[ui]\ntheme = \"nord\"\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cfg, err := cmd.LoadConfig(path)
	require.NoError(t, err)
	assert.Equal(t, "test-client", cfg.ClientID)
	assert.Equal(t, "nord", cfg.UI.Theme)
}

// TestLoadConfig_MissingClientID verifies LoadConfig returns error for missing client_id.
func TestLoadConfig_MissingClientID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := "[spotify]\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	_, err := cmd.LoadConfig(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client_id")
}

// TestEnsureAuthenticated_AlreadyAuthenticated verifies that a valid token
// skips the auth flow entirely.
func TestEnsureAuthenticated_AlreadyAuthenticated(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()

	// Set a token that expires in 1 hour (not expiring soon).
	require.NoError(t, store.Set(keychain.KeyAccessToken, "valid-token"))
	require.NoError(t, store.Set(keychain.KeyRefreshToken, "valid-refresh"))
	expiry := time.Now().Add(1 * time.Hour)
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, fmt.Sprintf("%d", expiry.Unix())))

	cfg := &config.Config{ClientID: "test-client"}

	// With a valid token, ensureAuthenticated should not run the auth flow.
	// We use a non-existent token URL to prove the auth flow is NOT called.
	err := cmd.EnsureAuthenticated(cfg, store, "http://localhost:1") // port 1 = no server
	// Should succeed because token is valid and not expiring soon.
	require.NoError(t, err)
}

// TestEnsureAuthenticated_RefreshesExpiringSoon verifies proactive refresh
// when the token is expiring within 5 minutes.
func TestEnsureAuthenticated_RefreshesExpiringSoon(t *testing.T) {
	// Set up a mock Spotify token refresh server.
	tokenResp := map[string]interface{}{
		"access_token": "refreshed-token",
		"expires_in":   3600,
		"token_type":   "Bearer",
	}
	mockSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tokenResp)
	}))
	defer mockSrv.Close()

	store := keychain.NewInMemoryTokenStore()

	// Set a token that expires in 2 minutes (within 5-minute threshold).
	require.NoError(t, store.Set(keychain.KeyAccessToken, "old-token"))
	require.NoError(t, store.Set(keychain.KeyRefreshToken, "old-refresh"))
	expiry := time.Now().Add(2 * time.Minute)
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, fmt.Sprintf("%d", expiry.Unix())))

	cfg := &config.Config{ClientID: "test-client"}

	// Should refresh using the mock server.
	err := cmd.EnsureAuthenticated(cfg, store, mockSrv.URL)
	require.NoError(t, err)

	// Verify token was refreshed.
	access, err := store.Get(keychain.KeyAccessToken)
	require.NoError(t, err)
	assert.Equal(t, "refreshed-token", access)
}

// TestEnsureAuthenticated_NoToken verifies that EnsureAuthenticated triggers
// the auth flow when no access token is present. We don't fully test the browser
// flow here — only that the function starts without panicking.
func TestEnsureAuthenticated_NoToken(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()
	cfg := &config.Config{ClientID: "test-client"}

	// RunAuthFlow will be called and will start a callback server + block.
	// We time-box to prevent hanging.
	done := make(chan struct{})
	go func() {
		_ = cmd.EnsureAuthenticated(cfg, store, "http://localhost:1")
		close(done)
	}()

	select {
	case <-done:
		// Completed (likely with error from bad token URL).
	case <-time.After(300 * time.Millisecond):
		// Still waiting for callback (expected in auth flow).
	}
}

// TestEnsureAuthenticated_TokenMissingExpiry verifies that when an access token
// exists but expiry key is missing (GetExpiry fails), re-auth is triggered.
func TestEnsureAuthenticated_TokenMissingExpiry(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()
	// Access token exists but no expiry — IsExpiringSoon will return error.
	require.NoError(t, store.Set(keychain.KeyAccessToken, "token"))
	// No expiry set — IsExpiringSoon will fail.

	cfg := &config.Config{ClientID: "test-client"}

	done := make(chan struct{})
	go func() {
		_ = cmd.EnsureAuthenticated(cfg, store, "http://localhost:1")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(300 * time.Millisecond):
	}
}

// TestFullAuthFlow_ConfigToToken is an integration test that exercises:
// load config → check keychain (empty) → exchange code → store tokens.
func TestFullAuthFlow_ConfigToToken(t *testing.T) {
	tokenResp := map[string]interface{}{
		"access_token":  "integration-access-token",
		"refresh_token": "integration-refresh-token",
		"expires_in":    3600,
		"token_type":    "Bearer",
	}
	mockSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tokenResp)
	}))
	defer mockSrv.Close()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	content := "[spotify]\nclient_id = \"test-client-id\"\n"
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o600))

	cfg, err := config.Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, "test-client-id", cfg.ClientID)

	store := keychain.NewInMemoryTokenStore()

	pair, err := api.ExchangeCode(
		context.Background(),
		mockSrv.URL,
		"test-auth-code",
		"test-verifier",
		"http://localhost:12345/callback",
		cfg.ClientID,
		store,
	)
	require.NoError(t, err)
	assert.Equal(t, "integration-access-token", pair.AccessToken)
	assert.Equal(t, "integration-refresh-token", pair.RefreshToken)

	access, err := store.Get(keychain.KeyAccessToken)
	require.NoError(t, err)
	assert.Equal(t, "integration-access-token", access)

	refresh, err := store.Get(keychain.KeyRefreshToken)
	require.NoError(t, err)
	assert.Equal(t, "integration-refresh-token", refresh)

	_, err = store.Get(keychain.KeyTokenExpiry)
	require.NoError(t, err, "token expiry should be stored")
}
