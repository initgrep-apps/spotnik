---
title: "Search Redesign: App context cancellation and staleness keys"
feature: 19-search-redesign
status: done
---

## Background

Currently, `buildSearchPageCmd` uses `context.Background()` — HTTP calls are never
cancelled. When the user types a new query or presses Escape, in-flight requests continue
running, hold gateway semaphore slots, and can deliver stale results into the overlay.

Additionally, app.go tracks search state entirely through the Store
(`store.SetSearchQuery`, `store.SetSearchLoading`). After story 97 removes the Store
search subsystem, app.go needs its own fields for staleness checking and loading state.

This story adds four fields to `App`, rewrites the `SearchRequestMsg` handler, updates
`closeSearch` and `openSearch`, and wires `SearchLoadingMsg` dispatch so the overlay
receives loading state before the HTTP call goes out. It also handles loading-flag
clearing when a search error occurs, so the overlay's spinner never gets stuck.

## Architecture Context

### Layer: App — the orchestrator

App sits between the overlay (user intent) and the commands layer (HTTP calls). It is
responsible for two things:
1. **Cancellation** — killing in-flight HTTP when the user moves on
2. **Staleness** — discarding late-arriving responses that no longer match current intent

```
SearchRequestMsg{Query, Types, Page}        ← emitted by overlay (story 99)
  │
  ▼
app.go SearchRequestMsg handler             ← THIS STORY
  │
  ├── a.searchCancel()                      cancel prior in-flight HTTP
  ├── a.searchQuery = m.Query               staleness key
  ├── a.searchPage  = m.Page                staleness key
  ├── a.searchLoading = true
  ├── ctx, a.searchCancel = context.WithCancel(...)
  │
  ├──► SearchLoadingMsg{IsFirstPage}        → overlay sets loadingFirstPage/Next
  │
  └──► buildSearchPageCmd(ctx, a.search, m.Query, m.Types, m.Page)   ← story 101 body
         │
         ▼
       SearchPageLoadedMsg{Query, Page, Results, Total, Err}
         │
         ▼
       app.go SearchPageLoadedMsg handler   ← THIS STORY
         │
         ├── m.Err != nil → toast + forward msg to overlay (clears loading flags)
         │
         └── staleness check:
               m.Query == a.searchQuery && m.Page == a.searchPage?
                 │ yes
                 ▼
               a.searchLoading = false
               searchPane.Update(m) → overlay updates results/total (story 102)
```

### Cancellation paths

```
Esc / closeSearch()          → a.searchCancel() → in-flight HTTP cancelled
new SearchRequestMsg arrives → a.searchCancel() → prior HTTP cancelled before new dispatch
openSearch()                 → a.searchCancel = func(){} → clean slate
```

### Key invariant

> `a.searchCancel` is **never nil**. It is initialized to `func(){}` in `NewApp()` and
> reset to a fresh no-op in `openSearch()`. Calling it is always safe.

## Design

### New fields on `App`

Add to `internal/app/app.go` `App` struct:

```go
// Search session state — staleness keys and cancellation only.
// These replace all store.searchState fields (removed in story 97).
searchQuery   string              // staleness key; "" when no active search
searchPage    int                 // staleness key; 0 when no active search
searchLoading bool                // true while HTTP call is in-flight
searchCancel  context.CancelFunc  // cancels in-flight HTTP; never nil
```

Initialize `searchCancel` in `NewApp()` (or equivalent constructor):
```go
a.searchCancel = func() {}
```

### `Results() []SearchListItem` stub on `SearchOverlay`

The `SearchRequestMsg` handler needs `len(a.searchPane.Results()) == 0` to determine
`IsFirstPage`. Add this stub to `SearchOverlay` now (story 102 completes the
implementation):

```go
// Results returns the current page of search results.
// Returns nil until the first successful search response arrives.
func (o *SearchOverlay) Results() []SearchListItem {
    return nil // TODO(19-search-redesign): returns o.results once added in story 102
}
```

### `SearchRequestMsg` handler — full rewrite

```go
case panes.SearchRequestMsg:
    // Cancel any in-flight HTTP call before starting a new one.
    a.searchCancel()

    // Record staleness keys for the incoming request.
    a.searchQuery = m.Query
    a.searchPage = m.Page
    a.searchLoading = true

    // Create a cancellable context for this request.
    ctx, cancel := context.WithCancel(context.Background())
    a.searchCancel = cancel

    // Tell the overlay we are loading before the HTTP call goes out.
    isFirst := len(a.searchPane.Results()) == 0
    loadingCmd := func() tea.Msg { return panes.SearchLoadingMsg{IsFirstPage: isFirst} }

    // Dispatch the fetch command (body rewritten in story 101).
    fetchCmd := buildSearchPageCmd(ctx, a.search, m.Query, m.Types, m.Page)

    return a, tea.Batch(loadingCmd, fetchCmd)
```

