---
title: "Stats Dashboard"
description: "Surfaces the user's listening data — top tracks, top artists, and recently played history — in navigable views with time-range filtering, making music habits feel like inspectable developer data."
status: done
stories: [08, 48]
---

# Stats Dashboard

## Background

Spotnik's Stats Dashboard is the feature that differentiates it from a plain Spotify client. It surfaces the user's listening data — top tracks, top artists, and recently played history — in a clean, navigable interface with time-range filtering. For developers who live in the terminal, this turns music habits into inspectable data.

The feature was originally built as a single full-screen `StatsView` toggled via key `2`, rendering all three sections (top tracks, top artists, recently played) in one large view with Tab-based section focus cycling. The view included time range toggling (4wk/6mo/all), relative time formatting for recently played items, and play-on-Enter for tracks and artists.

As the architecture evolved toward a grid-based pane layout managed by `LayoutManager`, the monolithic `StatsView` was split into three independent panes — `RecentlyPlayedPane`, `TopTracksPane`, and `TopArtistsPane` — each implementing the `layout.Pane` interface with dense table format, in-pane filtering, and time range selection where applicable. This split aligns with the design system's pane definitions (DESIGN.md sections 2, 9, 23) and enables each stats section to participate in the grid layout independently.

---

## Story: Stats Dashboard (spec 08)

### Background
This story built the initial Stats Dashboard as a full-screen alternate view. It implemented the Spotify API client methods for fetching user listening data, created the `StatsView` Bubble Tea model with three sections, added time range cycling with store-level caching, relative time formatting for recently played items, and wired the view into the root app model via the `2` key.

### Acceptance Criteria
- [ ] `2` opens Stats view with data loaded within 3 seconds
- [ ] Time range switching via `f` shows correct data without flicker
- [ ] `Enter` on a top track or artist plays it immediately
- [ ] Recently played shows correct relative timestamps
- [ ] `1` returns to library view with three-pane layout intact
- [ ] Cached data avoids re-fetch when switching back to same time range
- [ ] All API calls and view Update() handlers tested

### Tasks

1. **Task 7.1 — User/stats API calls** — Implement the Spotify API client methods for fetching the user's top tracks, top artists, and recently played history. Define the `Artist` model struct.
   - Files: `internal/api/user.go`, `internal/api/user_test.go`
   - Implementation steps:
     - [ ] `GetTopTracks(ctx, timeRange string, limit int) ([]Track, error)`
     - [ ] `GetTopArtists(ctx, timeRange string, limit int) ([]Artist, error)`
     - [ ] `GetRecentlyPlayed(ctx, limit int) ([]PlayHistory, error)` (may reuse from Feature 03)
     - [ ] `Artist` struct: id, name, genres, popularity, external_urls
     - [ ] Test each with fixture JSON
   - Acceptance criteria:
     - All three API methods parse Spotify JSON responses correctly
     - Empty result sets return empty slices, not nil
     - Errors are wrapped with context (`fmt.Errorf`)
     - Artist struct unmarshals all required fields
   - Tests:
     - `TestGetTopTracks_Success` — returns parsed tracks for time range
     - `TestGetTopTracks_EmptyResults` — returns empty slice
     - `TestGetTopArtists_Success` — returns parsed artists
     - `TestGetTopArtists_EmptyResults` — returns empty slice
     - `TestGetRecentlyPlayed_Success` — returns play history items with timestamps
     - `TestArtist_Unmarshal` — parses Artist JSON (id, name, genres, popularity)

