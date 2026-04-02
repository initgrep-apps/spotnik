---
title: "Search Results: bubbles/list with Custom Delegate"
feature: 19-search-redesign
status: done
---

## Background

The current search overlay renders results as manually built strings: section headers (`TRACKS`, `ARTISTS`, etc.) followed by rows of truncated text. There is no scrolling, no selection highlighting beyond a simple `▶` prefix, and results are clamped to 5 per section.

This story replaces the custom string rendering with `bubbles/list` using a custom `ItemDelegate`. The list handles scrolling (via internal viewport), keyboard navigation, and selection highlighting. A custom delegate renders each item with a type badge, name, and secondary info.

## Bubble Tea Components

This story's core component is `bubbles/list` — a Tier 1 charmbracelet component that provides scrollable lists with custom rendering, keyboard navigation, and built-in viewport.

| Component | Import | Role in This Story |
|---|---|---|
| **list** | `github.com/charmbracelet/bubbles/list` | Scrollable results area with custom `ItemDelegate` for themed rendering |
| **list.Item** | `github.com/charmbracelet/bubbles/list` | Interface implemented by `SearchListItem` (`Title()`, `Description()`, `FilterValue()`) |
| **list.ItemDelegate** | `github.com/charmbracelet/bubbles/list` | `SearchItemDelegate` renders type badge + name + subtitle per item |

**Reference**: See `/bubbletea` skill `references/components.md` for list patterns. Key APIs used:
- `list.New(items, delegate, width, height)` — constructor
- `list.SetShowTitle(false)`, `SetShowStatusBar(false)`, `SetShowFilter(false)`, `SetShowHelp(false)`, `SetShowPagination(false)` — disable built-in chrome (we render our own tab bar and help panel)
- `list.SetItems(items)` — replace items when store data changes or tab switches
- `list.Index()` — current cursor position (used for prefetch threshold check)
- `list.SelectedItem()` — get the selected `SearchListItem` for play/queue actions
- `list.SetSize(w, h)` — propagate dimensions from SetSize
- Custom `ItemDelegate.Render(w, m, index, item)` — renders each item with `fmt.Fprintf(w, ...)`, uses `m.Index()` to detect selection

**Why `bubbles/list` over `bubble-table`**: The list component handles mixed-type results naturally via custom delegates (type badges, variable subtitles), has built-in viewport scrolling for prefetch detection, and provides a richer visual experience than columnar tables for search results.

## Design

### Search Result Item Type

Define a unified item type that implements `list.Item`:

```go
// SearchListItem represents a single search result in the list.
type SearchListItem struct {
    category string // "track", "artist", "album", "playlist"
    name     string
    subtitle string // artist name for tracks/albums, owner for playlists, empty for artists
    uri      string
    isTrack  bool   // true for tracks (play as track vs context)
}

func (i SearchListItem) Title() string       { return i.name }
func (i SearchListItem) Description() string  { return i.subtitle }
func (i SearchListItem) FilterValue() string  { return i.name }
```

### Custom ItemDelegate

Create `SearchItemDelegate` implementing `list.ItemDelegate`:

```go
type SearchItemDelegate struct {
    theme theme.Theme
}

func (d SearchItemDelegate) Height() int                          { return 2 }
func (d SearchItemDelegate) Spacing() int                         { return 0 }
func (d SearchItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d SearchItemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
    si, ok := item.(SearchListItem)
    if !ok { return }

    isSelected := index == m.Index()

    // Type badge with category-specific color
    badgeColor := d.badgeColor(si.category)
    badge := lipgloss.NewStyle().
        Foreground(badgeColor).
        Bold(true).
        Render(categorySymbol(si.category))

    // Title line
    titleStyle := lipgloss.NewStyle().Foreground(d.theme.TextPrimary())
    if isSelected {
        titleStyle = titleStyle.
            Background(d.theme.SelectedBg()).
            Foreground(d.theme.SelectedFg())
    }

    // Subtitle line (secondary info)
    subtitleStyle := lipgloss.NewStyle().Foreground(d.theme.TextSecondary())

    fmt.Fprintf(w, "%s %s\n", badge, titleStyle.Render(si.name))
    if si.subtitle != "" {
        fmt.Fprintf(w, "  %s\n", subtitleStyle.Render(si.subtitle))
    }
}
```

### Category Badges

```go
func categorySymbol(category string) string {
    switch category {
    case "track":    return "♫"
    case "artist":   return "●"
    case "album":    return "◆"
    case "playlist": return "☰"
    default:         return "·"
    }
}

func (d SearchItemDelegate) badgeColor(category string) lipgloss.Color {
    switch category {
    case "track":    return d.theme.Success()       // green
    case "artist":   return d.theme.KeyHint()        // purple
    case "album":    return d.theme.SeekBar()         // cyan
    case "playlist": return d.theme.SectionHeader()   // blue
    default:         return d.theme.TextMuted()
    }
}
```

### List Integration in SearchOverlay

Add `list.Model` to `SearchOverlay`:

```go
type SearchOverlay struct {
    // ... existing fields ...
    resultList list.Model
    delegate   SearchItemDelegate
}
```

Initialize in `NewSearchOverlay`:

```go
delegate := SearchItemDelegate{theme: t}
l := list.New([]list.Item{}, delegate, 0, 0)
l.SetShowTitle(false)
l.SetShowStatusBar(false)
l.SetShowFilter(false)      // we use tabs, not list's built-in filter
l.SetShowHelp(false)        // we render help separately
l.SetShowPagination(false)  // we use prefetch, not pages
l.InfiniteScrolling = true  // wraps around
```

### Populating the List from Store

When the overlay's `Update()` receives data changes (detected via a new `SearchDataUpdatedMsg` or by reading the store on each render cycle), rebuild the list items:

```go
func (o *SearchOverlay) rebuildListItems() {
    var items []list.Item

    switch o.activeTab {
    case tabAll:
        // Interleave: tracks, artists, albums, playlists in sections
        items = append(items, tracksToListItems(o.store.SearchTracks().Items)...)
        items = append(items, artistsToListItems(o.store.SearchArtists().Items)...)
        items = append(items, albumsToListItems(o.store.SearchAlbums().Items)...)
        items = append(items, playlistsToListItems(o.store.SearchPlaylists().Items)...)
    case tabSongs:
        items = tracksToListItems(o.store.SearchTracks().Items)
    case tabArtists:
        items = artistsToListItems(o.store.SearchArtists().Items)
    case tabAlbums:
        items = albumsToListItems(o.store.SearchAlbums().Items)
    case tabPlaylists:
        items = playlistsToListItems(o.store.SearchPlaylists().Items)
    }

    o.resultList.SetItems(items)
}
```

Conversion helpers (`tracksToListItems`, etc.) map domain types to `SearchListItem`.

### Scroll-Based Prefetch Trigger

After each `list.Model` update, check if the cursor has passed the 60% threshold:

```go
func (o *SearchOverlay) checkPrefetch() tea.Cmd {
    total := len(o.resultList.Items())
    if total == 0 { return nil }

    cursor := o.resultList.Index()
    threshold := int(float64(total) * searchPrefetchThreshold)

    if cursor >= threshold {
        types := tabToAPITypes[o.activeTab]
        nextOffset := o.nextOffsetForTab()
        if nextOffset < 0 { return nil } // no more data
        return func() tea.Msg {
            return SearchPrefetchMsg{
                Query:      o.store.SearchQuery(),
                Types:      types,
                NextOffset: nextOffset,
            }
        }
    }
    return nil
}
```

### Key Routing Updates

- `up`/`down`/`j`/`k` → forwarded to `list.Model` for cursor movement
- `Enter` → get selected `SearchListItem`, emit `PlayTrackMsg` or `PlayContextMsg`
- `Ctrl+A` → get selected item, emit `AddToQueueMsg` if track
- `Tab`/`Shift+Tab` → cycle tabs (not forwarded to list)

### SetSize Propagation

When `SetSize` is called, calculate the results area height and update `list.Model`:

```go
func (o *SearchOverlay) SetSize(width, height int) {
    o.width = width
    o.height = height
    resultsHeight := o.overlayHeight() - 6 // border(2) + input(1) + sep(1) + tabs(1) + help(1)
    resultsWidth := o.overlayWidth() - 2   // border(2)
    o.resultList.SetSize(resultsWidth, resultsHeight)
}
```

## Acceptance Criteria

- [ ] `SearchListItem` implements `list.Item` with Title/Description/FilterValue
- [ ] `SearchItemDelegate` renders type badge + name + subtitle with theme colors
- [ ] Selected item highlighted with `SelectedBg()`/`SelectedFg()`
- [ ] List populated from Store based on active tab
- [ ] "All" tab shows all types interleaved with category badges
- [ ] Filtered tabs show only the selected type
- [ ] Scroll past 60% emits `SearchPrefetchMsg`
- [ ] Enter plays selected item (track vs context)
- [ ] Ctrl+A adds selected track to queue
- [ ] up/down/j/k navigate the list
- [ ] List respects overlay dimensions via SetSize
- [ ] make ci passes

## Tasks

- [ ] Define `SearchListItem` struct implementing `list.Item` in `search.go`
      - test: Title() returns name; Description() returns subtitle; FilterValue() returns name
- [ ] Implement `SearchItemDelegate` with themed rendering
      - test: Render produces output with badge symbol; selected item has different style; all 4 category badges render correctly
- [ ] Initialize `list.Model` in `NewSearchOverlay` with correct settings
      - test: list has no title, no status bar, no filter, no help, no pagination
- [ ] Implement `rebuildListItems()` and conversion helpers
      - test: tabAll includes all 4 types; tabSongs includes only tracks; empty store produces empty list
- [ ] Implement scroll-based prefetch trigger (`checkPrefetch`)
      - test: cursor at 30% → no prefetch; cursor at 60% → emits SearchPrefetchMsg; no more data → nil
- [ ] Wire list.Model into Update() for key forwarding and selection
      - test: down key moves cursor; Enter on track emits PlayTrackMsg; Enter on album emits PlayContextMsg; Ctrl+A on track emits AddToQueueMsg
- [ ] Wire SetSize to propagate to list.Model
      - test: SetSize(120, 40) → list gets correct inner dimensions
- [ ] Update View() to render list.Model output in Zone 2
      - test: View contains list items; height matches allocated zone; no overflow
