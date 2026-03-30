---
title: "API Gateway & Data Architecture"
description: "Centralizes all outbound Spotify HTTP traffic through a prioritized, rate-limited gateway while enforcing Elm Architecture purity, toast-based error notifications, TTL-based cache staleness, and adaptive idle polling to deliver a resilient, responsive playback experience."
status: done
stories: [29, 30, 31, 32, 33]
---

# API Gateway & Data Architecture

## Background

Spotnik's interaction with the Spotify API spans playback polling, library browsing, search, queue management, device switching, and stats dashboards. As these features accumulated, several architectural gaps emerged: command functions mutated the Store directly inside goroutine closures (violating the Elm Architecture contract), HTTP requests fired with no throttling or deduplication, error feedback was limited to a single status string with no severity distinction, cached data had no concept of freshness, and polling ran at full speed regardless of user activity or playback state.

This feature consolidates five coordinated efforts that together establish a robust data architecture. First, all data-fetching commands were refactored to carry their results in typed message payloads, restoring the Elm purity guarantee that only `Update()` may write to the Store. Second, a centralized API Gateway was introduced to control all outbound HTTP traffic with token-bucket rate limiting, concurrency capping, request deduplication, priority classification, and 429 backoff. Third, the primitive `statusMsg` string was replaced with a BubbleUp-based toast notification system that routes all API errors through severity-typed overlays. Fourth, TTL-based staleness tracking was added to the Store so that `Update()` can make informed decisions about when to re-fetch versus reuse cached data. Fifth, an adaptive polling system was built to reduce API traffic when the user is idle or playback is paused, resuming full-speed polling on interaction.

Together these five stories form the backbone of Spotnik's data flow: commands produce data, the gateway controls how it reaches Spotify, errors surface through toasts, freshness is tracked with timestamps, and polling adapts to actual usage patterns. The gateway is the reactive rate-management layer; idle backoff is the proactive layer. They are independent but complementary.

---

## Story: Elm Purity — Data-Carrying Messages (spec 29)

### Background

Nine `build*Cmd` / `fetch*Cmd` functions in `internal/app/commands.go` mutated the Store directly inside `tea.Cmd` closures (goroutines). Their Msg types were empty notification structs (e.g., `QueueLoadedMsg{}`) that signaled panes to re-read from the Store. This violated the Elm Architecture principle: only `Update()` may mutate state. This story refactored every violating command to return data in typed Msg payloads and moved all Store writes into `Update()` handlers.

Already compliant commands (no changes needed): `buildAddToQueueCmd` (line 251), `buildPlayContextCmd` (line 104), `buildPlayTrackCmd` (line 124), `buildToggleLikeCmd` (line 407), `buildCreatePlaylistCmd` (line 582), `buildRenamePlaylistCmd` (line 599), `buildRemovePlaylistTrackCmd` (line 612), `buildReorderPlaylistTracksCmd` (line 625). These already returned data-carrying Msgs without Store writes.

Gap reference: G1 in `docs/superpowers/plans/2026-03-24-architecture-baseline.md`

### Acceptance Criteria
- [ ] Zero `store.Set*` or `store.Clear*` calls remain in `internal/app/commands.go`
- [ ] All Msg types carry `Data` + `Err error` fields
- [ ] All Store writes happen exclusively in `Update()` handlers
- [ ] `ui/` never imports `api/` — domain types live in `internal/domain/`
- [ ] `make ci` passes — lint, tests, 80% coverage

### Tasks

1. **Extract domain types into `internal/domain/`** — Move shared domain types out of `internal/api/models.go` into a new `internal/domain/types.go` package to break the `ui/ → api/` import dependency, allowing Msg types to carry API data payloads.
   - Files: Create `internal/domain/types.go`; modify `internal/api/models.go` (re-export domain types), `internal/state/store.go`, `internal/ui/panes/messages.go`, `internal/app/commands.go`, and all files referencing `api.Track`, `api.PlaybackState`, etc.
   - Types moved: `PlaybackState`, `Track`, `Artist`, `Album`, `SimplePlaylist`, `SimplePlaylistOwner`, `FullAlbum`, `SavedAlbum`, `SavedTrack`, `PlayHistory`, `FullArtist`, `SearchResult` (+ inner list types), `QueueResponse`, `Device`, `PlayOptions`
   - Tests: `make ci` must pass — pure refactor, no behavior change
   - Commit: `refactor(domain): extract shared types into internal/domain package`

