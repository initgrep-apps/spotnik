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

Integration tests live in `internal/app/` and drive the full Update loop with mock
HTTP servers or stubbed commands, exercising real message routing.

## Design

### Test: SearchRequestMsg handler — cancel + dispatch

Verify that sending two `SearchRequestMsg` in sequence:
1. Cancels the context from the first before dispatching the second
2. Sets `a.searchQuery` and `a.searchPage` to the second msg's values
3. `a.searchLoading == true` after the second dispatch

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

### Test: SearchPageLoadedMsg staleness check

Table-driven:

| m.Query | a.searchQuery | m.Page | a.searchPage | forwarded to overlay? |
|---|---|---|---|---|
| "jazz" | "jazz" | 1 | 1 | yes |
| "jazz" | "rock" | 1 | 1 | no |
| "jazz" | "jazz" | 2 | 1 | no |
| "jazz" | "jazz" | 1 | 1 (err set) | no — toast only |

### Test: Stale result after closeSearch

```go
func TestApp_SearchPageLoadedMsg_DiscardedAfterClose(t *testing.T) {
    // Open search. Dispatch request (searchQuery="jazz", searchPage=1).
    // closeSearch() → searchQuery="" searchPage=0.
    // Send SearchPageLoadedMsg{Query:"jazz", Page:1, Results:[...]}.
    // Assert: overlay.Results() still nil (message discarded).
}
```

### Test: Rapid page flipping — single HTTP call

```go
func TestSearchOverlay_RapidPageFlip_SingleRequest(t *testing.T) {
    // Construct overlay with query="jazz", page=1, total=50.
    // Simulate Ctrl+Right pressed 3 times in rapid succession (< 300ms apart in test time).
    // Collect all tea.Cmds returned.
    // Run the last cmd's tick immediately; call handleDebounce.
    // Assert: exactly one SearchRequestMsg emitted with Page=4.
    // The prior ticks are stale (intent.page moved on) and produce no msg.
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

### Test: Full search flow (open → type → results → paginate → close)

```go
func TestApp_SearchFlow_OpenTypeResultsPaginateClose(t *testing.T) {
    // 1. openSearch() → overlay in fresh state (results=nil, page=1)
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

### Test: API error preserves previous results

```go
func TestApp_SearchPageLoadedMsg_ErrorPreservesResults(t *testing.T) {
    // Load page 1 successfully → results=[10 items], total=50.
    // Ctrl+Right → page 2 request.
    // Handle SearchPageLoadedMsg{Query:"jazz", Page:2, Err: someErr}.
    // Assert: toast emitted; overlay.Results() still [10 items] (page 1).
    // Assert: overlay.loadingNextPage == false.
}
```

### Coverage

After all tests, run `make test-coverage`. Target: ≥80% across all packages.

Packages most likely to need additional test coverage:
- `internal/api/` — `interactiveDebounce`, new `buildSearchPageCmd` ctx-cancel path
- `internal/ui/panes/` — `hasNextPage`, `renderPaginationBar`, loading-state rendering, all guard conditions
- `internal/app/` — staleness check paths, `closeSearch`/`openSearch` cancellation

## Acceptance Criteria

- [ ] `TestApp_SearchRequestMsg_CancelsPrior` passes
- [ ] `TestApp_CloseSearch_CancelsAndClears` passes
- [ ] `SearchPageLoadedMsg` staleness table test passes (all 4 rows)
- [ ] `TestApp_SearchPageLoadedMsg_DiscardedAfterClose` passes
- [ ] `TestSearchOverlay_RapidPageFlip_SingleRequest` passes
- [ ] `TestSearchOverlay_NoQuery_PaginationNoOp` passes
- [ ] `TestGateway_InteractiveDebounce_LastWins` passes
- [ ] `TestGateway_InteractiveDebounce_DifferentPathsIndependent` passes
- [ ] `TestGateway_Background_NoDebounce` passes
- [ ] `TestApp_SearchFlow_OpenTypeResultsPaginateClose` passes
- [ ] `TestApp_SearchPageLoadedMsg_ErrorPreservesResults` passes
- [ ] `make test-coverage` passes at ≥80%
- [ ] `make ci` passes (lint + tests + coverage)

## Tasks

- [ ] Write `TestApp_SearchRequestMsg_CancelsPrior` and `TestApp_CloseSearch_CancelsAndClears`
      in `internal/app/search_test.go` (create file if needed)
      - verify ctx cancellation and field clearing

- [ ] Write `SearchPageLoadedMsg` staleness table test (4 rows) + discard-after-close test
      - verify overlay receives results only on exact query+page match

- [ ] Write `TestSearchOverlay_RapidPageFlip_SingleRequest` in `internal/ui/panes/search_test.go`
      - verify debounce settles on last intent; prior ticks discard

- [ ] Write no-query pagination no-op test and all guard condition tests for Ctrl+Right/Ctrl+Left
      - cover: no query, first-page prev, last-page next, loading-first-page next

- [ ] Write Gateway Interactive debounce tests (last-wins, independent paths, Background bypass)
      in `internal/api/gateway_test.go` using `httptest.NewServer`

- [ ] Write full flow integration test `TestApp_SearchFlow_OpenTypeResultsPaginateClose`

- [ ] Write `TestApp_SearchPageLoadedMsg_ErrorPreservesResults`

- [ ] Run `make test-coverage`; identify and fill any gaps below 80% in affected packages

- [ ] `make ci` passes (all tests green, lint clean, coverage ≥80%)
