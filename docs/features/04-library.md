# Feature 04 — Library Browser

> **Depends on:** Features 02 (Auth) and 03 (Playback) complete and committed.

## Implementation Context

### Store fields this feature uses
```go
Playlists      []api.Playlist  // user's saved playlists
SavedAlbums    []api.Album     // user's saved albums
LikedSongs     []api.Track     // first page only; load more on scroll
RecentlyPlayed []api.Track     // from GET /me/player/recently-played
```

### List component
Use `github.com/charmbracelet/bubbles/list` — do not build a custom list.
Cast selected item: `item, ok := m.list.SelectedItem().(api.Playlist)`.
Section headers (PLAYLISTS, ALBUMS, etc.) are static labels rendered above each list.

### Message types for this feature
```go
type libraryLoadedMsg struct {
    playlists      []api.Playlist
    savedAlbums    []api.Album
    recentlyPlayed []api.Track
}
type playContextMsg struct{ contextURI string } // sent to root → dispatches play cmd
```

### Design tokens used in this feature
`theme.PlayingIndicator()` · `theme.SelectedBg()` · `theme.SelectedFg()` ·
`theme.SectionHeader()` · `theme.TextPrimary()` · `theme.TextMuted()`

---

---

## Goal

The left pane is a navigable library browser. Users can explore their playlists, saved albums,
liked songs, and recently played tracks — and play anything by pressing Enter.

---

## User Stories

- **As a user**, I see my playlists in the left pane on startup.
- **As a user**, I press `j/k` to navigate the library list.
- **As a user**, I press `Enter` on a playlist to load and play it.
- **As a user**, I press `Enter` on an album to load and play it.
- **As a user**, I can see my Liked Songs as a navigable section.
- **As a user**, I can see recently played tracks at the bottom of the left pane.
- **As a user**, I press `Tab` to move from the library pane to the player pane.
- **As a user**, library sections collapse and expand with `Enter` on a section header.

---

## Left Pane Layout

```
│  LIBRARY                    │
│  ─────────────────────────  │
│                             │
│  ▸ Playlists           (12) │  ← collapsed section header
│  ▾ Albums               (8) │  ← expanded section header
│    ▸ After Hours            │    ← album item
│    ▸ Future Nostalgia       │
│    ▸ Justice                │
│  ▸ Liked Songs         (287)│
│  ▸ Podcasts              (3)│
│                             │
│  ─────────────────────────  │
│  RECENTLY PLAYED            │
│  ─────────────────────────  │
│  ▶ Blinding Lights          │  ← ▶ = currently playing
│    Save Your Tears          │
│    Starboy                  │
│    Levitating               │
│    Peaches                  │
│                             │
```

When a section is expanded (e.g., Playlists), it shows the playlist list inline:

```
│  ▾ Playlists           (12) │
│    Chill Vibes              │  ← selected: Lavender bg
│    Workout Mix              │
│    Late Night Coding        │
│    ...                      │
│  ▸ Albums               (8) │
```

---

## Sections

| Section | Source | Count |
|---|---|---|
| Playlists | `GET /me/playlists` | Total from API |
| Albums | `GET /me/albums` | Total from API |
| Liked Songs | `GET /me/tracks` | Total from API |
| Podcasts | `GET /me/shows` | Total from API |
| Recently Played | `GET /me/player/recently-played` | Last 20 |

**Podcasts**: display in library but playback is out of scope for MVP — show "Podcast playback coming soon" status message if selected.

---

## Loading Strategy

- **On app start**: fetch playlists and recently played immediately (they load fast)
- **On section expand**: lazily load that section's content
- **On playlist selection**: fetch playlist tracks if not already cached in store
- **Pagination**: load first 50 items, load more as user scrolls near bottom

```go
// Lazy load on expand
case expandSectionMsg{section: SectionAlbums}:
    if !m.store.AlbumsLoaded() {
        return m, fetchAlbums(m.client)
    }
```

---

## Playback Behavior

| User Action | Result |
|---|---|
| Enter on playlist | `PUT /me/player/play` with `context_uri: spotify:playlist:{id}` |
| Enter on album | `PUT /me/player/play` with `context_uri: spotify:album:{id}` |
| Enter on liked songs | `PUT /me/player/play` with `context_uri: spotify:collection` |
| Enter on a specific track | `PUT /me/player/play` with `uris: [spotify:track:{id}]` |
| Enter on recently played track | Play that specific track via URI |

