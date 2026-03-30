---
title: "Spotify OAuth Authentication"
feature: 02-auth
status: done
---

## Background
This story built the complete authentication lifecycle for Spotnik: config loading from TOML, a keychain abstraction for secure token storage, PKCE primitives (verifier, challenge, auth URL), a local callback server for the OAuth redirect, token exchange and refresh against the Spotify API, CLI wiring for `spotnik`, `spotnik auth`, `spotnik auth logout`, and `spotnik auth status`, and first-run UX polish including browser auto-open and spinner feedback.

Auth runs before the Bubble Tea app starts. Token state lives in the OS keychain, not in the Store. The flow supports first-time users (browser OAuth), returning users (instant start), expired sessions (silent refresh), and explicit logout.

## Design

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
- Client credentials flow (not needed -- all endpoints require user auth)
- Device authorization flow (not applicable to CLI)

## Acceptance Criteria
- [ ] First-time user runs `spotnik` and completes auth via browser within 60 seconds
- [ ] Returning user runs `spotnik` and app starts in under 500ms (no browser, no prompts)
- [ ] Expired token refreshes silently without user intervention
- [ ] Failed refresh triggers re-auth flow, never crashes
- [ ] `spotnik auth logout` clears all tokens; next run requires fresh auth
- [ ] Missing `client_id` in config shows clear, actionable error and exits
- [ ] No credentials appear in logs, error output, or tracked files
- [ ] All auth code has >= 80% test coverage

## Tasks
- [ ] Config loading -- Create config package that reads `~/.config/spotnik/config.toml`, validates required fields, returns typed struct with defaults
      - test: `TestLoad_MissingFile_ReturnsDefaults`, `TestLoad_ValidFile`, `TestLoad_MissingClientID_ReturnsError`, `TestLoad_InvalidTOML_ReturnsError`, `TestLoad_DefaultTheme`, `TestLoad_PartialConfig_MergesWithDefaults`
- [ ] Keychain abstraction -- Create `TokenStore` interface with `KeychainTokenStore` and `InMemoryTokenStore` implementations
      - test: `TestInMemoryStore_SetAndGet`, `TestInMemoryStore_Delete`, `TestInMemoryStore_GetMissing`, `TestIsExpiringSoon_True`, `TestIsExpiringSoon_False`, `TestIsExpiringSoon_AlreadyExpired`, `TestGetExpiry_ValidTimestamp`, `TestGetExpiry_InvalidTimestamp`, `TestKeychain_ImplementsInterface`
- [ ] PKCE flow -- Implement verifier generation, challenge computation, and auth URL construction
      - test: `TestGenerateCodeVerifier_Length`, `TestGenerateCodeVerifier_Base64URLSafe`, `TestGenerateCodeVerifier_Unique`, `TestComputeCodeChallenge_KnownVector`, `TestBuildAuthURL_ContainsAllParams`
- [ ] Local callback server -- Start temporary HTTP server for OAuth callback, extract auth code
      - test: `TestCallbackServer_ExtractsCode`, `TestCallbackServer_HandlesError`, `TestCallbackServer_RandomPort`
- [ ] Token exchange -- Exchange authorization code for access/refresh tokens via Spotify token endpoint
      - test: `TestExchangeCode_Success`, `TestExchangeCode_ServerError`, `TestExchangeCode_InvalidJSON`, `TestExchangeCode_MissingFields`
- [ ] Token refresh -- Implement token refresh with proactive and reactive strategies
      - test: `TestRefresh_Success`, `TestRefresh_InvalidGrant`, `TestRefresh_NetworkError`
- [ ] CLI wiring -- Wire root command and auth subcommands in `cmd/root.go`
      - test: `TestRootCmd_Executes`, `TestAuthLogout_ClearsTokens`, `TestAuthStatus_PrintsExpiry`, `TestFullAuthFlow_ConfigToToken`
- [ ] First-run UX -- Polish first-run experience with clear errors, browser auto-open, spinner
      - test: `TestMissingClientID_PrintsInstructions`, `TestMissingClientID_ExitsWithCode1`
