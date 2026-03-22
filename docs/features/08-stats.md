# Feature 08 — Stats Dashboard

> **Depends on:** Feature 02 (Auth). No dependency on playback features.
> This is the differentiating feature — it makes Spotnik more than a Spotify client.

## Implementation Context

### Store fields this feature uses
```go
TopTracks      []api.Track   // from GET /me/top/tracks
TopArtists     []api.Artist  // from GET /me/top/artists
StatsTimeRange string        // "short_term" | "medium_term" | "long_term"
```

### Time range cycling
Three ranges map to human labels: `short_term` → "4 weeks", `medium_term` → "6 months",
`long_term` → "all time". User toggles with `[4wk]` / `[6mo]` / `[all]` key hints.
On range change: re-fetch both endpoints, show spinner until data arrives.

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
`theme.TextSecondary()` · `theme.TextMuted()` · `theme.SelectedBg()`

---

---

## Goal

Surface the user's listening data in a clean, navigable dashboard. Show top tracks, top artists,
and recently played history with time-range filtering. This view makes developers feel like
their music habits are data — which they are.

---

## User Stories

- **As a user**, I press `2` to switch to the Stats view.
- **As a user**, I see my top tracks for the last 4 weeks by default.
- **As a user**, I press `Tab` to switch between top tracks, top artists, and recently played.
- **As a user**, I press `1`, `2`, or `3` within the stats view to change time range (4wk / 6mo / all-time).
- **As a user**, I press `Enter` on any track or artist to play it immediately.
- **As a user**, I press `1` (outside stats view) to return to the main library view.

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
│  Tab next section   j/k move   Enter play   [4wk][6mo][all] time range       │
╰──────────────────────────────────────────────────────────────────────────────╯
```

**Layout notes:**
- Top half: split 50/50 between Top Tracks (left) and Top Artists (right)
- Bottom quarter: Recently Played (full width)
- Time range toggles affect the top section currently focused
- Active time range bracket highlighted: `[4wk]` → Blue background

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
| `f` | Change time range: 4wk → 6mo → all (filter key) |
| `Enter` | Play selected track or artist |
| `1` | Switch to Library view |
| `PgUp/Dn` | Scroll page |

> Note: `1`, `2`, `3` as time range shortcuts conflict with view switching.
> Use `f` (filter) to cycle time range instead. This avoids the conflict.

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
- [ ] `GetTopTracks(ctx, timeRange string, limit int) ([]Track, error)`
- [ ] `GetTopArtists(ctx, timeRange string, limit int) ([]Artist, error)`
- [ ] `GetRecentlyPlayed(ctx, limit int) ([]PlayHistory, error)` (may reuse from Feature 03)
- [ ] `Artist` struct: id, name, genres, popularity, external_urls
- [ ] Test each with fixture JSON

### Task 7.2 — StatsView model
- [ ] Implement `tea.Model`: `Init()`, `Update()`, `View()`
- [ ] `Init()` fetches top tracks (short_term) + top artists (short_term) + recently played
- [ ] Three sections: TopTracks, TopArtists, RecentlyPlayed
- [ ] Active section tracked in model
- [ ] Test: init fires correct commands, section switching

### Task 7.3 — Time range switching
- [ ] `f` key cycles time range for active section
- [ ] On switch: check store cache first, fetch if missing
- [ ] Time range toggle renders with active range highlighted
- [ ] Test: cache hit skips fetch, cache miss fires fetch

### Task 7.4 — Recently played rendering
- [ ] Relative time formatting function `FormatRelativeTime(t time.Time) string`
- [ ] Test: all time ranges (just now, minutes, hours, days, older)
- [ ] Render: track · artist · album, right-aligned time

### Task 7.5 — View switching
- [ ] Root model: `2` switches to StatsView, `1` switches back to main
- [ ] StatsView lazy-initializes on first open (not on app start)
- [ ] Test: view switching preserves state (cursor position) when returning

---

## Acceptance Criteria

- [ ] `2` opens stats view with data loaded within 3 seconds
- [ ] Time range switching shows correct data with no flicker
- [ ] `Enter` on a top track plays it immediately
- [ ] Recently played shows correct relative timestamps
- [ ] `1` returns to library view with layout intact
- [ ] All API calls and view update handlers tested

---

## Out of Scope

- Listening time statistics (not in Spotify API)
- Playlist recommendations (removed endpoint)
- Export/share stats
- Charts or bar graphs (terminal ASCII charts are out of scope for MVP)

---

*Last updated: 2026-02-21*
