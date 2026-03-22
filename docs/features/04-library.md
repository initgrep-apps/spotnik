# Feature 04 — Library Browser

> **Depends on:** Features 02 (Auth) and 03 (Playback) complete and committed.

## Goal

The left pane is a navigable library browser. Users can explore their playlists, saved albums,
liked songs, and recently played tracks — and play anything by pressing Enter.

---

## Feature Acceptance Criteria

- [ ] Playlists visible within 2 seconds of app start
- [ ] Pressing Enter on a playlist starts playing it within 500ms
- [ ] Recently played list loads on app init and shows last 20 tracks
- [ ] Cursor navigation works at list boundaries without crash
- [ ] Sections expand/collapse on Enter, with lazy loading on first expand
- [ ] Loading spinners shown during all API fetches
- [ ] Like/unlike reflects in UI immediately (optimistic update)
- [ ] All API functions and pane Update() handlers tested

---

## Implementation Context

### Store fields this feature uses
```go
Playlists      []api.SimplePlaylist  // user's saved playlists
SavedAlbums    []api.SavedAlbum      // user's saved albums
LikedTracks    []api.SavedTrack      // first page only; load more on scroll
RecentlyPlayed []api.PlayHistory     // from GET /me/player/recently-played
```

> Section expand/collapse state is held in the LibraryPane model, not in the Store.
> The Store only holds the fetched data (playlists, albums, etc.).

### List component
Use `github.com/charmbracelet/bubbles/list` — do not build a custom list.
Cast selected item: `item, ok := m.list.SelectedItem().(api.SimplePlaylist)`.
Section headers (PLAYLISTS, ALBUMS, etc.) are static labels rendered above each list.

### Message types for this feature
```go
type libraryLoadedMsg struct{ playlists []api.SimplePlaylist }
type playContextMsg   struct{ contextURI string }
```

> `libraryLoadedMsg` only contains playlists. Saved albums, liked tracks, and recently played
> are loaded separately on section expand (or on init for recently played) and arrive via
> their own message types (`savedAlbumsLoadedMsg`, `likedTracksLoadedMsg`, etc.).

### Playback via PlayOptions

Play commands use `api.PlayOptions` which supports both `ContextURI string` (for playlists/albums)
and `URIs []string` (for individual tracks). See `internal/api/models.go`.

### Recently played ownership

Feature 04 fetches recently played on init via `GetRecentlyPlayed()`. This is an on-demand
fetch, not part of the polling loop. The playback polling loop (Feature 03) does NOT fetch
recently played data.

### Design tokens used in this feature
`theme.PlayingIndicator()` · `theme.SelectedBg()` · `theme.SelectedFg()` ·
`theme.SectionHeader()` · `theme.TextPrimary()` · `theme.TextMuted()`

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
│    Chill Vibes              │  ← selected: `SelectedBg()` token
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

Play commands use `api.PlayOptions` which supports both `ContextURI string` (for playlists/albums)
and `URIs []string` (for individual tracks). See `internal/api/models.go`.

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
| `internal/api/models.go` | Extend with library models |
| `internal/api/models_test.go` | Extend with unmarshal tests |
| `internal/ui/panes/library.go` | LibraryPane model |
| `internal/ui/panes/library_test.go` | Update tests |

---

## Task Breakdown

### Task 3.1 — Library API calls

**Description:** Implement all Spotify API methods required by the library browser. Each method
calls a single Spotify endpoint, parses the JSON response, and returns typed Go structs. The
SpotifyClient interface returns `(items, error)` only — the total count for pagination is handled
internally by the `fetchAll` helper and is not exposed in the interface signatures.

**Files:** `internal/api/library.go`, `internal/api/library_test.go`

**Implementation steps:**
- [ ] `GetPlaylists(ctx, limit, offset) ([]SimplePlaylist, error)`
- [ ] `GetPlaylistTracks(ctx, id, limit, offset) ([]Track, error)`
- [ ] `GetSavedAlbums(ctx, limit, offset) ([]SavedAlbum, error)`
- [ ] `GetLikedTracks(ctx, limit, offset) ([]SavedTrack, error)`
- [ ] `GetRecentlyPlayed(ctx, limit) ([]PlayHistory, error)`
- [ ] `LikeTrack(ctx, id) error`
- [ ] `UnlikeTrack(ctx, id) error`
- [ ] Test each with fixture JSON + mock server

