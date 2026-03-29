# Feature 68 — Request Flow Replay Engine

> **Enhancement:** Rewrite the Request Flow pane to consume gateway events from the
> event journal and replay them at human-observable speed (200ms per event). The GATEWAY
> box becomes a rich decision log showing every internal gateway action. Replaces the
> old snapshot-polling approach entirely.

## Background

The Request Flow pane currently observes gateway state by:
1. Polling `Snapshot()` on `viz.TickMsg` (200ms) and `TickMsg` (1s)
2. Reading completed requests from `NetLog` via `syncFromNetLog()`
3. Tracking watermarks (`minTokens`, `peakConcurrent`) for activity annotations

This approach fails because gateway decisions happen in microseconds and self-heal
before any poll can observe them. Feature 67 instrumented the Gateway to emit fine-grained
lifecycle events into a `GatewayEventLog`. This feature rewrites the pane to consume
those events and replay them as a slow-motion time machine.

**Key behavioral changes:**
- The pane no longer holds a `GatewaySnapshotter` or calls `Snapshot()`
- The pane reads events from `store.ReadEventsFrom()` using a cursor
- Events are queued and replayed one per `viz.TickMsg` (200ms minimum visibility)
- The GATEWAY box shows state bars + a scrolling decision log with icons
- Multiple requests animate concurrently at staggered phases

**Design spec:** `docs/superpowers/specs/2026-03-29-gateway-event-journal-design.md`

**Depends on:** Feature 67 (Gateway Event Instrumentation)

---

## Gap Summary

| # | Gap | Severity | Description |
|---|-----|----------|-------------|
| G1 | Pane polls snapshots instead of consuming events | Critical | No event cursor, no replay queue, no display state model |
| G2 | GATEWAY box is a static state dashboard | Critical | Should be state bars + scrolling decision log with icons |
| G3 | No request animation lifecycle | High | Requests should animate through phases (Entered → AtGateway → InFlight → Completed) |
| G4 | Old snapshot/netlog code still active | High | `syncFromNetLog`, `lastSnapshot`, `recentReqs`, `RequestCompletedMsg` need removal |

---

## Task 1: Add replay data model types

**Problem:** The pane has no data model for event replay. It uses `reqDisplay` (a flat
struct) and `lastSnapshot` (a point-in-time copy). The replay engine needs a richer
model with animation phases, request tracking by ID, and a decision log.

**Fix:**

Add types to `internal/ui/panes/requestflow_pane.go` (or a new file
`requestflow_replay.go` if the file is getting large):

```go
// animationPhase tracks where a request is in its visual lifecycle.
type animationPhase int

const (
    phaseEntered    animationPhase = iota // appeared in APP box
    phaseAtGateway                        // gateway decision rendered
    phaseInFlight                         // HTTP call in progress (arrow animating right)
    phaseCompleted                        // response received
    phaseDone                             // aged out, ready for removal
)

// requestAnimation tracks one request's visual state across all three boxes.
type requestAnimation struct {
    requestID   uint64
    method      string
    path        string
    priority    domain.RequestPriority
    phase       animationPhase
    decision    domain.EventKind // the gateway decision event kind
    statusCode  int
    durationMs  int64
    enteredAt   time.Time // when it appeared in APP box (replay time, not event time)
}

// decisionEntry is one line in the GATEWAY box's scrolling decision log.
type decisionEntry struct {
    kind    domain.EventKind
    label   string    // formatted: "✓ GET /player allowed", "↻ refilled → 10"
    shownAt time.Time // when this entry was added (for age-out)
}

// replayDisplayState is the render model that View() reads from.
// Updated by the replay loop on each viz.TickMsg.
type replayDisplayState struct {
    // snapshot is the gateway state from the most recently replayed event.
    snapshot domain.GatewayStateSnapshot
    // requests tracks active requests keyed by RequestID.
    requests map[uint64]*requestAnimation
    // decisions is the scrolling decision log for the GATEWAY box.
    decisions []decisionEntry
}
```

**Files:**
- Modify: `internal/ui/panes/requestflow_pane.go` — add types (or create
  `requestflow_replay.go` if file is >600 lines)

**Tests:**
- Unit: `replayDisplayState` zero value has nil map and empty decisions
- Unit: `animationPhase` constants have expected ordering

