---
title: "Search Overlay Tabbed Redesign"
feature: 18-search-redesign
status: open
---

## Background

The current search overlay is narrow (max 50 chars) and stacks all 4 result sections
vertically with minimal metadata. Names get heavily truncated, there's no duration or
album info for tracks, and users must scroll through all sections to find what they want.

This story redesigns the overlay into a wide tabbed interface where each section gets
full-width table presentation with rich metadata columns. The tab bar shows result counts
so users can immediately see which sections have matches.

**API constraints (Feb 2026 changes):**
- Search limit is now **max 10 per type** (reduced from 50) — we currently use 5, this
  story bumps to 10 (the API max). Pagination via `offset` is available but out of scope.
- `Artist.followers` and `Artist.popularity` fields have been **removed** from the API —
  the Artists tab can only show name (no genre, no follower count).
- `Album.label` and `Album.popularity` also removed — no impact on our design.

## Design

### Phase 1: Enrich Data Types

The Spotify API already returns rich metadata that the conversion layer currently discards.
Extend the UI-side item types and the `convertSearchResult` function to carry it through.

#### Extend SearchTrackItem

```go
// internal/ui/panes/messages.go

type SearchTrackItem struct {
    URI        string
    Name       string
    Artist     string
    Album      string // NEW: album name from Track.Album.Name
    DurationMs int    // NEW: duration from Track.DurationMs
}
```

#### Extend SearchAlbumItem

```go
type SearchAlbumItem struct {
    URI         string
    Name        string
    Artist      string
    ReleaseYear string // NEW: first 4 chars of SearchAlbum.ReleaseDate (e.g. "2024")
    TotalTracks int    // NEW: from SearchAlbum.TotalTracks
}
```

#### Extend SearchPlaylistItem

```go
type SearchPlaylistItem struct {
    URI        string
    Name       string
    Owner      string
    TrackCount int    // NEW: from SearchPlaylist.TrackCount
}
```

#### Add Total Counts to SearchResultData

```go
type SearchResultData struct {
    Tracks    []SearchTrackItem
    Artists   []SearchArtistItem
    Albums    []SearchAlbumItem
    Playlists []SearchPlaylistItem

    // Total counts from the API (may exceed len of Items slices due to limit)
    TotalTracks    int // NEW
    TotalArtists   int // NEW
    TotalAlbums    int // NEW
    TotalPlaylists int // NEW
}
```

#### Update convertSearchResult in commands.go

```go
func convertSearchResult(r *api.SearchResult) *panes.SearchResultData {
    if r == nil {
        return nil
    }

    data := &panes.SearchResultData{
        TotalTracks:    r.Tracks.Total,
        TotalArtists:   r.Artists.Total,
        TotalAlbums:    r.Albums.Total,
        TotalPlaylists: r.Playlists.Total,
    }

    for _, t := range r.Tracks.Items {
        item := panes.SearchTrackItem{
            URI:        t.URI,
            Name:       t.Name,
            DurationMs: t.DurationMs,
        }
        if len(t.Artists) > 0 {
            item.Artist = t.Artists[0].Name
        }
        item.Album = t.Album.Name
        data.Tracks = append(data.Tracks, item)
    }

    for _, a := range r.Artists.Items {
        data.Artists = append(data.Artists, panes.SearchArtistItem{
            URI:  a.URI,
            Name: a.Name,
        })
    }

    for _, a := range r.Albums.Items {
        item := panes.SearchAlbumItem{
            URI:         a.URI,
            Name:        a.Name,
            TotalTracks: a.TotalTracks,
        }
        if len(a.Artists) > 0 {
            item.Artist = a.Artists[0].Name
        }
        if len(a.ReleaseDate) >= 4 {
            item.ReleaseYear = a.ReleaseDate[:4]
        }
        data.Albums = append(data.Albums, item)
    }

    for _, p := range r.Playlists.Items {
        data.Playlists = append(data.Playlists, panes.SearchPlaylistItem{
            URI:        p.URI,
            Name:       p.Name,
            Owner:      p.Owner.DisplayName,
            TrackCount: p.TrackCount,
        })
    }

    return data
}
```

