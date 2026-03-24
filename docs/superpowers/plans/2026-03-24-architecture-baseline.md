# Architecture Baseline — Gap Analysis & Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring Spotnik's architecture into full alignment with the Elm Architecture expectations: pure Update-only state mutations, centralized API gateway with rate limiting, toast notifications via BubbleUp, staleness tracking, and idle polling backoff.

**Architecture:** The plan is organized as 5 features that can be implemented sequentially. Each feature addresses a specific architectural gap identified in the analysis. Features are ordered by dependency — later features build on earlier ones.

**Tech Stack:** Go 1.22+, Bubble Tea v0.27+, Lip Gloss, BubbleUp (`go.dalton.dog/bubbleup`), Go stdlib `net/http`

---

## Gap Analysis Summary

### What the codebase gets RIGHT

| Dimension | Status |
|---|---|
| Command pattern (side effects in `tea.Cmd`) | Compliant — no direct API calls in Update() or View() |
| Message types (typed structs) | Fully compliant — no string-based dispatch |
| View purity (no mutations in View) | Compliant — reads only, no side effects |
| Import boundaries (ui/ ↔ api/) | Enforced via conversion layer in commands.go |
| Thread safety (Store mutex) | Correct and consistent `sync.RWMutex` usage |
| 429 detection and backoff for polling | Solid — `Retry-After` header read, 10s floor, polling suspended |

### What the codebase gets WRONG or is MISSING

| # | Gap | Severity | Current State | Expected State |
|---|---|---|---|---|
| G1 | **Store mutations inside `tea.Cmd` closures** | Critical | Data-fetch commands (`fetchPlaybackStateCmd`, `fetchQueueCmd`, 4 library commands, `buildFetchStatsCmd`, `buildSearchCmd`, `buildFetchDevicesCmd` error path) write to Store inside goroutine closures; their Msgs are empty notifications. Note: action commands (`buildAddToQueueCmd`, `buildPlayContextCmd`, `buildToggleLikeCmd`, playlist mutations) are already compliant — they return data-carrying Msgs. | Cmds return data in Msg payloads; only `Update()` writes to Store |
| G2 | **No API Gateway / request controller** | Critical | All requests fire directly to Spotify; no throttling, dedup, priority, or concurrency cap | Single gateway with token bucket, dedup, priority (interactive > background), backoff |
| G3 | **No toast notification system** | Major | Single `statusMsg` string in error-red color for both success and error | BubbleUp-based toast with typed severity (info/success/warning/error), overlay rendering |
| G4 | **No staleness tracking (`fetchedAt`)** | Major | No timestamps; library data stale for entire session; no cache invalidation | Per-domain `fetchedAt` timestamps; TTL-based refresh decisions in Update() |
| G5 | **Silent errors in Library, Queue, Player panes** | Major | Store error fields written but never read in View(); user sees empty state on failure | All errors routed through notification system (toast) with error message + retry hint |
| G6 | **No idle polling backoff** | Moderate | Polling at full speed (3s/9s) regardless of activity or playback state | Back off to 30s+ when paused/idle; resume on user interaction |
| G7 | **No batch endpoints** | Moderate | Everything fetched individually; stats makes 2 sequential calls | Use Spotify "get several" endpoints where available; parallelize independent fetches |
| G8 | **No request dedup/coalescing** | Moderate | Duplicate in-flight requests possible (tick + user action) | In-flight tracking; coalesce identical requests |
| G9 | **No request classification** | Moderate | Interactive and background requests treated identically | Tag requests; gateway favors interactive over background |
| G10 | **Success messages shown in error color** | Minor | `renderStatusBar()` uses `t.Error()` for all `statusMsg` | Distinct success/error/warning styling (solved by G3/BubbleUp) |
| G11 | **Device error write in Cmd closure** | Minor | `buildFetchDevicesCmd` writes `store.SetDevicesError()` inside closure; device list itself is correctly delivered via Msg to DeviceOverlay local state (legitimate pane-local UI state) | Move error write to Update() as part of F1; device list stays in overlay (not a violation) |
| G12 | **No normalized entity storage** | Low | Flat slices; same Track can appear in queue, library, search without dedup | Optional: maps keyed by Spotify ID for entities that appear in multiple domains |

