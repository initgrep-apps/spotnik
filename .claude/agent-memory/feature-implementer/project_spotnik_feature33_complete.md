---
name: project_spotnik_feature33_complete
description: Feature 33 (Idle Polling Backoff): adaptive 4-state polling matrix, isIdle helper, pollIntervals method, tick handler wiring
type: project
---

## Feature 33 — Idle Polling Backoff

**What was built:**
- `lastInteraction time.Time` and `idleThreshold time.Duration` fields in `App` struct
- `isIdle()` unexported helper: `time.Since(lastInteraction) > idleThreshold`
- `pollIntervals()` method: 4-state matrix (active/idle x playing/paused) → (playbackInterval, queueInterval int)
- Exported wrappers for testing: `IsIdle()`, `SetLastInteraction()`, `PollIntervals()`
- `idleThresholdSecs = 60` constant
- 6 interval constants replacing old hardcoded `playbackFetchInterval/queueFetchInterval`
- Tick handler updated to call `a.pollIntervals()` instead of hardcoded constants
- Idle-to-active reset: KeyMsg after idle sets `tickCount = 0` for immediate next-tick fetch
- `internal/app/idle_test.go` with 12 tests
- `docs/ARCHITECTURE.md` updated with "Idle Polling Backoff (Feature 33)" subsection

**Key files:**
- `internal/app/app.go` — all idle polling infrastructure (constants, fields, methods, KeyMsg handler, tick handler)
- `internal/app/idle_test.go` — 12 TDD tests for all 4 states and edge cases

**4-state matrix:**
```
Active + Playing  →  3s / 9s   (full speed)
Active + Paused   → 10s / 30s  (reduced)
Idle   + Playing  → 10s / 30s  (reduced)
Idle   + Paused   → 30s / 60s  (slowest)
```

**Patterns established:**
- Exported wrapper pattern for test helpers: `func (a *App) IsIdle() bool { return a.isIdle() }`
- `SetLastInteraction(t time.Time)` injected for test time control (avoids real sleep)
- Idle-to-active reset in `tea.KeyMsg` handler: `wasIdle := a.isIdle(); a.lastInteraction = time.Now(); if wasIdle { a.tickCount = 0 }`
- `pollIntervals()` uses `switch` with explicit `!idle && playing` etc. branches (not nested if)

**Gotchas:**
- Do NOT call `cmd()` on the result of a tick update in tests — it executes `tea.Tick(time.Second, ...)` which BLOCKS for 1 second
- To verify tick handler behavior without blocking: use `PollIntervals()` directly (which is what the tick handler calls), rather than inspecting `cmd()` output
- `tea.Batch(nextTick)` creates a BatchMsg with 1 item. Inspecting batch length works but requires calling `cmd()` (which blocks). Avoid this pattern.
- The `isIdle()` method uses strict `>` not `>=`, so at exactly 60s the app is NOT yet idle

**Testing notes:**
- Coverage: 82.9% total (above 80% threshold)
- TDD: 12 tests written before implementation
- `SetLastInteraction(time.Now().Add(-61 * time.Second))` simulates idle without sleep
- The `TestTickHandler_IdlePaused_LongerIntervals` test should verify `PollIntervals()` returns slow intervals (30/60) rather than inspecting batch output — avoids the 1s blocking gotcha
- The `for range 3` syntax (Go 1.22+) works cleanly for advancing tick count
