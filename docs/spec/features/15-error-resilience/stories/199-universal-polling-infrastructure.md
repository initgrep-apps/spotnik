---
title: "Universal Polling Infrastructure"
feature: 15-error-resilience
status: open
---

## Background

All library data panes (playlists, albums, liked songs, recently played, stats) currently fetch
once at startup via `initialFetchCmds()`. On failure, a toast fires with "Press Tab to retry" —
but Tab only rotates focus; it does not trigger a re-fetch. Users are stranded with an empty pane
until they restart the app.

The existing tick-based polling model (playback state + queue) is sound. This story extends it to
cover all data panes. A new `pollState` struct tracks per-pane health. Six `pollState` fields are
added to `App`. The `TickMsg` handler is extended to drive polling for all panes and the devices
overlay. `initialFetchCmds()` is removed — first load is handled by retry-mode polling (5s
interval until first success).

## Design

### `internal/app/app.go` — `pollState`, `libraryIntervals`, interval constants

Add after the `pollIntervals()` method:

```go
// pollState tracks per-pane polling health.
// Isolated from the global 429 backoff — a failed library fetch does not
// affect playback polling intervals.
type pollState struct {
    backoffTicks int  // ticks remaining before next retry after an error
    errorCount   int  // consecutive errors; drives exponential backoff calculation
    hasData      bool // true after first successful load; switches interval regime
}

// libraryIntervals defines the polling cadence (seconds) for a library data type.
type libraryIntervals struct {
    playing, paused, idle int
}

var (
    recentPlayedIntervals = libraryIntervals{playing: 30, paused: 60, idle: 120}
    likedSongsIntervals   = libraryIntervals{playing: 60, paused: 120, idle: 300}
    playlistsIntervals    = libraryIntervals{playing: 60, paused: 120, idle: 300}
    albumsIntervals       = libraryIntervals{playing: 120, paused: 300, idle: 600}
    statsIntervals        = libraryIntervals{playing: 3600, paused: 3600, idle: 3600}
)

// calcBackoffTicks computes per-pane exponential backoff: min(5 * 2^(errorCount-1), 60).
func calcBackoffTicks(errorCount int) int {
    if ticks := 5 * (1 << uint(errorCount-1)); ticks < 60 {
        return ticks
    }
    return 60
}

// libraryInterval returns the polling interval in seconds for the given pane.
// Returns 5 if the pane has never loaded data (retry mode).
// Music playing → Playing interval regardless of user activity.
// Idle only applies when paused.
func (a *App) libraryInterval(p *pollState, iv libraryIntervals) int {
    if !p.hasData {
        return 5
    }
    state := a.store.PlaybackState()
    if state != nil && state.IsPlaying {
        return iv.playing
    }
    if a.isIdle() {
        return iv.idle
    }
    return iv.paused
}
```

Add to `App` struct after `consecutivePlaybackErrors`:

```go
playlistsPoll    pollState
albumsPoll       pollState
likedSongsPoll   pollState
recentPlayedPoll pollState
statsPoll        pollState
devicesPoll      pollState
```

### `internal/app/app.go` — remove `initialFetchCmds`

Delete the `initialFetchCmds()` function. In `Init()`, remove the call to it. Keep:
- `tea.Tick(...)` for `TickMsg`
- `a.buildFetchCurrentUserCmd()` (needed for playlist ownership markers)

### `internal/app/handlers.go` — extend TickMsg handler

After the existing queue dispatch block and before `a.tickCount++`, add per-pane polling blocks
for playlists, albums, liked songs, recently played, stats, and devices overlay:

```go
// Library pane polling — per-pane exponential backoff, isolated from global 429.
if a.backoffTicks <= 0 {
    for _, entry := range []struct {
        p        *pollState
        iv       libraryIntervals
        fetching func() bool
        setFetch func(bool)
        cmd      func() tea.Cmd
    }{
        {&a.playlistsPoll, playlistsIntervals, a.store.PlaylistsFetching, a.store.SetPlaylistsFetching, func() tea.Cmd { return a.buildFetchPlaylistsCmd(0) }},
        {&a.albumsPoll, albumsIntervals, a.store.AlbumsFetching, a.store.SetAlbumsFetching, func() tea.Cmd { return a.buildFetchAlbumsCmd(0) }},
        {&a.likedSongsPoll, likedSongsIntervals, a.store.LikedFetching, a.store.SetLikedFetching, func() tea.Cmd { return a.buildFetchLikedTracksCmd(0) }},
        {&a.recentPlayedPoll, recentPlayedIntervals, a.store.RecentFetching, a.store.SetRecentFetching, func() tea.Cmd { return a.buildFetchRecentlyPlayedCmd() }},
        {&a.statsPoll, statsIntervals, nil, nil, func() tea.Cmd { return a.buildFetchStatsCmd("short_term") }},
    } {
        p := entry.p
        if p.backoffTicks > 0 {
            p.backoffTicks--
        } else if interval := a.libraryInterval(p, entry.iv); a.tickCount%interval == 0 {
            if entry.fetching == nil || !entry.fetching() {
                if entry.setFetch != nil {
                    entry.setFetch(true)
                }
                cmds = append(cmds, entry.cmd())
            }
        }
    }
}
// Devices overlay polling (10s) — only while the overlay is open.
if a.deviceOverlayOpen {
    const devicePollInterval = 10
    p := &a.devicesPoll
    if p.backoffTicks > 0 {
        p.backoffTicks--
    } else if a.tickCount%devicePollInterval == 0 {
        cmds = append(cmds, a.buildFetchDevicesCmd())
    }
}
```

## Acceptance Criteria

- [ ] `pollState`, `libraryIntervals`, interval constants, `calcBackoffTicks`, and `libraryInterval()` defined in `app.go`
- [ ] Six `pollState` fields on `App`: `playlistsPoll`, `albumsPoll`, `likedSongsPoll`, `recentPlayedPoll`, `statsPoll`, `devicesPoll`
- [ ] `initialFetchCmds()` deleted; `Init()` no longer dispatches library fetch commands
- [ ] TickMsg handler drives polling for 5 library panes and devices overlay
- [ ] Devices overlay polled every 10s only while `deviceOverlayOpen == true`
- [ ] Global 429 backoff (`a.backoffTicks > 0`) still suppresses all library polling
- [ ] Per-pane backoff (`p.backoffTicks`) counts down independently per tick
- [ ] `make ci` passes

## Tasks

- [ ] Write failing tests in `internal/app/poll_internal_test.go`:
      `TestCalcBackoffTicks`, `TestLibraryInterval_RetryMode`, `TestLibraryInterval_Playing`,
      `TestLibraryInterval_Paused`, `TestLibraryInterval_Idle_OnlyWhenPaused`,
      `TestLibraryInterval_PlayingOverridesIdle`, `TestLibraryInterval_Albums`,
      `TestLibraryInterval_Stats`
      - test: `go test ./internal/app/ -run "TestCalcBackoff|TestLibraryInterval" -v` → compile error

- [ ] Add `pollState`, `libraryIntervals`, interval constants, `calcBackoffTicks`, and
      `libraryInterval()` to `internal/app/app.go`; add 6 `pollState` fields to `App` struct
      - test: `go test ./internal/app/ -run "TestCalcBackoff|TestLibraryInterval" -v` → all PASS

- [ ] Write failing test `TestApp_Init_NoInitialFetchCmds` in `internal/app/app_test.go` —
      asserts that Init() batch does not contain any `FetchPlaylistsRequestMsg`,
      `FetchAlbumsRequestMsg`, `FetchLikedTracksRequestMsg`, `FetchRecentlyPlayedRequestMsg`,
      or `FetchStatsMsg`
      - test: `go test ./internal/app/ -run "TestApp_Init_NoInitialFetchCmds" -v` → FAIL

- [ ] Delete `initialFetchCmds()` from `app.go`; remove call from `Init()`
      - test: `go test ./internal/app/ -run "TestApp_Init_NoInitialFetchCmds" -v` → PASS
      - test: `go test ./internal/app/ -timeout 60s` → PASS (fix any tests that assumed the batch)

- [ ] Write failing tests in `internal/app/poll_test.go`:
      `TestApp_TickMsg_LibraryPollDispatchesAtTick0`,
      `TestApp_TickMsg_DevicesPollWhileOverlayOpen`,
      `TestApp_TickMsg_DevicesNotPolledWhenOverlayClosed`
      - test: `go test ./internal/app/ -run "TestApp_TickMsg" -v` → FAIL

- [ ] Extend TickMsg handler in `internal/app/handlers.go` with library + devices polling blocks
      - test: `go test ./internal/app/ -run "TestApp_TickMsg" -v` → all PASS

- [ ] `make ci` passes
