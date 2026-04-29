---
title: "cliout integration with uikit catalogue (statusGlyph, Hint, spinner, test rebaseline)"
feature: 13-tui-design-system
status: done
---

## Background

`internal/cliout` was shipped (feature 12) before `internal/uikit` had a stable catalogue
(feature 13). To avoid coupling the two while feature 13 was in flight, cliout maintains
its own hardcoded glyph set:

- `cliout/message.go:36–53` `statusGlyph()` returns hardcoded `◉ ◎ ✓ ✗ ◬ ◌` per
  `Status` value, never reads `uikit.ActiveMode()`.
- `cliout/message.go:176` `Hint.render()` writes a literal `→` for the arrow.
- `cliout/spinner.go:16` declares its own `spinnerFrames = ["⠋", ..., "⠏"]` array with
  no ASCII fallback — when `ui.glyphs = "ascii"` the cliout spinner renders mojibake on
  non-UTF-8 terminals.
- `cliout/tty.go` `pinASCII` controls the lipgloss colour profile only; it does not
  pin glyph mode. `SetTestMode(true)` therefore strips colour but leaves glyphs as
  unicode — every cliout test asserts unicode glyphs and "passes" while never exercising
  the ASCII fallback path.

Today the values match by accident; tomorrow's drift is a bug. Per audit §4.4, the
shared-catalogue rule (design-doc §7) is now compiler-enforced: `cliout` imports
`uikit` and resolves every glyph and spinner frame through `GlyphFor` /
`SpinnerFrames(mode)`. The dependency direction is one-way: `cliout` → `uikit`; `uikit`
never imports `cliout`.

This story does not change cliout's user-visible API — `Status` values, `Message`
shape, `Hint`, and `Spinner` constructors stay identical. Only the glyph resolution
path changes.

**Depends on:** story 183 (`uikit.SpinnerFrames` and the catalogue must be in place
before cliout can consume them).

**Plan tasks:** 2.1, 2.2, 2.3, 2.4, 2.5, 2.6 in
`docs/superpowers/plans/2026-04-29-glyph-fallback.md`.

**Files:** `internal/cliout/message.go`, `internal/cliout/message_test.go`,
`internal/cliout/spinner.go`, `internal/cliout/spinner_test.go`,
`internal/cliout/tty.go`, every `internal/cliout/*_test.go` that asserts unicode
glyphs.

## Design

### `Status` → `GlyphRole` map

Replace `statusGlyph()` (`message.go:36–53`):

```go
var statusGlyphRole = map[Status]uikit.GlyphRole{
    Active:        uikit.GlyphActive,
    Inactive:      uikit.GlyphInactive,
    StatusSuccess: uikit.GlyphSuccess,
    StatusFailure: uikit.GlyphError,
    StatusWarning: uikit.GlyphWarning,
    Pending:       uikit.GlyphLocked,
}

func statusGlyph(s Status) string {
    role, ok := statusGlyphRole[s]
    if !ok {
        return "?"
    }
    return uikit.GlyphFor(role, uikit.ActiveMode())
}
```

The map mirrors `Status` ↔ `Toast` intent semantics so cliout and uikit speak the same
glyph vocabulary.

### `Hint` arrow

`message.go:176` — replace the literal `→`:

```go
arrow := lipgloss.NewStyle().
    Foreground(p.Accent).Bold(true).
    Render(uikit.GlyphFor(uikit.GlyphInfo, uikit.ActiveMode()))
```

(`GlyphInfo` = `→` unicode / `>` ascii — the same role the toast `Info` intent uses.)

### Spinner frames

Drop the inline `spinnerFrames` array in `cliout/spinner.go:16`. Replace with a helper
that resolves on each spinner start:

```go
func resolveSpinnerFrames() []string {
    return uikit.SpinnerFrames(uikit.ActiveMode())
}
```

The `(*SpinnerHandle).run` goroutine snapshots the slice once at start so the animation
does not switch mid-flight.

### `SetTestMode` aligned with uikit

`internal/cliout/tty.go`:

```go
func SetTestMode(enabled bool) {
    testModeMu.Lock()
    defer testModeMu.Unlock()
    testMode = enabled
    if enabled {
        pinASCII()
        uikit.SetModeForTest(uikit.GlyphASCII)
    }
}
```

Add a doc comment to `pinASCII` clarifying it controls colour only — glyph mode is
sourced independently from `uikit.ActiveMode()`.

### Test rebaseline

After `SetTestMode(true)` pins uikit to ASCII, every existing cliout assertion against
unicode glyphs (`◉`, `◎`, `✓`, `✗`, `◬`, `◌`, `→`) under test mode will fail —
confirming the previous tests were masking the bug. Each failing assertion is rewritten
to resolve expected output through `uikit.GlyphFor`, so the test asserts the **role**,
not the codepoint, and runs correctly in either mode:

```go
expected := uikit.GlyphFor(uikit.GlyphActive, uikit.ActiveMode())
if !strings.Contains(out, expected) {
    t.Errorf("expected %q, got %q", expected, out)
}
```

Tests that intentionally validate one specific mode wrap themselves with
`uikit.SetModeForTest(uikit.GlyphASCII)` (or `GlyphUnicode`) plus a `defer` restore.

### Wiring sanity

`cmd/root.go` already calls `uikit.Use(cfg.UI.Glyphs)` before any TUI rendering (story
172). cliout entrypoints all run after `runApp` enters, so `ActiveMode()` is set by the
time cliout writes anything. No changes needed in `cmd/root.go`. Story 184 adds a
regression test (`TestStatusGlyph_HonoursUikitMode` in `cliout/message_test.go`) that
documents the dependency direction.

## Acceptance Criteria

- [ ] `internal/cliout/message.go` imports `internal/uikit`
- [ ] `cliout.statusGlyph(StatusSuccess)` returns `✓` in unicode mode and `+` in ASCII
      mode; `StatusFailure` returns `✗` / `x`; `StatusWarning` returns `◬` / `!`;
      `Pending` returns `◌` / `(r)`; `Active` returns `◉` / `(*)`; `Inactive` returns
      `◎` / `( )`
- [ ] `cliout.Hint.render()` arrow resolves through
      `uikit.GlyphFor(uikit.GlyphInfo, uikit.ActiveMode())`; ASCII output starts with
      `> `, never `→ `
- [ ] `internal/cliout/spinner.go` no longer declares `spinnerFrames`; frames source
      from `uikit.SpinnerFrames(uikit.ActiveMode())` at spinner start
- [ ] In ASCII mode, the cliout spinner cycles through `|`, `/`, `-`, `\`
- [ ] `cliout.SetTestMode(true)` calls `uikit.SetModeForTest(uikit.GlyphASCII)` in
      addition to `pinASCII`
- [ ] `pinASCII` has a doc comment stating it is colour-only and that glyph mode is
      sourced from `uikit.ActiveMode()`
- [ ] Every glyph assertion in `internal/cliout/*_test.go` resolves via
      `uikit.GlyphFor` (or pins `uikit.SetModeForTest` explicitly when validating one
      specific mode)
- [ ] New test `TestStatusGlyph_HonoursUikitMode` switches uikit between unicode and
      ASCII modes and asserts cliout output follows
- [ ] `go test ./internal/cliout/ -v` passes; full suite passes under both
      `LANG=en_US.UTF-8` and `LANG=C`
- [ ] No catalogue characters appear inline in `internal/cliout/*.go` (only via
      `uikit.GlyphFor`)
- [ ] `make ci` → PASS

## Tasks

Step-by-step: Tasks 2.1–2.6 in `docs/superpowers/plans/2026-04-29-glyph-fallback.md`.

- [ ] Branch: `feat/13-cliout-uikit-integration`
- [ ] Write failing `TestStatusGlyph_AsciiMode` covering all six `Status` values → FAIL
- [ ] Add `import "github.com/initgrep-apps/spotnik/internal/uikit"` to `message.go`;
      replace `statusGlyph()` with the `Status` → `GlyphRole` map → PASS
- [ ] Commit: `refactor(cliout): route statusGlyph through uikit.GlyphFor`
- [ ] Write failing `TestHint_AsciiArrow` → FAIL
- [ ] Replace the literal `→` in `Hint.render()` with `uikit.GlyphFor(uikit.GlyphInfo,
      uikit.ActiveMode())` → PASS
- [ ] Commit: `fix(cliout): route Hint arrow through uikit.GlyphFor(GlyphInfo)`
- [ ] Write failing `TestSpinnerFrames_AsciiSet` → FAIL
- [ ] Add `resolveSpinnerFrames()` helper; remove the inline `spinnerFrames` var; update
      `(*SpinnerHandle).run` to snapshot frames at start → PASS
- [ ] Commit: `fix(cliout): source spinner frames from uikit.SpinnerFrames`
- [ ] Add `TestStatusGlyph_HonoursUikitMode` regression test pinning the
      cliout-honours-uikit-mode contract → PASS
- [ ] Commit: `test(cliout): pin the cliout-honours-uikit-mode contract`
- [ ] Update `cliout.SetTestMode(true)` to call `uikit.SetModeForTest(uikit.GlyphASCII)`
- [ ] Document `pinASCII` as colour-only
- [ ] Commit: `fix(cliout): align SetTestMode with uikit.SetModeForTest(GlyphASCII)`
- [ ] Run full cliout suite, list failing assertions, rewrite each to resolve via
      `uikit.GlyphFor` (or pin a specific mode) → ALL PASS
- [ ] Commit: `test(cliout): rebaseline tests to assert glyph roles, not codepoints`
- [ ] `make ci` → PASS
- [ ] Push branch + open PR
