---
title: "State store + messages"
feature: 18-podcasts
status: done
---

## Background

The Store needs fields for podcast data (followed shows, saved episodes,
selected show episodes) following the same flat-field pattern as existing
library data. New message types are needed for fetch requests, loaded data,
selection changes, and episode playback commands.

## Design

### Store fields (`internal/state/store.go`)

Add after the existing library fields (before `queue []domain.Track`):

```go
// Podcast data
followedShows       []domain.SavedShow
savedEpisodes       []domain.SavedEpisode
showEpisodes        []domain.Episode
showEpisodesTotal   int
selectedShowID     string
selectedShow       *domain.Show
```

Add to the staleness tracking section:

```go
followedShowsFetchedAt   time.Time
savedEpisodesFetchedAt   time.Time
showEpisodesFetchedAt    time.Time
```

Add to the fetching sentinels section:

```go
followedShowsFetching bool
savedEpisodesFetching bool
showEpisodesFetching  bool
```

Add to the error state section (before `playlistsError`):

```go
followedShowsFetchErr    error
savedEpisodesFetchErr    error
showEpisodesFetchErr     error
```

### TTL constants

Add after the existing TTL constants:

```go
FollowedShowsTTL  = 5 * time.Minute
SavedEpisodesTTL  = 5 * time.Minute
ShowEpisodesTTL   = 5 * time.Minute
```

### Store accessors

Follow the exact same `RLock`/`Lock` pattern as existing accessors. Group of
methods to add, each with:
- Getter (RLock, return value)
- Setter (Lock, set value, stamp fetchedAt on non-empty for data fields)
- Loaded check (non-zero fetchedAt)
- FetchedAt getter/setter
- Stale check (delegates to `IsStale(fetchedAt, TTL)`)
- Fetching getter/setter
- Error getter/setter/clearer

Data accessor pairs:

| Getter | Setter | Loaded check |
|--------|--------|-------------|
| `FollowedShows()` | `SetFollowedShows()` | `FollowedShowsLoaded()` |
| `SavedEpisodes()` | `SetSavedEpisodes()` | `SavedEpisodesLoaded()` |
| `ShowEpisodes()` | `SetShowEpisodes()` | `ShowEpisodesLoaded()` |
| `ShowEpisodesTotal()` | `SetShowEpisodesTotal()` | — |
| `SelectedShowID()` | `SetSelectedShowID()` | — |
| `SelectedShow()` | `SetSelectedShow()` | — |

Staleness methods for each data domain:

| FetchedAt | Stale |
|-----------|-------|
| `FollowedShowsFetchedAt()` / `SetFollowedShowsFetchedAt()` | `FollowedShowsStale()` |
| `SavedEpisodesFetchedAt()` / `SetSavedEpisodesFetchedAt()` | `SavedEpisodesStale()` |
| `ShowEpisodesFetchedAt()` / `SetShowEpisodesFetchedAt()` | `ShowEpisodesStale()` |

Fetching sentinels:

| Getter | Setter |
|--------|--------|
| `FollowedShowsFetching()` | `SetFollowedShowsFetching(bool)` |
| `SavedEpisodesFetching()` | `SetSavedEpisodesFetching(bool)` |
| `ShowEpisodesFetching()` | `SetShowEpisodesFetching(bool)` |

Error state:

| Getter | Setter | Clearer |
|--------|--------|---------|
| `FollowedShowsFetchError()` | `SetFollowedShowsFetchError(error)` | `ClearFollowedShowsFetchError()` |
| `SavedEpisodesFetchError()` | `SetSavedEpisodesFetchError(error)` | `ClearSavedEpisodesFetchError()` |
| `ShowEpisodesFetchError()` | `SetShowEpisodesFetchError(error)` | `ClearShowEpisodesFetchError()` |

### Message types (`internal/ui/panes/messages.go`)

Add before `PollingSnapshotMsg`:

