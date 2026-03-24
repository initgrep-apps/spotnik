# Feature 29 — Elm Purity: Data-Carrying Messages

> **Feature:** Refactor all data-fetching `tea.Cmd` functions so they return data in
> their Msg payloads instead of writing to the Store inside goroutine closures.
> Only `Update()` may mutate the Store — this is the core Elm Architecture contract.

## Context

Nine `build*Cmd` / `fetch*Cmd` functions in `internal/app/commands.go` currently
mutate the Store directly inside `tea.Cmd` closures (goroutines). Their Msg types
are empty notification structs (e.g., `QueueLoadedMsg{}`) that signal panes to
re-read from the Store. This violates the Elm Architecture principle: only `Update()`
may mutate state.

**Already compliant commands** (no changes needed): `buildAddToQueueCmd` (line 251),
`buildPlayContextCmd` (line 104), `buildPlayTrackCmd` (line 124), `buildToggleLikeCmd`
(line 407), `buildCreatePlaylistCmd` (line 582), `buildRenamePlaylistCmd` (line 599),
`buildRemovePlaylistTrackCmd` (line 612), `buildReorderPlaylistTracksCmd` (line 625).
These already return data-carrying Msgs without Store writes.

**Gap reference:** G1 in `docs/superpowers/plans/2026-03-24-architecture-baseline.md`

**Depends on:** Nothing — can start immediately.

---

## Task 0: Extract domain types into `internal/domain/`

**Problem:** Msg types in `internal/ui/panes/messages.go` cannot carry API data
payloads without importing `internal/api/`, which would violate the `ui/ → api/`
import boundary. The domain types (Track, PlaybackState, etc.) currently live in
`internal/api/models.go`, coupling them to the API layer.

**Fix:**

1. Create `internal/domain/` package with `types.go`
2. Move the following types from `internal/api/models.go` to `internal/domain/types.go`:
   - `PlaybackState`, `Track`, `Artist`, `Album`
   - `SimplePlaylist`, `SimplePlaylistOwner`, `FullAlbum`
   - `SavedAlbum`, `SavedTrack`, `PlayHistory`
   - `FullArtist`, `SearchResult` (+ inner list types)
   - `QueueResponse`, `Device`, `PlayOptions`
3. Update `internal/api/models.go` to type-alias or re-export from `domain/`
4. Update all imports across ~20 files: `api/`, `state/`, `app/`, `ui/panes/`

**Files:**
- Create: `internal/domain/types.go`
- Modify: `internal/api/models.go` — re-export domain types
- Modify: `internal/state/store.go` — import `domain` instead of `api` for types
- Modify: `internal/ui/panes/messages.go` — import `domain` for Msg payloads
- Modify: `internal/app/commands.go` — update type references
- Modify: all files that reference `api.Track`, `api.PlaybackState`, etc.

**Tests:**
- `make ci` must pass — this is a pure refactor, no behavior change

**Commit:** `refactor(domain): extract shared types into internal/domain package`

---

## Task 1: Data-carrying PlaybackStateFetchedMsg + QueueLoadedMsg

**Problem:** `fetchPlaybackStateCmd` (commands.go line 491) writes `store.SetPlaybackState(ps)`
inside the goroutine. `fetchQueueCmd` (line 465) writes `store.SetQueue(qr.Queue)`,
`store.SetQueueError(err)`, and `store.ClearQueueError()` inside the goroutine.
Both return empty Msg structs.

**Fix:**

1. Update `messages.go`:
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

2. Update `fetchPlaybackStateCmd` (commands.go line 491-509):
   - Remove `store.SetPlaybackState(ps)` from the closure
   - Return `PlaybackStateFetchedMsg{State: ps}` on success
   - Return `PlaybackStateFetchedMsg{Err: err}` on non-rate-limit, non-401 error

3. Update `fetchQueueCmd` (commands.go line 465-487):
   - Remove `store.SetQueue(qr.Queue)`, `store.SetQueueError(err)`, `store.ClearQueueError()`
   - Return `QueueLoadedMsg{Tracks: qr.Queue}` on success
   - Return `QueueLoadedMsg{Err: err}` on non-rate-limit, non-401 error

4. Update `app.go` `Update()` handler for `PlaybackStateFetchedMsg` (around line 483+):
   - On non-nil `State`: call `a.store.SetPlaybackState(m.State)`
   - On non-nil `Err`: (log or ignore — transient polling error)
   - Forward to playerPane as before

