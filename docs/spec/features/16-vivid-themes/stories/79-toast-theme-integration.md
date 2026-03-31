---
title: "Fix: Toast Notification Theme Integration"
feature: 16-vivid-themes
status: done
---

## Background

Toast notifications (success, error, warning, info, ratelimit) are rendered by
the `bubbleup.AlertModel` created in `components.NewNotifications(t)`. Two issues
prevent toasts from fully participating in the theme system:

1. **Missing `Info()` token** — The Theme interface defines `Success()`, `Warning()`,
   and `Error()` but has no `Info()`. The `info` toast borrows `KeyHint()`, a status
   bar token, which produces a blue tint unrelated to the theme's canonical info color.

2. **Alerts not recreated on theme switch** — `NewNotifications(t)` is called once at
   app init (`app.go:229`). The `ThemeSwitchMsg` handler (line 1354) propagates the
   new theme to all panes and overlays but never recreates `a.alerts`. Alert type
   definitions retain the old theme's hex colors as baked-in `ForeColor` strings.

## Design

### Task 1: Add `Info()` Token to Theme System

**Files:** `internal/ui/theme/theme.go`, `internal/ui/theme/config_theme.go`, all 11 TOML files

Add `Info() lipgloss.Color` to the Theme interface alongside the existing semantic
colours (`Success`, `Warning`, `Error`). Add `Info string` field with TOML tag
`"info"` to `themeColors`. Add the `Info()` accessor to `ConfigTheme`.

Place the new method in the "Semantic colours" group in the Theme interface, after
`Error()` and before `DeviceActive()`. In `config_theme.go`, place the accessor
after `Error()` in the same group.

Canonical info colors per theme (sourced from each theme's official palette):

| Theme | TOML id | Info Color | Palette Source |
|---|---|---|---|
| True Black | `black` | `#00afff` | Matches existing accent blue |
| Light (Catppuccin Latte) | `light` | `#179299` | Catppuccin Latte Teal |
| Dracula | `dracula` | `#8BE9FD` | Dracula Cyan |
| Gruvbox Dark | `gruvbox` | `#83A598` | Gruvbox bright_blue |
| Nord | `nord` | `#88C0D0` | Nord8 / Frost primary |
| Rose Pine | `rosepine` | `#9CCFD8` | Foam |
| Catppuccin Mocha | `catppuccin` | `#94E2D5` | Catppuccin Teal (official info diagnostic) |
| Tokyo Night | `tokyonight` | `#0DB9D7` | Dedicated info key in palette |
| Synthwave '84 | `synthwave` | `#36F9F6` | Neon cyan |
| Monokai | `monokai` | `#66D9EF` | Classic Monokai cyan |
| Solarized Dark | `solarized` | `#268BD2` | Solarized blue |

In each TOML file, add `info = "<hex>"` in the `[colors]` section after `error`.

### Task 2: Wire Info Toast and Recreate Alerts on Theme Switch

**Files:** `internal/ui/components/notifications.go`, `internal/app/app.go`

**2a — Wire `info` toast to `Info()` token:**

In `NewNotifications()`, change the `infoAlert` definition from:
```go
ForeColor: string(t.KeyHint()),
```
to:
```go
ForeColor: string(t.Info()),
```

**2b — Recreate alerts on theme switch:**

In `app.go`, in the `ThemeSwitchMsg` handler (after propagating theme to all panes
and overlays), recreate the alerts model:

```go
a.alerts = *components.NewNotifications(newTheme)
```

This must come before the `NewAlertCmd("success", ...)` call so the success toast
itself uses the new theme's colors. The new `AlertModel` starts with no active
alerts (previous toast is lost on switch — acceptable since theme change is
user-initiated and the success toast fires immediately after).

## Acceptance Criteria

- [ ] `Info()` method exists on Theme interface and all 11 themes implement it
- [ ] `info` toast uses `t.Info()` color, not `t.KeyHint()`
- [ ] After theme switch, new toasts render with the new theme's colors
- [ ] The "Theme: {name}" success toast after switching uses the new theme's success color
- [ ] All existing theme validation tests pass (reflect-based field check catches missing TOML fields)
- [ ] `make ci` passes (lint + tests + 80% coverage)

## Tasks

- [ ] **Task 1:** Add `Info() lipgloss.Color` to Theme interface in `theme.go`. Add `Info string` with `toml:"info"` tag to `themeColors` in `config_theme.go`. Add `Info()` accessor to `ConfigTheme`. Add `info` field with canonical hex value to all 11 TOML files.
    - Existing `TestParseTheme_AllFieldsRequired` and `TestAllEmbeddedThemes_Valid` must pass (they use reflection to validate all fields are non-empty, so adding the field + TOML values is sufficient).
    - Add `TestConfigTheme_Info` to verify `Info()` returns the expected color for a parsed theme.
- [ ] **Task 2:** Change `infoAlert.ForeColor` in `NewNotifications()` from `string(t.KeyHint())` to `string(t.Info())`. In `app.go` `ThemeSwitchMsg` handler, add `a.alerts = *components.NewNotifications(newTheme)` before the `NewAlertCmd` call.
    - Update `TestNewNotifications_RegistersAllTypes` (if it asserts specific ForeColor values) to expect `Info()` instead of `KeyHint()`.
    - Add `TestApp_ThemeSwitch_RecreatesAlerts` — switch theme, fire a toast, verify the alert command is non-nil (alerts model is functional after recreation).
