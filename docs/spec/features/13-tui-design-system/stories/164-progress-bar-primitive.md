---
title: "ProgressBar — seek and volume bars with unicode partial blocks, ascii fallback"
feature: 13-tui-design-system
status: open
---

## Background

`ProgressBar` consolidates the seek-bar and volume-bar rendering currently
spread across `internal/ui/components/controls.go`. Unicode mode uses
partial-block glyphs (`▏▎▍▌▋▊▉█`) for 1/8 resolution on the boundary cell; ascii
mode collapses to `#` / `=` / `-` / `.` based on coarse thresholds. Input
clamps `Progress ∈ [0, 1]`.

**Depends on:** S1. Design record §5.7 (graphical fills), §7.1 row 17. Full
step-by-step: Task 15 (S15) in
`docs/superpowers/plans/2026-04-24-tui-design-system.md`.

## Design

### Struct

```go
type ProgressBar struct {
    Width    int
    Progress float64 // clamped to [0,1]
    Theme    theme.Theme
}
```

### Algorithm

1. `filledFloat = Progress × Width`; `filled = floor(filledFloat)`;
   `remainder = filledFloat − filled`.
2. Emit `filled` full blocks (`█` / `#`).
3. If `remainder > 0` and `filled < Width`, emit one partial block whose glyph
   is picked from `remainder` by `partialGlyph`:
   - `≥ 7/8` → `▉` / `#`
   - `≥ 6/8` → `▊` / `#`
   - `≥ 5/8` → `▋` / `=`
   - `≥ 4/8` → `▌` / `=`
   - `≥ 3/8` → `▍` / `-`
   - `≥ 2/8` → `▎` / `-`
   - `else` → `▏` / `.`
4. Fill the remaining cells with `░` / `.`.

Fill colour: `theme.Gradient1()`. Empty colour: `Muted`.

### Call-site migration

`internal/ui/components/controls.go` — locate the seek-bar and volume-bar
rendering functions (likely `RenderSeekBar`, `RenderVolumeBar`) and replace
inline character composition with `uikit.ProgressBar{...}.Render()`.

### Roles

| Field | Role |
|---|---|
| ProgressBar.Fill | `theme.Gradient1/2/3()` per position |
| ProgressBar.Empty | Muted |

## Acceptance Criteria

- [ ] `internal/uikit/progress_bar.go` defines `ProgressBar` with `Render() string`
      using the partial-block algorithm
- [ ] `progress_bar_test.go` covers:
      - `TestProgressBar_Unicode_HalfFilled` — output has exactly 20 cells
        combining `█` and `░`
      - `TestProgressBar_ASCII_HalfFilled` — 10 `#` + 10 `.`
      - `TestProgressBar_ClampsProgress` — `Progress: 2.0` renders identically
        to `Progress: 1.0`; `Progress: -0.5` identically to `Progress: 0.0`
- [ ] `internal/ui/components/controls.go` seek bar and volume bar render via
      `uikit.ProgressBar`
- [ ] `controls_test.go` ascii snapshots added
- [ ] `make ci` → PASS

## Tasks

Step-by-step: Task 15 (S15) in plan.

- [ ] Branch: `feat/13-uikit-progress-bar`
- [ ] Write failing `progress_bar_test.go` (Step 15.1)
- [ ] Implement `progress_bar.go` with `partialGlyph` helper (Step 15.2)
- [ ] Migrate seek + volume in `components/controls.go` (Step 15.3)
- [ ] Update `controls_test.go` with ascii snapshots
- [ ] `make ci` → PASS
- [ ] Commit + push + open PR (Step 15.4)
