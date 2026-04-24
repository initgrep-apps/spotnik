---
title: "internal/uikit scaffold — glyph catalogue, role matrix, config, capture helper"
feature: 13-tui-design-system
status: open
---

## Background

Gate story for the TUI design system migration. Creates the `internal/uikit` package
with the frozen glyph catalogue, the emphasis-role → theme-token matrix, the
`GlyphMode` config knob with env-based auto-detection, and the `Capture` test helper.
No primitives yet — those arrive in S3–S17 — but every subsequent story depends on
this scaffold existing.

Adds a single new config field `[ui] glyphs = "auto" | "unicode" | "ascii"` (default
`"auto"`) that the package reads once via `sync.Once` on first use.

**Depends on:** nothing. Design record §3, §5, §6, §8.2 of
`docs/superpowers/specs/2026-04-24-tui-design-system-design.md`. Full step-by-step
TDD guide: Task 1 (S1) in `docs/superpowers/plans/2026-04-24-tui-design-system.md`.

## Design

### Package layout

```
internal/uikit/
├── doc.go          — package doc pointing at docs/TUI-DESIGN-SYSTEM.md and the spec
├── glyph.go        — GlyphRole constants, glyphTable map, GlyphFor, GlyphWidth, AllGlyphRoles
├── glyph_test.go   — catalogue integrity: both forms present, banned ᐅ/⚠ absent,
│                     ascii is pure ASCII, corners rounded only
├── role.go         — Role constants, ColourFor(role, theme), Apply(role, style, theme)
├── role_test.go    — every role resolves to a non-empty colour; Accent/Plain/Muted
│                     map to the expected theme token
├── config.go       — GlyphMode enum, ActiveMode(), SetModeForTest, Resolve(cfgValue)
│                     with LANG/LC_ALL auto-detection under sync.Once
├── config_test.go  — auto-detect paths for "UTF-8", "utf8", empty locale
├── capture.go      — Capture(fn func() string) []string — strips ANSI and splits
├── capture_test.go — ANSI strip correctness + line splitting on real styled output
```

### Glyph catalogue (frozen)

Every glyph the TUI uses, with unicode + ascii forms. Table matches §5 of the design
record exactly. Banned glyphs — `ᐅ` (U+1405), `⚠` (U+26A0), sharp corners
`┌┐└┘`, double corners `╔╗╚╝`, `✅❌❗` — do not appear and are asserted absent by
`TestGlyph_ActionPrefixIsBanned` and `TestGlyph_WarningIsCirclePlusInsideTriangle`.

Categories: structural/borders, intent/feedback, state/availability,
navigation/scroll, playback controls, domain/music, graphical fills, spinner frames,
keyboard chords, superscripts. See design record §5 for the full table and the plan's
Task 1 Step 1.3 for the exact Go literal.

### Role matrix

10 roles from §6.1: `Accent`, `Strong`, `Plain`, `Muted`, `Success`, `Error`,
`Warning`, `Info`, `Selection`, plus per-pane `PaneBorder-<ID>` (looked up by pane
ID). `ColourFor(role, theme)` returns a `lipgloss.TerminalColor`. `Strong` is bold
(weight, not colour) — `Apply(RoleStrong, style, theme)` sets `.Bold(true)` without
overriding the foreground. `Accent` resolves to `theme.Accent()` with `SeekBar`
fallback (already implemented by Story 146 when Feature 12 shipped).

### Glyph mode resolution

```go
type GlyphMode int

const (
    GlyphUnicode GlyphMode = iota
    GlyphASCII
)

func Resolve(cfgValue string) GlyphMode {
    switch cfgValue {
    case "unicode":
        return GlyphUnicode
    case "ascii":
        return GlyphASCII
    }
    // "auto" or unknown — inspect env.
    for _, v := range []string{os.Getenv("LC_ALL"), os.Getenv("LANG")} {
        if strings.Contains(strings.ToLower(v), "utf-8") ||
           strings.Contains(strings.ToLower(v), "utf8") {
            return GlyphUnicode
        }
    }
    return GlyphASCII
}
```

`ActiveMode()` caches via `sync.Once`; `SetModeForTest(mode)` overrides for tests.

### Capture helper

```go
// Capture runs fn and returns the ANSI-stripped output split on "\n".
// Used by primitive snapshot tests: assert lines[0] == expectedTopLine, etc.
func Capture(fn func() string) []string
```

### Config wiring

Add to `internal/config/config.go`:

