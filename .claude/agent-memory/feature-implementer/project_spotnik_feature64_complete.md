---
name: project_spotnik_feature64_complete
description: Feature 64 (Gateway Liveness Watermarks): 200ms snapshot refresh on viz.TickMsg, peak watermark fields, annotation rendering, exported accessor pattern
type: project
---

## Feature 64 — Gateway Liveness & Peak Watermarks

**What was built:**
- `viz.TickMsg` handler now calls `gateway.Snapshot()` + `syncFromNetLog()` in addition to advancing `frameIndex` (200ms resolution instead of 1s)
- `peakConcurrent` and `minTokens` fields added to `RequestFlowPane` struct
- Watermarks tracked on each `viz.TickMsg`, reset on `TickMsg` (1s boundary)
- `(min: N)` and `(peak: N)` muted annotations in `gatewayStateLines()`
- `MinTokens()` and `PeakConcurrent()` exported accessors for testing from external package

**Key files:**
- `internal/ui/panes/requestflow_pane.go` — struct fields, constructor init, viz.TickMsg handler, TickMsg handler, exported accessors
- `internal/ui/panes/requestflow_boxed.go` — `gatewayStateLines()` annotation logic
- `internal/ui/panes/requestflow_pane_test.go` — external package tests using mockGateway
- `internal/ui/panes/requestflow_boxed_test.go` — internal package tests with direct field access

**Patterns established:**
- Exported accessor methods (`MinTokens()`, `PeakConcurrent()`) for test-only observation of unexported state from external test packages — follows existing `FrameIndex()` pattern
- Internal test package (`package panes`) can directly set unexported fields like `p.minTokens = 6` for unit testing rendering logic in isolation
- When testing token count display in boxed layout, use `SetSize(40, 20)` to force flat layout (avoids `TruncateOrPad` truncating "3/10" to "3/…")

**Gotchas:**
- Box layout truncates content lines via `layout.TruncateOrPad`. A test asserting "3/10" appears in the view must use flat layout (width < 60) or the gateway box may truncate it to "3/…".
- `minTokens` reset on `TickMsg` uses `p.lastSnapshot.TokensMax` (before the fresh snapshot). This is correct because `TokensMax` is a constant (always 10) — but the ordering looks like "stale data". The spec explicitly requires this pattern.
- The nil-gateway case is safe: if `gateway == nil`, `minTokens = 0` and `lastSnapshot.TokensAvailable = 0`, so `0 < 0 = false` — no spurious annotation rendered. This is not explicitly tested (pre-existing gap) but was verified by reasoning.
- `mockGateway` struct is already defined in `requestflow_pane_test.go` (external package) at line 390. Reused for all new watermark tests.

**Testing notes:**
- 9 new tests: 6 in external package, 3 in internal package
- Coverage: 86.4% overall, 90.1% for `internal/ui/panes`
- All 9 tests use TDD (failing first confirmed by build errors before implementation)
