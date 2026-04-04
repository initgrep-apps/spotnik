---
title: "Search Redesign: Pagination controls, loading states, and Panel 2 view"
feature: 19-search-redesign
status: open
---

## Background

After the debounce, cancellation, and commands stories (99–101), the overlay needs the
user-facing parts of the new architecture:

1. **`Ctrl+Right` / `Ctrl+Left` keybindings** — intercept at overlay Update before textinput
2. **`hasNextPage()` / page guard** — enforce pagination bounds silently
3. **Two loading states** — `loadingFirstPage` (spinner only) vs `loadingNextPage`
   (spinner line + existing results + pagination bar)
4. **Pagination bar** — `[ ←  page N of M  → ]` fixed at bottom of Panel 2
5. **`resizeList()` adjustment** — subtract 1 line for pagination bar when total > 0

All edge cases from the approved spec must be handled: no-query no-op, last-page no-op,
first-page prev no-op, rapid paging settled by debounce, Ctrl+U reset.

## Design

### New fields on `SearchOverlay`

```go
results          []SearchListItem // current page; nil = no results yet
total            int              // for hasNextPage() and pagination bar
loadingFirstPage bool             // results==nil, fetch in-flight → spinner only
loadingNextPage  bool             // results!=nil, fetch in-flight → list + spinner
```

Remove the old `results *SearchResultData` field (store-backed). These fields are owned
by the overlay directly.

Add `Results() []SearchListItem` accessor:
```go
func (o *SearchOverlay) Results() []SearchListItem { return o.results }
```

### `SearchLoadingMsg` handler on overlay

```go
case SearchLoadingMsg:
    if m.IsFirstPage {
        o.loadingFirstPage = true
        o.loadingNextPage = false
    } else {
        o.loadingFirstPage = false
        o.loadingNextPage = true
    }
    return *o, nil
```

### `SearchPageLoadedMsg` handler on overlay

```go
case SearchPageLoadedMsg:
    // Err is handled by app.go (toast); overlay only sees success msgs forwarded by app.
    o.loadingFirstPage = false
    o.loadingNextPage = false
    o.results = m.Results
    o.total = m.Total
    o.rebuildListItems()
    return *o, nil
```

`rebuildListItems()` calls `o.resultList.SetItems(...)` with the new results converted to
`list.Item` values.

### `hasNextPage()` method

```go
func (o *SearchOverlay) hasNextPage() bool {
    return o.total > 0 && o.intent.page*SearchPageSize < o.total
}
```

### `Ctrl+Right` / `Ctrl+Left` keybindings

Add to `searchKeyMap`:
```go
nextPage key.Binding
prevPage key.Binding
```

Bind in `newSearchKeyMap()`:
```go
nextPage: key.NewBinding(
    key.WithKeys("ctrl+right"),
    key.WithHelp("ctrl+→", "next page"),
),
prevPage: key.NewBinding(
    key.WithKeys("ctrl+left"),
    key.WithHelp("ctrl+←", "prev page"),
),
```

**Guard conditions (silent no-op for all):**

| Key | Guard | Action |
|---|---|---|
| `Ctrl+Right` | `o.intent.query == ""` | no-op |
| `Ctrl+Right` | `o.loadingFirstPage` | no-op |
| `Ctrl+Right` | `!o.hasNextPage()` | no-op |
| `Ctrl+Right` | none of above | `o.intent.page++; return o, o.scheduleDebounce()` |
| `Ctrl+Left` | `o.intent.query == ""` | no-op |
| `Ctrl+Left` | `o.loadingFirstPage` | no-op |
| `Ctrl+Left` | `o.intent.page <= 1` | no-op |
| `Ctrl+Left` | none of above | `o.intent.page--; return o, o.scheduleDebounce()` |

Intercept these keys in `Update()` **before** forwarding the key message to `o.input`.

### Pagination bar — `renderPaginationBar`

```go
// renderPaginationBar renders the [ ←  page N of M  → ] line.
// prev/next arrows are dimmed (TextMuted) when navigation is not possible.
func (o *SearchOverlay) renderPaginationBar(w int) string {
    totalPages := (o.total + SearchPageSize - 1) / SearchPageSize
    if totalPages == 0 {
        totalPages = 1
    }
    center := fmt.Sprintf("  page %d of %d  ", o.intent.page, totalPages)

    prevStyle := o.theme.Text()      // active
    nextStyle := o.theme.Text()      // active
    if o.intent.page <= 1 {
        prevStyle = o.theme.TextMuted()
    }
    if !o.hasNextPage() {
        nextStyle = o.theme.TextMuted()
    }

    left  := prevStyle.Render("[ ←")
    right := nextStyle.Render("→ ]")
    bar   := lipgloss.JoinHorizontal(lipgloss.Center, left, center, right)
    return lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(bar)
}
```

