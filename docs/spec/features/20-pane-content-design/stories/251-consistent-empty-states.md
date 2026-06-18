---
title: "Add consistent EmptyState to panes that lack it"
feature: 20-pane-content-design
status: open
---

## Background

Five table panes render an empty table when the store has zero items and no filter is active. This wastes space and looks broken — an empty header row with nothing below. Per design rule §1.5, all table panes must check for zero data rows (with inactive filter) and render `uikit.EmptyState` instead.

**Panes that already have EmptyState** (verify, no changes needed):
- `queue.go:136-140` — already renders EmptyState when queue empty + filter inactive
- `recentlyplayed_pane.go:135` — already renders EmptyState
- `savedepisodes.go:121` — already renders EmptyState
- `followedshows.go:259` — already renders EmptyState

**Panes missing EmptyState** (must add):
- `likedsongs_pane.go` — renders empty table
- `toptracks_pane.go` — renders empty table
- `topartists_pane.go` — renders empty table
- `playlists_pane.go` — renders empty table in list view (track sub-view has data or not reached)
- `albums_pane.go` — renders empty table in list view

**Depends on:** Story 247 (# column removal) — EmptyState check is added to `View()` methods which have been refactored.

## Design

### Standard EmptyState pattern

Add this check at the top of each pane's `View()` method, before the table rendering path:

```go
func (p *PaneName) View() string {
    if len(p.store.SomeData()) == 0 && !p.Filter().IsActive() {
        return uikit.EmptyState{
            Text:   "No items message",
            Hint:   "Optional hint text",
            Width:  p.width,
            Height: p.height,
            Theme:  p.theme,
        }.Render()
    }
    // ... existing View logic
}
```

Filter-active with zero results shows the filter bar + empty table (not EmptyState) — user must see their search/filter query.

### Per-pane EmptyState messages

| Pane | Text | Hint |
|------|------|------|
| LikedSongs | `No liked songs` | `Press / to search for tracks` |
| TopTracks | `No top tracks` | `Listen to more music to populate this list` |
| TopArtists | `No top artists` | `Listen to more music to populate this list` |
| Playlists (list) | `No playlists` | `Create playlists in Spotify or search with /` |
| Albums (list) | `No saved albums` | `Save albums in Spotify or search with /` |

### Import uikit

If a pane doesn't already import `"github.com/initgrep-apps/spotnik/internal/uikit"`, add it.

## Files

### Modify

- `internal/ui/panes/likedsongs_pane.go` — add EmptyState check to `View()`
- `internal/ui/panes/toptracks_pane.go` — add EmptyState check to `View()`
- `internal/ui/panes/topartists_pane.go` — add EmptyState check to `View()`
- `internal/ui/panes/playlists_pane.go` — add EmptyState check to `View()` for list view only (`!p.inTrackView`)
- `internal/ui/panes/albums_pane.go` — add EmptyState check to `View()` for list view only (`!p.inTrackView`)

## Acceptance Criteria

- [ ] LikedSongs: EmptyState renders when `store.LikedTracks()` returns 0 and filter inactive
- [ ] TopTracks: EmptyState renders when `store.TopTracks()` returns 0 and filter inactive
- [ ] TopArtists: EmptyState renders when `store.TopArtists()` returns 0 and filter inactive
- [ ] Playlists: EmptyState renders in list view when `store.Playlists()` returns 0 and filter inactive
- [ ] Albums: EmptyState renders in list view when `store.Albums()` returns 0 and filter inactive
- [ ] EmptyState does NOT render when filter is active (filter bar + empty table shown instead)
- [ ] Table renders normally when data is present (existing behavior preserved)
- [ ] `go build ./...` compiles
- [ ] `make test` passes

## Tasks

- [ ] **Task 1: Add EmptyState to likedsongs_pane.go**
  Add check for `len(p.store.LikedTracks()) == 0 && !p.Filter().IsActive()` at top of `View()`. Render `uikit.EmptyState{Text: "No liked songs", Hint: "Press / to search for tracks", ...}`.
  - test: `go test ./internal/ui/panes/ -v -run "TestLikedSongs"` — all pass

- [ ] **Task 2: Add EmptyState to toptracks_pane.go**
  Add check for `len(p.store.TopTracks()) == 0 && !p.Filter().IsActive()` at top of `View()`.
  - test: `go test ./internal/ui/panes/ -v -run "TestTopTracks"` — all pass

- [ ] **Task 3: Add EmptyState to topartists_pane.go**
  Add check for `len(p.store.TopArtists()) == 0 && !p.Filter().IsActive()` at top of `View()`.
  - test: `go test ./internal/ui/panes/ -v -run "TestTopArtists"` — all pass

- [ ] **Task 4: Add EmptyState to playlists_pane.go**
  Add check for `!p.inTrackView && len(p.store.Playlists()) == 0 && !p.Filter().IsActive()` at top of `View()`.
  - test: `go test ./internal/ui/panes/ -v -run "TestPlaylists"` — all pass

- [ ] **Task 5: Add EmptyState to albums_pane.go**
  Add check for `!p.inTrackView && len(p.store.Albums()) == 0 && !p.Filter().IsActive()` at top of `View()`.
  - test: `go test ./internal/ui/panes/ -v -run "TestAlbums"` — all pass

- [ ] **Task 6: Run full test suite**
  - test: `make test` — all pass
