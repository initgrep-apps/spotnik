# ARCHITECTURE.md — Technical Reference

> **Reference only.** Feature specs embed the patterns you need inline.
> Consult this document when you need deeper context on a pattern the feature spec points to.
> Do not read cover-to-cover before implementing a feature.

---

## Architectural Overview

Spotnik follows the **Elm Architecture** as enforced by Bubble Tea. The entire application is a pure function of state: `View(State) → UI`. Side effects happen only through commands and messages.

```
┌──────────────────────────────────────────────────────────┐
│                        main.go                           │
│                    (entry point only)                    │
└─────────────────────────┬────────────────────────────────┘
                          │
┌─────────────────────────▼────────────────────────────────┐
│                      cmd/root.go                         │
│         (flag parsing, config loading, auth check)       │
└─────────────────────────┬────────────────────────────────┘
                          │
┌─────────────────────────▼────────────────────────────────┐
│                    internal/app/                          │
│              Root Bubble Tea Model (tea.Model)            │
│   app.go      — Init/Update, handleMsg, polling tick     │
│   render.go   — View composition, grid, overlays         │
│   routing.go  — Key/mouse dispatch, focus rotation       │
│   commands.go — 30+ build*Cmd factories (no store writes)│
│   auth.go     — PKCE flow, API client wiring             │
│   splash.go   — Startup splash screen                    │
└──────┬─────────────┬──────────────┬──────────────────────┘
       │             │              │
       │    ┌────────▼────────┐    │
       │    │ internal/domain │    │
       │    │ (shared types)  │    │
       │    └──┬──────────┬───┘    │
       │       │          │        │
┌──────▼───────▼──┐  ┌───▼────────▼──────┐
│  internal/ui/   │  │  internal/api/    │
│  (10 panes,     │  │  (HTTP clients,   │
│   components,   │  │   gateway,        │
│   layout,       │  │   logging)        │
│   theme)        │  │                   │
└────────┬────────┘  └─────────┬─────────┘
         │                     │
         └──────────┬──────────┘
                    │
         ┌──────────▼──────────┐
         │  internal/state/    │
         │  Store (single      │
         │  source of truth)   │
         └──────────┬──────────┘
                    │
    ┌───────────────┼───────────────┐
    │               │               │
┌───▼────┐    ┌─────▼─────┐   ┌────▼────┐
│config/ │    │ keychain/ │   │domain/  │
└────────┘    └───────────┘   └─────────┘

PANES (10 total):
  Page A (Music, 8 panes):
    NowPlayingPane, QueuePane, PlaylistsPane, AlbumsPane,
    LikedSongsPane, RecentlyPlayedPane, TopTracksPane, TopArtistsPane
  Page B (Nerd Status, 2 panes):
    RequestFlowPane, NetworkLogPane
  Floating overlays (not in grid):
    SearchOverlay, DeviceOverlay
```

### The Domain Package

`internal/domain/` contains shared types that bridge `api/` and `ui/` without creating import cycles. Key files:

- `types.go` — Core types: `PlaybackState`, `Track`, `Artist`, `Album`, `Device`, `SimplePlaylist`, `SavedAlbum`, `SavedTrack`, `PlayHistory`, `QueueResponse`, `FullArtist`, `PlayOptions`
- `gateway.go` — `GatewaySnapshotter` interface, `GatewayState` struct, `RequestPriority` constants
- `search.go` — `SearchResult` type

Panes import `domain/` types, not `api/` types. API clients return `domain/` types. This is how the import boundary is enforced — `ui/` and `api/` never import each other.

---

## View States

The app has three view modes, managed by the `currentView` field:

1. **`viewSplash`** — 5-second startup screen with ASCII banner (rendered by `splash.go`)
2. **`viewAuth`** — OAuth panel when authentication is needed (rendered by `auth.go`)
3. **`viewGrid`** — Normal operation with pane grid, header, and status bar

Transitions: `viewSplash` → `viewAuth` (if unauthenticated) → `viewGrid`, or `viewSplash` → `viewGrid` (if already authenticated). The splash timer fires `splashDismissMsg` after 5 seconds.

---

## Render Pipeline

The full view composition flow in `internal/app/render.go`:

