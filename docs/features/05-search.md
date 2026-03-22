# Feature 05 — Search

> **Depends on:** Feature 02 (Auth). Can be built in parallel with Feature 04.

## Implementation Context

### Store fields this feature uses
```go
SearchResults *api.SearchResults // nil until first search completes
```

### Debounce pattern (300ms — never fire on every keystroke)
```go
// On each keypress: store current query in m.pendingQuery, schedule a tick
func debounceCmd(query string) tea.Cmd {
    return tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
        return searchDebounceMsg{query: query}
    })
}
// In Update on searchDebounceMsg: only dispatch search if msg.query == m.currentQuery
// (discards stale ticks from earlier keypresses)
```

### Overlay rendering
SearchPane is a floating overlay. Root app:
1. Renders main three-pane view dimmed (`lipgloss.NewStyle().Faint(true)`)
2. Renders search overlay on top via `lipgloss.Place()`
Search pane does **not** replace any pane — it layers above.

### Message types for this feature
```go
type searchDebounceMsg  struct{ query string }
type searchResultsMsg   struct{ results *api.SearchResults }
type searchClosedMsg    struct{}
```

### Design tokens used in this feature
`theme.SurfaceAlt()` · `theme.ActiveBorder()` · `theme.TextPrimary()` ·
`theme.TextMuted()` · `theme.SelectedBg()` · `theme.SelectedFg()`

---

---

## Goal

A fast, keyboard-native search overlay. Press `/`, type, and see live results grouped by
tracks, artists, albums, and playlists. Play directly or add to queue without leaving the overlay.

---

## User Stories

- **As a user**, I press `/` from anywhere in the app to open the search overlay.
- **As a user**, I start typing and see results appear after 300ms without pressing Enter.
- **As a user**, I see results grouped into Tracks, Artists, Albums, Playlists sections.
- **As a user**, I press `j/k` to navigate results, `Tab` to jump between sections.
- **As a user**, I press `Enter` on a track to play it immediately.
- **As a user**, I press `a` on a result to add it to the queue.
- **As a user**, I press `Esc` to close search and return to where I was.
- **As a user**, empty query shows nothing (no search fired on empty string).

---

## Search Overlay Layout (from DESIGN.md)

```
╭─────────────────────────────────────╮
│  🔍 Search                          │
│  ┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄  │
│  > blinding lig█                    │
│  ┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄  │
│                                     │
│  ● TRACKS                           │
│  ▶ Blinding Lights   · The Weeknd   │  ← selected
│    Blinding Lights   · Sunday Ser.  │
│    Blinding Light    · Maroon 5     │
│                                     │
│  ● ARTISTS                          │
│    The Weeknd                       │
│    Sunday Service Choir             │
│                                     │
│  ● PLAYLISTS                        │
│    Blinding Pop Hits                │
│    Late Night Drives                │
│                                     │
╰─────────────────────────────────────╯
```

- Overlay width: 50 chars (or 60% of terminal width, whichever is smaller)
- Overlay height: dynamic, max 60% of terminal height
- Background behind overlay: dimmed (50% opacity simulation via color)
- Input cursor: blinking block (use Bubbles `textinput`)

---

## Search Behavior

### Debouncing
- Fire search 300ms after the last keypress
- If user types faster than 300ms between keys, reset the timer
- Implementation: use `tea.Tick` with a debounce flag in state

```go
// Pseudo-logic in Update():
case tea.KeyMsg:
    m.query = newQuery
    m.debounceTimer = true
    return m, tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
        return searchDebounceMsg{query: newQuery}
    })

case searchDebounceMsg:
    if msg.query != m.query {
        return m, nil  // query changed since timer set — ignore
    }
    return m, doSearch(m.client, m.query)
```

### Results Grouping
Search returns results grouped into:
1. **Tracks** — show name + artist (truncated to fit)
2. **Artists** — show name
3. **Albums** — show name + artist
4. **Playlists** — show name + owner

Show max 5 results per section in the overlay (prevent overflow).

