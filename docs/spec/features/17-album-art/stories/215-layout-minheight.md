---
title: "Layout Manager: MinHeight guarantee for rows"
feature: 17-album-art
status: open
---

## Background

The Stats page (`PresetStats`) currently assigns the NowPlaying row
`HeightWeight: 1` in a 6-weight grid. At a 30-row terminal this yields ≈ 4 rows
for NowPlaying — too small for any image rendering and barely enough for the
compact title bar. The NowPlaying pane needs at least 14 rows (`bodyHeight` ≥ 10
after borders) to enter the base rendering tier introduced in story 217.

The fix is a new `MinHeight int` field on `Row`. LayoutManager reserves this
height first, then distributes remaining rows proportionally by weight. This
is fully backward-compatible: rows where `MinHeight == 0` behave exactly as
before.

## Design

### `internal/ui/layout/presets.go` — add `MinHeight` to `Row`

```go
// Row represents a horizontal strip of cells in the grid with its relative height.
type Row struct {
    HeightWeight int
    MinHeight    int // if > 0, this row is guaranteed at least MinHeight rows
    Cells        []Cell
}
```

### `internal/ui/layout/layout.go` — update height distribution

Replace the current single-pass weight-based distribution with a two-step algorithm:

```
reserved  = sum(row.MinHeight for all rows)
remaining = contentH - reserved          // may be negative — clamp to 0
totalW    = sum(row.HeightWeight for all rows)

for each row:
    share     = 0
    if totalW > 0:
        share = remaining * row.HeightWeight / totalW
    rowHeight = row.MinHeight + share
```

Last row still gets the rounding remainder so that `sum(rowHeights) == contentH`
exactly. When `remaining < 0` (terminal too small to honour all MinHeights),
each row receives only its `MinHeight` and the last row absorbs the deficit —
this is the same overflow behaviour as the current algorithm.

### `internal/ui/layout/presets.go` — update `PresetStats`

```go
var PresetStats = Preset{
    ...
    Grid: []Row{
        {HeightWeight: 1, MinHeight: 14, Cells: []Cell{  // ← MinHeight added
            {PaneID: PaneNowPlaying, WidthWeight: 1},
        }},
        {HeightWeight: 3, Cells: []Cell{
            {PaneID: PaneGatewayHealth, WidthWeight: 1},
            {PaneID: PanePollingTraffic, WidthWeight: 1},
            {PaneID: PaneGatewayLive, WidthWeight: 3},
        }},
        {HeightWeight: 2, Cells: []Cell{
            {PaneID: PaneNetworkLog, WidthWeight: 1},
        }},
    },
}
```

**Height breakdown at a 30-row terminal** (`contentH = 26`):

| Row | MinHeight | Weight share (rem=12, totalW=6) | Final height |
|---|---|---|---|
| NowPlaying | 14 | +2 (1/6 × 12) | **16** |
| Gateway row | 0 | +6 (3/6 × 12) | **6** |
| NetworkLog | 0 | +4 (2/6 × 12) | **4** |
| **Total** | | | **26** ✓ |

NowPlaying at 16 rows → `bodyHeight ≈ 12` → enters mid tier in story 217.

At a 50-row terminal (`contentH = 46`, reserved=14, remaining=32):
- NowPlaying: 14 + 5 = **19** → mid tier
- Gateway: 0 + 16 = **16**
- NetworkLog: 0 + 11 = **11** (last row absorbs rounding remainder)

## Acceptance Criteria

- [ ] `Row.MinHeight int` field exists in `presets.go`
- [ ] LayoutManager distributes heights using the two-step algorithm
- [ ] `PresetStats` NowPlaying row has `MinHeight: 14`
- [ ] At 30-row terminal: NowPlaying gets 16 rows on Stats page
- [ ] At 50-row terminal: NowPlaying gets 19 rows on Stats page
- [ ] Music page presets (no MinHeight set) distribute heights identically to before
- [ ] `make ci` passes

## Tasks

- [ ] Add `MinHeight int` to `Row` struct in `internal/ui/layout/presets.go`
      - test: `go build ./internal/ui/layout/...` compiles; existing preset
        definitions (no MinHeight field set) continue to compile with zero value

- [ ] Update height distribution in `internal/ui/layout/layout.go` to the
      two-step algorithm (reserve MinHeight first, distribute remaining by weight)
      - test: table-driven `TestLayoutManager_MinHeight`:
        - 3 rows, middle has MinHeight=10, contentH=30 → middle gets exactly 10 + proportional share
        - all rows MinHeight=0 → identical output to current algorithm (regression guard)
        - MinHeight sum > contentH → last row absorbs overflow, no panic

- [ ] Set `MinHeight: 14` on the NowPlaying row in `PresetStats`
      in `internal/ui/layout/presets.go`
      - test: `TestPresetStats_NowPlayingMinHeight` — construct a LayoutManager
        with `contentH = 26`; call `Distribute(PresetStats, contentH, fullWidth)`;
        assert NowPlaying rect height == 16

- [ ] `make ci` passes
