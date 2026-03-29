# Feature 67 — Gateway Event Instrumentation

> **Enhancement:** Instrument `Gateway.Do()` to emit fine-grained lifecycle events at
> every decision point, and add periodic internal event emission for token refills and
> backoff expiry. Retire the old `GatewayRecorder`, `Snapshot()`, `ResetWatermarks()`,
> watermark fields, and double-recording prevention — all replaced by the event journal.

## Background

Feature 66 added the domain types (`EventKind`, `GatewayEvent`, `GatewayStateSnapshot`)
and the `GatewayEventLog` ring buffer. This feature wires the Gateway as the event
producer — it emits events at every decision point in `Do()` and for internal state
changes (token refills, backoff expiry).

Currently, `Gateway.Do()` calls `recorder.RecordGatewayCall()` at 4-5 points to record
a single summary record per request. This feature replaces that with ~5-6 fine-grained
events per request, each carrying a snapshot of the gateway's state at that exact moment.

**Design spec:** `docs/superpowers/specs/2026-03-29-gateway-event-journal-design.md`

**Depends on:** Feature 66 (Gateway Event Types & Storage)

---

## Gap Summary

| # | Gap | Severity | Description |
|---|-----|----------|-------------|
| G1 | No event emission in Do() | Critical | Gateway decisions are recorded as flat summary records, not lifecycle events |
| G2 | No request ID linking | Critical | No way to link multiple events belonging to the same request |
| G3 | No internal event emission | High | Token refills and backoff expiry are invisible — no events emitted |
| G4 | Old recording system still active | High | `GatewayRecorder`, `Snapshot()`, watermarks need retirement |

---

## Task 1: Add `emitEvent()` helper and `captureSnapshot()` to Gateway

**Problem:** The Gateway has no mechanism to emit events with state snapshots. Each
emission point in `Do()` would need to manually build a `GatewayEvent` and capture the
gateway state, leading to duplicated code.

**Fix:**

In `internal/api/gateway.go`:

1. Add `nextRequestID atomic.Uint64` field to `Gateway` struct.

2. Add `recorder` field typed as `domain.GatewayEventRecorder` (replaces the current
   `GatewayRecorder` field). Update `SetRecorder()` to accept the new interface.

3. Add `captureSnapshot()` method that reads current state under locks:
   ```go
   // captureSnapshot reads the gateway's current state under locks.
   // Returns a GatewayStateSnapshot suitable for embedding in a GatewayEvent.
   //
   // Lock ordering: acquires bucket.mu first, then g.mu. The caller must NOT
   // hold g.mu when calling this method. If the caller already holds g.mu,
   // use captureSnapshotLocked() instead.
   func (g *Gateway) captureSnapshot() domain.GatewayStateSnapshot

   // captureSnapshotLocked reads gateway state when g.mu is already held.
   // Only acquires bucket.mu (safe — bucket.mu is never held when g.mu is acquired).
   // Reads semaphore length without a lock (channel len is always safe).
   // Reads inflight/backoff from g.mu-protected fields without re-acquiring.
   func (g *Gateway) captureSnapshotLocked() domain.GatewayStateSnapshot
   ```

   Both apply the same refill arithmetic as current `Snapshot()` for token level.

4. Add `emitEvent()` helper:
   ```go
   // emitEvent records a gateway event if a recorder is attached.
   // Captures a state snapshot at the current moment.
   // The caller must NOT hold g.mu — use emitEventLocked() if g.mu is held.
   func (g *Gateway) emitEvent(kind domain.EventKind, reqID uint64, method, path string,
       priority domain.RequestPriority, statusCode int, durationMs int64) {
       g.mu.Lock()
       rec := g.recorder
       g.mu.Unlock()
       if rec == nil {
           return
       }
       rec.RecordEvent(domain.GatewayEvent{
           Timestamp:  time.Now(),
           Kind:       kind,
           RequestID:  reqID,
           Method:     method,
           Path:       path,
           Priority:   priority,
           StatusCode: statusCode,
           DurationMs: durationMs,
           Snapshot:   g.captureSnapshot(),
       })
   }

   // emitEventLocked is like emitEvent but for use when g.mu is already held.
   // Reads recorder from the locked state and uses captureSnapshotLocked().
   func (g *Gateway) emitEventLocked(kind domain.EventKind, reqID uint64, method, path string,
       priority domain.RequestPriority, statusCode int, durationMs int64) {
       rec := g.recorder
       if rec == nil {
           return
       }
       rec.RecordEvent(domain.GatewayEvent{
           Timestamp:  time.Now(),
           Kind:       kind,
           RequestID:  reqID,
           Method:     method,
           Path:       path,
           Priority:   priority,
           StatusCode: statusCode,
           DurationMs: durationMs,
           Snapshot:   g.captureSnapshotLocked(),
       })
   }
   ```

