# Feature 51 — Page B: Request Flow + Network Log

> **Feature:** Build the two Page B panes: `RequestFlowPane` (live APP→GATEWAY→SPOTIFY
> flow visualization) and `NetworkLogPane` (scrollable API request history table).
> These provide developer visibility into Spotnik's internal request pipeline.

## Context

Page B ("Nerd Status") is toggled via key `0`. It shows the NowPlaying compact strip
(row 1) plus two new panes below:

- **Request Flow** — Live animated visualization of requests flowing from APP through
  GATEWAY to SPOTIFY, showing token bucket state, semaphore, backoff, and dedup
- **Network Log** — Scrollable table of all API requests from `store.NetLogEntries()`

All data is internal — no new Spotify API calls. These panes read from `*Gateway`
(token bucket, semaphore, inflight map) and `*Store` (net log entries, throttle state,
fetching sentinels, staleness timestamps).

**Design reference:** `docs/DESIGN.md` §19 (Page B — Nerd Status Specification, full detail
on both panes, animation design, data sources)

**Depends on:** Feature 41 (Pane interface), Feature 42 (border renderer), Feature 43 (Table),
Feature 49 (app migration — pane registration)

---

## Design Diagram

```
Page B Layout:

╭─ ¹Now Playing ── Martbaan · Samar Mehdi ── ▶ 1:41/5:30 ──────────────╮  Row 1 (wt 1)
│  ████████████░░░░░░░  |<  ||  >|  ~  =>   VOL ████░░ 65%             │
╰──────────────────────────────────────────────────────────────────────╯

╭─ Request Flow ───────────────────────────────────────────────────────╮  Row 2 (wt 3)
│                                                                      │
│   APP                          GATEWAY                  SPOTIFY      │
│  ╭──────────────╮           ╭──────────────────╮     ╭────────────╮  │
│  │ ▶ /player    │───────→───│ ●●●●●●●●○○ 8/10  │──→──│ 200  45ms │  │
│  │   /queue     │───→ dedup │ ■■■■■□□□□□  3/5  │  ╳  │ 200  62ms │  │
│  │   /playlists │─── wait ──│ ⏳ backoff  2.1s  │──→──│ 429  12ms │  │
│  ╰──────────────╯           ╰──────────────────╯     ╰────────────╯  │
│                                                                      │
│  POLLING  tick: 1s  state: active  idle: 0s                          │
│  STORE  fetching: [playlists, queue]  stale: albums(12s)             │
╰──────────────────────────────────────────────────────────────────────╯

╭─ Network Log ────────────────────── ᐅf filter ── ᐅj/k scroll ───────╮  Row 3 (wt 2)
│  TIME      METHOD  ENDPOINT                STATUS  LATENCY  NOTES   │
│  12:03:45  GET     /me/player              200     45ms     ██      │
│  12:03:45  GET     /me/player/queue        200     62ms     ███     │
│  12:03:44  GET     /me/playlists           200     128ms    ██████  │
│  12:03:43  GET     /me/player              429     12ms     █  ⚠    │
│  12:03:42  GET     /me/top/tracks          200     95ms     ████    │
│  ▼ more below (200 entry ring buffer)                                │
╰──────────────────────────────────────────────────────────────────────╯
```

---

## Task 1: Expose Gateway observability state

**Problem:** `Gateway` fields (bucket, semaphore, inflight) are private. The RequestFlowPane
needs read access to visualize gateway state.

**Fix:**

Add observability methods to `internal/api/gateway.go`:

```go
// GatewayState holds a snapshot of gateway internal state for display.
type GatewayState struct {
    TokensAvailable  int   // Current token bucket level (0-10)
    TokensMax        int   // Token bucket capacity (10)
    ConcurrentActive int   // Current in-flight request count
    ConcurrentMax    int   // Semaphore capacity (5)
    BackoffRemaining float64 // Seconds until backoff clears (0 if not in backoff)
    DedupWaiters     int   // Number of requests waiting on dedup
    InFlightKeys     []RequestKey // Currently in-flight request keys
}

// Snapshot returns a read-only snapshot of gateway state.
// Thread-safe — acquires lock internally.
func (g *Gateway) Snapshot() GatewayState
```

**Files:**
- Modify: `internal/api/gateway.go`

**Tests:**
- Unit: Snapshot returns correct token count after requests
- Unit: Snapshot returns correct concurrent count
- Unit: Snapshot shows backoff remaining during 429 backoff
- Unit: Snapshot is thread-safe (concurrent access)

**Commit:** `feat(api): add Gateway.Snapshot() for observability`

---

## Task 2: Create RequestFlowPane

