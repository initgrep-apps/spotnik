---
title: "Search Prefetch Pagination Engine"
feature: 19-search-redesign
status: open
---

## Background

The current search fires a single API call with `limit=5` per type and no pagination. The Spotify API now caps `limit` at 10 per type per request (Feb 2026 change), with `offset` up to 1000.

This story implements the prefetch pagination engine: on each search query, fire 5 sequential API calls (offsets 0, 10, 20, 30, 40) to load 50 items per type upfront. When the user scrolls past 60% of loaded items, fire the next 5-page batch (offsets 50-90), continuing until `offset >= total` or `offset > 1000`.

The gateway supports 5 concurrent requests max, so pages are fetched sequentially within a batch (using `tea.Sequence`), not in parallel.

## Bubble Tea Components

This story is primarily about command/message plumbing (Elm architecture), not UI components. However, it relies on these Bubble Tea patterns:

| Pattern | Import | Role in This Story |
|---|---|---|
| **tea.Sequence** | `github.com/charmbracelet/bubbletea` | Sequences 5 page-fetch commands so they execute in order (respecting gateway's 5-concurrency cap) |
| **tea.Cmd / tea.Msg** | `github.com/charmbracelet/bubbletea` | Each `buildSearchPageCmd` returns a `tea.Cmd` that produces a `SearchPageLoadedMsg` |
| **tea.Batch** | `github.com/charmbracelet/bubbletea` | NOT used for page fetches (would overwhelm gateway); used only for combining unrelated commands |

**Reference**: See `/bubbletea` skill for Cmd/Msg flow patterns. Key distinction:
- `tea.Batch(cmds...)` — runs commands concurrently (NOT appropriate here — gateway has 5-slot semaphore)
- `tea.Sequence(cmds...)` — runs commands in order, each waits for previous to complete (correct for sequential page fetching)

## Design

### Constants

```go
const (
    searchPageSize       = 10  // API max per request (Feb 2026)
    searchPrefetchPages  = 5   // pages per batch
    searchPrefetchItems  = searchPageSize * searchPrefetchPages // 50
    searchPrefetchThreshold = 0.6 // trigger next batch at 60% scroll
    searchMaxOffset      = 1000   // Spotify API hard cap
)
```

### buildSearchBatchCmd

Replace `buildSearchCmd` with `buildSearchBatchCmd` that fetches a batch of pages:

```go
func (a *App) buildSearchBatchCmd(query string, types []string, startOffset int) tea.Cmd {
    // Build 5 sequential page-fetch commands
    var cmds []tea.Cmd
    for i := 0; i < searchPrefetchPages; i++ {
        offset := startOffset + (i * searchPageSize)
        if offset > searchMaxOffset { break }
        cmds = append(cmds, a.buildSearchPageCmd(query, types, offset))
    }
    return tea.Sequence(cmds...)
}
```

Each `buildSearchPageCmd` fires one API call and returns `SearchPageLoadedMsg`:

```go
func (a *App) buildSearchPageCmd(query string, types []string, offset int) tea.Cmd {
    search := a.search
    return func() tea.Msg {
        if search == nil {
            return SearchPageLoadedMsg{Query: query, Err: errNilClient}
        }
        results, err := search.Search(
            api.WithPriority(context.Background(), api.Interactive),
            query, types, searchPageSize, offset,
        )
        if err != nil {
            // Handle rate limit and auth errors as before
            if retryAfter := parse429RetryAfter(err); retryAfter > 0 {
                return panes.RateLimitedMsg{RetryAfterSecs: retryAfter}
            }
            if isUnauthorizedError(err) {
                return unauthorizedMsg{}
            }
            return SearchPageLoadedMsg{Query: query, Offset: offset, Err: err}
        }
        return SearchPageLoadedMsg{
            Query:   query,
            Offset:  offset,
            Results: convertSearchResult(results),
        }
    }
}
```

### SearchPageLoadedMsg Handler (`app.go`)

```go
case panes.SearchPageLoadedMsg:
    // Discard stale results (query changed while fetching)
    if m.Query != a.store.SearchQuery() {
        return a, nil
    }
    if m.Err != nil {
        a.store.SetSearchError(m.Err)
        return a, a.alerts.NewAlertCmd("error", m.Err.Error())
    }
    a.store.ClearSearchError()
    // Append page results to store
    a.store.AppendSearchTracks(m.Results.Tracks, m.Results.TracksTotal)
    a.store.AppendSearchArtists(m.Results.Artists, m.Results.ArtistsTotal)
    a.store.AppendSearchAlbums(m.Results.Albums, m.Results.AlbumsTotal)
    a.store.AppendSearchPlaylists(m.Results.Playlists, m.Results.PlaylistsTotal)
    // Clear loading on last page of batch
    if m.Offset >= (startOffset + (searchPrefetchPages-1) * searchPageSize) {
        a.store.SetSearchLoading(false)
    }
    // Forward to overlay for re-render
    return a, nil
```

### Prefetch Trigger

The overlay (Story 84) tracks scroll position via the `list.Model` cursor index. When `cursorIndex / totalLoadedItems >= 0.6`, it emits:

```go
type SearchPrefetchMsg struct {
    Query string
    Types []string
    NextOffset int
}
```

The `app.go` handler:

```go
case panes.SearchPrefetchMsg:
    if m.Query != a.store.SearchQuery() { return a, nil }
    if !a.store.SearchHasMore(m.Types) { return a, nil }
    a.store.SetSearchLoading(true)
    return a, a.buildSearchBatchCmd(m.Query, m.Types, m.NextOffset)
```

### SearchRequestMsg Handler Update

The existing `SearchRequestMsg` handler changes to:
1. Clear previous results (`store.ClearSearchResults()`)
2. Set query and loading state
3. Dispatch `buildSearchBatchCmd` with offset=0

### Tab Change Flow

When `SearchTabChangedMsg` arrives:
1. `store.SetSearchActiveType(tabType)`
2. `store.ClearSearchResults()` — clear all type pages
3. `store.SetSearchLoading(true)`
4. Dispatch `buildSearchBatchCmd(query, tabTypes, 0)`

This ensures switching tabs always fetches fresh data for the selected type.

## Acceptance Criteria

- [ ] Initial search fires 5 sequential API calls (offsets 0-40, limit=10)
- [ ] Each page appends results to Store without overwriting previous pages
- [ ] Stale results (query changed mid-fetch) are discarded
- [ ] `SearchPrefetchMsg` triggers next 5-page batch when scroll > 60%
- [ ] Prefetch stops when `offset >= total` or `offset > 1000`
- [ ] Tab change clears results and fires fresh batch with filtered types
- [ ] Rate limit and auth error handling preserved
- [ ] Sequential execution via `tea.Sequence` (not parallel)
- [ ] make ci passes

## Tasks

- [ ] Define prefetch constants in `internal/app/commands.go`
      - test: constants match spec values (pageSize=10, prefetchPages=5, threshold=0.6, maxOffset=1000)
- [ ] Implement `buildSearchPageCmd` (single-page fetch returning `SearchPageLoadedMsg`)
      - test: httptest server receives correct offset/limit params; result carries query+offset; nil client returns error msg
- [ ] Implement `buildSearchBatchCmd` (sequences 5 page commands)
      - test: batch creates exactly 5 commands for offset 0; creates fewer when near maxOffset; offset 990 creates only 2 commands
- [ ] Wire `SearchPageLoadedMsg` handler in `app.go`
      - test: stale query results discarded; fresh results appended to store; error triggers toast; loading cleared on last page
- [ ] Add `SearchPrefetchMsg` and handler in `app.go`
      - test: prefetch dispatches batch with correct nextOffset; skipped when no more results; skipped when query stale
- [ ] Update `SearchRequestMsg` handler to use batch commands
      - test: clears previous results; sets loading; dispatches batch with offset=0
- [ ] Update `SearchTabChangedMsg` handler to clear and re-fetch
      - test: tab change clears results; sets active type; dispatches batch with tab-specific types
- [ ] Elm purity test: verify no Store writes inside command closures
      - test: execute buildSearchPageCmd closure; store unchanged; msg carries data payload
