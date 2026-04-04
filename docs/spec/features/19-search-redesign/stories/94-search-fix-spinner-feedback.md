---
title: "Fix: Spinner never animates and no loading feedback on re-search"
feature: 19-search-redesign
status: done
---

## Background

The search overlay has two distinct spinner bugs that together mean the user sees
**zero visual feedback** while waiting for search results.

### Root cause A — Spinner tick type mismatch (spinner never animates)

The overlay wraps `spinner.TickMsg` into a private type to avoid cross-component
interference:

```go
// search.go
type searchSpinnerTickMsg spinner.TickMsg
```

The Update handler matches on this private type:

```go
case searchSpinnerTickMsg:
    var cmd tea.Cmd
    o.spinner, cmd = o.spinner.Update(spinner.TickMsg(m))
    return o, cmd
```

But `Init()` starts the spinner tick using the **standard** bubbles tick command:

```go
func (o *SearchOverlay) Init() tea.Cmd {
    return tea.Batch(textinput.Blink, o.spinner.Tick, clearCmd, searchPlaceholderTick())
    //                                  ^^^^^^^^^^^^
    //                    This returns spinner.TickMsg, NOT searchSpinnerTickMsg
}
```

`o.spinner.Tick` is a `tea.Cmd` that fires `spinner.TickMsg{}`. When that message
arrives at the overlay's `Update()`, none of the `case` branches match it — it falls
through to:

```go
var cmd tea.Cmd
o.input, cmd = o.input.Update(msg)
return o, cmd
```

`textinput.Update` ignores `spinner.TickMsg`, returns nil cmd. No new tick is
scheduled. The spinner freezes on its initial frame (a single dot) and never moves.

The `SearchSpinnerTickMsgForTest()` test helper (bottom of `search.go`) manually
injects `searchSpinnerTickMsg{}` so tests pass — but in production the real
`spinner.TickMsg` is never processed.

**Fix A:** Change `Init()` to wrap the spinner tick in a command that returns
`searchSpinnerTickMsg` instead of `spinner.TickMsg`:

```go
// spinnerTick wraps the bubbles spinner tick so only searchSpinnerTickMsg
// messages drive the search spinner — preventing any other spinner.TickMsg
// in the app from accidentally advancing this spinner.
func searchSpinnerTick() tea.Cmd {
    return func() tea.Msg {
        // spinner.New().Tick fires spinner.TickMsg; we wrap it as our private type.
        // We can't call o.spinner.Tick directly (would still return spinner.TickMsg),
        // so we use tea.Tick with the spinner's interval. The bubbles spinner.Dot
        // interval is 130ms.
        return func() tea.Msg {
            return searchSpinnerTickMsg{}
        }
    }
}
```

Actually, the cleaner approach is to use `tea.Tick` directly with a fixed interval
and return `searchSpinnerTickMsg`. Replace `o.spinner.Tick` in `Init()` with a
`tea.Cmd` that fires `searchSpinnerTickMsg`:

```go
func searchSpinnerTick() tea.Cmd {
    // spinner.Dot ticks every 130ms — match that interval.
    return tea.Tick(130*time.Millisecond, func(_ time.Time) tea.Msg {
        return searchSpinnerTickMsg{}
    })
}
```

Update `Init()`:

```go
func (o *SearchOverlay) Init() tea.Cmd {
    clearCmd := func() tea.Msg { return SearchClearedMsg{} }
    return tea.Batch(textinput.Blink, searchSpinnerTick(), clearCmd, searchPlaceholderTick())
}
```

The `searchSpinnerTickMsg` handler already re-arms itself:

```go
case searchSpinnerTickMsg:
    var cmd tea.Cmd
    o.spinner, cmd = o.spinner.Update(spinner.TickMsg(m))
    return o, cmd
```

`spinner.Update()` returns a new `spinner.Tick` cmd — but now that will be a plain
`spinner.TickMsg` again, not `searchSpinnerTickMsg`. So the handler must re-arm with
`searchSpinnerTick()` explicitly instead of relying on `spinner.Update`'s returned cmd:

```go
case searchSpinnerTickMsg:
    // Advance the spinner frame — ignore the cmd returned by spinner.Update
    // because it fires spinner.TickMsg, not searchSpinnerTickMsg.
    o.spinner, _ = o.spinner.Update(spinner.TickMsg(m))
    return o, searchSpinnerTick()
```