**Files:**
- Modify: `internal/api/gateway.go` — add `nextRequestID`, change `recorder` type,
  add `captureSnapshot()`, `captureSnapshotLocked()`, `emitEvent()`, `emitEventLocked()`

**Tests:**
- Unit: `captureSnapshot()` returns correct token level (with refill applied)
- Unit: `captureSnapshot()` returns correct `ConcurrentActive` (len of semaphore)
- Unit: `emitEvent()` with nil recorder does not panic
- Unit: `emitEvent()` with recorder calls `RecordEvent()` with correct fields
- Unit: `nextRequestID` increments atomically across calls

**Commit:** `feat(api): add emitEvent helper and captureSnapshot for gateway events`

---

## Task 2: Instrument `Do()` with lifecycle events

**Problem:** `Do()` currently calls `recorder.RecordGatewayCall()` at 4-5 points,
producing one summary record per request. The event journal needs ~5-6 events per
request lifecycle.

**Fix:**

In `internal/api/gateway.go`, modify `Do()`:

1. At the top of `Do()`, generate a request ID and emit `EventRequestEntered`:
   ```go
   reqID := g.nextRequestID.Add(1)
   domainPriority := priorityToDomain(priority)
   g.emitEvent(domain.EventRequestEntered, reqID, key.Method, key.Path,
       domainPriority, 0, 0)
   ```

2. Replace each `recorder.RecordGatewayCall(...)` call with the appropriate event:

   | Current call | Replacement event | Notes |
   |---|---|---|
   | Background backoff → `RecordGatewayCall(DecisionBlocked)` | `emitEventLocked(EventRequestBlocked, ...)` | g.mu is held |
   | Background bucket.wait cancelled → `RecordGatewayCall(DecisionBlocked)` | `emitEvent(EventRequestBlocked, ...)` | g.mu not held |
   | Interactive backoff wait cancelled → `RecordGatewayCall(DecisionWaited)` | `emitEvent(EventRequestWaited, ...)` | g.mu not held |
   | Dedup waiter (phase 2) → `RecordGatewayCall(DecisionDeduped)` | `emitEvent(EventDedupJoined, ...)` then on resolution `emitEvent(EventDedupResolved, ...)` | |
   | Dedup waiter (phase 4) → same | same pattern | |
   | Final allowed → `RecordGatewayCall(DecisionAllowed/DecisionWaited)` | `emitEvent(EventRequestAllowed, ...)` | |

3. Add new events at points that were not previously recorded:

   | Point in Do() | Event | Current line ref |
   |---|---|---|
   | After `bucket.wait()` returns (token consumed) | `EventTokenConsumed` | After line ~104 in bucket.wait() — but bucket.wait() is on tokenBucket. Emit after bucket.wait() returns in Do() instead. |
   | After semaphore acquired | `EventSemaphoreAcquired` | After `g.semaphore <- struct{}{}` (~line 410) |
   | After HTTP response received | `EventHttpCompleted` | After `fn()` returns (~line 479) |
   | After 429 backoff set | `EventBackoffStarted` | After setting `g.backoffUntil` (~line 512) |
   | In deferred semaphore release | `EventSemaphoreReleased` | In `defer func() { <-g.semaphore }()` (~line 419) |

4. For Interactive requests that wait on backoff: emit `EventRequestWaited` before
   calling `waitForBackoff()`, not after.

