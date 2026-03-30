---
title: "Nerd Status"
description: "Developer visibility into Spotnik's internal request pipeline via Page B, featuring a live Request Flow visualization of gateway decisions and a scrollable Network Log of all API requests, both powered by a gateway event journal with replay engine."
status: done
stories: [51, 61, 62, 64, 66, 67, 68, 69]
---

# Nerd Status

## Background

Spotnik is a terminal Spotify client for developers, and developers want to see what their tools are doing under the hood. The Nerd Status feature provides full developer visibility into Spotnik's internal request pipeline through Page B, toggled via key `0`. Page B shows the NowPlaying compact strip (row 1) plus two dedicated panes below: the Request Flow pane and the Network Log pane. Together these panes expose every gateway decision, every HTTP call, and every internal state change that occurs between the app and Spotify's API.

The Request Flow pane is the visual centerpiece -- a live animated visualization of requests flowing from APP through GATEWAY to SPOTIFY. It renders three bordered sub-boxes connected by animated arrows, showing the token bucket state, concurrent semaphore, backoff timers, dedup behavior, and per-request gateway decisions (allowed, waited, deduped, blocked). Over the course of development, this pane evolved from a flat column layout polling snapshots at 1-second intervals to a rich event-driven replay engine that consumes fine-grained lifecycle events from a gateway event journal, replaying them at human-observable speed (200ms per event) so that even microsecond-level decisions become visible.

The Network Log pane is a scrollable table of all API requests, sourced from the same gateway event journal. It shows timestamp, method, endpoint, status code, latency, priority (interactive vs background), and gateway decision for every request -- including background requests blocked by backoff that never reached the HTTP layer. The underlying data infrastructure progressed from a simple NetLog ring buffer with flat records to a cursor-based GatewayEventLog with 13 distinct event kinds, state snapshots embedded in every event, and independent consumer cursors. The old NetLog system was fully retired once both panes migrated to the event journal.

---

## Story: Page B: Request Flow + Network Log (spec 51)

### Background
This story built the two foundational Page B panes from scratch. Page B ("Nerd Status") is toggled via key `0` and shows the NowPlaying compact strip plus two new panes: RequestFlowPane (live animated visualization of the APP-GATEWAY-SPOTIFY request pipeline) and NetworkLogPane (scrollable table of API request history). All data is internal -- no new Spotify API calls. The panes read from `*Gateway` (token bucket, semaphore, inflight map) via a new `Snapshot()` method and from `*Store` (net log entries, throttle state, fetching sentinels, staleness timestamps).

### Acceptance Criteria
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

### Tasks
1. **Expose Gateway observability state** -- Add observability methods to `internal/api/gateway.go`: a `GatewayState` struct with `TokensAvailable`, `TokensMax`, `ConcurrentActive`, `ConcurrentMax`, `BackoffRemaining`, `DedupWaiters`, `InFlightKeys` fields, and a thread-safe `Gateway.Snapshot()` method that returns a read-only snapshot.
   - Files: `internal/api/gateway.go`
   - Tests: Snapshot returns correct token count after requests; correct concurrent count; backoff remaining during 429 backoff; thread-safe concurrent access

2. **Create RequestFlowPane** -- Create `internal/ui/panes/requestflow_pane.go` implementing `layout.Pane` with ID `PaneRequestFlow`. View() renders three columns: APP column (endpoint names with priority coloring), GATEWAY column (token bucket bar, semaphore bar, backoff timer, dedup waiters), SPOTIFY column (status + latency with color coding). Connecting arrows between columns animate on 200ms `VisualizerTickMsg`. Bottom status strip shows polling state and store fetching sentinels/staleness. Update() handles `TickMsg` (1s) for gateway snapshot refresh and `VisualizerTickMsg` (200ms) for arrow animation.
   - Files: `internal/ui/panes/requestflow_pane.go`
   - Tests: Interface satisfaction; 3 columns render; token bucket bar ratio; semaphore bar ratio; backoff visibility; arrow animation advancement; request fade after 3s; status strip shows polling state and fetching sentinels; color coding (200=green, 429=yellow, 500=red)

3. **Create NetworkLogPane** -- Create `internal/ui/panes/networklog_pane.go` implementing `layout.Pane` with ID `PaneNetworkLog`. Table columns: TIME, METHOD, ENDPOINT, STATUS, LATENCY, NOTES. Data from `store.NetLogEntries()` (200-entry ring buffer). Color coding per status. Latency bar in NOTES column (1-10 `█` characters, max 200ms). Newest entries at top. Filter by endpoint path or status code.
   - Files: `internal/ui/panes/networklog_pane.go`
   - Tests: Interface satisfaction; 6 columns with correct headers; newest-first sorting; color coding (2xx green, 429 yellow with warning, 5xx red); latency bar proportional; filter by endpoint; filter by status code; j/k scrolling; empty log clean state; 200 entries full buffer scrolling

4. **Register Page B panes in App** -- Update `App.New()` to create and register both panes. RequestFlowPane receives `*Gateway` reference for `Snapshot()` calls. Route `TickMsg` and `VisualizerTickMsg` to both panes.
   - Files: `internal/app/app.go`
   - Tests: Page B shows 3 panes (NowPlaying compact + RequestFlow + NetworkLog); key `0` switches to Page B; TickMsg reaches both panes; gateway state reflected in RequestFlowPane

5. **Comprehensive tests** -- Integration and edge case tests for the full Page B lifecycle.
   - Files: `internal/ui/panes/requestflow_pane_test.go`, `internal/ui/panes/networklog_pane_test.go`
   - Tests: Full Page B lifecycle (toggle, verify 3 panes); RequestFlowPane gateway activity simulation; NetworkLogPane table updates on log entries; page switch preserves pane state; animation across multiple VisualizerTickMsg; filter by "429"; gateway with no activity shows idle state; empty net log clean state; backoff active shows timer and blocked arrows

---

## Story: Fix Request Flow Gateway Visualization (spec 61)

