---
name: project_spotnik_story180_complete
description: Story 180 (Stacked Page B Layout): RowSpan support in layout engine, 4-row PresetNerdStatus, spanning algorithm
type: project
---

## Story 180 — Stacked Page B Layout (RowSpan Support)

**What was built:**
- `Cell.RowSpan` field and `rowSpan()` helper in `internal/ui/layout/presets.go`
- Rewrote `recompute()` in `internal/ui/layout/layout.go` with 5-step spanning algorithm
- Updated `PresetNerdStatus` to 4-row stacked grid in `internal/ui/layout/presets.go`
- Migrated all `Cell{}` positional literals to named-field style
- Added RowSpan tests in `internal/ui/layout/layout_test.go`
- Replaced `TestPresetNerdStatus_GridHasThreeRows` with `TestPresetNerdStatus_GridHasFourRows` in `presets_test.go`

**Key files:**
- `internal/ui/layout/presets.go` — Cell struct, rowSpan() helper, all preset definitions
- `internal/ui/layout/layout.go` — recompute() spanning algorithm
- `internal/ui/layout/layout_test.go` — RowSpan behavioral tests

**Algorithm pattern (5-step recompute):**
1. Build `ownCellsByRow[][]cellSpec` — visible cells per original grid row index
2. Build `spannerCoverageByRow[][]PaneID` — which rows are covered from above by a spanner
3. Compute live rows (has own cells OR has spanner coverage)
4. Distribute height among live rows
5. Spanning pass: for each origin row with spanners, assign X/W in declaration order; accumulate H across covered rows
6. Per-row placement: reserve spanner intervals BOTH from continuation-coverage AND from origin-row spanners; place non-spanners in remaining space

**Critical gotcha — two-phase reservation:**
In Step 4 (per-row placement), reserved intervals must include:
- (a) spanners from `rl.spannerCoverage` (originated in earlier rows), AND
- (b) spanners that ORIGINATE in the current row

Without (b), non-spanner cells in the origin row get placed across the full width instead of just the left portion.

**Preset structure for RowSpan:**
```go
// Continuation row has only non-spanner cells; spanner is NOT repeated
{HeightWeight: 2, Cells: []Cell{
    {PaneID: PaneGatewayHealth, WidthWeight: 1},
    {PaneID: PaneGatewayLive, WidthWeight: 3, RowSpan: 2}, // ORIGIN — span starts here
}},
{HeightWeight: 2, Cells: []Cell{
    {PaneID: PanePollingTraffic, WidthWeight: 1},
    // GatewayLive continuation — NO cell here, recompute() handles it
}},
```

**Testing notes:**
- `TestRecompute_RowSpan_GatewayLive` covers the main assertions from the spec
- Added two extra toggle tests beyond spec: ToggleHidingSpanner and ToggleHidingSpannedPanes
- Layout package coverage: 91.0% (up from 88.9%)
- All 88 layout tests pass, no regressions in Page A presets