```
View()
  └── alerts.Render(buildView())     ← toast overlay is ALWAYS the last step
        │
        └── buildView()
              ├── Terminal too small? → renderTooSmall()  (min 120x30)
              ├── viewSplash?        → renderSplash()
              ├── viewAuth?          → renderAuthPanel()
              └── viewGrid:
                    ├── renderHeader()      (1 line: app name, page, shortcuts, device)
                    ├── renderGrid()        (pane grid with borders)
                    ├── renderStatusBar()   (1 line: global keybinding hints)
                    └── Overlay compositing:
                          ├── deviceOverlayOpen? → btoverlay.Composite(device, dimmed, Right, Top)
                          └── searchOpen?        → btoverlay.Composite(search, dimmed, Center, Center)
```

**Key rules:**
- `alerts.View()` always returns `""` — must use `alerts.Render(content)` for toast compositing
- `renderGrid()` groups panes by row, wraps each in btop-style borders via `layout.RenderPaneBorder()`, applies `lipgloss.Width/MaxWidth/Height/MaxHeight` caps, then joins horizontally per row and vertically across rows
- Overlays use the `bubbletea-overlay` library (`btoverlay.Composite`) — background is dimmed with `Faint(true)`
- Total view output must equal exactly `terminalHeight` lines

---

## Message Flow

```
User Keypress / Mouse Wheel
     │
     ▼
routing.go: handleKeyMsg / handleMouseMsg
     │
     ├── Guard 1: Device overlay open → all keys to DeviceOverlay
     ├── Guard 2: Search overlay open → all keys to SearchOverlay
     ├── Guard 3: Auth view → only quit keys
     ├── Guard 4: Pane has active filter → all keys to pane
     ├── Global keys (q, /, d, 0, p, 1-8, Tab, Shift+Tab)
     ├── Playback keys (Space, n, +, -, s, r, v, ←, →) → always NowPlayingPane
     └── All other keys → focused pane
              │
              ▼
         pane.Update(msg)
              │
              └── Returns (model, cmd)
                       │
                       ▼ (cmd executes)
                  tea.Cmd runs async
                       │
                       ▼
                  Returns tea.Msg with DATA payload
                       │
                       ▼
              app.go: handleMsg(resultMsg)
                       │
                       ├── Write data from msg payload to Store
                       ├── Emit toast notification if error
                       └── Forward to pane, re-render
```

### Mouse Support

`handleMouseMsg` in `routing.go`: wheel-up/down events are converted to `j`/`k` key messages, hit-tested via `layout.PaneAt(x, y)` to find the target pane, and routed to that pane WITHOUT changing keyboard focus. Mouse events are ignored when overlays are open.

### Data-Carrying Messages (Elm Architecture Purity)

**Rule: `build*Cmd` / `fetch*Cmd` functions MUST NOT write to the Store.** Only `Update()` may mutate the Store.

Commands return data in their Msg payloads. `Update()` reads the payload and writes to the Store. This is the Elm Architecture contract.

**Before (violation):**
```go
// WRONG — Store write inside goroutine closure
func fetchQueueCmd(player api.PlayerAPI, store *state.Store) tea.Cmd {
    return func() tea.Msg {
        qr, err := player.Queue(ctx)
        store.SetQueue(qr.Queue)   // ← violates Elm contract
        store.ClearQueueError()
        return panes.QueueLoadedMsg{} // empty notification
    }
}
```

**After (correct):**
```go
// CORRECT — data in payload, Store write in Update()
func fetchQueueCmd(player api.PlayerAPI) tea.Cmd {
    return func() tea.Msg {
        qr, err := player.Queue(ctx)
        if err != nil { return panes.QueueLoadedMsg{Err: err} }
        return panes.QueueLoadedMsg{Tracks: qr.Queue} // data-carrying
    }
}

// In app.go Update():
case panes.QueueLoadedMsg:
    if m.Err != nil {
        a.store.SetQueueError(m.Err)
    } else {
        a.store.ClearQueueError()
        a.store.SetQueue(m.Tracks) // ← Store write only here
    }
```

All message types in `internal/ui/panes/messages.go` carry their data payload and an `Err error` field. `Update()` is the sole writer to the Store.

---

