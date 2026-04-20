# Onboarding & Auth UX Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the fragmented CLI-only auth flow with a guided TUI onboarding experience: stepwise registration screen, full-URL OAuth wait screen, error/retry, profile overlay logout/forget actions, and parity CLI subcommands.

**Architecture:** A new `viewOnboarding` view mode sits alongside `viewSplash`, `viewAuth`, and `viewGrid`. The OAuth callback server is started before the TUI with a fixed configurable port (default 8888) so the redirect URI is always stable. All screens are pure `View()` renders driven by typed messages — no API calls inside `Update()`.

**Tech Stack:** Go 1.22+, Bubble Tea v0.27+, `github.com/charmbracelet/bubbles/textinput`, `github.com/charmbracelet/bubbles/spinner`, Lip Gloss, cobra, TOML config.

**Spec:** `docs/superpowers/specs/2026-04-20-onboarding-design.md`

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `internal/config/config.go` | Modify | Add `CallbackPort` field, `ClearClientID()` |
| `internal/config/config_test.go` | Modify | Tests for new config behaviour |
| `internal/api/auth.go` | Modify | `StartCallbackServer` accepts explicit port |
| `internal/api/auth_test.go` | Modify | Port-binding tests |
| `cmd/root.go` | Modify | Remove ldflags var; add register/login/forget subcommands; restructure startup |
| `cmd/root_test.go` | Modify | Tests for new CLI commands |
| `internal/app/app.go` | Modify | Add `viewOnboarding`, onboarding fields, update `AppOptions`, `New()` |
| `internal/app/auth.go` | Modify | Rename `prepareAuthCmd` → `prepareOAuthCmd` (server pre-started); add `saveClientIDCmd`, onboarding message types |
| `internal/app/clipboard.go` | Create | `copyToClipboard(text string) error` |
| `internal/app/handlers.go` | Modify | Handlers for onboarding messages; profile logout/forget |
| `internal/app/routing.go` | Modify | Key routing for `viewOnboarding` |
| `internal/app/render.go` | Modify | `renderOnboarding()`, `wrapURL()`, updated `renderAuthPanel()` |
| `internal/ui/panes/profile.go` | Modify | Add logout/forget actions with double-key confirmation |
| `internal/ui/panes/messages.go` | Modify | Add `ProfileLogoutMsg`, `ProfileForgetMsg` |
| `internal/app/auth_test.go` | Modify | Tests for new commands |
| `internal/app/render_test.go` | Modify | Tests for new render helpers |

---

## Task 1: Config — `CallbackPort` field and `ClearClientID()`

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

### Step 1.1 — Write failing tests

Add to `internal/config/config_test.go`:

```go
func TestLoad_CallbackPort_defaultsTo8888(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "config.toml")
    // Write config without callback_port
    err := os.WriteFile(path, []byte("[spotify]\nclient_id = \"abc\"\n"), 0o600)
    require.NoError(t, err)

    cfg, err := Load(path)
    require.NoError(t, err)
    assert.Equal(t, 8888, cfg.CallbackPort)
}

func TestLoad_CallbackPort_fromFile(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "config.toml")
    err := os.WriteFile(path, []byte("[spotify]\nclient_id = \"abc\"\ncallback_port = 9999\n"), 0o600)
    require.NoError(t, err)

    cfg, err := Load(path)
    require.NoError(t, err)
    assert.Equal(t, 9999, cfg.CallbackPort)
}

func TestClearClientID_removesClientIDPreservesPreferences(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "config.toml")
    content := "[spotify]\nclient_id = \"abc123\"\ncallback_port = 9000\n\n[preferences]\ntheme = \"nord\"\n"
    require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

    err := ClearClientID(path)
    require.NoError(t, err)

    cfg, err := Load(path)
    require.NoError(t, err)
    assert.Equal(t, "", cfg.ClientID)
    assert.Equal(t, "nord", cfg.Preferences.Theme)
    assert.Equal(t, 9000, cfg.CallbackPort)
}

func TestClearClientID_noClientID_isNoOp(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "config.toml")
    content := "[spotify]\ncallback_port = 8888\n"
    require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

    err := ClearClientID(path)
    require.NoError(t, err)

    cfg, err := Load(path)
    require.NoError(t, err)
    assert.Equal(t, "", cfg.ClientID)
}

func TestClearClientID_missingFile_returnsError(t *testing.T) {
    err := ClearClientID("/nonexistent/path/config.toml")
    assert.Error(t, err)
}
```

- [ ] Run `go test ./internal/config/... -run "TestLoad_CallbackPort|TestClearClientID" -v`
  Expected: FAIL — `CallbackPort` undefined, `ClearClientID` undefined

### Step 1.2 — Implement `CallbackPort` and `ClearClientID`

In `internal/config/config.go`, update `spotifyConfig`:

```go
// spotifyConfig holds Spotify-specific configuration.
type spotifyConfig struct {
    ClientID     string `toml:"client_id"`
    CallbackPort int    `toml:"callback_port"`
}
```

Add `CallbackPort` to `Config`:

```go
// Config holds all application configuration.
type Config struct {
    // ClientID is the Spotify application client ID.
    ClientID     string
    // CallbackPort is the fixed port for the OAuth callback server.
    // Defaults to 8888 if not set in config.
    CallbackPort int
    Preferences  PreferencesConfig `toml:"preferences"`
}
```

Update `Default()`:

```go
func Default() *Config {
    return &Config{
        CallbackPort: 8888,
        Preferences: PreferencesConfig{
            Theme: "black",
        },
    }
}
```

In `Load()`, after applying parsed fields:

```go
cfg.ClientID = raw.Spotify.ClientID
cfg.Preferences = raw.Preferences
if raw.Spotify.CallbackPort > 0 {
    cfg.CallbackPort = raw.Spotify.CallbackPort
}
// (existing theme/preset/visualizer clamping follows)
```

Add `ClearClientID` function after `Bootstrap`:

```go
// ClearClientID removes the client_id entry from the config file at path.
// All other settings (preferences, callback_port) are preserved.
// Returns an error if the file does not exist or cannot be written.
func ClearClientID(path string) error {
    data, err := os.ReadFile(path)
    if err != nil {
        return fmt.Errorf("reading config for client ID removal: %w", err)
    }

    // Remove lines that set client_id, preserving all other content.
    var out []string
    for _, line := range strings.Split(string(data), "\n") {
        trimmed := strings.TrimSpace(line)
        if strings.HasPrefix(trimmed, "client_id") {
            continue
        }
        out = append(out, line)
    }

    result := strings.Join(out, "\n")
    if err := os.WriteFile(path, []byte(result), 0o600); err != nil {
        return fmt.Errorf("writing config after client ID removal: %w", err)
    }
    return nil
}
```

Add `"strings"` to the imports block in `config.go`.

- [ ] Run `go test ./internal/config/... -run "TestLoad_CallbackPort|TestClearClientID" -v`
  Expected: PASS all five tests

- [ ] Run `make ci`
  Expected: PASS

- [ ] Commit:
```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): add CallbackPort field and ClearClientID function"
```

---

## Task 2: API — `StartCallbackServer` accepts explicit port

**Files:**
- Modify: `internal/api/auth.go`
- Modify: `internal/api/auth_test.go`

### Step 2.1 — Write failing tests

Add to `internal/api/auth_test.go`:

```go
func TestStartCallbackServer_fixedPort(t *testing.T) {
    srv, ch, err := StartCallbackServer(18765)
    require.NoError(t, err)
    defer srv.Close()

    assert.Contains(t, srv.URL, ":18765")
    assert.NotNil(t, ch)
}

func TestStartCallbackServer_portZero_randomPort(t *testing.T) {
    // Port 0 falls back to OS-assigned random port (used in tests).
    srv, ch, err := StartCallbackServer(0)
    require.NoError(t, err)
    defer srv.Close()

    assert.NotEmpty(t, srv.URL)
    assert.NotNil(t, ch)
}

func TestStartCallbackServer_portBusy_returnsError(t *testing.T) {
    // Bind the port first.
    ln, err := net.Listen("tcp", "127.0.0.1:18766")
    require.NoError(t, err)
    defer ln.Close()

    _, _, err = StartCallbackServer(18766)
    assert.Error(t, err)
}
```

- [ ] Run `go test ./internal/api/... -run "TestStartCallbackServer" -v`
  Expected: FAIL — `StartCallbackServer` takes 0 args, not 1

### Step 2.2 — Update `StartCallbackServer` signature

In `internal/api/auth.go`, change the function signature and body:

```go
// StartCallbackServer starts a local HTTP server on the given port to receive
// the OAuth callback from Spotify. Pass port=0 to let the OS assign a random
// port (useful in tests). Returns the server, a result channel, and any error.
func StartCallbackServer(port int) (*callbackServer, <-chan CallbackResult, error) {
    addr := fmt.Sprintf("127.0.0.1:%d", port)
    ln, err := net.Listen("tcp", addr)
    if err != nil {
        return nil, nil, fmt.Errorf("starting callback server on %s: %w", addr, err)
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

    go func() {
        _ = srv.Serve(ln)
    }()

    return cs, resultCh, nil
}
```

