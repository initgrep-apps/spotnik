---
title: "Playlist Manager"
description: "Full playlist management from the terminal — create, rename, reorder, and remove tracks without leaving Spotnik, turning it into a music curation tool."
status: done
stories: [09]
---

# Playlist Manager

## Background

Spotnik aims to be more than a playback-only client. The Playlist Manager is the v1.0.0 power feature that turns Spotnik into a music curation tool, letting users create playlists, add/remove tracks, rename, and reorder — all without leaving the terminal.

The Playlist Manager uses a dedicated view (activated by pressing `3`) that temporarily replaces the three-pane layout. Pressing `1` returns to the main Library | Player | Queue view. This does not violate the three-pane freeze — see `docs/features/00-overview.md` for the view-switching concept. The feature depends on Feature 04 (Library Browser) being complete and committed, and reuses library playlist data from the store rather than making fresh API calls on view switch.

The view is a dual-pane layout: the left pane lists playlists with track counts, and the right pane displays tracks for the selected playlist with duration, artist, and reorder/remove capabilities. All mutation operations (create, rename, remove, reorder) use optimistic updates that revert on API error, with errors shown via toast notifications in the status bar.

---

## Story: Playlist Manager (spec 09)

### Background

This story built the complete playlist management experience: a PlaylistsClient with all CRUD API methods, a dual-pane PlaylistManager UI model with inline text input for create/rename, optimistic track removal and reordering with rollback on error, and view-switching wired through the root model. The text input pattern uses `github.com/charmbracelet/bubbles/textinput`, and all styling uses Theme interface tokens.

### Acceptance Criteria
- [ ] `3` opens Playlist Manager with playlists loaded from store
- [ ] New playlist created and visible within 1 second
- [ ] Playlist rename updates immediately (optimistic), reverts on API error
- [ ] Track removed from playlist immediately (optimistic), reverts on error
- [ ] Track reorder moves visually before API confirms, reverts on error
- [ ] On any API error: list reverts to previous state, error shown in status bar
- [ ] All API calls and pane `Update()` handlers tested

### Implementation Context

**Store fields:**
```go
Playlists []api.SimplePlaylist // shared with Library — mutations refresh via re-fetch
```

**Text input pattern (new playlist name modal):**
Use `github.com/charmbracelet/bubbles/textinput` for the name input. Call `m.nameInput.Focus()` when the modal opens. Input value is `m.nameInput.Value()`. Pressing `Enter` submits; pressing `Esc` cancels and restores previous state.

**Message types:**
```go
type playlistCreatedMsg      struct{ playlist api.SimplePlaylist }
type playlistRenamedMsg      struct{ playlistID, newName string }
type playlistTracksAddedMsg  struct{ playlistID string; count int }
```

**API methods (see `docs/ARCHITECTURE.md` -> SpotifyClient interface):**
```go
CreatePlaylist(ctx context.Context, name, description string, public bool) (*Playlist, error)
UpdatePlaylist(ctx context.Context, id, name, description string) error
AddTracksToPlaylist(ctx context.Context, playlistID string, uris []string) error
RemoveTracksFromPlaylist(ctx context.Context, playlistID string, uris []string) error
ReorderPlaylistTracks(ctx context.Context, id string, rangeStart, insertBefore, rangeLength int) error
```

**Duration display:**
Track duration comes from the `DurationMs` field on the `api.Track` struct, available from the `GetPlaylistTracks()` response. Format as `m:ss` (e.g., "4:20").

**Design tokens:**
`theme.SurfaceAlt()` . `theme.ActiveBorder()` . `theme.TextPrimary()` . `theme.Error()` . `theme.SelectedBg()` . `theme.KeyHint()`

### Playlist Manager Layout

```
+---------------------------------------------------------------------------+
|  Spotnik  [PLAYLISTS]                                 * MacBook Pro Speakers   |
+------------------------+--------------------------------------------------+
|  MY PLAYLISTS          |  Chill Vibes                    R Rename  + Add     |
|  --------------------  |  ------------------------------------------------- |
|                        |                                                     |
|  > Chill Vibes   (24)  |  1  Blinding Lights  .  The Weeknd         4:20    |
|    Workout Mix   (48)  |  2  Levitating        .  Dua Lipa           3:23   |
|    Late Night   (112)  |  3  Save Your Tears  .  The Weeknd         3:35    |
|    Road Trip     (67)  |  4  Peaches          .  Justin Bieber      3:18    |
|    Coding Focus  (33)  |  5  Mood             .  24kGoldn           2:21    |
|    + New Playlist      |  ...                                                |
|                        |                                                     |
|                        |  24 tracks . ~1hr 34min                             |
+------------------------+--------------------------------------------------+
|  Enter play   r rename   n new playlist   x remove track   arrows reorder    |
+---------------------------------------------------------------------------+
```

