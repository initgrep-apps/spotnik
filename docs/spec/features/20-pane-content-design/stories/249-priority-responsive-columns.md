---
title: "Add Priority field to ColumnDef and implement responsive column hiding"
feature: 20-pane-content-design
status: done
---

## Background

Dashboard preset panes (~30 cols) and Focus preset panes (~60 cols) currently show the same 4–6 columns, causing illegible truncation at narrow widths. There is no mechanism to adapt column visibility to available space.

This story adds a `Priority` field to `ColumnDef` and a `filterColumnsByPriority()` function that filters the column set at render time based on pane width. Column visibility changes when width crosses 40-col or 60-col thresholds (detected by `crossesThreshold()` from story 246).

**Depends on:** Story 247 (# column removal) + Story 248 (icon position fixes) — needs final column sets to assign correct priorities.

## Design

### `ColumnDef` — add `Priority` field

```go
type ColumnDef struct {
    Key        string
    Header     string
    FlexFactor int
    Priority   int // 0/1=always, 2=≥40cols, 3=≥60cols
    Color      lipgloss.Color
}
```

Zero-value (0) treated as Priority 1 — backward compatible with all existing constructors that don't set it.

### `filterColumnsByPriority()`

```go
func filterColumnsByPriority(cols []ColumnDef, width int) []ColumnDef {
    filtered := make([]ColumnDef, 0, len(cols))
    for _, c := range cols {
        switch c.Priority {
        case 0, 1:
            filtered = append(filtered, c)
        case 2:
            if width >= 40 {
                filtered = append(filtered, c)
            }
        case 3:
            if width >= 60 {
                filtered = append(filtered, c)
            }
        }
    }
    // Safety: ensure at least one column remains to avoid empty table crash.
    if len(filtered) == 0 && len(cols) > 0 {
        filtered = append(filtered, cols[0])
    }
    return filtered
}
```

### Integrate into `rebuild()`

Before building `btCols`, filter columns:

```go
func (t *Table) rebuild() {
    th := t.config.Theme
    activeCols := filterColumnsByPriority(t.config.Columns, t.width)

    btCols := make([]btable.Column, len(activeCols))
    for i, col := range activeCols {
        btCols[i] = btable.NewFlexColumn(col.Key, col.Header, col.FlexFactor).
            WithStyle(lipgloss.NewStyle().Foreground(col.Color).Align(lipgloss.Left))
    }
    // ... rest of rebuild() unchanged, using activeCols instead of t.config.Columns
}
```

### Update `Columns()` accessor

`Columns()` currently returns `t.config.Columns` (unfiltered). Change to return filtered columns so tests see current active set:

```go
func (t *Table) Columns() []ColumnDef {
    return filterColumnsByPriority(t.config.Columns, t.width)
}
```

### Priority assignments per pane

| Pane | Priority 1 | Priority 2 | Priority 3 |
|------|-----------|-----------|-----------|
| Queue | `type`, `title` | `artist` | `duration` |
| LikedSongs | `track` | `artist` | `duration` |
| RecentlyPlayed | `track` | `artist` | `played` |
| TopTracks | `track` | `artist` | `dur` |
| TopArtists | `name` | — | `pop`, `flw` |
| Playlists list | `access`, `name` | — | `tracks` |
| Playlists tracks | `track` | `artist` | `duration` |
| Albums list | `name` | `artist` | `year` |
| Albums tracks | `name` | `artist` | `duration` |
| SavedEpisodes | `icon`, `episode` | `show` | `duration` |
| FollowedShows shows | `media`, `show` | `publisher` | `eps` |
| FollowedShows eps | `icon`, `title` | `released` | `duration` |
| NetworkLog | (all p2) | all 7 cols | — |
| GatewayLive | `glyph`, `event` | — | — |

### Update all `SetTheme` methods

Every pane's `SetTheme` must include `Priority` values in column definitions, matching the constructor definitions exactly.

## Files

### Modify

- `internal/ui/components/table.go` — add `Priority` to `ColumnDef`, add `filterColumnsByPriority()`, integrate into `rebuild()`, update `Columns()` accessor
- `internal/ui/components/table_test.go` — add `TestTable_PriorityFiltering_*` tests
- `internal/ui/panes/queue.go` — add `Priority` to all column defs (constructor + SetTheme)
- `internal/ui/panes/likedsongs_pane.go` — add `Priority` to all column defs
- `internal/ui/panes/recentlyplayed_pane.go` — add `Priority` to all column defs
- `internal/ui/panes/toptracks_pane.go` — add `Priority` to all column defs
- `internal/ui/panes/topartists_pane.go` — add `Priority` to all column defs
- `internal/ui/panes/playlists_pane.go` — add `Priority` to both list + track column defs
- `internal/ui/panes/albums_pane.go` — add `Priority` to both list + track column defs
- `internal/ui/panes/savedepisodes.go` — add `Priority` to all column defs
- `internal/ui/panes/followedshows.go` — add `Priority` to both show + episode column defs
- `internal/ui/panes/networklog_pane.go` — add `Priority: 2` to all column defs
- `internal/ui/panes/gateway_live_pane.go` — add `Priority: 1` to both column defs

## Acceptance Criteria

- [ ] `ColumnDef` has `Priority int` field
- [ ] `filterColumnsByPriority()` filters columns by width thresholds (40, 60)
- [ ] Priority 0 (zero-value) treated as Priority 1 (always visible)
- [ ] Safety fallback: at least one column always remains
- [ ] `rebuild()` filters columns before building `btCols`
- [ ] `Columns()` returns priority-filtered column set
- [ ] All 14 pane column sets have correct `Priority` assignments per the table above
- [ ] Resize terminal across 40-col threshold: priority-2 columns appear/disappear
- [ ] Resize terminal across 60-col threshold: priority-3 columns appear/disappear
- [ ] `go test ./internal/ui/components/ -v -run "TestTable_Priority"` — 4 new tests pass
- [ ] `go build ./...` compiles
- [ ] `make test` passes

## Tasks

- [ ] **Task 1: Add Priority field and filterColumnsByPriority to table.go**
  Add `Priority int` to `ColumnDef`. Add `filterColumnsByPriority()` function. Integrate into `rebuild()` — filter `t.config.Columns` before building `btCols`. Update `Columns()` to return filtered columns.
  - test: `go build ./internal/ui/components/` — no errors

- [ ] **Task 2: Add Priority tests to table_test.go**
  Add four tests:
  - `TestTable_PriorityFiltering_HidesColumnsAtNarrowWidth` — verify only priority-1 cols at width=30
  - `TestTable_PriorityFiltering_MediumWidthShowsPriority2` — verify priority-1+2 at width=50
  - `TestTable_PriorityFiltering_WideWidthShowsAll` — verify all cols at width=80
  - `TestTable_PriorityFiltering_WidthThresholdCrossing` — verify columns appear/disappear across 40-col boundary
  - test: `go test ./internal/ui/components/ -v -run "TestTable_Priority"` — all pass

- [ ] **Task 3: Assign Priority to queue.go columns**
  Add `Priority` field to all 4 column defs in constructor + SetTheme.
  - test: `go test ./internal/ui/panes/ -v -run "TestQueue"` — all pass

- [ ] **Task 4: Assign Priority to likedsongs_pane.go columns**
  - test: `go test ./internal/ui/panes/ -v -run "TestLikedSongs"` — all pass

- [ ] **Task 5: Assign Priority to recentlyplayed_pane.go columns**
  - test: `go test ./internal/ui/panes/ -v -run "TestRecentlyPlayed"` — all pass

- [ ] **Task 6: Assign Priority to toptracks_pane.go columns**
  - test: `go test ./internal/ui/panes/ -v -run "TestTopTracks"` — all pass

- [ ] **Task 7: Assign Priority to topartists_pane.go columns**
  - test: `go test ./internal/ui/panes/ -v -run "TestTopArtists"` — all pass

- [ ] **Task 8: Assign Priority to playlists_pane.go columns**
  - test: `go test ./internal/ui/panes/ -v -run "TestPlaylists"` — all pass

- [ ] **Task 9: Assign Priority to albums_pane.go columns**
  - test: `go test ./internal/ui/panes/ -v -run "TestAlbums"` — all pass

- [ ] **Task 10: Assign Priority to savedepisodes.go columns**
  - test: `go test ./internal/ui/panes/ -v -run "TestSavedEpisodes"` — all pass

- [ ] **Task 11: Assign Priority to followedshows.go columns**
  - test: `go test ./internal/ui/panes/ -v -run "TestFollowedShows"` — all pass

- [ ] **Task 12: Assign Priority to networklog and gateway_live panes**
  NetworkLog: `Priority: 2` on all 7 columns. GatewayLive: `Priority: 1` on both columns.
  - test: `go test ./internal/ui/panes/ -v -run "TestNetworkLog|TestGatewayLive"` — all pass

- [ ] **Task 13: Run full test suite**
  - test: `make test` — all pass