Update all callers of `StartCallbackServer` to pass the port:

- `cmd/root.go` `RunAuthFlow`: `api.StartCallbackServer(0)` (random, CLI flow — will be updated in Task 3 to use `cfg.CallbackPort`)
- `internal/app/auth.go` `prepareAuthCmd`: also uses `StartCallbackServer` — will be replaced in Task 5

For now, just add `0` to every existing call site to keep compilation:

In `cmd/root.go`:
```go
callbackSrv, codeCh, err := api.StartCallbackServer(0)
```

In `internal/app/auth.go`:
```go
callbackSrv, codeCh, err := api.StartCallbackServer(0)
```

- [ ] Run `go build ./...`
  Expected: PASS

- [ ] Run `go test ./internal/api/... -run "TestStartCallbackServer" -v`
  Expected: PASS all three tests

- [ ] Run `make ci`
  Expected: PASS

- [ ] Commit:
```bash
git add internal/api/auth.go internal/api/auth_test.go cmd/root.go internal/app/auth.go
git commit -m "feat(api): StartCallbackServer accepts explicit port (0 = random)"
```

---

## Task 3: CLI — subcommand restructure and remove ldflags

**Files:**
- Modify: `cmd/root.go`
- Modify: `cmd/root_test.go`

### Step 3.1 — Write failing tests

Add to `cmd/root_test.go`:

```go
func TestAuthForgetCmd_clearsTokensAndClientID(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "config.toml")
    content := "[spotify]\nclient_id = \"abc123\"\n"
    require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

    store := keychain.NewInMemoryTokenStore()
    _ = store.Set(keychain.KeyAccessToken, "tok")

    err := RunForget(store, path)
    require.NoError(t, err)

    // Tokens gone.
    _, err = store.Get(keychain.KeyAccessToken)
    assert.Error(t, err)

    // Client ID gone.
    cfg, err := config.Load(path)
    require.NoError(t, err)
    assert.Equal(t, "", cfg.ClientID)
}

func TestAuthStatusCmd_showsClientIDPresent(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "config.toml")
    require.NoError(t, os.WriteFile(path, []byte("[spotify]\nclient_id = \"abc\"\n"), 0o600))

    store := keychain.NewInMemoryTokenStore()
    _ = store.Set(keychain.KeyAccessToken, "tok")
    _ = store.Set(keychain.KeyTokenExpiry, fmt.Sprintf("%d", time.Now().Add(time.Hour).Unix()))

    var buf strings.Builder
    err := PrintAuthStatus(store, path, &buf)
    require.NoError(t, err)
    assert.Contains(t, buf.String(), "Client ID: present")
    assert.Contains(t, buf.String(), "Status: authenticated")
}

func TestAuthStatusCmd_showsClientIDMissing(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "config.toml")
    require.NoError(t, os.WriteFile(path, []byte("[spotify]\n"), 0o600))

    store := keychain.NewInMemoryTokenStore()
    var buf strings.Builder
    err := PrintAuthStatus(store, path, &buf)
    require.NoError(t, err)
    assert.Contains(t, buf.String(), "Client ID: not set")
    assert.Contains(t, buf.String(), "Status: not authenticated")
}
```

- [ ] Run `go test ./cmd/... -run "TestAuthForget|TestAuthStatus" -v`
  Expected: FAIL — `RunForget` undefined, `PrintAuthStatus` wrong signature

### Step 3.2 — Restructure `cmd/root.go`

Replace the entire `cmd/root.go` with:

```go
// Package cmd provides the CLI entry point for Spotnik via Cobra.
// It wires configuration, auth flow, theme loading, and application startup.
package cmd

import (
    "context"
    "errors"
    "fmt"
    "io"
    "net/http"
    "os"
    "time"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/initgrep-apps/spotnik/internal/api"
    "github.com/initgrep-apps/spotnik/internal/app"
    "github.com/initgrep-apps/spotnik/internal/config"
    "github.com/initgrep-apps/spotnik/internal/keychain"
    "github.com/initgrep-apps/spotnik/internal/ui/theme"
    "github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
    Use:   "spotnik",
    Short: "A terminal Spotify client for developers",
    Long:  "Spotnik — keyboard-driven Spotify client for developers who live in the terminal.",
    RunE:  runApp,
}

// RootCommand returns the root cobra command. Exported for testing.
func RootCommand() *cobra.Command {
    return rootCmd
}

var appVersion string

// Execute is the entry point called from main.go.
func Execute(version string) {
    appVersion = version
    rootCmd.Version = version
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func init() {
    rootCmd.AddCommand(authCmd)
    authCmd.AddCommand(authRegisterCmd)
    authCmd.AddCommand(authLoginCmd)
    authCmd.AddCommand(authLogoutCmd)
    authCmd.AddCommand(authForgetCmd)
    authCmd.AddCommand(authStatusCmd)
}

// authCmd is the `spotnik auth` group — prints usage when called without a subcommand.
var authCmd = &cobra.Command{
    Use:   "auth",
    Short: "Manage Spotify authentication",
    Long:  "Manage Spotify authentication. Run a subcommand: register, login, logout, forget, status.",
}

// authRegisterCmd is `spotnik auth register` — one-step first-time setup.
var authRegisterCmd = &cobra.Command{
    Use:   "register",
    Short: "Register your Spotify app and authenticate in one step",
    Long: `First-time setup: provide your Spotify Developer Client ID and authenticate.

Steps:
  1. Visit https://developer.spotify.com/dashboard and create an app.
  2. Add your callback URI to the app's Redirect URIs.
  3. Paste your Client ID when prompted.
  4. Approve access in the browser that opens.

