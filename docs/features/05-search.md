# Feature 05 — Search

> **Depends on:** Feature 02 (Auth). Can be built in parallel with Feature 04.

## Goal

A fast, keyboard-native search overlay. Press `/`, type, and see live results grouped by
tracks, artists, albums, and playlists. Play directly or add to queue without leaving the overlay.

---

## Feature Acceptance Criteria

- [ ] `/` opens search overlay from any pane without disrupting current state
- [ ] Results appear within 400ms of last keypress (300ms debounce + ~100ms API)
- [ ] `Enter` plays a track and closes the overlay
- [ ] `a` adds track to queue, shows status bar confirmation, keeps overlay open
- [ ] `Esc` closes overlay, previous pane is focused and unchanged
- [ ] Typing faster than 300ms between keys fires only one search (debounce works)
- [ ] Empty query shows hint text, no API call fired
- [ ] All search API functions and overlay handlers tested

---

## Implementation Context

### Store fields this feature uses
```go
SearchResults *api.SearchResult // nil until first search completes
SearchQuery   string
SearchLoading bool
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

### Overlay dimensions
- Overlay width: 50 chars or 60% of terminal width, whichever is smaller
- Overlay height: dynamic based on results, max 60% of terminal height
- Background dimmed using `lipgloss.NewStyle().Faint(true)` on the three-pane view

### Message types for this feature
```go
type searchDebounceMsg  struct{ query string }
type searchResultsMsg   struct{ results *api.SearchResult }
type searchErrMsg       struct{ err error }
type searchClosedMsg    struct{}
```

### Design tokens used in this feature
`theme.SurfaceAlt()` · `theme.ActiveBorder()` · `theme.TextPrimary()` ·
`theme.TextMuted()` · `theme.SelectedBg()` · `theme.SelectedFg()` · `theme.Error()`

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

### Empty / Loading / Error States
- Empty query: show hint text "Type to search tracks, artists, albums..."
- Loading (during 300ms debounce or API call): show spinner next to query input
- No results: show "No results for '{query}'"
- Error: show "Search failed. Try again." in `Error()` token color in the results area
- Error auto-dismisses on next successful search

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

**Description:**
Implement the Spotify search endpoint wrapper. The function accepts a query, types list, and
limit, calls `GET /search`, and returns a parsed `SearchResult` struct containing tracks,
artists, albums, and playlists.

**Files:** `internal/api/search.go`, `internal/api/search_test.go`

**Implementation steps:**
- [ ] `Search(ctx, query string, types []string, limit int) (*SearchResult, error)`
- [ ] `SearchResult` struct: `Tracks`, `Artists`, `Albums`, `Playlists` fields
- [ ] Each field: items list + total count
- [ ] Test with fixture JSON (`testdata/fixtures/search_result.json`)

**Acceptance criteria:**
- [ ] `Search()` sends correct query, type, limit, and market params to Spotify API
- [ ] Returns a fully parsed `SearchResult` with all four section types populated
- [ ] Returns descriptive error on non-2xx responses and invalid JSON

**Tests:**

*Unit tests:*
- `TestSearch_Success` — returns parsed SearchResult with tracks, artists, albums, playlists
- `TestSearch_EmptyResults` — returns empty slices, no error
- `TestSearch_ServerError` — returns descriptive error
- `TestSearch_InvalidJSON` — returns parse error
- `TestSearch_RequestParams` — verifies query, type, limit, market params in request

---

### Task 4.2 — Debounce logic

**Description:**
Implement the 300ms debounce mechanism so rapid keypresses do not fire overlapping searches.
Each keypress schedules a `searchDebounceMsg`; when the message arrives, it only fires the
search if the query has not changed since the tick was scheduled.

**Files:** `internal/ui/panes/search.go`

**Implementation steps:**
- [ ] Debounce state: last query string + pending flag
- [ ] `searchDebounceMsg` with query snapshot
- [ ] On receive: check if query still matches current input, only fire if so
- [ ] Test: rapid keystrokes only fire one search

**Acceptance criteria:**
- [ ] A burst of keypresses within 300ms produces exactly one API call
- [ ] Stale debounce ticks (where query has changed) are silently discarded
- [ ] Empty query debounce tick does not fire a search

**Tests:**

*Unit tests:*
- `TestDebounce_StaleQueryIgnored` — searchDebounceMsg with old query returns nil cmd
- `TestDebounce_CurrentQueryFires` — searchDebounceMsg matching current query fires search cmd
- `TestDebounce_EmptyQueryNoFire` — empty query does not fire search

---

### Task 4.3 — SearchOverlay model

**Description:**
Build the `SearchOverlay` Bubble Tea model. It embeds a `bubbles/textinput` for the query,
tracks the active section and cursor position within sections, and handles all keybindings
defined in the search keymap.

**Files:** `internal/ui/panes/search.go`, `internal/ui/panes/search_test.go`

**Implementation steps:**
- [ ] Implement `tea.Model`: `Init()`, `Update()`, `View()`
- [ ] Embed `bubbles/textinput` for the query input
- [ ] Track active section + cursor position within sections
- [ ] `View()` renders overlay with current results from store
- [ ] Test: all key handlers, section navigation, empty state

**Acceptance criteria:**
- [ ] Text input is focused on init, cursor is visible
- [ ] All keymap keys produce the correct commands or state changes
- [ ] Section navigation wraps correctly (Tab from last section goes to first)
- [ ] `j/k` navigation stays within the current section bounds

**Tests:**

*Unit tests:*
- `TestSearchOverlay_Init_FocusesInput` — text input focused on init
- `TestSearchOverlay_Update_Typing` — appends to query, schedules debounce tick
- `TestSearchOverlay_Update_Backspace` — removes last char
- `TestSearchOverlay_Update_Enter` — returns play command for selected item
- `TestSearchOverlay_Update_Esc` — returns searchClosedMsg
- `TestSearchOverlay_Update_A` — returns add-to-queue command for selected track
- `TestSearchOverlay_Update_Tab` — moves to next section
- `TestSearchOverlay_Update_ShiftTab` — moves to previous section
- `TestSearchOverlay_Update_JK` — moves selection within section
- `TestSearchOverlay_Update_CtrlU` — clears input

---

### Task 4.4 — Result rendering

**Description:**
Implement the `View()` rendering for search results. Each section (Tracks, Artists, Albums,
Playlists) has a header, items are truncated to fit the overlay width, and the selected item
is highlighted using the `SelectedBg()` token.

**Files:** `internal/ui/panes/search.go`

**Implementation steps:**
- [ ] Render each section with its header `● TRACKS`
- [ ] Truncate long names to fit overlay width
- [ ] Selected item: `SelectedBg()` token background, `▶` prefix if currently playing
- [ ] Test: truncation at various widths, active selection highlight

**Acceptance criteria:**
- [ ] Section headers render with correct labels and `SectionHeader` styling
- [ ] Items exceeding overlay width are truncated with ellipsis
- [ ] Selected item uses `SelectedBg()` token, not a hardcoded color
- [ ] Empty query shows hint text; no-results state shows "No results for '{query}'"
- [ ] Loading state shows spinner in the results area
- [ ] Error state shows "Search failed. Try again." in `Error()` token color

**Tests:**

*Unit tests:*
- `TestSearchOverlay_View_Results` — renders section headers and items
- `TestSearchOverlay_View_SelectedHighlight` — selected item has SelectedBg styling
- `TestSearchOverlay_View_Truncation` — long names truncated to fit overlay width
- `TestSearchOverlay_View_EmptyQuery` — shows hint text
- `TestSearchOverlay_View_NoResults` — shows "No results for 'query'"
- `TestSearchOverlay_View_Loading` — shows spinner during search

---

### Task 4.5 — Overlay focus management

**Description:**
Wire the search overlay into the root app model. `/` opens the overlay and captures all input;
`Esc` closes it and restores the previous pane focus. While the overlay is open, the three-pane
background is rendered with `lipgloss.NewStyle().Faint(true)`.

**Files:** `internal/app/app.go`, `internal/ui/panes/search.go`, `internal/ui/panes/search_test.go`

**Implementation steps:**
- [ ] Root app: on `/` keypress, set `m.searchOpen = true`, focus SearchOverlay
- [ ] Root app: on `Esc` in search, set `m.searchOpen = false`, restore previous focus
- [ ] Background panes dimmed when overlay active (render with `Faint(true)` style)
- [ ] Test: focus routing through root model

**Acceptance criteria:**
- [ ] `/` from any pane opens the search overlay without changing underlying pane state
- [ ] `Esc` restores exactly the pane that was focused before search opened
- [ ] Three-pane view is visibly dimmed while overlay is active
- [ ] `Enter` on a result plays the track and closes the overlay

**Tests:**

*Integration tests:*
- `TestApp_SlashOpensSearch` — `/` key sets searchOpen=true, overlay rendered
- `TestApp_EscClosesSearch` — Esc restores previous pane focus
- `TestApp_SearchPlayClosesOverlay` — Enter on result plays track and closes overlay
- `TestApp_BackgroundDimmed` — three-pane view rendered with Faint style when overlay open

---

## Out of Scope

- Podcast/show search results (filter them out)
- Audiobook search results
- Search history / recent searches
- Offline search suggestions

---

*Last updated: 2026-03-22*
