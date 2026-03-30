---
title: "Gateway State Liveness & Peak Watermarks"
feature: 14-nerd-status
status: done
---

## Background
The Request Flow pane's gateway metrics appeared static because the snapshot refreshed only every 1 second and transient state changes recovered faster than the polling interval. Token bucket capacity is 10 with a refill rate of 10/sec, so a consumed token recovers in ~100ms. Most HTTP requests complete in <100ms, so the semaphore always showed 0 active. This story added 200ms snapshot refresh on viz.TickMsg and peak activity watermarks (minTokens, peakConcurrent) over 1-second windows, displaying muted annotations.

## Design

### 200ms Snapshot Refresh
Modify viz.TickMsg handler to call Snapshot() and syncFromNetLog() at 200ms resolution.

### Peak Watermark Fields
Add `peakConcurrent int` and `minTokens int` to RequestFlowPane. Initialize minTokens to TokensMax.

### Tracking and Reset
On viz.TickMsg: if ConcurrentActive > peakConcurrent, update; if TokensAvailable < minTokens, update. On TickMsg (1s): reset peakConcurrent to 0 and minTokens to TokensMax.

### Annotations
Token line appends muted `(min: N)` when minTokens < TokensAvailable. Semaphore line appends muted `(peak: N)` when peakConcurrent > ConcurrentActive. No annotations when idle.

## Acceptance Criteria
- [ ] Gateway snapshot refreshes on viz.TickMsg (200ms)
- [ ] Net log syncs on viz.TickMsg
- [ ] minTokens tracks lowest token count in 1-second window
- [ ] peakConcurrent tracks highest concurrent count in window
- [ ] Watermarks reset on each TickMsg (1-second boundary)
- [ ] Token line shows (min: N) annotation when active
- [ ] Semaphore line shows (peak: N) annotation when active
- [ ] No annotations when idle
- [ ] All existing tests pass
- [ ] make ci passes

## Tasks
- [ ] Refresh gateway snapshot on viz.TickMsg in requestflow_pane.go
      - test: modify gateway state, send viz.TickMsg, verify snapshot reflects change; net log entry appears after viz.TickMsg
- [ ] Add peak watermark fields to RequestFlowPane
      - test: minTokens initialized to TokensMax
- [ ] Track and reset watermarks on viz.TickMsg and TickMsg
      - test: track min tokens; track peak concurrent; reset on TickMsg
- [ ] Render peak annotations in gatewayStateLines() in requestflow_boxed.go
      - test: token annotation when active; concurrent annotation when active; no annotation when idle
- [ ] Update documentation
      - test: docs change only
