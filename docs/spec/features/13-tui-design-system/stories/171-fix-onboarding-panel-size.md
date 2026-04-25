---
title: "Fix: onboarding panel — replace full-screen sizing with proportional PanelSize"
feature: 13-tui-design-system
status: open
---

## Background

Story 167 rewrote the three onboarding render functions (`renderOnboardingRegister`,
`renderOnboardingOAuth`, `renderOnboardingError`) to use `uikit.Panel`. The `Panel`
struct requires explicit `Width` and `Height`. Story 167 sized these as
`a.width - 4` × `a.height - 4`, making the panel nearly full-screen. The
`lipgloss.Place(Center, Center)` call in `renderOnboarding()` then has no visible
centering effect — the panel already spans the entire terminal, obscuring the
"smaller, clearly-modal dialog" appearance that onboarding had before the migration.

**Root cause:** Story 154 described `Panel` as "a full-screen bordered container,"
and Story 167 implemented sizing literally. The intent was a prominent but clearly
modal panel — not a full-screen cover. The pre-migration hand-rolled border was
content-sized, which produced a compact centred dialog.

**Files:** `internal/uikit/sizes.go` (new), `internal/app/render.go`

## Design

### New helper `uikit.PanelSize`

Add `internal/uikit/sizes.go` with a `PanelSize` function that encodes the sizing
policy for all full-screen modal panels (onboarding, auth, too-small). Centralising
in `uikit` keeps the policy with the primitive rather than scattered across
render.go:

```go
package uikit

// PanelSize returns (width, height) for a centered modal panel.
// 70% of terminal width (min 80) and 65% of terminal height (min 20) leaves
// visible margins on all sides so lipgloss.Place() produces a centred effect.
func PanelSize(termW, termH int) (w, h int) {
    w = termW * 70 / 100
    if w < 80 {
        w = 80
    }
    h = termH * 65 / 100
    if h < 20 {
        h = 20
    }
    return
}
```

### `render.go` — replace hardcoded sizing

In each of the three onboarding render functions and `renderAuthPanel`, replace:

```go
// before
panelW := a.width - 4
if panelW < 80 { panelW = 80 }
panelH := a.height - 4
if panelH < 20 { panelH = 20 }
```

with:

```go
// after
panelW, panelH := uikit.PanelSize(a.width, a.height)
```

Also update the `panelInnerWidth` calculations that derive from the old `a.width - 8`
to use `panelW - 8` so URL boxes and centred titles stay contained within the smaller
panel:

```go
// before
panelInnerWidth := a.width - 8

// after
panelInnerWidth := panelW - 8
```

## Acceptance Criteria

- [ ] `internal/uikit/sizes.go` defines `PanelSize(termW, termH int) (w, h int)` with
      the 70%/65% policy (min 80 × 20)
- [ ] `sizes_test.go` covers: `TestPanelSize_Proportional` (wide terminal → 70%/65%);
      `TestPanelSize_MinimumClamp` (narrow terminal → minimums); `TestPanelSize_Zero`
      (0×0 → 80×20)
- [ ] `uikit` coverage remains 100%
- [ ] `renderOnboardingRegister`, `renderOnboardingOAuth`, `renderOnboardingError`
      each use `uikit.PanelSize(a.width, a.height)` for panel dimensions
- [ ] `panelInnerWidth` in each render function derives from `panelW - 8` (not
      `a.width - 8`), keeping URL boxes and centred titles inside the panel
- [ ] On a 220×50 terminal: panel is ≈154×32, not 216×46 — visual smoke test via
      `make run` confirms the centering margin is visible
- [ ] `render_test.go` assertions updated where panel width/height is asserted
- [ ] `make ci` → PASS

## Tasks

- [ ] Branch: `fix/13-onboarding-panel-size`
- [ ] Write failing `sizes_test.go` (three cases above) → compile error
- [ ] Implement `internal/uikit/sizes.go` → tests PASS
- [ ] Update `renderOnboardingRegister`: use `PanelSize`; update `panelInnerWidth`
- [ ] Update `renderOnboardingOAuth`: same
- [ ] Update `renderOnboardingError`: same
- [ ] Update `render_test.go` where panel size is asserted
- [ ] `make ci` → PASS
- [ ] Commit + push + open PR
