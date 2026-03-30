---
title: "Playlist Manager"
feature: 09-playlists
status: done
---

## Background
This story built the complete playlist management experience: a PlaylistsClient with all CRUD API methods, a dual-pane PlaylistManager UI model with inline text input for create/rename, optimistic track removal and reordering with rollback on error, and view-switching wired through the root model. The text input pattern uses `github.com/charmbracelet/bubbles/textinput`, and all styling uses Theme interface tokens.

## Design

### Store fields
```go
Playlists []api.SimplePlaylist // shared with Library -- mutations refresh via re-fetch
```

### Text input pattern
Use `github.com/charmbracelet/bubbles/textinput` for the name input. Call `m.nameInput.Focus()` when the modal opens. Input value is `m.nameInput.Value()`. Pressing `Enter` submits; pressing `Esc` cancels.

### Message types
```go
type playlistCreatedMsg      struct{ playlist api.SimplePlaylist }
type playlistRenamedMsg      struct{ playlistID, newName string }
type playlistTracksAddedMsg  struct{ playlistID string; count int }
```

### API methods
```go
CreatePlaylist(ctx context.Context, name, description string, public bool) (*Playlist, error)
UpdatePlaylist(ctx context.Context, id, name, description string) error
AddTracksToPlaylist(ctx context.Context, playlistID string, uris []string) error
RemoveTracksFromPlaylist(ctx context.Context, playlistID string, uris []string) error
ReorderPlaylistTracks(ctx context.Context, id string, rangeStart, insertBefore, rangeLength int) error
```

### Design tokens
`theme.SurfaceAlt()` . `theme.ActiveBorder()` . `theme.TextPrimary()` . `theme.Error()` . `theme.SelectedBg()` . `theme.KeyHint()`

### Playlist Manager Layout

```
+---------------------------------------------------------------------------+
|  Spotnik  [PLAYLISTS]                                 * MacBook Pro        |
+------------------------+--------------------------------------------------+
|  MY PLAYLISTS          |  Chill Vibes                    R Rename  + Add   |
|  --------------------  |  ----------------------------------------------- |
|                        |                                                   |
|  > Chill Vibes   (24)  |  1  Blinding Lights  .  The Weeknd         4:20  |
|    Workout Mix   (48)  |  2  Levitating        .  Dua Lipa           3:23 |
|    Late Night   (112)  |  3  Save Your Tears  .  The Weeknd         3:35  |
|    Road Trip     (67)  |  4  Peaches          .  Justin Bieber      3:18  |
|    Coding Focus  (33)  |  5  Mood             .  24kGoldn           2:21  |
|    + New Playlist      |  ...                                              |
|                        |                                                   |
|                        |  24 tracks . ~1hr 34min                           |
+------------------------+--------------------------------------------------+
|  Enter play   r rename   n new playlist   x remove track   arrows reorder  |
+---------------------------------------------------------------------------+
```

### Operations

**Create Playlist:**
- User presses `n` in playlist list
- Inline text input appears: `New playlist name: _`
- Enter to create, Esc to cancel
- Calls `POST /me/playlists` with `{ "name": ..., "public": false }`

**Rename Playlist:**
- User presses `r` with a playlist selected
- Inline text input overlays the playlist name
- Enter to save, Esc to cancel
- Calls `PUT /playlists/{id}` with `{ "name": ... }`
- List updates immediately (optimistic)

**Remove Track:**
- User presses `x` on a track in the right pane
- Confirmation: `Remove "Blinding Lights"? [y/N]`
- `y` -> `DELETE /playlists/{id}/items` with track URI
- Track removed immediately (optimistic)

**Reorder Tracks:**
- `Shift+Up/Down` to move selected track
- Calls `PUT /playlists/{id}/items` with range params
- List updates immediately (optimistic)
- On API error: revert to original order

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

### Files

| File | Purpose |
|---|---|
| `internal/api/playlists.go` | Playlist CRUD API calls |
| `internal/api/playlists_test.go` | Tests with mock server |
| `internal/ui/panes/playlists.go` | PlaylistManager model |
| `internal/ui/panes/playlists_test.go` | Update tests |

### Out of Scope
- Collaborative playlist management
- Playlist image/cover art editing
- Playlist duplication
- Smart playlist rules
- Folder/group organization
- Public/private toggle (future enhancement)

## Acceptance Criteria
- [ ] `3` opens Playlist Manager with playlists loaded from store
- [ ] New playlist created and visible within 1 second
- [ ] Playlist rename updates immediately (optimistic), reverts on API error
- [ ] Track removed from playlist immediately (optimistic), reverts on error
- [ ] Track reorder moves visually before API confirms, reverts on error
- [ ] On any API error: list reverts to previous state, error shown in status bar
- [ ] All API calls and pane `Update()` handlers tested

## Tasks
- [ ] Playlist API calls -- Implement all playlist mutation methods
      - test: `TestCreatePlaylist_Success`, `TestCreatePlaylist_ServerError`, `TestUpdatePlaylist_Success`, `TestAddTracksToPlaylist_Success`, `TestRemoveTracksFromPlaylist_Success`, `TestReorderPlaylistTracks_Success`, `TestReorderPlaylistTracks_Error`
- [ ] PlaylistManager model (left pane) -- Playlist list with create/rename inline text input
      - test: `TestPlaylistManager_View_PlaylistList`, `TestPlaylistManager_View_PlayingIndicator`, `TestPlaylistManager_Update_N_OpensInput`, `TestPlaylistManager_Update_R_OpensRename`, `TestPlaylistManager_Update_Enter_SubmitsCreate`, `TestPlaylistManager_Update_Esc_CancelsInput`, `TestPlaylistManager_Update_Enter_SelectsPlaylist`
- [ ] Track list (right pane) -- Track display with remove/reorder and optimistic updates
      - test: `TestPlaylistTracks_View_TrackList`, `TestPlaylistTracks_View_Footer`, `TestPlaylistTracks_Update_X_ShowsConfirmation`, `TestPlaylistTracks_Update_Y_ConfirmsRemove`, `TestPlaylistTracks_Update_ShiftDown_ReordersDown`, `TestPlaylistTracks_Update_ShiftUp_ReordersUp`, `TestPlaylistTracks_ReorderRevert_OnError`, `TestPlaylistTracks_RemoveRevert_OnError`
- [ ] View switching -- Wire Playlist Manager into root model with `3`/`1` key switching
      - test: `TestApp_3KeyOpensPlaylists`, `TestApp_1KeyReturnsFromPlaylists`, `TestApp_PlaylistsReusesLibraryData`
