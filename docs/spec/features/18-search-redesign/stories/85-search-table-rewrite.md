---
title: "Rewrite Search Overlay Tables to Use components.Table"
feature: 18-search-redesign
status: open
---

## Background

The search overlay (stories 81-84) renders tables via manual string construction using
lipgloss and `fmt.Sprintf`. Every other pane in Spotnik (playlists, albums, liked songs,
top tracks, recently played, queue) delegates to `components.Table`, which wraps
`evertras/bubble-table` and provides: flex column widths, mouse scroll, selection
highlighting via `WithRowStyleFunc`, page indicators, and keyboard navigation.

This divergence causes three user-visible bugs:

1. **Row numbers don't update on pagination** — `renderActiveSection` uses `i+1` (loop
   index) instead of the absolute position. On page 4 of Albums (items 31-39 of 39),
   rows show 1-9 instead of 31-39. Every pane using `components.Table` bakes the
   absolute index into the row data (`fmt.Sprintf("%d", i+1)` over the full accumulated
   slice), so pagination displays correct numbers automatically.

2. **No mouse scroll** — `handleMouseMsg` in `routing.go:195` returns `nil` when
   `a.searchOpen == true`. This guard exists because `PaneAt(x, y)` hit-tests against
   the layout grid, which doesn't include overlays — scrolling would affect the hidden
   pane underneath, not the overlay. Mouse events need to be forwarded to the search
   overlay instead.

3. **Help bar keys not highlighted** — `renderHelpBar` renders everything in
   `TextMuted()`. The main status bar (`render.go:356-358`) and pane borders
   (`border.go:238`) use `KeyHint()` with `Bold(true)` for key labels and `TextMuted()`
   for descriptions. The search help bar should match.

Beyond these bugs, the manual rendering duplicates 300+ lines of column width arithmetic,
row styling, cursor management, and selection logic that `components.Table` already
handles. This story replaces the manual implementation with `components.Table` and adds
an accumulated page buffer with smart prefetch.

## Design

### Accumulated Search Buffer in the Store

**Pattern**: match playlists/albums which accumulate pages via `append(existing, new...)`.

Add per-section accumulated slices to the Store:

```go
// internal/state/store.go
searchBuffers struct {
    tracks    []SearchTrackItem
    artists   []SearchArtistItem
    albums    []SearchAlbumItem
    playlists []SearchPlaylistItem
}
searchTotals   [4]int          // total per section from API
searchFetched  [4]map[int]bool // which offsets have been fetched per section
searchBufQuery string          // query these buffers belong to
```

**Store methods:**

- `AppendSearchTracks(items []SearchTrackItem, offset int)` — appends items, marks
  offset as fetched. Similar methods for artists, albums, playlists.
- `SearchTracks() []SearchTrackItem` — returns the full accumulated buffer.
- `SearchSectionTotal(section int) int` — returns the API total for a section.
- `IsSearchOffsetFetched(section, offset int) bool` — guard against re-fetching.
- `ClearSearchBuffers()` — wipe all buffers + fetched map. Called on new query.

**New query flow:**
1. `SearchClearedMsg` or new `SearchResultsMsg` with `IsPaged: false` → `ClearSearchBuffers()`
2. Initial results arrive → `AppendSearchTracks(items, 0)` etc. for all 4 sections
3. Prefetch or user-triggered page → `AppendSearchTracks(newItems, offset)` for one section

### Per-Section `components.Table` Instances

Replace the manual cursor + rendering with 4 `components.Table` instances:

```go
// SearchOverlay struct changes
type SearchOverlay struct {
    // ... existing fields (store, theme, input, spinner, width, height)

    activeSection searchSection
    tables        [numSections]*components.Table  // one table per tab
    results       *SearchResultData               // removed — data lives in store now

    // cursorPos, sectionOffsets removed — bubble-table manages cursor and pagination
}
```

**Table construction** (per section, in `NewSearchOverlay` and `SetTheme`):