2. **Data-carrying PlaybackStateFetchedMsg + QueueLoadedMsg** — Refactor `fetchPlaybackStateCmd` (commands.go line 491) and `fetchQueueCmd` (line 465) to return data in Msg payloads instead of writing to Store in goroutine closures.
   - Files: `internal/ui/panes/messages.go` (add payload fields), `internal/app/commands.go` (remove Store writes), `internal/app/app.go` (add Store writes in Update() handlers)
   - Msg changes:
     ```go
     type PlaybackStateFetchedMsg struct {
         State *domain.PlaybackState
         Err   error
     }
     type QueueLoadedMsg struct {
         Tracks []domain.Track
         Err    error
     }
     ```
   - Update() handler for `PlaybackStateFetchedMsg`: on non-nil `State` call `a.store.SetPlaybackState(m.State)`; on non-nil `Err` log/ignore transient polling error
   - Update() handler for `QueueLoadedMsg`: on nil `Err` call `a.store.ClearQueueError()` then `a.store.SetQueue(m.Tracks)`; on non-nil `Err` call `a.store.SetQueueError(m.Err)`
   - Tests: Unit — `fetchPlaybackStateCmd` returns `PlaybackStateFetchedMsg{State: ...}` with no Store writes; `fetchQueueCmd` returns `QueueLoadedMsg{Tracks: ...}` with no Store writes; `Update(PlaybackStateFetchedMsg{State: ps})` writes to Store; `Update(QueueLoadedMsg{Tracks: tracks})` writes to Store
   - Commit: `refactor(arch): data-carrying PlaybackStateFetchedMsg + QueueLoadedMsg`

3. **Data-carrying library messages** — Refactor four library command builders that write to Store inside goroutine closures: `buildFetchPlaylistsCmd` (line 144), `buildFetchAlbumsCmd` (line 174), `buildFetchLikedTracksCmd` (line 199), `buildFetchRecentlyPlayedCmd` (line 225).
   - Files: `internal/ui/panes/messages.go` (add payload fields to 4 Msgs), `internal/app/commands.go` (remove Store writes from 4 closures), `internal/app/app.go` or `routing.go` (add Store writes in Update() handlers)
   - Msg changes:
     ```go
     type LibraryLoadedMsg struct {
         Items  []domain.SimplePlaylist
         Offset int
         Err    error
     }
     type AlbumsLoadedMsg struct {
         Items []domain.SavedAlbum
         Err   error
     }
     type LikedTracksLoadedMsg struct {
         Items  []domain.SavedTrack
         Offset int
         Err    error
     }
     type RecentlyPlayedLoadedMsg struct {
         Items []domain.PlayHistory
         Err   error
     }
     ```
   - Key detail — pagination: `buildFetchPlaylistsCmd` (line 163-168) currently does `if offset == 0 { store.SetPlaylists(playlists) } else { store.SetPlaylists(append(store.Playlists(), playlists...)) }`. This logic moves to `Update()`. The Msg carries the raw page of items + the offset so Update() can decide whether to replace or append.
   - Tests: Unit — each `build*Cmd` returns data in Msg, does not write Store; `Update()` handles pagination append for LibraryLoadedMsg; `Update()` handles error set/clear for each domain
   - Commit: `refactor(arch): data-carrying library messages`

4. **Data-carrying Stats, Search, Devices, PlaylistTracks** — Refactor remaining violating commands: `buildFetchStatsCmd` (line 426), `buildSearchCmd` (line 272), `buildFetchDevicesCmd` (line 359), `buildFetchPlaylistTracksCmd` (line 556).
   - Files: `internal/ui/panes/messages.go` (update Msg fields), `internal/app/commands.go` (remove Store writes from 4 closures), `internal/app/app.go` (add Store writes in Update() handlers), `internal/app/routing.go` (add Store writes for playlist tracks)
   - Msg changes:
     ```go
     type StatsLoadedMsg struct {
         TimeRange  string
         TopTracks  []domain.Track
         TopArtists []domain.FullArtist
         Err        error
     }
     type PlaylistTracksLoadedMsg struct {
         PlaylistID string
         Tracks     []domain.Track
         Err        error
     }
     ```
   - `SearchResultsMsg` already has `Results` + `Err` — just remove Store writes from the Cmd
   - `DevicesLoadedMsg` already carries `[]DeviceInfo` — move error write to Update()
   - For `buildSearchCmd` specifically: move `store.SetSearchQuery(query)` and `store.SetSearchLoading(true)` OUT of the Cmd builder and into the caller site in `Update()` (before dispatching the Cmd). Remove ALL `store.Set*` / `store.Clear*` from the closure. Store writes for results/loading/error happen in the `SearchResultsMsg` handler in `Update()`
   - Tests: Unit — each command returns data in Msg, does not write Store; `Update()` correctly writes Store for each Msg type; `buildSearchCmd` no longer calls any `store.Set*` methods
   - Commit: `refactor(arch): data-carrying stats/search/devices/playlist messages`

