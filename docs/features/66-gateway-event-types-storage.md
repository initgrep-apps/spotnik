# Feature 66 — Gateway Event Types & Storage

> **Foundation:** Add the domain types and ring buffer storage for the gateway event
> journal. This is the data layer that Features 67–69 build upon. No behavioral changes
> — the gateway still records via the old `GatewayRecorder` interface until Feature 67
> replaces it.

## Background

The Request Flow pane currently observes gateway state by polling `Snapshot()` at
200ms–1s intervals. This fails because gateway decisions (token consumption, semaphore
acquisition, dedup) happen in microseconds and self-heal before the next sample.

The design spec (`docs/superpowers/specs/2026-03-29-gateway-event-journal-design.md`)
describes replacing snapshot polling with an **event journal**: the Gateway records every
internal decision as a timestamped event with a state snapshot. The UI replays these
events at human-observable speed.

This feature creates the foundational types and storage. It does not modify the Gateway,
the Request Flow pane, or the Network Log pane — those are Features 67, 68, and 69.

**Depends on:** Feature 65 (Gateway-Internal Watermarks)

---

## Gap Summary

| # | Gap | Severity | Description |
|---|-----|----------|-------------|
| G1 | No event type system | Critical | No `EventKind` enum, no `GatewayEvent` struct, no `GatewayStateSnapshot` |
| G2 | No event storage | Critical | No ring buffer for gateway events with cursor-based reads |
| G3 | No recorder interface | Critical | No `GatewayEventRecorder` interface for the gateway to write events |

---

## Task 1: Add `EventKind` enum to `internal/domain/gateway.go`

**Problem:** There is no type system for classifying gateway events. The current
`GatewayDecision` enum only covers 4 request outcomes. The event journal needs 13
distinct event kinds covering request lifecycle, resource tracking, and internal
housekeeping.

**Fix:**

Add `EventKind` type and constants to `internal/domain/gateway.go`:

```go
// EventKind classifies a gateway event for the event journal.
type EventKind int

const (
    // EventRequestEntered means a request arrived at the gateway.
    EventRequestEntered EventKind = iota
    // EventTokenConsumed means a background request consumed a token bucket token.
    EventTokenConsumed
    // EventTokenRefilled means tokens recovered (periodic internal event).
    EventTokenRefilled
    // EventSemaphoreAcquired means a request acquired a concurrency semaphore slot.
    EventSemaphoreAcquired
    // EventSemaphoreReleased means a request released its concurrency semaphore slot.
    EventSemaphoreReleased
    // EventBackoffStarted means the gateway received a 429 and entered backoff mode.
    EventBackoffStarted
    // EventBackoffExpired means the 429 backoff period cleared.
    EventBackoffExpired
    // EventRequestAllowed means a request passed through the gateway normally.
    EventRequestAllowed
    // EventRequestWaited means an interactive request waited on the backoff timer.
    EventRequestWaited
    // EventRequestBlocked means a background request was rejected by active backoff.
    EventRequestBlocked
    // EventDedupJoined means a GET request joined an existing in-flight GET (dedup).
    EventDedupJoined
    // EventDedupResolved means dedup waiters received the shared response.
    EventDedupResolved
    // EventHttpCompleted means an HTTP response was received from Spotify.
    EventHttpCompleted
)
```

**Files:**
- Modify: `internal/domain/gateway.go` — add `EventKind` type and 13 constants

**Tests:**
- Unit: Verify all 13 `EventKind` constants have distinct values (table-driven)
- Unit: Verify `EventRequestEntered` is the zero value (iota starts at 0)

**Commit:** `feat(domain): add EventKind enum for gateway event journal`

---

## Task 2: Add `GatewayStateSnapshot` struct

**Problem:** The event journal needs to embed a snapshot of gateway state with each
event. The existing `GatewayState` struct includes watermark fields (`PeakConcurrent`,
`MinTokens`) that are specific to the old polling approach and will be retired in
Feature 67.

**Fix:**

Add `GatewayStateSnapshot` to `internal/domain/gateway.go`:

