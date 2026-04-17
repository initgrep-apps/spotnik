---
title: "Gateway Event Instrumentation"
feature: 14-nerd-status
status: done
---

## Background
Feature 66 added domain types and storage for the event journal. This story wired the Gateway as the event producer, emitting fine-grained lifecycle events at every decision point in Do() and for internal state changes (token refills, backoff expiry). The old GatewayRecorder produced one summary record per request; this replacement emits ~5-6 events per request lifecycle, each carrying a snapshot of gateway state at the exact moment. The old GatewayRecorder, Snapshot(), ResetWatermarks(), watermark fields, and double-recording prevention were retired.

## Design

### emitEvent() Helper
Add nextRequestID atomic.Uint64. Change recorder field to domain.GatewayEventRecorder. Add captureSnapshot() and captureSnapshotLocked(). Lock ordering: g.mu then bucket.mu.

### Do() Instrumentation
EventRequestEntered at top. Background backoff -> EventRequestBlocked. Dedup waiter -> EventDedupJoined then EventDedupResolved. Token consumed -> EventTokenConsumed. Semaphore acquired/released. HTTP response -> EventHttpCompleted. 429 -> EventBackoffStarted.

### Periodic Events
CheckAndEmitRefill(): emits EventTokenRefilled when level changes. CheckAndEmitBackoffExpiry(): detects active-to-clear transition, emits EventBackoffExpired. Called by app on viz.TickMsg (200ms).

### Old System Retirement
Remove GatewayRecorder interface, ResetWatermarks(), watermark fields, gatewayRecordedKey, MarkGatewayRecorded/IsGatewayRecorded. Keep Snapshot() as deprecated shim.

## Acceptance Criteria
- [ ] Gateway.Do() emits lifecycle events at every decision point
- [ ] All events carry correct GatewayStateSnapshot
- [ ] All events for same request share same RequestID
- [ ] CheckAndEmitRefill() emits EventTokenRefilled on change
- [ ] CheckAndEmitBackoffExpiry() emits EventBackoffExpired on transition
- [ ] Old watermark fields removed
- [ ] GatewayRecorder interface removed
- [ ] LoggingTransport no longer records to net log
- [ ] Snapshot() still works as deprecated shim
- [ ] All existing tests pass (updated)
- [ ] make ci passes

## Tasks
- [ ] Add emitEvent() helper and captureSnapshot() to Gateway
      - test: correct token level with refill; correct ConcurrentActive; nil recorder no panic; emitEvent calls RecordEvent; nextRequestID increments
- [ ] Instrument Do() with lifecycle events
      - test: normal request emits full lifecycle; blocked emits blocked; interactive wait emits waited; dedup emits join+resolve; 429 emits backoff; correct RequestID; snapshots present and correct
- [ ] Add CheckAndEmitRefill() and CheckAndEmitBackoffExpiry()
      - test: refill emits on change; no emit when stable; backoff expiry on transition; no emit when clear; nil recorder
- [ ] Wire event recorder in app.go and auth.go
      - test: SetRecorder(store) compiles; viz.TickMsg calls periodic methods
- [ ] Retire old recording system
      - test: removed types no longer compile if referenced; deprecated Snapshot() still returns valid fields
- [ ] Update documentation
      - test: docs change only
