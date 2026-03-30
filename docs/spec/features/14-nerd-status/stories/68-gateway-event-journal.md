---
title: "Request Flow Replay Engine"
feature: 14-nerd-status
status: done
---

## Background
The Request Flow pane used snapshot-polling and NetLog syncing to observe gateway state, but gateway decisions happen in microseconds and self-heal before any poll can catch them. Feature 67 instrumented the Gateway to emit fine-grained lifecycle events. This story rewrote the Request Flow pane to consume those events and replay them as a slow-motion time machine. The pane no longer holds a GatewaySnapshotter reference. Instead, it reads events from store.ReadEventsFrom() using a cursor, queues them, and replays one per viz.TickMsg (200ms minimum visibility).

## Design

### Replay Data Model
animationPhase enum (phaseEntered, phaseAtGateway, phaseInFlight, phaseCompleted, phaseDone). requestAnimation struct tracking requestID, method, path, priority, phase, decision, statusCode, durationMs, enteredAt. decisionEntry struct. replayDisplayState struct with snapshot, requests map, decisions list.

### Pane Rewrite
Remove gateway GatewaySnapshotter, lastSnapshot, recentReqs. Add eventCursor, replayQueue, displayState. Constructor takes *state.Store and theme.Theme only.

### Replay Loop
drainEvents() reads from store.ReadEventsFrom(cursor). processNextEvent() pops one per tick, updates snapshot, calls processRequestEvent(). ageOutEntries() removes decisions >3s, completed requests >5s. formatDecisionLabel() maps all 13 EventKinds to display strings with icons.

### View Updates
buildAppBoxLines() from displayState.requests sorted by enteredAt. buildSpotifyBoxLines() for phaseInFlight+. buildGatewayBoxLines() with state bars from snapshot plus scrolling decision log. Theme colors for decision log entries.

### Cleanup
Remove GatewayState, GatewaySnapshotter, GatewayDecision from domain/. Remove deprecated Snapshot() shim from gateway.

## Acceptance Criteria
- [ ] Pane reads events from store.ReadEventsFrom() using cursor
- [ ] Events replay at 200ms minimum visibility
- [ ] GATEWAY box shows state bars from replay snapshot plus decision log
- [ ] Multiple requests animate concurrently at staggered phases
- [ ] Blocked requests show in APP but skip InFlight/SPOTIFY
- [ ] Decisions age out after 3s, completed requests after 5s
- [ ] GatewayState, GatewaySnapshotter, GatewayDecision removed
- [ ] Deprecated Snapshot() shim removed
- [ ] All tests rewritten and passing
- [ ] make ci passes

## Tasks
- [ ] Add replay data model types
      - test: zero value; phase ordering
- [ ] Rewrite RequestFlowPane struct and constructor -- remove old fields, add replay fields
      - test: empty display state; ID/Title/ToggleKey unchanged
- [ ] Implement replay loop in Update() -- drainEvents, processNextEvent, processRequestEvent, ageOutEntries, formatDecisionLabel
      - test: drain events; process one per tick; snapshot updates; phase progression; blocked skips InFlight; decision log grows/ages; completed ages; formatDecisionLabel all 13 kinds
- [ ] Update View() and box rendering for replay display state
      - test: decision log visible; state bars from snapshot; requests in APP/SPOTIFY boxes; arrow states; flat fallback; theme colors
- [ ] Remove old snapshot/netlog code from pane and retired domain types
      - test: removed types no longer compile; update/remove affected tests
- [ ] Update existing tests for replay engine
      - test: test helper creates pane with Store; injects events; viz.TickMsg triggers replay
- [ ] Update documentation
      - test: docs change only