2. **Task 7.2 — StatsView model** — Build the StatsView Bubble Tea model with three sections (Top Tracks, Top Artists, Recently Played), section focus cycling via Tab, cursor navigation via j/k, and play-on-Enter for tracks and artists.
   - Files: `internal/ui/panes/stats.go`, `internal/ui/panes/stats_test.go`
   - Implementation steps:
     - [ ] Implement `tea.Model`: `Init()`, `Update()`, `View()`
     - [ ] `Init()` fetches top tracks (short_term) + top artists (short_term) + recently played
     - [ ] Three sections: TopTracks, TopArtists, RecentlyPlayed
     - [ ] Active section tracked in model (pane-local state)
     - [ ] Top Tracks focused by default on open
     - [ ] Tab cycles: Top Tracks → Top Artists → Recently Played → Top Tracks
     - [ ] j/k moves cursor within focused section
     - [ ] Enter on track or artist returns play command
     - [ ] Render empty state messages when sections have no data
   - Acceptance criteria:
     - Init returns a batch command fetching short_term tracks, short_term artists, and recently played
     - Tab cycles through all three sections in order
     - j/k moves cursor within the focused section without crossing section boundaries
     - Enter on a track returns a play command with the track URI
     - Enter on an artist returns a play command with the artist context URI
     - Empty sections show the appropriate "No listening data" message
   - Tests:
     - `TestStatsView_Init_FetchesShortTerm` — returns batch command for short_term tracks + artists + recently played
     - `TestStatsView_View_TopTracks` — renders numbered track list with artist
     - `TestStatsView_View_TopArtists` — renders numbered artist list
     - `TestStatsView_View_RecentlyPlayed` — renders track · artist · album with relative time
     - `TestStatsView_View_EmptySection` — shows "No listening data" message
     - `TestStatsView_Update_Tab` — cycles section focus
     - `TestStatsView_Update_JK` — moves cursor within section
     - `TestStatsView_Update_Enter_PlaysTrack` — returns play command

3. **Task 7.3 — Time range switching** — Implement time range cycling with the `f` key. Check the store cache before fetching — if data for the requested range already exists, render it immediately. Otherwise fire a fetch command and show a spinner. Highlight the active range in the time range display.
   - Files: `internal/ui/panes/stats.go`, `internal/ui/panes/stats_test.go`
   - Implementation steps:
     - [ ] `f` key cycles time range for active section: short_term → medium_term → long_term → short_term
     - [ ] On switch: check store cache first, fetch if missing
     - [ ] Time range toggle renders with active range highlighted using `ActiveBorder()` token
     - [ ] Show spinner while fetching uncached range data
   - Acceptance criteria:
     - `f` cycles through all three ranges in order and wraps around
     - Cached data renders immediately with no fetch command returned
     - Uncached data triggers a fetch command and shows a loading spinner
     - Active range bracket is visually distinct using `ActiveBorder()` token
   - Tests:
     - `TestStatsView_Update_F_CyclesRange` — short_term → medium_term → long_term → short_term
     - `TestStatsView_TimeRange_CacheHit` — cached data renders immediately, no fetch
     - `TestStatsView_TimeRange_CacheMiss` — uncached range triggers fetch command
     - `TestStatsView_View_ActiveRangeHighlighted` — active range bracket shown with highlight

4. **Task 7.4 — Recently played rendering** — Implement the `FormatRelativeTime` function for human-readable timestamps in the recently played section. Render each item as track · artist · album with right-aligned relative time.
   - Files: `internal/ui/panes/stats.go`, `internal/ui/panes/stats_test.go`
   - Implementation steps:
     - [ ] Relative time formatting function `FormatRelativeTime(t time.Time) string`
     - [ ] Handle all ranges: just now, minutes, hours, days, short date for older
     - [ ] Render: track · artist · album, right-aligned time
   - Acceptance criteria:
     - All five time ranges from the spec table produce correct output
     - Times older than 7 days display as short date (e.g. "Mar 12")
     - Recently played items render in the format: track · artist · album with right-aligned time
   - Tests:
     - `TestFormatRelativeTime_JustNow` — < 1 min → "just now"
     - `TestFormatRelativeTime_Minutes` — 5 min → "5 min ago"
     - `TestFormatRelativeTime_Hours` — 3 hr → "3 hr ago"
     - `TestFormatRelativeTime_Days` — 2 days → "2 days ago"
     - `TestFormatRelativeTime_OlderThanWeek` — 10 days → "Mar 12" short date