**Files:**
- Modify: `internal/api/gateway.go` — replace all `RecordGatewayCall` calls with
  `emitEvent`/`emitEventLocked`, add new event emissions

**Tests:**

Use a mock `GatewayEventRecorder` that collects events into a slice:

- `TestGateway_Do_NormalRequest_EmitsLifecycle` — make a normal background GET,
  verify events: `RequestEntered → TokenConsumed → SemaphoreAcquired → HttpCompleted →
  SemaphoreReleased → RequestAllowed`
- `TestGateway_Do_BlockedRequest_EmitsBlockedEvent` — background request during backoff,
  verify: `RequestEntered → RequestBlocked`
- `TestGateway_Do_InteractiveWait_EmitsWaitedEvent` — interactive request during backoff,
  verify: `RequestEntered → RequestWaited → SemaphoreAcquired → ... → RequestAllowed`
- `TestGateway_Do_DedupRequest_EmitsJoinAndResolve` — second GET to same endpoint,
  verify: `RequestEntered → DedupJoined → DedupResolved`
- `TestGateway_Do_429Response_EmitsBackoffStarted` — request gets 429,
  verify `EventBackoffStarted` is emitted
- `TestGateway_Do_EventsHaveCorrectRequestID` — all events for the same request share
  the same `RequestID`
- `TestGateway_Do_EventsHaveSnapshots` — every emitted event has a non-zero
  `Snapshot.TokensMax` (proving snapshot was captured)
- `TestGateway_Do_SnapshotReflectsStateAtMoment` — after token consumption, the
  `EventTokenConsumed` snapshot shows `TokensAvailable < TokensMax`

**Commit:** `feat(api): instrument Gateway.Do() with lifecycle event emission`

---

## Task 3: Add `CheckAndEmitRefill()` and `CheckAndEmitBackoffExpiry()`

**Problem:** Token refills happen lazily (only computed when a request checks). Backoff
expiry happens when a timer passes. Neither produces events visible to the journal.
Without explicit emission, the UI replay can't show token recovery or backoff clearing.

**Fix:**

In `internal/api/gateway.go`:

1. Add tracking fields to `Gateway` struct:
   ```go
   // lastEmittedTokens tracks the token level of the last TokenRefilled event.
   // Used to avoid emitting duplicate refill events when tokens haven't changed.
   lastEmittedTokens int
   // lastBackoffActive tracks whether backoff was active at the last check.
   // Used to detect the backoff→clear transition for BackoffExpired events.
   lastBackoffActive bool
   ```

2. Add `CheckAndEmitRefill()`:
   ```go
   // CheckAndEmitRefill checks if the token bucket level has changed since the
   // last emission and emits EventTokenRefilled if so. Called by the app on
   // viz.TickMsg (every 200ms). Does NOT mutate bucket.tokens — the lazy refill
   // stays as-is for the hot path.
   func (g *Gateway) CheckAndEmitRefill() {
       g.bucket.mu.Lock()
       now := time.Now()
       elapsed := now.Sub(g.bucket.lastFill).Seconds()
       tokens := g.bucket.tokens + elapsed*g.bucket.rate
       if tokens > g.bucket.max {
           tokens = g.bucket.max
       }
       currentLevel := int(tokens)
       g.bucket.mu.Unlock()

       g.mu.Lock()
       changed := currentLevel != g.lastEmittedTokens
       rec := g.recorder
       g.lastEmittedTokens = currentLevel
       g.mu.Unlock()

       if changed && rec != nil {
           rec.RecordEvent(domain.GatewayEvent{
               Timestamp: time.Now(),
               Kind:      domain.EventTokenRefilled,
               Snapshot:  g.captureSnapshot(),
           })
       }
   }
   ```

3. Add `CheckAndEmitBackoffExpiry()`:
   ```go
   // CheckAndEmitBackoffExpiry checks if the 429 backoff period has just expired
   // and emits EventBackoffExpired on the active→cleared transition. Called by the
   // app on viz.TickMsg (every 200ms).
   func (g *Gateway) CheckAndEmitBackoffExpiry() {
       g.mu.Lock()
       nowActive := time.Now().Before(g.backoffUntil)
       wasActive := g.lastBackoffActive
       g.lastBackoffActive = nowActive
       rec := g.recorder
       g.mu.Unlock()

       if wasActive && !nowActive && rec != nil {
           rec.RecordEvent(domain.GatewayEvent{
               Timestamp: time.Now(),
               Kind:      domain.EventBackoffExpired,
               Snapshot:  g.captureSnapshot(),
           })
       }
   }
   ```

