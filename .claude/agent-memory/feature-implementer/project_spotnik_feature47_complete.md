---
name: project_spotnik_feature47_complete
description: Feature 47 (Library Split): PlaylistsPane, AlbumsPane, LikedSongsPane — patterns, gotchas, test tips
type: project
---

## Feature 47 — Library Split

**What was built:**
- `internal/ui/panes/playlists_pane.go` — PlaylistsPane (layout.Pane, toggle key 3)
  - Playlist list with filter + track sub-view (Enter/Esc)
  - Management: n=create, r=rename, x=remove, Shift+↕=reorder
  - Columns: # 5% | Name 70% | Tracks 25% (flex 1:14:5)
  - Track sub-view columns: # 5% | Track 45% | Artist 35% | Duration 15% (flex 1:9:7:3)
- `internal/ui/panes/albums_pane.go` — AlbumsPane (layout.Pane, toggle key 4)
  - Columns: # 5% | Name 50% | Artist 30% | Year 15% (flex 1:10:6:3)
  - `extractYear()` helper for Spotify "YYYY-MM-DD" → "YYYY"
- `internal/ui/panes/likedsongs_pane.go` — LikedSongsPane (layout.Pane, toggle key 5)
  - Columns: # 5% | Track 45% | Artist 35% | Duration 15% (flex 1:9:7:3)
  - 'i' key emits LikeTrackRequestMsg{Unlike: true} (tracks in liked songs are already liked)

**Key files:**
- `internal/ui/panes/playlists_pane.go` — dual-table design (table + trackTable), inTrackView bool
- `internal/ui/panes/albums_pane.go` — simple table with filter, extractYear helper
- `internal/ui/panes/likedsongs_pane.go` — simple table with filter, like/unlike toggle

**Patterns established:**
- All three panes follow the QueuePane pattern (from F46): RefreshRows(), resizeTable(), filter pattern
- Both tables (list and track) allocated in constructor; only the active one is sized/focused
- `refreshPlaylistRows()` is only for list view; `refreshTrackRows()` is for track sub-view
- When filter closes, call `RefreshRows()` (not `refreshPlaylistRows()`) so it updates the active table

**Gotchas:**
- `refreshPlaylistRows()` inside the filter-close block caused a bug: when `inTrackView=true` and filter closes, it refreshed the wrong table. Fix: use `RefreshRows()` which delegates to the active table.
- When manually setting `inTrackView=true` in tests, also call `pane.trackTable.SetFocused(true)` — otherwise bubble-table doesn't respond to j/k (unfocused table ignores nav keys)
- When manually setting `inTrackView=true` in tests, also call `pane.resizeTable()` before checking View() — trackTable starts at 0 height if SetSize was called before entering track view
- Old LibraryPane and PlaylistManager files NOT deleted (deferred to Feature 49)

**Testing notes:**
- Final coverage: 87.7% for panes package, 85.3% total
- 52 new tests across 3 test files
- The `handleListViewKey` and `handleTrackViewKey` helper methods use the `key tea.KeyMsg` parameter for forwarding to tables — don't use `msg` (out of scope from these helpers)
- For ShiftUp/Down tests in track view: must manually focus trackTable with `pane.trackTable.SetFocused(true)` before pressing j to move cursor