```go
// GatewayStateSnapshot holds a frozen copy of gateway internal state at a specific
// moment in time. Embedded in GatewayEvent so the UI can replay state transitions
// without polling. Unlike GatewayState, this has no watermark fields — watermarks
// are replaced by the event journal itself.
type GatewayStateSnapshot struct {
    // TokensAvailable is the token bucket level at this moment (0–10).
    TokensAvailable int
    // TokensMax is the token bucket capacity (always 10).
    TokensMax int
    // ConcurrentActive is the number of in-flight requests holding semaphore slots.
    ConcurrentActive int
    // ConcurrentMax is the semaphore capacity (always 5).
    ConcurrentMax int
    // BackoffRemaining is seconds until the 429 backoff clears (0 if not throttled).
    BackoffRemaining float64
    // DedupWaiters is the number of in-flight GET requests in the dedup map.
    DedupWaiters int
    // InFlightKeys lists string descriptions of currently in-flight GET requests.
    InFlightKeys []string
}
```

This struct exists alongside `GatewayState` for now. `GatewayState` will be retired in
Feature 67 when the old `Snapshot()` method is removed.

**Files:**
- Modify: `internal/domain/gateway.go` — add `GatewayStateSnapshot` struct

**Tests:**
- Unit: `GatewayStateSnapshot` zero value has expected defaults (0 tokens, 0 concurrent)

**Commit:** `feat(domain): add GatewayStateSnapshot for event journal`

---

## Task 3: Add `GatewayEvent` struct and `GatewayEventRecorder` interface

**Problem:** There is no event struct to carry gateway decisions with their state
snapshots, and no interface for the gateway to write events to.

**Fix:**

Add to `internal/domain/gateway.go`:

```go
// GatewayEvent records a single gateway lifecycle event with a snapshot of the
// gateway's state at the exact moment the event occurred. Events are stored in a
// ring buffer and replayed by the UI at human-observable speed.
//
// For request-scoped events, RequestID links all events belonging to the same
// request. For internal events (TokenRefilled, BackoffExpired), RequestID is 0.
type GatewayEvent struct {
    // Timestamp is when the event occurred.
    Timestamp time.Time
    // Kind classifies the event.
    Kind EventKind
    // RequestID links events for the same request (0 for internal events).
    RequestID uint64
    // Method is the HTTP method (empty for internal events).
    Method string
    // Path is the API path, e.g. "/me/player" (empty for internal events).
    Path string
    // Priority is Interactive or Background (zero value for internal events).
    Priority RequestPriority
    // StatusCode is the HTTP response status (only set on EventHttpCompleted).
    StatusCode int
    // DurationMs is the HTTP round-trip time (only set on EventHttpCompleted).
    DurationMs int64
    // Snapshot is the gateway's state at this exact moment.
    Snapshot GatewayStateSnapshot
}

// GatewayEventRecorder records gateway lifecycle events.
// Implemented by *state.Store.
type GatewayEventRecorder interface {
    RecordEvent(event GatewayEvent)
}
```

**Files:**
- Modify: `internal/domain/gateway.go` — add `GatewayEvent` struct and
  `GatewayEventRecorder` interface

**Tests:**
- Unit: `GatewayEvent` with all fields populated round-trips correctly (no data loss)
- Unit: `GatewayEvent` zero value has `EventRequestEntered` kind and zero `RequestID`

**Commit:** `feat(domain): add GatewayEvent struct and GatewayEventRecorder interface`

---

## Task 4: Add `GatewayEventLog` ring buffer to `internal/state/`

**Problem:** There is no storage for gateway events. The existing `NetLog` ring buffer
stores flat `NetLogEntry` records with no event kind, no request ID, and no state
snapshot. A new ring buffer is needed with cursor-based reads so multiple consumers
(Request Flow pane, Network Log pane) can independently track their position.

**Fix:**

Create `internal/state/eventlog.go`:

```go
package state

import (
    "sync"

    "github.com/initgrep-apps/spotnik/internal/domain"
)

const defaultEventLogCapacity = 500

// GatewayEventLog is a fixed-size ring buffer of GatewayEvent values with
// cursor-based reads. Multiple consumers can independently track their
// position using monotonically increasing sequence numbers.
//
// Thread-safe: Add() takes a write lock, ReadFrom() takes a read lock.
type GatewayEventLog struct {
    mu       sync.RWMutex
    entries  []domain.GatewayEvent
    head     int    // next write position in the ring
    count    int    // entries stored (max capacity)
    sequence uint64 // monotonically increasing, incremented on each Add()
}

// NewGatewayEventLog creates an event log with the given capacity.
func NewGatewayEventLog(capacity int) *GatewayEventLog {
    if capacity <= 0 {
        capacity = defaultEventLogCapacity
    }
    return &GatewayEventLog{
        entries: make([]domain.GatewayEvent, capacity),
    }
}

// Add appends an event to the ring buffer, overwriting the oldest if full.
func (l *GatewayEventLog) Add(event domain.GatewayEvent) {
    l.mu.Lock()
    defer l.mu.Unlock()
    l.entries[l.head] = event
    l.head = (l.head + 1) % len(l.entries)
    if l.count < len(l.entries) {
        l.count++
    }
    l.sequence++
}

// ReadFrom returns events added since the given cursor position.
// Returns the new cursor and the slice of new events.
// First call should use cursor=0.
//
// If the cursor is older than the oldest retained event (due to ring buffer
// wraparound), all currently stored events are returned.
func (l *GatewayEventLog) ReadFrom(cursor uint64) (uint64, []domain.GatewayEvent) {
    l.mu.RLock()
    defer l.mu.RUnlock()

    if l.count == 0 || cursor >= l.sequence {
        return l.sequence, nil
    }

    // How many events have been added since the cursor?
    behind := l.sequence - cursor
    if behind > uint64(l.count) {
        // Cursor is too old — some events were overwritten. Return all stored.
        behind = uint64(l.count)
    }

    result := make([]domain.GatewayEvent, behind)
    // Start position in the ring: head points to the next write slot,
    // so the oldest stored entry is at (head - count) mod capacity.
    // The first event we want is at (head - behind) mod capacity.
    cap := len(l.entries)
    start := (l.head - int(behind) + cap) % cap
    for i := range result {
        result[i] = l.entries[(start+i)%cap]
    }

    return l.sequence, result
}

// Len returns the number of events currently stored.
func (l *GatewayEventLog) Len() int {
    l.mu.RLock()
    defer l.mu.RUnlock()
    return l.count
}
```

**Files:**
- Create: `internal/state/eventlog.go` — `GatewayEventLog` with `Add()`, `ReadFrom()`, `Len()`

**Tests:**

Create `internal/state/eventlog_test.go` with table-driven tests:

- `TestGatewayEventLog_Add_IncrementsCounts` — add 3 events, verify `Len()` returns 3
- `TestGatewayEventLog_Add_RingWraparound` — add more than capacity, verify `Len()` caps
  at capacity and oldest entries are overwritten
- `TestGatewayEventLog_ReadFrom_FirstCall` — cursor=0 returns all stored events
- `TestGatewayEventLog_ReadFrom_IncrementalReads` — add 3, read from 0 (get 3), add 2
  more, read from returned cursor (get 2)
- `TestGatewayEventLog_ReadFrom_CursorUpToDate` — cursor equals sequence → returns nil
- `TestGatewayEventLog_ReadFrom_CursorTooOld` — cursor older than oldest retained event
  → returns all stored (graceful recovery)
- `TestGatewayEventLog_ReadFrom_EventOrdering` — returned events are in insertion order
- `TestGatewayEventLog_ReadFrom_IndependentCursors` — two consumers with different cursors
  get correct subsets
- `TestGatewayEventLog_Add_ZeroCapacity` — capacity=0 defaults to 500
- `TestGatewayEventLog_ConcurrentAccess` — goroutine safety: concurrent Add + ReadFrom

**Commit:** `feat(state): add GatewayEventLog ring buffer with cursor-based reads`

---

## Task 5: Add `RecordEvent()` and `ReadEventsFrom()` to Store

