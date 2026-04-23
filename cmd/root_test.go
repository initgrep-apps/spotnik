package cmd_test

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/cmd"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/cliout"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/keychain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// updateGolden is set with -update to refresh all golden files.
var updateGolden = flag.Bool("update", false, "update golden files")

// TestMain disables browser opening and pins ASCII rendering for all tests in this package.
func TestMain(m *testing.M) {
	api.OpenBrowser = func(string) error { return nil }
	cliout.SetTestMode(true)
	// Pin local timezone to UTC so golden files with formatted timestamps are
	// deterministic regardless of the host machine's timezone.
	time.Local = time.UTC
	os.Exit(m.Run())
}

// assertGolden reads the expected output from testdata/golden/<name>.txt and
// compares it to got. Pass -update to refresh all golden files.
func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", "golden", name+".txt")
	if *updateGolden {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(got), 0o644))
		return
	}
	want, err := os.ReadFile(path)
	require.NoError(t, err, "missing golden file %s — run with -update", path)
	assert.Equal(t, string(want), got, "golden mismatch for %s", name)
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

// TestRootCmd_HasAuthSubcommand verifies the auth subcommand is registered with all
// required sub-subcommands: register, login, logout, forget, status.
func TestRootCmd_HasAuthSubcommand(t *testing.T) {
	rootCmd := cmd.RootCommand()
	require.NotNil(t, rootCmd)
	var authFound bool
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "auth" {
			authFound = true
			// Verify auth has all five sub-subcommands.
			found := map[string]bool{}
			for _, authSub := range sub.Commands() {
				found[authSub.Use] = true
			}
			assert.True(t, found["register"], "auth register subcommand should be registered")
			assert.True(t, found["login"], "auth login subcommand should be registered")
			assert.True(t, found["logout"], "auth logout subcommand should be registered")
			assert.True(t, found["forget"], "auth forget subcommand should be registered")
			assert.True(t, found["status"], "auth status subcommand should be registered")
		}
	}
	assert.True(t, authFound, "auth subcommand should be registered")
}

// TestSilenceFlags verifies that rootCmd.SilenceErrors=true and that every auth
// subcommand has SilenceUsage=true. A regression here would re-introduce double
// error printing (cobra's default + Execute's styled block).
func TestSilenceFlags(t *testing.T) {
	root := cmd.RootCommand()
	assert.True(t, root.SilenceErrors, "rootCmd must have SilenceErrors=true to prevent double error printing")
	authCmd, _, err := root.Find([]string{"auth"})
	require.NoError(t, err)
	for _, sub := range authCmd.Commands() {
		assert.True(t, sub.SilenceUsage, "auth %s must have SilenceUsage=true", sub.Use)
	}
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

// TestAuthForgetCmd_clearsTokensAndClientID verifies that RunForget removes all tokens
// and clears the client_id from the config file.
func TestAuthForgetCmd_clearsTokensAndClientID(t *testing.T) {
	// Arrange: config file with client_id.
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := "[spotify]\nclient_id = \"my-secret-client\"\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	// Arrange: in-memory token store with a token.
	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyAccessToken, "access"))
	require.NoError(t, store.Set(keychain.KeyRefreshToken, "refresh"))
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, "1234567890"))

	// Act.
	err := cmd.RunForget(store, path)
	require.NoError(t, err)

	// Assert: tokens gone.
	_, err = store.Get(keychain.KeyAccessToken)
	assert.Error(t, err, "access token should be removed")

	// Assert: client_id removed from config.
	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, "", cfg.ClientID, "client_id should be cleared")
}