### Phase 2: Widen Overlay Dimensions

#### overlayWidth

```go
func (o *SearchOverlay) overlayWidth() int {
    w := 90
    if o.width > 0 {
        eightyPct := o.width * 80 / 100
        if eightyPct < w {
            w = eightyPct
        }
    }
    if w < 40 {
        w = 40
    }
    return w
}
```

- Base: 90 chars (was 50)
- Cap: 80% terminal (was 60%)
- Min: 40 chars (was 20)

#### overlayHeight

```go
func (o *SearchOverlay) overlayHeight() int {
    h := 26
    if o.height > 0 {
        seventyFivePct := o.height * 75 / 100
        if seventyFivePct > h {
            h = seventyFivePct
        }
    }
    if h < 12 {
        h = 12
    }
    return h
}
```

- Base: 26 rows (was 20)
- Cap: 75% terminal (was 70%)
- Min: 12 rows (was 8)

#### Increase maxResultsPerSection

```go
const maxResultsPerSection = 10  // was 5 — matches Feb 2026 API max of 10 per type
```

### Phase 3: Theme-Aware Tab Colors

Each tab has a distinct color reusing existing `PaneBorder*()` theme tokens — zero
TOML changes required. This maintains the visual metaphor: "cyan = albums" everywhere
in the app, whether in the Albums pane border or the Albums search tab.

#### Tab-to-token mapping

| Tab | Theme Token | Black Theme | Dracula |
|-----|-------------|-------------|---------|
| Tracks | `PaneBorderTopTracks()` | `#bd93f9` purple | `#FF79C6` pink |
| Artists | `PaneBorderTopArtists()` | `#ff79c6` pink | `#FF5555` red |
| Albums | `PaneBorderAlbums()` | `#00e5cc` cyan | `#8BE9FD` cyan |
| Playlists | `PaneBorderPlaylists()` | `#00afff` blue | `#BD93F9` purple |

All 11 TOML themes already define these tokens in their `[pane_borders]` section, so
every theme automatically gets distinct, on-brand tab colors with no additional work.

#### Tab color method

```go
// tabColorForSection returns the accent color for a given search section tab.
// Reuses pane border tokens for visual consistency across the app.
func (o *SearchOverlay) tabColorForSection(sec searchSection) lipgloss.Color {
    switch sec {
    case sectionTracks:
        return o.theme.PaneBorderTopTracks()
    case sectionArtists:
        return o.theme.PaneBorderTopArtists()
    case sectionAlbums:
        return o.theme.PaneBorderAlbums()
    case sectionPlaylists:
        return o.theme.PaneBorderPlaylists()
    default:
        return o.theme.ActiveBorder()
    }
}
```

### Phase 4: Tab Bar Rendering

Replace the stacked section headers with a horizontal tab bar. The tab bar renders
below the separator line, showing all 4 sections with their result counts. Each tab
uses its own color from `tabColorForSection`.

#### Tab bar format

```
 ▪ Tracks 10     Artists 5     Albums 8     Playlists 12
   (purple)       (muted)       (muted)       (muted)
```

- Active tab: `▪` bullet prefix + bold + `tabColorForSection(sec)` color
- Inactive tabs: no bullet + `TextMuted()` color
- Counts come from `SearchResultData.TotalTracks` etc.
- Tab labels are fixed strings, counts are dynamic
- Tabs are separated by 5 spaces

#### Tab bar rendering method

