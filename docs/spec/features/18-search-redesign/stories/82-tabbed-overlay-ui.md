---
title: "Tabbed Search Overlay with Theme-Aware Rendering"
feature: 18-search-redesign
status: done
---

## Background

Story 81 enriches the search data types and widens the overlay. This story replaces the
stacked-section rendering with a tabbed interface: one section visible at a time, with
a tab bar showing counts, column headers, table-style rows, and a contextual help bar.

**Theming is not a bolt-on** — every rendering method uses theme tokens from the start.
The codebase already has `PaneBorder*` tokens for per-pane identity colors and `Column*`
tokens for table data. This story reuses both: tabs get pane border colors, table rows
get column colors. Zero TOML changes — all 11 themes work automatically.

## Design

### Color Strategy (applies to all rendering below)

Every method described below uses these tokens. They are not a separate phase — they
are woven into each method's rendering logic.

#### Tab colors — reuse PaneBorder* tokens

| Tab | Theme Token | Black Theme | Dracula |
|-----|-------------|-------------|---------|
| Tracks | `PaneBorderTopTracks()` | `#bd93f9` purple | `#FF79C6` pink |
| Artists | `PaneBorderTopArtists()` | `#ff79c6` pink | `#FF5555` red |
| Albums | `PaneBorderAlbums()` | `#00e5cc` cyan | `#8BE9FD` cyan |
| Playlists | `PaneBorderPlaylists()` | `#00afff` blue | `#BD93F9` purple |

The visual metaphor: "cyan = albums" everywhere — pane border, search tab, column headers.

#### Column colors — reuse Column* tokens

| Column Role | Theme Token | Usage |
|---|---|---|
| `#` (row number) | `ColumnIndex()` | Muted colorful index |
| Name (track/album/playlist/artist) | `ColumnPrimary()` | Vibrant main data |
| Artist/Owner | `ColumnSecondary()` | Contrasting secondary |
| Duration/Year/TrackCount | `ColumnTertiary()` | Additional accent |

#### Selection override

Selected row: `SelectedBg()` + `SelectedFg()` — overrides all column colors.
Matches how QueuePane, LikedSongsPane, and all other table panes handle selection.

### Tab Bar

Replace stacked section headers with a horizontal tab bar below the dot separator.

```
 ▪ Tracks 10     Artists 5     Albums 8     Playlists 12
```

- **Active tab**: `▪` bullet + bold + `tabColorForSection(sec)` (PaneBorder* token)
- **Inactive tabs**: no bullet + `TextMuted()` color
- Counts from `SearchResultData.TotalTracks` etc. (0 if nil results)
- Tabs separated by 5 spaces

**New method: `tabColorForSection(sec searchSection) lipgloss.Color`** — switch on section,
return `PaneBorderTopTracks()` / `PaneBorderTopArtists()` / `PaneBorderAlbums()` /
`PaneBorderPlaylists()`, default `ActiveBorder()`.

**New method: `totalForSection(sec searchSection) int`** — return total count for section
from SearchResultData (0 if nil).

**New method: `renderTabBar(width int) string`** — render the tab bar as described above.

### Column Headers

Each section has its own column layout. Below the tab bar separator, render a header
row and an underline row.

**Header text**: styled with `tabColorForSection(activeSection)` + bold
**Underline**: `TextMuted()` dashes

| Section | Columns | Fixed Widths |
|---|---|---|
| Tracks | #, Track, Artist, Album, Duration | 3, 35%, 25%, 25%, 8 |
| Artists | #, Artist | 3, remaining |
| Albums | #, Album, Artist, Year, Tracks | 3, 35%, 25%, 6, 8 |
| Playlists | #, Playlist, Owner, Tracks | 3, 40%, 30%, 8 |

The `#` column is always 3 chars. Duration/Year/Tracks columns are fixed-width.
Name columns fill remaining space proportionally.

**Narrow terminal graceful degradation**: if `contentWidth < 60`, the Tracks tab drops
the Album column (5 → 4 columns). All other tabs have fewer columns and degrade naturally.

**New method: `renderColumnHeaders(sec searchSection, contentWidth int) string`** —
return header + underline lines for the given section.

### Table-Style Result Rows

Replace the current `▶ name  artist` format with numbered, column-aligned rows.
Only the active section's results are shown (not all 4 stacked).