5. **Remove Store parameter from package-level functions** — `fetchPlaybackStateCmd` and `fetchQueueCmd` are the only package-level functions (not methods on `*App`) that take a `store *state.Store` parameter. After Tasks 1-3, they no longer need it.
   - Files: `internal/app/commands.go` (remove store param, add WaitGroup for stats), `internal/app/app.go` (update call sites)
   - Signature changes: `fetchPlaybackStateCmd(player api.PlayerAPI, store *state.Store) tea.Cmd` → `fetchPlaybackStateCmd(player api.PlayerAPI) tea.Cmd`; `fetchQueueCmd(player api.PlayerAPI, store *state.Store) tea.Cmd` → `fetchQueueCmd(player api.PlayerAPI) tea.Cmd`
   - Bonus — Parallelize stats fetches: refactor `buildFetchStatsCmd` to fetch top tracks and top artists concurrently using `sync.WaitGroup` (stdlib only — no `errgroup` per CLAUDE.md)
   - Tests: Unit — both functions compile without store param; stats Cmd returns both tracks and artists (verify parallel execution with timing)
   - Commit 1: `refactor(arch): remove store param from package-level command functions`
   - Commit 2: `perf(stats): parallelize top tracks and top artists fetches`

6. **Documentation updates** — Document the data-carrying Msg pattern in architecture docs and add the Store mutation rule to CLAUDE.md.
   - Files: `docs/ARCHITECTURE.md` (Section "Message Flow" — before/after examples, rule that `build*Cmd` functions must NEVER write to Store), `CLAUDE.md` (Section "Architecture Rules" — add rule: "Commands must not mutate the Store — return data in Msg payloads; only Update() writes to Store")
   - Commit: `docs: update architecture docs for data-carrying message pattern`

**Verification:**
```bash
grep -r 'store\.Set\|store\.Clear' internal/app/commands.go
# Expected: ZERO matches — no Store writes remain in command builders.
# Note: Store *reads* (e.g., store.PlaybackState() in buildPlaybackAPICmd) are legitimate.
make ci
# Expected: Full pass — lint, tests, 80% coverage.
```

---

## Story: API Gateway (spec 30)

### Background

All API requests fired directly to Spotify through `BaseClient.doJSON` / `doNoContent` (internal/api/base.go lines 73-107) with no throttling, dedup, concurrency cap, or priority. A burst of user actions plus polling could trigger rate limiting. There was no single control point for all HTTP traffic. This story introduced a centralized API gateway that controls all outbound HTTP traffic with token-bucket rate limiting, concurrency capping, in-flight request deduplication, priority classification (Interactive vs Background), and 429 backoff.

Gap reference: G2, G8, G9 in `docs/superpowers/plans/2026-03-24-architecture-baseline.md`

### Acceptance Criteria
- [ ] All API calls route through the gateway (or bypass only when gateway is nil for backwards compat)
- [ ] Token bucket limits requests to 10/second with burst of 10
- [ ] Max 5 concurrent in-flight requests
- [ ] Duplicate in-flight requests are deduplicated (same method+path → one HTTP call)
- [ ] 429 responses trigger global backoff; background requests rejected, interactive requests wait
- [ ] Interactive requests bypass token bucket
- [ ] `IsThrottled()` and `RetryAfterSecs()` expose gateway state for UI observability
- [ ] `make ci` passes

### Tasks

1. **Token bucket rate limiter** — Create `internal/api/gateway.go` with a `tokenBucket` struct implementing `wait(ctx context.Context) error`. Default: 10 tokens/second, burst of 10. `wait()` refills tokens based on elapsed time, then either returns immediately (tokens available) or blocks until a token is available or ctx is cancelled.
   - Files: Create `internal/api/gateway.go`, create `internal/api/gateway_test.go`
   - Key types:
     ```go
     type tokenBucket struct {
         mu       sync.Mutex
         tokens   float64
         max      float64
         rate     float64     // tokens per second
         lastFill time.Time
     }
     ```
   - Tests: Unit — token bucket allows burst up to max; blocks when empty, unblocks after refill interval; respects context cancellation
   - Commit: `feat(api): token bucket rate limiter for gateway`

2. **Concurrency limiter + Gateway struct** — Add `Gateway` struct with semaphore-based concurrency limiting (buffered channel of size 5) and `Do()` method.
   - Files: `internal/api/gateway.go`, `internal/api/gateway_test.go`
   - Key types:
     ```go
     type Gateway struct {
         mu           sync.Mutex
         bucket       *tokenBucket
         semaphore    chan struct{}     // concurrency limiter (buffered channel, size 5)
         inflight     map[RequestKey]*inflightEntry
         backoffUntil time.Time
         retryAfter   int
     }
     type RequestKey struct {
         Method string
         Path   string
     }
     func NewGateway() *Gateway
     func (g *Gateway) Do(ctx context.Context, priority Priority, key RequestKey,
         fn func() (*http.Response, error)) (*http.Response, error)
     ```
   - Tests: Unit — max 5 concurrent requests; 6th blocks until one completes; semaphore respects context cancellation
   - Commit: `feat(api): concurrency limiter and Gateway struct`

