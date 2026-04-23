---
title: "[cli] palette config — auto/fixed/theme resolution at CLI startup"
feature: 12-cli-output
status: open
---

## Background

Story 146 built `cliout.resolve()` and `cliout.Use()` but no caller feeds them.
`cliout.current()` always returns `Fixed`.

This story:

1. Adds a new `[cli]` section to `internal/config/config.go` with a `Palette`
   string field.
2. Wires palette resolution at CLI startup: `cmd/root.go` loads config, resolves
   mode + TTY + NO_COLOR, optionally loads the active TUI theme, calls
   `cliout.Use(resolved)`.
3. Registers a config validator (similar to `ThemeValidator`) that clamps
   unknown values to `"auto"` on load with a warning.
4. Updates `config.Bootstrap` so new config files contain the `[cli]` section
   with the default.

**Depends on:** Stories 146 (package), 147 (migration complete so palette
actually drives output).

## Design

### `internal/config/config.go` — new type + field

```go
// CLIConfig holds CLI-specific settings. Currently only palette mode.
type CLIConfig struct {
    Palette string `toml:"palette"` // "auto" | "fixed" | "theme"
}

// Config adds a CLI field.
type Config struct {
    // ... existing fields ...
    CLI CLIConfig `toml:"cli"`
}

// Default adds the CLI default.
func Default() *Config {
    return &Config{
        // ... existing defaults ...
        CLI: CLIConfig{Palette: "auto"},
    }
}

// Valid palette modes.
var validPalettes = map[string]bool{
    "auto":  true,
    "fixed": true,
    "theme": true,
}

// Load validates cfg.CLI.Palette after TOML decode. Unknown values are clamped
// to "auto" with a warning printed to stderr.
func Load(path string) (*Config, error) {
    // ... existing TOML decode ...
    if !validPalettes[cfg.CLI.Palette] {
        _, _ = fmt.Fprintf(os.Stderr, "config: invalid cli.palette %q — using \"auto\"\n", cfg.CLI.Palette)
        cfg.CLI.Palette = "auto"
    }
    return cfg, nil
}
```

