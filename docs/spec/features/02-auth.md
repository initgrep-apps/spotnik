---
title: "Authentication"
description: "Enables secure Spotify OAuth (PKCE) login, automatic token refresh, and keychain-based credential storage so users authenticate once and never need to log in again."
status: done
stories: [02]
---

# Authentication

## Background

Spotnik requires authenticated access to the Spotify Web API for every feature beyond this one. The authentication system implements the Authorization Code + PKCE flow (required by Spotify post-November 2025), which is a public-client OAuth flow that does not require a client secret. This makes it suitable for a single-binary CLI tool distributed to end users.

The auth system runs entirely before the Bubble Tea TUI starts, in `cmd/root.go`. There are no panes, no lipgloss styles, and no design tokens involved. Tokens are stored in the OS keychain via `go-keyring`, never in plaintext files or environment variables. The system handles first-time login (browser-based OAuth), automatic token refresh on startup when tokens are expiring soon, and graceful re-auth when refresh tokens are rejected.

This feature also establishes the config loading system (`internal/config/`) which reads `~/.config/spotnik/config.toml` for the user's Spotify `client_id` and UI preferences like theme selection. Together, config + auth form the foundation that every subsequent feature depends on.

---

## Story: Spotify OAuth Authentication (spec 02)

### Background

This story built the complete authentication lifecycle for Spotnik: config loading from TOML, a keychain abstraction for secure token storage, PKCE primitives (verifier, challenge, auth URL), a local callback server for the OAuth redirect, token exchange and refresh against the Spotify API, CLI wiring for `spotnik`, `spotnik auth`, `spotnik auth logout`, and `spotnik auth status`, and first-run UX polish including browser auto-open and spinner feedback.

Auth runs before the Bubble Tea app starts. Token state lives in the OS keychain, not in the Store. The flow supports first-time users (browser OAuth), returning users (instant start), expired sessions (silent refresh), and explicit logout.

### Acceptance Criteria

- [ ] First-time user runs `spotnik` and completes auth via browser within 60 seconds
- [ ] Returning user runs `spotnik` and app starts in under 500ms (no browser, no prompts)
- [ ] Expired token refreshes silently without user intervention
- [ ] Failed refresh triggers re-auth flow, never crashes
- [ ] `spotnik auth logout` clears all tokens; next run requires fresh auth
- [ ] Missing `client_id` in config shows clear, actionable error and exits
- [ ] No credentials appear in logs, error output, or tracked files
- [ ] All auth code has >= 80% test coverage

### Tasks

1. **Config loading** — Create the config package that reads `~/.config/spotnik/config.toml`, validates required fields, and returns a typed struct with sensible defaults.
   - Files: `internal/config/config.go`, `internal/config/config_test.go`
   - Implementation:
     - Create `Config` struct with `ClientID string` (toml: `client_id`, `[spotify]` section) and `Theme string` (toml: `theme`, `[ui]` section, default `"black"`)
     - `Load()` reads config file, returns error if `client_id` is missing
     - Handle missing file gracefully (use defaults)
     - Validate `client_id` is present, return descriptive error if not
   - Acceptance:
     - Missing config file returns defaults (no error)
     - Valid file with all fields parses correctly
     - Missing `client_id` returns a descriptive error
     - Invalid TOML returns a parse error with file path context
     - Default theme is `"black"`
   - Tests:
     - `TestLoad_MissingFile_ReturnsDefaults` — no config file returns default Config
     - `TestLoad_ValidFile` — parses all fields correctly
     - `TestLoad_MissingClientID_ReturnsError` — descriptive error when client_id absent
     - `TestLoad_InvalidTOML_ReturnsError` — parse error with context
     - `TestLoad_DefaultTheme` — default theme is "black" when not specified
     - `TestLoad_PartialConfig_MergesWithDefaults` — only specified fields override defaults

2. **Keychain abstraction** — Create the `TokenStore` interface and two implementations: one backed by the OS keychain for production, one in-memory for tests. The interface owns expiry checking.
   - Files: `internal/keychain/keychain.go`, `internal/keychain/keychain_test.go`
   - Implementation:
     - Create `TokenStore` interface with `Get`, `Set`, `Delete`, `GetExpiry`, `IsExpiringSoon`
     - Implement `KeychainTokenStore` using `go-keyring` (service name `"spotnik"`)
     - Implement `InMemoryTokenStore` for tests (same interface)
     - `GetExpiry()` parses stored Unix timestamp string to `time.Time`
     - Keychain keys: `spotnik:access_token`, `spotnik:refresh_token`, `spotnik:token_expiry`
   - Acceptance:
     - `TokenStore` interface has `Get`, `Set`, `Delete`, `GetExpiry`, `IsExpiringSoon`
     - Both `KeychainTokenStore` and `InMemoryTokenStore` satisfy the interface
     - `IsExpiringSoon()` returns true when token expires within 5 minutes
     - `GetExpiry()` parses the stored Unix timestamp string to `time.Time`
     - `Delete` removes all three keys (access_token, refresh_token, token_expiry)
   - Tests:
     - `TestInMemoryStore_SetAndGet` — round-trip store/retrieve
     - `TestInMemoryStore_Delete` — removes value, subsequent Get errors
     - `TestInMemoryStore_GetMissing` — returns descriptive error
     - `TestIsExpiringSoon_True` — returns true when expiry < 5 minutes
     - `TestIsExpiringSoon_False` — returns false when expiry > 5 minutes
     - `TestIsExpiringSoon_AlreadyExpired` — returns true for past timestamps
     - `TestGetExpiry_ValidTimestamp` — parses Unix string to correct time.Time
     - `TestGetExpiry_InvalidTimestamp` — returns error for non-numeric string
     - `TestKeychain_ImplementsInterface` — compile-time interface check