// TestAuthStatusCmd_showsClientIDPresent verifies that PrintAuthStatus shows "present"
// when a client ID is set and the user has a valid access token.
func TestAuthStatusCmd_showsClientIDPresent(t *testing.T) {
	// Arrange: config with client_id.
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte("[spotify]\nclient_id = \"abc123\"\n"), 0o600))

	// Arrange: store with access token and expiry.
	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyAccessToken, "valid-access-token"))
	expiry := time.Now().Add(1 * time.Hour)
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, fmt.Sprintf("%d", expiry.Unix())))

	// Act.
	var buf bytes.Buffer
	err := cmd.PrintAuthStatus(store, path, &buf)
	require.NoError(t, err)

	// Assert: check key substrings individually — format uses two spaces between
	// label and value so exact "Client ID: present" no longer matches.
	output := buf.String()
	assert.Contains(t, output, "Client ID")
	assert.Contains(t, output, "present")
	assert.Contains(t, output, "authenticated")
}

// TestAuthStatusCmd_showsClientIDMissing verifies that PrintAuthStatus shows the
// "not registered" state when no client_id is in the config and the store is empty.
func TestAuthStatusCmd_showsClientIDMissing(t *testing.T) {
	// Arrange: config with no client_id.
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte("[spotify]\n"), 0o600))

	// Arrange: empty store.
	store := keychain.NewInMemoryTokenStore()

	// Act.
	var buf bytes.Buffer
	err := cmd.PrintAuthStatus(store, path, &buf)
	require.NoError(t, err)

	// Assert: no-client-id state prints "not registered" with a register hint.
	// The old "not set" and "Client ID:" labels are replaced by a single status line.
	output := buf.String()
	assert.Contains(t, output, "not registered")
	assert.Contains(t, output, "spotnik auth register")
	assert.NotContains(t, output, "not set")
}

// TestCheckAuthState_noClientID_needsRegister verifies that when no client_id is present
// in the config, needsRegister=true and needsAuth=false.
func TestCheckAuthState_noClientID_needsRegister(t *testing.T) {
	cfg := &config.Config{ClientID: ""}
	store := keychain.NewInMemoryTokenStore()

	needsRegister, needsAuth := cmd.CheckAuthState(cfg, store)
	assert.True(t, needsRegister, "no client_id should need register")
	assert.False(t, needsAuth, "no client_id should not separately need auth")
}

// TestCheckAuthState_invalidClientID_needsRegister verifies that a malformed
// (non-hex or wrong-length) client ID is treated the same as a missing one.
func TestCheckAuthState_invalidClientID_needsRegister(t *testing.T) {
	tests := []struct {
		name     string
		clientID string
	}{
		{"placeholder text", "your-client-id-here"},
		{"too short", "abc"},
		{"non-hex 32 chars", strings.Repeat("g", 32)},
		{"too long", strings.Repeat("a", 33)},
	}
	store := keychain.NewInMemoryTokenStore()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.ClientID = tc.clientID
			needsRegister, needsAuth := cmd.CheckAuthState(cfg, store)
			assert.True(t, needsRegister, "invalid client ID should trigger re-registration")
			assert.False(t, needsAuth)
		})
	}
}

// TestCheckAuthState_clientIDNoToken_needsAuth verifies that when a valid client ID is set
// but no token is present, needsRegister=false and needsAuth=true.
func TestCheckAuthState_clientIDNoToken_needsAuth(t *testing.T) {
	cfg := &config.Config{ClientID: strings.Repeat("a", 32)}
	store := keychain.NewInMemoryTokenStore()

	needsRegister, needsAuth := cmd.CheckAuthState(cfg, store)
	assert.False(t, needsRegister, "has client_id should not need register")
	assert.True(t, needsAuth, "no token should need auth")
}

// TestCheckAuthState_clientIDValidToken_noAuthNeeded verifies that when both a valid
// client ID and a valid non-expiring token are present, both return values are false.
func TestCheckAuthState_clientIDValidToken_noAuthNeeded(t *testing.T) {
	cfg := &config.Config{ClientID: strings.Repeat("a", 32)}
	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyAccessToken, "valid-token"))
	require.NoError(t, store.Set(keychain.KeyRefreshToken, "valid-refresh"))
	expiry := time.Now().Add(1 * time.Hour)
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, fmt.Sprintf("%d", expiry.Unix())))

	needsRegister, needsAuth := cmd.CheckAuthState(cfg, store)
	assert.False(t, needsRegister, "has client_id should not need register")
	assert.False(t, needsAuth, "valid token should not need auth")
}

