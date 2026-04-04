---
title: "Search Redesign: Remove all search state from Store"
feature: 19-search-redesign
status: open
---

## Background

The current Store holds all search display state: per-type result slices in `TypePage[T]`
structs, `searchActiveType`, `searchLoading`, `searchQuery`, `searchHasMore`, and error
fields. The overlay reads from the Store on every `View()` call. This indirection adds
complexity without benefit — search results are only ever read by `SearchOverlay`, never by
any other pane.

The redesigned architecture moves all display state onto the overlay itself and all
staleness/cancellation keys onto `App`. The Store retains zero search state after this story.

**This story must land first.** It breaks compilation intentionally — the call sites that
read the old Store methods are fixed in subsequent stories (98–102). All tests that test the
deleted Store methods must be removed; new tests for overlay-owned state come in story 102.

## Architecture Context

### Layer: Store → zero search state

This story attacks the root of the complexity problem: the Store as an intermediary between
the API and the overlay. Right now the data path is:

```
Spotify API → buildSearchPageCmd → SearchPageLoadedMsg
  → app.go: store.AppendSearchTracks() / store.SetSearchLoading() / ...
    → SearchOverlay.View(): store.SearchTracks() / store.SearchLoading() / ...
```

After this story the Store is cut out entirely. The path becomes (completed across 97–102):

```
Spotify API → buildSearchPageCmd → SearchPageLoadedMsg
  → app.go: staleness check → forward to overlay
    → SearchOverlay: o.results / o.total / o.loadingFirstPage
```

### What this story changes vs what comes later

| Component | This story | Later story |
|---|---|---|
| `Store` | Delete all search fields + methods | — |
| `SearchOverlay.store` | Remove field; stub `View()`/`Update()` | — |
| `SearchOverlay.results` | Leave `*SearchResultData` in place (used in View until story 102) | Replaced in story 102 |
| `App` search fields | Add TODO stubs | New fields added in story 100 |
| `commands.go` store calls | Remove store reads/writes; keep function shells | `buildSearchBatchCmd` deleted in story 101 |

### Key invariant this story establishes

> **No search state lives in Store after this story.** The overlay is temporarily in a
> broken/empty state (View() may show hints or empty panels) — that is acceptable.
> Stories 99–102 restore full functionality.

## Design

### Fields and methods to delete from `internal/state/store.go`

**Struct:**
```go
// delete entire searchState embedded struct and its field on Store
type searchState struct { ... }
```

**Generic struct (delete entirely):**
```go
type TypePage[T any] struct {
    Items  []T
    Offset int
    Total  int
}
```

**Methods to delete:**
- `SearchQuery() string` / `SetSearchQuery(q string)`
- `SearchLoading() bool` / `SetSearchLoading(loading bool)`
- `SearchActiveType() SearchType` / `SetSearchActiveType(t SearchType)`
- `SearchTracks() TypePage[domain.Track]` / `AppendSearchTracks(page TypePage[domain.Track])`
- `SearchArtists() TypePage[domain.Artist]` / `AppendSearchArtists(...)`
- `SearchAlbums() TypePage[domain.Album]` / `AppendSearchAlbums(...)`
- `SearchPlaylists() TypePage[domain.Playlist]` / `AppendSearchPlaylists(...)`
- `ClearSearchResults()`
- `SearchHasMore() bool`
- `SetSearchError(err error)` / `ClearSearchError()`

### Call sites to fix

After deleting the above, these locations will fail to compile — fix them:

| File | What to do |
|---|---|
| `internal/app/app.go` | Remove all `a.store.Set/Get/Append/Clear*Search*` calls; replace with `// TODO(19-search-redesign): replaced by App.searchQuery/searchPage/searchLoading in story 100` |
| `internal/app/commands.go` | Remove only the store read/write calls inside `buildSearchBatchCmd` and `buildSearchPageCmd` closures. **Do not delete the function shells** — `buildSearchBatchCmd` is deleted in story 101; `buildSearchPageCmd` is rewritten in story 101. Mark each removed store call with `// TODO(19-search-redesign): store access removed; context cancellation added in story 101` |
| `internal/ui/panes/search.go` | `o.store.*` reads in `View()`, `Update()`, `Init()` — remove field usage and return safe zero values. Add `// TODO(19-search-redesign): replaced by o.results in story 102` at each break point. **Do not remove `results *SearchResultData`** — that field is replaced in story 102. |

### `store *state.Store` field on SearchOverlay

`SearchOverlay` holds `store *state.Store` used in `View()` to read search results. Remove
this field only (not `results *SearchResultData` — that stays until story 102). Where
`o.store.*` is accessed, add safe stubs that return zero values:

```go
// TODO(19-search-redesign): o.store removed; results now read from o.results in story 102.
// Return hint state until story 102 adds overlay-owned results.
```

Do not leave panics or nil-dereferences.

### Store tests

Delete all `TestStore_Search*` test functions in `internal/state/store_test.go` that test
the deleted methods. Do not replace them here; story 102 adds the overlay tests and story
104 adds the integration tests.

## Acceptance Criteria

- [ ] `Store` has no search-related fields, methods, or types after this change
- [ ] `TypePage[T]` generic struct is deleted
- [ ] All deleted methods' call sites are removed or marked with `// TODO(19-search-redesign):`
- [ ] `SearchOverlay.store` field is removed; `o.store` references replaced with no-ops + TODO comments
- [ ] `results *SearchResultData` field is **left in place** (removed in story 102)
- [ ] `buildSearchBatchCmd` and `buildSearchPageCmd` function shells remain in `commands.go`; only their internal store calls are removed
- [ ] `make build` compiles (no compilation errors, only TODO stubs)
- [ ] Old `TestStore_Search*` test functions are deleted; `store_test.go` still compiles and passes remaining tests
- [ ] `make lint` passes (no unused imports, no dead code warnings)

## Tasks

- [ ] Delete `searchState` struct and its embedded field from `Store` in `internal/state/store.go`;
      also delete `TypePage[T any]` generic struct
      - test: `TestStore_SetAndGet*Search*` tests deleted; `store_test.go` still compiles and passes remaining tests

- [ ] Delete all search getter/setter methods from `Store`
      (`SearchQuery`, `SetSearchQuery`, `SearchLoading`, `SetSearchLoading`, `SearchActiveType`,
      `SetSearchActiveType`, `SearchTracks`, `AppendSearchTracks`, `SearchArtists`,
      `AppendSearchArtists`, `SearchAlbums`, `AppendSearchAlbums`, `SearchPlaylists`,
      `AppendSearchPlaylists`, `ClearSearchResults`, `SearchHasMore`, `SetSearchError`, `ClearSearchError`)
      - test: `make build` succeeds after fixing all call sites in this task

- [ ] Remove `store *state.Store` from `SearchOverlay` struct; replace `o.store.*` reads with
      safe stubs (`return o, nil` / hint text) and `// TODO(19-search-redesign):` comments.
      Leave `results *SearchResultData` field untouched.
      - test: `make build` compiles; search overlay renders (shows hint/empty state, no panic)

- [ ] Remove all `a.store.Set/Get/Append/Clear*Search*` calls from `internal/app/app.go` handlers;
      mark with TODO comments
      - test: `make build` compiles; `make lint` passes

- [ ] Remove store read/write calls from inside `buildSearchBatchCmd` and `buildSearchPageCmd`
      closures in `internal/app/commands.go`; keep function shells; mark with TODO comments
      - test: `make build` compiles; `make lint` passes

- [ ] `make ci` passes (lint + tests; coverage may drop temporarily — acceptable for this story only)
