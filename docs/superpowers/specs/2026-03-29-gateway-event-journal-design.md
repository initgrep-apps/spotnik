# Gateway Event Journal — Design Spec

> **Problem:** The Request Flow pane tries to observe gateway state that changes in
> microseconds by polling snapshots at 200ms–1s intervals. No polling frequency can
> capture sub-millisecond token consumption, semaphore acquisition, or dedup decisions.
> Features 64–65 added watermarks as a workaround, but these are summary numbers — not
> a story of what happened.
>
> **Solution:** The Gateway records every internal decision as a timestamped event with
> a state snapshot. Events are stored in a ring buffer. The UI replays them at
> human-observable speed (200ms minimum per event), creating a slow-motion time machine
> of gateway activity.

---

## Architecture Overview

```
Gateway.Do() ──→ emitEvent() ──→ GatewayEventLog.Add()
                                        │
                    ┌───────────────────┤
                    ▼                    ▼
          RequestFlowPane         NetworkLogPane
          (200ms replay)          (1s table refresh)
          cursor-based read       cursor-based read
          replay queue            filter HttpCompleted
          displayState            table rows
```

The Gateway is the sole event producer. Both UI panes are consumers that read from the
same ring buffer using independent cursors. No polling of live gateway state. No sampling.
No watermarks.

---

## 1. Gateway Event Types

All types live in `internal/domain/`.

### EventKind Enum

```go
type EventKind int

const (
    EventRequestEntered     EventKind = iota // request arrived at gateway
    EventTokenConsumed                       // background request consumed a token
    EventTokenRefilled                       // tokens recovered (periodic)
    EventSemaphoreAcquired                   // concurrency slot taken
    EventSemaphoreReleased                   // concurrency slot freed
    EventBackoffStarted                      // 429 received, entering backoff
    EventBackoffExpired                      // backoff timer cleared
    EventRequestAllowed                      // request passed through normally
    EventRequestWaited                       // interactive request waited on backoff
    EventRequestBlocked                      // background request rejected by backoff
    EventDedupJoined                         // GET joined existing in-flight request
    EventDedupResolved                       // dedup waiters received shared response
    EventHttpCompleted                       // HTTP response received
)
```

### GatewayStateSnapshot

Renamed/trimmed from current `GatewayState`. No watermark fields.

```go
type GatewayStateSnapshot struct {
    TokensAvailable  int
    TokensMax        int
    ConcurrentActive int
    ConcurrentMax    int
    BackoffRemaining float64
    DedupWaiters     int
    InFlightKeys     []string
}
```

### GatewayEvent

```go
type GatewayEvent struct {
    Timestamp  time.Time
    Kind       EventKind
    RequestID  uint64              // links events for the same request (0 for internal events)
    Method     string              // GET, PUT, etc. (empty for internal events)
    Path       string              // /me/player (empty for internal events)
    Priority   RequestPriority     // Interactive vs Background
    StatusCode int                 // only on HttpCompleted
    DurationMs int64               // only on HttpCompleted
    Snapshot   GatewayStateSnapshot // gateway state at this exact moment
}
```

### GatewayEventRecorder Interface

```go
type GatewayEventRecorder interface {
    RecordEvent(event GatewayEvent)
}
```

Replaces the current `GatewayRecorder` interface. Implemented by `*state.Store`.

### What Gets Retired from domain/

- `GatewayState` struct — replaced by `GatewayStateSnapshot`
- `GatewaySnapshotter` interface — no more live snapshot polling
- `GatewayState.PeakConcurrent` and `GatewayState.MinTokens` — no more watermarks

---

## 2. GatewayEventLog (Storage)

New ring buffer in `internal/state/`. Replaces `NetLog`.

### Structure

```go
type GatewayEventLog struct {
    mu       sync.RWMutex
    entries  []domain.GatewayEvent
    head     int      // next write position
    count    int      // entries stored (max capacity)
    sequence uint64   // monotonically increasing, incremented on each Add()
}
```

