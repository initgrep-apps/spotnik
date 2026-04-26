---
title: "Fix: glyph config — additive Bootstrap + wire uikit.Use at startup"
feature: 13-tui-design-system
status: done
---

## Background

Setting `glyphs = "ascii"` (or any value) in `config.toml` has no effect. Two
gaps combine to cause this:

**Gap 1 — `uikit.Use()` never called.** `uikit/config.go` exposes `Use(cfg string)`
which resolves and caches the `GlyphMode` via `sync.Once`. Every primitive calls
`ActiveMode()` to read the cached value. But `Use()` is never called at app startup,
so `activeMode` stays at its Go zero value — `GlyphMode(0)` = `GlyphUnicode` — and
the config setting is silently ignored.

**Gap 2 — Bootstrap is write-once (new files only).** `config.Bootstrap()` creates
the config file from a full template if it does not exist, but is a no-op when the
file already exists. Existing users who had a `config.toml` before feature 13 shipped
will not have a `[ui]` section, so `cfg.UI.Glyphs` decodes as `""`. `Validate()`
currently rejects `""` with a hard error, breaking startup for all pre-feature-13
users on upgrade.

**Fix strategy:** Make Bootstrap additive — on each startup it patches any missing
sections into existing files. This ensures every user's config has `glyphs = "auto"`
after their first run on the new version. Then wire `uikit.Use(cfg.UI.Glyphs)` in
`runApp` so the loaded value is activated before any TUI rendering. Accept `""` in
`Validate()` as a safety net for edge cases (hand-edited configs, test fixtures).

**Files:** `internal/config/config.go`, `cmd/root.go`

## Design

### Additive Bootstrap — `internal/config/config.go`

Add a `sectionPatches` table listing known config sections that must be present.
`Bootstrap` retains its existing create-from-scratch path for new files, and gains
a new patch path for existing files:

```go
// sectionPatches lists sections that Bootstrap ensures are present in the config.
// When the file exists but is missing a section, Bootstrap appends the patch block.
// Add new entries here whenever a future feature introduces a new config section.
var sectionPatches = []struct {
    header string
    block  string
}{
    {
        header: "[ui]",
        block: `
[ui]
# Glyph rendering mode: "auto" (default), "unicode", or "ascii"
# - auto:    use unicode glyphs when LC_ALL/LANG contains UTF-8, else ASCII
# - unicode: always use unicode glyphs (requires a UTF-8 capable terminal)
# - ascii:   always use ASCII fallback glyphs (safe for all terminals)
glyphs = "auto"
`,
    },
}

func Bootstrap(path string) error {
    if _, statErr := os.Stat(path); errors.Is(statErr, os.ErrNotExist) {
        // New file — write the full template (existing behaviour).
        if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
            return fmt.Errorf("creating config directory: %w", err)
        }
        if err := os.WriteFile(path, []byte(defaultTemplate), 0o600); err != nil {
            return fmt.Errorf("writing config template: %w", err)
        }
        return nil
    } else if statErr != nil {
        return fmt.Errorf("checking config file: %w", statErr)
    }

    // File exists — append any missing sections.
    data, err := os.ReadFile(path)
    if err != nil {
        return fmt.Errorf("reading config for patch: %w", err)
    }
    content := string(data)

    var additions strings.Builder
    for _, patch := range sectionPatches {
        if !strings.Contains(content, patch.header) {
            additions.WriteString(patch.block)
        }
    }
    if additions.Len() == 0 {
        return nil
    }

    patched := content
    if len(patched) > 0 && patched[len(patched)-1] != '\n' {
        patched += "\n"
    }
    patched += additions.String()
    return os.WriteFile(path, []byte(patched), 0o600)
}
```

### `Validate()` — accept `""` as `"auto"`

```go
func (c *UIConfig) Validate() error {
    switch strings.ToLower(strings.TrimSpace(c.Glyphs)) {
    case "auto", "unicode", "ascii", "": // "" treated as "auto" for hand-edited or legacy configs
        return nil
    default:
        return fmt.Errorf("ui.glyphs must be one of auto|unicode|ascii, got %q", c.Glyphs)
    }
}
```

### Wire `uikit.Use` — `cmd/root.go:runApp`

After `loadConfig()` succeeds and before `app.New()`:

```go
func runApp(_ *cobra.Command, _ []string) error {
    cfg, err := loadConfig()
    if err != nil {
        return err
    }

    uikit.Use(cfg.UI.Glyphs) // activate glyph mode before any TUI rendering

    store := keychain.NewKeychainTokenStore()
    // ...
}
```

Add `"github.com/initgrep-apps/spotnik/internal/uikit"` to `cmd/root.go` imports.

## Acceptance Criteria

- [ ] `config.Bootstrap()` for a **new** file: behaviour unchanged — writes the full
      `defaultTemplate` (including `[ui]` section)
- [ ] `config.Bootstrap()` for an **existing file without `[ui]`**: appends the `[ui]`
      section block; subsequent `config.Load()` returns `cfg.UI.Glyphs == "auto"`
- [ ] `config.Bootstrap()` for an **existing file with `[ui]`**: no-op (file unchanged)
- [ ] `sectionPatches` is a package-level var — new config sections added in future
      features need only a new entry here
- [ ] `config.UIConfig.Validate()` accepts `""` without error (treated as `"auto"`)
- [ ] `cmd/root.go:runApp` calls `uikit.Use(cfg.UI.Glyphs)` after `loadConfig()`,
      before `app.New()`
- [ ] Setting `glyphs = "ascii"` in `config.toml` causes `uikit.ActiveMode()` to
      return `GlyphASCII` during the TUI session
- [ ] `config_test.go` — new test `TestBootstrap_AppendsUISectionToExistingFile`:
      write a minimal config without `[ui]`, call `Bootstrap`, assert `[ui]` and
      `glyphs = "auto"` appear in the file; original content is preserved
- [ ] `config_test.go` — new test `TestBootstrap_NoopWhenUISectionPresent`:
      write a config that already has `[ui]`, call `Bootstrap`, assert file is
      unchanged (compare byte-for-byte)
- [ ] `config_test.go` — existing `TestValidate_UIConfig` extended: assert `""` is
      valid, assert unknown value `"emoji"` is still an error
- [ ] `make ci` → PASS

## Tasks

- [ ] Branch: `fix/13-glyph-config-wiring`
- [ ] Write failing `TestBootstrap_AppendsUISectionToExistingFile` → FAIL
- [ ] Write failing `TestBootstrap_NoopWhenUISectionPresent` → FAIL
- [ ] Add `sectionPatches` var and rewrite `Bootstrap` in `config.go` → both tests PASS
- [ ] Extend `TestValidate_UIConfig` for `""` and `"emoji"` cases → PASS
- [ ] Accept `""` in `Validate()` → PASS
- [ ] Add `uikit.Use(cfg.UI.Glyphs)` to `cmd/root.go:runApp`; add import
- [ ] `make ci` → PASS
- [ ] Commit + push + open PR
