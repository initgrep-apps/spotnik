# Feature 36 — Command Safety & Error Handling

> **Feature:** Fix a data race in playback command closures, add error propagation
> to nil-client fallbacks, and add throttled playback error toasts.

## Context

PR reviews identified a data race where `buildPlaybackAPICmd` reads Store fields inside
goroutine closures instead of capturing values in Update(). Additionally, 7 command
builders silently return zero-value messages when the API client is nil, and playback
poll errors are silently ignored despite the "all errors via toast" rule.

**Source:** `docs/issues.md` — PR #34 issues 4, 6, 7; PR #36 issue 5

**Depends on:** Feature 35

---

## Task 1: Fix data race in buildPlaybackAPICmd closures

**Problem:** `buildPlaybackAPICmd` in `internal/app/commands.go` (lines 24-95) captures
`store := a.store` at line 27, then reads `store.PlaybackState()` inside the returned
closure (lines 49, 60, 71, 78). These closures execute asynchronously as `tea.Cmd`,
meaning they read Store while Update() may be writing — a data race.

**Fix:**

1. Snapshot all needed Store values in the `buildPlaybackAPICmd` function body
   (executed in Update() context, which is safe):
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

2. Update each action case in the closure to use the snapshotted values instead of
   calling `store.PlaybackState()`:
   - `ActionVolumeUp`: use `currentVolume` instead of `store.PlaybackState().VolumePercent`
   - `ActionVolumeDown`: use `currentVolume`
   - `ActionToggleShuffle`: use `isShuffled`
   - `ActionCycleRepeat`: use `repeatMode`

3. Remove the `store := a.store` capture at line 27 — it should no longer be needed

**Files:**
- Modify: `internal/app/commands.go` — buildPlaybackAPICmd

**Tests:**
- Unit: verify buildPlaybackAPICmd snapshots values (pass a mock store, change store after
  building the command, verify the command uses the original values)
- Run with `-race` flag: `go test -race ./internal/app/...`

**Commit:** `fix(app): snapshot store values in buildPlaybackAPICmd to prevent data race`

---

## Task 2: Add error to nil-client fallback messages

**Problem:** Seven command builders return zero-value messages with no error when the API
client is nil. This silently produces empty data that can stamp fetchedAt (see staleness
issue) and confuse error tracking.

**Nil-client sites in `internal/app/commands.go`:**
- Line 33: `buildPlaybackAPICmd` → `PlaybackCmdSentMsg{}`
- Line 157: `buildFetchPlaylistsCmd` → `LibraryLoadedMsg{Offset: offset}`
- Line 179: `buildFetchAlbumsCmd` → `AlbumsLoadedMsg{}`
- Line 201: `buildFetchLikedTracksCmd` → `LikedTracksLoadedMsg{Offset: offset}`
- Line 223: `buildFetchRecentlyPlayedCmd` → `RecentlyPlayedLoadedMsg{}`
- Line 269: `buildSearchCmd` → `SearchResultsMsg{}`
- Line 477: `fetchQueueCmd` → `QueueLoadedMsg{}`

**Fix:**

1. Create a sentinel error in `internal/app/commands.go`:
   ```go
   // errNilClient is returned when a command is built but the required API client is nil.
   // This typically means authentication has not completed yet.
   var errNilClient = fmt.Errorf("API client not initialized")
   ```

2. Update each nil-client fallback to include the error:
   ```go
   if a.player == nil {
       return func() tea.Msg { return panes.PlaybackCmdSentMsg{Err: errNilClient} }
   }
   ```

3. In the Update() handlers for each message type, when `m.Err` matches `errNilClient`,
   silently skip (no toast) — this is expected during startup. For other errors, emit toast.

**Files:**
- Modify: `internal/app/commands.go` — add errNilClient, update 7 fallbacks

**Tests:**
- Unit: verify each command builder returns message with errNilClient when client is nil
- Unit: verify Update() handlers don't emit toast for errNilClient

**Commit:** `fix(app): propagate errNilClient from nil-client command fallbacks`

---

## Task 3: Add throttled playback error toast

**Problem:** `PlaybackStateFetchedMsg.Err` is never checked (app.go lines 703-718). This is
the most frequently fired message (every 1-3s). When Err is non-nil, the handler silently
skips. Despite the "all errors via toast" rule, playback errors produce no feedback.

**Fix:**

1. Add a `consecutivePlaybackErrors int` field to the App struct
2. In the `PlaybackStateFetchedMsg` handler:
   ```go
   case panes.PlaybackStateFetchedMsg:
       if m.Err != nil {
           a.consecutivePlaybackErrors++
           if a.consecutivePlaybackErrors == 5 {
               cmd := a.alerts.NewAlertCmd("warning", "Playback updates failing — check connection")
               return a, cmd
           }
           // Don't toast on every single error — too noisy at 1-3s intervals
           return a, nil
       }
       a.consecutivePlaybackErrors = 0
       if m.State != nil {
           a.store.SetPlaybackState(m.State)
       }
       // ... rest of handler
   ```

3. The threshold of 5 consecutive errors means ~5-15 seconds of failures before alerting.
   Reset to 0 on any successful fetch.

**Files:**
- Modify: `internal/app/app.go` — add field, update PlaybackStateFetchedMsg handler

**Tests:**
- Unit: verify no toast on first playback error
- Unit: verify toast emitted on 5th consecutive error
- Unit: verify counter resets on successful fetch after errors

**Commit:** `feat(app): add throttled toast for consecutive playback errors`

---

## Task 4: Update issues.md

**Fix:** Mark PR #34 issues 4, 6, 7 and PR #36 issue 5 as resolved.

**Files:**
- Modify: `docs/issues.md`

**Commit:** `docs: mark command safety issues as resolved`

---

## Verification

```bash
# No store reads inside closures in buildPlaybackAPICmd
# (visual inspection — store.PlaybackState() should not appear inside func() tea.Msg{})

# Nil-client fallbacks include error
grep -n 'errNilClient' internal/app/commands.go
# Expected: 8+ matches (declaration + 7 usages)

# Playback error handling
grep -n 'consecutivePlaybackErrors' internal/app/app.go
# Expected: multiple matches

go test -race ./internal/app/...
# Expected: PASS (no race conditions)

make ci
# Expected: Full pass
```

---

*Depends on: Feature 35*
*Blocks: Feature 38*
