# Feature 69 — Network Log Event Migration

> **Enhancement:** Migrate the Network Log pane from `NetLog`/`NetLogEntry` to the
> gateway event journal. Add PRIORITY and DECISION columns. Show blocked requests that
> were previously invisible. Retire `NetLog`, `NetLogEntry`, `RecordNetCall`,
> `RecordGatewayCall`, and `NetLogEntries`.

## Background

The Network Log pane currently reads from `store.NetLogEntries()` which returns
`[]NetLogEntry` — flat records with timestamp, method, path, status, and latency. This
was adequate when the only data source was `LoggingTransport`, but Feature 67 moved all
recording to the gateway event journal.

After Features 66–68:
- The `GatewayEventLog` is the authoritative event source
- The Request Flow pane already reads from it
- `NetLog` is redundant — it contains a subset of the event journal's data
- Background requests blocked by backoff (status 0, never reached HTTP) are recorded
  as events but invisible in the Network Log because `NetLogEntry` only has HTTP-level data

This feature migrates the Network Log pane to read from `GatewayEventLog` and retires
the old `NetLog` system entirely.

**Design spec:** `docs/superpowers/specs/2026-03-29-gateway-event-journal-design.md`

**Depends on:** Feature 68 (Request Flow Replay Engine)

---

## Gap Summary

| # | Gap | Severity | Description |
|---|-----|----------|-------------|
| G1 | Pane reads from old NetLog | Critical | Should read from GatewayEventLog with cursor-based reads |
| G2 | No PRIORITY column | Medium | Can't distinguish interactive vs background requests |
| G3 | No DECISION column | Medium | Can't see gateway decisions (allowed/blocked/deduped/waited) |
| G4 | Blocked requests invisible | High | Background requests rejected by backoff never appear in the table |
| G5 | Old NetLog system is dead code | Medium | NetLog, NetLogEntry, RecordNetCall, RecordGatewayCall are unreferenced |

---

## Task 1: Add cursor-based event reading to NetworkLogPane

**Problem:** `NetworkLogPane.refreshRows()` calls `p.store.NetLogEntries()` which
copies the entire `NetLog` ring buffer. The pane needs to use cursor-based reads from
the `GatewayEventLog` instead.

**Fix:**

In `internal/ui/panes/networklog_pane.go`:

1. Add `eventCursor uint64` field to `NetworkLogPane` struct.

2. Add a local buffer to accumulate completed request records:
   ```go
   type networkLogRow struct {
       timestamp  time.Time
       method     string
       path       string
       statusCode int
       durationMs int64
       priority   domain.RequestPriority
       decision   domain.EventKind
   }
   ```

3. Add `completedRequests []networkLogRow` field — stores extracted rows from events.
   Capped at a max (e.g., 200 to match the old NetLog capacity).

4. Rewrite `refreshRows()`:
   ```go
   func (p *NetworkLogPane) refreshRows() {
       // Drain new events since last cursor.
       newCursor, events := p.store.ReadEventsFrom(p.eventCursor)
       p.eventCursor = newCursor

       // Extract completed requests and blocked requests from new events.
       // Build a map of RequestID → decision kind for lookups.
       decisions := make(map[uint64]domain.EventKind)
       for _, e := range events {
           switch e.Kind {
           case domain.EventRequestAllowed, domain.EventRequestBlocked,
               domain.EventRequestWaited, domain.EventDedupJoined:
               decisions[e.RequestID] = e.Kind
           }
       }

       for _, e := range events {
           switch e.Kind {
           case domain.EventHttpCompleted:
               row := networkLogRow{
                   timestamp:  e.Timestamp,
                   method:     e.Method,
                   path:       e.Path,
                   statusCode: e.StatusCode,
                   durationMs: e.DurationMs,
                   priority:   e.Priority,
                   decision:   decisions[e.RequestID],
               }
               p.completedRequests = append(p.completedRequests, row)

           case domain.EventRequestBlocked:
               // Blocked requests never reach HTTP — show them with status 0.
               row := networkLogRow{
                   timestamp:  e.Timestamp,
                   method:     e.Method,
                   path:       e.Path,
                   statusCode: 0,
                   durationMs: 0,
                   priority:   e.Priority,
                   decision:   domain.EventRequestBlocked,
               }
               p.completedRequests = append(p.completedRequests, row)
           }
       }

       // Cap at max entries (newest kept).
       const maxRows = 200
       if len(p.completedRequests) > maxRows {
           p.completedRequests = p.completedRequests[len(p.completedRequests)-maxRows:]
       }

       // Build table rows (newest first).
       p.buildTableRows()
   }
   ```

