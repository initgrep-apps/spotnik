---
title: "Stats component + integration tests"
feature: 21-test-infrastructure
status: done
---

## Background

The Stats feature (07) provides three panes: TopTracks, TopArtists, RecentlyPlayed. Each
supports filter (`f`), Enter-to-play (tracks: PlayTrackListMsg, artists: PlayContextMsg),
and time-range cycling (`g`) on TopTracks/TopArtists. RecentlyPlayed renders relative
timestamps. Current unit tests cover Update() handlers but not the rendered table layout,
time-range label updates, or empty states.

## Design

### Golden tests: `internal/ui/panes/toptracks_golden_test.go`

Snapshots:
- `TestTopTracksPane_View_Tracks` — 10 tracks loaded, 80×24, short_term range
- `TestTopTracksPane_View_EmptyState` — no data, empty state shown
- `TestTopTracksPane_View_MediumTerm` — time range label "past 6 months"
- `TestTopTracksPane_View_Narrow` — 40×24, column hiding
- `TestTopTracksPane_View_FilterActive` — 'f' pressed, tracks filtered by name or artist
- `TestTopTracksPane_View_FilterActive_NoMatches` — 'f' pressed, query matches nothing, "no results" shown

### Golden tests: `internal/ui/panes/topartists_golden_test.go`

Snapshots:
- `TestTopArtistsPane_View_Artists` — 10 artists loaded, 80×24
- `TestTopArtistsPane_View_EmptyState` — no data
- `TestTopArtistsPane_View_LongTerm` — "all time" label
- `TestTopArtistsPane_View_Narrow` — 40×24
- `TestTopArtistsPane_View_FilterActive` — 'f' pressed, artists filtered by name
- `TestTopArtistsPane_View_FilterActive_NoMatches` — 'f' pressed, query matches nothing

### Golden tests: `internal/ui/panes/recentlyplayed_golden_test.go`

Snapshots:
- `TestRecentlyPlayedPane_View_Tracks` — 5 recently played with timestamps, 80×24
- `TestRecentlyPlayedPane_View_EmptyState` — no data
- `TestRecentlyPlayedPane_View_Narrow` — 40×24
- `TestRecentlyPlayedPane_View_FilterActive` — 'f' pressed, filtered by track or artist name
- `TestRecentlyPlayedPane_View_FilterActive_NoMatches` — 'f' pressed, query matches nothing

### Integration test: `internal/ui/panes/stats_flow_test.go`

```go
func TestTopTracksTimeRangeCycle_UpdatesView(t *testing.T) {
    // Setup TopTracksPane with short_term data
    // 1. Send 'g' → assert medium_term label in View()
    // 2. Send 'g' → assert long_term label in View()
    // 3. Send 'g' → assert short_term label again (wraps)
}

func TestStatsEnterPlaysContextOrTrackList(t *testing.T) {
    // TopTracks Enter → verify PlayTrackListMsg cmd produced
    // TopArtists Enter → verify PlayContextMsg cmd produced
}
```

## Files

### Create

- `internal/ui/panes/toptracks_golden_test.go`
- `internal/ui/panes/topartists_golden_test.go`
- `internal/ui/panes/recentlyplayed_golden_test.go`
- `internal/ui/panes/stats_flow_test.go`
- `internal/ui/panes/testdata/TestTopTracksPane_View_*.golden` (4 files)
- `internal/ui/panes/testdata/TestTopArtistsPane_View_*.golden` (4 files)
- `internal/ui/panes/testdata/TestRecentlyPlayedPane_View_*.golden` (3 files)

## Acceptance Criteria

- [ ] TopTracksPane: 6 golden snapshots (tracks, empty, medium_term, narrow, filter active, filter no matches)
- [ ] TopArtistsPane: 6 golden snapshots (artists, empty, long_term, narrow, filter active, filter no matches)
- [ ] RecentlyPlayedPane: 5 golden snapshots (tracks with timestamps, empty, narrow, filter active, filter no matches)
- [ ] Integration: time-range cycle updates View() label correctly
- [ ] Integration: Enter produces correct PlayTrackListMsg / PlayContextMsg
- [ ] `make ci` passes

## Tasks

- [ ] Create TopTracksPane golden tests (6 snapshots)
      - test: `TestTopTracksPane_View_Tracks`, `TestTopTracksPane_View_EmptyState`, `TestTopTracksPane_View_MediumTerm`, `TestTopTracksPane_View_Narrow`, `TestTopTracksPane_View_FilterActive`, `TestTopTracksPane_View_FilterActive_NoMatches`
- [ ] Create TopArtistsPane golden tests (6 snapshots)
      - test: `TestTopArtistsPane_View_Artists`, `TestTopArtistsPane_View_EmptyState`, `TestTopArtistsPane_View_LongTerm`, `TestTopArtistsPane_View_Narrow`, `TestTopArtistsPane_View_FilterActive`, `TestTopArtistsPane_View_FilterActive_NoMatches`
- [ ] Create RecentlyPlayedPane golden tests (5 snapshots)
      - test: `TestRecentlyPlayedPane_View_Tracks`, `TestRecentlyPlayedPane_View_EmptyState`, `TestRecentlyPlayedPane_View_Narrow`, `TestRecentlyPlayedPane_View_FilterActive`, `TestRecentlyPlayedPane_View_FilterActive_NoMatches`
- [ ] Create stats integration flow tests
      - test: `TestTopTracksTimeRangeCycle_UpdatesView`, `TestStatsEnterPlaysContextOrTrackList`
- [ ] Generate golden files and verify all tests pass
