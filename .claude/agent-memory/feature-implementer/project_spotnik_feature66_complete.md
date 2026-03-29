---
name: project_spotnik_feature66_complete
description: Feature 66 (Gateway Event Types & Storage): EventKind enum, GatewayStateSnapshot, GatewayEvent, GatewayEventRecorder, GatewayEventLog ring buffer, Store integration
type: project
---

## Feature 66 — Gateway Event Types & Storage

**What was built:**
- `EventKind` enum (13 constants, iota) in `internal/domain/gateway.go`
- `GatewayStateSnapshot` struct in `internal/domain/gateway.go` — frozen state copy per event (no watermark fields unlike `GatewayState`)
- `GatewayEvent` struct in `internal/domain/gateway.go` — timestamped event with kind, request ID, method/path/priority, status code, duration, and embedded snapshot
- `GatewayEventRecorder` interface in `internal/domain/gateway.go` — single `RecordEvent(GatewayEvent)` method, implemented by `*state.Store`
- `GatewayEventLog` ring buffer in `internal/state/eventlog.go` — 500-entry capacity, cursor-based reads with monotonic sequence number
- `Store.RecordEvent()` and `Store.ReadEventsFrom()` methods in `internal/state/store.go`

**Key files:**
- `internal/domain/gateway.go` — now has both old `GatewayState`/`GatewayDecision` (polling era) and new `GatewayEvent`/`EventKind`/`GatewayStateSnapshot`/`GatewayEventRecorder` (journal era); they coexist until Feature 69
- `internal/state/eventlog.go` — `GatewayEventLog` struct; `Add()`, `ReadFrom()`, `Len()`; local var renamed `ringCap` (not `cap`) to avoid shadowing builtin
- `internal/state/store.go` — `eventLog *GatewayEventLog` field; initialized in `New()` alongside `netLog`; compile-time check `var _ domain.GatewayEventRecorder = &Store{}`

**Patterns established:**
- The compile-time interface check `var _ domain.GatewayEventRecorder = &Store{}` belongs in `store.go` (production code), NOT duplicated in `store_test.go`
- `GatewayEventLog.ReadFrom()` uses `ringCap := len(l.entries)` (not `cap`) to avoid shadowing the built-in `cap()` function
- Ring buffer cursor logic: `behind = sequence - cursor`; clamp to `count` if cursor is stale; `start = (head - behind + ringCap) % ringCap`

**Coexistence of old and new:**
- `NetLog` and `GatewayEventLog` both live in `internal/state/` until Feature 69 retires `NetLog`
- `GatewayState` and `GatewayStateSnapshot` coexist in `domain/gateway.go` — `GatewayState` has watermark fields, `GatewayStateSnapshot` does not
- No behavioral changes to existing gateway code in this feature — purely additive

**Testing notes:**
- 17 new tests: 5 in domain, 10 in eventlog_test.go, 2 in store_test.go
- `makeEvent()` helper in `eventlog_test.go` (package `state_test`) — no conflict with `store_test.go` since they're separate files in same external test package
- `TestGatewayEventLog_ConcurrentAccess` verifies goroutine safety with 10 writers + 5 readers
- Coverage: state 96.5%, domain 47.4%, overall 86.5%
- gofmt gotcha: struct literal fields with mixed alignment (like `StatusCode: 200, DurationMs: 125`) will be re-aligned by gofmt — always run `gofmt -w` before committing test files with composite literals