## State Management

### The Store

`internal/state/store.go` is the single source of truth. All API data lives here. Panes **read** from the store but **never write** to it directly — they dispatch messages that the root model uses to update the store.

See `internal/state/store.go` for the full struct definition and accessor methods.

### Staleness Tracking

Each data domain in the Store carries a `fetchedAt time.Time` timestamp. When data is written via `Set*()`, the timestamp is stamped to `time.Now()`. `Update()` uses these timestamps to avoid unnecessary re-fetches.

**`IsStale` helper:**

```go
// IsStale returns true if fetchedAt is zero (never fetched) or older than ttl.
func IsStale(fetchedAt time.Time, ttl time.Duration) bool {
    return fetchedAt.IsZero() || time.Since(fetchedAt) > ttl
}
```

**TTL constants (defined in `internal/state/store.go`):**

| Domain | TTL | Rationale |
|---|---|---|
| `PlaylistsTTL` | 5 min | Changes infrequently |
| `AlbumsTTL` | 5 min | Changes infrequently |
| `LikedTracksTTL` | 5 min | Changes infrequently |
| `RecentlyPlayedTTL` | 2 min | Changes with every playback event |
| `StatsTTL` | 10 min | Spotify updates these slowly |
| `DevicesTTL` | 5 sec | Volatile — short cooldown prevents rapid-fire API calls while ensuring fresh data on user request |

**Convenience methods** — one per domain, e.g.:

```go
func (s *Store) PlaylistsStale() bool {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return IsStale(s.playlistsFetchedAt, PlaylistsTTL)
}
```

**`Update()` gate pattern:**

```go
// In app.go handleMsg:
case panes.FetchPlaylistsRequestMsg:
    // Skip fetch if playlists are fresh and this is a non-paginated initial load.
    if m.Offset == 0 && !a.store.PlaylistsStale() {
        return a, nil
    }
    return a, a.buildFetchPlaylistsCmd(m.Offset)
```

**Staleness classification** (see `store.go` comment block for canonical definition):

| Category | Domains | TTL | Rationale |
|---|---|---|---|
| **Stable** (staleness-gated, long TTL) | Playlists, Albums, Liked Tracks, Stats | 5-10 min | Only change when user acts within Spotnik or very slowly |
| **Volatile** (staleness-gated, short TTL) | Devices, Recently Played | 5s-2min | Change externally — short TTL balances freshness with API efficiency |
| **Real-time** (polled on tick, no TTL) | Playback State, Queue | N/A | Overwritten every tick cycle with adaptive polling intervals |

Playback state and queue are **not** staleness-tracked because they are overwritten on every tick cycle (polling). Staleness tracking is only for data fetched on demand: library, stats, and devices.

The former `albumsLoaded` and `likedLoaded` boolean sentinels have been replaced by `AlbumsLoaded()` and `LikedLoaded()` methods derived from the `fetchedAt` timestamps. Library pane expansion uses `AlbumsStale()` and `LikedTracksStale()` to decide whether to trigger a lazy fetch.

### Fetching Sentinels (TOCTOU Prevention)

Between a staleness check and the API response arriving, a second identical request could slip through. Boolean sentinel fields prevent this:

- `playlistsFetching`, `albumsFetching`, `likedFetching`, `recentFetching`, `devicesFetching` — one per domain
- `statsFetching map[string]bool` — keyed by time range

**Guard pattern (in `app.go` handleMsg):**
1. Check `*Stale()` — if fresh, return cached data
2. Check `*Fetching()` — if already in-flight, return nil (skip)
3. Set `*Fetching(true)` — mark as in-flight
4. Dispatch `build*Cmd()` — API call
5. On response (`*LoadedMsg`): set `*Fetching(false)`, write data to Store

Pagination requests (offset > 0) bypass both staleness and sentinel checks — they are explicit continuations.

---

### Message Types

Every distinct piece of data coming back from async operations has its own message type. Named consistently as `<noun><verb>Msg`.

All message types are defined in `internal/ui/panes/messages.go`. Convention: `<Noun><Verb>Msg`, exported, with data payload + `Err error` fields.

---

## API Client Design