**Problem:** The Store needs methods to write and read gateway events. It currently
has `RecordNetCall()`, `RecordGatewayCall()`, and `NetLogEntries()` for the old
`NetLog`. The new methods will coexist with the old ones until Feature 69 retires
`NetLog`.

**Fix:**

In `internal/state/store.go`:

1. Add `eventLog *GatewayEventLog` field to the Store struct.

2. Initialize in `NewStore()`:
   ```go
   eventLog: NewGatewayEventLog(defaultEventLogCapacity),
   ```

3. Add methods:
   ```go
   // RecordEvent records a gateway lifecycle event. Implements domain.GatewayEventRecorder.
   func (s *Store) RecordEvent(event domain.GatewayEvent) {
       s.eventLog.Add(event)
   }

   // ReadEventsFrom returns gateway events added since the given cursor.
   // Returns the new cursor and the slice of new events.
   func (s *Store) ReadEventsFrom(cursor uint64) (uint64, []domain.GatewayEvent) {
       return s.eventLog.ReadFrom(cursor)
   }
   ```

4. Verify Store satisfies `domain.GatewayEventRecorder`:
   ```go
   var _ domain.GatewayEventRecorder = &Store{}
   ```

**Files:**
- Modify: `internal/state/store.go` — add `eventLog` field, `RecordEvent()`,
  `ReadEventsFrom()`, compile-time interface check

**Tests:**
- Unit: `Store.RecordEvent()` stores event retrievable via `ReadEventsFrom(0)`
- Unit: `Store` satisfies `domain.GatewayEventRecorder` interface (compile-time check)
- Unit: `ReadEventsFrom()` returns incremental events with correct cursor advancement

**Commit:** `feat(state): add RecordEvent and ReadEventsFrom to Store`

---

## Task 6: Update documentation

**Fix:**

1. Update `docs/features/00-overview.md` — add Feature 66 row
2. Update `docs/ARCHITECTURE.md` — add a note in the State Management section about
   the `GatewayEventLog` alongside `NetLog` (both coexist until Feature 69)

**Files:**
- Modify: `docs/features/00-overview.md`
- Modify: `docs/ARCHITECTURE.md`

**Commit:** `docs: add Feature 66 gateway event types and storage`

---

## Acceptance Criteria

- [ ] `EventKind` enum exists with 13 constants in `domain/gateway.go`
- [ ] `GatewayStateSnapshot` struct exists in `domain/gateway.go`
- [ ] `GatewayEvent` struct exists with `Timestamp`, `Kind`, `RequestID`, `Snapshot` etc.
- [ ] `GatewayEventRecorder` interface exists with single `RecordEvent()` method
- [ ] `GatewayEventLog` ring buffer exists in `state/eventlog.go` with `Add()`, `ReadFrom()`, `Len()`
- [ ] Cursor-based reads work correctly (incremental, wraparound, stale cursor recovery)
- [ ] `Store.RecordEvent()` and `Store.ReadEventsFrom()` work correctly
- [ ] `Store` satisfies `domain.GatewayEventRecorder` at compile time
- [ ] All existing tests pass unchanged (no behavioral changes in this feature)
- [ ] `make ci` passes

---

## Verification

```bash
# Event types exist
grep 'EventKind' internal/domain/gateway.go
grep 'EventRequestEntered' internal/domain/gateway.go
grep 'GatewayStateSnapshot' internal/domain/gateway.go
grep 'GatewayEvent' internal/domain/gateway.go
grep 'GatewayEventRecorder' internal/domain/gateway.go

# Event log exists
grep 'GatewayEventLog' internal/state/eventlog.go
grep 'ReadFrom' internal/state/eventlog.go

# Store integration
grep 'RecordEvent' internal/state/store.go
grep 'ReadEventsFrom' internal/state/store.go

# Tests pass
go test ./internal/domain/ -run 'EventKind' -v
go test ./internal/state/ -run 'GatewayEventLog|RecordEvent' -v

# Full CI
make ci
```

---

*Depends on: Feature 65*
*Blocks: Feature 67*
