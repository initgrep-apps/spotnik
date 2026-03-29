---
name: project_spotnik_feature67_complete
description: Feature 67 (Gateway Event Instrumentation): emitEvent helpers, Do() lifecycle events, periodic emission, GatewayRecorder retirement, nolint for deprecated shims
type: project
---

## Feature 67 — Gateway Event Instrumentation

**What was built:**
- `emitEvent()` / `emitEventLocked()` helpers in `internal/api/gateway.go`
- `captureSnapshot()` / `captureSnapshotLocked()` for state snapshots under correct lock ordering
- `nextRequestID atomic.Uint64` for per-request event correlation
- `Do()` fully instrumented: EventRequestEntered → TokenConsumed → SemaphoreAcquired → HttpCompleted → SemaphoreReleased → RequestAllowed/Waited/Blocked/DedupJoined/DedupResolved/BackoffStarted
- `CheckAndEmitRefill()` and `CheckAndEmitBackoffExpiry()` for periodic internal events
- `GatewayRecorder` interface retired; replaced by `domain.GatewayEventRecorder`
- `MarkGatewayRecorded`/`IsGatewayRecorded` removed from `api/base.go` and `api/logging.go`
- `tokenBucket.minTokens` and `Gateway.peakConcurrent` watermark fields removed
- `Snapshot()` retained as deprecated shim; `GatewayState`/`GatewaySnapshotter`/`GatewayDecision` in domain/ retained with deprecation notices until Feature 68

**Key files:**
- `internal/api/gateway.go` — primary file. Lines ~250–300: emitEvent helpers. Lines ~415–630: instrumented Do().
- `internal/api/gateway_test.go` — 15 new event tests (CaptureSnapshot, EmitEvent, NextRequestID, Do_NormalRequest/Blocked/InteractiveWait/Dedup/429/EventsHaveCorrectRequestID/EventsHaveSnapshots/SnapshotReflectsState, CheckAndEmitRefill/Backoff)
- `internal/app/app.go` — viz.TickMsg handler calls `a.gateway.CheckAndEmitRefill()` and `a.gateway.CheckAndEmitBackoffExpiry()` before forwarding to panes

**Patterns established:**
- `emitEvent()` for call sites that don't hold g.mu; `emitEventLocked()` for call sites inside a g.mu-locked section
- `captureSnapshot()` lock order: bucket.mu → g.mu (both acquired then released); `captureSnapshotLocked()` only acquires bucket.mu (g.mu already held)
- Periodic internal events called from viz.TickMsg (200ms) in app.go — same tick that drives visualizer animations
- `SemaphoreReleased` emitted in the deferred func after `<-g.semaphore` — this correctly fires last, capturing the moment the slot was freed

**Gotchas:**
- Tasks 1 and 2 had to be committed together because changing `recorder` type from `GatewayRecorder` to `domain.GatewayEventRecorder` immediately broke all RecordGatewayCall calls in Do()
- `EventRequestWaited` is the FINAL decision event (not EventRequestAllowed) for interactive requests that waited on backoff — test was initially wrong about this
- golangci-lint 2.x SA1019 (deprecated use) fires on all uses of `GatewayDecision`, `GatewayState`, `GatewaySnapshotter` — must add `//nolint:staticcheck` per-line on production code, and file-level `//nolint:staticcheck` header on test files that use deprecated types extensively
- The file-level nolint comment must appear BEFORE `package` — not after. Format: doc comment then `//nolint:staticcheck` then `package <name>`
- `EventBackoffStarted` is emitted BEFORE `EventHttpCompleted` in the 429 response path (because backoff is set during body processing, before the final emit loop). This ordering is intentional and correct for replay.

**Testing notes:**
- `mockEventRecorder` struct with `events []domain.GatewayEvent` slice and `RecordEvent` method used in all event tests
- `collectKinds(events)` helper extracts just the EventKind values for sequence assertions
- Test helpers `assertContainsKind` and `assertNotContainsKind` used for presence checks
- All 15 new tests pass with `-race` flag; no data races

**Coverage achieved:** 86.9% for internal/api, 86.7% overall
