---
title: "Search Redesign: Cleanup and Integration"
feature: 19-search-redesign
status: open
---

## Background

After the core search redesign stories (81-85) are implemented, this story handles final cleanup: removing dead code from the old search implementation, ensuring all edge cases work, verifying theme integration, and running comprehensive integration tests.

## Design

### Dead Code Removal

Remove from `search.go`:
- `searchSection` enum (`sectionTracks`, `sectionArtists`, etc.) — replaced by `searchTab`
- `searchSectionLabels` array — replaced by `tabLabels`
- `maxResultsPerSection` constant — replaced by prefetch pagination
- All `clamped*Items` helpers — list handles all items now
- All `clamped*ItemsAsRows` helpers — delegate renders items now
- `renderSection()` method — replaced by list delegate
- `renderResults()` method — replaced by list.Model.View()
- `truncate()` helper — delegate uses lipgloss width capping
- Old cursor/section navigation (`moveSectionForward`, `moveSectionBackward`, `moveCursorUp`, `moveCursorDown`, `maxCursorForActiveSection`)
- Old `selectedURI()` — replaced by list.SelectedItem()
- Old `handleEnter()` and `handleAddToQueue()` — rewritten to use list selection

Remove from `messages.go`:
- Old `SearchResultsMsg` (replaced by `SearchPageLoadedMsg`)
- Any unused message types

Remove from `commands.go`:
- Old `buildSearchCmd` (replaced by `buildSearchBatchCmd`)
- Old `convertSearchResult` if replaced

Remove from `app.go`:
- Old `SearchResultsMsg` handler (replaced by `SearchPageLoadedMsg` handler)

### Theme Integration

Verify `SetTheme()` propagates to:
- `SearchItemDelegate.theme`
- `help.Model` styles
- Tab bar styles
- Input prompt style
- Prefix hint style

### Edge Cases

- Empty query after prefix lock (`:songs ` with no query) — should show empty state, not search
- Very long query string — input should scroll horizontally (textinput handles this)
- Terminal resize while overlay is open — `SetSize` propagates to all sub-components
- Opening search when results exist from last search — show cached results or clear?
  - **Decision**: Clear on open. Each search session starts fresh. `SearchOverlay.Init()` emits `SearchClearedMsg`.
- Rapid tab switching — stale result discard handles this via query+type matching
- Reaching offset 1000 — prefetch stops, no error shown

### Integration Tests

Write integration tests in `internal/app/` that verify the full flow:

1. **Basic search flow**: Open overlay → type query → debounce fires → SearchRequestMsg → batch fetch → results appear in store → overlay reads from store
2. **Tab switching**: Search "kk" on All → switch to Songs tab → store cleared → new batch fired with type=track → results populate
3. **Prefix flow**: Type `:songs kk` → prefix locks → Songs tab active → search fires with query="kk", type=track
4. **Prefetch trigger**: Load 50 items → scroll to item 31 → SearchPrefetchMsg fires → next batch loads
5. **Stale result discard**: Search "kk" → immediately search "jazz" → first results arrive with query="kk" → discarded
6. **Close and reopen**: Close overlay → reopen → results cleared → input empty

## Acceptance Criteria

- [ ] All dead code from old search implementation removed
- [ ] No unused exports, types, or helpers remain
- [ ] Theme switching updates all search overlay components
- [ ] Terminal resize propagates correctly to all sub-components
- [ ] Opening search clears previous results
- [ ] Integration tests cover all 6 scenarios listed above
- [ ] `make ci` passes (lint + tests + 80% coverage)
- [ ] No regressions in existing overlay routing (Guard 2 in routing.go)

## Tasks

- [ ] Remove dead code: old section enum, clamp helpers, render helpers, old navigation
      - test: build succeeds; no unused export warnings from lint
- [ ] Remove old message types and command builders
      - test: all references updated; no compilation errors
- [ ] Verify and fix theme propagation in `SetTheme()`
      - test: call SetTheme with different theme; delegate badge colors change; help bar updates
- [ ] Add clear-on-open behavior to `Init()`
      - test: Init emits SearchClearedMsg; store is clean after handling
- [ ] Handle edge cases: empty query after prefix, resize, offset cap
      - test: `:songs ` (no query) shows empty state; resize updates list dimensions; offset 1000 stops prefetch
- [ ] Write integration test: basic search flow
      - test: open overlay → type → debounce → batch → store populated → overlay reads results
- [ ] Write integration test: tab switching and prefix flow
      - test: tab change re-fires; prefix syncs tab; clean query sent to API
- [ ] Write integration test: prefetch and stale discard
      - test: scroll triggers prefetch; stale results ignored; close+reopen clears
