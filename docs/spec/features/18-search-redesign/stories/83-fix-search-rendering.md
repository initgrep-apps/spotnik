---
title: "Fix Search Overlay Rendering Bugs"
feature: 18-search-redesign
status: done
---

## Background

After the tabbed search overlay was implemented (stories 81–82), manual testing revealed
three rendering bugs visible across all four tabs:

1. **Column overflow / text wrapping** — column headers wrap to a second line ("Duration"
   drops below), and the selected row spills to two lines. Root cause: double width
   subtraction plus column width math that ignores inter-column gap space.

2. **Help bar not anchored to bottom** — the contextual keybindings line ("Tab next section
   ↑↓ navigate...") floats immediately after the last result row instead of being pinned
   to the bottom of the overlay. When there are fewer than 10 results, there's dead space
   below the help bar instead of between the results and the help bar.

3. **Playlist TrackCount always 0** — every playlist shows "0" in the Tracks column.
   The `SearchPlaylist.UnmarshalJSON` in `domain/search.go` correctly extracts
   `tracks.total`, so this may be a Spotify API quirk where search results don't always
   include the nested `tracks.total` field. Needs investigation and a fallback.

## Design

### Task 1: Fix Content Width Double-Subtraction

**File:** `internal/ui/panes/search.go`

`View()` at line 371 computes `innerWidth = totalWidth - 2` (strips the border), then
passes it to `renderResults(innerWidth)`. But `renderResults()` at line 457 subtracts
4 more:

```go
contentWidth := overlayWidth - 4 // subtract border + padding
```

The parameter `overlayWidth` is actually `innerWidth` (already border-stripped). Subtracting
4 more loses 6 total chars. Fix: reduce the subtraction to account for only left/right
padding (1 char each side):

```go
contentWidth := overlayWidth - 2 // left + right padding within the border
```

- test: `TestContentWidth_NoDoubleSubtraction` — verify contentWidth equals innerWidth - 2

### Task 2: Fix Column Width Math to Account for Gaps

**File:** `internal/ui/panes/search.go`

In both `renderColumnHeaders` and `renderActiveSection`, the wide-mode column widths are
computed as percentages of `contentWidth` without subtracting the inter-column gap space.
Each gap is 2 chars (`"  "` separator in `strings.Join`). With N columns, there are
`(N-1) * 2` chars of gaps plus the `indexW` (3 chars) as fixed overhead.

**Current (Tracks wide, 5 columns):**
```go
nameW := contentWidth * 35 / 100
artistW := contentWidth * 25 / 100
albumW := contentWidth * 25 / 100
durationW := 8
```
Sum with gaps: `3 + nameW + artistW + albumW + 8 + (4 gaps × 2) = exceeds contentWidth`

**Fix pattern — subtract fixed + gaps, then distribute remainder:**
```go
gaps := (numCols - 1) * 2  // 4 gaps × 2 = 8 for 5 columns
fixed := indexW + durationW + gaps  // 3 + 8 + 8 = 19
flex := contentWidth - fixed         // remaining space for name/artist/album
nameW := flex * 40 / 100
artistW := flex * 30 / 100
albumW := flex - nameW - artistW     // remainder prevents overflow
```

Apply the same pattern to **all sections** in both `renderColumnHeaders` and
`renderActiveSection`:

| Section | Fixed cols | Gaps | Flex cols |
|---------|-----------|------|-----------|
| Tracks wide (5 cols) | indexW(3) + durationW(8) | 4×2=8 | name, artist, album |
| Tracks narrow (4 cols) | indexW(3) + durationW(8) | 3×2=6 | name, artist |
| Artists (2 cols) | indexW(3) | 1×2=2 | artist |
| Albums (5 cols) | indexW(3) + yearW(6) + tracksW(8) | 4×2=8 | name, artist |
| Playlists (4 cols) | indexW(3) + tracksW(8) | 3×2=6 | name, owner |

**Critical**: `renderColumnHeaders` and `renderActiveSection` must use identical width
calculations. Extract a shared helper or define width constants to prevent drift:

```go
func trackColumnWidths(contentWidth int, narrow bool) (nameW, artistW, albumW, durationW int)
func albumColumnWidths(contentWidth int) (nameW, artistW, yearW, tracksW int)
func playlistColumnWidths(contentWidth int) (nameW, ownerW, tracksW int)
func artistColumnWidths(contentWidth int) (artistW int)
```

These helpers return the computed widths for each section, called by both
`renderColumnHeaders` and `renderActiveSection`.

- test: `TestTrackColumnWidths_SumEqualsContentWidth` — verify all widths + gaps = contentWidth
- test: `TestAlbumColumnWidths_SumEqualsContentWidth`
- test: `TestPlaylistColumnWidths_SumEqualsContentWidth`
- test: `TestRenderColumnHeaders_FitsOnOneLine` — verify header line has no embedded newlines
- test: `TestRenderActiveSection_RowFitsOnOneLine` — verify no newlines within a single row

### Task 3: Anchor Help Bar to Bottom of Overlay

**File:** `internal/ui/panes/search.go`

Currently `renderResults()` concatenates rows then the help bar with no vertical padding.
The lipgloss `Height(innerHeight)` in `View()` pads at the end, but the help bar is
embedded inside the content — so padding goes below the help bar, not above it.

**Fix approach**: Pass available height into `renderResults` and compute padding.

1. Change signature: `renderResults(width, availableHeight int) string`
2. In `View()`, pass `innerHeight - 2` (subtract input line + dot separator)
3. In `renderResults()`, count chrome lines: tab bar(1) + separator(1) + header(1) + underline(1) = 4 lines
4. Count help bar lines: separator(1) + text(1) = 2 lines
5. Available for rows + padding: `availableHeight - 4 - 2`
6. After rendering active section rows, insert padding newlines to push help bar to bottom:

```go
resultLineCount := strings.Count(activeSection, "\n")
rowBudget := availableHeight - 4 - 2  // chrome - help bar
padLines := rowBudget - resultLineCount
if padLines > 0 {
    sb.WriteString(strings.Repeat("\n", padLines))
}
sb.WriteString(o.renderHelpBar(contentWidth))
```

- test: `TestSearchOverlay_HelpBarAtBottom` — verify help bar is at the expected line position (near bottom of overlay)
- test: `TestSearchOverlay_HelpBarAtBottom_FewResults` — with only 3 results, help bar still at bottom

### Task 4: Investigate Playlist TrackCount = 0

**Files:** `internal/domain/search.go`, `internal/app/commands.go`, `testdata/fixtures/`

The `SearchPlaylist.UnmarshalJSON` correctly extracts `tracks.total`. But the Spotify
search API may return playlist objects where the `tracks` field is a simplified form
that doesn't include `total`, or `total` is 0 for search results.

**Investigation steps:**
1. Add a test with a real Spotify search JSON response captured via curl to verify
   the actual response shape for playlists
2. If `tracks.total` is genuinely 0 in search results, consider: is this a display
   issue or should we hide the column when all values are 0?
3. If the unmarshal is broken, fix it and add a regression test

- test: `TestSearchPlaylist_UnmarshalJSON_TracksTotal` — verify extraction from realistic fixture
- test: `TestConvertSearchResult_PlaylistTrackCount` — end-to-end from API response to UI type

### Files Changed

| Action | File | Purpose |
|--------|------|---------|
| Modify | `internal/ui/panes/search.go` | Fix contentWidth, column math, help bar anchoring |
| Modify | `internal/ui/panes/search_test.go` | ~10 new tests for fixed rendering |
| Modify | `internal/app/commands_test.go` | Playlist TrackCount test |
| Possibly | `internal/domain/search.go` | If unmarshal fix needed |

## Acceptance Criteria

- [ ] Column headers fit on one line for all 4 tabs (no wrapping)
- [ ] Selected row fits on one line (no text spill to second line)
- [ ] Unselected rows fit on one line
- [ ] Column widths + gaps sum to exactly contentWidth for all sections
- [ ] `renderColumnHeaders` and `renderActiveSection` use identical width calculations
- [ ] Help bar is anchored to the bottom of the overlay
- [ ] With fewer than 10 results, empty space appears between rows and help bar (not below help bar)
- [ ] Playlist TrackCount shows actual values (investigated; fix if possible)
- [ ] All existing tests pass
- [ ] `make ci` passes
