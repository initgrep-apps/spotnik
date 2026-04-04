---
title: "Search Redesign: Per-page fetch command and batch engine removal"
feature: 19-search-redesign
status: open
---

## Background

`buildSearchBatchCmd` dispatches the first page and relies on `SearchPageLoadedMsg`
handlers to chain subsequent pages (chain-through-Update). This produces up to 5
in-flight requests that continue running even after the user has moved on, and requires
`SearchPrefetchMsg` to drive the next batch. There is no context cancellation.

This story replaces the batch engine with a single clean per-page command:
- `buildSearchPageCmd(ctx, client, query, types, page)` — one HTTP call, `limit=10`,
  returns `nil` on context cancellation
- `buildSearchBatchCmd` and all prefetch constants are deleted
- `convertSearchResult` is updated to return a flat `[]SearchListItem` + total int
- `checkPrefetch()`, `nextOffsetForTab()`, `CallCheckPrefetch()` on the overlay are deleted

## Architecture Context

### Layer: Commands — the transport boundary

Commands are the only place where side effects (HTTP calls) are allowed. They run outside
the Update loop in goroutines and return a single message when done. This story is
responsible for the bottom half of the search data flow:

```
buildSearchPageCmd(ctx, client, query, types, page)   ← THIS STORY rewrites this
  │
  ▼
if ctx.Err() != nil → return nil          cancel check before HTTP
  │
  ▼
Gateway.Do(ctx, Interactive, path, fn)    debounced at gateway layer (story 103)
  │
  ▼
Spotify API (limit=SearchPageSize, offset=(page-1)*SearchPageSize)
  │
  ▼
if ctx.Err() != nil → return nil          cancel check after HTTP (user moved on)
  │
  ▼
convertSearchResult(r) → ([]SearchListItem, total int)   ← THIS STORY rewrites this
  │
  ▼
SearchPageLoadedMsg{Query, Page, Results, Total}
  │
  ▼
app.go staleness check (story 100) → overlay update (story 102)
```

### Why `nil` return on cancellation?

BubbleTea drops `nil` messages silently — they never enter the `Update()` loop. Returning
`nil` from a cancelled command is the correct Elm-architecture pattern for "this result is
no longer relevant." It avoids a dedicated `SearchCancelledMsg` type and keeps the Update
loop clean.

### Signature alignment with story 100

Story 100's `SearchRequestMsg` handler calls:
```go
fetchCmd := buildSearchPageCmd(ctx, a.search, m.Query, m.Types, m.Page)
```

This story defines `buildSearchPageCmd` as a standalone function (not a method on App)
with that exact signature. The function uses the `client` parameter rather than a captured
`a.search` variable, making it easier to test in isolation with a mock client.

### All tab — flat interleaved list

When `types` includes all four types (Tab = All), the Spotify API returns up to 10 items
per type (40 items total). `convertSearchResult` interleaves them into a single flat slice:
tracks first, then artists, albums, playlists. The `total` is the **maximum** across all
type totals — the deepest result set determines how many pages exist. `hasNextPage()` in
the overlay (story 102) uses this total.

```
total = max(r.Tracks.Total, r.Artists.Total, r.Albums.Total, r.Playlists.Total)
```

For single-type tabs, only one field is populated; `total` equals that type's total.

## Design

### `buildSearchPageCmd` — full rewrite as standalone function

```go
// SearchPageSize is the number of results fetched per page.
// Matches Spotify's recommended default; named for test clarity.
const SearchPageSize = 10

// buildSearchPageCmd fetches a single page of search results.
// ctx is cancelled by App when a new search starts or the overlay closes.
// Returns nil if ctx is already cancelled — Bubble Tea drops nil messages
// silently, preventing stale SearchPageLoadedMsg from entering the Update loop.
func buildSearchPageCmd(
    ctx context.Context,
    client api.SearchAPI,
    query string,
    types []api.SearchType,
    page int,
) tea.Cmd {
    return func() tea.Msg {
        if ctx.Err() != nil {
            return nil
        }
        offset := (page - 1) * SearchPageSize
        result, err := client.Search(ctx, query, types, SearchPageSize, offset)
        if ctx.Err() != nil {
            // Request completed but context was cancelled — caller has moved on.
            return nil
        }
        if err != nil {
            return SearchPageLoadedMsg{Query: query, Page: page, Err: err}
        }
        items, total := convertSearchResult(result)
        return SearchPageLoadedMsg{
            Query:   query,
            Page:    page,
            Results: items,
            Total:   total,
        }
    }
}
```

### `convertSearchResult` — flat list + total

