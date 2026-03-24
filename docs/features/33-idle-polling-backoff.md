# Feature 33 — Idle Polling Backoff

> **Feature:** Reduce API polling frequency when the user is idle or playback is paused.
> Resume full-speed polling on user interaction.

## Context

The tick loop polls at full speed (3s playback, 9s queue) regardless of user activity
or playback state (app.go lines 40-47, tick handler at lines 483-521). When music is
paused or the user is on the stats/playlists view, polling wastes bandwidth and risks
rate limiting.

This feature is the proactive layer (Layer 1) of rate management. Feature 30 (API Gateway)
is the reactive layer (Layer 2). They are independent — this feature reduces the *number*
of requests entering the gateway; the gateway limits the *rate* of requests that pass through.

**Gap reference:** G6 in `docs/superpowers/plans/2026-03-24-architecture-baseline.md`

**Depends on:** Feature 29 (cleaner tick handler after Elm purity refactor)

---

## Polling Schedule

| State | Playback Interval | Queue Interval |
|---|---|---|
| Active + Playing | 3s (current) | 9s (current) |
| Active + Paused | 10s | 30s |
| Idle (60s no input) + Playing | 10s | 30s |
| Idle + Paused | 30s | 60s |

"Active" = user interacted within the last 60 seconds.
"Idle" = no `tea.KeyMsg` received for 60+ seconds.

---

## Task 1: Track lastInteraction time

**Problem:** The app has no concept of user activity recency.

**Fix:**

1. Add fields to App struct (app.go):
   ```go
   // lastInteraction is the last time a tea.KeyMsg was received.
   // Used to determine idle state for polling backoff.
   lastInteraction time.Time

   // idleThreshold is how long without input before the app is considered idle.
   idleThreshold time.Duration
   ```

2. In `New()`: initialize `lastInteraction: time.Now()`, `idleThreshold: 60 * time.Second`

3. In the `tea.KeyMsg` handler (app.go `handleKeyMsg` or wherever KeyMsg is first received):
   - Set `a.lastInteraction = time.Now()` before any other processing

4. Add helper:
   ```go
   // isIdle returns true if no user input has been received within idleThreshold.
   func (a *App) isIdle() bool {
       return time.Since(a.lastInteraction) > a.idleThreshold
   }
   ```

**Files:**
- Modify: `internal/app/app.go` — add fields, update New(), update KeyMsg handler

**Tests:**
- Unit: `isIdle()` returns false immediately after creation
- Unit: `isIdle()` returns true after threshold elapses
- Unit: KeyMsg resets lastInteraction

**Commit:** `feat(app): track last user interaction for idle detection`

---

## Task 2: Adaptive pollIntervals() method

**Problem:** Polling intervals are hardcoded constants.

**Fix:**

1. Add interval constants:
   ```go
   const (
       // Active + playing (current behavior)
       activePlayingPlaybackInterval = 3
       activePlayingQueueInterval    = 9

       // Active + paused OR idle + playing
       reducedPlaybackInterval = 10
       reducedQueueInterval    = 30

       // Idle + paused
       idlePlaybackInterval = 30
       idleQueueInterval    = 60
   )
   ```

2. Add method:
   ```go
   // pollIntervals returns the current playback and queue polling intervals
   // based on user activity and playback state.
   func (a *App) pollIntervals() (playbackInterval, queueInterval int) {
       idle := a.isIdle()
       playing := false
       if ps := a.store.PlaybackState(); ps != nil {
           playing = ps.IsPlaying
       }

       switch {
       case !idle && playing:
           return activePlayingPlaybackInterval, activePlayingQueueInterval
       case !idle && !playing:
           return reducedPlaybackInterval, reducedQueueInterval
       case idle && playing:
           return reducedPlaybackInterval, reducedQueueInterval
       default: // idle && !playing
           return idlePlaybackInterval, idleQueueInterval
       }
   }
   ```

**Files:**
- Modify: `internal/app/app.go` — add constants and method

**Tests:**
- Unit: active + playing → 3s/9s
- Unit: active + paused → 10s/30s
- Unit: idle + playing → 10s/30s
- Unit: idle + paused → 30s/60s

**Commit:** `feat(app): adaptive polling interval calculation`

---

## Task 3: Wire into tick handler + idle-to-active reset + docs

**Problem:** Tick handler uses hardcoded `playbackFetchInterval` and `queueFetchInterval`.

**Fix:**

1. Update tick handler (app.go lines 483-521):
   - Replace `a.tickCount%playbackFetchInterval == 0` with dynamic intervals:
   ```go
   playbackInterval, queueInterval := a.pollIntervals()
   if a.tickCount%playbackInterval == 0 {
       cmds = append(cmds, fetchPlaybackStateCmd(a.player))
   }
   if a.tickCount%queueInterval == 0 {
       cmds = append(cmds, fetchQueueCmd(a.player))
   }
   ```

2. Idle-to-active reset: when a `tea.KeyMsg` arrives after the app was idle,
   reset `a.tickCount = 0` to force an immediate fetch on the next tick. This
   gives instant feedback when the user returns to the app:
   ```go
   case tea.KeyMsg:
       wasIdle := a.isIdle()
       a.lastInteraction = time.Now()
       if wasIdle {
           // User returned from idle — force immediate poll on next tick.
           a.tickCount = 0
       }
       return a.handleKeyMsg(m)
   ```

3. Remove the old hardcoded constants `playbackFetchInterval` and `queueFetchInterval`
   (app.go lines 41-44). Replace with the new named constants in the adaptive system.
   Keep `defaultBackoffTicks` — it's used by the 429 handler, not polling.

4. Update docs:
   - **`docs/ARCHITECTURE.md`** → "Polling Architecture": Document idle backoff, the
     4-state polling schedule, and the two-layer rate management design (this feature
     is Layer 1: proactive; Feature 30 gateway is Layer 2: reactive)

**Files:**
- Modify: `internal/app/app.go` — update tick handler, add idle-to-active reset
- Modify: `docs/ARCHITECTURE.md` — add idle polling backoff documentation

**Tests:**
- Unit: tick handler uses dynamic intervals
- Unit: active+playing state fires at 3s/9s intervals
- Unit: idle+paused state fires at 30s/60s intervals
- Unit: KeyMsg after idle resets tickCount to 0
- Integration: verify polling interval changes when playback state changes

**Commit 1:** `feat(app): adaptive polling with idle backoff`
**Commit 2:** `docs: add idle polling backoff to architecture docs`

---

## Verification

```bash
# Old hardcoded constants removed
grep -r 'playbackFetchInterval\b\|queueFetchInterval\b' internal/app/app.go
# Expected: ZERO matches (replaced by adaptive constants)

# New adaptive system in place
grep -r 'pollIntervals\|isIdle\|lastInteraction' internal/app/app.go
# Expected: multiple matches

make ci
# Expected: Full pass
```

---

*Depends on: Feature 29*
*Blocked by: Feature 29*
*Blocks: Nothing*
