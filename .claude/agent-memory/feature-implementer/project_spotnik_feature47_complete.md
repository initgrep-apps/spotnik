---
name: project_spotnik_feature47_complete
description: Feature 47 (Library Split): PlaylistsPane, AlbumsPane, LikedSongsPane — patterns, gotchas, test tips
type: project
---

## Feature 47 — Library Split

**What was built:**
- `internal/ui/panes/playlists_pane.go` — PlaylistsPane (layout.Pane, toggle key 3)
  - Playlist list w/ filter + track sub-view (Enter/Esc)
  - Mgmt: n=create, r=rename, x=remove, Shift+↕=reorder
  - Cols: # 5% | Name 70% | Tracks 25% (flex 1:14:5)
  - Track sub-view cols: # 5% | Track 45% | Artist 35% | Duration 15% (flex 1:9:7:3)
- `internal/ui/panes/albums_pane.go` — AlbumsPane (layout.Pane, toggle key 4)
  - Cols: # 5% | Name 50% | Artist 30% | Year 15% (flex 1:10:6:3)
  - `extractYear()` helper: Spotify "YYYY-MM-DD" → "YYYY"
- `internal/ui/panes/likedsongs_pane.go` — LikedSongsPane (layout.Pane, toggle key 5)
  - Cols: # 5% | Track 45% | Artist 35% | Duration 15% (flex 1:9:7:3)
  - 'i' emits LikeTrackRequestMsg{Unlike: true} (liked tracks already liked)

**Key files:**
- `internal/ui/panes/playlists_pane.go` — dual-table (table + trackTable), inTrackView bool
- `internal/ui/panes/albums_pane.go` — simple table w/ filter, extractYear helper
- `internal/ui/panes/likedsongs_pane.go` — simple table w/ filter, like/unlike toggle

**Patterns established:**
- All 3 panes follow QueuePane pattern (F46): RefreshRows(), resizeTable(), filter pattern
- Both tables (list + track) allocated in ctor; only active one sized/focused
- `refreshPlaylistRows()` list view only; `refreshTrackRows()` track sub-view only
- Filter close → call `RefreshRows()` (not `refreshPlaylistRows()`) → updates active table

**Gotchas:**
- `refreshPlaylistRows()` in filter-close block = bug: when `inTrackView=true` + filter closes, refreshed wrong table. Fix: `RefreshRows()` delegates to active table.
- Tests setting `inTrackView=true` manually: also call `pane.trackTable.SetFocused(true)` — else bubble-table ignores j/k (unfocused table = no nav keys)
- Tests setting `inTrackView=true` manually: also call `pane.resizeTable()` before View() — trackTable=0 height if SetSize called pre track view
- Old LibraryPane + PlaylistManager files NOT deleted (deferred Feature 49)

**Testing notes:**
- Final coverage: 87.7% panes pkg, 85.3% total
- 52 new tests across 3 test files
- `handleListViewKey` + `handleTrackViewKey` helpers use `key tea.KeyMsg` param for forwarding to tables — don't use `msg` (out of scope)
- ShiftUp/Down tests in track view: must manually focus trackTable via `pane.trackTable.SetFocused(true)` before j to move cursor