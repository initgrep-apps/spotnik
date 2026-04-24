---
title: "Panel primitive — full-screen bordered panel with title in border"
feature: 13-tui-design-system
status: done
---

## Background

`Panel` is a full-screen bordered container used by onboarding screens, the auth
panel, the too-small screen, and the splash screen. Unlike `PaneChrome`, the
panel title lives **in the top border** — there is no separate step-header
string. Two intents: `PanelIntentDefault` (Accent border) and `PanelIntentError`
(Error border, used by the onboarding failure screen).

No call-site migration in this story — onboarding + auth + splash + too-small
migrate in S18 once all composing primitives are available.

**Depends on:** S1. Design record §7.1 row 3; §7 Panel stub. Full step-by-step:
Task 5 (S5) in `docs/superpowers/plans/2026-04-24-tui-design-system.md`.

## Design

### Struct

```go
type PanelIntent int
const (
    PanelIntentDefault PanelIntent = iota
    PanelIntentError
)

type Panel struct {
    Width, Height int
    Title         string
    Intent        PanelIntent
    Theme         theme.Theme
}
```

### Rendering

Delegates to `layout.RenderPaneBorder` with `AccentColor` set from intent:
- `PanelIntentDefault` → `theme.Accent()`
- `PanelIntentError` → `theme.Error()`

Title renders via the same rules as `PaneChrome` — absorbs the step-header role
(the caller no longer emits a separate header line above the panel body).

### Roles

| Field | Role |
|---|---|
| Border (default) | Accent |
| Border (error) | Error |
| Title | Strong |

## Acceptance Criteria

- [ ] `internal/uikit/panel.go` defines `Panel` + `PanelIntent` enum
- [ ] `panel_test.go` covers:
      - `TestPanel_TitleInBorder` — title appears on the top border line
      - `TestPanel_ErrorIntent_UsesErrorBorder` — error intent renders
- [ ] Uikit coverage remains 100%
- [ ] `make ci` → PASS

## Tasks

Step-by-step: Task 5 (S5) in plan.

- [ ] Branch: `feat/13-uikit-panel`
- [ ] Write failing `panel_test.go` (Step 5.1)
- [ ] Implement `panel.go` with `Panel`, `PanelIntent`, `Render` (Step 5.2)
- [ ] Run tests → PASS
- [ ] `make ci` → PASS
- [ ] Commit + push + open PR (Step 5.3)
