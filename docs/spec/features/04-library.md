---
title: "Library Browser"
description: "Provides browsable access to the user's Spotify library — playlists, saved albums, and liked songs — with keyboard navigation, playback, filtering, and playlist management, all rendered as independent panes with dense table layouts."
status: done
stories: [04, 47]
---

# Library Browser

## Background

The Library Browser is one of Spotnik's core features, giving users keyboard-driven access to their Spotify library directly in the terminal. It covers three data domains: playlists (with full management capabilities), saved albums, and liked songs. Users can navigate, filter, play, and manage their library without leaving the terminal.

The feature was originally built as a monolithic `LibraryPane` with collapsible tree sections for playlists, albums, liked songs, and recently played tracks. Playlist management lived in a separate `PlaylistManager` full-screen view. A subsequent redesign split the monolith into three independent panes — `PlaylistsPane`, `AlbumsPane`, and `LikedSongsPane` — each implementing the `layout.Pane` interface with dense table format and in-pane filtering. The `PlaylistManager` functionality (create, rename, delete, reorder) was merged into `PlaylistsPane`.

All panes read data from the central Store and emit request messages for side effects. No pane calls the API directly. Data arrives via typed messages and panes refresh their table rows accordingly. RecentlyPlayed was moved out of the library domain into the Stats feature (Feature 48).

---

## Story: Library Browser (spec 04)

### Background

The initial library browser implementation. The left pane is a navigable library browser where users can explore their playlists, saved albums, liked songs, and recently played tracks — and play anything by pressing Enter. It includes the API client for all library endpoints, model structs for Spotify library data, a collapsible tree data structure, the LibraryPane Bubble Tea model, lazy loading with pagination, and integration with the root app model.

### Acceptance Criteria

- [ ] Playlists visible within 2 seconds of app start
- [ ] Pressing Enter on a playlist starts playing it within 500ms
- [ ] Recently played list loads on app init and shows last 20 tracks
- [ ] Cursor navigation works at list boundaries without crash
- [ ] Sections expand/collapse on Enter, with lazy loading on first expand
- [ ] Loading spinners shown during all API fetches
- [ ] Like/unlike reflects in UI immediately (optimistic update)
- [ ] All API functions and pane Update() handlers tested

### Tasks

1. **Library API calls (Task 3.1)** — Implement all Spotify API methods required by the library browser. Each method calls a single Spotify endpoint, parses the JSON response, and returns typed Go structs. The SpotifyClient interface returns `(items, error)` only — the total count for pagination is handled internally by the `fetchAll` helper and is not exposed in the interface signatures.
   - Files: `internal/api/library.go`, `internal/api/library_test.go`
   - Implementation steps:
     - `GetPlaylists(ctx, limit, offset) ([]SimplePlaylist, error)`
     - `GetPlaylistTracks(ctx, id, limit, offset) ([]Track, error)`
     - `GetSavedAlbums(ctx, limit, offset) ([]SavedAlbum, error)`
     - `GetLikedTracks(ctx, limit, offset) ([]SavedTrack, error)`
     - `GetRecentlyPlayed(ctx, limit) ([]PlayHistory, error)`
     - `LikeTrack(ctx, id) error`
     - `UnlikeTrack(ctx, id) error`
     - Test each with fixture JSON + mock server
   - Acceptance criteria:
     - Every method matches the `SpotifyClient` interface signature exactly
     - Errors are wrapped with context (`fmt.Errorf("getting playlists: %w", err)`)
     - All tests pass with `httptest.NewServer` mocks
   - Tests:
     - `TestGetPlaylists_Success` — returns parsed playlists
     - `TestGetPlaylists_Empty` — returns empty slice, no error
     - `TestGetPlaylistTracks_Success` — returns tracks for playlist ID
     - `TestGetSavedAlbums_Success` — returns parsed albums
     - `TestGetLikedTracks_Success` — returns parsed saved tracks
     - `TestGetRecentlyPlayed_Success` — returns play history items
     - `TestLikeTrack_SendsPUT` — correct method, path, body
     - `TestUnlikeTrack_SendsDELETE` — correct method and path

