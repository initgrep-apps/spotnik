---
title: "DevTools panes component tests"
feature: 21-test-infrastructure
status: open
---

## Background

The Stats page (Feature 10) hosts four diagnostic panes: GatewayHealth (4 health rows with
colored status bars), PollingTraffic (playback cadence + cache freshness), GatewayLive
(500-entry reverse-chronological request buffer), and NetworkLog (scrollable request log).
Current tests verify data rendering logic but never snapshot the full pane output.

## Design

### Golden tests: `internal/ui/panes/gateway_health_golden_test.go`

- `TestGatewayHealthPane_View_AllHealthy` — all 4 rows green/healthy
- `TestGatewayHealthPane_View_MixedHealth` — rate limit warning, auth error, healthy

### Golden tests: `internal/ui/panes/polling_traffic_golden_test.go`

- `TestPollingTrafficPane_View_Fresh` — cache fresh, all intervals normal
- `TestPollingTrafficPane_View_Stale` — library cache stale, warning shown

### Golden tests: `internal/ui/panes/gateway_live_golden_test.go`

- `TestGatewayLivePane_View_WithEntries` — 10 recent requests shown, reverse-chronological
- `TestGatewayLivePane_View_Empty` — no requests yet

### Golden tests: `internal/ui/panes/networklog_golden_test.go`

- `TestNetworkLogPane_View_WithEntries` — 5 requests shown with timestamps
- `TestNetworkLogPane_View_Empty` — no requests

## Files

### Create

- `internal/ui/panes/gateway_health_golden_test.go`
- `internal/ui/panes/polling_traffic_golden_test.go`
- `internal/ui/panes/gateway_live_golden_test.go`
- `internal/ui/panes/networklog_golden_test.go`
- `internal/ui/panes/testdata/TestGatewayHealthPane_View_*.golden` (2 files)
- `internal/ui/panes/testdata/TestPollingTrafficPane_View_*.golden` (2 files)
- `internal/ui/panes/testdata/TestGatewayLivePane_View_*.golden` (2 files)
- `internal/ui/panes/testdata/TestNetworkLogPane_View_*.golden` (2 files)

## Acceptance Criteria

- [ ] GatewayHealth: 2 golden snapshots (healthy, mixed)
- [ ] PollingTraffic: 2 golden snapshots (fresh, stale)
- [ ] GatewayLive: 2 golden snapshots (entries, empty)
- [ ] NetworkLog: 2 golden snapshots (entries, empty)
- [ ] `make ci` passes

## Tasks

- [ ] Create GatewayHealthPane golden tests (2 snapshots)
      - test: `TestGatewayHealthPane_View_AllHealthy`, `TestGatewayHealthPane_View_MixedHealth`
- [ ] Create PollingTrafficPane golden tests (2 snapshots)
      - test: `TestPollingTrafficPane_View_Fresh`, `TestPollingTrafficPane_View_Stale`
- [ ] Create GatewayLivePane golden tests (2 snapshots)
      - test: `TestGatewayLivePane_View_WithEntries`, `TestGatewayLivePane_View_Empty`
- [ ] Create NetworkLogPane golden tests (2 snapshots)
      - test: `TestNetworkLogPane_View_WithEntries`, `TestNetworkLogPane_View_Empty`
- [ ] Generate golden files and verify all tests pass