**Acceptance criteria:**
- Every method matches the `SpotifyClient` interface signature exactly
- Errors are wrapped with context (`fmt.Errorf("getting playlists: %w", err)`)
- All tests pass with `httptest.NewServer` mocks

**Tests (Unit):**
- `TestGetPlaylists_Success` — returns parsed playlists
- `TestGetPlaylists_Empty` — returns empty slice, no error
- `TestGetPlaylistTracks_Success` — returns tracks for playlist ID
- `TestGetSavedAlbums_Success` — returns parsed albums
- `TestGetLikedTracks_Success` — returns parsed saved tracks
- `TestGetRecentlyPlayed_Success` — returns play history items
- `TestLikeTrack_SendsPUT` — correct method, path, body
- `TestUnlikeTrack_SendsDELETE` — correct method and path

---

### Task 3.2 — Library models

**Description:** Add model structs for library data. These represent Spotify API response shapes
and must support JSON unmarshaling from fixture data. Extend the existing `internal/api/models.go`
file — do not create a separate models file for library types.

**Files:** `internal/api/models.go` (extend), `internal/api/models_test.go` (extend)

**Implementation steps:**
- [ ] `SimplePlaylist` struct (id, name, trackCount, owner)
- [ ] `SavedAlbum` struct (album details, saved_at)
- [ ] `SavedTrack` struct (track details, saved_at)
- [ ] `PlayHistory` struct (track, played_at)
- [ ] JSON unmarshaling tests with fixtures

**Acceptance criteria:**
- All structs map correctly to Spotify JSON response shapes
- Fixtures in `testdata/fixtures/` named descriptively (e.g., `simple_playlist.json`)
- No custom unmarshal logic unless the Spotify response shape requires it

**Tests (Unit):**
- `TestSimplePlaylist_Unmarshal` — JSON fixture parsing
- `TestSavedAlbum_Unmarshal` — JSON fixture parsing
- `TestSavedTrack_Unmarshal` — JSON fixture parsing
- `TestPlayHistory_Unmarshal` — JSON fixture with played_at timestamp

---

### Task 3.3 — Section/tree data structure

**Description:** Build a tree data structure for the collapsible section UI. This manages cursor
position, expand/collapse state, and item visibility. Section expand/collapse state is pane-local
(held in the LibraryPane model, not in the Store).

**Files:** `internal/ui/panes/library.go`

**Implementation steps:**
- [ ] `Section` type: name, items, expanded bool, loading bool, total int
- [ ] `LibraryTree` struct: ordered list of sections + active cursor position
- [ ] Navigation: `MoveDown()`, `MoveUp()`, `ToggleSection()`, `SelectedItem()`
- [ ] Cursor does not wrap at boundaries (stays at top/bottom)

**Acceptance criteria:**
- Cursor movement respects expanded/collapsed sections (skips hidden items)
- `SelectedItem()` returns the correct item regardless of which sections are expanded
- Section toggle updates `expanded` bool without modifying store data

**Tests (Unit):**
- `TestLibraryTree_MoveDown` — cursor moves to next item
- `TestLibraryTree_MoveUp` — cursor moves to previous item
- `TestLibraryTree_MoveDown_AtBottom` — cursor stays at bottom (no wrap)
- `TestLibraryTree_ToggleSection_Expands` — section expands on toggle
- `TestLibraryTree_ToggleSection_Collapses` — expanded section collapses
- `TestLibraryTree_SelectedItem` — returns correct item at cursor position

---

### Task 3.4 — LibraryPane model

**Description:** Implement the LibraryPane as a `tea.Model`. It reads data from the Store via
read methods, renders the tree structure, and returns commands for playback, likes, and queue
additions. The pane never calls API methods directly — it returns `tea.Cmd` functions that the
Bubble Tea runtime executes asynchronously.

**Files:** `internal/ui/panes/library.go`, `internal/ui/panes/library_test.go`