```go
func (o *SearchOverlay) renderTabBar(width int) string {
    type tabDef struct {
        label string
        total int
    }
    tabs := []tabDef{
        {"Tracks", o.totalForSection(sectionTracks)},
        {"Artists", o.totalForSection(sectionArtists)},
        {"Albums", o.totalForSection(sectionAlbums)},
        {"Playlists", o.totalForSection(sectionPlaylists)},
    }

    var parts []string
    for i, tab := range tabs {
        text := fmt.Sprintf("%s %d", tab.label, tab.total)
        sec := searchSection(i)
        if sec == o.activeSection {
            activeStyle := lipgloss.NewStyle().
                Foreground(o.tabColorForSection(sec)).
                Bold(true)
            parts = append(parts, activeStyle.Render("▪ "+text))
        } else {
            inactiveStyle := lipgloss.NewStyle().
                Foreground(o.theme.TextMuted())
            parts = append(parts, inactiveStyle.Render("  "+text))
        }
    }
    return " " + strings.Join(parts, "     ")
}

func (o *SearchOverlay) totalForSection(sec searchSection) int {
    if o.results == nil {
        return 0
    }
    switch sec {
    case sectionTracks:
        return o.results.TotalTracks
    case sectionArtists:
        return o.results.TotalArtists
    case sectionAlbums:
        return o.results.TotalAlbums
    case sectionPlaylists:
        return o.results.TotalPlaylists
    }
    return 0
}
```

### Phase 5: Column Headers Per Section

Each section has its own column layout. Below the tab bar separator, render a header
row with column names and an underline row. **Header text uses the active tab's
`tabColorForSection()` color** — reinforcing tab identity in the table.

#### Column definitions

| Section | Columns | Widths (proportional within content area) |
|---|---|---|
| Tracks | #, Track, Artist, Album, Duration | 3, 35%, 25%, 25%, 8 |
| Artists | #, Artist | 3, remaining |
| Albums | #, Album, Artist, Year, Tracks | 3, 35%, 25%, 6, 8 |
| Playlists | #, Playlist, Owner, Tracks | 3, 40%, 30%, 8 |

Column widths are computed relative to contentWidth. The `#` column is always 3 chars.
Duration/Year/Tracks columns are fixed-width. Name columns fill the remaining space.

#### Narrow terminal graceful degradation

If `contentWidth < 60`, the Tracks tab drops the Album column (5 → 4 columns) to avoid
extreme truncation. The remaining columns expand to fill the space. All other tabs have
fewer columns and degrade naturally.

#### Header rendering

```go
func (o *SearchOverlay) renderColumnHeaders(sec searchSection, contentWidth int) string {
    // Returns two lines: header labels + underline dashes
    // Header text styled with tabColorForSection(sec) + bold
    headerStyle := lipgloss.NewStyle().
        Foreground(o.tabColorForSection(sec)).
        Bold(true)
    // Underline uses TextMuted()
    underlineStyle := lipgloss.NewStyle().
        Foreground(o.theme.TextMuted())
    // ...
}
```

### Phase 6: Table-Style Result Rows with Column Colors

Replace the current `▶ name  artist` format with numbered column-aligned rows.
Each column uses a semantic color token from the Theme interface — the same pattern
used by QueuePane, LikedSongsPane, TopTracksPane, and all other table panes.

#### Column color mapping

| Column Role | Theme Token | Purpose |
|---|---|---|
| `#` (index) | `ColumnIndex()` | Muted but colorful row number |
| Name (track/album/playlist/artist) | `ColumnPrimary()` | Vibrant main data |
| Artist/Owner | `ColumnSecondary()` | Contrasting supporting info |
| Duration/Year/TrackCount | `ColumnTertiary()` | Additional accent metadata |

**Selected row override:** When a row is selected (`▶` prefix), all columns use
`SelectedBg()` / `SelectedFg()` — the semantic column colors are suppressed to
make the selection highlight clear, matching how all other panes handle selection.

#### Track row format

```
 ▶  Alpha                          Kingside              Alpha EP           3:42
 2  Forever Young                  Alphaville            Forever Young      3:44
 ─  ColumnPrimary                  ColumnSecondary       ColumnSecondary    ColumnTertiary
```

#### Album row format