---

## Data Fetching Modes & Relationships

### Polling (tick-driven)

| Endpoint | Interval | Notes |
|---|---|---|
| GET /v1/me/player (playback state) | 3s | Also fetched immediately after playback controls & device transfer |
| GET /v1/me/player/queue | 9s | No on-demand trigger currently |

### On-demand (user-triggered)

All other endpoints: search (300ms debounce), library sections (on navigate), stats (on view switch), devices (on 'd' press), playlist mutations, playback controls, add-to-queue, like/unlike.

### Cascade Relationships

Documents what refreshes after each user action and where gaps exist:

| Action | Immediate Cascade | Gap? |
|---|---|---|
| Playback control (pause/next/prev/vol/shuffle/repeat) | fetchPlaybackState | No |
| Device transfer | fetchPlaybackState | No |
| Create/rename playlist | fetchPlaylistsList | No |
| Add to queue | **None** — waits for 9s tick | Yes |
| Like/unlike track | **None** — liked tracks list not updated | Yes |
| Play track/context | **None** — waits for 3s tick | Minor |
| Remove/reorder playlist tracks | Pane handles locally | No |

### Deferred: Data Reactivity Design

A reactive data graph — where action A immediately triggers refresh of all related domains (B, C) instead of waiting for the next poll cycle — is a future design topic. This interacts with the gateway's dedup and rate-limiting (rapid cascading fetches could burst requests). **Needs a separate brainstorm session.** Not part of this architecture baseline.

---

## Feature Breakdown

The gaps are grouped into 5 implementable features, ordered by dependency:

| Feature | Gaps Addressed | Dependency |
|---|---|---|
| **F1: Elm Purity — Data-Carrying Messages** | G1 | None |
| **F2: API Gateway** | G2, G8, G9 | None |
| **F3: Notifications + Error Routing (BubbleUp)** | G3, G5, G10 | F1 (needs data-carrying error Msgs) |
| **F5: Staleness Tracking** | G4 | F1 (Store schema change) |
| **F6: Idle Polling Backoff** | G6 | F1 (cleaner tick handler after Elm purity refactor) |

**Removed features:**
- ~~F4 (Error Propagation)~~ — merged into F3. All errors route through the notification system (toast with error message + retry hint). No inline error rendering in pane View().
- ~~F7 (Batch Endpoints)~~ — dropped after research. See "Research Note: Batch Endpoints" below. Parallelizing stats calls moved to F1 as a bonus step.

G12 (normalized storage) is deferred — the current denormalized approach is appropriate for the app's read-heavy, low-entity-count use case. It can be revisited if cross-domain entity relationships become important.

---

## Feature F1: Elm Purity — Data-Carrying Messages

### Problem

The data-fetching `build*Cmd` functions in `commands.go` mutate the Store directly inside `tea.Cmd` closures (goroutines). This violates the Elm Architecture principle that only `Update()` mutates state. The affected commands are: `fetchPlaybackStateCmd`, `fetchQueueCmd`, `buildFetchPlaylistsCmd`, `buildFetchAlbumsCmd`, `buildFetchLikedTracksCmd`, `buildFetchRecentlyPlayedCmd`, `buildFetchStatsCmd`, `buildSearchCmd` (closure writes), and `buildFetchDevicesCmd` (error path only). Their Msg types are empty notification structs (e.g., `QueueLoadedMsg{}`) that signal panes to re-read from the Store.

**Already compliant commands** (no changes needed): `buildAddToQueueCmd`, `buildPlayContextCmd`, `buildPlayTrackCmd`, `buildToggleLikeCmd`, `buildCreatePlaylistCmd`, `buildRenamePlaylistCmd`, `buildRemovePlaylistTrackCmd`, `buildReorderPlaylistTracksCmd` — these already return data-carrying Msgs without Store writes.

