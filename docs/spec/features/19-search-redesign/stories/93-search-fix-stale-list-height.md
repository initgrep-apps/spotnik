---
title: "Fix: Stale list height causes UI disintegration on typing/scrolling"
feature: 19-search-redesign
status: open
---

## Background

This is the most critical visual bug in the search overlay. Typing a character,
pressing backspace, or scrolling can cause the results panel to disintegrate —
duplicate lines appear inside the panel, panel borders overlap with content, and
the entire TUI looks broken until the terminal is resized.

### Root cause

The search overlay has three stacked panels. The top panel (Search) has a dynamic
height: it is **4 lines** when the prefix hint row is visible, and **3 lines** when
hidden. The hint row is controlled by `showHintLine()` in `search_prefix.go`:

```go
func (o *SearchOverlay) showHintLine() bool {
    if o.prefixState == PrefixLocked {
        return false
    }
    return o.input.Value() == "" || o.prefixState == PrefixTyping
}
```

So the hint line is visible when the input is empty or the user is typing a command
prefix (e.g. `:so`). It disappears the moment the user types a normal character.

`panelHeights()` in `search.go` uses `showHintLine()` to compute `resultsH` and
therefore `listH` (the height passed to `resultList.SetSize()`):

```go
func (o *SearchOverlay) panelHeights() (searchH, resultsH, helpH int) {
    searchH = 3
    if o.showHintLine() {
        searchH = 4   // +1 for hint row
    }
    helpH = 3
    resultsH = o.overlayHeight() - searchH - helpH
    ...
}
```

`SetSize()` is the only place `resultList.SetSize(listW, listH)` is called:

```go
func (o *SearchOverlay) SetSize(width, height int) {
    ...
    _, resultsH, _ := o.panelHeights()
    listH := resultsH - 4
    o.resultList.SetSize(listW, listH)
}
```

`SetSize()` is called from `app.go` **only** when `tea.WindowSizeMsg` arrives
(i.e., when the terminal is resized). It is never called in response to
`showHintLine()` changing.

### Concrete failure sequence

1. User opens overlay → input is empty → `showHintLine()=true` → `searchH=4` →
   `listH = totalH - 4 - 3 - 4 = N` → `resultList.SetSize(w, N)` is called once.
2. User types "jazz" → `showHintLine()=false` → `searchH=3` in `renderSearchPanel()`.
   - `View()` renders a 3-line search panel, allocating 1 extra line to `resultsH`.
   - But `resultList` was set to height `N` and still renders `N` lines.
   - The outer `resultsPanel` container renders `N+1` lines of inner content but the
     list only produces `N` → blank line OR the list `View()` overflows by 1.
3. Conversely: user presses Ctrl+U to clear → `showHintLine()` flips back to true →
   `searchH=4` → `resultsPanel` has 1 fewer line, but list still renders at height
   `N` → list overflows its container by 1 line.
4. Each backspace/type that toggles the hint line compounds the misalignment until
   visual artifacts (duplicated rows, broken borders) are severe.

### Why scrolling triggers it too

`KeyUp`/`KeyDown` call `o.resultList.Update(m)` which advances the list cursor. The
bubbles `list.Model.View()` re-renders relative to the cursor position. If the list's
internal height doesn't match the container the rendered view will clip or pad
incorrectly, producing the same overflow artefacts.

### Fix

Every time `showHintLine()` might change (any key that modifies input state,
Ctrl+U, SearchClearedMsg), the list height must be recalculated and
`resultList.SetSize()` called with the fresh values.

Add a private helper `resizeList()` that reads the current `panelHeights()` and
calls `resultList.SetSize()`:

```go
// resizeList recomputes the list dimensions from current panelHeights() and
// applies them. Must be called after any state change that could affect
// showHintLine() (typing, backspace, Ctrl+U, clear).
func (o *SearchOverlay) resizeList() {
    w := o.overlayWidth()
    _, resultsH, _ := o.panelHeights()
    listW := w - 2
    if listW < 1 {
        listW = 1
    }
    listH := resultsH - 4
    if listH < 1 {
        listH = 1
    }
    o.resultList.SetSize(listW, listH)
}
```