**File: `internal/ui/panes/search.go`**
- Add `searchSpinnerTick()` function (uses `tea.Tick` at 130ms, returns `searchSpinnerTickMsg`)
- Replace `o.spinner.Tick` in `Init()` with `searchSpinnerTick()`
- Update the `searchSpinnerTickMsg` handler to re-arm with `searchSpinnerTick()` instead
  of using the cmd from `spinner.Update`

### Root cause B — Spinner only shown for first load, not re-searches

`renderResults()` only shows the spinner when **both** conditions are met:

```go
if loading && len(o.resultList.Items()) == 0 {
    return fmt.Sprintf("%s Searching...\n", o.spinner.View())
}
```

The `len == 0` guard was added to avoid blanking the screen when refreshing with
existing results. But it means that when the user refines a query (e.g. changes
"jazz" to "jazz rock"), the old results stay fully visible with zero feedback that
a new fetch is in flight. From the user's perspective, the search appears broken.

**Fix B:** Show a compact loading indicator in the tab bar when `loading` is true,
regardless of whether existing items are present. The tab bar already renders in
`renderTabBar()`. Append a small spinner frame + "Searching..." text to the right
side of the tab bar row when `loading`:

```go
func (o *SearchOverlay) renderTabBar(innerWidth int) string {
    // ... existing tab parts rendering ...

    tabLine := strings.Join(parts, "  ")

    if o.store.SearchLoading() {
        // Append spinner to the right of the tab bar so it doesn't blank results.
        spinnerStr := lipgloss.NewStyle().
            Foreground(o.theme.TextMuted()).
            Render(o.spinner.View())
        // Right-align: pad tab labels to fill the row, then append spinner.
        padding := innerWidth - lipgloss.Width(tabLine) - lipgloss.Width(spinnerStr)
        if padding < 1 {
            padding = 1
        }
        tabLine = tabLine + strings.Repeat(" ", padding) + spinnerStr
    }

    return lipgloss.NewStyle().Width(innerWidth).MaxWidth(innerWidth).Render(tabLine)
}
```

This keeps the existing results visible (not blanked) while giving clear visual
feedback that a fetch is in progress.

**File: `internal/ui/panes/search.go`**
- Update `renderTabBar()` to append spinner to the right when `store.SearchLoading()` is true

## Acceptance Criteria

- [ ] The spinner dot animates visibly (advances frames) while the search overlay is open
- [ ] A spinner appears in the tab bar on the right side whenever a search is in flight,
      regardless of whether existing results are already displayed
- [ ] When the first search fires (no results yet), the spinner also appears in the tab bar
- [ ] When the spinner is not loading, the tab bar shows only the tab labels with no spinner
- [ ] The spinner animation does not require a terminal resize to start — it begins
      immediately when the overlay opens

## Tasks

- [ ] Add `searchSpinnerTick()` function that returns `tea.Cmd` firing `searchSpinnerTickMsg`
      after 130ms (matching `spinner.Dot` interval)
      - test: `searchSpinnerTick()` returns a non-nil `tea.Cmd`; executing the cmd returns
        a value whose type is `searchSpinnerTickMsg`

- [ ] Replace `o.spinner.Tick` in `Init()` with `searchSpinnerTick()`
      - test: call `Init()` and drive all returned cmds; verify at least one fires a
        `searchSpinnerTickMsg` (not `spinner.TickMsg`); no `spinner.TickMsg` should appear
        in the batch

- [ ] Update `searchSpinnerTickMsg` handler to re-arm with `searchSpinnerTick()` instead
      of the cmd from `spinner.Update`
      - test: send `searchSpinnerTickMsg{}` to the overlay's `Update()`; inspect the
        returned cmd; execute it and verify it returns `searchSpinnerTickMsg` (not
        `spinner.TickMsg`); send the message 5 times in sequence and verify the spinner
        frame advances each time (compare `o.spinner.View()` before and after)

- [ ] Update `renderTabBar()` to append the spinner on the right when `store.SearchLoading()`
      is true
      - test: construct an overlay with `store.SetSearchLoading(true)` and non-empty list
        items; call `View()` (or `renderTabBar()` directly); verify the spinner frame
        string appears somewhere in the tab bar output
      - test: with `store.SearchLoading()=false`, verify the spinner frame does NOT appear
        in the tab bar output
      - test: with loading=true and zero items, spinner still appears in the tab bar

- [ ] Delete `SearchSpinnerTickMsgForTest()` from `search.go` — it was a workaround for
      the broken tick; tests must now drive the spinner via `searchSpinnerTick()` or by
      sending `searchSpinnerTickMsg{}` directly
      - test: no test references `SearchSpinnerTickMsgForTest`; `make ci` passes

- [ ] `make ci` passes with no regressions