5. Extract `buildTableRows()` from the current inline logic in `refreshRows()` to
   iterate `p.completedRequests` in reverse order, apply filter, and set table rows.

**Files:**
- Modify: `internal/ui/panes/networklog_pane.go` — add cursor, `networkLogRow`,
  `completedRequests`, rewrite `refreshRows()`

**Tests:**
- `TestNetworkLogPane_RefreshRows_CursorAdvances` — add events to store, call
  refreshRows, verify cursor advanced
- `TestNetworkLogPane_RefreshRows_IncrementalDrain` — add 3 events, refresh (get 3),
  add 2 more, refresh (get 2 more, total 5)
- `TestNetworkLogPane_RefreshRows_HttpCompletedAppearsInTable` — add `EventHttpCompleted`,
  refresh, verify row in table
- `TestNetworkLogPane_RefreshRows_BlockedRequestAppearsInTable` — add
  `EventRequestBlocked`, refresh, verify row appears with status "0"
- `TestNetworkLogPane_RefreshRows_CapsAt200` — add 250 events, verify only 200 kept

**Commit:** `feat(ui): migrate NetworkLogPane to cursor-based event log reads`

---

## Task 2: Add PRIORITY and DECISION columns

**Problem:** The table has 6 columns (TIME, METHOD, ENDPOINT, STATUS, LATENCY, NOTES).
The event journal provides priority and gateway decision data that was previously
unavailable.

**Fix:**

1. Add two new columns to the table definition in `NewNetworkLogPane()`:
   ```go
   columns := []components.ColumnDef{
       {Key: "time", Header: "TIME", FlexFactor: 3, Color: th.TextMuted()},
       {Key: "method", Header: "METHOD", FlexFactor: 2, Color: th.TextSecondary()},
       {Key: "endpoint", Header: "ENDPOINT", FlexFactor: 7, Color: th.TextPrimary()},
       {Key: "status", Header: "STATUS", FlexFactor: 2, Color: th.TextPrimary()},
       {Key: "latency", Header: "LATENCY", FlexFactor: 2, Color: th.TextMuted()},
       {Key: "priority", Header: "PRI", FlexFactor: 1, Color: th.TextMuted()},
       {Key: "decision", Header: "DECISION", FlexFactor: 3, Color: th.TextSecondary()},
       {Key: "notes", Header: "NOTES", FlexFactor: 3, Color: th.TextMuted()},
   }
   ```

2. In `buildTableRows()`, populate the new columns:
   ```go
   pri := "bg"
   if row.priority == domain.PriorityInteractive {
       pri = "int"
   }

   dec := ""
   switch row.decision {
   case domain.EventRequestAllowed:
       dec = "allowed"
   case domain.EventRequestBlocked:
       dec = "blocked"
   case domain.EventRequestWaited:
       dec = "waited"
   case domain.EventDedupJoined:
       dec = "dedup"
   }
   ```

3. Update the NOTES column to include a decision icon alongside the latency bar:
   ```go
   notes := latencyBar(row.durationMs)
   if row.statusCode == 429 {
       notes += " ⚠"
   }
   if row.decision == domain.EventRequestBlocked {
       notes = "✗ blocked"
   }
   ```

4. Reduce ENDPOINT flex factor from 8 to 7 to make room for the new columns.

**Files:**
- Modify: `internal/ui/panes/networklog_pane.go` — add columns, populate in
  `buildTableRows()`