```go
// Tracks table — matches the column layout from story 82
trackCols := []components.ColumnDef{
    {Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
    {Key: "name", Header: "Track", FlexFactor: 9, Color: th.ColumnPrimary()},
    {Key: "artist", Header: "Artist", FlexFactor: 7, Color: th.ColumnSecondary()},
    {Key: "album", Header: "Album", FlexFactor: 7, Color: th.ColumnSecondary()},
    {Key: "duration", Header: "Duration", FlexFactor: 2, Color: th.ColumnTertiary()},
}
```

Albums, playlists, artists follow the same pattern with their respective columns and
flex factors matching the story 82 visual design.

**Row data format** — each row is `map[string]string` with absolute index:

```go
func (o *SearchOverlay) refreshTrackRows() {
    tracks := o.store.SearchTracks()
    rows := make([]map[string]string, len(tracks))
    for i, t := range tracks {
        rows[i] = map[string]string{
            "index":    fmt.Sprintf("%d", i+1),  // absolute position in buffer
            "name":     t.Name,
            "artist":   t.Artist,
            "album":    t.Album,
            "duration": formatDurationMs(t.DurationMs),
        }
    }
    o.tables[sectionTracks].SetRows(rows)
}
```

### Narrow Terminal Graceful Degradation

The current overlay drops the Album column for tracks when `contentWidth < 60`.
`components.Table` doesn't support conditional columns natively. Handle this by
rebuilding the tracks table with or without the album column in `SetSize()`:

```go
func (o *SearchOverlay) rebuildTrackTable() {
    cols := o.trackColumns() // includes album column check based on o.width
    o.tables[sectionTracks] = components.NewTable(components.TableConfig{
        Columns: cols, Theme: o.theme, PlayingIndex: -1, ShowHeader: true,
    })
    o.tables[sectionTracks].SetFocused(o.activeSection == sectionTracks)
    o.refreshTrackRows()
}
```

### Column Header Colors

`components.Table` uses `theme.TableHeader()` for all headers. The search overlay uses
per-tab colors (e.g., `PaneBorderTopTracks()` for Tracks headers). Two options:

**Option A** — Extend `components.TableConfig` with an optional `HeaderColor` override.
When set, `rebuild()` uses it instead of `th.TableHeader()`.

**Option B** — Accept `TableHeader()` color for consistency with all other panes.

**Recommendation**: Option A. The per-tab header colors are a deliberate design choice
from story 82 that reinforces tab identity. The change to `components.Table` is small:

```go
// components/table.go — add to TableConfig
HeaderColor lipgloss.Color // optional; overrides th.TableHeader() when non-empty

// In rebuild():
headerColor := th.TableHeader()
if t.config.HeaderColor != "" {
    headerColor = t.config.HeaderColor
}
inner := btable.New(btCols).
    HeaderStyle(lipgloss.NewStyle().Foreground(headerColor).Bold(false).Align(lipgloss.Left)).
    ...
```

### Smart Prefetch Algorithm

**Trigger**: when bubble-table's cursor crosses the 50% mark of the last fetched page.

The overlay doesn't directly see cursor moves inside bubble-table. Instead, after every
`table.Update(msg)` call, check whether a prefetch should fire:

```go
func (o *SearchOverlay) checkPrefetch() tea.Cmd {
    sec := o.activeSection
    cursor := o.tables[sec].SelectedIndex()  // absolute index in buffer
    bufLen := o.sectionBufferLen(sec)         // len of accumulated buffer
    total := o.store.SearchSectionTotal(int(sec))

    if bufLen >= total {
        return nil // all pages loaded
    }

    // Prefetch when cursor crosses 50% of the last page
    lastPageStart := bufLen - min(bufLen, maxResultsPerSection)
    midpoint := lastPageStart + maxResultsPerSection/2

    if cursor >= midpoint {
        nextOffset := bufLen // next unloaded offset
        if o.store.IsSearchOffsetFetched(int(sec), nextOffset) {
            return nil // already fetched or in-flight
        }
        return o.requestPage(nextOffset)
    }
    return nil
}
```

