---
title: "Fix: Like/Unlike UX — Revert Heart Prefix, Add Border Action, Fix 403 Error"
feature: 05-library
status: open
---

## Background

Stories 267 and 268 shipped the like/unlike feature. Three issues found in testing:

1. **♥ prefix visual clutter** — Heart glyph prepended to track names in all 8 panes. Doesn't look good. Must revert.

2. **Missing `l` border action hint** — Pane borders show `f filter` but not `l like`. Should show conditionally when a track row is selected, following the same pattern as `f`. Currently users have no visual cue that `l` exists.

3. **Wrong error toast on 403** — Pressing `l` shows toast "Failed to load library forbidden". Root cause: `routing.go:566` maps the error to `uikit.OpLibrary` which has title "Failed to load library" — wrong for a save operation. Need new `OpLikeTracks` operation with correct title "Like track failed" + forbidden body "Premium required to like tracks." The 403 itself comes from Spotify rejecting `PUT /v1/me/tracks` — likely a Premium enforcement issue where `store.IsPremium()` passed but Spotify server-side rejected.

## Design

### Bug 1: Revert ♥ prefix

Remove the `heart := uikit.GlyphFor(uikit.GlyphLiked, ...)` + concat logic from all 8 panes. Track names render as-is, no heart prefix. The `GlyphLiked` constant stays in `glyph.go` (may be used later for border action or other indicators).

Files to modify:
- `internal/ui/panes/nowplaying.go` — remove heart prepend from `buildInfoLines()` track name
- `internal/ui/panes/likedsongs_pane.go` — remove heart from `refreshRows()` track column
- `internal/ui/panes/queue.go` — remove heart from `refreshRows()` track column
- `internal/ui/panes/toptracks_pane.go` — remove heart from `refreshRows()` track column
- `internal/ui/panes/recentlyplayed_pane.go` — remove heart from `refreshRows()` track column
- `internal/ui/panes/playlists_pane.go` — remove heart from track sub-view row rendering
- `internal/ui/panes/albums_pane.go` — remove heart from track sub-view row rendering
- `internal/ui/panes/search_delegate.go` — remove heart from search result row rendering

After revert: regenerate golden files (`go test ./... -update`).

### Bug 2: Conditional `l` border action hint

Add `layout.Action{Key: "l", Label: "like"}` to the `Actions()` method of each track-displaying pane, conditional on a track row being selected.

**Pattern** — follow NowPlaying's conditional action pattern (`nowplaying.go:182-195`):

```go
// Example for a TableBasedPane subclass in track view:
func (p *XPane) Actions() []layout.Action {
    if p.inTrackView() {
        return []layout.Action{
            {Key: "Esc", Label: "back"},
            {Key: "l", Label: "like"},
        }
    }
    return []layout.Action{p.BaseFilterAction()}
}
```

**Per-pane logic:**

| Pane | Condition for `l` hint |
|------|------------------------|
| NowPlaying | `ps.Item != nil` (track loaded) |
| LikedSongs | `len(filteredTracks()) > 0` (has tracks) |
| Queue | `len(filteredTracks()) > 0` and selected item is a track |
| TopTracks | `len(filteredTracks()) > 0` (has tracks) |
| RecentlyPlayed | `len(filteredTracks()) > 0` (has tracks) |
| Playlists (track view) | `inTrackView()` and `len(loadedTracks) > 0` |
| Albums (track view) | `inTrackView()` and `len(loadedTracks) > 0` |
| Search | selected item `IsTrack == true` |

The `l` action should appear alongside existing actions (`f filter`, `Esc back`). When no track is selected, `l` must NOT appear.

### Bug 3: Fix 403 error toast

**New Operation type** in `internal/uikit/error_mapper.go`:

```go
OpLikeTracks Operation = "like-tracks"
```

**Add to `opTitle` map:**
```go
OpLikeTracks: "Like track failed",
```

**Add to `opForbiddenBody` map:**
```go
OpLikeTracks: "Premium required to like tracks.",
```

**Update routing** in `internal/app/routing.go` — change `ToggleLikeResultMsg` error handler from `uikit.OpLibrary` to `uikit.OpLikeTracks`:

```go
toast := a.errorMapper.Map(uikit.OpLikeTracks, m.Err)
```

This fixes the wrong title ("Failed to load library" → "Like track failed") and provides a clear forbidden body ("Premium required to like tracks." instead of raw "forbidden" message).

**Premium gate investigation:** The 403 means Spotify rejected the `PUT /v1/me/tracks` call. The premium gate at `routing.go:489` checks `store.IsPremium()` which reads `userProfile.Product == "premium"`. If the gate passed but Spotify still returned 403, either:
- `IsPremium()` returned true with stale profile data (profile loaded from a previous session)
- The token's account is actually Free but profile wasn't refreshed after token refresh
- Verify: ensure `IsPremium()` is fresh — profile should be re-fetched on token refresh

No code change needed for the gate itself — the new `OpLikeTracks` forbidden body "Premium required to like tracks." will correctly inform the user when Spotify rejects the call.

## Files

### Modify