### Background
The Request Flow pane rendered the basic 3-column layout but was missing its core value: per-request gateway decision visualization. All requests appeared as generic animated arrows because the NetLog only captured HTTP-level data with no knowledge of gateway decisions (allowed, waited, deduped, blocked). Background requests rejected by backoff (which never make an HTTP call) were completely invisible. The pane also used no theme colors and was missing staleness display. This story added the `GatewayDecision` type, extended `NetLogEntry` with priority and decision fields, instrumented `Gateway.Do()` with decision recording via a new `GatewayRecorder` interface, applied full theme color coding, implemented four arrow states, added staleness display to the status strip, and rendered InFlightKeys in the gateway state block.

### Acceptance Criteria
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

### Tasks
1. **Add `GatewayDecision` type and extend NetLogEntry** -- Add `GatewayDecision` enum (`DecisionAllowed`, `DecisionWaited`, `DecisionDeduped`, `DecisionBlocked`) to `internal/domain/gateway.go`. Extend `NetLogEntry` in `internal/state/netlog.go` with `Priority` (`domain.RequestPriority`) and `GatewayDecision` (`domain.GatewayDecision`) fields. Add `RecordGatewayCall()` to `internal/state/store.go` for recording with decision metadata. Existing `RecordNetCall()` remains unchanged.
   - Files: `internal/domain/gateway.go`, `internal/state/netlog.go`, `internal/state/store.go`
   - Tests: NetLogEntry with Priority/GatewayDecision round-trips through ring buffer; RecordGatewayCall populates all fields; existing RecordNetCall backward compat

2. **Instrument Gateway.Do() with decision recording** -- Define `GatewayRecorder` interface in `internal/api/gateway.go`. Add `recorder` field and `SetRecorder()` to Gateway. Instrument `Do()` at each decision point: background blocked by backoff records `DecisionBlocked` with StatusCode=0; GET dedup waiter records `DecisionDeduped`; normal completion records `DecisionAllowed`. Add `gatewayRecordedKey` context key so `LoggingTransport.RoundTrip()` skips double-recording. Wire in `internal/app/auth.go` via `gateway.SetRecorder(store)`.
   - Files: `internal/api/gateway.go`, `internal/api/logging.go`, `internal/app/auth.go`
   - Tests: Background request during backoff records DecisionBlocked; GET dedup waiter records DecisionDeduped; normal request records DecisionAllowed; LoggingTransport skips when gateway-recorded context set; existing gateway tests pass

3. **Theme color coding in Request Flow pane** -- Apply lipgloss styles throughout `requestflow_pane.go` View() methods: column headers in TextSecondary bold; interactive requests in TextPrimary, background in TextMuted; aged requests (>3s) in TextMuted; 2xx in Success, 429 in Warning, 5xx in Error, status 0 in TextMuted; token bucket filled dots in Success; semaphore filled squares in Warning; backoff timer in Error; dedup label in TextSecondary; status strip labels in TextSecondary, values in TextMuted.
   - Files: `internal/ui/panes/requestflow_pane.go`
   - Tests: View() output contains ANSI escape sequences; existing string-contains tests still pass

4. **Four arrow states for gateway decisions** -- Add `decision` field to `reqDisplay` struct. Update `syncFromNetLog()` to read decision from NetLogEntry. Replace `renderArrow()` logic with decision-based switch: `DecisionAllowed` shows animated arrow (or `╳` for 429), `DecisionWaited` shows `── wait ──`, `DecisionDeduped` shows `──→ dedup`, `DecisionBlocked` shows `── ╳ ──` in Error color.
   - Files: `internal/ui/panes/requestflow_pane.go`
   - Tests: DecisionAllowed shows animated arrow; DecisionWaited shows "wait" text; DecisionDeduped shows "dedup" text; DecisionBlocked shows "╳" with Error color; DecisionAllowed + StatusCode 429 shows "╳" with Warning color

5. **Staleness display in status strip** -- Add `renderStalenessStatus()` method checking `PlaylistsFetchedAt()`, `AlbumsFetchedAt()`, `LikedTracksFetchedAt()`, `RecentPlayedFetchedAt()` against their TTLs. Display `stale: playlists(Ns), albums(Ns)` in the status strip. Integrate into `renderStoreStatus()`.
   - Files: `internal/ui/panes/requestflow_pane.go`
   - Tests: PlaylistsFetchedAt 10 min ago shows "stale: playlists"; FetchedAt within TTL shows no "stale:"; multiple stale domains show comma-separated list; never-fetched domain (zero FetchedAt) does not show as stale

6. **Render InFlightKeys in gateway state block** -- In `renderGatewayState()`, when `snap.InFlightKeys` is non-empty, render up to 3 keys with `→ keyname` format, plus `… +N more` truncation. Styled with `p.theme.TextMuted()`.
   - Files: `internal/ui/panes/requestflow_pane.go`
   - Tests: Mock returns 2 InFlightKeys, both appear; mock returns 5, 3 shown + "+2 more"; empty InFlightKeys, no section rendered

7. **Update documentation** -- Update `docs/ARCHITECTURE.md` Gateway Observability section with GatewayDecision type, GatewayRecorder interface, data flow description. Update `docs/features/00-overview.md`.
   - Files: `docs/ARCHITECTURE.md`, `docs/features/00-overview.md`

---

## Story: Request Flow Boxed Layout (spec 62)

### Background
The Request Flow pane rendered as a flat text table with three padded columns. DESIGN.md section 19 specified three bordered sub-boxes (APP, GATEWAY, SPOTIFY) connected by animated arrows -- a graphical visualization, not a table. Gateway metrics (token bucket, semaphore, backoff, dedup) rendered as a separate block below all request rows instead of inside the center column alongside them. This story restructured View() to render three independently bordered boxes with rounded corners arranged horizontally, with gateway metrics vertically inside the center GATEWAY box, dual arrow columns (APP-to-GW showing gateway decisions, GW-to-SPOTIFY showing HTTP outcomes), row alignment across all boxes, and a graceful flat fallback for terminals narrower than 60 columns.