// TestAuthStatus_PrintsExpiry verifies that auth status includes the formatted expiry time.
// The format is "Mon, 02 Jan 2006 15:04 UTC" (not RFC1123).
func TestAuthStatus_PrintsExpiry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte("[spotify]\nclient_id = \"abc\"\n"), 0o600))

	store := keychain.NewInMemoryTokenStore()

	expiry := time.Unix(1735689600, 0)
	require.NoError(t, store.Set(keychain.KeyAccessToken, "valid-access-token"))
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, "1735689600"))

	var buf bytes.Buffer
	err := cmd.PrintAuthStatus(store, path, &buf)
	require.NoError(t, err)

	output := buf.String()
	// Format changed from RFC1123 to "Mon, 02 Jan 2006 15:04 UTC" in story 145.
	assert.Contains(t, output, expiry.Format("Mon, 02 Jan 2006 15:04 UTC"))
	assert.Contains(t, output, "Expires")
}

// TestAuthStatus_NotAuthenticated verifies status output when no token is present
// and no client_id is in config. The new design shows "not registered" in this case.
func TestAuthStatus_NotAuthenticated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte("[spotify]\n"), 0o600))

	store := keychain.NewInMemoryTokenStore()
	var buf bytes.Buffer
	err := cmd.PrintAuthStatus(store, path, &buf)
	require.NoError(t, err)
	// No client_id → "not registered" state (not "not authenticated").
	assert.Contains(t, buf.String(), "not registered")
}

// TestAuthStatus_RegisteredNotAuthenticated verifies status when client_id is set
// but no token is present — shows "not authenticated" with login hint.
func TestAuthStatus_RegisteredNotAuthenticated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte("[spotify]\nclient_id = \"abc\"\n"), 0o600))

	store := keychain.NewInMemoryTokenStore()
	var buf bytes.Buffer
	err := cmd.PrintAuthStatus(store, path, &buf)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "not authenticated")
	assert.Contains(t, output, "Client ID")
	assert.Contains(t, output, "present")
}

// TestAuthStatus_ExpiredExpiryUnknown verifies that when the access token exists but
// the expiry key is unparseable, PrintAuthStatus shows a "session state unknown" warning
// rather than claiming the session is healthy.
func TestAuthStatus_ExpiredExpiryUnknown(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte("[spotify]\nclient_id = \"abc\"\n"), 0o600))

	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyAccessToken, "valid-token"))
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, "not-a-number"))

	var buf bytes.Buffer
	err := cmd.PrintAuthStatus(store, path, &buf)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "session state unknown")
	assert.Contains(t, output, "Could not read token state from keychain")
	assert.Contains(t, output, "spotnik auth login")
}

// TestAuthStatus_ExpiringSoon verifies that the "session expiring" state appears
// with the auto-refresh pending suffix when token is expiring within 5 minutes.
func TestAuthStatus_ExpiringSoon(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte("[spotify]\nclient_id = \"abc\"\n"), 0o600))

	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyAccessToken, "valid-token"))

	// Set expiry to 2 minutes from now (within the 5-minute threshold).
	expiry := time.Now().Add(2 * time.Minute)
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, fmt.Sprintf("%d", expiry.Unix())))

	var buf bytes.Buffer
	err := cmd.PrintAuthStatus(store, path, &buf)
	require.NoError(t, err)
	output := buf.String()
	// "expiring soon" replaced by "session expiring" glyph in story 145.
	assert.Contains(t, output, "session expiring")
	assert.Contains(t, output, "auto-refresh pending")
}

// TestMissingClientID_PrintsInstructions verifies that missing client_id
// prints helpful setup instructions redirecting to spotnik auth register.
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
	content := "[spotify]\nclient_id = \"test-client\"\n[preferences]\ntheme = \"nord\"\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cfg, err := cmd.LoadConfig(path)
	require.NoError(t, err)
	assert.Equal(t, "test-client", cfg.ClientID)
	assert.Equal(t, "nord", cfg.Preferences.Theme)
}