**Why 50%**: with 10 items per page and a cursor at item 5, there are 5 rows of visual
runway. At normal scroll speed (~200ms per arrow press), that's ~1 second — more than
enough to hide a typical API round-trip (~200-400ms). For fast scrolling (holding the
key), bubble-table's `WithPageSize` will show a page break at the buffer boundary, then
the prefetched data arrives and `refreshRows` extends the buffer seamlessly.

**Gateway safety**: the gateway's in-flight dedup (`inflight map[RequestKey]`) collapses
duplicate GET requests with the same path. Even if `checkPrefetch` fires twice before
the first request completes, only one API call executes.

**Buffer clearing**: new query → `store.ClearSearchBuffers()` → all 4 `refreshXxxRows()`
calls → `SetRows(nil)` → tables reset to empty. The fetched-offsets map is also cleared,
so subsequent prefetches for the new query start fresh.

### Mouse Scroll Forwarding

Modify `handleMouseMsg` in `routing.go` to forward wheel events to the search overlay
when open, instead of discarding them:

```go
func (a *App) handleMouseMsg(m tea.MouseMsg) tea.Cmd {
    // When search overlay is open, forward scroll events to it.
    if a.searchOpen {
        if m.Action == tea.MouseActionPress &&
            (m.Button == tea.MouseButtonWheelUp || m.Button == tea.MouseButtonWheelDown) {
            var scrollKey tea.KeyMsg
            if m.Button == tea.MouseButtonWheelUp {
                scrollKey = tea.KeyMsg{Type: tea.KeyUp}
            } else {
                scrollKey = tea.KeyMsg{Type: tea.KeyDown}
            }
            updated, cmd := a.searchPane.Update(scrollKey)
            if sp, ok := updated.(*SearchOverlay); ok {
                a.searchPane = sp
            }
            return cmd
        }
        return nil
    }
    // ... existing pane hit-test logic unchanged
}
```

This converts wheel scroll to up/down arrow keys, which the overlay routes to the
active table. The search input doesn't consume arrow keys when results are shown
(existing behavior), so scrolling works naturally.

### Help Bar Key Highlighting

Replace the single `TextMuted()` rendering with per-key `KeyHint()` + `TextMuted()`:

```go
func (o *SearchOverlay) renderHelpBar(contentWidth int) string {
    mutedStyle := lipgloss.NewStyle().Foreground(o.theme.TextMuted())
    keyStyle := lipgloss.NewStyle().Foreground(o.theme.KeyHint()).Bold(true)
    separator := mutedStyle.Render(strings.Repeat("─", contentWidth))

    hints := []struct{ Key, Label string }{
        {"Tab", "next section"},
        {"↑↓", "navigate"},
        {"Enter", "play"},
    }
    if o.activeSection == sectionTracks {
        hints = append(hints, struct{ Key, Label string }{"Ctrl+A", "queue"})
    }
    hints = append(hints, struct{ Key, Label string }{"Esc", "close"})

    var parts []string
    for _, h := range hints {
        parts = append(parts, keyStyle.Render(h.Key)+" "+mutedStyle.Render(h.Label))
    }
    keysLine := strings.Join(parts, "  ")

    // Right-align page indicator when present.
    indicator := o.pageIndicator()
    if indicator != "" {
        keysWidth := lipgloss.Width(keysLine)
        indicatorWidth := utf8.RuneCountInString(indicator)
        gap := contentWidth - keysWidth - indicatorWidth
        if gap > 0 {
            keysLine += strings.Repeat(" ", gap) + mutedStyle.Render(indicator)
        } else {
            keysLine += " " + mutedStyle.Render(indicator)
        }
    }

    return separator + "\n" + keysLine
}
```

### Page Indicator