Your Client ID is saved to ~/.config/spotnik/config.toml.`,
    RunE: runRegister,
}

// authLoginCmd is `spotnik auth login` — force re-authentication.
var authLoginCmd = &cobra.Command{
    Use:   "login",
    Short: "Force re-authentication with Spotify",
    RunE:  runAuthLogin,
}

// authLogoutCmd is `spotnik auth logout` — clear session tokens only.
var authLogoutCmd = &cobra.Command{
    Use:   "logout",
    Short: "Clear session tokens (keeps Client ID in config)",
    RunE: func(cmd *cobra.Command, args []string) error {
        store := keychain.NewKeychainTokenStore()
        if err := LogoutTokens(store); err != nil {
            return err
        }
        _, _ = fmt.Fprintln(cmd.OutOrStdout(), "Logged out. Session tokens removed.")
        return nil
    },
}

// authForgetCmd is `spotnik auth forget` — clear tokens AND remove client ID.
var authForgetCmd = &cobra.Command{
    Use:   "forget",
    Short: "Remove session tokens and Client ID from config (full reset)",
    RunE: func(cmd *cobra.Command, args []string) error {
        store := keychain.NewKeychainTokenStore()
        if err := RunForget(store, config.DefaultConfigPath()); err != nil {
            return err
        }
        _, _ = fmt.Fprintln(cmd.OutOrStdout(), "Client ID and session forgotten. Run 'spotnik auth register' to set up again.")
        return nil
    },
}

// authStatusCmd is `spotnik auth status` — show current state.
var authStatusCmd = &cobra.Command{
    Use:   "status",
    Short: "Show authentication status",
    RunE: func(cmd *cobra.Command, args []string) error {
        store := keychain.NewKeychainTokenStore()
        return PrintAuthStatus(store, config.DefaultConfigPath(), cmd.OutOrStdout())
    },
}

// LogoutTokens removes all token keys from the token store. Exported for testing.
func LogoutTokens(store keychain.TokenStore) error {
    if err := store.Delete(); err != nil {
        return fmt.Errorf("logging out: %w", err)
    }
    return nil
}

// RunForget clears tokens AND removes client_id from the config file at path.
// Exported for testing.
func RunForget(store keychain.TokenStore, configPath string) error {
    _ = store.Delete() // best-effort: ignore keychain errors
    if err := config.ClearClientID(configPath); err != nil {
        return fmt.Errorf("forget: %w", err)
    }
    return nil
}

// PrintAuthStatus writes the current authentication and registration status to w.
// Exported for testing.
func PrintAuthStatus(store keychain.TokenStore, configPath string, w io.Writer) error {
    cfg, _ := config.Load(configPath) // ignore error — show "not set" if unreadable

    if cfg != nil && cfg.ClientID != "" {
        _, _ = fmt.Fprintln(w, "Client ID: present")
    } else {
        _, _ = fmt.Fprintln(w, "Client ID: not set (run 'spotnik auth register')")
    }

    access, err := store.Get(keychain.KeyAccessToken)
    if err != nil || access == "" {
        _, _ = fmt.Fprintln(w, "Status: not authenticated")
        return nil
    }

    _, _ = fmt.Fprintln(w, "Status: authenticated")

    expiry, err := store.GetExpiry()
    if err == nil {
        _, _ = fmt.Fprintf(w, "Token expiry: %s\n", expiry.Format(time.RFC1123))
    }

    expiringSoon, _ := store.IsExpiringSoon()
    if expiringSoon {
        _, _ = fmt.Fprintln(w, "Note: token is expiring soon and will be refreshed automatically")
    }
    return nil
}

// LoadConfig reads the config file at path and bootstraps it if missing.
// Exported for testing.
func LoadConfig(path string) (*config.Config, error) {
    return loadConfigFromPath(path)
}

func init() {
    config.ThemeValidator = func(id string) bool {
        for _, valid := range theme.Available() {
            if valid == id {
                return true
            }
        }
        return false
    }
}

func loadConfigFromPath(path string) (*config.Config, error) {
    if err := config.Bootstrap(path); err != nil {
        return nil, fmt.Errorf("bootstrapping config: %w", err)
    }
    cfg, err := config.Load(path)
    if err != nil {
        return nil, err
    }
    return cfg, nil
}

// EnsureAuthenticated checks the token state and runs the auth flow if needed.
// Exported for testing.
func EnsureAuthenticated(cfg *config.Config, store keychain.TokenStore, tokenBaseURL string) error {
    access, err := store.Get(keychain.KeyAccessToken)
    if err != nil || access == "" {
        return RunAuthFlow(cfg, store, tokenBaseURL)
    }

    expiringSoon, err := store.IsExpiringSoon()
    if err != nil {
        return RunAuthFlow(cfg, store, tokenBaseURL)
    }

    if expiringSoon {
        refreshToken, err := store.Get(keychain.KeyRefreshToken)
        if err != nil || refreshToken == "" {
            return RunAuthFlow(cfg, store, tokenBaseURL)
        }
        if err := api.Refresh(context.Background(), http.DefaultClient, tokenBaseURL, refreshToken, cfg.ClientID, store); err != nil {
            if errors.Is(err, api.ErrInvalidGrant) {
                fmt.Fprintln(os.Stderr, "Session expired. Please re-authenticate.")
                _ = store.Delete()
                return RunAuthFlow(cfg, store, tokenBaseURL)
            }
            return fmt.Errorf("refreshing token: %w", err)
        }
    }
    return nil
}

// RunAuthFlow executes the full OAuth PKCE authorization flow (CLI mode).
// Uses cfg.CallbackPort for the fixed callback server.
// Exported for testing.
func RunAuthFlow(cfg *config.Config, store keychain.TokenStore, tokenBaseURL string) error {
    verifier, err := api.GenerateCodeVerifier()
    if err != nil {
        return fmt.Errorf("generating PKCE verifier: %w", err)
    }
    challenge := api.ComputeCodeChallenge(verifier)

    callbackSrv, codeCh, err := api.StartCallbackServer(cfg.CallbackPort)
    if err != nil {
        return fmt.Errorf("starting callback server on port %d: %w", cfg.CallbackPort, err)
    }
    defer callbackSrv.Close()

    redirectURI := callbackSrv.URL + "/callback"
    authURL := api.BuildAuthURL(cfg.ClientID, redirectURI, challenge, api.SpotifyScopes)

    fmt.Printf("\nVisit this URL to authorize:\n  %s\n\nWaiting for authorization...\n", authURL)

    if err := api.OpenBrowser(authURL); err != nil {
        _ = err // best-effort
    }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    select {
    case result := <-codeCh:
        if result.Err != nil {
            return fmt.Errorf("authorization failed: %w", result.Err)
        }
        _, err := api.ExchangeCode(
            context.Background(),
            http.DefaultClient,
            tokenBaseURL,
            result.Code,
            verifier,
            redirectURI,
            cfg.ClientID,
            store,
        )
        if err != nil {
            return fmt.Errorf("exchanging authorization code: %w", err)
        }
        fmt.Println("Authorization successful! Starting spotnik...")
        return nil

    case <-ctx.Done():
        return fmt.Errorf("authorization timed out after 5 minutes — please try again")
    }
}

// CheckAuthState returns (needsRegister, needsAuth).
// needsRegister: no client_id in config → must register first.
// needsAuth: has client_id but no valid token → must log in.
// Exported for testing.
func CheckAuthState(cfg *config.Config, store keychain.TokenStore) (needsRegister, needsAuth bool) {
    if cfg.ClientID == "" {
        return true, false
    }

    access, err := store.Get(keychain.KeyAccessToken)
    if err != nil || access == "" {
        return false, true
    }

    expiringSoon, err := store.IsExpiringSoon()
    if err != nil {
        return false, true
    }

    if expiringSoon {
        refreshToken, err := store.Get(keychain.KeyRefreshToken)
        if err != nil || refreshToken == "" {
            return false, true
        }
        if err := api.Refresh(context.Background(), http.DefaultClient, "", refreshToken, cfg.ClientID, store); err != nil {
            _ = store.Delete()
            return false, true
        }
    }

    return false, false
}

// runRegister is the handler for `spotnik auth register`.
// Shows instructions, prompts for client ID, saves it, then runs OAuth.
func runRegister(cmd *cobra.Command, _ []string) error {
    cfg, err := loadConfigFromPath(config.DefaultConfigPath())
    if err != nil {
        return err
    }

    _, _ = fmt.Fprintln(cmd.OutOrStdout(), `
Spotnik — First-time Setup
══════════════════════════

1. Visit https://developer.spotify.com/dashboard
2. Click "Create app" — any name and description will do
3. Under "Redirect URIs" add exactly:

   http://127.0.0.1:`+fmt.Sprintf("%d", cfg.CallbackPort)+`/callback

4. Tick "Web API" under "Which API/SDKs are you planning to use?"
5. Click Save → Settings → copy your Client ID (32-character hex string)

⚠  Spotify Premium is required to use playback controls.
✓  Your Client ID will be saved to ~/.config/spotnik/config.toml
`)

    _, _ = fmt.Fprint(cmd.OutOrStdout(), "Paste your Client ID: ")
    var clientID string
    if _, err := fmt.Fscan(cmd.InOrStdin(), &clientID); err != nil {
        return fmt.Errorf("reading client ID: %w", err)
    }
    clientID = strings.TrimSpace(clientID)
    if clientID == "" {
        return fmt.Errorf("client ID cannot be empty")
    }

    // Save client ID to config.
    if err := config.SetClientID(config.DefaultConfigPath(), clientID); err != nil {
        return fmt.Errorf("saving client ID: %w", err)
    }
    cfg.ClientID = clientID

    _, _ = fmt.Fprintln(cmd.OutOrStdout(), "Client ID saved. Launching authorization...")

    store := keychain.NewKeychainTokenStore()
    return RunAuthFlow(cfg, store, "")
}

// runAuthLogin is the handler for `spotnik auth login` — force re-auth.
func runAuthLogin(_ *cobra.Command, _ []string) error {
    cfg, err := loadConfigFromPath(config.DefaultConfigPath())
    if err != nil {
        return err
    }
    if cfg.ClientID == "" {
        return fmt.Errorf("no Client ID found — run 'spotnik auth register' first")
    }
    store := keychain.NewKeychainTokenStore()
    _ = store.Delete() // force fresh login
    return RunAuthFlow(cfg, store, "")
}

// runApp is the main command handler — loads config, checks auth state, launches TUI.
func runApp(_ *cobra.Command, _ []string) error {
    cfg, err := loadConfigFromPath(config.DefaultConfigPath())
    if err != nil {
        return err
    }

    store := keychain.NewKeychainTokenStore()
    needsRegister, needsAuth := CheckAuthState(cfg, store)

    opts := app.AppOptions{
        NeedsRegister: needsRegister,
        NeedsAuth:     needsAuth,
        ClientID:      cfg.ClientID,
        TokenStore:    store,
        Version:       appVersion,
    }

    // Start callback server early when auth is needed so the port is known
    // before any TUI screen renders.
    if needsRegister || needsAuth {
        srv, codeCh, err := api.StartCallbackServer(cfg.CallbackPort)
        if err != nil {
            return fmt.Errorf("port %d is busy — set a different callback_port in ~/.config/spotnik/config.toml: %w", cfg.CallbackPort, err)
        }
        opts.CallbackPort = cfg.CallbackPort
        opts.CallbackCodeCh = codeCh
        opts.CallbackClose = srv.Close
    }

    a := app.New(cfg, opts)

    if !needsRegister && !needsAuth {
        accessToken, _ := store.Get(keychain.KeyAccessToken)
        a.InitAPIClients(accessToken)
    }

    p := tea.NewProgram(a, tea.WithAltScreen(), tea.WithMouseCellMotion())
    _, err = p.Run()
    return err
}

// PrintMissingClientIDInstructions is kept for backward-compatibility with existing tests.
// Exported for testing.
func PrintMissingClientIDInstructions(w io.Writer) error {
    lines := []string{
        "╭─────────────────────────────────────────────────────╮",
        "│  Spotnik setup required                             │",
        "│                                                     │",
        "│  Run: spotnik auth register                         │",
        "╰─────────────────────────────────────────────────────╯",
    }
    for _, line := range lines {
        if _, err := fmt.Fprintln(w, line); err != nil {
            return err
        }
    }
    return nil
}
```

Add `"strings"` to the imports. Add `config.SetClientID` call — we need to add that function to config too (see Step 3.3).

