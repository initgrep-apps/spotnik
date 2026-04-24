---
title: "StatusBar + KeyBar — bottom key bar with bubbles/help integration"
feature: 13-tui-design-system
status: done
---

## Background

`KeyBar` is a stateless strip that renders `[]key.Binding` values as
`key desc · key desc` with a `·` separator that becomes `|` in ascii mode.
`StatusBar` wraps `KeyBar` with a muted top/bottom border (via
`layout.RenderPaneBorder`) and uses `bubbles/help` to route between short-help
(single row) and full-help modes. `KeyBar` is reusable anywhere a key strip is
needed (overlay footers, inline hints); `StatusBar` is the specific bottom-of-app
composition.

Migrates `internal/app/render.go:renderStatusBar` to the new primitives.

**Depends on:** S1, S2 (for muted border rendering). Design record §7.1 rows
11–12. Full step-by-step: Task 7 (S7) in
`docs/superpowers/plans/2026-04-24-tui-design-system.md`.

## Design

### KeyBar

```go
type KeyBar struct {
    Bindings []key.Binding
    Theme    theme.Theme
}
```

Renders keys via `theme.KeyHint()`, descriptions via `theme.TextMuted()`, joined
with ` · ` (unicode) or ` | ` (ascii).

### StatusBar

```go
type StatusBar struct {
    Width    int
    Bindings help.KeyMap
    Theme    theme.Theme
}
```

3 lines tall (top border + content + bottom border). Uses `bubbles/help` for
short-help rendering. Minimum width 160 columns to match the current layout.

### Call-site migration

```go
// internal/app/render.go
func (a *App) renderStatusBar() string {
    km := a.statusKeyMap
    km.activePage = a.layout.ActivePage()
    return uikit.StatusBar{Width: a.width, Bindings: km, Theme: a.theme}.Render()
}
```

### Roles

| Field | Role |
|---|---|
| StatusBar.Bg | `theme.StatusBarBg()` |
| StatusBar.Key | `theme.KeyHint()` |
| StatusBar.Desc | Muted |
| KeyBar.Key | `theme.KeyHint()` |
| KeyBar.Desc | Muted |
| KeyBar.Separator | Muted |

## Acceptance Criteria

- [ ] `internal/uikit/key_bar.go` defines `KeyBar` with `Render() string`
- [ ] `internal/uikit/status_bar.go` defines `StatusBar` with `Render() string`
- [ ] `key_bar_test.go` covers unicode `·` separator + ascii `|` fallback
- [ ] `status_bar_test.go` covers 3-line output + page-aware bindings
- [ ] `internal/app/render.go:renderStatusBar` uses `uikit.StatusBar`
- [ ] `render_test.go` assertions updated
- [ ] `make ci` → PASS

## Tasks

Step-by-step: Task 7 (S7) in plan.

- [ ] Branch: `feat/13-uikit-status-key-bar`
- [ ] Write failing `key_bar_test.go` + implement `key_bar.go` (Step 7.1)
- [ ] Write failing `status_bar_test.go` + implement `status_bar.go` (Step 7.2)
- [ ] Migrate `render.go:renderStatusBar` (Step 7.3)
- [ ] `make ci` → PASS (Step 7.4)
- [ ] Commit + push + open PR
