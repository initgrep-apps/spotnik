---
title: "Stats Split into Independent Panes"
feature: 08-stats
status: done
---

## Background
The original `StatsView` (`internal/ui/panes/stats.go`, ~20.2KB) was a full-screen alternate view toggled via key `2` that rendered top tracks, top artists, and recently played in a single large view. As the architecture moved to a grid-based pane layout managed by `LayoutManager`, this story split the monolithic view into three independent panes -- `RecentlyPlayedPane`, `TopTracksPane`, and `TopArtistsPane` -- each implementing the `layout.Pane` interface. These are panes 6, 7, 8 on Page A, each with their own toggle key, dense table format, in-pane filtering, and time range selection where applicable.

Design reference: `docs/DESIGN.md` section 2 (Pane Definitions), section 9 (Dense Table column widths per pane), section 23 (Migration -- StatsView split). Depends on: Feature 41 (Pane interface), Feature 43 (Table + Filter components).

## Design

### Design Diagram

```
Current: StatsView (full-screen view, key '2')
  +-- Top Tracks section (with time range tabs)
  +-- Top Artists section (with time range tabs)
  +-- Recently Played section

New: 3 independent panes in grid

+-- 6Recently Played -------------------- >f filter -+
|  #   Track                Artist        Played      |
|  1   Martbaan             Samar Mehdi   2m ago      |
|  2   Starboy              The Weeknd    15m ago     |
|  3   Heat Waves           Glass Animals 1h ago      |
|  4   Levitating           Dua Lipa      3h ago      |
|  v more below                                       |
+-----------------------------------------------------+

+-- 7Top Tracks ------ >f filter -- >4wk -- >6mo -- >all -+
|  #   Track                Artist        Popularity       |
|  1   Blinding Lights      The Weeknd    85               |
|  2   Martbaan             Samar Mehdi   72               |
|  3   Save Your Tears      The Weeknd    80               |
|  v more below                                            |
+----------------------------------------------------------+

+-- 8Top Artists ----- >f filter -- >4wk -- >6mo -- >all -+
|  #   Artist                         Genre                |
|  1   The Weeknd                     pop                  |
|  2   Drake                          hip-hop              |
|  3   Dua Lipa                       dance pop            |
|  v more below                                            |
+----------------------------------------------------------+

Column Widths (DESIGN.md section 9):
  RecentlyPlayed: # 5% | Track 45% | Artist 35% | Played 15%
  TopTracks:      # 5% | Track 45% | Artist 35% | Pop 15%
  TopArtists:     # 5% | Name 70% | Genre 25%
```

### Notes
- The `StatsLoadedMsg` carries both top tracks AND top artists for a given time range. Both `TopTracksPane` and `TopArtistsPane` handle this message.
- The old `StatsView` remains until Feature 49 rewires the app.
- RecentlyPlayed data is loaded via the existing `FetchRecentlyPlayedRequestMsg` / `RecentlyPlayedLoadedMsg` flow.
- The `t` key for time range cycling only works when the pane is focused.

## Acceptance Criteria
- [ ] `RecentlyPlayedPane`, `TopTracksPane`, `TopArtistsPane` all satisfy `layout.Pane`
- [ ] All 3 panes use bubble-table with correct column widths from DESIGN.md section 9
- [ ] All 3 panes support in-pane filtering with `f` key
- [ ] TopTracks and TopArtists support time range cycling with `t` key
- [ ] Time range shown in border actions label
- [ ] RecentlyPlayed shows relative time ("2m ago", "1h ago")
- [ ] Per-column colors match design (TextMuted, TextPrimary, TextSecondary)
- [ ] Each pane reads from Store, emits request messages
- [ ] `FormatRelativeTime` extracted to shared utility
- [ ] Old `StatsView` is NOT deleted yet
- [ ] `make ci` passes

## Tasks
- [ ] Create RecentlyPlayedPane -- Independent pane with relative time column
      - test: Interface satisfaction; correct columns; "Played" column shows relative time; Enter emits PlayTrackMsg; filter by track/artist; empty state
- [ ] Create TopTracksPane -- Independent pane with time range cycling
      - test: Interface satisfaction; correct columns; `t` cycles time range; change emits FetchStatsMsg; Actions shows current range; Enter emits PlayTrackMsg; filter works; popularity column
- [ ] Create TopArtistsPane -- Independent pane with genre column
      - test: Interface satisfaction; Name and Genre columns; `t` cycles time range; filter by name/genre; genre shows first from list
- [ ] Extract FormatRelativeTime utility to shared location
      - test: 30s ago -> "30s ago"; 5m ago -> "5m ago"; 2h ago -> "2h ago"; 3d ago -> "3d ago"
- [ ] Comprehensive tests -- Full integration and edge case coverage
      - test: Load/scroll/filter/play lifecycle per pane; resize handling; time range sync; empty data states; artist with no genres shows "--"; long name truncation
