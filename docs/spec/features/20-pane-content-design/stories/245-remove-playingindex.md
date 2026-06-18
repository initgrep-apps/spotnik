---
title: "Remove PlayingIndex dead code from Table and all panes"
feature: 20-pane-content-design
status: open
---

## Background

The `PlayingIndex` mechanism was intended to show a `▶` playing indicator on the currently-playing row in table panes. In practice, `PlayingIndex` is always `-1` (disabled) in every pane except `SavedEpisodesPane`, where it was never visible per user testing. The `playingSymbol()` function and associated uikit import in `table.go` are dead code. The `SetPlayingIndex` method on `Table` and `QueuePane` is dead code. Removing this cruft cleans up ~40 lines across the table wrapper and simplifies all pane constructors.

**Dependencies:** None — this is the first story and unblocks all subsequent stories.

## Design

### Remove from `TableConfig`

Delete `PlayingIndex int` field and its doc comment from `TableConfig` struct (`table.go:58-59`).

### Remove from `Table` struct

Delete `playingIndex int` field from `Table` struct (`table.go:71`). Remove assignment `playingIndex: cfg.PlayingIndex,` from `NewTable` (`table.go:81`).

### Remove `SetPlayingIndex` method

Delete the entire method (`table.go:250-255`).

### Remove `playingSymbol()` function

Delete the function and its doc comment (`table.go:13-17`).

### Remove playing-row logic from `applyRows()`

Two code paths in `applyRows()` check `i == t.playingIndex` and inject a styled playing symbol into the first column:

- **Rich-rows path** (`table.go:153-161`): Remove the `if i == t.playingIndex { ... }` block
- **Plain-rows path** (`table.go:183-192`): Remove the `if i == t.playingIndex { ... }` block

After removal, `btRows[i] = btable.NewRow(data)` follows directly from `maps.Copy(data, rowData)` / the for-range loop.

### Remove uikit import

Remove `"github.com/initgrep-apps/spotnik/internal/uikit"` from imports in `table.go`. It was only used by `playingSymbol()`.

### Remove from callers

- `table_theme.go:25`: Remove `PlayingIndex: -1,` from the `TableConfig` literal in `RebuildTableTheme`.
- `table_chrome.go:29`: Remove `PlayingIndex: -1,` from the `TableConfig` literal in `NewTableChrome`'s `Update()` method.
- All 11 pane files: Remove `PlayingIndex: -1,` from every `NewTable(components.TableConfig{...})` call and every `SetTheme` method that constructs a `TableConfig`.
- `queue.go:88-91`: Remove `SetPlayingIndex` method on `QueuePane`.
- `savedepisodes.go:170-179`: Remove the `playingIdx` computation and `p.Table().SetPlayingIndex(playingIdx)` call from `buildRows()`.

### Remove from tests

- `table_test.go`: Delete `TestTable_SetPlayingIndex` (lines 135-156) and `TestTable_PlayingSymbol_AsciiMode` (lines 369-424) test functions.
- `table_test.go`: Remove `PlayingIndex: -1,` and `PlayingIndex: 0,` from all `TableConfig` literals (lines 39, 50, 63, 89, 108, 139, 162, 178, 215, 230, 292, 321, 347, 387, 410).
- `table_test.go`: Remove `"github.com/initgrep-apps/spotnik/internal/uikit"` import (only used in the deleted `TestTable_PlayingSymbol_AsciiMode`).
- `queue_test.go:330,597`: Remove `pane.SetPlayingIndex(0)` calls.

## Files

### Modify

