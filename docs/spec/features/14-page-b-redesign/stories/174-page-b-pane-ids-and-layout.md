---
title: "Page B PaneID Constants and Preset Layout"
feature: 14-page-b-redesign
status: open
---

## Background

`PaneRequestFlow` is removed and replaced by three new PaneID constants. The `TogglePane`
guard that prevents Page B panes from being toggled via number keys must be updated from
`>= PaneRequestFlow` to `>= PaneNetworkLog`. The `PresetNerdStatus` grid is updated to
the new 3-row 5-pane layout.

These two tasks are paired in one story because the PaneID constants (Task 5) must be
committed before the preset update (Task 11) can reference the new constants.

**Source:** `docs/superpowers/plans/2026-04-26-page-b-redesign.md` Tasks 5, 11.

**Depends on:** Story 173 (Universal Esc Scroll Reset).

---

## Design

### Task 5 — Add new PaneID constants, update TogglePane guard

**Files to modify:** `internal/ui/layout/pane.go`, `internal/ui/layout/layout.go`,
`internal/ui/layout/layout_test.go`.

In `pane.go`, replace:
```go
PaneRequestFlow
PaneNetworkLog
```

With:
```go
PaneNetworkLog                   // Page B — not toggleable via number keys
PaneGatewayHealth                // Page B — not toggleable via number keys
PanePollingTraffic               // Page B — not toggleable via number keys
PaneGatewayLive                  // Page B — not toggleable via number keys
```

`PaneRequestFlow` is removed. `PaneNetworkLog` shifts down to occupy the old slot;
the three new panes follow it.

In `layout.go`, update the `TogglePane` guard:
```go
// Before:
if id >= PaneRequestFlow {
// After:
if id >= PaneNetworkLog {
```

After updating pane.go, `go build ./...` will emit compile errors for every remaining
reference to `PaneRequestFlow`. Fix each minimally so the build stays green:
- `presets.go` — temporarily replace `PaneRequestFlow` with `PaneGatewayHealth` (overwritten in Task 11)
- `border.go` — add a `default` or stub case (cleaned up in Story 177)
- `app.go` / `handlers.go` — add nil stub pane entries so the map compiles (replaced in Story 177)

Test: `TestPaneIDs_PageBConstants_AreDistinct` — verifies the 4 Page B PaneIDs are unique
and all `>= PaneNetworkLog`.

### Task 11 — Update PresetNerdStatus grid

**Files to modify:** `internal/ui/layout/presets.go`, `internal/ui/layout/presets_test.go`.

Replace the existing `PresetNerdStatus` variable:

```go
var PresetNerdStatus = Preset{
    Name: "Nerd Status",
    Visible: map[PaneID]bool{
        PaneNowPlaying: true, PaneGatewayHealth: true,
        PanePollingTraffic: true, PaneGatewayLive: true, PaneNetworkLog: true,
    },
    Grid: []Row{
        {HeightWeight: 1, Cells: []Cell{{PaneNowPlaying, 1}}},
        {HeightWeight: 3, Cells: []Cell{
            {PaneGatewayHealth, 1}, {PanePollingTraffic, 1}, {PaneGatewayLive, 2},
        }},
        {HeightWeight: 2, Cells: []Cell{{PaneNetworkLog, 1}}},
    },
}
```

Tests:
- `TestPresetNerdStatus_HasFivePanes` — asserts exactly 5 panes in `Visible`
- `TestPresetNerdStatus_GridHasThreeRows` — asserts row 2 has 3 cells with correct PaneIDs
  and width weights (GatewayHealth=1, PollingTraffic=1, GatewayLive=2)

---

## Acceptance Criteria

- [ ] `PaneGatewayHealth`, `PanePollingTraffic`, `PaneGatewayLive` constants exist in `pane.go`
- [ ] `PaneRequestFlow` constant does not exist
- [ ] `TogglePane` guard uses `>= PaneNetworkLog`
- [ ] `TestPaneIDs_PageBConstants_AreDistinct` passes
- [ ] `PresetNerdStatus` has 5 panes visible and a 3-row grid matching the spec
- [ ] `TestPresetNerdStatus_HasFivePanes` and `TestPresetNerdStatus_GridHasThreeRows` pass
- [ ] `go build ./...` is clean after each commit (minimal stubs acceptable before Story 177 wiring)
- [ ] `make ci` passes

## Tasks

- [ ] Add `PaneGatewayHealth`, `PanePollingTraffic`, `PaneGatewayLive` to `pane.go`; remove `PaneRequestFlow`
      - test: `TestPaneIDs_PageBConstants_AreDistinct` passes; `go build ./...` clean
- [ ] Update `TogglePane` guard in `layout.go` to `>= PaneNetworkLog`
      - test: existing layout tests still pass
- [ ] Update `PresetNerdStatus` in `presets.go` with the new 3-row 5-pane grid
      - test: `TestPresetNerdStatus_HasFivePanes` and `TestPresetNerdStatus_GridHasThreeRows` pass