5. Update `app.go` `Update()` handler for `QueueLoadedMsg`:
   - On nil `Err`: call `a.store.ClearQueueError()` then `a.store.SetQueue(m.Tracks)`
   - On non-nil `Err`: call `a.store.SetQueueError(m.Err)`
   - Forward to queuePane as before

**Files:**
- Modify: `internal/ui/panes/messages.go` — add payload fields
- Modify: `internal/app/commands.go` — remove Store writes from both closures
- Modify: `internal/app/app.go` — add Store writes in Update() handlers

**Tests:**
- Unit: `fetchPlaybackStateCmd` returns `PlaybackStateFetchedMsg{State: ...}` — no Store writes
- Unit: `fetchQueueCmd` returns `QueueLoadedMsg{Tracks: ...}` — no Store writes
- Unit: `Update(PlaybackStateFetchedMsg{State: ps})` writes to Store
- Unit: `Update(QueueLoadedMsg{Tracks: tracks})` writes to Store

**Commit:** `refactor(arch): data-carrying PlaybackStateFetchedMsg + QueueLoadedMsg`

---

## Task 2: Data-carrying library messages

**Problem:** Four library command builders write to the Store inside goroutine closures:
- `buildFetchPlaylistsCmd` (line 144) — writes `store.SetPlaylists()`, `store.SetPlaylistsTotal()`, pagination append logic
- `buildFetchAlbumsCmd` (line 174) — writes `store.SetSavedAlbums()`, error set/clear
- `buildFetchLikedTracksCmd` (line 199) — writes `store.SetLikedTracks()`, `store.SetLikedTotal()`, error set/clear
- `buildFetchRecentlyPlayedCmd` (line 225) — writes `store.SetRecentlyPlayed()`, error set/clear

All return empty Msg structs (`LibraryLoadedMsg{}`, `AlbumsLoadedMsg{}`, etc.).

**Fix:**

1. Update `messages.go` — add payload fields to the 4 library Msgs:
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

2. Update all 4 `build*Cmd` functions:
   - Remove all `store.Set*()` / `store.Clear*()` calls from closures
   - Return data in Msg payloads

3. Update `app.go` / `routing.go` `Update()` handlers:
   - Handle pagination append logic for `LibraryLoadedMsg`: if `m.Offset == 0`, set; else append
   - Handle error set/clear for all 4 domains
   - Forward to libraryPane as before

**Key detail — pagination:** `buildFetchPlaylistsCmd` (line 163-168) currently does:
```go
if offset == 0 {
    store.SetPlaylists(playlists)
} else {
    store.SetPlaylists(append(store.Playlists(), playlists...))
}
```
This logic moves to `Update()`. The Msg carries the raw page of items + the offset
so Update() can decide whether to replace or append.

**Files:**
- Modify: `internal/ui/panes/messages.go` — add payload fields to 4 Msgs
- Modify: `internal/app/commands.go` — remove Store writes from 4 closures
- Modify: `internal/app/app.go` or `routing.go` — add Store writes in Update() handlers

**Tests:**
- Unit: each `build*Cmd` returns data in Msg, does not write Store
- Unit: `Update()` handles pagination append for LibraryLoadedMsg
- Unit: `Update()` handles error set/clear for each domain

**Commit:** `refactor(arch): data-carrying library messages`

---

## Task 3: Data-carrying Stats, Search, Devices, PlaylistTracks

**Problem:** Remaining violating commands:
- `buildFetchStatsCmd` (line 426) — writes `store.SetTopTracks()`, `store.SetTopArtists()`, `store.SetStatsError()`, `store.ClearStatsError()`
- `buildSearchCmd` (line 272) — writes `store.SetSearchQuery()`, `store.SetSearchLoading()`, `store.SetSearchResults()`, `store.SetSearchError()`, `store.ClearSearchError()`
- `buildFetchDevicesCmd` (line 359) — writes `store.SetDevicesError()`, `store.ClearDevicesError()`
- `buildFetchPlaylistTracksCmd` (line 556) — writes `store.SetPlaylistTracks()`, `store.SetPlaylistsError()`, `store.ClearPlaylistsError()`

**Fix:**

1. Update Msg types:
   ```go
   type StatsLoadedMsg struct {
       TimeRange  string
       TopTracks  []domain.Track
       TopArtists []domain.FullArtist
       Err        error
   }
   ```
   - `SearchResultsMsg` already has `Results` + `Err` — just remove Store writes from the Cmd
   - `DevicesLoadedMsg` already carries `[]DeviceInfo` — move error write to Update()
   - Add payload to `PlaylistTracksLoadedMsg`:
   ```go
   type PlaylistTracksLoadedMsg struct {
       PlaylistID string
       Tracks     []domain.Track
       Err        error
   }
   ```