With `components.Table` using `WithPageSize`, bubble-table renders its own `"1/2"` page
indicator at the bottom. However, for search we also need the `"1-10 of 39"` range
indicator in the help bar to communicate absolute position against the API total (not
just in-buffer pages). Keep `pageIndicator()` but recompute from the table's selected
index and the store's total:

```go
func (o *SearchOverlay) pageIndicator() string {
    sec := o.activeSection
    total := o.store.SearchSectionTotal(int(sec))
    bufLen := o.sectionBufferLen(sec)
    if total <= maxResultsPerSection {
        return ""
    }
    // Show "1-10 of 39" based on the visible page window
    cursor := o.tables[sec].SelectedIndex()
    pageSize := maxResultsPerSection
    pageStart := (cursor / pageSize) * pageSize
    start := pageStart + 1
    end := min(pageStart + pageSize, bufLen)
    if end > total {
        end = total
    }
    return fmt.Sprintf("%d-%d of %d", start, end, total)
}
```

### View() Assembly

The overlay `View()` structure stays similar but the active section is now rendered
by `components.Table.View()` instead of manual string construction:

```
╭─ Search ─────────────────────── Enter play  Esc close ──╮
│ > query                                                   │  ← textinput
│ ·······················                                   │  ← dot separator
│  ▪ Tracks 14   Artists 5   Albums 39   Playlists 12      │  ← renderTabBar (manual, kept)
│ ──────────────────────────────────────────────            │  ← tab separator (manual, kept)
│  [components.Table.View() for active section]             │  ← bubble-table handles headers,
│  # Album         Artist         Year  Tracks              │     rows, selection, pagination
│  1 Interlude     Benjamin M...  2025  1                   │     indicator ("1/4")
│  2 International Aadesh Shri... 1999  7                   │
│  ...                                                      │
│                                                 1/4       │  ← bubble-table page indicator
│ ──────────────────────────────────────────────            │  ← help bar separator
│  Tab next section  ↑↓ navigate  Enter play  Esc close    │  ← help bar with KeyHint colors
│                                              31-39 of 39  │  ← range indicator (right-aligned)
╰──────────────────────────────────────────────────────────╯
```

The `renderResults` method simplifies significantly: tab bar + tab separator + table
View + padding + help bar. The 300+ lines of `renderActiveSection`,
`renderColumnHeaders`, column width helpers, `searchCol` struct, `truncate()`,
`renderRow` closure — all deleted.

### SearchResultsMsg Routing Changes

**New query** (`IsPaged: false`):
1. `store.ClearSearchBuffers()`
2. `store.AppendSearchTracks(results.Tracks, 0)` etc. for all 4 sections
3. `store.SetSearchTotals(results.TotalTracks, ...)`
4. All 4 `refreshXxxRows()` → `SetRows()` with accumulated buffer
5. Switch to tracks tab, table focused

**Paginated load** (`IsPaged: true`):
1. `store.AppendSearchXxx(results.Items, offset)` for the relevant section
2. `refreshXxxRows()` for that section only
3. Bubble-table now has more rows in its buffer — user can keep scrolling

**Query change** (new `SearchRequestMsg`):
1. `store.ClearSearchBuffers()` is called when `SearchClearedMsg` fires
2. All tables get `SetRows(nil)`

### Deleted Code

The following functions/types in `search.go` are deleted entirely:

- `searchCol` struct
- `renderActiveSection()` + `RenderActiveSection()`
- `renderColumnHeaders()` + `RenderColumnHeaders()`
- `trackColumnWidths()`, `albumColumnWidths()`, `playlistColumnWidths()`,
  `artistColumnWidths()`
- `truncate()`
- `clampedTrackItems()`, `clampedArtistItems()`, `clampedAlbumItems()`,
  `clampedPlaylistItems()`
- `mergePageResults()`
- `moveCursorDown()`, `moveCursorUp()` — cursor movement handled by bubble-table
- `maxCursorForActiveSection()`
- `cursorPos`, `sectionOffsets` fields
- All exported test helpers for cursor/offset inspection

