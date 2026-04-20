---
title: "CLI — Auth Subcommands + Remove Embedded ldflags Client ID"
feature: 09-auth-and-profile
status: open
---

## Background

The previous design embedded a Spotify Developer client ID at build time via ldflags:

```
-X cmd.spotifyClientID=<id>
```

Spotify does not allow shared OAuth credentials across users, so this model is fundamentally
broken for public distribution: every user who installs the binary uses the same developer's
Spotify app, exhausting rate limits and violating Spotify's Terms of Service.

The redesign is **config-first**: client ID lives in `~/.config/spotnik/config.toml`, written by
the user during first-time setup. The ldflags variable is removed entirely. The `spotnik auth`
command group is restructured with explicit subcommands for every auth action.

**Depends on:** Stories 134 and 135 (`Config.CallbackPort`, `StartCallbackServer(port)`).

## Design

### Remove `spotifyClientID` var from `cmd/root.go`

Delete the package-level variable:

```go
// REMOVE:
var spotifyClientID string
```

Remove from `loadConfigFromPath`: the function no longer accepts or uses an embedded client ID
parameter. `cfg.ClientID` comes solely from `config.Load()`.

### Remove ldflags from build tooling

Search for `spotifyClientID` in build and CI files:

```bash
grep -r "spotifyClientID" . --include="*.yaml" --include="*.yml" \
    --include="Makefile" --include="*.sh" --include="*.json"
```

Remove any `-X cmd.spotifyClientID=...` lines found in `.goreleaser.yaml`, `Makefile`, GitHub
Actions workflow files, or any other build scripts.

### Restructure `cmd/root.go`

**`spotnik auth` (no subcommand)** — currently calls `runApp`. After this change it prints
usage/help listing all subcommands. `authCmd` has no `RunE`.

**New subcommands** (all under `authCmd`):

| Subcommand | Function | Behaviour |
|---|---|---|
| `register` | `runRegister` | Show setup instructions, prompt for client ID, save to config, run OAuth flow |
| `login` | `runAuthLogin` | Force re-authentication; requires client ID in config; clears tokens and runs OAuth |
| `logout` | (inline) | Clear tokens only via `LogoutTokens`; print confirmation; exit 0 |
| `forget` | (inline) | Clear tokens + remove client ID via `RunForget`; print confirmation; exit 0 |
| `status` | (inline) | Print auth state via `PrintAuthStatus` |

**Exported helper functions** (testable without cobra wiring):

```go
// LogoutTokens removes all token keys from the store.
func LogoutTokens(store keychain.TokenStore) error

// RunForget clears tokens AND removes client_id from config at path.
func RunForget(store keychain.TokenStore, configPath string) error

// PrintAuthStatus writes current auth + registration state to w.
func PrintAuthStatus(store keychain.TokenStore, configPath string, w io.Writer) error

// CheckAuthState returns (needsRegister, needsAuth).
// needsRegister: no client_id in config.
// needsAuth: client_id present but no valid token.
func CheckAuthState(cfg *config.Config, store keychain.TokenStore) (needsRegister, needsAuth bool)
```

**`RunAuthFlow`** — updated to use `cfg.CallbackPort` (no longer passes `0`):

```go
callbackSrv, codeCh, err := api.StartCallbackServer(cfg.CallbackPort)
```

**`runApp`** — the main TUI entry point — updated to call `CheckAuthState` and pass
`NeedsRegister` + `NeedsAuth` into `AppOptions`. Starts the callback server early when auth is
needed:

```go
if needsRegister || needsAuth {
    srv, codeCh, err := api.StartCallbackServer(cfg.CallbackPort)
    if err != nil {
        return fmt.Errorf("port %d is busy — set a different callback_port in "+
            "~/.config/spotnik/config.toml: %w", cfg.CallbackPort, err)
    }
    opts.CallbackPort = cfg.CallbackPort
    opts.CallbackCodeCh = codeCh
    opts.CallbackClose = srv.Close
}
```

`PrintMissingClientIDInstructions` may be retained as a thin stub for backward compatibility
with existing tests; it should redirect the user to `spotnik auth register`.

### Tests — `cmd/root_test.go`

Write **failing** tests first (TDD):

