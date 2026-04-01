---
title: "Enrich Search Data Types & Widen Overlay"
feature: 18-search-redesign
status: open
---

## Background

The current search overlay discards rich metadata that the Spotify API already returns:
tracks lack Album and Duration, albums lack Year and TrackCount, playlists lack TrackCount,
and total result counts per section are dropped entirely. The overlay is also narrow
(max 50 chars, limit 5 results) — too small for the tabbed redesign in story 82.

This story extends the data layer and dimensions without changing the rendering. Story 82
builds the tabbed UI on top of these enriched types.

**API constraints (Feb 2026):**
- Search limit is now **max 10 per type** (reduced from 50) — bump from 5 to 10
- `Artist.followers` and `Artist.popularity` removed — Artists tab can only show name
- `Album.label` and `Album.popularity` removed — no impact

## Design

### Enrich UI-Side Data Types

In `internal/ui/panes/messages.go`, extend the existing item structs and result container:

**SearchTrackItem** — add:
- `Album string` — from `Track.Album.Name`
- `DurationMs int` — from `Track.DurationMs`

**SearchAlbumItem** — add:
- `ReleaseYear string` — first 4 chars of `SearchAlbum.ReleaseDate` (guard short dates)
- `TotalTracks int` — from `SearchAlbum.TotalTracks`

**SearchPlaylistItem** — add:
- `TrackCount int` — from `SearchPlaylist.TrackCount`

**SearchResultData** — add total counts from each API section's `.Total` field:
- `TotalTracks int`
- `TotalArtists int`
- `TotalAlbums int`
- `TotalPlaylists int`

### Update convertSearchResult

In `internal/app/commands.go`, `convertSearchResult()` currently discards the new fields.
Populate them:
- `item.Album = t.Album.Name`
- `item.DurationMs = t.DurationMs`
- `item.ReleaseYear = a.ReleaseDate[:4]` with `len(a.ReleaseDate) >= 4` guard
- `item.TotalTracks = a.TotalTracks`
- `item.TrackCount = p.TrackCount` (from `SearchPlaylist.TrackCount`)
- `data.TotalTracks = r.Tracks.Total` (and similarly for Artists, Albums, Playlists)

### Bump Search API Limit

In `internal/app/commands.go`, `buildSearchCmd()` currently passes `limit=5`. Change to `10`
to match the Feb 2026 API max and the new `maxResultsPerSection`.

### Widen Overlay Dimensions

In `internal/ui/panes/search.go`:

- `maxResultsPerSection`: `5` → `10`
- `overlayWidth()`: base `90` (was 50), cap `80%` (was 60%), min `40` (was 20)
- `overlayHeight()`: base `26` (was 20), cap `75%` (was 70%), min `12` (was 8)

The rendering code (`renderResults`, `renderSection`) continues to work unchanged — it
just has more data and more space. The current stacked-section rendering remains until
story 82 replaces it with the tabbed UI.

### Files Changed

| Action | File | Purpose |
|---|---|---|
| Modify | `internal/ui/panes/messages.go` | Add fields to item types + total counts to SearchResultData |
| Modify | `internal/app/commands.go` | Populate new fields in convertSearchResult, bump limit to 10 |
| Modify | `internal/ui/panes/search.go` | Widen dimensions, bump maxResultsPerSection |
| Modify | `internal/ui/panes/search_test.go` | Update sampleSearchResultData, add dimension tests |
| Modify | `internal/app/commands_test.go` | Test enriched fields in conversion |

## Acceptance Criteria

- [ ] `SearchTrackItem` has `Album` and `DurationMs` fields, populated from API response
- [ ] `SearchAlbumItem` has `ReleaseYear` and `TotalTracks` fields, populated correctly
- [ ] `SearchPlaylistItem` has `TrackCount` field, populated correctly
- [ ] `SearchResultData` has `TotalTracks/TotalArtists/TotalAlbums/TotalPlaylists` fields
- [ ] `convertSearchResult` populates all new fields (Album from nested struct, year from date substring with guard)
- [ ] `buildSearchCmd` passes `limit=10` to the search API
- [ ] `maxResultsPerSection` is 10
- [ ] Overlay width is `min(90, 80% terminal)` with min 40
- [ ] Overlay height is `max(26, 75% terminal)` with min 12
- [ ] Existing rendering still works (stacked sections, just wider with more data)
- [ ] `make ci` passes

## Tasks

- [ ] **Enrich search item types** — add `Album string`, `DurationMs int` to `SearchTrackItem`; add `ReleaseYear string`, `TotalTracks int` to `SearchAlbumItem`; add `TrackCount int` to `SearchPlaylistItem`; add 4 total count fields to `SearchResultData`. In `internal/ui/panes/messages.go`.
      - test: `TestSearchResultData_EnrichedFields` — construct enriched data, verify all fields accessible

- [ ] **Update convertSearchResult** — populate all new fields from domain types. Guard `ReleaseYear` extraction with `len(a.ReleaseDate) >= 4`. In `internal/app/commands.go`.
      - test: `TestConvertSearchResult_EnrichedFields` — verify Album, DurationMs, ReleaseYear, TotalTracks, TrackCount, and all Total counts are populated
      - test: `TestConvertSearchResult_ShortReleaseDate` — verify no panic when ReleaseDate is < 4 chars

- [ ] **Bump search API limit** — change `buildSearchCmd` from `limit=5` to `limit=10`. In `internal/app/commands.go`.
      - test: `TestBuildSearchCmd_Limit10` — verify the search client receives limit 10

- [ ] **Widen overlay dimensions** — update `overlayWidth` (base 90, cap 80%, min 40), `overlayHeight` (base 26, cap 75%, min 12), `maxResultsPerSection` to 10. In `internal/ui/panes/search.go`.
      - test: `TestOverlayWidth_Wider` — verify wider base and cap
      - test: `TestOverlayWidth_NarrowTerminal` — verify min 40
      - test: `TestOverlayHeight_Taller` — verify taller base and cap

- [ ] **Update sampleSearchResultData** — add enriched fields to the test helper so all existing tests continue to compile and pass. In `internal/ui/panes/search_test.go`.
      - no new test — this unblocks existing tests
