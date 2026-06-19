---
title: "Revert # column removal — restore index column to all panes"
feature: 20-pane-content-design
status: done
---

## Background

Story 247 removed the `#` (row-number) column from all 9 table panes. User testing revealed the visual result is worse without it — the index column provides navigational context that the pagination footer alone does not deliver. This story reverts the removal while preserving all other feature 20 improvements (PlayingIndex removal, page size fix, icon positions, priority system, headers, empty states).

**Root cause:** The `#` column was removed based on a speculative design rule ("wastes ~5% width") that turned out to be wrong — the small width cost is worth the UX benefit.

## Design

### Re-add index column definition to every pane

Each pane gets back:
```go
{Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex(), Priority: 1},
```

The column goes at **position 1** (before all other columns). `Priority: 1` ensures it is always visible — this also fixes the "bare 1-column at dashboard" problem (story 254).

### Re-add index to row data

Each row-building loop adds back:
```go
"index": fmt.Sprintf("%d", i+1),
```

### Re-add `"fmt"` import

Three panes had `fmt` removed when the index column was dropped (it was the sole consumer):
- `queue.go` — re-add `"fmt"` to imports
- `likedsongs_pane.go` — re-add `"fmt"` to imports
- `recentlyplayed_pane.go` — re-add `"fmt"` to imports

### Update SetTheme methods

Every pane's `SetTheme` method must include the index column definition matching the constructor.

### Column sets after reversion

