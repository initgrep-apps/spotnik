---
title: "Config — CallbackPort, ClearClientID, SetClientID"
feature: 09-auth-and-profile
status: open
---

## Background

The onboarding redesign replaces the embedded ldflags client ID with a config-first model. Three
config-layer changes are required before any TUI or CLI work can happen:

1. **`CallbackPort`** — the OAuth callback server now binds to a fixed configurable port (default
   8888) so the redirect URI `http://127.0.0.1:8888/callback` is stable across launches. Users
   register it once in their Spotify Developer app and never touch it again.

2. **`ClearClientID()`** — used by `spotnik auth forget` and the profile overlay forget action.
   Removes `client_id` from `config.toml` while preserving all other settings (preferences,
   `callback_port`, etc.).

3. **`SetClientID()`** — used by `spotnik auth register` and the onboarding TUI registration
   step. Writes or updates `client_id` in `config.toml`.

**Depends on:** nothing — pure config layer, no TUI or CLI changes.

## Design

### `internal/config/config.go`

**`spotifyConfig` struct** — add `CallbackPort`:

```go
type spotifyConfig struct {
    ClientID     string `toml:"client_id"`
    CallbackPort int    `toml:"callback_port"`
}
```

**`Config` struct** — add `CallbackPort`:

```go
type Config struct {
    ClientID     string
    CallbackPort int
    Preferences  PreferencesConfig `toml:"preferences"`
}
```

**`Default()`** — set `CallbackPort: 8888`:

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

**`Load()`** — apply `CallbackPort` after parsing; only override default when value > 0:

```go
cfg.ClientID = raw.Spotify.ClientID
if raw.Spotify.CallbackPort > 0 {
    cfg.CallbackPort = raw.Spotify.CallbackPort
}
```

**`ClearClientID(path string) error`** — reads the TOML file line by line, removes any line
whose trimmed content starts with `client_id`, and writes the result back. The `[spotify]`
section header and all other keys are preserved.

```go
func ClearClientID(path string) error {
    data, err := os.ReadFile(path)
    if err != nil {
        return fmt.Errorf("reading config for client ID removal: %w", err)
    }
    var out []string
    for _, line := range strings.Split(string(data), "\n") {
        if strings.HasPrefix(strings.TrimSpace(line), "client_id") {
            continue
        }
        out = append(out, line)
    }
    if err := os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o600); err != nil {
        return fmt.Errorf("writing config after client ID removal: %w", err)
    }
    return nil
}
```

**`SetClientID(path string, clientID string) error`** — three cases:
- `client_id` line already exists → replace it in-place
- `[spotify]` section exists but no `client_id` → insert after the `[spotify]` header
- Neither exists → append `\n[spotify]\nclient_id = "..."\n`

If the file does not exist, create it. Ensures `~/.config/spotnik/` directory exists before
writing.

Add `"strings"` and `"errors"` to `config.go` imports as needed.

### Tests — `internal/config/config_test.go`

Write **failing** tests first (TDD):

```go
func TestLoad_CallbackPort_defaultsTo8888(t *testing.T) { ... }
func TestLoad_CallbackPort_fromFile(t *testing.T) { ... }
func TestClearClientID_removesClientIDPreservesPreferences(t *testing.T) { ... }
func TestClearClientID_noClientID_isNoOp(t *testing.T) { ... }
func TestClearClientID_missingFile_returnsError(t *testing.T) { ... }
func TestSetClientID_writesNewValue(t *testing.T) { ... }
func TestSetClientID_replacesExistingValue(t *testing.T) { ... }
```

Test `TestLoad_CallbackPort_defaultsTo8888`: write a config with `client_id` but no
`callback_port`; assert `cfg.CallbackPort == 8888`.

Test `TestLoad_CallbackPort_fromFile`: write `callback_port = 9999`; assert
`cfg.CallbackPort == 9999`.

Test `TestClearClientID_removesClientIDPreservesPreferences`: config contains `client_id`,
`callback_port = 9000`, and `[preferences] theme = "nord"`. After `ClearClientID`: `cfg.ClientID
== ""`, `cfg.CallbackPort == 9000`, `cfg.Preferences.Theme == "nord"`.

Test `TestClearClientID_noClientID_isNoOp`: config has no `client_id`; `ClearClientID` returns
`nil`; file is unchanged.

Test `TestClearClientID_missingFile_returnsError`: path does not exist; assert error.

Test `TestSetClientID_writesNewValue`: config has `[spotify]` section but no `client_id`;
after `SetClientID` the loaded config has the new value and other settings are preserved.

Test `TestSetClientID_replacesExistingValue`: config has `client_id = "old"`; after
`SetClientID("new")` the loaded config has `"new"`.

## Acceptance Criteria

- [ ] `spotifyConfig` struct has `CallbackPort int `toml:"callback_port"``
- [ ] `Config` struct has `CallbackPort int`
- [ ] `Default()` returns `CallbackPort: 8888`
- [ ] `Load()` applies `callback_port` from file; defaults to 8888 when absent or zero
- [ ] `ClearClientID` removes `client_id` line, preserves `[spotify]` section and preferences
- [ ] `ClearClientID` returns error when file does not exist
- [ ] `SetClientID` inserts or replaces `client_id` without touching other keys
- [ ] `SetClientID` creates the config file and directory if they do not exist
- [ ] All 7 tests pass; `make ci` passes

## Tasks

- [ ] Write failing tests in `internal/config/config_test.go` for `CallbackPort`,
      `ClearClientID`, and `SetClientID`
      - test: `go test ./internal/config/... -run "TestLoad_CallbackPort|TestClearClientID|TestSetClientID" -v`
        → compile error (functions undefined)
- [ ] Add `CallbackPort` to `spotifyConfig`, `Config`, `Default()`, and `Load()` in
      `internal/config/config.go`
      - test: `TestLoad_CallbackPort_*` → PASS
- [ ] Implement `ClearClientID(path string) error` in `internal/config/config.go`
      - test: `TestClearClientID_*` → PASS
- [ ] Implement `SetClientID(path string, clientID string) error` in `internal/config/config.go`
      - test: `TestSetClientID_*` → PASS
- [ ] `make ci` passes