```
 ▶  Happy Patel – Khatarnak        Vir Das               2023    12
 2  Alpha's Goodbye                King                  2024    8
 ─  ColumnPrimary                  ColumnSecondary       ColTertiary ColTertiary
```

#### Playlist row format

```
 ▶  Alpha Male Songs               Am I real?            45
 2  Alpha songs                    SIDAN                 23
 ─  ColumnPrimary                  ColumnSecondary       ColumnTertiary
```

#### Artist row format

```
 ▶  Alphaville
 2  Alpha Blondy
 ─  ColumnPrimary
```

### Phase 7: Help Bar

A bottom separator line followed by a contextual help line showing available keybindings.

```go
func (o *SearchOverlay) renderHelpBar(contentWidth int) string {
    help := "Tab next section  ↑↓ navigate  Enter play"
    if o.activeSection == sectionTracks {
        help += "  Ctrl+A queue"
    }
    help += "  Esc close"

    separator := strings.Repeat("─", contentWidth)
    sepStyle := lipgloss.NewStyle().Foreground(o.theme.TextMuted())
    helpStyle := lipgloss.NewStyle().Foreground(o.theme.TextMuted())

    return sepStyle.Render(separator) + "\n" + helpStyle.Render(" "+help)
}
```

The help bar is contextual: `Ctrl+A queue` only appears on the Tracks tab (it's the
only section that supports add-to-queue).

### Phase 8: Rewrite View() and renderResults()

#### New View() structure

```
Line 1: text input (> query)
Line 2: dot separator
Line 3: tab bar (▪ Tracks 10  Artists 5  Albums 8  Playlists 12)
Line 4: tab separator (────)
Line 5: column headers
Line 6: column underlines (─────)
Lines 7-N: result rows (up to 8)
Line N+1: bottom separator (────)
Line N+2: help bar
```

#### Updated renderResults

```go
func (o *SearchOverlay) renderResults(innerWidth int) string {
    // ... existing loading/empty/no-results checks ...

    contentWidth := innerWidth - 4
    if contentWidth < 20 {
        contentWidth = 20
    }

    var sb strings.Builder

    // Tab bar
    sb.WriteString(o.renderTabBar(contentWidth))
    sb.WriteString("\n")

    // Tab separator
    sepStyle := lipgloss.NewStyle().Foreground(o.theme.TextMuted())
    sb.WriteString(sepStyle.Render(strings.Repeat("─", contentWidth)))
    sb.WriteString("\n")

    // Column headers for active section
    sb.WriteString(o.renderColumnHeaders(o.activeSection, contentWidth))

    // Result rows for active section only
    sb.WriteString(o.renderActiveSection(contentWidth))
    sb.WriteString("\n")

    // Help bar
    sb.WriteString(o.renderHelpBar(contentWidth))

    return sb.String()
}
```

### Phase 9: Duration Formatting

Track durations arrive as milliseconds. Format as `m:ss` for tracks under an hour,
`h:mm:ss` for tracks an hour or longer.

Check if `timeutil.FormatDuration` already handles this. If so, reuse it. If not, add
a helper:

```go
func formatDurationMs(ms int) string {
    totalSec := ms / 1000
    if totalSec >= 3600 {
        h := totalSec / 3600
        m := (totalSec % 3600) / 60
        s := totalSec % 60
        return fmt.Sprintf("%d:%02d:%02d", h, m, s)
    }
    m := totalSec / 60
    s := totalSec % 60
    return fmt.Sprintf("%d:%02d", m, s)
}
```

### Files Changed

