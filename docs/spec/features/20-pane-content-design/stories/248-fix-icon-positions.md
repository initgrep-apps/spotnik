---
title: "Fix icon column positions in SavedEpisodes, FollowedShows, Queue"
feature: 20-pane-content-design
status: done
---

## Background

Icon/glyph columns are inconsistently positioned across panes:
- **SavedEpisodes:** `icon` at position 6 (last), after `saved` and `duration` — should be first
- **FollowedShows show list:** `media` at position 5 (last) — should be first
- **FollowedShows episode view:** `icon` at position 5 (last) — should be first
- **Queue:** Trailing empty `icon` column (always `""` in data) — dead space, should be removed

Icons convey playability status (played, unplayed, in-progress) and must be visually scannable. The design rule from the spec §1.1 mandates: `[Icon/Glyph] → [Primary Identifier] → [Secondary Info] → [Tertiary/Metadata]`.

Additionally, the `Saved` column in SavedEpisodes is redundant — all items on this page are saved by definition. The column header and data should be removed to reclaim space, leaving `icon`, `episode`, `show`, and `duration`.

**Depends on:** Story 247 (# column removal) — icon positions shift after # column is gone. Implementing both in one pass avoids duplicate file editing.

## Design

### SavedEpisodes — column reorder + saved removal

```go
// Before (6 cols): [#, episode, show, saved, duration, icon]
// After # removed + reorder (4 cols):
columns := []components.ColumnDef{
    {Key: "icon", Header: "", FlexFactor: 1, Color: th.ColumnSecondary()},
    {Key: "episode", Header: "Episode", FlexFactor: 9, Color: th.ColumnPrimary()},
    {Key: "show", Header: "Show", FlexFactor: 6, Color: th.ColumnSecondary()},
    {Key: "duration", Header: "Duration", FlexFactor: 3, Color: th.ColumnTertiary()},
}
```

Remove `"saved": formatSavedDate(se.AddedAt),` from row data in `buildRows()`. The `formatSavedDate` function may become unused — verify and remove if so.

### FollowedShows show list — media to first

```go
// Before (5 cols): [#, show, publisher, eps, media]
// After # removed + reorder (4 cols):
showColumns := []components.ColumnDef{
    {Key: "media", Header: "", FlexFactor: 1, Color: th.ColumnSecondary()},
    {Key: "show", Header: "Show", FlexFactor: 10, Color: th.ColumnPrimary()},
    {Key: "publisher", Header: "Publisher", FlexFactor: 6, Color: th.ColumnSecondary()},
    {Key: "episodes", Header: "Eps", FlexFactor: 3, Color: th.ColumnTertiary()},
}
```

Row data keys don't change — only column definition order changes.

### FollowedShows episode view — icon to first

```go
// Before (5 cols): [#, title, released, duration, icon]
// After # removed + reorder (4 cols):
episodeColumns := []components.ColumnDef{
    {Key: "icon", Header: "", FlexFactor: 1, Color: th.ColumnSecondary()},
    {Key: "title", Header: "Title", FlexFactor: 9, Color: th.ColumnPrimary()},
    {Key: "released", Header: "Released", FlexFactor: 4, Color: th.ColumnSecondary()},
    {Key: "duration", Header: "Duration", FlexFactor: 3, Color: th.ColumnTertiary()},
}
```

### Queue — remove trailing empty icon column

```go
// Before (6 cols): [#, type, title, artist, duration, icon]
// After # removed + icon dropped (4 cols):
columns := []components.ColumnDef{
    {Key: "type", Header: "", FlexFactor: 1, Color: th.ColumnSecondary()},
    {Key: "title", Header: "Title", FlexFactor: 7, Color: th.ColumnPrimary()},
    {Key: "artist", Header: "Artist", FlexFactor: 4, Color: th.ColumnSecondary()},
    {Key: "duration", Header: "Duration", FlexFactor: 2, Color: th.ColumnTertiary()},
}
```

Remove `row["icon"] = ""` from both track and episode branches in `refreshRows()`.

### Update all `SetTheme` methods

Every pane's `SetTheme` method must have column definitions matching the new order.

## Files

### Modify

- `internal/ui/panes/savedepisodes.go` — move `icon` to position 1, remove `saved` column from defs and row data; update `SetTheme`
- `internal/ui/panes/followedshows.go` — move `media` to position 1 in show list, move `icon` to position 1 in episode view; update `SetTheme`
- `internal/ui/panes/queue.go` — remove trailing `icon` column from defs and row data; update `SetTheme`

## Acceptance Criteria

- [ ] SavedEpisodes: `icon` column is first (position 1), `saved` column removed
- [ ] FollowedShows show list: `media` column is first (position 1)
- [ ] FollowedShows episode view: `icon` column is first (position 1)
- [ ] Queue: no trailing `icon` column in defs or row data
- [ ] No `row["icon"] = ""` assignments in `queue.go`
- [ ] No `"saved": formatSavedDate(...)` in `savedepisodes.go`
- [ ] Icon glyphs render as first data column in all three panes
- [ ] `go build ./...` compiles
- [ ] `make test` passes

## Tasks

- [ ] **Task 1: Fix SavedEpisodes icon position + remove saved column**
  Reorder `icon` to first column. Remove `saved` column from defs and row data. Update `SetTheme` to match.
  - test: `go test ./internal/ui/panes/ -v -run "TestSavedEpisodes"` — all pass

- [ ] **Task 2: Fix FollowedShows icon positions**
  Move `media` to first position in show list columns. Move `icon` to first position in episode view columns. Update `SetTheme` to match.
  - test: `go test ./internal/ui/panes/ -v -run "TestFollowedShows"` — all pass

- [ ] **Task 3: Remove trailing empty icon column from Queue**
  Remove `icon` column def. Remove `row["icon"] = ""` from both row-building branches. Update `SetTheme` to match.
  - test: `go test ./internal/ui/panes/ -v -run "TestQueue"` — all pass

- [ ] **Task 4: Run full test suite**
  - test: `make test` — all pass
