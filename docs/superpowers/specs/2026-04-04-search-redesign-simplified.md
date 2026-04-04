# Search Redesign — Simplified Architecture
**Date:** 2026-04-04
**Feature:** 19-search-redesign
**Status:** Approved — ready for story decomposition

---

## Problem Statement

The current search implementation (Feature 19 stories 81–96) has accumulated significant complexity that causes real reliability issues:

- **Tab switches fire immediate API calls** — no debounce, rapid Tab cycling triggers multiple Spotify API calls per keypress, directly causing 429 rate-limit responses
- **Prefetch batch engine** — 5-page sequential chain dispatched on every search, producing up to 5 in-flight requests that continue running after the user types a new query or presses Escape
- **No request cancellation** — `context.Background()` means abandoned HTTP calls hold gateway semaphore slots and count against the rate limit even after the user has moved on
- **Store as intermediary** — `TypePage[T]` structs, `AppendSearch*`, `SearchHasMore`, `SearchActiveType` live in the central Store, but search results are only ever read by the SearchOverlay — the indirection adds complexity without benefit
- **Scroll-threshold prefetch** — `checkPrefetch()` fires on every `↑`/`↓` keypress, creating a probabilistic trigger that misfires during fast scrolling

---

## Design Goals

1. **Every search-triggering interaction is debounced** — typing, tab switch, next page, prev page all go through one mechanism, last one wins
2. **At most one HTTP call live at any time** — enforced at two independent layers (BubbleTea + Gateway)
3. **Simple per-page fetching** — `limit=10`, one API call per page, no prefetch, no batch chaining
4. **Store holds zero search state** — overlay owns display state, App owns staleness keys only
5. **Explicit pagination** — `[` prev / `]` next, user controls page navigation consciously
6. **Cancellable requests** — Escape or new query immediately kills in-flight HTTP via context cancellation

---

## Architecture

### High-Level Design

**Component ownership:**

```
SearchOverlay
├── intent{query, tab, page}   — single source of truth for what the user wants
├── results[]                  — current page items (nil = no results yet)
├── total int                  — drives hasNextPage() and pagination bar
├── loadingFirstPage bool      — true on first fetch: show spinner only
└── loadingNextPage  bool      — true on page change: show spinner + existing results

App (staleness keys + cancellation only)
├── searchQuery string         — staleness check key
├── searchPage  int            — staleness check key
├── searchLoading bool         — true while HTTP call in-flight
└── searchCancel CancelFunc    — kills in-flight HTTP; init to func(){}

Gateway (transport-layer backstop)
└── debounceEntries map[path]*entry  — 100ms hold window for Interactive requests
```

**Overlay state machine:**

```
                    type / Tab / Ctrl+Right / Ctrl+Left
         ┌────────────────────────────────────────────────┐
         │                                                │
       Empty ──── keypress ────► Typing                  │
         ▲                          │                     │
         │           debounce fires │                     │
         │         (query == "")    ▼                     │
         │              no-op ──► Empty                   │
         │                          │                     │
         │         debounce fires   │                     │
         │         (query != "")    ▼                     │
         │                    LoadingFirst                │
         │                          │                     │
         │                  results arrive                │
         │                          ▼                     │
         │                      Results ◄────────────────-┘
         │                          │
         │           Ctrl+Right /   │
         │           Ctrl+Left      ▼
         │                    LoadingNext
         │                          │
         │                  results arrive
         │                          │
         │                          └──────► Results
         │
         └──── Ctrl+U (clear) ──────────────────────────► Empty
         └──── Esc ─────────────────────────────────────► Closed
                    (searchCancel kills in-flight HTTP)
```

**Data flow summary:**

```
User action (type / Tab / Ctrl+Right / Ctrl+Left)
  → intent updated → scheduleDebounce() → tea.Tick(300ms)
    → stale? discard | fresh? SearchRequestMsg
      → app: cancel prior + new ctx → SearchLoadingMsg to overlay
        → buildSearchPageCmd(ctx) → Gateway.Do(Interactive, path, ...)
          → 100ms path-debounce → Spotify API (limit=10, offset=(page-1)*10)
            → SearchPageLoadedMsg
              → app: stale? discard | fresh? forward to overlay
                → overlay: results[], total, loading=false, rebuildListItems()
```

