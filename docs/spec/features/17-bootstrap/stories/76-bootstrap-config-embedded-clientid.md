---
title: "Bootstrap Config & Embedded Client ID"
feature: 17-bootstrap
status: open
---

## Background

Currently Spotnik requires the user to manually create `~/.config/spotnik/config.toml`
with a `[spotify] client_id` before the app can start. This is a developer workflow —
the client ID is the app's identity with Spotify, not a user secret. It should ship
with the binary. The config file should be auto-created on first launch as a preferences
file, with a commented client_id placeholder for power users who register their own
Spotify app.

## Design

### Embedded Client ID

Add a package-level variable in `cmd/root.go` set via `ldflags`:

```go
// spotifyClientID is the Spotify application client ID, embedded at build time
// via -ldflags "-X github.com/initgrep-apps/spotnik/cmd.spotifyClientID=...".
// Users can override this by setting client_id in their config.toml.
var spotifyClientID string
```

### Client ID Resolution

In `loadConfig()`, after loading the config file:

```go
func loadConfig() (*config.Config, error) {
    path := config.DefaultConfigPath()

    // Bootstrap config file on first launch.
    if err := config.Bootstrap(path); err != nil {
        return nil, fmt.Errorf("bootstrapping config: %w", err)
    }

    cfg, err := config.Load(path)
    if err != nil {
        return nil, err
    }

    // Config client_id overrides embedded. Use embedded as fallback.
    if cfg.ClientID == "" {
        cfg.ClientID = spotifyClientID
    }

    // If still empty (no embedded, no config), show setup instructions.
    if cfg.ClientID == "" {
        printSetupInstructions()
        return nil, fmt.Errorf("no client_id available — see setup instructions above")
    }

    return cfg, nil
}
```

### config.Load() Change

Remove the `client_id` validation error. An empty `client_id` in config is no longer
an error — it means "use the embedded value." The caller handles the fallback.

```go
// Before:
if cfg.ClientID == "" {
    return nil, fmt.Errorf("missing client_id in [spotify] section...")
}

// After: just return the config, empty ClientID is fine
```

### Bootstrap Function

Add to `internal/config/config.go`:

```go
// Bootstrap creates the config file at path with a default template if it does
// not already exist. Creates the parent directory if needed. If the file already
// exists, Bootstrap is a no-op.
func Bootstrap(path string) error {
    if _, err := os.Stat(path); err == nil {
        return nil // file exists, nothing to do
    }

    // Create directory.
    if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
        return fmt.Errorf("creating config directory: %w", err)
    }

    // Write template.
    if err := os.WriteFile(path, []byte(defaultTemplate), 0o600); err != nil {
        return fmt.Errorf("writing config template: %w", err)
    }
    return nil
}
```

### Rename `[ui]` → `[preferences]`

The TOML section and Go types are renamed for clarity:

- TOML key: `[ui]` → `[preferences]`
- Go type: `UIConfig` → `PreferencesConfig`
- Go field: `Config.UI` → `Config.Preferences`

This touches `config.go`, `config_test.go`, and all files referencing `cfg.UI`
(app.go, root_test.go, various test files). Since there are no existing users
or config files in the wild, this is a clean rename with no migration needed.

### Default Template

```go
const defaultTemplate = `# Spotnik configuration
# https://github.com/initgrep-apps/spotnik

[spotify]
# To use your own Spotify app credentials, uncomment and set:
# client_id = "your-client-id-from-spotify-developer-dashboard"

[preferences]
theme = "black"
volume_step = 5
`
```

### Makefile Change

Add `SPOTIFY_CLIENT_ID` variable and pass via ldflags:

```makefile
SPOTIFY_CLIENT_ID ?=
LDFLAGS := -ldflags "-X github.com/initgrep-apps/spotnik/cmd.spotifyClientID=$(SPOTIFY_CLIENT_ID)"

build:
    go build $(LDFLAGS) -o bin/spotnik .
```

For local development without an embedded ID, the app falls back to config.toml.
CI/release builds inject the real client ID.

### PersistTheme Interaction

