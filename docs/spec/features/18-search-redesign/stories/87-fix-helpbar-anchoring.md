---
title: "Fix: Anchor help bar to overlay bottom"
feature: 18-search-redesign
status: open
---

## Background

The help bar (keybindings + page indicator) floats directly below the last table
row instead of being pinned to the bottom of the overlay. When there are fewer
results than the available height, the help bar drifts upward and empty space
appears below it.

This was fixed once in story 83 (manual rendering era), but story 85 replaced
the manual rendering with `components.Table.View()` and the padding logic was not
carried over. Currently `renderResults` (search.go:717-731) concatenates:

```
tab bar → separator → table.View() → help bar
```

There is no vertical padding between the table output and the help bar. The
`View()` method applies `Height(innerHeight)` to the _entire_ content block
(including the help bar), so padding goes after the help bar rather than before it.

## Design

Insert vertical padding between the table output and the help bar to fill
remaining height. The help bar is always 2 lines (separator + keybindings).

In `renderResults`, after writing the table view, compute how many blank lines
are needed to push the help bar to the bottom:

```go
// Lines consumed: tabbar(1) + tabsep(1) + tableView lines + helpbar(2)
tableLines := strings.Count(o.tables[o.activeSection].View(), "\n") + 1
usedLines := 1 + 1 + tableLines  // tabbar + tabsep + table
helpBarLines := 2
padLines := availableHeight - usedLines - helpBarLines
if padLines > 0 {
    sb.WriteString(strings.Repeat("\n", padLines))
}
sb.WriteString(o.renderHelpBar(contentWidth))
```

### Files Changed

| Action | File | Purpose |
|--------|------|---------|
| Modify | `internal/ui/panes/search.go` | Add padding before help bar in `renderResults` |
| Modify | `internal/ui/panes/search_test.go` | Verify help bar position |

## Acceptance Criteria

- [ ] Help bar (separator + keybindings) is always at the bottom of the overlay
- [ ] Empty space between table rows and help bar when fewer results than capacity
- [ ] Help bar stays at bottom regardless of active section or result count
- [ ] `make ci` passes

## Tasks

- [ ] **Add vertical padding before help bar** — in `renderResults`, count the lines
      consumed by tab bar + separator + table view, then insert blank lines to fill
      remaining height before appending the help bar. In `internal/ui/panes/search.go`.
      - test: `TestSearchOverlay_HelpBar_AnchoredToBottom` — verify help bar is on
        the last 2 lines of the results area regardless of result count
      - test: `TestSearchOverlay_HelpBar_FullResults` — verify no extra padding when
        results fill the space
