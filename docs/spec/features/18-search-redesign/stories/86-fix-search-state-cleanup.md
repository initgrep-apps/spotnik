---
title: "Fix: Clear search state on Esc and backspace-to-empty"
feature: 18-search-redesign
status: open
---

## Background

Two related bugs leave stale search state visible to the user:

1. **Esc doesn't clear query or results.** `handleKey` Esc (search.go:463) emits
   `SearchClosedMsg`. `closeSearch()` in app.go:638 sets `searchOpen = false` but
   does not clear the text input, store query, or search buffers. Reopening the
   overlay shows the old query and old results.

2. **Backspace-to-empty keeps old results.** The backspace handler (search.go:492)
   updates the text input and schedules a debounce. `handleDebounce` (line 450)
   returns nil on empty query — no `SearchClearedMsg` is emitted. The store still
   holds the old query and buffers, so `renderResults` sees `query != ""` and
   renders stale results.

Both stem from incomplete state cleanup paths. Ctrl+U correctly clears everything
because it explicitly calls `o.input.SetValue("")`, `o.clearBuffers()`, and emits
`SearchClearedMsg`. The Esc and backspace paths need equivalent cleanup.

## Design

### Esc cleanup

In `handleKey` Esc case (search.go:463), before emitting `SearchClosedMsg`:

```go
case tea.KeyEsc:
    o.input.SetValue("")
    o.clearBuffers()
    return o, tea.Batch(
        func() tea.Msg { return SearchClearedMsg{} },
        func() tea.Msg { return SearchClosedMsg{} },
    )
```

This clears the input, wipes table rows, and tells app.go to clear store state —
exactly mirroring Ctrl+U, plus closing the overlay.

### Backspace-to-empty cleanup

In the backspace handler, after updating the text input, check if the value
became empty. If so, clear buffers and emit `SearchClearedMsg`:

```go
case tea.KeyBackspace:
    var cmd tea.Cmd
    o.input, cmd = o.input.Update(m)
    q := o.input.Value()
    if q == "" {
        o.clearBuffers()
        return o, tea.Batch(cmd, func() tea.Msg { return SearchClearedMsg{} })
    }
    return o, tea.Batch(cmd, debounceSearch(q))
```

### Files Changed

| Action | File | Purpose |
|--------|------|---------|
| Modify | `internal/ui/panes/search.go` | Clear state on Esc and backspace-to-empty |
| Modify | `internal/ui/panes/search_test.go` | New tests |

## Acceptance Criteria

- [ ] Pressing Esc clears the text input, table rows, and store query/buffers
- [ ] Reopening search after Esc shows an empty input and placeholder text
- [ ] Backspacing until the input is empty clears results and shows placeholder
- [ ] Typing after backspace-to-empty triggers a fresh search with no stale data
- [ ] Ctrl+U behavior unchanged
- [ ] `make ci` passes

## Tasks

- [ ] **Clear state on Esc** — update `handleKey` Esc case to call `o.input.SetValue("")`,
      `o.clearBuffers()`, and emit `SearchClearedMsg` before `SearchClosedMsg`. In
      `internal/ui/panes/search.go`.
      - test: `TestSearchOverlay_Esc_ClearsInput` — verify input is empty after Esc
      - test: `TestSearchOverlay_Esc_EmitsClearedMsg` — verify `SearchClearedMsg` emitted

- [ ] **Clear state on backspace-to-empty** — update `handleKey` backspace case to check
      `o.input.Value() == ""` after updating, and if so call `o.clearBuffers()` and emit
      `SearchClearedMsg` instead of scheduling a debounce. In `internal/ui/panes/search.go`.
      - test: `TestSearchOverlay_Backspace_ToEmpty_ClearsResults` — verify buffers cleared
      - test: `TestSearchOverlay_Backspace_ToEmpty_EmitsClearedMsg` — verify msg emitted
      - test: `TestSearchOverlay_Backspace_NonEmpty_SchedulesDebounce` — verify normal path