2. **Library models (Task 3.2)** — Add model structs for library data. These represent Spotify API response shapes and must support JSON unmarshaling from fixture data. Extend the existing `internal/api/models.go` file — do not create a separate models file for library types.
   - Files: `internal/api/models.go` (extend), `internal/api/models_test.go` (extend)
   - Implementation steps:
     - `SimplePlaylist` struct (id, name, trackCount, owner)
     - `SavedAlbum` struct (album details, saved_at)
     - `SavedTrack` struct (track details, saved_at)
     - `PlayHistory` struct (track, played_at)
     - JSON unmarshaling tests with fixtures
   - Acceptance criteria:
     - All structs map correctly to Spotify JSON response shapes
     - Fixtures in `testdata/fixtures/` named descriptively (e.g., `simple_playlist.json`)
     - No custom unmarshal logic unless the Spotify response shape requires it
   - Tests:
     - `TestSimplePlaylist_Unmarshal` — JSON fixture parsing
     - `TestSavedAlbum_Unmarshal` — JSON fixture parsing
     - `TestSavedTrack_Unmarshal` — JSON fixture parsing
     - `TestPlayHistory_Unmarshal` — JSON fixture with played_at timestamp

3. **Section/tree data structure (Task 3.3)** — Build a tree data structure for the collapsible section UI. This manages cursor position, expand/collapse state, and item visibility. Section expand/collapse state is pane-local (held in the LibraryPane model, not in the Store).
   - Files: `internal/ui/panes/library.go`
   - Implementation steps:
     - `Section` type: name, items, expanded bool, loading bool, total int
     - `LibraryTree` struct: ordered list of sections + active cursor position
     - Navigation: `MoveDown()`, `MoveUp()`, `ToggleSection()`, `SelectedItem()`
     - Cursor does not wrap at boundaries (stays at top/bottom)
   - Acceptance criteria:
     - Cursor movement respects expanded/collapsed sections (skips hidden items)
     - `SelectedItem()` returns the correct item regardless of which sections are expanded
     - Section toggle updates `expanded` bool without modifying store data
   - Tests:
     - `TestLibraryTree_MoveDown` — cursor moves to next item
     - `TestLibraryTree_MoveUp` — cursor moves to previous item
     - `TestLibraryTree_MoveDown_AtBottom` — cursor stays at bottom (no wrap)
     - `TestLibraryTree_ToggleSection_Expands` — section expands on toggle
     - `TestLibraryTree_ToggleSection_Collapses` — expanded section collapses
     - `TestLibraryTree_SelectedItem` — returns correct item at cursor position

4. **LibraryPane model (Task 3.4)** — Implement the LibraryPane as a `tea.Model`. It reads data from the Store via read methods, renders the tree structure, and returns commands for playback, likes, and queue additions. The pane never calls API methods directly — it returns `tea.Cmd` functions that the Bubble Tea runtime executes asynchronously.
   - Files: `internal/ui/panes/library.go`, `internal/ui/panes/library_test.go`
   - Implementation steps:
     - Implement `tea.Model`: `Init()`, `Update()`, `View()`
     - `Init()` returns `fetchPlaylists` + `fetchRecentlyPlayed` commands
     - `Update()` handles key events, section messages, track loaded messages
     - `View()` renders tree from store data using section/tree structure
     - Playing track indicated with `▶` using `PlayingIndicator()` theme token
     - Selected item highlighted with `SelectedBg()` and `SelectedFg()` theme tokens
     - Section headers rendered with `SectionHeader()` theme token, bold
     - Test: all key handlers, loading states, empty states
   - Acceptance criteria:
     - `Init()` returns a batch command that fetches playlists and recently played
     - All keybindings from the keymap table are handled in `Update()`
     - `View()` is pure — reads from store only, no external calls
     - No hardcoded color values — all styling via Theme interface tokens
   - Tests:
     - `TestLibraryPane_Init_FetchesPlaylistsAndRecent` — returns batch command
     - `TestLibraryPane_View_ShowsSections` — renders section headers
     - `TestLibraryPane_View_PlayingIndicator` — shows ▶ next to playing track
     - `TestLibraryPane_Update_Enter_OnPlaylist` — returns play command with context URI
     - `TestLibraryPane_Update_Enter_OnSection` — toggles section expand
     - `TestLibraryPane_Update_A_AddsToQueue` — returns add-to-queue command for selected track
     - `TestLibraryPane_Update_L_ToggleLike` — returns like/unlike command
     - `TestLibraryPane_View_EmptySection` — shows loading spinner for unexpanded section

