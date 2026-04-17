---
title: "Gateway Event Types & Storage"
feature: 14-nerd-status
status: done
---

## Background
The Request Flow pane observed gateway state by polling Snapshot() at 200ms-1s intervals, which failed because gateway decisions happen in microseconds and self-heal before the next sample. The design spec described replacing snapshot polling with an event journal. This story created the foundational domain types and ring buffer storage for the gateway event journal -- no behavioral changes to the Gateway, Request Flow pane, or Network Log pane.

## Design

### EventKind Enum (13 constants)
EventRequestEntered, EventTokenConsumed, EventTokenRefilled, EventSemaphoreAcquired, EventSemaphoreReleased, EventBackoffStarted, EventBackoffExpired, EventRequestAllowed, EventRequestWaited, EventRequestBlocked, EventDedupJoined, EventDedupResolved, EventHttpCompleted.

### GatewayStateSnapshot
Holds frozen copy of gateway internal state: TokensAvailable, TokensMax, ConcurrentActive, ConcurrentMax, BackoffRemaining, DedupWaiters, InFlightKeys. Exists alongside GatewayState temporarily.

### GatewayEvent
Carries Timestamp, Kind, RequestID (uint64), Method, Path, Priority, StatusCode, DurationMs, and embedded GatewayStateSnapshot. GatewayEventRecorder interface with single RecordEvent() method.

### GatewayEventLog Ring Buffer
Fixed-size (capacity 500) with Add() (write lock), ReadFrom(cursor) (read lock, cursor-based incremental reads), Len(). Monotonically increasing sequence numbers. Stale cursor returns all stored events.

### Store Integration
Store.RecordEvent() and Store.ReadEventsFrom() delegate to GatewayEventLog. Store satisfies domain.GatewayEventRecorder at compile time.

## Acceptance Criteria
- [ ] EventKind enum with 13 constants
- [ ] GatewayStateSnapshot struct
- [ ] GatewayEvent struct with all fields
- [ ] GatewayEventRecorder interface with RecordEvent()
- [ ] GatewayEventLog ring buffer with Add(), ReadFrom(), Len()
- [ ] Cursor-based reads work correctly (incremental, wraparound, stale recovery)
- [ ] Store satisfies GatewayEventRecorder at compile time
- [ ] All existing tests pass unchanged
- [ ] make ci passes

## Tasks
- [ ] Add EventKind enum to internal/domain/gateway.go -- 13 event kinds
      - test: all 13 constants have distinct values; EventRequestEntered is zero value
- [ ] Add GatewayStateSnapshot struct to domain/gateway.go
      - test: zero value has expected defaults
- [ ] Add GatewayEvent struct and GatewayEventRecorder interface
      - test: GatewayEvent round-trips correctly; zero value has expected kind and RequestID
- [ ] Add GatewayEventLog ring buffer to internal/state/eventlog.go
      - test: Add increments; ring wraparound; ReadFrom first call/incremental/up-to-date/too-old/ordering/independent cursors; zero capacity defaults; concurrent access
- [ ] Add RecordEvent() and ReadEventsFrom() to Store
      - test: Store.RecordEvent stores retrievable event; satisfies GatewayEventRecorder; incremental reads
- [ ] Update documentation
      - test: docs change only