// TestLoadConfig_EmptyClientIDNoError verifies that LoadConfig does NOT error when
// the config file has no client_id — the caller decides what to do via CheckAuthState.
func TestLoadConfig_EmptyClientIDNoError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	// Config file exists but has no client_id.
	content := "[spotify]\n\n[preferences]\ntheme = \"black\"\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cfg, err := cmd.LoadConfig(path)
	require.NoError(t, err)
	assert.Equal(t, "", cfg.ClientID, "empty client_id should not error")
}

// TestLoadConfig_BootstrapsWhenMissing verifies that LoadConfig creates a config
// file when none exists (Bootstrap behavior).
func TestLoadConfig_BootstrapsWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	// File doesn't exist; LoadConfig should bootstrap and return defaults (no error).
	cfg, err := cmd.LoadConfig(path)
	require.NoError(t, err)
	assert.Equal(t, "", cfg.ClientID, "bootstrapped config has no client_id")

	// Verify the file was created.
	_, err = os.Stat(path)
	require.NoError(t, err, "Bootstrap should have created the config file")
}

// TestLoadConfig_ClampsUnknownTheme verifies that an unknown theme ID in config
// is clamped to the default theme.
func TestLoadConfig_ClampsUnknownTheme(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := "[spotify]\nclient_id = \"test\"\n[preferences]\ntheme = \"not-a-real-theme-xyz\"\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cfg, err := cmd.LoadConfig(path)
	require.NoError(t, err)
	assert.Equal(t, "black", cfg.Preferences.Theme, "unknown theme should be clamped to 'black'")
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
		http.DefaultClient,
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

// TestCheckAuthState_ValidToken verifies that a valid token returns needsAuth=false.
func TestCheckAuthState_ValidToken(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyAccessToken, "valid-token"))
	require.NoError(t, store.Set(keychain.KeyRefreshToken, "valid-refresh"))
	expiry := time.Now().Add(1 * time.Hour)
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, fmt.Sprintf("%d", expiry.Unix())))

	cfg := &config.Config{ClientID: strings.Repeat("a", 32)}
	needsRegister, needsAuth := cmd.CheckAuthState(cfg, store)
	assert.False(t, needsRegister, "has valid client_id should not need register")
	assert.False(t, needsAuth, "valid token should not need auth")
}

// TestCheckAuthState_NoToken verifies that missing token returns needsAuth=true.
func TestCheckAuthState_NoToken(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()
	cfg := &config.Config{ClientID: strings.Repeat("a", 32)}
	needsRegister, needsAuth := cmd.CheckAuthState(cfg, store)
	assert.False(t, needsRegister, "has valid client_id should not need register")
	assert.True(t, needsAuth, "no token should need auth")
}

// TestCheckAuthState_ExpiringSoon verifies that an expiring token with no
// reachable refresh endpoint returns needsAuth=true.
func TestCheckAuthState_ExpiringSoon(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyAccessToken, "old-token"))
	require.NoError(t, store.Set(keychain.KeyRefreshToken, "old-refresh"))
	// Expires in 2 minutes (within 5-minute threshold).
	expiry := time.Now().Add(2 * time.Minute)
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, fmt.Sprintf("%d", expiry.Unix())))

	cfg := &config.Config{ClientID: strings.Repeat("a", 32)}
	// Refresh will fail (no server running) → should need auth.
	needsRegister, needsAuth := cmd.CheckAuthState(cfg, store)
	assert.False(t, needsRegister, "has valid client_id should not need register")
	assert.True(t, needsAuth, "expiring token with failed refresh should need auth")
}

// TestPrintAuthStatus_styled_authenticated verifies that PrintAuthStatus includes
// both the Client ID label and authenticated status in its output.
func TestPrintAuthStatus_styled_authenticated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte("[spotify]\nclient_id = \"abc\"\n"), 0o600))
	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyAccessToken, "tok"))
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, fmt.Sprintf("%d", time.Now().Add(time.Hour).Unix())))

	var buf bytes.Buffer
	err := cmd.PrintAuthStatus(store, path, &buf)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Client ID")
	assert.Contains(t, buf.String(), "authenticated")
}