4. Initialize `lastEmittedTokens` in `NewGateway()`:
   ```go
   g.lastEmittedTokens = int(g.bucket.max)
   ```

**Files:**
- Modify: `internal/api/gateway.go` — add fields, `CheckAndEmitRefill()`,
  `CheckAndEmitBackoffExpiry()`, update `NewGateway()`

**Tests:**
- `TestGateway_CheckAndEmitRefill_EmitsOnChange` — consume tokens, call
  `CheckAndEmitRefill()`, verify `EventTokenRefilled` emitted with updated snapshot
- `TestGateway_CheckAndEmitRefill_NoEmitWhenStable` — call twice without changes,
  verify only one event emitted
- `TestGateway_CheckAndEmitBackoffExpiry_EmitsOnTransition` — set backoff, let it
  expire, call `CheckAndEmitBackoffExpiry()`, verify `EventBackoffExpired` emitted
- `TestGateway_CheckAndEmitBackoffExpiry_NoEmitWhenAlreadyClear` — call when no
  backoff active, verify no event emitted
- `TestGateway_CheckAndEmitRefill_NilRecorder` — no panic when recorder is nil

**Commit:** `feat(api): add periodic event emission for token refills and backoff expiry`

---

## Task 4: Wire event recorder in `app.go` and `auth.go`

**Problem:** The new `GatewayEventRecorder` interface needs to be wired the same way
as the old `GatewayRecorder`. The periodic check methods need to be called on
`viz.TickMsg`.

**Fix:**

1. In `internal/app/auth.go`, update `initAPIClients()`:
   ```go
   // Replace: a.gateway.SetRecorder(a.store)  (old GatewayRecorder)
   // With:    a.gateway.SetRecorder(a.store)  (new GatewayEventRecorder)
   // The method name stays the same, but the accepted interface type changes.
   ```
   Since `Store` now implements `domain.GatewayEventRecorder` (added in Feature 66),
   and `SetRecorder()` now accepts that interface (Task 1), no code change is needed
   beyond the type change in `SetRecorder()`.

2. In `internal/app/app.go`, update the `viz.TickMsg` handler to call periodic checks
   before forwarding to panes:
   ```go
   case viz.TickMsg:
       // Emit periodic gateway events (token refills, backoff expiry).
       a.gateway.CheckAndEmitRefill()
       a.gateway.CheckAndEmitBackoffExpiry()
       // Forward viz.TickMsg to panes (unchanged).
       ...
   ```

**Files:**
- Modify: `internal/app/auth.go` — update `SetRecorder` call type (may be no-op if
  method signature already updated in Task 1)
- Modify: `internal/app/app.go` — add `CheckAndEmitRefill()`/`CheckAndEmitBackoffExpiry()`
  calls in `viz.TickMsg` handler

**Tests:**
- Integration: verify `SetRecorder(store)` compiles (Store satisfies new interface)
- Integration: verify `viz.TickMsg` handler calls both periodic methods (mock gateway
  or verify events appear in event log after tick)

**Commit:** `feat(app): wire gateway event recorder and periodic emission in tick handler`

---

## Task 5: Retire old recording system

**Problem:** The old `GatewayRecorder` interface, `Snapshot()`, `ResetWatermarks()`,
watermark fields, and double-recording prevention are no longer needed. They should be
removed to avoid confusion and dead code.

**Fix:**

1. **Remove from `internal/api/gateway.go`:**
   - `GatewayRecorder` interface (replaced by `domain.GatewayEventRecorder`)
   - `Snapshot()` method (snapshots are embedded in events)
   - `ResetWatermarks()` method
   - `tokenBucket.minTokens` field and tracking in `wait()`
   - `Gateway.peakConcurrent` field and tracking in `Do()`
   - `Gateway.minTokensInit` field
   - `gatewayRecordedKey` context key type and value
   - `MarkGatewayRecorded()` function
   - `IsGatewayRecorded()` function