| Action | File | Purpose |
|---|---|---|
| Modify | `internal/ui/panes/messages.go` | Add Album, DurationMs to SearchTrackItem; add ReleaseYear, TotalTracks to SearchAlbumItem; add TrackCount to SearchPlaylistItem; add TotalTracks/TotalArtists/TotalAlbums/TotalPlaylists to SearchResultData |
| Modify | `internal/app/commands.go` | Update convertSearchResult to populate new fields |
| Modify | `internal/ui/panes/search.go` | Widen dimensions, increase maxResultsPerSection to 10, add renderTabBar, renderColumnHeaders, renderActiveSection, renderHelpBar, formatDurationMs; rewrite View() and renderResults() |
| Modify | `internal/app/commands.go` | Update `buildSearchCmd` to pass `limit=10` (was 5) to match new maxResultsPerSection |
| Modify | `internal/ui/panes/search_test.go` | Update tests for new dimensions, tab bar, column headers, enriched data, help bar |
| Modify | `internal/app/commands_test.go` | Update convertSearchResult tests for new fields |

## Acceptance Criteria

- [ ] Overlay width is `min(90, 80% terminal)` — nearly double the current 50-char max
- [ ] Overlay height is `max(26, 75% terminal)`
- [ ] Tab bar renders all 4 sections with result counts (e.g. "Tracks 10")
- [ ] Active tab is visually distinct (▪ marker + bold + tab-specific color)
- [ ] Each tab has a distinct color via `PaneBorder*` theme tokens (Tracks=TopTracks, Artists=TopArtists, Albums=Albums, Playlists=Playlists)
- [ ] Inactive tabs use `TextMuted()` color
- [ ] Only the active section's results are shown
- [ ] Tab/Shift+Tab cycles through tabs, resets cursor to 0
- [ ] Tracks tab: columns #, Track, Artist, Album, Duration
- [ ] Albums tab: columns #, Album, Artist, Year, Tracks
- [ ] Playlists tab: columns #, Playlist, Owner, Tracks
- [ ] Artists tab: columns #, Artist
- [ ] Column headers styled with active tab's `tabColorForSection()` color + bold
- [ ] Column data uses `ColumnIndex/Primary/Secondary/Tertiary` theme tokens (same as all table panes)
- [ ] Selected row overrides column colors with `SelectedBg()/SelectedFg()`
- [ ] Tracks tab drops Album column when `contentWidth < 60` (graceful narrow terminal degradation)
- [ ] Up to 10 results per section (API max since Feb 2026)
- [ ] Column headers with underline separators
- [ ] Bottom help bar with contextual keybindings (Ctrl+A only on Tracks tab)
- [ ] Duration formatted as m:ss or h:mm:ss
- [ ] Existing keybindings unchanged: Enter, Ctrl+A, Esc, ↑↓, Tab/Shift+Tab
- [ ] Theme switching (`t` key) updates all tab colors and column colors correctly
- [ ] `make ci` passes

## Tasks

- [ ] **Enrich SearchTrackItem** — add `Album string` and `DurationMs int` fields to `SearchTrackItem` in `messages.go`
      - test: `TestSearchTrackItem_HasAlbumAndDuration` (verify struct fields exist and are populated)
- [ ] **Enrich SearchAlbumItem** — add `ReleaseYear string` and `TotalTracks int` fields to `SearchAlbumItem` in `messages.go`
      - test: `TestSearchAlbumItem_HasYearAndTotalTracks`
- [ ] **Enrich SearchPlaylistItem** — add `TrackCount int` field to `SearchPlaylistItem` in `messages.go`
      - test: `TestSearchPlaylistItem_HasTrackCount`
- [ ] **Add total counts to SearchResultData** — add `TotalTracks`, `TotalArtists`, `TotalAlbums`, `TotalPlaylists` int fields
      - test: `TestSearchResultData_HasTotalCounts`
- [ ] **Update convertSearchResult** — populate Album, DurationMs for tracks; ReleaseYear, TotalTracks for albums; TrackCount for playlists; all Total counts. In `internal/app/commands.go`.
      - test: `TestConvertSearchResult_EnrichedFields` (verify all new fields are populated from domain types)
- [ ] **Widen overlay dimensions** — change `overlayWidth` to `min(90, 80% terminal)` min 40; change `overlayHeight` to `max(26, 75% terminal)` min 12; increase `maxResultsPerSection` to 10
      - test: `TestOverlayWidth_Wider`, `TestOverlayHeight_Taller`, `TestMaxResultsPerSection_Ten`