### Solution

1. Add data payloads to all result Msg types
2. Move all Store writes from `build*Cmd` closures into `Update()` handlers
3. `build*Cmd` functions become pure: API call → return Msg with data or error

### Files

- Modify: `internal/ui/panes/messages.go` — add payload fields to all Msg types
- Modify: `internal/app/commands.go` — remove all `store.Set*()` / `store.Clear*()` calls from closures
- Modify: `internal/app/app.go` — add Store writes to each Msg handler in `Update()`
- Modify: `internal/app/routing.go` — same for playlist-routed messages
- Test: `internal/app/commands_test.go` — verify Cmds return data, don't write Store
- Test: `internal/app/app_test.go` — verify Update() writes Store from Msg payloads

### Affected Message Types

| Current Msg | Add Payload |
|---|---|
| `PlaybackStateFetchedMsg{}` | `State *api.PlaybackState, Err error` |
| `QueueLoadedMsg{}` | `Tracks []api.Track, Err error` |
| `LibraryLoadedMsg{}` | Split into `PlaylistsLoadedMsg{Items []api.SimplePlaylist, Total int, Err error}`, `AlbumsLoadedMsg{...}`, `LikedTracksLoadedMsg{...}`, `RecentlyPlayedLoadedMsg{...}` |
| `StatsLoadedMsg{TimeRange string}` | `TopTracks []api.Track, TopArtists []api.FullArtist, TimeRange string, Err error` |
| `SearchResultsMsg{Results *SearchResultData}` | Already has payload — just remove Store writes from the Cmd |
| `DevicesLoadedMsg` | Already carries `[]DeviceInfo` — but `buildFetchDevicesCmd` also writes `store.SetDevicesError`; move that to Update |

### Task Sequence

- [ ] **Step 1:** Update `messages.go` — add data/error fields to `PlaybackStateFetchedMsg`, `QueueLoadedMsg`
- [ ] **Step 2:** Update `fetchPlaybackStateCmd` in `commands.go` — return data in Msg instead of writing Store
- [ ] **Step 3:** Update `app.go` `Update()` handler for `PlaybackStateFetchedMsg` — write Store from Msg payload
- [ ] **Step 4:** Write tests verifying the Cmd returns data and Update writes Store
- [ ] **Step 5:** Commit: `refactor(arch): data-carrying PlaybackStateFetchedMsg`
- [ ] **Step 6:** Repeat for `fetchQueueCmd` / `QueueLoadedMsg`
- [ ] **Step 7:** Commit: `refactor(arch): data-carrying QueueLoadedMsg`
- [ ] **Step 8:** Split `LibraryLoadedMsg` into 4 domain-specific Msgs with payloads
- [ ] **Step 9:** Update all 4 library `build*Cmd` functions
- [ ] **Step 10:** Update `Update()` handlers for library Msgs
- [ ] **Step 11:** Write tests
- [ ] **Step 12:** Commit: `refactor(arch): data-carrying library messages`
- [ ] **Step 13:** Repeat for Stats, Search (remove Store writes from closure), Devices
- [ ] **Step 14:** Write tests
- [ ] **Step 15:** Commit: `refactor(arch): data-carrying stats/search/devices messages`
- [ ] **Step 16:** Remove `store *state.Store` parameter from the two package-level functions: `fetchPlaybackStateCmd` and `fetchQueueCmd` (the method-based `build*Cmd` functions on `*App` capture `a.store` via receiver — they have no store parameter)
- [ ] **Step 17:** Commit: `refactor(arch): remove store param from package-level command functions`
- [ ] **Step 18 (bonus):** Refactor `buildFetchStatsCmd` to fetch top tracks and top artists concurrently using `sync.WaitGroup` + a 2-slot result struct (stdlib only — no `errgroup` dependency per CLAUDE.md rules)
- [ ] **Step 19:** Write test verifying both stats calls run in parallel
- [ ] **Step 20:** Commit: `perf(stats): parallelize top tracks and top artists fetches`
- [ ] **Step 21:** Run `make ci` — full pass required