**Capacity:** 500 events. A typical request produces ~5 events, so this holds ~100
requests worth of history.

### API

```go
// NewGatewayEventLog creates a new event log with the given capacity.
func NewGatewayEventLog(capacity int) *GatewayEventLog

// Add appends an event, overwriting the oldest if full.
func (l *GatewayEventLog) Add(event domain.GatewayEvent)

// ReadFrom returns events added since the given cursor.
// Returns the new cursor and the slice of new events.
// First call uses cursor=0.
func (l *GatewayEventLog) ReadFrom(cursor uint64) (uint64, []domain.GatewayEvent)

// Len returns the number of events stored.
func (l *GatewayEventLog) Len() int
```

**Cursor-based reads:** Each consumer keeps its own cursor (a sequence number). `ReadFrom`
returns only events the consumer hasn't seen. No full-buffer copies. The cursor is
monotonically increasing (not a ring index), so it handles wraparound correctly.

### Store Integration

```go
// On Store:
func (s *Store) RecordEvent(event domain.GatewayEvent)        // implements GatewayEventRecorder
func (s *Store) ReadEventsFrom(cursor uint64) (uint64, []domain.GatewayEvent)
```

### What Gets Retired from state/

- `NetLog` struct and `NetLogEntry` type (`netlog.go`)
- `Store.RecordNetCall()`
- `Store.RecordGatewayCall()`
- `Store.NetLogEntries()`

---

## 3. Gateway Instrumentation

How the Gateway emits events.

### Request ID Generation

Each `Do()` call gets a unique ID from an atomic counter:

```go
type Gateway struct {
    // ... existing fields ...
    nextRequestID atomic.Uint64
    recorder      GatewayEventRecorder
}
```

### Emission Helper

```go
func (g *Gateway) emitEvent(kind domain.EventKind, reqID uint64, method, path string,
    priority domain.RequestPriority, statusCode int, durationMs int64) {
    rec := g.recorder // read under existing mu where applicable
    if rec == nil { return }
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
```

`captureSnapshot()` reads current token level (with refill), `len(semaphore)`,
`backoffRemaining`, inflight keys. Same two-lock pattern as the current `Snapshot()`.

**Lock ordering:** `emitEvent()` is called from inside `Do()`, which sometimes holds
`g.mu` (e.g., after the backoff check). `captureSnapshot()` must accept an optional
"already holding mu" flag or split into `captureSnapshotLocked()` (reads only
`bucket.mu` + channel len, skips `g.mu` reads) and `captureSnapshot()` (acquires both).
The bucket mutex is never held when `emitEvent()` is called (token consumption happens
inside `bucket.wait()` which releases `tb.mu` before returning), so `bucket.mu` is
always safe to acquire. Only `g.mu` has re-entrancy risk.

### Emission Points in Do()

| Phase | Event Emitted |
|---|---|
| Entry (top of Do) | `EventRequestEntered` |
| Background backoff rejection | `EventRequestBlocked` |
| Interactive backoff wait starts | `EventRequestWaited` |
| `bucket.wait()` succeeds (`tb.tokens--`) | `EventTokenConsumed` |
| Semaphore acquired (`g.semaphore <- struct{}{}`) | `EventSemaphoreAcquired` |
| Dedup waiter joins | `EventDedupJoined` |
| Dedup waiter receives response | `EventDedupResolved` (with status) |
| HTTP response received | `EventHttpCompleted` (with status, latency) |
| 429 backoff set | `EventBackoffStarted` (with retry-after) |
| Semaphore released (deferred) | `EventSemaphoreReleased` |
| Final success (primary caller) | `EventRequestAllowed` |

### Request Lifecycle Examples