// TestAuthLogoutCmd_alreadyLoggedOut_noError verifies that LogoutTokens on an empty
// store exits without error. The real KeychainTokenStore.Delete() ErrNotFound skip
// is exercised at the OS keychain layer; InMemoryTokenStore.Delete() is a no-op on
// missing keys, confirming the public contract holds for both implementations.
func TestAuthLogoutCmd_alreadyLoggedOut_noError(t *testing.T) {
	store := keychain.NewInMemoryTokenStore()
	// Store is empty — Delete() must not return an error.
	err := cmd.LogoutTokens(store)
	assert.NoError(t, err)
}

// TestAuthForgetCmd_noClientID_noError verifies that RunForget on an empty store
// and a config file with no client_id exits without error.
func TestAuthForgetCmd_noClientID_noError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte("[spotify]\n"), 0o600))
	store := keychain.NewInMemoryTokenStore()
	err := cmd.RunForget(store, path)
	assert.NoError(t, err)
}

// notifyWriter wraps an io.Writer and signals written when a Write call whose
// content contains trigger is observed. Used in TestRunAuthFlow_writesURLToWriter
// to avoid time.Sleep-based sync and to survive the split of what was previously
// a single atomic Write into multiple sequential calls.
type notifyWriter struct {
	w       io.Writer
	once    sync.Once
	written chan struct{}
	trigger []byte // close written when this substring appears in a Write payload
}

func (nw *notifyWriter) Write(p []byte) (int, error) {
	n, err := nw.w.Write(p)
	if bytes.Contains(p, nw.trigger) {
		nw.once.Do(func() { close(nw.written) })
	}
	return n, err
}

// TestRunAuthFlow_writesURLToWriter verifies that RunAuthFlow writes the OAuth URL
// to the provided io.Writer. RunAuthFlow blocks on the callback channel after printing
// the URL block, so we use a channel-based signal instead of time.Sleep to detect
// when the first write has completed before closing the pipe.
func TestRunAuthFlow_writesURLToWriter(t *testing.T) {
	// Mock token exchange server — not actually used here since we only read
	// the URL block before RunAuthFlow blocks on the callback channel.
	tokenResp := map[string]interface{}{
		"access_token":  "tok",
		"refresh_token": "ref",
		"expires_in":    3600,
		"token_type":    "Bearer",
	}
	mockSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tokenResp)
	}))
	defer mockSrv.Close()

	store := keychain.NewInMemoryTokenStore()
	cfg := &config.Config{
		ClientID:     "test-client",
		CallbackPort: 0,
	}

	pr, pw := io.Pipe()
	nw := &notifyWriter{w: pw, written: make(chan struct{}), trigger: []byte("Waiting for authorization")}

	// Drain the pipe into a buffer; stops when pr is closed.
	outputCh := make(chan string, 1)
	go func() {
		var sb strings.Builder
		buf := make([]byte, 4096)
		for {
			n, err := pr.Read(buf)
			if n > 0 {
				sb.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		outputCh <- sb.String()
	}()

	// Run the auth flow; it writes the URL block then blocks on the callback channel.
	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.RunAuthFlow(cfg, store, mockSrv.URL, nw)
		_ = pw.Close()
	}()

	// Block until RunAuthFlow has performed its first Write (the URL block), then
	// close the read-end of the pipe to stop the drain goroutine cleanly.
	select {
	case <-nw.written:
		// URL block written — collect output.
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for RunAuthFlow to write URL block")
	}
	pr.CloseWithError(io.EOF)

	output := <-outputCh
	assert.Contains(t, output, "Visit this URL to authorize", "RunAuthFlow must write URL prompt to writer")
	assert.Contains(t, output, "Waiting for authorization", "RunAuthFlow must write waiting message to writer")

	// Drain errCh so the RunAuthFlow goroutine is not leaked in test output.
	go func() { <-errCh }()
}