3. **PKCE flow** — Implement the three PKCE primitives: verifier generation, challenge computation, and authorization URL construction.
   - Files: `internal/api/auth.go`, `internal/api/auth_test.go`
   - Implementation:
     - `GenerateCodeVerifier()` — random 64 bytes, base64url encoded, trimmed to 128 chars
     - `ComputeCodeChallenge(verifier string) string` — SHA256 + base64url, no padding
     - `BuildAuthURL(clientID, redirectURI, challenge, scopes string) string`
   - Acceptance:
     - Code verifier is 128 chars of base64url-safe characters
     - Code challenge is SHA256 of verifier, base64url-encoded, no padding
     - Auth URL contains all required params: client_id, response_type, redirect_uri, code_challenge, code_challenge_method, scope
   - Tests:
     - `TestGenerateCodeVerifier_Length` — exactly 128 characters
     - `TestGenerateCodeVerifier_Base64URLSafe` — only contains [A-Za-z0-9_-]
     - `TestGenerateCodeVerifier_Unique` — two calls produce different values
     - `TestComputeCodeChallenge_KnownVector` — test with a known input/output pair
     - `TestBuildAuthURL_ContainsAllParams` — URL has client_id, response_type, redirect_uri, code_challenge, code_challenge_method, scope

4. **Local callback server** — Start a temporary HTTP server on a random port to receive the OAuth callback from Spotify, extract the authorization code, and shut down.
   - Files: `internal/api/auth.go`, `internal/api/auth_test.go`
   - Implementation:
     - Start `net/http` server on random available port
     - Handle `GET /callback?code=...` — extract code, signal channel, shut down server
     - Timeout after 5 minutes if no callback received
     - Handle `?error=access_denied` — show message, exit cleanly
   - Acceptance:
     - Server starts on a random available port
     - `GET /callback?code=abc` extracts code and signals channel
     - `GET /callback?error=access_denied` returns descriptive error
     - Server shuts down after receiving callback
     - Server times out after 5 minutes with descriptive error
   - Tests:
     - `TestCallbackServer_ExtractsCode` — sends GET with code, verifies code received on channel
     - `TestCallbackServer_HandlesError` — sends error=access_denied, verifies error
     - `TestCallbackServer_RandomPort` — server binds to port > 0

5. **Token exchange** — Exchange the authorization code for access and refresh tokens via Spotify's token endpoint, then store them in the keychain.
   - Files: `internal/api/auth.go`, `internal/api/auth_test.go`
   - Implementation:
     - `ExchangeCode(ctx, code, verifier, redirectURI, clientID) (TokenPair, error)`
     - POST to `https://accounts.spotify.com/api/token`
     - Parse response, store in keychain
   - Acceptance:
     - POST to Spotify token endpoint with correct form body (grant_type, code, redirect_uri, client_id, code_verifier)
     - Successful response parsed into access_token + refresh_token + expires_in
     - Tokens stored in keychain via TokenStore
     - Error response returns descriptive error
   - Tests (using `httptest.NewServer`):
     - `TestExchangeCode_Success` — mock returns valid token JSON, verify parsed correctly
     - `TestExchangeCode_ServerError` — mock returns 500, verify descriptive error
     - `TestExchangeCode_InvalidJSON` — mock returns garbage, verify parse error
     - `TestExchangeCode_MissingFields` — mock returns partial JSON, verify error

6. **Token refresh** — Implement token refresh using the stored refresh token. Handle both success (update keychain) and failure (trigger re-auth).
   - Files: `internal/api/auth.go`, `internal/api/auth_test.go`
   - Implementation:
     - `Refresh(ctx, refreshToken, clientID) (TokenPair, error)`
     - Proactive refresh: check on startup if expiring soon
     - Reactive refresh: triggered on 401 response from API
     - On HTTP 400 (invalid grant): delete tokens, return specific error for re-auth
   - Acceptance:
     - POST with grant_type=refresh_token, refresh_token, client_id
     - On success: updates keychain with new tokens
     - On 400 (invalid grant): triggers re-auth flow (deletes tokens, returns specific error type)
   - Refresh failure recovery:
     - Delete all stored tokens from keychain
     - Print: "Session expired. Please re-authenticate."
     - Start the full auth flow (same as first-run)
     - Never crash or exit with an error — always offer re-auth
   - Tests (using `httptest.NewServer`):
     - `TestRefresh_Success` — new tokens stored in keychain
     - `TestRefresh_InvalidGrant` — returns specific error indicating re-auth needed
     - `TestRefresh_NetworkError` — returns wrapped network error