5. **Lazy loading + pagination (Task 3.5)** — Sections other than Playlists and Recently Played load lazily on first expand. Pagination loads more items as the user scrolls near the bottom of a section. Show item counts next to section headers and loading spinners during fetches.
   - Files: `internal/ui/panes/library.go`, `internal/ui/panes/library_test.go`
   - Implementation steps:
     - Trigger album/liked song fetch on section expand if not cached
     - Load more tracks when cursor is within 5 items of list end
     - Show item count `(12)` next to section headers
     - Loading spinner shown during fetch (using Bubbles spinner, `ActiveBorder()` token)
   - Acceptance criteria:
     - Expanding an already-cached section does not re-fetch
     - Scroll-near-bottom triggers exactly one fetch (no duplicate requests)
     - Spinner appears during fetch and disappears when data arrives
   - Tests (Unit):
     - `TestLibraryPane_ExpandSection_FetchesIfNotCached` — expand triggers fetch command
     - `TestLibraryPane_ExpandSection_SkipsFetchIfCached` — cached data skips fetch
     - `TestLibraryPane_ScrollNearBottom_LoadsMore` — within 5 items of end triggers next page fetch
   - Tests (Integration):
     - `TestLibraryPane_LazyLoad_EndToEnd` — expand section → fetch command → libraryLoadedMsg → data in store → renders items

6. **Integration (Task 3.6)** — Wire the LibraryPane into the root app model. Verify that keyboard focus routing works (Tab moves between panes), and that play commands flow from the library pane through the root model to the API client.
   - Files: `internal/app/app.go` (extend), `internal/app/app_integration_test.go` (extend)
   - Implementation steps:
     - Wire LibraryPane into root app model
     - Handle Enter → play commands flowing through root → API
     - Keyboard focus routing: Tab moves from Library → Player
   - Acceptance criteria:
     - Tab key moves focus to library pane; keys are routed to LibraryPane when focused
     - Enter on a playlist produces a `playContextMsg` that the root model dispatches as a play command
     - Playback state updates after play command completes
   - Tests (Integration):
     - `TestApp_LibraryPaneRouting` — Tab moves focus to library, keys routed correctly
     - `TestApp_LibraryPlay_UpdatesPlayback` — Enter on playlist → play command → playback state changes

### Implementation Context

**Store fields:**
```go
Playlists      []api.SimplePlaylist  // user's saved playlists
SavedAlbums    []api.SavedAlbum      // user's saved albums
LikedTracks    []api.SavedTrack      // first page only; load more on scroll
RecentlyPlayed []api.PlayHistory     // from GET /me/player/recently-played
```

> Section expand/collapse state is held in the LibraryPane model, not in the Store. The Store only holds the fetched data.

**List component:** Use `github.com/charmbracelet/bubbles/list`. Cast selected item: `item, ok := m.list.SelectedItem().(api.SimplePlaylist)`. Section headers are static labels rendered above each list.

**Message types:**
```go
type libraryLoadedMsg struct{ playlists []api.SimplePlaylist }
type playContextMsg   struct{ contextURI string }
```

> `libraryLoadedMsg` only contains playlists. Saved albums, liked tracks, and recently played are loaded separately on section expand (or on init for recently played) and arrive via their own message types (`savedAlbumsLoadedMsg`, `likedTracksLoadedMsg`, etc.).