**Commit:** `feat(ui): add replay data model types for Request Flow pane`

---

## Task 2: Rewrite `RequestFlowPane` struct and constructor

**Problem:** The pane struct holds `gateway domain.GatewaySnapshotter`, `lastSnapshot`,
`recentReqs`, and related fields from the old approach. These need to be replaced with
the replay engine fields.

**Fix:**

In `internal/ui/panes/requestflow_pane.go`:

1. Replace the struct:
   ```go
   type RequestFlowPane struct {
       theme   theme.Theme
       store   *state.Store
       focused bool
       width   int
       height  int

       frameIndex   int                     // animation frame counter (200ms)
       eventCursor  uint64                  // cursor into GatewayEventLog
       replayQueue  []domain.GatewayEvent   // events waiting to be displayed
       displayState replayDisplayState      // what View() renders from

       pollingState PollingSnapshotMsg      // app-level polling snapshot
   }
   ```

2. Update constructor — no more `GatewaySnapshotter` parameter:
   ```go
   // NewRequestFlowPane creates a RequestFlowPane that reads gateway events from
   // the store's event log. The pane does not hold a gateway reference — it only
   // reads from the store, preserving the ui/ → state/ dependency direction.
   func NewRequestFlowPane(s *state.Store, t theme.Theme) *RequestFlowPane {
       return &RequestFlowPane{
           theme: t,
           store: s,
           displayState: replayDisplayState{
               requests: make(map[uint64]*requestAnimation),
           },
       }
   }
   ```

3. Update `app.go` call site to remove the gateway argument:
   ```go
   // Old: panes.NewRequestFlowPane(gw, s, t)
   // New: panes.NewRequestFlowPane(s, t)
   ```

4. Remove `RequestCompletedMsg` type and its handler from `Update()`.

5. Remove `reqDisplay` type (replaced by `requestAnimation`).

**Files:**
- Modify: `internal/ui/panes/requestflow_pane.go` — rewrite struct, constructor,
  remove `RequestCompletedMsg`, `reqDisplay`
- Modify: `internal/app/app.go` — update `NewRequestFlowPane()` call

**Tests:**
- Unit: `NewRequestFlowPane(store, theme)` returns a pane with empty display state
- Unit: Pane ID, Title, ToggleKey unchanged

**Commit:** `refactor(ui): rewrite RequestFlowPane struct for event replay`

---

## Task 3: Implement the replay loop in `Update()`

**Problem:** The `Update()` handler currently polls `Snapshot()` and calls
`syncFromNetLog()`. It needs to drain events from the journal, queue them, and
process one per tick.

**Fix:**

Replace the `viz.TickMsg` and `TickMsg` handlers in `Update()`:

```go
case viz.TickMsg:
    p.frameIndex++
    p.drainEvents()
    p.processNextEvent()
    p.ageOutEntries()
    return p, nil

case TickMsg:
    // TickMsg also triggers a drain/process cycle and updates polling state.
    p.drainEvents()
    p.processNextEvent()
    p.ageOutEntries()
    return p, nil
```

Add the replay methods:

```go
// drainEvents reads new events from the store's event log and appends
// them to the replay queue.
func (p *RequestFlowPane) drainEvents() {
    if p.store == nil {
        return
    }
    newCursor, events := p.store.ReadEventsFrom(p.eventCursor)
    p.eventCursor = newCursor
    p.replayQueue = append(p.replayQueue, events...)
}

// processNextEvent pops one event from the replay queue and updates
// the display state. One event per tick = 200ms minimum visibility.
func (p *RequestFlowPane) processNextEvent() {
    if len(p.replayQueue) == 0 {
        return
    }
    event := p.replayQueue[0]
    p.replayQueue = p.replayQueue[1:]

    // Update snapshot to this event's state.
    p.displayState.snapshot = event.Snapshot

    // Process request-scoped events.
    if event.RequestID > 0 {
        p.processRequestEvent(event)
    }

    // Append to decision log.
    p.displayState.decisions = append(p.displayState.decisions, decisionEntry{
        kind:    event.Kind,
        label:   formatDecisionLabel(event),
        shownAt: time.Now(),
    })
}

// processRequestEvent updates the requestAnimation for the given event.
func (p *RequestFlowPane) processRequestEvent(event domain.GatewayEvent) {
    anim, exists := p.displayState.requests[event.RequestID]
    if !exists {
        anim = &requestAnimation{
            requestID: event.RequestID,
            method:    event.Method,
            path:      event.Path,
            priority:  event.Priority,
            phase:     phaseEntered,
            enteredAt: time.Now(),
        }
        p.displayState.requests[event.RequestID] = anim
    }

    switch event.Kind {
    case domain.EventRequestEntered:
        anim.phase = phaseEntered
    case domain.EventSemaphoreAcquired:
        anim.phase = phaseAtGateway
    case domain.EventDedupJoined:
        anim.phase = phaseAtGateway
        anim.decision = domain.EventDedupJoined
    case domain.EventRequestWaited:
        anim.phase = phaseAtGateway
        anim.decision = domain.EventRequestWaited
    case domain.EventHttpCompleted:
        anim.phase = phaseInFlight
        anim.statusCode = event.StatusCode
        anim.durationMs = event.DurationMs
    case domain.EventRequestAllowed:
        anim.phase = phaseCompleted
        if anim.decision == 0 {
            anim.decision = domain.EventRequestAllowed
        }
    case domain.EventRequestBlocked:
        anim.phase = phaseCompleted
        anim.decision = domain.EventRequestBlocked
    case domain.EventDedupResolved:
        anim.phase = phaseCompleted
        anim.statusCode = event.StatusCode
        anim.decision = domain.EventDedupJoined
    }
}

// ageOutEntries removes old decisions and completed requests.
func (p *RequestFlowPane) ageOutEntries() {
    now := time.Now()
    // Age out decisions older than 3s.
    cutoff := now.Add(-3 * time.Second)
    filtered := p.displayState.decisions[:0]
    for _, d := range p.displayState.decisions {
        if d.shownAt.After(cutoff) {
            filtered = append(filtered, d)
        }
    }
    p.displayState.decisions = filtered

    // Age out completed requests older than 5s.
    completedCutoff := now.Add(-5 * time.Second)
    for id, anim := range p.displayState.requests {
        if anim.phase == phaseCompleted && anim.enteredAt.Before(completedCutoff) {
            delete(p.displayState.requests, id)
        }
    }
}

// formatDecisionLabel builds the display string for a decision log entry.
func formatDecisionLabel(e domain.GatewayEvent) string {
    switch e.Kind {
    case domain.EventRequestEntered:
        tag := "bg"
        if e.Priority == domain.PriorityInteractive {
            tag = "int"
        }
        return fmt.Sprintf("→ %s %s entered [%s]", e.Method, e.Path, tag)
    case domain.EventTokenConsumed:
        return fmt.Sprintf("⊖ token consumed → %d", e.Snapshot.TokensAvailable)
    case domain.EventTokenRefilled:
        return fmt.Sprintf("↻ tokens refilled → %d", e.Snapshot.TokensAvailable)
    case domain.EventSemaphoreAcquired:
        return fmt.Sprintf("⊞ semaphore acquired (%d/%d)",
            e.Snapshot.ConcurrentActive, e.Snapshot.ConcurrentMax)
    case domain.EventSemaphoreReleased:
        return fmt.Sprintf("⊟ semaphore released (%d/%d)",
            e.Snapshot.ConcurrentActive, e.Snapshot.ConcurrentMax)
    case domain.EventBackoffStarted:
        return fmt.Sprintf("⏳ backoff started %.0fs", e.Snapshot.BackoffRemaining)
    case domain.EventBackoffExpired:
        return "✓ backoff cleared"
    case domain.EventRequestAllowed:
        return fmt.Sprintf("✓ %s %s allowed", e.Method, e.Path)
    case domain.EventRequestWaited:
        return fmt.Sprintf("⧖ %s %s waited", e.Method, e.Path)
    case domain.EventRequestBlocked:
        return fmt.Sprintf("✗ %s %s blocked", e.Method, e.Path)
    case domain.EventDedupJoined:
        return fmt.Sprintf("⧖ %s %s dedup", e.Method, e.Path)
    case domain.EventDedupResolved:
        return fmt.Sprintf("✓ dedup resolved %d", e.StatusCode)
    case domain.EventHttpCompleted:
        return fmt.Sprintf("✓ %d %dms", e.StatusCode, e.DurationMs)
    default:
        return "? unknown event"
    }
}
```