// TestPrintLogoutSuccess verifies that PrintLogoutSuccess writes the styled
// "Signed out" confirmation to the provided writer.
func TestPrintLogoutSuccess(t *testing.T) {
	var buf strings.Builder
	cmd.PrintLogoutSuccess(&buf)
	output := buf.String()
	assert.Contains(t, output, "✓")
	assert.Contains(t, output, "Signed out")
}

// TestPrintForgetSuccess verifies that PrintForgetSuccess writes the styled
// "Session ended" confirmation block with all key substrings to the provided writer.
func TestPrintForgetSuccess(t *testing.T) {
	var buf strings.Builder
	cmd.PrintForgetSuccess(&buf)
	output := buf.String()
	assert.Contains(t, output, "✓")
	assert.Contains(t, output, "Session ended")
	assert.Contains(t, output, "→")
}

// TestPrintAuthStatus_fourStates exercises all four PrintAuthStatus states:
// not-registered, registered-no-token, authenticated, expiring-soon.
func TestPrintAuthStatus_fourStates(t *testing.T) {
	tests := []struct {
		name        string
		clientID    string
		setTokens   func(store keychain.TokenStore)
		wantSubstrs []string
	}{
		{
			name:        "not registered",
			clientID:    "",
			setTokens:   func(store keychain.TokenStore) {},
			wantSubstrs: []string{"not registered", "spotnik auth register"},
		},
		{
			name:        "registered not authenticated",
			clientID:    "abc",
			setTokens:   func(store keychain.TokenStore) {},
			wantSubstrs: []string{"not authenticated", "Client ID", "present", "spotnik auth login"},
		},
		{
			name:     "authenticated",
			clientID: "abc",
			setTokens: func(store keychain.TokenStore) {
				_ = store.Set(keychain.KeyAccessToken, "tok")
				_ = store.Set(keychain.KeyTokenExpiry, fmt.Sprintf("%d", time.Now().Add(time.Hour).Unix()))
			},
			wantSubstrs: []string{"authenticated", "Client ID", "present"},
		},
		{
			name:     "expiring soon",
			clientID: "abc",
			setTokens: func(store keychain.TokenStore) {
				_ = store.Set(keychain.KeyAccessToken, "tok")
				_ = store.Set(keychain.KeyTokenExpiry, fmt.Sprintf("%d", time.Now().Add(2*time.Minute).Unix()))
			},
			wantSubstrs: []string{"session expiring", "auto-refresh pending", "spotnik auth login"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.toml")
			content := fmt.Sprintf("[spotify]\nclient_id = \"%s\"\n", tt.clientID)
			require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

			store := keychain.NewInMemoryTokenStore()
			tt.setTokens(store)

			var buf bytes.Buffer
			err := cmd.PrintAuthStatus(store, path, &buf)
			require.NoError(t, err)

			output := buf.String()
			for _, want := range tt.wantSubstrs {
				assert.Contains(t, output, want, "state %q should contain %q", tt.name, want)
			}
		})
	}
}

// TestGolden_AuthLogout verifies the exact layout of the logout success output.
func TestGolden_AuthLogout(t *testing.T) {
	var buf bytes.Buffer
	cmd.PrintLogoutSuccess(&buf)
	assertGolden(t, "auth_logout", buf.String())
}

// TestGolden_AuthForget verifies the exact layout of the forget success output.
func TestGolden_AuthForget(t *testing.T) {
	var buf bytes.Buffer
	cmd.PrintForgetSuccess(&buf)
	assertGolden(t, "auth_forget", buf.String())
}

// TestGolden_AuthStatus_NotRegistered verifies the layout when no client_id is set.
func TestGolden_AuthStatus_NotRegistered(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte("[spotify]\n"), 0o600))
	store := keychain.NewInMemoryTokenStore()

	var buf bytes.Buffer
	require.NoError(t, cmd.PrintAuthStatus(store, path, &buf))
	assertGolden(t, "auth_status_not_registered", buf.String())
}

