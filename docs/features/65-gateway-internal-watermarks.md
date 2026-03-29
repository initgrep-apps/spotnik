# Feature 65 — Gateway-Internal Watermarks

> **Fix:** Feature 64's UI-side watermarks never show activity because `Snapshot()` refills
> tokens before reading — consumed tokens are invisible by the time the UI samples them.
> This feature moves watermark tracking into the Gateway itself, where consumption events
> are observed atomically at the moment they occur.

## Background

Feature 64 added `minTokens` and `peakConcurrent` tracking in `RequestFlowPane`, sampled
every 200ms via `viz.TickMsg`. However, this approach is fundamentally broken:

1. **Token bucket refill in Snapshot()**: `gateway.go:196` computes
   `tokens + elapsed*rate` before returning. A token consumed 50ms ago is already
   recovered by the time any external caller reads it.

2. **Semaphore releases in <100ms**: Most HTTP requests complete before the next 200ms
   sample. `len(g.semaphore)` is 0 by the time `Snapshot()` reads it.

**Result:** UI watermarks always equal current values — annotations never appear.

**The fix:** Track watermarks inside the `Gateway` struct itself. When `tb.tokens--`
happens (token consumed) and when `g.semaphore <- struct{}{}` happens (slot acquired),
update watermark fields atomically under the existing locks. `Snapshot()` returns these
watermarks alongside current values. The UI reads them passively — no sampling needed.

**Depends on:** Feature 64 (200ms snapshot refresh — kept for smooth backoff countdowns)

---

## Task 1: Add watermark fields to `GatewayState` and `Gateway`

**Problem:** `GatewayState` has no fields for historical activity. The Gateway has no
internal watermark tracking.

**Fix:**

### 1a. Add fields to `domain.GatewayState` (`internal/domain/gateway.go`):

```go
type GatewayState struct {
    // ... existing fields ...

    // PeakConcurrent is the highest ConcurrentActive seen since the last watermark reset.
    PeakConcurrent int
    // MinTokens is the lowest TokensAvailable seen since the last watermark reset.
    MinTokens int
}
```

### 1b. Add fields to `api.Gateway` (`internal/api/gateway.go`):

```go
type Gateway struct {
    // ... existing fields ...

    // peakConcurrent tracks the max semaphore occupancy since last reset.
    // Updated under gateway mutex when semaphore is acquired.
    peakConcurrent int
    // minTokens tracks the min token level since last reset.
    // Updated under bucket mutex when a token is consumed.
    minTokens int
    // minTokensInit tracks whether minTokens has been initialized this window.
    minTokensInit bool
}
```

Initialize in `NewGateway()`:
```go
func NewGateway() *Gateway {
    g := &Gateway{
        bucket:    newTokenBucket(10, 10),
        semaphore: make(chan struct{}, 5),
        inflight:  make(map[RequestKey]*inflightEntry),
    }
    g.minTokens = int(g.bucket.max) // start at max
    g.minTokensInit = true
    return g
}
```

### 1c. Add `ResetWatermarks()` to `GatewaySnapshotter` interface (`internal/domain/gateway.go`):

```go
type GatewaySnapshotter interface {
    Snapshot() GatewayState
    // ResetWatermarks resets peak activity watermarks. Called by the UI
    // on each 1-second boundary so annotations reflect recent activity.
    ResetWatermarks()
}
```

**Files:**
- Modify: `internal/domain/gateway.go` — add `PeakConcurrent`, `MinTokens` to struct; add `ResetWatermarks()` to interface
- Modify: `internal/api/gateway.go` — add watermark fields to `Gateway`, initialize in constructor

**Tests:**
- `TestGateway_NewGateway_InitializesWatermarks` — verify `minTokens` starts at bucket max

**Commit:** `feat(api): add watermark fields to Gateway and GatewayState`

---

## Task 2: Track watermarks on token consumption and semaphore acquisition

**Problem:** Watermark fields exist but are never updated when events occur.

**Fix:**

### 2a. Track `minTokens` when token is consumed

In `tokenBucket.wait()` (`gateway.go:103-104`), after `tb.tokens--`, update the gateway's
`minTokens`. But `wait()` is on `tokenBucket`, not `Gateway` — so we need to add a callback
or move the tracking.

**Simplest approach:** Add a `minTokens` field directly to `tokenBucket` (it already has its
own mutex), and read it in `Snapshot()`.

```go
type tokenBucket struct {
    mu        sync.Mutex
    tokens    float64
    max       float64
    rate      float64
    lastFill  time.Time
    minTokens float64 // lowest token level since last reset
}
```