### Step 3.3 — Add `config.SetClientID`

In `internal/config/config.go`, add:

```go
// SetClientID writes or updates the client_id entry in the config file at path.
// If the [spotify] section exists, it updates or inserts client_id there.
// If the section does not exist, it appends it.
// All other settings are preserved.
func SetClientID(path string, clientID string) error {
    data, err := os.ReadFile(path)
    if err != nil {
        // File may not exist yet — create it.
        if !errors.Is(err, os.ErrNotExist) {
            return fmt.Errorf("reading config: %w", err)
        }
        data = []byte{}
    }

    content := string(data)
    newLine := fmt.Sprintf(`client_id = %q`, clientID)

    // If client_id line already exists, replace it.
    if strings.Contains(content, "client_id") {
        var out []string
        for _, line := range strings.Split(content, "\n") {
            if strings.HasPrefix(strings.TrimSpace(line), "client_id") {
                out = append(out, newLine)
            } else {
                out = append(out, line)
            }
        }
        content = strings.Join(out, "\n")
    } else if strings.Contains(content, "[spotify]") {
        // Insert after [spotify] header.
        var out []string
        for _, line := range strings.Split(content, "\n") {
            out = append(out, line)
            if strings.TrimSpace(line) == "[spotify]" {
                out = append(out, newLine)
            }
        }
        content = strings.Join(out, "\n")
    } else {
        // No [spotify] section — append one.
        content = content + "\n[spotify]\n" + newLine + "\n"
    }

    if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
        return fmt.Errorf("creating config directory: %w", err)
    }
    if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
        return fmt.Errorf("writing config: %w", err)
    }
    return nil
}
```

Add a test in `internal/config/config_test.go`:

```go
func TestSetClientID_writesNewValue(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "config.toml")
    // Start with a config that has [spotify] but no client_id.
    require.NoError(t, os.WriteFile(path, []byte("[spotify]\ncallback_port = 9000\n\n[preferences]\ntheme = \"nord\"\n"), 0o600))

    err := SetClientID(path, "newclientid123")
    require.NoError(t, err)

    cfg, err := Load(path)
    require.NoError(t, err)
    assert.Equal(t, "newclientid123", cfg.ClientID)
    assert.Equal(t, "nord", cfg.Preferences.Theme)
}

func TestSetClientID_replacesExistingValue(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "config.toml")
    require.NoError(t, os.WriteFile(path, []byte("[spotify]\nclient_id = \"old\"\n"), 0o600))

    err := SetClientID(path, "newvalue")
    require.NoError(t, err)

    cfg, err := Load(path)
    require.NoError(t, err)
    assert.Equal(t, "newvalue", cfg.ClientID)
}
```

- [ ] Run `go test ./internal/config/... -v`
  Expected: PASS

- [ ] Run `go test ./cmd/... -run "TestAuthForget|TestAuthStatus" -v`
  Expected: PASS

- [ ] Run `go build ./...`
  Expected: PASS

- [ ] Run `make ci`
  Expected: PASS

- [ ] Commit:
```bash
git add cmd/root.go cmd/root_test.go internal/config/config.go internal/config/config_test.go
git commit -m "feat(cmd): add auth register/login/forget subcommands; remove ldflags client ID"
```

---

## Task 4: App struct — `viewOnboarding`, new fields, updated `AppOptions`

**Files:**
- Modify: `internal/app/app.go`

### Step 4.1 — Add constants and extend `App` and `AppOptions`

In `internal/app/app.go`, update the `viewMode` constants:

```go
const (
    viewSplash    viewMode = iota // Splash screen shown on startup
    viewOnboarding                // First-time registration + OAuth flow
    viewAuth                      // OAuth-only for returning user with no tokens
    viewGrid                      // Grid layout: 10 panes across 2 pages
)
```

Add onboarding step constants (new block, after viewMode block):

```go
// onboardingStep tracks which sub-screen of viewOnboarding is active.
const (
    stepRegister = iota // Step 1: client ID input + instructions
    stepOAuth           // Step 2: browser wait + full URL display
    stepError           // Step 2 error: retry options
)
```

Add imports needed for new fields at the top of `app.go`:

```go
"github.com/charmbracelet/bubbles/spinner"
"github.com/charmbracelet/bubbles/textinput"
```

Add new fields to the `App` struct (after `needsAuth bool`):

```go
// needsRegister is true when no client_id is in config — registration flow required.
needsRegister bool

// Onboarding fields — used only during viewOnboarding.
onboardingStep        int                        // stepRegister | stepOAuth | stepError
onboardingInput       textinput.Model            // client ID text input
onboardingError       string                     // error message on stepError
onboardingPort        int                        // fixed callback port (from AppOptions)
onboardingCodeCh      <-chan api.CallbackResult  // pre-started server code channel
onboardingClose       func()                     // cleanup for callback server
onboardingAuthURL     string                     // full auth URL for clipboard copy
onboardingSpinner     spinner.Model              // waiting indicator on stepOAuth
```

Update `AppOptions`:

```go
// AppOptions carries optional startup configuration into the app.
type AppOptions struct {
    NeedsRegister bool                       // true when no client_id in config
    NeedsAuth     bool                       // true when client_id present but no tokens
    ClientID      string
    TokenStore    keychain.TokenStore
    TokenBaseURL  string
    Version       string
    // CallbackPort is the fixed OAuth callback server port.
    // Zero means auth is not needed (server was not started).
    CallbackPort  int
    // CallbackCodeCh is the channel from the pre-started callback server.
    CallbackCodeCh <-chan api.CallbackResult
    // CallbackClose shuts down the callback server.
    CallbackClose func()
}
```

In `New()`, initialize the new fields:

```go
// Build the textinput for the registration step.
ti := textinput.New()
ti.Placeholder = "your-client-id-here"
ti.CharLimit = 64
ti.Width = 60

// Build the spinner for the OAuth wait step.
sp := spinner.New()
sp.Spinner = spinner.Dot

// Callback server cleanup: default to a no-op if server was not started.
callbackClose := opts.CallbackClose
if callbackClose == nil {
    callbackClose = func() {}
}

a := &App{
    // ... existing fields ...
    needsRegister:    opts.NeedsRegister,
    needsAuth:        opts.NeedsAuth, // keep needsAuth for viewAuth transition
    onboardingInput:  ti,
    onboardingSpinner: sp,
    onboardingPort:   opts.CallbackPort,
    onboardingCodeCh: opts.CallbackCodeCh,
    onboardingClose:  callbackClose,
    // ... rest of existing fields ...
}
```

Also update `Init()` to handle `needsRegister`:

```go
func (a *App) Init() tea.Cmd {
    splashTimer := tea.Tick(5*time.Second, func(_ time.Time) tea.Msg {
        return splashDismissMsg{}
    })
    alertsInitCmd := a.alerts.Init()

    if a.needsRegister || a.needsAuth {
        // Unauthenticated: only show splash, defer everything else.
        return tea.Batch(splashTimer, alertsInitCmd)
    }

    // ... rest of Init() unchanged ...
}
```

- [ ] Run `go build ./...`
  Expected: PASS (may need minor fixes to `New()` call sites in tests — update as needed)

- [ ] Run `make ci`
  Expected: PASS

- [ ] Commit:
```bash
git add internal/app/app.go
git commit -m "feat(app): add viewOnboarding mode, onboarding fields, extended AppOptions"
```

---

## Task 5: Onboarding commands — `prepareOAuthCmd`, `saveClientIDCmd`, clipboard

**Files:**
- Modify: `internal/app/auth.go`
- Create: `internal/app/clipboard.go`
- Modify: `internal/app/auth_test.go`

### Step 5.1 — Write failing tests

Add to `internal/app/auth_test.go`:

```go
func TestSaveClientIDCmd_writesAndEmitsMsg(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "config.toml")
    require.NoError(t, os.WriteFile(path, []byte("[spotify]\n"), 0o600))

    cmd := saveClientIDCmd(path, "testclientid")
    msg := cmd()

    saved, ok := msg.(onboardingClientIDSavedMsg)
    require.True(t, ok)
    assert.Equal(t, "testclientid", saved.clientID)

    // Verify it was written to disk.
    cfg, err := config.Load(path)
    require.NoError(t, err)
    assert.Equal(t, "testclientid", cfg.ClientID)
}

func TestSaveClientIDCmd_writeError_emitsErrorMsg(t *testing.T) {
    cmd := saveClientIDCmd("/nonexistent/path/config.toml", "id")
    msg := cmd()

    _, ok := msg.(authErrorMsg)
    assert.True(t, ok)
}
```

- [ ] Run `go test ./internal/app/... -run "TestSaveClientIDCmd" -v`
  Expected: FAIL — `saveClientIDCmd` undefined, `onboardingClientIDSavedMsg` undefined

### Step 5.2 — Update `internal/app/auth.go`

Replace the existing content with:

```go
package app

import (
    "context"
    "fmt"
    "net/http"
    "time"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/initgrep-apps/spotnik/internal/api"
    "github.com/initgrep-apps/spotnik/internal/config"
    "github.com/initgrep-apps/spotnik/internal/keychain"
    "github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// onboardingClientIDSavedMsg is sent when the client ID has been written to config.
// clientID carries the saved value so App can set cfg.ClientID and proceed to OAuth.
type onboardingClientIDSavedMsg struct {
    clientID string
}

// onboardingRetryMsg is sent when the user presses 'r' on the error screen.
// It resets onboarding back to stepRegister.
type onboardingRetryMsg struct{}

// authPreparedMsg is sent after PKCE setup is complete and the browser is open.
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

// saveClientIDCmd writes clientID to the config file at path and returns
// onboardingClientIDSavedMsg on success, authErrorMsg on failure.
func saveClientIDCmd(path, clientID string) tea.Cmd {
    return func() tea.Msg {
        if err := config.SetClientID(path, clientID); err != nil {
            return authErrorMsg{err: fmt.Errorf("saving client ID: %w", err)}
        }
        return onboardingClientIDSavedMsg{clientID: clientID}
    }
}

// prepareOAuthCmd generates PKCE credentials, builds the auth URL, and opens the browser.
// The callback server is already running — codeCh and serverClose are passed in.
// This replaces the old prepareAuthCmd which also started the server.
func prepareOAuthCmd(clientID string, port int, codeCh <-chan api.CallbackResult, serverClose func()) tea.Cmd {
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
            serverClose: serverClose,
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
                http.DefaultClient,
                "",
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

// InitAPIClients constructs and wires all Spotify API clients. Called from cmd/ for
// pre-authenticated startup.
func (a *App) InitAPIClients(token string) {
    a.initAPIClients(token)
}

// initAPIClients constructs all Spotify API clients with the centralized API gateway.
func (a *App) initAPIClients(token string) {
    a.gateway.SetRecorder(a.store)

    httpClient := &http.Client{}

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

// renderAuthPanel renders the OAuth wait screen for returning users (viewAuth).
// title is "Re-authenticate with Spotify". URL is never truncated.
func renderAuthPanel(t theme.Theme, width, height int, authURL, status string) string {
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

    innerWidth := 100
    if width > 20 {
        innerWidth = width - 20
    }

    wrappedURL := wrapURL(authURL, innerWidth-4)

    urlBox := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(t.TextMuted()).
        Padding(0, 1).
        Render(urlStyle.Render(wrappedURL))

    content := lipgloss.JoinVertical(lipgloss.Left,
        titleStyle.Render("Re-authenticate with Spotify"),
        "",
        "Your session has expired. A browser window has been opened to log you back in.",
        "",
        "On a headless server or browser didn't open? Copy and visit this URL:",
        "",
        urlBox,
        "",
        statusStyle.Render(status),
        "",
        statusStyle.Render("c  copy URL  ·  q  quit"),
    )

    box := boxStyle.Render(content)

    if width <= 0 || height <= 0 {
        return box
    }
    return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
```

### Step 5.3 — Create `internal/app/clipboard.go`

```go
package app

import (
    "fmt"
    "os/exec"
    "strings"
)

// copyToClipboard attempts to copy text to the system clipboard.
// Tries pbcopy (macOS), xclip -selection clipboard (Linux X11), wl-copy (Wayland).
// Returns nil on success; returns an error if all methods fail.
// Callers should treat failure silently — the URL remains visible for manual selection.
func copyToClipboard(text string) error {
    commands := [][]string{
        {"pbcopy"},
        {"xclip", "-selection", "clipboard"},
        {"wl-copy"},
    }
    for _, args := range commands {
        cmd := exec.Command(args[0], args[1:]...)
        cmd.Stdin = strings.NewReader(text)
        if err := cmd.Run(); err == nil {
            return nil
        }
    }
    return fmt.Errorf("no clipboard command available (tried pbcopy, xclip, wl-copy)")
}
```

- [ ] Run `go test ./internal/app/... -run "TestSaveClientIDCmd" -v`
  Expected: PASS

- [ ] Run `go build ./...`
  Expected: PASS

- [ ] Run `make ci`
  Expected: PASS

- [ ] Commit:
```bash
git add internal/app/auth.go internal/app/clipboard.go internal/app/auth_test.go
git commit -m "feat(app): add saveClientIDCmd, prepareOAuthCmd, clipboard copy helper"
```

---

## Task 6: Onboarding message handlers

**Files:**
- Modify: `internal/app/handlers.go`

### Step 6.1 — Update `splashDismissMsg` handler and add onboarding handlers

In `internal/app/handlers.go`, update the `splashDismissMsg` case:

```go
case splashDismissMsg:
    if a.currentView == viewSplash {
        switch {
        case a.needsRegister:
            a.currentView = viewOnboarding
            a.onboardingStep = stepRegister
            a.onboardingInput.Focus()
        case a.needsAuth:
            a.currentView = viewAuth
            a.authStatus = "Opening browser for authorization..."
            return a, prepareOAuthCmd(a.clientID, a.onboardingPort, a.onboardingCodeCh, a.onboardingClose)
        default:
            a.currentView = viewGrid
        }
    }
    return a, nil
```

Add the new onboarding message handlers inside the `switch m := msg.(type)` block:

```go
case onboardingClientIDSavedMsg:
    // Client ID saved — set it on the app and proceed to OAuth.
    a.clientID = m.clientID
    a.onboardingStep = stepOAuth
    a.authStatus = "Opening browser for authorization..."
    return a, tea.Batch(
        prepareOAuthCmd(a.clientID, a.onboardingPort, a.onboardingCodeCh, a.onboardingClose),
        a.onboardingSpinner.Tick,
    )

case onboardingRetryMsg:
    // User pressed 'r' on error screen — go back to client ID input.
    a.onboardingStep = stepRegister
    a.onboardingError = ""
    a.onboardingInput.Reset()
    a.onboardingInput.Focus()
    return a, nil
```

Update the `authPreparedMsg` handler:

```go
case authPreparedMsg:
    a.onboardingAuthURL = m.authURL
    a.authURL = m.authURL
    if m.browserErr != nil {
        a.authStatus = "Browser didn't open. Visit the URL above manually."
    } else {
        a.authStatus = "Waiting for authorization..."
    }
    return a, waitForCallbackCmd(a.clientID, a.tokenStore, m.verifier, m.redirectURI, m.codeCh, m.serverClose)
```

Update the `authErrorMsg` handler:

```go
case authErrorMsg:
    if a.currentView == viewOnboarding {
        // Show retry screen inside onboarding flow.
        a.onboardingStep = stepError
        a.onboardingError = m.err.Error()
        return a, nil
    }
    // viewAuth: show error in status.
    a.authStatus = fmt.Sprintf("Error: %s — press q to quit", m.err.Error())
    return a, nil
```

Add spinner tick handler (in the same switch, before the closing default):

```go
case spinner.TickMsg:
    if a.currentView == viewOnboarding && a.onboardingStep == stepOAuth {
        var cmd tea.Cmd
        a.onboardingSpinner, cmd = a.onboardingSpinner.Update(m)
        return a, cmd
    }
    return a, nil
```

Add imports to `handlers.go`:

```go
"github.com/charmbracelet/bubbles/spinner"
```

- [ ] Run `go build ./...`
  Expected: PASS

- [ ] Run `go test ./internal/app/... -v`
  Expected: PASS (existing tests continue passing)

- [ ] Commit:
```bash
git add internal/app/handlers.go
git commit -m "feat(app): add onboarding message handlers; update auth flow for pre-started server"
```

---

## Task 7: Onboarding key routing

**Files:**
- Modify: `internal/app/routing.go`

### Step 7.1 — Add onboarding key routing

In `handleKeyMsg`, add onboarding routing BEFORE the `viewAuth` guard:

```go
// During onboarding, route keys to the appropriate step handler.
if a.currentView == viewOnboarding {
    return a.handleOnboardingKey(m)
}
```

Add the `handleOnboardingKey` method to `routing.go`:

```go
// handleOnboardingKey routes key events during viewOnboarding.
func (a *App) handleOnboardingKey(m tea.KeyMsg) (tea.Model, tea.Cmd) {
    // q always quits from any onboarding step.
    if m.Type == tea.KeyCtrlC || (m.Type == tea.KeyRunes && string(m.Runes) == "q") {
        return a, tea.Quit
    }

    switch a.onboardingStep {
    case stepRegister:
        // Enter submits the client ID.
        if m.Type == tea.KeyEnter {
            clientID := strings.TrimSpace(a.onboardingInput.Value())
            if clientID == "" {
                return a, nil // ignore empty
            }
            return a, saveClientIDCmd(config.DefaultConfigPath(), clientID)
        }
        // All other keys go to the textinput.
        var cmd tea.Cmd
        a.onboardingInput, cmd = a.onboardingInput.Update(m)
        return a, cmd

    case stepOAuth:
        // 'c' copies the auth URL to clipboard (best-effort, silent on failure).
        if m.Type == tea.KeyRunes && string(m.Runes) == "c" {
            _ = copyToClipboard(a.onboardingAuthURL)
            return a, nil
        }
        return a, nil

    case stepError:
        // 'r' → back to stepRegister.
        if m.Type == tea.KeyRunes && string(m.Runes) == "r" {
            return a, func() tea.Msg { return onboardingRetryMsg{} }
        }
        // 'l' → retry OAuth with current client ID (re-run prepareOAuthCmd).
        if m.Type == tea.KeyRunes && string(m.Runes) == "l" {
            a.onboardingStep = stepOAuth
            a.onboardingError = ""
            return a, tea.Batch(
                prepareOAuthCmd(a.clientID, a.onboardingPort, a.onboardingCodeCh, a.onboardingClose),
                a.onboardingSpinner.Tick,
            )
        }
        return a, nil
    }

    return a, nil
}
```