**Normal background GET:**
```
EventRequestEntered      "GET /me/player, bg"       snap: 10/10 tok, 0/5 conc
EventTokenConsumed       "token used"               snap:  9/10 tok, 0/5 conc
EventSemaphoreAcquired   "slot taken"               snap:  9/10 tok, 1/5 conc
EventHttpCompleted       "200, 45ms"                snap:  9/10 tok, 1/5 conc
EventSemaphoreReleased   "slot freed"               snap:  9/10 tok, 0/5 conc
EventRequestAllowed      "allowed"                  snap:  9/10 tok, 0/5 conc
```

**Blocked background request:**
```
EventRequestEntered      "GET /me/player, bg"       snap: 10/10 tok, 0/5 conc, backoff 2.1s
EventRequestBlocked      "rejected by backoff"      snap: 10/10 tok, 0/5 conc, backoff 2.1s
```

**Interactive request during backoff:**
```
EventRequestEntered      "PUT /me/player/play, int" snap: 10/10 tok, 0/5 conc, backoff 1.5s
EventRequestWaited       "waiting on backoff"       snap: 10/10 tok, 0/5 conc, backoff 1.5s
EventSemaphoreAcquired   "slot taken"               snap: 10/10 tok, 1/5 conc, backoff 0.0s
EventHttpCompleted       "204, 30ms"                snap: 10/10 tok, 1/5 conc
EventSemaphoreReleased   "slot freed"               snap: 10/10 tok, 0/5 conc
EventRequestAllowed      "allowed (waited)"         snap: 10/10 tok, 0/5 conc
```

**GET dedup:**
```
EventRequestEntered      "GET /me/player, bg"       snap:  9/10 tok, 1/5 conc
EventDedupJoined         "joined existing GET"      snap:  9/10 tok, 1/5 conc
EventDedupResolved       "shared response 200"      snap:  9/10 tok, 1/5 conc
```

### Periodic Internal Events

Two gateway-internal events happen independently of requests.

**Token refill:** `Gateway.CheckAndEmitRefill()`
- Called by the app on `viz.TickMsg` (every 200ms)
- Computes refilled token level (same lazy math, does not mutate `bucket.tokens`)
- If level changed since last emitted value, emits `EventTokenRefilled`
- Tracks `lastEmittedTokens` on the Gateway to avoid duplicate emissions
- Idle example: `TokenRefilled snap: 7/10` → `TokenRefilled snap: 9/10` → `TokenRefilled snap: 10/10` → (stable, no more)

**Backoff expiry:** `Gateway.CheckAndEmitBackoffExpiry()`
- Called alongside refill check on `viz.TickMsg`
- If `backoffUntil` transitioned from future to past since last check, emits `EventBackoffExpired`
- Tracks `lastBackoffActive bool` on the Gateway

Both are lightweight (one lock, one comparison) and only emit when a recorder is attached.

**Wiring in `app.go`:** On `viz.TickMsg`, before forwarding to panes:
```go
a.gateway.CheckAndEmitRefill()
a.gateway.CheckAndEmitBackoffExpiry()
```

### What Gets Retired from api/

- `GatewayRecorder` interface — replaced by `GatewayEventRecorder` (in `domain/`)
- `Gateway.Snapshot()` method — snapshots are embedded in events
- `Gateway.ResetWatermarks()` method
- `tokenBucket.minTokens` field — no watermark tracking
- `Gateway.peakConcurrent` field — no watermark tracking
- `Gateway.minTokensInit` field
- `gatewayRecordedKey` context key
- `MarkGatewayRecorded()` / `IsGatewayRecorded()` helpers
- `LoggingTransport.RoundTrip()` recording logic (transport can stay as a wrapper, just no net log writes)

---

## 4. Request Flow Pane Replay Engine

The core behavioral change. The pane no longer polls live gateway state. It consumes
events from the journal and replays them at human-observable speed.

### New Fields on RequestFlowPane

