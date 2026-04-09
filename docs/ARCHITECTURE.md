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
- `gateway.go` — `EventKind` (13 constants), `GatewayStateSnapshot`, `GatewayEvent`, `GatewayEventRecorder` interface, `RequestPriority` constants
- `search.go` — `SearchResult` type

Panes import `domain/` types, not `api/` types. API clients return `domain/` types. This is how the import boundary is enforced — `ui/` and `api/` never import each other.

### Pane Interface

Every pane in `internal/ui/panes/` implements the `layout.Pane` interface:

```go
type Pane interface {
    tea.Model                          // Init, Update, View
    SetSize(width, height int)         // Content area dimensions (inside border)
    SetFocused(focused bool)           // Keyboard focus state
    IsFocused() bool                   // Query focus state
    ID() PaneID                        // Slot identifier
    Title() string                     // Display title for border
    ToggleKey() int                    // Toggle key number (1-8), 0 if not toggleable
    Actions() []Action                 // Pane-specific shortcuts for border
    SetTheme(th theme.Theme)           // Updates the pane's theme for runtime switching
}
```

`SetTheme` is called by the root model whenever the user changes the theme via the theme
switcher overlay. Table-based panes must rebuild their tables with new column colors when
`SetTheme` is called — lipgloss column styles are baked into the table at creation time
and must be refreshed explicitly.

---

## View Lifecycle

The app has three view modes managed by the `currentView` field in `internal/app/app.go`:

1. **`viewSplash`** — 5-second startup screen with ASCII banner (rendered by `splash.go`)
2. **`viewAuth`** — OAuth panel when authentication is needed (rendered by `auth.go`)
3. **`viewGrid`** — Normal operation with pane grid, header, and status bar

**Transitions:**

```
viewSplash
  ├── unauthenticated → viewAuth → viewGrid  (after PKCE flow completes)
  └── already authenticated → viewGrid       (splashDismissMsg fires after 5s)
```

The splash timer fires `splashDismissMsg` after 5 seconds. `viewAuth` transitions to
`viewGrid` once `authCompleteMsg` is received (PKCE callback handled in `cmd/root.go`).
No view can transition backwards — the lifecycle is strictly one-directional.

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
                    ├── renderStatusBar()   (3 lines: border + keybinding hints + border)
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

## Page / Preset / Toggle System

The grid layout has two pages and a preset system for switching between curated layouts.

### Pages

- **Page A (Music)** — 8 panes: NowPlaying, Queue, Playlists, Albums, LikedSongs,
  RecentlyPlayed, TopTracks, TopArtists
- **Page B (Nerd Status)** — 2 panes: RequestFlow, NetworkLog

`TogglePage()` switches between pages. The current page is stored as `currentPage` in `App`.
Key: `0`.

### Preset Cycling

`CyclePreset()` advances to the next preset within the current page and wraps around.
Key: `p`.

- **Page A** has 4 presets (Full Dashboard, Listening, Library, Stats)
- **Page B** has 1 preset

Each preset is a `[]Row` grid definition. Switching a preset resets all manual pane toggles.
Preset index is persisted via `PreferenceStore` so it survives restarts.

### Pane Toggling

`TogglePane(id layout.PaneID)` hides or shows an individual pane on Page A.
Keys: `1`–`8` (one per Page A pane).

- When a pane hides, its siblings in the same row expand to fill the space
- When all panes in a row hide, the row collapses and other rows expand
- Toggle state is independent of presets — switching preset resets manual toggles
- Page B panes are not individually toggleable

---

## Message Flow

