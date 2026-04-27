---
name: project_spotnik_feature66_complete
description: Feature 66 (Gateway Event Types & Storage): EventKind enum, GatewayStateSnapshot, GatewayEvent, GatewayEventRecorder, GatewayEventLog ring buffer, Store integration
type: project
---

## Feature 66 — Gateway Event Types & Storage

**Built:**
- `EventKind` enum (13 constants, iota) in `internal/domain/gateway.go`
- `GatewayStateSnapshot` struct in `internal/domain/gateway.go` — frozen state copy per event (no watermark fields, unlike `GatewayState`)
- `GatewayEvent` struct in `internal/domain/gateway.go` — timestamped event w/ kind, request ID, method/path/priority, status code, duration, embedded snapshot
- `GatewayEventRecorder` interface in `internal/domain/gateway.go` — single `RecordEvent(GatewayEvent)` method, implemented by `*state.Store`
- `GatewayEventLog` ring buffer in `internal/state/eventlog.go` — 500-entry cap, cursor reads w/ monotonic seq number
- `Store.RecordEvent()`, `Store.ReadEventsFrom()` in `internal/state/store.go`

**Key files:**
- `internal/domain/gateway.go` — holds old `GatewayState`/`GatewayDecision` (polling era) + new `GatewayEvent`/`EventKind`/`GatewayStateSnapshot`/`GatewayEventRecorder` (journal era); coexist til Feature 69
- `internal/state/eventlog.go` — `GatewayEventLog` struct; `Add()`, `ReadFrom()`, `Len()`; local var `ringCap` (not `cap`) — avoid shadow builtin
- `internal/state/store.go` — `eventLog *GatewayEventLog` field; init in `New()` next to `netLog`; compile-time check `var _ domain.GatewayEventRecorder = &Store{}`

**Patterns:**
- Compile-time iface check `var _ domain.GatewayEventRecorder = &Store{}` lives in `store.go` (prod code), NOT dup in `store_test.go`
- `GatewayEventLog.ReadFrom()` uses `ringCap := len(l.entries)` (not `cap`) — avoid shadow `cap()` builtin
- Ring buf cursor: `behind = sequence - cursor`; clamp to `count` if cursor stale; `start = (head - behind + ringCap) % ringCap`

**Old/new coexist:**
- `NetLog` + `GatewayEventLog` both in `internal/state/` til Feature 69 retires `NetLog`
- `GatewayState` + `GatewayStateSnapshot` coexist in `domain/gateway.go` — `GatewayState` has watermark fields, `GatewayStateSnapshot` does not
- No behavior change to existing gateway code — purely additive

**Tests:**
- 17 new tests: 5 domain, 10 eventlog_test.go, 2 store_test.go
- `makeEvent()` helper in `eventlog_test.go` (pkg `state_test`) — no conflict w/ `store_test.go` (separate files, same external test pkg)
- `TestGatewayEventLog_ConcurrentAccess` checks goroutine safety, 10 writers + 5 readers
- Coverage: state 96.5%, domain 47.4%, overall 86.5%
- gofmt gotcha: struct literal fields w/ mixed alignment (e.g. `StatusCode: 200, DurationMs: 125`) re-aligned by gofmt — run `gofmt -w` before commit test files w/ composite literals