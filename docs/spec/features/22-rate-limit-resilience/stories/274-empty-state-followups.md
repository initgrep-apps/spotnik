---
title: "Empty State Follow-ups from PR #406 Review"
feature: 22-rate-limit-resilience
status: done
---

## Background

PR #406 (story 272) shipped context-aware empty states for 8 table panes. Review found 6 follow-up issues — all minor, all with clear fixes.

## Issues

### 1. SavedEpisodes missing Hint for EmptyStatusNone

`internal/ui/panes/savedepisodes.go` — All 7 other panes set a contextual Hint when `EmptyStatusNone`. SavedEpisodes does not.

**Fix:** Add hint block after `PaneEmptyStatus` call:
```go
if es.Status == uikit.EmptyStatusNone {
    es.Hint = "Save episodes in Spotify or search with /"
}
```

### 2. EmptyStatusNone untested at pane level

5 panes (Albums, Playlists, LikedSongs, TopTracks, TopArtists) have tests for NeverFetched, Fetching, Error, RateLimited — but none for EmptyStatusNone (genuinely empty: API returned 200, zero items).

**Fix:** Add per-pane test: set data to empty slice, stamp `FetchedAt`, clear errors, verify "No X" + hint renders.

### 3. FollowedShows/SavedEpisodes/RecentlyPlayed missing pane-level status tests

These 3 panes have updated `View()` using `PaneEmptyStatus` but only got golden file updates — not the 4 new status tests (NeverFetched, Fetching, Error, RateLimited) that the other 5 panes got.

**Fix:** Add the same 4 status tests to these 3 panes, matching the pattern from the other 5 panes.

### 4. No golden test for non-never-fetched states

All 8 golden tests use `state.New()` (fresh store → EmptyStatusNeverFetched). No golden snapshot captures visual output of error, fetching, or rate-limited states.

**Fix:** Add at least one golden test per status variant (e.g., `TestAlbumsPane_View_EmptyState_Fetching` golden test). One pane is sufficient to cover the visual rendering — the uikit-level tests already cover the logic.

### 5. EmptyState Text dual role is fragile

`internal/uikit/empty_state.go` `Render()` uses `strings.TrimPrefix(e.Text, "No ")` to derive category from Text. Text serves dual role: display text when Status=None, category-source when Status≠None. Implicit contract, fragile.

**Fix:** Add a `Category` field to `EmptyState`:
```go
type EmptyState struct {
    Category string // e.g. "followed shows" — used by status-driven rendering
    Text     string // fallback when Status is None
    Hint     string
    Status   EmptyStatus
    Width    int
    Height   int
    Theme    theme.Theme
}
```
Update `Render()` to use `e.Category` instead of `strings.TrimPrefix(e.Text, "No ")`. Update `PaneEmptyStatus()` to set `Category` directly. Update all 8 pane call sites to pass category.

### 6. PaneEmptyStatus has 6 positional parameters

`PaneEmptyStatus(category string, isFetching bool, fetchErr error, neverFetched, isThrottled bool, retryAfterSecs int)` — 5 bool/error params in sequence, error-prone.

**Fix:** Introduce a struct parameter:
```go
type PaneFetchState struct {
    IsFetching     bool
    FetchErr       error
    NeverFetched   bool
    IsThrottled    bool
    RetryAfterSecs int
}

func PaneEmptyStatus(category string, s PaneFetchState) EmptyState
```
Update all 8 pane call sites.

## Acceptance Criteria

- [ ] SavedEpisodes sets Hint for EmptyStatusNone
- [ ] EmptyStatusNone tested at pane level for all 5 panes
- [ ] FollowedShows, SavedEpisodes, RecentlyPlayed have 4 status tests each
- [ ] At least one golden test for non-never-fetched state
- [ ] `EmptyState.Category` field added; `Render()` uses it instead of TrimPrefix
- [ ] `PaneFetchState` struct introduced; `PaneEmptyStatus` uses it
- [ ] All 8 pane call sites updated for Category + PaneFetchState
- [ ] Golden files regenerated
- [ ] `make ci` passes

## Tasks

- [ ] Add Hint to SavedEpisodes EmptyStatusNone path
- [ ] Add EmptyStatusNone tests to Albums, Playlists, LikedSongs, TopTracks, TopArtists
- [ ] Add 4 status tests to FollowedShows, SavedEpisodes, RecentlyPlayed
- [ ] Add golden test for non-never-fetched state (one pane)
- [ ] Add `Category` field to `EmptyState`; update `Render()` and `PaneEmptyStatus()`
- [ ] Add `PaneFetchState` struct; update `PaneEmptyStatus()` signature
- [ ] Update all 8 pane call sites for Category + PaneFetchState
- [ ] Regenerate golden files: `go test ./... -update`
- [ ] `make ci` passes