```
User Keypress / Mouse Wheel
     │
     ▼
routing.go: handleKeyMsg / handleMouseMsg
     │
     ├── Guard 1: Theme overlay open → all keys to ThemeOverlay
     ├── Guard 2: Help overlay open → all keys to HelpOverlay
     ├── Guard 3: Device overlay open → all keys to DeviceOverlay
     ├── Guard 4: Search overlay open → all keys to SearchOverlay
     ├── Guard 5: Auth view → only quit keys
     ├── Guard 6: Pane has active filter → all keys to pane
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

### Overlay Routing Precedence

Key events are checked against guards in strict priority order. Earlier guards intercept
all input and prevent lower-priority handlers from running:

| Priority | Guard | Action |
|----------|-------|--------|
| 1 | Theme overlay open | All keys → ThemeOverlay |
| 2 | Help overlay open | All keys → HelpOverlay |
| 3 | Device overlay open | All keys → DeviceOverlay |
| 4 | Search overlay open | All keys → SearchOverlay |
| 5 | Auth view active | Only quit keys (`q`, `ctrl+c`) pass; all others dropped |
| 6 | Pane has active filter | All keys → focused pane (filter captures input) |
| 7 | Global shortcuts | `q`, `/`, `d`, `t`, `0`, `p`, `1`–`8`, `Tab`, `Shift+Tab` |
| 8 | Playback keys | `Space`, `n`, `+`, `-`, `s`, `r`, `v`, `←`, `→` → always NowPlayingPane |
| 8 | Default | All other keys → focused pane |

This means: if the device overlay is open, `q` goes to the overlay (not quit). Theme
overlay has the highest priority because it is opened by `t` after the global keys
check — once open, it must fully capture input.

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

## PreferenceStore

`internal/prefs/prefs.go` provides a coalescing preference writer that batches in-memory
changes and flushes them to disk in a single debounced write.

### Design

`PreferenceStore` holds a `pending map[string]any` of dirty preferences protected by a
`sync.Mutex`. `Set(key, value)` adds to the `pending` map under the mutex without writing
to disk. `FlushCmd()` returns a `tea.Cmd` that snapshots and clears `pending` (under the
mutex), then reads the existing TOML config, applies the snapshot to the `[preferences]`
section, and writes it back. On write failure the snapshot entries are re-queued for any
keys not superseded by a newer `Set()` call.

### Supported Preferences

| Key | Type | Description |
|-----|------|-------------|
| `theme` | `string` | Active theme ID (e.g., `"black"`, `"dracula"`) |
| `preset` | `int` | Active preset index within the current page |
| `visualizer` | `int` | Active visualizer animation pattern index |

### Disk Path

Preferences are read from and written to the same TOML config file as the main config
(default: `~/.config/spotnik/config.toml`). The `[preferences]` section is updated
in-place; all other sections are preserved.

### Wiring

`App` holds a `*prefs.PreferenceStore`. When the user changes a preference (theme switch,
preset cycle, visualizer toggle), `Update()` calls `prefs.Set(key, value)` and returns
`prefs.FlushCmd()` as a command. The resulting `prefs.FlushedMsg` is handled in `handleMsg`
— errors are surfaced as toast notifications.

---

## API Client Design

**Interfaces:** All Spotify operations are defined as interfaces (`PlayerAPI`, `LibraryAPI`, `DevicesAPI`, `UserAPI`, `SearchAPI`, `PlaylistsAPI`) in `internal/api/`. Panes depend on these interfaces for mockability. See `internal/api/player_interfaces.go`, `internal/api/library_interfaces.go`, `internal/api/devices_interfaces.go`, `internal/api/user_interfaces.go`, `internal/api/search_interfaces.go`, and `internal/api/playlists_interfaces.go`.

**HTTP Pattern:** All requests route through `BaseClient.doJSON`/`doNoContent` which handles auth headers, error parsing, and gateway routing. See `internal/api/base.go`.

**Pagination:** Generic `fetchAll[T]` helper fetches all pages with a safety cap. See `internal/api/pagination.go`.

**Additional API files:**
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
- `make ci` — runs unit tests only (fmt-check → tidy-check → lint → test-coverage → build); run `make test-integration` separately for integration tests

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

`RequestFlowPane` (Page B) reads gateway events from the store's event journal using a
cursor-based replay model. The pane never holds a gateway reference — it only reads from
`*state.Store`, preserving the `ui/ → state/` dependency direction.

- `PollingSnapshotMsg` — carries app-level polling diagnostics (tick interval, idle state) to RequestFlowPane
- `replayDisplayState` — single render model that `View()` reads from; updated by the replay loop on each `viz.TickMsg`
- `eventCursor uint64` — cursor into `GatewayEventLog`; advanced by `drainEvents()` on each tick
- `replayQueue []domain.GatewayEvent` — events waiting to be displayed at 200ms minimum visibility
- `requestAnimation` — tracks one request's visual state (method, path, priority, phase, decision, status) across all three boxes
- `decisionEntry` — one line in the GATEWAY box's scrolling decision log (kind, label, shownAt for age-out)

#### Request Flow Replay Loop

On each `viz.TickMsg` (200ms), the pane:
1. **`drainEvents()`** — reads new events from `store.ReadEventsFrom(cursor)`, appends to `replayQueue`
2. **`processNextEvent()`** — pops one event, updates `displayState.snapshot` and request animation phases
3. **`ageOutEntries()`** — removes decisions older than 3s, completed requests older than 5s

#### Request Flow Rendering

`RequestFlowPane.View()` uses a **boxed layout** (Feature 62) when pane width ≥ 60 columns:

```
╭─ APP ──────────╮           ╭─ GATEWAY ──────────╮           ╭─ SPOTIFY ──────╮
│ ▶ /player      │───────→───│ tokens  ●●●● 10/10 │───────→───│  200  45ms     │
│   /queue       │───→ dedup │ conc    □□□□□  0/5 │    ╳      │  200  62ms     │
│                │           │ ✓ GET /player allow │           │                │
╰────────────────╯           ╰────────────────────╯           ╰────────────────╯
POLLING  tick: 1000ms  state: active    STORE  fetching: []
```

- **Three sub-boxes**: APP (endpoints), GATEWAY (state bars + decision log), SPOTIFY (responses) with rounded corners
- **Dual arrow columns**: left (APP→GW decision based on `requestAnimation.decision` EventKind), right (GW→SPOTIFY outcome based on phase + status code)
- **GATEWAY box sections**: state bars (token bucket bar + semaphore bar + optional backoff timer) from `displayState.snapshot`; scrolling decision log below with per-EventKind theme colors
- **Decision log colors**: `✓` allowed/expired → Success; `✗` blocked → Error; `⧖` waited/dedup → Warning; resource events → TextSecondary; `↻` refill → TextMuted
- **`renderSubBox(title, lines, width)`** — pure helper that draws rounded-corner box; used by all three sub-boxes
- **`formatDecisionLabel(e GatewayEvent) string`** — maps all 13 EventKind values to display strings
- **Flat fallback** (`viewFlat()`): original column headers + request rows + gateway state block; used when width < 60
- Status strip always spans full pane width below the boxes

#### Gateway Event Emission (Feature 67)

`Gateway.Do()` emits fine-grained lifecycle events at every decision point via `emitEvent()`/`emitEventLocked()`:

| Event | When emitted |
|---|---|
| `EventRequestEntered` | Entry to `Do()`, before any policy checks |
| `EventTokenConsumed` | After `bucket.wait()` returns for Background requests |
| `EventSemaphoreAcquired` | After acquiring a concurrency slot |
| `EventSemaphoreReleased` | Deferred on slot release |
| `EventRequestBlocked` | Background request rejected by backoff or bucket cancellation |
| `EventRequestWaited` | Interactive request waited on backoff timer |
| `EventDedupJoined` | GET waiter joins an in-flight dedup entry |
| `EventDedupResolved` | Dedup waiter received the shared response |
| `EventHttpCompleted` | After `fn()` returns, with status code and latency |
| `EventBackoffStarted` | After 429 response sets `backoffUntil` |
| `EventRequestAllowed` | Primary caller succeeded (no backoff wait) |

Lock ordering: `emitEvent()` acquires `g.mu` then `bucket.mu`. `emitEventLocked()` assumes `g.mu` is already held and only acquires `bucket.mu`. `bucket.mu` is never held when calling `emitEvent()`.

Periodic events are emitted on `viz.TickMsg` (every 200ms) from `app.go`:

- **`CheckAndEmitRefill()`** — emits `EventTokenRefilled` when the bucket level changes from the last emission (lazy: does not mutate `bucket.tokens`)
- **`CheckAndEmitBackoffExpiry()`** — emits `EventBackoffExpired` on the active→cleared transition only

Each request gets a unique `RequestID` from `nextRequestID atomic.Uint64`. All events for the same request share this ID. Internal events (TokenRefilled, BackoffExpired) use `RequestID = 0`.

#### Feature 68: Replay Engine (completed)

`GatewayState`, `GatewaySnapshotter`, and `GatewayDecision` have been removed from
`domain/gateway.go`. The deprecated `Snapshot()` shim and `ResetWatermarks()` no-op have
been removed from `api/gateway.go`. All snapshot-based tests have been rewritten to use
event injection via `store.RecordEvent()` + `viz.TickMsg`.

#### Feature 69: Network Log Event Migration (completed)

`NetLog`, `NetLogEntry`, `RecordNetCall`, `RecordGatewayCall`, `NetLogEntries`, and
`LoggingTransport`/`NetLogRecorder` have been removed. NetworkLogPane now reads directly
from `GatewayEventLog` via cursor-based `ReadEventsFrom()`. Blocked requests that never
reached HTTP are now visible (status 0, "✗ blocked"). PRIORITY and DECISION columns added.

### Network Logging (Feature 69+)

All HTTP requests are logged to the `GatewayEventLog` for the NetworkLogPane (Page B).
`NetLog`, `NetLogEntry`, `RecordNetCall`, and `LoggingTransport` were retired in Feature 69
— `GatewayEventLog` is the single authoritative source for both NetworkLogPane and
RequestFlowPane.

- Data flow: `BaseClient.doJSON` → `Gateway.Do()` → `store.RecordEvent()` → `GatewayEventLog`
- NetworkLogPane drains events via cursor-based `store.ReadEventsFrom(cursor)` on each 1s tick
- Columns: TIME, METHOD, ENDPOINT, STATUS, LATENCY, PRI (int/bg), DECISION (allowed/blocked/waited/dedup), NOTES
- Blocked requests (`EventRequestBlocked`) appear with status 0 and "✗ blocked" in NOTES
- Interactive vs background priority visible in PRI column; gateway decision in DECISION column

### Gateway Event Journal (Feature 66+)

The gateway event journal is a timestamped event stream that replaced snapshot-polling
and the old `NetLog` ring buffer.

- `internal/domain/gateway.go` — `EventKind` (13 constants), `GatewayStateSnapshot`, `GatewayEvent`, `GatewayEventRecorder` interface
- `internal/state/eventlog.go` — `GatewayEventLog`: 500-entry thread-safe ring buffer with cursor-based reads
  - `Add(event)` — write path; called by `Store.RecordEvent()`
  - `ReadFrom(cursor)` — returns events since cursor; multiple independent consumers (RequestFlowPane, NetworkLogPane) each hold their own cursor
- `internal/state/store.go` — `RecordEvent()` implements `domain.GatewayEventRecorder`; `ReadEventsFrom()` exposes cursor reads to the UI

### Integration Points

- `internal/api/gateway.go` — Gateway struct, tokenBucket, inflightEntry, Priority, emitEvent helpers
- `internal/api/base.go` — `BaseClient.SetGateway()`, `doJSON`/`doNoContent` routing
- `internal/app/app.go` — Gateway created in `New()`, `throttleExpiredMsg` handler
- `internal/app/auth.go` — `initAPIClients()` creates plain `http.Client`, calls `SetGateway()` and `SetRecorder(store)`
- `internal/state/store.go` — `SetThrottle()`, `IsThrottled()`, `ThrottleRetryAfterSecs()`, `RecordEvent()`, `ReadEventsFrom()`
- `internal/state/eventlog.go` — `GatewayEventLog` ring buffer with cursor-based reads
- `internal/domain/gateway.go` — `RequestPriority`, `EventKind`, `GatewayStateSnapshot`, `GatewayEvent`, `GatewayEventRecorder`

*Last updated: 2026-03-29*
