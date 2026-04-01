---
name: project_spotnik_feature82_complete
description: Story 82 (Tabbed Search Overlay UI): tab bar, column headers, column-aligned rows, help bar, formatDurationMs extension, exported test wrappers
type: project
---

## Story 82 — Tabbed Search Overlay UI

**What was built:**
- `tabColorForSection(sec)` — maps each section to its PaneBorder* theme token; falls back to ActiveBorder()
- `totalForSection(sec)` — returns TotalTracks/Artists/Albums/Playlists from SearchResultData (0 if nil)
- `renderTabBar(width)` — horizontal tab bar; active: ▪ + bold + tab color; inactive: TextMuted; 5-space separator
- `renderColumnHeaders(sec, contentWidth)` — header row + TextMuted underline; per-section widths; Tracks drops Album when contentWidth < 60
- `renderActiveSection(contentWidth)` — column-aligned rows using searchCol named type; selected uses SelectedBg/Fg; unselected uses ColumnIndex/Primary/Secondary/Tertiary
- `renderHelpBar(contentWidth)` — separator + contextual keybindings; Ctrl+A queue only on Tracks
- `renderResults` rewritten — tab bar → sep → headers → active rows → help bar
- `View()` border actions updated: [Enter play, Esc close]
- `searchSectionLabels` updated to title case ("Tracks" not "TRACKS")
- Removed `renderSection`, `clampedTrackItemsAsRows`, `clampedArtistItemsAsRows`, `clampedAlbumItemsAsRows`, `clampedPlaylistItemsAsRows`
- Extended `formatDurationMs` in nowplaying.go to support h:mm:ss for tracks >= 1 hour

**Key files:**
- `internal/ui/panes/search.go` — all new rendering methods + exported test wrappers + searchCol type
- `internal/ui/panes/nowplaying.go` — formatDurationMs extended for h:mm:ss
- `internal/ui/panes/search_test.go` — 20 new tests, 5 updated tests

**Patterns established:**
- Named struct type `searchCol` for column definitions — avoids anonymous struct slice literal that gofmt mangles when used in func parameters
- Exported test wrappers pattern: `TabColorForSection`, `TotalForSection`, `RenderTabBar`, `RenderColumnHeaders`, `RenderActiveSection`, `RenderHelpBar`, `FormatDurationMs` — all unexported methods get an exported wrapper for test access
- `type SearchSection = searchSection` type alias (not new type) + exported constants `SectionTracks/Artists/Albums/Playlists` — allows test packages to reference sections by name
- Column width computation: fixed widths (indexW=3, durationW=8, yearW=6) + proportional for name/artist/album

**Gotchas:**
- Anonymous struct slices in function closures get gofmt-collapsed to semicolons: `func(i int, isSelected bool, cols []struct { text string; style lipgloss.Style; width int })` → gofmt puts everything on one line. Fix: use a named type (`searchCol`) instead.
- `formatDurationMs` already existed in `nowplaying.go` — extending it is correct, not adding a new function. The old code only handled m:ss (missing hours); extending it was required.
- Custom `min()` function: In Go 1.22, `min` is a builtin. Adding a custom package-level `min` shadows it. Lint doesn't flag it but it's cleaner to remove the custom one and use the builtin.
- `renderResults` parameter is named `overlayWidth` but receives `innerWidth` (borders already removed) — this is pre-existing and naming is misleading, but changing it would be out of scope.
- Test updates needed for old tests: 3 tests checked for uppercase "TRACKS" labels → updated to title case "Tracks"; 1 test checked for "section" in border → updated to "close"; `TestSearchOverlay_NoAPIImportBoundary` assumed all sections visible at once → updated to navigate per tab.

**Testing notes:**
- 90.4% panes coverage (was 90.3%)
- Overall 87.2% (was 87.1%)
- 20 new tests + 5 updated existing tests
- `stripANSIForTest` helper already in search_test.go from story 81 — reused for all stripped-text assertions