Add required imports to `routing.go`:

```go
"strings"
"github.com/initgrep-apps/spotnik/internal/config"
```

Also add the `viewAuth` key guard update (handle `c` for clipboard on viewAuth):

```go
// During viewAuth, allow quit keys and URL copy.
if a.currentView == viewAuth {
    if m.Type == tea.KeyCtrlC || (m.Type == tea.KeyRunes && string(m.Runes) == "q") || m.Type == tea.KeyEsc {
        return a, tea.Quit
    }
    if m.Type == tea.KeyRunes && string(m.Runes) == "c" {
        _ = copyToClipboard(a.authURL)
        return a, nil
    }
    return a, nil
}
```

- [ ] Run `go build ./...`
  Expected: PASS

- [ ] Run `go test ./internal/app/... -v`
  Expected: PASS

- [ ] Commit:
```bash
git add internal/app/routing.go
git commit -m "feat(app): add onboarding key routing (register input, OAuth copy, error retry)"
```

---

## Task 8: Onboarding render functions

**Files:**
- Modify: `internal/app/render.go`

### Step 8.1 — Add `wrapURL` and onboarding render helpers

Add to `internal/app/render.go`:

```go
// wrapURL wraps a long URL across multiple lines at the given width.
// It tries to break at '&' query-parameter boundaries when possible.
// This ensures the full URL is always visible — never truncated.
func wrapURL(rawURL string, width int) string {
    if len(rawURL) <= width {
        return rawURL
    }
    var lines []string
    for len(rawURL) > width {
        breakAt := width
        // Prefer breaking just before an '&' in the latter half of the window.
        if idx := strings.LastIndex(rawURL[:width], "&"); idx > width/2 {
            breakAt = idx
        }
        lines = append(lines, rawURL[:breakAt])
        rawURL = rawURL[breakAt:]
    }
    if rawURL != "" {
        lines = append(lines, rawURL)
    }
    return strings.Join(lines, "\n")
}

// onboardingTitle renders the shared spotnik header used on all onboarding screens.
func (a *App) onboardingTitle() string {
    nameStyle := lipgloss.NewStyle().
        Foreground(a.theme.TextPrimary()).
        Bold(true)
    tagStyle := lipgloss.NewStyle().
        Foreground(a.theme.TextMuted())
    return lipgloss.JoinVertical(lipgloss.Center,
        nameStyle.Render("♪  spotnik"),
        tagStyle.Render("A terminal Spotify client for developers"),
    )
}

// renderOnboarding dispatches to the correct step renderer.
func (a *App) renderOnboarding() string {
    title := a.onboardingTitle()
    var content string
    switch a.onboardingStep {
    case stepRegister:
        content = a.renderOnboardingRegister()
    case stepOAuth:
        content = a.renderOnboardingOAuth()
    case stepError:
        content = a.renderOnboardingError()
    default:
        content = a.renderOnboardingRegister()
    }
    body := lipgloss.JoinVertical(lipgloss.Left, title, "", "", content)
    if a.width > 0 && a.height > 0 {
        return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, body)
    }
    return body
}

// renderOnboardingRegister renders Step 1: client ID input + instructions.
func (a *App) renderOnboardingRegister() string {
    th := a.theme
    boxW := a.width - 4
    if boxW < 80 {
        boxW = 80
    }
    innerW := boxW - 6

    labelStyle := lipgloss.NewStyle().Foreground(th.TextPrimary())
    mutedStyle := lipgloss.NewStyle().Foreground(th.TextMuted())
    warnStyle := lipgloss.NewStyle().Foreground(th.Warning())
    successStyle := lipgloss.NewStyle().Foreground(th.Success())
    urlStyle := lipgloss.NewStyle().Foreground(th.ActiveBorder())

    redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", a.onboardingPort)

    redirectBox := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(th.TextMuted()).
        Padding(0, 1).
        Render(urlStyle.Render(redirectURI) + "  " + mutedStyle.Render("← copy and paste this"))

    inputBox := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(th.ActiveBorder()).
        Padding(0, 1).
        Width(innerW).
        Render(a.onboardingInput.View())

    instructions := lipgloss.JoinVertical(lipgloss.Left,
        mutedStyle.Render("Spotnik requires your own Spotify Developer credentials. Spotify does not allow"),
        mutedStyle.Render("shared app credentials, so this is a one-time setup. Takes about 2 minutes."),
        "",
        labelStyle.Render("1.  Open  →  https://developer.spotify.com/dashboard"),
        labelStyle.Render(`2.  Click "Create app" — any name and description will do`),
        labelStyle.Render(`3.  Under "Redirect URIs" paste this URL exactly:`),
        "",
        "    "+redirectBox,
        "",
        labelStyle.Render(`4.  Tick "Web API" under "Which API/SDKs are you planning to use?"`),
        labelStyle.Render("5.  Click Save → open Settings → copy your Client ID (32-character hex string)"),
        "",
        warnStyle.Render("⚠   Spotify Premium is required to use playback controls"),
        successStyle.Render("✓   Your Client ID will be saved to ~/.config/spotnik/config.toml"),
        "",
        mutedStyle.Render("Paste your Client ID below:"),
        inputBox,
    )

    panel := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(th.ActiveBorder()).
        Padding(1, 3).
        Width(boxW).
        Render(
            lipgloss.JoinVertical(lipgloss.Left,
                lipgloss.NewStyle().Foreground(th.TextPrimary()).Bold(true).Render("── Step 1 of 2 — Set up your Spotify Developer App"),
                "",
                instructions,
            ),
        )

    hint := lipgloss.NewStyle().Foreground(th.TextMuted()).Render("Enter  confirm  ·  q  quit")
    return lipgloss.JoinVertical(lipgloss.Left, panel, "", hint)
}

// renderOnboardingOAuth renders Step 2: OAuth wait with full URL.
func (a *App) renderOnboardingOAuth() string {
    th := a.theme
    boxW := a.width - 4
    if boxW < 80 {
        boxW = 80
    }
    innerW := boxW - 10

    mutedStyle := lipgloss.NewStyle().Foreground(th.TextMuted())
    urlStyle := lipgloss.NewStyle().Foreground(th.ActiveBorder())

    wrappedURL := wrapURL(a.onboardingAuthURL, innerW)
    urlBox := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(th.TextMuted()).
        Padding(0, 1).
        Width(innerW+4).
        Render(urlStyle.Render(wrappedURL))

    body := lipgloss.JoinVertical(lipgloss.Left,
        "A browser window has been opened for you. Log in to Spotify and click Agree.",
        "",
        mutedStyle.Render("On a headless server or browser didn't open? Copy and visit this URL:"),
        "",
        urlBox,
        "",
        a.onboardingSpinner.View()+" "+mutedStyle.Render("Waiting for authorization...  (times out in 5 minutes)"),
        "",
        mutedStyle.Render("Once you approve in the browser, Spotnik continues automatically."),
    )

    panel := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(th.ActiveBorder()).
        Padding(1, 3).
        Width(boxW).
        Render(
            lipgloss.JoinVertical(lipgloss.Left,
                lipgloss.NewStyle().Foreground(th.TextPrimary()).Bold(true).Render("── Step 2 of 2 — Authorize Spotnik with Spotify"),
                "",
                body,
            ),
        )

    hint := lipgloss.NewStyle().Foreground(th.TextMuted()).Render("c  copy URL  ·  q  quit")
    return lipgloss.JoinVertical(lipgloss.Left, panel, "", hint)
}

// renderOnboardingError renders the error + retry screen.
func (a *App) renderOnboardingError() string {
    th := a.theme
    boxW := a.width - 4
    if boxW < 80 {
        boxW = 80
    }

    errStyle := lipgloss.NewStyle().Foreground(th.Error())
    mutedStyle := lipgloss.NewStyle().Foreground(th.TextMuted())
    labelStyle := lipgloss.NewStyle().Foreground(th.TextPrimary())

    redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", a.onboardingPort)

    body := lipgloss.JoinVertical(lipgloss.Left,
        errStyle.Render("✗  Authorization failed"),
        "",
        errStyle.Render("Error: "+a.onboardingError),
        "",
        labelStyle.Render("Common causes:"),
        mutedStyle.Render("  •  Client ID was mistyped or truncated"),
        mutedStyle.Render("  •  Redirect URI in your Spotify app does not match:"),
        mutedStyle.Render("     "+redirectURI),
        mutedStyle.Render("  •  The Spotify app was deleted or suspended"),
        "",
        labelStyle.Render("What would you like to do?"),
        "",
        labelStyle.Render("  r  Re-enter Client ID  (go back to Step 1)"),
        labelStyle.Render("  l  Try again           (keep current Client ID, retry OAuth)"),
        labelStyle.Render("  q  Quit"),
    )

    return lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(th.Error()).
        Padding(1, 3).
        Width(boxW).
        Render(
            lipgloss.JoinVertical(lipgloss.Left,
                lipgloss.NewStyle().Foreground(th.TextPrimary()).Bold(true).Render("── Step 2 of 2 — Authorization Failed"),
                "",
                body,
            ),
        )
}
```

