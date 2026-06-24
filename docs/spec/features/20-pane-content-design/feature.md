---
title: "Pane Content Design Language"
status: done
---

## Description

Define and implement a consistent design language for all pane content. Fixes observed layout issues: mispositioned icon columns, inconsistent empty states, blank rows, column wrapping, and unresponsive column layouts. Result: sharp, space-efficient, production-quality terminal UI.

**Note:** Story 247 (remove `#` column) was reverted by story 253 — the `#` column is restored in all panes with `Priority: 1` for always-on visibility.

**Design record:** `docs/superpowers/specs/2026-06-18-pane-content-design-language.md`

**Implementation plan:** `docs/superpowers/plans/2026-06-18-pane-content-design-language.md`

### Changes at a glance

- Remove PlayingIndex dead code from Table wrapper and all panes
- Fix page size miscalculation (`height - 6` → `height - 4`)
- Move icon/glyph columns to position 1 in all panes
- Add `Priority` field to `ColumnDef` for responsive column hiding at width thresholds
- Shorten verbose column headers (`Duration` → `Dur`, `Popularity` → `Pop`, `Publisher` → `Pub`)
- Add consistent `EmptyState` to panes that lack it
- Update `docs/system/design.md` and `docs/system/tui.md`

### Design rules

1. **Column ordering:** `[Icon/Glyph] → [Primary Identifier] → [Secondary Info] → [Tertiary/Metadata]`
2. **Column priority:** Priority 1 always visible, Priority 2 ≥40 cols, Priority 3 ≥60 cols
3. **No padding around table** — content flush with pane border
4. **No blank rows** — fix page size formula, use EmptyState for zero data
5. **Headers shorter than typical content** — abbreviations where unambiguous

## Acceptance Criteria

- [ ] No `PlayingIndex` field in `TableConfig`, `Table` struct, or any pane code
- [ ] No `playingSymbol()` function or uikit import in `table.go`
- [ ] Page size unified to `height - 4` (header) / `height - 2` (no header) in both `SetSize` and `rebuild`
- [ ] `#` column present in all table panes with `Priority: 1` and `Color: ColumnIndex()`
- [ ] All panes show at least 2 columns at dashboard preset (~30 cols width)
- [ ] Primary identifier columns get dominant flex factors (≥50% width share)
- [ ] Icon columns at position 1 in SavedEpisodes, FollowedShows (shows + episodes); trailing empty icon column removed from Queue
- [ ] `Saved` column removed from SavedEpisodes
- [ ] `ColumnDef` has `Priority int` field; `filterColumnsByPriority()` filters at render time
- [ ] `SetSize` triggers column rebuild on width threshold crossings (40, 60 cols)
- [ ] Column headers use short forms: `Dur`, `Pop`, `Pub`
- [ ] EmptyState renders in LikedSongs, TopTracks, TopArtists, Playlists (list view), Albums (list view) when data is empty and filter inactive
- [ ] `make ci` passes after every story
- [ ] `docs/system/design.md` and `docs/system/tui.md` updated with new rules
- [ ] `#` column restored in all 9 table panes (constructors + SetTheme + row data)
- [ ] Primary columns given dominant flex factors in LikedSongs, RecentlyPlayed, TopTracks, Albums tracks

## Stories

| # | Story | File | Depends on |
|---|-------|------|------------|
| 245 | Remove PlayingIndex dead code | `stories/245-remove-playingindex.md` | — |
| 246 | Fix page size calculation + threshold crossing | `stories/246-fix-page-size.md` | — |
| 247 | Remove # column from all panes | `stories/247-remove-hash-column.md` | 245 |
| 248 | Fix icon column positions | `stories/248-fix-icon-positions.md` | 247 |
| 249 | Add Priority system and responsive column hiding | `stories/249-priority-responsive-columns.md` | 247, 248 |
| 250 | Optimize column headers | `stories/250-optimize-headers.md` | 247 |
| 251 | Add consistent EmptyState to all panes | `stories/251-consistent-empty-states.md` | 247 |
| 252 | Update design documentation | `stories/252-update-design-docs.md` | 245–251 |
| 253 | Revert # column removal — restore index | `stories/253-revert-hash-column.md` | 247, 249 |
| 254 | Tune responsive flex factors | `stories/254-responsive-width-tuning.md` | 253 |
| 255 | Fix pagination footer positioning | `stories/255-fix-pagination-footer.md` | 246 |
