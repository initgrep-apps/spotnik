---
title: "HeaderBar + Chip — extract renderHeader from render.go"
feature: 13-tui-design-system
status: done
---

## Background

Two primitives ship together because the header bar is built from a sequence of
chips. `Chip` renders an inline pill (`<glyph> <label>` on `StatusBarBg`); used
for the device chip and profile chip on the right edge of the app header.
`HeaderBar` composes the left segment (`spotnik ─ Page X ─ preset N`) with a fill
gap and an array of pre-rendered chips on the right.

Migrates `internal/app/render.go:renderHeader`, `renderProfileChip`, and the
device-chip builder onto these primitives.

**Depends on:** S1. Design record §7.1 rows 10 + 13. Full step-by-step: Task 6
(S6) in `docs/superpowers/plans/2026-04-24-tui-design-system.md`.

## Design

### Chip

```go
type Chip struct {
    Glyph  GlyphRole
    Label  string
    Intent Role
    Theme  theme.Theme
}
```

Renders `" <glyph> <label> "` on `theme.StatusBarBg()` with the glyph coloured
by `Intent` and the label by `theme.HeaderChipFg()`.

### HeaderBar

```go
type HeaderBar struct {
    Width      int
    AppName    string
    Page       string // "A" or "B"
    Preset     int    // -1 hides preset segment (Page B)
    RightChips []string // pre-rendered chip strings from Chip.Render()
    Theme      theme.Theme
}
```

Layout: `spotnik ─ Page A ─ preset 1` on left, chips on right, background-filled
gap between. On Page B the `preset` segment is hidden.

### Call-site migration

`render.go:renderHeader` constructs chips for active device + profile, then
returns a `HeaderBar{...}.Render()` string. Premium profile → `GlyphPremium` +
`RoleInfo`; free-tier profile → `GlyphAvailable` + `RoleMuted`.

### Roles

| Field | Role |
|---|---|
| HeaderBar.Bg | `theme.StatusBarBg()` |
| HeaderBar.AppName | Strong |
| HeaderBar.Separator | Muted |
| HeaderBar.PageKey (A/B) | Accent |
| HeaderBar.PresetLabel | Muted |
| Chip.Glyph | intent role |
| Chip.Label | `theme.HeaderChipFg()` |
| Chip.Bg | `theme.StatusBarBg()` |

## Acceptance Criteria

- [ ] `internal/uikit/chip.go` defines `Chip` with `Render() string`
- [ ] `internal/uikit/header_bar.go` defines `HeaderBar` with `Render() string`
- [ ] `chip_test.go` covers unicode active-device chip + ascii premium chip
- [ ] `header_bar_test.go` covers left/right segments + background-filled gap
- [ ] `internal/app/render.go:renderHeader` uses `HeaderBar` + `Chip`; no more
      inline `lipgloss.NewStyle()` calls for the header
- [ ] `renderProfileChip` and device-chip builder deleted; their bodies inlined
      into `renderHeader` via `Chip{...}.Render()`
- [ ] `render_test.go` updated
- [ ] `make ci` → PASS

## Tasks

Step-by-step: Task 6 (S6) in plan.

- [ ] Branch: `feat/13-uikit-header-bar-chip`
- [ ] Write failing `chip_test.go` (Step 6.1)
- [ ] Implement `chip.go` (Step 6.2)
- [ ] Write failing `header_bar_test.go` + implement `header_bar.go` (Step 6.3)
- [ ] Migrate `render.go:renderHeader` to use `HeaderBar` + `Chip`;
      delete `renderProfileChip` + device-chip builder (Step 6.4)
- [ ] Update `render_test.go`
- [ ] `make ci` → PASS
- [ ] Commit + push + open PR (Step 6.5)
