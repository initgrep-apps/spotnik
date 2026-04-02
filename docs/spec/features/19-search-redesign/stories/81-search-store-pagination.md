---
title: "Search Store: Per-Type Paginated Storage"
feature: 19-search-redesign
status: open
---

## Background

The current Store holds search data as a single `*domain.SearchResult` blob with flat fields: `searchResults`, `searchQuery`, `searchLoading`, `searchError`. This supports only one page of results (5 items per type) with no pagination tracking.

The redesign requires per-type paginated storage: each category (tracks, artists, albums, playlists) maintains its own slice of items, current offset, and total count. This enables the prefetch engine (Story 83) to append pages incrementally and the UI to know when more data is available.

This story replaces the flat search fields in `state/store.go` with a structured `SearchState` that supports per-type pagination. The existing `SearchAPI` interface also needs an `offset` parameter added.

## Design

### Store Changes (`internal/state/store.go`)

Replace the flat search fields:

```go
// Old (remove):
searchResults *domain.SearchResult
searchQuery   string
searchLoading bool
searchError   error

// New:
search SearchState
```

New `SearchState` struct (defined in `internal/state/store.go` or a new `internal/state/search_state.go`):

```go
type SearchState struct {
    Query       string
    ActiveType  string // "all", "track", "artist", "album", "playlist"
    Loading     bool
    Error       error

    // Per-type paginated results
    Tracks    TypePage[domain.Track]
    Artists   TypePage[domain.SearchArtist]
    Albums    TypePage[domain.SearchAlbum]
    Playlists TypePage[domain.SearchPlaylist]
}

type TypePage[T any] struct {
    Items  []T
    Offset int // next offset to fetch (= len(Items) when contiguous)
    Total  int // total available from API
}
```

### Accessor Methods

Replace existing accessors with new ones that read from `SearchState`:

- `SearchQuery() string` — reads `search.Query`
- `SetSearchQuery(q string)` — writes `search.Query`
- `SearchLoading() bool` — reads `search.Loading`
- `SetSearchLoading(b bool)` — writes `search.Loading`
- `SearchError() error` — reads `search.Error`
- `SetSearchError(err error)` / `ClearSearchError()` — writes `search.Error`
- `SearchActiveType() string` — reads `search.ActiveType`
- `SetSearchActiveType(t string)` — writes `search.ActiveType`
- `SearchTracks() TypePage[domain.Track]` — reads tracks page
- `AppendSearchTracks(items []domain.Track, total int)` — appends to tracks, updates offset/total
- (Same pattern for Artists, Albums, Playlists)
- `ClearSearchResults()` — resets all TypePage slices, offsets, totals
- `SearchHasMore(typeName string) bool` — returns `offset < total` for the given type

### API Changes (`internal/api/search.go`)

Add `offset` parameter to `Search()`:

```go
// Old:
Search(ctx context.Context, query string, types []string, limit int) (*SearchResult, error)

// New:
Search(ctx context.Context, query string, types []string, limit, offset int) (*SearchResult, error)
```

Update `SearchAPI` interface in `search_interfaces.go` to match. The `offset` param maps to `&offset=N` in the query string.

### Message Changes (`internal/ui/panes/messages.go`)

Update `SearchResultsMsg` to carry per-page data:

```go
type SearchPageLoadedMsg struct {
    Query   string // to discard stale results if query changed
    Type    string // "all", "track", etc.
    Offset  int
    Results *SearchResultData
    Err     error
}
```

Keep `SearchResultData` but it now represents one page of results (up to 10 items per type), not the entire result set. Add `Total` fields per type:

```go
type SearchResultData struct {
    Tracks       []SearchTrackItem
    TracksTotal  int
    Artists      []SearchArtistItem
    ArtistsTotal int
    Albums       []SearchAlbumItem
    AlbumsTotal  int
    Playlists       []SearchPlaylistItem
    PlaylistsTotal  int
}
```

### App Update Handler Changes (`internal/app/app.go`)

The `SearchRequestMsg` handler sets store state and dispatches `buildSearchCmd`. The `SearchPageLoadedMsg` handler:
1. Checks `msg.Query == store.SearchQuery()` — discards stale results
2. Appends items to the correct TypePage in the Store via `AppendSearch*`
3. Clears loading when the batch is complete

The `SearchClearedMsg` handler calls `store.ClearSearchResults()`.

## Acceptance Criteria

- [ ] `SearchState` struct with per-type `TypePage` in store
- [ ] All existing Store search accessors migrated to new struct
- [ ] `SearchAPI.Search()` accepts `offset` parameter
- [ ] `SearchPageLoadedMsg` carries per-page data with query staleness check
- [ ] `AppendSearch*` methods append without overwriting existing data
- [ ] `ClearSearchResults()` resets all type pages
- [ ] `SearchHasMore()` correctly reports when more pages are available
- [ ] All existing tests updated to use new API signatures
- [ ] make ci passes

## Tasks

- [ ] Create `SearchState` and `TypePage` structs in `internal/state/store.go`
      - test: TypePage append works correctly; offset tracks appended count; ClearSearchResults resets all fields
- [ ] Migrate existing Store search accessors to read/write from `SearchState`
      - test: all existing store_test.go search tests pass with new accessors; SearchQuery/SearchLoading/SearchError behave identically
- [ ] Add `AppendSearch*` and `SearchHasMore` methods
      - test: table-driven: append 10 items → offset=10; append again → offset=20; total=100 → HasMore=true; total=20 → HasMore=false
- [ ] Add `offset` parameter to `SearchAPI` interface and `SearchClient.Search()`
      - test: httptest server verifies offset query param is sent; existing search_test.go updated
- [ ] Update `SearchPageLoadedMsg` and `convertSearchResult` in `commands.go`
      - test: convertSearchResult populates Total fields from API response
- [ ] Update `app.go` handlers for `SearchPageLoadedMsg` and `SearchClearedMsg`
      - test: stale query results are discarded; fresh results appended to store; clear resets all