- `internal/ui/components/table.go` — remove `PlayingIndex` field, `playingIndex` field, `SetPlayingIndex` method, `playingSymbol()` function, playing-row logic in `applyRows()`, uikit import
- `internal/ui/components/table_theme.go` — remove `PlayingIndex: -1` from `TableConfig` literal
- `internal/ui/components/table_chrome.go` — remove `PlayingIndex: -1` from `TableConfig` literal
- `internal/ui/components/table_test.go` — remove `TestTable_SetPlayingIndex`, `TestTable_PlayingSymbol_AsciiMode`, all `PlayingIndex` fields from `TableConfig` literals, uikit import
- `internal/ui/panes/queue.go` — remove `PlayingIndex: -1` from `NewQueuePane`, remove `SetPlayingIndex` method
- `internal/ui/panes/queue_test.go` — remove `SetPlayingIndex` calls
- `internal/ui/panes/savedepisodes.go` — remove `PlayingIndex: -1` from `TableConfig`, remove `playingIdx` computation and `SetPlayingIndex` call from `buildRows()`
- `internal/ui/panes/likedsongs_pane.go` — remove `PlayingIndex: -1`
- `internal/ui/panes/recentlyplayed_pane.go` — remove `PlayingIndex: -1`
- `internal/ui/panes/toptracks_pane.go` — remove `PlayingIndex: -1`
- `internal/ui/panes/topartists_pane.go` — remove `PlayingIndex: -1`
- `internal/ui/panes/playlists_pane.go` — remove `PlayingIndex: -1` from both `TableConfig` literals and `SetTheme`
- `internal/ui/panes/albums_pane.go` — remove `PlayingIndex: -1` from both `TableConfig` literals and `SetTheme`
- `internal/ui/panes/followedshows.go` — remove `PlayingIndex: -1` from both `TableConfig` literals and `SetTheme`
- `internal/ui/panes/networklog_pane.go` — remove `PlayingIndex: -1`
- `internal/ui/panes/gateway_live_pane.go` — remove `PlayingIndex: -1` from both `TableConfig` literals

## Acceptance Criteria

- [ ] `TableConfig` has no `PlayingIndex` field
- [ ] `Table` struct has no `playingIndex` field
- [ ] No `SetPlayingIndex` method on `Table` or `QueuePane`
- [ ] No `playingSymbol()` function
- [ ] No `uikit` import in `table.go`
- [ ] No `PlayingIndex` references in any `.go` file (zero `grep` matches)
- [ ] No playing-row logic in `applyRows()` (the string `t.playingIndex` has zero matches)
- [ ] `SavedEpisodesPane.buildRows()` ends with `p.Table().SetRows(rows)` — no `SetPlayingIndex` call
- [ ] `go build ./...` compiles without errors
- [ ] `go test ./internal/ui/components/...` passes (all `TestTable*` tests)
- [ ] `go test ./internal/ui/panes/... -run "TestQueue"` passes
- [ ] `make test` passes

## Tasks

- [ ] **Task 1: Remove PlayingIndex from table.go**
  Remove `PlayingIndex` from `TableConfig` and `Table` struct. Remove `playingSymbol()`. Remove `SetPlayingIndex`. Remove playing-row logic from `applyRows()`. Remove `uikit` import.
  - test: `go build ./internal/ui/components/` — no errors

- [ ] **Task 2: Remove PlayingIndex from table_theme.go and table_chrome.go**
  Remove `PlayingIndex: -1,` from both files' `TableConfig` literals.
  - test: `go build ./internal/ui/components/` — no errors

- [ ] **Task 3: Remove PlayingIndex from all pane TableConfig literals**
  Remove `PlayingIndex: -1,` from 11 pane files. For `QueuePane`, also remove the `SetPlayingIndex` method. For `SavedEpisodesPane`, remove the `playingIdx` computation and `SetPlayingIndex(playingIdx)` call.
  - test: `go build ./...` — no errors

- [ ] **Task 4: Update table_test.go**
  Delete `TestTable_SetPlayingIndex` and `TestTable_PlayingSymbol_AsciiMode`. Remove all `PlayingIndex` fields from `TableConfig` literals. Remove `uikit` import.
  - test: `go test ./internal/ui/components/ -v -run "TestTable"` — all pass

- [ ] **Task 5: Update queue_test.go**
  Remove `pane.SetPlayingIndex(0)` calls (lines 330, 597).
  - test: `go test ./internal/ui/panes/ -v -run "TestQueue"` — all pass

- [ ] **Task 6: Run full test suite**
  - test: `make test` — all pass
