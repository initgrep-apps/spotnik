---
title: "Nerd Status"
status: done
---

## Description
Developer visibility into Spotnik's internal request pipeline via Page B, featuring a live Request Flow visualization of gateway decisions and a scrollable Network Log of all API requests, both powered by a gateway event journal with replay engine.

Spotnik is a terminal Spotify client for developers, and developers want to see what their tools are doing under the hood. The Nerd Status feature provides full developer visibility into Spotnik's internal request pipeline through Page B, toggled via key 0. Page B shows the NowPlaying compact strip (row 1) plus two dedicated panes below: the Request Flow pane and the Network Log pane. Together these panes expose every gateway decision, every HTTP call, and every internal state change that occurs between the app and Spotify's API.

The Request Flow pane evolved from a flat column layout polling snapshots at 1-second intervals to a rich event-driven replay engine that consumes fine-grained lifecycle events from a gateway event journal, replaying them at human-observable speed (200ms per event). The Network Log pane migrated from a simple NetLog ring buffer to cursor-based reads from the GatewayEventLog, with PRIORITY and DECISION columns making blocked requests visible.

## Acceptance Criteria
- [ ] Gateway.Snapshot() provides thread-safe read access to internal state
- [ ] RequestFlowPane satisfies layout.Pane, shows 3 bordered sub-boxes
- [ ] NetworkLogPane satisfies layout.Pane, shows scrollable table
- [ ] Gateway event journal with 13 event kinds and cursor-based reads
- [ ] Gateway.Do() emits lifecycle events at every decision point
- [ ] Request Flow pane replays events at 200ms minimum visibility
- [ ] Network Log reads from GatewayEventLog with PRIORITY and DECISION columns
- [ ] Old NetLog system fully retired
- [ ] make ci passes
