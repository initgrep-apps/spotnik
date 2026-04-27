---
name: project_spotnik_feature67_complete
description: Feature 67 (Gateway Event Instrumentation): emitEvent helpers, Do() lifecycle events, periodic emission, GatewayRecorder retirement, nolint for deprecated shims
type: project
---

## Feature 67 — Gateway Event Instrumentation

**What was built:**
- `emitEvent()` / `emitEventLocked()` helpers in `internal/api/gateway.go`
- `captureSnapshot()` / `captureSnapshotLocked()` for snapshots under correct lock order
- `nextRequestID atomic.Uint64` for per-request event correlation
- `Do()` instrumented: EventRequestEntered → TokenConsumed → SemaphoreAcquired → HttpCompleted → SemaphoreReleased → RequestAllowed/Waited/Blocked/DedupJoined/DedupResolved/BackoffStarted
- `CheckAndEmitRefill()` + `CheckAndEmitBackoffExpiry()` for periodic internal events
- `GatewayRecorder` interface retired; replaced by `domain.GatewayEventRecorder`
- `MarkGatewayRecorded`/`IsGatewayRecorded` removed from `api/base.go` + `api/logging.go`
- `tokenBucket.minTokens` + `Gateway.peakConcurrent` watermark fields removed
- `Snapshot()` kept as deprecated shim; `GatewayState`/`GatewaySnapshotter`/`GatewayDecision` in domain/ kept with deprecation notes until Feature 68

**Key files:**
- `internal/api/gateway.go` — primary. Lines ~250–300: emitEvent helpers. Lines ~415–630: instrumented Do().
- `internal/api/gateway_test.go` — 15 new event tests (CaptureSnapshot, EmitEvent, NextRequestID, Do_NormalRequest/Blocked/InteractiveWait/Dedup/429/EventsHaveCorrectRequestID/EventsHaveSnapshots/SnapshotReflectsState, CheckAndEmitRefill/Backoff)
- `internal/app/app.go` — viz.TickMsg handler calls `a.gateway.CheckAndEmitRefill()` + `a.gateway.CheckAndEmitBackoffExpiry()` before forward to panes

**Patterns established:**
- `emitEvent()` for call sites without g.mu; `emitEventLocked()` for sites inside g.mu-locked section
- `captureSnapshot()` lock order: bucket.mu → g.mu (both acquired then released); `captureSnapshotLocked()` acquires only bucket.mu (g.mu already held)
- Periodic internal events called from viz.TickMsg (200ms) in app.go — same tick driving visualizer animations
- `SemaphoreReleased` emitted in deferred func after `<-g.semaphore` — fires last, captures moment slot freed

**Gotchas:**
- Tasks 1+2 committed together: changing `recorder` type from `GatewayRecorder` to `domain.GatewayEventRecorder` broke all RecordGatewayCall calls in Do()
- `EventRequestWaited` is FINAL decision event (not EventRequestAllowed) for interactive requests that waited on backoff — test initially wrong
- golangci-lint 2.x SA1019 (deprecated use) fires on all uses of `GatewayDecision`, `GatewayState`, `GatewaySnapshotter` — add `//nolint:staticcheck` per-line on prod code, file-level `//nolint:staticcheck` header on test files using deprecated types heavily
- File-level nolint comment goes BEFORE `package` — not after. Format: doc comment then `//nolint:staticcheck` then `package <name>`
- `EventBackoffStarted` emitted BEFORE `EventHttpCompleted` in 429 path (backoff set during body processing, before final emit loop). Ordering intentional + correct for replay.

**Testing notes:**
- `mockEventRecorder` struct with `events []domain.GatewayEvent` slice + `RecordEvent` method used in all event tests
- `collectKinds(events)` helper extracts EventKind values for sequence assertions
- Test helpers `assertContainsKind` + `assertNotContainsKind` for presence checks
- All 15 new tests pass with `-race` flag; no data races

**Coverage:** 86.9% for internal/api, 86.7% overall