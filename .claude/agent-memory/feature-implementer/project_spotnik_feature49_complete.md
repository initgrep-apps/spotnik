---
name: project_spotnik_feature49_complete
description: Feature 49 (App Migration): LayoutManager integration, viewMode consolidation, grid rendering, key routing, test update patterns
type: project
---

## Feature 49 — App Migration

**What was built:**
- Replaced viewMode (viewMain/viewStats/viewPlaylists) → viewGrid only
- Removed focusedPane enum; replaced with layout.Manager.FocusedPane()
- Added `panes map[layout.PaneID]layout.Pane` field with all 8 Page A panes
- Implemented renderGrid() using VisiblePanes() + PaneRect() + RenderPaneBorder()
- Wired '0'=page toggle, 'p'=preset cycle, '1'-'8'=pane toggle in routing.go
- Wired Tab/Shift+Tab focus rotation through layout.Manager.RotateFocus()
- Propagated WindowSizeMsg via layout.Resize() + propagateSizes() + syncFocus()
- Routed all data-loaded messages to correct new split panes
- Updated minimum terminal size from 100×24 to 120×30
- Removed renderPaneWithBorder() — replaced by layout.RenderPaneBorder()
- Updated all tests to use new exported API

**Key files:**
- `internal/app/app.go` — constructor, panes map, propagateSizes(), syncFocus(), exported focus helpers
- `internal/app/render.go` — renderGrid(), groupPanesByRow(), renderTooSmall() (120×30), gridHints()
- `internal/app/routing.go` — toggleKeyMap, handleKeyMsg with '0'/'p'/'1'-'8'/Tab/Shift+Tab
- `internal/app/app_test.go` — comprehensive updated tests with new API
- `internal/app/render_test.go` — new tests for renderGrid, renderTooSmall, gridHints
- `internal/app/auth_transition_test.go` — viewGrid instead of viewMain

**Patterns established:**
- `propagateSizes()`: iterates visible panes, calls pane.SetSize(rect.ContentWidth(), rect.ContentHeight())
- `syncFocus()`: sets IsFocused on all panes based on layout.FocusedPane()
- Focus rotation requires prior Resize() — layout.Manager.focusOrder is nil before first Resize()
- Helper accessors: nowPlayingPane(), queuePane(), playlistsPane(), albumsPane(), etc. return typed pointers
- Exported focus methods: NowPlayingFocused(), QueueFocused(), PlaylistsFocused(), FocusedPane()

**Gotchas:**
- `layout.Manager.RotateFocus()` / `FocusedPane()` is a no-op until `Resize()` is called (focusOrder nil). Tests that test Tab rotation MUST send `WindowSizeMsg{Width: 160, Height: 50}` first.
- `buildView()` splash fallthrough: if `currentView == viewSplash` AND `a.width == 0`, falls through to grid for tests. Sending `WindowSizeMsg` causes the real splash to render. Tests using `View()` should NOT send WindowSizeMsg if they want the grid fallthrough.
- `GridViewOpen()` returns false until `splashDismissMsg` fires (transitions viewSplash→viewGrid). `splashDismissMsg` is unexported, so external tests (package app_test) can't use it directly. Use `GridViewOpen()` only after auth flow or explicitly transition.
- `LibraryLoadedMsg.Items` is `[]domain.SimplePlaylist` (= `api.SimplePlaylist` type alias). Use `api.SimplePlaylist` in tests.
- '2' key now toggles QueuePane, '3' toggles PlaylistsPane. Old tests that checked StatsViewOpen()/PlaylistViewOpen() after pressing '2'/'3' must be replaced.
- 'p' key removed from isPlaybackKey() — now used for preset cycling, not "previous track"
- `panes.PlaylistsLoadedMsg` does not exist. Use `panes.LibraryLoadedMsg{Items: []api.SimplePlaylist{...}}` to send playlists to PlaylistsPane.

**Testing notes:**
- All focus rotation tests need `WindowSizeMsg{Width: 160, Height: 50}` before Tab/Shift+Tab
- Tests checking `View()` output without WindowSizeMsg get header+empty grid+status bar (no pane content since VisiblePanes() returns empty without Resize)
- Two-pass pattern for toasts: cmd = a.Update(errorMsg); alertMsg = cmd(); _, alertCmd = a.Update(alertMsg); then check View()
- Coverage: 85.6% (well above 80% threshold)
