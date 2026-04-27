---
name: project_spotnik_feature64_complete
description: Feature 64 (Gateway Liveness Watermarks): 200ms snapshot refresh on viz.TickMsg, peak watermark fields, annotation rendering, exported accessor pattern
type: project
---

## Feature 64 — Gateway Liveness & Peak Watermarks

**What was built:**
- `viz.TickMsg` handler now calls `gateway.Snapshot()` + `syncFromNetLog()` plus advances `frameIndex` (200ms res, not 1s)
- `peakConcurrent` + `minTokens` fields added to `RequestFlowPane` struct
- Watermarks tracked each `viz.TickMsg`, reset on `TickMsg` (1s boundary)
- `(min: N)` + `(peak: N)` muted annotations in `gatewayStateLines()`
- `MinTokens()` + `PeakConcurrent()` exported accessors for external-pkg testing

**Key files:**
- `internal/ui/panes/requestflow_pane.go` — struct fields, ctor init, viz.TickMsg handler, TickMsg handler, exported accessors
- `internal/ui/panes/requestflow_boxed.go` — `gatewayStateLines()` annotation logic
- `internal/ui/panes/requestflow_pane_test.go` — external pkg tests w/ mockGateway
- `internal/ui/panes/requestflow_boxed_test.go` — internal pkg tests, direct field access

**Patterns established:**
- Exported accessors (`MinTokens()`, `PeakConcurrent()`) = test-only observation of unexported state from external test pkg. Follows `FrameIndex()` pattern.
- Internal test pkg (`package panes`) sets unexported fields directly (e.g. `p.minTokens = 6`) for isolated render unit tests.
- Token count display in boxed layout: use `SetSize(40, 20)` to force flat layout. Avoids `TruncateOrPad` cutting "3/10" → "3/…".

**Gotchas:**
- Box layout truncates content via `layout.TruncateOrPad`. Test asserting "3/10" must use flat layout (width < 60) else gateway box truncates to "3/…".
- `minTokens` reset on `TickMsg` uses `p.lastSnapshot.TokensMax` (pre-fresh-snapshot). Correct: `TokensMax` constant (always 10). Looks like stale data but spec requires this.
- Nil-gateway safe: if `gateway == nil`, `minTokens = 0` + `lastSnapshot.TokensAvailable = 0`, so `0 < 0 = false` — no spurious annotation. Not tested (pre-existing gap), verified by reasoning.
- `mockGateway` already defined in `requestflow_pane_test.go` (external pkg) line 390. Reused for all new watermark tests.

**Testing notes:**
- 9 new tests: 6 external pkg, 3 internal pkg
- Coverage: 86.4% overall, 90.1% `internal/ui/panes`
- All 9 TDD (failing-first confirmed via build errors pre-impl)