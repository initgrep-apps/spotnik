# Feature 08 — Stats Dashboard

> **Depends on:** Feature 02 (Auth). No dependency on playback features.
> This is the differentiating feature — it makes Spotnik more than a Spotify client.

> **Layout note:** This feature uses the Stats view, which temporarily replaces the three-pane
> layout. Pressing `1` returns to the main Library | Player | Queue view. This does not
> violate the three-pane freeze — see `docs/features/00-overview.md` for the view-switching concept.

## Implementation Context

### Store fields this feature uses
```go
// Store fields this feature uses
TopTracks      map[string][]api.Track   // keyed by time range ("short_term", "medium_term", "long_term")
TopArtists     map[string][]api.Artist  // keyed by time range
// Note: StatsTimeRange (the currently selected range) is pane-local state in StatsView,
// not in the Store. The Store only holds cached data keyed by range.
```

### Time range cycling
Three ranges map to human labels: `short_term` → "4 weeks", `medium_term` → "6 months",
`long_term` → "all time". User cycles with `f` key. Display hint: `[4wk]` / `[6mo]` / `[all]`.
On range change: check store cache first — if data exists, render immediately with no fetch.
If cache misses, fetch from API and show spinner until data arrives.

### Message types for this feature
```go
type statsLoadedMsg struct {
    topTracks  []api.Track
    topArtists []api.Artist
    timeRange  string
}
type statsTimeRangeChangedMsg struct{ timeRange string }
```

### Design tokens used in this feature
`theme.SectionHeader()` · `theme.PlayingIndicator()` · `theme.TextPrimary()` ·
`theme.TextSecondary()` · `theme.TextMuted()` · `theme.SelectedBg()` · `theme.ActiveBorder()`

---

---

## Goal

Surface the user's listening data in a clean, navigable dashboard. Show top tracks, top artists,
and recently played history with time-range filtering. This view makes developers feel like
their music habits are data — which they are.

---

## Feature Acceptance Criteria

- [ ] `2` opens Stats view with data loaded within 3 seconds
- [ ] Time range switching via `f` shows correct data without flicker
- [ ] `Enter` on a top track or artist plays it immediately
- [ ] Recently played shows correct relative timestamps
- [ ] `1` returns to library view with three-pane layout intact
- [ ] Cached data avoids re-fetch when switching back to same time range
- [ ] All API calls and view Update() handlers tested

---

## User Stories

- **As a user**, I press `2` to switch to the Stats view.
- **As a user**, I see my top tracks for the last 4 weeks by default.
- **As a user**, I press `Tab` to switch between top tracks, top artists, and recently played.
- **As a user**, I press `f` to cycle the time range (4wk → 6mo → all-time) for the focused section.
- **As a user**, I press `Enter` on any track or artist to play it immediately.
- **As a user**, I press `1` to return to the main library view.

---

## Stats View Layout (from DESIGN.md)

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

**Layout notes:**
- Top half: split 50/50 between Top Tracks (left) and Top Artists (right)
- Bottom quarter: Recently Played (full width)
- Time range toggles affect the top section currently focused
- Active time range bracket highlighted with `ActiveBorder()` token background

### Initial Focus
When opening Stats, Top Tracks section is focused by default. Tab cycles:
Top Tracks → Top Artists → Recently Played → Top Tracks.

### Empty States
- No top tracks/artists: show "No listening data for this period" centered in the section
- No recently played: show "No recent listening history" centered

---

## API Calls

| Data | Endpoint | Params |
|---|---|---|
| Top Tracks (4wk) | `GET /me/top/tracks` | `time_range=short_term&limit=25` |
| Top Tracks (6mo) | `GET /me/top/tracks` | `time_range=medium_term&limit=25` |
| Top Tracks (all) | `GET /me/top/tracks` | `time_range=long_term&limit=25` |
| Top Artists (4wk) | `GET /me/top/artists` | `time_range=short_term&limit=25` |
| Top Artists (6mo) | `GET /me/top/artists` | `time_range=medium_term&limit=25` |
| Top Artists (all) | `GET /me/top/artists` | `time_range=long_term&limit=25` |
| Recently Played | `GET /me/player/recently-played` | `limit=50` |

**Loading strategy:**
- Fetch `short_term` data for both tracks and artists on first load (default view)
- Lazy-load other time ranges when user switches to them
- Cache all fetched data in store — don't re-fetch until view is re-opened

---

## Time Range Display