**Files:**
- Modify: `internal/ui/panes/requestflow_pane.go` — replace `Update()` handlers,
  add replay methods, `formatDecisionLabel()`

**Tests:**
- `TestRequestFlowPane_Replay_DrainEvents` — add events to store, send viz.TickMsg,
  verify `eventCursor` advanced and events in queue
- `TestRequestFlowPane_Replay_ProcessOnePerTick` — add 3 events, send 1 viz.TickMsg,
  verify only 1 event processed (2 remain in queue)
- `TestRequestFlowPane_Replay_SnapshotUpdates` — process event, verify
  `displayState.snapshot` matches event's snapshot
- `TestRequestFlowPane_Replay_RequestPhaseProgression` — inject full request lifecycle,
  send ticks, verify phase advances through Entered → AtGateway → InFlight → Completed
- `TestRequestFlowPane_Replay_BlockedRequestSkipsInFlight` — blocked request goes
  directly to Completed (no InFlight phase)
- `TestRequestFlowPane_Replay_DecisionLogGrows` — inject events, verify decision log
  entries appear with correct labels
- `TestRequestFlowPane_Replay_DecisionLogAgesOut` — set shownAt to 4s ago, call
  ageOutEntries, verify entry removed
- `TestRequestFlowPane_Replay_CompletedRequestAgesOut` — completed request with
  enteredAt 6s ago is removed
- `TestFormatDecisionLabel_AllKinds` — table-driven test for all 13 event kinds

**Commit:** `feat(ui): implement replay loop for Request Flow pane`

---

## Task 4: Update `View()` and box rendering for replay display state

**Problem:** `View()` and the boxed layout builders read from `p.recentReqs` and
`p.lastSnapshot`. They need to read from `p.displayState` instead. The GATEWAY box
needs a decision log section below the state bars.

**Fix:**

1. Update `buildAppBoxLines()` in `requestflow_boxed.go` to read from
   `p.displayState.requests` — sort by `enteredAt`, newest first, render each
   request's endpoint with priority coloring. Dim completed requests.

2. Update `buildSpotifyBoxLines()` to read from `p.displayState.requests` — show
   status + latency only for requests in `phaseInFlight` or `phaseCompleted`.

3. Update `buildGatewayBoxLines()` to render two sections:
   - **State bars** (top): token bucket bar + semaphore bar + optional backoff timer,
     read from `p.displayState.snapshot`
   - **Decision log** (below): render `p.displayState.decisions` with icons and colors

4. Update `buildLeftArrowLines()` and `buildRightArrowLines()` to render arrows based
   on `requestAnimation.phase` and `requestAnimation.decision`:
   - Left arrow: shows gateway decision based on `anim.decision` field
   - Right arrow: shows HTTP outcome based on `anim.phase` and `anim.statusCode`
   - Arrows animate at phases where they're active, solid when phase has passed

5. Update `gatewayStateLines()` to include decision log entries. Remove the old
   watermark annotation code (`(min: N)`, `(peak: N)`).

6. Apply theme colors to decision log entries:
   | Element | Token |
   |---|---|
   | `→` enter (interactive) | `TextPrimary()` |
   | `→` enter (background) | `TextMuted()` |
   | `✓` allowed/completed/expired | `Success()` |
   | `✗` blocked | `Error()` |
   | `⧖` waited/dedup | `Warning()` |
   | `⊖`/`⊞`/`⊟` resource events | `TextSecondary()` |
   | `↻` refill | `TextMuted()` |

**Files:**
- Modify: `internal/ui/panes/requestflow_boxed.go` — update all `build*BoxLines()`,
  `gatewayStateLines()`, arrow builders; add decision log rendering
- Modify: `internal/ui/panes/requestflow_pane.go` — update `renderArrow()` to accept
  `requestAnimation` instead of `reqDisplay`

**Tests:**
- `TestRequestFlowPane_View_Boxed_ShowsDecisionLog` — inject events via replay,
  verify View() output contains decision log entries (icons + labels)
