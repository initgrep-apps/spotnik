---
title: "Stats Dashboard"
feature: 08-stats
status: done
---

## Background
This story built the initial Stats Dashboard as a full-screen alternate view. It implemented the Spotify API client methods for fetching user listening data, created the `StatsView` Bubble Tea model with three sections, added time range cycling with store-level caching, relative time formatting for recently played items, and wired the view into the root app model via the `2` key.

## Design

### Store fields
```go
TopTracks      map[string][]api.Track   // keyed by time range ("short_term", "medium_term", "long_term")
TopArtists     map[string][]api.Artist  // keyed by time range
// Note: StatsTimeRange (the currently selected range) is pane-local state in StatsView, not in the Store.
```

### Time range cycling
Three ranges map to human labels: `short_term` -> "4 weeks", `medium_term` -> "6 months", `long_term` -> "all time". User cycles with `f` key. Display hint: `[4wk]` / `[6mo]` / `[all]`. On range change: check store cache first -- if data exists, render immediately. If cache misses, fetch and show spinner.

### Message types
```go
type statsLoadedMsg struct {
    topTracks  []api.Track
    topArtists []api.Artist
    timeRange  string
}
type statsTimeRangeChangedMsg struct{ timeRange string }
```

### Design tokens
`theme.SectionHeader()` . `theme.PlayingIndicator()` . `theme.TextPrimary()` . `theme.TextSecondary()` . `theme.TextMuted()` . `theme.SelectedBg()` . `theme.ActiveBorder()`

### Stats View Layout
```
+---------------------------------------------------------------------------+
|  Spotnik  [STATS]                                     * MacBook Pro        |
+------------------------------+--------------------------------------------+
|  TOP TRACKS                  |  TOP ARTISTS                               |
|  Time range: [4wk] [6mo] [all]  |  Time range: [4wk] [6mo] [all]         |
|                              |                                            |
|  1  Blinding Lights  Weeknd  |  1  The Weeknd                             |
|  2  Levitating       Dua Lipa|  2  Dua Lipa                               |
|  ... (25 total)              |  ... (25 total)                            |
+------------------------------+--------------------------------------------+
|  RECENTLY PLAYED                                                           |
|  Blinding Lights  .  The Weeknd  .  After Hours              3 min ago     |
|  Levitating       .  Dua Lipa    .  Future Nostalgia         18 min ago    |
+----------------------------------------------------------------------------+
|  Tab next section   j/k move   Enter play   f cycle time range             |
+----------------------------------------------------------------------------+
```

### Time Range Display

| API Value | Display Label |
|---|---|
| `short_term` | `4wk` |
| `medium_term` | `6mo` |
| `long_term` | `all` |

### Recently Played -- Time Formatting

| Elapsed | Display |
|---|---|
| < 1 min | `just now` |
| 1-59 min | `{n} min ago` |
| 1-23 hr | `{n} hr ago` |
| 1-6 days | `{n} days ago` |
| >= 7 days | `Jan 15` (short date) |

### Keymap (Stats View)

| Key | Action |
|---|---|
| `Tab` | Cycle section focus: Top Tracks -> Top Artists -> Recently Played |
| `j` / `down` | Move selection down |
| `k` / `up` | Move selection up |
| `f` | Cycle time range: 4wk -> 6mo -> all |
| `Enter` | Play selected track or artist |
| `1` | Switch to Library view |
| `PgUp/Dn` | Scroll page |

### API Calls

| Data | Endpoint | Params |
|---|---|---|
| Top Tracks | `GET /me/top/tracks` | `time_range={range}&limit=25` |
| Top Artists | `GET /me/top/artists` | `time_range={range}&limit=25` |
| Recently Played | `GET /me/player/recently-played` | `limit=50` |

### Files Created

| File | Purpose |
|---|---|
| `internal/api/user.go` | Top tracks, top artists, recently played API calls |
| `internal/api/user_test.go` | Tests with fixture JSON |
| `internal/ui/panes/stats.go` | StatsView model |
| `internal/ui/panes/stats_test.go` | Update tests |

### Out of Scope
- Listening time statistics (not in Spotify API)
- Playlist recommendations (removed endpoint)
- Export/share stats
- Charts or bar graphs

## Acceptance Criteria
- [ ] `2` opens Stats view with data loaded within 3 seconds
- [ ] Time range switching via `f` shows correct data without flicker
- [ ] `Enter` on a top track or artist plays it immediately
- [ ] Recently played shows correct relative timestamps
- [ ] `1` returns to library view with three-pane layout intact
- [ ] Cached data avoids re-fetch when switching back to same time range
- [ ] All API calls and view Update() handlers tested

## Tasks
- [ ] User/stats API calls -- Implement GetTopTracks, GetTopArtists, GetRecentlyPlayed
      - test: `TestGetTopTracks_Success`, `TestGetTopTracks_EmptyResults`, `TestGetTopArtists_Success`, `TestGetTopArtists_EmptyResults`, `TestGetRecentlyPlayed_Success`, `TestArtist_Unmarshal`
- [ ] StatsView model -- Build three-section view with Tab cycling and j/k navigation
      - test: `TestStatsView_Init_FetchesShortTerm`, `TestStatsView_View_TopTracks`, `TestStatsView_View_TopArtists`, `TestStatsView_View_RecentlyPlayed`, `TestStatsView_View_EmptySection`, `TestStatsView_Update_Tab`, `TestStatsView_Update_JK`, `TestStatsView_Update_Enter_PlaysTrack`
- [ ] Time range switching -- Implement `f` key cycling with store cache checking
      - test: `TestStatsView_Update_F_CyclesRange`, `TestStatsView_TimeRange_CacheHit`, `TestStatsView_TimeRange_CacheMiss`, `TestStatsView_View_ActiveRangeHighlighted`
- [ ] Recently played rendering -- Implement FormatRelativeTime and time-stamped rendering
      - test: `TestFormatRelativeTime_JustNow`, `TestFormatRelativeTime_Minutes`, `TestFormatRelativeTime_Hours`, `TestFormatRelativeTime_Days`, `TestFormatRelativeTime_OlderThanWeek`
- [ ] View switching -- Wire StatsView into root model with `2`/`1` key switching
      - test: `TestApp_2KeyOpensStats`, `TestApp_1KeyReturnsToLibrary`, `TestApp_StatsPreservesCursor`