```go
type RequestFlowPane struct {
    theme        theme.Theme
    store        *state.Store          // reads event log
    focused      bool
    width, height int

    frameIndex   int                   // animation frame counter (200ms)

    eventCursor  uint64                // cursor into GatewayEventLog
    replayQueue  []domain.GatewayEvent // events waiting to be displayed
    displayState replayDisplayState    // what View() renders from

    pollingState PollingSnapshotMsg    // app-level polling snapshot
}
```

The pane has no gateway reference — it only reads events from the Store. The periodic
gateway calls (`CheckAndEmitRefill()`/`CheckAndEmitBackoffExpiry()`) are wired in
`app.go`, preserving the `ui/ never imports api/` rule.

### Display State (Render Model)

```go
type replayDisplayState struct {
    snapshot  domain.GatewayStateSnapshot    // from most recently replayed event
    requests  map[uint64]*requestAnimation   // active requests keyed by RequestID
    decisions []decisionEntry                // decision log for GATEWAY box
}

type animationPhase int

const (
    PhaseEntered    animationPhase = iota // appeared in APP box
    PhaseAtGateway                        // gateway decision rendered
    PhaseInFlight                         // HTTP call in progress
    PhaseCompleted                        // response received
    PhaseDone                             // aged out, ready for removal
)

type requestAnimation struct {
    requestID   uint64
    method      string
    path        string
    priority    domain.RequestPriority
    phase       animationPhase
    decision    domain.EventKind       // Allowed/Blocked/Waited/Deduped
    statusCode  int
    durationMs  int64
    enteredAt   time.Time              // when it appeared in APP box
}

type decisionEntry struct {
    kind    domain.EventKind
    label   string        // "✓ GET /player allowed", "↻ refilled → 10", etc.
    shownAt time.Time
}
```

### Replay Loop (on viz.TickMsg, Every 200ms)

```
1. Drain: cursor, newEvents = store.ReadEventsFrom(eventCursor)
         eventCursor = cursor
         append newEvents to replayQueue

2. Pop: if replayQueue is non-empty, pop the next event

3. Process the popped event:
   a. Update displayState.snapshot to event's Snapshot
   b. For request events (RequestID > 0):
      - EventRequestEntered:     create requestAnimation in PhaseEntered
      - EventTokenConsumed:      (no phase change, logged in decisions)
      - EventSemaphoreAcquired:  advance to PhaseAtGateway
      - EventDedupJoined:        advance to PhaseAtGateway, set decision
      - EventHttpCompleted:      advance to PhaseInFlight, set status/latency
      - EventSemaphoreReleased:  (no phase change, logged in decisions)
      - EventRequestAllowed:     advance to PhaseCompleted
      - EventRequestWaited:      set decision, advance to PhaseAtGateway
      - EventRequestBlocked:     set decision, advance to PhaseCompleted (no HTTP)
      - EventDedupResolved:      advance to PhaseCompleted, set status
   c. For internal events (RequestID == 0):
      - EventTokenRefilled:      (just snapshot + decision log)
      - EventBackoffStarted:     (just snapshot + decision log)
      - EventBackoffExpired:     (just snapshot + decision log)
   d. Append a decisionEntry to displayState.decisions

4. Age out:
   - Decisions older than 3s: remove from decisions list
   - Requests in PhaseCompleted for > 5s: move to PhaseDone, then remove

5. Advance frameIndex for arrow animation
```

**One event per tick = 200ms minimum visibility.** If 10 events queue up in a burst,
they replay over 10 ticks (2 seconds). The queue absorbs bursts naturally.

**Staggered parallel requests:** Multiple requests animate concurrently at different
phases. Row 1 might be `PhaseCompleted` while row 2 is `PhaseAtGateway`. The GATEWAY
box snapshot always reflects the most recently replayed event's state.

### How Animation Phases Map to the Three Boxes

| Phase | APP box | Left arrow | GATEWAY box | Right arrow | SPOTIFY box |
|---|---|---|---|---|---|
| Entered | Row appears | empty | snapshot updates | empty | empty |
| AtGateway | Row stays | decision arrow animates | decision annotation appears | empty | empty |
| InFlight | Row stays | decision arrow solid | snapshot updates | arrow animates → | empty |
| Completed | Row dims | decision arrow solid | snapshot updates | arrow solid | Status + latency |
| Done | Row removed | | | | Row removed |