### Files Changed

| Action | File | Purpose |
|--------|------|---------|
| Modify | `internal/ui/components/table.go` | Add optional `HeaderColor` to `TableConfig` |
| Modify | `internal/ui/components/table_test.go` | Test `HeaderColor` override |
| Modify | `internal/state/store.go` | Add search buffer fields and methods |
| Modify | `internal/state/store_test.go` | Test buffer accumulation, clearing, fetched tracking |
| Modify | `internal/ui/panes/search.go` | Replace manual rendering with 4 `components.Table` instances, prefetch, help bar |
| Modify | `internal/ui/panes/search_test.go` | Rewrite tests for table-based rendering |
| Modify | `internal/ui/panes/messages.go` | Remove `sectionOffsets` test helpers if unused |
| Modify | `internal/app/routing.go` | Forward mouse scroll to search overlay when open |
| Modify | `internal/app/app.go` | Update `SearchResultsMsg` handler for store-based buffers |
| Modify | `internal/app/commands.go` | Update `convertSearchResult` to write to store buffers |
| Modify | `internal/app/app_test.go` | Update search-related tests |
| Modify | `internal/app/routing_test.go` | Add mouse scroll forwarding tests |

## Acceptance Criteria

- [ ] Search overlay renders all 4 tabs using `components.Table` (not manual rendering)
- [ ] Row numbers show absolute position (31-39, not 1-9) on paginated pages
- [ ] Mouse wheel scroll navigates rows in the search overlay when open
- [ ] Help bar keys use `KeyHint()` color with `Bold(true)`, labels use `TextMuted()`
- [ ] Column headers use per-tab colors (`PaneBorderTopTracks()` etc.) via `HeaderColor`
- [ ] Prefetch fires when cursor crosses 50% of the last fetched page
- [ ] Prefetch does not fire when all pages are loaded or offset already fetched
- [ ] New query clears all search buffers and resets all 4 tables
- [ ] Tab switch changes which table is visible and focused
- [ ] Tracks tab drops Album column on narrow terminals (`contentWidth < 60`)
- [ ] Page indicator shows `"1-10 of 39"` format in help bar (right-aligned)
- [ ] Bubble-table's native page indicator shows `"1/4"` below the table
- [ ] Gateway dedup prevents duplicate prefetch API calls
- [ ] Selected row uses `SelectedBg()/SelectedFg()` (via `WithRowStyleFunc`)
- [ ] Theme switching updates all table colors and help bar colors
- [ ] All existing search functionality preserved (Enter play, Ctrl+A queue, Esc close, Tab cycle)
- [ ] `make ci` passes

## Tasks

- [ ] **Add `HeaderColor` to `components.Table`** — add optional `HeaderColor lipgloss.Color`
      field to `TableConfig`. In `rebuild()`, use it instead of `th.TableHeader()` when
      non-empty. In `internal/ui/components/table.go`.
      - test: `TestTable_HeaderColor_Override` — verify custom header color applied
      - test: `TestTable_HeaderColor_Default` — verify `TableHeader()` used when empty

- [ ] **Add search buffer to Store** — add per-section accumulated slices, totals array,
      fetched-offsets map, and buffer query string. Add `AppendSearchXxx`, `SearchXxx`,
      `SearchSectionTotal`, `IsSearchOffsetFetched`, `ClearSearchBuffers` methods.
      In `internal/state/store.go`.
      - test: `TestStore_AppendSearchTracks` — verify accumulation
      - test: `TestStore_ClearSearchBuffers` — verify wipe on new query
      - test: `TestStore_IsSearchOffsetFetched` — verify tracking

