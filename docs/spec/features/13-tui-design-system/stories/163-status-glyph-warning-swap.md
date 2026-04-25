---
title: "StatusGlyph + swap ⚠ → ◬ across cliout, notifications, onboarding"
feature: 13-tui-design-system
status: done
---

## Background

Two concerns ship together: the `StatusGlyph` atomic primitive and the codebase-wide
swap of `⚠` (U+26A0, variation-selector sensitive) to `◬` (U+25EC). The swap
touches `internal/cliout` (Feature 12 shipped with `⚠` in `statusGlyph`),
`internal/ui/components/notifications.go` (warning alert prefix), and any
scattered inline `warnStyle.Render("⚠ …")` sites in `internal/app/render.go`.

`StatusGlyph` itself is the primitive callers use for persistent informational
state — **not** for notifications (those are `Toast`). Rule from §10: "Do not
use StatusGlyph + text inline for things a Toast handles — toasts are for
completion acknowledgements and async events; StatusGlyph is for persistent
informational state."

**Depends on:** S1. Design record §5.2 (warning glyph decision), §10 rules 4 & 8.
Full step-by-step: Task 14 (S14) in
`docs/superpowers/plans/2026-04-24-tui-design-system.md`.

## Design

### StatusGlyph

```go
type StatusGlyph struct {
    Role  Role
    Text  string
    Theme theme.Theme
}
```

`Render() string` emits `<glyph> <text>` with the glyph coloured by role. Supported
roles: `RoleSuccess` (`✓`), `RoleError` (`✗`), `RoleWarning` (`◬`), `RoleInfo` (`→`).

### `⚠` → `◬` swap

- `internal/ui/components/notifications.go` — warning alert `Prefix` field changes
  to `◬` (or update to use `uikit.GlyphFor(uikit.GlyphWarning, uikit.ActiveMode())`)
- `internal/cliout/message.go` — `statusGlyph(StatusWarning)` returns `◬` / `!`
- `internal/app/render.go` — `warnStyle.Render("⚠ …")` sites replaced with
  `uikit.StatusGlyph{Role: uikit.RoleWarning, Text: "...", Theme: t}.Render()`

After this story: `grep -rn "⚠" internal/ cmd/` must return **no matches**.

### Roles

| Field | Role |
|---|---|
| StatusGlyph | intent role |

## Acceptance Criteria

- [ ] `internal/uikit/status_glyph.go` defines `StatusGlyph` with `Render() string`
      supporting Success/Error/Warning/Info roles
- [ ] `status_glyph_test.go` covers:
      - `TestStatusGlyph_WarningRendersCircleTriangle` — output contains
        `◬ Premium required`; does **not** contain `⚠`
      - `TestStatusGlyph_ASCII_Warning` — ascii mode emits `! X`
- [ ] `internal/ui/components/notifications.go` warning prefix is `◬`
- [ ] `internal/ui/components/notifications_test.go` asserts `◬` prefix
- [ ] `internal/cliout/message.go` `StatusWarning` glyph is `◬`; ascii `!`
- [ ] `internal/cliout/message_test.go` asserts `◬`
- [ ] `internal/app/render.go` has no `⚠` literals; all warning lines route
      through `uikit.StatusGlyph`
- [ ] `grep -rn "⚠" internal/ cmd/` → no matches
- [ ] `make ci` → PASS

## Tasks

Step-by-step: Task 14 (S14) in plan.

- [ ] Branch: `feat/13-uikit-status-glyph-warning-swap`
- [ ] Write failing `status_glyph_test.go` (Step 14.1)
- [ ] Implement `status_glyph.go` (Step 14.2)
- [ ] Swap warning glyph in `notifications.go` + update test (Step 14.3)
- [ ] Swap warning glyph in `cliout/message.go` + update
      `cliout/message_test.go` (Step 14.3)
- [ ] Replace `⚠` inline lines in `render.go` with `uikit.StatusGlyph`
      (Step 14.3)
- [ ] Verify `grep -rn "⚠" internal/ cmd/` → no matches (Step 14.4)
- [ ] `make ci` → PASS
- [ ] Commit + push + open PR (Step 14.5)