```go
// convertSearchResult converts a Spotify search API response into a flat list
// of SearchListItems and a total result count.
//
// For the All tab the total is the maximum across all returned types — the
// deepest result set determines how many pages exist. For single-type tabs
// only one field is non-zero so the max equals that type's total.
func convertSearchResult(r *api.SearchResult) ([]SearchListItem, int) {
    var items []SearchListItem

    for _, t := range r.Tracks.Items {
        items = append(items, trackToListItem(t))
    }
    for _, a := range r.Artists.Items {
        items = append(items, artistToListItem(a))
    }
    for _, a := range r.Albums.Items {
        items = append(items, albumToListItem(a))
    }
    for _, p := range r.Playlists.Items {
        items = append(items, playlistToListItem(p))
    }

    total := max(r.Tracks.Total, r.Artists.Total, r.Albums.Total, r.Playlists.Total)
    return items, total
}
```

### `buildSearchBatchCmd` — deleted

Delete `buildSearchBatchCmd` from `commands.go` entirely. Its call sites in `app.go`
already call `buildSearchPageCmd` after story 100's handler rewrite.

### Deleted constants

Remove from `commands.go` or wherever they are defined:
```go
SearchPrefetchPages     = 5
SearchPrefetchItems     = 50
SearchPrefetchThreshold = 0.6
SearchMaxOffset         = 1000
```

Keep `SearchPageSize = 10` (new, defined here).

### `checkPrefetch()`, `nextOffsetForTab()`, `CallCheckPrefetch()` — deleted

Delete from `internal/ui/panes/search.go`:
- `checkPrefetch()` — scroll-threshold prefetch trigger
- `nextOffsetForTab()` — per-type offset calculator
- `CallCheckPrefetch()` — test export (only existed to expose the un-exported method)

Remove their calls from `Update()` (KeyUp / KeyDown handlers). Update any test files that
called `CallCheckPrefetch`.

## Acceptance Criteria

- [ ] `buildSearchPageCmd` is a standalone function (not a method on App); signature is
      `(ctx, client api.SearchAPI, query, types, page)`
- [ ] `buildSearchPageCmd` returns `nil` when ctx is cancelled before or after the HTTP call
- [ ] `buildSearchPageCmd` uses `limit=SearchPageSize (10)` and `offset=(page-1)*SearchPageSize`
- [ ] `convertSearchResult` returns `([]SearchListItem, int)` where int is `max(all type totals)`
- [ ] Items are interleaved: tracks → artists → albums → playlists
- [ ] `buildSearchBatchCmd` is deleted; no references remain
- [ ] Prefetch constants (`SearchPrefetchPages`, `SearchPrefetchItems`, `SearchPrefetchThreshold`, `SearchMaxOffset`) are deleted
- [ ] `checkPrefetch()`, `nextOffsetForTab()`, `CallCheckPrefetch()` are deleted from `search.go`
- [ ] `make ci` passes

## Tasks

- [ ] Rewrite `buildSearchPageCmd` as standalone function with ctx, client, query, types, page
      parameters; add `SearchPageSize = 10` constant
      - test: call with already-cancelled ctx → returns nil tea.Cmd;
        call with valid ctx and mock HTTP server → returns
        `SearchPageLoadedMsg{Query, Page, Results, Total}`;
        call where ctx cancelled after HTTP response → returns nil

- [ ] Rewrite `convertSearchResult` to return `([]SearchListItem, int)` with flat interleaved
      list and max-total
      - test table for All tab (all types populated):
        | tracks.Total | artists.Total | albums.Total | playlists.Total | expected total |
        |---|---|---|---|---|
        | 100 | 50 | 30 | 20 | 100 |
        | 0 | 0 | 0 | 0 | 0 |
        | 10 | 10 | 10 | 10 | 10 |
        | 7 | 0 | 0 | 0 | 7 (Songs tab) |
      - test: items slice is ordered tracks → artists → albums → playlists
      - test (All tab): tracks.Total=100, artists.Total=50 → total=100;
        `hasNextPage()` at page 1 with SearchPageSize=10 is true (100 > 10)

- [ ] Delete `buildSearchBatchCmd` and all prefetch constants
      - test: grep for `buildSearchBatchCmd`, `SearchPrefetchPages`, `SearchPrefetchItems`,
        `SearchPrefetchThreshold`, `SearchMaxOffset` returns zero hits

- [ ] Delete `checkPrefetch()`, `nextOffsetForTab()`, `CallCheckPrefetch()` from `search.go`;
      remove their calls from `Update()` (KeyUp / KeyDown handlers)
      - test: `make build` compiles; grep for these names returns zero hits

- [ ] `make ci` passes
