---
title: "Fix: overlay positioning — restore Right/Top for theme, device, profile overlays"
feature: 13-tui-design-system
status: done
---

## Background

Story 153 consolidated 5 per-overlay `renderWith*Overlay` helpers into a single
`renderWithOverlayChrome(background, overlayView string)`. The original helpers
positioned three compact overlays (theme, profile, device) at the top-right corner
via `btoverlay.Right, btoverlay.Top`, while search and help were centered with
`btoverlay.Center, btoverlay.Center`. The new consolidated helper hardcoded
`btoverlay.Center, btoverlay.Center` for all overlays, so pressing `t`, `u`, and
`d` now shows those overlays centered instead of at the top-right corner.

**Root cause:** The spec for S153 included a code example that wrote
`btoverlay.Center, btoverlay.Center` for all callers — losing per-overlay
positioning. The five original functions encoded position at the call site;
consolidation without preserving per-call position was the regression.

**Files:** `internal/app/render.go`

## Design

Add `hPos, vPos btoverlay.Position` parameters to `renderWithOverlayChrome` so
each call site can declare its own position:

```go
func (a *App) renderWithOverlayChrome(background, overlayView string, hPos, vPos btoverlay.Position) string {
    dimmed := lipgloss.NewStyle().Faint(true).Render(background)
    if a.width <= 0 || a.height <= 0 {
        return dimmed + "\n" + overlayView
    }
    return btoverlay.Composite(overlayView, dimmed, hPos, vPos, 0, 0)
}
```

Update the 5 call sites in `buildView` (render.go ~line 388):

```go
// compact corner overlays — top-right
return a.renderWithOverlayChrome(body, a.themeOverlay.View(), btoverlay.Right, btoverlay.Top)
return a.renderWithOverlayChrome(body, a.devicePane.View(),   btoverlay.Right, btoverlay.Top)
return a.renderWithOverlayChrome(body, a.profilePane.View(),  btoverlay.Right, btoverlay.Top)

// full-screen overlays — centered
return a.renderWithOverlayChrome(body, a.searchPane.View(),  btoverlay.Center, btoverlay.Center)
return a.renderWithOverlayChrome(body, a.helpOverlay.View(), btoverlay.Center, btoverlay.Center)
```

## Acceptance Criteria

- [ ] `renderWithOverlayChrome` signature takes `hPos, vPos btoverlay.Position`
- [ ] Theme overlay (`t`), profile overlay (`u`), device overlay (`d`) composite
      at `btoverlay.Right, btoverlay.Top`
- [ ] Search overlay and help overlay remain at `btoverlay.Center, btoverlay.Center`
- [ ] `render_test.go` — existing overlay compositing tests updated to pass the
      position params; assert theme overlay appears in the top-right region (x offset
      ≈ bg width − fg width, y offset = 0)
- [ ] `make ci` → PASS

## Tasks

- [ ] Branch: `fix/13-overlay-position`
- [ ] Add `hPos, vPos btoverlay.Position` to `renderWithOverlayChrome` (render.go:500)
- [ ] Update all 5 call sites with correct position constants
- [ ] Update `render_test.go` assertions that call `renderWithOverlayChrome`
- [ ] `make ci` → PASS
- [ ] Commit + push + open PR
