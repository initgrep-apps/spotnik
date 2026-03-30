# Feature 61 — Fix Request Flow Gateway Visualization

> **Enhancement:** The Request Flow pane renders the basic 3-column layout but is missing
> the core value: **per-request gateway decision visualization**. All requests appear as
> generic animated arrows because the NetLog only captures HTTP-level data — it has no
> knowledge of what the gateway decided for each request (allowed, waited, deduped, blocked).
> Additionally, the pane uses no theme colors and is missing staleness display.

## Background

DESIGN.md §19 specifies a rich visualization where each request's arrow shows its gateway
decision: `───────→───` (flowing), `─── wait ──` (semaphore full), `───→ dedup` (dedup hit),
`── ╳ ──` (blocked by backoff). Status codes should be color-coded (2xx=green, 429=yellow,
5xx=red), and the status strip should show data staleness.

Feature 56 fixed the empty request list by syncing from NetLog, but NetLog only records
`Method, Path, StatusCode, DurationMs` — no priority, no gateway decision. Background
requests rejected by backoff (which never make an HTTP call) are completely invisible.

**Depends on:** Feature 56 (Fix Request Flow Data)

---

## Gap Summary

| # | Gap | Severity | Description |
|---|-----|----------|-------------|
| G1 | No per-request gateway decisions | Critical | NetLogEntry lacks Priority + GatewayDecision fields. Background requests rejected by backoff never appear in UI. |
| G2 | No theme colors | High | Entire pane is plain text — no lipgloss styling for priorities, status codes, or gateway state bars |
| G3 | Arrow states incomplete | High | Only 2 types (normal + 429 block). Design specifies 4: flowing, wait, dedup, blocked |
| G4 | Status strip missing staleness | Medium | No `stale: albums(12s)` display — only shows fetching sentinels |
| G5 | InFlightKeys not rendered | Low | GatewayState.InFlightKeys captured by Snapshot() but never displayed |

---

## Task 1: Add `GatewayDecision` type and extend NetLogEntry

**Problem:** The data layer has no concept of gateway decisions. `NetLogEntry` only tracks
HTTP-level metadata. Background requests rejected by backoff never reach the HTTP layer
and are completely invisible.

**Fix:**

1. Add `GatewayDecision` enum to `internal/domain/gateway.go`:
   ```go
   // GatewayDecision classifies the outcome of a request's passage through the gateway.
   type GatewayDecision int

   const (
       // DecisionAllowed means the request passed through the gateway normally.
       DecisionAllowed GatewayDecision = iota
       // DecisionWaited means the request waited at the token bucket or semaphore.
       DecisionWaited
       // DecisionDeduped means the request joined an existing in-flight GET (dedup hit).
       DecisionDeduped
       // DecisionBlocked means the request was rejected by 429 backoff (Background only).
       DecisionBlocked
   )
   ```

2. Extend `NetLogEntry` in `internal/state/netlog.go` with two new fields:
   ```go
   type NetLogEntry struct {
       Timestamp       time.Time
       Method          string
       Path            string
       StatusCode      int
       DurationMs      int64
       Priority        domain.RequestPriority  // NEW
       GatewayDecision domain.GatewayDecision  // NEW
   }
   ```
   Zero-valued defaults (`PriorityBackground` + `DecisionAllowed`) preserve backward
   compatibility — existing code that creates `NetLogEntry` without these fields compiles
   unchanged.

3. Add `RecordGatewayCall()` to `internal/state/store.go`:
   ```go
   // RecordGatewayCall records a gateway-level API call with decision metadata.
   func (s *Store) RecordGatewayCall(method, path string, statusCode int, durationMs int64,
       priority domain.RequestPriority, decision domain.GatewayDecision) {
       s.netLog.Add(state.NetLogEntry{
           Timestamp:       time.Now(),
           Method:          method,
           Path:            path,
           StatusCode:      statusCode,
           DurationMs:      durationMs,
           Priority:        priority,
           GatewayDecision: decision,
       })
   }
   ```
   Existing `RecordNetCall()` remains unchanged (used by `LoggingTransport`).

**Files:**
- Modify: `internal/domain/gateway.go` — add GatewayDecision type + constants
- Modify: `internal/state/netlog.go` — add Priority + GatewayDecision fields to NetLogEntry
- Modify: `internal/state/store.go` — add RecordGatewayCall() method

**Tests:**
- Unit: `NetLogEntry` with Priority/GatewayDecision round-trips through ring buffer
- Unit: `RecordGatewayCall()` populates all fields correctly
- Unit: Existing `RecordNetCall()` still works (backward compat)