### What View() Reads

Pure function. Reads only from `displayState`:
- `displayState.snapshot` → GATEWAY state bars
- `displayState.decisions` → GATEWAY decision log
- `displayState.requests` → APP rows, arrow states, SPOTIFY rows
- `p.frameIndex` → arrow animation frame
- `p.pollingState` → status strip

No `Snapshot()` calls. No `syncFromNetLog()`. No live gateway queries.

### What Gets Retired from RequestFlowPane

- `gateway domain.GatewaySnapshotter` field — no more live snapshot polling
- `lastSnapshot domain.GatewayState` field — replaced by `displayState.snapshot`
- `recentReqs []reqDisplay` — replaced by `displayState.requests`
- `syncFromNetLog()` method — replaced by cursor-based event drain
- `RequestCompletedMsg` type and handler — dead code, fully removed
- All watermark-related code from Features 64/65

---

## 5. GATEWAY Box Visual Design

Two sections: state bars at top, scrolling decision log below.

```
╭─ GATEWAY ──────────────────────╮
│ tokens  ●●●●●●●○○○ 7/10       │  state bars (from event snapshot)
│ conc    ■■□□□□□□□□  2/5        │
│ ⏳ backoff  2.1s               │  only when active
│                                │
│  ↻ tokens refilled → 10       │  decision log (scrolling feed)
│  → GET /player entered [bg]   │
│  ⊖ token consumed → 9        │
│  ⊞ semaphore acquired (1/5)   │
│  ✓ GET /player allowed        │
│  ⊟ semaphore released (0/5)   │
│  → GET /queue entered [bg]    │
│  ✗ GET /queue blocked [bg]    │
│  ⧖ GET /player dedup [bg]     │
╰────────────────────────────────╯
```

### Decision Log Icons

| Icon | Event Kind | Format |
|---|---|---|
| `→` | RequestEntered | `→ METHOD /path entered [int\|bg]` |
| `⊖` | TokenConsumed | `⊖ token consumed → N` |
| `↻` | TokenRefilled | `↻ tokens refilled → N` |
| `⊞` | SemaphoreAcquired | `⊞ semaphore acquired (N/M)` |
| `⊟` | SemaphoreReleased | `⊟ semaphore released (N/M)` |
| `⏳` | BackoffStarted | `⏳ backoff started Ns` |
| `✓` | BackoffExpired | `✓ backoff cleared` |
| `✓` | RequestAllowed | `✓ METHOD /path allowed` |
| `⧖` | RequestWaited | `⧖ METHOD /path waited` |
| `✗` | RequestBlocked | `✗ METHOD /path blocked` |
| `⧖` | DedupJoined | `⧖ METHOD /path dedup` |
| `✓` | DedupResolved | `✓ dedup resolved STATUS` |
| `✓` | HttpCompleted | `✓ STATUS LATENCYms` |

### Color Coding (Theme Tokens)

| Element | Token |
|---|---|
| `→` enter (interactive) | `TextPrimary()` |
| `→` enter (background) | `TextMuted()` |
| `✓` allowed/completed/expired | `Success()` |
| `✗` blocked | `Error()` |
| `⧖` waited/dedup | `Warning()` |
| `⊖`/`⊞`/`⊟` resource events | `TextSecondary()` |
| `↻` refill | `TextMuted()` |
| `[int]` tag | `TextPrimary()` |
| `[bg]` tag | `TextMuted()` |

### Scrolling Behavior

New entries appear at the bottom of the decision log. When the log fills the available
vertical space (box height minus state bars), oldest entries scroll off the top. Entries
older than 3 seconds are removed.

---

## 6. Network Log Pane Migration

The Network Log pane reads from the same `GatewayEventLog` using its own cursor.

