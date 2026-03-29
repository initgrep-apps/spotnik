---
name: project_spotnik_feature69_complete
description: Feature 69 (Network Log Event Migration): NetLog retirement, cursor-based reads, PRIORITY/DECISION columns, blocked request visibility
type: project
---

## Feature 69 — Network Log Event Migration

**What was built:**
- Migrated `NetworkLogPane` from `NetLogEntries()` to cursor-based `ReadEventsFrom()` on `GatewayEventLog`
- Added `eventCursor uint64` and `completedRequests []networkLogRow` to `NetworkLogPane`
- Added PRIORITY (PRI) and DECISION columns to the network log table (8 columns total)
- Blocked requests (`EventRequestBlocked`) now appear with status 0 and "✗ blocked" in NOTES
- Extended filter to match on "int"/"bg" (priority) and "allowed"/"blocked"/"waited"/"dedup" (decision)
- Deleted `internal/state/netlog.go`, `netlog_test.go`, `internal/api/logging.go`, `logging_test.go`
- Removed `RecordNetCall`, `NetLogEntries`, `NetLog` accessor, `netLog` field from `store.go`
- Updated `auth.go` and `cmd/root.go` to use plain `http.Client` instead of `LoggingTransport`

**Key files:**
- `internal/ui/panes/networklog_pane.go` — complete rewrite of `refreshRows()` + new `buildTableRows()` split
- `internal/ui/panes/networklog_pane_test.go` — fully rewritten; 34 tests; new helpers `recordHttpCompleted`, `recordBlocked`
- `internal/state/store.go` — NetLog section removed; eventLog is the only recording mechanism

**Patterns established:**
- `refreshRows()` drains events (cursor advances), populates `completedRequests` buffer, then calls `buildTableRows()`
- `buildTableRows()` is called from both `refreshRows()` (new events) and `handleKey()` (filter changes) — avoids re-draining events on filter keystrokes
- Decision lookup: build `decisions map[uint64]EventKind` from first pass over events, then second pass for HttpCompleted/Blocked
- `EventCursor()` and `CompletedRequestsLen()` exported as test accessors (consistent with `SelectedIndex()` pattern)

**Gotchas:**
- `NewNetworkLogPane()` calls `refreshRows()` in the constructor, which advances the cursor to the current sequence. Events added BEFORE construction are drained immediately, so the initial `View()` call (no TickMsg needed) shows pre-existing events.
- The `cmd/root.go` pre-auth startup path does NOT call `initAPIClients()`, so gateway recorder is never wired for already-authenticated users. This is a pre-existing architectural gap — the comment was corrected to acknowledge it with a NOTE.
- Stale `LoggingTransport` references existed in `player.go` and `base.go` doc comments after deleting `logging.go` — caught in review and fixed.
- `gofmt` alignment: the `NetworkLogPane` struct fields needed alignment (trailing spaces issue). Always run `gofmt -w` before CI.
- `buildTableRows()` replacing `refreshRows()` call in `handleKey()` is intentional — filter changes should not re-drain the event log.

**Testing notes:**
- `recordHttpCompleted` helper: records `EventRequestAllowed` then `EventHttpCompleted` with same RequestID — matches real gateway event ordering
- `recordBlocked` helper: records only `EventRequestBlocked` — no HTTP event since blocked requests never reach HTTP
- `TestNetworkLogPane_View_NewestFirst`: events added before pane construction work because constructor calls `refreshRows()` immediately
- 34 tests pass; `internal/ui/panes` coverage: 90.3%
