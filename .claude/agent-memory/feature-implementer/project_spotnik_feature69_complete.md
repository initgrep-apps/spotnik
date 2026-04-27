---
name: project_spotnik_feature69_complete
description: Feature 69 (Network Log Event Migration): NetLog retirement, cursor-based reads, PRIORITY/DECISION columns, blocked request visibility
type: project
---

## Feature 69 — Network Log Event Migration

**What was built:**
- Migrated `NetworkLogPane` from `NetLogEntries()` to cursor-based `ReadEventsFrom()` on `GatewayEventLog`
- Added `eventCursor uint64` + `completedRequests []networkLogRow` to `NetworkLogPane`
- Added PRIORITY (PRI) + DECISION columns to network log table (8 cols total)
- Blocked requests (`EventRequestBlocked`) now show status 0 + "✗ blocked" in NOTES
- Filter extended: matches "int"/"bg" (priority) + "allowed"/"blocked"/"waited"/"dedup" (decision)
- Deleted `internal/state/netlog.go`, `netlog_test.go`, `internal/api/logging.go`, `logging_test.go`
- Removed `RecordNetCall`, `NetLogEntries`, `NetLog` accessor, `netLog` field from `store.go`
- `auth.go` + `cmd/root.go` now use plain `http.Client` instead of `LoggingTransport`

**Key files:**
- `internal/ui/panes/networklog_pane.go` — full rewrite of `refreshRows()` + new `buildTableRows()` split
- `internal/ui/panes/networklog_pane_test.go` — full rewrite; 34 tests; new helpers `recordHttpCompleted`, `recordBlocked`
- `internal/state/store.go` — NetLog removed; eventLog sole recording mechanism

**Patterns established:**
- `refreshRows()` drains events (cursor advances), fills `completedRequests` buffer, calls `buildTableRows()`
- `buildTableRows()` called from `refreshRows()` (new events) + `handleKey()` (filter changes) — avoids re-draining on filter keystrokes
- Decision lookup: first pass builds `decisions map[uint64]EventKind`, second pass for HttpCompleted/Blocked
- `EventCursor()` + `CompletedRequestsLen()` exported as test accessors (matches `SelectedIndex()` pattern)

**Gotchas:**
- `NewNetworkLogPane()` calls `refreshRows()` in constructor → advances cursor to current sequence. Events added BEFORE construction drained immediately, so initial `View()` (no TickMsg) shows pre-existing events.
- `cmd/root.go` pre-auth startup path does NOT call `initAPIClients()` → gateway recorder never wired for already-authed users. Pre-existing architectural gap; comment corrected w/ NOTE.
- Stale `LoggingTransport` refs in `player.go` + `base.go` doc comments after deleting `logging.go` — caught in review, fixed.
- `gofmt` alignment: `NetworkLogPane` struct fields needed alignment (trailing spaces). Run `gofmt -w` before CI.
- `buildTableRows()` replacing `refreshRows()` in `handleKey()` intentional — filter changes must not re-drain event log.

**Testing notes:**
- `recordHttpCompleted` helper: records `EventRequestAllowed` then `EventHttpCompleted` w/ same RequestID — matches real gateway event ordering
- `recordBlocked` helper: records only `EventRequestBlocked` — no HTTP event, blocked requests never reach HTTP
- `TestNetworkLogPane_View_NewestFirst`: events added pre-construction work because constructor calls `refreshRows()` immediately
- 34 tests pass; `internal/ui/panes` coverage: 90.3%