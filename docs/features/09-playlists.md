# Feature 09 — Playlist Manager

> **Depends on:** Feature 04 (Library Browser) complete and committed.
> This is the v1.0.0 power feature — do not implement before all prior features are stable.

## Implementation Context

### Store fields this feature uses
```go
Playlists []api.Playlist // shared with Library — mutations refresh the same slice
```

### Text input pattern (new playlist name modal)
Use `github.com/charmbracelet/bubbles/textinput` for the name input.
Call `m.nameInput.Focus()` when the modal opens. Input value is `m.nameInput.Value()`.
Pressing `Enter` submits; pressing `Esc` cancels and restores previous state.

### Message types for this feature
```go
type playlistCreatedMsg struct{ playlist api.Playlist }
type playlistDeletedMsg struct{ playlistID string }
type playlistRenamedMsg struct{ playlistID, newName string }
type playlistTracksAddedMsg struct{ playlistID string; count int }
```

### Design tokens used in this feature
`theme.SurfaceAlt()` · `theme.ActiveBorder()` · `theme.TextPrimary()` ·
`theme.Error()` · `theme.SelectedBg()` · `theme.KeyHint()`

---

---

## Goal

Full playlist management from the terminal. Create playlists, add/remove tracks, rename,
and reorder — all without leaving spotnik. This turns Spotnik into a music curation tool,
not just a player.

---

## User Stories

- **As a user**, I press `3` to open the Playlist Manager view.
- **As a user**, I can create a new playlist with a name and optional description.
- **As a user**, I can rename an existing playlist.
- **As a user**, I can add tracks from search or library to a playlist with a keypress.
- **As a user**, I can remove a track from a playlist.
- **As a user**, I can reorder tracks in a playlist using keyboard shortcuts.
- **As a user**, I can change a playlist between public and private.

---

## Playlist Manager Layout

```
╭──────────────────────────────────────────────────────────────────────────────╮
│  Spotnik  [PLAYLISTS]                                 ◉ MacBook Pro Speakers   │
├────────────────────────┬─────────────────────────────────────────────────────┤
│  MY PLAYLISTS          │  Chill Vibes                    ✎ Rename  + Add     │
│  ────────────────────  │  ───────────────────────────────────────────────── │
│                        │                                                     │
│  ▶ Chill Vibes   (24)  │  1  Blinding Lights  ·  The Weeknd         4:20    │
│    Workout Mix   (48)  │  2  Levitating        ·  Dua Lipa           3:23   │
│    Late Night   (112)  │  3  Save Your Tears  ·  The Weeknd         3:35    │
│    Road Trip     (67)  │  4  Peaches          ·  Justin Bieber      3:18    │
│    Coding Focus  (33)  │  5  Mood             ·  24kGoldn           2:21    │
│    + New Playlist      │  ...                                                │
│                        │                                                     │
│                        │  24 tracks · ~1hr 34min                             │
├────────────────────────┴─────────────────────────────────────────────────────┤
│  Enter play   r rename   n new playlist   x remove track   ↑↓ reorder       │
╰──────────────────────────────────────────────────────────────────────────────╯
```

---

## Operations

### Create Playlist
- User presses `n` in playlist list
- Inline text input appears at bottom of list: `New playlist name: █`
- Press Enter to create, Esc to cancel
- Calls `POST /me/playlists` with `{ "name": ..., "public": false }`
- New playlist appears in list, is selected automatically

### Rename Playlist
- User presses `r` with a playlist selected
- Inline text input overlays the playlist name: `█ Chill Vibes`
- Press Enter to save, Esc to cancel
- Calls `PUT /playlists/{id}` with `{ "name": ... }`
- List updates immediately (optimistic)

### Remove Track
- User presses `x` on a track in the right pane
- Confirmation prompt: `Remove "Blinding Lights"? [y/N]`
- `y` → `DELETE /playlists/{id}/items` with track URI
- Track removed from list immediately (optimistic)

### Reorder Tracks
- User selects a track, presses `Shift+↑` or `Shift+↓` to move it
- Calls `PUT /playlists/{id}/items` with `range_start`, `insert_before`, `range_length: 1`
- List updates immediately (optimistic)
- On API error: revert to original order

### Add Tracks from Search
- From search overlay: select a track, press `p` to add to a specific playlist
- Sub-overlay shows playlist picker: list of playlists with cursor
- Select playlist, press Enter → calls `POST /playlists/{id}/items`

---

## Keymap (Playlist Manager)

| Key | Pane | Action |
|---|---|---|
| `j` / `↓` | Left | Move playlist selection down |
| `k` / `↑` | Left | Move playlist selection up |
| `n` | Left | Create new playlist |
| `r` | Left | Rename selected playlist |
| `Enter` | Left | Select playlist (load tracks in right pane) |
| `Tab` | Either | Switch between left and right pane |
| `j` / `↓` | Right | Move track selection down |
| `k` / `↑` | Right | Move track selection up |
| `Shift+↓` | Right | Move selected track down |
| `Shift+↑` | Right | Move selected track up |
| `x` | Right | Remove selected track (confirm) |
| `Enter` | Right | Play selected track |
| `a` | Right | Add track to queue |
| `1` | Either | Return to Library view |

---

## Files to Create

| File | Purpose |
|---|---|
| `internal/api/playlists.go` | Playlist CRUD API calls |
| `internal/api/playlists_test.go` | Tests with mock server |
| `internal/ui/panes/playlists.go` | PlaylistManager model |
| `internal/ui/panes/playlists_test.go` | Update tests |

---

## Task Breakdown

### Task 8.1 — Playlist API calls
- [ ] `CreatePlaylist(ctx, name, description string, public bool) (*Playlist, error)`
- [ ] `UpdatePlaylist(ctx, id, name, description string) error`
- [ ] `AddTracksToPlaylist(ctx, id string, uris []string, position *int) error`
- [ ] `RemoveTracksFromPlaylist(ctx, id string, uris []string) error`
- [ ] `ReorderPlaylistTracks(ctx, id string, rangeStart, insertBefore, rangeLength int) error`
- [ ] Test each with mock server

### Task 8.2 — PlaylistManager model (left pane)
- [ ] Playlist list with selection
- [ ] `n` key opens inline name input using `bubbles/textinput`
- [ ] `r` key opens rename input pre-filled with current name
- [ ] On confirm: fire create/rename command
- [ ] Test: create flow, rename flow, cancel with Esc

### Task 8.3 — Track list (right pane)
- [ ] Track list with duration column (right-aligned)
- [ ] Total duration + track count in footer
- [ ] `x` key: confirmation prompt before delete
- [ ] `Shift+↑/↓`: reorder with optimistic update
- [ ] Test: remove, reorder, empty playlist

### Task 8.4 — View switching
- [ ] Root model: `3` switches to PlaylistManager
- [ ] Reuse library playlist data from store (no fresh fetch needed)
- [ ] `1` returns to Library view

---

## Acceptance Criteria

- [ ] `3` opens Playlist Manager with playlists loaded
- [ ] New playlist created and visible within 1 second
- [ ] Track removed immediately from list, API call fires in background
- [ ] Reorder moves track visually before API confirm
- [ ] On API error for reorder/remove: list reverts, error shown in status bar
- [ ] All API calls and pane update handlers tested

---

## Out of Scope

- Collaborative playlist management (shared with other users)
- Playlist image/cover art editing
- Playlist duplication
- Smart playlist rules
- Folder/group organization

---

*Last updated: 2026-02-21*
