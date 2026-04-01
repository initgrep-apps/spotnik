---
name: project_spotnik_feature81_complete
description: Story 81 (Enrich Search Data Types & Widen Overlay): enriched item types, convertSearchResult population, limit bump, overlay dimensions
type: project
---

## Story 81 — Enrich Search Data Types & Widen Overlay

**What was built:**
- Added `Album string` and `DurationMs int` to `SearchTrackItem`
- Added `ReleaseYear string` and `TotalTracks int` to `SearchAlbumItem`
- Added `TrackCount int` to `SearchPlaylistItem`
- Added `TotalTracks/TotalArtists/TotalAlbums/TotalPlaylists int` to `SearchResultData`
- Updated `convertSearchResult` to populate all new fields with `len >= 4` guard for ReleaseYear
- Bumped `buildSearchCmd` limit from 5 to 10
- Bumped `maxResultsPerSection` from 5 to 10 in search.go
- Updated `overlayWidth` to `min(90, 80% terminal)` with min 40
- Updated `overlayHeight` to `max(26, 75% terminal)` with min 12

**Key files:**
- `internal/ui/panes/messages.go` — enriched search item types
- `internal/app/commands.go` — convertSearchResult and buildSearchCmd
- `internal/app/commands_test.go` — NEW: tests via HTTP mock server
- `internal/ui/panes/search.go` — overlay dimensions and maxResultsPerSection
- `internal/ui/panes/search_test.go` — enriched type tests, dimension tests with ANSI stripping

**Patterns established:**
- Dimension tests in search_test.go use `stripANSIForTest()` helper to strip ANSI escape sequences before counting runes in the first border line
- Height test counts `len(strings.Split(view, "\n"))` directly
- `commands_test.go` tests `convertSearchResult` indirectly via the HTTP mock → buildSearchCmd → SearchResultsMsg pipeline (function is unexported, so no direct call)

**Gotchas:**
- The clamping helpers comment said "max 5 per section" — needed updating to "max 10" after bumping the constant. Caught in PR self-review.
- ANSI stripping: lipgloss adds escape codes to every styled character. The first border line rune count (raw) was 344 but stripped was 50 for a width-50 overlay. Always strip ANSI before measuring visual width in tests.
- `gofmt` formatted the aligned map literals in commands_test.go (added trailing whitespace alignment), causing lint failure. Always run `gofmt -w` before CI.

**Testing notes:**
- 87.1% total coverage after implementation
- ANSI-stripping approach for dimension tests is reliable: stripped first line rune count == overlay width