```go
// Podcast messages

type FetchFollowedShowsRequestMsg struct{}

type FetchSavedEpisodesRequestMsg struct{}

type FetchShowEpisodesRequestMsg struct {
	ShowID string
}

type FollowedShowsLoadedMsg struct {
	Items []domain.SavedShow
	Err   error
}

type SavedEpisodesLoadedMsg struct {
	Items []domain.SavedEpisode
	Err   error
}

type ShowEpisodesLoadedMsg struct {
	ShowID  string
	Items   []domain.Episode
	Total   int
	HasNext bool
	Err     error
}

type SelectedShowChangedMsg struct {
	ShowID string
}

type PlayEpisodeMsg struct {
	EpisodeURI  string
	PlaylistURI string // show URI for context, empty for saved episodes
}
```

## Acceptance Criteria

- [ ] All 9 data accessor methods compile (getter + setter for 3 data domains + total + ID + show)
- [ ] All 3 `*Loaded()` methods compile
- [ ] All 9 staleness/fetched-at methods compile
- [ ] All 6 fetching sentinel methods compile
- [ ] All 9 error state methods compile
- [ ] 3 TTL constants defined
- [ ] All 7 message types compile
- [ ] `go test ./internal/state/... -v` passes (existing tests + new table-driven tests for podcast fields)
- [ ] Existing tests are not broken by new store fields (zero-value defaults)

## Tasks

- [ ] Add TTL constants (`FollowedShowsTTL`, `SavedEpisodesTTL`, `ShowEpisodesTTL`) to `internal/state/store.go`
- [ ] Add tests: verify TTL constants are positive and distinct — in `internal/state/store_test.go`
- [ ] Add flat store fields (followedShows, savedEpisodes, showEpisodes, showEpisodesTotal, selectedShowID, selectedShow) to Store struct
- [ ] Add fetchedAt, fetching sentinel, and error fields to Store struct
- [ ] Add accessor methods for followed shows (get/set/loaded/fetchedAt/stale/fetching/error/clear) following existing Playlists pattern
- [ ] Add accessor methods for saved episodes following same pattern
- [ ] Add accessor methods for show episodes following same pattern
- [ ] Add accessor methods for SelectedShowID and SelectedShow (get/set)
- [ ] Add table-driven tests for each accessor group:
  - `TestStore_SetGetFollowedShows` — verify set returns via get
  - `TestStore_SetGetSavedEpisodes` — same pattern
  - `TestStore_SetGetShowEpisodes` — same pattern, includes total
  - `TestStore_SetGetSelectedShowID`, `TestStore_SetGetSelectedShow`
  - `TestStore_FollowedShowsFetchedAt_InitialZero`, `TestStore_FollowedShowsFetchedAt_NonEmptyStamp`
  - `TestStore_FollowedShowsStale_NeverFetched`, `TestStore_FollowedShowsStale_WithinTTL`, `TestStore_FollowedShowsStale_ExpiredTTL`
  - `TestStore_FollowedShowsFetching_DefaultFalse`, `TestStore_SetFollowedShowsFetching_SetsAndClears`
  - `TestStore_FollowedShowsFetchError_DefaultNil`, `TestStore_SetFollowedShowsFetchError_Sets`, `TestStore_ClearFollowedShowsFetchError`
  - Same staleness/fetching/error test patterns for SavedEpisodes and ShowEpisodes
- [ ] Extend `TestStore_ErrorState` with 3 new table entries for `FollowedShowsFetchError`, `SavedEpisodesFetchError`, `ShowEpisodesFetchError`
- [ ] Extend `TestStore_FetchedAt_Accessors` with podcast initial-zero + non-empty-stamp assertions
- [ ] Add message types to `internal/ui/panes/messages.go` (7 types)
- [ ] Add test: verify message types can be created with zero values (compilation check) — in `internal/ui/panes/messages_test.go`
- [ ] Run `go test ./internal/state/... -v` — all ~120+ tests pass
- [ ] Run `go build ./internal/ui/panes/...` — compiles