**Problem:** No visualization of the request pipeline exists.

**Fix:**

Create `internal/ui/panes/requestflow_pane.go`:

```go
type RequestFlowPane struct {
    theme      theme.Theme
    gateway    *api.Gateway
    store      *state.Store
    width      int
    height     int
    focused    bool
    frameIndex int           // animation frame counter
    recentReqs []reqDisplay  // last N requests with state
}

type reqDisplay struct {
    endpoint    string
    priority    string       // "Interactive" or "Background"
    state       string       // "flowing", "wait", "dedup", "blocked"
    statusCode  int
    latencyMs   int
    age         time.Duration // how long ago this request completed
}
```

**Pane interface:**
```go
func (r *RequestFlowPane) ID() layout.PaneID       { return layout.PaneRequestFlow }
func (r *RequestFlowPane) Title() string            { return "Request Flow" }
func (r *RequestFlowPane) ToggleKey() int           { return 0 } // not toggleable
func (r *RequestFlowPane) Actions() []layout.Action { return nil }
```

**View() renders three columns:**

1. **APP column** (left): Last 4-6 endpoint names, newest at top
   - Active: `▶ /endpoint` in TextPrimary
   - Interactive priority: bright text; Background: TextMuted
   - Completed (>3s old): dimmed

2. **GATEWAY column** (center):
   - Token bucket: `●●●●●●●●○○ 8/10` (filled = available)
   - Semaphore: `■■■■■□□□□□ 3/5` (filled = in-use)
   - Backoff: `⏳ backoff 2.1s` (hidden when 0)
   - Dedup: `N waiters` when active

3. **SPOTIFY column** (right): Response status + latency
   - 2xx: `Success()` color
   - 429: `Warning()` color with ⚠
   - 5xx: `Error()` color

4. **Connecting arrows** between columns:
   - `───→───` flowing through
   - `─── wait ──` queued at semaphore
   - `───→ dedup` hitting dedup
   - `╳` blocked by backoff

5. **Bottom status strip:**
   - Left: `POLLING  tick: 1000ms  state: active|idle  idle: 0s|45s`
   - Right: `STORE  fetching: [playlists, queue]  stale: albums(12s)`

**Animation (200ms tick):**
- Arrow characters shift right: `─→─` → `──→` → `→──`
- Token bucket dots refill animation
- Backoff timer decrements on 1s tick

**Update():**
- `TickMsg` (1s): refresh gateway snapshot, store state, age out old requests
- `VisualizerTickMsg` (200ms): advance arrow animation frameIndex

**Files:**
- Create: `internal/ui/panes/requestflow_pane.go`

**Tests:**
- Unit: Interface satisfaction: `var _ layout.Pane = &RequestFlowPane{}`
- Unit: View renders 3 columns (APP, GATEWAY, SPOTIFY)
- Unit: Token bucket bar shows correct filled/empty ratio
- Unit: Semaphore bar shows correct in-use/available ratio
- Unit: Backoff timer visible when store is throttled, hidden when not
- Unit: Arrow animation advances on VisualizerTickMsg
- Unit: Recent requests fade after 3 seconds
- Unit: Status strip shows polling state
- Unit: Status strip shows store fetching sentinels
- Unit: Color coding: 200=green, 429=yellow, 500=red

**Commit:** `feat(ui): RequestFlowPane with live gateway visualization`

---

## Task 3: Create NetworkLogPane

**Problem:** No scrollable API request log exists.

**Fix:**

Create `internal/ui/panes/networklog_pane.go`:

```go
type NetworkLogPane struct {
    store   *state.Store
    theme   theme.Theme
    table   components.Table
    filter  *components.Filter
    focused bool
    width   int
    height  int
}
```

**Pane interface:**
```go
func (n *NetworkLogPane) ID() layout.PaneID       { return layout.PaneNetworkLog }
func (n *NetworkLogPane) Title() string            { return "Network Log" }
func (n *NetworkLogPane) ToggleKey() int           { return 0 }
func (n *NetworkLogPane) Actions() []layout.Action {
    if n.filter.IsActive() {
        return []layout.Action{{Key: "Esc", Label: "close"}}
    }
    return []layout.Action{{Key: "f", Label: "filter"}, {Key: "j/k", Label: "scroll"}}
}
```

**Table columns:** TIME | METHOD | ENDPOINT | STATUS | LATENCY | NOTES

**Data source:** `store.NetLogEntries()` — 200-entry ring buffer. Each entry has
`Timestamp`, `Method`, `Path`, `StatusCode`, `DurationMs`.