5. **Task 7.5 — View switching** — Wire the Stats view into the root app model. `2` opens the Stats view (lazy-initialized on first open). `1` returns to the main three-pane library view. Cursor position and section focus are preserved when switching away and back.
   - Files: `internal/app/app.go`, `internal/ui/panes/stats.go`
   - Implementation steps:
     - [ ] Root model: `2` switches to StatsView, `1` switches back to main
     - [ ] StatsView lazy-initializes on first open (not on app start)
     - [ ] Preserve cursor position and section focus when switching views
   - Acceptance criteria:
     - Pressing `2` switches the view to StatsView and triggers data loading
     - Pressing `1` from Stats restores the three-pane layout intact
     - Returning to Stats preserves cursor position and focused section
     - StatsView is not initialized until the user first presses `2`
   - Tests:
     - `TestApp_2KeyOpensStats` — pressing 2 switches to StatsView
     - `TestApp_1KeyReturnsToLibrary` — pressing 1 from stats restores three-pane layout
     - `TestApp_StatsPreservesCursor` — returning to stats preserves cursor position

### Implementation Context

#### Store fields
```go
TopTracks      map[string][]api.Track   // keyed by time range ("short_term", "medium_term", "long_term")
TopArtists     map[string][]api.Artist  // keyed by time range
// Note: StatsTimeRange (the currently selected range) is pane-local state in StatsView,
// not in the Store. The Store only holds cached data keyed by range.
```

#### Time range cycling
Three ranges map to human labels: `short_term` → "4 weeks", `medium_term` → "6 months", `long_term` → "all time". User cycles with `f` key. Display hint: `[4wk]` / `[6mo]` / `[all]`. On range change: check store cache first — if data exists, render immediately with no fetch. If cache misses, fetch from API and show spinner until data arrives.

#### Message types
```go
type statsLoadedMsg struct {
    topTracks  []api.Track
    topArtists []api.Artist
    timeRange  string
}
type statsTimeRangeChangedMsg struct{ timeRange string }
```

#### Design tokens
`theme.SectionHeader()` · `theme.PlayingIndicator()` · `theme.TextPrimary()` · `theme.TextSecondary()` · `theme.TextMuted()` · `theme.SelectedBg()` · `theme.ActiveBorder()`

#### Stats View Layout
```
╭──────────────────────────────────────────────────────────────────────────────╮
│  Spotnik  [STATS]                                     ◉ MacBook Pro Speakers   │
├─────────────────────────────────┬────────────────────────────────────────────┤
│  TOP TRACKS                     │  TOP ARTISTS                               │
│  ─────────────────────────────  │  ──────────────────────────────────────── │
│  Time range: [4wk] [6mo] [all]  │  Time range: [4wk] [6mo] [all]            │
│                                 │                                            │
│  1  Blinding Lights  The Weeknd │  1  The Weeknd                             │
│  2  Levitating       Dua Lipa   │  2  Dua Lipa                               │
│  3  Save Your Tears  The Weeknd │  3  Post Malone                            │
│  4  Peaches          Bieber     │  4  Justin Bieber                          │
│  5  Mood             24kGoldn   │  5  Taylor Swift                           │
│  ... (25 total)                 │  ... (25 total)                            │
├─────────────────────────────────┴────────────────────────────────────────────┤
│  RECENTLY PLAYED                                                              │
│  ───────────────────────────────────────────────────────────────────────────  │
│  Blinding Lights  ·  The Weeknd  ·  After Hours              3 min ago        │
│  Levitating       ·  Dua Lipa    ·  Future Nostalgia         18 min ago       │
│  Starboy          ·  The Weeknd  ·  Starboy                  34 min ago       │
├────────────────────────────────────────────────────────────────────────────── │
│  Tab next section   j/k move   Enter play   f cycle time range               │
╰──────────────────────────────────────────────────────────────────────────────╯
```

