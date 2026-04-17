---
title: "Search Redesign: Pagination controls, loading states, and Panel 2 view"
feature: 19-search-redesign
status: done
---

## Background

After the debounce (story 99), cancellation (story 100), and commands (story 101) stories,
the overlay needs the user-facing parts of the new architecture:

1. **Overlay-owned result fields** — `results []SearchListItem`, `total int`,
   `loadingFirstPage bool`, `loadingNextPage bool` (replacing store-backed state)
2. **`Ctrl+Right` / `Ctrl+Left` keybindings** — intercept before textinput
3. **`hasNextPage()` / page guards** — enforce pagination bounds silently
4. **Two loading states** — `loadingFirstPage` (spinner only) vs `loadingNextPage`
   (spinner line + existing results)
5. **Pagination bar** — `[ ←  page N of M  → ]` fixed at bottom of Panel 2
6. **`resizeList()` adjustment** — subtract 1 line for pagination bar when `total > 0`

All edge cases from the approved spec must be handled: no-query no-op, last-page no-op,
first-page prev no-op, rapid paging settled by debounce, Ctrl+U reset, error preserving
previous results.

## Architecture Context

### Layer: SearchOverlay — display state and View rendering

This story is the final "output" half of the overlay. Stories 99–101 built the input
side (intent → request → HTTP). This story wires the response back into the overlay's
own state and drives the View.

```
SearchLoadingMsg{IsFirstPage}              ← dispatched by app.go (story 100)
  │
  ▼
SearchOverlay.Update(SearchLoadingMsg)     ← THIS STORY handles this
  │
  └──► loadingFirstPage = true / loadingNextPage = true

                        ...HTTP in flight...

SearchPageLoadedMsg{Query, Page, Results, Total, Err}
  │
  ▼
app.go: staleness check (story 100)
  │  success → forward to overlay
  │  error   → forward to overlay + send toast
  ▼
SearchOverlay.Update(SearchPageLoadedMsg)  ← THIS STORY handles both success and error
  │
  ├── always: loadingFirstPage=false, loadingNextPage=false   (clear spinners)
  │
  ├── Err != nil  → keep existing results; return (loading flags cleared, error toast in app.go)
  │
  └── Err == nil  → o.results = m.Results; o.total = m.Total; rebuildListItems()
```

### State machine — complete overlay states

```
                  Ctrl+Right / Ctrl+Left (story 99 mechanism, wired here)
     ┌──────────────────────────────────────────────────────────────────┐
     │                                                                  │
   Empty ──── keypress ────► Typing                                     │
     ▲                           │                                      │
     │            debounce fires │                                      │
     │            (query == "")  ▼                                      │
     │                 no-op ──► Empty                                  │
     │                           │                                      │
     │            debounce fires │                                      │
     │            (query != "")  ▼                                      │
     │                     LoadingFirst  ◄──── SearchLoadingMsg(first)  │
     │                     (spinner)           loadingFirstPage=true    │
     │                           │                                      │
     │               results arrive (success)                           │
     │                           ▼                                      │
     │                       Results ◄─────────────────────────────────-┘
     │                      (list + bar)
     │                           │
     │          Ctrl+Right/Left  │
     │                           ▼
     │                     LoadingNext  ◄─── SearchLoadingMsg(next)
     │                  (spinner + list)      loadingNextPage=true
     │                           │
     │               results arrive (success)
     │                           └──────────► Results
     │
     │               results arrive (error)  → clear loading flags, keep prior results
     │
     └──── Ctrl+U (clear) ──────────────────────────────────────────── Empty
     └──── Esc ─────────────────────────────────────────────────────── Closed
```

### Removing `results *SearchResultData`

This story removes the old `results *SearchResultData` field that has been left as a
stub since story 97, and replaces it with the direct `results []SearchListItem` field.
All `o.results.Tracks` / `o.results.Artists` etc. accesses in `View()` are replaced by
reading `o.results` (flat slice) and `o.total` directly.

## Design

### New fields on `SearchOverlay`

Remove `results *SearchResultData`. Add:

```go
results          []SearchListItem // current page; nil = no results yet
total            int              // for hasNextPage() and pagination bar
loadingFirstPage bool             // results==nil, fetch in-flight → spinner only
loadingNextPage  bool             // results!=nil, fetch in-flight → list + spinner
```

### `Results()` accessor — complete the stub from story 100

```go
// Results returns the current page of search results.
// Returns nil until the first successful search response arrives.
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

The overlay always clears loading flags first, regardless of error. When an error
occurred, app.go already sent a toast — the overlay only needs to stop showing the spinner
and preserve whatever results were visible before.

```go
case SearchPageLoadedMsg:
    // Always clear loading flags — the spinner must not stay visible after
    // any response (success or error). App.go handles the error toast.
    o.loadingFirstPage = false
    o.loadingNextPage = false
    if m.Err != nil {
        // Keep existing results visible (previous page preserved on page-change error).
        return *o, nil
    }
    o.results = m.Results
    o.total = m.Total
    o.rebuildListItems()
    return *o, nil
