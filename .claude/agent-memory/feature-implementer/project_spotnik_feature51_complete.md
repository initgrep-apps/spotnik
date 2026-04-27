---
name: project_spotnik_feature51_complete
description: Feature 51 (Page B Nerd Status): Gateway.Snapshot(), RequestFlowPane, NetworkLogPane, Page B registration
type: project
---

## Feature 51 — Page B: Request Flow + Network Log

**Built:**
- `Gateway.Snapshot()` in `internal/api/gateway.go` — reads token bucket (under bucket.mu), backoff/inflight/dedup (under gateway.mu), semaphore (len()/cap() of chan, no lock)
- `GatewayState` struct: TokensAvailable, TokensMax, ConcurrentActive, ConcurrentMax, BackoffRemaining, DedupWaiters, InFlightKeys
- `RequestFlowPane` in `internal/ui/panes/requestflow_pane.go` — 3-col APP/GATEWAY/SPOTIFY view, animated arrows (VisualizerTickMsg), token/semaphore bars, backoff timer
- `PollingSnapshotMsg` + `RequestCompletedMsg` in requestflow_pane.go (exported msg types)
- `NetworkLogPane` in `internal/ui/panes/networklog_pane.go` — 6-col scrollable table, filter, latency bars
- Both panes registered in app.go New() w/ layout.PaneRequestFlow + layout.PaneNetworkLog

**Files:**
- `internal/api/gateway.go` — GatewayState + Snapshot() after NewGateway()
- `internal/ui/panes/requestflow_pane.go` — RequestFlowPane, PollingSnapshotMsg, RequestCompletedMsg
- `internal/ui/panes/networklog_pane.go` — NetworkLogPane, latencyBar() helper
- `internal/app/app.go` — Page B pane create+register, RequestFlowPane()/NetworkLogPane() accessors, TickMsg/VisualizerTickMsg forward, PollingSnapshotMsg dispatch

**Patterns:**
- `Snapshot()` reads bucket.mu then gateway.mu (never both at once — token bucket has own mutex)
- `len(semaphore)` / `cap(semaphore)` chan reads safe sans mutex (atomic Go runtime)
- PollingSnapshotMsg sent AFTER TickMsg within same tick handler — pane gets both in sequence (TickMsg refreshes snapshot, PollingSnapshotMsg updates status strip)
- RequestFlowPane double-updates in TickMsg handler: 1st call gets pane from map, 2nd gets updated pane (correct — 1st update writes back to map)
- `DedupWaiters` name misnomer — counts in-flight primary GET reqs (not secondary waiters). Comment clarifies.

**Gotchas:**
- `tea_keyMsg` + `tea_keyMsgRune` helpers defined inline in networklog_pane_test.go (no shared test helper file in panes pkg)
- requestflow_pane_test.go in `panes_test` pkg (external test) — must use `panes.TickMsg{}` not `TickMsg{}`
- `padRight` + `truncateStr` new helpers in requestflow_pane.go — `truncate()` exists in search.go but diff semantics (no ellipsis). No conflict, diff names
- `RequestCompletedMsg` defined but NOT yet sent from app.go — APP col empty until gateway response logging wired. Intentional fwd-compat design
- Page B layout (PageBPresets, PaneRequestFlow, PaneNetworkLog in layout pkg) already defined by prior features — this feature only created panes + registered

**Testing:**
- Gateway.Snapshot() tests use time.Sleep(30ms) to let goroutines acquire semaphore — flaky-risk tests use GreaterOrEqual(snap.ConcurrentActive, 1) not Equal
- Race tests run w/ -race in make ci
- NetworkLogPane tests use store.NetLog().Add() direct (bypass store.RecordAPICall)
- Coverage: 86.1% all pkgs