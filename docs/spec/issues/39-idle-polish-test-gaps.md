# Feature 39 — Idle Polish & Test Coverage Gaps

> **Feature:** Polish idle backoff behavior (WindowSizeMsg, backoff interaction, nil state
> observability) and fill test coverage gaps across search, stats, and toast assertions.

## Context

PR reviews identified 3 idle backoff UX issues and 8 test coverage gaps spread across
multiple features. Grouping them here for a single cleanup pass.

**Source:** `docs/issues.md` — PR #38 issues 11-13; PR #36 issue 1; PR #34 issues 8-10

**Depends on:** Feature 38

---

## Task 1: Reset idle on tea.WindowSizeMsg

**Problem:** Only `tea.KeyMsg` resets `lastInteraction` and idle state (app.go lines 614-621).
Terminal resize (`tea.WindowSizeMsg`) implies user presence but does not reset idle polling.

**Fix:**

1. In the `tea.WindowSizeMsg` handler (app.go lines ~559-578), add idle reset:
   ```go
   case tea.WindowSizeMsg:
       wasIdle := a.isIdle()
       a.lastInteraction = time.Now()
       if wasIdle {
           a.tickCount = 0
       }
       a.width = m.Width
       a.height = m.Height
       // ... rest of handler
   ```

**Files:**
- Modify: `internal/app/app.go` — WindowSizeMsg handler

**Tests:**
- Unit: verify WindowSizeMsg resets lastInteraction
- Unit: verify WindowSizeMsg resets tickCount when previously idle

**Commit:** `fix(app): reset idle state on terminal resize`

---

## Task 2: Show info toast on idle-return during backoff

**Problem:** When a user returns from idle during an active 429 backoff, `tickCount` resets
to 0 but the backoff guard prevents any fetches. No status indicator explains the stale data.

**Fix:**

1. In the `tea.KeyMsg` handler, after idle-to-active reset, check for active backoff:
   ```go
   case tea.KeyMsg:
       wasIdle := a.isIdle()
       a.lastInteraction = time.Now()
       if wasIdle {
           a.tickCount = 0
           if a.backoffTicks > 0 {
               cmd := a.alerts.NewAlertCmd("ratelimit",
                   fmt.Sprintf("Rate limited — resuming in %ds", a.backoffTicks))
               return a, tea.Batch(cmd, a.handleKeyMsg(m))
           }
       }
       return a.handleKeyMsg(m)
   ```

**Files:**
- Modify: `internal/app/app.go` — KeyMsg handler

**Tests:**
- Unit: verify returning from idle during backoff emits ratelimit toast
- Unit: verify returning from idle without backoff does NOT emit toast

**Commit:** `feat(app): show rate limit toast when returning from idle during backoff`

---

## Task 3: Track nil PlaybackState duration

**Problem:** `pollIntervals()` silently defaults to "paused" when `store.PlaybackState()`
returns nil. If this persists beyond startup, it indicates a bug but produces no feedback.

**Fix:**

1. Add `nilPlaybackStateTicks int` field to App struct
2. In the tick handler, after calling `pollIntervals()`:
   ```go
   if a.store.PlaybackState() == nil {
       a.nilPlaybackStateTicks++
       if a.nilPlaybackStateTicks == 30 { // ~30-90s depending on interval
           cmd := a.alerts.NewAlertCmd("warning", "No playback state received — check Spotify connection")
           // Only warn once
       }
   } else {
       a.nilPlaybackStateTicks = 0
   }
   ```

3. The threshold of 30 ticks means warning fires after ~30s at active polling or ~15min
   at idle polling — reasonable for detecting a stuck state.

**Files:**
- Modify: `internal/app/app.go` — add field and tick handler logic

**Tests:**
- Unit: verify no warning before 30 nil ticks
- Unit: verify warning emitted at 30th nil tick
- Unit: verify counter resets on non-nil state

**Commit:** `feat(app): warn after prolonged nil playback state`

---

## Task 4: Strengthen weak toast assertion tests

**Problem:** Five tests in `app_test.go` only assert `cmd != nil` instead of verifying
toast content and type. They should use the two-pass pattern.

