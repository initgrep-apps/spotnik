# Feature 64 — Gateway State Liveness & Peak Watermarks

> **Fix:** The Request Flow pane's gateway metrics (token bucket, concurrent semaphore)
> appear static because the snapshot refreshes only every 1s and transient state changes
> recover faster than the polling interval. This feature adds 200ms snapshot refresh and
> peak activity watermarks so the UI reflects real gateway activity.

## Background

The Request Flow pane displays gateway metrics via `p.lastSnapshot` (a `domain.GatewayState`).
Currently, `gateway.Snapshot()` is called only on `TickMsg` (every 1 second). The `viz.TickMsg`
(every 200ms) only advances arrow animation frames — it doesn't refresh the snapshot.

**Why values appear static:**
- Token bucket: capacity=10, refill rate=10/sec. A consumed token recovers in ~100ms. By the
  next 1s snapshot, tokens are back to 10/10.
- Concurrent semaphore: most HTTP requests complete in <100ms. The semaphore slot is released
  before the next snapshot, so `ConcurrentActive` always shows 0.

**Fix (two parts):**
1. **Part A:** Refresh `Snapshot()` on `viz.TickMsg` (200ms) — helps for longer-lived events
   (backoff timers, slow requests, dedup keys).
2. **Part B:** Track peak activity watermarks (`minTokens`, `peakConcurrent`) over 1-second
   windows. Display muted annotations like `(min: 8)` and `(peak: 2)` when activity occurred
   between snapshots.

**Depends on:** Feature 63 (Request Flow Boxed Guards)

---

## Task 1: Refresh gateway snapshot on viz.TickMsg

**Problem:** `viz.TickMsg` (200ms) only increments `frameIndex`. Gateway state is stale for up
to 1 second between `TickMsg` events.

**Fix:**

In `internal/ui/panes/requestflow_pane.go`, modify the `viz.TickMsg` handler in `Update()`:

```go
case viz.TickMsg:
    p.frameIndex++
    // Refresh gateway snapshot at 200ms resolution (5x faster than TickMsg).
    if p.gateway != nil {
        p.lastSnapshot = p.gateway.Snapshot()
    }
    p.syncFromNetLog()
    return p, nil
```

`Snapshot()` is cheap (reads under two locks, no allocations beyond the struct copy). Calling
it 5x more often adds negligible overhead.

**Files:**
- Modify: `internal/ui/panes/requestflow_pane.go` — `Update()` viz.TickMsg handler

**Tests:**
- `TestRequestFlowPane_VizTickMsg_RefreshesSnapshot` — modify gateway state, send viz.TickMsg,
  verify `lastSnapshot` reflects the change
- `TestRequestFlowPane_VizTickMsg_SyncsNetLog` — add entry to store net log, send viz.TickMsg,
  verify it appears in `recentReqs`

**Commit:** `fix(ui): refresh gateway snapshot on viz.TickMsg for 200ms resolution`

---

## Task 2: Add peak watermark fields to RequestFlowPane

**Problem:** Even at 200ms polling, sub-200ms events (single token consumption, fast concurrent
requests) are invisible because they recover before the next sample.

**Fix:**

Add watermark fields to `RequestFlowPane` struct in `requestflow_pane.go`:

```go
// peakConcurrent is the max ConcurrentActive seen since last TickMsg reset.
peakConcurrent int
// minTokens is the min TokensAvailable seen since last TickMsg reset.
minTokens int
```

Initialize `minTokens` in `NewRequestFlowPane()`:
```go
if gw != nil {
    p.lastSnapshot = gw.Snapshot()
    p.minTokens = p.lastSnapshot.TokensMax
}
```

**Files:**
- Modify: `internal/ui/panes/requestflow_pane.go` — struct fields, constructor

**Tests:**
- `TestRequestFlowPane_New_InitializesMinTokens` — verify `minTokens` is set to `TokensMax`
  after construction

**Commit:** `feat(ui): add peak watermark fields for gateway activity tracking`

---

## Task 3: Track and reset watermarks

**Problem:** Watermark fields exist but are never updated.

**Fix:**

In `requestflow_pane.go`, update the `viz.TickMsg` handler (after setting `p.lastSnapshot`):

```go
snap := p.gateway.Snapshot()
p.lastSnapshot = snap
// Track peak activity watermarks.
if snap.ConcurrentActive > p.peakConcurrent {
    p.peakConcurrent = snap.ConcurrentActive
}
if snap.TokensAvailable < p.minTokens {
    p.minTokens = snap.TokensAvailable
}
```

In the `TickMsg` handler, reset peaks before refreshing the snapshot:

```go
case TickMsg:
    // Reset peak watermarks every 1s so annotations reflect recent activity.
    p.peakConcurrent = 0
    if p.gateway != nil {
        p.minTokens = p.lastSnapshot.TokensMax
    }
    // Refresh gateway snapshot and sync requests from net log.
    if p.gateway != nil {
        p.lastSnapshot = p.gateway.Snapshot()
    }
    p.syncFromNetLog()
    return p, nil
```

**Files:**
- Modify: `internal/ui/panes/requestflow_pane.go` — viz.TickMsg and TickMsg handlers

**Tests:**
- `TestRequestFlowPane_PeakWatermarks_TrackMinTokens` — consume tokens on gateway, send
  viz.TickMsg, verify `minTokens` decreases below `TokensMax`
- `TestRequestFlowPane_PeakWatermarks_TrackPeakConcurrent` — acquire semaphore slot on gateway,
  send viz.TickMsg, verify `peakConcurrent` increases above 0
