---
title: "Fix: Overlay shows stale query, tab, and results when reopened after Esc"
feature: 19-search-redesign
status: done
---

## Background

When the user presses Esc to close the search overlay and then presses `/` to reopen
it, they see the previous session's query text, locked prefix tag, active tab
selection, and result list items. The overlay is supposed to open fresh every time.

### Why the design intends a fresh start

`SearchOverlay.Init()` deliberately emits `SearchClearedMsg` every time the overlay
opens, as the clear-on-open mechanism:

```go
func (o *SearchOverlay) Init() tea.Cmd {
    clearCmd := func() tea.Msg { return SearchClearedMsg{} }
    return tea.Batch(textinput.Blink, searchSpinnerTick(), clearCmd, searchPlaceholderTick())
}
```

The overlay's own `SearchClearedMsg` handler is supposed to reset local state:

```go
case SearchClearedMsg:
    o.results = nil
    o.resultList.SetItems(nil)
    return o, nil
```

And app.go handles the same message to clear the store:

```go
case panes.SearchClearedMsg:
    a.store.ClearSearchResults()
    a.store.SetSearchQuery("")
    a.store.SetSearchLoading(false)
    a.store.ClearSearchError()
    return a, nil
```

### Root cause — `SearchClearedMsg` never reaches the overlay

The app's `handleMsg()` switch handles `SearchClearedMsg` and **returns early**:

```go
case panes.SearchClearedMsg:
    a.store.ClearSearchResults()
    ...
    return a, nil   // ← exits handleMsg immediately
```

The search-pane forwarding block at the bottom of `handleMsg` is only reached when
the main switch falls through without returning:

```go
// (bottom of handleMsg, after the switch)
if a.searchOpen {
    updated, cmd := a.searchPane.Update(msg)
    ...
}
```

Because `SearchClearedMsg` is handled with an early `return`, the message **never
reaches `a.searchPane.Update()`**. The overlay's `SearchClearedMsg` handler —
`o.results = nil; o.resultList.SetItems(nil)` — is dead code in production. The
overlay's input, tab, prefix lock, and result list are never reset.

### What persists between sessions

After pressing Esc and reopening:

| Field | Expected | Actual |
|---|---|---|
| `o.input.Value()` | `""` | previous query text |
| `o.input.Prompt` | `"> "` | styled prefix tag (if locked) |
| `o.activeTab` | `TabAll` | whatever tab was active |
| `o.prefixState` | `PrefixNone` | `PrefixLocked` (if prefix was set) |
| `o.lockedPrefix` | `""` | previous locked prefix |
| `o.results` | `nil` | previous `*SearchResultData` |
| `o.resultList.Items()` | `[]` | previous search results |

### Fix

The cleanest fix is to reset the overlay's local state directly inside
`openSearch()` in `app.go`, **before** calling `Init()`. This is authoritative and
does not rely on message routing. The overlay gets a clean slate before `Init()`
fires its batch of commands.

Add a `Reset()` method to `SearchOverlay` that restores all fields to their
initial values (same as `NewSearchOverlay` construction, minus store/theme):

**File: `internal/ui/panes/search.go`**

```go
// Reset restores the overlay to its initial empty state, as if it had just been
// constructed. Called by the root app when the overlay is opened (openSearch) to
// guarantee a fresh start every session, regardless of the previous session's state.
func (o *SearchOverlay) Reset() {
    o.input.SetValue("")
    o.input.Prompt = "> "
    o.input.Placeholder = searchPlaceholders[0]
    o.placeholderIdx = 0
    o.activeTab = TabAll
    o.prefixState = PrefixNone
    o.lockedPrefix = ""
    o.results = nil
    o.resultList.SetItems(nil)
}
```

**File: `internal/app/app.go` — `openSearch()`**

```go
func (a *App) openSearch() (*App, tea.Cmd) {
    a.searchPane.Reset()   // ← reset overlay before Init
    a.searchOpen = true
    cmd := a.searchPane.Init()
    return a, cmd
}
```

`Init()` still runs after `Reset()` to start the cursor blink, spinner tick, and
placeholder tick commands. `SearchClearedMsg` from `Init()` still goes to app.go's
handler and clears the store — that part works correctly today.

Note: `Reset()` does **not** call `resizeList()` because the terminal size may not
be set at the moment of Reset (overlay not yet rendered). The first `SetSize()` call
during render will size the list correctly.

### Why not forward SearchClearedMsg to the overlay

An alternative is to forward `SearchClearedMsg` from app.go to the overlay after
handling it. This would require removing the `return a, nil` early exit and instead
continuing to the forwarding block. The downside is that `SearchClearedMsg` would then
be processed twice — once in `handleMsg` (for store clearing) and once in the overlay
— which is harder to reason about and test. The `Reset()` approach is explicit,
direct, and easy to test in isolation.

## Acceptance Criteria

- [ ] After pressing Esc and reopening with `/`, the text input is empty
- [ ] After pressing Esc and reopening with `/`, the active tab is `TabAll`
- [ ] After pressing Esc and reopening with `/`, no prefix tag is shown in the input
      Prompt (Prompt is `"> "`)
- [ ] After pressing Esc and reopening with `/`, the result list is empty
- [ ] After pressing Esc and reopening with `/`, `o.results` is nil
- [ ] `Reset()` is idempotent — calling it multiple times in a row produces the same
      clean state
- [ ] Reopening the overlay still starts the cursor blink, spinner, and placeholder tick
      (Init() still runs after Reset())

## Tasks

- [ ] Add `Reset()` method to `SearchOverlay` in `search.go`
      - test: construct overlay; set query, lock a prefix (:songs), cycle to TabSongs,
        load fake results into `o.results` and `o.resultList`; call `Reset()`; assert
        `o.input.Value() == ""`; `o.input.Prompt == "> "`; `o.activeTab == TabAll`;
        `o.prefixState == PrefixNone`; `o.lockedPrefix == ""`; `o.results == nil`;
        `len(o.resultList.Items()) == 0`; `o.placeholderIdx == 0`

- [ ] Call `a.searchPane.Reset()` in `openSearch()` before `a.searchPane.Init()`
      - test: open search, type query, send SearchClosedMsg; call `openSearch()` again;
        verify `a.searchPane.Query() == ""`; verify `a.searchPane.ActiveTab() == TabAll`;
        verify `len(a.searchPane.ResultListItems()) == 0`

- [ ] `make ci` passes with no regressions
