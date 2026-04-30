---
title: "Fix Recently Played Empty State"
feature: 08-stats
status: closed
---

## Background
Recently Played pane shows headers but no data rows, with no indication to the user that data is empty or loading. The data fetch chain is complete and correct -- `initialFetchCmds()` emits `FetchRecentlyPlayedRequestMsg{}`, the handler in `app.go` dispatches the API call, and the response is stored and forwarded to the pane. All other library panes (Playlists, Albums, Liked Songs) use the same `a.library` client and work correctly.

The Spotify recently-played API (`GET /v1/me/player/recently-played`) can return empty results depending on account settings and listening history. When it does, the pane renders an empty table with column headers and "1/1" pagination -- giving the appearance of a broken pane rather than communicating the absence of data.

Other panes already have empty state messages: NowPlaying ("Nothing playing"), Devices ("No devices found"), Search ("No results").

## Design

Add an empty state message in the `View()` method following established patterns.

### `internal/ui/panes/recentlyplayed_pane.go`

**`View()` method change:**
```go
// Before:
func (r *RecentlyPlayedPane) View() string {
    var parts []string
    if r.filter.IsActive() {
        parts = append(parts, r.filter.View(r.width))
    }
    parts = append(parts, r.table.View())
    return strings.Join(parts, "\n")
}

// After:
func (r *RecentlyPlayedPane) View() string {
    var parts []string
    if r.filter.IsActive() {
        parts = append(parts, r.filter.View(r.width))
    }
    if len(r.store.RecentlyPlayed()) == 0 && !r.filter.IsActive() {
        parts = append(parts, "  No recently played tracks")
    } else {
        parts = append(parts, r.table.View())
    }
    return strings.Join(parts, "\n")
}
```

### Files
- `internal/ui/panes/recentlyplayed_pane.go` -- Add empty state in `View()`
- `internal/ui/panes/recentlyplayed_pane_test.go` -- Add test for empty state message

## Acceptance Criteria
- [ ] When store has no recently played data, pane shows "No recently played tracks"
- [ ] When data is present, pane renders the table as before
- [ ] Filter mode still works correctly
- [ ] Test verifies empty state message
- [ ] `make ci` passes

## Tasks
- [ ] Add empty state message in RecentlyPlayedPane View() when store has no data
      - test: Empty store renders "No recently played tracks"; data present renders table; filter mode still works