2. For `buildSearchCmd` specifically:
   - Move `store.SetSearchQuery(query)` and `store.SetSearchLoading(true)` OUT of the Cmd builder
     and into the caller site in `Update()` (before dispatching the Cmd)
   - Remove ALL `store.Set*` / `store.Clear*` from the closure
   - Store writes for results/loading/error happen in the `SearchResultsMsg` handler in `Update()`

3. Update all `Update()` handlers to write Store from Msg payloads.

**Files:**
- Modify: `internal/ui/panes/messages.go` — update Msg fields
- Modify: `internal/app/commands.go` — remove Store writes from 4 closures
- Modify: `internal/app/app.go` — add Store writes in Update() handlers
- Modify: `internal/app/routing.go` — add Store writes for playlist tracks

**Tests:**
- Unit: each command returns data in Msg, does not write Store
- Unit: `Update()` correctly writes Store for each Msg type
- Unit: `buildSearchCmd` no longer calls any `store.Set*` methods

**Commit:** `refactor(arch): data-carrying stats/search/devices/playlist messages`

---

## Task 4: Remove Store parameter from package-level functions

**Problem:** `fetchPlaybackStateCmd` and `fetchQueueCmd` are the only package-level
functions (not methods on `*App`) that take a `store *state.Store` parameter. After
Tasks 1-3, they no longer need it — all Store writes moved to `Update()`.

**Fix:**

1. Update `fetchPlaybackStateCmd(player api.PlayerAPI, store *state.Store) tea.Cmd`
   → `fetchPlaybackStateCmd(player api.PlayerAPI) tea.Cmd`
2. Update `fetchQueueCmd(player api.PlayerAPI, store *state.Store) tea.Cmd`
   → `fetchQueueCmd(player api.PlayerAPI) tea.Cmd`
3. Update all call sites in `app.go` (tick handler, backoff recovery, etc.)

**Bonus — Parallelize stats fetches:**

Refactor `buildFetchStatsCmd` to fetch top tracks and top artists concurrently:
```go
func (a *App) buildFetchStatsCmd(timeRange string) tea.Cmd {
    userAPI := a.userAPI
    return func() tea.Msg {
        // ...
        var wg sync.WaitGroup
        var tracks []domain.Track
        var artists []domain.FullArtist
        var tracksErr, artistsErr error

        wg.Add(2)
        go func() { defer wg.Done(); tracks, tracksErr = userAPI.TopTracks(ctx, timeRange, 25) }()
        go func() { defer wg.Done(); artists, artistsErr = userAPI.TopArtists(ctx, timeRange, 25) }()
        wg.Wait()
        // return combined result...
    }
}
```

Uses `sync.WaitGroup` (stdlib only — no `errgroup` per CLAUDE.md).

**Files:**
- Modify: `internal/app/commands.go` — remove store param, add WaitGroup
- Modify: `internal/app/app.go` — update call sites

**Tests:**
- Unit: both functions compile without store param
- Unit: stats Cmd returns both tracks and artists (verify parallel execution with timing)

**Commit 1:** `refactor(arch): remove store param from package-level command functions`
**Commit 2:** `perf(stats): parallelize top tracks and top artists fetches`

---

## Task 5: Documentation updates

**Updates:**

1. **`docs/ARCHITECTURE.md`** — Section "Message Flow":
   - Document the data-carrying Msg pattern
   - Show before/after: empty Msg vs data-carrying Msg
   - Explain that `build*Cmd` functions must NEVER write to Store

2. **`CLAUDE.md`** — Section "Architecture Rules":
   - Add rule: "Commands must not mutate the Store — return data in Msg payloads; only Update() writes to Store"

**Commit:** `docs: update architecture docs for data-carrying message pattern`

---

## Verification

After all tasks complete:

```bash
grep -r 'store\.Set\|store\.Clear' internal/app/commands.go
```

**Expected:** ZERO matches — no Store writes remain in command builders.

Note: Store *reads* (e.g., `store.PlaybackState()` in `buildPlaybackAPICmd` for volume/shuffle state)
are legitimate and expected to remain.

```bash
make ci
```

**Expected:** Full pass — lint, tests, 80% coverage.

---

*Depends on: None*
*Blocked by: Nothing*
*Blocks: Features 31, 32, 33*
