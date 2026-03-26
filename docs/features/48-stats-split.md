# Feature 48 — Stats Split

> **Feature:** Split `StatsView` into 3 independent panes: `RecentlyPlayedPane`,
> `TopTracksPane`, `TopArtistsPane`. Each implements `layout.Pane` with dense table
> format, filtering, and time range selection where applicable.

## Context

The current `StatsView` (`internal/ui/panes/stats.go`, ~20.2KB) is a full-screen
alternate view (toggled via key `2`) that renders top tracks, top artists, and
recently played in a single large view with time range toggle (4wk/6mo/all).

The new DESIGN.md (§2, §23) specifies splitting this into 3 independent grid panes:
1. **RecentlyPlayedPane** — recently played tracks with "played ago" relative time
2. **TopTracksPane** — top tracks with time range toggle as border actions
3. **TopArtistsPane** — top artists with genre column and time range toggle

These are panes 6, 7, 8 on Page A, each with their own toggle key.

**Design reference:** `docs/DESIGN.md` §2 (Pane Definitions — RecentlyPlayed/TopTracks/TopArtists),
§9 (Dense Table — column widths per pane), §23 (Migration — StatsView split)

**Depends on:** Feature 41 (Pane interface), Feature 43 (Table + Filter components)

---

## Design Diagram

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

---

## Task 1: Create RecentlyPlayedPane

**Problem:** Recently played is buried inside StatsView.

**Fix:**

Create `internal/ui/panes/recentlyplayed_pane.go`:

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

**Pane interface:**
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

**Key handling:**
- `Enter` → emit `PlayTrackMsg` with track URI
- `f` → toggle filter
- `j/k` → scroll

**Data source:** `store.RecentlyPlayed()` — loaded via existing `RecentlyPlayedLoadedMsg`.

**Columns:** `# 5% | Track 45% | Artist 35% | Played 15%`

**"Played" column:** Uses `FormatRelativeTime()` (already exists in stats.go) to show
"2m ago", "1h ago", "3d ago" etc. This utility should be extracted to a shared location
if not already accessible.

**Filter matches:** track name, artist name

**Files:**
- Create: `internal/ui/panes/recentlyplayed_pane.go`

**Tests:**
- Unit: Interface satisfaction: `var _ layout.Pane = &RecentlyPlayedPane{}`
- Unit: Recently played renders with correct columns
- Unit: "Played" column shows relative time (2m ago, 1h ago)
- Unit: Enter key → emits PlayTrackMsg
- Unit: Filter filters by track and artist name
- Unit: Empty data → clean empty state

**Commit:** `feat(ui): create RecentlyPlayedPane with relative time column`

---

## Task 2: Create TopTracksPane

**Problem:** Top tracks is a section inside StatsView with time range toggling.

**Fix:**

Create `internal/ui/panes/toptracks_pane.go`:

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

**Pane interface:**
```go
func (t *TopTracksPane) ID() layout.PaneID       { return layout.PaneTopTracks }
func (t *TopTracksPane) Title() string            { return "Top Tracks" }
func (t *TopTracksPane) ToggleKey() int           { return 7 }
func (t *TopTracksPane) Actions() []layout.Action {
    if t.filter.IsActive() {
        return []layout.Action{{Key: "Esc", Label: "close"}}
    }
    return []layout.Action{
        {Key: "f", Label: "filter"},
        {Key: "4wk", Label: ""},   // time range actions — active one is highlighted
        {Key: "6mo", Label: ""},
        {Key: "all", Label: ""},
    }
}
```

**Time range handling:**
- Border actions show `4wk`, `6mo`, `all` — the active range is visually highlighted
- Key `1` while focused → short_term (4 weeks)
- Key `2` while focused → medium_term (6 months)
- Key `3` while focused → long_term (all time)
- **Wait — keys 1-3 are pane toggles!** So use different keys for time range.
  Options: border action labels as clickable (future), or cycle with a dedicated key.
  **Recommendation:** Use a single key to cycle time ranges. Add a `t` key action:
  `{Key: "t", Label: "4wk"}` → cycles through ranges. Label shows current range.

**Revised approach:**
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

- `t` key → cycle timeRange: short→medium→long→short
- On change: emit `FetchStatsMsg{TimeRange: t.timeRange}` to fetch new data

**Data source:** `store.TopTracks(timeRange)` — from `StatsLoadedMsg`.

**Columns:** `# 5% | Track 45% | Artist 35% | Popularity 15%`

**Filter matches:** track name, artist name

**Files:**
- Create: `internal/ui/panes/toptracks_pane.go`

**Tests:**
- Unit: Interface satisfaction: `var _ layout.Pane = &TopTracksPane{}`
- Unit: Top tracks renders with correct columns
- Unit: `t` key cycles time range: short→medium→long→short
- Unit: Time range change emits FetchStatsMsg
- Unit: Actions label shows current range ("4wk", "6mo", "all")
- Unit: Enter key → emits PlayTrackMsg
- Unit: Filter filters by track and artist name
- Unit: Popularity column shows numeric value