Call `resizeList()` in `handleKey()` after every branch that touches input state:

- After `tea.KeyCtrlU` (clears input → hint line reappears)
- After `tea.KeyBackspace` (may clear input → hint line may reappear)
- After the `default` branch (typing → hint line may disappear)
- After `tea.KeyTab` / `tea.KeyShiftTab` (prefix state changes → hint line may toggle)

Also call `resizeList()` in the `SearchClearedMsg` handler (which is triggered on
overlay open via `Init()` — input is cleared so hint line should be visible and list
height must reflect the 4-line search panel).

**File: `internal/ui/panes/search.go`**

Add `resizeList()` helper.

In `handleKey()`, insert `o.resizeList()` **before** the `return` in each of the
following branches:

```go
case tea.KeyCtrlU:
    o.input.Prompt = "> "
    o.input.SetValue("")
    o.lockedPrefix = ""
    o.prefixState = PrefixNone
    o.resizeList()   // ← add this
    return o, tea.Batch(
        func() tea.Msg { return SearchClearedMsg{} },
        searchPlaceholderTick(),
    )

case tea.KeyBackspace:
    // ... existing backspace logic ...
    o.resizeList()   // ← add before return
    // ... returns ...

case tea.KeyTab, tea.KeyShiftTab:
    // ... existing tab cycling ...
    // already returns early for PrefixTyping; add resizeList before return
    o.resizeList()   // ← add before cycleTab* calls return

default:
    // ... existing typing logic ...
    o.resizeList()   // ← add before return
```

In the `SearchClearedMsg` handler:

```go
case SearchClearedMsg:
    o.results = nil
    o.resultList.SetItems(nil)
    o.resizeList()   // ← add this
    return o, nil
```

## Acceptance Criteria

- [ ] Typing a query after opening the overlay does not produce duplicate lines or
      broken panel borders in the results panel
- [ ] Pressing backspace one or more times (including clearing the input) does not
      cause visual artifacts
- [ ] Ctrl+U to clear the input does not cause visual artifacts
- [ ] Scrolling (Up/Down) after any of the above does not cause visual artifacts
- [ ] `resizeList()` is called after every state change that affects `showHintLine()`
- [ ] The hint row toggling (visible ↔ hidden) is reflected in the list height
      without a terminal resize

## Tasks

- [ ] Add `resizeList()` helper to `SearchOverlay` in `search.go`
      - test: call `resizeList()` after setting input to non-empty value (hint hidden →
        searchH=3), then call `panelHeights()` directly and verify `listH` matches
        `o.overlayHeight() - 3 - 3 - 4`; repeat with empty input (hint visible →
        searchH=4)

- [ ] Call `resizeList()` in `handleKey` — `KeyCtrlU` branch — after clearing input
      - test: send KeyCtrlU message to overlay with a non-empty query; after update,
        verify `o.resultList`'s rendered height matches hint-visible list height
        (i.e. `overlayHeight - 4 - 3 - 4`)

- [ ] Call `resizeList()` in `handleKey` — `KeyBackspace` branch — after updating input
      - test: set input to single character "j"; send KeyBackspace; after update
        verify list height reflects empty-input state (hint visible, searchH=4)

- [ ] Call `resizeList()` in `handleKey` — `default` branch — after updating input
      - test: open overlay with empty input (list height = hint-visible); send a rune
        key 'j'; after update verify list height reflects no-hint state (searchH=3);
        height must differ by 1 from the prior empty-input height

- [ ] Call `resizeList()` in `handleKey` — `KeyTab`/`KeyShiftTab` branches
      - test: cycle to a non-All tab (PrefixLocked, no hint); verify list height =
        no-hint height; cycle back to All (PrefixNone, hint visible); verify list
        height = hint-visible height

- [ ] Call `resizeList()` in the `SearchClearedMsg` handler
      - test: seed overlay with non-empty input (hint hidden, list sized for searchH=3);
        send `SearchClearedMsg`; verify `o.resultList` size matches hint-visible height

- [ ] `make ci` passes with no regressions