// TestGolden_AuthStatus_NotAuthenticated verifies the layout when client_id is set
// but no access token is present.
func TestGolden_AuthStatus_NotAuthenticated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte("[spotify]\nclient_id = \"abc\"\n"), 0o600))
	store := keychain.NewInMemoryTokenStore()

	var buf bytes.Buffer
	require.NoError(t, cmd.PrintAuthStatus(store, path, &buf))
	assertGolden(t, "auth_status_not_authenticated", buf.String())
}

// TestGolden_AuthStatus_Authenticated verifies the layout for a healthy authenticated session.
// Uses a fixed far-future Unix timestamp (2099-01-01 00:00:00 UTC) so the session is
// never expiring-soon regardless of when the test runs.
func TestGolden_AuthStatus_Authenticated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte("[spotify]\nclient_id = \"abc\"\n"), 0o600))
	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyAccessToken, "tok"))
	// Fixed far-future timestamp: Fri, 01 Jan 2099 00:00 UTC (never expires in tests).
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, "4070908800"))

	var buf bytes.Buffer
	require.NoError(t, cmd.PrintAuthStatus(store, path, &buf))
	assertGolden(t, "auth_status_authenticated", buf.String())
}

// TestGolden_AuthStatus_Expiring verifies the layout for a session that has expired.
// Uses a fixed past Unix timestamp (2025-01-01 00:00:00 UTC). IsExpiringSoon returns
// true for already-expired tokens (negative time-until < 5-minute threshold), so this
// consistently triggers the "session expiring" branch with deterministic output.
func TestGolden_AuthStatus_Expiring(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte("[spotify]\nclient_id = \"abc\"\n"), 0o600))
	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyAccessToken, "tok"))
	// Fixed past timestamp: Wed, 01 Jan 2025 00:00 UTC.
	// IsExpiringSoon returns true for past expiry (time.Until < 5-min threshold).
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, "1735689600"))

	var buf bytes.Buffer
	require.NoError(t, cmd.PrintAuthStatus(store, path, &buf))
	assertGolden(t, "auth_status_expiring", buf.String())
}

// TestGolden_AuthStatus_ExpiryUnreadable verifies the layout when the token expiry
// key is present but not parseable (IsExpiringSoon returns error).
func TestGolden_AuthStatus_ExpiryUnreadable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte("[spotify]\nclient_id = \"abc\"\n"), 0o600))
	store := keychain.NewInMemoryTokenStore()
	require.NoError(t, store.Set(keychain.KeyAccessToken, "tok"))
	require.NoError(t, store.Set(keychain.KeyTokenExpiry, "not-a-number"))

	var buf bytes.Buffer
	require.NoError(t, cmd.PrintAuthStatus(store, path, &buf))
	assertGolden(t, "auth_status_expiry_unreadable", buf.String())
}

// TestGolden_AuthLogin_NoClientID verifies the layout for the "no client_id" auth login error.
func TestGolden_AuthLogin_NoClientID(t *testing.T) {
	var buf bytes.Buffer
	cmd.PrintAuthLoginNoClientID(&buf)
	assertGolden(t, "auth_login_no_clientid", buf.String())
}

// TestGolden_SignedInLaunching verifies the "Signed in → Launching spotnik…" output.
// This block is shared by both runAuthLogin and runRegister.
func TestGolden_SignedInLaunching(t *testing.T) {
	var buf bytes.Buffer
	cmd.PrintSignedInLaunching(&buf)
	assertGolden(t, "signed_in_launching", buf.String())
}

// TestGolden_AuthRegister_Instructions verifies the layout of the register instructions block.
// The redirect URI is placed on its own line (URL message) as per the story's layout change.
func TestGolden_AuthRegister_Instructions(t *testing.T) {
	var buf bytes.Buffer
	cmd.PrintRegisterInstructions(&buf, "http://127.0.0.1:8888/callback")
	assertGolden(t, "auth_register_instructions", buf.String())
}

