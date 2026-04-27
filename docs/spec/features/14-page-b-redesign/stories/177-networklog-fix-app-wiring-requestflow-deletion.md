---
title: "NetworkLog Fix, App Wiring, and RequestFlow Deletion"
feature: 14-page-b-redesign
status: done
---

## Background

This is the final story in the Page B redesign. It fixes the `NetworkLogPane` decision
cross-tick bug, wires the three new panes into `app.go`/`handlers.go`, updates `border.go`
to handle the new PaneIDs, deletes the six `requestflow_*` source files, and runs the full
CI gate. `docs/DESIGN.md` is updated to reflect the new Page B layout.

**Source:** `docs/superpowers/plans/2026-04-26-page-b-redesign.md` Tasks 10, 12, 13.

**Depends on:** Story 176 (GatewayLivePane).

---

## Design

### Task 10 — Fix NetworkLogPane

**Files to modify:** `internal/ui/panes/networklog_pane.go`,
`internal/ui/panes/networklog_pane_test.go`.

**Bug:** `decisions` is currently a local map rebuilt on every tick. If `EventRequestAllowed`
arrives on tick N and `EventHttpCompleted` arrives on tick N+1, the decision map is empty
on tick N+1 and the Decision column shows blank.

**Fix:** Promote to a persistent struct field `pendingDecisions map[uint64]domain.EventKind`
initialized in the constructor. Accumulate decision events (`EventRequestAllowed`,
`EventRequestBlocked`, `EventDedupJoined`) into the map on every tick. Consume the entry
(and `delete()` it to prevent unbounded growth) when `EventHttpCompleted` is processed.

**Column headers:** Change from all-caps to Title Case:
`"TIME"` → `"Time"`, `"METHOD"` → `"Method"`, `"ENDPOINT"` → `"Endpoint"`,
`"STATUS"` → `"Status"`, `"LATENCY"` → `"Latency"`, `"PRIORITY"` → `"Priority"`,
`"DECISION"` → `"Decision"`.

**ToggleKey:** Update `ToggleKey()` to return `5` (spec assigns key 5 to Network Log on Page B).

Test: `TestNetworkLogPane_Decision_PersistedAcrossTicks` — emits `EventRequestAllowed`
(tick 1) then `EventHttpCompleted` (tick 2) for the same `RequestID`; asserts `view`
contains `"Allowed"`.

Also update any existing tests that assert on the old `ToggleKey()` value or uppercase header strings.

### Task 12 — Wire new panes in app.go, update border.go, delete RequestFlow files

**Files to modify:** `internal/app/app.go`, `internal/app/handlers.go`,
`internal/ui/layout/border.go`.

**Files to delete:** `internal/ui/panes/requestflow_pane.go`,
`internal/ui/panes/requestflow_pane_test.go`, `internal/ui/panes/requestflow_boxed.go`,
`internal/ui/panes/requestflow_boxed_test.go`, `internal/ui/panes/requestflow_replay.go`,
`internal/ui/panes/requestflow_replay_test.go`.

**border.go** — in `PaneBorderColor()` switch, remove `case PaneRequestFlow` and add:
```go
case PaneGatewayHealth, PanePollingTraffic, PaneGatewayLive:
    return t.PaneBorderRequestFlow()
```

**app.go** — in `New()` pane map initialization, replace `requestFlowPane` entry with:
```go
layout.PaneGatewayHealth:  panes.NewGatewayHealthPane(s, t),
layout.PanePollingTraffic: panes.NewPollingTrafficPane(s, t),
layout.PaneGatewayLive:    panes.NewGatewayLivePane(s, t),
```

Remove `RequestFlowPane()` accessor; add test accessors:
```go
func (a *App) GatewayHealthPane() *panes.GatewayHealthPane { ... }
func (a *App) PollingTrafficPane() *panes.PollingTrafficPane { ... }
func (a *App) GatewayLivePane() *panes.GatewayLivePane { ... }
```

**handlers.go** — replace the block that forwards `PollingSnapshotMsg` to `RequestFlowPane`
with:
```go
if ptp, ok := a.panes[layout.PanePollingTraffic]; ok {
    updated, _ := ptp.Update(pollingSnapshot)
    a.panes[layout.PanePollingTraffic] = updated
}
```

Remove all other blocks referencing `PaneRequestFlow` or `RequestFlowPane()`.

**File deletion:** `git rm` all six `requestflow_*` files. After deletion, `humanAge` and
`humanInterval` are gone. `cacheAge` (polling_traffic_pane.go) and `pollingHumanInterval`
(polling_traffic_pane.go) remain — do **not** rename them to `humanAge`.

Verify: `go build ./...` clean; `go test ./...` all green.

### Task 13 — Full CI gate and docs update

Run `make ci`. If coverage drops below 80%, identify uncovered lines via `make test-coverage`
and add focused tests for:
- `GatewayHealthPane.View()` Warning state paths (≤ 2 tokens, all slots full)
- `PollingTrafficPane.View()` `fetchedAt.IsZero()` path
- `GatewayLivePane.handleKey()` Esc with active filter, Esc with committed query, Esc bare

Update `docs/DESIGN.md` Page B section:
- Remove references to `RequestFlowPane`
- Document the new 4-pane layout (GatewayHealth, PollingTraffic, GatewayLive, NetworkLog)
- Add toggle key table for Page B (key 1 = NetworkLog, 2 = GatewayHealth, 3 = PollingTraffic,
  4 = GatewayLive, 5 = NetworkLog full pane)

Commit: `docs(design): update Page B spec — 4-pane layout, remove RequestFlowPane`

---

## Acceptance Criteria

- [ ] `NetworkLogPane.pendingDecisions` is a persistent struct field initialized in constructor
- [ ] `TestNetworkLogPane_Decision_PersistedAcrossTicks` passes
- [ ] NetworkLog column headers are Title Case; `ToggleKey()` returns 5
- [ ] `border.go` handles `PaneGatewayHealth`, `PanePollingTraffic`, `PaneGatewayLive` in its switch
- [ ] `app.go` wires all three new panes; `RequestFlowPane()` accessor removed
- [ ] `handlers.go` forwards `PollingSnapshotMsg` to `PanePollingTraffic`; no references to `PaneRequestFlow`
- [ ] All six `requestflow_*` files deleted; `go build ./...` clean
- [ ] `cacheAge` and `pollingHumanInterval` remain in `polling_traffic_pane.go`; no `humanAge` callers remain
- [ ] `docs/DESIGN.md` Page B section reflects new 4-pane layout; no references to `RequestFlowPane`
- [ ] `make ci` passes (lint + tests + 80% coverage)

## Tasks

- [ ] Fix `NetworkLogPane.pendingDecisions` cross-tick bug; update headers to Title Case; update `ToggleKey()` to 5
      - test: `TestNetworkLogPane_Decision_PersistedAcrossTicks` passes; all existing NetworkLog tests pass
- [ ] Update `border.go` switch to handle new PaneIDs
      - test: `go build ./...` clean
- [ ] Wire `GatewayHealthPane`, `PollingTrafficPane`, `GatewayLivePane` in `app.go` and `handlers.go`
      - test: `go build ./...` clean; app constructors compile
- [ ] Delete six `requestflow_*` files via `git rm`
      - test: `go build ./...` clean; `go test ./...` all green
- [ ] Run `make ci` and fix any coverage gaps
      - test: `make ci` passes (lint + tests + 80% coverage threshold)
- [ ] Update `docs/DESIGN.md` Page B section
      - test: content review — no `RequestFlowPane` references; new 4-pane layout documented