Empty string after decode (field absent or explicit `""`) also clamps to `"auto"`
(same warning suppressed for empty since that's the "not-set" case):

```go
if cfg.CLI.Palette == "" {
    cfg.CLI.Palette = "auto"
} else if !validPalettes[cfg.CLI.Palette] {
    _, _ = fmt.Fprintf(os.Stderr, "config: invalid cli.palette %q — using \"auto\"\n", cfg.CLI.Palette)
    cfg.CLI.Palette = "auto"
}
```

### `internal/config/config.go` — default config template

The `Bootstrap` function writes the `defaultTemplate` string (top of
`internal/config/config.go`, not a separate `bootstrap.go`) when the config
file doesn't exist. Append the `[cli]` section to that constant:

```go
const defaultTemplate = `# Spotnik configuration
# ...existing sections unchanged...

[cli]
# CLI palette: "auto" (default), "fixed", or "theme"
# - auto:  theme colours on dark-bg terminals, fixed elsewhere
# - fixed: always the built-in Spotnik palette
# - theme: inherit the TUI theme (may be unreadable on light terminals)
palette = "auto"
`
```

**Backward compatibility:** existing config files without `[cli]` get
`cfg.CLI.Palette = ""` after decode, which `Load` clamps to `"auto"`. Their
files are not rewritten — migration is lazy. If a user edits the file, they can
add the section manually following the snippet printed above.

### `cmd/root.go` — resolution at startup

Add a new function in `cmd/root.go` (next to `loadConfigFromPath`):

```go
// resolveCLIPalette converts cfg.CLI.Palette + runtime environment into a
// cliout.Palette and installs it via cliout.Use. Must be called once before any
// user-facing output.
func resolveCLIPalette(cfg *config.Config, w io.Writer) {
    mode := cliout.ModeAuto
    switch cfg.CLI.Palette {
    case "fixed":
        mode = cliout.ModeFixed
    case "theme":
        mode = cliout.ModeTheme
    case "auto":
        mode = cliout.ModeAuto
    }

    var activeTheme theme.Theme
    if mode == cliout.ModeAuto || mode == cliout.ModeTheme {
        // theme.Load never errors — it falls back to the default theme internally.
        // Preferences.Theme is the canonical config key for the active theme.
        activeTheme = theme.Load(cfg.Preferences.Theme)
    }

    isTTY := cliout.IsTTY(w) // exported helper — see below
    noColor := os.Getenv("NO_COLOR") != ""

    cliout.Use(cliout.Resolve(mode, isTTY, noColor, activeTheme))
}
```

Call from `runApp`, `runRegister`, `runAuthLogin`, and inside `Execute()`
fallback — anywhere user-facing output happens. Simplest: call once at the top
of `Execute()` using a zero-arg variant that loads config internally:

```go
func Execute(version string) {
    appVersion = version
    rootCmd.Version = version

    // Resolve CLI palette once, using stderr as the TTY reference
    // (error fallback writes to stderr; subcommands decide stdout/stderr individually).
    if cfg, err := loadConfig(); err == nil {
        resolveCLIPalette(cfg, os.Stderr)
    }

    if err := rootCmd.Execute(); err != nil {
        if !errors.Is(err, errAlreadyPrinted) {
            cliout.Write(os.Stderr, cliout.Step{Status: cliout.StatusFailure, Text: err.Error()})
        }
        os.Exit(1)
    }
}
```

**Gotcha:** `loadConfig()` triggers `config.Bootstrap` — it's safe to call
unconditionally. The `err` branch (config file unreadable) keeps `cliout`
defaulting to `Fixed` since `Use` is never called.

### Config layout note

The existing `Config` struct puts user-facing preferences under
`cfg.Preferences` (theme, preset, visualizer). The new `CLI` section stays
top-level (not under Preferences) because palette resolution is a runtime
concern, not a user preference in the same sense. `cfg.CLI.Palette` is the
access path; `cfg.Preferences.Theme` is still the access path for the theme
ID consumed by `theme.Load`.

### `internal/cliout` — export `IsTTY` and `Resolve`

Story 146 kept `isTTY` and `resolve` unexported. This story exports them for
`cmd/` consumption:

```go
// IsTTY returns whether w is *os.File pointing at a terminal.
func IsTTY(w io.Writer) bool { return isTTY(w) }

// Resolve picks a Palette given mode, TTY status, NO_COLOR, and theme.
// Exported so cmd/ can compose the resolution.
func Resolve(mode PaletteMode, isTTY bool, noColor bool, t theme.Theme) Palette {
    return resolve(mode, isTTY, noColor, t)
}
```

Rename internal `isTTY` → `detectTTY` if a clash with the exported wrapper is
too confusing; or keep the shim as shown.

### Tests

#### `internal/config/config_test.go`

```go
func TestLoad_defaultCLIPaletteIsAuto(t *testing.T) {
    // Write a config that omits [cli]. Expect cfg.CLI.Palette == "auto".
    path := writeTempConfig(t, "theme = \"black\"\n")
    cfg, err := config.Load(path)
    require.NoError(t, err)
    assert.Equal(t, "auto", cfg.CLI.Palette)
}

func TestLoad_validCLIPaletteValues(t *testing.T) {
    for _, v := range []string{"auto", "fixed", "theme"} {
        path := writeTempConfig(t, fmt.Sprintf("[cli]\npalette = %q\n", v))
        cfg, err := config.Load(path)
        require.NoError(t, err)
        assert.Equal(t, v, cfg.CLI.Palette)
    }
}

func TestLoad_invalidCLIPaletteClampsToAuto(t *testing.T) {
    path := writeTempConfig(t, "[cli]\npalette = \"neon\"\n")
    cfg, err := config.Load(path)
    require.NoError(t, err)
    assert.Equal(t, "auto", cfg.CLI.Palette)
}

func TestDefault_CLIPaletteAuto(t *testing.T) {
    cfg := config.Default()
    assert.Equal(t, "auto", cfg.CLI.Palette)
}

func TestBootstrap_writesCLISection(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "config.toml")
    require.NoError(t, config.Bootstrap(path))
    body, err := os.ReadFile(path)
    require.NoError(t, err)
    assert.Contains(t, string(body), "[cli]")
    assert.Contains(t, string(body), `palette = "auto"`)
}
```

#### `cmd/root_test.go`

```go
func TestResolveCLIPalette_fixedMode_usesFixed(t *testing.T) {
    cfg := config.Default()
    cfg.CLI.Palette = "fixed"
    var buf bytes.Buffer
    resolveCLIPalette(cfg, &buf) // buf is not a TTY
    // No direct way to read the installed palette back; verify via a render round-trip:
    got := cliout.Capture(func(w io.Writer) {
        cliout.Write(w, cliout.Step{Status: cliout.StatusSuccess, Text: "ok"})
    })
    // Palette is captured indirectly — Step struct doesn't carry a colour.
    // Assert Use was called with Fixed via a new test hook:
    assert.Equal(t, cliout.Fixed, cliout.CurrentForTest())
}

func TestResolveCLIPalette_themeMode_usesThemeTokens(t *testing.T) {
    cfg := config.Default()
    cfg.Preferences.Theme = "black"
    cfg.CLI.Palette = "theme"
    var buf bytes.Buffer
    resolveCLIPalette(cfg, &buf)
    // Expect theme.Load("black").Accent() to match current palette Accent.
    th := theme.Load("black")
    assert.Equal(t, th.Accent(), cliout.CurrentForTest().Accent)
}

func TestResolveCLIPalette_autoMode_nonTTY_usesFixed(t *testing.T) {
    cfg := config.Default()
    cfg.CLI.Palette = "auto"
    var buf bytes.Buffer // non-TTY
    resolveCLIPalette(cfg, &buf)
    assert.Equal(t, cliout.Fixed, cliout.CurrentForTest())
}
```

Add the test hook to `internal/cliout/testing.go`:

```go
// CurrentForTest returns the active palette. Test-only helper.
func CurrentForTest() Palette { return current() }
```

### Interaction with golden files from Story 147

Golden files were generated under `SetTestMode(true)` which pins `termenv.Ascii`.
The palette resolution path doesn't affect ANSI output in test mode — palette
choice picks colours, but test mode strips them. Golden files keep passing.

To add a golden for "theme-palette output", a separate test mode without ASCII
pinning would be needed. Out of scope — Story 147 goldens are enough.

## Acceptance Criteria

- [ ] `config.Config` has `CLI CLIConfig` field with `Palette string` tag `toml:"cli"`/`toml:"palette"`
- [ ] `config.Default().CLI.Palette == "auto"`
- [ ] `config.Load` clamps empty or invalid `CLI.Palette` to `"auto"` (warning
      printed to stderr for non-empty invalid values)
- [ ] `config.Bootstrap` writes a `[cli]` section with `palette = "auto"` into
      new config files
- [ ] `cliout.IsTTY(w io.Writer) bool` exported
- [ ] `cliout.Resolve(mode, isTTY, noColor, theme) Palette` exported
- [ ] `cliout.CurrentForTest() Palette` exported (test helper)
- [ ] `cmd.resolveCLIPalette(cfg, w)` converts config + runtime into a palette
      and calls `cliout.Use`
- [ ] `cmd.Execute` calls `resolveCLIPalette` before running the root command
- [ ] `palette = "fixed"` in config forces the built-in palette regardless of
      TTY / theme
- [ ] `palette = "theme"` uses the TUI theme's tokens (caveat: user's
      responsibility if unreadable on light terminals)