Layout notes:
- Top half: split 50/50 between Top Tracks (left) and Top Artists (right)
- Bottom quarter: Recently Played (full width)
- Time range toggles affect the top section currently focused
- Active time range bracket highlighted with `ActiveBorder()` token background
- Initial focus: Top Tracks section by default. Tab cycles: Top Tracks → Top Artists → Recently Played → Top Tracks.
- Empty states: No top tracks/artists shows "No listening data for this period" centered; no recently played shows "No recent listening history" centered.

#### API Calls

| Data | Endpoint | Params |
|---|---|---|
| Top Tracks (4wk) | `GET /me/top/tracks` | `time_range=short_term&limit=25` |
| Top Tracks (6mo) | `GET /me/top/tracks` | `time_range=medium_term&limit=25` |
| Top Tracks (all) | `GET /me/top/tracks` | `time_range=long_term&limit=25` |
| Top Artists (4wk) | `GET /me/top/artists` | `time_range=short_term&limit=25` |
| Top Artists (6mo) | `GET /me/top/artists` | `time_range=medium_term&limit=25` |
| Top Artists (all) | `GET /me/top/artists` | `time_range=long_term&limit=25` |
| Recently Played | `GET /me/player/recently-played` | `limit=50` |

Loading strategy:
- Fetch `short_term` data for both tracks and artists on first load (default view)
- Lazy-load other time ranges when user switches to them
- Cache all fetched data in store — don't re-fetch until view is re-opened

#### Time Range Display

| API Value | Display Label | What It Means |
|---|---|---|
| `short_term` | `4wk` | Last ~4 weeks |
| `medium_term` | `6mo` | Last ~6 months |
| `long_term` | `all` | All-time history |

#### Recently Played — Time Formatting

| Elapsed | Display |
|---|---|
| < 1 min | `just now` |
| 1–59 min | `{n} min ago` |
| 1–23 hr | `{n} hr ago` |
| 1–6 days | `{n} days ago` |
| >= 7 days | `Jan 15` (short date) |

#### Keymap (Stats View)

| Key | Action |
|---|---|
| `Tab` | Cycle section focus: Top Tracks → Top Artists → Recently Played |
| `j` / `↓` | Move selection down |
| `k` / `↑` | Move selection up |
| `f` | Cycle time range: 4wk → 6mo → all (filter key) |
| `Enter` | Play selected track or artist |
| `1` | Switch to Library view |
| `PgUp/Dn` | Scroll page |

#### Play Behavior from Stats

| Selected Type | Play Action |
|---|---|
| Track | `PUT /me/player/play` with `uris: [track_uri]` |
| Artist | `PUT /me/player/play` with `context_uri: artist_uri` |
| Recently Played track | `PUT /me/player/play` with `uris: [track_uri]` |

#### Files Created

| File | Purpose |
|---|---|
| `internal/api/user.go` | Top tracks, top artists, recently played API calls |
| `internal/api/user_test.go` | Tests with fixture JSON |
| `internal/ui/panes/stats.go` | StatsView model |
| `internal/ui/panes/stats_test.go` | Update tests |

#### Out of Scope
- Listening time statistics (not in Spotify API)
- Playlist recommendations (removed endpoint)
- Export/share stats
- Charts or bar graphs (terminal ASCII charts are out of scope for MVP)

---

## Story: Stats Split into Independent Panes (spec 48)

### Background
The original `StatsView` (`internal/ui/panes/stats.go`, ~20.2KB) was a full-screen alternate view toggled via key `2` that rendered top tracks, top artists, and recently played in a single large view. As the architecture moved to a grid-based pane layout managed by `LayoutManager`, this story split the monolithic view into three independent panes — `RecentlyPlayedPane`, `TopTracksPane`, and `TopArtistsPane` — each implementing the `layout.Pane` interface. These are panes 6, 7, 8 on Page A, each with their own toggle key, dense table format, in-pane filtering, and time range selection where applicable. The old `StatsView` is preserved until Feature 49/53 rewires the app.