Update `buildView()` in `render.go` to dispatch to `renderOnboarding`:

```go
// Onboarding — first-time registration + OAuth.
if a.currentView == viewOnboarding {
    return a.renderOnboarding()
}

// Auth panel for returning user with no tokens.
if a.currentView == viewAuth {
    return renderAuthPanel(a.theme, a.width, a.height, a.authURL, a.authStatus)
}
```

(Add this block BEFORE the existing `viewAuth` check.)

Add `"fmt"` to the imports of `render.go` if not already present.

### Step 8.2 — Write render tests

Add to `internal/app/render_test.go`:

```go
func TestWrapURL_shortURL_unchanged(t *testing.T) {
    url := "https://example.com/short"
    result := wrapURL(url, 80)
    assert.Equal(t, url, result)
}

func TestWrapURL_longURL_breaksAtAmpersand(t *testing.T) {
    url := "https://accounts.spotify.com/authorize?client_id=abc123&response_type=code&redirect_uri=http://127.0.0.1:8888/callback"
    result := wrapURL(url, 60)
    lines := strings.Split(result, "\n")
    assert.Greater(t, len(lines), 1)
    for _, line := range lines {
        assert.LessOrEqual(t, len(line), 60)
    }
}

func TestWrapURL_noAmpersand_breaksAtWidth(t *testing.T) {
    url := strings.Repeat("a", 150)
    result := wrapURL(url, 60)
    lines := strings.Split(result, "\n")
    assert.Equal(t, 3, len(lines)) // 60 + 60 + 30
}
```

- [ ] Run `go test ./internal/app/... -run "TestWrapURL" -v`
  Expected: PASS

- [ ] Run `go build ./...`
  Expected: PASS

- [ ] Run `make ci`
  Expected: PASS

- [ ] Commit:
```bash
git add internal/app/render.go internal/app/render_test.go
git commit -m "feat(app): add renderOnboarding screens (register, OAuth wait, error) and wrapURL"
```

---

## Task 9: Profile overlay — logout and forget with confirmation

**Files:**
- Modify: `internal/ui/panes/profile.go`
- Modify: `internal/ui/panes/messages.go`
- Modify: `internal/app/handlers.go`
- Modify: `internal/ui/panes/profile_test.go`

### Step 9.1 — Add message types

In `internal/ui/panes/messages.go`, add:

```go
// ProfileLogoutMsg is emitted when the user confirms logout from the profile overlay.
// The app clears tokens and quits.
type ProfileLogoutMsg struct{}

// ProfileForgetMsg is emitted when the user confirms forget from the profile overlay.
// The app clears tokens and client_id from config, then quits.
type ProfileForgetMsg struct{}
```

### Step 9.2 — Write failing tests

Add to `internal/ui/panes/profile_test.go`:

```go
func TestProfileOverlay_logoutFirstPress_showsConfirmation(t *testing.T) {
    store := state.New()
    pane := NewProfileOverlay(store, theme.NewBlack())

    updated, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
    model := updated.(*ProfileOverlay)

    assert.Nil(t, cmd)
    assert.Contains(t, model.View(), "Press l again to confirm")
}

func TestProfileOverlay_logoutSecondPress_emitsLogoutMsg(t *testing.T) {
    store := state.New()
    pane := NewProfileOverlay(store, theme.NewBlack())

    // First press.
    updated, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
    // Second press.
    updated, cmd := updated.(*ProfileOverlay).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
    _ = updated

    require.NotNil(t, cmd)
    msg := cmd()
    _, ok := msg.(ProfileLogoutMsg)
    assert.True(t, ok)
}

func TestProfileOverlay_forgetFirstPress_showsConfirmation(t *testing.T) {
    store := state.New()
    pane := NewProfileOverlay(store, theme.NewBlack())

    updated, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
    model := updated.(*ProfileOverlay)

    assert.Contains(t, model.View(), "Press f again to confirm")
}

func TestProfileOverlay_forgetSecondPress_emitsForgetMsg(t *testing.T) {
    store := state.New()
    pane := NewProfileOverlay(store, theme.NewBlack())

    updated, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
    updated, cmd := updated.(*ProfileOverlay).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
    _ = updated

    require.NotNil(t, cmd)
    msg := cmd()
    _, ok := msg.(ProfileForgetMsg)
    assert.True(t, ok)
}

func TestProfileOverlay_differentKeyAfterFirstPress_cancelsConfirmation(t *testing.T) {
    store := state.New()
    pane := NewProfileOverlay(store, theme.NewBlack())

    // Press l then press f — should show forget confirmation, not trigger logout.
    updated, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
    updated, cmd := updated.(*ProfileOverlay).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
    _ = updated
    // cmd is nil because pressing f after l resets to forget confirmation, not emission.
    _ = cmd
    assert.Contains(t, updated.(*ProfileOverlay).View(), "Press f again to confirm")
}
```

- [ ] Run `go test ./internal/ui/panes/... -run "TestProfileOverlay_logout|TestProfileOverlay_forget" -v`
  Expected: FAIL — `pendingAction` undefined

### Step 9.3 — Update `profile.go`

Replace `internal/ui/panes/profile.go` with:

```go
// Package panes — ProfileOverlay is the floating user profile overlay.
// It displays the authenticated user's display name, subscription tier,
// country, and session management actions (logout, forget).
package panes

import (
    "strings"
    "unicode/utf8"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/initgrep-apps/spotnik/internal/state"
    "github.com/initgrep-apps/spotnik/internal/ui/layout"
    "github.com/initgrep-apps/spotnik/internal/ui/theme"
)

const maxProfileNameLen = 20

// profileAction tracks which action is awaiting confirmation.
type profileAction int

const (
    profileActionNone   profileAction = iota
    profileActionLogout               // awaiting second 'l'
    profileActionForget               // awaiting second 'f'
)

// ProfileOverlay renders the authenticated user's profile as a floating overlay.
type ProfileOverlay struct {
    store         state.StateReader
    theme         theme.Theme
    width         int
    height        int
    pendingAction profileAction // tracks confirmation state
}

// NewProfileOverlay constructs a ProfileOverlay wired to the given store and theme.
func NewProfileOverlay(store state.StateReader, t theme.Theme) *ProfileOverlay {
    return &ProfileOverlay{
        store: store,
        theme: t,
    }
}

// Init returns nil — data is already in the store.
func (p *ProfileOverlay) Init() tea.Cmd { return nil }

// Update handles messages for the ProfileOverlay.
// Esc closes the overlay. l/f handle logout/forget with double-key confirmation.
func (p *ProfileOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    key, ok := msg.(tea.KeyMsg)
    if !ok {
        return p, nil
    }

    if key.Type == tea.KeyEsc {
        p.pendingAction = profileActionNone
        return p, func() tea.Msg { return ProfileOverlayClosedMsg{} }
    }

    if key.Type != tea.KeyRunes {
        return p, nil
    }
    ch := string(key.Runes)

    switch ch {
    case "l":
        if p.pendingAction == profileActionLogout {
            // Second press — confirm.
            p.pendingAction = profileActionNone
            return p, func() tea.Msg { return ProfileLogoutMsg{} }
        }
        // First press — arm confirmation (reset any other pending action).
        p.pendingAction = profileActionLogout

    case "f":
        if p.pendingAction == profileActionForget {
            // Second press — confirm.
            p.pendingAction = profileActionNone
            return p, func() tea.Msg { return ProfileForgetMsg{} }
        }
        // First press — arm forget confirmation.
        p.pendingAction = profileActionForget

    default:
        // Any other key cancels pending action.
        p.pendingAction = profileActionNone
    }

    return p, nil
}

// View renders the profile overlay content.
func (p *ProfileOverlay) View() string {
    profile := p.store.UserProfile()
    isPremium := p.store.IsPremium()

    const innerWidth = 38

    var lines []string

    if profile.ID == "" {
        loadingStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())
        lines = append(lines, loadingStyle.Render("Loading profile..."))
    } else {
        nameStyle := lipgloss.NewStyle().
            Foreground(p.theme.TextPrimary()).
            Bold(true)
        name := truncateRunes(profile.DisplayName, maxProfileNameLen)
        lines = append(lines, nameStyle.Render(name))

        sepStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())
        lines = append(lines, sepStyle.Render("────────────────────"))

        if isPremium {
            badgeStyle := lipgloss.NewStyle().Foreground(p.theme.Info())
            lines = append(lines, badgeStyle.Render("♛  Premium"))
        } else {
            badgeStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())
            lines = append(lines, badgeStyle.Render("○  Free"))
        }

        if profile.Country != "" {
            iconStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())
            codeStyle := lipgloss.NewStyle().Foreground(p.theme.TextPrimary())
            lines = append(lines, iconStyle.Render("◎  ")+codeStyle.Render(profile.Country))
        }
    }

    // Separator before actions.
    sepStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())
    lines = append(lines, "")
    lines = append(lines, sepStyle.Render("────────────────────"))

    // Logout action.
    actionStyle := lipgloss.NewStyle().Foreground(p.theme.TextPrimary())
    mutedStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())

    switch p.pendingAction {
    case profileActionLogout:
        warnStyle := lipgloss.NewStyle().Foreground(p.theme.Warning())
        lines = append(lines, warnStyle.Render("!! Press l again to confirm logout"))
        lines = append(lines, mutedStyle.Render("   f  Forget"))
        lines = append(lines, mutedStyle.Render("      removes session + Client ID"))
    case profileActionForget:
        lines = append(lines, mutedStyle.Render("   l  Logout"))
        lines = append(lines, mutedStyle.Render("      ends session · keeps Client ID"))
        warnStyle := lipgloss.NewStyle().Foreground(p.theme.Warning())
        lines = append(lines, warnStyle.Render("!! Press f again to confirm forget"))
    default:
        lines = append(lines, actionStyle.Render("  l  Logout"))
        lines = append(lines, mutedStyle.Render("     ends session · keeps Client ID"))
        lines = append(lines, "")
        lines = append(lines, actionStyle.Render("  f  Forget"))
        lines = append(lines, mutedStyle.Render("     removes session + Client ID"))
    }

    lines = append(lines, "")
    lines = append(lines, mutedStyle.Render("  Esc  close"))

    inner := strings.Join(lines, "\n")
    inner = lipgloss.NewStyle().
        Width(innerWidth).MaxWidth(innerWidth).
        Render(inner)

    cfg := layout.BorderConfig{
        Width:       innerWidth + 2,
        Height:      strings.Count(inner, "\n") + 3,
        Title:       "Profile",
        AccentColor: p.theme.ActiveBorder(),
        Focused:     true,
        Theme:       p.theme,
    }

    return layout.RenderPaneBorder(inner, cfg)
}

// SetSize updates the render dimensions.
func (p *ProfileOverlay) SetSize(width, height int) {
    p.width = width
    p.height = height
}

// SetTheme updates the theme reference for runtime theme switching.
func (p *ProfileOverlay) SetTheme(t theme.Theme) { p.theme = t }

// truncateRunes truncates s to at most max runes, appending … if truncated.
func truncateRunes(s string, max int) string {
    if utf8.RuneCountInString(s) <= max {
        return s
    }
    runes := []rune(s)
    return string(runes[:max-1]) + "…"
}
```