- [ ] `palette = "auto"` uses theme tokens only when TTY + dark bg; fixed otherwise
- [ ] `NO_COLOR=1 spotnik auth status` prints no ANSI escapes
- [ ] All existing tests continue to pass
- [ ] `make ci` passes

## Tasks

- [ ] Add `CLIConfig` type + `CLI` field to `Config` in `internal/config/config.go`;
      add `"auto"` to `Default()` output
      - test: `go build ./internal/config/...` → clean

- [ ] Add validation to `config.Load`: empty → `"auto"` (no warning);
      invalid non-empty → `"auto"` + stderr warning
      - test: `go build ./internal/config/...` → clean

- [ ] Write `TestLoad_defaultCLIPaletteIsAuto`,
      `TestLoad_validCLIPaletteValues`,
      `TestLoad_invalidCLIPaletteClampsToAuto`, `TestDefault_CLIPaletteAuto` in
      `internal/config/config_test.go`
      - test: `go test ./internal/config/... -run TestLoad_.*CLIPalette -v` → PASS;
        `go test ./internal/config/... -run TestDefault_CLIPaletteAuto -v` → PASS

- [ ] Update `config.Bootstrap` default TOML content to include the `[cli]`
      section with `palette = "auto"` and the explanatory comment block
      - test: `go build ./...` → clean

- [ ] Write `TestBootstrap_writesCLISection`
      - test: `go test ./internal/config/... -run TestBootstrap_writesCLISection -v` → PASS

- [ ] Export `cliout.IsTTY`, `cliout.Resolve`, `cliout.CurrentForTest` in
      `internal/cliout/`
      - test: `go build ./internal/cliout/...` → clean

- [ ] Add `resolveCLIPalette` function in `cmd/root.go`
      - test: `go build ./cmd/...` → clean

- [ ] Call `resolveCLIPalette` near the top of `Execute()`; wrap in `if cfg, err :=
      loadConfig(); err == nil` so unreadable config files don't panic
      - test: `go build ./...` → clean

- [ ] Write the three `TestResolveCLIPalette_*` tests in `cmd/root_test.go`
      - test: `go test ./cmd/... -run TestResolveCLIPalette -v` → PASS

- [ ] Run existing `TestGolden_*` tests — must still pass unchanged (SetTestMode
      pins ASCII so palette choice doesn't alter output)
      - test: `go test ./cmd/... -run TestGolden_ -v` → PASS

- [ ] Manual TTY check: `NO_COLOR=1 bin/spotnik auth status` → plain text, no
      ANSI escapes; `bin/spotnik auth status` on a dark terminal → coloured
      output; editing config to `palette = "fixed"` → matches story-145 colours
      - test: visual confirmation on terminal

- [ ] `make ci` → PASS
