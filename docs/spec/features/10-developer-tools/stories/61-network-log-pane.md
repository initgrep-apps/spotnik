---
title: "Fix Request Flow Gateway Visualization"
feature: 14-nerd-status
status: done
---

## Background
The Request Flow pane rendered the basic 3-column layout but was missing its core value: per-request gateway decision visualization. All requests appeared as generic animated arrows because the NetLog only captured HTTP-level data with no knowledge of gateway decisions (allowed, waited, deduped, blocked). Background requests rejected by backoff were completely invisible. The pane also used no theme colors and was missing staleness display. This story added the GatewayDecision type, extended NetLogEntry with priority and decision fields, instrumented Gateway.Do() with decision recording via a GatewayRecorder interface, applied full theme color coding, implemented four arrow states, added staleness display, and rendered InFlightKeys.

## Design

### GatewayDecision Type
`DecisionAllowed`, `DecisionWaited`, `DecisionDeduped`, `DecisionBlocked` enum in internal/domain/gateway.go. Extended NetLogEntry with Priority and GatewayDecision fields. Added RecordGatewayCall() to Store.

### Gateway Instrumentation
GatewayRecorder interface in gateway.go. SetRecorder() on Gateway. Do() instrumented at each decision point. gatewayRecordedKey context key prevents LoggingTransport double-recording.

### Theme Colors
Interactive requests TextPrimary, background TextMuted. Aged requests (>3s) TextMuted. 2xx Success, 429 Warning, 5xx Error. Token bucket filled dots Success. Semaphore squares Warning. Backoff timer Error.

### Four Arrow States
DecisionAllowed: animated arrow (or X for 429). DecisionWaited: `-- wait --`. DecisionDeduped: `-->dedup`. DecisionBlocked: `-- X --` in Error color.

### Staleness Display
renderStalenessStatus() checks fetchedAt timestamps against TTLs. Display `stale: playlists(Ns), albums(Ns)`.

### InFlightKeys
Render up to 3 keys with `-> keyname` format, plus `... +N more` truncation.

## Acceptance Criteria
- [ ] Gateway decisions tracked per-request
- [ ] Background blocked requests visible
- [ ] Interactive vs Background different colors
- [ ] Status codes color-coded
- [ ] Four arrow states render correctly
- [ ] Gateway state bars use theme colors
- [ ] Status strip shows stale data domains
- [ ] InFlightKeys displayed when non-empty
- [ ] LoggingTransport no double-recording
- [ ] All existing tests pass
- [ ] make ci passes

## Tasks
- [ ] Add GatewayDecision type and extend NetLogEntry -- domain types, store methods
      - test: NetLogEntry with Priority/GatewayDecision round-trips; RecordGatewayCall populates all fields
- [ ] Instrument Gateway.Do() with decision recording -- GatewayRecorder interface, SetRecorder, context key
      - test: blocked records DecisionBlocked; dedup records DecisionDeduped; normal records DecisionAllowed; LoggingTransport skips
- [ ] Theme color coding in Request Flow pane
      - test: View() contains ANSI escapes; existing tests still pass
- [ ] Four arrow states for gateway decisions
      - test: allowed animated; waited shows "wait"; deduped shows "dedup"; blocked shows X; 429 shows X with Warning
- [ ] Staleness display in status strip
      - test: stale domain shown; within TTL not shown; multiple stale comma-separated; never-fetched not shown
- [ ] Render InFlightKeys in gateway state block
      - test: 2 keys both appear; 5 keys shows 3 + "+2 more"; empty no section
- [ ] Update documentation
      - test: docs change only