`PersistTheme()` uses `toml.DecodeFile` + `toml.Encoder.Encode` which strips
comments from the template. This is acceptable — the comments serve as first-launch
documentation. Once the user changes a theme, the file becomes a pure preferences
file.

After a theme change, the file will look like:

```toml
[spotify]

[preferences]
  theme = "nord"
  volume_step = 5
```

The commented client_id is gone, but it served its purpose. The `[spotify]` section
remains (empty) because PersistTheme preserves the full TOML structure.

### Files Changed

| Action | File | Purpose |
|---|---|---|
| Modify | `internal/config/config.go` | Rename `UIConfig`→`PreferencesConfig`, `UI`→`Preferences`, `[ui]`→`[preferences]`; add `Bootstrap()`, `defaultTemplate`; remove client_id validation error |
| Modify | `internal/config/config_test.go` | Update all `cfg.UI` → `cfg.Preferences`; add Bootstrap tests |
| Modify | `cmd/root.go` | Add `spotifyClientID` var, update `loadConfig()` with bootstrap + fallback |
| Modify | `cmd/root_test.go` | Update `cfg.UI` → `cfg.Preferences`; add client_id resolution tests |
| Modify | `internal/app/app.go` | Update all `cfg.UI` → `cfg.Preferences` references |
| Modify | `internal/app/*_test.go` | Update `cfg.UI` → `cfg.Preferences` in test helpers |
| Modify | `Makefile` | Add `SPOTIFY_CLIENT_ID`, update LDFLAGS |

## Acceptance Criteria

- [ ] `config.Bootstrap()` creates template file when none exists
- [ ] `config.Bootstrap()` is a no-op when file already exists
- [ ] `config.Bootstrap()` creates parent directory if missing
- [ ] `config.Load()` returns config with empty `ClientID` when not set (no error)
- [ ] `loadConfig()` uses embedded client_id when config has none
- [ ] `loadConfig()` uses config client_id when present (overrides embedded)
- [ ] `loadConfig()` errors only when both embedded and config are empty
- [ ] `PersistTheme()` works correctly with bootstrapped config file
- [ ] Template file has correct permissions (0600 file, 0750 dir)
- [ ] `make build SPOTIFY_CLIENT_ID=test123` embeds the value
- [ ] `make ci` passes

## Tasks

- [ ] Rename `UIConfig` → `PreferencesConfig`, `Config.UI` → `Config.Preferences`, TOML key `[ui]` → `[preferences]` in `internal/config/config.go`. Update `PersistTheme` internals to use `[preferences]`.
      - test: Update all existing config tests (`cfg.UI` → `cfg.Preferences`), verify TOML output uses `[preferences]`
- [ ] Update all `cfg.UI` references in `internal/app/app.go`, `cmd/root.go`, `cmd/root_test.go`, and `internal/app/*_test.go`
      - test: `make ci` passes after rename
- [ ] Add `defaultTemplate` constant and `Bootstrap()` function to `internal/config/config.go`
      - test: `TestBootstrap_CreatesFileWhenMissing`, `TestBootstrap_NoopWhenExists`, `TestBootstrap_CreatesDirectory`, `TestBootstrap_TemplateContent`
- [ ] Remove client_id validation error from `config.Load()` — empty client_id is valid
      - test: `TestLoad_EmptyClientID_NoError`, `TestLoad_WithClientID_StillWorks`
- [ ] Add `spotifyClientID` var to `cmd/root.go`, update `loadConfig()` with bootstrap call and embedded fallback
      - test: `TestLoadConfig_UsesEmbeddedWhenConfigEmpty`, `TestLoadConfig_ConfigOverridesEmbedded`, `TestLoadConfig_ErrorWhenBothEmpty`
- [ ] Update `Makefile` with `SPOTIFY_CLIENT_ID` variable and ldflags
      - test: `make build SPOTIFY_CLIENT_ID=test123` compiles successfully
- [ ] Verify `PersistTheme()` works with bootstrapped config (theme change round-trip)
      - test: `TestPersistTheme_WithBootstrappedConfig`
