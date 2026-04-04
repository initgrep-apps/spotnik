---
title: "Search Redesign: Universal debounce primitive (searchIntent)"
feature: 19-search-redesign
status: open
---

## Background

The current `SearchOverlay` has three separate code paths that trigger searches:
1. `debounceSearch(query string)` — called on keypresses, query-only snapshot
2. `cycleTabForward()` / `cycleTabBackward()` — dispatches `SearchTabChangedMsg` immediately
   (no debounce), which app.go handles by clearing all results and firing a new batch
3. Pagination — not yet implemented

Each path uses different state and has different timing. Tab switching is never debounced,
meaning rapid Tab cycling fires one API call per Tab press. There is no shared "what does
the user currently want?" snapshot.

This story replaces all three paths with a single `searchIntent` struct and one
`scheduleDebounce()` method. Every trigger writes to `o.intent` and calls
`scheduleDebounce()`. Stale ticks are discarded by comparing the tick's snapshot to the
current `o.intent` at fire time.

## Design

### `searchIntent` struct

Add to `internal/ui/panes/search.go`:

```go
// searchIntent captures the full desired search state at a point in time.
// All four triggers (type, Tab, Ctrl+Right, Ctrl+Left) write to this struct
// and call scheduleDebounce(). The debounce tick carries a snapshot; if the
// snapshot differs from the current intent at fire time, the tick is stale and discarded.
type searchIntent struct {
    query string
    tab   SearchTab
    page  int
}
```

### `SearchOverlay` struct changes

**Remove field:**
```go
activeTab SearchTab  // moves into intent.tab
```

**Add field:**
```go
intent searchIntent
```

All existing references to `o.activeTab` become `o.intent.tab`. Update all read and write
sites in `search.go`.

### `scheduleDebounce()` method

Replace the existing `debounceSearch(query string) tea.Cmd` with:

```go
// scheduleDebounce snapshots the current intent and returns a 300ms tick.
// When the tick fires, handleDebounce compares the snapshot to the current
// intent — if they differ, the tick is discarded (the user has since moved on).
func (o *SearchOverlay) scheduleDebounce() tea.Cmd {
    snapshot := o.intent
    return tea.Tick(300*time.Millisecond, func(_ time.Time) tea.Msg {
        return searchDebounceMsg{intent: snapshot}
    })
}
```

### `searchDebounceMsg` type

```go
// searchDebounceMsg is the internal tick fired by scheduleDebounce.
// It is never routed to app.go — handled entirely within SearchOverlay.Update().
type searchDebounceMsg struct {
    intent searchIntent
}
```

### `handleDebounce` method

```go
// handleDebounce is called when a searchDebounceMsg arrives. It discards stale
// ticks (intent has changed since the tick was scheduled) and no-ops on empty
// or prefix-only queries.
func (o *SearchOverlay) handleDebounce(m searchDebounceMsg) (SearchOverlay, tea.Cmd) {
    // Stale: user has moved on since this tick was scheduled.
    if m.intent != o.intent {
        return *o, nil
    }
    // No-op: nothing to search.
    query := cleanQuery(o.intent.query)
    if query == "" || o.prefixState == PrefixTyping {
        return *o, nil
    }
    types := searchTypesForTab(o.intent.tab)
    return *o, func() tea.Msg {
        return SearchRequestMsg{Query: query, Types: types, Page: o.intent.page}
    }
}
```

### Trigger rules — what each trigger must do

| Trigger | Intent update | Page reset |
|---|---|---|
| Keypress in input | `o.intent.query = input.Value()` | Yes → `o.intent.page = 1` |
| Tab / Shift+Tab | `o.intent.tab = newTab` | Yes → `o.intent.page = 1` |
| `Ctrl+Right` (next page) | `o.intent.page++` (only if hasNextPage()) | No |
| `Ctrl+Left` (prev page) | `o.intent.page--` (only if page > 1) | No |