2. **Remove from `internal/api/logging.go`:**
   - The `IsGatewayRecorded(req)` check in `RoundTrip()`. Since the gateway is now the
     sole recorder, `LoggingTransport` no longer needs to skip recording. Remove
     `NetLogRecorder` interface too — `LoggingTransport` becomes a plain timing transport
     that doesn't record to any log.
   - Actually, since `LoggingTransport` was only recording to the old `NetLog`, and the
     gateway event journal replaces it entirely, `LoggingTransport` can be simplified to
     just pass through (or removed if no other code depends on it for timing).

3. **Remove from `internal/domain/gateway.go`:**
   - `GatewaySnapshotter` interface
   - `GatewayState` struct (replaced by `GatewayStateSnapshot`)
   - `GatewayState.PeakConcurrent` and `GatewayState.MinTokens` fields
   - `GatewayDecision` type and constants (replaced by `EventKind` — the event kinds
     `EventRequestAllowed`, `EventRequestBlocked`, etc. carry the same information)

4. **Remove from `internal/api/base.go`:**
   - `MarkGatewayRecorded(req)` call in `doJSON`/`doNoContent`/`doJSONOptional` — no
     longer needed since `LoggingTransport` doesn't record.

5. **Update all callers** — any code that references the removed types/methods needs
   updating. The Request Flow pane and Network Log pane will be updated in Features 68
   and 69 respectively, so this task must ensure that the removed code is not called
   by those panes yet. Since the panes still use the old system in this feature, the
   retirement should be done carefully:
   - **Strategy:** Remove the `GatewayRecorder` and `Snapshot()` from the gateway side.
     Keep `GatewaySnapshotter` and `GatewayState` in `domain/` temporarily as they're
     still referenced by the pane code. Mark them `// Deprecated: will be removed in
     Feature 68`. The pane will still compile because it reads `lastSnapshot` which is
     a `GatewayState` — but the field will never be updated (the gateway no longer has
     `Snapshot()`). This is acceptable because Feature 68 rewrites the pane to use the
     event journal.

   - **Alternative (safer):** Keep `Snapshot()` as a thin wrapper that builds a
     `GatewayState` from a `captureSnapshot()` call. This preserves pane compatibility
     until Feature 68. Remove watermark fields from `GatewayState` (set to 0). Add
     `// Deprecated` comment.

   We go with the **safer alternative** — keep `Snapshot()` as a compatibility shim
   until Feature 68 removes it.

**Files:**
- Modify: `internal/api/gateway.go` — remove watermarks, `ResetWatermarks()`,
  `GatewayRecorder`, double-recording helpers; keep `Snapshot()` as deprecated shim
- Modify: `internal/api/logging.go` — remove `IsGatewayRecorded` check and
  `NetLogRecorder` interface; simplify `LoggingTransport`
- Modify: `internal/api/base.go` — remove `MarkGatewayRecorded` calls
- Modify: `internal/domain/gateway.go` — mark `GatewayState` and `GatewaySnapshotter`
  as deprecated; remove `PeakConcurrent`/`MinTokens` from `GatewayState`

**Tests:**
- Update gateway tests that check `Snapshot()` watermark fields — remove those assertions
- Update gateway tests that check `ResetWatermarks()` — remove those tests
- Remove tests for `MarkGatewayRecorded`/`IsGatewayRecorded`
- Verify `Snapshot()` still returns valid (non-watermark) fields for pane compatibility
- All existing tests pass

**Commit:** `refactor(api): retire old GatewayRecorder, watermarks, and double-recording prevention`

---

## Task 6: Update documentation

**Fix:**

1. Update `docs/features/00-overview.md` — add Feature 67 row
2. Update `docs/ARCHITECTURE.md` — update Gateway section to describe event emission
   model, note deprecated `Snapshot()`/`GatewayState`

**Files:**
- Modify: `docs/features/00-overview.md`
- Modify: `docs/ARCHITECTURE.md`

