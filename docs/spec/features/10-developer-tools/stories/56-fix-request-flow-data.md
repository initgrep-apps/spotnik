---
title: "Fix Request Flow Pane Data"
feature: 14-nerd-status
status: done
---

## Background
Request Flow pane shows only static gateway/polling info -- the per-request flow visualization (APP -> GATEWAY -> SPOTIFY) never displays because RequestCompletedMsg is never emitted. The handler exists in RequestFlowPane.Update() but no code ever emits RequestCompletedMsg. The recentReqs slice is always empty. Meanwhile, the Store's network log successfully records all API calls via RecordNetCall() -- the Network Log pane reads from it and works perfectly.

## Design
On each TickMsg, populate recentReqs from the Store's existing NetLogEntries(). This reuses the working infrastructure without the invasive approach of emitting RequestCompletedMsg from every API response handler.

### syncFromNetLog() Method
```go
func (p *RequestFlowPane) syncFromNetLog() {
    entries := p.store.NetLogEntries()
    cutoff := time.Now().Add(-requestAgeOut)
    p.recentReqs = p.recentReqs[:0]
    for i := len(entries) - 1; i >= 0; i-- {
        e := entries[i]
        if e.Timestamp.Before(cutoff) { continue }
        p.recentReqs = append(p.recentReqs, reqDisplay{
            endpoint: e.Path, statusCode: e.StatusCode,
            latencyMs: int(e.DurationMs), priority: domain.PriorityBackground,
            completedAt: e.Timestamp,
        })
        if len(p.recentReqs) >= maxRecentReqs { break }
    }
}
```

Key details: NetLogEntries() returns oldest-first, loop iterates backward. Priority defaults to Background (net log doesn't track priority). RequestCompletedMsg handler remains for direct injection (tests).

## Acceptance Criteria
- [ ] Request Flow pane shows live per-request entries
- [ ] Entries age out after requestAgeOut (5 seconds)
- [ ] Maximum maxRecentReqs (6) entries displayed
- [ ] Gateway state and polling state continue rendering
- [ ] Existing tests pass without modification
- [ ] New test verifies store-sourced request display
- [ ] make ci passes

## Tasks
- [ ] Add syncFromNetLog() method and update TickMsg handler in requestflow_pane.go
      - test: populate store net log -> send TickMsg -> View contains request entries; entries age out; max 6; existing tests unchanged