**Interfaces:** All Spotify operations are defined as interfaces (`PlayerAPI`, `LibraryAPI`, `DevicesAPI`, `UserAPI`, `SearchAPI`, `PlaylistsAPI`) in `internal/api/`. Panes depend on these interfaces for mockability. See `internal/api/player_interfaces.go`, `internal/api/library_interfaces.go`, `internal/api/devices_interfaces.go`, `internal/api/user_interfaces.go`, `internal/api/search_interfaces.go`, and `internal/api/playlists_interfaces.go`.

**HTTP Pattern:** All requests route through `BaseClient.doJSON`/`doNoContent` which handles auth headers, error parsing, and gateway routing. See `internal/api/base.go`.

**Pagination:** Generic `fetchAll[T]` helper fetches all pages with a safety cap. See `internal/api/pagination.go`.

**Additional API files:**
- `internal/api/logging.go` — `LoggingTransport` HTTP middleware (records all requests to Store NetLog)
- `internal/api/errors.go` — Custom error types: `RateLimitError`, auth errors, parsing helpers
- `internal/api/token.go` — Token refresh and validation helpers
- `internal/api/models.go` — Spotify API response model definitions
- `internal/api/browser.go` — Opens default browser for OAuth callback

---

## Auth Flow

PKCE OAuth 2.0 (Authorization Code + Proof Key). Tokens stored in OS keychain (`internal/keychain/`). Proactive refresh 5 minutes before expiry; on 401, refresh immediately and retry once. See `internal/keychain/` and `cmd/root.go`.

---

## Polling Architecture

Playback state must stay fresh. Use `tea.Tick` — never `time.Sleep`.

### Polling Ownership

The root model's 1-second tick loop is the single polling mechanism in the app. The base tick rate is 1 second; actual fetch intervals vary by idle state — see Idle Polling Backoff below.

| Tick Cycle | Endpoint | Owner | Consumers |
|---|---|---|---|
| Adaptive (3-30s) | `GET /me/player` | Feature 03 (Playback) | Features 03, 04, 07, 08 |
| Adaptive (9-60s) | `GET /me/player/queue` | Feature 06 (Queue) | Feature 06 |

**Rules:**
- Feature 03 owns the tick loop and dispatches `fetchPlaybackState` on each `tickMsg`
- Feature 06 extends the tick to also dispatch `fetchQueue` alongside the playback fetch
- Feature 04 (Library) fetches on-demand (Init, section expand, view open) with staleness gating — data within its TTL (albums/liked/playlists: 5m, recently played: 2m) is not re-fetched
- Feature 08 (Stats) fetches on-demand per time range with staleness gating — data within `StatsTTL` (10m) is not re-fetched even if the view is closed and reopened
- Feature 07 (Devices) only fetches when the device overlay opens — the overlay emits `FetchDevicesRequestMsg` on `Init()` and the app dispatches the API call. Device list is considered stale after `DevicesTTL` (5s). The short TTL balances freshness (devices appear/disappear externally) with API efficiency. Device fetches use `api.Interactive` priority to bypass the gateway token bucket.
- No feature other than 03 and 06 should add recurring poll commands to the tick cycle
- Library/stats use **staleness-based refresh**, not polling — see "Staleness Tracking" in State Management

### Idle Polling Backoff (Feature 33)

The actual polling intervals adapt based on user activity and playback state. This is
**Layer 1** (proactive) rate management; Feature 30 (API Gateway) is **Layer 2** (reactive).

| State | Playback Interval | Queue Interval |
|---|---|---|
| Active + Playing | 3s | 9s |
| Active + Paused | 10s | 30s |
| Idle + Playing | 10s | 30s |
| Idle + Paused | 30s | 60s |

"Active" means a `tea.KeyMsg` was received within the last 60 seconds.
"Idle" means no `tea.KeyMsg` for 60+ seconds.

**Implementation:**
- `App.lastInteraction` (type `time.Time`) is set to `time.Now()` in the `tea.KeyMsg` handler
- `App.isIdle()` returns `time.Since(lastInteraction) > idleThreshold` (60s)
- `App.pollIntervals()` reads `isIdle()` and `store.PlaybackState().IsPlaying` to pick intervals
- The tick handler calls `a.pollIntervals()` on every tick to get current intervals
- When a `KeyMsg` arrives after idle (`wasIdle && now active`), `tickCount` is reset to 0 to
  force an immediate fetch on the next tick — gives instant feedback on return from idle