| API Value | Display Label | What It Means |
|---|---|---|
| `short_term` | `4wk` | Last ~4 weeks |
| `medium_term` | `6mo` | Last ~6 months |
| `long_term` | `all` | All-time history |

---

## Recently Played — Time Formatting

Show relative time for recent items, absolute for older:

| Elapsed | Display |
|---|---|
| < 1 min | `just now` |
| 1–59 min | `{n} min ago` |
| 1–23 hr | `{n} hr ago` |
| 1–6 days | `{n} days ago` |
| >= 7 days | `Jan 15` (short date) |

---

## Keymap (Stats View)

| Key | Action |
|---|---|
| `Tab` | Cycle section focus: Top Tracks → Top Artists → Recently Played |
| `j` / `↓` | Move selection down |
| `k` / `↑` | Move selection up |
| `f` | Cycle time range: 4wk → 6mo → all (filter key) |
| `Enter` | Play selected track or artist |
| `1` | Switch to Library view |
| `PgUp/Dn` | Scroll page |

---

## Play Behavior from Stats

| Selected Type | Play Action |
|---|---|
| Track | `PUT /me/player/play` with `uris: [track_uri]` |
| Artist | `PUT /me/player/play` with `context_uri: artist_uri` |
| Recently Played track | `PUT /me/player/play` with `uris: [track_uri]` |

---

## Files to Create

| File | Purpose |
|---|---|
| `internal/api/user.go` | Top tracks, top artists, recently played API calls |
| `internal/api/user_test.go` | Tests with fixture JSON |
| `internal/ui/panes/stats.go` | StatsView model |
| `internal/ui/panes/stats_test.go` | Update tests |

---

## Task Breakdown

### Task 7.1 — User/stats API calls

**Description:**
Implement the Spotify API client methods for fetching the user's top tracks, top artists,
and recently played history. Define the `Artist` model struct.

**Files:** `internal/api/user.go`, `internal/api/user_test.go`

**Implementation steps:**
- [ ] `GetTopTracks(ctx, timeRange string, limit int) ([]Track, error)`
- [ ] `GetTopArtists(ctx, timeRange string, limit int) ([]Artist, error)`
- [ ] `GetRecentlyPlayed(ctx, limit int) ([]PlayHistory, error)` (may reuse from Feature 03)
- [ ] `Artist` struct: id, name, genres, popularity, external_urls
- [ ] Test each with fixture JSON

**Acceptance criteria:**
- All three API methods parse Spotify JSON responses correctly
- Empty result sets return empty slices, not nil
- Errors are wrapped with context (`fmt.Errorf`)
- Artist struct unmarshals all required fields

**Tests:**

*Unit tests:*
- `TestGetTopTracks_Success` — returns parsed tracks for time range
- `TestGetTopTracks_EmptyResults` — returns empty slice
- `TestGetTopArtists_Success` — returns parsed artists
- `TestGetTopArtists_EmptyResults` — returns empty slice
- `TestGetRecentlyPlayed_Success` — returns play history items with timestamps
- `TestArtist_Unmarshal` — parses Artist JSON (id, name, genres, popularity)

---

### Task 7.2 — StatsView model

**Description:**
Build the StatsView Bubble Tea model with three sections (Top Tracks, Top Artists,
Recently Played), section focus cycling via Tab, cursor navigation via j/k, and
play-on-Enter for tracks and artists.

**Files:** `internal/ui/panes/stats.go`, `internal/ui/panes/stats_test.go`

**Implementation steps:**
- [ ] Implement `tea.Model`: `Init()`, `Update()`, `View()`
- [ ] `Init()` fetches top tracks (short_term) + top artists (short_term) + recently played
- [ ] Three sections: TopTracks, TopArtists, RecentlyPlayed
- [ ] Active section tracked in model (pane-local state)
- [ ] Top Tracks focused by default on open
- [ ] Tab cycles: Top Tracks → Top Artists → Recently Played → Top Tracks
- [ ] j/k moves cursor within focused section
- [ ] Enter on track or artist returns play command
- [ ] Render empty state messages when sections have no data

**Acceptance criteria:**
- Init returns a batch command fetching short_term tracks, short_term artists, and recently played
- Tab cycles through all three sections in order
- j/k moves cursor within the focused section without crossing section boundaries
- Enter on a track returns a play command with the track URI
- Enter on an artist returns a play command with the artist context URI
- Empty sections show the appropriate "No listening data" message

**Tests:**

