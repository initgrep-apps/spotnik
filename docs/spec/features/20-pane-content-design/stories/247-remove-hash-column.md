---
title: "Remove # column from all table panes"
feature: 20-pane-content-design
status: open
---

## Background

Every table pane includes a `#` (row-number) column with `FlexFactor: 1` and `Color: th.ColumnIndex()`. This column wastes ~5% of pane width, and row numbering is already visible via the pagination footer ("Page X/Y"). Removing it frees space for content columns and simplifies column definitions across all 10 table panes plus `table_chrome.go`, `table_theme.go`, and `table_test.go`.

**Depends on:** Story 245 (PlayingIndex removal) — column definitions in panes currently reference `PlayingIndex: -1`, which must be removed first to avoid merge conflicts.

## Design

### Columns to remove

From every pane's column definition slice, remove:
```go
{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
```

### Row data to remove

From every row-building loop, remove:
```go
"index": fmt.Sprintf("%d", i+1),
```
If `i` becomes unused, replace with `_` or remove the loop index variable.

### Panes affected (with current column count)

| Pane | Before columns | After columns | Flex total |
|------|---------------|---------------|------------|
| Queue | [#, type, title, artist, duration, icon] | [type, title, artist, duration] | 16→14 |
| LikedSongs | [#, track, artist, duration] | [track, artist, duration] | 20→19 |
| RecentlyPlayed | [#, track, artist, played] | [track, artist, played] | 20→19 |
| TopTracks | [#, track, artist, dur] | [track, artist, dur] | 20→19 |
| TopArtists | [#, name, pop, flw] | [name, pop, flw] | 20→19 |
| Playlists list | [#, access, name, tracks] | [access, name, tracks] | 20→19 |
| Playlists tracks | [#, track, artist, duration] | [track, artist, duration] | 20→19 |
| Albums list | [#, name, artist, year] | [name, artist, year] | 20→19 |
| Albums tracks | [#, name, artist, duration] | [name, artist, duration] | 20→19 |
| SavedEpisodes | [#, episode, show, saved, duration, icon] | [episode, show, saved, duration, icon] | 23→22 |
| FollowedShows shows | [#, show, publisher, eps, media] | [show, publisher, eps, media] | 21→20 |
| FollowedShows eps | [#, title, released, duration, icon] | [title, released, duration, icon] | 18→17 |

Note: Icon position fixes (SavedEpisodes `saved` removal, FollowedShows `media`/`icon` move to first) are addressed in story 248.

### `SetTheme` methods

Every pane has a `SetTheme(th theme.Theme)` method that rebuilds the table with updated color tokens. These methods contain their own column definition slices. Update every `SetTheme` column definition to match the new column set (remove the `index` entry).

### `table_chrome.go`

`NewTableChrome` creates an inner `Table` with columns set from `t.Columns`. No explicit `index` column there — but verify no indirect reference.

### `table_theme.go`

`RebuildTableTheme` has a comment referencing `{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()}`. Update this comment to reflect the new column set.

### `table_test.go` — `makeColumns()` helper

```go
// Before:
func makeColumns() []components.ColumnDef {
    t := testTheme()
    return []components.ColumnDef{
        {Key: "index", Header: "#", FlexFactor: 1, Color: t.TextMuted()},
        {Key: "track", Header: "Track", FlexFactor: 4, Color: t.TextPrimary()},
        {Key: "artist", Header: "Artist", FlexFactor: 3, Color: t.TextSecondary()},
        {Key: "duration", Header: "Duration", FlexFactor: 2, Color: t.TextMuted()},
    }
}

// After:
func makeColumns() []components.ColumnDef {
    t := testTheme()
    return []components.ColumnDef{
        {Key: "track", Header: "Track", FlexFactor: 4, Color: t.TextPrimary()},
        {Key: "artist", Header: "Artist", FlexFactor: 3, Color: t.TextSecondary()},
        {Key: "duration", Header: "Duration", FlexFactor: 2, Color: t.TextMuted()},
    }
}
```

Update `TestTable_ColumnDefsHaveCorrectColors` assertions accordingly (was `cols[0]=TextMuted`, now `cols[0]=TextPrimary`).

### Row data in table_test.go

Every test that creates rows with `{"index": "1", ...}` must remove the `"index"` key. Find all such literals and update.

## Files

### Modify

- `internal/ui/panes/queue.go` — remove `index` column from defs + row data; update `SetTheme`
- `internal/ui/panes/likedsongs_pane.go` — remove `index` column from defs + row data; update `SetTheme`
- `internal/ui/panes/recentlyplayed_pane.go` — remove `index` column from defs + row data; update `SetTheme`
- `internal/ui/panes/toptracks_pane.go` — remove `index` column from defs + row data; update `SetTheme`
- `internal/ui/panes/topartists_pane.go` — remove `index` column from defs + row data; update `SetTheme`
- `internal/ui/panes/playlists_pane.go` — remove `index` column from both list+track defs + row data; update `SetTheme`
- `internal/ui/panes/albums_pane.go` — remove `index` column from both list+track defs + row data; update `SetTheme`
- `internal/ui/panes/savedepisodes.go` — remove `index` column from defs + row data; update `SetTheme`
- `internal/ui/panes/followedshows.go` — remove `index` column from both show+ep defs + row data; update `SetTheme`
- `internal/ui/panes/networklog_pane.go` — verify no index column (already correct)
- `internal/ui/panes/gateway_live_pane.go` — verify no index column (already correct)
- `internal/ui/components/table_theme.go` — update example column comment
- `internal/ui/components/table_test.go` — update `makeColumns()`, column assertions, row data literals

## Acceptance Criteria

- [ ] No `{Key: "index", Header: "#"...}` in any Go file (zero `grep` matches)
- [ ] No `"index": fmt.Sprintf(...)` in any pane file
- [ ] Pagination footer still shows "Page X/Y" (bubble-table built-in)
- [ ] All `SetTheme` methods have updated column definitions
- [ ] `go build ./...` compiles without errors
- [ ] `go test ./internal/ui/components/ -v -run "TestTable"` passes
- [ ] `make test` passes (all pane tests adapt to new column counts)

## Tasks

- [ ] **Task 1: Remove # from queue.go**
  Remove `index` column from column defs (constructor + SetTheme). Remove `"index": fmt.Sprintf(...)` from row data in both track and episode branches.
  - test: `go test ./internal/ui/panes/ -v -run "TestQueue"` — all pass

- [ ] **Task 2: Remove # from likedsongs_pane.go**
  Remove `index` column from defs (constructor + SetTheme). Remove `"index":` from row data.
  - test: `go test ./internal/ui/panes/ -v -run "TestLikedSongs"` — all pass

- [ ] **Task 3: Remove # from recentlyplayed_pane.go**
  Remove `index` column from defs (constructor + SetTheme). Remove `"index":` from row data.
  - test: `go test ./internal/ui/panes/ -v -run "TestRecentlyPlayed"` — all pass

- [ ] **Task 4: Remove # from toptracks_pane.go**
  Remove `index` column from defs (constructor + SetTheme). Remove `"index":` from row data.
  - test: `go test ./internal/ui/panes/ -v -run "TestTopTracks"` — all pass

- [ ] **Task 5: Remove # from topartists_pane.go**
  Remove `index` column from defs (constructor + SetTheme). Remove `"index":` from row data.
  - test: `go test ./internal/ui/panes/ -v -run "TestTopArtists"` — all pass

- [ ] **Task 6: Remove # from playlists_pane.go**
  Remove `index` from both list and track column defs (constructor + SetTheme). Remove `"index":` from both `refreshPlaylistRows` and `refreshTrackRows`.
  - test: `go test ./internal/ui/panes/ -v -run "TestPlaylists"` — all pass

- [ ] **Task 7: Remove # from albums_pane.go**
  Remove `index` from both list and track column defs (constructor + SetTheme). Remove `"index":` from both list and track row data.
  - test: `go test ./internal/ui/panes/ -v -run "TestAlbums"` — all pass

- [ ] **Task 8: Remove # from savedepisodes.go**
  Remove `index` column from defs (constructor + SetTheme). Remove `"index":` from row data in `buildRows()`.
  - test: `go test ./internal/ui/panes/ -v -run "TestSavedEpisodes"` — all pass

- [ ] **Task 9: Remove # from followedshows.go**
  Remove `index` from both show and episode column defs (constructor + SetTheme). Remove `"index":` from both `buildShowRows()` and `buildEpisodeRows()`.
  - test: `go test ./internal/ui/panes/ -v -run "TestFollowedShows"` — all pass

- [ ] **Task 10: Update table_theme.go and table_test.go**
  Update comment in `table_theme.go`. Update `makeColumns()`, `TestTable_ColumnDefsHaveCorrectColors` assertions, and all row data literals in `table_test.go`.
  - test: `go test ./internal/ui/components/ -v -run "TestTable"` — all pass

- [ ] **Task 11: Run full test suite**
  - test: `make test` — all pass