- `TestRequestFlowPane_PeakWatermarks_ResetOnTickMsg` — set peaks to non-default, send TickMsg,
  verify `peakConcurrent` resets to 0 and `minTokens` resets to `TokensMax`

**Commit:** `feat(ui): track and reset gateway peak watermarks on tick events`

---

## Task 4: Render peak annotations in gatewayStateLines()

**Problem:** `gatewayStateLines()` renders token/semaphore bars from `p.lastSnapshot` but doesn't
show peak activity.

**Fix:**

In `internal/ui/panes/requestflow_boxed.go`, modify `gatewayStateLines()`:

For the token bucket line, after building `tokenLine`:
```go
tokenLine := fmt.Sprintf("tokens  %s %d/%d", tokenBar, snap.TokensAvailable, snap.TokensMax)
if p.minTokens < snap.TokensAvailable {
    mutedAnnotation := mutedStyle.Render(fmt.Sprintf(" (min: %d)", p.minTokens))
    tokenLine += mutedAnnotation
}
```

For the semaphore line, after building `semLine`:
```go
semLine := fmt.Sprintf("conc    %s %d/%d", semBar, snap.ConcurrentActive, snap.ConcurrentMax)
if p.peakConcurrent > snap.ConcurrentActive {
    mutedAnnotation := mutedStyle.Render(fmt.Sprintf(" (peak: %d)", p.peakConcurrent))
    semLine += mutedAnnotation
}
```

Annotations only appear during activity. When idle, the display is unchanged.

**Files:**
- Modify: `internal/ui/panes/requestflow_boxed.go` — `gatewayStateLines()`

**Tests:**
- `TestGatewayStateLines_PeakAnnotation_Tokens` — set `minTokens` < current tokens, verify
  output contains "(min: N)"
- `TestGatewayStateLines_PeakAnnotation_Concurrent` — set `peakConcurrent` > current active,
  verify output contains "(peak: N)"
- `TestGatewayStateLines_NoPeakAnnotation_WhenIdle` — when peaks match current values, verify
  no "(min:" or "(peak:" in output

**Commit:** `feat(ui): render peak activity annotations in gateway state lines`

---

## Task 5: Update documentation

**Fix:**

1. Update `docs/features/00-overview.md` — mark Feature 64 complete
2. Update `docs/ARCHITECTURE.md` — in the Request Flow Rendering section, note:
   - Snapshot refreshes on viz.TickMsg (200ms) in addition to TickMsg (1s)
   - Peak watermarks (`minTokens`, `peakConcurrent`) track activity between snapshots
   - Annotations shown in gateway metrics when activity detected

**Files:**
- Modify: `docs/features/00-overview.md`
- Modify: `docs/ARCHITECTURE.md`

**Commit:** `docs: add Feature 64 gateway liveness to architecture docs`

---

## Acceptance Criteria

- [ ] Gateway snapshot refreshes on viz.TickMsg (every 200ms), not just TickMsg (1s)
- [ ] Net log syncs on viz.TickMsg so completed requests appear within 200ms
- [ ] `minTokens` tracks the lowest token count seen in the current 1-second window
- [ ] `peakConcurrent` tracks the highest concurrent active count in the current window
- [ ] Watermarks reset to defaults on each TickMsg (1-second boundary)
- [ ] Token line shows `(min: N)` annotation when `minTokens < TokensAvailable`
- [ ] Semaphore line shows `(peak: N)` annotation when `peakConcurrent > ConcurrentActive`
- [ ] No annotations shown when idle (peaks match current values)
- [ ] Existing viz.TickMsg frame advancement still works
- [ ] All existing tests pass
- [ ] `make ci` passes

---

## Verification

```bash
# viz.TickMsg refreshes snapshot
grep -A8 'case viz.TickMsg' internal/ui/panes/requestflow_pane.go

# Watermark fields exist
grep 'peakConcurrent\|minTokens' internal/ui/panes/requestflow_pane.go

# Peak annotations in rendering
grep 'min:\|peak:' internal/ui/panes/requestflow_boxed.go

# Run specific tests
go test ./internal/ui/panes/ -run 'VizTickMsg_Refreshes|PeakWatermarks|PeakAnnotation' -v

# Full CI
make ci
```

---

## Implementation Notes for Agents

### Key files to read first
- `internal/ui/panes/requestflow_pane.go` — Update() handler, struct, constructor
- `internal/ui/panes/requestflow_boxed.go` — gatewayStateLines() rendering
- `internal/api/gateway.go` — Snapshot() method, token bucket behavior (lines 186-232)
- `internal/domain/gateway.go` — GatewayState struct, GatewaySnapshotter interface

### Patterns to follow
- Watermark fields follow existing field patterns in the struct (e.g., `frameIndex`, `pollingState`)
- Peak annotations use `mutedStyle` (already defined in `gatewayStateLines()`)
- Test gateway state by calling `api.NewGateway()` and manipulating its state before Snapshot()

### Do NOT
- Change the `GatewaySnapshotter` interface or `GatewayState` struct
- Add new message types — reuse existing `viz.TickMsg` and `TickMsg`
- Import new dependencies
- Change the 200ms or 1s tick intervals
- Modify the token bucket or semaphore implementation in gateway.go

### Testing watermarks
To test that watermarks track correctly, you need to manipulate the gateway's internal state
between snapshots. Options:
- Use `api.NewGateway()` with an `httptest.NewServer` to make real requests that consume tokens
- Or use a mock `GatewaySnapshotter` that returns controlled `GatewayState` values — this is
  simpler and more deterministic for unit tests

---

*Depends on: Feature 63*
*Blocks: Nothing*
