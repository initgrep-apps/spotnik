---
title: "layout/border.go — remove ᐅ from filter mode, ascii-mode corner lookup"
feature: 13-tui-design-system
status: open
---

## Background

Second foundation story. Cleans up `internal/ui/layout/border.go` so the border
renderer is compatible with the uikit design system:

1. **Removes `ᐅ`** (U+1405 Canadian Syllabics Pa, banned per §5.4 of the design
   record). Filter-mode action hints use the same corner-notch format as actions
   mode: `filtering: "query" ─╮ Esc close ╭`.
2. **Routes corner/rule characters through `uikit.GlyphFor`** using
   `uikit.ActiveMode()` — so the same renderer emits `╭╮╰╯─│` in unicode mode and
   `++++-|` in ascii mode.

This is a prerequisite for `PaneChrome` (S3) which wraps `layout.RenderPaneBorder`
as its internal implementation, and for `StatusBar` (S7) which reuses the same
border helpers.

**Depends on:** S1 (`internal/uikit` scaffold exists). Design record §5.1, §5.4,
§7.3 format rules. Full step-by-step: Task 2 (S2) in
`docs/superpowers/plans/2026-04-24-tui-design-system.md`.

## Design

### Filter-mode notch format

Current output (to be replaced):

```
╭─ ²Queue ᐅEsc close──────────────────╮
```

New output — same notch format as actions mode:

```
╭─ ²Queue────filtering: "rock" ─╮ Esc close ╭─╮
```

Preamble `filtering: "query"` renders in `Muted`. The `Esc close` notch sits in the
standard `╮ key label ╭` frame, joined to the preamble by a single `─`. The final
`╭` butts against the top-right corner `╮` producing `╭╮` — intentional, matches
the actions-mode rule in §7.3.

### Glyph lookup

Replace hardcoded corner constants with a helper that reads `uikit.ActiveMode()`:

```go
func corners() (tl, tr, bl, br, h, v string) {
    m := uikit.ActiveMode()
    return uikit.GlyphFor(uikit.GlyphCornerTL, m),
           uikit.GlyphFor(uikit.GlyphCornerTR, m),
           uikit.GlyphFor(uikit.GlyphCornerBL, m),
           uikit.GlyphFor(uikit.GlyphCornerBR, m),
           uikit.GlyphFor(uikit.GlyphHRule, m),
           uikit.GlyphFor(uikit.GlyphVRule, m)
}
```

`RenderPaneBorder` reads the six values once per invocation (no perf concern —
panes re-render on resize/state change, not every tick).

### Docstring updates

Lines 28, 36, 213, 220, 224 of `border.go` mention `ᐅ` — rewrite to describe the
notch format.

### Test updates

`border_test.go` line 742 currently asserts `Contains(topLine, "ᐅ")`. Flip to
`NotContains` + two new assertions (preamble present; notch present). Add a new
`TestRenderPaneBorder_ASCIIMode_SwapsCorners` test that calls
`uikit.SetModeForTest(uikit.GlyphASCII)` and verifies corners are `+`.

## Acceptance Criteria

- [ ] `internal/ui/layout/border.go` no longer emits the rune `ᐅ` — verified by
      `grep -n 'ᐅ' internal/ui/layout/border.go` → no matches
- [ ] Filter-mode output matches `filtering: "<query>" ─╮ Esc close ╭` —
      preamble in `Muted`, key `Esc` in `Accent` (via `theme.KeyHint()`), label
      `close` in `Muted`
- [ ] Corners, horizontal rule, and vertical rule are looked up via
      `uikit.GlyphFor(role, uikit.ActiveMode())`
- [ ] `border_test.go:742` assertion is inverted — `NotContains(topLine, "ᐅ")`
      plus `Contains(topLine, "filtering: \"rock\"")` and
      `Contains(topLine, "╮ Esc close ╭")`
- [ ] New `TestRenderPaneBorder_ASCIIMode_SwapsCorners` test passes — top line
      contains `+`, not `╭`/`╮`; bottom line contains `+`, not `╰`/`╯`
- [ ] All existing `border_test.go` tests still PASS
- [ ] Docstrings at lines 28, 36, 213, 220, 224 no longer reference `ᐅ`
- [ ] `make ci` → PASS

## Tasks

Step-by-step TDD guide: Task 2 (S2) in
`docs/superpowers/plans/2026-04-24-tui-design-system.md`.

- [ ] Branch: `feat/13-border-no-arrow-ascii-mode`
- [ ] Rewrite `border_test.go:742` filter-mode assertion (Step 2.2) → FAIL
- [ ] Add `TestRenderPaneBorder_ASCIIMode_SwapsCorners` (Step 2.3) → FAIL
- [ ] Import `internal/uikit` in `border.go` (Step 2.4)
- [ ] Add `corners()` helper returning glyphs per `ActiveMode()` (Step 2.4)
- [ ] Replace hardcoded corner consts with `corners()` call sites (Step 2.4)
- [ ] Rewrite `buildRightSegment` filter branch to emit the notch format
      (Step 2.4) → filter-mode test PASSES
- [ ] Update docstrings on lines 28, 36, 213, 220, 224 (Step 2.4)
- [ ] Run `go test ./internal/ui/layout/ -v` → all PASS (Step 2.5)
- [ ] `make ci` → PASS
- [ ] Commit + push + open PR (Step 2.6)
