---
name: project_spotnik_feature65_complete
description: Feature 65 (Gateway-Internal Watermarks): minTokens in tokenBucket, peakConcurrent in Gateway.Do(), ResetWatermarks() with refill, mockGateway interface update
type: project
---

## Feature 65 — Gateway-Internal Watermarks

**What was built:**
- `tokenBucket.minTokens` field tracked inside `tokenBucket` struct (float64), updated under `tb.mu` at the exact moment of `tb.tokens--` in `wait()`
- `Gateway.peakConcurrent` field tracked in `Gateway` struct (int), updated under `g.mu` after semaphore acquisition in `Do()`
- `GatewayState.PeakConcurrent` and `GatewayState.MinTokens` added to domain struct
- `GatewaySnapshotter.ResetWatermarks()` added to interface — called by `RequestFlowPane.Update(TickMsg)` after `Snapshot()`
- `RequestFlowPane` lost its local `minTokens`/`peakConcurrent` fields and `MinTokens()`/`PeakConcurrent()` accessors
- `gatewayStateLines()` now reads `snap.MinTokens`/`snap.PeakConcurrent` from snapshot

**Key files:**
- `internal/domain/gateway.go` — `GatewayState` got `PeakConcurrent`/`MinTokens`; `GatewaySnapshotter` got `ResetWatermarks()`
- `internal/api/gateway.go` — `tokenBucket.minTokens`, `Gateway.peakConcurrent`, updated `Snapshot()`, new `ResetWatermarks()`
- `internal/ui/panes/requestflow_pane.go` — removed local watermark fields/accessors, updated handlers
- `internal/ui/panes/requestflow_boxed.go` — reads from `snap.MinTokens`/`snap.PeakConcurrent`

**Patterns established:**
- When adding a field to a watermark-tracking system, apply refill in the reset function too, otherwise the baseline can be set below the refilled value that `Snapshot()` returns, causing false annotations
- `tokenBucket` struct is the right place to track `minTokens` (not `Gateway`) since it has its own mutex covering `tb.tokens--`
- `GatewaySnapshotter` interface additions break all test mocks — search all test files for `Snapshot() domain.GatewayState` to find affected mocks

**Gotchas:**
- `ResetWatermarks()` MUST apply token refill (same as `Snapshot()`) before setting `minTokens`. Without this, after a 1s reset, the baseline gets set to the unfilled token level (e.g. 5.0 when 100ms has passed and tokens are at 5.3). The subsequent `Snapshot()` returns `TokensAvailable = 5` but `MinTokens = 5` set without refill, causing a false `(min: 5)` annotation. Fixed by computing `tokens + elapsed*rate` in `ResetWatermarks()` like `Snapshot()` does.
- `ResetWatermarks()` must NOT update `lastFill` — it only reads the current level, does not consume tokens
- `peakConcurrent` in `Do()` reads `len(g.semaphore)` under `g.mu`. This is safe because `semaphore <- struct{}{}` already happened — `len(g.semaphore)` gives the post-acquisition count including the current slot
- `mockGateway` in `requestflow_pane_test.go` needed a no-op `ResetWatermarks()` added to satisfy the updated interface

**Testing notes:**
- 6 new gateway watermark tests in `internal/api/gateway_test.go`
- 3 updated annotation tests in `requestflow_boxed_test.go` — now set `MinTokens`/`PeakConcurrent` directly on `p.lastSnapshot` (internal package test can set struct fields directly)
- External package tests in `requestflow_pane_test.go` were rewritten from pane accessor assertions to snapshot-value verification
- Coverage: 86.5% overall, api: 85.8%, ui/panes: 90.1%
