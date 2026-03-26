---
name: project_spotnik_feature48_complete
description: Feature 48 (Stats Split): RecentlyPlayedPane, TopTracksPane, TopArtistsPane — patterns, gotchas, test tips
type: project
---

## Feature 48 — Stats Split

**What was built:**
- `internal/ui/components/timeutil.go` — `FormatRelativeTime(t time.Time) string` shared utility
- `internal/ui/panes/recentlyplayed_pane.go` — RecentlyPlayedPane (layout.Pane, toggle key 6)
  - Columns: # 5% | Track 45% | Artist 35% | Played 15% (flex 1:9:7:3)
  - `formatPlayedAtFromHistory()` parses RFC3339 string → FormatRelativeTime
  - Enter emits PlayTrackMsg; filter by track name OR artist
- `internal/ui/panes/toptracks_pane.go` — TopTracksPane (layout.Pane, toggle key 7)
  - Columns: # 5% | Track 45% | Artist 35% | Pop 15% (flex 1:9:7:3)
  - Popularity column shows "—" (domain.Track has no Popularity field)
  - `t` key cycles short→medium→long→short; cache-hit skips FetchStatsMsg
  - `StatsLoadedMsg` only refreshes if `m.TimeRange == p.timeRange`
- `internal/ui/panes/topartists_pane.go` — TopArtistsPane (layout.Pane, toggle key 8)
  - Columns: # 5% | Name 70% | Genre 25% (flex 1:14:5)
  - Genre shows artist.Genres[0] or "—" for empty Genres slice
  - Filter matches on artist name OR first genre
  - No Enter action (artists aren't directly playable)
- `stats.go` updated: `FormatRelativeTime` now delegates to `components.FormatRelativeTime`

**Key files:**
- `internal/ui/components/timeutil.go` — shared relative time formatter
- `internal/ui/panes/recentlyplayed_pane.go` — follows likedsongs_pane.go pattern exactly
- `internal/ui/panes/toptracks_pane.go` — has `cycleTimeRange()` private method
- `internal/ui/panes/topartists_pane.go` — same pattern as TopTracksPane but different data/columns

**Patterns established:**
- `RecentlyPlayedPane` handles `RecentlyPlayedLoadedMsg` regardless of focus (broadcasts)
- `TopTracksPane` and `TopArtistsPane` handle `StatsLoadedMsg` but ONLY refresh when `m.TimeRange == pane.timeRange` — each pane manages its own time range independently
- Cache-hit check for tracks: `store.TopTracks(nextRange) != nil` — nil means absent key (not fetched); empty slice means fetched but no data
- Time ranges duplicated per-pane (not shared package var) because `stats.go` has `timeRanges` and `topTracks`/`topArtists` have their own prefixed copies

**Gotchas:**
- `domain.Track` has no `Popularity` field — the spec column "Pop" shows "—" as placeholder for all tracks
- `FormatRelativeTime` was in `stats.go` under the panes package. It was extracted to `components/timeutil.go`. stats.go still has a wrapper for backward compatibility until F49/53 removes it.
- Test files use `package panes` (same-package), not `package panes_test`
- `//nolint` bare comments are inconsistent with codebase — must be `//nolint:errcheck`

**Testing notes:**
- Final coverage: 85.9% total, 88.7% for panes package
- 58 new tests: 15 RecentlyPlayed + 17 TopTracks + 19 TopArtists + 7 integration
- Test helpers: `populateStoreTopTracks()`, `populateStoreTopArtists()` in same-package tests
- Integration tests in `stats_split_integration_test.go` verify cross-pane independence
- Use `theme.Load("black")` (not `&theme.BlackTheme{}`) for test theme instantiation
