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
- `buildSearchPageCmd(ctx, query, types, offset)` — one HTTP call, `limit=10`, returns
  `nil` on context cancellation
- `buildSearchBatchCmd` and all prefetch constants are deleted
- `convertSearchResult` is updated to return a flat `[]SearchListItem` + total int

## Design

### `buildSearchPageCmd` — full rewrite

```go
// buildSearchPageCmd fetches a single page of search results.
// ctx is cancelled by App when a new search starts or the overlay closes.
// Returns nil if ctx is already cancelled — Bubble Tea drops nil messages silently,
// preventing stale SearchPageLoadedMsg from entering the Update loop.
func buildSearchPageCmd(
    ctx context.Context,
    client SearchClient,
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

**Constants:**
```go
// SearchPageSize is the number of results per page.
// Matches Spotify's default; kept as a named constant for tests.
const SearchPageSize = 10
```

### `convertSearchResult` — flat list + total

```go
// convertSearchResult converts a Spotify search API response into a flat list of
// SearchListItems and a total result count. The total is the maximum across all
// returned types — the type with the most results determines how many pages exist.
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

For single-type tabs (Songs, Artists, Albums, Playlists) only one type field is populated
in the response — `total` will naturally equal that type's total.

For the "All" tab, `total = max(...)` represents the deepest result set across all types.
`hasNextPage()` = `intent.page * SearchPageSize < total`. Some types may be exhausted on
later pages (returning fewer items) — that is acceptable; the list shows whatever the API
returns.

### `buildSearchBatchCmd` — deleted

Delete `buildSearchBatchCmd` from `commands.go` entirely. Update all call sites in
`app.go` that called it — they now call `buildSearchPageCmd` instead (wired in story 100).

### Deleted constants

Remove from `commands.go` or wherever they are defined:
```go
SearchPrefetchPages     = 5
SearchPrefetchItems     = 50
SearchPrefetchThreshold = 0.6
SearchMaxOffset         = 1000
```

Keep `SearchPageSize = 10` (new).

### `checkPrefetch()`, `nextOffsetForTab()`, `CallCheckPrefetch()` — deleted

Delete from `internal/ui/panes/search.go`:
- `checkPrefetch()` — scroll-threshold prefetch trigger
- `nextOffsetForTab()` — per-type offset calculator
- `CallCheckPrefetch()` — test export (only existed because `checkPrefetch` was un-exported)

Remove their calls from `Update()`. Update any test files that called `CallCheckPrefetch`.

## Acceptance Criteria

- [ ] `buildSearchPageCmd` accepts `ctx context.Context` and returns `nil` when ctx is cancelled
- [ ] `buildSearchPageCmd` uses `limit=SearchPageSize (10)` and `offset=(page-1)*10`
- [ ] `convertSearchResult` returns `([]SearchListItem, int)` where int is `max(totals)`
- [ ] `buildSearchBatchCmd` is deleted; no references remain
- [ ] Prefetch constants (`SearchPrefetchPages`, `SearchPrefetchItems`, `SearchPrefetchThreshold`, `SearchMaxOffset`) are deleted
- [ ] `checkPrefetch()`, `nextOffsetForTab()`, `CallCheckPrefetch()` are deleted
- [ ] `make ci` passes

## Tasks

- [ ] Rewrite `buildSearchPageCmd` with ctx parameter, limit=10, nil-on-cancel behaviour
      - test: call with already-cancelled ctx → returns nil tea.Cmd (message is nil);
        call with valid ctx and mock HTTP → returns SearchPageLoadedMsg{Query, Page, Results, Total};
        call where ctx cancelled mid-flight (after HTTP) → returns nil

- [ ] Rewrite `convertSearchResult` to return `([]SearchListItem, int)` with flat list and max-total
      - test table:
        | tracks.Total | artists.Total | albums.Total | playlists.Total | expected total |
        |---|---|---|---|---|
        | 100 | 50 | 30 | 20 | 100 |
        | 0 | 0 | 0 | 0 | 0 |
        | 10 | 10 | 10 | 10 | 10 |
        | 7 | 0 | 0 | 0 | 7 (Songs tab) |
      - test: items slice contains track items first, then artist, album, playlist in order

- [ ] Delete `buildSearchBatchCmd` and all prefetch constants
      - test: grep for `buildSearchBatchCmd`, `SearchPrefetchPages`, `SearchPrefetchItems`,
        `SearchPrefetchThreshold`, `SearchMaxOffset` returns zero hits

- [ ] Delete `checkPrefetch()`, `nextOffsetForTab()`, `CallCheckPrefetch()` from search.go
      - remove calls from `Update()` (KeyUp / KeyDown handlers)
      - test: `make build` compiles; grep for these names returns zero hits

- [ ] `make ci` passes