### Search Debounce

Search uses a 300ms debounce via `tea.Tick`. When the user types, a debounce timer is scheduled. If the query changes before the timer fires, the stale timer is ignored. Only when 300ms elapse without further input is `SearchRequestMsg` dispatched with `api.Interactive` priority.

---

## Configuration

TOML-based (`internal/config/`). All fields have sensible defaults — an empty or missing config file is fine. Default theme is `black`. See `internal/config/config.go`.

---

## Testing Architecture

### Mock Client

**Mocking:** API interfaces are mocked via `internal/api/apitest/mock.go`. No external mock libraries.

### Pane Update Tests

See test files in `internal/ui/panes/` for update test patterns.

### Integration Test Convention

Integration tests verify multi-component interactions: message routing through the root model,
state updates across panes, and end-to-end user workflows with mocked HTTP.

**File naming:** `*_integration_test.go` (e.g., `app_integration_test.go`, `player_integration_test.go`)

**Build tag:** Every integration test file starts with:
```go
//go:build integration
```

**Running tests:**
- `make test` — runs unit tests only (fast, default)
- `make test-integration` — runs integration tests only
- `make ci` — runs both unit and integration tests

**What qualifies as an integration test:**
- Tests that exercise message routing through the root `app.Model`
- Tests that verify state changes propagate from one pane to another
- Tests that combine `httptest.NewServer` with multiple model updates in sequence
- Tests that verify the polling tick produces correct downstream state changes

**What stays as a unit test:**
- Individual API client methods with `httptest.NewServer` (testing one function)
- Store mutation methods (Get/Set)
- Bubble Tea model `Update()` handlers (testing one key → one command)
- `View()` output assertions
- Config loading, PKCE helpers, time formatters

---

## Notification System

All user-facing notifications use `go.dalton.dog/bubbleup` toast notifications rendered
by `internal/ui/components.NewNotifications`. Toast alerts overlay the current view and
auto-dismiss after a configurable duration.

### Toast Alert Types

| Key | Theme Token | Prefix | Use |
|---|---|---|---|
| `"success"` | `Success()` | `✓` | Successful user actions (queue add, transfer) |
| `"error"` | `Error()` | `✗` | API errors, failures |
| `"warning"` | `Warning()` | `!` | Soft failures (Premium required) |
| `"info"` | `KeyHint()` | `→` | Informational messages (device transfer initiated) |
| `"ratelimit"` | `Warning()` | `⧖` | 429 rate-limit back-off |

### How to Emit a Toast

Return `a.alerts.NewAlertCmd(alertType, message)` from any `Update()` handler:

```go
case SomeFailedMsg:
    if m.Err != nil {
        return a, a.alerts.NewAlertCmd("error", m.Err.Error())
    }
    return a, nil
```

### BubbleUp Integration Pattern

See `internal/app/app.go` Update() and View() for BubbleUp integration wiring.

**Important:** `AlertModel.View()` always returns `""`. Only `Render(content)` produces output.
Toast activation requires two Update passes: the first pass returns an `alertCmd`; executing
that `alertCmd` returns the internal `alertMsg`; feeding `alertMsg` to Update activates display.

---

## Error Handling Conventions

### Error Handling in build*Cmd Functions

API errors in `build*Cmd` functions MUST be surfaced to the user via toast notification.
**Silent swallowing is prohibited.**

**In build*Cmd (commands.go)** — commands NEVER write to the Store:
```go
// Return data or error in Msg payload — no store mutations here
if err != nil {
    return XxxLoadedMsg{Err: err}
}
return XxxLoadedMsg{Data: data}
```

**In Update() handler (app.go)** — Store writes happen here:
```go
case XxxLoadedMsg:
    if m.Err != nil {
        a.store.SetXxxError(m.Err)
        return a, a.alerts.NewAlertCmd("error", m.Err.Error())
    }
    a.store.ClearXxxError()
    a.store.SetXxx(m.Data)
```

