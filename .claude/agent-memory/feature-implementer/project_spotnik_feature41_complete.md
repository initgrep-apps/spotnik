---
name: project_spotnik_feature41_complete
description: Feature 41 (Layout Infrastructure): new internal/ui/layout package, Manager struct, preset system, focus rotation, PaneAt hit-test
type: project
---

## Feature 41 — Layout Infrastructure

**Built:**
- `internal/ui/layout/pane.go` — PaneID (10 iota consts), PageID (2), Rect struct w/ ContentWidth/ContentHeight, Action struct, Pane interface (extends tea.Model, 7 methods)
- `internal/ui/layout/presets.go` — Cell, Row, Preset structs; 5 preset vars (Dashboard, Listening, Library, Discovery, NerdStatus); PageAPresets/PageBPresets slices
- `internal/ui/layout/layout.go` — Manager struct, full space distribution algo, page toggle, preset cycling, pane toggling (last-pane guard), focus rotation, PaneAt hit-test

**Key files:**
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/ui/layout/layout.go` — core Manager; all layout logic, one file (372 lines)
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/ui/layout/pane.go` — types, interface
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/ui/layout/presets.go` — preset defs

**Patterns:**
- `Manager.recompute()` = single truth point — called after every state change (Resize, TogglePage, CyclePreset, TogglePane). Never call layout logic direct, always via recompute().
- `focusOrder []PaneID` rebuilt every recompute() — row-by-row, left-to-right per row. Canonical source for VisiblePanes() + all focus ops.
- `rects map[PaneID]Rect` rebuilt every recompute(). `IsPaneVisible(id)` checks `id` in `rects`. `PaneRect(id)` returns zero Rect if absent (Go map zero-value).
- `prevFocused` captured from OLD focusOrder pre-update — `restoreFocus()` searches new order. Preserves focus across recomputes.
- Space distribution: last row/cell absorbs remainder via `contentH - y` or `m.width - x`. No pixel gaps.
- Page B panes (PaneRequestFlow, PaneNetworkLog) identified by `id >= PaneRequestFlow` (iota 8+). TogglePane() guards on this.
- `TogglePane` last-pane guard: counts `visibleAfter` by iterating preset.Grid, excluding target pane. If 0, reject toggle.

**PaneAt coord system:**
- Terminal coords (0-based, top-left of full terminal)
- Header at y=0 (headerHeight=1), content starts y=1
- Status bar at y=height-1 (statusHeight=1)
- PaneAt adjusts: `contentY = y - headerHeight`, then checks rect membership
- Returns PaneID(-1) for header, status bar, or outside any pane

**Gotchas:**
- `recompute()` defines local types (`activeCell`, `activeRow`, `rowLayout`) inside func body. Go allows this for impl-detail types. Linter accepts.
- `clampFocusIndex()` only used in `len(activeRows)==0` early-return path. Normal path uses `restoreFocus()`. Slightly inconsistent but correct.
- `PageBPresets` has 1 entry. `CyclePreset()` on Page B wraps: (0+1) % 1 = 0. Cycles back to preset 0 immediately. Correct (Page B = one layout).
- PaneAt when `m.height == 0`: `m.height - m.statusHeight = -1`, check `y >= -1` always true → returns -1 for everything. Correct defensive behavior.

**Tests:**
- 43 tests across 3 files; all in `layout_test` pkg (external)
- `TestResize_RectsNonOverlapping` — O(n²) pairwise overlap check; correctness-critical
- `TestResize_TilesContentArea` — verifies all rects within bounds
- `TestResize_HeightWeightDistribution` — pins pixel values for weight 2:3:3 at 28 content rows
- `TestResize_LastCellAbsorbsWidthRemainder` — odd width (121) forces remainder
- `TestRowCollapseHeightRedistributed` — hides all 3 panes row 2, verifies rows 1+3 sum to contentH
- Coverage: 85.6% layout pkg; 84.1% overall
- Edge cases: zero-size, 1x1 terminal — tested for no-panic only

**Architecture boundary:**
- `internal/ui/layout/` imports ONLY `github.com/charmbracelet/bubbletea` (for Pane interface)
- No imports from `app/`, `api/`, `state/`, or other internal pkgs
- Features 42-53 build ON TOP