- `TestRequestFlowPane_View_Boxed_StateBarsFromSnapshot` — inject event with specific
  snapshot, verify token bar and semaphore bar match
- `TestRequestFlowPane_View_Boxed_RequestInAppBox` — inject RequestEntered event,
  verify endpoint appears in APP box section
- `TestRequestFlowPane_View_Boxed_ResponseInSpotifyBox` — inject HttpCompleted event,
  verify status + latency in SPOTIFY box
- `TestRequestFlowPane_View_Boxed_ArrowStates` — test all arrow states per animation phase
- `TestRequestFlowPane_View_Flat_StillWorks` — flat fallback still renders at width < 60
- `TestDecisionLog_ThemeColors` — verify decision entries use correct theme tokens

**Commit:** `feat(ui): update Request Flow View() to render from replay display state`

---

## Task 5: Remove old snapshot/netlog code from pane

**Problem:** The old code (`syncFromNetLog`, `lastSnapshot`, `recentReqs`, watermark
handlers) is now dead. Remove it cleanly.

**Fix:**

1. Remove from `requestflow_pane.go`:
   - `syncFromNetLog()` method
   - `reqDisplay` type (already done in Task 1 if not earlier)
   - `RequestCompletedMsg` type and handler (already done in Task 2 if not earlier)
   - Any remaining references to `lastSnapshot` or `recentReqs`

2. Remove from `requestflow_boxed.go`:
   - Old `gatewayStateLines()` that reads from `p.lastSnapshot` with watermark annotations
   - Old `buildAppBoxLines()` / `buildSpotifyBoxLines()` that read from `p.recentReqs`

3. Remove from `domain/gateway.go`:
   - `GatewayState` struct (replaced by `GatewayStateSnapshot`)
   - `GatewaySnapshotter` interface
   - `GatewayDecision` type and constants (replaced by `EventKind`)

4. Remove deprecated `Snapshot()` shim from `api/gateway.go` (added in Feature 67).

5. Update all imports and callers.

**Files:**
- Modify: `internal/ui/panes/requestflow_pane.go` — remove dead code
- Modify: `internal/ui/panes/requestflow_boxed.go` — remove dead code
- Modify: `internal/domain/gateway.go` — remove `GatewayState`, `GatewaySnapshotter`,
  `GatewayDecision`
- Modify: `internal/api/gateway.go` — remove deprecated `Snapshot()` shim

**Tests:**
- Verify all removed types/methods no longer compile if referenced
- Update/remove all tests that reference removed types
- All remaining tests pass

**Commit:** `refactor(ui): remove deprecated snapshot, netlog, and watermark code`

---

## Task 6: Update existing tests

**Problem:** The existing ~70 pane tests and ~35 boxed tests reference removed types
and methods. They need to be rewritten to inject events via `GatewayEventLog` and
verify output from the replay display state.

**Fix:**

1. Replace test helper pattern:
   - **Old:** `newTestRequestFlowPane()` creates pane with mock `GatewaySnapshotter`,
     injects data via `RequestCompletedMsg` or `syncFromNetLog`
   - **New:** `newTestRequestFlowPane()` creates pane with a `*state.Store`, injects
     events via `store.RecordEvent()`, then sends `viz.TickMsg` to trigger replay

2. Replace mock `GatewaySnapshotter` with direct event injection:
   ```go
   // Old pattern:
   mock := &mockGatewaySnapshotter{state: domain.GatewayState{TokensAvailable: 8}}
   pane := panes.NewRequestFlowPane(mock, store, theme)

   // New pattern:
   store.RecordEvent(domain.GatewayEvent{
       Kind: domain.EventTokenConsumed,
       Snapshot: domain.GatewayStateSnapshot{TokensAvailable: 8, TokensMax: 10},
   })
   pane.Update(viz.TickMsg{})
   ```

3. Tests to rewrite:
   - Arrow state tests → inject events with appropriate `EventKind`
   - Theme color tests → unchanged (just the data source changes)
   - Boxed/flat layout tests → unchanged (layout logic is the same)
   - Staleness tests → unchanged (staleness reads from Store, not events)
   - Snapshot refresh tests → replace with event drain tests
   - Watermark tests → replace with decision log tests

4. New tests to add:
   - Decision log rendering tests
   - Staggered parallel request animation tests
   - Event queue backlog handling tests