**Tests:**
- `TestNetworkLogPane_View_ShowsPriorityColumn` — interactive event shows "int",
  background shows "bg"
- `TestNetworkLogPane_View_ShowsDecisionColumn` — blocked shows "blocked",
  allowed shows "allowed", dedup shows "dedup"
- `TestNetworkLogPane_View_BlockedNotesColumn` — blocked request shows "✗ blocked"
  instead of latency bar

**Commit:** `feat(ui): add PRIORITY and DECISION columns to Network Log pane`

---

## Task 3: Update filter to support new columns

**Problem:** The filter currently matches on endpoint path and status code. It should
also match on priority ("int", "bg") and decision ("blocked", "dedup", etc.) so users
can filter for specific request types.

**Fix:**

In `buildTableRows()`, add the new fields to the filter match:
```go
if query != "" {
    if !p.filter.MatchesAny(row.path, statusStr, pri, dec) {
        continue
    }
}
```

**Files:**
- Modify: `internal/ui/panes/networklog_pane.go` — update filter match in
  `buildTableRows()`

**Tests:**
- `TestNetworkLogPane_Filter_MatchesPriority` — filter "int", verify only interactive
  requests shown
- `TestNetworkLogPane_Filter_MatchesDecision` — filter "blocked", verify only blocked
  requests shown
- `TestNetworkLogPane_Filter_MatchesEndpoint` — existing behavior preserved

**Commit:** `feat(ui): extend Network Log filter to match priority and decision columns`

---

## Task 4: Retire `NetLog`, `NetLogEntry`, and related Store methods

**Problem:** `NetLog`, `NetLogEntry`, `RecordNetCall`, `RecordGatewayCall`, and
`NetLogEntries` are no longer referenced by any production code. They should be removed.

**Fix:**

1. Delete `internal/state/netlog.go` — the entire file.

2. Remove from `internal/state/store.go`:
   - `netLog *NetLog` field from Store struct
   - `NewNetLog()` call in `NewStore()`
   - `RecordNetCall()` method
   - `RecordGatewayCall()` method
   - `NetLogEntries()` method
   - `NetLog()` method

3. Remove from `internal/api/logging.go`:
   - `NetLogRecorder` interface
   - Recording logic in `LoggingTransport.RoundTrip()` — the transport can remain as a
     plain passthrough wrapper (it still provides request timing for debugging), or be
     removed entirely if nothing depends on it for non-logging purposes.
   - Check if `LoggingTransport` is still used in `auth.go` — if so, simplify it to
     just a passthrough.

4. Update `internal/app/auth.go`:
   - If `LoggingTransport` is removed, replace with `http.DefaultTransport` or a
     simpler wrapper.
   - If `LoggingTransport` is kept (as passthrough), no change needed.

5. Remove test files:
   - `internal/state/netlog_test.go` (if it exists)
   - Any test helper functions that create `NetLogEntry` values

**Files:**
- Delete: `internal/state/netlog.go`
- Modify: `internal/state/store.go` — remove `netLog` field and all NetLog methods
- Modify: `internal/api/logging.go` — remove `NetLogRecorder`, simplify transport
- Modify: `internal/app/auth.go` — update if `LoggingTransport` removed
- Delete or modify: `internal/state/netlog_test.go`

**Tests:**
- Verify removal compiles cleanly (no dangling references)
- All remaining tests pass
- `make ci` passes

**Commit:** `refactor(state): retire NetLog, NetLogEntry, and related recording methods`

---

## Task 5: Update existing NetworkLogPane tests

**Problem:** Existing tests reference `NetLogEntries()` and `NetLogEntry`. They need
to inject data via `store.RecordEvent()` instead.

**Fix:**

Rewrite all `NetworkLogPane` tests:

1. Replace test data injection:
   ```go
   // Old:
   store.RecordNetCall("GET", "/me/player", 200, 45)

   // New:
   store.RecordEvent(domain.GatewayEvent{
       Timestamp:  time.Now(),
       Kind:       domain.EventHttpCompleted,
       RequestID:  1,
       Method:     "GET",
       Path:       "/me/player",
       StatusCode: 200,
       DurationMs: 45,
       Priority:   domain.PriorityBackground,
       Snapshot:   domain.GatewayStateSnapshot{TokensMax: 10, ConcurrentMax: 5},
   })
   // Also inject the decision event for the same RequestID:
   store.RecordEvent(domain.GatewayEvent{
       Kind:      domain.EventRequestAllowed,
       RequestID: 1,
   })
   ```

2. Tests to rewrite:
   - Table rendering tests
   - Filter tests
   - Scroll tests
   - Latency bar tests (unchanged — `latencyBar()` is a pure function)

3. New tests:
   - Blocked request visibility
   - Priority column rendering
   - Decision column rendering
   - Cursor-based incremental reads

**Files:**
- Modify: `internal/ui/panes/networklog_pane_test.go` — rewrite all tests

**Commit:** `test(ui): rewrite Network Log tests for event journal migration`

---

## Task 6: Update documentation

**Fix:**

1. Update `docs/features/00-overview.md` — add Feature 69 row
2. Update `docs/ARCHITECTURE.md`:
   - Remove NetLog references
   - Note that GatewayEventLog is the single source of truth for both panes
   - Update Network Log pane description with new columns

**Files:**
- Modify: `docs/features/00-overview.md`
- Modify: `docs/ARCHITECTURE.md`

**Commit:** `docs: add Feature 69 Network Log event migration`

---

## Acceptance Criteria

- [ ] NetworkLogPane reads from `GatewayEventLog` via cursor-based `ReadEventsFrom()`
- [ ] `EventHttpCompleted` events appear as table rows
- [ ] `EventRequestBlocked` events appear as table rows (status 0, "✗ blocked")
- [ ] PRIORITY column shows "int" or "bg" for each request
- [ ] DECISION column shows "allowed", "blocked", "waited", or "dedup"
- [ ] Filter matches priority and decision values
- [ ] `NetLog` struct and `NetLogEntry` type are deleted
- [ ] `RecordNetCall()`, `RecordGatewayCall()`, `NetLogEntries()` are removed from Store
- [ ] `NetLogRecorder` interface is removed from api/
- [ ] `LoggingTransport` simplified or removed
- [ ] No dangling references to removed types
- [ ] All tests rewritten and passing
- [ ] `make ci` passes

---

## Verification

```bash
# Cursor-based reads in pane
grep 'eventCursor\|ReadEventsFrom' internal/ui/panes/networklog_pane.go

# New columns exist
grep 'priority.*PRIORITY\|decision.*DECISION' internal/ui/panes/networklog_pane.go

# Blocked requests visible
grep 'EventRequestBlocked' internal/ui/panes/networklog_pane.go

# Old NetLog removed
! test -f internal/state/netlog.go
! grep 'NetLogEntry\|RecordNetCall\|RecordGatewayCall' internal/state/store.go
! grep 'NetLogRecorder' internal/api/logging.go

# Tests
go test ./internal/ui/panes/ -run 'NetworkLog' -v

# Full CI
make ci
```

---

## Implementation Notes for Agents

### Key files to read first
- `internal/ui/panes/networklog_pane.go` — the file being modified (~233 lines)
- `internal/ui/panes/networklog_pane_test.go` — existing tests to rewrite
- `internal/state/eventlog.go` — GatewayEventLog API (from Feature 66)
- `internal/state/netlog.go` — file to delete
- `internal/state/store.go` — methods to remove
- `internal/api/logging.go` — LoggingTransport to simplify

### Patterns to follow
- Use `components.Table` for rendering (same as current)
- Use `components.Filter` for filtering (same as current)
- Cursor-based reads: keep `eventCursor` on the pane, call `ReadEventsFrom(cursor)` in
  `refreshRows()`, only called on `TickMsg` (1s, not 200ms)

### Do NOT
- Change the table component or filter component
- Import `api/` from the pane
- Change the refresh cadence (stays at 1s via TickMsg)
- Remove `latencyBar()` function (still used for NOTES column)

---

*Depends on: Feature 68*
*Blocks: Nothing*
