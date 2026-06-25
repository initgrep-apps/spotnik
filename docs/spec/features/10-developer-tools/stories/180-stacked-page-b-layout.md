---
title: "Fix: Stacked Page B layout — RowSpan support + 30/70 Health+Traffic vs GatewayLive"
feature: 10-developer-tools
status: done
---

## Background

The current Page B middle row puts Gateway Health, Polling Traffic, and Gateway Live
side-by-side in a single row with equal widths (1:1:1 bug; the planned 1:1:2 was never
committed correctly). The user wants a different layout:

- **Left column (30%):** GatewayHealth on top, PollingTraffic below — two separate bordered
  panes stacked vertically.
- **Right column (70%):** GatewayLive — a single tall pane spanning the full height of
  the left column.

```
╭─ ¹Now Playing ─────────────────────────────────────────────────────────────────────╮
│                                                                                     │
╰─────────────────────────────────────────────────────────────────────────────────────╯
╭─ ²Gateway Health ──────────╮ ╭─ ⁴Gateway Live ─────────────────────── f filter ───╮
│                            │ │                                                      │
╰────────────────────────────╯ │                                                      │
╭─ ³Polling Traffic ─────────╮ │                                                      │
│                            │ │                                                      │
╰────────────────────────────╯ ╰──────────────────────────────────────────────────────╯
╭─ ⑤Network Log ──────────────────────────────────────────────── f filter ────────────╮
│                                                                                      │
╰──────────────────────────────────────────────────────────────────────────────────────╯
```

**Root cause — LayoutManager has no RowSpan support.**
The grid system places panes into independent rows. A pane cannot span across multiple
rows. To achieve the layout above, GatewayLive must span rows 2 and 3 (GatewayHealth row
and PollingTraffic row). This requires a new `RowSpan` field on `Cell` and updated
`recompute()` logic.

**Quick fix also included:** `PresetNerdStatus` currently has `{PaneGatewayLive, 1}` —
the WidthWeight should be 3 (70% of the middle row, 1:3 ratio). This story corrects it
as part of the new preset definition.

---

## Design

### Task 1 — Add `RowSpan` field to `Cell`

**File:** `internal/ui/layout/pane.go`

Current:

```go
type Cell struct {
    PaneID      PaneID
    WidthWeight int
}
```

New:

```go
// Cell is one pane slot in a grid Row.
type Cell struct {
    PaneID      PaneID
    WidthWeight int
    // RowSpan is the number of rows this cell spans vertically (default 0 or 1 = one row).
    // A cell with RowSpan 2 occupies its own row plus the next row in its column position.
    RowSpan int
}
```

> All existing presets use struct literal initialisation with two positional fields
> (`{PaneID, WidthWeight}`). Add a `rowSpan() int` helper to Cell that returns `max(1, c.RowSpan)`
> so unset cells default to 1.

```go
func (c Cell) rowSpan() int {
    if c.RowSpan < 2 {
        return 1
    }
    return c.RowSpan
}
```

### Task 2 — Update `recompute()` to handle RowSpan

**File:** `internal/ui/layout/layout.go`

The updated algorithm:

1. **Pre-pass** — compute `rowY` and `rowH` for every active row (same as today).
2. **Spanning pass** — for each cell with `rowSpan() > 1` in row `i`:
   - Compute its `X` and `Width` from row `i` as normal.
   - Set its `Height` = sum of `rowH[i]` through `rowH[i + rowSpan - 1]`.
   - Record: `spannerX[paneID]`, `spannerW[paneID]` so continuation rows can reserve the same column.
3. **Per-row placement** — for row `i`, before placing cells:
   - Identify all spanning cells from earlier rows that still occupy row `i`.
   - Reserve their `[X, X+Width]` interval from the available horizontal space.
   - Place the row's own cells in the remaining horizontal space proportionally by WidthWeight.

**Continuation row placement detail:**

For each continuation row that has spanning cells occupying it, calculate the available
width by subtracting the spanning cells' widths. Compute remaining cells' rects proportionally.

```go
// pseudocode for per-row placement with span reservation
available := totalWidth
reservedIntervals := []interval{} // from spanning cells occupying this row

for each spanning cell from earlier rows occupying rowIndex i:
    reservedIntervals = append(reservedIntervals, {spannerX[id], spannerW[id]})
    available -= spannerW[id]

// Place row's own cells in the remaining available width
totalOwnWeight := sum of WidthWeight for own cells
x := 0
for each own cell in row left-to-right:
    // skip reserved intervals to find next free x
    x = nextFreeX(x, reservedIntervals)
    w := (cell.WidthWeight * available) / totalOwnWeight
    rects[cell.PaneID] = Rect{X: x, Y: rowY[i], Width: w, Height: rowH[i]}
    x += w
```

> The `nextFreeX` helper walks `x` forward past any reserved interval that starts at or
> before `x`. For the Page B preset (one spanning cell on the right), this simplifies to:
> place left cells normally; their combined width equals the reserved space's left edge.

### Task 3 — Update `PresetNerdStatus` to use the stacked layout

**File:** `internal/ui/layout/presets.go`

Replace the existing `PresetNerdStatus`:

```go
// PresetNerdStatus shows NowPlaying strip, GatewayHealth + PollingTraffic stacked on the left
// (30%), GatewayLive spanning full height on the right (70%), NetworkLog full-width below.
var PresetNerdStatus = Preset{
    Name: "Nerd Status",
    Visible: map[PaneID]bool{
        PaneNowPlaying:     true,
        PaneGatewayHealth:  true,
        PanePollingTraffic: true,
        PaneGatewayLive:    true,
        PaneNetworkLog:     true,
    },
    Grid: []Row{
        {HeightWeight: 1, Cells: []Cell{{PaneNowPlaying, 1, 0}}},
        {HeightWeight: 2, Cells: []Cell{
            {PaneGatewayHealth, 1, 0},
            {PaneGatewayLive, 3, 2}, // RowSpan 2: spans this row and the next
        }},
        {HeightWeight: 2, Cells: []Cell{
            {PanePollingTraffic, 1, 0},
            // GatewayLive continuation — no cell here; recompute() handles it
        }},
        {HeightWeight: 2, Cells: []Cell{{PaneNetworkLog, 1, 0}}},
    },
}
```

> The continuation row for PollingTraffic has only 1 cell. `recompute()` detects that
> GatewayLive (RowSpan=2) still occupies the right 75% of that row and places
> PollingTraffic in the left 25% at the correct X.

### Task 4 — Tests

**`internal/ui/layout/layout_test.go`** — `TestRecompute_RowSpan_GatewayLive`:

```go
func TestRecompute_RowSpan_GatewayLive(t *testing.T) {
    m := layout.NewManager()
    m.Resize(200, 80)
    m.TogglePage() // switch to Page B (PresetNerdStatus)

    healthRect  := m.PaneRect(layout.PaneGatewayHealth)
    trafficRect := m.PaneRect(layout.PanePollingTraffic)
    liveRect    := m.PaneRect(layout.PaneGatewayLive)
    netRect     := m.PaneRect(layout.PaneNetworkLog)

    // GatewayLive must be taller than GatewayHealth alone
    assert.Greater(t, liveRect.Height, healthRect.Height,
        "GatewayLive must span both Health and Traffic rows")

    // GatewayLive height must equal Health + Traffic combined height
    assert.Equal(t, healthRect.Height+trafficRect.Height, liveRect.Height,
        "GatewayLive height = Health + Traffic")

    // Health and Traffic must share the same left column (same X, same Width)
    assert.Equal(t, healthRect.X, trafficRect.X, "Health and Traffic must align left")
    assert.Equal(t, healthRect.Width, trafficRect.Width, "Health and Traffic must have equal width")

    // GatewayLive must start where Health ends (same row, to the right)
    assert.Equal(t, healthRect.X+healthRect.Width, liveRect.X,
        "GatewayLive must be immediately right of GatewayHealth")

    // NetworkLog must be full-width below all middle panes
    assert.Equal(t, 0, netRect.X)
    assert.Greater(t, netRect.Y, liveRect.Y, "NetworkLog is below GatewayLive")
}
```

**`internal/ui/layout/presets_test.go`** — update `TestPresetNerdStatus_GridHasThreeRows` to `TestPresetNerdStatus_GridHasFourRows` and add span assertion:

```go
func TestPresetNerdStatus_GridHasFourRows(t *testing.T) {
    require.Len(t, layout.PresetNerdStatus.Grid, 4)
    row2 := layout.PresetNerdStatus.Grid[1]
    require.Len(t, row2.Cells, 2)
    assert.Equal(t, layout.PaneGatewayHealth, row2.Cells[0].PaneID)
    assert.Equal(t, layout.PaneGatewayLive,   row2.Cells[1].PaneID)
    assert.Equal(t, 2, row2.Cells[1].RowSpan, "GatewayLive must span 2 rows")
    assert.Equal(t, 3, row2.Cells[1].WidthWeight, "GatewayLive must have weight 3 (70%)")
}
```

---

## Acceptance Criteria

- [ ] `Cell.RowSpan` field exists; `Cell.rowSpan()` helper returns 1 for unset cells
- [ ] `recompute()` correctly places a spanning cell across its declared row count
- [ ] `TestRecompute_RowSpan_GatewayLive` passes: GatewayLive height = Health + Traffic heights combined
- [ ] Health and Traffic share the same X and Width (same left column)
- [ ] GatewayLive starts immediately to the right of GatewayHealth
- [ ] `PresetNerdStatus` has 4 rows; GatewayLive has WidthWeight=3 and RowSpan=2
- [ ] All existing Page A layout and preset tests still pass (RowSpan=0 cells behave as before)
- [ ] Toggle keys from Story 179 work correctly with the new layout (hiding GatewayHealth
      or PollingTraffic shrinks the left column; hiding GatewayLive removes the right column
      and expands the left panes to full width)
- [ ] `make ci` passes

## Tasks

- [ ] Add `RowSpan int` to `Cell` and `rowSpan()` helper in `internal/ui/layout/pane.go`
      — verify existing presets still compile (two-field struct literals need updating to
      three-field or named-field style)
- [ ] Update `recompute()` in `internal/ui/layout/layout.go` to handle RowSpan
      — test: `TestRecompute_RowSpan_GatewayLive`
- [ ] Update `PresetNerdStatus` in `internal/ui/layout/presets.go` to 4-row stacked grid
      — test: `TestPresetNerdStatus_GridHasFourRows`; update/remove old 3-row test
- [ ] Update all existing `Cell` struct literals across the codebase from positional
      `{PaneID, Weight}` to named fields `{PaneID: ..., WidthWeight: ..., RowSpan: 0}`
      OR accept that Go zero-values the new field automatically (two-field positional
      literals become compile errors when a third field is added — must migrate all)
- [ ] `make ci` passes
