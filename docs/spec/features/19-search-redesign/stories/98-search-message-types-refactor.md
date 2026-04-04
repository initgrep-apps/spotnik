---
title: "Search Redesign: Refactor search message types"
feature: 19-search-redesign
status: open
---

## Background

The current message types were designed for the prefetch-batch architecture:
- `SearchPageLoadedMsg` carries per-type totals (`TracksTotal`, `ArtistsTotal`, etc.) for
  the 5-page batch chain and has no `Page` field
- `SearchPrefetchMsg` drives the chain-through-Update prefetch engine
- `SearchTabChangedMsg` triggers an immediate store-clear + re-batch on tab change
- No loading-state message exists — app.go sets `store.SetSearchLoading(true)` synchronously

The new architecture needs:
- `SearchPageLoadedMsg` with a `Page` field (staleness key) and a flat `Total int`
- `SearchLoadingMsg{IsFirstPage bool}` — sent from app.go to overlay before HTTP dispatch
- `SearchPrefetchMsg` and `SearchTabChangedMsg` deleted entirely

This story updates `internal/app/messages.go` (or wherever message types live). Story 99
wires `SearchTabChangedMsg` removal into the overlay; story 100 wires `SearchLoadingMsg`
dispatch into app.go.

## Design

### `SearchPageLoadedMsg` — updated shape

```go
// SearchPageLoadedMsg is sent when a single page of search results has loaded.
// Query and Page are staleness keys — app.go discards this message if either
// does not match the current a.searchQuery / a.searchPage.
type SearchPageLoadedMsg struct {
    Query   string
    Page    int              // 1-based page number; used as staleness key
    Results []SearchListItem // current page items (max SearchPageSize=10)
    Total   int              // total results across all types/pages (for pagination bar)
    Err     error
}
```

**Deleted fields** (remove from existing struct):
- `TracksTotal int`
- `ArtistsTotal int`
- `AlbumsTotal int`
- `PlaylistsTotal int`
- Any `NextOffset int` or `BatchEnd int` fields used by the prefetch chain

**Added fields:**
- `Page int`
- `Total int` (flat, replaces per-type totals)

### `SearchLoadingMsg` — new

```go
// SearchLoadingMsg is sent by app.go to the search overlay immediately before
// dispatching a new HTTP request. IsFirstPage=true means results==nil (spinner
// only); IsFirstPage=false means previous results are still visible (spinner
// line above list).
type SearchLoadingMsg struct {
    IsFirstPage bool
}
```

### Deleted message types

Remove entirely from `messages.go`:
- `SearchPrefetchMsg` — prefetch engine is gone
- `SearchTabChangedMsg` — tab changes now go through universal debounce (story 99)

### Handler cleanup in `app.go`

After deleting `SearchPrefetchMsg` and `SearchTabChangedMsg`, remove their `case` blocks
from the `handleMsg` switch in `internal/app/app.go`:

```go
// DELETE these entire case blocks:
case panes.SearchPrefetchMsg:
    ...
case panes.SearchTabChangedMsg:
    ...
```

Update the `SearchPageLoadedMsg` handler to use the new fields. The full handler is
rewritten in story 100 — for now, update it to compile (remove references to
`TracksTotal`/`ArtistsTotal` etc. that no longer exist, replace with `m.Total`).

### `SearchRequestMsg` — add `Page` field

```go
// SearchRequestMsg is sent by the overlay's handleDebounce to request a new
// search. App.go handles this message.
type SearchRequestMsg struct {
    Query string
    Types []api.SearchType // nil = all types
    Page  int              // 1-based page number
}
```

Add `Page int` field. Update the one place that constructs this message
(overlay's `handleDebounce`, cleaned up in story 99).

## Acceptance Criteria

- [ ] `SearchPageLoadedMsg` has `Page int` and flat `Total int`; per-type total fields removed
- [ ] `SearchLoadingMsg{IsFirstPage bool}` exists in `messages.go`
- [ ] `SearchPrefetchMsg` and `SearchTabChangedMsg` are deleted from `messages.go`
- [ ] Their `case` handlers are removed from `app.go` `handleMsg` switch
- [ ] `SearchRequestMsg` has `Page int` field
- [ ] `make build` compiles
- [ ] `make lint` passes

## Tasks

- [ ] Update `SearchPageLoadedMsg`: add `Page int`, `Total int`; delete per-type total fields
      - fix all construction sites (`buildSearchPageCmd`, any test helpers)
      - test: `make build` compiles; grep confirms no references to deleted fields

- [ ] Add `SearchLoadingMsg` to `messages.go`
      - test: struct exists and compiles; `IsFirstPage bool` field present

- [ ] Delete `SearchPrefetchMsg` and `SearchTabChangedMsg` from `messages.go`
      - remove their `case` blocks from `app.go` `handleMsg`
      - test: `make build` compiles; grep for `SearchPrefetchMsg` and `SearchTabChangedMsg` returns zero hits

- [ ] Add `Page int` to `SearchRequestMsg`
      - update construction site in overlay's `handleDebounce` to pass `Page: o.intent.page`
        (intent.page may be zero until story 99 adds it — use `1` as default)
      - test: `make build` compiles

- [ ] `make ci` passes
