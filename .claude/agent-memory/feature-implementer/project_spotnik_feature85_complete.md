---
name: project_spotnik_feature85_complete
description: Story 85 (Search Table Rewrite): components.Table replacement, accumulated buffers, smart prefetch, mouse scroll forwarding, Elm purity gotcha
type: project
---

## Story 85 — Search Table Rewrite

**What was built:**
- Replaced manual string-construction table rendering in SearchOverlay with 4 `components.Table` instances (one per section tab)
- Added accumulated page buffers (`bufTracks/bufArtists/bufAlbums/bufPlaylists`) that append across prefetched pages
- Smart 50%-midpoint prefetch via `checkPrefetch()` guarded by `IsSearchOffsetFetched()` store method
- Mouse wheel scroll forwarding in `routing.go` — converts WheelUp/Down to KeyUp/KeyDown sent to searchPane
- KeyHint()+Bold(true) help bar key highlighting matching the main status bar pattern
- Optional `HeaderColor` override in `components.TableConfig` for per-tab column header colors
- Narrow-mode track table rebuild (< 60 content chars) drops the Album column

**Key files:**
- `internal/ui/panes/search.go` — Full rewrite; `buildAllTables()`, `refreshXxxRows()`, `checkPrefetch()`, `renderResults()` (now pure)
- `internal/app/routing.go` — `handleMouseMsg` now forwards wheel events to search overlay when open
- `internal/state/store.go` — Added `searchTotals[4]int`, `searchFetched[4]map[int]bool`, `searchBufQuery string`
- `internal/ui/components/table.go` — Added `HeaderColor lipgloss.Color` to `TableConfig`

**Import boundary constraint:**
Store cannot hold panes item types (SearchTrackItem etc.) because `state/` cannot import `ui/panes/`. Solution: store only holds primitive metadata (totals, fetched-offset map, query). The overlay accumulates typed items in local `buf*` slice fields.

**Elm purity gotcha (critical, caught in review):**
The initial `renderResults()` implementation called `tables[i].SetSize()` and `rebuildTrackTable()` inside the View() call chain — a state mutation in View(). This violates CLAUDE.md: "View() must be pure — no external calls, no heavy computation, just read state → string." Fixed by removing those mutations from `renderResults` with a NOTE comment. The `SetSize()` method already handles all of this on WindowSizeMsg. Always check whether render methods mutate state.

**Test patterns:**
- Old tests checked pixel-counting column widths → replaced with `Tables()[sec].Columns()` introspection
- Old tests expected `▶` symbol for selection → replaced with ANSI escape code check (bubble-table uses background-color highlight, not ▶ prefix)
- Help bar "bottom quarter" position test → replaced with "appears after content" check (bubble-table renders compactly without bottom-padding)
- Prefetch tests: create 10-item buffer with total=100, move cursor to midpoint (pos 5), next down triggers prefetch
- `newTestSearchOverlayWithResultsCustom` (legacy helper using WithSectionOffsets) → removed (deleted function)

**Patterns established:**
- `HandleMouseMsg` search-open branch: forward wheel as KeyUp/KeyDown, return nil for non-wheel; then continue with device-overlay check and pane hit-test
- `checkPrefetch` formula: `lastPageStart = bufLen - min(bufLen, maxResultsPerSection)`, `midpoint = lastPageStart + maxResultsPerSection/2`
- Buffer accumulation: `!m.IsPaged` → clearBuffers() + append all sections; `m.IsPaged` → append single section by switch on m.Section
- `totals[sec]` in overlay duplicates `store.SearchSectionTotal()` — both are kept in sync, overlay uses local field for performance in checkPrefetch

**Gotchas:**
- `renderResults` was calling `rebuildTrackTable` and `SetSize` inside View() path — Elm purity violation caught in PR review, fixed in final commit `aa15e4c`
- `▶` symbol is the PLAYING indicator (set via PlayingIndex), not the SELECTION indicator; selection uses SelectedBg/SelectedFg background colors via WithRowStyleFunc
- Pagination tests that used `WithSectionOffsets`/`SectionOffsets`/`CursorPos` all needed deletion — the new model accumulates buffers, no explicit page offsets
- Help bar "bottom quarter" tests fail because bubble-table renders compactly and lipgloss Height() pads BELOW the content, not above the help bar
