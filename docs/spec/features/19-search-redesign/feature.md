---
title: "Search Redesign"
status: open
---

## Description

Redesign the search overlay from a compact 60%-width floating panel with 5 hardcoded results per section into a full-featured 80%-screen overlay with three vertical zones: a prominent search bar with inline command prefixes (`:songs`, `:artists`, `:albums`, `:playlists`), a tabbed results area (All | Songs | Artists | Albums | Playlists) using `bubbles/list` with custom delegates, and a context-sensitive keybinding help bar via `bubbles/help`.

The current search fetches 5 items per type in a single API call with no pagination. The redesign adds a prefetch pagination engine: initial search fires 5 sequential API calls (offset 0..40, limit=10 each) yielding 50 results per type, then prefetches the next 5-page batch when the user scrolls past 60% of loaded items. Results are stored per-type in the Store (Elm architecture) — the overlay reads from the Store, never from API directly.

Tab switching always re-fires the search with the selected type filter to ensure fresh, complete data for the chosen category.

### Key Changes from Current Implementation

| Aspect | Current (Feature 05) | Redesign (Feature 19) |
|---|---|---|
| Overlay size | 60% width, 70% height | 80% width, 80% height |
| Results per type | 5, single API call | 50+ via 5-page prefetch batches |
| Pagination | None | Cursor-based prefetch at 60% scroll |
| Category filtering | None (all 4 shown) | Tab bar + `:prefix` input commands |
| Results component | Custom string rendering | `bubbles/list` with custom ItemDelegate |
| Help bar | Actions in border only | Dedicated `bubbles/help` zone at bottom |
| Store design | Single `SearchResult` blob | Per-type paginated storage with offset/total |

### Components Used

- `bubbles/textinput` — search bar with `:prefix` autocomplete
- `bubbles/list` — scrollable results with custom delegate per view mode
- `bubbles/help` + `bubbles/key` — context-sensitive keybinding bar
- `bubbles/spinner` — loading state
- `bubbletea-overlay` — overlay compositing (existing)
- `layout.RenderPaneBorder` — btop-style border (existing)

## Acceptance Criteria

- [ ] Search overlay renders at 80% terminal width and height
- [ ] Three-zone layout: search bar, tabbed results, help bar
- [ ] Tab bar with All | Songs | Artists | Albums | Playlists, cycled via Tab/Shift+Tab
- [ ] Input command prefixes (`:songs`, `:artists`, `:albums`, `:playlists`) with inline autocomplete hints
- [ ] `bubbles/list` with custom delegate renders results (type badge + name + secondary info)
- [ ] Initial search prefetches 5 pages (50 items per type) via sequential API calls
- [ ] Scroll past 60% triggers next 5-page prefetch batch
- [ ] Tab switching re-fires search with filtered type
- [ ] Per-type paginated results stored in Store (Elm architecture)
- [ ] `bubbles/help` renders context-sensitive keybindings at bottom
- [ ] All existing actions preserved: Enter=play, Ctrl+A=add to queue, Esc=close
- [ ] 300ms debounce on input (existing behavior preserved)
- [ ] make ci passes
