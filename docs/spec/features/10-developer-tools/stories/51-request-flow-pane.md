---
title: "Page B: Request Flow + Network Log"
feature: 14-nerd-status
status: done
---

## Background
This story built the two foundational Page B panes from scratch. Page B ("Nerd Status") is toggled via key 0 and shows the NowPlaying compact strip plus two new panes: RequestFlowPane (live animated visualization of the APP-GATEWAY-SPOTIFY request pipeline) and NetworkLogPane (scrollable table of API request history). All data is internal -- no new Spotify API calls. The panes read from *Gateway (token bucket, semaphore, inflight map) via a new Snapshot() method and from *Store (net log entries, throttle state, fetching sentinels, staleness timestamps).

## Design

### Gateway Observability
`GatewayState` struct with TokensAvailable, TokensMax, ConcurrentActive, ConcurrentMax, BackoffRemaining, DedupWaiters, InFlightKeys fields. Thread-safe `Gateway.Snapshot()` method.

### RequestFlowPane
Implements layout.Pane with ID PaneRequestFlow. Three columns: APP (endpoint names with priority coloring), GATEWAY (token bucket bar, semaphore bar, backoff timer, dedup waiters), SPOTIFY (status + latency with color coding). Connecting arrows animate on 200ms VisualizerTickMsg. Bottom status strip shows polling state and store fetching sentinels/staleness. Refresh: TickMsg (1s) for gateway snapshot, VisualizerTickMsg (200ms) for arrows.

### NetworkLogPane
Implements layout.Pane with ID PaneNetworkLog. Table columns: TIME, METHOD, ENDPOINT, STATUS, LATENCY, NOTES. Data from store.NetLogEntries() (200-entry ring buffer). Color coding per status. Latency bar in NOTES (1-10 chars, max 200ms). Newest first. Filter by endpoint/status.

## Acceptance Criteria
- [ ] Gateway.Snapshot() provides thread-safe read access
- [ ] RequestFlowPane satisfies layout.Pane, shows 3 columns
- [ ] Token bucket bar, semaphore bar, backoff timer render correctly
- [ ] Arrow animation advances on 200ms tick
- [ ] Request states visible: flowing, wait, dedup, blocked
- [ ] Status strip shows polling and store state
- [ ] NetworkLogPane satisfies layout.Pane, shows scrollable table
- [ ] Log entries color-coded by status
- [ ] Latency bars proportional to response time
- [ ] Filter works on endpoint and status code
- [ ] Both panes registered and visible on Page B
- [ ] make ci passes

## Tasks
- [ ] Expose Gateway observability state -- GatewayState struct and Snapshot() in internal/api/gateway.go
      - test: correct token count; concurrent count; backoff remaining; thread-safe access
- [ ] Create RequestFlowPane in internal/ui/panes/requestflow_pane.go
      - test: interface satisfaction; 3 columns; token/semaphore bars; backoff; arrows; request fade; status strip; color coding
- [ ] Create NetworkLogPane in internal/ui/panes/networklog_pane.go
      - test: interface satisfaction; 6 columns; newest-first; color coding; latency bar; filter; j/k scrolling; empty log; full buffer
- [ ] Register Page B panes in App
      - test: Page B shows 3 panes; key 0 switches; TickMsg reaches both; gateway state reflected
- [ ] Comprehensive tests
      - test: full lifecycle; gateway activity simulation; table updates; page switch preserves state; filter 429; idle state; empty log; backoff arrows
