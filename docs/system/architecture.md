# Architecture — Technical Reference

> Reference only. Feature specs embed patterns inline.
> Consult when feature spec points deeper. Don't read cover-to-cover.

---

## Architectural Overview

Spotnik = Elm Architecture (Bubble Tea enforced). App = pure function of state: `View(State) → UI`. Side effects only via commands + messages.

```
┌──────────────────────────────────────────────────────────┐
│                        main.go                           │
│                    (entry point only)                    │
└─────────────────────────┬────────────────────────────────┘
                          │
┌─────────────────────────▼────────────────────────────────┐
│                      cmd/root.go                         │
│         (flag parsing, config load, PKCE callback,       │
│          auth check, app launch)                         │
└─────────────────────────┬────────────────────────────────┘
                          │
┌─────────────────────────▼────────────────────────────────┐
│                    internal/app/                          │
│              Root Bubble Tea Model (tea.Model)            │
│   app.go       — Init/Update, handleMsg, polling tick     │
│   render.go    — View composition, grid, overlays         │
│   routing.go   — Key/mouse dispatch, focus rotation       │
│   handlers.go  — Per-msg handlers                         │
│   commands.go  — 30+ build*Cmd factories (no store writes)│
│   auth.go      — PKCE flow, API client wiring             │
│   splash.go    — Startup splash screen                    │
│   clipboard.go — copyToClipboardCmd side-effect Cmd       │
│   prefs.go     — PreferenceStore wiring                   │
│   seek_test.go, volume_test.go, volume_internal_test.go   │
└──┬─────────────┬──────────────┬──────────────┬───────────┘
   │             │              │              │
   │    ┌────────▼────────┐    │              │
   │    │ internal/domain │    │              │
   │    │ (shared types)  │    │              │
   │    └──┬──────────┬───┘    │              │
   │       │          │        │              │
┌──▼───────▼──┐  ┌────▼─────────▼──┐  ┌──────▼───────┐
│ internal/ui/│  │  internal/api/  │  │internal/uikit│
│  panes/     │  │  (HTTP clients, │  │ (design-sys: │
│  layout/    │  │   gateway,      │  │  EmptyState, │
│  components/│  │   token bucket, │  │  ToastMgr,   │
│  theme/     │  │   dedup, auth)  │  │  ErrorMapper,│
│             │  │                 │  │  Border,     │
│             │  │                 │  │  Panel, Chip)│
└──────┬──────┘  └────────┬────────┘  └──────────────┘
       │                  │
       └─────────┬────────┘
                 │
       ┌─────────▼─────────┐    ┌────────────────────┐
       │  internal/state/  │    │ internal/cliout/   │
       │  Store (single    │    │ (CLI render lib:   │
       │  source of truth) │    │  Builder, Message, │
       │  + eventlog       │    │  Palette, Spinner) │
       └─────────┬─────────┘    └────────────────────┘
                 │
   ┌─────────────┼─────────────┬──────────────┐
   │             │             │              │
┌──▼────┐  ┌─────▼─────┐  ┌────▼─────┐  ┌────▼─────────┐
│config/│  │ keychain/ │  │ prefs/   │  │ goldentest/  │
│       │  │           │  │          │  │ +testhelpers │
└───────┘  └───────────┘  └──────────┘  └──────────────┘

Codebase knowledge graph: graphify-out/ (GRAPH_REPORT.md, graph.json,
manifest.json — PR #387. Not Go code. Navigation aid only.)
```

### Panes (14 PaneIDs, 2 pages)

```
Grid panes (LayoutManager):
  Player page (10): NowPlaying (content-aware: track vs episode),
                    Queue, FollowedShows (drill-down),
                    SavedEpisodes, Playlists, Albums, LikedSongs,
                    RecentlyPlayed, TopTracks, TopArtists
  Stats page (5):   NowPlaying (shared), GatewayHealth, PollingTraffic,
                    GatewayLive, NetworkLog

Floating overlays (7, not in grid):
  SearchOverlay, DeviceOverlay, ProfileOverlay, ThemeOverlay,
  HelpOverlay, EpisodeDetailsOverlay, OnboardingPermissionsOverlay

Deleted panes: PodcastPlaybackPane, ShowEpisodesPane
  (unified into content-aware NowPlaying + FollowedShows drill-down)
```

### Domain Package

`internal/domain/` = shared types bridging `api/` ↔ `ui/` without import cycles. Key files:

- `types.go` — `PlaybackState`, `Track`, `Artist`, `Album`, `Device`, `SimplePlaylist`, `SavedAlbum`, `SavedTrack`, `PlayHistory`, `QueueResponse` (`Queue []QueueItem` — mixed content, #336), `FullArtist`, `PlayOptions`, `QueueItem` (`Type` + `Track *Track` + `Episode *Episode`), `QueueItemType` enum
- `gateway.go` — `EventKind` (**12 constants**), `GatewayStateSnapshot`, `GatewayEvent`, `GatewayEventRecorder` interface, `RequestPriority` constants
- `search.go` — `SearchResult` type

Panes import `domain/` types, not `api/` types. API returns `domain/` types. This enforces boundary — `ui/` ↔ `api/` never import each other.

### Pane Interface

Every pane in `internal/ui/panes/` implements `layout.Pane` (`internal/ui/layout/pane.go:72`):

```go
type Pane interface {
    tea.Model
    SetSize(width, height int)
    SetFocused(focused bool)
    IsFocused() bool
    ID() PaneID
    Title() string
    ToggleKey() int
    Actions() []Action
    SetTheme(th theme.Theme)
}
```

Optional: `FilterablePane` (`pane.go:98`), `FilterQueryPane` (`pane.go:107`).

`SetTheme` called by root on theme switch. Table panes rebuild tables with new column colors — lipgloss column styles baked at creation, must refresh explicitly.

### uikit — Design-System Layer

`internal/uikit/` (56 files, PR #407). Design-system primitives shared by panes + app:

- `EmptyState` + `PaneFetchState` — 5-status empty-state protocol: `None`, `NeverFetched`, `Fetching`, `Error`, `RateLimited` (`uikit/empty_state.go:12,36`)
- `PaneEmptyStatus()` factory — derives status from `PaneFetchState` (IsFetching/FetchErr/NeverFetched/IsThrottled/RetryAfterSecs) (`uikit/empty_state.go:148`)
- `ToastManager` + `ErrorMapper` — typed toast API, replaces direct `bubbleup` usage (`app.go:75-76`)
- `Border`, `Panel`, `Chip`, `SectionLabel`, `ListRow`, `ProgressBar`, `Spinner`, `StatusGlyph`, `UrlBox`
- `PaneChrome`, `OverlayChrome` — border rendering contracts (see `tui.md §3.1`, `§3.2`)
- `Role`, `Glyph`, `Sizes`, `Config`, `Capture`
- `InfoBox` — sub-pane with rounded border, used by NowPlaying + onboarding (`uikit/info_box.go:20`)

`uikit/` imports `ui/theme/` only. No `api/` or `state/` deps.

---

## View Lifecycle

App has 3 view modes (`currentView` in `internal/app/app.go:32-35`):

1. **`viewSplash`** — 5s startup ASCII banner (`splash.go`)
2. **`viewOnboarding`** — 3 sub-steps (`app.go:38-42`):
   - `stepRegister` — Client ID input
   - `stepOAuth` — browser auth + callback wait
   - `stepError` — retry/relaunch/quit
3. **`viewGrid`** — normal ops: pane grid + header + status bar

```
viewSplash
  ├── unauthenticated → viewOnboarding
  │     ├── stepRegister → stepOAuth → viewGrid  (PKCE callback OK)
  │     └── stepOAuth → stepError → (retry/relaunch/quit)
  └── already authenticated → viewGrid  (splashDismissMsg after 5s)
```

No backward transitions. Lifecycle strictly one-directional.

---

## Render Pipeline

View composition flow (`internal/app/render.go`):

```
View()
  └── alerts.Render(buildView())     ← toast overlay ALWAYS last
        └── buildView()
              ├── Terminal too small? → renderTooSmall()  (min 120x30)
              ├── viewSplash?        → renderSplash()
              ├── viewOnboarding?    → renderOnboarding()
              │     ├── renderOnboardingRegister()
              │     ├── renderOnboardingOAuth()
              │     │     └── OnboardingPermissionsOverlay if open
              │     └── renderOnboardingError()
              └── viewGrid:
                    ├── renderHeader()      (1 line: app, page, shortcuts, device)
                    ├── renderGrid()        (pane grid with borders)
                    ├── renderStatusBar()   (3 lines: border + hints + border)
                    └── Overlay compositing:
                          ├── deviceOverlayOpen?           → btoverlay.Composite(device, dimmed, Right, Top)
                          ├── searchOpen?                  → btoverlay.Composite(search, dimmed, Center, Center)
                          ├── profileOverlayOpen?          → btoverlay.Composite(profile, ...)
                          ├── themeOverlayOpen?            → btoverlay.Composite(theme, ...)
                          ├── helpOpen?                    → btoverlay.Composite(help, ...)
                          ├── episodeDetailsOpen?          → btoverlay.Composite(episodeDetails, ...)
                          └── onboardingPermissionsOpen?   → (only in onboarding view)
```

**Rules:**
- `alerts.View()` always returns `""` — must use `alerts.Render(content)`
- `renderGrid()` groups panes by row, wraps each in borders via `layout.RenderPaneBorder()`, applies `lipgloss.Width/MaxWidth/Height/MaxHeight` caps, joins horizontally per row, vertically across rows
- Overlays use `bubbletea-overlay` (`btoverlay.Composite`) — background dimmed with `Faint(true)`
- Total view output must equal exactly `terminalHeight` lines

---

## Page / Preset / Toggle System

2-page model + preset system for switching curated layouts. Podcast content integrated directly into Player page panes — no separate Podcasts page.

### Pages

- **Player page** — 10 panes across 6 presets: NowPlaying (content-aware), Queue, FollowedShows (drill-down), SavedEpisodes, Playlists, Albums, LikedSongs, RecentlyPlayed, TopTracks, TopArtists
- **Stats page** — 5 panes: NowPlaying, GatewayHealth, PollingTraffic, GatewayLive, NetworkLog

`TogglePage()` (`layout.go:245`) switches pages. `currentPage` stored in `App`. Key `0` cycles Player → Stats → Player (2-cycle, enforced by `layout_test.go:157`).

### Preset Cycling

`CyclePreset()` advances within current page, wraps. Key `p`.

| Player Preset | Name | Visible Panes |
|---|---|---|
| 0 | Dashboard | 8 panes (3 rows) — no FollowedShows/SavedEpisodes |
| 1 | Listening | NowPlaying, Queue, RecentlyPlayed |
| 2 | Podcast | NowPlaying, FollowedShows, Queue |
| 3 | Library | NowPlaying (compact), Playlists, Albums, LikedSongs |
| 4 | Discovery | NowPlaying (compact), TopTracks, TopArtists, RecentlyPlayed |
| 5 | PodcastDashboard | NowPlaying, FollowedShows, SavedEpisodes, Queue |

Stats page: 1 preset. Row 2 weights `1:1:3` (GatewayHealth:PollingTraffic:GatewayLive — GatewayLive dominant, #409 traffic shaping).

Each preset = `[]Row` grid definition. Switching resets manual pane toggles. Preset index persisted via `PreferenceStore`.

### Pane Toggling (Context-Aware)

`TogglePane(id layout.PaneID)` hides/shows individual pane. Toggle keys **context-aware** — adapt by preset page:

- **Player page**: keys `1`–`8` toggle 8 Player panes
- **Stats page**: keys `2`–`5` toggle Stats diagnostic panes (key `1` unused)

When pane hides, siblings in row expand. When all panes in row hide, row collapses. Toggle state independent of presets — switching resets manual toggles.

---

## Message Flow

```
User Keypress / Mouse Wheel
     │
     ▼
routing.go: handleKeyMsg / handleMouseMsg
     │
     ├── Guard 1: Theme overlay open     → all keys to ThemeOverlay
     ├── Guard 2: Help overlay open      → all keys to HelpOverlay
     ├── Guard 3: EpisodeDetails open    → all keys to EpisodeDetailsOverlay
     ├── Guard 4: Profile overlay open   → all keys to ProfileOverlay
     ├── Guard 5: Device overlay open    → all keys to DeviceOverlay
     ├── Guard 6: Search overlay open    → all keys to SearchOverlay
     ├── Guard 7: Onboarding view        → only quit keys
     ├── Guard 8: Pane has active filter → all keys to pane
     ├── Global keys (q, /, d, u, t, 0, p, 1-8, Tab, Shift+Tab)
     ├── Playback keys (Space, n, +, -, s, r, v, i, ←, →, Shift+←, Shift+→) → NowPlayingPane
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
                     ├── Emit toast if error
                     └── Forward to pane, re-render
```

### Overlay Routing Precedence

Strict priority order. Earlier guards intercept all input, prevent lower handlers:

| Priority | Guard | Action |
|---|---|---|
| 1 | Theme overlay open | All keys → ThemeOverlay |
| 2 | Help overlay open | All keys → HelpOverlay |
| 3 | EpisodeDetails overlay open | All keys → EpisodeDetailsOverlay |
| 4 | Profile overlay open | All keys → ProfileOverlay |
| 5 | Device overlay open | All keys → DeviceOverlay |
| 6 | Search overlay open | All keys → SearchOverlay |
| 7 | Onboarding view | Only `q`, `ctrl+c` pass |
| 8 | Pane has active filter | All keys → focused pane |
| 9 | Global shortcuts | `q`, `/`, `d`, `u`, `t`, `0`, `p`, `1`–`8`, `Tab`, `Shift+Tab` |
| 10 | Playback keys | `Space`, `n`, `+`, `-`, `s`, `r`, `v`, `i`, `←`, `→`, `Shift+←`, `Shift+→` → NowPlayingPane |
| 11 | Default | All other keys → focused pane |

If device overlay open, `q` goes to overlay (not quit). Theme overlay highest priority — opened by `t` after global check, must fully capture once open.

### Mouse Support

`handleMouseMsg` (`routing.go`): wheel up/down → `j`/`k` msgs, hit-tested via `layout.PaneAt(x, y)`, routed to target pane WITHOUT changing keyboard focus. Mouse ignored when overlays open.

### Data-Carrying Messages (Elm Purity)

**Rule: `build*Cmd` / `fetch*Cmd` MUST NOT write Store.** Only `Update()` mutates Store. Cmds return data in Msg payloads. `Update()` reads payload, writes Store.

**Wrong:**
```go
func fetchQueueCmd(player api.PlayerAPI, store *state.Store) tea.Cmd {
    return func() tea.Msg {
        qr, err := player.Queue(ctx)
        store.SetQueue(qr.Queue)   // ← violates Elm
        return panes.QueueLoadedMsg{}
    }
}
```

**Correct:**
```go
func fetchQueueCmd(player api.PlayerAPI) tea.Cmd {
    return func() tea.Msg {
        qr, err := player.Queue(ctx)
        if err != nil { return panes.QueueLoadedMsg{Err: err} }
        return panes.QueueLoadedMsg{Tracks: qr.Queue}
    }
}

// In app.go Update():
case panes.QueueLoadedMsg:
    if m.Err != nil {
        a.store.SetQueueError(m.Err)
    } else {
        a.store.ClearQueueError()
        a.store.SetQueue(m.Tracks)  // ← Store write only here
    }
```

All msg types in `internal/ui/panes/messages.go` carry data payload + `Err error`. `Update()` sole writer to Store.

Verified clean: `internal/app/elm_purity_test.go` enforces.

---

## State Management

### Store

`internal/state/store.go` = single source of truth. All API data lives here. Panes **read** via `StateReader` interface (`state/reader.go:19`), **never write** directly — dispatch msgs, root model updates store.

### Staleness Tracking

Each data domain carries `fetchedAt time.Time`. `Set*()` stamps `time.Now()`. `Update()` uses timestamps to skip unnecessary re-fetches.

```go
func IsStale(fetchedAt time.Time, ttl time.Duration) bool {
    return fetchedAt.IsZero() || time.Since(fetchedAt) > ttl
}
```

**TTL constants (`internal/state/store.go:29-50`):**

| Domain | TTL | Rationale |
|---|---|---|
| `PlaylistsTTL` | 5 min | Changes infrequently |
| `AlbumsTTL` | 5 min | Changes infrequently |
| `LikedTracksTTL` | 5 min | Changes infrequently |
| `FollowedShowsTTL` | 5 min | Changes infrequently |
| `SavedEpisodesTTL` | 5 min | Changes infrequently |
| `ShowEpisodesTTL` | 5 min | Changes infrequently |
| `RecentlyPlayedTTL` | 2 min | Changes with every playback event |
| `StatsTTL` | 10 min | Spotify updates slowly |
| `DevicesTTL` | 5 sec | Volatile — short cooldown prevents rapid-fire while ensuring fresh on user request |

**Staleness classification:**

| Category | Domains | TTL |
|---|---|---|
| **Stable** (staleness-gated, long TTL) | Playlists, Albums, Liked Tracks, FollowedShows, SavedEpisodes, Stats | 5-10 min |
| **Volatile** (staleness-gated, short TTL) | Devices, Recently Played | 5s-2min |
| **Real-time** (polled on tick, no TTL) | Playback State, Queue | N/A — overwritten every tick |

Playback + queue not staleness-tracked (overwritten every tick). Staleness only for on-demand: library, stats, devices, podcasts.

### Fetching Sentinels (TOCTOU Prevention)

Boolean sentinels prevent duplicate requests between staleness check + API response:

- `playlistsFetching`, `albumsFetching`, `likedFetching`, `recentFetching`, `devicesFetching`
- `followedShowsFetching`, `savedEpisodesFetching` (+ err fields)
- `statsFetching map[string]bool` — keyed by time range

**Guard pattern (`app.go` handleMsg):**
1. Check `*Stale()` — fresh → return cached
2. Check `*Fetching()` — in-flight → return nil
3. Set `*Fetching(true)`
4. Dispatch `build*Cmd()`
5. On `*LoadedMsg`: set `*Fetching(false)`, write Store

Pagination (offset > 0) bypasses staleness + sentinel checks.

### Like/Unlike Infrastructure (#384/#385)

Store holds like-state: `AddLikedTrack()`, `RemoveLikedTrack()`, `IsTrackLiked()` (`store.go:361-427`). Toggle is optimistic — store updated immediately, rolled back on failure.

Msgs: `ToggleLikeRequestMsg`, `ToggleLikeResultMsg` (carries rollback flag on err).

Emitting panes: LikedSongs, Queue, TopTracks, RecentlyPlayed, Playlists (track sub-view), Albums (track sub-view). **NowPlaying does NOT emit** (`nowplaying_test.go:2138` — "playback control pane"). Search: only track results. Queue: only track rows (episodes no-op).

### Message Types

Every distinct data piece from async ops has own msg type. Convention: `<Noun><Verb>Msg`, exported, data payload + `Err error` field. All in `internal/ui/panes/messages.go`.

---

## PreferenceStore

`internal/prefs/prefs.go` — coalescing preference writer. Batches in-memory changes, flushes debounced single write.

### Design

`PreferenceStore` holds `pending map[string]any` under `sync.Mutex`. `Set(key, value)` adds to `pending` without disk write. `FlushCmd()` returns `tea.Cmd` that snapshots + clears `pending`, reads existing TOML config, applies snapshot to `[preferences]` section, writes back. On write failure, snapshot re-queued for keys not superseded by newer `Set()`.

### Supported Preferences

| Key | Type | Description |
|---|---|---|
| `theme` | `string` | Active theme ID (e.g. `"black"`, `"dracula"`, `"mono-dark"`) |
| `preset` | `int` | Active preset index within current page |
| `visualizer` | `int` | Active visualizer animation pattern index |

### Disk Path

Same TOML config as main config (default `~/.config/spotnik/config.toml`). `[preferences]` section updated in-place; other sections preserved.

### Wiring

`App` holds `*prefs.PreferenceStore`. On preference change (theme switch, preset cycle, viz toggle), `Update()` calls `prefs.Set(key, value)`, returns `prefs.FlushCmd()`. `prefs.FlushedMsg` handled in `handleMsg` — errors as toasts.

---

## API Client Design

**Interfaces** (`internal/api/`): `PlayerAPI`, `LibraryAPI`, `DevicesAPI`, `UserAPI`, `SearchAPI`, `PlaylistsAPI`, **`PodcastAPI`** (`api/podcast_interfaces.go:11`). Panes depend on interfaces for mockability.

**HTTP Pattern:** All requests route through `BaseClient.doJSON`/`doNoContent` → `*Gateway` when attached. Never `http.Client.Do` directly.

**Pagination:** Generic `fetchAll[T]` helper, all pages + safety cap (`api/pagination.go`).

**API files:**
- `base.go` — `BaseClient`, `doJSON`/`doNoContent`, gateway routing
- `gateway.go` — `Gateway`, priority queues, emitEvent
- `gateway_bucket.go` — `tokenBucket` (split from gateway.go)
- `gateway_dedup.go` — `inflightEntry`, dedup map (split from gateway.go)
- `auth.go`, `token.go` — `RefreshableTokenProvider` (#396 proactive refresh)
- `player.go`, `library.go`, `search.go`, `playlists.go`, `devices.go`, `user.go`, `podcast.go`
- `podcast_interfaces.go` — `PodcastAPI`
- `models.go` — Spotify response models
- `errors.go` — `RateLimitError`, auth errors
- `browser.go` — OAuth browser open
- `apitest/mock.go` — mock client (no external mock libs)

---

## Auth Flow

PKCE OAuth 2.0 (Authorization Code + Proof Key). Tokens in OS keychain (`internal/keychain/`).

**Proactive refresh (#396):** `RefreshableTokenProvider` (`token.go:37`) refreshes 5 min before expiry (`refreshThreshold`, `token.go:35`) on `AccessToken()` call — no 401 needed. Single-flight mutex (`token.go:63`) prevents concurrent refresh.

**401 path:** refresh immediately, retry once. **403:** `"warning"` toast "Spotify Premium required".

See `internal/keychain/` + `cmd/root.go`.

---

## Polling Architecture

Playback state stays fresh. Use `tea.Tick` — never `time.Sleep`.

### Polling Ownership

Root model 1s tick = single polling mechanism. Base 1s; actual intervals vary by idle state.

| Tick Cycle | Endpoint | Owner | Consumers |
|---|---|---|---|
| Adaptive (3-30s) | `GET /me/player` | Feature 03 | 03, 04, 07, 08 |
| Adaptive (9-60s) | `GET /me/player/queue` | Feature 06 | 06 |

### Idle Polling Backoff (Feature 33)

Adaptive intervals by activity + playback state. **Layer 1** proactive; Gateway (Feature 30) = **Layer 2** reactive.

| State | Playback | Queue |
|---|---|---|
| Active + Playing | 3s | 9s |
| Active + Paused | 10s | 30s |
| Idle + Playing | 10s | 30s |
| Idle + Paused | 30s | 60s |

"Active" = `tea.KeyMsg` within last 60s. "Idle" = 60s+ since last KeyMsg.

**Impl:**
- `App.lastInteraction` set `time.Now()` in KeyMsg handler
- `App.isIdle()` → `time.Since(lastInteraction) > 60s`
- `App.pollIntervals()` reads `isIdle()` + `store.PlaybackState().IsPlaying`
- Tick handler calls `pollIntervals()` every tick
- KeyMsg after idle: `tickCount` reset to 0 → immediate fetch on next tick

### Visibility-Gated Polling

Library panes (Playlists, Albums, LikedSongs, RecentlyPlayed, TopTracks, TopArtists, FollowedShows, SavedEpisodes) use visibility-gated polling. Tick loop iterates domains, **skips fetches for panes not visible** in current preset/page.

**Rules:**
1. Each polling entry checks `layout.IsPaneVisible(paneID)` at top — skip if not visible
2. On preset switch (`CyclePreset`/`SetPreset`), app checks staleness for newly visible panes, fetches if stale
3. **Always poll**: Playback state + Queue (consumed by panes on every preset)
4. Stats page panes poll every tick regardless of Stats pane visibility — read internal event logs, not Spotify API

```go
for domain, paneID := range libraryDomains {
    if layout.IsPaneVisible(paneID) {
        if store.IsStale(domain) && !store.IsFetching(domain) {
            cmds = append(cmds, buildFetchCmd(domain))
        }
    }
}
```

**What Polls When:**

| Pane | Polls when | Interval | Endpoint |
|---|---|---|---|
| NowPlaying | Always | Adaptive (3-30s) | `GET /me/player` |
| Queue | Always | Adaptive (9-60s) | `GET /me/player/queue` |
| Playlists | Player, visible | Staleness 5m | `GET /me/playlists` |
| Albums | Player, visible | Staleness 5m | `GET /me/albums` |
| LikedSongs | Player, visible | Staleness 5m | `GET /me/tracks` |
| RecentlyPlayed | Player, visible | Staleness 2m | `GET /me/player/recently-played` |
| TopTracks | Player, visible | Staleness 10m | `GET /me/top/tracks` |
| TopArtists | Player, visible | Staleness 10m | `GET /me/top/artists` |
| FollowedShows | Player, visible | Staleness 5m | `GET /me/shows` |
| SavedEpisodes | Player, visible | Staleness 5m | `GET /me/episodes` |
| GatewayHealth | Stats page | Every tick | Internal event log |
| PollingTraffic | Stats page | Every tick | Internal store |
| GatewayLive | Stats page | Every tick | Internal event log |
| NetworkLog | Stats page | Every tick | Internal event log |

### Input Debounce (#318 seek, #269 volume)

Seek + volume use debounce via `components.DebounceTickMsg`. Intent snapshot captured locally, seq-numbered. Stale ticks discarded on seq mismatch. Only final position sent to Spotify after 300ms idle.

- `SeekDebounceTickMsg` — `nowplaying.go:245`
- `VolumeDebounceTickMsg` — `nowplaying.go:231`
- `HasPending()` preserves local progress during debounce (`nowplaying.go:640`)
- `SeekAppliedMsg` confirm/cancel (`nowplaying.go:251-255`)

### Search Debounce

300ms debounce via `tea.Tick`. Query change before fire → stale timer ignored. After 300ms idle, `SearchRequestMsg` dispatched with `api.Interactive` priority. Stale-response guard via `gen` counter (`search.go:503,599,631`).

---

## Configuration

TOML-based (`internal/config/`). All fields have defaults — empty/missing config OK. Default theme `black`. See `internal/config/config.go`.

Config keys: `spotify.client_id`, `spotify.callback_port`, `preferences.theme`, `preferences.preset`, `preferences.visualizer`, `cli.palette` (`auto`/`fixed`/`theme`), `ui.glyphs` (`auto`/`unicode`/`ascii`).

---

## Testing Architecture

### Mock Client

API interfaces mocked via `internal/api/apitest/mock.go`. No external mock libs.

### Pane Update Tests

See `internal/ui/panes/*_test.go` for update test patterns.

### Integration Test Convention

Multi-component: msg routing through root model, state propagation, end-to-end workflows with mocked HTTP.

**File naming:** `*_integration_test.go` (e.g. `app_integration_test.go`, `player_integration_test.go`)

**Build tag:** `//go:build integration`

**Running:**
- `make test` — unit tests only (default)
- `make test-integration` — integration only
- `make ci` — fmt-check → tidy-check → lint → test-coverage (80% min) → check-glyphs → build

**Integration test qualifies:**
- Msg routing through root `app.Model`
- State changes propagating across panes
- `httptest.NewServer` + multiple model updates in sequence
- Polling tick → downstream state changes

**Unit test stays:**
- Individual API methods with `httptest.NewServer`
- Store Get/Set
- `Update()` handlers (one key → one cmd)
- `View()` output assertions
- Config loading, PKCE helpers, time formatters

### Golden File Protocol

`internal/goldentest/golden.go` (154 lines). Snapshot tests capture pane `View()` output, committed in `testdata/`. Catch visual regressions: layout shifts, border breakage, padding, glyph misalignment.

| Helper | Signature | Location |
|---|---|---|
| `NewPaneTest` | `func NewPaneTest(t *testing.T, model tea.Model, width, height int) *teatest.TestModel` | `golden.go:42` |
| `AssertGolden` | `func AssertGolden(t *testing.T, got string)` | `golden.go:56` |
| `ReadOutput` | `func ReadOutput(tm *teatest.TestModel) (string, error)` | `golden.go:88` |
| `WaitAndReadOutput` | `func WaitAndReadOutput(t *testing.T, tm *teatest.TestModel) string` | `golden.go:96` |

Golden file path: `testdata/<TestName>.golden` (relative to test file). `-update` flag regenerates.

**Regeneration:**
```
go test ./... -update              # all golden files
make test-golden-ascii             # ASCII glyph mode (GOLDEN_MODE=ascii)
```

CI runs golden tests in `make ci` — mismatch fails build. Never change rendering output without regenerating golden files.

testdata inventory: `testdata/fixtures/` (21 API JSON fixtures), `internal/ui/panes/testdata/` (157 `*.golden` files at 80×24 + narrow).

---

## Notification System

User-facing notifications via `go.dalton.dog/bubbleup` toasts, wrapped by `uikit.ToastManager` (`app.go:75`). Rendered by `internal/ui/components.NewNotifications`. Auto-dismiss after configurable duration.

### Toast Alert Types

| Key | Theme Token | Prefix | Use |
|---|---|---|---|
| `"success"` | `Success()` | `✓` | Successful user actions (queue add, transfer) |
| `"error"` | `Error()` | `✗` | API errors, failures |
| `"warning"` | `Warning()` | `!` | Soft failures (Premium required) |
| `"info"` | `KeyHint()` | `→` | Informational (device transfer initiated) |
| `"ratelimit"` | `Warning()` | `⧖` | 429 rate-limit back-off |

### How to Emit

Return `a.alerts.NewAlertCmd(alertType, message)` from `Update()`:

```go
case SomeFailedMsg:
    if m.Err != nil {
        return a, a.alerts.NewAlertCmd("error", m.Err.Error())
    }
    return a, nil
```

### BubbleUp Integration

`AlertModel.View()` always returns `""`. Only `Render(content)` produces output. Toast activation = two Update passes: first returns `alertCmd`; executing `alertCmd` returns `alertMsg`; `alertMsg` to Update activates display.

---

## Error Handling Conventions

### build*Cmd Functions

API errors in `build*Cmd` MUST surface as toast. **Silent swallowing prohibited.**

**In build*Cmd (commands.go)** — never write Store:
```go
if err != nil {
    return XxxLoadedMsg{Err: err}
}
return XxxLoadedMsg{Data: data}
```

**In Update() (app.go)** — Store writes here:
```go
case XxxLoadedMsg:
    if m.Err != nil {
        a.store.SetXxxError(m.Err)
        return a, a.alerts.NewAlertCmd("error", m.Err.Error())
    }
    a.store.ClearXxxError()
    a.store.SetXxx(m.Data)
```

Store error fields preserved for retry logic (panes check to decide re-request on `f`/`Enter`) but **never read in `View()`** — toast's job.

### Pane Rendering Constraints

`View()` output MUST NOT exceed height from `SetSize()`. Panes with unbounded content (queue, library sections, search results) implement viewport scrolling. Loop rendering without height cap = bug.

### User-Facing Errors

| Error | Toast Type | Message |
|---|---|---|
| 401 (re-auth) | `"error"` | `Session expired. Run: spotnik auth` |
| 403 (no premium) | `"warning"` | `Spotify Premium required for playback` |
| 429 (rate limited) | `"ratelimit"` | `Too many requests. Retrying in Ns...` |
| 503 (Spotify down) | `"error"` | `Spotify is unavailable. Retrying...` |
| Network error | `"error"` | `No connection to Spotify` |

Toasts auto-dismiss after `notificationDuration` (`components/notifications.go`).

---

## Dependency Rules (Import Boundaries)

```
main.go
  └── cmd/
        └── internal/app/
              ├── internal/domain/      ← shared types (api/ ↔ ui/ bridge)
              ├── internal/state/       ← reads store via StateReader
              ├── internal/ui/panes/    ← render from store
              │     ├── internal/ui/layout/
              │     ├── internal/ui/components/
              │     ├── internal/ui/theme/
              │     └── internal/uikit/ ← design-system primitives
              ├── internal/api/         ← HTTP calls only
              ├── internal/config/      ← reads config
              ├── internal/keychain/    ← token storage
              ├── internal/prefs/       ← preference persistence
              └── internal/goldentest/ ← test framework

SHARED IMPORTS (allowed):
  internal/api/   → internal/domain/
  internal/ui/    → internal/domain/
  internal/state/ → internal/domain/
  internal/uikit/ → internal/ui/theme/

FORBIDDEN IMPORTS:
  internal/api/   → internal/ui/     (API must not know UI)
  internal/ui/    → internal/api/    (UI must not call API directly)
  internal/state/ → internal/ui/     (State must not know UI)
  internal/state/ → internal/api/    (State must not call API)
```

**Verified clean:** `grep -rn "internal/api" internal/ui/` = 0 production matches (2 test-only imports in `nowplaying_test.go:12`, `pipeline_test.go:10`). `api/ → ui/` = 0 violations. Store holds `domain.*` types only.

---

## Build & Release

`make build` → `bin/spotnik` single binary. Cross-compilation + flags in Makefile.

### CI Workflows (`.github/workflows/`)

| Workflow | Trigger | Permissions | Runs |
|---|---|---|---|
| `ci.yml` (CI) | push any branch, PR to main | `contents: read` (#418 least-priv) | `make ci` + `make check-glyphs`; matrix `LANG=en_US.UTF-8` + `LANG=C` |
| `release.yml` (Release) | push tag `v*`, workflow_dispatch | `contents: write`, `id-token: write`, `attestations: write` | goreleaser + Sigstore SLSA provenance attestation |
| `release-please.yml` | push to main, workflow_dispatch | `contents: write`, `pull-requests: write` | release-please automation v0.1.1+ |

### Makefile Targets

| Target | Command |
|---|---|
| `make build` | compile → `bin/spotnik` |
| `make run` | build + run |
| `make test` | `go test ./... -race -count=1` |
| `make test-integration` | `go test -tags integration ./... -race -count=1` |
| `make test-coverage` | coverage with 80% threshold |
| `make test-golden-ascii` | `GOLDEN_MODE=ascii go test ./internal/ui/panes/ -run "Golden\|golden" -update` |
| `make lint` | `golangci-lint run ./...` |
| `make check-glyphs` | banned-glyphs + catalogue-leaks + render-pane-border + lipgloss-borders checks |
| `make ci` | fmt-check → tidy-check → lint → test-coverage → check-glyphs → build |

---

## API Gateway

All outbound HTTP to Spotify routes through `internal/api/Gateway` (Feature 30). 4 services in order:

### 1. Token Bucket Rate Limiter

Token-bucket (10 tokens/sec, burst 10) limits total throughput. Background requests throttled through bucket. Interactive requests bypass bucket entirely.

**Adaptive rate reduction (#409):** on repeated 429s, bucket capacity auto-reduces. Recovers after `recoveryInterval`.

### 2. Concurrency Cap

Buffered channel size 5 = semaphore. Max 5 HTTP in-flight. 6th blocks until one completes or context cancelled.

### 3. In-Flight Request Deduplication

**Background-only.** Inflight map keyed by `RequestKey{Method, Path, Priority}`.

- **Background GET**: matching request in-flight → caller joins as waiter, receives copy of buffered response, no second HTTP call. Prevents tick-storm duplicates.
- **Interactive GET**: always fresh HTTP call — never joins in-flight. Prevents post-command reconcile joining stale pre-command Background poll.
- **PUT/POST/DELETE**: never deduped (non-idempotent).

### 4. 429 Backoff with Priority Bypass

On 429:
- Gateway sets `backoffUntil = now + Retry-After seconds`
- **Background** requests rejected immediately with `*RateLimitError`
- **Interactive** requests wait (blocking) for backoff expiry, then proceed

App receives `RateLimitedMsg`, updates `store.SetThrottle()`. `throttleExpiredMsg` fires after Retry-After to clear throttle state.

### Priority Context

Callers tag ctx with `api.WithPriority(ctx, api.Interactive)` for user actions. Background = default. `api.PriorityFromContext(ctx)` read in `BaseClient`.

**Interactive priority set in `internal/app/commands.go` for:**
play, pause, next, previous, volume, shuffle, repeat, search, add-to-queue, like/unlike, transfer playback, create/rename/remove/reorder playlist, fetch devices.

Post-command reconcile GET (`fetchPlaybackStateCmd`) also Interactive on success paths of `PlaybackCmdSentMsg` + `DeviceTransferredMsg` — ensures fresh, no join with stale pre-command Background poll.

### Gateway Live Pane (Stats page)

`GatewayLivePane` (`internal/ui/panes/gateway_live_pane.go:38`) reads gateway events from store's event journal via cursor-based replay. Pane never holds gateway reference — only reads `*state.Store`, preserves `ui/ → state/` direction.

- `PollingSnapshotMsg` — app-level polling diagnostics to GatewayLivePane
- `replayDisplayState` — single render model `View()` reads; updated by replay loop on each `viz.TickMsg`
- `eventCursor uint64` — cursor into `GatewayEventLog`; advanced by `drainEvents()` per tick
- `replayQueue []domain.GatewayEvent` — events waiting for 200ms min visibility
- `requestAnimation` — tracks one request's visual state across boxes
- `decisionEntry` — one line in GATEWAY box decision log

#### Replay Loop

On each `viz.TickMsg` (200ms):
1. `drainEvents()` — read new events from `store.ReadEventsFrom(cursor)`, append to `replayQueue`
2. `processNextEvent()` — pop one event, update `displayState.snapshot` + animation phases
3. `ageOutEntries()` — remove decisions > 3s, completed requests > 5s

#### Rendering

`GatewayLivePane.View()` uses boxed layout (Feature 62) when width ≥ 60:

```
╭─ APP ──────────╮           ╭─ GATEWAY ──────────╮           ╭─ SPOTIFY ──────╮
│ ▶ /player      │───────→───│ tokens  ●●●● 10/10 │───────→───│  200  45ms     │
│   /queue       │───→ dedup │ conc    □□□□□  0/5 │    ╳      │  200  62ms     │
│                │           │ ✓ GET /player allow │           │                │
╰────────────────╯           ╰────────────────────╯           ╰────────────────╯
POLLING  tick: 1000ms  state: active    STORE  fetching: []
```

- 3 sub-boxes: APP, GATEWAY, SPOTIFY — rounded corners
- Dual arrow columns: left (APP→GW decision), right (GW→SPOTIFY outcome)
- GATEWAY: state bars (token bucket + semaphore + backoff timer) + scrolling decision log
- Decision colors: `✓` allowed/expired → Success; `✗` blocked → Error; `⧖` dedup → Warning; resource → TextSecondary; `↻` refill → TextMuted
- `renderSubBox(title, lines, width)` — pure helper
- `formatDecisionLabel(e GatewayEvent)` — maps all 12 EventKind values
- Flat fallback (`viewFlat()`) when width < 60

#### Gateway Event Emission (Feature 67)

`Gateway.Do()` emits fine-grained lifecycle events at every decision via `emitEvent()`/`emitEventLocked()`:

| Event | When |
|---|---|
| `EventRequestEntered` | Entry to `Do()`, before any policy checks |
| `EventTokenConsumed` | After `bucket.wait()` returns for Background requests |
| `EventSemaphoreAcquired` | After acquiring concurrency slot |
| `EventSemaphoreReleased` | Deferred on slot release |
| `EventRequestBlocked` | Rejected by active backoff or ctx cancellation during bucket wait (both priorities) |
| `EventDedupJoined` | GET waiter joins in-flight dedup entry |
| `EventDedupResolved` | Dedup waiter received shared response |
| `EventHttpCompleted` | After `fn()` returns, with status + latency |
| `EventBackoffStarted` | After 429 response sets `backoffUntil` |
| `EventRequestAllowed` | Primary caller succeeded (no backoff wait) |
| `EventRequestFailed` | Primary caller passed gateway but HTTP returned error (429, 5xx, transport) |
| `EventTokenRefilled` | Periodic — bucket level changed since last emission |
| `EventBackoffExpired` | Periodic — active→cleared backoff transition |

12 EventKind constants (verified by `domain/gateway_test.go:31`).

**Lock ordering:** `emitEvent()` acquires `g.mu` to read recorder, releases `g.mu` before `captureSnapshot()` which acquires `bucket.mu` then `g.mu` (never both at once). `bucket.mu` never held when calling `emitEvent()`.

Periodic events on `viz.TickMsg` (200ms) from `app.go`:
- `CheckAndEmitRefill()` — emits `EventTokenRefilled` when bucket level changes (lazy, no mutation)
- `CheckAndEmitBackoffExpiry()` — emits `EventBackoffExpired` on active→cleared transition only

Each request gets unique `RequestID` from `nextRequestID atomic.Uint64`. All events for same request share ID. Internal events (TokenRefilled, BackoffExpired) use `RequestID = 0`.

#### Feature 68: Replay Engine (completed)

`GatewayState`, `GatewaySnapshotter`, `GatewayDecision` removed from `domain/gateway.go`. Deprecated `Snapshot()` shim + `ResetWatermarks()` no-op removed from `api/gateway.go`. Snapshot tests rewritten to event injection via `store.RecordEvent()` + `viz.TickMsg`.

#### Feature 69: Network Log Event Migration (completed)

`NetLog`, `NetLogEntry`, `RecordNetCall`, `RecordGatewayCall`, `NetLogEntries`, `LoggingTransport`, `NetLogRecorder` removed. NetworkLogPane reads directly from `GatewayEventLog` via cursor-based `ReadEventsFrom()`. Blocked requests visible (status 0, "✗ blocked"). PRIORITY + DECISION columns added.

### Network Logging

All HTTP logged to `GatewayEventLog` for NetworkLogPane. `GatewayEventLog` = single authoritative source for both NetworkLogPane + GatewayLivePane.

- Data flow: `BaseClient.doJSON` → `Gateway.Do()` → `store.RecordEvent()` → `GatewayEventLog`
- NetworkLogPane drains via `store.ReadEventsFrom(cursor)` on each 1s tick
- Columns: TIME, METHOD, ENDPOINT, STATUS, LATENCY, PRI (int/bg), DECISION (allowed/blocked/dedup), NOTES
- Blocked (`EventRequestBlocked`) appear with status 0, "✗ blocked" in NOTES

### Gateway Event Journal

Timestamped event stream. Replaced snapshot-polling + old `NetLog` ring buffer.

- `internal/domain/gateway.go` — `EventKind` (12 constants), `GatewayStateSnapshot`, `GatewayEvent`, `GatewayEventRecorder`
- `internal/state/eventlog.go` — `GatewayEventLog`: 500-entry thread-safe ring buffer with cursor reads
  - `Add(event)` — write path; called by `Store.RecordEvent()`
  - `ReadFrom(cursor)` — events since cursor; multiple independent consumers (GatewayLivePane, NetworkLogPane) each hold own cursor
- `internal/state/store.go` — `RecordEvent()` implements `domain.GatewayEventRecorder`; `ReadEventsFrom()` exposes cursor reads to UI

### Integration Points

- `internal/api/gateway.go` — Gateway struct, Priority, emitEvent helpers
- `internal/api/gateway_bucket.go` — tokenBucket
- `internal/api/gateway_dedup.go` — inflightEntry, dedup map
- `internal/api/base.go` — `BaseClient.SetGateway()`, `doJSON`/`doNoContent` routing
- `internal/app/app.go` — Gateway created in `New()`, `throttleExpiredMsg` handler
- `internal/app/auth.go` — `initAPIClients()` creates plain `http.Client`, calls `SetGateway()` + `SetRecorder(store)`
- `internal/state/store.go` — `SetThrottle()`, `IsThrottled()`, `ThrottleRetryAfterSecs()`, `RecordEvent()`, `ReadEventsFrom()`
- `internal/state/eventlog.go` — `GatewayEventLog` ring buffer
- `internal/domain/gateway.go` — `RequestPriority`, `EventKind`, `GatewayStateSnapshot`, `GatewayEvent`, `GatewayEventRecorder`

---

*Last updated: 2026-07-18*
