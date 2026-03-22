# Feature 02 — Authentication

> **Depends on:** Feature 01 (Theme System).
> This feature is the foundation for everything that follows — get it right before moving on.

---

## Implementation Context

### Config loading
```go
// You will create internal/config/config.go.
// Config is loaded once at startup by cmd/root.go — not by this package.
type Config struct {
    ClientID string `toml:"client_id"` // [spotify] section
    Theme    string `toml:"theme"`      // [ui] section, default "black"
}
// Load() reads ~/.config/spotnik/config.toml, returns error if client_id is missing.
```

### Keychain key format
```
spotnik:access_token   → Spotify access token string
spotnik:refresh_token  → Spotify refresh token string
spotnik:token_expiry   → Unix timestamp as string (e.g. "1735689600")
```
Use `github.com/zalando/go-keyring` — service name is always `"spotnik"`.

### Message types
```go
type authSuccessMsg struct{}
type authErrMsg    struct{ err error }
```

### No UI in this feature
Auth runs entirely before the Bubble Tea app starts — in `cmd/root.go`.
No pane model, no lipgloss styles, no design tokens.

### Store fields this feature uses
None — auth runs before the Bubble Tea app starts. Token state lives in the OS keychain.

---

## Goal

Enable a user to authenticate with Spotify once, store tokens securely, and have tokens
refresh automatically so they never need to log in again unless they explicitly log out.

---

## Feature Acceptance Criteria

- First-time user runs `spotnik` and completes auth via browser within 60 seconds
- Returning user runs `spotnik` and app starts in under 500ms (no browser, no prompts)
- Expired token refreshes silently without user intervention
- Failed refresh triggers re-auth flow, never crashes
- `spotnik auth logout` clears all tokens; next run requires fresh auth
- Missing `client_id` in config shows clear, actionable error and exits
- No credentials appear in logs, error output, or tracked files
- All auth code has >= 80% test coverage

---

## User Stories

- **As a first-time user**, I run `spotnik` and am guided through connecting my Spotify account
  via browser OAuth, after which the app starts immediately.
- **As a returning user**, I run `spotnik` and the app starts immediately — no auth needed.
- **As a user with an expired session**, the app silently refreshes my token in the background.
- **As a user**, I can run `spotnik auth logout` to remove my stored credentials.

---

## Spotify OAuth Requirements

- Flow: **Authorization Code + PKCE** (required post-November 2025)
- Redirect URI: `http://localhost:{random_port}/callback` (dynamic port)
- Scopes: see `PRODUCT.md` — request all scopes at once on first auth
- Client ID: provided by user in `~/.config/spotnik/config.toml`
- No client secret needed (PKCE is public-client flow)

---

## First-Run Flow (Detailed)

```
spotnik starts
     │
     ▼
Load config from ~/.config/spotnik/config.toml
     │
     ├── No client_id? → Print setup instructions + exit
     │   "Set your Spotify client_id in ~/.config/spotnik/config.toml"
     │   "Create an app at https://developer.spotify.com/dashboard"
     │
     ▼
Check keychain for stored tokens
     │
     ├── Valid token (not expiring soon) → launch app ✓
     │
     ├── Token expiring within 5 min → refresh, then launch app ✓
     │
     ├── Refresh token present but expired → full re-auth
     │
     └── No tokens → start auth flow:
              │
              ▼
         Generate code_verifier (64 random bytes → base64url, trim to 128 chars)
         Compute code_challenge = base64url(sha256(code_verifier))
              │
              ▼
         Start local HTTP server on random available port
              │
              ▼
         Print to terminal:
         ╭─────────────────────────────────────────────────────╮
         │  Opening Spotify login in your browser...           │
         │                                                     │
         │  If it doesn't open automatically, visit:          │
         │  https://accounts.spotify.com/authorize?...        │
         │                                                     │
         │  Waiting for authorization...  ⣾                   │
         ╰─────────────────────────────────────────────────────╯
              │
              ▼
         Open browser (use xdg-open / open / start depending on OS)
              │
              ▼
         Wait for GET /callback?code=...
              │
              ▼
         Exchange code for tokens via POST to /api/token
              │
              ├── Success → store in keychain, launch app ✓
              │
              └── Failure → print error, exit with code 1
```

---

## Token Storage

