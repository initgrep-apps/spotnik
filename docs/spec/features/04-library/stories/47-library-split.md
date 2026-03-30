---
title: "Library Split"
feature: 04-library
status: done
---

## Background
The monolithic `LibraryPane` (~18.5KB) handled all four data domains (playlists, albums, liked songs, recently played) in a single pane with collapsible tree sections. The `PlaylistManager` (~24.2KB) was a separate full-screen view with dual-pane layout for playlist management. This story splits the library into three independent panes -- `PlaylistsPane`, `AlbumsPane`, and `LikedSongsPane` -- each implementing `layout.Pane` with dense table format and filtering. PlaylistManager functionality (create, rename, delete, reorder) is merged into PlaylistsPane. RecentlyPlayed moves to Feature 48 (Stats Split).

Design reference: `docs/DESIGN.md` section 2 (Pane Definitions), section 9 (Dense Table column widths), section 23 (Migration). Depends on Feature 41 (Pane interface) and Feature 43 (Table + Filter components).

## Design

### Design Diagram

```
Current Architecture:
  LibraryPane (18.5KB) -- monolithic tree with 4 collapsible sections
  PlaylistManager (24.2KB) -- separate full-screen view (key '3')

New Architecture (3 independent panes):

+-- 3Playlists ------ >f filter -- >n new -- >r rename -- >x delete -+
|  #   Name                              Tracks                       |
|  1   LoFi                              42                           |
|  2   Best of Coke Studio               28                           |
|  3   Soul                              15                           |
|  4   Workout                           67                           |
|  v more below                                                       |
+---------------------------------------------------------------------+

  Enter -> opens track sub-view for selected playlist:
+-- 3Playlists -- LoFi (42 tracks) ------ >Esc back -- >Shift+arrows reorder -+
|  #   Track                    Artist              Duration                    |
|  1   Snowman                  Sia                 3:21                        |
|  2   Coffee                  Beabadoobee         3:44                        |
|  v more below                                                                |
+------------------------------------------------------------------------------+

+-- 4Albums ------------ >f filter -+
|  #   Name                 Artist     Year |
|  1   After Hours          Weeknd     2020 |
|  2   OK Computer          Radiohead  1997 |
|  3   In Rainbows          Radiohead  2007 |
|  v more below                             |
+-------------------------------------------+

+-- 5Liked Songs ------ >f filter -- >i like -+
|  #   Track              Artist       Duration |
|  1   Blinding Lights    The Weeknd   3:22     |
|  2   Save Your Tears    The Weeknd   3:35     |
|  3   Levitating         Dua Lipa     3:23     |
|  v more below                                 |
+-----------------------------------------------+

Column Widths (DESIGN.md section 9):
  Playlists:   # 5% | Name 70% | Tracks 25%
  Albums:      # 5% | Name 50% | Artist 30% | Year 15%
  LikedSongs:  # 5% | Track 45% | Artist 35% | Duration 15%
```

### Notes

- **RecentlyPlayed** is NOT part of this story. It moves to Feature 48 (Stats Split).
- Old `LibraryPane` and `PlaylistManager` files remain until Feature 49 (App Migration) rewires the app.
- PlaylistsPane's track sub-view is internal state -- it doesn't change the page or layout.
- Playlist mutations (create, rename, delete, reorder) emit request messages. The app's `Update()` dispatches the API commands.

## Acceptance Criteria
- [ ] `PlaylistsPane`, `AlbumsPane`, `LikedSongsPane` all satisfy `layout.Pane`
- [ ] PlaylistsPane merges PlaylistManager features (create, rename, delete, reorder, track sub-view)
- [ ] All 3 panes use bubble-table with correct column widths from DESIGN.md section 9
- [ ] All 3 panes support in-pane filtering with `f` key
- [ ] Per-column colors match DESIGN.md section 9 (TextMuted, TextPrimary, TextSecondary, TextMuted)
- [ ] Each pane reads from Store, emits request messages (no direct API calls)
- [ ] PlaylistsPane track sub-view: Enter opens, Esc returns to list
- [ ] LikedSongsPane: `i` key toggles like/unlike
- [ ] Old `LibraryPane` and `PlaylistManager` files are NOT deleted yet
- [ ] `make ci` passes

## Tasks
- [ ] Create PlaylistsPane -- Unified pane merging LibraryPane playlist list and PlaylistManager management
      - test: Interface satisfaction; playlist list renders; Enter opens track sub-view; Esc returns; n/r/x emit requests; Shift+arrows emit reorder; filter works; dynamic title in track sub-view
- [ ] Create AlbumsPane -- Dedicated album browsing pane with dense table format
      - test: Interface satisfaction; album list renders; Enter emits PlayContextMsg; filter by name/artist; year column; empty state
- [ ] Create LikedSongsPane -- Dedicated liked songs pane with like/unlike toggle
      - test: Interface satisfaction; track list renders; Enter emits PlayTrackMsg; `i` emits LikeTrackRequestMsg; filter by track/artist; duration as M:SS
- [ ] Data loading integration -- Route existing message types to new panes
      - test: PlaylistsPane handles LibraryLoadedMsg; AlbumsPane handles AlbumsLoadedMsg; LikedSongsPane handles LikedTracksLoadedMsg; PlaylistsPane handles mutation messages
- [ ] Comprehensive tests -- Full integration and edge case coverage
      - test: Full lifecycle tests per pane; resize handling; independent filtering; large dataset scrolling; empty data states
