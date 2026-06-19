---
title: "Tune responsive column visibility — ensure at least 2 cols at dashboard widths"
feature: 20-pane-content-design
status: done
---

## Background

After story 253 (revert # column), all panes have the `#` column back at Priority 1. This naturally gives every pane at least 2 columns at dashboard preset (~30 cols): `#` + the primary identifier column. However, some panes still show only 2 columns at medium widths (~40-50 cols) where 3 would be appropriate. Flex factor distributions also need review — the primary column (track name, show name, etc.) should dominate width at all sizes.

**Root cause:** The Priority 2 threshold (≥40 cols) is reasonable for secondary info, but the flex factor balance could better favor primary columns. After `#` column restoration, we can verify 2-column minimum is met everywhere.

## Design

### Verify 2-column minimum after story 253

With `#` column at Priority 1 restored:

| Pane | P1 cols (visible at any width) | Visible at dashboard (~30 cols) |
|------|-------------------------------|------|
| Queue | #, type, title | 3 cols ✓ |
| LikedSongs | #, track | 2 cols ✓ |
| RecentlyPlayed | #, track | 2 cols ✓ |
| TopTracks | #, track | 2 cols ✓ |
| TopArtists | #, name | 2 cols ✓ |
| Playlists list | #, access, name | 3 cols ✓ |
| Albums list | #, name | 2 cols ✓ |
| SavedEpisodes | #, icon, episode | 3 cols ✓ |
| FollowedShows shows | #, media, show | 3 cols ✓ |
| FollowedShows eps | #, icon, title | 3 cols ✓ |

All panes show at least 2 columns at dashboard. ✓

### Flex factor tuning

Current flex factors give primary column ~45-55% of width. For better readability at narrow widths, boost primary column flex factor and reduce secondary:

| Pane | Old primary flex | New primary flex | Old secondary flex | New secondary flex |
|------|-----------------|------------------|--------------------|--------------------|
| LikedSongs | track:9 (45%) | track:12 (55%) | artist:7 (32%) | artist:6 (27%) |
| RecentlyPlayed | track:9 (45%) | track:12 (55%) | artist:7 (32%) | artist:6 (27%) |
| TopTracks | track:9 (45%) | track:12 (55%) | artist:7 (32%) | artist:6 (27%) |
| Albums tracks | name:10 (50%) | name:12 (57%) | artist:6 (29%) | artist:6 (29%) |

These adjustments shift ~5-10% more width to the primary column without making secondary columns illegible.

### Panes NOT tuned (already good)
- **Queue:** title:7/15 = 47% with 4 other cols — reasonable. Keep.
- **TopArtists:** name:11/20 = 55% — already dominant. Keep.
- **Playlists list:** name:13/20 = 65% — already dominant. Keep.
- **Playlists tracks:** track:10/20 = 50% — good. Keep.
- **Albums list:** name:10/20 = 50% — good. Keep.
- **SavedEpisodes:** episode:9/20 = 45% but icon:1 is tiny — width split between episode+show is 9:6 = 60/40. Keep.
- **FollowedShows shows:** show:10/21 = 48% — reasonable. Keep.
- **FollowedShows eps:** title:9/18 = 50% — good. Keep.

## Files

### Modify

- `internal/ui/panes/likedsongs_pane.go` — track FlexFactor 9→12, artist 7→6 (constructor + SetTheme)
- `internal/ui/panes/recentlyplayed_pane.go` — track FlexFactor 9→12, artist 7→6 (constructor + SetTheme)
- `internal/ui/panes/toptracks_pane.go` — track FlexFactor 9→12, artist 7→6 (constructor + SetTheme)
- `internal/ui/panes/albums_pane.go` — track sub-view: name FlexFactor 10→12 (constructor + SetTheme)

## Acceptance Criteria

- [ ] All panes show at least 2 columns at dashboard preset (~30 cols width)
- [ ] LikedSongs: track FlexFactor=12, artist FlexFactor=6
- [ ] RecentlyPlayed: track FlexFactor=12, artist FlexFactor=6
- [ ] TopTracks: track FlexFactor=12, artist FlexFactor=6
- [ ] Albums tracks: name FlexFactor=12
- [ ] All other panes keep existing flex factors unchanged
- [ ] All SetTheme methods match constructor flex factors
- [ ] `go build ./...` compiles
- [ ] `make ci` passes

## Tasks

- [ ] **Task 1: Tune LikedSongs flex factors**
  Change track FlexFactor 9→12, artist 7→6. Update SetTheme.
  - test: `go test ./internal/ui/panes/ -v -run "TestLikedSongs"`

- [ ] **Task 2: Tune RecentlyPlayed flex factors**
  Change track FlexFactor 9→12, artist 7→6. Update SetTheme.
  - test: `go test ./internal/ui/panes/ -v -run "TestRecentlyPlayed"`

- [ ] **Task 3: Tune TopTracks flex factors**
  Change track FlexFactor 9→12, artist 7→6. Update SetTheme.
  - test: `go test ./internal/ui/panes/ -v -run "TestTopTracks"`

- [ ] **Task 4: Tune Albums tracks flex factor**
  Change track sub-view name FlexFactor 10→12. Update SetTheme.
  - test: `go test ./internal/ui/panes/ -v -run "TestAlbums"`

- [ ] **Task 5: Run full test suite**
  - test: `make test` — all pass
