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

This story must land first. It breaks compilation intentionally — the call sites that read
the old Store methods are fixed in subsequent stories (98–102). All tests that test the
deleted Store methods must be removed; new tests for overlay-owned state come in story 102.

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

After deleting the above, these locations will fail to compile — fix them by removing the
call (or replacing with the new overlay-owned approach added in subsequent stories):

| File | Call to remove/replace |
|---|---|
| `internal/app/app.go` | All `a.store.SetSearchQuery(...)`, `a.store.SetSearchLoading(...)`, `a.store.ClearSearchResults()`, `a.store.SearchQuery()`, `a.store.SearchLoading()`, `a.store.SearchHasMore()`, `a.store.SetSearchError(...)`, `a.store.ClearSearchError()`, `a.store.SearchActiveType()`, `a.store.SetSearchActiveType(...)` |
| `internal/app/commands.go` | All store read/write inside `buildSearchBatchCmd`, `buildSearchPageCmd` closures — **do not delete these commands yet** (story 101); instead remove only the store accesses and add `// TODO(19-search-redesign):` comments |
| `internal/ui/panes/search.go` | `o.store` field read in `View()`, `Update()`, `Init()` — stub or remove; the overlay's own `results` field (added story 99) is not yet present so temporarily return early/no-op where needed |

### `store *state.Store` field on SearchOverlay

The `SearchOverlay` struct currently holds a `store *state.Store` pointer used in
`View()` to read search results. Remove this field. Compilation will break in `View()` and
`Update()` — add `// TODO(19-search-redesign): replaced by o.results in story 99` comments
at each break point and return a safe zero value. Do not leave panics.

### Store tests

Delete all `TestStore_Search*` test functions in `internal/state/store_test.go` that test
the deleted methods. Do not replace them here; story 104 adds the new overlay/App tests.

## Acceptance Criteria

- [ ] `Store` has no search-related fields, methods, or types after this change
- [ ] `TypePage[T]` generic struct is deleted
- [ ] All deleted methods' call sites are either removed or marked with `// TODO(19-search-redesign):`
- [ ] `SearchOverlay.store` field is removed; `o.store` references replaced with no-ops + TODO comments
- [ ] `make build` compiles (no compilation errors, only TODO stubs)
- [ ] Old `TestStore_Search*` test functions are deleted
- [ ] `make lint` passes (no unused imports, no dead code warnings)

## Tasks

- [ ] Delete `searchState` struct and its embedded field from `Store` in `internal/state/store.go`
      - also delete `TypePage[T any]` generic struct
      - test: `TestStore_SetAndGet*Search*` tests deleted; store_test.go still compiles and passes remaining tests

- [ ] Delete all search getter/setter methods from `Store`
      (`SearchQuery`, `SetSearchQuery`, `SearchLoading`, `SetSearchLoading`, `SearchActiveType`,
      `SetSearchActiveType`, `SearchTracks`, `AppendSearchTracks`, `SearchArtists`,
      `AppendSearchArtists`, `SearchAlbums`, `AppendSearchAlbums`, `SearchPlaylists`,
      `AppendSearchPlaylists`, `ClearSearchResults`, `SearchHasMore`, `SetSearchError`, `ClearSearchError`)
      - test: `make build` succeeds after fixing all call sites in this task

- [ ] Remove `store *state.Store` from `SearchOverlay` struct; replace `o.store.*` reads with
      safe stubs (`return o, nil`) and `// TODO(19-search-redesign):` comments
      - test: `make build` compiles; search overlay renders (may show empty/hint state)

- [ ] Remove all `a.store.Set/Get/Append/Clear*Search*` calls from `internal/app/app.go` handlers
      and `internal/app/commands.go` command closures; mark with TODO comments
      - test: `make build` compiles; `make lint` passes

- [ ] `make ci` passes (lint + tests; coverage may drop temporarily — acceptable for this story only)