| Pane | Columns (after reversion) | Total flex |
|------|--------------------------|------------|
| Queue | [#:1p1, type:1p1, title:7p1, artist:4p2, dur:2p3] | 15 |
| LikedSongs | [#:1p1, track:9p1, artist:7p2, dur:3p3] | 20 |
| RecentlyPlayed | [#:1p1, track:9p1, artist:7p2, played:3p3] | 20 |
| TopTracks | [#:1p1, track:9p1, artist:7p2, dur:3p3] | 20 |
| TopArtists | [#:1p1, name:11p1, pop:4p3, flw:4p3] | 20 |
| Playlists list | [#:1p1, access:1p1, name:13p1, tracks:5p3] | 20 |
| Playlists tracks | [#:1p1, track:10p1, artist:6p2, dur:3p3] | 20 |
| Albums list | [#:1p1, name:10p1, artist:6p2, year:3p3] | 20 |
| Albums tracks | [#:1p1, name:10p1, artist:6p2, dur:3p3] | 20 |
| SavedEpisodes | [#:1p1, icon:1p1, episode:9p1, show:6p2, dur:3p3] | 20 |
| FollowedShows shows | [#:1p1, media:1p1, show:10p1, pub:6p2, eps:3p3] | 21 |
| FollowedShows eps | [#:1p1, icon:1p1, title:9p1, released:4p2, dur:3p3] | 18 |

Note: NetworkLogPane and GatewayLivePane never had a `#` column — no change needed.

### Update tests

- `table_test.go`: re-add `{Key: "index", Header: "#", FlexFactor: 1}` to `makeColumns()` helper
- `table_test.go`: re-add `"index": "1"` to row data in all test literals
- `table_test.go`: update `TestTable_ColumnDefsHaveCorrectColors` — re-add `ColumnIndex()` assertion at `cols[0]`
- All pane test files: re-add `#` column to expected counts, re-add `ColumnIndex()` assertions
- `integration_test.go`: re-add `"index"` keys to test row data
- `table_theme.go`: re-add example comment referencing index column

## Files

### Modify

- `internal/ui/panes/queue.go` — re-add index column def, row data, fmt import; update SetTheme
- `internal/ui/panes/likedsongs_pane.go` — re-add index column def, row data, fmt import; update SetTheme
- `internal/ui/panes/recentlyplayed_pane.go` — re-add index column def, row data, fmt import; update SetTheme
- `internal/ui/panes/toptracks_pane.go` — re-add index column def, row data; update SetTheme
- `internal/ui/panes/topartists_pane.go` — re-add index column def, row data; update SetTheme
- `internal/ui/panes/playlists_pane.go` — re-add index column def (list + tracks), row data; update SetTheme
- `internal/ui/panes/albums_pane.go` — re-add index column def (list + tracks), row data; update SetTheme
- `internal/ui/panes/savedepisodes.go` — re-add index column def, row data; update SetTheme
- `internal/ui/panes/followedshows.go` — re-add index column def (shows + eps), row data; update SetTheme
- `internal/ui/components/table_test.go` — re-add index to makeColumns(), assertions, row data
- `internal/ui/components/integration_test.go` — re-add index keys to row data
- `internal/ui/components/table_theme.go` — re-add index comment

## Acceptance Criteria

- [ ] Index column (`{Key: "index", Header: "#", ...}`) present in all 9 panes (constructors + SetTheme)
- [ ] `"index": fmt.Sprintf("%d", i+1)` present in all row-building loops
- [ ] `fmt` import present in queue.go, likedsongs_pane.go, recentlyplayed_pane.go
- [ ] Index column at position 1 in every pane (before all other columns)
- [ ] Index column has `Priority: 1` (always visible)
- [ ] All other columns keep their existing Priority, FlexFactor, Header, Color values
- [ ] `ColumnIndex()` theme token used for index column color in all panes
- [ ] `go build ./...` compiles without errors
- [ ] `make ci` passes

## Tasks

- [ ] **Task 1: Revert index column in queue.go**
  Re-add index column def, `"index": fmt.Sprintf("%d", i+1)` to both track/episode row branches, re-add `"fmt"` import, update SetTheme.
  - test: `go test ./internal/ui/panes/ -v -run "TestQueue"`

- [ ] **Task 2: Revert index column in likedsongs_pane.go**
  Re-add index column def, row data key, `"fmt"` import, update SetTheme.
  - test: `go test ./internal/ui/panes/ -v -run "TestLikedSongs"`

- [ ] **Task 3: Revert index column in recentlyplayed_pane.go**
  Re-add index column def, row data key, `"fmt"` import, update SetTheme.
  - test: `go test ./internal/ui/panes/ -v -run "TestRecentlyPlayed"`

- [ ] **Task 4: Revert index column in toptracks_pane.go**
  Re-add index column def, row data key, update SetTheme.
  - test: `go test ./internal/ui/panes/ -v -run "TestTopTracks"`

- [ ] **Task 5: Revert index column in topartists_pane.go**
  Re-add index column def, row data key, update SetTheme.
  - test: `go test ./internal/ui/panes/ -v -run "TestTopArtists"`

- [ ] **Task 6: Revert index column in playlists_pane.go**
  Re-add index to both list and track column defs + row data, update SetTheme.
  - test: `go test ./internal/ui/panes/ -v -run "TestPlaylists"`

- [ ] **Task 7: Revert index column in albums_pane.go**
  Re-add index to both list and track column defs + row data, update SetTheme.
  - test: `go test ./internal/ui/panes/ -v -run "TestAlbums"`

- [ ] **Task 8: Revert index column in savedepisodes.go**
  Re-add index column def, row data key, update SetTheme.
  - test: `go test ./internal/ui/panes/ -v -run "TestSavedEpisodes"`

- [ ] **Task 9: Revert index column in followedshows.go**
  Re-add index to both show and episode column defs + row data, update SetTheme.
  - test: `go test ./internal/ui/panes/ -v -run "TestFollowedShows"`

- [ ] **Task 10: Update table_test.go, integration_test.go, table_theme.go**
  Re-add index to makeColumns(), color assertions, row data in table_test and integration_test. Re-add index comment in table_theme.go.
  - test: `go test ./internal/ui/components/ -v -run "TestTable"`

- [ ] **Task 11: Run full test suite**
  - test: `make test` — all pass