**Commit:** `feat(domain): add GatewayDecision type and extend NetLogEntry`

---

## Task 2: Instrument Gateway.Do() with decision recording

**Problem:** The gateway makes per-request decisions (allow, dedup, block) but never records
them. `LoggingTransport` operates at the HTTP layer, below the gateway, so it cannot see
gateway-level decisions. Background requests rejected by backoff never make HTTP calls and
are completely invisible.

**Fix:**

1. Define `GatewayRecorder` interface in `internal/api/gateway.go`:
   ```go
   // GatewayRecorder records per-request gateway decisions for visualization.
   type GatewayRecorder interface {
       RecordGatewayCall(method, path string, statusCode int, durationMs int64,
           priority domain.RequestPriority, decision domain.GatewayDecision)
   }
   ```

2. Add `recorder` field + `SetRecorder()` to Gateway struct:
   ```go
   type Gateway struct {
       // ... existing fields ...
       recorder GatewayRecorder // optional, for request flow visualization
   }

   // SetRecorder sets the gateway decision recorder. Pass nil to disable.
   func (g *Gateway) SetRecorder(r GatewayRecorder) {
       g.mu.Lock()
       defer g.mu.Unlock()
       g.recorder = r
   }
   ```

3. Instrument `Gateway.Do()` at each decision point:
   - **Background blocked by backoff** (line ~244): Record `DecisionBlocked` with
     StatusCode=0, DurationMs=0 immediately before returning `RateLimitError`
   - **GET dedup waiter** (line ~260-273): Record `DecisionDeduped` after the dedup wait
     completes, using the shared response's status code and total wait duration
   - **Normal completion** (after line ~319): Record `DecisionAllowed` after `fn()`
     completes, with actual status code and HTTP duration

4. Prevent double-recording: Add a context key `gatewayRecordedKey` set by `Gateway.Do()`.
   `LoggingTransport.RoundTrip()` checks this key and skips its own `RecordNetCall()` when
   the gateway has already recorded the request.

5. Wire in `internal/app/auth.go`: Call `gateway.SetRecorder(store)` after gateway creation
   in `initAPIClients()`.

**Files:**
- Modify: `internal/api/gateway.go` — add GatewayRecorder interface, recorder field,
  SetRecorder(), instrument Do() at 3 decision points
- Modify: `internal/api/logging.go` — check gatewayRecordedKey to skip double-recording
- Modify: `internal/app/auth.go` — wire `gateway.SetRecorder(store)`

**Tests:**
- Unit: Background request during backoff → recorder receives `DecisionBlocked`
- Unit: GET dedup waiter → recorder receives `DecisionDeduped`
- Unit: Normal request → recorder receives `DecisionAllowed`
- Unit: `LoggingTransport` skips recording when gateway-recorded context is set
- Unit: Existing gateway tests pass unchanged

**Commit:** `feat(api): instrument Gateway.Do() with per-request decision recording`

---

## Task 3: Theme color coding in Request Flow pane

**Problem:** The entire pane renders as plain text with no lipgloss styling. DESIGN.md §19
specifies theme-colored status codes, priority-colored endpoints, and styled gateway bars.

**Fix:**

Apply lipgloss styles throughout `requestflow_pane.go` View() methods:

| Element | Style |
|---------|-------|
| Column headers (APP/GATEWAY/SPOTIFY) | `p.theme.TextSecondary()` bold |
| Interactive priority requests in APP column | `p.theme.TextPrimary()` |
| Background priority requests in APP column | `p.theme.TextMuted()` |
| Aged requests (>3s) | `p.theme.TextMuted()` regardless of priority |
| 2xx status codes in SPOTIFY column | `p.theme.Success()` |
| 429 status codes | `p.theme.Warning()` |
| 5xx status codes | `p.theme.Error()` |
| 0 status (blocked/no HTTP call) | `p.theme.TextMuted()` |
| Token bucket filled dots (●) | `p.theme.Success()` |
| Semaphore filled squares (■) | `p.theme.Warning()` |
| Backoff timer text | `p.theme.Error()` |
| Dedup label | `p.theme.TextSecondary()` |
| Status strip labels (POLLING/STORE) | `p.theme.TextSecondary()` |
| Status strip values | `p.theme.TextMuted()` |

**Files:**
- Modify: `internal/ui/panes/requestflow_pane.go` — add lipgloss.NewStyle() calls in all
  render methods using Theme interface tokens