```

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

**Guard conditions (all produce silent no-op):**

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
// Arrows are dimmed (TextMuted) when navigation in that direction is not possible.
func (o *SearchOverlay) renderPaginationBar(w int) string {
    totalPages := (o.total + SearchPageSize - 1) / SearchPageSize
    if totalPages == 0 {
        totalPages = 1
    }
    center := fmt.Sprintf("  page %d of %d  ", o.intent.page, totalPages)

    prevStyle := o.theme.Text()
    nextStyle := o.theme.Text()
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

### `resizeList()` adjustment

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
| Error | false | false | Previous results visible if any; otherwise hint text |

### `Ctrl+U` (clear input) — reset page

When `Ctrl+U` clears the input, also reset `o.intent.page = 1` and `o.intent.query = ""`.
No `scheduleDebounce` — clearing the input must not fire a search for the empty string.

### `Reset()` — zero all new fields

```go
func (o *SearchOverlay) Reset() {
    o.intent = searchIntent{query: "", tab: TabAll, page: 1}
    o.results = nil
    o.total = 0
    o.loadingFirstPage = false
    o.loadingNextPage = false
    // ... rest of existing reset logic
}
```

## Acceptance Criteria

- [ ] `results *SearchResultData` is removed; `results []SearchListItem`, `total int`,
      `loadingFirstPage bool`, `loadingNextPage bool` are added
- [ ] `Results() []SearchListItem` accessor returns `o.results` (completes the story 100 stub)
- [ ] `SearchLoadingMsg` handler sets the correct loading flag; clears the other
- [ ] `SearchPageLoadedMsg` handler: always clears both loading flags first; on error, keeps existing results; on success, updates results + total + rebuildListItems
- [ ] `hasNextPage()` correctly handles: total=0, total=10/page=1, total=11/page=1, total=100/page=10
- [ ] `Ctrl+Right` / `Ctrl+Left` keybindings added to `searchKeyMap` and visible in `ShortHelp()`
- [ ] All guard conditions for `Ctrl+Right` / `Ctrl+Left` produce silent no-ops
- [ ] Pagination bar renders with dimmed `[ ←` on page 1 and dimmed `→ ]` on last page
- [ ] `resizeList()` subtracts 1 line for pagination bar when `total > 0`
- [ ] `Ctrl+U` resets `intent.page = 1` and `intent.query = ""`
- [ ] `Reset()` zeros `results`, `total`, `loadingFirstPage`, `loadingNextPage`
- [ ] `make ci` passes

## Tasks

- [ ] Remove `results *SearchResultData`; add `results []SearchListItem`, `total int`,
      `loadingFirstPage bool`, `loadingNextPage bool`; complete `Results()` accessor
      - test: fields zero-valued on construction; `Results()` returns nil initially;
        after `Reset()`, all fields zero/nil/false

- [ ] Handle `SearchLoadingMsg` in overlay `Update()`
      - test: `IsFirstPage=true` → `loadingFirstPage=true`, `loadingNextPage=false`;
        `IsFirstPage=false` → `loadingFirstPage=false`, `loadingNextPage=true`

- [ ] Handle `SearchPageLoadedMsg` in overlay `Update()` — success and error branches
      - test (success): both loading flags false; `results == m.Results`; `total == m.Total`
      - test (error): both loading flags false; existing results preserved; `total` unchanged

- [ ] Implement `hasNextPage() bool`
      - test table: `{total:0,page:1}`→false; `{total:10,page:1}`→false (exactly one page);
        `{total:11,page:1}`→true; `{total:100,page:10}`→false; `{total:100,page:9}`→true

- [ ] Add `nextPage`/`prevPage` bindings to `searchKeyMap`; handle in `Update()` with all guard
      conditions; intercept before forwarding to `o.input`
      - test: no query + `Ctrl+Right` → no `SearchRequestMsg`; on last page + `Ctrl+Right` → no-op;
        on page 1 + `Ctrl+Left` → no-op; `loadingFirstPage` + `Ctrl+Right` → no-op;
        valid next → `intent.page++`, `scheduleDebounce` cmd returned;
        valid prev → `intent.page--`, `scheduleDebounce` cmd returned

- [ ] Implement `renderPaginationBar(w int) string`; integrate into Panel 2 View;
      update `resizeList()`
      - test: page=1 → prev arrow uses `TextMuted` style; last page → next arrow uses `TextMuted`;
        mid page → both arrows use `Text` style; `total=0` → bar not rendered

- [ ] Update `Ctrl+U` handler to reset `intent.page = 1` and `intent.query = ""`
      - test: on page 5 with query "jazz", press `Ctrl+U` →
        `o.intent == {query:"", tab:current, page:1}`

- [ ] Update `Reset()` to zero `results`, `total`, `loadingFirstPage`, `loadingNextPage`
      - test: set all fields to non-zero values, call `Reset()`, assert all zero/nil/false

- [ ] `make ci` passes