### Acceptance Criteria
- [ ] Three bordered sub-boxes (APP, GATEWAY, SPOTIFY) render with rounded corners
- [ ] Gateway metrics (token bucket, semaphore, backoff, dedup) appear inside the center GATEWAY box, not below
- [ ] Two arrow columns connect the boxes: APP->GW (gateway decision) and GW->SPOTIFY (HTTP outcome)
- [ ] Request rows align horizontally across all three boxes and both arrow columns
- [ ] Arrow states match Feature 61: flowing, wait, dedup, blocked (left arrows) and 2xx, 429, 5xx, blocked (right arrows)
- [ ] Theme colors are preserved: all existing color coding from Feature 61 works in the boxed layout
- [ ] Status strip renders below the three boxes spanning full width
- [ ] Flat fallback activates for pane width < 60 columns (identical to current behavior)
- [ ] Staleness display works unchanged in the status strip
- [ ] InFlightKeys render inside the GATEWAY box
- [ ] Animation frames advance correctly in both arrow columns
- [ ] All existing tests pass (updated for new layout) + new tests added
- [ ] `make ci` passes

### Tasks
1. **Create `renderSubBox()` helper** -- Add a method to `requestflow_pane.go` that renders a bordered box with a title label using rounded corners (`╭╮╰╯`). Takes title, content lines, and width. Top border: `╭─ TITLE ──...──╮`, content lines: `│ line... │`, bottom border: `╰──...──╯`. Border color: `theme.TextSecondary()`, title: `theme.TextSecondary()` bold. Width < 8 returns empty string.
   - Files: `internal/ui/panes/requestflow_pane.go`
   - Tests: renderSubBox contains rounded corners and title; content lines padded; long lines truncated with `...`; width < 8 returns empty; empty lines gives border-only box

2. **Create `renderRightArrow()` for GATEWAY->SPOTIFY** -- Add method reflecting HTTP response outcome: 2xx animated flowing arrow (Success), 429 `── ╳ ──` (Warning), 5xx animated arrow (Error), status 0 `── ╳ ──` (TextMuted, blocked). Reuses animation frames and `frameIndex`.
   - Files: `internal/ui/panes/requestflow_pane.go`
   - Tests: StatusCode 200 shows animated arrow with Success; 429 shows "╳" with Warning; 500 shows animated arrow with Error; 0 shows "╳" with TextMuted; arrow width respects colWidth

3. **Build content generators for each sub-box** -- Three methods: `buildAppBoxLines(maxRows)` returns styled endpoint lines per request (active or dimmed, padded to maxRows); `buildGatewayBoxLines(maxRows)` returns gateway metric lines (token bucket, semaphore, backoff when throttled, dedup when active, in-flight keys up to 3, padded); `buildSpotifyBoxLines(maxRows)` returns styled status+latency lines per request.
   - Files: `internal/ui/panes/requestflow_pane.go`
   - Tests: buildAppBoxLines with fewer requests pads; caps at maxRows; buildGatewayBoxLines includes backoff when throttled, omits when not; buildSpotifyBoxLines shows warning for 429, TextMuted for status 0; all handle maxRows=0

4. **Restructure `View()` to render boxed layout** -- Replace core of View() with boxed composition. Calculate proportional column widths (APP ~25%, left arrow ~8%, GATEWAY ~26%, right arrow ~8%, SPOTIFY ~20%). Build content lines and arrows, render three sub-boxes via `renderSubBox()`, compose with `lipgloss.JoinHorizontal(lipgloss.Top, ...)`. Status strip below. Minimum width check: < 60 falls back to `viewFlat()`. Add `buildLeftArrowLines()` and `buildRightArrowLines()`. Rename current View() body to `viewFlat()`.
   - Files: `internal/ui/panes/requestflow_pane.go`
   - Tests: Width 80 shows three bordered boxes (`╭─ APP`, `╭─ GATEWAY`, `╭─ SPOTIFY`); width 40 falls back to flat (no boxes); 3 requests show 3 arrow rows; status strip below boxes; height 5 renders with minimal rows; height 0 returns empty

5. **Align arrow animation with sub-box rows** -- Arrow columns need vertical alignment with content rows inside boxes. Generate exactly `innerRows` lines per arrow block. Prepend/append blank lines to match box border rows. Each arrow line padded to `arrowW` with `padRightVisible()`.
   - Files: `internal/ui/panes/requestflow_pane.go`
   - Tests: Content lines between `╭` and `╰` align with non-blank arrow lines; arrow block line count matches box height; first and last arrow lines are blank

6. **Preserve existing rendering methods** -- `viewFlat()` calls original methods unchanged. New `build*BoxLines()` methods internally call `renderAppEntry()`, `renderArrow()`, `renderSpotifyEntry()`. Extract `gatewayStateLines() []string` from `renderGatewayState()` for reuse by both layouts.
   - Files: `internal/ui/panes/requestflow_pane.go`
   - Tests: renderGatewayState() output unchanged; gatewayStateLines() returns correct line count; viewFlat() matches previous View() output for same state

7. **Update existing tests** -- Tests with `SetSize(80, 20)` now expect bordered box output. Add flat fallback tests at width 40. Add `viewContainsBox(t, output, title)` helper. Preserve arrow state, color coding, staleness, and animation tests.
   - Files: `internal/ui/panes/requestflow_pane_test.go`
   - Tests: TestRequestFlowPane_View_FlatFallback; TestRequestFlowPane_View_BoxedLayout; TestRequestFlowPane_View_BoxedLayout_GatewayInCenter; TestRequestFlowPane_View_BoxedLayout_DualArrows; preserved decision arrow/color/staleness/animation tests

8. **Update documentation** -- Update `docs/ARCHITECTURE.md` with three bordered sub-boxes layout, dual arrow columns, flat fallback, `renderSubBox()` pattern. Update `docs/features/00-overview.md`.
   - Files: `docs/ARCHITECTURE.md`, `docs/features/00-overview.md`

---

## Story: Gateway State Liveness & Peak Watermarks (spec 64)

