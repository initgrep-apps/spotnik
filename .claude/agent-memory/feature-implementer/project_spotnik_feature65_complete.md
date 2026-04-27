---
name: project_spotnik_feature65_complete
description: Feature 65 (Gateway-Internal Watermarks): minTokens in tokenBucket, peakConcurrent in Gateway.Do(), ResetWatermarks() with refill, mockGateway interface update
type: project
---

## Feature 65 — Gateway-Internal Watermarks

**Built:**
- `tokenBucket.minTokens` field in `tokenBucket` struct (float64), updated under `tb.mu` at moment of `tb.tokens--` in `wait()`
- `Gateway.peakConcurrent` field in `Gateway` struct (int), updated under `g.mu` after semaphore acquisition in `Do()`
- `GatewayState.PeakConcurrent`/`GatewayState.MinTokens` added to domain struct
- `GatewaySnapshotter.ResetWatermarks()` added to interface — called by `RequestFlowPane.Update(TickMsg)` after `Snapshot()`
- `RequestFlowPane` lost local `minTokens`/`peakConcurrent` fields + `MinTokens()`/`PeakConcurrent()` accessors
- `gatewayStateLines()` reads `snap.MinTokens`/`snap.PeakConcurrent` from snapshot

**Files:**
- `internal/domain/gateway.go` — `GatewayState` got `PeakConcurrent`/`MinTokens`; `GatewaySnapshotter` got `ResetWatermarks()`
- `internal/api/gateway.go` — `tokenBucket.minTokens`, `Gateway.peakConcurrent`, updated `Snapshot()`, new `ResetWatermarks()`
- `internal/ui/panes/requestflow_pane.go` — removed local watermark fields/accessors, updated handlers
- `internal/ui/panes/requestflow_boxed.go` — reads `snap.MinTokens`/`snap.PeakConcurrent`

**Patterns:**
- Adding field to watermark system → apply refill in reset func too, else baseline set below refilled value `Snapshot()` returns → false annotations
- `tokenBucket` struct = right place for `minTokens` (not `Gateway`) — has own mutex covering `tb.tokens--`
- `GatewaySnapshotter` interface adds break all test mocks — grep all tests for `Snapshot() domain.GatewayState`

**Gotchas:**
- `ResetWatermarks()` MUST apply token refill (like `Snapshot()`) before setting `minTokens`. Without: after 1s reset, baseline = unfilled level (e.g. 5.0 when 100ms passed, tokens at 5.3). Next `Snapshot()` returns `TokensAvailable = 5` but `MinTokens = 5` set without refill → false `(min: 5)` annotation. Fix: compute `tokens + elapsed*rate` in `ResetWatermarks()` like `Snapshot()`.
- `ResetWatermarks()` must NOT update `lastFill` — reads current level only, no consume
- `peakConcurrent` in `Do()` reads `len(g.semaphore)` under `g.mu`. Safe: `semaphore <- struct{}{}` already happened → `len(g.semaphore)` = post-acquisition count incl. current slot
- `mockGateway` in `requestflow_pane_test.go` needed no-op `ResetWatermarks()` for updated interface

**Tests:**
- 6 new gateway watermark tests in `internal/api/gateway_test.go`
- 3 updated annotation tests in `requestflow_boxed_test.go` — set `MinTokens`/`PeakConcurrent` directly on `p.lastSnapshot` (internal pkg test sets struct fields directly)
- External pkg tests in `requestflow_pane_test.go` rewritten from pane accessor assertions → snapshot-value verification
- Coverage: 86.5% overall, api: 85.8%, ui/panes: 90.1%