---

## Keymap (Library Pane Focus)

| Key | Action |
|---|---|
| `j` / `↓` | Move selection down |
| `k` / `↑` | Move selection up |
| `Enter` | Expand section OR play item |
| `Backspace` | Collapse current section, move to section header |
| `PgDown` | Scroll down 10 items |
| `PgUp` | Scroll up 10 items |
| `g` | Jump to top |
| `G` | Jump to bottom |
| `a` | Add selected item to queue (if it's a track) |
| `l` | Like/unlike selected track (toggle) |
| `Tab` | Move focus to Player pane |

---

## Files to Create

| File | Purpose |
|---|---|
| `internal/api/library.go` | Library API calls |
| `internal/api/library_test.go` | Tests with mock HTTP |
| `internal/ui/panes/library.go` | LibraryPane model |
| `internal/ui/panes/library_test.go` | Update tests |

---

## Task Breakdown

### Task 3.1 — Library API calls
- [ ] `GetPlaylists(ctx, limit, offset) ([]SimplePlaylist, total int, error)`
- [ ] `GetPlaylistTracks(ctx, id, limit, offset) ([]Track, total int, error)`
- [ ] `GetSavedAlbums(ctx, limit, offset) ([]SavedAlbum, total int, error)`
- [ ] `GetLikedTracks(ctx, limit, offset) ([]SavedTrack, total int, error)`
- [ ] `GetRecentlyPlayed(ctx, limit) ([]PlayHistory, error)`
- [ ] `LikeTrack(ctx, id) error`
- [ ] `UnlikeTrack(ctx, id) error`
- [ ] Test each with fixture JSON + mock server

### Task 3.2 — Library models
- [ ] `SimplePlaylist` struct (id, name, trackCount, owner)
- [ ] `SavedAlbum` struct (album details, saved_at)
- [ ] `SavedTrack` struct (track details, saved_at)
- [ ] `PlayHistory` struct (track, played_at)
- [ ] JSON unmarshaling tests

### Task 3.3 — Section/tree data structure
- [ ] `Section` type: name, items, expanded bool, loading bool, total int
- [ ] `LibraryTree` struct: ordered list of sections + active cursor position
- [ ] Navigation: `MoveDown()`, `MoveUp()`, `ToggleSection()`, `SelectedItem()`
- [ ] Test: cursor wraps at boundaries, expand/collapse toggles correctly

### Task 3.4 — LibraryPane model
- [ ] Implement `tea.Model`: `Init()`, `Update()`, `View()`
- [ ] `Init()` returns `fetchPlaylists` + `fetchRecentlyPlayed` commands
- [ ] `Update()` handles key events, section messages, track loaded messages
- [ ] `View()` renders tree from store data using section/tree structure
- [ ] Playing track indicated with `▶` in correct position
- [ ] Test: all key handlers, loading states, empty states

### Task 3.5 — Lazy loading + pagination
- [ ] Trigger album/liked song fetch on section expand if not cached
- [ ] Load more tracks when cursor is within 5 items of list end
- [ ] Show item count `(12)` next to section headers
- [ ] Loading spinner shown during fetch

### Task 3.6 — Integration
- [ ] Wire LibraryPane into root app model
- [ ] Handle Enter → play commands flowing through root → API
- [ ] Keyboard focus routing: Tab moves from Library → Player

---

## Acceptance Criteria

- [ ] Playlists visible within 2 seconds of app start
- [ ] Pressing Enter on a playlist starts playing it within 500ms
- [ ] Recently played list updates after each track change
- [ ] Cursor navigation works at list boundaries without crashing
- [ ] Loading states shown during all API fetches
- [ ] Like/unlike reflects in UI immediately (optimistic)
- [ ] All API functions and pane update handlers tested

---

## Out of Scope

- Podcast episode browsing (show podcasts in list, but don't load episodes)
- Audiobook browsing
- Folder/group support for playlists (Spotify API doesn't expose folders)
- Artist browse page

---

*Last updated: 2026-02-21*
