---
title: "Library Browser"
feature: 04-library
status: done
---

## Background
The initial library browser implementation. The left pane is a navigable library browser where users can explore their playlists, saved albums, liked songs, and recently played tracks -- and play anything by pressing Enter. It includes the API client for all library endpoints, model structs for Spotify library data, a collapsible tree data structure, the LibraryPane Bubble Tea model, lazy loading with pagination, and integration with the root app model.

## Design

### Store fields
```go
Playlists      []api.SimplePlaylist  // user's saved playlists
SavedAlbums    []api.SavedAlbum      // user's saved albums
LikedTracks    []api.SavedTrack      // first page only; load more on scroll
RecentlyPlayed []api.PlayHistory     // from GET /me/player/recently-played
```

> Section expand/collapse state is held in the LibraryPane model, not in the Store.

### Message types
```go
type libraryLoadedMsg struct{ playlists []api.SimplePlaylist }
type playContextMsg   struct{ contextURI string }
```

### Design tokens
`theme.PlayingIndicator()` . `theme.SelectedBg()` . `theme.SelectedFg()` . `theme.SectionHeader()` . `theme.TextPrimary()` . `theme.TextMuted()`

### Left Pane Layout
```
|  LIBRARY                    |
|  -----------------------    |
|                             |
|  > Playlists           (12) |  <- collapsed section header
|  v Albums               (8) |  <- expanded section header
|    > After Hours            |    <- album item
|    > Future Nostalgia       |
|    > Justice                |
|  > Liked Songs         (287)|
|  > Podcasts              (3)|
|                             |
|  -----------------------    |
|  RECENTLY PLAYED            |
|  -----------------------    |
|  > Blinding Lights          |  <- currently playing
|    Save Your Tears          |
|    Starboy                  |
|    Levitating               |
|    Peaches                  |
|                             |
```

### Keymap (Library Pane Focus)

| Key | Action |
|---|---|
| `j` / `down` | Move selection down |
| `k` / `up` | Move selection up |
| `Enter` | Expand section OR play item |
| `Backspace` | Collapse current section, move to section header |
| `PgDown` | Scroll down 10 items |
| `PgUp` | Scroll up 10 items |
| `g` | Jump to top |
| `G` | Jump to bottom |
| `a` | Add selected item to queue (if it's a track) |
| `l` | Like/unlike selected track (toggle) |
| `Tab` | Move focus to Player pane |

### Loading Strategy
- **On app start**: fetch playlists and recently played immediately
- **On section expand**: lazily load that section's content
- **On playlist selection**: fetch playlist tracks if not already cached
- **Pagination**: load first 50 items, load more as user scrolls near bottom

### Playback Behavior

| User Action | Result |
|---|---|
| Enter on playlist | `PUT /me/player/play` with `context_uri: spotify:playlist:{id}` |
| Enter on album | `PUT /me/player/play` with `context_uri: spotify:album:{id}` |
| Enter on liked songs | `PUT /me/player/play` with `context_uri: spotify:collection` |
| Enter on a specific track | `PUT /me/player/play` with `uris: [spotify:track:{id}]` |
| Enter on recently played track | Play that specific track via URI |

### Files created

| File | Purpose |
|---|---|
| `internal/api/library.go` | Library API calls |
| `internal/api/library_test.go` | Tests with mock HTTP |
| `internal/api/models.go` | Extend with library models |
| `internal/api/models_test.go` | Extend with unmarshal tests |
| `internal/ui/panes/library.go` | LibraryPane model |
| `internal/ui/panes/library_test.go` | Update tests |

### Out of Scope
- Podcast episode browsing
- Audiobook browsing
- Folder/group support for playlists
- Artist browse page

## Acceptance Criteria
- [ ] Playlists visible within 2 seconds of app start
- [ ] Pressing Enter on a playlist starts playing it within 500ms
- [ ] Recently played list loads on init and shows last 20 tracks
- [ ] Cursor navigation works at list boundaries without crash
- [ ] Sections expand/collapse on Enter, with lazy loading on first expand
- [ ] Loading spinners shown during all API fetches
- [ ] Like/unlike reflects in UI immediately (optimistic update)
- [ ] All API functions and pane Update() handlers tested

## Tasks
- [ ] Library API calls -- Implement all Spotify library endpoint methods
      - test: `TestGetPlaylists_Success`, `TestGetPlaylists_Empty`, `TestGetPlaylistTracks_Success`, `TestGetSavedAlbums_Success`, `TestGetLikedTracks_Success`, `TestGetRecentlyPlayed_Success`, `TestLikeTrack_SendsPUT`, `TestUnlikeTrack_SendsDELETE`
- [ ] Library models -- Add model structs for library data (SimplePlaylist, SavedAlbum, SavedTrack, PlayHistory)
      - test: `TestSimplePlaylist_Unmarshal`, `TestSavedAlbum_Unmarshal`, `TestSavedTrack_Unmarshal`, `TestPlayHistory_Unmarshal`
- [ ] Section/tree data structure -- Build collapsible section tree with cursor navigation
      - test: `TestLibraryTree_MoveDown`, `TestLibraryTree_MoveUp`, `TestLibraryTree_MoveDown_AtBottom`, `TestLibraryTree_ToggleSection_Expands`, `TestLibraryTree_ToggleSection_Collapses`, `TestLibraryTree_SelectedItem`
- [ ] LibraryPane model -- Implement as tea.Model with Init/Update/View
      - test: `TestLibraryPane_Init_FetchesPlaylistsAndRecent`, `TestLibraryPane_View_ShowsSections`, `TestLibraryPane_View_PlayingIndicator`, `TestLibraryPane_Update_Enter_OnPlaylist`, `TestLibraryPane_Update_Enter_OnSection`, `TestLibraryPane_Update_A_AddsToQueue`, `TestLibraryPane_Update_L_ToggleLike`, `TestLibraryPane_View_EmptySection`
- [ ] Lazy loading + pagination -- Sections load lazily on expand with scroll-near-bottom pagination
      - test: `TestLibraryPane_ExpandSection_FetchesIfNotCached`, `TestLibraryPane_ExpandSection_SkipsFetchIfCached`, `TestLibraryPane_ScrollNearBottom_LoadsMore`, `TestLibraryPane_LazyLoad_EndToEnd`
- [ ] Integration -- Wire LibraryPane into root app with Tab focus routing
      - test: `TestApp_LibraryPaneRouting`, `TestApp_LibraryPlay_UpdatesPlayback`