Use `github.com/zalando/go-keyring`.

| Keychain Key | Value Stored |
|---|---|
| `spotnik:access_token` | Spotify access token string |
| `spotnik:refresh_token` | Spotify refresh token string |
| `spotnik:token_expiry` | Unix timestamp (int64 as string) |

**Never store tokens in plaintext files or environment variables.**

---

## Token Refresh Logic

```go
// internal/keychain/keychain.go — TokenStore interface and implementations live here.

// IsExpiringSoon returns true if the token expires within 5 minutes.
func (s *KeychainTokenStore) IsExpiringSoon() (bool, error) {
    expiry, err := s.GetExpiry()
    if err != nil {
        return false, err
    }
    return time.Until(expiry) < 5*time.Minute, nil
}

// GetExpiry parses the stored Unix timestamp string to time.Time.
func (s *KeychainTokenStore) GetExpiry() (time.Time, error) {
    // Read spotnik:token_expiry from keychain, parse as int64, return time.Unix(ts, 0)
}
```

```go
// internal/api/auth.go — PKCE flow, token exchange, and refresh logic.

// Refresh exchanges the refresh token for a new access token.
// Updates keychain on success.
func Refresh(ctx context.Context, store keychain.TokenStore, clientID string) error {
    // POST to https://accounts.spotify.com/api/token
    // grant_type=refresh_token
    // refresh_token=...
    // client_id=...
}
```

### Refresh Failure Recovery
If the refresh token is rejected (HTTP 400 from Spotify):
1. Delete all stored tokens from keychain
2. Print: "Session expired. Please re-authenticate."
3. Start the full auth flow (same as first-run)
4. Never crash or exit with an error — always offer re-auth

---

## CLI Commands

```bash
spotnik              # Start app (auto-auth if needed)
spotnik auth         # Re-run auth flow (force fresh login)
spotnik auth logout  # Remove all stored tokens from keychain
spotnik auth status  # Show current auth state (token expiry, scopes)
```

---

## Files to Create

| File | Purpose |
|---|---|
| `internal/api/auth.go` | PKCE flow, token exchange, refresh |
| `internal/api/auth_test.go` | Tests for token generation, exchange, refresh |
| `internal/keychain/keychain.go` | `TokenStore` interface + `KeychainTokenStore` + `InMemoryTokenStore` |
| `internal/keychain/keychain_test.go` | Tests with in-memory mock keychain |
| `internal/config/config.go` | Config struct and `Load()` function |
| `internal/config/config_test.go` | Tests for config loading |
| `cmd/root.go` | Cobra root + auth sub-commands |
| `main.go` | Entry point only (calls cmd.Execute()) |

---

## Task Breakdown

### Task 1.1 — Config loading

**Description:** Create the config package that reads `~/.config/spotnik/config.toml`,
validates required fields, and returns a typed struct with sensible defaults.

**Files:** `internal/config/config.go`, `internal/config/config_test.go`

**Implementation steps:**
- [ ] Create `internal/config/config.go` with `Config` struct and `Load()` function
- [ ] Handle missing file gracefully (use defaults)
- [ ] Validate `client_id` is present, return descriptive error if not
- [ ] Write tests: missing file, valid file, invalid TOML, missing client_id

**Acceptance criteria:**
- Missing config file returns defaults (no error)
- Valid file with all fields parses correctly
- Missing `client_id` returns a descriptive error
- Invalid TOML returns a parse error with file path context
- Default theme is "black"

**Unit tests:**
- `TestLoad_MissingFile_ReturnsDefaults` — no config file returns default Config
- `TestLoad_ValidFile` — parses all fields correctly
- `TestLoad_MissingClientID_ReturnsError` — descriptive error when client_id absent
- `TestLoad_InvalidTOML_ReturnsError` — parse error with context
- `TestLoad_DefaultTheme` — default theme is "black" when not specified
- `TestLoad_PartialConfig_MergesWithDefaults` — only specified fields override defaults

---

### Task 1.2 — Keychain abstraction

**Description:** Create the `TokenStore` interface and two implementations: one backed by
the OS keychain for production, one in-memory for tests. The interface owns expiry checking.

**Files:** `internal/keychain/keychain.go`, `internal/keychain/keychain_test.go`

