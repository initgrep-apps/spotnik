---
title: "Search Redesign"
status: done
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

### Bubble Tea Components (Mandatory)

Implementers **must** use these specific Bubble Tea components. Refer to the `/bubbletea` skill (`references/components.md`) for API signatures, usage patterns, and initialization examples.

| Component | Import | Where Used | Story |
|---|---|---|---|
| **textinput** | `github.com/charmbracelet/bubbles/textinput` | Panel 1: search bar with `:prefix` autocomplete. `SetValue()` for Tab completion, `Value()` for prefix parsing. | 82, 85 |
| **spinner** | `github.com/charmbracelet/bubbles/spinner` | Panel 2: loading indicator during API fetch. `spinner.Dot` style, themed with `TextMuted()`. | 82 |
| **list** | `github.com/charmbracelet/bubbles/list` | Panel 2: scrollable results with custom `ItemDelegate`. Replaces all manual string rendering. Handles viewport scroll, keyboard nav, selection. Disable built-in title/filter/help/pagination (we render our own). | 84 |
| **list.ItemDelegate** | `github.com/charmbracelet/bubbles/list` | Custom `SearchItemDelegate` renders type badge + name + subtitle. `Height()=2`, `Render()` writes to `io.Writer`. | 84 |
| **help** | `github.com/charmbracelet/bubbles/help` | Panel 3: keybinding bar. `help.New()` + `help.View(searchKeyMap)`. | 82 |
| **key** | `github.com/charmbracelet/bubbles/key` | Defines `key.Binding` entries for the `searchKeyMap` used by help component. `key.NewBinding(key.WithKeys(...), key.WithHelp(...))`. | 82 |
| **tea.Sequence** | `github.com/charmbracelet/bubbletea` | Prefetch engine: sequences 5 page-fetch commands so they execute in order (NOT `tea.Batch`). | 83 |

**Existing components (no changes needed):**

| Component | Import | Where Used |
|---|---|---|
| **bubbletea-overlay** | `github.com/rmhubbert/bubbletea-overlay` | Composites search overlay on dimmed background (`btoverlay.Composite`) |
| **RenderPaneBorder** | `internal/ui/layout` | Wraps each of the 3 panels in btop-style rounded-corner borders |
| **lipgloss** | `github.com/charmbracelet/lipgloss` | Styling, `JoinVertical` for panel composition, width/height capping |

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
- [ ] Rich metadata in results: tracks show all artists, album, duration, explicit; artists show genres, followers; albums show type, year, track count; playlists show owner, track count
- [ ] All result metadata styled with distinct theme color tokens
- [ ] Enter plays selected item without closing overlay (only Esc closes)
- [ ] Overlay width reduced to 70% of terminal
- [ ] Category icons are theme-colorable Unicode symbols (♪ ★ ◎ ▤), not emoji
- [ ] Prefix commands discoverable via placeholder text and ghost hints
- [ ] Animated cycling placeholder shows prefix commands in theme color when input is empty
- [ ] Native inline ghost completion for prefixes via `textinput.SetSuggestions`
- [ ] Locked prefix shown as styled tag/chip in textinput Prompt with colored background
- [ ] Hint pills row with category badge colors, dimming non-matching prefixes
- [ ] Tab/Shift+Tab tab switching syncs bidirectionally with Prompt-based prefix tag
- [ ] Three panels render flush (zero margin), hints inside Search panel, per-panel border colors
- [ ] make ci passes