### Empty / Loading States
- Empty query: show hint text "Type to search tracks, artists, albums..."
- Loading (during 300ms debounce or API call): show spinner next to query input
- No results: show "No results for '{query}'"
- Error: show "Search failed. Try again." in Pink

---

## Spotify Search API Usage

```
GET /search?q={query}&type=track,artist,album,playlist&limit=5&market=from_token
```

- Always include `market=from_token` (uses user's country)
- `limit=5` per type (returns up to 5 of each)
- `type=track,artist,album,playlist` — always search all four simultaneously

---

## Keymap (Search Overlay)

| Key | Action |
|---|---|
| Any character | Append to search query |
| `Backspace` | Delete last character |
| `j` / `↓` | Move selection down |
| `k` / `↑` | Move selection up |
| `Tab` | Jump to next section (Tracks → Artists → Albums → Playlists) |
| `Shift+Tab` | Jump to previous section |
| `Enter` | Play selected item |
| `a` | Add selected item to queue (tracks only) |
| `Esc` | Close search, restore previous state |
| `Ctrl+A` | Select all in input (for easy clearing) |
| `Ctrl+U` | Clear entire input |

---

## Play Behavior from Search

| Selected Result Type | Play Action |
|---|---|
| Track | `PUT /me/player/play` with `uris: [track_uri]` |
| Artist | `PUT /me/player/play` with `context_uri: artist_uri` |
| Album | `PUT /me/player/play` with `context_uri: album_uri` |
| Playlist | `PUT /me/player/play` with `context_uri: playlist_uri` |

---

## Files to Create

| File | Purpose |
|---|---|
| `internal/api/search.go` | Search API call |
| `internal/api/search_test.go` | Tests with fixture JSON |
| `internal/ui/panes/search.go` | SearchOverlay model |
| `internal/ui/panes/search_test.go` | Update tests |

---

## Task Breakdown

### Task 4.1 — Search API
- [ ] `Search(ctx, query string, types []string, limit int) (*SearchResult, error)`
- [ ] `SearchResult` struct: `Tracks`, `Artists`, `Albums`, `Playlists` fields
- [ ] Each field: items list + total count
- [ ] Test with fixture JSON (`testdata/fixtures/search_result.json`)

### Task 4.2 — Debounce logic
- [ ] Debounce state: last query string + pending flag
- [ ] `searchDebounceMsg` with query snapshot
- [ ] On receive: check if query still matches current input, only fire if so
- [ ] Test: rapid keystrokes only fire one search

### Task 4.3 — SearchOverlay model
- [ ] Implement `tea.Model`: `Init()`, `Update()`, `View()`
- [ ] Embed `bubbles/textinput` for the query input
- [ ] Track active section + cursor position within sections
- [ ] `View()` renders overlay with current results from store
- [ ] Test: all key handlers, section navigation, empty state

### Task 4.4 — Result rendering
- [ ] Render each section with its header `● TRACKS`
- [ ] Truncate long names to fit overlay width
- [ ] Selected item: `Lavender` background, `▶` prefix if currently playing
- [ ] Test: truncation at various widths, active selection highlight

### Task 4.5 — Overlay focus management
- [ ] Root app: on `/` keypress, set `m.searchOpen = true`, focus SearchOverlay
- [ ] Root app: on `Esc` in search, set `m.searchOpen = false`, restore previous focus
- [ ] Background panes dimmed when overlay active (render with `Overlay0` text color)
- [ ] Test: focus routing through root model

---

## Acceptance Criteria

- [ ] `/` opens search overlay from any pane without disrupting current state
- [ ] Results appear within 400ms of last keypress (300ms debounce + ~100ms API)
- [ ] `Enter` plays a track and closes the overlay
- [ ] `a` adds track to queue, shows status bar confirmation, keeps overlay open
- [ ] `Esc` closes overlay, previous pane is focused and unchanged
- [ ] Typing too fast doesn't cause multiple overlapping API calls
- [ ] All search API functions and overlay handlers tested

---

## Out of Scope

- Podcast/show search results (filter them out)
- Audiobook search results
- Search history / recent searches
- Offline search suggestions

---

*Last updated: 2026-02-21*
