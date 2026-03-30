---
title: "Elm Purity -- Data-Carrying Messages"
feature: 11-api-gateway
status: done
---

## Background
Nine `build*Cmd` / `fetch*Cmd` functions in `internal/app/commands.go` mutated the Store directly inside `tea.Cmd` closures (goroutines). Their Msg types were empty notification structs (e.g., `QueueLoadedMsg{}`) that signaled panes to re-read from the Store. This violated the Elm Architecture principle: only `Update()` may mutate state. This story refactored every violating command to return data in typed Msg payloads and moved all Store writes into `Update()` handlers.

Already compliant commands (no changes needed): `buildAddToQueueCmd`, `buildPlayContextCmd`, `buildPlayTrackCmd`, `buildToggleLikeCmd`, `buildCreatePlaylistCmd`, `buildRenamePlaylistCmd`, `buildRemovePlaylistTrackCmd`, `buildReorderPlaylistTracksCmd`.

Gap reference: G1 in `docs/superpowers/plans/2026-03-24-architecture-baseline.md`

## Design

### Domain Type Extraction
Move shared domain types out of `internal/api/models.go` into a new `internal/domain/types.go` package to break the `ui/ -> api/` import dependency. Types moved: `PlaybackState`, `Track`, `Artist`, `Album`, `SimplePlaylist`, `SimplePlaylistOwner`, `FullAlbum`, `SavedAlbum`, `SavedTrack`, `PlayHistory`, `FullArtist`, `SearchResult`, `QueueResponse`, `Device`, `PlayOptions`.

### Data-Carrying Message Pattern
Each Msg type carries `Data` + `Err error` fields:
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

Update() handlers perform Store writes: on non-nil data call `a.store.Set*()`, on non-nil Err handle error.

### Pagination
`buildFetchPlaylistsCmd` currently does `if offset == 0 { store.SetPlaylists(playlists) } else { store.SetPlaylists(append(store.Playlists(), playlists...)) }`. This logic moves to `Update()`. The Msg carries the raw page of items + the offset so Update() can decide whether to replace or append.

### Store Parameter Removal
After refactoring, `fetchPlaybackStateCmd` and `fetchQueueCmd` no longer need the `store *state.Store` parameter.

### Parallel Stats
Refactor `buildFetchStatsCmd` to fetch top tracks and top artists concurrently using `sync.WaitGroup`.

### Verification
```bash
grep -r 'store\.Set\|store\.Clear' internal/app/commands.go
# Expected: ZERO matches
make ci
```

## Acceptance Criteria
- [ ] Zero `store.Set*` or `store.Clear*` calls remain in `internal/app/commands.go`
- [ ] All Msg types carry `Data` + `Err error` fields
- [ ] All Store writes happen exclusively in `Update()` handlers
- [ ] `ui/` never imports `api/` -- domain types live in `internal/domain/`
- [ ] `make ci` passes -- lint, tests, 80% coverage

## Tasks
- [ ] Extract domain types into `internal/domain/` to break ui/ -> api/ import dependency
      - test: make ci passes -- pure refactor, no behavior change
- [ ] Data-carrying PlaybackStateFetchedMsg + QueueLoadedMsg -- refactor fetchPlaybackStateCmd and fetchQueueCmd
      - test: fetchPlaybackStateCmd returns PlaybackStateFetchedMsg with data, no Store writes
      - test: fetchQueueCmd returns QueueLoadedMsg with data, no Store writes
      - test: Update(PlaybackStateFetchedMsg{State: ps}) writes to Store
      - test: Update(QueueLoadedMsg{Tracks: tracks}) writes to Store
- [ ] Data-carrying library messages -- refactor buildFetchPlaylistsCmd, buildFetchAlbumsCmd, buildFetchLikedTracksCmd, buildFetchRecentlyPlayedCmd
      - test: each build*Cmd returns data in Msg, does not write Store
      - test: Update() handles pagination append for LibraryLoadedMsg
      - test: Update() handles error set/clear for each domain
- [ ] Data-carrying Stats, Search, Devices, PlaylistTracks -- refactor remaining violating commands
      - test: each command returns data in Msg, does not write Store
      - test: Update() correctly writes Store for each Msg type
      - test: buildSearchCmd no longer calls any store.Set* methods
- [ ] Remove Store parameter from package-level functions and parallelize stats fetches
      - test: both functions compile without store param
      - test: stats Cmd returns both tracks and artists
- [ ] Documentation updates -- document data-carrying Msg pattern in architecture docs
      - test: docs change only
