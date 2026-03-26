---
name: project_spotnik_feature41_complete
description: Feature 41 (Layout Infrastructure): new internal/ui/layout package, Manager struct, preset system, focus rotation, PaneAt hit-test
type: project
---

## Feature 41 — Layout Infrastructure

**What was built:**
- `internal/ui/layout/pane.go` — PaneID (10 iota constants), PageID (2), Rect struct with ContentWidth/ContentHeight, Action struct, Pane interface (extends tea.Model with 7 methods)
- `internal/ui/layout/presets.go` — Cell, Row, Preset data structs; 5 preset variables (Dashboard, Listening, Library, Discovery, NerdStatus); PageAPresets/PageBPresets slices
- `internal/ui/layout/layout.go` — Manager struct with full space distribution algorithm, page toggle, preset cycling, pane toggling (with last-pane guard), focus rotation, PaneAt hit-test

**Key files:**
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/ui/layout/layout.go` — core Manager; all layout logic in one file (372 lines)
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/ui/layout/pane.go` — types and interface
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/ui/layout/presets.go` — preset definitions

**Patterns established:**
- `Manager.recompute()` is the single truth point — called after every state change (Resize, TogglePage, CyclePreset, TogglePane). Never call layout logic directly, always go through recompute().
- `focusOrder []PaneID` is rebuilt on every recompute() — row-by-row, left-to-right within each row. This is the canonical source for VisiblePanes() and all focus operations.
- `rects map[PaneID]Rect` is rebuilt on every recompute(). `IsPaneVisible(id)` checks if `id` is in `rects`. `PaneRect(id)` returns zero Rect if not present (Go map zero-value).
- `prevFocused` captured from OLD focusOrder before update — then `restoreFocus()` searches new order. This preserves focus across recomputes.
- Space distribution: last row/cell absorbs remainder via `contentH - y` or `m.width - x`. Guarantees no pixel gaps.
- Page B panes (PaneRequestFlow, PaneNetworkLog) are identified by `id >= PaneRequestFlow` (iota 8+). TogglePane() uses this to guard.
- `TogglePane` last-pane guard: counts `visibleAfter` by iterating preset.Grid, excluding the target pane. If 0, rejects toggle.

**PaneAt coordinate system:**
- Terminal coordinates (0-based from top-left of full terminal)
- Header occupies y=0 (headerHeight=1), content starts at y=1
- Status bar occupies y=height-1 (statusHeight=1)
- PaneAt adjusts: `contentY = y - headerHeight`, then checks rect membership
- Returns PaneID(-1) for header, status bar, or outside any pane

**Gotchas:**
- `recompute()` defines local types (`activeCell`, `activeRow`, `rowLayout`) inside the function body. Go allows this for types that are purely implementation details. The linter accepts it.
- The `clampFocusIndex()` method is only used in the `len(activeRows)==0` early-return path. The normal path uses `restoreFocus()`. This is slightly inconsistent but correct.
- `PageBPresets` has only 1 entry. `CyclePreset()` on Page B wraps: (0+1) % 1 = 0. So it cycles back to preset 0 immediately. This is correct behavior (Page B has only one layout).
- PaneAt when `m.height == 0`: `m.height - m.statusHeight = -1`, check `y >= -1` always true → returns -1 for everything. Correct defensive behavior.

**Testing notes:**
- 43 tests across 3 test files; all in `layout_test` package (external test package)
- `TestResize_RectsNonOverlapping` — O(n²) pairwise overlap check; important for correctness
- `TestResize_TilesContentArea` — verifies all rects stay within bounds
- `TestResize_HeightWeightDistribution` — pin-tests specific pixel values for weight 2:3:3 at 28 content rows
- `TestResize_LastCellAbsorbsWidthRemainder` — uses odd width (121) to force remainder
- `TestRowCollapseHeightRedistributed` — hides all 3 panes in row 2, verifies rows 1+3 sum to contentH
- Coverage: 85.6% on layout package; 84.1% overall
- Edge cases: zero-size and 1x1 terminal — tested for no panics only

**Architecture boundary:**
- `internal/ui/layout/` imports ONLY `github.com/charmbracelet/bubbletea` (for the Pane interface)
- No imports from `app/`, `api/`, `state/`, or any other internal package
- Features 42-53 build ON TOP of this package
