---
title: "Per-Pane Error State in Loaded-Message Handlers"
feature: 15-error-resilience
status: done
---

## Background

Story 199 wires the tick-driven polling dispatch. This story wires the other end: the
loaded-message handlers that receive poll results. Without this story, polls fire but error
counts never accumulate, backoff never engages, and duplicate error toasts fire on every failed
tick.

Two related fixes land here: the playback polling error threshold (currently fires toast on 5th
consecutive error — should be 3rd), and the "Press Tab to retry" recovery hint body (now
obsolete since polling handles retry automatically).

**Depends on:** Story 199 (`pollState`, `calcBackoffTicks`, interval constants in `app.go`)

## Design

### `internal/app/handlers.go` — per-pane pollState updates

For each of `LibraryLoadedMsg`, `AlbumsLoadedMsg`, `LikedTracksLoadedMsg`,
`RecentlyPlayedLoadedMsg`, `StatsLoadedMsg`, and `DevicesLoadedMsg`:

**Error path — replace existing error block:**

```go
if m.Err != nil {
    if errors.Is(m.Err, errNilClient) {
        return a, nil
    }
    a.<pane>Poll.errorCount++
    a.<pane>Poll.backoffTicks = calcBackoffTicks(a.<pane>Poll.errorCount)
    // ... existing store.Set*FetchError and pane forward ...
    if a.<pane>Poll.errorCount == 1 {
        return a, a.toasts.Cmd(uikit.Toast{
            Intent: uikit.ToastError,
            Title:  "Failed to load <DataType>",
            Body:   "Retrying automatically.",
        })
    }
    return a, nil  // suppress duplicate toasts on consecutive failures
}
```

**Success path — add poll state reset before existing pane forward:**

```go
wasErr := a.<pane>Poll.errorCount > 0
a.<pane>Poll.errorCount = 0
a.<pane>Poll.backoffTicks = 0
a.<pane>Poll.hasData = true
// ... existing store clear and pane forward ...
var cmds []tea.Cmd
if wasErr {
    cmds = append(cmds, a.toasts.Cmd(uikit.Toast{
        Intent: uikit.ToastInfo,
        Title:  "<DataType> loaded",
    }))
}
// append pane forward cmd if any
if len(cmds) > 0 {
    return a, tea.Batch(cmds...)
}
return a, nil
```

Pane-specific titles:
- Playlists: `"Failed to load playlists"` / `"Playlists loaded"`
- Albums: `"Failed to load albums"` / `"Albums loaded"`
- Liked tracks: `"Failed to load liked tracks"` / `"Liked tracks loaded"`
- Recently played: `"Failed to load recently played"` / `"Recently played loaded"`
- Stats: `"Failed to load stats"` / `"Stats loaded"`
- Devices: update error/success state only — no recovery toast (overlay auto-refreshes visually)

### `internal/app/handlers.go` — playback error threshold

Change the `consecutivePlaybackErrors` comparison in the `PlaybackStateFetchedMsg` handler:

```go
// OLD:
if a.consecutivePlaybackErrors == 5 {
// NEW:
if a.consecutivePlaybackErrors == 3 {
```

Update the adjacent comment from "5th consecutive failure" to "3rd consecutive failure".

### `internal/app/handlers.go` — replace "Press Tab to retry" body

The `RecoveryPressTabRetry` body is now obsolete for library panes — polling handles retry.
The per-pane error blocks added above already use `"Retrying automatically."`. Verify no
remaining `uikit.RecoveryPressTabRetry` references exist in library loaded-msg handlers after
this story. (References in other handlers — e.g. playlist/album tracks — are unaffected.)

## Acceptance Criteria

- [ ] `LibraryLoadedMsg` error path: increments `playlistsPoll.errorCount`, sets `backoffTicks`; emits toast only on `errorCount == 1`; subsequent errors are silent
- [ ] `LibraryLoadedMsg` success path: resets `playlistsPoll` fields; emits `ToastInfo "Playlists loaded"` if `wasErr`
- [ ] Same pattern applied to `AlbumsLoadedMsg`, `LikedTracksLoadedMsg`, `RecentlyPlayedLoadedMsg`, `StatsLoadedMsg`
- [ ] `DevicesLoadedMsg`: error increments `devicesPoll.errorCount` + backoff; success resets — no recovery toast
- [ ] Playback error toast fires on 3rd consecutive error, not 5th
- [ ] No `uikit.RecoveryPressTabRetry` body remaining in library or stats loaded-msg handlers
- [ ] `make ci` passes

## Tasks

- [ ] Write failing tests in `internal/app/poll_test.go`:
      `TestApp_LibraryLoaded_SuccessSetsPollHasData`,
      `TestApp_LibraryLoaded_ErrorFirstOnlyEmitsToast`,
      `TestApp_LibraryLoaded_RecoveryEmitsInfoToast`
      - test: `go test ./internal/app/ -run "TestApp_LibraryLoaded" -v` → FAIL

- [ ] Update `LibraryLoadedMsg` handler in `internal/app/handlers.go` per the Design section
      - test: `go test ./internal/app/ -run "TestApp_LibraryLoaded" -v` → all PASS

- [ ] Apply the same error/success pattern to `AlbumsLoadedMsg`, `LikedTracksLoadedMsg`,
      `RecentlyPlayedLoadedMsg`, `StatsLoadedMsg`, `DevicesLoadedMsg` handlers; write a
      corresponding test for each (`TestApp_AlbumsLoaded_*`, `TestApp_LikedTracksLoaded_*`, etc.)
      - test: `go test ./internal/app/ -run "TestApp_.*Loaded_" -v` → all PASS

- [ ] Write failing test `TestApp_PlaybackErrors_ToastOnThird`; change threshold
      `consecutivePlaybackErrors == 5` → `== 3` in `handlers.go`; update comment
      - test: `go test ./internal/app/ -run "TestApp_PlaybackErrors_ToastOnThird" -v` → PASS

- [ ] Grep for remaining `RecoveryPressTabRetry` in library loaded-msg handlers; replace with
      `"Retrying automatically."` where still present (should be zero after handler rewrites above)
      - test: `grep -n "RecoveryPressTabRetry" internal/app/handlers.go` — zero matches in library blocks

- [ ] `make ci` passes
