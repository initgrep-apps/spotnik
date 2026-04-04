---
title: "Search Redesign: Integration tests and coverage gate"
feature: 19-search-redesign
status: open
---

## Background

Stories 97–103 implement the redesigned architecture in isolated units. This story writes
the integration and end-to-end tests that verify the pieces work together correctly, covers
the edge cases enumerated in the approved spec, and ensures `make test-coverage` passes at
≥80%.

Integration tests live in `internal/app/` and drive the full Update loop with mock HTTP
servers or stubbed commands, exercising real message routing.

## Architecture Context

### Layer: Full data flow — end-to-end verification

Each prior story had unit tests for its own component. This story tests the vertical
slice: user action → overlay intent → debounce → app handler → HTTP command → response →
overlay state. It verifies that all the independently-built pieces compose correctly.

```
[Test drives]
     │
     ▼
SearchOverlay.Update() — intent + debounce (story 99)
     │
     ▼ SearchRequestMsg
     │
     ▼
app.go handler — cancel + ctx + loading (story 100)
     │
     ├──► SearchLoadingMsg → overlay loading flags (story 102)
     │
     └──► buildSearchPageCmd (story 101) ──► mock HTTP server
                                                  │
                                                  ▼
                                            SearchPageLoadedMsg
                                                  │
                                                  ▼
                                      app.go staleness check (story 100)
                                                  │
                                                  ▼
                                      overlay results / loading state (story 102)
```

The integration tests also verify:
- The two debounce layers (BubbleTea 300ms + Gateway 100ms) each enforce last-wins
  independently
- Elm purity: no Store writes inside command closures (a regression risk from the old
  architecture)
- All edge cases from the approved spec's edge case table

### Edge cases to cover

From the approved spec:

| Scenario | Key | Query | Page | Expected |
|---|---|---|---|---|
| No query — press Next | `Ctrl+Right` | `""` | 1 | Silent no-op |
| No query — press Prev | `Ctrl+Left` | `""` | 1 | Silent no-op |
| Rapid page flipping (5×) | 5× `Ctrl+Right` | `"foo"` | 1 | Debounce settles on page 5 |
| New query typed while on page 5 | keypress | new text | 5 | page resets to 1; prior cancelled |
| Esc while loading first page | Esc | `"foo"` | 1 | searchCancel() kills HTTP; no stale result |
| API returns 0 results | response | `"foo"` | 1 | total=0; "No results"; bar hidden |
| API error on first page | error | `"foo"` | 1 | Toast; loadingFirstPage=false; hint shown |
| API error on subsequent page | error | `"foo"` | N>1 | Toast; loadingNextPage=false; page N-1 results visible |
| Context cancelled (new query) | new keypress | new | any | nil returned; no state change |
| Ctrl+U while on page 5 | Ctrl+U | any | 5 | intent={query:"", tab:current, page:1} |
| Stale result after Esc | timing | old | old | searchQuery=="" → discarded |

## Design

### Test: SearchRequestMsg handler — cancel + dispatch

```go
func TestApp_SearchRequestMsg_CancelsPrior(t *testing.T) {
    // Create app with a mock search command that records ctx cancellation.
    // Send SearchRequestMsg{Query:"jazz", Page:1}.
    // Record the ctx passed to buildSearchPageCmd.
    // Send SearchRequestMsg{Query:"rock", Page:1}.
    // Assert: first ctx.Err() == context.Canceled.
    // Assert: a.searchQuery == "rock".
    // Assert: a.searchLoading == true.
}
```

### Test: closeSearch — cancellation and field clearing

```go
func TestApp_CloseSearch_CancelsAndClears(t *testing.T) {
    // Open search, dispatch SearchRequestMsg.
    // Record ctx. Call closeSearch().
    // Assert: ctx.Err() == context.Canceled.
    // Assert: a.searchQuery == "", a.searchPage == 0, a.searchLoading == false.
    // Assert: a.searchOpen == false.
}
```

### Test: SearchPageLoadedMsg staleness check (table-driven)

| m.Query | a.searchQuery | m.Page | a.searchPage | m.Err | forwarded to overlay? |
|---|---|---|---|---|---|
| "jazz" | "jazz" | 1 | 1 | nil | yes |
| "jazz" | "rock" | 1 | 1 | nil | no |
| "jazz" | "jazz" | 2 | 1 | nil | no |
| "jazz" | "jazz" | 1 | 1 | someErr | yes (loading flags cleared; toast sent) |

