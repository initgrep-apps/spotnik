---
title: "Search Redesign: App context cancellation and staleness keys"
feature: 19-search-redesign
status: open
---

## Background

Currently, `buildSearchPageCmd` uses `context.Background()` — HTTP calls are never
cancelled. When the user types a new query or presses Escape, in-flight requests continue
running, hold gateway semaphore slots, and can deliver stale results into the overlay.

Additionally, app.go tracks search state entirely through the Store
(`store.SetSearchQuery`, `store.SetSearchLoading`). After story 97 removes the Store
search subsystem, app.go needs its own fields for staleness checking and loading state.

This story adds four fields to `App`, rewrites the `SearchRequestMsg` handler, updates
`closeSearch`, and wires `SearchLoadingMsg` dispatch so the overlay receives loading state
before the HTTP call goes out.

## Design

### New fields on `App`

Add to `internal/app/app.go` `App` struct:

```go
// Search session state — staleness keys and cancellation only.
// These replace all store.searchState fields.
searchQuery   string              // staleness key; "" when no active search
searchPage    int                 // staleness key; 0 when no active search
searchLoading bool                // true while HTTP call is in-flight
searchCancel  context.CancelFunc  // cancels in-flight HTTP; init to func(){}
```

Initialize `searchCancel` in `NewApp()` (or equivalent constructor):
```go
a.searchCancel = func() {}
```

### `SearchRequestMsg` handler — full rewrite

```go
case panes.SearchRequestMsg:
    // Cancel any in-flight HTTP call.
    a.searchCancel()

    // Record staleness keys.
    a.searchQuery = m.Query
    a.searchPage = m.Page
    a.searchLoading = true

    // Create a cancellable context for this request.
    ctx, cancel := context.WithCancel(context.Background())
    a.searchCancel = cancel

    // Notify overlay of loading state before dispatch.
    isFirst := len(a.searchPane.Results()) == 0
    loadingCmd := func() tea.Msg { return panes.SearchLoadingMsg{IsFirstPage: isFirst} }

    // Dispatch the fetch command.
    offset := (m.Page - 1) * panes.SearchPageSize
    fetchCmd := buildSearchPageCmd(ctx, m.Query, m.Types, offset)

    return a, tea.Batch(loadingCmd, fetchCmd)
```

`SearchOverlay` must expose a `Results() []SearchListItem` accessor for the `len == 0`
check (returns `o.results`). Add it in story 102 if not already present — for now, add a
stub that returns nil.

### `SearchPageLoadedMsg` handler — staleness check

Update (or confirm) the existing handler:

```go
case panes.SearchPageLoadedMsg:
    if m.Err != nil {
        a.searchLoading = false
        return a, a.alerts.NewAlertCmd(notifications.Warning, "Search failed: "+m.Err.Error())
    }
    // Discard stale results.
    if m.Query != a.searchQuery || m.Page != a.searchPage {
        return a, nil
    }
    a.searchLoading = false
    updated, cmd := a.searchPane.Update(m)
    a.searchPane = updated.(panes.SearchPane)
    return a, cmd
```

### `closeSearch()` — add cancellation

```go
func (a *App) closeSearch() (*App, tea.Cmd) {
    a.searchCancel()          // immediately abort in-flight HTTP
    a.searchCancel = func() {}
    a.searchQuery = ""
    a.searchPage = 0
    a.searchLoading = false
    a.searchOpen = false
    return a, nil
}
```

### `openSearch()` — reset cancel func

When reopening search, reset the cancel func so the new session starts clean:

```go
func (a *App) openSearch() (*App, tea.Cmd) {
    a.searchCancel = func() {}
    a.searchPane.Reset()
    a.searchOpen = true
    return a, a.searchPane.Init()
}
```

### `buildSearchPageCmd` — accept context

The command function signature changes (done in full in story 101):

```go
func buildSearchPageCmd(ctx context.Context, query string, types []api.SearchType, offset int) tea.Cmd
```

For this story, ensure the App handler passes the context correctly and that
`buildSearchPageCmd` compiles with the new signature. The body is updated in story 101.

## Acceptance Criteria

- [ ] `App` has `searchQuery`, `searchPage`, `searchLoading`, `searchCancel` fields
- [ ] `searchCancel` is initialized to `func(){}` in app constructor (no nil dereference)
- [ ] `SearchRequestMsg` handler: calls `searchCancel()`, sets staleness keys, creates new ctx, sends `SearchLoadingMsg`, dispatches fetch
- [ ] `SearchPageLoadedMsg` handler: discards message if `m.Query != a.searchQuery || m.Page != a.searchPage`
- [ ] `closeSearch()` calls `searchCancel()` and clears all four fields
- [ ] `openSearch()` resets `searchCancel` to `func(){}` before calling `Reset()` + `Init()`
- [ ] No nil-cancel panics under any code path
- [ ] `make ci` passes

## Tasks

- [ ] Add `searchQuery`, `searchPage`, `searchLoading`, `searchCancel` fields to `App` struct;
      initialize `searchCancel = func(){}` in constructor
      - test: `NewApp()` does not panic; calling `a.searchCancel()` immediately after construction is safe

- [ ] Rewrite `SearchRequestMsg` handler: cancel prior, record keys, create ctx, send SearchLoadingMsg + fetchCmd
      - test: send two SearchRequestMsgs in sequence; verify only one HTTP call is in-flight
        (first ctx is cancelled before second dispatch); verify `a.searchQuery` matches second msg

- [ ] Update `SearchPageLoadedMsg` handler with staleness check
      - test table:
        | m.Query | a.searchQuery | m.Page | a.searchPage | expected |
        |---|---|---|---|---|
        | "jazz" | "jazz" | 1 | 1 | forward to overlay |
        | "jazz" | "rock" | 1 | 1 | discard (return nil cmd) |
        | "jazz" | "jazz" | 2 | 1 | discard (page mismatch) |
        | "jazz" | "jazz" | 1 | 1, err set | toast, no overlay update |

- [ ] Update `closeSearch()` to call `searchCancel()` and clear all four fields
      - test: open search, dispatch request, close; verify ctx passed to buildSearchPageCmd is cancelled;
        verify `a.searchQuery == ""` and `a.searchPage == 0`

- [ ] Update `openSearch()` to reset cancel func before Reset()/Init()
      - test: open → close → reopen; no stale cancel func; `a.searchCancel` is a fresh no-op

- [ ] `make ci` passes
