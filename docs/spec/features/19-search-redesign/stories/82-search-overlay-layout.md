---
title: "Search Overlay: Three-Panel Vertical Layout with Tabs and Help Bar"
feature: 19-search-redesign
status: open
---

## Background

The current `SearchOverlay` in `internal/ui/panes/search.go` is a compact overlay (~60%x70%) that renders everything inside a single bordered box: a text input, a dot separator, and four vertically stacked result sections. It has no tab bar, no help component, and no concept of category filtering.

This story rebuilds the `SearchOverlay` as **three visually distinct bordered panels** stacked vertically at 80% terminal size, with 1-line margins between them. Each panel is its own `layout.RenderPaneBorder()` call with rounded corners. The overlay is NOT a single box with internal divisions — it is three separate boxes composed vertically.

## Design

### Visual Layout

The overlay renders as three separate bordered panels stacked vertically. A 1-line margin separates Panel 1 (Search) from Panel 2 (Results). Panel 2 and Panel 3 (Keys) sit flush against each other with no margin:

```
╭─ Search ─────────────────────────────────────────────────────────╮
│  > :songs query text here_                                       │
│    :songs  :artists  :albums  :playlists                         │  ← hint line (only when typing prefix)
╰─────────────────────────────────────────────────────────────────╯
                                                                      ← 1-line margin
╭─ Results ─────────────────────── Enter play  Ctrl+A queue ──────╮
│  [All]  Songs  Artists  Albums  Playlists                        │  ← tab bar row
│  ─────────────────────────────────────────────────────────────── │  ← thin separator
│  ♫ Labon Ko                                    Pritam            │
│  ♫ Kya Mujhe Pyar Hai                          Pritam            │
│  ● KK                                                            │
│  ● Atif Aslam                                                    │
│  ◆ Pal                                          KK               │
│  ◆ Teri Yaadon Mein                             KK               │
│  ☰ KK Super Hit Songs                          Samiran Dey       │
│                                                                   │
│                                                                  │
╰─────────────────────────────────────────────────────────────────╯
╭─ Keys ──────────────────────────────────────────────────────────╮
│  Enter play  Ctrl+A queue  Tab filter  Shift+Tab prev  Esc close │
╰─────────────────────────────────────────────────────────────────╯
```

### Panel 1: Search Bar (top)

A bordered panel containing:
- The `textinput.Model` (1 line)
- Optional prefix hint line (1 line, only when `prefixState == prefixTyping` — see Story 85)

Height: **3 lines** (border top + input + border bottom) or **4 lines** when hint is visible.

Border config:
```go
searchBarCfg := layout.BorderConfig{
    Width:       overlayWidth,
    Height:      searchBarHeight, // 3 or 4
    Title:       "Search",
    Actions:     []layout.Action{}, // no actions in search bar border
    AccentColor: o.theme.ActiveBorder(),
    Focused:     true,
    Theme:       o.theme,
}
```

The search bar panel is always focused (bright border) since the overlay captures all input.

### Panel 2: Results (middle)

A bordered panel containing:
- Tab bar row (1 line): `[All]  Songs  Artists  Albums  Playlists`
- Thin separator (1 line): styled `─` characters in `TextMuted()`
- Results list area: fills remaining height — `list.Model` output (Story 84 wires this; until then, placeholder or existing section rendering)

Height: **overlayHeight - searchBarHeight - helpBarHeight - 2** (2 lines for margins).

This panel is the largest and expands/contracts with terminal size.

Border config:
```go
resultsCfg := layout.BorderConfig{
    Width:       overlayWidth,
    Height:      resultsHeight,
    Title:       "Results",
    Actions: []layout.Action{
        {Key: "Enter", Label: "play"},
        {Key: "Ctrl+A", Label: "queue"},
    },
    AccentColor: o.theme.SectionHeader(),
    Focused:     false, // dimmer border than search bar
    Theme:       o.theme,
}
```