**Playback via PlayOptions:** Play commands use `api.PlayOptions` which supports both `ContextURI string` (for playlists/albums) and `URIs []string` (for individual tracks). See `internal/api/models.go`.

**Recently played ownership:** Feature 04 fetches recently played on init via `GetRecentlyPlayed()`. This is an on-demand fetch, not part of the polling loop. The playback polling loop (Feature 03) does NOT fetch recently played data.

**Design tokens:** `theme.PlayingIndicator()` · `theme.SelectedBg()` · `theme.SelectedFg()` · `theme.SectionHeader()` · `theme.TextPrimary()` · `theme.TextMuted()`

**Left Pane Layout:**
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

**Sections:**

| Section | Source | Count |
|---|---|---|
| Playlists | `GET /me/playlists` | Total from API |
| Albums | `GET /me/albums` | Total from API |
| Liked Songs | `GET /me/tracks` | Total from API |
| Podcasts | `GET /me/shows` | Total from API |
| Recently Played | `GET /me/player/recently-played` | Last 20 |

**Podcasts**: display in library but playback is out of scope for MVP — show "Podcast playback coming soon" status message if selected.

**Loading Strategy:**
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

**Playback Behavior:**

| User Action | Result |
|---|---|
| Enter on playlist | `PUT /me/player/play` with `context_uri: spotify:playlist:{id}` |
| Enter on album | `PUT /me/player/play` with `context_uri: spotify:album:{id}` |
| Enter on liked songs | `PUT /me/player/play` with `context_uri: spotify:collection` |
| Enter on a specific track | `PUT /me/player/play` with `uris: [spotify:track:{id}]` |
| Enter on recently played track | Play that specific track via URI |

**Keymap (Library Pane Focus):**

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

**Files created:**

| File | Purpose |
|---|---|
| `internal/api/library.go` | Library API calls |
| `internal/api/library_test.go` | Tests with mock HTTP |
| `internal/api/models.go` | Extend with library models |
| `internal/api/models_test.go` | Extend with unmarshal tests |
| `internal/ui/panes/library.go` | LibraryPane model |
| `internal/ui/panes/library_test.go` | Update tests |

### Out of Scope