**Color coding per status:**
- 2xx: `Success()` color on status column
- 429: `Warning()` color + `⚠` in notes column
- Other 4xx: `TextMuted()`
- 5xx: `Error()` color

**Latency bar (NOTES column):** Inline `█` characters (1-10) proportional to response time.
Max latency for scaling: 200ms (anything above gets full 10 bars).

**Ordering:** Newest entries at top (reverse chronological).

**Filter:** By endpoint path or status code.

**Files:**
- Create: `internal/ui/panes/networklog_pane.go`

**Tests:**
- Unit: Interface satisfaction: `var _ layout.Pane = &NetworkLogPane{}`
- Unit: Table shows 6 columns with correct headers
- Unit: Entries sorted newest-first
- Unit: 2xx entries in green, 429 in yellow with ⚠, 5xx in red
- Unit: Latency bar proportional to response time
- Unit: Filter by endpoint (e.g., "/me/player")
- Unit: Filter by status code (e.g., "429")
- Unit: Scrolling works with j/k
- Unit: Empty log → clean empty state
- Unit: 200 entries → scrolling through full buffer

**Commit:** `feat(ui): NetworkLogPane with scrollable API request history`

---

## Task 4: Register Page B panes in App

**Problem:** RequestFlowPane and NetworkLogPane need to be registered in the app.

**Fix:**

1. Update `App.New()` to create and register both panes:
   ```go
   panes[layout.PaneRequestFlow] = panes.NewRequestFlowPane(gateway, store, theme)
   panes[layout.PaneNetworkLog]  = panes.NewNetworkLogPane(store, theme)
   ```

2. The RequestFlowPane receives the `*Gateway` reference for `Snapshot()` calls.

3. Route messages: both panes handle `TickMsg` and `VisualizerTickMsg`.

**Files:**
- Modify: `internal/app/app.go`

**Tests:**
- Unit: Page B shows 3 panes (NowPlaying compact + RequestFlow + NetworkLog)
- Unit: Key `0` switches to Page B with correct layout
- Unit: TickMsg reaches both Page B panes
- Unit: Gateway state reflected in RequestFlowPane

**Commit:** `feat(app): register Page B panes (RequestFlow + NetworkLog)`

---

## Task 5: Comprehensive tests

**Files:**
- Create: `internal/ui/panes/requestflow_pane_test.go`
- Create: `internal/ui/panes/networklog_pane_test.go`

**Tests:**
- Integration: Full Page B lifecycle — toggle to Page B → verify 3 panes visible
- Integration: RequestFlowPane — simulate gateway activity → verify visualization updates
- Integration: NetworkLogPane — add log entries → verify table updates
- Integration: Page switch preserves pane state on both pages
- Integration: RequestFlowPane animation — multiple VisualizerTickMsg → arrows shift
- Integration: NetworkLogPane filter — filter by "429" → only 429 entries shown
- Edge: Gateway with no activity → RequestFlowPane shows empty/idle state
- Edge: Empty net log → NetworkLogPane shows clean state
- Edge: Backoff active → backoff timer visible, blocked arrows shown

**Commit:** `test(ui): comprehensive Page B pane tests`

---

## Acceptance Criteria

- [ ] `Gateway.Snapshot()` provides thread-safe read access to internal state
- [ ] `RequestFlowPane` satisfies `layout.Pane`, shows 3 columns (APP/GATEWAY/SPOTIFY)
- [ ] Token bucket bar, semaphore bar, backoff timer render correctly
- [ ] Arrow animation advances on 200ms tick
- [ ] Request states visible: flowing, wait, dedup, blocked
- [ ] Status strip shows polling and store state
- [ ] `NetworkLogPane` satisfies `layout.Pane`, shows scrollable table
- [ ] Log entries color-coded by status (2xx green, 429 yellow, 5xx red)
- [ ] Latency bars proportional to response time
- [ ] Filter works on endpoint and status code
- [ ] Both panes registered and visible on Page B
- [ ] `make ci` passes

---

## Notes

- RequestFlowPane needs access to `*Gateway` — this is the only pane that imports from
  `api/` indirectly (via the Gateway reference). This is acceptable because the Gateway
  is an infrastructure component, not an API client. The pane doesn't make API calls.
- The animation tick (200ms `VisualizerTickMsg`) is shared with the NowPlaying visualizer.
  Both panes handle the same message type. This is intentional — one tick drives all animations.
- Net log data comes from `store.NetLogEntries()` which is already populated by the Gateway's
  response logging. No new logging infrastructure needed.
- The RequestFlowPane is the most visually complex component in the app. Implementation should
  start with a static layout (no animation), then add animation incrementally.
