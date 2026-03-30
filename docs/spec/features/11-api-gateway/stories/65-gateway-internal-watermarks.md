---
title: "Gateway-Internal Watermarks"
feature: 11-api-gateway
status: done
---

## Background
Feature 64's UI-side watermarks never show activity because Snapshot() refills tokens before reading -- consumed tokens are invisible by the time the UI samples them. Token bucket capacity is 10 with a refill rate of 10/sec, so a consumed token recovers in ~100ms. Most HTTP requests complete in <100ms, so the semaphore always showed 0 active. This feature moves watermark tracking into the Gateway itself, where consumption events are observed atomically at the moment they occur.

Depends on: Feature 64 (200ms snapshot refresh -- kept for smooth backoff countdowns).

## Design

### Watermark Fields in Gateway
Add `minTokens float64` to `tokenBucket` (updated under bucket mutex when `tb.tokens--` happens). Add `peakConcurrent int` to `Gateway` (updated under gateway mutex when semaphore acquired).

### GatewayState Extension
Add `PeakConcurrent int` and `MinTokens int` fields to `domain.GatewayState`.

### GatewaySnapshotter Extension
Add `ResetWatermarks()` to the interface. Called by UI on each 1-second boundary.

### Tracking Logic
In `tokenBucket.wait()`, after `tb.tokens--`:
```go
if tb.tokens < tb.minTokens {
    tb.minTokens = tb.tokens
}
```

In `Do()`, after semaphore acquire:
```go
active := len(g.semaphore)
if active > g.peakConcurrent {
    g.peakConcurrent = active
}
```

### Reset Pattern
In TickMsg handler (1s): snapshot first (captures peaks), then reset:
```go
p.lastSnapshot = p.gateway.Snapshot()
p.gateway.ResetWatermarks()
```

### UI-Side Cleanup
Remove `peakConcurrent` and `minTokens` fields from RequestFlowPane. Read from snapshot instead.

## Acceptance Criteria
- [ ] `GatewayState` includes `PeakConcurrent` and `MinTokens` fields
- [ ] `GatewaySnapshotter` interface includes `ResetWatermarks()` method
- [ ] Token consumption in `wait()` updates `minTokens` under bucket mutex
- [ ] Semaphore acquisition in `Do()` updates `peakConcurrent` under gateway mutex
- [ ] `Snapshot()` returns gateway-tracked watermarks
- [ ] `ResetWatermarks()` resets to current values (not zero)
- [ ] `RequestFlowPane` no longer has `minTokens`/`peakConcurrent` fields
- [ ] Making 3+ requests shows `(min: N)` where N < TokensMax
- [ ] Making concurrent slow requests shows `(peak: N)` where N > 0
- [ ] Idle state shows no annotations
- [ ] All existing tests pass (updated for new interface)
- [ ] `make ci` passes

## Tasks
- [ ] Add watermark fields to GatewayState and Gateway, add ResetWatermarks() to interface
      - test: NewGateway initializes minTokens at bucket max
- [ ] Track watermarks on token consumption and semaphore acquisition
      - test: 3 requests -> Snapshot().MinTokens < TokensMax; 3 concurrent slow -> PeakConcurrent >= 2
- [ ] Return watermarks in Snapshot() and implement ResetWatermarks()
      - test: Snapshot includes watermarks; ResetWatermarks resets to current values
- [ ] Remove UI-side watermark tracking, use gateway watermarks in pane
      - test: update all watermark tests to verify via Snapshot(); annotation tests use gateway watermarks
- [ ] Update tests for gateway watermarks
      - test: gateway-level watermark tests; pane tests use mock GatewaySnapshotter
- [ ] Update documentation
      - test: docs change only