**Tests to fix:**
- `TestApp_LikeToggleResultMsg_WithError` (line 436)
- `TestApp_PlaybackCmdSentMsg_WithError` (line 460)
- `TestApp_AddToQueueResultMsg_Success` (line 608)
- `TestApp_AddToQueueResultMsg_Error` (line 621)
- `TestApp_DeviceTransfer_ShowsStatusMessage` (line 1008)

**Fix:**

For each test, replace `assert.NotNil(t, cmd)` with the two-pass pattern:
```go
// Execute the command to get the alert message.
alertMsg := cmd()
// Feed the alert message to Update to render the toast.
_, alertCmd := a.Update(alertMsg)
// The alert should now be visible in the rendered view.
view := a.View()
assert.Contains(t, view, "expected text")
```

**Note:** The exact "expected text" depends on what each handler emits. Read the
corresponding Update() handler to determine the expected toast message.

**Files:**
- Modify: `internal/app/app_test.go` — strengthen 5 test functions

**Tests:**
- Self-verifying — the strengthened tests ARE the verification

**Commit:** `test(app): strengthen toast assertion tests with two-pass pattern`

---

## Task 5: Add buildSearchCmd store isolation test

**Problem:** No test verifies that `buildSearchCmd` does NOT write to the store.

**Fix:**

Add to `internal/app/elm_purity_test.go`:
```go
func TestBuildSearchCmd_DoesNotWriteToStore(t *testing.T) {
    // Setup app with search client
    // Take snapshot of store state
    // Call buildSearchCmd
    // Execute the returned command
    // Verify store state unchanged
}
```

**Files:**
- Modify: `internal/app/elm_purity_test.go` — add store isolation test

**Commit:** `test(app): add buildSearchCmd store isolation test`

---

## Task 6: Add SearchResultsMsg error path test

**Problem:** `SearchResultsMsg` error and clear paths are only tested indirectly.

**Fix:**

Add to `internal/app/elm_purity_test.go`:
```go
func TestSearchResultsMsg_ErrorPath(t *testing.T) {
    // Send SearchResultsMsg with non-nil Err
    // Verify store search results not updated
    // Verify error toast emitted
}

func TestSearchResultsMsg_ClearPath(t *testing.T) {
    // First set search results in store
    // Send SearchClearedMsg
    // Verify store search results cleared
}
```

**Files:**
- Modify: `internal/app/elm_purity_test.go` — add error and clear path tests

**Commit:** `test(app): add SearchResultsMsg error and clear path tests`

---

## Task 7: Add concurrent stats partial failure test

**Problem:** When `TopTracks` succeeds but `TopArtists` fails (or vice versa), the behavior
is untested.

**Fix:**

Add test that verifies partial failure handling:
```go
func TestStatsLoadedMsg_PartialFailure(t *testing.T) {
    // Send StatsLoadedMsg with TopTracks data but nil TopArtists and Err set
    // Verify store has TopTracks but not TopArtists for that range
    // Verify error toast emitted
}
```

**Note:** The current implementation in `buildFetchStatsCmd` (commands.go) uses
`sync.WaitGroup` to fetch both in parallel. Check how partial failures are currently
handled — the Msg may carry both or only the successful result.

**Files:**
- Modify: `internal/app/elm_purity_test.go` or `internal/app/app_test.go` — add partial failure test

**Commit:** `test(app): add concurrent stats partial failure test`

---

## Task 8: Update issues.md

**Fix:** Mark all remaining issues as resolved: PR #38 issues 11-13, PR #36 issue 1,
PR #34 issues 8-10.

**Files:**
- Modify: `docs/issues.md`

**Commit:** `docs: mark all remaining issues as resolved`

---

## Verification

```bash
# WindowSizeMsg resets idle
grep -n 'lastInteraction' internal/app/app.go | grep -i window
# Expected: match in WindowSizeMsg handler

# Backoff toast on idle return
grep -n 'resuming in' internal/app/app.go
# Expected: match in KeyMsg handler

# All weak tests strengthened
grep -c 'assert.NotNil.*toast' internal/app/app_test.go
# Expected: ZERO (replaced with Contains assertions)

# New elm purity tests
grep -c 'TestBuildSearchCmd\|TestSearchResultsMsg.*Error\|TestStatsLoadedMsg.*Partial' internal/app/elm_purity_test.go
# Expected: 3+ matches

make ci
# Expected: Full pass
```

---

*Depends on: Feature 38*
*Blocks: Nothing*