**Selected row** (`▶` prefix): `SelectedBg()` / `SelectedFg()` — overrides column colors.
**Unselected rows** (number prefix): per-column colors from `ColumnIndex/Primary/Secondary/Tertiary`.

Track row format:
```
 ▶  Alpha                          Kingside              Alpha EP           3:42
 2  Forever Young                  Alphaville            Forever Young      3:44
```

Album row format (Year and Tracks in tertiary):
```
 ▶  Happy Patel – Khatarnak        Vir Das               2023    12
```

Playlist row format:
```
 ▶  Alpha Male Songs               Am I real?            45
```

Artist row format (just primary):
```
 ▶  Alphaville
```

**New method: `renderActiveSection(contentWidth int) string`** — render rows for the
active section only, with column alignment and per-column colors.

### Duration Formatting

Track durations arrive as milliseconds. Format as `m:ss` for < 1 hour, `h:mm:ss` for >= 1 hour.
Check if `timeutil.FormatDuration` already handles this. If so, reuse it. If not, add a
`formatDurationMs(ms int) string` helper in `search.go`.

### Help Bar

Bottom separator + contextual keybindings line, both in `TextMuted()`.

```
Tab next section  ↑↓ navigate  Enter play  Ctrl+A queue  Esc close
```

`Ctrl+A queue` only appears on the Tracks tab (only section that supports add-to-queue).

**New method: `renderHelpBar(contentWidth int) string`** — separator line + help text.

### Rewrite View() and renderResults()

New `renderResults()` assembly order:
```
tab bar
tab separator (───)
column headers + underline
result rows (active section only)
[spacer if needed]
help separator (───)
help bar
```

The input line and dot separator remain in `View()` above the results area.

### Remove Old Rendering

Delete these methods (replaced by the new rendering):
- `renderSection()` — replaced by `renderActiveSection()`
- `clampedTrackItemsAsRows()`, `clampedArtistItemsAsRows()`, `clampedAlbumItemsAsRows()`,
  `clampedPlaylistItemsAsRows()` — replaced by column-aligned rendering in `renderActiveSection()`

Keep these helpers (still needed):
- `clampedTrackItems()`, `clampedArtistItems()`, `clampedAlbumItems()`, `clampedPlaylistItems()` — for clamping slices
- `truncate()` — for column text truncation

### Update Section Labels and Border Actions

- `searchSectionLabels`: `"TRACKS"` → `"Tracks"` etc. (title case for tab bar)
- Border Actions: change from `[{Enter, play}, {Tab, section}]` to `[{Enter, play}, {Esc, close}]`
  since the help bar now shows full keybindings

### Files Changed

| Action | File | Purpose |
|---|---|---|
| Modify | `internal/ui/panes/search.go` | Add tabColorForSection, totalForSection, renderTabBar, renderColumnHeaders, renderActiveSection, formatDurationMs, renderHelpBar; rewrite renderResults/View; delete renderSection and *AsRows helpers; update labels and border actions |
| Modify | `internal/ui/panes/search_test.go` | ~20 new tests + update existing for label/action changes |

## Acceptance Criteria

- [ ] Tab bar renders all 4 sections with result counts (e.g. "Tracks 10")
- [ ] Active tab has ▪ marker + bold + distinct color from `PaneBorder*` token
- [ ] Each tab has a different color (Tracks≠Artists≠Albums≠Playlists in every theme)
- [ ] Inactive tabs use `TextMuted()` color
- [ ] Only the active section's results are shown (not all 4 stacked)
- [ ] Tab/Shift+Tab cycles tabs, resets cursor to 0
- [ ] Tracks tab: columns #, Track, Artist, Album, Duration
- [ ] Albums tab: columns #, Album, Artist, Year, Tracks
- [ ] Playlists tab: columns #, Playlist, Owner, Tracks
- [ ] Artists tab: columns #, Artist
- [ ] Column headers styled with active tab's `tabColorForSection()` color + bold
- [ ] Column data uses `ColumnIndex/Primary/Secondary/Tertiary` theme tokens
- [ ] Selected row uses `SelectedBg()/SelectedFg()` (overrides column colors)
- [ ] Tracks tab drops Album column when `contentWidth < 60`
- [ ] Track duration formatted as `m:ss` or `h:mm:ss`
- [ ] Column headers render below tab bar with underline separators
- [ ] Bottom help bar shows contextual keybindings (Ctrl+A only on Tracks tab)
- [ ] Section labels are title case ("Tracks" not "TRACKS")
- [ ] Border actions show `[Enter play, Esc close]`
- [ ] Old `renderSection` and `*AsRows` methods are removed
- [ ] Existing keybindings unchanged: Enter, Ctrl+A, Esc, ↑↓, Tab/Shift+Tab
- [ ] Theme switching (`t`) updates all tab colors and column colors correctly
- [ ] `make ci` passes