*Unit tests:*
- `TestStatsView_Init_FetchesShortTerm` — returns batch command for short_term tracks + artists + recently played
- `TestStatsView_View_TopTracks` — renders numbered track list with artist
- `TestStatsView_View_TopArtists` — renders numbered artist list
- `TestStatsView_View_RecentlyPlayed` — renders track · artist · album with relative time
- `TestStatsView_View_EmptySection` — shows "No listening data" message
- `TestStatsView_Update_Tab` — cycles section focus
- `TestStatsView_Update_JK` — moves cursor within section
- `TestStatsView_Update_Enter_PlaysTrack` — returns play command

---

### Task 7.3 — Time range switching

**Description:**
Implement time range cycling with the `f` key. Check the store cache before fetching —
if data for the requested range already exists, render it immediately. Otherwise fire a
fetch command and show a spinner. Highlight the active range in the time range display.

**Files:** `internal/ui/panes/stats.go`, `internal/ui/panes/stats_test.go`

**Implementation steps:**
- [ ] `f` key cycles time range for active section: short_term → medium_term → long_term → short_term
- [ ] On switch: check store cache first, fetch if missing
- [ ] Time range toggle renders with active range highlighted using `ActiveBorder()` token
- [ ] Show spinner while fetching uncached range data

**Acceptance criteria:**
- `f` cycles through all three ranges in order and wraps around
- Cached data renders immediately with no fetch command returned
- Uncached data triggers a fetch command and shows a loading spinner
- Active range bracket is visually distinct using `ActiveBorder()` token

**Tests:**

*Unit tests:*
- `TestStatsView_Update_F_CyclesRange` — short_term → medium_term → long_term → short_term
- `TestStatsView_TimeRange_CacheHit` — cached data renders immediately, no fetch
- `TestStatsView_TimeRange_CacheMiss` — uncached range triggers fetch command
- `TestStatsView_View_ActiveRangeHighlighted` — active range bracket shown with highlight

---

### Task 7.4 — Recently played rendering

**Description:**
Implement the `FormatRelativeTime` function for human-readable timestamps in the recently
played section. Render each item as track · artist · album with right-aligned relative time.

**Files:** `internal/ui/panes/stats.go`, `internal/ui/panes/stats_test.go`

**Implementation steps:**
- [ ] Relative time formatting function `FormatRelativeTime(t time.Time) string`
- [ ] Handle all ranges: just now, minutes, hours, days, short date for older
- [ ] Render: track · artist · album, right-aligned time

**Acceptance criteria:**
- All five time ranges from the spec table produce correct output
- Times older than 7 days display as short date (e.g. "Mar 12")
- Recently played items render in the format: track · artist · album with right-aligned time

**Tests:**

*Unit tests:*
- `TestFormatRelativeTime_JustNow` — < 1 min → "just now"
- `TestFormatRelativeTime_Minutes` — 5 min → "5 min ago"
- `TestFormatRelativeTime_Hours` — 3 hr → "3 hr ago"
- `TestFormatRelativeTime_Days` — 2 days → "2 days ago"
- `TestFormatRelativeTime_OlderThanWeek` — 10 days → "Mar 12" short date

---

### Task 7.5 — View switching

**Description:**
Wire the Stats view into the root app model. `2` opens the Stats view (lazy-initialized on
first open). `1` returns to the main three-pane library view. Cursor position and section
focus are preserved when switching away and back.

**Files:** `internal/app/app.go`, `internal/ui/panes/stats.go`

**Implementation steps:**
- [ ] Root model: `2` switches to StatsView, `1` switches back to main
- [ ] StatsView lazy-initializes on first open (not on app start)
- [ ] Preserve cursor position and section focus when switching views

**Acceptance criteria:**
- Pressing `2` switches the view to StatsView and triggers data loading
- Pressing `1` from Stats restores the three-pane layout intact
- Returning to Stats preserves cursor position and focused section
- StatsView is not initialized until the user first presses `2`

**Tests:**

*Integration tests:*
- `TestApp_2KeyOpensStats` — pressing 2 switches to StatsView
- `TestApp_1KeyReturnsToLibrary` — pressing 1 from stats restores three-pane layout
- `TestApp_StatsPreservesCursor` — returning to stats preserves cursor position

---

## Out of Scope

- Listening time statistics (not in Spotify API)
- Playlist recommendations (removed endpoint)
- Export/share stats
- Charts or bar graphs (terminal ASCII charts are out of scope for MVP)

---

*Last updated: 2026-03-22*
