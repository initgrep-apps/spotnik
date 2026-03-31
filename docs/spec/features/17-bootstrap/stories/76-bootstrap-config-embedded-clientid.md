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

This story also renames `[ui]` to `[preferences]` and adds new preference fields
(preset, visualizer) that will be wired into the PreferenceStore engine in story 80.

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

### Rename `[ui]` → `[preferences]` and Add New Fields

The TOML section and Go types are renamed for clarity:

- TOML key: `[ui]` → `[preferences]`
- Go type: `UIConfig` → `PreferencesConfig`
- Go field: `Config.UI` → `Config.Preferences`

New fields added to `PreferencesConfig`:

```go
type PreferencesConfig struct {
    Theme      string `toml:"theme"`
    VolumeStep int    `toml:"volume_step"`
    Preset     int    `toml:"preset"`     // Page A layout preset index
    Visualizer int    `toml:"visualizer"` // viz engine pattern index (0-6)
}
```

Since there are no existing users or config files in the wild, this is a clean rename
with no migration needed.

### Validation / Clamping on Load

After decoding TOML, `config.Load()` clamps preference values to valid ranges. This
handles manual edits with invalid values — the app never crashes, it just falls back
to defaults:

```go
// Clamp theme: if unknown, fall back to default.
if !theme.IsValidID(cfg.Preferences.Theme) {
    cfg.Preferences.Theme = "black"
}

// Clamp preset: valid range depends on the number of Page A presets.
// Config only stores the raw int — the layout package validates on SetPreset.
// Negative values are clamped here; out-of-range positive values are handled
// by layout.SetPreset() which silently ignores invalid indices.
if cfg.Preferences.Preset < 0 {
    cfg.Preferences.Preset = 0
}

// Clamp visualizer: valid range is 0 to PatternCount-1.
// We clamp to 0 here; the viz engine also ignores out-of-range values.
if cfg.Preferences.Visualizer < 0 {
    cfg.Preferences.Visualizer = 0
}
```

Note: upper-bound clamping for preset and visualizer cannot happen in the config
package because it doesn't know the total count of presets or patterns. Those packages
(`layout` and `viz`) already guard against out-of-range indices — `SetPreset` is a
no-op for invalid indices, and the viz engine wraps with modulo. The config layer
only needs to catch negatives and obviously wrong values.

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
# preset = 0          # Page A layout preset index (0-based)
# visualizer = 0      # Visualizer pattern index (0-6)
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

`PersistTheme()` is removed in story 79 when the PreferenceStore replaces it.
Until then, it continues to work — this story does not touch persistence logic
beyond the rename.

### Files Changed

| Action | File | Purpose |
|---|---|---|
| Modify | `internal/config/config.go` | Rename `UIConfig`→`PreferencesConfig`, `UI`→`Preferences`, `[ui]`→`[preferences]`; add `Preset`/`Visualizer` fields; add `Bootstrap()`, `defaultTemplate`; remove client_id validation error; add clamping |
| Modify | `internal/config/config_test.go` | Update all `cfg.UI` → `cfg.Preferences`; add Bootstrap tests; add clamping tests |
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
- [ ] `config.Load()` clamps negative `Preset` to 0
- [ ] `config.Load()` clamps negative `Visualizer` to 0
- [ ] `config.Load()` clamps unknown theme ID to `"black"`
- [ ] `loadConfig()` uses embedded client_id when config has none
- [ ] `loadConfig()` uses config client_id when present (overrides embedded)
- [ ] `loadConfig()` errors only when both embedded and config are empty
- [ ] `PersistTheme()` works correctly with bootstrapped config file
- [ ] Template file has correct permissions (0600 file, 0750 dir)
- [ ] `make build SPOTIFY_CLIENT_ID=test123` embeds the value
- [ ] `make ci` passes

## Tasks

- [ ] Rename `UIConfig` → `PreferencesConfig`, `Config.UI` → `Config.Preferences`, TOML key `[ui]` → `[preferences]` in `internal/config/config.go`. Add `Preset int` and `Visualizer int` fields to `PreferencesConfig`. Update `PersistTheme` internals to use `[preferences]`.
      - test: Update all existing config tests (`cfg.UI` → `cfg.Preferences`), verify TOML output uses `[preferences]`
- [ ] Add validation/clamping in `config.Load()`: clamp `Preset < 0` to 0, clamp `Visualizer < 0` to 0, clamp unknown theme to `"black"`
      - test: `TestLoad_NegativePreset_ClampsToZero`, `TestLoad_NegativeVisualizer_ClampsToZero`, `TestLoad_UnknownTheme_ClampsToBlack`, `TestLoad_ValidPreferences_Preserved`
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