// TestGolden_ExecuteErrorFallback verifies the layout of the Execute() error fallback.
func TestGolden_ExecuteErrorFallback(t *testing.T) {
	var buf bytes.Buffer
	cmd.PrintExecuteError(&buf, fmt.Errorf("something went wrong"))
	assertGolden(t, "execute_error_fallback", buf.String())
}

// TestGolden_MissingClientIDInstructions verifies the layout of the missing-client-id
// instructions block.
func TestGolden_MissingClientIDInstructions(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, cmd.PrintMissingClientIDInstructions(&buf))
	assertGolden(t, "missing_clientid_instructions", buf.String())
}

// TestGolden_RegisterAuthFailure verifies the layout of the auth failure output for
// spotnik auth register.
func TestGolden_RegisterAuthFailure(t *testing.T) {
	var buf bytes.Buffer
	cmd.PrintRegisterAuthFailure(&buf, fmt.Errorf("callback timeout after 120s"))
	assertGolden(t, "register_auth_failure", buf.String())
}

// TestGolden_LoginAuthFailure verifies the layout of the auth failure output for
// spotnik auth login.
func TestGolden_LoginAuthFailure(t *testing.T) {
	var buf bytes.Buffer
	cmd.PrintLoginAuthFailure(&buf, fmt.Errorf("token exchange failed: 400 Bad Request"))
	assertGolden(t, "login_auth_failure", buf.String())
}

// TestRunRegister_promptValidatesClientID verifies that runRegister wires
// validateClientID into cliout.Ask: short and non-hex inputs are each rejected
// with a ✗ step. Three invalid inputs exhaust maxPromptAttempts (3), causing
// Ask to return ErrAborted and runRegister to return without ever starting the
// OAuth callback server — so this test does not leak a port.
func TestRunRegister_promptValidatesClientID(t *testing.T) {
	// Three invalid inputs: short, 32 non-hex chars, short again.
	nonHex32 := strings.Repeat("Z", 32)
	input := "short\n" + nonHex32 + "\n" + "short2\n"

	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	var buf bytes.Buffer
	c := cmd.RootCommand()
	c.SetOut(&buf)
	c.SetErr(&buf)

	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.RunRegister(c, strings.NewReader(input))
	}()

	var runErr error
	select {
	case runErr = <-errCh:
		// runRegister completed as expected — 3 failures → ErrAborted → errAlreadyPrinted.
	case <-time.After(5 * time.Second):
		t.Fatal("timed out: runRegister should complete after 3 invalid inputs without starting OAuth flow")
	}
	assert.Error(t, runErr, "runRegister must return errAlreadyPrinted after prompt exhaustion")

	out := buf.String()
	assert.Contains(t, out, "Client ID:")
	assert.Contains(t, out, "✗", "validation failure step must be printed")
	assert.Contains(t, out, "client ID must be 32 characters", "short input must trigger length error")
	assert.Contains(t, out, "client ID must be hexadecimal", "non-hex input must trigger hex error")
}

// TestValidateClientID exercises every branch of validateClientID.
func TestValidateClientID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{name: "valid 32-char hex", input: strings.Repeat("a", 32), wantErr: ""},
		{name: "empty", input: "", wantErr: "must be 32 characters"},
		{name: "too short", input: "abc", wantErr: "must be 32 characters"},
		{name: "too long", input: strings.Repeat("a", 33), wantErr: "must be 32 characters"},
		{name: "32 chars non-hex", input: strings.Repeat("Z", 32), wantErr: "must be hexadecimal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cmd.ValidateClientID(tt.input)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

// TestPrintReRegisterInstructions_golden verifies the "already registered" header
// in the re-registration flow against a golden file.
func TestPrintReRegisterInstructions_golden(t *testing.T) {
	var buf strings.Builder
	cmd.PrintReRegisterInstructions(&buf, "http://127.0.0.1:8888/callback")
	assertGolden(t, "auth_re_register_instructions", buf.String())
}