### Data Flow (Detailed)

```
User action (type / Tab / ] / [)
        │
        ▼
o.intent updated  →  scheduleDebounce()  →  tea.Tick(300ms)
                                                    │
                              stale? (snapshot != o.intent) → discard
                                                    │ fresh
                                                    ▼
                                        SearchRequestMsg{Query, Types, Page}
                                                    │
                                                    ▼
                            app.go: a.searchCancel()          ← kills prior HTTP call
                                    ctx, a.searchCancel = context.WithCancel(...)
                                    a.searchQuery = m.Query
                                    a.searchPage  = m.Page
                                    a.searchLoading = true
                                    buildSearchPageCmd(ctx, query, types, offset)
                                                    │
                                    ┌───────────────┘
                                    ▼
                            Gateway.Do(ctx, Interactive, key, fn)
                                    │
                              path debounce (100ms):
                              new Interactive request for same path?
                              cancel previous, only last proceeds
                                    │
                                    ▼
                              HTTP → Spotify API (limit=10, offset=(page-1)*10)
                                    │
                                    ▼
                        SearchPageLoadedMsg{Query, Page, Results, Total}
                                    │
                                    ▼
                        app.go: staleness check
                          m.Query == a.searchQuery && m.Page == a.searchPage?
                                    │ yes
                                    ▼
                        a.searchLoading = false
                        searchPane.Update(msg) → overlay.results = m.Results
                                                  overlay.total   = m.Total
                                                  rebuildListItems()
```

### Key Invariants

- **At most one HTTP call live at any time** — enforced by `searchCancel` (BubbleTea) + 100ms path debounce (Gateway), independently
- **No search state in Store** — overlay owns display state, App owns staleness keys
- **All triggers are equivalent** — typing, tab, `[`, `]` all update `o.intent` and call `scheduleDebounce()`
- **Last wins** — both layers enforce this independently; they do not coordinate

---

## Component Designs

### SearchIntent — universal debounce primitive

All four triggers update a single `searchIntent` struct and call one method:

```go
type searchIntent struct {
    query string
    tab   SearchTab
    page  int
}

func (o *SearchOverlay) scheduleDebounce() tea.Cmd {
    snapshot := o.intent
    return tea.Tick(300*time.Millisecond, func(_ time.Time) tea.Msg {
        return searchDebounceMsg{intent: snapshot}
    })
}
```

**Trigger rules:**

| Trigger | Intent update | Page reset |
|---|---|---|
| Keypress in input | `intent.query = cleanQuery()` | Yes → page=1 |
| Tab / Shift+Tab | `intent.tab = newTab` | Yes → page=1 |
| `]` (next page) | `intent.page++` (if hasNextPage) | No |
| `[` (prev page) | `intent.page--` (if page > 1) | No |

`handleDebounce` discards stale ticks via `m.intent != o.intent` comparison. One code path, four triggers.

---

### SearchOverlay — struct changes

**Removed:**
- `results *SearchResultData` — replaced by direct slice
- `store *state.Store` — overlay no longer reads Store at all
- `activeTab SearchTab` — moves into `intent.tab`

**New / changed fields:**
```go
type SearchOverlay struct {
    results          []SearchListItem  // current page, nil = no results yet
    total            int               // for hasNextPage() and pagination bar
    loadingFirstPage bool              // results==nil, fetch in-flight → spinner only
    loadingNextPage  bool              // results!=nil, fetch in-flight → results + spinner
    intent           searchIntent      // drives all debounce

    // UI components — unchanged
    input      textinput.Model
    spinner    spinner.Model
    help       help.Model
    keyMap     searchKeyMap
    resultList list.Model
    theme      theme.Theme
    width, height    int
    lastSetListH     int
    prefixState      prefixState
    lockedPrefix     string
    placeholderIdx   int
}
```

**Deleted methods:**
- `checkPrefetch()` — prefetch engine gone
- `nextOffsetForTab()` — no per-type pagination
- `CallCheckPrefetch()` — test export, gone with the method