### Operations

**Create Playlist:**
- User presses `n` in playlist list
- Inline text input appears at bottom of list: `New playlist name: _`
- Press Enter to create, Esc to cancel
- Calls `POST /me/playlists` with `{ "name": ..., "public": false }`
- New playlist appears in list, is selected automatically

**Rename Playlist:**
- User presses `r` with a playlist selected
- Inline text input overlays the playlist name: `_ Chill Vibes`
- Press Enter to save, Esc to cancel
- Calls `PUT /playlists/{id}` with `{ "name": ... }`
- List updates immediately (optimistic)

**Remove Track:**
- User presses `x` on a track in the right pane
- Confirmation prompt: `Remove "Blinding Lights"? [y/N]`
- `y` -> `DELETE /playlists/{id}/items` with track URI
- Track removed from list immediately (optimistic)

**Reorder Tracks:**
- User selects a track, presses `Shift+Up` or `Shift+Down` to move it
- Calls `PUT /playlists/{id}/items` with `range_start`, `insert_before`, `range_length: 1`
- List updates immediately (optimistic)
- On API error: revert to original order

**Add Tracks from Search:**
- From search overlay: select a track, press `p` to add to a specific playlist
- Sub-overlay shows playlist picker: list of playlists with cursor
- Select playlist, press Enter -> calls `POST /playlists/{id}/items`

**Toggle Public/Private:**
- Not available in MVP. Playlists are created as private by default.
- Future enhancement: add `v` key to toggle visibility.

### Keymap (Playlist Manager)

| Key | Pane | Action |
|---|---|---|
| `j` / `Down` | Left | Move playlist selection down |
| `k` / `Up` | Left | Move playlist selection up |
| `n` | Left | Create new playlist |
| `r` | Left | Rename selected playlist |
| `Enter` | Left | Select playlist (load tracks in right pane) |
| `Tab` | Either | Switch between left and right pane |
| `j` / `Down` | Right | Move track selection down |
| `k` / `Up` | Right | Move track selection up |
| `Shift+Down` | Right | Move selected track down |
| `Shift+Up` | Right | Move selected track up |
| `x` | Right | Remove selected track (confirm) |
| `Enter` | Right | Play selected track |
| `a` | Right | Add track to queue |
| `1` | Either | Return to Library view |

### Tasks

1. **Playlist API calls** — Implement all playlist mutation methods on the API client. Each method maps to a single Spotify Web API endpoint. All methods must wrap errors with context and conform to the `SpotifyClient` interface in `docs/ARCHITECTURE.md`.
   - Files: `internal/api/playlists.go`, `internal/api/playlists_test.go`
   - Implementation:
     - `CreatePlaylist(ctx, name, description string, public bool) (*Playlist, error)`
     - `UpdatePlaylist(ctx, id, name, description string) error`
     - `AddTracksToPlaylist(ctx, playlistID string, uris []string) error`
     - `RemoveTracksFromPlaylist(ctx, playlistID string, uris []string) error`
     - `ReorderPlaylistTracks(ctx, id string, rangeStart, insertBefore, rangeLength int) error`
   - Acceptance: Each method sends the correct HTTP method, path, and JSON body. Errors are wrapped with `fmt.Errorf` including the operation name. All methods tested with `httptest.NewServer`.
   - Tests:
     - `TestCreatePlaylist_Success` — returns created playlist
     - `TestCreatePlaylist_ServerError` — returns descriptive error
     - `TestUpdatePlaylist_Success` — sends correct PUT body
     - `TestAddTracksToPlaylist_Success` — sends correct POST body with URIs
     - `TestRemoveTracksFromPlaylist_Success` — sends correct DELETE body
     - `TestReorderPlaylistTracks_Success` — sends correct PUT body with range params
     - `TestReorderPlaylistTracks_Error` — returns error with context