**Implementation steps:**
- [ ] Create `TokenStore` interface with `Get`, `Set`, `Delete`, `GetExpiry`, `IsExpiringSoon`
- [ ] Implement `KeychainTokenStore` using `go-keyring`
- [ ] Implement `InMemoryTokenStore` for tests (same interface)
- [ ] `GetExpiry()` parses the stored Unix timestamp string to `time.Time`
- [ ] Write tests using `InMemoryTokenStore`

**Acceptance criteria:**
- `TokenStore` interface has `Get`, `Set`, `Delete`, `GetExpiry`, `IsExpiringSoon`
- Both `KeychainTokenStore` and `InMemoryTokenStore` satisfy the interface
- `IsExpiringSoon()` returns true when token expires within 5 minutes
- `GetExpiry()` parses the stored Unix timestamp string to `time.Time`
- `Delete` removes all three keys (access_token, refresh_token, token_expiry)

**Unit tests:**
- `TestInMemoryStore_SetAndGet` — round-trip store/retrieve
- `TestInMemoryStore_Delete` — removes value, subsequent Get errors
- `TestInMemoryStore_GetMissing` — returns descriptive error
- `TestIsExpiringSoon_True` — returns true when expiry < 5 minutes
- `TestIsExpiringSoon_False` — returns false when expiry > 5 minutes
- `TestIsExpiringSoon_AlreadyExpired` — returns true for past timestamps
- `TestGetExpiry_ValidTimestamp` — parses Unix string to correct time.Time
- `TestGetExpiry_InvalidTimestamp` — returns error for non-numeric string
- `TestKeychain_ImplementsInterface` — compile-time interface check

---

### Task 1.3 — PKCE flow

**Description:** Implement the three PKCE primitives: verifier generation,
challenge computation, and authorization URL construction.

**Files:** `internal/api/auth.go`, `internal/api/auth_test.go`

**Implementation steps:**
- [ ] Implement `GenerateCodeVerifier()` — random 64 bytes, base64url encoded
- [ ] Implement `ComputeCodeChallenge(verifier string) string` — SHA256 + base64url
- [ ] Implement `BuildAuthURL(clientID, redirectURI, challenge, scopes string) string`
- [ ] Write unit tests for all three functions

**Acceptance criteria:**
- Code verifier is 128 chars of base64url-safe characters
- Code challenge is SHA256 of verifier, base64url-encoded, no padding
- Auth URL contains all required params: client_id, response_type, redirect_uri, code_challenge, code_challenge_method, scope

**Unit tests:**
- `TestGenerateCodeVerifier_Length` — exactly 128 characters
- `TestGenerateCodeVerifier_Base64URLSafe` — only contains [A-Za-z0-9_-]
- `TestGenerateCodeVerifier_Unique` — two calls produce different values
- `TestComputeCodeChallenge_KnownVector` — test with a known input/output pair
- `TestBuildAuthURL_ContainsAllParams` — URL has client_id, response_type, redirect_uri, code_challenge, code_challenge_method, scope

---

### Task 1.4 — Local callback server

**Description:** Start a temporary HTTP server on a random port to receive the OAuth
callback from Spotify, extract the authorization code, and shut down.

**Files:** `internal/api/auth.go`, `internal/api/auth_test.go`

**Implementation steps:**
- [ ] Start `net/http` server on random available port
- [ ] Handle `GET /callback?code=...` — extract code, signal channel, shut down server
- [ ] Timeout after 5 minutes if no callback received
- [ ] Handle `?error=access_denied` — show message, exit cleanly

**Acceptance criteria:**
- Server starts on a random available port
- `GET /callback?code=abc` extracts code and signals channel
- `GET /callback?error=access_denied` returns descriptive error
- Server shuts down after receiving callback
- Server times out after 5 minutes with descriptive error

**Unit tests:**
- `TestCallbackServer_ExtractsCode` — sends GET with code, verifies code received on channel
- `TestCallbackServer_HandlesError` — sends error=access_denied, verifies error
- `TestCallbackServer_RandomPort` — server binds to port > 0

---

### Task 1.5 — Token exchange

**Description:** Exchange the authorization code for access and refresh tokens via
Spotify's token endpoint, then store them in the keychain.

**Files:** `internal/api/auth.go`, `internal/api/auth_test.go`

