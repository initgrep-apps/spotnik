---
title: "Fix table page size calculation and add width threshold crossing"
feature: 20-pane-content-design
status: open
---

## Background

The `SetSize()` method in `table.go:208` calculates page size as `height - 6`, while `rebuild()` at `table.go:122` uses `height - 4`. This 2-row discrepancy causes blank space between the pane border and the first table row, and between the last table row and the bottom border. The fix unifies both code paths to `height - 4` (with header) and `height - 2` (without header).

Additionally, `SetSize` must detect when pane width crosses column priority thresholds (40 and 60 terminal columns) and trigger a `rebuild()` to re-filter the column set. This is needed by story 249 (Priority system) but the infrastructure is added here since `SetSize` is being modified anyway.

**Dependencies:** None — independent of story 245, can run in parallel.

## Design

### Fix page size formula in `SetSize()`

```go
func (t *Table) SetSize(width, height int) {
    oldWidth := t.width
    t.width = width
    t.height = height

    // Rebuild if width crossed a priority threshold — column set may change.
    if crossesThreshold(oldWidth, width) {
        t.rebuild()
        return
    }

    t.inner = t.inner.WithTargetWidth(width)

    pageSize := height - 4 // was: height - 6
    if !t.config.ShowHeader {
        pageSize = height - 2 // was: height - 4
    }
    if pageSize < 1 {
        pageSize = 1
    }
    t.inner = t.inner.WithPageSize(pageSize)
}
```

### Add `crossesThreshold()` helper

```go
// crossesThreshold reports whether oldW and newW fall on opposite sides of a
// column-priority width threshold (40 or 60 terminal columns).
func crossesThreshold(oldW, newW int) bool {
    if (oldW < 40 && newW >= 40) || (oldW >= 40 && newW < 40) {
        return true
    }
    if (oldW < 60 && newW >= 60) || (oldW >= 60 && newW < 60) {
        return true
    }
    return false
}
```

### Update test comments

Twelve test files reference `height - 6` in comments explaining page size. Update these to `height - 4` to match the new formula:

- `table_test.go:234`
- `albums_pane_test.go:631`
- `followedshows_test.go:` (check)
- `likedsongs_pane_test.go:352`
- `queue_test.go:721`
- `toptracks_pane_test.go:317`
- `networklog_pane_test.go:639`
- `playlists_pane_test.go:950`
- `recentlyplayed_pane_test.go:250`
- `topartists_pane_test.go:344`

Also check `savedepisodes_test.go` for similar comment.

## Files

### Modify

- `internal/ui/components/table.go` — fix page size formula in `SetSize()`, add `crossesThreshold()` function, add threshold-crossing check
- `internal/ui/components/table_test.go` — update `height - 6` comment
- `internal/ui/panes/albums_pane_test.go` — update `height - 6` comment
- `internal/ui/panes/likedsongs_pane_test.go` — update `height - 6` comment
- `internal/ui/panes/queue_test.go` — update `height - 6` comment
- `internal/ui/panes/toptracks_pane_test.go` — update `height - 6` comment
- `internal/ui/panes/networklog_pane_test.go` — update `height - 6` comment
- `internal/ui/panes/playlists_pane_test.go` — update `height - 6` comment
- `internal/ui/panes/recentlyplayed_pane_test.go` — update `height - 6` comment
- `internal/ui/panes/topartists_pane_test.go` — update `height - 6` comment

## Acceptance Criteria

- [ ] `SetSize()` page size formula is `height - 4` (header) / `height - 2` (no-header), matching `rebuild()`
- [ ] No blank rows visible between pane border and first table row
- [ ] `crossesThreshold()` correctly detects crossings at 40 and 60 column thresholds
- [ ] `SetSize()` calls `rebuild()` when width crosses threshold, preserves old behavior otherwise
- [ ] Comment references to `height - 6` updated to `height - 4` across test files
- [ ] `go build ./internal/ui/components/` compiles
- [ ] `go test ./internal/ui/components/ -v -run "TestTable"` passes
- [ ] `make test` passes

## Tasks

- [ ] **Task 1: Fix page size formula and add threshold crossing in SetSize()**
  Change `pageSize := height - 6` to `pageSize := height - 4` in `SetSize()`. Change no-header path from `height - 4` to `height - 2`. Add `crossesThreshold()` function. Add threshold-crossing check before existing `WithTargetWidth`/`WithPageSize` calls.
  - test: `go build ./internal/ui/components/` — no errors
  - test: `go test ./internal/ui/components/ -v -run "TestTable_GotoTop"` — still passes (page count should be consistent)

- [ ] **Task 2: Update test comments referencing height - 6**
  Search all `*_test.go` files for `height - 6` comments. Replace with `height - 4`. Do not change test logic — comments only.
  - test: `go test ./internal/ui/components/... ./internal/ui/panes/...` — all pass

- [ ] **Task 3: Run full test suite**
  - test: `make test` — all pass
