# Feature 56 — Fix Request Flow Pane Data

> **Bug fix:** Request Flow pane shows only static gateway/polling info — the per-request
> flow visualization (APP → GATEWAY → SPOTIFY) never displays because `RequestCompletedMsg`
> is never emitted.

## Root Cause

`RequestCompletedMsg` is defined (`messages.go:38-49`) and the handler exists in
`RequestFlowPane.Update()` (`requestflow_pane.go:151-168`), but **no code in `app.go`
or anywhere else ever emits `RequestCompletedMsg`**. The `recentReqs` slice is always
empty, so `renderRequestRows()` returns nil.

Meanwhile, the Store's network log (`store.NetLogEntries()`) successfully records all
API calls via `RecordNetCall()` — the Network Log pane reads from it and works perfectly.

---

## Fix

On each `TickMsg`, populate `recentReqs` from the Store's existing `NetLogEntries()`.
This reuses the working infrastructure without the invasive approach of emitting
`RequestCompletedMsg` from every API response handler in `app.go`.

### `internal/ui/panes/requestflow_pane.go`

**Update `TickMsg` handler (lines 139-145):**
```go
// Before:
case TickMsg:
    if p.gateway != nil {
        p.lastSnapshot = p.gateway.Snapshot()
    }
    p.pruneOldRequests()
    return p, nil

// After:
case TickMsg:
    if p.gateway != nil {
        p.lastSnapshot = p.gateway.Snapshot()
    }
    p.syncFromNetLog()
    return p, nil
```

**Add new `syncFromNetLog()` method:**
```go
// syncFromNetLog reads the store's network log and populates recentReqs
// with the most recent entries within the requestAgeOut window.
func (p *RequestFlowPane) syncFromNetLog() {
    if p.store == nil {
        return
    }
    entries := p.store.NetLogEntries()
    cutoff := time.Now().Add(-requestAgeOut)

    // Rebuild from store — newest first, capped at maxRecentReqs.
    p.recentReqs = p.recentReqs[:0]
    for i := len(entries) - 1; i >= 0; i-- {
        e := entries[i]
        if e.Timestamp.Before(cutoff) {
            continue
        }
        p.recentReqs = append(p.recentReqs, reqDisplay{
            endpoint:    e.Path,
            statusCode:  e.StatusCode,
            latencyMs:   int(e.DurationMs),
            priority:    domain.PriorityBackground,
            completedAt: e.Timestamp,
        })
        if len(p.recentReqs) >= maxRecentReqs {
            break
        }
    }
}
```

### Key details

- **Field mapping:** `NetLogEntry.Path` → endpoint, `StatusCode` → statusCode,
  `DurationMs(int64)` → latencyMs(int), `Timestamp` → completedAt
- **Priority** defaults to `domain.PriorityBackground` (net log doesn't track priority)
- **Constants reused:** `requestAgeOut = 5s`, `maxRecentReqs = 6`
- **`NetLogEntries()` returns oldest-first** — loop iterates backward for newest-first
- **`pruneOldRequests()` no longer called from TickMsg** — age filtering is inline in `syncFromNetLog`
- **`RequestCompletedMsg` handler remains** — still works for direct injection (tests use it)

### Existing test compatibility

Tests that inject `RequestCompletedMsg` without a subsequent `TickMsg` still pass because
`syncFromNetLog` only runs on tick. Tests that send `TickMsg` after `RequestCompletedMsg`
(like `TestRequestFlowPane_View_RequestAgedOutOnTick`) also pass because the store-based
rebuild produces the same "aged out" result.

---

## Files

- `internal/ui/panes/requestflow_pane.go` — Add `syncFromNetLog()`, update `TickMsg` handler
- `internal/ui/panes/requestflow_pane_test.go` — Add test: populate store net log → send TickMsg → verify View contains request entries

---

## Acceptance Criteria

- [ ] Request Flow pane shows live per-request entries (endpoint, status, latency, arrow animation)
- [ ] Entries age out after `requestAgeOut` (5 seconds)
- [ ] Maximum `maxRecentReqs` (6) entries displayed
- [ ] Gateway state and polling state continue rendering correctly
- [ ] Existing tests pass without modification
- [ ] New test verifies store-sourced request display
- [ ] `make ci` passes
