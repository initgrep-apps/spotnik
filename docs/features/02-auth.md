# Feature 02 — Authentication

> **Depends on:** Feature 01 (Theme System).
> This feature is the foundation for everything that follows — get it right before moving on.

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

---

---

## Goal

Enable a user to authenticate with Spotify once, store tokens securely, and have tokens
refresh automatically so they never need to log in again unless they explicitly log out.

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
// internal/api/auth.go

// IsExpiringSoon returns true if the token expires within 5 minutes.
func (t *TokenStore) IsExpiringSoon() bool {
    return time.Until(t.Expiry) < 5*time.Minute
}

// Refresh exchanges the refresh token for a new access token.
// Updates keychain on success.
func (t *TokenStore) Refresh(ctx context.Context, clientID string) error {
    // POST to https://accounts.spotify.com/api/token
    // grant_type=refresh_token
    // refresh_token=...
    // client_id=...
}
```

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
| `internal/keychain/keychain.go` | Keychain read/write/delete abstraction |
| `internal/keychain/keychain_test.go` | Tests with in-memory mock keychain |
| `cmd/root.go` | Cobra root + auth sub-commands |
| `main.go` | Entry point only (calls cmd.Execute()) |

---

## Task Breakdown

### Task 1.1 — Config loading
- [ ] Create `internal/config/config.go` with `Config` struct and `Load()` function
- [ ] Handle missing file gracefully (use defaults)
- [ ] Validate `client_id` is present, return descriptive error if not
- [ ] Write tests: missing file, valid file, invalid TOML, missing client_id

### Task 1.2 — Keychain abstraction
- [ ] Create `TokenStore` interface with `Get`, `Set`, `Delete`, `IsExpiringSoon`
- [ ] Implement `KeychainTokenStore` using `go-keyring`
- [ ] Implement `InMemoryTokenStore` for tests (same interface)
- [ ] Write tests using `InMemoryTokenStore`

### Task 1.3 — PKCE flow
- [ ] Implement `GenerateCodeVerifier()` — random 64 bytes, base64url encoded
- [ ] Implement `ComputeCodeChallenge(verifier string) string` — SHA256 + base64url
- [ ] Implement `BuildAuthURL(clientID, redirectURI, challenge, scopes string) string`
- [ ] Write unit tests for all three functions

### Task 1.4 — Local callback server
- [ ] Start `net/http` server on random available port
- [ ] Handle `GET /callback?code=...` — extract code, signal channel, shut down server
- [ ] Timeout after 5 minutes if no callback received
- [ ] Handle `?error=access_denied` — show message, exit cleanly

### Task 1.5 — Token exchange
- [ ] `ExchangeCode(ctx, code, verifier, redirectURI, clientID) (TokenPair, error)`
- [ ] POST to `https://accounts.spotify.com/api/token`
- [ ] Parse response, store in keychain
- [ ] Write tests with `httptest.NewServer` mock

### Task 1.6 — Token refresh
- [ ] `Refresh(ctx, refreshToken, clientID) (TokenPair, error)`
- [ ] Proactive refresh: check on startup if expiring soon
- [ ] Reactive refresh: triggered on 401 response from API
- [ ] Write tests with mock server

### Task 1.7 — CLI wiring
- [ ] `cmd/root.go` — check auth state, run flow if needed, then launch app
- [ ] `spotnik auth` — force re-auth
- [ ] `spotnik auth logout` — delete keychain tokens
- [ ] `spotnik auth status` — print token info

### Task 1.8 — First-run UX
- [ ] Print clear setup instructions if `client_id` missing
- [ ] Show browser-opening feedback with spinner
- [ ] Show "Authorization successful! Starting spotnik..." on success

---

## Acceptance Criteria

- [ ] `spotnik` with no prior auth opens browser and completes flow within 60 seconds
- [ ] `spotnik` on second run starts in under 500ms (no browser, no prompts)
- [ ] Expired token is refreshed silently without user intervention
- [ ] `spotnik auth logout` clears all tokens, next `spotnik` run re-auths
- [ ] Missing `client_id` shows a clear, actionable error message
- [ ] All auth code has >= 80% test coverage
- [ ] No credentials appear in logs or error output

---

## Out of Scope for This Feature

- Multi-account support (not planned)
- Client credentials flow (not needed — all endpoints require user auth)
- Device authorization flow (not applicable to CLI)

---

*Last updated: 2026-02-21*