7. **CLI wiring** — Wire up the root command and auth subcommands in `cmd/root.go`, and ensure `main.go` is a thin entry point.
   - Files: `cmd/root.go`, `main.go`
   - Implementation:
     - `cmd/root.go` — check auth state, run flow if needed, then launch app
     - `spotnik auth` — force re-auth
     - `spotnik auth logout` — delete keychain tokens
     - `spotnik auth status` — print token info
   - Acceptance:
     - `main.go` calls `cmd.Execute()` only
     - `spotnik` (no args): checks auth, runs flow if needed, launches app
     - `spotnik auth`: forces fresh re-auth
     - `spotnik auth logout`: deletes all keychain tokens, prints confirmation
     - `spotnik auth status`: prints token expiry and scopes
   - CLI commands:
     - `spotnik` — Start app (auto-auth if needed)
     - `spotnik auth` — Re-run auth flow (force fresh login)
     - `spotnik auth logout` — Remove all stored tokens from keychain
     - `spotnik auth status` — Show current auth state (token expiry, scopes)
   - Tests:
     - `TestRootCmd_Executes` — root command runs without error
     - `TestAuthLogout_ClearsTokens` — logout deletes all 3 keychain keys
     - `TestAuthStatus_PrintsExpiry` — shows formatted expiry time
   - Integration tests:
     - `TestFullAuthFlow_ConfigToToken` — load config, check keychain, exchange code, store tokens (uses httptest mock for Spotify)

8. **First-run UX** — Polish the first-run experience: clear error for missing config, browser auto-open, spinner during callback wait, and success message.
   - Files: `cmd/root.go` (modify)
   - Implementation:
     - Print clear setup instructions if `client_id` missing
     - Show browser-opening feedback with spinner
     - Show "Authorization successful! Starting spotnik..." on success
     - Open browser using xdg-open / open / start depending on OS
   - Acceptance:
     - Missing client_id prints setup URL and instructions, exits with code 1
     - Browser auto-opens on macOS/Linux (best-effort)
     - Shows spinner "Waiting for authorization..." during callback wait
     - Shows "Authorization successful! Starting spotnik..." on success
   - First-run flow:
     - Load config from `~/.config/spotnik/config.toml`
     - No client_id: print setup instructions + exit
     - Check keychain for stored tokens
     - Valid token (not expiring soon): launch app
     - Token expiring within 5 min: refresh, then launch app
     - Refresh token present but expired: full re-auth
     - No tokens: start auth flow (generate verifier/challenge, start local server, open browser, wait for callback, exchange code, store tokens, launch app)
   - Tests:
     - `TestMissingClientID_PrintsInstructions` — captures stdout, verifies setup message
     - `TestMissingClientID_ExitsWithCode1` — verifies exit code

### Spotify OAuth Requirements

- Flow: Authorization Code + PKCE (required post-November 2025)
- Redirect URI: `http://localhost:{random_port}/callback` (dynamic port)
- Scopes: all scopes requested at once on first auth
- Client ID: provided by user in `~/.config/spotnik/config.toml`
- No client secret needed (PKCE is public-client flow)

### Token Storage

| Keychain Key | Value Stored |
|---|---|
| `spotnik:access_token` | Spotify access token string |
| `spotnik:refresh_token` | Spotify refresh token string |
| `spotnik:token_expiry` | Unix timestamp (int64 as string) |

Use `github.com/zalando/go-keyring` with service name `"spotnik"`. Never store tokens in plaintext files or environment variables.

### Message Types

```go
type authSuccessMsg struct{}
type authErrMsg    struct{ err error }
```

### Files

| File | Purpose |
|---|---|
| `internal/api/auth.go` | PKCE flow, token exchange, refresh |
| `internal/api/auth_test.go` | Tests for token generation, exchange, refresh |
| `internal/keychain/keychain.go` | `TokenStore` interface + `KeychainTokenStore` + `InMemoryTokenStore` |
| `internal/keychain/keychain_test.go` | Tests with in-memory mock keychain |
| `internal/config/config.go` | Config struct and `Load()` function |
| `internal/config/config_test.go` | Tests for config loading |
| `cmd/root.go` | Cobra root + auth sub-commands |
| `main.go` | Entry point only (calls `cmd.Execute()`) |

### Out of Scope

- Multi-account support (not planned)
- Client credentials flow (not needed — all endpoints require user auth)
- Device authorization flow (not applicable to CLI)