**Implementation steps:**
- [ ] `ExchangeCode(ctx, code, verifier, redirectURI, clientID) (TokenPair, error)`
- [ ] POST to `https://accounts.spotify.com/api/token`
- [ ] Parse response, store in keychain
- [ ] Write tests with `httptest.NewServer` mock

**Acceptance criteria:**
- POST to Spotify token endpoint with correct form body (grant_type, code, redirect_uri, client_id, code_verifier)
- Successful response parsed into access_token + refresh_token + expires_in
- Tokens stored in keychain via TokenStore
- Error response returns descriptive error

**Unit tests (using httptest.NewServer):**
- `TestExchangeCode_Success` — mock returns valid token JSON, verify parsed correctly
- `TestExchangeCode_ServerError` — mock returns 500, verify descriptive error
- `TestExchangeCode_InvalidJSON` — mock returns garbage, verify parse error
- `TestExchangeCode_MissingFields` — mock returns partial JSON, verify error

---

### Task 1.6 — Token refresh

**Description:** Implement token refresh using the stored refresh token. Handle both
success (update keychain) and failure (trigger re-auth).

**Files:** `internal/api/auth.go`, `internal/api/auth_test.go`

**Implementation steps:**
- [ ] `Refresh(ctx, refreshToken, clientID) (TokenPair, error)`
- [ ] Proactive refresh: check on startup if expiring soon
- [ ] Reactive refresh: triggered on 401 response from API
- [ ] On HTTP 400 (invalid grant): delete tokens, return specific error for re-auth
- [ ] Write tests with mock server

**Acceptance criteria:**
- POST with grant_type=refresh_token, refresh_token, client_id
- On success: updates keychain with new tokens
- On 400 (invalid grant): triggers re-auth flow (deletes tokens, returns specific error type)

**Unit tests (using httptest.NewServer):**
- `TestRefresh_Success` — new tokens stored in keychain
- `TestRefresh_InvalidGrant` — returns specific error indicating re-auth needed
- `TestRefresh_NetworkError` — returns wrapped network error

---

### Task 1.7 — CLI wiring

**Description:** Wire up the root command and auth subcommands in `cmd/root.go`, and
ensure `main.go` is a thin entry point.

**Files:** `cmd/root.go`, `main.go`

**Implementation steps:**
- [ ] `cmd/root.go` — check auth state, run flow if needed, then launch app
- [ ] `spotnik auth` — force re-auth
- [ ] `spotnik auth logout` — delete keychain tokens
- [ ] `spotnik auth status` — print token info

**Acceptance criteria:**
- `main.go` calls `cmd.Execute()` only
- `spotnik` (no args): checks auth, runs flow if needed, launches app
- `spotnik auth`: forces fresh re-auth
- `spotnik auth logout`: deletes all keychain tokens, prints confirmation
- `spotnik auth status`: prints token expiry and scopes

**Unit tests:**
- `TestRootCmd_Executes` — root command runs without error
- `TestAuthLogout_ClearsTokens` — logout deletes all 3 keychain keys
- `TestAuthStatus_PrintsExpiry` — shows formatted expiry time

**Integration tests:**
- `TestFullAuthFlow_ConfigToToken` — load config → check keychain → exchange code → store tokens (uses httptest mock for Spotify)

---

### Task 1.8 — First-run UX

**Description:** Polish the first-run experience: clear error for missing config,
browser auto-open, spinner during callback wait, and success message.

**Files:** `cmd/root.go` (modify)

**Implementation steps:**
- [ ] Print clear setup instructions if `client_id` missing
- [ ] Show browser-opening feedback with spinner
- [ ] Show "Authorization successful! Starting spotnik..." on success

**Acceptance criteria:**
- Missing client_id prints setup URL and instructions, exits with code 1
- Browser auto-opens on macOS/Linux (best-effort)
- Shows spinner "Waiting for authorization..." during callback wait
- Shows "Authorization successful! Starting spotnik..." on success

**Unit tests:**
- `TestMissingClientID_PrintsInstructions` — captures stdout, verifies setup message
- `TestMissingClientID_ExitsWithCode1` — verifies exit code

---

## Out of Scope for This Feature

- Multi-account support (not planned)
- Client credentials flow (not needed — all endpoints require user auth)
- Device authorization flow (not applicable to CLI)

---

*Last updated: 2026-03-22*