### Verification

After this feature, `grep -r 'store\.Set\|store\.Clear' internal/app/commands.go` should return ZERO matches — no Store writes remain in command builders. Note: Store *reads* (e.g., `store.PlaybackState()` in `buildPlaybackAPICmd` for volume/shuffle state) are legitimate and expected to remain.

---

## Feature F2: API Gateway

### Problem

All API requests fire directly to Spotify with no throttling, dedup, concurrency cap, or priority. A burst of user actions + polling can trigger rate limiting. There's no single control point for all HTTP traffic.

### Solution

Introduce an `internal/api/gateway.go` that wraps all Spotify HTTP calls. Every request goes through the gateway. The gateway provides:

1. **Token bucket rate limiter** — global request rate cap (e.g., 10 req/s)
2. **Concurrency limiter** — max N parallel in-flight requests (e.g., 5)
3. **Request classification** — `interactive` requests skip the token bucket wait; `background` requests respect it. This is a simple bypass, not a preemptive priority queue (true preemption is impractical in a goroutine-per-Cmd model since by the time `Gateway.Do()` is called, the goroutine is already running)
4. **In-flight dedup** — if an identical request is already in-flight, wait for its result
5. **Backoff state** — on 429, pause all requests for `Retry-After` duration
6. **Observability** — expose throttle state for UI display

### Files

- Create: `internal/api/gateway.go` — the gateway implementation
- Create: `internal/api/gateway_test.go` — tests
- Modify: `internal/api/base.go` — route `doJSON`/`doNoContent` through gateway
- Modify: `internal/app/app.go` — pass gateway to command builders; read throttle state
- Modify: `internal/state/store.go` — add rate-limit observability fields (`IsThrottled`, `RetryAfterSecs`, `Last429At`)

### Design

```go
// internal/api/gateway.go

// Priority classifies request urgency.
type Priority int

const (
    Background  Priority = iota // polling, prefetch
    Interactive                  // user-initiated actions
)

// RequestKey uniquely identifies a request for dedup purposes.
type RequestKey struct {
    Method string
    Path   string
}

// Gateway controls all outbound HTTP traffic to Spotify.
type Gateway struct {
    mu            sync.Mutex
    bucket        *tokenBucket     // rate limiter
    semaphore     chan struct{}     // concurrency limiter
    inflight      map[RequestKey]*inflightEntry // dedup
    backoffUntil  time.Time        // 429 backoff
    retryAfter    int              // seconds
}

// Do executes a request through the gateway.
// Blocks if rate limited, dedupes identical in-flight requests,
// and respects priority ordering.
func (g *Gateway) Do(ctx context.Context, priority Priority, key RequestKey, fn func() (*http.Response, error)) (*http.Response, error)
```

### Task Sequence

- [ ] **Step 1:** Write failing test for token bucket rate limiter
- [ ] **Step 2:** Implement `tokenBucket` in `gateway.go`
- [ ] **Step 3:** Run test — pass
- [ ] **Step 4:** Write failing test for concurrency limiter
- [ ] **Step 5:** Implement semaphore-based concurrency cap
- [ ] **Step 6:** Run test — pass
- [ ] **Step 7:** Write failing test for in-flight dedup
- [ ] **Step 8:** Implement `inflight` map with wait channels
- [ ] **Step 9:** Run test — pass
- [ ] **Step 10:** Write failing test for 429 backoff
- [ ] **Step 11:** Implement `backoffUntil` + `Retry-After` handling
- [ ] **Step 12:** Run test — pass
- [ ] **Step 13:** Write failing test for priority (interactive requests bypass token bucket wait; background requests respect it)
- [ ] **Step 14:** Implement priority bypass in gateway (interactive skips bucket, not a priority queue)
- [ ] **Step 15:** Run test — pass
- [ ] **Step 16:** Commit: `feat(api): add API gateway with rate limiting and dedup`
- [ ] **Step 17:** Integrate gateway into `BaseClient.doJSON` / `doNoContent`
- [ ] **Step 18:** Update command builders to pass priority classification
- [ ] **Step 19:** Add throttle state fields to Store
- [ ] **Step 20:** Write integration test
- [ ] **Step 21:** Commit: `feat(api): integrate gateway into all API calls`
- [ ] **Step 22:** Run `make ci` — full pass required