### Test: Stale result after closeSearch

```go
func TestApp_SearchPageLoadedMsg_DiscardedAfterClose(t *testing.T) {
    // Open search. Dispatch request (searchQuery="jazz", searchPage=1).
    // closeSearch() → searchQuery="" searchPage=0.
    // Send SearchPageLoadedMsg{Query:"jazz", Page:1, Results:[...]}.
    // Assert: overlay.Results() still nil (message discarded).
}
```

### Test: API error preserves previous results + clears loading flags

```go
func TestApp_SearchPageLoadedMsg_ErrorPreservesResults(t *testing.T) {
    // Load page 1 successfully → results=[10 items], total=50.
    // Ctrl+Right → page 2 request → loadingNextPage=true.
    // Handle SearchPageLoadedMsg{Query:"jazz", Page:2, Err: someErr}.
    // Assert: toast emitted.
    // Assert: overlay.Results() still [10 items from page 1].
    // Assert: overlay.loadingNextPage == false.
    // Assert: overlay.loadingFirstPage == false.
}
```

### Test: Rapid page flipping — single HTTP call

```go
func TestSearchOverlay_RapidPageFlip_SingleRequest(t *testing.T) {
    // Construct overlay with query="jazz", page=1, total=50.
    // Simulate Ctrl+Right pressed 5 times in rapid succession.
    // Collect all tea.Cmds returned.
    // Run the last cmd's tick immediately; call handleDebounce.
    // Assert: exactly one SearchRequestMsg emitted with Page=6.
    // Prior ticks are stale (intent.page moved on) → produce no msg.
}
```

### Test: No-query guard — pagination no-op

```go
func TestSearchOverlay_NoQuery_PaginationNoOp(t *testing.T) {
    // Overlay with empty query (intent.query="").
    // Send ctrl+right key msg.
    // Assert: no SearchRequestMsg emitted; intent.page still 1.
}
```

### Test: Gateway Interactive debounce — end-to-end

```go
func TestGateway_InteractiveDebounce_LastWins(t *testing.T) {
    // Use httptest.NewServer to record which requests arrive.
    // Dispatch two Interactive Do() calls for same path within 10ms.
    // Wait 200ms.
    // Assert: exactly 1 HTTP request reached the server (the second one).
}

func TestGateway_InteractiveDebounce_DifferentPathsIndependent(t *testing.T) {
    // Dispatch Interactive Do() calls for /v1/search and /v1/me/player/devices simultaneously.
    // Assert: both HTTP requests reach the server.
}

func TestGateway_Background_NoDebounce(t *testing.T) {
    // Dispatch two Background Do() calls for same path within 10ms.
    // Assert: both HTTP requests reach the server (Background bypasses debounce).
}
```

### Test: All tab — total is max across all types

```go
func TestConvertSearchResult_AllTab_TotalIsMax(t *testing.T) {
    // tracks.Total=100, artists.Total=50, albums.Total=30, playlists.Total=20.
    // Assert: total == 100.
    // tracks.Total=0, artists.Total=0, albums.Total=0, playlists.Total=0.
    // Assert: total == 0.
    // tracks.Total=7 (Songs tab, others zero).
    // Assert: total == 7.
}

func TestSearchOverlay_AllTab_HasNextPage(t *testing.T) {
    // total=100, SearchPageSize=10: hasNextPage at page 1 → true.
    // total=100, page 10 → false.
    // total=10, page 1 → false (exactly one page).
}
```

### Test: Full search flow (open → type → results → paginate → close)

```go
func TestApp_SearchFlow_OpenTypeResultsPaginateClose(t *testing.T) {
    // 1. openSearch() → overlay fresh state (results=nil, page=1)
    // 2. Simulate typing "jazz" → scheduleDebounce → handleDebounce → SearchRequestMsg
    // 3. Handle SearchRequestMsg → SearchLoadingMsg sent, fetchCmd dispatched
    // 4. Handle SearchLoadingMsg on overlay → loadingFirstPage=true
    // 5. Mock SearchPageLoadedMsg{Query:"jazz", Page:1, Results:[10 items], Total:50}
    // 6. Handle on app → forward to overlay → loadingFirstPage=false, results=[10 items]
    // 7. Ctrl+Right → intent.page=2 → scheduleDebounce → SearchRequestMsg{Page:2}
    // 8. Handle SearchRequestMsg → SearchLoadingMsg{IsFirstPage:false}
    //    → loadingNextPage=true; results still [10 items from page 1]
    // 9. closeSearch() → searchCancel() called; searchQuery="" searchPage=0
}
```

