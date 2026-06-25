---
title: "Developer Tools (Stats Page)"
status: done
stories: 51, 56, 61–69, 109–113, 173–182, 210, 211
---

## Description

The Stats page (`0` key) surfaces the internals of Spotnik's API layer for developers. The original RequestFlow pane visualizes each gateway decision in real time — showing request type, priority, dedup result, rate-limit status, and backoff state — with a replay engine for stepping through past events. The NetworkLog pane is a scrollable table of every API request with timestamp, method, endpoint, HTTP status, priority classification, and gateway decision. Developer foundations stories add onboarding docs, test infrastructure, StateReader interface, BasePane pattern, and RebuildTableTheme helper.

**Page B Redesign (stories 173–182, 210–211, absorbed from feature 14):** Replaced the monolithic RequestFlowPane with three focused diagnostic panes (GatewayHealth, PollingTraffic, GatewayLive). Established universal Esc scroll-reset across all table panes. Introduced TableBasedPane to consolidate duplicated filter routing. Added universal filter border with Esc-clear and graded label shrink.

## Stories

| # | Story | File |
|---|---|---|
| 51 | RequestFlow pane | `stories/51-request-flow-pane.md` |
| 56 | Fix request flow data | `stories/56-fix-request-flow-data.md` |
| 61 | NetworkLog pane | `stories/61-network-log-pane.md` |
| 62 | Gateway events pane | `stories/62-gateway-events-pane.md` |
| 63 | RequestFlow boxed guards | `stories/63-requestflow-boxed-guards.md` |
| 64 | Page B integration | `stories/64-page-b-integration.md` |
| 66 | RequestFlow replay | `stories/66-request-flow-replay.md` |
| 67 | RequestFlow boxed | `stories/67-request-flow-boxed.md` |
| 68 | Gateway event journal | `stories/68-gateway-event-journal.md` |
| 69 | NetworkLog polish | `stories/69-network-log-polish.md` |
| 109 | Onboarding docs + test infra | `stories/109-onboarding-docs-test-infra.md` |
| 110 | StateReader + file splits | `stories/110-statereader-file-splits-table-tests.md` |
| 111 | BasePane + TableHelper + auth fix | `stories/111-basepane-tablehelper-auth-fix.md` |
| 112 | Test coverage gaps | `stories/112-test-coverage-gaps.md` |
| 113 | StateReader cleanup + nil guard | `stories/113-statereader-cleanup-nil-guard.md` |
| 173 | Universal Esc scroll-reset | `stories/173-universal-esc-scroll-reset.md` |
| 174 | Page B pane IDs and layout | `stories/174-page-b-pane-ids-and-layout.md` |
| 175 | GatewayHealth + PollingTraffic panes | `stories/175-gateway-health-and-polling-traffic-panes.md` |
| 176 | GatewayLive pane | `stories/176-gateway-live-pane.md` |
| 177 | NetworkLog fix + RequestFlow deletion | `stories/177-networklog-fix-app-wiring-requestflow-deletion.md` |
| 178 | Universal filter UX | `stories/178-universal-filter-ux.md` |
| 179 | Page B toggle keys | `stories/179-page-b-toggle-keys.md` |
| 180 | Stacked page B layout | `stories/180-stacked-page-b-layout.md` |
| 181 | TableBasedPane + flat page B | `stories/181-table-based-pane-and-flat-page-b.md` |
| 182 | GatewayLive multi-column | `stories/182-gateway-live-multi-column.md` |
| 210 | Fix stats page first-show + health bars | `stories/210-fix-stats-page-first-show-and-health-bars.md` |
| 211 | PollingTraffic stats row | `stories/211-polling-traffic-stats-row.md` |

## Acceptance Criteria

- [ ] Stats page toggles between Player and Stats via `0` key
- [ ] GatewayHealth pane renders 4 health rows with correct warning colours
- [ ] PollingTraffic pane shows playback poll cadence + library cache freshness
- [ ] GatewayLive pane maintains 500-entry reverse-chronological buffer; scrollable with filter
- [ ] NetworkLog scrolls through all requests; correct cross-tick decision association
- [ ] Universal Esc: clear filter first, then reset scroll to page 1 on all table panes
- [ ] TableBasedPane consolidates duplicated filter routing across 9 panes
- [ ] StateReader interface decouples panes from concrete Store type for testing
- [ ] All developer panes covered by unit tests using StateReader mocks
