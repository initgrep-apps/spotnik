---
title: "Command Safety & Error Handling"
feature: 03-playback
status: open
---

## Background
PR reviews identified a data race where `buildPlaybackAPICmd` reads Store fields inside goroutine closures instead of capturing values in Update(). Additionally, 7 command builders silently return zero-value messages when the API client is nil, and playback poll errors are silently ignored despite the "all errors via toast" rule.

**Source:** `docs/issues.md` -- PR #34 issues 4, 6, 7; PR #36 issue 5

**Depends on:** Feature 35 (Type Design Alignment)

## Design

### Task 1: Fix data race in buildPlaybackAPICmd closures

**Problem:** `buildPlaybackAPICmd` in `internal/app/commands.go` captures `store := a.store` then reads `store.PlaybackState()` inside the returned closure. These closures execute asynchronously as `tea.Cmd`, meaning they read Store while Update() may be writing -- a data race.

**Fix:** Snapshot all needed Store values in the `buildPlaybackAPICmd` function body (executed in Update() context, which is safe):

```go
func (a *App) buildPlaybackAPICmd(action panes.PlaybackAction) tea.Cmd {
    if a.player == nil {
        return func() tea.Msg { return panes.PlaybackCmdSentMsg{} }
    }
    player := a.player

    // Snapshot store values in Update() context (thread-safe).
    ps := a.store.PlaybackState()
    currentVolume := 0
    isShuffled := false
    repeatMode := ""
    if ps != nil {
        currentVolume = ps.VolumePercent
        isShuffled = ps.ShuffleState
        repeatMode = ps.RepeatState
    }
    // ... use snapshots inside closure, never read store inside closure
```

### Task 2: Add error to nil-client fallback messages

**Problem:** Seven command builders return zero-value messages with no error when the API client is nil. This silently produces empty data.

**Nil-client sites in `internal/app/commands.go`:**
- `buildPlaybackAPICmd` -> `PlaybackCmdSentMsg{}`
- `buildFetchPlaylistsCmd` -> `LibraryLoadedMsg{Offset: offset}`
- `buildFetchAlbumsCmd` -> `AlbumsLoadedMsg{}`
- `buildFetchLikedTracksCmd` -> `LikedTracksLoadedMsg{Offset: offset}`
- `buildFetchRecentlyPlayedCmd` -> `RecentlyPlayedLoadedMsg{}`
- `buildSearchCmd` -> `SearchResultsMsg{}`
- `fetchQueueCmd` -> `QueueLoadedMsg{}`

**Fix:** Create sentinel error `errNilClient` and include it in all nil-client fallback messages. In Update() handlers, silently skip `errNilClient` (expected during startup).

### Task 3: Add throttled playback error toast

**Problem:** `PlaybackStateFetchedMsg.Err` is never checked. When Err is non-nil, the handler silently skips. Playback errors produce no feedback.

**Fix:** Add `consecutivePlaybackErrors int` to App struct. Toast on 5th consecutive error ("Playback updates failing -- check connection"). Reset to 0 on success.

### Verification

```bash
# No store reads inside closures in buildPlaybackAPICmd
# Nil-client fallbacks include error
grep -n 'errNilClient' internal/app/commands.go
# Expected: 8+ matches

# Playback error handling
grep -n 'consecutivePlaybackErrors' internal/app/app.go
# Expected: multiple matches

go test -race ./internal/app/...
# Expected: PASS (no race conditions)

make ci
```

## Acceptance Criteria
- [ ] `buildPlaybackAPICmd` snapshots store values before closure, no reads inside closure
- [ ] All 7 nil-client fallbacks include `errNilClient` error
- [ ] `errNilClient` errors are silently skipped in Update() handlers (no toast)
- [ ] 5th consecutive playback error emits warning toast
- [ ] Counter resets to 0 on successful fetch
- [ ] `go test -race ./internal/app/...` passes
- [ ] `make ci` passes

## Tasks
- [ ] Snapshot store values in `buildPlaybackAPICmd` to prevent data race
      - test: verify command uses original values after store changes; `go test -race` passes
- [ ] Add `errNilClient` sentinel error to all 7 nil-client command fallbacks
      - test: verify each command builder returns message with errNilClient when client is nil; Update() handlers don't emit toast for errNilClient
- [ ] Add throttled toast for consecutive playback errors (threshold: 5)
      - test: no toast on first error; toast emitted on 5th consecutive error; counter resets on success
- [ ] Update `docs/issues.md` to mark resolved issues
      - test: None needed (documentation-only)