Design reference: `docs/DESIGN.md` section 2 (Pane Definitions), section 9 (Dense Table column widths per pane), section 23 (Migration — StatsView split). Depends on: Feature 41 (Pane interface), Feature 43 (Table + Filter components).

### Acceptance Criteria
- [ ] `RecentlyPlayedPane`, `TopTracksPane`, `TopArtistsPane` all satisfy `layout.Pane`
- [ ] All 3 panes use bubble-table with correct column widths from DESIGN.md section 9
- [ ] All 3 panes support in-pane filtering with `f` key
- [ ] TopTracks and TopArtists support time range cycling with `t` key
- [ ] Time range shown in border actions label
- [ ] RecentlyPlayed shows relative time ("2m ago", "1h ago")
- [ ] Per-column colors match design (TextMuted, TextPrimary, TextSecondary)
- [ ] Each pane reads from Store, emits request messages
- [ ] `FormatRelativeTime` extracted to shared utility
- [ ] Old `StatsView` is NOT deleted yet (done in Feature 49/53)
- [ ] `make ci` passes

### Tasks

1. **Task 1 — Create RecentlyPlayedPane** — Recently played is buried inside StatsView. Create it as an independent pane implementing `layout.Pane`.
   - Files: Create `internal/ui/panes/recentlyplayed_pane.go`
   - Struct:
     ```go
     type RecentlyPlayedPane struct {
         store   *state.Store
         theme   theme.Theme
         table   components.Table
         filter  *components.Filter
         focused bool
         width   int
         height  int
     }
     ```
   - Pane interface:
     ```go
     func (r *RecentlyPlayedPane) ID() layout.PaneID       { return layout.PaneRecentlyPlayed }
     func (r *RecentlyPlayedPane) Title() string            { return "Recently Played" }
     func (r *RecentlyPlayedPane) ToggleKey() int           { return 6 }
     func (r *RecentlyPlayedPane) Actions() []layout.Action {
         if r.filter.IsActive() {
             return []layout.Action{{Key: "Esc", Label: "close"}}
         }
         return []layout.Action{{Key: "f", Label: "filter"}}
     }
     ```
   - Key handling: `Enter` → emit `PlayTrackMsg` with track URI; `f` → toggle filter; `j/k` → scroll
   - Data source: `store.RecentlyPlayed()` — loaded via existing `RecentlyPlayedLoadedMsg`
   - Columns: `# 5% | Track 45% | Artist 35% | Played 15%`
   - "Played" column: Uses `FormatRelativeTime()` to show "2m ago", "1h ago", "3d ago" etc.
   - Filter matches: track name, artist name
   - Tests:
     - Unit: Interface satisfaction: `var _ layout.Pane = &RecentlyPlayedPane{}`
     - Unit: Recently played renders with correct columns
     - Unit: "Played" column shows relative time (2m ago, 1h ago)
     - Unit: Enter key → emits PlayTrackMsg
     - Unit: Filter filters by track and artist name
     - Unit: Empty data → clean empty state
   - Commit: `feat(ui): create RecentlyPlayedPane with relative time column`