- [ ] **Bump search API limit** — update `buildSearchCmd` in `commands.go` to pass `limit=10` (was 5) to `SearchClient.Search()`, matching the new `maxResultsPerSection` and the Feb 2026 API max
      - test: `TestBuildSearchCmd_Limit10`
- [ ] **Add tabColorForSection** — maps each section to its `PaneBorder*` theme token: Tracks→`PaneBorderTopTracks()`, Artists→`PaneBorderTopArtists()`, Albums→`PaneBorderAlbums()`, Playlists→`PaneBorderPlaylists()`. Falls back to `ActiveBorder()`.
      - test: `TestTabColorForSection_Tracks`, `TestTabColorForSection_Artists`, `TestTabColorForSection_Albums`, `TestTabColorForSection_Playlists`
- [ ] **Add renderTabBar** — renders horizontal tab bar with section labels and counts; active tab has ▪ + bold + `tabColorForSection(sec)` color, inactive tabs use `TextMuted()`
      - test: `TestRenderTabBar_ActiveHighlighted`, `TestRenderTabBar_ShowsCounts`, `TestRenderTabBar_NilResults_ZeroCounts`, `TestRenderTabBar_ActiveTabUsesTabColor`
- [ ] **Add totalForSection helper** — returns the total count for a given section from SearchResultData
      - test: `TestTotalForSection_AllSections`, `TestTotalForSection_NilResults`
- [ ] **Add renderColumnHeaders** — returns header labels + underline row for each section type; header text styled with `tabColorForSection(activeSection)` + bold; underline uses `TextMuted()`. If `contentWidth < 60`, Tracks tab drops Album column (4 columns instead of 5).
      - test: `TestRenderColumnHeaders_Tracks`, `TestRenderColumnHeaders_Artists`, `TestRenderColumnHeaders_Albums`, `TestRenderColumnHeaders_Playlists`, `TestRenderColumnHeaders_Tracks_NarrowDropsAlbum`
- [ ] **Add renderActiveSection** — renders numbered, column-aligned rows for the active section only; selected row uses ▶ prefix + `SelectedBg()`/`SelectedFg()` (overrides column colors); unselected rows use `ColumnIndex()` for #, `ColumnPrimary()` for name, `ColumnSecondary()` for artist/owner, `ColumnTertiary()` for duration/year/count
      - test: `TestRenderActiveSection_Tracks_ShowsAlbumAndDuration`, `TestRenderActiveSection_Albums_ShowsYearAndCount`, `TestRenderActiveSection_Playlists_ShowsTrackCount`, `TestRenderActiveSection_SelectedRow_Highlight`, `TestRenderActiveSection_Tracks_NarrowNoAlbumColumn`
- [ ] **Add formatDurationMs** — format milliseconds as `m:ss` or `h:mm:ss`; reuse timeutil if suitable
      - test: `TestFormatDurationMs_ShortTrack`, `TestFormatDurationMs_LongTrack`, `TestFormatDurationMs_Zero`
- [ ] **Add renderHelpBar** — bottom separator + contextual keybindings; Ctrl+A only on Tracks tab
      - test: `TestRenderHelpBar_TracksTab_ShowsCtrlA`, `TestRenderHelpBar_OtherTab_NoCtrlA`
- [ ] **Rewrite View() and renderResults()** — assemble: input → separator → tab bar → tab separator → column headers → result rows → help bar; use lipgloss Width/Height capping
      - test: `TestSearchOverlayView_ContainsTabBar`, `TestSearchOverlayView_ContainsColumnHeaders`, `TestSearchOverlayView_ContainsHelpBar`
- [ ] **Update border actions** — change border Actions from `[{Enter, play}, {Tab, section}]` to `[{Enter, play}, {Esc, close}]` since help bar now shows full keybindings
      - test: `TestSearchOverlayView_BorderActions`