After every intent update, call `o.scheduleDebounce()` and return the result as the Cmd.

### `cycleTabForward` / `cycleTabBackward` simplification

Current implementation dispatches `SearchTabChangedMsg`. Replace with:

```go
func (o *SearchOverlay) cycleTabForward() (SearchOverlay, tea.Cmd) {
    o.intent.tab = nextTab(o.intent.tab)
    o.intent.page = 1
    return *o, o.scheduleDebounce()
}

func (o *SearchOverlay) cycleTabBackward() (SearchOverlay, tea.Cmd) {
    o.intent.tab = prevTab(o.intent.tab)
    o.intent.page = 1
    return *o, o.scheduleDebounce()
}
```

No `SearchTabChangedMsg` is dispatched. The tab bar renders from `o.intent.tab` (was
`o.activeTab`).

### `Reset()` update

`Reset()` (added in story 96) currently resets `o.activeTab = TabAll`. Update it to reset
the full intent:

```go
o.intent = searchIntent{query: "", tab: TabAll, page: 1}
```

### Remove deleted items

- Delete `debounceSearch(query string) tea.Cmd` method
- Delete `activeTab SearchTab` field (replaced by `intent.tab`)
- Remove `searchTypesForActiveType()` standalone helper if it exists — logic is now inline
  in `handleDebounce` via `searchTypesForTab(tab)`

## Acceptance Criteria

- [ ] `searchIntent{query, tab, page}` struct exists on `SearchOverlay`
- [ ] `scheduleDebounce() tea.Cmd` is the single debounce factory; `debounceSearch` is deleted
- [ ] `cycleTabForward/Backward` update `intent.tab`, reset `intent.page=1`, call `scheduleDebounce()` — no `SearchTabChangedMsg`
- [ ] `handleDebounce` discards stale ticks via `m.intent != o.intent`
- [ ] `handleDebounce` no-ops on empty query and `PrefixTyping` state
- [ ] `o.activeTab` field is removed; all references use `o.intent.tab`
- [ ] `Reset()` resets full `intent` struct
- [ ] `make ci` passes

## Tasks

- [ ] Add `searchIntent` struct and `intent searchIntent` field to `SearchOverlay`; remove `activeTab` field
      - update all `o.activeTab` reads/writes to `o.intent.tab`
      - test: `TestSearchOverlay_CycleTab*` still passes; tab bar renders from `o.intent.tab`

- [ ] Replace `debounceSearch(query)` with `scheduleDebounce()` + `searchDebounceMsg` type
      - update all call sites in `Update()` that called `debounceSearch`
      - test: scheduling a debounce then calling `handleDebounce` with matching intent fires
        `SearchRequestMsg`; calling with non-matching intent fires nothing

- [ ] Implement `handleDebounce(searchDebounceMsg)` with stale-tick detection and empty-query no-op
      - test table:
        | scenario | intent at fire | snapshot | query | expected |
        |---|---|---|---|---|
        | fresh, non-empty | {q:"jazz",tab:All,page:1} | same | "jazz" | returns SearchRequestMsg |
        | stale (user typed more) | {q:"jazz rock",tab:All,page:1} | {q:"jazz",...} | — | no-op |
        | empty query | {q:"",tab:All,page:1} | same | "" | no-op |
        | prefix only | {q:":songs",tab:Songs,page:1} | same | "" after cleanQuery | no-op |

- [ ] Simplify `cycleTabForward` / `cycleTabBackward` to update intent + scheduleDebounce
      - test: tab change with empty query → no SearchRequestMsg (handleDebounce no-ops);
        tab change with "jazz" query → SearchRequestMsg{Types: trackTypes, Page: 1}

- [ ] Update `Reset()` to reset full intent struct
      - test: after Reset(), `o.intent == searchIntent{query:"", tab:TabAll, page:1}`

- [ ] `make ci` passes