### Background
The Request Flow pane's gateway metrics (token bucket, concurrent semaphore) appeared static because the snapshot refreshed only every 1 second and transient state changes recovered faster than the polling interval. Token bucket capacity is 10 with a refill rate of 10/sec, so a consumed token recovers in ~100ms. Most HTTP requests complete in <100ms, so the semaphore always showed 0 active. This story added 200ms snapshot refresh on `viz.TickMsg` and peak activity watermarks (`minTokens`, `peakConcurrent`) over 1-second windows, displaying muted annotations like `(min: 8)` and `(peak: 2)` when activity occurred between snapshots.

### Acceptance Criteria
- [ ] Gateway snapshot refreshes on viz.TickMsg (every 200ms), not just TickMsg (1s)
- [ ] Net log syncs on viz.TickMsg so completed requests appear within 200ms
- [ ] `minTokens` tracks the lowest token count seen in the current 1-second window
- [ ] `peakConcurrent` tracks the highest concurrent active count in the current window
- [ ] Watermarks reset to defaults on each TickMsg (1-second boundary)
- [ ] Token line shows `(min: N)` annotation when `minTokens < TokensAvailable`
- [ ] Semaphore line shows `(peak: N)` annotation when `peakConcurrent > ConcurrentActive`
- [ ] No annotations shown when idle (peaks match current values)
- [ ] Existing viz.TickMsg frame advancement still works
- [ ] All existing tests pass
- [ ] `make ci` passes

### Tasks
1. **Refresh gateway snapshot on viz.TickMsg** -- Modify `viz.TickMsg` handler in `Update()` to call `p.gateway.Snapshot()` and `p.syncFromNetLog()` at 200ms resolution in addition to advancing `frameIndex`. `Snapshot()` is cheap (reads under two locks, no allocations beyond struct copy).
   - Files: `internal/ui/panes/requestflow_pane.go`
   - Tests: TestRequestFlowPane_VizTickMsg_RefreshesSnapshot (modify gateway state, send viz.TickMsg, verify lastSnapshot reflects change); TestRequestFlowPane_VizTickMsg_SyncsNetLog (add entry, send viz.TickMsg, verify appears in recentReqs)

2. **Add peak watermark fields to RequestFlowPane** -- Add `peakConcurrent int` and `minTokens int` fields to struct. Initialize `minTokens` to `TokensMax` in `NewRequestFlowPane()`.
   - Files: `internal/ui/panes/requestflow_pane.go`
   - Tests: TestRequestFlowPane_New_InitializesMinTokens (verify minTokens set to TokensMax)

3. **Track and reset watermarks** -- In `viz.TickMsg` handler, after refreshing snapshot: if `ConcurrentActive > peakConcurrent`, update; if `TokensAvailable < minTokens`, update. In `TickMsg` handler, reset `peakConcurrent` to 0 and `minTokens` to `TokensMax` before refreshing snapshot.
   - Files: `internal/ui/panes/requestflow_pane.go`
   - Tests: TestRequestFlowPane_PeakWatermarks_TrackMinTokens; TestRequestFlowPane_PeakWatermarks_TrackPeakConcurrent; TestRequestFlowPane_PeakWatermarks_ResetOnTickMsg

4. **Render peak annotations in gatewayStateLines()** -- In `internal/ui/panes/requestflow_boxed.go`, modify `gatewayStateLines()`: token bucket line appends muted `(min: N)` when `minTokens < TokensAvailable`; semaphore line appends muted `(peak: N)` when `peakConcurrent > ConcurrentActive`. Annotations only appear during activity.
   - Files: `internal/ui/panes/requestflow_boxed.go`
   - Tests: TestGatewayStateLines_PeakAnnotation_Tokens; TestGatewayStateLines_PeakAnnotation_Concurrent; TestGatewayStateLines_NoPeakAnnotation_WhenIdle

5. **Update documentation** -- Update `docs/features/00-overview.md` and `docs/ARCHITECTURE.md` with snapshot refresh on viz.TickMsg, peak watermarks, annotations.
   - Files: `docs/features/00-overview.md`, `docs/ARCHITECTURE.md`

---

## Story: Gateway Event Types & Storage (spec 66)

### Background
The Request Flow pane observed gateway state by polling `Snapshot()` at 200ms-1s intervals, which failed because gateway decisions (token consumption, semaphore acquisition, dedup) happen in microseconds and self-heal before the next sample. The design spec described replacing snapshot polling with an event journal: the Gateway records every internal decision as a timestamped event with a state snapshot, and the UI replays these events at human-observable speed. This story created the foundational domain types and ring buffer storage for the gateway event journal -- no behavioral changes to the Gateway, Request Flow pane, or Network Log pane.

### Acceptance Criteria
- [ ] `EventKind` enum exists with 13 constants in `domain/gateway.go`
- [ ] `GatewayStateSnapshot` struct exists in `domain/gateway.go`
- [ ] `GatewayEvent` struct exists with `Timestamp`, `Kind`, `RequestID`, `Snapshot` etc.
- [ ] `GatewayEventRecorder` interface exists with single `RecordEvent()` method
- [ ] `GatewayEventLog` ring buffer exists in `state/eventlog.go` with `Add()`, `ReadFrom()`, `Len()`
- [ ] Cursor-based reads work correctly (incremental, wraparound, stale cursor recovery)
- [ ] `Store.RecordEvent()` and `Store.ReadEventsFrom()` work correctly
- [ ] `Store` satisfies `domain.GatewayEventRecorder` at compile time
- [ ] All existing tests pass unchanged (no behavioral changes in this feature)
- [ ] `make ci` passes

### Tasks
1. **Add `EventKind` enum to `internal/domain/gateway.go`** -- 13 event kinds covering request lifecycle, resource tracking, and internal housekeeping: `EventRequestEntered`, `EventTokenConsumed`, `EventTokenRefilled`, `EventSemaphoreAcquired`, `EventSemaphoreReleased`, `EventBackoffStarted`, `EventBackoffExpired`, `EventRequestAllowed`, `EventRequestWaited`, `EventRequestBlocked`, `EventDedupJoined`, `EventDedupResolved`, `EventHttpCompleted`.
   - Files: `internal/domain/gateway.go`
   - Tests: All 13 constants have distinct values (table-driven); EventRequestEntered is the zero value