### Changes

- Own `eventCursor uint64` field, reads on `TickMsg` (1s)
- Filters for `EventHttpCompleted` events to build table rows
- Also includes `EventRequestBlocked` events (blocked requests become visible)
- Finds gateway decision per request by scanning for the decision event with the same
  `RequestID` in a local buffer of recent events

### New Columns Available

| Column | Source | Notes |
|---|---|---|
| TIME | `event.Timestamp` | Same as today |
| METHOD | `event.Method` | Same as today |
| ENDPOINT | `event.Path` | Same as today |
| STATUS | `event.StatusCode` | Same as today |
| LATENCY | `event.DurationMs` | Same as today |
| PRIORITY | `event.Priority` | NEW — `int` or `bg` |
| DECISION | Lookup by `RequestID` | NEW — allowed/blocked/deduped/waited |
| NOTES | Latency bar + decision icon | Richer than just `█` blocks |

### What Gets Retired

- `store.NetLogEntries()` call
- Direct `NetLogEntry` usage

---

## 7. Removal Summary

### Retired from `state/`
- `NetLog` struct and `NetLogEntry` type (`netlog.go`)
- `Store.RecordNetCall()`
- `Store.RecordGatewayCall()`
- `Store.NetLogEntries()`

### Retired from `api/`
- `GatewayRecorder` interface
- `Gateway.Snapshot()` method
- `Gateway.ResetWatermarks()` method
- `tokenBucket.minTokens` field
- `Gateway.peakConcurrent` field
- `Gateway.minTokensInit` field
- `gatewayRecordedKey` context key
- `MarkGatewayRecorded()` / `IsGatewayRecorded()` helpers
- `LoggingTransport.RoundTrip()` recording logic

### Retired from `domain/`
- `GatewayState` struct — replaced by `GatewayStateSnapshot`
- `GatewaySnapshotter` interface
- `GatewayState.PeakConcurrent` / `GatewayState.MinTokens` fields

### Retired from `ui/panes/`
- `RequestFlowPane.syncFromNetLog()`
- `RequestFlowPane.lastSnapshot`
- `RequestFlowPane.recentReqs`
- `RequestCompletedMsg` type and handler
- All watermark-related code from Features 64/65

### Unchanged
- Three-box layout (`renderSubBox()`, boxed/flat switching, defensive guards)
- Arrow rendering helpers (`renderArrow()`, `renderRightArrow()`)
- Theme color coding
- Status strip (polling state + staleness)
- Tick intervals (200ms viz, 1s app)
- All keybindings and pane lifecycle
- `LoggingTransport` as an HTTP transport wrapper (just no recording logic)

---

## 8. Testing Strategy

### New Tests

- `GatewayEventLog`: Add/ReadFrom cursor semantics, wraparound, concurrent access
- `Gateway` event emission: verify each decision point emits the correct EventKind with
  correct snapshot values
- `CheckAndEmitRefill`: tokens change → event emitted; stable → no event
- `CheckAndEmitBackoffExpiry`: transition detection
- `RequestFlowPane` replay: inject events, verify displayState phases advance correctly
- `RequestFlowPane` View(): verify three-box output reflects displayState
- `NetworkLogPane`: verify cursor-based read, HttpCompleted filtering, blocked request visibility

### Updated Tests

- All existing `RequestFlowPane` tests that use `syncFromNetLog()` or `RequestCompletedMsg`
  → rewrite to inject events via `GatewayEventLog`
- All existing `NetworkLogPane` tests that use `NetLogEntries()` → rewrite to use event log
- All existing gateway tests that check `Snapshot()`/`ResetWatermarks()` → remove or adapt
- Mock `GatewaySnapshotter` in test files → replace with mock `GatewayEventRecorder`

### Coverage

- 80% minimum enforced by `make ci` (existing gate)
- Table-driven tests following existing codebase convention
- `httptest.NewServer` for gateway integration tests