In `wait()`, after `tb.tokens--` (line 104):
```go
tb.tokens--
if tb.tokens < tb.minTokens {
    tb.minTokens = tb.tokens
}
```

In `newTokenBucket()`:
```go
func newTokenBucket(max, rate float64) *tokenBucket {
    return &tokenBucket{
        tokens:    max,
        max:       max,
        rate:      rate,
        lastFill:  time.Now(),
        minTokens: max,
    }
}
```

### 2b. Track `peakConcurrent` when semaphore is acquired

In `Do()` (`gateway.go:373-374`), after acquiring the semaphore slot, update
`peakConcurrent` under the gateway mutex:

```go
case g.semaphore <- struct{}{}:
    // Track peak concurrent usage for watermark display.
    g.mu.Lock()
    active := len(g.semaphore)
    if active > g.peakConcurrent {
        g.peakConcurrent = active
    }
    g.mu.Unlock()
    defer func() { <-g.semaphore }()
```

**Files:**
- Modify: `internal/api/gateway.go` — update `wait()` for minTokens, `Do()` for peakConcurrent

**Tests:**
- `TestGateway_TokenConsumption_TracksMinTokens` — make 3 requests, verify `Snapshot().MinTokens < TokensMax`
- `TestGateway_ConcurrentRequests_TracksPeakConcurrent` — fire 3 concurrent slow requests,
  verify `Snapshot().PeakConcurrent >= 2`

**Commit:** `feat(api): track watermarks on token consumption and semaphore acquisition`

---

## Task 3: Return watermarks in `Snapshot()` and implement `ResetWatermarks()`

**Problem:** `Snapshot()` doesn't include watermark values. No reset mechanism exists.

**Fix:**

### 3a. Update `Snapshot()` to include watermarks

```go
func (g *Gateway) Snapshot() domain.GatewayState {
    g.bucket.mu.Lock()
    // ... existing refill + read logic ...
    minTokens := int(g.bucket.minTokens)
    g.bucket.mu.Unlock()

    g.mu.Lock()
    // ... existing backoff + dedup reads ...
    peakConcurrent := g.peakConcurrent
    g.mu.Unlock()

    return domain.GatewayState{
        // ... existing fields ...
        PeakConcurrent: peakConcurrent,
        MinTokens:      minTokens,
    }
}
```

### 3b. Implement `ResetWatermarks()`

```go
// ResetWatermarks resets peak activity watermarks to their default values.
// Called by the UI on each 1-second boundary so annotations reflect recent activity.
func (g *Gateway) ResetWatermarks() {
    g.bucket.mu.Lock()
    g.bucket.minTokens = g.bucket.tokens // reset to current level
    if g.bucket.minTokens > g.bucket.max {
        g.bucket.minTokens = g.bucket.max
    }
    g.bucket.mu.Unlock()

    g.mu.Lock()
    g.peakConcurrent = len(g.semaphore) // reset to current occupancy
    g.mu.Unlock()
}
```

**Files:**
- Modify: `internal/api/gateway.go` — update `Snapshot()`, add `ResetWatermarks()`

**Tests:**
- `TestGateway_Snapshot_IncludesWatermarks` — consume tokens, verify `Snapshot().MinTokens < TokensMax`
- `TestGateway_ResetWatermarks` — consume tokens, call `ResetWatermarks()`, verify watermarks
  are reset to current values

**Commit:** `feat(api): return watermarks in Snapshot and add ResetWatermarks`

---

## Task 4: Remove UI-side watermark tracking, use gateway watermarks

**Problem:** `RequestFlowPane` has `peakConcurrent` and `minTokens` fields that sample
from `Snapshot()` — which is broken. Now that the gateway tracks these internally,
the UI should read them passively from `GatewayState`.

**Fix:**

### 4a. Remove UI-side watermark fields from `RequestFlowPane`

Remove from struct:
- `peakConcurrent int`
- `minTokens int`

Remove exported getters:
- `MinTokens() int`
- `PeakConcurrent() int`

### 4b. Update `viz.TickMsg` handler

Remove watermark tracking logic. Keep the snapshot refresh (still useful for 200ms resolution):

```go
case viz.TickMsg:
    p.frameIndex++
    if p.gateway != nil {
        p.lastSnapshot = p.gateway.Snapshot()
    }
    p.syncFromNetLog()
    return p, nil
```

### 4c. Update `TickMsg` handler

Call `ResetWatermarks()` on the gateway instead of resetting local fields:

```go
case TickMsg:
    if p.gateway != nil {
        p.gateway.ResetWatermarks()
        p.lastSnapshot = p.gateway.Snapshot()
    }
    p.syncFromNetLog()
    return p, nil
```

Note: `ResetWatermarks()` is called BEFORE `Snapshot()` so the snapshot reflects the
fresh (reset) state. The peak values from the previous window were already returned
by the last `viz.TickMsg` snapshot.

Actually — wait. The issue is that `ResetWatermarks()` clears the peaks, then `Snapshot()`
returns the cleared values. The annotations would never show because the snapshot at render
time always sees reset values. The correct approach:

**Alternative:** Don't call `ResetWatermarks()` before `Snapshot()`. Instead, call it AFTER:

```go
case TickMsg:
    if p.gateway != nil {
        p.lastSnapshot = p.gateway.Snapshot()  // captures peaks from last window
        p.gateway.ResetWatermarks()             // start fresh for next window
    }
    p.syncFromNetLog()
    return p, nil
```

But then the `viz.TickMsg` snapshots between TickMsgs accumulate peaks from the growing
window, which is exactly what we want — they show activity that occurred since the last reset.

### 4d. Update `gatewayStateLines()` in `requestflow_boxed.go`

Read watermarks from `p.lastSnapshot` instead of `p.minTokens`/`p.peakConcurrent`:

```go
if snap.MinTokens < snap.TokensAvailable {
    tokenLine += mutedStyle.Render(fmt.Sprintf(" (min: %d)", snap.MinTokens))
}

if snap.PeakConcurrent > snap.ConcurrentActive {
    semLine += mutedStyle.Render(fmt.Sprintf(" (peak: %d)", snap.PeakConcurrent))
}
```

### 4e. Update constructor

Remove `minTokens` initialization:
```go
func NewRequestFlowPane(gw domain.GatewaySnapshotter, s *state.Store, t theme.Theme) *RequestFlowPane {
    p := &RequestFlowPane{
        theme:   t,
        gateway: gw,
        store:   s,
    }
    if gw != nil {
        p.lastSnapshot = gw.Snapshot()
    }
    return p
}
```

**Files:**
- Modify: `internal/ui/panes/requestflow_pane.go` — remove watermark fields/getters, update handlers
- Modify: `internal/ui/panes/requestflow_boxed.go` — read from `snap.MinTokens`/`snap.PeakConcurrent`

**Tests:**
- Update all watermark tests to verify via `Snapshot()` return values instead of pane getters
- Remove `TestRequestFlowPane_New_InitializesMinTokens` (field no longer exists on pane)
- Update `TestRequestFlowPane_PeakWatermarks_*` to verify `Snapshot().PeakConcurrent` and
  `Snapshot().MinTokens` from the gateway
- Update annotation tests to set gateway watermarks instead of pane fields

**Commit:** `refactor(ui): use gateway-internal watermarks instead of UI-side sampling`

---

## Task 5: Update tests for gateway watermarks

**Problem:** Existing gateway tests don't cover watermark behavior. Existing pane tests
reference removed fields.

**Fix:**

### 5a. Gateway tests (`internal/api/gateway_test.go`)

Add tests:
- `TestGateway_MinTokens_TrackedOnConsumption` — make requests through gateway, verify
  `Snapshot().MinTokens` is less than `TokensMax`
- `TestGateway_PeakConcurrent_TrackedOnAcquisition` — fire concurrent slow requests,
  verify `Snapshot().PeakConcurrent >= 2`
- `TestGateway_ResetWatermarks_ClearsToCurrentValues` — consume tokens, call
  `ResetWatermarks()`, verify `MinTokens` reset to current level
- `TestGateway_Watermarks_IdleShowsDefaults` — no requests, verify
  `MinTokens == TokensMax` and `PeakConcurrent == 0`

### 5b. Pane tests — update or remove

- Remove tests that reference `pane.MinTokens()` or `pane.PeakConcurrent()` (those getters
  no longer exist)
- Update annotation tests: instead of directly setting `p.minTokens`, make real requests
  through the gateway to trigger watermarks, then verify View() output
- Alternatively: use a mock `GatewaySnapshotter` that returns controlled `GatewayState`
  with `MinTokens` and `PeakConcurrent` pre-set — this is simpler and more deterministic

### 5c. Mock GatewaySnapshotter for pane tests

Create or update a mock in the test file:

```go
type mockGatewaySnapshotter struct {
    state domain.GatewayState
}

func (m *mockGatewaySnapshotter) Snapshot() domain.GatewayState { return m.state }
func (m *mockGatewaySnapshotter) ResetWatermarks()              {}
```

Use this mock in pane annotation tests, setting `MinTokens` and `PeakConcurrent` to
controlled values.

**Files:**
- Modify: `internal/api/gateway_test.go` — add watermark tests
- Modify: `internal/ui/panes/requestflow_pane_test.go` — update/remove watermark tests
- Modify: `internal/ui/panes/requestflow_boxed_test.go` — update annotation tests

**Commit:** `test(api,ui): add gateway watermark tests, update pane tests for new interface`

---

## Task 6: Update documentation

**Fix:**

1. Update `docs/ARCHITECTURE.md` — note that watermarks are tracked inside the Gateway
   (not sampled from the UI), and `ResetWatermarks()` is called on 1s boundaries
2. Update `docs/features/00-overview.md` — mark Feature 65 complete

**Files:**
- Modify: `docs/ARCHITECTURE.md`
- Modify: `docs/features/00-overview.md`

**Commit:** `docs: add Feature 65 gateway-internal watermarks to architecture docs`

---

## Acceptance Criteria

- [ ] `GatewayState` includes `PeakConcurrent` and `MinTokens` fields
- [ ] `GatewaySnapshotter` interface includes `ResetWatermarks()` method
- [ ] Token consumption in `wait()` updates `minTokens` under bucket mutex
- [ ] Semaphore acquisition in `Do()` updates `peakConcurrent` under gateway mutex
- [ ] `Snapshot()` returns gateway-tracked watermarks
- [ ] `ResetWatermarks()` resets to current values (not zero)
- [ ] `RequestFlowPane` no longer has `minTokens`/`peakConcurrent` fields
- [ ] `TickMsg` handler calls `ResetWatermarks()` then `Snapshot()`
- [ ] `gatewayStateLines()` reads annotations from `snap.MinTokens`/`snap.PeakConcurrent`
- [ ] Making 3+ requests shows `(min: N)` where N < TokensMax in the GATEWAY box
- [ ] Making concurrent slow requests shows `(peak: N)` where N > 0
- [ ] Idle state shows no annotations
- [ ] All existing tests pass (updated for new interface)
- [ ] `make ci` passes

---

## Verification

```bash
# Gateway watermark fields exist
grep 'minTokens\|peakConcurrent' internal/api/gateway.go

# GatewayState has new fields
grep 'PeakConcurrent\|MinTokens' internal/domain/gateway.go

# ResetWatermarks in interface
grep 'ResetWatermarks' internal/domain/gateway.go

# UI reads from snapshot, not local fields
grep 'snap.MinTokens\|snap.PeakConcurrent' internal/ui/panes/requestflow_boxed.go

# UI no longer has local watermark fields
! grep 'minTokens\|peakConcurrent' internal/ui/panes/requestflow_pane.go

# Gateway tests
go test ./internal/api/ -run 'Watermark|MinTokens|PeakConcurrent' -v

# Full CI
make ci
```

---

## Implementation Notes for Agents

### Key insight
The previous approach (Feature 64) sampled `Snapshot()` from the UI at 200ms intervals.
This failed because `Snapshot()` refills tokens before reading — consumed tokens are
invisible. The fix is to track the minimum token level **at the moment of consumption**
(inside `tokenBucket.wait()` under `tb.mu`), not by sampling the refilled value later.

### Existing patterns to follow
- `tokenBucket` has its own `sync.Mutex` — add `minTokens` field there, update under the
  same lock as `tb.tokens--`
- `Gateway` has `sync.Mutex` for inflight/backoff — add `peakConcurrent` there, update
  after semaphore acquire
- `domain.GatewayState` is a plain struct with exported fields — just add two more
- `GatewaySnapshotter` is a one-method interface in `domain/` — adding `ResetWatermarks()`
  is minimal. All callers must be updated (pane tests, app tests)

### Do NOT
- Change the token bucket rate or capacity
- Change the semaphore capacity
- Remove the 200ms snapshot refresh from `viz.TickMsg` (still needed for smooth backoff display)
- Add new message types
- Import new dependencies

### Interface change impact
Adding `ResetWatermarks()` to `GatewaySnapshotter` will break any mock/test that implements
the interface. Search for all implementations:
- `internal/api/gateway.go` — the real implementation (add the method)
- Test files — any mock `GatewaySnapshotter` (add a no-op `ResetWatermarks()`)

---

*Depends on: Feature 64*
*Blocks: Nothing*