**Commit:** `feat(ui): create TopTracksPane with time range cycling`

---

## Task 3: Create TopArtistsPane

**Problem:** Top artists is a section inside StatsView.

**Fix:**

Create `internal/ui/panes/topartists_pane.go`:

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

**Pane interface:**
```go
func (t *TopArtistsPane) ID() layout.PaneID       { return layout.PaneTopArtists }
func (t *TopArtistsPane) Title() string            { return "Top Artists" }
func (t *TopArtistsPane) ToggleKey() int           { return 8 }
func (t *TopArtistsPane) Actions() []layout.Action { /* same pattern as TopTracks */ }
```

**Key handling:** Same `t` key cycle for time range. `Enter` has no action (artists aren't directly playable).

**Data source:** `store.TopArtists(timeRange)` — from `StatsLoadedMsg`.

**Columns:** `# 5% | Name 70% | Genre 25%`

**Filter matches:** artist name, genre

**Files:**
- Create: `internal/ui/panes/topartists_pane.go`

**Tests:**
- Unit: Interface satisfaction: `var _ layout.Pane = &TopArtistsPane{}`
- Unit: Artist list renders with Name and Genre columns
- Unit: `t` key cycles time range
- Unit: Filter filters by artist name and genre
- Unit: Genre column shows first genre from artist's genre list

**Commit:** `feat(ui): create TopArtistsPane with genre column`

---

## Task 4: Extract FormatRelativeTime utility

**Problem:** `FormatRelativeTime` is currently inside `stats.go`. The RecentlyPlayedPane
needs it but shouldn't import from the old StatsView.

**Fix:**

1. Extract `FormatRelativeTime(playedAt time.Time) string` to `internal/ui/components/timeutil.go`
2. Update `stats.go` to import from the new location (temporary, until stats.go is deleted)
3. Use in `RecentlyPlayedPane`

**Files:**
- Create: `internal/ui/components/timeutil.go`
- Modify: `internal/ui/panes/stats.go` (update import)
- Modify: `internal/ui/panes/recentlyplayed_pane.go` (use extracted function)

**Tests:**
- Unit: `FormatRelativeTime` with time 30 seconds ago → "30s ago"
- Unit: `FormatRelativeTime` with time 5 minutes ago → "5m ago"
- Unit: `FormatRelativeTime` with time 2 hours ago → "2h ago"
- Unit: `FormatRelativeTime` with time 3 days ago → "3d ago"

**Commit:** `refactor(ui): extract FormatRelativeTime to shared utility`

---

## Task 5: Comprehensive tests

**Files:**
- Create: `internal/ui/panes/recentlyplayed_pane_test.go`
- Create: `internal/ui/panes/toptracks_pane_test.go`
- Create: `internal/ui/panes/topartists_pane_test.go`

**Tests:**
- Integration: RecentlyPlayedPane — load data → scroll → filter → play track
- Integration: TopTracksPane — load data → cycle time range → verify data refreshes
- Integration: TopArtistsPane — load data → filter by genre → cycle time range
- Integration: All 3 panes resize correctly
- Integration: Time range sync — both TopTracks and TopArtists cycle independently
- Edge: Empty data per pane → clean empty state
- Edge: Artist with no genres → genre column shows "—"
- Edge: Very long artist/track names → truncated in columns

**Commit:** `test(ui): comprehensive stats split pane tests`

---

## Acceptance Criteria

- [ ] `RecentlyPlayedPane`, `TopTracksPane`, `TopArtistsPane` all satisfy `layout.Pane`
- [ ] All 3 panes use bubble-table with correct column widths from DESIGN.md §9
- [ ] All 3 panes support in-pane filtering with `f` key
- [ ] TopTracks and TopArtists support time range cycling with `t` key
- [ ] Time range shown in border actions label
- [ ] RecentlyPlayed shows relative time ("2m ago", "1h ago")
- [ ] Per-column colors match design (TextMuted, TextPrimary, TextSecondary)
- [ ] Each pane reads from Store, emits request messages
- [ ] `FormatRelativeTime` extracted to shared utility
- [ ] Old `StatsView` is NOT deleted yet (done in Feature 49/53)
- [ ] `make ci` passes

---

## Notes

- The `StatsLoadedMsg` carries both top tracks AND top artists for a given time range.
  Both `TopTracksPane` and `TopArtistsPane` handle this message. Each pane extracts its
  relevant data. If the time ranges diverge (one shows 4wk, other shows 6mo), each pane
  emits its own `FetchStatsMsg` with its own time range.
- The old `StatsView` remains until Feature 49 rewires the app. The `viewStats` mode and
  key `2` binding will be removed at that point — key `2` becomes the Queue toggle.
- RecentlyPlayed data is loaded via the existing `FetchRecentlyPlayedRequestMsg` /
  `RecentlyPlayedLoadedMsg` flow. No new API calls needed.
- The `t` key for time range cycling only works when the pane is focused. It does NOT
  conflict with any global key binding.