2. **Add `GatewayStateSnapshot` struct** -- Holds a frozen copy of gateway internal state: `TokensAvailable`, `TokensMax`, `ConcurrentActive`, `ConcurrentMax`, `BackoffRemaining`, `DedupWaiters`, `InFlightKeys []string`. Unlike `GatewayState`, has no watermark fields. Exists alongside `GatewayState` temporarily.
   - Files: `internal/domain/gateway.go`
   - Tests: Zero value has expected defaults (0 tokens, 0 concurrent)

3. **Add `GatewayEvent` struct and `GatewayEventRecorder` interface** -- `GatewayEvent` carries `Timestamp`, `Kind`, `RequestID` (uint64, 0 for internal events), `Method`, `Path`, `Priority`, `StatusCode`, `DurationMs`, and embedded `GatewayStateSnapshot`. `GatewayEventRecorder` interface has single `RecordEvent(event GatewayEvent)` method, implemented by `*state.Store`.
   - Files: `internal/domain/gateway.go`
   - Tests: GatewayEvent with all fields populated round-trips correctly; zero value has EventRequestEntered kind and zero RequestID

4. **Add `GatewayEventLog` ring buffer to `internal/state/`** -- Fixed-size ring buffer (default capacity 500) with `Add()` (write lock), `ReadFrom(cursor)` (read lock, cursor-based incremental reads), `Len()`. Monotonically increasing sequence numbers. Stale cursor returns all stored events (graceful recovery).
   - Files: `internal/state/eventlog.go`
   - Tests: TestGatewayEventLog_Add_IncrementsCounts; TestGatewayEventLog_Add_RingWraparound; TestGatewayEventLog_ReadFrom_FirstCall; TestGatewayEventLog_ReadFrom_IncrementalReads; TestGatewayEventLog_ReadFrom_CursorUpToDate; TestGatewayEventLog_ReadFrom_CursorTooOld; TestGatewayEventLog_ReadFrom_EventOrdering; TestGatewayEventLog_ReadFrom_IndependentCursors; TestGatewayEventLog_Add_ZeroCapacity (defaults to 500); TestGatewayEventLog_ConcurrentAccess

5. **Add `RecordEvent()` and `ReadEventsFrom()` to Store** -- Add `eventLog *GatewayEventLog` field to Store struct. Initialize in `NewStore()`. Add `RecordEvent()` (implements `domain.GatewayEventRecorder`) and `ReadEventsFrom(cursor)` methods. Compile-time check: `var _ domain.GatewayEventRecorder = &Store{}`.
   - Files: `internal/state/store.go`
   - Tests: Store.RecordEvent stores event retrievable via ReadEventsFrom(0); Store satisfies GatewayEventRecorder (compile-time); ReadEventsFrom returns incremental events with correct cursor

6. **Update documentation** -- Add Feature 66 row to `docs/features/00-overview.md`. Add GatewayEventLog note in `docs/ARCHITECTURE.md` State Management section.
   - Files: `docs/features/00-overview.md`, `docs/ARCHITECTURE.md`

---

## Story: Gateway Event Instrumentation (spec 67)

### Background
Feature 66 added domain types and storage for the event journal. This story wired the Gateway as the event producer, emitting fine-grained lifecycle events at every decision point in `Do()` and for internal state changes (token refills, backoff expiry). The old `GatewayRecorder` produced one summary record per request; this replacement emits ~5-6 events per request lifecycle, each carrying a snapshot of gateway state at the exact moment. The old `GatewayRecorder`, `Snapshot()`, `ResetWatermarks()`, watermark fields, and double-recording prevention were retired and replaced by the event journal, with `Snapshot()` kept as a deprecated compatibility shim until Feature 68.

### Acceptance Criteria
- [ ] `Gateway.Do()` emits `EventRequestEntered` at entry with unique `RequestID`
- [ ] `Gateway.Do()` emits `EventTokenConsumed` after token bucket wait
- [ ] `Gateway.Do()` emits `EventSemaphoreAcquired`/`EventSemaphoreReleased`
- [ ] `Gateway.Do()` emits `EventRequestBlocked` for background requests during backoff
- [ ] `Gateway.Do()` emits `EventRequestWaited` for interactive requests during backoff
- [ ] `Gateway.Do()` emits `EventDedupJoined`/`EventDedupResolved` for dedup
- [ ] `Gateway.Do()` emits `EventHttpCompleted` with status and latency
- [ ] `Gateway.Do()` emits `EventBackoffStarted` on 429
- [ ] All events carry correct `GatewayStateSnapshot` at the moment of emission
- [ ] All events for the same request share the same `RequestID`
- [ ] `CheckAndEmitRefill()` emits `EventTokenRefilled` when level changes
- [ ] `CheckAndEmitBackoffExpiry()` emits `EventBackoffExpired` on transition
- [ ] Old watermark fields removed from Gateway and tokenBucket
- [ ] `GatewayRecorder` interface removed, replaced by `GatewayEventRecorder`
- [ ] `LoggingTransport` no longer records to net log
- [ ] `MarkGatewayRecorded`/`IsGatewayRecorded` removed
- [ ] `Snapshot()` still works as a deprecated compatibility shim
- [ ] All existing tests pass (updated for removed watermarks)
- [ ] `make ci` passes

### Tasks
1. **Add `emitEvent()` helper and `captureSnapshot()` to Gateway** -- Add `nextRequestID atomic.Uint64` field. Change `recorder` field to `domain.GatewayEventRecorder`. Add `captureSnapshot()` (acquires bucket.mu then g.mu) and `captureSnapshotLocked()` (when g.mu already held, only acquires bucket.mu). Add `emitEvent()` (not holding g.mu) and `emitEventLocked()` (g.mu held) helpers that build `GatewayEvent` with snapshot and call `rec.RecordEvent()`. Lock ordering: g.mu then bucket.mu (standard order).
   - Files: `internal/api/gateway.go`
   - Tests: captureSnapshot returns correct token level with refill; correct ConcurrentActive; emitEvent with nil recorder no panic; emitEvent with recorder calls RecordEvent with correct fields; nextRequestID increments atomically

