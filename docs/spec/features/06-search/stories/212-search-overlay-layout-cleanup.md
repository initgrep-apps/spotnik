---
title: "Search Overlay Layout Cleanup: Remove Hints, Bottom Panel, Stabilize Heights"
feature: 06-search
status: done
---

## Background

The search overlay currently renders three panels: Search (input + hint pills), Results (tabs + list), and Keys (bottom command bar). This wastes vertical space and creates visual clutter:

1. **Prefix hint pills** (`:songs :artists ...`) below the input duplicate the tab bar and crowd the Search panel.
2. **Bottom Keys panel** consumes 3 lines that could go to results.
3. **Variable panel heights** (`searchH = 3` or `4` depending on hint line) complicate `resizeList()` and cause visual artifacts.

This story removes all three problems: hints gone, bottom panel gone, heights stable.

## Design

### Target Layout

```
╭─ Search ───────────────────────────────────────────────────────╮
│ [ :songs ] search tracks                                       │
╰────────────────────────────────────────────────────────────────╯
╭─ Results ─────────── ctrl+a queue ─╮─ tab filter ─╮─ pgdn prev ─╮─ pgup next ─╭
│ [All]  Songs  Artists  Albums  Playlists                        │
│ ────────────────────────────────────────────────────────────── │
│                                                                │
│        Type to search tracks, artists, albums...               │
│                                                                │
│               [ ←  page 1  → ]                                 │
╰────────────────────────────────────────────────────────────────╯
```

### Changes

1. **`renderPrefixHints()`** returns `""` unconditionally.
2. **`showHintLine()`** returns `false` unconditionally.
3. **`panelHeights()`** always returns `searchH = 3`, `helpH = 0`.
4. **Remove** `searchKeyMap` type, `NewSearchKeyMap()`, `renderHelpPanel()`, `hintBindings()`.
5. **`resultActions()`** returns `[]layout.Action` for the Results panel border: `ctrl+a queue`, `tab filter`, `pgdn prev`, `pgup next`.
6. **`renderResultsPanel()`** passes `Actions: o.resultActions()` into `OverlayChrome`.
7. **`View()`** joins only 2 panels (Search + Results).

## Acceptance Criteria

- [ ] `renderPrefixHints(40)` returns `""` in all states (empty input, typing prefix, locked prefix, normal query)
- [ ] `showHintLine()` always returns `false`
- [ ] `panelHeights()` always returns `searchH = 3`, `helpH = 0`
- [ ] `View()` contains exactly 2 `╭` and 2 `╰` border lines (2 panels, not 3)
- [ ] Results panel border contains action notches when pane is wide enough
- [ ] `resultActions()` returns 4 actions with correct keys and labels
- [ ] All existing tests updated or removed for removed behavior
- [ ] Full test suite passes (`make test`)

## Tasks

1. **Make `renderPrefixHints()` and `showHintLine()` no-ops** — update `search_prefix.go` and tests.
2. **Stabilize `panelHeights()`** — update `search.go` to always return `searchH = 3`, `helpH = 0`.
3. **Remove bottom Keys panel** — delete `searchKeyMap`, `renderHelpPanel()`, `hintBindings()`; add `resultActions()`; wire into `OverlayChrome.Actions`; update `View()` to 2 panels.
4. **Update tests** — adjust/remove tests asserting 3-panel output, hint lines, `NewSearchKeyMap`, bottom bar content.
