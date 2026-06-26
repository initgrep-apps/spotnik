---
title: "Library component + integration tests"
feature: 21-test-infrastructure
status: done
---

## Background

The Library feature (05) exposes three table panes — Playlists, Albums, LikedSongs — each
with filter support, Enter drill-down (playlist/album track sub-views), and Esc back
navigation. Current unit tests verify Update() logic but never assert the rendered output
after state transitions. A column priority change or header rename can break the visual
layout silently.

This story adds golden snapshots for all three panes in multiple states (loaded, empty,
filtered, drill-down) and integration tests for the drill-down lifecycle.

## Design

### Golden tests: `internal/ui/panes/playlists_golden_test.go`

Snapshots for PlaylistsPane:
- `TestPlaylistsPane_View_ListView` — loaded playlists, normal width (80×24)
- `TestPlaylistsPane_View_EmptyState` — no playlists, empty state message
- `TestPlaylistsPane_View_TrackSubView` — Enter on playlist, tracks shown
- `TestPlaylistsPane_View_TrackSubView_FilterActive` — filter activated during track sub-view
- `TestPlaylistsPane_View_SpotifyOwnedLocked` — playlist with `Owner.ID == "spotify"` shows locked glyph (◌)
- `TestPlaylistsPane_View_Narrow` — 40×24, verify column hiding
- `TestPlaylistsPane_View_FilterActive` — 'f' pressed, filter input bar visible with matches

### Golden tests: `internal/ui/panes/albums_golden_test.go`

Snapshots for AlbumsPane:
- `TestAlbumsPane_View_AlbumList` — loaded albums at 80×24
- `TestAlbumsPane_View_EmptyState` — no albums
- `TestAlbumsPane_View_TrackSubView` — Enter on album, tracks shown
- `TestAlbumsPane_View_TrackSubView_FilterActive` — filter active in album track sub-view
- `TestAlbumsPane_View_Narrow` — 40×24
- `TestAlbumsPane_View_FilterActive` — 'f' pressed, albums filtered by name

### Golden tests: `internal/ui/panes/likedsongs_golden_test.go`

Snapshots for LikedSongsPane:
- `TestLikedSongsPane_View_Tracks` — loaded songs at 80×24
- `TestLikedSongsPane_View_EmptyState` — no liked songs
- `TestLikedSongsPane_View_Narrow` — 40×24
- `TestLikedSongsPane_View_FilterActive` — 'f' pressed, songs filtered by name or artist

### Integration test: `internal/ui/panes/library_flow_test.go`

```go
func TestPlaylistsDrillDown_EnterThenEsc(t *testing.T) {
    // Setup: PlaylistsPane with 3 playlists, playlist 0 has tracks loaded in store
    // 1. Send Enter → assert track sub-view appears in View()
    // 2. Send Esc → assert list view restored
    // 3. Assert title changes correctly on Enter/Esc
}

func TestPlaylistsPane_TrackView_DeleteTrack(t *testing.T) {
    // Setup: PlaylistsPane in track sub-view, cursor on track 1
    // 1. Send 'x' → assert PlaylistRemoveRequestMsg cmd produced
    // 2. Assert cmd carries correct PlaylistID and TrackURI
}

func TestPlaylistsPane_EnterOnSpotifyOwned_EmitsAccessDenied(t *testing.T) {
    // Setup: cursor on playlist with Owner.ID == "spotify"
    // 1. Send Enter → assert PlaylistAccessDeniedMsg cmd produced
    // 2. Assert sub-view does NOT open
}
```

### Data setup

Each test populates `state.Store` with realistic domain data:
- Playlists: 3 `SimplePlaylist` with name, track count, owner
- Albums: 3 `SavedAlbum` with album name, artist, release year
- LikedSongs: 3 `SavedTrack` with name, artist, duration
- For drill-down: `state.Store` receives `PlaylistTracksLoadedMsg` / `AlbumTracksLoadedMsg` via pane Update()

## Files

### Create

- `internal/ui/panes/playlists_golden_test.go`
- `internal/ui/panes/albums_golden_test.go`
- `internal/ui/panes/likedsongs_golden_test.go`
- `internal/ui/panes/library_flow_test.go` — integration: drill-down + Esc back
- `internal/ui/panes/testdata/TestPlaylistsPane_View_*.golden` (4 files)
- `internal/ui/panes/testdata/TestAlbumsPane_View_*.golden` (4 files)
- `internal/ui/panes/testdata/TestLikedSongsPane_View_*.golden` (3 files)

## Acceptance Criteria

- [ ] PlaylistsPane: 7 golden snapshots (list, empty, track sub-view, track sub-view filter, spotify-owned, narrow, filter active)
- [ ] AlbumsPane: 6 golden snapshots (list, empty, track sub-view, track sub-view filter, narrow, filter active)
- [ ] LikedSongsPane: 4 golden snapshots (tracks, empty, narrow, filter active)
- [ ] Integration: Enter drill-down shows tracks, Esc returns to list
- [ ] Integration: 'x' in track sub-view produces PlaylistRemoveRequestMsg
- [ ] Integration: Enter on Spotify-owned playlist produces PlaylistAccessDeniedMsg
- [ ] All golden tests pass with committed golden files (no `-update` required)
- [ ] `make ci` passes

## Tasks

- [ ] Create PlaylistsPane golden tests (7 snapshots)
      - test: `TestPlaylistsPane_View_ListView`, `TestPlaylistsPane_View_EmptyState`, `TestPlaylistsPane_View_TrackSubView`, `TestPlaylistsPane_View_TrackSubView_FilterActive`, `TestPlaylistsPane_View_SpotifyOwnedLocked`, `TestPlaylistsPane_View_Narrow`, `TestPlaylistsPane_View_FilterActive`
- [ ] Create AlbumsPane golden tests (6 snapshots)
      - test: `TestAlbumsPane_View_AlbumList`, `TestAlbumsPane_View_EmptyState`, `TestAlbumsPane_View_TrackSubView`, `TestAlbumsPane_View_TrackSubView_FilterActive`, `TestAlbumsPane_View_Narrow`, `TestAlbumsPane_View_FilterActive`
- [ ] Create LikedSongsPane golden tests (4 snapshots)
      - test: `TestLikedSongsPane_View_Tracks`, `TestLikedSongsPane_View_EmptyState`, `TestLikedSongsPane_View_Narrow`, `TestLikedSongsPane_View_FilterActive`
- [ ] Create drill-down integration test
      - test: `TestPlaylistsDrillDown_EnterThenEsc`, `TestAlbumsDrillDown_EnterThenEsc`
- [ ] Create playlist delete + access denied integration tests
      - test: `TestPlaylistsPane_TrackView_DeleteTrack`, `TestPlaylistsPane_EnterOnSpotifyOwned_EmitsAccessDenied`
- [ ] Generate golden files and verify tests pass
