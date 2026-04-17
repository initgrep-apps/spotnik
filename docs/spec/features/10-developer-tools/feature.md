---
title: "Developer Visibility (Page B)"
status: done
---

## Description

Page B (`0` key) surfaces the internals of Spotnik's API layer for developers. The RequestFlow pane visualizes each gateway decision in real time — showing request type, priority, dedup result, rate-limit status, and backoff state — with a replay engine for stepping through past events. The NetworkLog pane is a scrollable table of every API request with timestamp, method, endpoint, HTTP status, priority classification, and gateway decision. Developer foundations stories add onboarding docs, test infrastructure, StateReader interface, BasePane pattern, and RebuildTableTheme helper.

## Acceptance Criteria

- [ ] Page B (`0` key) toggles between Page A (music) and Page B (developer view)
- [ ] RequestFlow pane renders each gateway decision with correct decision reason
- [ ] Replay engine steps through past request flow events correctly
- [ ] NetworkLog scrolls through all requests; filter (f) narrows by endpoint or status
- [ ] StateReader interface decouples panes from concrete Store type for testing
- [ ] All developer panes covered by unit tests using StateReader mocks