#### Tab Bar Rendering

Rendered as the first line inside the results panel. Active tab gets `SelectedBg()`/`SelectedFg()` styling with brackets. Inactive tabs use `TextMuted()`:

```
 [All]  Songs  Artists  Albums  Playlists
```

#### Tab Separator

A thin line of `─` characters in `TextMuted()` below the tab bar, spanning inner width. Visually separates the tab selector from the scrollable results.

### Panel 3: Help Bar (bottom)

A bordered panel containing a single line from `help.Model`:

```
 Enter play  Ctrl+A queue  Tab filter  Shift+Tab prev  Esc close
```

Height: **3 lines** (border top + help content + border bottom).

Border config:
```go
helpCfg := layout.BorderConfig{
    Width:       overlayWidth,
    Height:      3,
    Title:       "Keys",
    Actions:     []layout.Action{},
    AccentColor: o.theme.TextMuted(),
    Focused:     false,
    Theme:       o.theme,
}
```

### View() Composition

`View()` renders the three panels and joins them with 1-line margins:

```go
func (o *SearchOverlay) View() string {
    w := o.overlayWidth()
    totalH := o.overlayHeight()

    // Panel 1: Search bar
    searchBarH := 3
    if o.prefixState == prefixTyping { searchBarH = 4 }
    searchPanel := o.renderSearchPanel(w, searchBarH)

    // Panel 3: Help bar (fixed height)
    helpH := 3
    helpPanel := o.renderHelpPanel(w, helpH)

    // Panel 2: Results (takes remaining space)
    resultsH := totalH - searchBarH - helpH - 1 // 1 margin line (between search and results only)
    if resultsH < 5 { resultsH = 5 }
    resultsPanel := o.renderResultsPanel(w, resultsH)

    // Compose: panel1 + margin + panel2 + panel3 (no margin between results and keys)
    margin := ""  // empty line as visual gap
    return lipgloss.JoinVertical(lipgloss.Left,
        searchPanel,
        margin,
        resultsPanel,
        helpPanel,
    )
}
```

Each `render*Panel()` method builds inner content, then wraps it with `layout.RenderPaneBorder(content, cfg)`.

### Height Budget

For a 40-line terminal (80% = 32 lines):

| Element | Lines |
|---|---|
| Panel 1: Search bar | 3 |
| Margin | 1 |
| Panel 2: Results | 23 |
| Panel 3: Help bar (flush) | 3 |
| **Total** | **30** |

Panel 2 and Panel 3 sit flush — no margin between them. They are still separate bordered panels (each with their own `layout.RenderPaneBorder()` call and rounded corners), but visually adjacent.

Results panel inner area: 23 - 2 (border) = 21 lines. Tab bar takes 1, separator takes 1, leaving **19 lines** for the `list.Model` — enough for ~9 items at height=2 per item.

### Overlay Size

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

var tabToAPITypes = map[searchTab][]string{
    tabAll:       {"track", "artist", "album", "playlist"},
    tabSongs:     {"track"},
    tabArtists:   {"artist"},
    tabAlbums:    {"album"},
    tabPlaylists: {"playlist"},
}
```

Add `activeTab searchTab` field to `SearchOverlay`. Default: `tabAll`.

### Tab Navigation

- `Tab` cycles active tab forward, `Shift+Tab` cycles backward (when not in prefix-typing mode — Story 85 handles that)
- Switching tab emits `SearchTabChangedMsg` which the root app handles by re-firing the search with the new type filter

```go
type SearchTabChangedMsg struct {
    Types []string
    Query string
}
```

### Help Bar (`bubbles/help`)

```go
type searchKeyMap struct {
    Play     key.Binding
    Queue    key.Binding
    TabNext  key.Binding
    TabPrev  key.Binding
    Close    key.Binding
    Clear    key.Binding
}

func (k searchKeyMap) ShortHelp() []key.Binding {
    return []key.Binding{k.Play, k.Queue, k.TabNext, k.TabPrev, k.Close}
}