**Tests:**
- Unit: View() output contains ANSI escape sequences (non-plain text)
- Unit: Existing string-contains tests still pass (ANSI wraps content, doesn't remove it)

**Commit:** `feat(ui): add theme color coding to Request Flow pane`

---

## Task 4: Four arrow states for gateway decisions

**Problem:** `renderArrow()` only has 2 states: normal animated arrow and 429 block (`── ╳ ─`).
DESIGN.md §19 specifies 4 arrow types representing the 4 gateway decisions.

**Fix:**

1. Add `decision` field to `reqDisplay` struct:
   ```go
   type reqDisplay struct {
       endpoint    string
       statusCode  int
       latencyMs   int
       priority    domain.RequestPriority
       decision    domain.GatewayDecision  // NEW
       completedAt time.Time
   }
   ```

2. Update `syncFromNetLog()` to read decision from NetLogEntry:
   ```go
   p.recentReqs = append(p.recentReqs, reqDisplay{
       endpoint:    e.Path,
       statusCode:  e.StatusCode,
       latencyMs:   int(e.DurationMs),
       priority:    e.Priority,           // was: domain.PriorityBackground
       decision:    e.GatewayDecision,    // NEW
       completedAt: e.Timestamp,
   })
   ```

3. Replace `renderArrow()` logic with decision-based switch:
   ```go
   func (p *RequestFlowPane) renderArrow(r reqDisplay, colWidth int) string {
       frames := []string{"──→──", "───→─", "────→"}
       var arrow string
       switch r.decision {
       case domain.DecisionAllowed:
           if r.statusCode == 429 {
               arrow = "── ╳ ─"   // Warning color
           } else {
               arrow = frames[p.frameIndex%3]  // Success color (animated)
           }
       case domain.DecisionWaited:
           arrow = "── wait ──"   // Warning color
       case domain.DecisionDeduped:
           arrow = "──→ dedup"    // TextSecondary color
       case domain.DecisionBlocked:
           arrow = "── ╳ ──"     // Error color
       default:
           arrow = frames[p.frameIndex%3]
       }
       return padRight(arrow, colWidth)
   }
   ```

4. Update `RequestCompletedMsg` handler to accept decision field (for test injection).

**Files:**
- Modify: `internal/ui/panes/requestflow_pane.go` — update reqDisplay, syncFromNetLog,
  renderArrow, RequestCompletedMsg handler

**Tests:**
- Unit: Inject reqDisplay with `DecisionAllowed` → animated arrow text appears
- Unit: Inject reqDisplay with `DecisionWaited` → "wait" text appears
- Unit: Inject reqDisplay with `DecisionDeduped` → "dedup" text appears
- Unit: Inject reqDisplay with `DecisionBlocked` → "╳" text appears with Error color
- Unit: `DecisionAllowed` + StatusCode 429 → "╳" with Warning color

**Commit:** `feat(ui): implement four arrow states for gateway decisions`

---

## Task 5: Staleness display in status strip

**Problem:** The status strip only shows `STORE  fetching: [playlists, albums]`. DESIGN.md §19
specifies an additional `stale: albums(12s), recent(45s)` section showing which data
domains have exceeded their TTL.

**Fix:**

1. Add `renderStalenessStatus()` method to `requestflow_pane.go`:
   ```go
   func (p *RequestFlowPane) renderStalenessStatus() string {
       if p.store == nil {
           return ""
       }
       var stale []string
       // Check each domain: if FetchedAt is not zero and data is stale, show elapsed time
       if fa := p.store.PlaylistsFetchedAt(); !fa.IsZero() && state.IsStale(fa, state.PlaylistsTTL) {
           stale = append(stale, fmt.Sprintf("playlists(%ds)", int(time.Since(fa).Seconds())))
       }
       if fa := p.store.AlbumsFetchedAt(); !fa.IsZero() && state.IsStale(fa, state.AlbumsTTL) {
           stale = append(stale, fmt.Sprintf("albums(%ds)", int(time.Since(fa).Seconds())))
       }
       if fa := p.store.LikedTracksFetchedAt(); !fa.IsZero() && state.IsStale(fa, state.LikedTracksTTL) {
           stale = append(stale, fmt.Sprintf("liked(%ds)", int(time.Since(fa).Seconds())))
       }
       if fa := p.store.RecentPlayedFetchedAt(); !fa.IsZero() && state.IsStale(fa, state.RecentlyPlayedTTL) {
           stale = append(stale, fmt.Sprintf("recent(%ds)", int(time.Since(fa).Seconds())))
       }
       if len(stale) == 0 {
           return ""
       }
       return fmt.Sprintf("stale: %s", strings.Join(stale, ", "))
   }
   ```

2. Integrate into `renderStoreStatus()`:
   ```go
   func (p *RequestFlowPane) renderStoreStatus() string {
       // ... existing fetching logic ...
       stalePart := p.renderStalenessStatus()
       if stalePart != "" {
           result += "  " + stalePart
       }
       return result
   }
   ```

**Files:**
- Modify: `internal/ui/panes/requestflow_pane.go` — add renderStalenessStatus(), update
  renderStoreStatus()

**Tests:**
- Unit: Set PlaylistsFetchedAt to 10 minutes ago → View contains "stale: playlists"
- Unit: Set FetchedAt within TTL → "stale:" does not appear
- Unit: Multiple stale domains → comma-separated list appears
- Unit: Never-fetched domain (zero FetchedAt) → does not show as stale

**Commit:** `feat(ui): add staleness display to Request Flow status strip`

---

## Task 6: Render InFlightKeys in gateway state block

**Problem:** `GatewayState.InFlightKeys` is captured by `Snapshot()` but never rendered.
These show which GET requests are currently in-flight through the gateway, giving visibility
into active deduplication.

**Fix:**

In `renderGatewayState()`, after the dedup line, when `snap.InFlightKeys` is non-empty,
render up to 3 keys:
```go
if len(snap.InFlightKeys) > 0 {
    max := 3
    if len(snap.InFlightKeys) < max {
        max = len(snap.InFlightKeys)
    }
    for i := 0; i < max; i++ {
        lines = append(lines, fmt.Sprintf("  → %s", snap.InFlightKeys[i]))
    }
    if len(snap.InFlightKeys) > 3 {
        lines = append(lines, fmt.Sprintf("  … +%d more", len(snap.InFlightKeys)-3))
    }
}
```

Styled with `p.theme.TextMuted()`.

**Files:**
- Modify: `internal/ui/panes/requestflow_pane.go` — update renderGatewayState()

**Tests:**
- Unit: Mock GatewaySnapshotter returns 2 InFlightKeys → both appear in View
- Unit: Mock returns 5 InFlightKeys → 3 shown + "+2 more" truncation
- Unit: Empty InFlightKeys → no in-flight section rendered

**Commit:** `feat(ui): render InFlightKeys in gateway state block`

---

## Task 7: Update documentation

**Fix:**

1. Update `docs/ARCHITECTURE.md` Gateway Observability section to mention:
   - `GatewayDecision` type and its 4 values
   - `GatewayRecorder` interface for per-request decision recording
   - Data flow: Gateway.Do() → RecordGatewayCall() → NetLog → RequestFlowPane

2. Update `docs/features/00-overview.md` with Feature 61 entry

**Files:**
- Modify: `docs/ARCHITECTURE.md` — update Gateway Observability subsection
- Modify: `docs/features/00-overview.md` — add row to feature table

**Commit:** `docs: add Feature 61 gateway visualization to architecture docs`

---

## Acceptance Criteria

- [ ] Gateway decisions (allowed, waited, deduped, blocked) are tracked per-request
- [ ] Background requests rejected by backoff appear in the Request Flow pane
- [ ] Interactive vs Background requests show different colors (bright vs muted)
- [ ] Status codes are color-coded: 2xx=Success, 429=Warning, 5xx=Error
- [ ] Four arrow states render correctly: flowing (animated), wait, dedup, blocked
- [ ] Gateway state bars (token bucket, semaphore) use theme colors
- [ ] Status strip shows stale data domains with elapsed time
- [ ] InFlightKeys are displayed when non-empty
- [ ] LoggingTransport does not double-record gateway-tracked requests
- [ ] All existing tests pass without modification
- [ ] `make ci` passes

---

## Verification

```bash
# New types exist
grep 'GatewayDecision' internal/domain/gateway.go
grep 'DecisionBlocked' internal/domain/gateway.go

# NetLogEntry has new fields
grep 'GatewayDecision' internal/state/netlog.go

# Gateway records decisions
grep 'GatewayRecorder' internal/api/gateway.go

# Arrow states in pane
grep 'DecisionDeduped\|DecisionWaited\|DecisionBlocked' internal/ui/panes/requestflow_pane.go

# Theme styling in pane
grep 'p.theme' internal/ui/panes/requestflow_pane.go

# Staleness in status strip
grep 'stale:' internal/ui/panes/requestflow_pane.go

# Full CI passes
make ci
```

---

*Depends on: Feature 56*
*Blocks: Nothing*
