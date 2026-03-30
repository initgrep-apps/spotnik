---
title: "Search Overlay with Live Results"
feature: 05-search
status: done
---

## Background
This story implements the complete search experience: the API client for Spotify's search endpoint, the 300ms debounce mechanism to prevent excessive API calls, the SearchOverlay Bubble Tea model with full keyboard navigation, result rendering with section grouping and truncation, and the focus management wiring in the root app model that makes the overlay appear and disappear cleanly.

## Design

### Store fields
```go
SearchResults *api.SearchResult // nil until first search completes
SearchQuery   string
SearchLoading bool
```

### Debounce pattern (300ms)
```go
// On each keypress: store current query in m.pendingQuery, schedule a tick
func debounceCmd(query string) tea.Cmd {
    return tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
        return searchDebounceMsg{query: query}
    })
}
// In Update on searchDebounceMsg: only dispatch search if msg.query == m.currentQuery
```

### Overlay rendering
SearchPane is a floating overlay. Root app renders main view dimmed (`lipgloss.NewStyle().Faint(true)`) then renders search overlay on top via `lipgloss.Place()`.

### Overlay dimensions
- Width: 50 chars or 60% of terminal width, whichever is smaller
- Height: dynamic based on results, max 60% of terminal height

### Message types
```go
type searchDebounceMsg  struct{ query string }
type searchResultsMsg   struct{ results *api.SearchResult }
type searchErrMsg       struct{ err error }
type searchClosedMsg    struct{}
```

### Design tokens
`theme.SurfaceAlt()` . `theme.ActiveBorder()` . `theme.TextPrimary()` . `theme.TextMuted()` . `theme.SelectedBg()` . `theme.SelectedFg()` . `theme.Error()`

### Search Overlay Layout

```
+-------------------------------------+
|  Search                             |
|  ...............................    |
|  > blinding lig_                    |
|  ...............................    |
|                                     |
|  * TRACKS                           |
|  > Blinding Lights   . The Weeknd   |  <- selected
|    Blinding Lights   . Sunday Ser.  |
|    Blinding Light    . Maroon 5     |
|                                     |
|  * ARTISTS                          |
|    The Weeknd                       |
|    Sunday Service Choir             |
|                                     |
|  * PLAYLISTS                        |
|    Blinding Pop Hits                |
|    Late Night Drives                |
|                                     |
+-------------------------------------+
```

### Spotify Search API Usage

```
GET /search?q={query}&type=track,artist,album,playlist&limit=5&market=from_token
```

### Keymap (Search Overlay)

| Key | Action |
|---|---|
| Any character | Append to search query |
| `Backspace` | Delete last character |
| `j` / `down` | Move selection down |
| `k` / `up` | Move selection up |
| `Tab` | Jump to next section |
| `Shift+Tab` | Jump to previous section |
| `Enter` | Play selected item |
| `a` | Add selected item to queue (tracks only) |
| `Esc` | Close search, restore previous state |
| `Ctrl+A` | Select all in input |
| `Ctrl+U` | Clear entire input |

### Play Behavior from Search

| Selected Result Type | Play Action |
|---|---|
| Track | `PUT /me/player/play` with `uris: [track_uri]` |
| Artist | `PUT /me/player/play` with `context_uri: artist_uri` |
| Album | `PUT /me/player/play` with `context_uri: album_uri` |
| Playlist | `PUT /me/player/play` with `context_uri: playlist_uri` |

### Files

| File | Purpose |
|---|---|
| `internal/api/search.go` | Search API call |
| `internal/api/search_test.go` | Tests with fixture JSON |
| `internal/ui/panes/search.go` | SearchOverlay model |
| `internal/ui/panes/search_test.go` | Update tests |

### Out of Scope
- Podcast/show search results
- Audiobook search results
- Search history / recent searches
- Offline search suggestions

## Acceptance Criteria
- [ ] `/` opens search overlay from any pane without disrupting current state
- [ ] Results appear within 400ms of last keypress (300ms debounce + ~100ms API)
- [ ] `Enter` plays a track and closes the overlay
- [ ] `a` adds track to queue, shows status bar confirmation, keeps overlay open
- [ ] `Esc` closes overlay, previous pane is focused and unchanged
- [ ] Typing faster than 300ms between keys fires only one search (debounce works)
- [ ] Empty query shows hint text, no API call fired
- [ ] All search API functions and overlay handlers tested

## Tasks
- [ ] Search API -- Implement Spotify search endpoint wrapper
      - test: `TestSearch_Success`, `TestSearch_EmptyResults`, `TestSearch_ServerError`, `TestSearch_InvalidJSON`, `TestSearch_RequestParams`
- [ ] Debounce logic -- Implement 300ms debounce mechanism
      - test: `TestDebounce_StaleQueryIgnored`, `TestDebounce_CurrentQueryFires`, `TestDebounce_EmptyQueryNoFire`
- [ ] SearchOverlay model -- Build Bubble Tea model with textinput and keyboard navigation
      - test: `TestSearchOverlay_Init_FocusesInput`, `TestSearchOverlay_Update_Typing`, `TestSearchOverlay_Update_Backspace`, `TestSearchOverlay_Update_Enter`, `TestSearchOverlay_Update_Esc`, `TestSearchOverlay_Update_A`, `TestSearchOverlay_Update_Tab`, `TestSearchOverlay_Update_ShiftTab`, `TestSearchOverlay_Update_JK`, `TestSearchOverlay_Update_CtrlU`
- [ ] Result rendering -- Implement View() for search results with section grouping and truncation
      - test: `TestSearchOverlay_View_Results`, `TestSearchOverlay_View_SelectedHighlight`, `TestSearchOverlay_View_Truncation`, `TestSearchOverlay_View_EmptyQuery`, `TestSearchOverlay_View_NoResults`, `TestSearchOverlay_View_Loading`
- [ ] Overlay focus management -- Wire search overlay into root app with `/` open and `Esc` close
      - test: `TestApp_SlashOpensSearch`, `TestApp_EscClosesSearch`, `TestApp_SearchPlayClosesOverlay`, `TestApp_BackgroundDimmed`