## Tasks

- [ ] **Add tabColorForSection** — maps each section to its `PaneBorder*` theme token. Falls back to `ActiveBorder()` for unknown sections.
      - test: `TestTabColorForSection_ReturnsCorrectTokens` — verify each section maps to the expected token

- [ ] **Add totalForSection** — returns total count for a given section from SearchResultData (0 if nil).
      - test: `TestTotalForSection_AllSections`, `TestTotalForSection_NilResults`

- [ ] **Add renderTabBar** — horizontal tab bar with counts. Active: ▪ + bold + tab color. Inactive: TextMuted.
      - test: `TestRenderTabBar_ActiveHighlighted` — active tab has ▪ marker
      - test: `TestRenderTabBar_ShowsCounts` — counts from TotalTracks etc.
      - test: `TestRenderTabBar_NilResults_ZeroCounts` — zero counts when results nil

- [ ] **Add renderColumnHeaders** — header labels + underline for each section. Headers use `tabColorForSection(activeSection)` + bold. Underline uses `TextMuted()`. Tracks tab drops Album column if `contentWidth < 60`.
      - test: `TestRenderColumnHeaders_Tracks` — 5 column headers
      - test: `TestRenderColumnHeaders_Artists` — 2 column headers
      - test: `TestRenderColumnHeaders_Albums` — 5 column headers
      - test: `TestRenderColumnHeaders_Playlists` — 4 column headers
      - test: `TestRenderColumnHeaders_Tracks_NarrowDropsAlbum` — 4 columns at contentWidth < 60

- [ ] **Add renderActiveSection** — numbered column-aligned rows for active section. Selected row: ▶ + `SelectedBg/Fg`. Unselected: `ColumnIndex/Primary/Secondary/Tertiary` per column. Tracks show Album + formatted duration. Albums show Year + TotalTracks. Playlists show TrackCount. Narrow Tracks tab hides Album column.
      - test: `TestRenderActiveSection_Tracks_ShowsAlbumAndDuration`
      - test: `TestRenderActiveSection_Albums_ShowsYearAndCount`
      - test: `TestRenderActiveSection_Playlists_ShowsTrackCount`
      - test: `TestRenderActiveSection_SelectedRow_UsesSelectedColors`
      - test: `TestRenderActiveSection_Tracks_NarrowNoAlbumColumn`

- [ ] **Add formatDurationMs** — format milliseconds as `m:ss` (< 1 hour) or `h:mm:ss` (>= 1 hour). Check timeutil first.
      - test: `TestFormatDurationMs_ShortTrack` — e.g. 222000ms → "3:42"
      - test: `TestFormatDurationMs_LongTrack` — e.g. 7290000ms → "2:01:30"
      - test: `TestFormatDurationMs_Zero` — 0ms → "0:00"

- [ ] **Add renderHelpBar** — separator + contextual keybindings in `TextMuted()`. `Ctrl+A queue` only on Tracks tab.
      - test: `TestRenderHelpBar_TracksTab_ShowsCtrlA`
      - test: `TestRenderHelpBar_OtherTab_NoCtrlA`

- [ ] **Rewrite renderResults and View** — assemble: tab bar → separator → column headers → active section rows → help bar. Delete old `renderSection()` and all `clamped*AsRows()` methods. Update `searchSectionLabels` to title case. Change border Actions to `[{Enter, play}, {Esc, close}]`.
      - test: `TestSearchOverlayView_ContainsTabBar` — output contains tab labels
      - test: `TestSearchOverlayView_ContainsColumnHeaders` — output contains header row
      - test: `TestSearchOverlayView_ContainsHelpBar` — output contains help text
      - test: `TestSearchOverlayView_BorderActions` — border shows "Esc close" not "Tab section"
