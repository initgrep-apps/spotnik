---
title: "Search Overlay: Three-Zone Layout with Tabs and Help Bar"
feature: 19-search-redesign
status: open
---

## Background

The current `SearchOverlay` in `internal/ui/panes/search.go` is a compact overlay (~60%x70%) that renders a text input, a dot separator, and four vertically stacked result sections with custom string rendering. It has no tab bar, no help component, and no concept of category filtering.

This story rebuilds the `SearchOverlay` as a three-zone layout at 80% terminal size:
1. **Zone 1 (top)**: `textinput.Model` search bar — preserved from current, resized
2. **Zone 2 (middle)**: Tab bar + `list.Model` results area (Story 84 wires the list; this story sets up the layout skeleton and tab state)
3. **Zone 3 (bottom)**: `help.Model` keybinding bar

This story focuses on the structural layout, tab bar rendering/navigation, and help bar integration. The `list.Model` wiring and custom delegate are in Story 84. Until then, Zone 2 renders a placeholder or the existing section-based rendering.

## Design

### Overlay Size

Change `overlayWidth()` and `overlayHeight()` to return 80% of terminal dimensions:

```go
func (o *SearchOverlay) overlayWidth() int {
    w := o.width * 80 / 100
    if w < 40 { w = 40 }
    return w
}

func (o *SearchOverlay) overlayHeight() int {
    h := o.height * 80 / 100
    if h < 15 { h = 15 }
    return h
}
```

### Tab State

Add tab tracking to `SearchOverlay`:

```go
type searchTab int

const (
    tabAll       searchTab = iota
    tabSongs
    tabArtists
    tabAlbums
    tabPlaylists
    numTabs      = 5
)

var tabLabels = [numTabs]string{"All", "Songs", "Artists", "Albums", "Playlists"}

// Maps tab → API type parameter
var tabToAPITypes = map[searchTab][]string{
    tabAll:       {"track", "artist", "album", "playlist"},
    tabSongs:     {"track"},
    tabArtists:   {"artist"},
    tabAlbums:    {"album"},
    tabPlaylists: {"playlist"},
}
```

Add `activeTab searchTab` field to `SearchOverlay`. Default: `tabAll`.

### Tab Bar Rendering

Render a horizontal row of tab labels. The active tab gets `SelectedBg()`/`SelectedFg()` styling with brackets. Inactive tabs use `TextMuted()`:

```
 [All]  Songs  Artists  Albums  Playlists
```

Tab bar is rendered between the separator line and the results area in `View()`.

### Tab Navigation

`Tab` key behavior changes based on context:
- When input has a `:prefix` being typed → `Tab` completes the prefix (Story 85)
- Otherwise → `Tab` cycles active tab forward, `Shift+Tab` cycles backward
- Switching tab emits `SearchTabChangedMsg{Tab, Query}` which the root app handles by re-firing the search with the new type filter

```go
type SearchTabChangedMsg struct {
    Types []string // API type values for the selected tab
    Query string
}
```

### Help Bar (`bubbles/help`)

Add `help.Model` and a `searchKeyMap` struct implementing `help.KeyMap`:

```go
type searchKeyMap struct {
    Search   key.Binding
    Play     key.Binding
    Queue    key.Binding
    TabNext  key.Binding
    TabPrev  key.Binding
    Close    key.Binding
    Clear    key.Binding
}

func (k searchKeyMap) ShortHelp() []key.Binding {
    return []key.Binding{k.Play, k.Queue, k.TabNext, k.Close}
}

func (k searchKeyMap) FullHelp() [][]key.Binding {
    return [][]key.Binding{{k.Search, k.Play, k.Queue, k.TabNext, k.TabPrev, k.Close, k.Clear}}
}
```

The help bar renders at the bottom of the overlay via `h.View(searchKeys)`, styled with `TextMuted()`.

### View() Rewrite

```
┌──────────────── Search ─── Enter play  Tab filter  Esc close ──┐
│  > query text here_                                             │  ← Zone 1: textinput
│  ·····························································  │  ← separator
│  [All]  Songs  Artists  Albums  Playlists                       │  ← tab bar
│                                                                 │
│  (results area — placeholder until Story 84)                    │  ← Zone 2: list.Model
│                                                                 │
│                                                                 │
│  Enter play  Ctrl+A queue  Tab next  Shift+Tab prev  Esc close  │  ← Zone 3: help bar
└─────────────────────────────────────────────────────────────────┘
```

Height budget:
- Border: 2 lines (top + bottom)
- Input: 1 line
- Separator: 1 line
- Tab bar: 1 line
- Help bar: 1 line
- Results area: `totalHeight - 6` lines

### Border Actions Update

Update the `layout.BorderConfig` actions to reflect the new keybindings:

```go
Actions: []layout.Action{
    {Key: "Enter", Label: "play"},
    {Key: "Tab", Label: "filter"},
    {Key: "Esc", Label: "close"},
},
```

## Acceptance Criteria

- [ ] Overlay renders at 80% terminal width and height
- [ ] Three-zone layout: input + separator, tab bar + results, help bar
- [ ] Tab bar renders with active tab highlighted
- [ ] Tab/Shift+Tab cycles active tab
- [ ] Tab change emits `SearchTabChangedMsg` with correct API types
- [ ] `bubbles/help` renders keybindings at bottom
- [ ] Border actions updated
- [ ] Existing key handlers (Esc, Enter, Ctrl+A, Ctrl+U) still work
- [ ] make ci passes

## Tasks

- [ ] Define `searchTab` enum, labels, and API type mapping in `search.go`
      - test: tabToAPITypes returns correct types for each tab; numTabs == 5
- [ ] Add `activeTab` field and tab navigation methods to `SearchOverlay`
      - test: Tab cycles forward wrapping; Shift+Tab cycles backward wrapping; default is tabAll
- [ ] Add `SearchTabChangedMsg` to `messages.go` and emit on tab change
      - test: switching tab emits msg with correct Types and current Query
- [ ] Implement `searchKeyMap` and `help.Model` integration
      - test: ShortHelp returns 4 bindings; FullHelp returns all 7; help.View() produces non-empty string
- [ ] Rewrite `View()` with three-zone layout and tab bar rendering
      - test: View output contains tab labels; active tab has highlight; help bar visible; height matches overlay dimensions
- [ ] Update `overlayWidth`/`overlayHeight` to 80%
      - test: 100-wide terminal → overlay 80 wide; 50-high terminal → overlay 40 high; minimum clamps work
- [ ] Wire `SearchTabChangedMsg` handler in `app.go` to re-fire search
      - test: receiving SearchTabChangedMsg updates store.SearchActiveType and dispatches buildSearchCmd with correct types