```go
func TestAuthForgetCmd_clearsTokensAndClientID(t *testing.T) {
    // Arrange: config file with client_id, in-memory token store with a token.
    // Act: RunForget(store, path).
    // Assert: store.Get(KeyAccessToken) → error (gone); cfg.ClientID == "".
}

func TestAuthStatusCmd_showsClientIDPresent(t *testing.T) {
    // Arrange: config with client_id, store with access token and expiry.
    // Act: PrintAuthStatus(store, path, &buf).
    // Assert: buf contains "Client ID: present" and "Status: authenticated".
}

func TestAuthStatusCmd_showsClientIDMissing(t *testing.T) {
    // Arrange: config with no client_id, empty store.
    // Act: PrintAuthStatus(store, path, &buf).
    // Assert: buf contains "Client ID: not set" and "Status: not authenticated".
}

func TestCheckAuthState_noClientID_needsRegister(t *testing.T) {
    // Arrange: empty config (no client_id).
    // Assert: needsRegister == true, needsAuth == false.
}

func TestCheckAuthState_clientIDNoToken_needsAuth(t *testing.T) {
    // Arrange: config with client_id, empty token store.
    // Assert: needsRegister == false, needsAuth == true.
}

func TestCheckAuthState_clientIDValidToken_noAuthNeeded(t *testing.T) {
    // Arrange: config with client_id, store with non-expired token.
    // Assert: needsRegister == false, needsAuth == false.
}
```

### Build-file cleanup

After removing `spotifyClientID` from `root.go`:
- Remove `-X cmd.spotifyClientID=...` from `.goreleaser.yaml` `ldflags` array
- Remove the same from `Makefile` `build` or `run` targets if present
- Remove from any GitHub Actions workflow `env` or `run` steps that set the ldflag

Verify with `go build ./...` that there are no undefined references.

## Acceptance Criteria

- [ ] `var spotifyClientID string` deleted from `cmd/root.go`
- [ ] `loadConfigFromPath` takes no embedded ID parameter
- [ ] `spotnik auth` with no subcommand prints usage, not the TUI
- [ ] `spotnik auth register` — shows instructions, prompts for client ID, saves it, runs OAuth
- [ ] `spotnik auth login` — clears tokens and runs OAuth; errors if no client ID in config
- [ ] `spotnik auth logout` — calls `LogoutTokens`, prints confirmation, exits 0
- [ ] `spotnik auth forget` — calls `RunForget`, prints confirmation, exits 0
- [ ] `spotnik auth status` — calls `PrintAuthStatus`; shows client ID + token state
- [ ] `RunForget` clears tokens and removes `client_id` from config; all 6 tests pass
- [ ] `PrintAuthStatus` outputs correct status for present/missing client ID and tokens
- [ ] `CheckAuthState` returns correct `(needsRegister, needsAuth)` for all 3 token states
- [ ] No `spotifyClientID` reference remains in any build file (`.goreleaser.yaml`, `Makefile`,
      `*.yml`/`*.yaml` workflow files)
- [ ] `go build ./...` passes; `make ci` passes

## Tasks

- [ ] Write failing tests in `cmd/root_test.go` for `RunForget`, `PrintAuthStatus`, and
      `CheckAuthState`
      - test: `go test ./cmd/... -run "TestAuthForgetCmd|TestAuthStatusCmd|TestCheckAuthState" -v`
        → compile errors
- [ ] Delete `spotifyClientID` var; update `loadConfigFromPath`; restructure `authCmd` with
      `register`, `login`, `logout`, `forget`, `status` subcommands in `cmd/root.go`
      - test: `go build ./...` → clean
- [ ] Implement `LogoutTokens`, `RunForget`, `PrintAuthStatus`, `CheckAuthState` as exported
      functions in `cmd/root.go`
      - test: `TestAuthForgetCmd|TestAuthStatusCmd|TestCheckAuthState` → PASS
- [ ] Update `RunAuthFlow` to use `cfg.CallbackPort` instead of `0`
      - test: `go build ./...` clean
- [ ] Update `runApp` to call `CheckAuthState` and start the callback server early with
      `cfg.CallbackPort`
      - test: `go build ./...` clean
- [ ] Search for and remove all `-X cmd.spotifyClientID` ldflags from `.goreleaser.yaml`,
      `Makefile`, and any GitHub Actions workflow files
      - test: `grep -r "spotifyClientID" . --include="*.yaml" --include="*.yml" --include="Makefile"` → no matches
- [ ] `make ci` passes