**Commit:** `docs: add Feature 67 gateway event instrumentation`

---

## Acceptance Criteria

- [ ] `Gateway.Do()` emits `EventRequestEntered` at entry with unique `RequestID`
- [ ] `Gateway.Do()` emits `EventTokenConsumed` after token bucket wait
- [ ] `Gateway.Do()` emits `EventSemaphoreAcquired`/`EventSemaphoreReleased`
- [ ] `Gateway.Do()` emits `EventRequestBlocked` for background requests during backoff
- [ ] `Gateway.Do()` emits `EventRequestWaited` for interactive requests during backoff
- [ ] `Gateway.Do()` emits `EventDedupJoined`/`EventDedupResolved` for dedup
- [ ] `Gateway.Do()` emits `EventHttpCompleted` with status and latency
- [ ] `Gateway.Do()` emits `EventBackoffStarted` on 429
- [ ] All events carry correct `GatewayStateSnapshot` at the moment of emission
- [ ] All events for the same request share the same `RequestID`
- [ ] `CheckAndEmitRefill()` emits `EventTokenRefilled` when level changes
- [ ] `CheckAndEmitBackoffExpiry()` emits `EventBackoffExpired` on transition
- [ ] Old watermark fields removed from Gateway and tokenBucket
- [ ] `GatewayRecorder` interface removed, replaced by `GatewayEventRecorder`
- [ ] `LoggingTransport` no longer records to net log
- [ ] `MarkGatewayRecorded`/`IsGatewayRecorded` removed
- [ ] `Snapshot()` still works as a deprecated compatibility shim
- [ ] All existing tests pass (updated for removed watermarks)
- [ ] `make ci` passes

---

## Verification

```bash
# Event emission in Do()
grep 'emitEvent' internal/api/gateway.go | wc -l  # should be ~12+

# RequestID generation
grep 'nextRequestID' internal/api/gateway.go

# Periodic emission methods
grep 'CheckAndEmitRefill\|CheckAndEmitBackoffExpiry' internal/api/gateway.go

# Old recording removed
! grep 'GatewayRecorder' internal/api/gateway.go  # interface gone
! grep 'RecordGatewayCall' internal/api/gateway.go  # method calls gone
! grep 'minTokens' internal/api/gateway.go  # watermark gone
! grep 'peakConcurrent' internal/api/gateway.go  # watermark gone
! grep 'MarkGatewayRecorded' internal/api/base.go  # double-recording gone

# Deprecated Snapshot still compiles
grep 'Deprecated' internal/api/gateway.go

# Wired in app
grep 'CheckAndEmitRefill' internal/app/app.go

# Tests
go test ./internal/api/ -run 'Gateway_Do_.*Emit|CheckAndEmit' -v

# Full CI
make ci
```

---

## Implementation Notes for Agents

### Lock ordering is critical

`Do()` acquires `g.mu` at several points. `emitEvent()` also needs to read `g.recorder`
under `g.mu` and call `captureSnapshot()` which acquires `g.bucket.mu`. The rule:

- `emitEvent()` acquires `g.mu` then `g.bucket.mu` — safe, this is the standard order
- `emitEventLocked()` assumes `g.mu` is held, only acquires `g.bucket.mu` — for use
  inside locked sections of `Do()`
- `bucket.mu` is NEVER held when calling `emitEvent()` — token consumption in
  `bucket.wait()` releases `tb.mu` before returning

### Key files to read first

- `internal/api/gateway.go` — the file being heavily modified
- `internal/api/gateway_test.go` — existing tests to update
- `internal/domain/gateway.go` — new types from Feature 66
- `internal/state/eventlog.go` — storage from Feature 66
- `internal/app/app.go` — viz.TickMsg handler for wiring
- `internal/app/auth.go` — SetRecorder wiring

### Do NOT

- Remove `GatewayState` or `GatewaySnapshotter` from domain/ — Feature 68 needs them
  temporarily. Just mark deprecated.
- Change token bucket rate/capacity or semaphore capacity
- Modify the Request Flow pane or Network Log pane — that's Features 68/69
- Import new dependencies

---

*Depends on: Feature 66*
*Blocks: Feature 68*
