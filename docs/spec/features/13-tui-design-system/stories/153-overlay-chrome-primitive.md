---
title: "OverlayChrome primitive — consolidate the 5 renderWith*Overlay helpers"
feature: 13-tui-design-system
status: open
---

## Background

`OverlayChrome` renders a floating overlay panel — visually identical to a
focused `PaneChrome` but with `Accent` as the border colour (overlays always own
input focus). This story creates the primitive **and** collapses the 5 existing
`renderWithThemeOverlay` / `renderWithDeviceOverlay` / `renderWithProfileOverlay` /
`renderWithSearchOverlay` / `renderWithHelpOverlay` helpers in `internal/app/render.go`
into one `renderWithOverlayChrome(bg, overlayView)` helper that composites over a
dimmed background.

**Depends on:** S1. Design record §7 summary, §7.1. Full step-by-step: Task 4 (S4)
in `docs/superpowers/plans/2026-04-24-tui-design-system.md`.

## Design

### Struct

```go
type OverlayChrome struct {
    Width, Height int
    Title         string
    Actions       []Action // Action = layout.Action
    Theme         theme.Theme
}

type Action = layout.Action
```

### Rendering

Delegates to `layout.RenderPaneBorder` with `AccentColor = theme.Accent()` and
`Focused = true`. The individual overlay panes (`SearchOverlay`, `DeviceOverlay`,
etc.) compose their own bodies and pass them into `OverlayChrome.Render(content)`.

### `render.go` consolidation

Replace the 5 per-overlay helpers with one:

```go
func (a *App) renderWithOverlayChrome(background, overlayView string) string {
    dimmed := lipgloss.NewStyle().Faint(true).Render(background)
    if a.width <= 0 || a.height <= 0 {
        return dimmed + "\n" + overlayView
    }
    return btoverlay.Composite(overlayView, dimmed, btoverlay.Center, btoverlay.Center, 0, 0)
}
```

`buildView` calls the new helper for every overlay state — search, device, profile,
theme, help. The overlay panes themselves already return rendered strings; this
story wraps those strings with the primitive where appropriate.

### Roles

| Field | Role |
|---|---|
| Border | Accent |
| Title | Strong |
| Action.Key | Accent |
| Action.Label | Muted |

## Acceptance Criteria

- [ ] `internal/uikit/overlay_chrome.go` defines `OverlayChrome` with struct above
      and `Render(content string) string`
- [ ] `internal/uikit` re-exports `Action = layout.Action` for call-site ergonomics
- [ ] `overlay_chrome_test.go` covers unicode default border + ascii fallback
- [ ] `internal/app/render.go` has exactly one overlay helper
      (`renderWithOverlayChrome`); the 5 `renderWith*Overlay` funcs are deleted
- [ ] `render_test.go` assertions updated to reference the consolidated helper
- [ ] `make ci` → PASS

## Tasks

Step-by-step: Task 4 (S4) in plan.

- [ ] Branch: `feat/13-uikit-overlay-chrome`
- [ ] Write failing `overlay_chrome_test.go` (Step 4.1)
- [ ] Implement `overlay_chrome.go` + export `Action` type alias (Step 4.2)
- [ ] Migrate `render.go` — replace 5 helpers with `renderWithOverlayChrome`
      (Step 4.3)
- [ ] Update `render_test.go` assertions (Step 4.3)
- [ ] `make ci` → PASS (Step 4.4)
- [ ] Commit + push + open PR