---

## Feature F3: Notifications + Error Routing (BubbleUp)

### Problem

The current notification system is a single `statusMsg` string rendered in error-red color for both success and error messages. There's no visual distinction between severity levels. The status bar conflates keybinding hints with transient feedback.

### Solution

Integrate [BubbleUp](https://github.com/DaltonSW/BubbleUp) (`go.dalton.dog/bubbleup`) as the toast notification component. BubbleUp provides:

- Floating overlay toasts with auto-dismiss
- Severity types (Info, Warning, Error, Debug + custom)
- Positioning (top-right for our use case)
- Theme-compatible foreground colors
- Proper Bubble Tea integration (Model/Update/View)

### Files

- Create: `internal/ui/components/notifications.go` — thin wrapper around BubbleUp with Spotnik theme colors
- Modify: `internal/app/app.go` — embed `bubbleup.AlertModel` in root App; replace `statusMsg` usage; add error→toast routing in Update() handlers
- Modify: `internal/app/render.go` — call `alert.Render(existingView)` in root View() as final overlay step (BubbleUp's `Render` takes the full view string and composites the alert on top)
- Modify: `internal/app/routing.go` — replace `statusMsg` assignments with `NewAlertCmd` calls
- Modify: `internal/ui/panes/library.go` — remove Store error field reads from View() (errors route through toast)
- Modify: `internal/ui/panes/queue.go` — remove Store error field reads from View() (errors route through toast)
- Modify: `go.mod` — add `go.dalton.dog/bubbleup` dependency
- Test: `internal/ui/components/notifications_test.go`

### Owner Approval: GRANTED

> BubbleUp dependency (`go.dalton.dog/bubbleup`) approved by owner on 2026-03-24. MIT-licensed, depends only on bubbletea + lipgloss (already in our deps).

### Custom Alert Types for Spotnik

> **Note:** BubbleUp's `AlertDefinition.ForeColor` field is a `string` (hex color). Since `lipgloss.Color` is `type Color string`, use explicit conversion: `string(theme.Success())`. Verify this compiles after `go get` in Step 1.

```go
// Register custom alert types matching our theme
// ForeColor requires string — explicit conversion from lipgloss.Color
successAlert := bubbleup.AlertDefinition{
    Key:       "success",
    ForeColor: string(theme.Success()),    // "#00ff88"
    Prefix:    "✓",
}
errorAlert := bubbleup.AlertDefinition{
    Key:       "error",
    ForeColor: string(theme.Error()),      // "#ff5555"
    Prefix:    "✗",
}
warningAlert := bubbleup.AlertDefinition{
    Key:       "warning",
    ForeColor: string(theme.Warning()),    // "#ffcc00"
    Prefix:    "!",
}
infoAlert := bubbleup.AlertDefinition{
    Key:       "info",
    ForeColor: string(theme.KeyHint()),    // "#00afff"
    Prefix:    "→",
}
rateLimitAlert := bubbleup.AlertDefinition{
    Key:       "ratelimit",
    ForeColor: string(theme.Warning()),    // "#ffcc00"
    Prefix:    "⏳",
}
```

### Task Sequence

#### Phase 1: Toast Infrastructure

- [ ] **Step 1:** `go get go.dalton.dog/bubbleup` — then inspect `AlertDefinition` struct to confirm `ForeColor` type and the `Render()` method signature (BubbleUp's `View()` is empty; rendering is done via the `Render(content string) string` method which overlays the alert onto the provided content)
- [ ] **Step 2:** Create `notifications.go` — wrapper that creates AlertModel with Spotnik theme colors and registers custom alert types
- [ ] **Step 3:** Write tests for notification wrapper
- [ ] **Step 4:** Commit: `feat(ui): add BubbleUp notification wrapper`
- [ ] **Step 5:** Embed `bubbleup.AlertModel` in root `App` struct
- [ ] **Step 6:** Wire `Init()` — batch alert model's `Init()` with existing commands
- [ ] **Step 7:** Wire `Update()` — pass all messages to `alert.Update()`; batch alert commands
- [ ] **Step 8:** Wire `View()` — call `alert.Render(existingView)` as the final step in root `View()`. BubbleUp's `Render(content string) string` method takes the full rendered view string and overlays the active alert on top. The `View()` method on AlertModel is intentionally empty and must NOT be called.
- [ ] **Step 9:** Commit: `feat(ui): integrate BubbleUp into root App model`
- [ ] **Step 10:** Replace all `a.statusMsg = "..."` + `statusDismissMsg` patterns with `alert.NewAlertCmd(key, msg)`
- [ ] **Step 11:** Remove `statusMsg`, `statusDismissMsg` from App struct
- [ ] **Step 12:** Update `renderStatusBar()` to always show keybinding hints (no more error override)
- [ ] **Step 13:** Write integration tests
- [ ] **Step 14:** Commit: `refactor(ui): replace statusMsg with BubbleUp toasts`

#### Phase 2: Error Routing (formerly F4)

All API errors flow through the notification system. No inline error rendering in pane View(). Toast shows error message + retry hint (e.g., "Failed to load playlists. Press R to retry").

- [ ] **Step 15:** In `app.go` Update() handlers, for every data-carrying Msg with non-nil Err field (from F1), emit a toast notification via `alert.NewAlertCmd("error", errorMsg + retryHint)`
- [ ] **Step 16:** For `PlaybackStateFetchedMsg` with Err: emit toast (transient polling error, not persistent state). Do NOT add a `playbackError` Store field.
- [ ] **Step 17:** For library Msgs (`PlaylistsLoadedMsg`, `AlbumsLoadedMsg`, `LikedTracksLoadedMsg`, `RecentlyPlayedLoadedMsg`) with Err: emit toast with domain-specific message + retry hint
- [ ] **Step 18:** For `QueueLoadedMsg` with Err: emit toast
- [ ] **Step 19:** For `StatsLoadedMsg` with Err: emit toast
- [ ] **Step 20:** Remove Store error field reads from pane View() methods — no inline error rendering. Store error fields remain for retry logic only, never read in View()
- [ ] **Step 21:** Write tests: verify each error Msg triggers a toast, verify no Store error fields are read in View()
- [ ] **Step 22:** Commit: `feat(ui): route all API errors through toast notifications`
- [ ] **Step 23:** Run `make ci` — full pass required

### Notification Mapping

| Current `statusMsg` Usage | New BubbleUp Alert |
|---|---|
| `"✓ Added to queue: ..."` | `success` — "Added to queue: ..." |
| `"✗ <error>"` | `error` — "<error>" |
| `"Playback control not available..."` | `warning` — "Playback control not available..." |
| `"Rate limited — pausing requests for Ns"` | `ratelimit` — "Rate limited, retrying in Ns" |
| `"Session expired. Run: spotnik auth"` | `error` — "Session expired. Run: spotnik auth" |
| `"Switching to <device>..."` | `info` — "Switching to <device>..." |

---

---

## Feature F5: Staleness Tracking

### Problem

No `fetchedAt` timestamps exist. Library data (albums, liked tracks) is fetched once per session and never refreshed. Stats data is cached per time-range forever. There's no way for `Update()` to decide whether to reuse cached data or trigger a refresh.

### Solution

Add `fetchedAt time.Time` per domain in the Store. Provide a `IsStale(domain, ttl)` helper. Update `Update()` logic to check staleness before dispatching fetch commands.

### Files

- Modify: `internal/state/store.go` — add `*FetchedAt time.Time` fields per domain + `IsStale()` helper
- Modify: `internal/app/app.go` — check staleness before dispatching library/stats fetches
- Test: `internal/state/store_test.go` — test staleness logic

### Staleness TTLs

| Domain | TTL | Rationale |
|---|---|---|
| Playback state | N/A | Always polled, overwritten each cycle |
| Queue | N/A | Always polled |
| Playlists list | 5 min | Changes infrequently |
| Albums | 5 min | Changes infrequently |
| Liked tracks | 5 min | Changes infrequently |
| Recently played | 2 min | Changes with playback |
| Stats (per range) | 10 min | Spotify updates these slowly |
| Devices | 30 sec | Can change quickly when switching |

### Task Sequence

- [ ] **Step 1:** Add `*FetchedAt time.Time` fields to Store for each domain
- [ ] **Step 2:** Write `IsStale(fetchedAt time.Time, ttl time.Duration) bool` helper
- [ ] **Step 3:** Write tests for staleness logic
- [ ] **Step 4:** Update `Set*` methods to stamp `time.Now()` on write
- [ ] **Step 5:** Commit: `feat(state): add staleness tracking to Store`
- [ ] **Step 6:** Update library Init/navigation to check `IsStale` before fetching
- [ ] **Step 7:** Update stats view to check staleness on re-open
- [ ] **Step 8:** Write tests
- [ ] **Step 9:** Commit: `feat(app): use staleness checks before data fetches`
- [ ] **Step 10:** Run `make ci` — full pass required

---

## Feature F6: Idle Polling Backoff

### Problem

The tick loop polls at full speed (3s playback, 9s queue) regardless of user activity or playback state. When music is paused or the user is on the stats/playlists view, polling wastes bandwidth and risks rate limiting.

### Two-Layer Rate Management Design

F6 and F2 (Gateway) are two layers of the same rate management strategy:

```
Layer 1: Proactive (F6 / tick loop in app.go)
  - Reduces DEMAND based on app state
  - Answers: "Should we even poll right now?"
  - Controls: tick interval (3s → 10s → 30s)
  - Inputs: user activity, playback state

Layer 2: Reactive (F2 / Gateway)
  - Enforces LIMITS on whatever demand arrives
  - Answers: "Given this request, should we allow it?"
  - Controls: token bucket, concurrency, dedup, 429 backoff
  - Inputs: HTTP responses, request patterns
```

**Why they're separate:** The gateway can't reduce polling frequency — it sees individual requests, not the tick schedule. Only the tick loop knows app state (paused, idle, which view is active). F6 reduces the *number* of requests entering the gateway. F2 limits the *rate* of requests that pass through. They don't conflict.

**F6 depends on F1, NOT F2.** These layers are independent.

### Solution

Track last user interaction time and playback state. When idle (no interaction for 60s) or paused, extend poll intervals. Resume full-speed polling on user interaction.

### Files

- Modify: `internal/app/app.go` — add `lastInteraction time.Time`, `idleThreshold`, adaptive tick intervals
- Modify: `internal/state/store.go` — (if needed) expose `IsPlaying()` convenience method
- Test: `internal/app/app_test.go` — test interval adaptation

### Polling Schedule

| State | Playback Interval | Queue Interval |
|---|---|---|
| Active + playing | 3s (current) | 9s (current) |
| Active + paused | 10s | 30s |
| Idle (60s no input) + playing | 10s | 30s |
| Idle + paused | 30s | 60s |

### Task Sequence

- [ ] **Step 1:** Add `lastInteraction time.Time` to App struct
- [ ] **Step 2:** Update all `tea.KeyMsg` handling to stamp `lastInteraction = time.Now()`
- [ ] **Step 3:** Write `pollInterval()` method that returns intervals based on idle + playback state
- [ ] **Step 4:** Update tick handler to use adaptive intervals
- [ ] **Step 5:** Write tests for each state combination
- [ ] **Step 6:** Commit: `feat(app): adaptive polling with idle backoff`
- [ ] **Step 7:** Run `make ci` — full pass required

---

## Research Note: Batch Endpoints (Dropped)

Spotify batch endpoints exist: Get Several Tracks (`/v1/tracks?ids=`), Get Several Artists (`/v1/artists?ids=`), Get Several Albums (`/v1/albums?ids=`), Check User's Saved Tracks/Albums (`/v1/me/tracks/contains?ids=`).

**After research (2026-03-24), Spotnik has no current use case for any of them:**

- We never fetch individual entities by ID — all track/artist/album data comes from list endpoints (search results, playlist tracks, queue, recently played, top tracks/artists)
- The only optimization was parallelizing the 2 sequential stats calls (top tracks + top artists) — this is now a bonus step in F1
- `Check User's Saved Tracks` could show "liked" status on queue/search items, but that's a **new feature**, not an architecture gap

**Revisit if:** We add features that fetch entities by ID (e.g., "related artists" view, "album details" view).

---

## Documentation Gaps

The following documentation updates should accompany the implementation:

### `docs/ARCHITECTURE.md` Updates Needed

1. **Section: "Message Flow"** — Update to show data-carrying messages pattern (after F1)
2. **Section: "API Client Design"** — Add Gateway documentation (after F2)
3. **Section: "State Management" → "The Store"** — Add staleness tracking fields and `IsStale()` pattern (after F5)
4. **Section: "Polling Architecture"** — Add idle backoff documentation (after F6)
5. **New Section: "API Gateway"** — Rate limiting, dedup, priority, backoff (after F2)
6. **New Section: "Notification System"** — BubbleUp integration, alert types, severity mapping, error routing through toasts (after F3)
7. **Section: "Error Handling Conventions"** — Update `build*Cmd` pattern to show data-in-Msg flow; document that all errors route through notification system, not inline pane rendering (after F1+F3)

### `docs/DESIGN.md` Updates Needed

1. **Section: "Status Bar"** — Remove "Error mode" (replaced by toast overlay); status bar becomes hints-only (after F3)
2. **New Section: "Toast Notifications"** — Position, severity colors, dismiss behavior, error routing (after F3)

### `CLAUDE.md` Updates Needed

1. **Section: "Architecture Rules"** — Add: "Commands must not mutate the Store; return data in Msg payloads" (after F1)
2. **Section: "API Rules"** — Add: "All requests go through the API Gateway" (after F2)
3. **Section: "Architecture Rules"** — Add: "All API errors route through the notification system (toast), not inline pane rendering" (after F3)

---

## Implementation Order & Dependencies

```
F1: Elm Purity ──────┬──→ F3: Notifications + Error Routing
                     ├──→ F5: Staleness Tracking
                     └──→ F6: Idle Polling Backoff

F2: API Gateway ─────────→ (independent, can parallel with F1)
```

**F1 and F2 can be done in parallel** — they touch different files and have no overlapping changes. F3 depends on F1 (needs data-carrying error Msgs). F5 and F6 depend on F1 only.

Recommended serial order: **F1 → F3 → F5 → F6** (main track) and **F2** (parallel track)

---

## Risk Notes

1. **F1 is the largest refactor** — touches 8 data-fetch command functions and their corresponding Msg types. Action commands (add-to-queue, playback, like, playlist mutations) are already compliant and need no changes. Do it incrementally (one domain at a time) with commits between each.
2. **BubbleUp dependency (F3) approved** — owner signed off on 2026-03-24. Still verify the API after `go get`: run `go doc go.dalton.dog/bubbleup AlertDefinition` and `go doc go.dalton.dog/bubbleup AlertModel.Render` to confirm `ForeColor` type and `Render()` signature.
3. **Gateway (F2) must not break existing behavior** — introduce it as a pass-through first (no-op), then add limiting. Feature-flag the limiting if needed.
4. **F1 changes are breaking for any in-flight feature branches** — coordinate with any ongoing work.
5. **F2 priority model is simplified** — true priority preemption is impractical in Bubble Tea's goroutine-per-Cmd model. The design uses a simpler approach: interactive requests bypass the token bucket, background requests respect it. This is sufficient for the use case.