- Podcast episode browsing (show podcasts in list, but don't load episodes)
- Audiobook browsing
- Folder/group support for playlists (Spotify API doesn't expose folders)
- Artist browse page

---

## Story: Library Split (spec 47)

### Background

The monolithic `LibraryPane` (~18.5KB) handled all four data domains (playlists, albums, liked songs, recently played) in a single pane with collapsible tree sections. The `PlaylistManager` (~24.2KB) was a separate full-screen view with dual-pane layout for playlist management. This story splits the library into three independent panes — `PlaylistsPane`, `AlbumsPane`, and `LikedSongsPane` — each implementing `layout.Pane` with dense table format and filtering. PlaylistManager functionality (create, rename, delete, reorder) is merged into PlaylistsPane. RecentlyPlayed moves to Feature 48 (Stats Split).

Design reference: `docs/DESIGN.md` §2 (Pane Definitions), §9 (Dense Table column widths), §23 (Migration — LibraryPane split, PlaylistManager merge). Depends on Feature 41 (Pane interface) and Feature 43 (Table + Filter components).

### Acceptance Criteria

- [ ] `PlaylistsPane`, `AlbumsPane`, `LikedSongsPane` all satisfy `layout.Pane`
- [ ] PlaylistsPane merges PlaylistManager features (create, rename, delete, reorder, track sub-view)
- [ ] All 3 panes use bubble-table with correct column widths from DESIGN.md §9
- [ ] All 3 panes support in-pane filtering with `f` key
- [ ] Per-column colors match DESIGN.md §9 (TextMuted, TextPrimary, TextSecondary, TextMuted)
- [ ] Each pane reads from Store, emits request messages (no direct API calls)
- [ ] PlaylistsPane track sub-view: Enter opens, Esc returns to list
- [ ] LikedSongsPane: `i` key toggles like/unlike
- [ ] Old `LibraryPane` and `PlaylistManager` files are NOT deleted yet (done in Feature 49/53)
- [ ] `make ci` passes

### Tasks

1. **Create PlaylistsPane** — Playlist functionality is split between LibraryPane (list) and PlaylistManager (management). Create a unified `PlaylistsPane` that merges both.
   - Files: Create `internal/ui/panes/playlists_pane.go`
   - Struct:
     ```go
     type PlaylistsPane struct {
         store   *state.Store
         theme   theme.Theme
         table   components.Table
         filter  *components.Filter
         focused bool
         width   int
         height  int

         // Track sub-view state
         inTrackView   bool
         selectedID    string           // Spotify playlist ID
         selectedName  string
         trackTable    components.Table  // tracks for selected playlist
     }
     ```
   - Pane interface:
     ```go
     func (p *PlaylistsPane) ID() layout.PaneID       { return layout.PanePlaylists }
     func (p *PlaylistsPane) Title() string {
         if p.inTrackView {
             return fmt.Sprintf("Playlists ── %s (%d tracks)", p.selectedName, trackCount)
         }
         return "Playlists"
     }
     func (p *PlaylistsPane) ToggleKey() int           { return 3 }
     func (p *PlaylistsPane) Actions() []layout.Action {
         if p.filter.IsActive() {
             return []layout.Action{{Key: "Esc", Label: "close"}}
         }
         if p.inTrackView {
             return []layout.Action{{Key: "Esc", Label: "back"}, {Key: "Shift+↕", Label: "reorder"}}
         }
         return []layout.Action{
             {Key: "f", Label: "filter"}, {Key: "n", Label: "new"},
             {Key: "r", Label: "rename"}, {Key: "x", Label: "delete"},
         }
     }
     ```
   - Key handling:
     - `Enter` → open track sub-view (emit `FetchPlaylistTracksRequestMsg`)
     - `Esc` in track view → return to playlist list
     - `n` → emit `PlaylistCreateRequestMsg`
     - `r` → emit `PlaylistRenameRequestMsg`
     - `x` → emit `PlaylistRemoveRequestMsg` (follow existing PlaylistManager pattern)
     - `Shift+↑/↓` → emit `PlaylistReorderRequestMsg`
     - `f` → toggle filter
     - `j/k` → scroll (forwarded to table)
   - Data source: `store.Playlists()` for playlist list, playlist tracks via message
   - Playlist list columns: `# 5% | Name 70% | Tracks 25%`
   - Track sub-view columns: `# 5% | Track 45% | Artist 35% | Duration 15%`
   - Tests:
     - Unit: Interface satisfaction: `var _ layout.Pane = &PlaylistsPane{}`
     - Unit: Playlist list renders with correct columns
     - Unit: Enter key on selected playlist → emits FetchPlaylistTracksRequestMsg
     - Unit: Track sub-view shows tracks for selected playlist
     - Unit: Esc in track view → returns to playlist list
     - Unit: `n` key → emits create request
     - Unit: `r` key → emits rename request
     - Unit: `x` key → emits remove request
     - Unit: `Shift+↑/↓` → emits reorder request
     - Unit: Filter filters playlists by name
     - Unit: Dynamic title shows playlist name in track sub-view

2. **Create AlbumsPane** — Album browsing is buried in LibraryPane's tree sections. Create a dedicated `AlbumsPane`.
   - Files: Create `internal/ui/panes/albums_pane.go`
   - Struct:
     ```go
     type AlbumsPane struct {
         store   *state.Store
         theme   theme.Theme
         table   components.Table
         filter  *components.Filter
         focused bool
         width   int
         height  int
     }
     ```
   - Pane interface:
     ```go
     func (a *AlbumsPane) ID() layout.PaneID       { return layout.PaneAlbums }
     func (a *AlbumsPane) Title() string            { return "Albums" }
     func (a *AlbumsPane) ToggleKey() int           { return 4 }
     func (a *AlbumsPane) Actions() []layout.Action {
         if a.filter.IsActive() {
             return []layout.Action{{Key: "Esc", Label: "close"}}
         }
         return []layout.Action{{Key: "f", Label: "filter"}}
     }
     ```
   - Key handling:
     - `Enter` → emit `PlayContextMsg` with album URI
     - `f` → toggle filter
     - `j/k` → scroll
   - Data source: `store.Albums()` for album list
   - Columns: `# 5% | Name 50% | Artist 30% | Year 15%`
   - Filter matches: album name, artist name
   - Tests:
     - Unit: Interface satisfaction: `var _ layout.Pane = &AlbumsPane{}`
     - Unit: Album list renders with correct columns
     - Unit: Enter key → emits PlayContextMsg with album URI
     - Unit: Filter filters by album name and artist
     - Unit: Albums display year correctly
     - Unit: Empty albums → clean empty state

3. **Create LikedSongsPane** — Liked songs browsing is buried in LibraryPane. Create a dedicated `LikedSongsPane`.
   - Files: Create `internal/ui/panes/likedsongs_pane.go`
   - Struct:
     ```go
     type LikedSongsPane struct {
         store   *state.Store
         theme   theme.Theme
         table   components.Table
         filter  *components.Filter
         focused bool
         width   int
         height  int
     }
     ```
   - Pane interface:
     ```go
     func (l *LikedSongsPane) ID() layout.PaneID       { return layout.PaneLikedSongs }
     func (l *LikedSongsPane) Title() string            { return "Liked Songs" }
     func (l *LikedSongsPane) ToggleKey() int           { return 5 }
     func (l *LikedSongsPane) Actions() []layout.Action {
         if l.filter.IsActive() {
             return []layout.Action{{Key: "Esc", Label: "close"}}
         }
         return []layout.Action{{Key: "f", Label: "filter"}, {Key: "i", Label: "like"}}
     }
     ```
   - Key handling:
     - `Enter` → emit `PlayTrackMsg` with track URI
     - `i` → emit `LikeTrackRequestMsg` (toggle like/unlike for selected track)
     - `f` → toggle filter
     - `j/k` → scroll
   - Data source: `store.LikedTracks()` for track list
   - Columns: `# 5% | Track 45% | Artist 35% | Duration 15%`
   - Filter matches: track name, artist name
   - Tests:
     - Unit: Interface satisfaction: `var _ layout.Pane = &LikedSongsPane{}`
     - Unit: Track list renders with correct columns
     - Unit: Enter key → emits PlayTrackMsg
     - Unit: `i` key → emits LikeTrackRequestMsg
     - Unit: Filter filters by track name and artist
     - Unit: Duration formatted as M:SS

4. **Data loading integration** — The new panes need to receive data that previously flowed through LibraryPane. Each pane handles the existing message types in its `Update()`.
   - Files: Modify `internal/ui/panes/playlists_pane.go`, `internal/ui/panes/albums_pane.go`, `internal/ui/panes/likedsongs_pane.go`
   - Message routing:

     | Pane | Message | Data |
     |------|---------|------|
     | PlaylistsPane | `LibraryLoadedMsg` (playlists data) | `store.Playlists()` |
     | PlaylistsPane | `PlaylistTracksLoadedMsg` | Track list for selected playlist |
     | PlaylistsPane | `PlaylistCreatedMsg`, `PlaylistRenamedMsg`, etc. | Mutation results |
     | AlbumsPane | `AlbumsLoadedMsg` | `store.Albums()` |
     | LikedSongsPane | `LikedTracksLoadedMsg` | `store.LikedTracks()` |

   - The panes read from Store on data-loaded messages and update their table rows. No new message types needed — reuse existing ones from `messages.go`.
   - Tests:
     - Unit: PlaylistsPane handles LibraryLoadedMsg → refreshes table
     - Unit: AlbumsPane handles AlbumsLoadedMsg → refreshes table
     - Unit: LikedSongsPane handles LikedTracksLoadedMsg → refreshes table
     - Unit: PlaylistsPane handles PlaylistCreatedMsg → refreshes list
     - Unit: PlaylistsPane handles PlaylistTracksLoadedMsg → shows tracks in sub-view

5. **Comprehensive tests** — Full integration and edge case test coverage for all three split panes.
   - Files: Create `internal/ui/panes/playlists_pane_test.go`, `internal/ui/panes/albums_pane_test.go`, `internal/ui/panes/likedsongs_pane_test.go`
   - Tests (Integration):
     - PlaylistsPane — load playlists → select → Enter → track view → Esc → back to list
     - PlaylistsPane — create playlist → list refreshes
     - PlaylistsPane — rename playlist → list updates
     - PlaylistsPane — reorder tracks with Shift+↑/↓
     - AlbumsPane — load albums → filter → select → play
     - LikedSongsPane — load tracks → like/unlike → filter
     - All 3 panes handle resize correctly
     - All 3 panes filter independently
   - Tests (Edge):
     - Large dataset (100+ items) → scrolling works
     - Empty data → clean empty state per pane

### Design Diagram

```
Current Architecture:
  LibraryPane (18.5KB) — monolithic tree with 4 collapsible sections
  PlaylistManager (24.2KB) — separate full-screen view (key '3')

New Architecture (3 independent panes):

╭─ ³Playlists ────── ᐅf filter ─ ᐅn new ─ ᐅr rename ─ ᐅx delete ╮
│  #   Name                              Tracks                   │
│  1   LoFi                              42                       │
│  2   Best of Coke Studio               28                       │
│  3   Soul                              15                       │
│  4   Workout                           67                       │
│  ▼ more below                                                   │
╰─────────────────────────────────────────────────────────────────╯

  Enter → opens track sub-view for selected playlist:
╭─ ³Playlists ── LoFi (42 tracks) ──────── ᐅEsc back ─ ᐅShift+↕ reorder ╮
│  #   Track                    Artist              Duration              │
│  1   Snowman                  Sia                 3:21                  │
│  2   Coffee                  Beabadoobee         3:44                  │
│  ▼ more below                                                          │
╰────────────────────────────────────────────────────────────────────────╯

╭─ ⁴Albums ───────────────────── ᐅf filter ╮
│  #   Name                 Artist     Year │
│  1   After Hours          Weeknd     2020 │
│  2   OK Computer          Radiohead  1997 │
│  3   In Rainbows          Radiohead  2007 │
│  ▼ more below                             │
╰───────────────────────────────────────────╯

╭─ ⁵Liked Songs ──────── ᐅf filter ─ ᐅi like ╮
│  #   Track              Artist       Duration │
│  1   Blinding Lights    The Weeknd   3:22     │
│  2   Save Your Tears    The Weeknd   3:35     │
│  3   Levitating         Dua Lipa     3:23     │
│  ▼ more below                                 │
╰───────────────────────────────────────────────╯

Column Widths (DESIGN.md §9):
  Playlists:   # 5% | Name 70% | Tracks 25%
  Albums:      # 5% | Name 50% | Artist 30% | Year 15%
  LikedSongs:  # 5% | Track 45% | Artist 35% | Duration 15%
```

### Notes

- **RecentlyPlayed** is NOT part of this story. It moves to Feature 48 (Stats Split) since it was originally a section of StatsView and uses `store.RecentlyPlayed()`.
- The old `LibraryPane` and `PlaylistManager` files remain until Feature 49 (App Migration) rewires the app to use the new panes. At that point, the old files become dead code and are deleted in Feature 53 (Cleanup).
- PlaylistsPane's track sub-view is internal state — it doesn't change the page or layout. The pane renders either the playlist list or the track list based on `inTrackView` flag.
- Playlist mutations (create, rename, delete, reorder) emit request messages. The app's `Update()` dispatches the API commands. This flow is unchanged from the current architecture.
- The column flex factors are approximations of the percentage widths. bubble-table distributes remaining space after fixed columns, so flex factors 1:14:6:3 ≈ 5%/58%/25%/12%. Fine-tune during implementation to match the visual design.

---

*Last updated: 2026-03-30*
