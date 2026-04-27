---
name: project_spotnik_feature33_complete
description: Feature 33 (Idle Polling Backoff): adaptive 4-state polling matrix, isIdle helper, pollIntervals method, tick handler wiring
type: project
---

## Feature 33 — Idle Polling Backoff

**Built:**
- `lastInteraction time.Time`, `idleThreshold time.Duration` fields on `App` struct
- `isIdle()` unexported helper: `time.Since(lastInteraction) > idleThreshold`
- `pollIntervals()` method: 4-state matrix (active/idle x playing/paused) → (playbackInterval, queueInterval int)
- Test wrappers exported: `IsIdle()`, `SetLastInteraction()`, `PollIntervals()`
- `idleThresholdSecs = 60` constant
- 6 interval constants replace old `playbackFetchInterval/queueFetchInterval`
- Tick handler calls `a.pollIntervals()` not hardcoded constants
- Idle→active reset: KeyMsg post-idle sets `tickCount = 0` for immediate fetch next tick
- `internal/app/idle_test.go` — 12 tests
- `docs/ARCHITECTURE.md` adds "Idle Polling Backoff (Feature 33)" subsection

**Key files:**
- `internal/app/app.go` — all idle polling (constants, fields, methods, KeyMsg + tick handlers)
- `internal/app/idle_test.go` — 12 TDD tests, all 4 states + edges

**4-state matrix:**
```
Active + Playing  →  3s / 9s   (full speed)
Active + Paused   → 10s / 30s  (reduced)
Idle   + Playing  → 10s / 30s  (reduced)
Idle   + Paused   → 30s / 60s  (slowest)
```

**Patterns:**
- Exported test wrapper: `func (a *App) IsIdle() bool { return a.isIdle() }`
- `SetLastInteraction(t time.Time)` injects time, avoids real sleep
- Idle→active reset in `tea.KeyMsg`: `wasIdle := a.isIdle(); a.lastInteraction = time.Now(); if wasIdle { a.tickCount = 0 }`
- `pollIntervals()` uses `switch` with explicit `!idle && playing` branches, not nested if

**Gotchas:**
- Don't call `cmd()` on tick update result in tests — runs `tea.Tick(time.Second, ...)`, BLOCKS 1s
- Verify tick handler without blocking: call `PollIntervals()` directly (what handler uses); don't inspect `cmd()` output
- `tea.Batch(nextTick)` makes BatchMsg with 1 item. Length check works but needs `cmd()` (blocks). Avoid.
- `isIdle()` uses strict `>` not `>=`; at exactly 60s, NOT idle

**Testing:**
- Coverage: 82.9% total (>80% threshold)
- TDD: 12 tests pre-implementation
- `SetLastInteraction(time.Now().Add(-61 * time.Second))` fakes idle, no sleep
- `TestTickHandler_IdlePaused_LongerIntervals` checks `PollIntervals()` returns 30/60 vs inspecting batch — dodges 1s block
- `for range 3` (Go 1.22+) advances tick count cleanly