3. **In-flight request dedup** — Before executing `fn` in `Do()`, check if `key` exists in `inflight` map. If yes, wait on its `done` channel and return the cached result (clone the response body). If no, add entry, execute `fn`, buffer the response body, broadcast to waiters, clean up.
   - Files: `internal/api/gateway.go`, `internal/api/gateway_test.go`
   - Key types:
     ```go
     type inflightEntry struct {
         done chan struct{}
         resp *http.Response
         body []byte
         err  error
     }
     ```
   - Response body buffering: read body into `[]byte`, store in entry, create new `io.ReadCloser` for each waiter from the buffer
   - Tests: Unit — two concurrent requests with same key → only one HTTP call, both get result; different keys execute independently; error result is shared with waiters too
   - Commit: `feat(api): in-flight request dedup in gateway`

4. **429 backoff + priority bypass** — Add `Priority` type and implement backoff/priority logic in `Do()`.
   - Files: `internal/api/gateway.go`, `internal/api/gateway_test.go`
   - Key types:
     ```go
     type Priority int
     const (
         Background  Priority = iota // polling, prefetch
         Interactive                  // user-initiated actions
     )
     ```
   - In `Do()`: check `backoffUntil` — if `time.Now().Before(g.backoffUntil)`, block background requests (return `RateLimitError`); interactive requests wait until backoff expires. On 429 response: parse `Retry-After`, set `g.backoffUntil`, return error. Interactive requests skip the token bucket wait. Background requests go through normal token bucket flow.
   - Expose `IsThrottled() bool` and `RetryAfterSecs() int` for UI observability
   - Tests: Unit — after 429, background requests rejected until backoff expires; interactive requests wait during backoff but eventually proceed; interactive requests bypass token bucket; `IsThrottled()` returns correct state
   - Commit: `feat(api): 429 backoff and priority bypass in gateway`

5. **Integration into BaseClient + Store + docs** — Wire the gateway into all existing API infrastructure.
   - Files: `internal/api/base.go` (add optional `gateway *Gateway` field, route `doJSON`/`doNoContent` through `gateway.Do()`), `internal/api/gateway.go` (add priority context, observability), `internal/state/store.go` (add throttle fields: `IsThrottled bool`, `RetryAfterSecs int`, `Last429At time.Time`), `internal/app/app.go` (create `Gateway` in `New()`, pass to `BaseClient` constructors, remove duplicate 429/backoff handling), `internal/app/commands.go` (set priority on contexts), `docs/ARCHITECTURE.md` (new section "API Gateway"), `CLAUDE.md` (add Gateway rule to "API Rules")
   - Priority passing: use `context.WithValue` with a package-private key to carry `Priority`. Command builders set `Interactive` for user-triggered actions, `Background` for polling.
   - `NewBaseClientWithProvider` accepts optional `Gateway`. Construct `RequestKey` from request method + path. Default priority: `Background`.
   - Tests: Integration — all API calls go through gateway (verify with httptest server + request counting); Unit — BaseClient with gateway routes through Do(); BaseClient without gateway works as before (backwards compat)
   - Commit 1: `feat(api): integrate gateway into BaseClient and all API calls`
   - Commit 2: `docs: add API Gateway documentation`

**Verification:**
```bash
# All API calls should go through the gateway
grep -r 'b.http.Do(' internal/api/base.go
# Expected: only inside gateway.Do's fn callback or when gateway is nil
make ci
# Expected: Full pass
```

---

## Story: Notifications + Error Routing (spec 31)

### Background

The notification system was a single `statusMsg` string field on the App struct (app.go line 93-94) rendered in error-red color (`theme.Error()`) for both success and error messages (render.go lines 194-210). There was no visual distinction between severity levels. The status bar conflated keybinding hints with transient feedback. After Feature 29 established data-carrying messages with `Err` fields, this story replaced the primitive status string with a BubbleUp-based toast notification system and routed all API errors through severity-typed toast overlays instead of inline pane error rendering.

Approved dependency: BubbleUp (`go.dalton.dog/bubbleup`) — approved by owner on 2026-03-24. MIT-licensed, depends only on bubbletea + lipgloss (already in deps).

Gap reference: G3, G5, G10 in `docs/superpowers/plans/2026-03-24-architecture-baseline.md`

### Acceptance Criteria
- [ ] `statusMsg` field and `statusDismissMsg` type completely removed from codebase
- [ ] All 16 former `statusMsg` sites emit typed toast commands via `a.alerts.NewAlertCmd()`
- [ ] Five alert types registered: `success`, `error`, `warning`, `info`, `ratelimit`
- [ ] Toast overlays render via `alerts.Render(content)` as the final step of `View()`
- [ ] Status bar always shows keybinding hints (no error override)
- [ ] All pane `View()` methods remove inline error rendering — errors route through toast only
- [ ] `make ci` passes

### Tasks