- `internal/ui/panes/nowplaying.go` — revert heart, add `l` to `Actions()` conditional on track loaded
- `internal/ui/panes/likedsongs_pane.go` — revert heart, add `l` to `Actions()` conditional on tracks present
- `internal/ui/panes/queue.go` — revert heart, add `l` to `Actions()` conditional on track selected
- `internal/ui/panes/toptracks_pane.go` — revert heart, add `l` to `Actions()` conditional on tracks present
- `internal/ui/panes/recentlyplayed_pane.go` — revert heart, add `l` to `Actions()` conditional on tracks present
- `internal/ui/panes/playlists_pane.go` — revert heart in track view, add `l` to `Actions()` in track view
- `internal/ui/panes/albums_pane.go` — revert heart in track view, add `l` to `Actions()` in track view
- `internal/ui/panes/search.go` — add `l` to `Actions()` conditional on track result selected
- `internal/ui/panes/search_delegate.go` — revert heart from search result rendering
- `internal/uikit/error_mapper.go` — add `OpLikeTracks` operation, title, forbidden body
- `internal/app/routing.go` — change `OpLibrary` to `OpLikeTracks` in ToggleLikeResultMsg error handler
- Golden files — regenerate after heart revert

## Acceptance Criteria

- [ ] No `♥` prefix on any track name in any pane
- [ ] Golden files regenerated, all golden tests pass
- [ ] Pane border shows `l like` action hint when a track row is selected in each of the 8 track-displaying panes
- [ ] Pane border does NOT show `l like` when no track is selected or pane is empty
- [ ] `l like` appears alongside existing actions (`f filter`, `Esc back`) without crowding
- [ ] Pressing `l` on a 403 error shows toast with title "Like track failed" (not "Failed to load library")
- [ ] 403 body shows "Premium required to like tracks." (not raw "forbidden")
- [ ] `make ci` passes

## Tasks

- [ ] Revert ♥ prefix from all 8 panes — remove `heart := GlyphFor(...)` + concat logic from track name rendering in nowplaying, likedsongs, queue, toptracks, recentlyplayed, playlists (track view), albums (track view), search_delegate
      - test: `TestNowPlayingPane_View_NoHeartPrefix`, `TestLikedSongsPane_View_NoHeartPrefix`, `TestQueuePane_View_NoHeartPrefix` (verify track name has no `♥`)
- [ ] Add `OpLikeTracks` operation to error_mapper.go — new `Operation` constant, `opTitle` entry "Like track failed", `opForbiddenBody` entry "Premium required to like tracks."
      - test: `TestErrorMapper_OpLikeTracks_Forbidden`, `TestErrorMapper_OpLikeTracks_Generic`
- [ ] Update routing.go — change `uikit.OpLibrary` to `uikit.OpLikeTracks` in `ToggleLikeResultMsg` error handler
      - test: `TestRouting_ToggleLikeResult_ErrorMapsToOpLikeTracks` (assert toast title is "Like track failed")
- [ ] Add conditional `l like` border action to NowPlayingPane `Actions()` — show when `ps.Item != nil`
      - test: `TestNowPlayingPane_Actions_ShowsLikeWhenTrackLoaded`, `TestNowPlayingPane_Actions_NoLikeWhenEmpty`
- [ ] Add conditional `l like` border action to LikedSongsPane `Actions()` — show when `len(filteredTracks()) > 0`
      - test: `TestLikedSongsPane_Actions_ShowsLikeWhenTracks`, `TestLikedSongsPane_Actions_NoLikeWhenEmpty`
- [ ] Add conditional `l like` border action to QueuePane `Actions()` — show when selected item is a track
      - test: `TestQueuePane_Actions_ShowsLikeWhenTrackSelected`, `TestQueuePane_Actions_NoLikeWhenEpisodeSelected`
- [ ] Add conditional `l like` border action to TopTracksPane `Actions()` — show when tracks present
      - test: `TestTopTracksPane_Actions_ShowsLikeWhenTracks`, `TestTopTracksPane_Actions_NoLikeWhenEmpty`
- [ ] Add conditional `l like` border action to RecentlyPlayedPane `Actions()` — show when tracks present
      - test: `TestRecentlyPlayedPane_Actions_ShowsLikeWhenTracks`, `TestRecentlyPlayedPane_Actions_NoLikeWhenEmpty`
- [ ] Add conditional `l like` border action to PlaylistsPane `Actions()` — show in track view when tracks loaded
      - test: `TestPlaylistsPane_Actions_TrackView_ShowsLike`, `TestPlaylistsPane_Actions_ListView_NoLike`
- [ ] Add conditional `l like` border action to AlbumsPane `Actions()` — show in track view when tracks loaded
      - test: `TestAlbumsPane_Actions_TrackView_ShowsLike`, `TestAlbumsPane_Actions_ListView_NoLike`
- [ ] Add conditional `l like` border action to SearchOverlay `Actions()` — show when selected result is a track
      - test: `TestSearchOverlay_Actions_ShowsLikeWhenTrackSelected`, `TestSearchOverlay_Actions_NoLikeWhenArtistSelected`
- [ ] Regenerate golden files — run `go test ./... -update`, review diffs, commit
      - test: all golden tests pass