2. **Task 2 — Create TopTracksPane** — Top tracks is a section inside StatsView with time range toggling. Create it as an independent pane.
   - Files: Create `internal/ui/panes/toptracks_pane.go`
   - Struct:
     ```go
     type TopTracksPane struct {
         store     *state.Store
         theme     theme.Theme
         table     components.Table
         filter    *components.Filter
         focused   bool
         width     int
         height    int
         timeRange string // "short_term", "medium_term", "long_term"
     }
     ```
   - Pane interface:
     ```go
     func (t *TopTracksPane) ID() layout.PaneID       { return layout.PaneTopTracks }
     func (t *TopTracksPane) Title() string            { return "Top Tracks" }
     func (t *TopTracksPane) ToggleKey() int           { return 7 }
     ```
   - Actions (revised approach — `t` key cycles time ranges since keys 1-3 are pane toggles):
     ```go
     func (t *TopTracksPane) Actions() []layout.Action {
         if t.filter.IsActive() {
             return []layout.Action{{Key: "Esc", Label: "close"}}
         }
         rangeLabel := map[string]string{
             "short_term": "4wk", "medium_term": "6mo", "long_term": "all",
         }[t.timeRange]
         return []layout.Action{
             {Key: "f", Label: "filter"},
             {Key: "t", Label: rangeLabel},
         }
     }
     ```
   - `t` key → cycle timeRange: short→medium→long→short. On change: emit `FetchStatsMsg{TimeRange: t.timeRange}` to fetch new data
   - Data source: `store.TopTracks(timeRange)` — from `StatsLoadedMsg`
   - Columns: `# 5% | Track 45% | Artist 35% | Popularity 15%`
   - Filter matches: track name, artist name
   - Tests:
     - Unit: Interface satisfaction: `var _ layout.Pane = &TopTracksPane{}`
     - Unit: Top tracks renders with correct columns
     - Unit: `t` key cycles time range: short→medium→long→short
     - Unit: Time range change emits FetchStatsMsg
     - Unit: Actions label shows current range ("4wk", "6mo", "all")
     - Unit: Enter key → emits PlayTrackMsg
     - Unit: Filter filters by track and artist name
     - Unit: Popularity column shows numeric value
   - Commit: `feat(ui): create TopTracksPane with time range cycling`