1. **BubbleUp dependency + notification wrapper** — Add `go.dalton.dog/bubbleup` and create a themed notification wrapper.
   - Files: Modify `go.mod`; create `internal/ui/components/notifications.go`, create `internal/ui/components/notifications_test.go`
   - `NewNotifications(t theme.Theme) bubbleup.AlertModel` — creates AlertModel with 5 custom alert types:
     ```go
     successAlert := bubbleup.AlertDefinition{Key: "success", ForeColor: string(theme.Success()), Prefix: "✓"}
     errorAlert := bubbleup.AlertDefinition{Key: "error", ForeColor: string(theme.Error()), Prefix: "✗"}
     warningAlert := bubbleup.AlertDefinition{Key: "warning", ForeColor: string(theme.Warning()), Prefix: "!"}
     infoAlert := bubbleup.AlertDefinition{Key: "info", ForeColor: string(theme.KeyHint()), Prefix: "→"}
     rateLimitAlert := bubbleup.AlertDefinition{Key: "ratelimit", ForeColor: string(theme.Warning()), Prefix: "⧖"}
     ```
   - Important: BubbleUp's `View()` method is intentionally empty. Use `Render(content)` not `View()`.
   - Tests: Unit — `NewNotifications` returns a valid AlertModel with all 5 custom types registered; verify `ForeColor` conversion compiles and produces expected hex strings
   - Commit: `feat(ui): add BubbleUp notification wrapper`