### Step 9.4 — Handle `ProfileLogoutMsg` and `ProfileForgetMsg` in `handlers.go`

Add to `internal/app/handlers.go` inside `handleMsg`:

```go
case panes.ProfileLogoutMsg:
    // Clear tokens and quit — client ID remains in config.
    _ = a.tokenStore.Delete()
    return a, tea.Quit

case panes.ProfileForgetMsg:
    // Clear tokens AND remove client_id from config, then quit.
    _ = a.tokenStore.Delete()
    _ = config.ClearClientID(config.DefaultConfigPath())
    return a, tea.Quit
```

Add import to `handlers.go`:

```go
"github.com/initgrep-apps/spotnik/internal/config"
```

Also add the `ProfileOverlayClosedMsg` handler if not already present — it should already be in the codebase routing the profile overlay closed event. Verify it exists and close the overlay:

```go
case panes.ProfileOverlayClosedMsg:
    a.profileOverlayOpen = false
    return a, nil
```

- [ ] Run `go test ./internal/ui/panes/... -run "TestProfileOverlay" -v`
  Expected: PASS all new tests

- [ ] Run `go build ./...`
  Expected: PASS

- [ ] Run `make ci`
  Expected: PASS

- [ ] Commit:
```bash
git add internal/ui/panes/profile.go internal/ui/panes/messages.go \
        internal/ui/panes/profile_test.go internal/app/handlers.go
git commit -m "feat(profile): add logout/forget actions with double-key confirmation"
```

---

## Task 10: Final wiring, feature branch, and CI gate

**Files:**
- Modify: `internal/app/app.go` (update `NeedsAuth` check in `Init`)
- Modify: `cmd/root.go` (verify `runApp` wires everything correctly)
- Modify: `.goreleaser.yaml` or `Makefile` if they reference `spotifyClientID` ldflags

### Step 10.1 — Remove ldflags reference from build tooling

Search for `spotifyClientID` in build files:

```bash
grep -r "spotifyClientID" . --include="*.yaml" --include="*.yml" --include="Makefile" --include="*.sh"
```

Remove any `-X cmd.spotifyClientID=...` ldflags lines found. The variable no longer exists in `cmd/root.go`.

### Step 10.2 — Verify `authSuccessMsg` handler starts grid correctly

In `handlers.go`, ensure `authSuccessMsg` still works after the server cleanup refactor:

```go
case authSuccessMsg:
    a.needsAuth = false
    a.needsRegister = false
    a.currentView = viewGrid
    a.initAPIClients(m.accessToken)
    var paneCmds []tea.Cmd
    for _, pane := range a.panes {
        if cmd := pane.Init(); cmd != nil {
            paneCmds = append(paneCmds, cmd)
        }
    }
    authCmds := append(paneCmds,
        fetchPlaybackStateCmd(a.player, api.Background),
        tea.Tick(time.Second, func(_ time.Time) tea.Msg {
            return panes.TickMsg{}
        }),
    )
    authCmds = append(authCmds, a.initialFetchCmds()...)
    return a, tea.Batch(authCmds...)
```

### Step 10.3 — Update keybinding documentation

Per `CLAUDE.md` rule: adding new keybindings requires updating all three locations in the same commit.

**New keybindings added:**
- Profile overlay: `l` (logout), `f` (forget)

Update `docs/keybinding.md` — add under Profile Overlay section:

```markdown
| l | Profile overlay | Logout (ends session, keeps Client ID). Press twice to confirm. |
| f | Profile overlay | Forget (removes session + Client ID). Press twice to confirm. |
```

Update `docs/DESIGN.md §17` keybinding table — add the same two rows under the profile overlay section.

Update `internal/ui/panes/help_overlay.go` `helpContent` var — add the two new bindings in the profile section.

### Step 10.4 — Run the full CI gate

```bash
make ci
```

Expected output:
```
golangci-lint: ok
go test ./...: ok  (coverage ≥ 80%)
go build ./...: ok
```

Fix any lint or coverage failures before proceeding.

### Step 10.5 — Final commit and push

```bash
git add -p   # stage only intentional changes
git commit -m "feat(onboarding): wire viewOnboarding into app startup and fix keybinding docs"

git add docs/keybinding.md docs/DESIGN.md internal/ui/panes/help_overlay.go
git commit -m "docs(keybindings): add profile overlay l/f logout/forget bindings to all three locations"

git push origin feat/09-onboarding-auth-ux
```

Open a PR: title `feat(auth): guided TUI onboarding, registration flow, logout/forget`

---

## Self-Review

**Spec coverage check:**

| Spec requirement | Task |
|---|---|
| Remove embedded ldflags client ID | Task 3 |
| Config-first client ID (config.toml) | Tasks 1, 3 |
| `callback_port` config field (default 8888) | Task 1 |
| `config.ClearClientID()` | Task 1 |
| `config.SetClientID()` | Task 3 |
| `StartCallbackServer` accepts port | Task 2 |
| Port-busy error before TUI | Task 3 (runApp) |
| `viewOnboarding` + step state machine | Tasks 4, 6 |
| stepRegister: textinput + instructions | Tasks 4, 8 |
| stepOAuth: full URL, no truncation, wrapURL | Tasks 5, 8 |
| stepOAuth: spinner | Tasks 4, 6 |
| stepError: retry options r/l/q | Tasks 6, 7 |
| `c` to copy URL to clipboard | Tasks 5, 7 |
| viewAuth: updated renderAuthPanel, no truncation | Task 5 |
| Profile overlay: logout with confirmation | Task 9 |
| Profile overlay: forget with confirmation | Task 9 |
| `ProfileLogoutMsg` / `ProfileForgetMsg` handlers | Task 9 |
| CLI `spotnik auth register` | Task 3 |
| CLI `spotnik auth login` | Task 3 |
| CLI `spotnik auth logout` | Task 3 |
| CLI `spotnik auth forget` | Task 3 |
| CLI `spotnik auth status` (shows client ID + token) | Task 3 |
| `spotnik auth` (no subcommand) prints help | Task 3 |
| Keybinding docs (l, f in profile overlay) | Task 10 |
| `CheckAuthState` returns needsRegister + needsAuth | Task 3 |
| Server starts before TUI, port passed via AppOptions | Tasks 3, 4 |

All spec sections covered. No gaps found.