**Files:**
- Modify: `internal/ui/panes/requestflow_pane_test.go` — rewrite all tests
- Modify: `internal/ui/panes/requestflow_boxed_test.go` — update boxed layout tests

**Commit:** `test(ui): rewrite Request Flow tests for event replay engine`

---

## Task 7: Update documentation

**Fix:**

1. Update `docs/features/00-overview.md` — add Feature 68 row
2. Update `docs/ARCHITECTURE.md` — update Request Flow Rendering section:
   - Event journal replay model
   - `replayDisplayState` render model
   - Decision log in GATEWAY box
   - No more `GatewaySnapshotter`, `GatewayState`, `GatewayDecision`

**Files:**
- Modify: `docs/features/00-overview.md`
- Modify: `docs/ARCHITECTURE.md`

**Commit:** `docs: add Feature 68 Request Flow replay engine`

---

## Acceptance Criteria

- [ ] Pane no longer holds a `GatewaySnapshotter` reference
- [ ] Pane reads events from `store.ReadEventsFrom()` using a cursor
- [ ] Events replay at 200ms minimum visibility (one per viz.TickMsg)
- [ ] Event queue absorbs bursts naturally
- [ ] GATEWAY box shows state bars (token bucket, semaphore, backoff) from replay snapshot
- [ ] GATEWAY box shows scrolling decision log with icons below state bars
- [ ] Decision log entries use correct theme colors
- [ ] Multiple requests animate concurrently at staggered phases
- [ ] APP box shows request endpoints from `requestAnimation.phase >= phaseEntered`
- [ ] SPOTIFY box shows responses from `requestAnimation.phase >= phaseInFlight`
- [ ] Left arrow shows gateway decision, right arrow shows HTTP outcome
- [ ] Blocked requests show in APP box but skip InFlight/SPOTIFY
- [ ] Decisions age out after 3s, completed requests after 5s
- [ ] Three-box layout unchanged, flat fallback unchanged
- [ ] Status strip unchanged
- [ ] `GatewayState`, `GatewaySnapshotter`, `GatewayDecision` removed from domain/
- [ ] Deprecated `Snapshot()` shim removed from gateway
- [ ] All tests rewritten and passing
- [ ] `make ci` passes

---

## Verification

```bash
# Replay engine fields
grep 'eventCursor\|replayQueue\|displayState' internal/ui/panes/requestflow_pane.go

# No more GatewaySnapshotter in pane
! grep 'GatewaySnapshotter\|lastSnapshot\|syncFromNetLog' internal/ui/panes/requestflow_pane.go

# Decision log rendering
grep 'decisionEntry\|formatDecisionLabel' internal/ui/panes/requestflow_pane.go

# Old domain types gone
! grep 'GatewaySnapshotter\|GatewayDecision' internal/domain/gateway.go

# Tests
go test ./internal/ui/panes/ -run 'RequestFlow' -v

# Full CI
make ci
```

---

## Implementation Notes for Agents

### Key files to read first
- `internal/ui/panes/requestflow_pane.go` — the file being rewritten (~560 lines)
- `internal/ui/panes/requestflow_boxed.go` — box rendering to update (~270 lines)
- `internal/ui/panes/requestflow_pane_test.go` — ~70 tests to rewrite
- `internal/ui/panes/requestflow_boxed_test.go` — ~35 tests to update
- `internal/state/eventlog.go` — GatewayEventLog API (from Feature 66)
- `internal/domain/gateway.go` — GatewayEvent, EventKind, GatewayStateSnapshot

### Patterns to follow
- `displayState` is the single render model — View() reads only from it
- Event processing is in `Update()`, rendering is in `View()` — Elm architecture
- Use `lipgloss.NewStyle().Foreground(p.theme.Success())` for decision colors
- Keep `renderSubBox()`, `padRightVisible()`, `truncateStr()` helpers unchanged
- Keep the boxed/flat switching and defensive guards from Features 62–63

### Do NOT
- Import `api/` from the pane — read events from `store` only
- Change tick intervals (200ms viz, 1s app)
- Modify the status strip (polling state + staleness)
- Change the outer pane border or keybindings
- Add new dependencies

---

*Depends on: Feature 67*
*Blocks: Feature 69*