2. **Integrate BubbleUp into root App** — Add `alerts bubbleup.AlertModel` field to App struct and wire into Init/Update/View lifecycle.
   - Files: `internal/app/app.go` (add alerts field, Init/Update wiring), `internal/app/render.go` (call `Render()` as final overlay step)
   - In `New()`: initialize with `components.NewNotifications(t)`
   - In `Init()`: batch `a.alerts.Init()` with existing commands
   - In `Update()`: for every message, also pass to `a.alerts.Update(msg)` and batch any returned command (necessary for BubbleUp's internal timer management / auto-dismiss)
   - In `View()` (render.go): as the final step, call `a.alerts.Render(existingView)` instead of returning `existingView` directly. Critical: Do NOT call `a.alerts.View()` — it returns empty string by design.
   - Tests: Unit — App Init() includes alerts Init command; App Update() forwards messages to alerts model; App View() calls alerts.Render() on the final output
   - Commit: `feat(ui): integrate BubbleUp into root App model`

3. **Replace statusMsg with toast commands** — Replace all 16 `a.statusMsg = "..."` assignment sites with `a.alerts.NewAlertCmd()` calls, remove the `statusMsg` field and `statusDismissMsg` type.
   - Files: `internal/app/app.go` (replace all statusMsg sites, remove field + dismiss type), `internal/app/routing.go` (replace all statusMsg sites), `internal/app/render.go` (simplify renderStatusBar()), `internal/app/render_test.go` (update tests)
   - Current statusMsg sites (all in `internal/app/`):

     | File | Line | Current Message | New Alert Type |
     |---|---|---|---|
     | app.go | 530 | `"Rate limited — pausing requests for %ds"` | `ratelimit` |
     | app.go | 541 | `"Session expired. Run: spotnik auth"` | `error` |
     | app.go | 566 | `"Playback control not available on this device"` | `warning` |
     | app.go | 568 | `"✗ %s"` (playback error) | `error` |
     | app.go | 601 | `"✗ %s"` (forbidden) | `error` |
     | app.go | 603 | `"✗ %s"` (add-to-queue error) | `error` |
     | app.go | 608 | `"✓ Added to queue: %s"` | `success` |
     | app.go | 610 | `"✓ Added to queue"` | `success` |
     | app.go | 632 | `"✗ %s"` (like toggle error) | `error` |
     | app.go | 647 | `"Switching to %s..."` | `info` |
     | app.go | 656 | `"✗ %s"` (device transfer error) | `error` |
     | app.go | 666 | `""` (statusDismissMsg clears) | N/A — removed |
     | routing.go | 218 | `"✗ %s"` (playlist created error) | `error` |
     | routing.go | 229 | `"✗ %s"` (playlist renamed error) | `error` |
     | routing.go | 251 | `"✗ %s"` (playlist remove error) | `error` |
     | routing.go | 268 | `"✗ %s"` (playlist reorder error) | `error` |

   - `renderStatusBar()` (render.go lines 194-210): remove the `if a.statusMsg != ""` branch. Status bar now ALWAYS shows keybinding hints. Toast notifications appear as overlays via `alerts.Render()`, not in the status bar.
   - Tests: Unit — verify each error/success case emits the correct alert type; `renderStatusBar()` always returns hints (no error override); Grep verification: `grep -r 'statusMsg' internal/app/` → ZERO matches
   - Commit: `refactor(ui): replace statusMsg with BubbleUp toasts`

4. **Route all API errors through toast** — Remove inline error rendering from pane `View()` methods; emit toasts for all data-carrying Msgs with non-nil `Err` fields.
   - Files: `internal/app/app.go` (add toast commands for all error Msgs), `internal/app/routing.go` (add toast commands for playlist error Msgs), `internal/ui/panes/search.go` (remove inline error rendering), `internal/ui/panes/playlists.go` (remove inline error rendering), `internal/ui/panes/stats.go` (remove inline error rendering), `internal/ui/panes/devices.go` (remove inline error rendering), `docs/ARCHITECTURE.md` (new section "Notification System" — BubbleUp integration, alert types, severity mapping, error routing; update "Error Handling Conventions"), `docs/DESIGN.md` (update "Status Bar" section — remove "Error mode"; new section "Toast Notifications" — position, severity colors, dismiss behavior), `CLAUDE.md` (add error routing rule to "Architecture Rules")
   - Examples: `PlaybackStateFetchedMsg{Err: err}` → toast "Playback update failed"; `LibraryLoadedMsg{Err: err}` → toast "Failed to load playlists. Press Tab to retry"
   - Store error fields remain for retry logic only — never read in `View()`
   - Tests: Unit — each error Msg triggers a toast command; no Store error fields are read in any pane View() method; Grep verification: `grep -r 'Error()' internal/ui/panes/*` should show no Store error reads
   - Commit 1: `feat(ui): route all API errors through toast notifications`
   - Commit 2: `docs: add notification system and error routing documentation`

**Notification Mapping Reference:**

| Current `statusMsg` Usage | New BubbleUp Alert Key | Message |
|---|---|---|
| `"✓ Added to queue: ..."` | `success` | "Added to queue: ..." |
| `"✗ <error>"` | `error` | "\<error\>" |
| `"Playback control not available..."` | `warning` | "Playback control not available..." |
| `"Rate limited — pausing requests for Ns"` | `ratelimit` | "Rate limited, retrying in Ns" |
| `"Session expired. Run: spotnik auth"` | `error` | "Session expired. Run: spotnik auth" |
| `"Switching to <device>..."` | `info` | "Switching to \<device\>..." |

**Verification:**
```bash
# statusMsg completely removed
grep -r 'statusMsg' internal/app/
# Expected: ZERO matches

# All errors trigger toast
grep -r 'statusDismissMsg' internal/app/
# Expected: ZERO matches

make ci
# Expected: Full pass
```

---

## Story: Staleness Tracking (spec 32)

### Background

The Store had no concept of data age. Library data (albums, liked tracks) was fetched once per session and never refreshed. Stats data was cached per time-range forever. Boolean sentinels like `albumsLoaded` (store.go line ~25) and `likedLoaded` (line ~28) tracked whether data had been fetched at all, but didn't support re-fetching after a TTL. After Feature 29 established that `Update()` owns all Store writes, this story added `fetchedAt` timestamps and TTL-based staleness checks so `Update()` can make informed decisions about when to re-fetch versus reuse cached data.

Gap reference: G4 in `docs/superpowers/plans/2026-03-24-architecture-baseline.md`

### Acceptance Criteria
- [ ] Every data domain has a `fetchedAt` timestamp set on successful write
- [ ] `IsStale(fetchedAt, ttl)` helper returns true for zero time or elapsed > TTL
- [ ] Boolean sentinels `albumsLoaded` and `likedLoaded` are removed
- [ ] Convenience `*Stale()` methods exist for all domains
- [ ] Library/stats data re-fetches after TTL expires; uses cached data within TTL
- [ ] `make ci` passes

### Staleness TTLs

| Domain | TTL | Rationale |
|---|---|---|
| Playback state | N/A | Always polled, overwritten each tick cycle |
| Queue | N/A | Always polled, overwritten each tick cycle |
| Playlists list | 5 min | Changes infrequently |
| Albums | 5 min | Changes infrequently |
| Liked tracks | 5 min | Changes infrequently |
| Recently played | 2 min | Changes with playback |
| Stats (per range) | 10 min | Spotify updates these slowly |
| Devices | 5 sec | Volatile — short cooldown; user-initiated fetches use Interactive priority |

### Tasks

1. **Add fetchedAt fields + IsStale() helper** — Add timestamp fields to Store for each data domain, update all `Set*` methods to stamp `fetchedAt = time.Now()`, and provide accessor methods.
   - Files: Modify `internal/state/store.go` (add fields, update Set methods, add accessors); create or modify `internal/state/store_test.go`
   - Fields added:
     ```go
     playlistsFetchedAt    time.Time
     albumsFetchedAt       time.Time
     likedTracksFetchedAt  time.Time
     recentPlayedFetchedAt time.Time
     statsFetchedAt        map[string]time.Time // keyed by time range
     devicesFetchedAt      time.Time
     ```
   - Helper:
     ```go
     func IsStale(fetchedAt time.Time, ttl time.Duration) bool {
         return fetchedAt.IsZero() || time.Since(fetchedAt) > ttl
     }
     ```
   - Accessors: `PlaylistsFetchedAt()`, `AlbumsFetchedAt()`, `LikedTracksFetchedAt()`, `RecentPlayedFetchedAt()`, `StatsFetchedAt(timeRange string)`, `DevicesFetchedAt()`
   - Tests: Unit — `IsStale` returns true for zero time; returns true when elapsed > TTL; returns false when elapsed < TTL; `SetPlaylists()` updates `playlistsFetchedAt`; `SetTopTracks()` updates `statsFetchedAt[range]`
   - Commit: `feat(state): add fetchedAt timestamps and IsStale helper`

2. **TTL constants + replace boolean sentinels** — Add TTL constants, convenience `*Stale()` methods, and remove `albumsLoaded`/`likedLoaded` boolean fields.
   - Files: Modify `internal/state/store.go` (add constants, convenience methods, remove booleans); modify `internal/ui/panes/library.go` (update any `albumsLoaded`/`likedLoaded` reads)
   - Constants:
     ```go
     const (
         PlaylistsTTL      = 5 * time.Minute
         AlbumsTTL         = 5 * time.Minute
         LikedTracksTTL    = 5 * time.Minute
         RecentlyPlayedTTL = 2 * time.Minute
         StatsTTL          = 10 * time.Minute
         DevicesTTL        = 5 * time.Second
     )
     ```
   - Convenience methods: `PlaylistsStale() bool`, `AlbumsStale() bool`, `LikedTracksStale() bool`, `RecentPlayedStale() bool`, `StatsStale(timeRange string) bool`, `DevicesStale() bool` — each acquires RLock and delegates to `IsStale()`
   - Where `albumsLoaded` was checked → use `!s.AlbumsStale()` or check `albumsFetchedAt.IsZero()`. Same for `likedLoaded`.
   - Tests: Unit — `PlaylistsStale()` returns true when never fetched; returns true after TTL expires; returns false within TTL; removing boolean sentinels doesn't break existing tests
   - Commit: `refactor(state): replace boolean sentinels with TTL-based staleness`

3. **Wire staleness checks into Update() + docs** — Gate library and stats fetch commands with staleness checks so cached data is reused within TTL.
   - Files: `internal/app/app.go` (add staleness checks before library/stats fetches), `internal/app/routing.go` (add staleness checks before playlist fetches), `internal/ui/panes/library.go` (update Init/navigation to emit fetch requests that `Update()` can gate), `docs/ARCHITECTURE.md` (add staleness tracking documentation to "State Management" → "The Store"; note in "Polling Architecture" that library/stats use staleness-based refresh, not polling)
   - Before dispatching `buildFetchPlaylistsCmd`, check `a.store.PlaylistsStale()`; before dispatching `buildFetchAlbumsCmd`, check `a.store.AlbumsStale()`; same for liked tracks, recently played
   - In stats view: when user switches to stats view or changes time range, check `a.store.StatsStale(timeRange)` before dispatching `buildFetchStatsCmd`
   - In library pane navigation (Init or section switch): currently `library.go` line ~350 dispatches fetches unconditionally on Init; after this change, `Update()` checks staleness before dispatching
   - Tests: Unit — library data re-fetches after TTL expires; library data NOT re-fetched within TTL; stats re-fetch on stale re-open; stats NOT re-fetched within TTL; Integration — switching away from stats and back within TTL uses cached data
   - Commit 1: `feat(app): staleness-gated data fetching`
   - Commit 2: `docs: add staleness tracking to architecture docs`

**Verification:**
```bash
# Boolean sentinels removed
grep -r 'albumsLoaded\|likedLoaded' internal/state/store.go
# Expected: ZERO matches

# Staleness checks exist before fetches
grep -r 'Stale()' internal/app/
# Expected: multiple matches for library/stats domains

make ci
# Expected: Full pass
```

---

## Story: Idle Polling Backoff (spec 33)

### Background

The tick loop polled at full speed (3s playback, 9s queue) regardless of user activity or playback state (app.go lines 40-47, tick handler at lines 483-521). When music was paused or the user was on the stats/playlists view, polling wasted bandwidth and risked rate limiting. This story is the proactive layer (Layer 1) of rate management — it reduces the *number* of requests entering the gateway. Feature 30 (API Gateway) is the reactive layer (Layer 2) — it limits the *rate* of requests that pass through. They are independent but complementary.

Gap reference: G6 in `docs/superpowers/plans/2026-03-24-architecture-baseline.md`

### Acceptance Criteria
- [ ] Polling intervals adapt based on a 4-state matrix (active/idle x playing/paused)
- [ ] User idle state detected after 60 seconds of no `tea.KeyMsg`
- [ ] Returning from idle resets `tickCount` to 0 for immediate data refresh
- [ ] Old hardcoded `playbackFetchInterval` and `queueFetchInterval` constants removed
- [ ] `make ci` passes

### Polling Schedule

| State | Playback Interval | Queue Interval |
|---|---|---|
| Active + Playing | 3s (current) | 9s (current) |
| Active + Paused | 10s | 30s |
| Idle (60s no input) + Playing | 10s | 30s |
| Idle + Paused | 30s | 60s |

"Active" = user interacted within the last 60 seconds.
"Idle" = no `tea.KeyMsg` received for 60+ seconds.

### Tasks

1. **Track lastInteraction time** — Add fields to App struct to track user activity recency and determine idle state.
   - Files: Modify `internal/app/app.go` (add fields, update New(), update KeyMsg handler)
   - Fields:
     ```go
     lastInteraction time.Time      // last time a tea.KeyMsg was received
     idleThreshold   time.Duration  // how long without input before idle
     ```
   - In `New()`: initialize `lastInteraction: time.Now()`, `idleThreshold: 60 * time.Second`
   - In `tea.KeyMsg` handler: set `a.lastInteraction = time.Now()` before any other processing
   - Helper:
     ```go
     func (a *App) isIdle() bool {
         return time.Since(a.lastInteraction) > a.idleThreshold
     }
     ```
   - Tests: Unit — `isIdle()` returns false immediately after creation; returns true after threshold elapses; KeyMsg resets lastInteraction
   - Commit: `feat(app): track last user interaction for idle detection`

2. **Adaptive pollIntervals() method** — Replace hardcoded polling constants with a method that returns intervals based on idle state and playback state.
   - Files: Modify `internal/app/app.go` (add constants and method)
   - Constants:
     ```go
     const (
         activePlayingPlaybackInterval = 3
         activePlayingQueueInterval    = 9
         reducedPlaybackInterval       = 10
         reducedQueueInterval          = 30
         idlePlaybackInterval          = 30
         idleQueueInterval             = 60
     )
     ```
   - Method:
     ```go
     func (a *App) pollIntervals() (playbackInterval, queueInterval int) {
         idle := a.isIdle()
         playing := false
         if ps := a.store.PlaybackState(); ps != nil {
             playing = ps.IsPlaying
         }
         switch {
         case !idle && playing:
             return activePlayingPlaybackInterval, activePlayingQueueInterval
         case !idle && !playing:
             return reducedPlaybackInterval, reducedQueueInterval
         case idle && playing:
             return reducedPlaybackInterval, reducedQueueInterval
         default: // idle && !playing
             return idlePlaybackInterval, idleQueueInterval
         }
     }
     ```
   - Tests: Unit — active + playing → 3s/9s; active + paused → 10s/30s; idle + playing → 10s/30s; idle + paused → 30s/60s
   - Commit: `feat(app): adaptive polling interval calculation`

3. **Wire into tick handler + idle-to-active reset + docs** — Replace hardcoded interval usage in the tick handler with dynamic intervals and add idle-to-active recovery.
   - Files: Modify `internal/app/app.go` (update tick handler, add idle-to-active reset); modify `docs/ARCHITECTURE.md` (add idle polling backoff documentation to "Polling Architecture")
   - Update tick handler (app.go lines 483-521):
     ```go
     playbackInterval, queueInterval := a.pollIntervals()
     if a.tickCount%playbackInterval == 0 {
         cmds = append(cmds, fetchPlaybackStateCmd(a.player))
     }
     if a.tickCount%queueInterval == 0 {
         cmds = append(cmds, fetchQueueCmd(a.player))
     }
     ```
   - Idle-to-active reset: when `tea.KeyMsg` arrives after app was idle, reset `a.tickCount = 0` to force immediate fetch on next tick:
     ```go
     case tea.KeyMsg:
         wasIdle := a.isIdle()
         a.lastInteraction = time.Now()
         if wasIdle {
             a.tickCount = 0
         }
         return a.handleKeyMsg(m)
     ```
   - Remove old hardcoded constants `playbackFetchInterval` and `queueFetchInterval` (app.go lines 41-44). Keep `defaultBackoffTicks` — used by 429 handler, not polling.
   - Docs: document the 4-state polling schedule and two-layer rate management design (Layer 1: proactive idle backoff, Layer 2: reactive gateway)
   - Tests: Unit — tick handler uses dynamic intervals; active+playing fires at 3s/9s; idle+paused fires at 30s/60s; KeyMsg after idle resets tickCount to 0; Integration — verify polling interval changes when playback state changes
   - Commit 1: `feat(app): adaptive polling with idle backoff`
   - Commit 2: `docs: add idle polling backoff to architecture docs`

**Verification:**
```bash
# Old hardcoded constants removed
grep -r 'playbackFetchInterval\b\|queueFetchInterval\b' internal/app/app.go
# Expected: ZERO matches (replaced by adaptive constants)

# New adaptive system in place
grep -r 'pollIntervals\|isIdle\|lastInteraction' internal/app/app.go
# Expected: multiple matches

make ci
# Expected: Full pass
```
