# Spotnik — Non-Critical Issues Log

> Issues logged during PR reviews. Not merge-blocking but should be addressed.

---

## Feature 61: Fix Request Flow Gateway Visualization

*All issues resolved in PR #75.*

### ~~I61-1~~ RESOLVED (PR #75)
### ~~I61-2~~ RESOLVED (PR #75)
### ~~I61-3~~ DISCARDED — acceptable for 1s refresh diagnostic pane
### ~~I61-4~~ RESOLVED (PR #75) — documented caller requirement with IMPORTANT comment
### ~~I61-5~~ RESOLVED (PR #75)
### ~~I61-6~~ RESOLVED (PR #75)
### ~~I61-7~~ RESOLVED (PR #75)
### ~~I61-8~~ RESOLVED (PR #75)

---

## Feature 62: Request Flow Boxed Layout

*All issues resolved in PR #77 (Feature 63).*

### ~~I62-1~~ RESOLVED (PR #77) — post-clamp overflow guard added
### ~~I62-2~~ RESOLVED (PR #77) — doc comment added documenting caller precondition
### ~~I62-3~~ RESOLVED (PR #77) — maxRows guard added to both arrow builders
### ~~I62-4~~ RESOLVED (PR #77) — height fallback to viewFlat() when boxAreaHeight < 3

---

## Feature 64: Gateway Liveness & Watermarks

### ~~I64-1~~ SUPERSEDED by Feature 65 — UI-side watermark fields removed entirely
### I64-2 — Peak annotations truncated to fragments in narrow boxed layouts
- **File:** `internal/ui/panes/requestflow_boxed.go:119-131`
- **Description:** `(min: N)` and `(peak: N)` annotations are appended unconditionally. In narrow pane layouts (< ~45 cols), `TruncateOrPad` clips them to meaningless fragments like `(mi` or `(pe`. Consider suppressing annotations when box inner width is too small to display them fully.

---

## Feature 65: Gateway-Internal Watermarks

### I65-1 — `TestRequestFlowPane_TickMsg_CallsResetWatermarks` doesn't verify the call
- **File:** `internal/ui/panes/requestflow_pane_test.go`
- **Description:** The mock `ResetWatermarks()` is a no-op. The test doesn't assert the method was called. If someone removes the `p.gateway.ResetWatermarks()` call from the TickMsg handler, this test still passes. Add a `resetCalled bool` to the mock and assert it.

### I65-2 — `len(g.semaphore)` in `Snapshot()` read outside mutex
- **File:** `internal/api/gateway.go` — `Snapshot()` method
- **Description:** Pre-existing: `concurrentActive := len(g.semaphore)` is read after both mutexes are released. `PeakConcurrent` is read under `g.mu`. This inconsistency means `ConcurrentActive` could briefly exceed `PeakConcurrent`, suppressing a `(peak: N)` annotation. Consider moving `len(g.semaphore)` inside the `g.mu` critical section.