```go
type UIConfig struct {
    Theme   string `toml:"theme"`
    Glyphs  string `toml:"glyphs"` // NEW — "auto" | "unicode" | "ascii"
    // ... existing fields ...
}
```

Default `Glyphs = "auto"`. Validation rejects values outside the three legal strings.
`config.Bootstrap` writes `glyphs = "auto"` into new config files.

### Register feature row

Add to `docs/spec/00-overview.md`:

```
| 13 | TUI Design System | features/13-tui-design-system/ | in-progress | 150–168 | internal/uikit primitives, frozen glyph catalogue w/ ascii fallback, role-to-token matrix, Toast/Spinner/Panel/etc. |
```

## Acceptance Criteria

- [ ] `internal/uikit/doc.go` exists with package doc pointing at
      `docs/TUI-DESIGN-SYSTEM.md` and the design record
- [ ] `internal/uikit/glyph.go` defines `GlyphRole`, `GlyphMode`, the glyph constants
      from §5, `glyphTable`, `GlyphFor`, `AllGlyphRoles`, `GlyphWidth`
- [ ] `glyph_test.go` covers: all roles have both forms; ascii form is pure ASCII;
      banned `ᐅ` / `⚠` / `┌┐└┘` / `╔╗╚╝` absent; `GlyphWarning` unicode is `◬`,
      ascii is `!`; rounded corners `╭╮╰╯` present
- [ ] `internal/uikit/role.go` defines 10 roles + pane-border lookup; `ColourFor`
      returns non-empty for every role against `theme.Load("black")`; `Apply` sets
      `.Bold(true)` for `RoleStrong` without overriding foreground
- [ ] `role_test.go` asserts Accent/Plain/Muted map to the expected theme method
- [ ] `internal/uikit/config.go` defines `GlyphMode`, `Resolve`, `ActiveMode` (via
      `sync.Once`), `SetModeForTest` (for test override)
- [ ] `config_test.go` covers: `"unicode"` → `GlyphUnicode`; `"ascii"` → `GlyphASCII`;
      `"auto"` + `LANG=en_US.UTF-8` → `GlyphUnicode`; `"auto"` + empty env →
      `GlyphASCII`; unknown value falls through to auto-detect
- [ ] `internal/uikit/capture.go` strips ANSI escapes and returns `[]string` split
      on `"\n"`; empty input returns `[]string{}`
- [ ] `capture_test.go` round-trips styled input → stripped lines
- [ ] `internal/config.UIConfig` has a `Glyphs string` field (TOML `glyphs`);
      default `"auto"`; validation rejects values not in {`auto`,`unicode`,`ascii`}
- [ ] `config.Bootstrap` writes `glyphs = "auto"` into freshly created config files
- [ ] `docs/spec/features/13-tui-design-system/feature.md` exists (done)
- [ ] `docs/spec/00-overview.md` has row 13 for this feature
- [ ] `internal/uikit` coverage = 100%
- [ ] `make ci` → PASS

## Tasks

Step-by-step TDD guide: Task 1 (S1) in
`docs/superpowers/plans/2026-04-24-tui-design-system.md`.

- [ ] Branch: `feat/13-tui-design-system-scaffold`
- [ ] Write failing `glyph_test.go` (Step 1.2) → compile error
- [ ] Implement `internal/uikit/glyph.go` (Step 1.3) → glyph tests PASS
- [ ] Write failing `role_test.go` (Step 1.5) → compile error
- [ ] Implement `internal/uikit/role.go` (Step 1.6) → role tests PASS
- [ ] Write failing `config_test.go` (Step 1.7) → compile error
- [ ] Implement `internal/uikit/config.go` (Step 1.8) → config tests PASS
- [ ] Write `capture.go` + `capture_test.go` (Step 1.9) → capture tests PASS
- [ ] Extend `internal/config/config.go` with `UIConfig.Glyphs`; update
      `Bootstrap` to emit `glyphs = "auto"`; add validation (Step 1.10) →
      existing config tests still PASS, new glyphs-config tests PASS
- [ ] Create `internal/uikit/doc.go` with package doc comment (Step 1.11)
- [ ] Create `docs/spec/features/13-tui-design-system/feature.md` (done in this PR)
- [ ] Add row 13 to `docs/spec/00-overview.md` (Step 1.13)
- [ ] Verify `go test -cover ./internal/uikit/...` → 100%
- [ ] `make ci` → PASS
- [ ] Commit + push + open PR (Step 1.14)