### Test: Elm purity — no Store writes inside command closures

```go
func TestElmPurity_NoStoreWritesInCommandClosures(t *testing.T) {
    // Verify that buildSearchPageCmd's returned func() tea.Msg does not
    // call any store.Set* / store.Append* / store.Clear* methods.
    // This is a static/structural check: grep internal/app/commands.go
    // for store.* calls inside func() tea.Msg closures and assert zero hits.
    // Alternatively: run the command with a store spy; assert no calls recorded.
}
```

### Coverage

After all tests, run `make test-coverage`. Target: ≥80% across all packages.

Packages most likely to need additional test coverage:
- `internal/api/` — `interactiveDebounce`, new `buildSearchPageCmd` ctx-cancel path
- `internal/ui/panes/` — `hasNextPage`, `renderPaginationBar`, loading-state rendering,
  all guard conditions, error-preserves-results path
- `internal/app/` — staleness check paths, `closeSearch`/`openSearch` cancellation,
  error-forwards-to-overlay path

## Acceptance Criteria

- [ ] `TestApp_SearchRequestMsg_CancelsPrior` passes
- [ ] `TestApp_CloseSearch_CancelsAndClears` passes
- [ ] `SearchPageLoadedMsg` staleness table test passes (all 4 rows including error row)
- [ ] `TestApp_SearchPageLoadedMsg_DiscardedAfterClose` passes
- [ ] `TestApp_SearchPageLoadedMsg_ErrorPreservesResults` passes (loading flags cleared + results kept)
- [ ] `TestSearchOverlay_RapidPageFlip_SingleRequest` passes
- [ ] `TestSearchOverlay_NoQuery_PaginationNoOp` passes
- [ ] `TestGateway_InteractiveDebounce_LastWins` passes
- [ ] `TestGateway_InteractiveDebounce_DifferentPathsIndependent` passes
- [ ] `TestGateway_Background_NoDebounce` passes
- [ ] `TestConvertSearchResult_AllTab_TotalIsMax` passes
- [ ] `TestSearchOverlay_AllTab_HasNextPage` passes
- [ ] `TestApp_SearchFlow_OpenTypeResultsPaginateClose` passes
- [ ] Elm purity check passes (zero store writes in command closures)
- [ ] `make test-coverage` passes at ≥80%
- [ ] `make ci` passes (lint + tests + coverage)

## Tasks

- [ ] Write `TestApp_SearchRequestMsg_CancelsPrior` and `TestApp_CloseSearch_CancelsAndClears`
      in `internal/app/search_test.go` (create file if needed)
      - verify ctx cancellation and field clearing

- [ ] Write `SearchPageLoadedMsg` staleness table test (4 rows) + discard-after-close test
      + error-preserves-results test
      - verify overlay receives results only on exact query+page match
      - verify loading flags cleared on error; previous results preserved

- [ ] Write `TestSearchOverlay_RapidPageFlip_SingleRequest` and no-query guard tests
      in `internal/ui/panes/search_test.go`
      - verify debounce settles on last intent; prior ticks discard
      - cover all guard conditions for Ctrl+Right / Ctrl+Left

- [ ] Write All-tab tests: `TestConvertSearchResult_AllTab_TotalIsMax` and
      `TestSearchOverlay_AllTab_HasNextPage` in their respective packages

- [ ] Write Gateway Interactive debounce tests (last-wins, independent paths, Background bypass)
      in `internal/api/gateway_test.go` using `httptest.NewServer`

- [ ] Write full flow integration test `TestApp_SearchFlow_OpenTypeResultsPaginateClose`

- [ ] Write Elm purity check for `buildSearchPageCmd` closure

- [ ] Run `make test-coverage`; identify and fill any gaps below 80% in affected packages

- [ ] `make ci` passes (all tests green, lint clean, coverage ≥80%)