**New methods:**
- `scheduleDebounce() tea.Cmd` — single debounce factory
- `hasNextPage() bool` — `intent.page * SearchPageSize < total`
- `renderPaginationBar(w int) string` — `[ ← page N of M → ]`

---

### App struct — new fields

```go
// Search session state — for staleness and cancellation only.
// These replace all store.searchState fields entirely.
searchQuery    string              // staleness key; "" when no active search
searchPage     int                 // staleness key; 0 when no active search
searchLoading  bool                // true while HTTP call is in-flight
searchCancel   context.CancelFunc  // cancels in-flight HTTP; init to func(){}
```

All four fields live on `App`, not on `Store`. `Store` retains none of these.

**SearchRequestMsg handler:**
1. `a.searchCancel()` — kill prior in-flight call
2. Set `a.searchQuery`, `a.searchPage`, `a.searchLoading = true`
3. Create `ctx, cancel := context.WithCancel(context.Background())`; store cancel
4. Send `SearchLoadingMsg{IsFirstPage: len(overlay.Results()) == 0}` to overlay
5. Dispatch `buildSearchPageCmd(ctx, query, types, (page-1)*10)`

**closeSearch:**
1. `a.searchCancel()` — immediate HTTP abort
2. Clear `searchQuery`, `searchPage`, `searchLoading`, `searchOpen`

**buildSearchPageCmd:** accepts `ctx context.Context`; if `ctx.Err() != nil` after error, return `nil` (Bubble Tea drops nil msgs silently — no stale message enters Update loop).

---

### Store — deleted search subsystem

**Entirely removed:**
- `searchState` struct and all fields
- `TypePage[T any]` generic struct
- `SearchQuery()` / `SetSearchQuery()`
- `SearchLoading()` / `SetSearchLoading()`
- `SearchActiveType()` / `SetSearchActiveType()`
- `SearchTracks()` / `SearchArtists()` / `SearchAlbums()` / `SearchPlaylists()`
- `AppendSearchTracks()` / `AppendSearchArtists()` / `AppendSearchAlbums()` / `AppendSearchPlaylists()`
- `ClearSearchResults()`
- `SearchHasMore()`
- `SetSearchError()` / `ClearSearchError()`

Store retains zero search state after this change.

---

### Message types — changes

**Deleted:**
- `SearchTabChangedMsg` — tab changes go through universal debounce
- `SearchPrefetchMsg` — prefetch engine removed

**Updated:**
```go
// SearchPageLoadedMsg gains Page and flat Total; loses per-type totals.
type SearchPageLoadedMsg struct {
    Query   string
    Page    int               // NEW: staleness key
    Results []SearchListItem  // current page items only (max 10)
    Total   int               // NEW: total results across all types/pages
    Err     error
}
```

**New:**
```go
// SearchLoadingMsg sets loading state on the overlay before the HTTP call.
// IsFirstPage=true → loadingFirstPage (spinner only, no list).
// IsFirstPage=false → loadingNextPage (spinner + existing results visible).
type SearchLoadingMsg struct {
    IsFirstPage bool
}
```

---

### Gateway — Interactive debounce

**New fields on Gateway:**
```go
debounceMu      sync.Mutex
debounceEntries map[string]*interactiveDebounceEntry

type interactiveDebounceEntry struct {
    cancel context.CancelFunc
    ready  chan struct{}
}
```

**New phase in `Do()` — between rate-limit policy and in-flight dedup, for `Interactive` only:**

1. Create `wrappedCtx, wrappedCancel` from incoming `ctx`
2. Lock `debounceMu`; if entry exists for `key.Path`, call its `cancel()`, wait on `<-prev.ready`
3. Register new entry; unlock
4. `select { time.After(100ms) → proceed | wrappedCtx.Done() → return err | ctx.Done() → return err }`
5. Defer: `close(entry.ready)`, remove from map, call `wrappedCancel()`
6. Replace `ctx` with `wrappedCtx` for remainder of `Do()`

**Scope:** applies to all `api.Interactive` requests — search, devices, playback controls, anything user-triggered. Keyed by `RequestKey.Path` only (ignores query params). `Background` requests unaffected.