3. **Task 3 — Create TopArtistsPane** — Top artists is a section inside StatsView. Create it as an independent pane.
   - Files: Create `internal/ui/panes/topartists_pane.go`
   - Struct:
     ```go
     type TopArtistsPane struct {
         store     *state.Store
         theme     theme.Theme
         table     components.Table
         filter    *components.Filter
         focused   bool
         width     int
         height    int
         timeRange string
     }
     ```
   - Pane interface:
     ```go
     func (t *TopArtistsPane) ID() layout.PaneID       { return layout.PaneTopArtists }
     func (t *TopArtistsPane) Title() string            { return "Top Artists" }
     func (t *TopArtistsPane) ToggleKey() int           { return 8 }
     func (t *TopArtistsPane) Actions() []layout.Action { /* same pattern as TopTracks */ }
     ```
   - Key handling: Same `t` key cycle for time range. `Enter` has no action (artists aren't directly playable).
   - Data source: `store.TopArtists(timeRange)` — from `StatsLoadedMsg`
   - Columns: `# 5% | Name 70% | Genre 25%`
   - Filter matches: artist name, genre
   - Tests:
     - Unit: Interface satisfaction: `var _ layout.Pane = &TopArtistsPane{}`
     - Unit: Artist list renders with Name and Genre columns
     - Unit: `t` key cycles time range
     - Unit: Filter filters by artist name and genre
     - Unit: Genre column shows first genre from artist's genre list
   - Commit: `feat(ui): create TopArtistsPane with genre column`

4. **Task 4 — Extract FormatRelativeTime utility** — `FormatRelativeTime` is currently inside `stats.go`. The RecentlyPlayedPane needs it but shouldn't import from the old StatsView.
   - Files:
     - Create: `internal/ui/components/timeutil.go`
     - Modify: `internal/ui/panes/stats.go` (update import)
     - Modify: `internal/ui/panes/recentlyplayed_pane.go` (use extracted function)
   - Implementation: Extract `FormatRelativeTime(playedAt time.Time) string` to `internal/ui/components/timeutil.go`
   - Tests:
     - Unit: `FormatRelativeTime` with time 30 seconds ago → "30s ago"
     - Unit: `FormatRelativeTime` with time 5 minutes ago → "5m ago"
     - Unit: `FormatRelativeTime` with time 2 hours ago → "2h ago"
     - Unit: `FormatRelativeTime` with time 3 days ago → "3d ago"
   - Commit: `refactor(ui): extract FormatRelativeTime to shared utility`

5. **Task 5 — Comprehensive tests** — Full test coverage for all three new panes.
   - Files:
     - Create: `internal/ui/panes/recentlyplayed_pane_test.go`
     - Create: `internal/ui/panes/toptracks_pane_test.go`
     - Create: `internal/ui/panes/topartists_pane_test.go`
   - Tests:
     - Integration: RecentlyPlayedPane — load data → scroll → filter → play track
     - Integration: TopTracksPane — load data → cycle time range → verify data refreshes
     - Integration: TopArtistsPane — load data → filter by genre → cycle time range
     - Integration: All 3 panes resize correctly
     - Integration: Time range sync — both TopTracks and TopArtists cycle independently
     - Edge: Empty data per pane → clean empty state
     - Edge: Artist with no genres → genre column shows "—"
     - Edge: Very long artist/track names → truncated in columns
   - Commit: `test(ui): comprehensive stats split pane tests`

### Design Diagram

```
Current: StatsView (full-screen view, key '2')
  ├── Top Tracks section (with time range tabs)
  ├── Top Artists section (with time range tabs)
  └── Recently Played section

New: 3 independent panes in grid

╭─ ⁶Recently Played ──────────────────── ᐅf filter ╮
│  #   Track                Artist        Played     │
│  1   Martbaan             Samar Mehdi   2m ago     │
│  2   Starboy              The Weeknd    15m ago    │
│  3   Heat Waves           Glass Animals 1h ago     │
│  4   Levitating           Dua Lipa      3h ago     │
│  ▼ more below                                      │
╰────────────────────────────────────────────────────╯

╭─ ⁷Top Tracks ──────── ᐅf filter ─ ᐅ4wk ─ ᐅ6mo ─ ᐅall ╮
│  #   Track                Artist        Popularity      │
│  1   Blinding Lights      The Weeknd    85              │
│  2   Martbaan             Samar Mehdi   72              │
│  3   Save Your Tears      The Weeknd    80              │
│  ▼ more below                                           │
╰─────────────────────────────────────────────────────────╯

╭─ ⁸Top Artists ─────── ᐅf filter ─ ᐅ4wk ─ ᐅ6mo ─ ᐅall ╮
│  #   Artist                         Genre               │
│  1   The Weeknd                     pop                  │
│  2   Drake                          hip-hop              │
│  3   Dua Lipa                       dance pop            │
│  ▼ more below                                            │
╰──────────────────────────────────────────────────────────╯

Column Widths (DESIGN.md §9):
  RecentlyPlayed: # 5% | Track 45% | Artist 35% | Played 15%
  TopTracks:      # 5% | Track 45% | Artist 35% | Pop 15%
  TopArtists:     # 5% | Name 70% | Genre 25%
```

### Notes
- The `StatsLoadedMsg` carries both top tracks AND top artists for a given time range. Both `TopTracksPane` and `TopArtistsPane` handle this message. Each pane extracts its relevant data. If the time ranges diverge (one shows 4wk, other shows 6mo), each pane emits its own `FetchStatsMsg` with its own time range.
- The old `StatsView` remains until Feature 49 rewires the app. The `viewStats` mode and key `2` binding will be removed at that point — key `2` becomes the Queue toggle.
- RecentlyPlayed data is loaded via the existing `FetchRecentlyPlayedRequestMsg` / `RecentlyPlayedLoadedMsg` flow. No new API calls needed.
- The `t` key for time range cycling only works when the pane is focused. It does NOT conflict with any global key binding.
