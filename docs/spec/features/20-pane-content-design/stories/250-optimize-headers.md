---
title: "Optimize column headers for space efficiency"
feature: 20-pane-content-design
status: open
---

## Background

Several column headers are wider than their typical cell content, wasting horizontal space:
- `Duration` (8 chars) contains "3:45" (4 chars) — 4 chars wasted
- `Popularity` (10 chars) contains "100" (3 chars) — 7 chars wasted
- `Publisher` (9 chars) contains names like "Spotify Studios" (15 chars) — borderline, but abbreviation saves space in narrow panes

Design rule §1.4: headers must not exceed typical content width. Use abbreviations where natural and unambiguous.

**Depends on:** Story 247 (# column removal) — header changes apply to the new column sets.

## Design

### Header mappings

| Pane | Old Header | New Header | Applies to |
|------|-----------|-----------|------------|
| Queue | `Duration` | `Dur` | `duration` column |
| LikedSongs | `Duration` | `Dur` | `duration` column |
| TopTracks | `Duration` | `Dur` | `dur` column (already uses short key) |
| Playlists tracks | `Duration` | `Dur` | `duration` column |
| Albums tracks | `Duration` | `Dur` | `duration` column |
| SavedEpisodes | `Duration` | `Dur` | `duration` column |
| FollowedShows eps | `Duration` | `Dur` | `duration` column |
| TopArtists | `Popularity` | `Pop` | `pop` column |
| FollowedShows shows | `Publisher` | `Pub` | `publisher` column |

### Implementation

Each change is a single string replacement in the `Header` field of the relevant `ColumnDef`:

```go
// Before:
{Key: "duration", Header: "Duration", FlexFactor: 2, ...}
// After:
{Key: "duration", Header: "Dur", FlexFactor: 2, ...}
```

Update both the constructor and `SetTheme` method in each pane.

### Panes NOT changed

- `RecentlyPlayed` — header is `Played`, already short
- `NetworkLog` — headers are already optimized (`Time`, `Method`, `Endpoint`, `Status`, `Latency`, `Priority`, `Decision`)
- `GatewayLive` — no headers (`ShowHeader: false`)
- `Queue` `type` column — empty header, no change
- All glyph/icon columns — empty headers, no change

## Files

### Modify

- `internal/ui/panes/queue.go` — `Duration` → `Dur` (constructor + SetTheme)
- `internal/ui/panes/likedsongs_pane.go` — `Duration` → `Dur` (constructor + SetTheme)
- `internal/ui/panes/toptracks_pane.go` — `Duration` → `Dur` (constructor + SetTheme)
- `internal/ui/panes/topartists_pane.go` — `Popularity` → `Pop` (constructor + SetTheme)
- `internal/ui/panes/playlists_pane.go` — `Duration` → `Dur` in track sub-view (constructor + SetTheme)
- `internal/ui/panes/albums_pane.go` — `Duration` → `Dur` in track sub-view (constructor + SetTheme)
- `internal/ui/panes/savedepisodes.go` — `Duration` → `Dur` (constructor + SetTheme)
- `internal/ui/panes/followedshows.go` — `Duration` → `Dur` in episode view, `Publisher` → `Pub` in show list (constructor + SetTheme)

## Acceptance Criteria

- [ ] No column header equals `"Duration"` in any pane (all replaced with `"Dur"`)
- [ ] No column header equals `"Popularity"` in any pane (replaced with `"Pop"`)
- [ ] No column header equals `"Publisher"` in any pane (replaced with `"Pub"`)
- [ ] All `SetTheme` methods have updated headers matching constructor
- [ ] `go build ./...` compiles
- [ ] `go test ./internal/ui/components/ -v -run "TestTable_ViewRendersHeader"` passes (checks for "Track"/"Artist" — these didn't change)
- [ ] `make test` passes

## Tasks

- [ ] **Task 1: Duration → Dur across all panes**
  Replace `Header: "Duration"` with `Header: "Dur"` in 7 panes (constructor + SetTheme):
  queue, likedsongs, toptracks, playlists (tracks), albums (tracks), savedepisodes, followedshows (episodes).
  - test: `go build ./...` — no errors

- [ ] **Task 2: Popularity → Pop in TopArtists**
  Replace `Header: "Popularity"` with `Header: "Pop"` in both constructor and SetTheme.
  - test: `go test ./internal/ui/panes/ -v -run "TestTopArtists"` — all pass

- [ ] **Task 3: Publisher → Pub in FollowedShows**
  Replace `Header: "Publisher"` with `Header: "Pub"` in show list columns (constructor + SetTheme).
  - test: `go test ./internal/ui/panes/ -v -run "TestFollowedShows"` — all pass

- [ ] **Task 4: Run full test suite**
  - test: `make test` — all pass