**Implementation steps:**
- [ ] Implement `tea.Model`: `Init()`, `Update()`, `View()`
- [ ] `Init()` returns `fetchPlaylists` + `fetchRecentlyPlayed` commands
- [ ] `Update()` handles key events, section messages, track loaded messages
- [ ] `View()` renders tree from store data using section/tree structure
- [ ] Playing track indicated with `▶` using `PlayingIndicator()` theme token
- [ ] Selected item highlighted with `SelectedBg()` and `SelectedFg()` theme tokens
- [ ] Section headers rendered with `SectionHeader()` theme token, bold
- [ ] Test: all key handlers, loading states, empty states

**Acceptance criteria:**
- `Init()` returns a batch command that fetches playlists and recently played
- All keybindings from the keymap table are handled in `Update()`
- `View()` is pure — reads from store only, no external calls
- No hardcoded color values — all styling via Theme interface tokens

**Tests (Unit):**
- `TestLibraryPane_Init_FetchesPlaylistsAndRecent` — returns batch command
- `TestLibraryPane_View_ShowsSections` — renders section headers
- `TestLibraryPane_View_PlayingIndicator` — shows ▶ next to playing track
- `TestLibraryPane_Update_Enter_OnPlaylist` — returns play command with context URI
- `TestLibraryPane_Update_Enter_OnSection` — toggles section expand
- `TestLibraryPane_Update_A_AddsToQueue` — returns add-to-queue command for selected track
- `TestLibraryPane_Update_L_ToggleLike` — returns like/unlike command
- `TestLibraryPane_View_EmptySection` — shows loading spinner for unexpanded section

---

### Task 3.5 — Lazy loading + pagination

**Description:** Sections other than Playlists and Recently Played load lazily on first expand.
Pagination loads more items as the user scrolls near the bottom of a section. Show item counts
next to section headers and loading spinners during fetches.

**Files:** `internal/ui/panes/library.go`, `internal/ui/panes/library_test.go`

**Implementation steps:**
- [ ] Trigger album/liked song fetch on section expand if not cached
- [ ] Load more tracks when cursor is within 5 items of list end
- [ ] Show item count `(12)` next to section headers
- [ ] Loading spinner shown during fetch (using Bubbles spinner, `ActiveBorder()` token)

**Acceptance criteria:**
- Expanding an already-cached section does not re-fetch
- Scroll-near-bottom triggers exactly one fetch (no duplicate requests)
- Spinner appears during fetch and disappears when data arrives

**Tests (Unit):**
- `TestLibraryPane_ExpandSection_FetchesIfNotCached` — expand triggers fetch command
- `TestLibraryPane_ExpandSection_SkipsFetchIfCached` — cached data skips fetch
- `TestLibraryPane_ScrollNearBottom_LoadsMore` — within 5 items of end triggers next page fetch

**Tests (Integration):**
- `TestLibraryPane_LazyLoad_EndToEnd` — expand section → fetch command → libraryLoadedMsg → data in store → renders items

---

### Task 3.6 — Integration

**Description:** Wire the LibraryPane into the root app model. Verify that keyboard focus routing
works (Tab moves between panes), and that play commands flow from the library pane through the
root model to the API client.

**Files:** `internal/app/app.go` (extend), `internal/app/app_integration_test.go` (extend)

**Implementation steps:**
- [ ] Wire LibraryPane into root app model
- [ ] Handle Enter → play commands flowing through root → API
- [ ] Keyboard focus routing: Tab moves from Library → Player

**Acceptance criteria:**
- Tab key moves focus to library pane; keys are routed to LibraryPane when focused
- Enter on a playlist produces a `playContextMsg` that the root model dispatches as a play command
- Playback state updates after play command completes

**Tests (Integration):**
- `TestApp_LibraryPaneRouting` — Tab moves focus to library, keys routed correctly
- `TestApp_LibraryPlay_UpdatesPlayback` — Enter on playlist → play command → playback state changes

---

## Out of Scope

- Podcast episode browsing (show podcasts in list, but don't load episodes)
- Audiobook browsing
- Folder/group support for playlists (Spotify API doesn't expose folders)
- Artist browse page

---

*Last updated: 2026-03-22*