### Panel 2 layout — top to bottom

```
tab bar        (1 line)
separator      (1 line)
spinner line   (0 or 1 line, loadingNextPage only)
list           (fills remaining height)
pagination bar (1 line, only when total > 0)
```

`resizeList()` must subtract 1 extra line from list height when `total > 0`:
```go
paginationLine := 0
if o.total > 0 {
    paginationLine = 1
}
listH := availableH - tabBarH - separatorH - spinnerLineH - paginationLine
```

### Loading state rendering rules

| State | `loadingFirstPage` | `loadingNextPage` | Panel 2 content |
|---|---|---|---|
| No query | false | false | Hint text: `"Type to search"` |
| First fetch in-flight | true | false | Centered spinner: `"◉ Searching…"` |
| Results stable | false | false | List + pagination bar |
| Page change in-flight | false | true | Spinner line above list + list + pagination bar |
| Error (handled by app.go) | false | false | Previous results visible if any; otherwise hint text |

### `Ctrl+U` (clear input) — reset page

When `Ctrl+U` clears the input, also reset `o.intent.page = 1` and `o.intent.query = ""`.
No scheduleDebounce — clearing the input should not fire a search for empty string.

## Acceptance Criteria

- [ ] `loadingFirstPage` and `loadingNextPage` fields exist on `SearchOverlay`
- [ ] `Results() []SearchListItem` accessor exists
- [ ] `SearchLoadingMsg` handler sets the correct loading flag
- [ ] `SearchPageLoadedMsg` handler clears both loading flags, sets `results` and `total`, calls `rebuildListItems()`
- [ ] `hasNextPage()` correctly handles: total=0, total=10 page=1, total=11 page=1, total=10 page=2
- [ ] `Ctrl+Right` / `Ctrl+Left` keybindings added to `searchKeyMap` and visible in `ShortHelp()`
- [ ] All guard conditions for `Ctrl+Right` / `Ctrl+Left` produce silent no-ops
- [ ] Pagination bar renders with dimmed `[←` on page 1 and dimmed `→]` on last page
- [ ] `resizeList()` subtracts 1 line for pagination bar when `total > 0`
- [ ] `Ctrl+U` resets `intent.page = 1` and `intent.query = ""`
- [ ] `make ci` passes

## Tasks

- [ ] Add `results []SearchListItem`, `total int`, `loadingFirstPage bool`, `loadingNextPage bool`
      to `SearchOverlay`; add `Results()` accessor; remove old `results *SearchResultData`
      - test: fields zero-valued on construction; Results() returns nil initially

- [ ] Handle `SearchLoadingMsg` and `SearchPageLoadedMsg` in overlay Update()
      - test (SearchLoadingMsg): IsFirstPage=true → loadingFirstPage=true, loadingNextPage=false;
        IsFirstPage=false → loadingFirstPage=false, loadingNextPage=true
      - test (SearchPageLoadedMsg): both loading flags false; results == m.Results; total == m.Total

- [ ] Implement `hasNextPage() bool`
      - test table: {total:0,page:1}→false; {total:10,page:1}→false; {total:11,page:1}→true;
        {total:100,page:10}→false; {total:100,page:9}→true

- [ ] Add `nextPage`/`prevPage` bindings to `searchKeyMap`; handle in `Update()` with all guard conditions
      - test: no query + Ctrl+Right → no SearchRequestMsg; on last page + Ctrl+Right → no SearchRequestMsg;
        on page 1 + Ctrl+Left → no SearchRequestMsg; loadingFirstPage + Ctrl+Right → no SearchRequestMsg;
        valid next → intent.page++, scheduleDebounce cmd returned;
        valid prev → intent.page--, scheduleDebounce cmd returned

- [ ] Implement `renderPaginationBar(w int) string`; integrate into Panel 2 View; update `resizeList()`
      - test: page=1 → prev arrow uses TextMuted style; last page → next arrow uses TextMuted style;
        mid page → both arrows use Text style; total=0 → bar not rendered

- [ ] Update `Ctrl+U` handler to reset `intent.page = 1` and `intent.query = ""`
      - test: on page 5 with query "jazz", press Ctrl+U → `o.intent == {query:"", tab:current, page:1}`

- [ ] Update `Reset()` to zero `results`, `total`, `loadingFirstPage`, `loadingNextPage`
      - test: set all fields to non-zero values, call Reset(), assert all zero/nil/false

- [ ] `make ci` passes