---

### Loading states — View rendering

| State | `loadingFirstPage` | `loadingNextPage` | Panel 2 renders |
|---|---|---|---|
| No query | false | false | Hint text: `"Type to search"` |
| First fetch in-flight | true | false | Centered spinner: `"◉ Searching…"` |
| Results showing, stable | false | false | List + pagination bar |
| Page change in-flight | false | true | Spinner line above list + list (previous page) + pagination bar |
| Error | false | false | Previous results preserved; toast from app.go |

**Pagination bar** — fixed line at bottom of Panel 2 inner area, shown when `total > 0`:
```
  [ ←   page 3 of 8   → ]
```
- `[ ←` dims (TextMuted) when `page == 1`
- `→ ]` dims when `!hasNextPage()`
- `resizeList()` subtracts 1 line for the pagination bar so the list never overflows

**Panel 2 inner layout (top to bottom):**
```
tab bar (1 line)
separator (1 line)
spinner line (0 or 1 line, loadingNextPage only)
list (fills remaining)
pagination bar (1 line, when total > 0)
```

---

### Pagination — `Ctrl+Right` / `Ctrl+Left` keybindings

`[` and `]` are valid search characters and would be swallowed by the textinput. `Ctrl+Right` / `Ctrl+Left` are intercepted at the overlay `Update` level before forwarding to the input.

- `Ctrl+Right`: `if hasNextPage() { o.intent.page++; return o, o.scheduleDebounce() }`
- `Ctrl+Left`: `if o.intent.page > 1 { o.intent.page--; return o, o.scheduleDebounce() }`
- Both debounced — rapid pressing settles on final page number after 300ms idle
- Pagination bar updates immediately (intent.page changes for display); list updates when response arrives
- Added to `searchKeyMap` and shown in `ShortHelp()` as `ctrl+→ next • ctrl+← prev`

---

### "All" tab — flat interleaved list

When `intent.tab == TabAll`:
- API call: `type=track,artist,album,playlist&limit=10`
- Response contains up to 10 tracks + 10 artists + 10 albums + 10 playlists (40 items max)
- Rendered as a single flat interleaved list via `rebuildListItems()`
- Each item retains its category-specific delegate rendering (icon, fields, colors)
- `total` = `max(tracks.Total, artists.Total, albums.Total, playlists.Total)` from response — the type with the most results determines how many pages exist; `hasNextPage()` = `page * SearchPageSize < total`
- Pagination bar shows `page N of M` where `M = ceil(total / SearchPageSize)`
- `]` / `[` advances offset across all types simultaneously — some types may return fewer items on later pages (exhausted), which is fine; the list shows whatever the API returns

---

## What Is Deleted vs Kept

### Deleted entirely
- `buildSearchBatchCmd` in `commands.go`
- `SearchPrefetchPages`, `SearchPrefetchItems`, `SearchPrefetchThreshold` constants
- `SearchTabChangedMsg`, `SearchPrefetchMsg` message types
- `checkPrefetch()`, `nextOffsetForTab()`, `CallCheckPrefetch()` on `SearchOverlay`
- All `store.searchState` fields and methods
- `TypePage[T]` generic struct
- `searchTypesForActiveType()` standalone helper (logic moves inline to handleDebounce)

### Kept / adapted
- `SearchRequestMsg` — gains `Page int` field
- `SearchPageLoadedMsg` — gains `Page int`, flat `Total int`; loses per-type total fields
- `buildSearchPageCmd` — gains `ctx context.Context` parameter, returns `nil` on ctx cancel
- `convertSearchResult` — updated to return `[]SearchListItem` directly and flat `Total`
- Prefix autocomplete machinery (`prefixState`, `parsePrefix`, `promoteToPromptTag`) — unchanged
- `SearchItemDelegate` — unchanged
- `cycleTabForward/Backward` — simplified: update `intent.tab`, call `scheduleDebounce()`; no `SearchTabChangedMsg`
- All existing keybindings (Enter=play, Ctrl+A=queue, Esc=close, Ctrl+U=clear) — unchanged

---

## Edge Cases

All of these must be handled correctly and tested:

| Scenario | Trigger | Query | Page | Expected behaviour |
|---|---|---|---|---|
| No query — press Next | `Ctrl+Right` | `""` | 1 | Silent no-op. No API call. |
| No query — press Prev | `Ctrl+Left` | `""` | 1 | Silent no-op. No API call. |
| No query — Tab switch | Tab | `""` | 1 | Update `intent.tab` only. No API call. |
| Prefix only (`:songs` without search term) — debounce fires | tick | `""` after `cleanQuery` | 1 | No-op — treated as empty query. |
| Already on last page — press Next | `Ctrl+Right` | `"foo"` | N | `!hasNextPage()` → no-op. `→` dims in bar. |
| Already on first page — press Prev | `Ctrl+Left` | `"foo"` | 1 | `page <= 1` → no-op. `←` dims in bar. |
| Rapid page flipping (5× `Ctrl+Right` quickly) | 5 presses | `"foo"` | 1 | Debounce settles on page 5 after 300ms idle. One API call. |
| New query typed while on page 5 | keypress | new text | 5 | `intent.page` resets to 1. Prior in-flight cancelled. One new API call. |
| Tab switch while on page 3 | Tab | `"foo"` | 3 | `intent.page` resets to 1. Prior in-flight cancelled. One new API call. |
| Esc while loading first page | Esc | `"foo"` | 1 | `searchCancel()` kills HTTP. Overlay closes. No stale result appears. |
| API returns 0 results | response | `"foo"` | 1 | `total=0`. List shows "No results". Pagination bar hidden. Both nav keys no-op. |
| API returns 0 items on page N (type exhausted) | response | `"foo"` | N | Show empty list for this type. `hasNextPage()=false`. Prev still works. |
| API error on first page | error | `"foo"` | 1 | Toast alert. `loadingFirstPage=false`. results=nil. Hint text shown. |
| API error on subsequent page | error | `"foo"` | N>1 | Toast alert. `loadingNextPage=false`. Previous page results stay visible. |
| API 429 | error | `"foo"` | any | Toast "Rate limited". Retry-After respected. Previous results preserved. |
| Context cancelled (new query arrives before response) | new keypress | new | any | `buildSearchPageCmd` returns nil. Bubble Tea drops nil silently. No state change. |
| `Ctrl+U` (clear) while on page 5 | Ctrl+U | any | 5 | `intent = {query:"", tab:current, page:1}`. No API call. Return to empty state. |
| Enter (play) on selected item | Enter | `"foo"` | N | Play item. Overlay stays open. No new search API call. |
| Ctrl+A (queue) on selected item | Ctrl+A | `"foo"` | N | Add to queue. Overlay stays open. No new search API call. |
| Overlay closed and reopened | open | `""` | 1 | Fresh state: results=nil, page=1, both loading flags false. Hint text shown. |
| Stale result arrives after Esc | timing race | old | old | `searchQuery == ""` after close → staleness check discards. No state corruption. |
| BubbleTea 300ms + Gateway 100ms both fire for same intent | concurrent | same | same | Both layers enforce last-wins independently. Exactly one HTTP call proceeds. |

## Testing Strategy

- **Unit: `searchIntent` debounce** — multiple rapid triggers settle on last intent; stale ticks discard
- **Unit: `handleDebounce`** — empty query no-ops; PrefixTyping no-ops; stale snapshot discards
- **Unit: `hasNextPage()`** — boundary cases: total=0, total=10, total=11, total=100
- **Unit: pagination bar** — page 1 (prev dimmed), mid-page, last page (next dimmed)
- **Unit: `buildSearchPageCmd` with cancelled context** — returns nil msg, no toast
- **Unit: staleness check** — query mismatch discards; page mismatch discards; both match proceeds
- **Unit: gateway Interactive debounce** — second request within 100ms cancels first; requests for different paths independent; Background requests unaffected
- **Integration: SearchRequestMsg handler** — cancel called before new dispatch; store has zero search fields after change
- **Integration: closeSearch** — searchCancel called; searchQuery/page/loading cleared
- **Elm purity** — no Store writes inside command closures
- **Coverage gate** — `make test-coverage` must pass at ≥80%