`buildSearchPageCmd` is currently a method on App (`a.buildSearchPageCmd`). Story 101
converts it to a standalone function with signature
`buildSearchPageCmd(ctx, client, query, types, page)`. Update the call site here to match
the standalone signature, passing `a.search` as the client. The function body is
unchanged until story 101 rewrites it.

### `SearchPageLoadedMsg` handler — staleness check + error clears loading flags

```go
case panes.SearchPageLoadedMsg:
    if m.Err != nil {
        // Clear app-level loading flag.
        a.searchLoading = false
        // Forward the error msg to overlay so it can clear its loading flags
        // (loadingFirstPage / loadingNextPage). The overlay keeps its existing
        // results visible — it only clears spinners.
        updated, _ := a.searchPane.Update(m)
        if sp, ok := updated.(panes.SearchPane); ok {
            a.searchPane = sp
        }
        return a, a.alerts.NewAlertCmd(notifications.Warning, "Search failed: "+m.Err.Error())
    }
    // Discard stale results.
    if m.Query != a.searchQuery || m.Page != a.searchPage {
        return a, nil
    }
    a.searchLoading = false
    updated, cmd := a.searchPane.Update(m)
    if sp, ok := updated.(panes.SearchPane); ok {
        a.searchPane = sp
    }
    return a, cmd
```

### `closeSearch()` — add cancellation

```go
func (a *App) closeSearch() (*App, tea.Cmd) {
    a.searchCancel()           // immediately abort in-flight HTTP
    a.searchCancel = func() {}
    a.searchQuery = ""
    a.searchPage = 0
    a.searchLoading = false
    a.searchOpen = false
    return a, nil
}
```

### `openSearch()` — reset cancel func

```go
func (a *App) openSearch() (*App, tea.Cmd) {
    a.searchCancel = func() {} // fresh no-op; prior cancel already called by closeSearch
    a.searchPane.Reset()
    a.searchOpen = true
    return a, a.searchPane.Init()
}
```

## Acceptance Criteria

- [ ] `App` has `searchQuery`, `searchPage`, `searchLoading`, `searchCancel` fields
- [ ] `searchCancel` is initialized to `func(){}` in app constructor (no nil dereference)
- [ ] `SearchOverlay` has a `Results() []SearchListItem` stub (returns nil; completed in story 102)
- [ ] `SearchRequestMsg` handler: calls `searchCancel()`, sets staleness keys, creates new ctx, sends `SearchLoadingMsg`, dispatches fetch using standalone `buildSearchPageCmd(ctx, a.search, query, types, page)` signature
- [ ] `SearchPageLoadedMsg` error branch: clears `a.searchLoading`, forwards msg to overlay (clears loading flags), sends toast
- [ ] `SearchPageLoadedMsg` handler: discards message if `m.Query != a.searchQuery || m.Page != a.searchPage`
- [ ] `closeSearch()` calls `searchCancel()`, resets to `func(){}`, clears all four fields
- [ ] `openSearch()` resets `searchCancel` to `func(){}` before calling `Reset()` + `Init()`
- [ ] No nil-cancel panics under any code path
- [ ] `make ci` passes

## Tasks

- [ ] Add `searchQuery`, `searchPage`, `searchLoading`, `searchCancel` fields to `App` struct;
      initialize `searchCancel = func(){}` in constructor
      - test: `NewApp()` does not panic; calling `a.searchCancel()` immediately after construction is safe

- [ ] Add `Results() []SearchListItem` stub to `SearchOverlay` (returns nil)
      - test: `Results()` compiles and returns nil

- [ ] Rewrite `SearchRequestMsg` handler: cancel prior, record keys, create ctx, send
      `SearchLoadingMsg` + `buildSearchPageCmd` (standalone signature with `a.search` as client)
      - test: send two `SearchRequestMsg` in sequence; verify first ctx is cancelled before second
        dispatch; verify `a.searchQuery` matches second msg; verify `a.searchLoading == true`

- [ ] Update `SearchPageLoadedMsg` handler: error branch forwards msg to overlay + sends toast;
      success branch has staleness check
      - test table:
        | m.Query | a.searchQuery | m.Page | a.searchPage | m.Err | expected |
        |---|---|---|---|---|---|
        | "jazz" | "jazz" | 1 | 1 | nil | forward to overlay |
        | "jazz" | "rock" | 1 | 1 | nil | discard (return nil cmd) |
        | "jazz" | "jazz" | 2 | 1 | nil | discard (page mismatch) |
        | "jazz" | "jazz" | 1 | 1 | someErr | toast + forward to overlay (clear loading flags) |

- [ ] Update `closeSearch()` to call `searchCancel()`, reset to `func(){}`, clear all four fields
      - test: open search, dispatch request, close; verify ctx passed to `buildSearchPageCmd` is
        cancelled; verify `a.searchQuery == ""` and `a.searchPage == 0`

- [ ] Update `openSearch()` to reset cancel func before `Reset()`/`Init()`
      - test: open → close → reopen; `a.searchCancel` is a fresh no-op; no stale cancel func

- [ ] `make ci` passes