2. **PlaylistManager model (left pane)** — Build the left pane of the Playlist Manager view. Displays the user's playlists with track counts. Supports creating new playlists and renaming existing ones via inline text input. Selecting a playlist loads its tracks in the right pane.
   - Files: `internal/ui/panes/playlists.go`, `internal/ui/panes/playlists_test.go`
   - Implementation:
     - Playlist list with selection, track count display, and `>` indicator for currently playing
     - `n` key opens inline name input using `bubbles/textinput`
     - `r` key opens rename input pre-filled with current name
     - On confirm: fire create/rename command
     - On cancel (Esc): hide text input, restore previous state
   - Acceptance: Playlists render with name and track count. Currently playing playlist shows `>` indicator. `n` opens text input; Enter creates, Esc cancels. `r` opens text input pre-filled; Enter renames, Esc cancels. All styling uses `Theme` tokens — no hardcoded colors.
   - Tests:
     - `TestPlaylistManager_View_PlaylistList` — renders playlists with track counts
     - `TestPlaylistManager_View_PlayingIndicator` — shows `>` next to currently playing playlist
     - `TestPlaylistManager_Update_N_OpensInput` — text input appears for new playlist name
     - `TestPlaylistManager_Update_R_OpensRename` — text input pre-filled with current name
     - `TestPlaylistManager_Update_Enter_SubmitsCreate` — returns create command with name
     - `TestPlaylistManager_Update_Esc_CancelsInput` — hides text input, restores state
     - `TestPlaylistManager_Update_Enter_SelectsPlaylist` — loads playlist tracks in right pane

3. **Track list (right pane)** — Build the right pane of the Playlist Manager view. Displays tracks for the selected playlist with duration right-aligned. Supports removing tracks (with confirmation) and reordering via Shift+arrow keys. Both operations use optimistic updates that revert on API error.
   - Files: `internal/ui/panes/playlists.go`, `internal/ui/panes/playlists_test.go`
   - Implementation:
     - Track list with duration column (right-aligned, formatted as `m:ss`)
     - Total duration + track count in footer
     - `x` key: confirmation prompt before remove
     - `Shift+Up/Down`: reorder with optimistic update
     - On API error for remove/reorder: revert to previous state, show error in status bar
   - Acceptance: Tracks render with name, artist, and right-aligned duration. Footer shows total tracks and total duration. `x` shows `Remove? [y/N]` prompt; `y` removes optimistically; API failure reverts. `Shift+Up/Down` moves track visually; API failure reverts. All styling uses `Theme` tokens — no hardcoded colors.
   - Tests (unit):
     - `TestPlaylistTracks_View_TrackList` — renders tracks with duration right-aligned
     - `TestPlaylistTracks_View_Footer` — shows total tracks + total duration
     - `TestPlaylistTracks_Update_X_ShowsConfirmation` — shows `Remove? [y/N]` prompt
     - `TestPlaylistTracks_Update_Y_ConfirmsRemove` — returns remove command, track disappears optimistically
     - `TestPlaylistTracks_Update_ShiftDown_ReordersDown` — track moves down, returns reorder command
     - `TestPlaylistTracks_Update_ShiftUp_ReordersUp` — track moves up, returns reorder command
   - Tests (integration):
     - `TestPlaylistTracks_ReorderRevert_OnError` — reorder API fails -> track reverts to original position
     - `TestPlaylistTracks_RemoveRevert_OnError` — remove API fails -> track reappears

4. **View switching** — Wire the Playlist Manager into the root model's view-switching logic. Pressing `3` switches to the Playlist Manager view. Pressing `1` returns to the main three-pane layout. Playlist data is reused from the store (no fresh fetch on view switch).
   - Files: `internal/app/app.go`, `internal/ui/panes/playlists.go`
   - Implementation:
     - Root model: `3` switches to PlaylistManager view
     - Reuse library playlist data from store (no fresh fetch needed)
     - `1` returns to Library view
   - Acceptance: `3` opens Playlist Manager with playlists already loaded from store. `1` returns to the three-pane layout. No additional API call on view switch — data comes from store.
   - Tests (integration):
     - `TestApp_3KeyOpensPlaylists` — pressing `3` switches to PlaylistManager
     - `TestApp_1KeyReturnsFromPlaylists` — pressing `1` restores three-pane layout
     - `TestApp_PlaylistsReusesLibraryData` — playlists loaded from store (no fresh fetch)

### Files

| File | Purpose |
|---|---|
| `internal/api/playlists.go` | Playlist CRUD API calls |
| `internal/api/playlists_test.go` | Tests with mock server |
| `internal/ui/panes/playlists.go` | PlaylistManager model |
| `internal/ui/panes/playlists_test.go` | Update tests |

### Out of Scope

- Collaborative playlist management (shared with other users)
- Playlist image/cover art editing
- Playlist duplication
- Smart playlist rules
- Folder/group organization
- Public/private toggle (playlists are created as private by default; toggle is a future enhancement)
