---
name: project_spotnik_feature83_complete
description: Story 83 (Fix Search Rendering Bugs): content width double-subtraction fix, column width helpers, help bar bottom anchoring, playlist TrackCount investigation
type: project
---

## Story 83 — Fix Search Overlay Rendering Bugs

**What was built:**

### Task 1: Content Width Double-Subtraction Fix
- `renderResults` received `innerWidth` (border already stripped in `View()`) but was subtracting 4 more chars (`- 4` for border+padding)
- Fixed to subtract only 2 (`- 2` for left+right padding only)
- Renamed the parameter from `overlayWidth` → `innerWidth` to prevent future confusion

### Task 2: Column Width Math with Gap Accounting
- Extracted `trackColumnWidths(contentWidth, narrow)`, `albumColumnWidths(contentWidth)`, `playlistColumnWidths(contentWidth)`, `artistColumnWidths(contentWidth)` helpers
- Each helper subtracts `(numCols-1)*2` gap chars AND fixed column widths before distributing flex space proportionally
- The last flex column uses remainder (`flex - nameW - artistW`) to prevent rounding overflow
- `renderColumnHeaders` and `renderActiveSection` now call the same helpers — no drift possible
- Added exported wrappers: `TrackColumnWidths`, `AlbumColumnWidths`, `PlaylistColumnWidths`, `ArtistColumnWidths`

### Task 3: Help Bar Anchored to Bottom
- Changed `renderResults` signature to `renderResults(innerWidth, availableHeight int)`
- `View()` passes `innerHeight - 2` as `availableHeight`
- Padding lines computed as: `rowBudget - resultLineCount` where `rowBudget = availableHeight - 4(chrome) - 2(helpbar)`
- Uses `strings.Count(activeSection, "\n")` to count actual result rows

### Task 4: Playlist TrackCount Investigation
- `domain.SearchPlaylist.UnmarshalJSON` already correctly extracts `tracks.total`
- Spotify search API does return `tracks: {"href": "...", "total": N}` in search responses
- Added `TestSearchPlaylist_UnmarshalJSON_TracksTotal` and `TestConvertSearchResult_PlaylistTrackCount` as regression tests documenting correct behavior

**Key files:**
- `internal/ui/panes/search.go` — column width helpers, fixed renderResults, help bar padding
- `internal/ui/panes/search_test.go` — 10 new tests including narrow-mode column widths
- `internal/app/commands_test.go` — 2 new playlist TrackCount regression tests

**Patterns established:**
- Column width sum constraint: `indexW + col1W + col2W + ... + (numCols-1)*2 gaps = contentWidth` — always verify with a test
- Parameter naming matters: `overlayWidth` vs `innerWidth` caused a real bug (the double-subtraction)
- chrome+helpbar line counting pattern: `chromeLines = tab(1)+sep(1)+header(1)+underline(1)=4`, `helpBarLines=2`

**Gotchas:**
- `renderColumnHeaders` returns `"headerLine\nunderLine"` (2 visible lines, 1 `\n`). When `renderResults` appends a `\n` after it, the total is 2 newlines = 2 consumed lines. The chromeLines=4 constant accounts for this correctly.
- Narrow track mode test was missed initially (review found it): `TestTrackColumnWidths_NarrowSumEqualsContentWidth` added to cover the 4-column path.
- `artistColumnWidths` has no rounding issues (pure subtraction) but still got an exported wrapper for API consistency.

**Testing notes:**
- 87.3% total coverage (up from 87.2%)
- 10 new tests in search_test.go + 2 in commands_test.go
- Wide + narrow column width sum tests for tracks; only wide for albums/playlists (which have no narrow mode)