2. **Instrument `Do()` with lifecycle events** -- Replace all `RecordGatewayCall()` calls with appropriate `emitEvent`/`emitEventLocked` calls. Emit `EventRequestEntered` at top with generated requestID. Background backoff -> `EventRequestBlocked`. Dedup waiter -> `EventDedupJoined` then `EventDedupResolved`. Token consumed -> `EventTokenConsumed`. Semaphore acquired/released -> `EventSemaphoreAcquired`/`EventSemaphoreReleased`. HTTP response -> `EventHttpCompleted`. 429 backoff set -> `EventBackoffStarted`. Normal completion -> `EventRequestAllowed`.
   - Files: `internal/api/gateway.go`
   - Tests: TestGateway_Do_NormalRequest_EmitsLifecycle (RequestEntered -> TokenConsumed -> SemaphoreAcquired -> HttpCompleted -> SemaphoreReleased -> RequestAllowed); TestGateway_Do_BlockedRequest_EmitsBlockedEvent; TestGateway_Do_InteractiveWait_EmitsWaitedEvent; TestGateway_Do_DedupRequest_EmitsJoinAndResolve; TestGateway_Do_429Response_EmitsBackoffStarted; TestGateway_Do_EventsHaveCorrectRequestID; TestGateway_Do_EventsHaveSnapshots; TestGateway_Do_SnapshotReflectsStateAtMoment

3. **Add `CheckAndEmitRefill()` and `CheckAndEmitBackoffExpiry()`** -- Add `lastEmittedTokens` and `lastBackoffActive` tracking fields. `CheckAndEmitRefill()` computes current token level via lazy refill arithmetic, emits `EventTokenRefilled` on change. `CheckAndEmitBackoffExpiry()` detects active-to-clear transition, emits `EventBackoffExpired`. Initialize `lastEmittedTokens` to bucket max in `NewGateway()`. Called by app on `viz.TickMsg` (200ms).
   - Files: `internal/api/gateway.go`
   - Tests: TestGateway_CheckAndEmitRefill_EmitsOnChange; TestGateway_CheckAndEmitRefill_NoEmitWhenStable; TestGateway_CheckAndEmitBackoffExpiry_EmitsOnTransition; TestGateway_CheckAndEmitBackoffExpiry_NoEmitWhenAlreadyClear; TestGateway_CheckAndEmitRefill_NilRecorder

4. **Wire event recorder in `app.go` and `auth.go`** -- Update `SetRecorder()` call in `auth.go` (same method name, new interface type). In `app.go`, update `viz.TickMsg` handler to call `gateway.CheckAndEmitRefill()` and `gateway.CheckAndEmitBackoffExpiry()` before forwarding to panes.
   - Files: `internal/app/auth.go`, `internal/app/app.go`
   - Tests: Integration: SetRecorder(store) compiles; viz.TickMsg handler calls both periodic methods

5. **Retire old recording system** -- Remove `GatewayRecorder` interface, `ResetWatermarks()`, `tokenBucket.minTokens`, `Gateway.peakConcurrent`, `Gateway.minTokensInit`, `gatewayRecordedKey`, `MarkGatewayRecorded()`, `IsGatewayRecorded()` from `internal/api/gateway.go`. Remove `IsGatewayRecorded` check and `NetLogRecorder` interface from `internal/api/logging.go`. Remove `MarkGatewayRecorded` calls from `internal/api/base.go`. Mark `GatewayState`, `GatewaySnapshotter` as deprecated in `internal/domain/gateway.go`, remove `PeakConcurrent`/`MinTokens` from `GatewayState`. Keep `Snapshot()` as deprecated compatibility shim building `GatewayState` from `captureSnapshot()`.
   - Files: `internal/api/gateway.go`, `internal/api/logging.go`, `internal/api/base.go`, `internal/domain/gateway.go`
   - Tests: Remove watermark assertion tests, ResetWatermarks tests, MarkGatewayRecorded/IsGatewayRecorded tests; verify deprecated Snapshot() still returns valid non-watermark fields; all existing tests pass

6. **Update documentation** -- Add Feature 67 row to `docs/features/00-overview.md`. Update Gateway section in `docs/ARCHITECTURE.md` for event emission model and deprecated Snapshot.
   - Files: `docs/features/00-overview.md`, `docs/ARCHITECTURE.md`

---

## Story: Request Flow Replay Engine (spec 68)

### Background
The Request Flow pane used snapshot-polling and NetLog syncing to observe gateway state, but gateway decisions happen in microseconds and self-heal before any poll can catch them. Feature 67 instrumented the Gateway to emit fine-grained lifecycle events into a GatewayEventLog. This story rewrote the Request Flow pane to consume those events and replay them as a slow-motion time machine. The pane no longer holds a `GatewaySnapshotter` reference or calls `Snapshot()`. Instead, it reads events from `store.ReadEventsFrom()` using a cursor, queues them, and replays one per `viz.TickMsg` (200ms minimum visibility). The GATEWAY box became a rich state dashboard plus scrolling decision log with icons. Multiple requests animate concurrently at staggered phases. All old snapshot, netlog, and watermark code was removed, along with the deprecated `GatewayState`, `GatewaySnapshotter`, `GatewayDecision` types and the deprecated `Snapshot()` shim.