- [ ] **Replace manual tables with `components.Table` instances** — add
      `tables [numSections]*components.Table` to `SearchOverlay`. Create 4 tables in
      `NewSearchOverlay` with per-section columns and `HeaderColor` set to the tab color.
      Add `refreshTrackRows()`, `refreshArtistRows()`, `refreshAlbumRows()`,
      `refreshPlaylistRows()` that read from store buffers and call `SetRows()`.
      Delete `renderActiveSection`, `renderColumnHeaders`, all column width helpers,
      `searchCol`, `truncate`, `clampedXxxItems`, `cursorPos`, `sectionOffsets`.
      In `internal/ui/panes/search.go`.
      - test: `TestSearchOverlay_TracksTable_ColumnDefs` — verify correct columns and colors
      - test: `TestSearchOverlay_AlbumsTable_ColumnDefs`
      - test: `TestSearchOverlay_NarrowDropsAlbumColumn` — verify tracks table rebuilt without album

- [ ] **Update View() to use table.View()** — simplify `renderResults` to:
      tab bar + separator + `o.tables[o.activeSection].View()` + padding + help bar.
      Pass table dimensions via `SetSize()` in the overlay's `SetSize()` method.
      In `internal/ui/panes/search.go`.
      - test: `TestSearchOverlay_View_ContainsTableOutput` — verify table View() output present
      - test: `TestSearchOverlay_View_HelpBarAnchored` — verify help bar at bottom

- [ ] **Wire store-based pagination** — update `SearchResultsMsg` handler: new query →
      `store.ClearSearchBuffers()` + append all sections; paged → append single section.
      Update `refreshXxxRows()` after each. Remove `mergePageResults`. In
      `internal/ui/panes/search.go`, `internal/app/app.go`.
      - test: `TestSearchOverlay_NewQuery_ClearsBuffers`
      - test: `TestSearchOverlay_PagedResult_AccumulatesBuffer`
      - test: `TestSearchOverlay_RowNumbers_Absolute` — verify row 31 shows "31" not "1"

- [ ] **Add smart prefetch** — add `checkPrefetch()` method called after every
      `table.Update(msg)`. Fires `SearchPageRequestMsg` when cursor crosses 50% of
      last page and next offset not yet fetched. In `internal/ui/panes/search.go`.
      - test: `TestPrefetch_Fires_AtMidpoint` — cursor at item 5 triggers prefetch
      - test: `TestPrefetch_NoFire_AllLoaded` — no prefetch when buffer == total
      - test: `TestPrefetch_NoFire_AlreadyFetched` — no prefetch for known offset
      - test: `TestPrefetch_NoFire_BelowMidpoint` — cursor at item 3 does not trigger

- [ ] **Forward mouse scroll to search overlay** — modify `handleMouseMsg` in
      `routing.go`: when `a.searchOpen`, convert wheel up/down to KeyUp/KeyDown and
      forward to `a.searchPane.Update()`. In `internal/app/routing.go`.
      - test: `TestMouseScroll_SearchOpen_ForwardsToOverlay`
      - test: `TestMouseScroll_SearchOpen_IgnoresNonWheel`

- [ ] **Highlight help bar keys** — rewrite `renderHelpBar` to use `KeyHint()` +
      `Bold(true)` for key labels and `TextMuted()` for descriptions, matching the
      main status bar pattern from `render.go:356-358`. In `internal/ui/panes/search.go`.
      - test: `TestRenderHelpBar_KeysUseKeyHintColor`
      - test: `TestRenderHelpBar_LabelsUseTextMutedColor`

- [ ] **Update SetTheme** — rebuild all 4 tables with new theme colors and `HeaderColor`.
      Re-read store buffers via `refreshXxxRows()`. In `internal/ui/panes/search.go`.
      - test: `TestSearchOverlay_SetTheme_RebuildsTables`

- [ ] **Update app.go SearchResultsMsg handler** — route results to store buffers
      instead of directly to the overlay. Call `store.AppendSearchXxx` then forward
      the msg to the overlay for `refreshRows`. In `internal/app/app.go`,
      `internal/app/commands.go`.
      - test: `TestApp_SearchResults_WritesToStore`
      - test: `TestApp_SearchPageResults_AccumulatesInStore`
