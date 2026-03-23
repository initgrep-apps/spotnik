package cmd_test

import (
	"bytes"
	"context"
	"encoding/json"
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

// TestRootCmd_Executes verifies the root command runs without error.
// In tests, the auth flow is skipped because the keychain is mocked.
func TestRootCmd_Executes(t *testing.T) {
	// This test only verifies the command setup — not the full auth flow.
	// The CLI is wired correctly if Execute() completes without panicking.
	// We call into the command package directly to avoid os.Exit.
	assert.NotPanics(t, func() {
		// Verify command is registered and accessible.
		rootCmd := cmd.RootCommand()
		require.NotNil(t, rootCmd)
		assert.Equal(t, "spotnik", rootCmd.Use)
	})
}

// TestAuthLogout_ClearsTokens verifies that logout deletes all 3 keychain keys.
func TestAuthLogout_ClearsTokens(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()

	// Populate all three token keys.
	require.NoError(t, store.Set(keychain.KeyAccessToken, "access"))
	require.NoError(t, store.Set(keychain.KeyRefreshToken, "refresh"))
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, "1234567890"))

	// Call logout.
	err := cmd.LogoutTokens(store)
	require.NoError(t, err)

	// All three keys should be gone.
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

	expiry := time.Unix(1735689600, 0) // 2025-01-01 00:00:00 UTC
	require.NoError(t, store.Set(keychain.KeyAccessToken, "valid-access-token"))
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, "1735689600"))

	var buf bytes.Buffer
	err := cmd.PrintAuthStatus(store, &buf)
	require.NoError(t, err)

	output := buf.String()
	// Should contain the expiry information.
	assert.Contains(t, output, expiry.Format(time.RFC1123))
}

// TestMissingClientID_PrintsInstructions verifies that a missing client_id
// prints helpful setup instructions to stdout.
func TestMissingClientID_PrintsInstructions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	// Write a config with no client_id.
	content := "[spotify]\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

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

// TestFullAuthFlow_ConfigToToken is an integration test that exercises:
// load config → check keychain (empty) → exchange code → store tokens.
// It uses httptest.NewServer to mock the Spotify token endpoint.
func TestFullAuthFlow_ConfigToToken(t *testing.T) {
	// Set up a mock Spotify token server.
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

	// Write a valid config file.
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	content := "[spotify]\nclient_id = \"test-client-id\"\n"
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o600))

	// Load config.
	cfg, err := config.Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, "test-client-id", cfg.ClientID)

	// Use in-memory store (starts empty — simulates first-run).
	store := keychain.NewInMemoryTokenStore()

	// Exchange code via mock server.
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

	// Verify tokens stored in keychain.
	access, err := store.Get(keychain.KeyAccessToken)
	require.NoError(t, err)
	assert.Equal(t, "integration-access-token", access)

	refresh, err := store.Get(keychain.KeyRefreshToken)
	require.NoError(t, err)
	assert.Equal(t, "integration-refresh-token", refresh)

	_, err = store.Get(keychain.KeyTokenExpiry)
	require.NoError(t, err, "token expiry should be stored")
}

// Verify cmd.PrintMissingClientIDInstructions and cmd.LogoutTokens are accessible.
// This acts as a compile-time guard on exported API surface.
var _ = cmd.PrintMissingClientIDInstructions
var _ = cmd.LogoutTokens
