---
title: "Stats Dashboard"
status: done
---

## Description
Surfaces the user's listening data -- top tracks, top artists, and recently played history -- in navigable views with time-range filtering, making music habits feel like inspectable developer data. Originally built as a single full-screen `StatsView` toggled via key `2`, later split into three independent panes (`RecentlyPlayedPane`, `TopTracksPane`, `TopArtistsPane`) each implementing the `layout.Pane` interface with dense table format, in-pane filtering, and time range selection where applicable.

## Acceptance Criteria
- [ ] `RecentlyPlayedPane`, `TopTracksPane`, `TopArtistsPane` all satisfy `layout.Pane`
- [ ] All 3 panes use bubble-table with correct column widths from DESIGN.md
- [ ] All 3 panes support in-pane filtering with `f` key
- [ ] TopTracks and TopArtists support time range cycling with `t` key
- [ ] RecentlyPlayed shows relative time ("2m ago", "1h ago")
- [ ] Enter on a top track or artist plays it immediately
- [ ] Cached data avoids re-fetch when switching back to same time range
- [ ] All API calls and view Update() handlers tested
