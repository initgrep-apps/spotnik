---
title: "Network Log Event Migration"
feature: 14-nerd-status
status: done
---

## Background
The Network Log pane read from store.NetLogEntries() returning flat NetLogEntry records. After Features 66-68 moved all recording to the gateway event journal, the old NetLog was redundant -- it contained a subset of the event journal's data, and background requests blocked by backoff were invisible. This story migrated the Network Log pane to read from the GatewayEventLog via cursor-based reads, added PRIORITY and DECISION columns, made blocked requests visible, and retired the entire old NetLog system.

## Design

### Cursor-Based Event Reading
Add eventCursor field and networkLogRow struct. Add completedRequests []networkLogRow capped at 200. Rewrite refreshRows(): drain events, build RequestID-to-decision map, extract rows from EventHttpCompleted and EventRequestBlocked events.

### New Columns
PRI (FlexFactor 1): "int" or "bg". DECISION (FlexFactor 3): "allowed"/"blocked"/"waited"/"dedup". NOTES: blocked requests show "blocked" instead of latency bar.

### Filter Extension
filter.MatchesAny(path, statusStr, pri, dec) -- matches priority and decision values.

### NetLog Retirement
Delete internal/state/netlog.go. Remove netLog field, NewNetLog(), RecordNetCall(), RecordGatewayCall(), NetLogEntries(), NetLog() from Store. Remove NetLogRecorder from api/logging.go. Simplify or remove LoggingTransport. Delete netlog_test.go.

## Acceptance Criteria
- [ ] NetworkLogPane reads from GatewayEventLog via cursor-based ReadEventsFrom()
- [ ] EventHttpCompleted events appear as table rows
- [ ] EventRequestBlocked events appear as table rows (status 0)
- [ ] PRIORITY column shows "int" or "bg"
- [ ] DECISION column shows allowed/blocked/waited/dedup
- [ ] Filter matches priority and decision values
- [ ] NetLog struct and NetLogEntry type deleted
- [ ] RecordNetCall(), RecordGatewayCall(), NetLogEntries() removed from Store
- [ ] NetLogRecorder removed from api/
- [ ] All tests rewritten and passing
- [ ] make ci passes

## Tasks
- [ ] Add cursor-based event reading to NetworkLogPane
      - test: cursor advances; incremental drain; HttpCompleted appears; BlockedRequest appears (status 0); caps at 200
- [ ] Add PRIORITY and DECISION columns to table
      - test: PRI column shows "int"/"bg"; DECISION column values; blocked notes column
- [ ] Update filter to support new columns
      - test: filter matches priority "int"; matches decision "blocked"; matches endpoint preserved
- [ ] Retire NetLog, NetLogEntry, and related Store methods
      - test: removal compiles cleanly; all remaining tests pass
- [ ] Update existing NetworkLogPane tests for event journal
      - test: replace RecordNetCall with RecordEvent; rewrite table/filter/scroll/latency tests; add blocked/priority/decision/cursor tests
- [ ] Update documentation
      - test: docs change only