### Acceptance Criteria
- [ ] Pane no longer holds a `GatewaySnapshotter` reference
- [ ] Pane reads events from `store.ReadEventsFrom()` using a cursor
- [ ] Events replay at 200ms minimum visibility (one per viz.TickMsg)
- [ ] Event queue absorbs bursts naturally
- [ ] GATEWAY box shows state bars (token bucket, semaphore, backoff) from replay snapshot
- [ ] GATEWAY box shows scrolling decision log with icons below state bars
- [ ] Decision log entries use correct theme colors
- [ ] Multiple requests animate concurrently at staggered phases
- [ ] APP box shows request endpoints from `requestAnimation.phase >= phaseEntered`
- [ ] SPOTIFY box shows responses from `requestAnimation.phase >= phaseInFlight`
- [ ] Left arrow shows gateway decision, right arrow shows HTTP outcome
- [ ] Blocked requests show in APP box but skip InFlight/SPOTIFY
- [ ] Decisions age out after 3s, completed requests after 5s
- [ ] Three-box layout unchanged, flat fallback unchanged
- [ ] Status strip unchanged
- [ ] `GatewayState`, `GatewaySnapshotter`, `GatewayDecision` removed from domain/
- [ ] Deprecated `Snapshot()` shim removed from gateway
- [ ] All tests rewritten and passing
- [ ] `make ci` passes

### Tasks
1. **Add replay data model types** -- `animationPhase` enum (`phaseEntered`, `phaseAtGateway`, `phaseInFlight`, `phaseCompleted`, `phaseDone`). `requestAnimation` struct tracking `requestID`, `method`, `path`, `priority`, `phase`, `decision` (EventKind), `statusCode`, `durationMs`, `enteredAt`. `decisionEntry` struct with `kind`, `label`, `shownAt`. `replayDisplayState` struct with `snapshot` (GatewayStateSnapshot), `requests` (map[uint64]*requestAnimation), `decisions` ([]decisionEntry).
   - Files: `internal/ui/panes/requestflow_pane.go` (or `requestflow_replay.go`)
   - Tests: replayDisplayState zero value has nil map; animationPhase constants have expected ordering

2. **Rewrite `RequestFlowPane` struct and constructor** -- Replace struct fields: remove `gateway domain.GatewaySnapshotter`, `lastSnapshot`, `recentReqs`; add `eventCursor uint64`, `replayQueue []domain.GatewayEvent`, `displayState replayDisplayState`. Constructor takes `*state.Store` and `theme.Theme` only (no GatewaySnapshotter). Update `app.go` call site. Remove `RequestCompletedMsg` type and handler. Remove `reqDisplay` type.
   - Files: `internal/ui/panes/requestflow_pane.go`, `internal/app/app.go`
   - Tests: NewRequestFlowPane returns pane with empty display state; ID, Title, ToggleKey unchanged

3. **Implement the replay loop in `Update()`** -- Replace `viz.TickMsg` and `TickMsg` handlers. `drainEvents()` reads from `store.ReadEventsFrom(cursor)` into replayQueue. `processNextEvent()` pops one event per tick, updates `displayState.snapshot`, calls `processRequestEvent()` for request-scoped events, appends to `displayState.decisions` via `formatDecisionLabel()`. `processRequestEvent()` creates/updates `requestAnimation` by RequestID, advancing phase based on EventKind. `ageOutEntries()` removes decisions older than 3s and completed requests older than 5s. `formatDecisionLabel()` maps all 13 EventKinds to display strings with icons.
   - Files: `internal/ui/panes/requestflow_pane.go`
   - Tests: TestRequestFlowPane_Replay_DrainEvents; TestRequestFlowPane_Replay_ProcessOnePerTick; TestRequestFlowPane_Replay_SnapshotUpdates; TestRequestFlowPane_Replay_RequestPhaseProgression; TestRequestFlowPane_Replay_BlockedRequestSkipsInFlight; TestRequestFlowPane_Replay_DecisionLogGrows; TestRequestFlowPane_Replay_DecisionLogAgesOut; TestRequestFlowPane_Replay_CompletedRequestAgesOut; TestFormatDecisionLabel_AllKinds (table-driven, 13 kinds)

4. **Update `View()` and box rendering for replay display state** -- Update `buildAppBoxLines()` to read from `displayState.requests` sorted by enteredAt. Update `buildSpotifyBoxLines()` to show status+latency for phaseInFlight or phaseCompleted. Update `buildGatewayBoxLines()` to render state bars from `displayState.snapshot` plus scrolling decision log with icons below. Update arrow builders for `requestAnimation.phase` and `requestAnimation.decision`. Remove watermark annotation code. Apply theme colors to decision log entries: interactive enter TextPrimary, background TextMuted, allowed/completed Success, blocked Error, waited/dedup Warning, resource events TextSecondary, refill TextMuted.
   - Files: `internal/ui/panes/requestflow_boxed.go`, `internal/ui/panes/requestflow_pane.go`
   - Tests: TestRequestFlowPane_View_Boxed_ShowsDecisionLog; TestRequestFlowPane_View_Boxed_StateBarsFromSnapshot; TestRequestFlowPane_View_Boxed_RequestInAppBox; TestRequestFlowPane_View_Boxed_ResponseInSpotifyBox; TestRequestFlowPane_View_Boxed_ArrowStates; TestRequestFlowPane_View_Flat_StillWorks; TestDecisionLog_ThemeColors

5. **Remove old snapshot/netlog code from pane** -- Remove `syncFromNetLog()`, `reqDisplay`, `RequestCompletedMsg`, remaining `lastSnapshot`/`recentReqs` references from `requestflow_pane.go`. Remove old `gatewayStateLines()` with watermark annotations, old `buildAppBoxLines()`/`buildSpotifyBoxLines()` reading from `recentReqs` from `requestflow_boxed.go`. Remove `GatewayState`, `GatewaySnapshotter`, `GatewayDecision` from `internal/domain/gateway.go`. Remove deprecated `Snapshot()` shim from `internal/api/gateway.go`.
   - Files: `internal/ui/panes/requestflow_pane.go`, `internal/ui/panes/requestflow_boxed.go`, `internal/domain/gateway.go`, `internal/api/gateway.go`
   - Tests: Verify removed types no longer compile if referenced; update/remove tests referencing removed types

