---
title: "Polling optimization: skip invisible panes"
feature: 19-player-page-unification
status: done
---

## Background

The current implementation polls all library/podcast domains on every tick
cycle regardless of pane visibility. With the unified Player page, podcast panes
should not poll when a music preset is active, and music panes should not poll
when a podcast preset is active. This story adds a visibility gate to the
polling loop.

## Design

### Optimization rule

**Skip the entire polling iteration for a domain if its pane is not visible in
the current preset.**

```go
for _, entry := range pollingEntries {
    if !a.layout.IsPaneVisible(entry.paneID) {
        continue // skip: no staleness check, no sentinel check, no API dispatch
    }
    // existing staleness + sentinel + backoff logic...
}
```

### What's unchanged

- **Playback state** (`GET /me/player`): Always polls. Content-type switch happens
  in the response, not the request.
- **Queue** (`GET /me/player/queue`): Always polls. Queue contains both tracks
  and episodes, visible in both music and podcast presets.

### What's skipped

Staleness-gated data (playlists, albums, liked songs, recently played, top
tracks, top artists, stats, followed shows, saved episodes, show episodes) are
skipped entirely when their pane is not visible.

### What polls when

| Context | Playback State | Queue | Music Panes | Podcast Panes |
|---------|---------------|-------|-------------|---------------|
| Track, music preset | 3-10s adaptive | 9-30s adaptive | Poll if visible | Skip entirely |
| Track, podcast preset | 3-10s adaptive | 9-30s adaptive | Skip entirely | Poll if visible |
| Episode, podcast preset | 3-10s adaptive | 9-30s adaptive | Skip entirely | Poll if visible |
| Episode, music preset | 3-10s adaptive | 9-30s adaptive | Poll if visible | Skip entirely |
| Nothing, music preset | 10-30s adaptive | 30-60s adaptive | Poll if visible | Skip entirely |
| Nothing, podcast preset | 10-30s adaptive | 30-60s adaptive | Skip entirely | Poll if visible |

### Preset switch behavior

When the active preset changes (user presses `p`, or auto-switch fires):

1. Determine which panes are **newly visible** (in new preset's `Visible` map
   but not in old preset's)
2. For each newly visible pane, immediately check staleness
3. If stale (or never fetched), dispatch a fetch command right away
4. If fresh, data is already in Store — pane renders immediately with cached data

### Fetching sentinel interaction

A fetching sentinel set before a pane was hidden is NOT cleared by the tick
loop (the entry is skipped). It gets cleared normally when the fetch response
arrives via `*LoadedMsg` in the handler. Within one tick cycle, the sentinel
is naturally cleared, and the next time the pane becomes visible,
`IsFetching()` returns false and polling resumes.

### IsPaneVisible implementation

`IsPaneVisible(PaneID) bool` was added to the layout Manager in story 233.
It checks the current preset's `Visible` map for the given PaneID, AND checks
that the pane hasn't been manually toggled off (`!m.hidden[id]`).

## Files

### Modify

- `internal/app/handlers.go` — add `paneID` field to polling entries,
  add `IsPaneVisible()` check at top of loop
- `internal/ui/layout/layout.go` — verify `IsPaneVisible()` implementation
  (added in story 233)
- `internal/app/app.go` — add preset switch staleness check (call
  `checkNewlyVisiblePanes()` after `SetPreset()`)

## Acceptance Criteria

- [ ] Polling loop skips entries for panes not visible in the current preset
- [ ] Playback state and queue polling are NOT affected
- [ ] Preset switch triggers immediate staleness check for newly visible panes
- [ ] Stale data on preset switch dispatches fetch commands
- [ ] Fresh data on preset switch renders from cache without fetch
- [ ] Fetching sentinels are not leaked when panes are hidden
- [ ] FollowedShows pane episode data polled when in drill-down and visible
- [ ] `make ci` passes

## Tasks

- [ ] Add `paneID` field to polling entries in `handlers.go`
      - Modify `internal/app/handlers.go`: add `paneID layout.PaneID` to each polling entry struct
      - test: `TestPollingEntries_HavePaneID`, `TestPollingEntries_PlaybackAndQueue_NoPaneID`
- [ ] Add `IsPaneVisible()` check at top of library polling loop
      - Modify `internal/app/handlers.go`: `if !a.layout.IsPaneVisible(entry.paneID) { continue }` at top of loop
      - test: `TestPolling_SkipsInvisiblePane`, `TestPolling_PollsVisiblePane`
- [ ] Verify `IsPaneVisible(PaneID) bool` implementation on layout Manager (added in story 233)
      - Modify or verify `internal/ui/layout/layout.go`
      - test: `TestIsPaneVisible_CurrentPresetPane`, `TestIsPaneVisible_HiddenPane`
- [ ] Implement preset switch staleness check: `checkNewlyVisiblePanes()` method
      - Modify `internal/app/app.go`: after `SetPreset()`, determine newly visible panes and dispatch fetch if stale
      - test: `TestCheckNewlyVisiblePanes_DispatchesFetchForStalePanes`, `TestCheckNewlyVisiblePanes_SkipsFreshPanes`
- [ ] Verify fetching sentinels are not leaked when panes are hidden
      - test: `TestPolling_SentinelNotLeakedOnHide`, `TestPolling_SentinelClearedOnResponse`
- [ ] Verify playback state and queue polling are NOT affected by visibility gate
      - test: `TestPolling_PlaybackAlwaysPolled`, `TestPolling_QueueAlwaysPolled`
- [ ] Run `make ci` — all lint, tests, and 80% coverage pass