func (k searchKeyMap) FullHelp() [][]key.Binding {
    return [][]key.Binding{{k.Play, k.Queue, k.TabNext, k.TabPrev, k.Close, k.Clear}}
}
```

### SetSize Propagation

When `SetSize` is called, recalculate all panel heights and propagate the results area dimensions to the `list.Model`:

```go
func (o *SearchOverlay) SetSize(width, height int) {
    o.width = width
    o.height = height

    w := o.overlayWidth()
    totalH := o.overlayHeight()

    searchBarH := 3
    helpH := 3
    resultsH := totalH - searchBarH - helpH - 1  // 1 margin (between search and results only)

    listW := w - 2          // inside results border
    listH := resultsH - 4   // border(2) + tab bar(1) + separator(1)
    o.resultList.SetSize(listW, listH)
}
```

## Acceptance Criteria

- [ ] Overlay renders as **three separate bordered panels** stacked vertically
- [ ] 1-line margin between Panel 1 (Search) and Panel 2 (Results); Panel 2 and Panel 3 (Keys) sit flush with no margin
- [ ] Panel 1 (Search bar): rounded border, title "Search", contains textinput
- [ ] Panel 2 (Results): rounded border, title "Results", actions in border, contains tab bar + separator + results area
- [ ] Panel 3 (Help bar): rounded border, title "Keys", contains `help.Model` output
- [ ] Total overlay size = 80% of terminal width and height
- [ ] Tab bar renders inside Panel 2 with active tab highlighted
- [ ] Tab/Shift+Tab cycles active tab
- [ ] Tab change emits `SearchTabChangedMsg` with correct API types
- [ ] SetSize propagates to all sub-components including list.Model
- [ ] Existing key handlers (Esc, Enter, Ctrl+A, Ctrl+U) still work
- [ ] make ci passes

## Tasks

- [ ] Define `searchTab` enum, labels, and API type mapping in `search.go`
      - test: tabToAPITypes returns correct types for each tab; numTabs == 5
- [ ] Add `activeTab` field and tab navigation methods to `SearchOverlay`
      - test: Tab cycles forward wrapping; Shift+Tab cycles backward wrapping; default is tabAll
- [ ] Add `SearchTabChangedMsg` to `messages.go` and emit on tab change
      - test: switching tab emits msg with correct Types and current Query
- [ ] Implement `renderSearchPanel()` — Panel 1 with textinput inside its own border
      - test: panel output starts with `╭`; contains "Search" title; inner content has input prompt; height is 3 (or 4 with hints)
- [ ] Implement `renderResultsPanel()` — Panel 2 with tab bar, separator, and results area inside its own border
      - test: panel contains tab labels; active tab has highlight; separator line present; border has "Results" title and actions
- [ ] Implement `renderHelpPanel()` — Panel 3 with help.Model inside its own border
      - test: panel contains keybinding text; border has "Keys" title; height is exactly 3
- [ ] Implement `searchKeyMap` and `help.Model` integration
      - test: ShortHelp returns 5 bindings; FullHelp returns all 6; help.View() produces non-empty string
- [ ] Rewrite `View()` to compose three panels with margins via `lipgloss.JoinVertical`
      - test: View output contains three `╭` border starts; one margin line between search and results; results and keys flush; total height matches overlayHeight
- [ ] Update `overlayWidth`/`overlayHeight` to 80%
      - test: 100-wide terminal → overlay 80 wide; 40-high terminal → overlay 32 high; minimum clamps work
- [ ] Update `SetSize` to propagate dimensions to list.Model
      - test: SetSize(120, 40) → list gets correct inner dimensions accounting for all 3 panels + margins
- [ ] Wire `SearchTabChangedMsg` handler in `app.go` to re-fire search
      - test: receiving SearchTabChangedMsg updates store.SearchActiveType and dispatches buildSearchCmd with correct types