6. **Update existing tests** -- Replace test helper: `newTestRequestFlowPane()` creates pane with `*state.Store`, injects events via `store.RecordEvent()`, sends `viz.TickMsg` to trigger replay. Replace mock `GatewaySnapshotter` with direct event injection. Rewrite arrow state, theme color, boxed/flat layout, staleness, snapshot refresh, and watermark tests. Add decision log rendering, staggered parallel request animation, and event queue backlog tests.
   - Files: `internal/ui/panes/requestflow_pane_test.go`, `internal/ui/panes/requestflow_boxed_test.go`

7. **Update documentation** -- Add Feature 68 row to `docs/features/00-overview.md`. Update Request Flow Rendering section in `docs/ARCHITECTURE.md` with event journal replay model, `replayDisplayState`, decision log, removal of GatewaySnapshotter/GatewayState/GatewayDecision.
   - Files: `docs/features/00-overview.md`, `docs/ARCHITECTURE.md`

---

## Story: Network Log Event Migration (spec 69)

### Background
The Network Log pane read from `store.NetLogEntries()` returning flat `[]NetLogEntry` records with timestamp, method, path, status, and latency. After Features 66-68 moved all recording to the gateway event journal, the old NetLog was redundant -- it contained a subset of the event journal's data, and background requests blocked by backoff (status 0, never reached HTTP) were invisible. This story migrated the Network Log pane to read from the `GatewayEventLog` via cursor-based reads, added PRIORITY and DECISION columns, made blocked requests visible, extended the filter to match new columns, and retired the entire old NetLog system (`NetLog`, `NetLogEntry`, `RecordNetCall`, `RecordGatewayCall`, `NetLogEntries`, `NetLogRecorder`).

### Acceptance Criteria
- [ ] NetworkLogPane reads from `GatewayEventLog` via cursor-based `ReadEventsFrom()`
- [ ] `EventHttpCompleted` events appear as table rows
- [ ] `EventRequestBlocked` events appear as table rows (status 0, "blocked")
- [ ] PRIORITY column shows "int" or "bg" for each request
- [ ] DECISION column shows "allowed", "blocked", "waited", or "dedup"
- [ ] Filter matches priority and decision values
- [ ] `NetLog` struct and `NetLogEntry` type are deleted
- [ ] `RecordNetCall()`, `RecordGatewayCall()`, `NetLogEntries()` are removed from Store
- [ ] `NetLogRecorder` interface is removed from api/
- [ ] `LoggingTransport` simplified or removed
- [ ] No dangling references to removed types
- [ ] All tests rewritten and passing
- [ ] `make ci` passes

### Tasks
1. **Add cursor-based event reading to NetworkLogPane** -- Add `eventCursor uint64` field and `networkLogRow` struct (timestamp, method, path, statusCode, durationMs, priority, decision) to pane. Add `completedRequests []networkLogRow` field capped at 200. Rewrite `refreshRows()`: drain events from `store.ReadEventsFrom(cursor)`, build map of RequestID-to-decision from decision events, extract rows from `EventHttpCompleted` and `EventRequestBlocked` events, cap at 200, call `buildTableRows()` for reverse-chronological rendering.
   - Files: `internal/ui/panes/networklog_pane.go`
   - Tests: TestNetworkLogPane_RefreshRows_CursorAdvances; TestNetworkLogPane_RefreshRows_IncrementalDrain; TestNetworkLogPane_RefreshRows_HttpCompletedAppearsInTable; TestNetworkLogPane_RefreshRows_BlockedRequestAppearsInTable (status "0"); TestNetworkLogPane_RefreshRows_CapsAt200

2. **Add PRIORITY and DECISION columns** -- Add two columns to table definition: `PRI` (FlexFactor 1) and `DECISION` (FlexFactor 3). Populate in `buildTableRows()`: priority as "int"/"bg", decision as "allowed"/"blocked"/"waited"/"dedup". NOTES column: blocked requests show "blocked" instead of latency bar. Reduce ENDPOINT flex from 8 to 7.
   - Files: `internal/ui/panes/networklog_pane.go`
   - Tests: TestNetworkLogPane_View_ShowsPriorityColumn; TestNetworkLogPane_View_ShowsDecisionColumn; TestNetworkLogPane_View_BlockedNotesColumn

3. **Update filter to support new columns** -- Add priority and decision to filter match in `buildTableRows()`: `filter.MatchesAny(path, statusStr, pri, dec)`.
   - Files: `internal/ui/panes/networklog_pane.go`
   - Tests: TestNetworkLogPane_Filter_MatchesPriority ("int" shows only interactive); TestNetworkLogPane_Filter_MatchesDecision ("blocked" shows only blocked); TestNetworkLogPane_Filter_MatchesEndpoint (existing behavior preserved)

4. **Retire `NetLog`, `NetLogEntry`, and related Store methods** -- Delete `internal/state/netlog.go`. Remove `netLog` field, `NewNetLog()` call, `RecordNetCall()`, `RecordGatewayCall()`, `NetLogEntries()`, `NetLog()` from `internal/state/store.go`. Remove `NetLogRecorder` interface and recording logic from `internal/api/logging.go` (simplify LoggingTransport to passthrough or remove). Update `internal/app/auth.go` if LoggingTransport removed. Delete `internal/state/netlog_test.go`.
   - Files: `internal/state/netlog.go` (delete), `internal/state/store.go`, `internal/api/logging.go`, `internal/app/auth.go`, `internal/state/netlog_test.go` (delete)
   - Tests: Verify removal compiles cleanly; all remaining tests pass; make ci passes

5. **Update existing NetworkLogPane tests** -- Rewrite all tests: replace `store.RecordNetCall()` with `store.RecordEvent(domain.GatewayEvent{...})` for both HttpCompleted and decision events. Rewrite table rendering, filter, scroll, latency bar tests. Add blocked request visibility, priority column, decision column, cursor-based incremental read tests.
   - Files: `internal/ui/panes/networklog_pane_test.go`

6. **Update documentation** -- Add Feature 69 row to `docs/features/00-overview.md`. Update `docs/ARCHITECTURE.md`: remove NetLog references, note GatewayEventLog as single source for both panes, update Network Log description with new columns.
   - Files: `docs/features/00-overview.md`, `docs/ARCHITECTURE.md`
