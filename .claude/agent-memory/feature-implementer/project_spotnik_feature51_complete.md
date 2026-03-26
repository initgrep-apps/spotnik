---
name: project_spotnik_feature51_complete
description: Feature 51 (Page B Nerd Status): Gateway.Snapshot(), RequestFlowPane, NetworkLogPane, Page B registration
type: project
---

## Feature 51 — Page B: Request Flow + Network Log

**What was built:**
- `Gateway.Snapshot()` in `internal/api/gateway.go` — reads token bucket (under bucket.mu), backoff/inflight/dedup (under gateway.mu), semaphore (len()/cap() of chan, no lock needed)
- `GatewayState` struct with TokensAvailable, TokensMax, ConcurrentActive, ConcurrentMax, BackoffRemaining, DedupWaiters, InFlightKeys
- `RequestFlowPane` in `internal/ui/panes/requestflow_pane.go` — 3-column APP/GATEWAY/SPOTIFY view, animated arrows (VisualizerTickMsg), token/semaphore bars, backoff timer
- `PollingSnapshotMsg` and `RequestCompletedMsg` in requestflow_pane.go (exported message types)
- `NetworkLogPane` in `internal/ui/panes/networklog_pane.go` — 6-column scrollable table, filter, latency bars
- Both panes registered in app.go New() with layout.PaneRequestFlow and layout.PaneNetworkLog

**Key files:**
- `internal/api/gateway.go` — GatewayState + Snapshot() after NewGateway()
- `internal/ui/panes/requestflow_pane.go` — RequestFlowPane, PollingSnapshotMsg, RequestCompletedMsg
- `internal/ui/panes/networklog_pane.go` — NetworkLogPane, latencyBar() helper
- `internal/app/app.go` — Page B pane creation + registration, RequestFlowPane()/NetworkLogPane() accessors, TickMsg/VisualizerTickMsg forwarding, PollingSnapshotMsg dispatch

**Patterns established:**
- `Snapshot()` reads bucket.mu then gateway.mu (never hold both simultaneously — token bucket has its own mutex)
- `len(semaphore)` / `cap(semaphore)` reads from chan are safe without mutex (atomic in Go runtime)
- PollingSnapshotMsg is sent AFTER TickMsg within the same tick handler block — the pane receives both in sequence (TickMsg refreshes snapshot, then PollingSnapshotMsg updates status strip)
- RequestFlowPane double-updates in TickMsg handler: first call gets pane from map, second gets updated pane (this is correct after first update writes back to map)
- `DedupWaiters` field name is a misnomer — it actually counts in-flight primary GET requests (not secondary waiters). Comment clarifies this.

**Gotchas:**
- `tea_keyMsg` and `tea_keyMsgRune` helpers needed to be defined inline in networklog_pane_test.go (no shared test helper file in panes package)
- requestflow_pane_test.go is in `panes_test` package (external test) — must use `panes.TickMsg{}` not `TickMsg{}`
- `padRight` and `truncateStr` are new helpers in requestflow_pane.go — `truncate()` already exists in search.go but with different semantics (no ellipsis). Not a conflict since different names
- `RequestCompletedMsg` is defined but NOT yet sent from app.go — the APP column will be empty until gateway response logging is wired. This is intentional forward-compatible design
- Page B layout (PageBPresets, PaneRequestFlow, PaneNetworkLog in layout package) was already defined by prior features — this feature only needed to create the panes and register them

**Testing notes:**
- Gateway.Snapshot() tests use time.Sleep(30ms) to let goroutines acquire semaphore — flaky-risk tests use GreaterOrEqual(snap.ConcurrentActive, 1) not Equal
- Race condition tests run with -race flag in make ci
- NetworkLogPane tests use store.NetLog().Add() directly (bypassing store.RecordAPICall)
- Coverage: 86.1% across all packages
