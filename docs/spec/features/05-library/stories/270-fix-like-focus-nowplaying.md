---
title: "Fix: Remove l from NowPlaying, Show l Border Action Only on Focused Pane"
feature: 05-library
status: done
---

## Background

Story 269 attempted to fix like/unlike UX but two bugs remain:

1. **`l` triggers on NowPlayingPane** — NowPlaying is a playback control pane, not a track list pane. Pressing `l` while focused on NowPlaying triggers a like/unlike API call. This is wrong — `l` should only work on panes that display song LISTS (LikedSongs, Queue, TopTracks, RecentlyPlayed, Playlists track sub-view, Albums track sub-view, Search results).

2. **`l like` border hint shows on unfocused panes** — `Actions()` is called for ALL visible panes during `renderGrid()`, not just the focused one. The `l like` corner-notch hint appears on every pane that has tracks, regardless of focus. It should only appear on the FOCUSED pane that contains a track list.

The 403 "Like track failed" / "Premium required to like tracks." error is correctly handled by the error mapper (story 269 added `OpLikeTracks`). The 403 itself is Spotify rejecting the request — user needs a Premium account. No code change needed for this.

## Design

### Bug 1: Remove `l` from NowPlayingPane

**`internal/ui/panes/nowplaying.go`:**

Remove the `case "l":` block from the key handler (around line 794-809). NowPlayingPane should NOT handle `l` — it's a playback control pane showing the currently playing track, not a track list pane.

Remove `layout.Action{Key: "l", Label: "like"}` from `Actions()` (around line 195-197). The `l` hint should never appear on NowPlaying's border.

### Bug 2: Show `l like` border action only on focused pane

**All 7 track-list panes** — modify `Actions()` to check `IsFocused()`:

```go
func (p *XPane) Actions() []layout.Action {
    if !p.IsFocused() {
        // Return base actions without l — l is a focus-only action.
        if p.inTrackView() {
            return []layout.Action{{Key: "Esc", Label: "back"}}
        }
        return []layout.Action{p.BaseFilterAction()}
    }
    // Focused — show l like when tracks are present.
    if p.inTrackView() && len(p.loadedTracks) > 0 {
        return []layout.Action{
            {Key: "Esc", Label: "back"},
            {Key: "l", Label: "like"},
        }
    }
    if len(p.filteredTracks()) > 0 {
        return []layout.Action{
            p.BaseFilterAction(),
            {Key: "l", Label: "like"},
        }
    }
    return []layout.Action{p.BaseFilterAction()}
}
```

**Per-pane `Actions()` logic:**

| Pane | Focused + has tracks → show | Unfocused → show |
|------|------------------------------|------------------|
| LikedSongs | `f filter` + `l like` | `f filter` only |
| Queue | `f filter` + `l like` (if selected is track) | `f filter` only |
| TopTracks | `f filter` + `l like` | `f filter` only |
| RecentlyPlayed | `f filter` + `l like` | `f filter` only |
| Playlists (track view) | `Esc back` + `l like` | `Esc back` only |
| Albums (track view) | `Esc back` + `l like` | `Esc back` only |
| Search | `l like` (if selected is track) | no `l` |

**NowPlaying:** Remove `l` entirely — never shows `l like`.

**Key insight:** `Actions()` is called during `renderGrid()` for every visible pane. The `IsFocused()` check ensures `l like` only appears on the pane the user is currently interacting with. This matches the user's expectation: "it is a on focus action."

## Files

### Modify

- `internal/ui/panes/nowplaying.go` — remove `l` key handler case, remove `l` from `Actions()`
- `internal/ui/panes/likedsongs_pane.go` — add `IsFocused()` guard to `Actions()`
- `internal/ui/panes/queue.go` — add `IsFocused()` guard to `Actions()`
- `internal/ui/panes/toptracks_pane.go` — add `IsFocused()` guard to `Actions()`
- `internal/ui/panes/recentlyplayed_pane.go` — add `IsFocused()` guard to `Actions()`
- `internal/ui/panes/playlists_pane.go` — add `IsFocused()` guard to `Actions()` in track view
- `internal/ui/panes/albums_pane.go` — add `IsFocused()` guard to `Actions()` in track view
- `internal/ui/panes/search.go` — add `IsFocused()` guard to `Actions()`

## Acceptance Criteria

- [ ] Pressing `l` on NowPlayingPane does NOT trigger like/unlike (key is ignored)
- [ ] NowPlayingPane border never shows `l like` action hint
- [ ] `l like` border hint appears ONLY on the focused pane (not on unfocused panes)
- [ ] `l like` border hint appears only when focused pane has tracks (absent when empty)
- [ ] Unfocused panes show their base actions without `l like` (e.g., `f filter` only)
- [ ] `l` key still works correctly on all 7 track-list panes when focused
- [ ] `make ci` passes

## Tasks

- [ ] Remove `l` key handler from NowPlayingPane — delete the `case "l":` block in the key handler method
      - test: `TestNowPlayingPane_L_KeyIgnored` (assert no ToggleLikeRequestMsg emitted)
- [ ] Remove `l` from NowPlayingPane `Actions()` — delete the `{Key: "l", Label: "like"}` entry
      - test: `TestNowPlayingPane_Actions_NoLikeAction` (assert `l` not in actions list)
- [ ] Add `IsFocused()` guard to LikedSongsPane `Actions()` — show `l` only when focused + tracks present
      - test: `TestLikedSongsPane_Actions_ShowsLikeWhenFocused`, `TestLikedSongsPane_Actions_NoLikeWhenUnfocused`
- [ ] Add `IsFocused()` guard to QueuePane `Actions()` — show `l` only when focused + track selected
      - test: `TestQueuePane_Actions_ShowsLikeWhenFocused`, `TestQueuePane_Actions_NoLikeWhenUnfocused`
- [ ] Add `IsFocused()` guard to TopTracksPane `Actions()` — show `l` only when focused + tracks present
      - test: `TestTopTracksPane_Actions_ShowsLikeWhenFocused`, `TestTopTracksPane_Actions_NoLikeWhenUnfocused`
- [ ] Add `IsFocused()` guard to RecentlyPlayedPane `Actions()` — show `l` only when focused + tracks present
      - test: `TestRecentlyPlayedPane_Actions_ShowsLikeWhenFocused`, `TestRecentlyPlayedPane_Actions_NoLikeWhenUnfocused`
- [ ] Add `IsFocused()` guard to PlaylistsPane `Actions()` — show `l` only when focused + in track view + tracks loaded
      - test: `TestPlaylistsPane_Actions_TrackView_ShowsLikeWhenFocused`, `TestPlaylistsPane_Actions_TrackView_NoLikeWhenUnfocused`
- [ ] Add `IsFocused()` guard to AlbumsPane `Actions()` — show `l` only when focused + in track view + tracks loaded
      - test: `TestAlbumsPane_Actions_TrackView_ShowsLikeWhenFocused`, `TestAlbumsPane_Actions_TrackView_NoLikeWhenUnfocused`
- [ ] Add `IsFocused()` guard to SearchOverlay `Actions()` — show `l` only when focused + selected is track
      - test: `TestSearchOverlay_Actions_ShowsLikeWhenFocused`, `TestSearchOverlay_Actions_NoLikeWhenUnfocused`