Store error fields are **preserved** for retry logic (panes check them to decide whether to
re-request data on `f`/`Enter`) but are **never read in `View()`** — that is the toast's job.

### Pane Rendering Constraints

`View()` output MUST NOT exceed the height set by `SetSize()`. Panes with unbounded content
(queue, library sections, search results) must implement viewport scrolling. Rendering all
items in a loop without height capping is a bug.

### User-Facing Errors

All errors surface as toast notifications. Keep messages short and actionable.

| Error | Toast Type | User Message |
|---|---|---|
| 401 (re-auth needed) | `"error"` | `Session expired. Run: spotnik auth` |
| 403 (no premium) | `"warning"` | `Spotify Premium required for playback` |
| 429 (rate limited) | `"ratelimit"` | `Too many requests. Retrying in Ns...` |
| 503 (Spotify down) | `"error"` | `Spotify is unavailable. Retrying...` |
| Network error | `"error"` | `No connection to Spotify` |

Toasts auto-dismiss after `notificationDuration` (defined in `components/notifications.go`).
No manual dismissal timer or `statusDismissMsg` pattern is needed.

---

## Dependency Rules (Import Boundaries)

```
main.go
  └── cmd/
        └── internal/app/
              ├── internal/domain/   ← shared types (bridge between api/ and ui/)
              ├── internal/state/    ← reads store
              ├── internal/ui/       ← renders UI from store
              │     └── internal/ui/theme/
              ├── internal/api/      ← HTTP calls only
              ├── internal/config/   ← reads config
              └── internal/keychain/ ← token storage

SHARED IMPORTS (allowed):
  internal/api/   → internal/domain/  (API returns domain types)
  internal/ui/    → internal/domain/  (UI reads domain types)
  internal/state/ → internal/domain/  (Store holds domain types)

FORBIDDEN IMPORTS:
  internal/api/   → internal/ui/    (API must not know about UI)
  internal/ui/    → internal/api/   (UI must not call API directly)
  internal/state/ → internal/ui/    (State must not know about UI)
  internal/state/ → internal/api/   (State must not call API)
```

---

## Build & Release

`make build` produces a single binary at `bin/spotnik`. Cross-compilation targets and optimization flags are in the Makefile.

---

## API Gateway

All outbound HTTP traffic to Spotify routes through `internal/api/Gateway` (Feature 30).
The gateway provides four services in order:

### 1. Token Bucket Rate Limiter

A classic token-bucket (10 tokens/second, burst of 10) limits the total request throughput.
Background requests (polling, prefetch) are throttled through the bucket.
Interactive requests (user key presses) bypass the bucket entirely.

### 2. Concurrency Cap

A buffered channel of size 5 acts as a semaphore. At most 5 HTTP calls are in-flight
at any time. A 6th request blocks until one of the 5 completes or the context is cancelled.

### 3. In-Flight Request Deduplication

If two goroutines issue a request with the same `(Method, Path)` key simultaneously,
only one HTTP call is made. The second goroutine waits on a `done` channel and receives
a copy of the buffered response body. This prevents tick-storm duplicates during polling.

### 4. 429 Backoff with Priority Bypass

When Spotify returns a 429 response:
- The gateway sets `backoffUntil = now + Retry-After seconds`.
- Background requests are rejected immediately with `*RateLimitError`.
- Interactive requests wait (blocking) for the backoff to expire then proceed.

The app receives `RateLimitedMsg` and updates `store.SetThrottle()` for UI observability.
A `throttleExpiredMsg` fires after Retry-After seconds to clear the store throttle state.

### Priority Context

Callers tag a context with `api.WithPriority(ctx, api.Interactive)` for user actions.
Background is the default; `api.PriorityFromContext(ctx)` reads it in `BaseClient`.

Command builders in `internal/app/commands.go` set `Interactive` for:
play, pause, next, previous, volume, shuffle, repeat, search, add-to-queue,
like/unlike, transfer playback, create/rename/remove/reorder playlist, fetch devices.

### Gateway Observability

The gateway exposes its internal state for UI visualization via the `domain.GatewaySnapshotter` interface, keeping `internal/ui/` decoupled from `internal/api/`:

- `domain.GatewaySnapshotter` interface — implemented by `api.Gateway.Snapshot()`
- `domain.GatewayState` struct — snapshot of token bucket level, concurrency slots, backoff timer, dedup waiters, InFlightKeys
- `RequestFlowPane` (Page B) reads snapshots to render live gateway visualization
- `PollingSnapshotMsg` — carries app-level polling diagnostics (tick interval, idle state) to RequestFlowPane
- `RequestCompletedMsg` — carries per-request completion data (endpoint, status, latency, Priority, GatewayDecision) to RequestFlowPane

#### Gateway Decision Recording

Each request through the gateway is classified with a `domain.GatewayDecision`:

| Decision | Meaning |
|---|---|
| `DecisionAllowed` | Request proceeded normally through the gateway |
| `DecisionWaited` | Request was deduped — caller waited for an in-flight sibling to complete |
| `DecisionDeduped` | Joined an existing in-flight GET request; the caller shares the primary's cached response instead of making a new HTTP call |
| `DecisionBlocked` | Background request was rejected during active rate-limit backoff |

The `api.GatewayRecorder` interface (implemented by `state.Store`) bridges the `api/` → `state/` boundary without a direct import:

```go
type GatewayRecorder interface {
    RecordGatewayCall(method, path string, statusCode int, durationMs int64,
        priority domain.RequestPriority, decision domain.GatewayDecision)
}
```

Wired in `initAPIClients()` via `gateway.SetRecorder(store)`. Recorded entries are stored in the NetLog ring buffer alongside network-level entries.

#### Double-Recording Prevention

Both `Gateway.Do()` (via `GatewayRecorder`) and `LoggingTransport` (via `NetLogRecorder`) observe HTTP requests. To prevent double-counting gateway-routed requests:

- `api.MarkGatewayRecorded(req)` stamps an `http.Request` with a context key before it is dispatched through the gateway
- `api.IsGatewayRecorded(req)` returns true if the request was stamped
- `LoggingTransport.RoundTrip()` skips `RecordNetCall()` when `IsGatewayRecorded(req)` is true
- Direct API calls not routed through the gateway are still recorded by `LoggingTransport`

### Network Logging

All HTTP requests are logged to a ring buffer for the NetworkLogPane (Page B):

- `internal/state/netlog.go` — `NetLog` struct: 200-entry thread-safe ring buffer of `NetLogEntry` records (timestamp, method, path, status code, duration, Priority, GatewayDecision)
- `internal/api/logging.go` — `LoggingTransport` wraps `http.DefaultTransport`, intercepts requests not already recorded by the gateway
- `NetLogRecorder` interface bridges `api/` → `state/` without a direct import
- Data flow (gateway path): `BaseClient.doJSON` → `Gateway.Do()` → records via `GatewayRecorder` → `store.RecordGatewayCall()` → `NetLog`
- Data flow (direct path): `LoggingTransport.RoundTrip()` → `store.RecordNetCall()` → `NetLog`
- Wired in `initAPIClients()` — a shared `http.Client` with `LoggingTransport` is passed to all 6 API clients

### Integration Points

- `internal/api/gateway.go` — Gateway struct, tokenBucket, inflightEntry, Priority, GatewayRecorder interface, MarkGatewayRecorded/IsGatewayRecorded helpers
- `internal/api/base.go` — `BaseClient.SetGateway()`, `doJSON`/`doNoContent` routing, MarkGatewayRecorded call
- `internal/api/logging.go` — `LoggingTransport`, `NetLogRecorder` interface, double-recording skip
- `internal/app/app.go` — Gateway created in `New()`, `throttleExpiredMsg` handler
- `internal/app/auth.go` — `initAPIClients()` calls `SetGateway()`, wires `LoggingTransport`, calls `SetRecorder(store)`
- `internal/state/store.go` — `SetThrottle()`, `IsThrottled()`, `ThrottleRetryAfterSecs()`, `RecordGatewayCall()`
- `internal/state/netlog.go` — `NetLog` ring buffer, `NetLogEntry` (with Priority + GatewayDecision), `RecordNetCall()`, `RecordGatewayCall()`
- `internal/domain/gateway.go` — `GatewaySnapshotter`, `GatewayState`, `RequestPriority`, `GatewayDecision`

*Last updated: 2026-03-28*
