---
title: "Playlist full functionality — track sub-view, lazy pagination, playback"
feature: 20-playback-context
status: open
---

## Background

The `PlaylistsPane` has the complete UI skeleton — list view, track sub-view — but
the drill-down functionality is entirely non-functional:

1. **Track sub-view never loads** — `FetchPlaylistTracksRequestMsg` has no handler
   in `app.go`. The message is emitted, silently dropped, and the track table stays
   empty forever.

2. **Playing a track is missing** — `handleTrackViewKey` has no Enter handler.
   Even if tracks loaded, pressing Enter does nothing.

3. **Interactive data is stored globally** — the current `buildFetchPlaylistTracksCmd`
   writes to the global store via `SetPlaylistTracks`. Playlist track data is
   ephemeral and user-session-scoped: it exists only while the user is viewing that
   sub-view. When the user presses Esc it is no longer needed. The correct pattern —
   already used by search — is pane-owned data, not store-owned.

This story fixes the drill-down and introduces the correct interactive data-loading
pattern for the playlist track sub-view, modelled after `SearchOverlay`.

**Out of scope:** Playlist management operations (rename `r`, remove track `x`,
reorder `Shift+↑/↓`, create `n`) are explicitly excluded from this story. The
existing pane keys for these actions will remain non-functional. Management
operations will be addressed in a dedicated story.

---

## Architecture: Pane-owned interactive data (search pattern)

Playlist tracks are interactive data, not background polled data. The correct pattern:

```
BACKGROUND POLLED DATA (liked songs, albums, queue)         INTERACTIVE DATA (search, playlist tracks)
──────────────────────────────────────────────────         ──────────────────────────────────────────
Triggered by: timer tick                                    Triggered by: user keypress
Priority: api.Background                                    Priority: api.Interactive
Data lives: global Store                                    Data lives: pane fields only
Cleared: never (TTL-based)                                  Cleared: on Esc or new selection
Context: none                                               Context: cancellable (cancel on new request)
Debounce: none                                              Debounce: 150ms (protects rapid switching)
```

The pane owns three categories of fields:

```go
// Identity (what is open)
selectedID   string  // Spotify playlist ID
selectedName string  // display name
selectedURI  string  // Spotify playlist URI — needed for PlayContextMsg

// Data (pane-local, NOT in global store)
loadedTracks  []domain.Track  // all tracks fetched so far for this playlist
trackTotal    int             // total tracks in playlist (from API response)

// Pagination state
trackOffset   int   // count of tracks fetched so far (= len(loadedTracks))
hasMoreTracks bool  // last page was full (len(page) == 100), more likely exist
tracksFetching bool // a request is in-flight; blocks duplicate prefetch

// Debounce (protects rapid playlist switching)
playlistIntent playlistDebounceIntent  // snapshot of current desired playlist
```

---

## API: GET /playlists/{id}/tracks

### Endpoint

```
GET /v1/playlists/{playlist_id}/tracks
Authorization: Bearer {token}
Scope: playlist-read-private (for private playlists)
```

### Query parameters

| Parameter | Type    | Required | Notes                                    |
|-----------|---------|----------|------------------------------------------|
| `limit`   | integer | No       | 1–100, default 100. Use 100 always.      |
| `offset`  | integer | No       | 0-based item index. First page = 0.      |
| `fields`  | string  | No       | Not used — fetch all fields.             |
| `market`  | string  | No       | Not used.                                |

### Response (200 OK)

```json
{
  "href": "https://api.spotify.com/v1/playlists/ABC/tracks?offset=0&limit=100",
  "items": [
    {
      "added_at": "2024-01-15T10:30:00Z",
      "added_by": { "id": "user123" },
      "is_local": false,
      "track": {
        "id": "4iV5W9uYEdYUVa79Axb7Rh",
        "name": "Weightless",
        "uri": "spotify:track:4iV5W9uYEdYUVa79Axb7Rh",
        "duration_ms": 489620,
        "artists": [
          { "id": "xyz", "name": "Marconi Union", "uri": "spotify:artist:xyz" }
        ],
        "album": {
          "id": "abc",
          "name": "Weightless",
          "uri": "spotify:album:abc",
          "release_date": "2012-09-04"
        },
        "popularity": 72
      }
    }
  ],
  "limit": 100,
  "next": "https://api.spotify.com/v1/playlists/ABC/tracks?offset=100&limit=100",
  "offset": 0,
  "previous": null,
  "total": 342
}
```

### Key fields for pagination

- `total` — total number of tracks in the playlist (constant across pages)
- `next` — non-null string when more pages exist; null on last page
- `items[].is_local` — local files have no Spotify URI; skip them for playback
- `items[].track` — may be null for unavailable tracks; skip nulls

### Pagination logic

```
Page 1: offset=0,   limit=100 → items[0..99],   total=342, next=non-null
Page 2: offset=100, limit=100 → items[100..199], total=342, next=non-null
Page 3: offset=200, limit=100 → items[200..299], total=342, next=non-null
Page 4: offset=300, limit=100 → items[300..341], total=342, next=null  ← last page

hasMoreTracks = (next != "")  ← use this field, not len(items) == 100
                               ← edge case: exactly 100 tracks → next is null → correct
```

### Error cases

| HTTP Status | Meaning                                            | Action              |
|-------------|----------------------------------------------------|---------------------|
| 401         | Expired token                                      | Token refresh (auto)|
| 403         | Not owner/collaborator (Feb 2026 restriction)      | Toast warning       |
| 404         | Playlist not found or deleted                      | Toast error, Esc    |
| 429         | Rate limited                                       | RateLimitedMsg      |

---

## Required code changes

### 1. Update `LibraryAPI` interface and `PlaylistTracks` method

The current method signature discards `total` and `next`. Update to return them.

**`internal/api/library_interfaces.go`** — update interface:
```go
// PlaylistTracks fetches a page of playlist tracks. Returns tracks, total count,
// and whether a next page exists.
PlaylistTracks(ctx context.Context, playlistID string, limit, offset int) ([]Track, int, bool, error)
//                                                                                 ^total ^hasNext
```

**`internal/api/library.go`** — update implementation:
```go
func (l *LibraryClient) PlaylistTracks(ctx context.Context, playlistID string, limit, offset int) ([]Track, int, bool, error) {
    path := fmt.Sprintf("/v1/playlists/%s/tracks", playlistID)
    req, err := l.newRequest(ctx, http.MethodGet, path, nil)
    if err != nil {
        return nil, 0, false, fmt.Errorf("creating get playlist tracks request: %w", err)
    }
    q := req.URL.Query()
    q.Set("limit", strconv.Itoa(limit))
    q.Set("offset", strconv.Itoa(offset))
    req.URL.RawQuery = q.Encode()

    var response struct {
        Items []struct {
            IsLocal bool  `json:"is_local"`
            Track   *Track `json:"track"`    // pointer: can be null for unavailable tracks
        } `json:"items"`
        Total int    `json:"total"`
        Next  string `json:"next"`  // empty string when null in JSON
    }
    if err := l.doJSON(req, &response); err != nil {
        return nil, 0, false, fmt.Errorf("getting playlist tracks: %w", err)
    }

    tracks := make([]Track, 0, len(response.Items))
    for _, item := range response.Items {
        // Skip local files (no Spotify URI) and unavailable tracks (null track object).
        if item.IsLocal || item.Track == nil || item.Track.URI == "" {
            continue
        }
        tracks = append(tracks, *item.Track)
    }
    return tracks, response.Total, response.Next != "", nil
}
```

**Note on null track:** `item.Track` is declared as `*Track` (pointer). The current
implementation uses `Track` (value) — change to pointer to handle null entries in the
JSON array (unavailable tracks return `"track": null`).

### 2. Update `PlaylistTracksLoadedMsg` and `FetchPlaylistTracksRequestMsg`

**`internal/ui/panes/messages.go`** — update both types:

```go
// FetchPlaylistTracksRequestMsg is emitted by PlaylistsPane when it needs to
// load (or page) playlist tracks. Offset 0 = initial load; Offset > 0 = next page.
type FetchPlaylistTracksRequestMsg struct {
    PlaylistID string
    Offset     int  // NEW: 0 for first page, len(loadedTracks) for subsequent pages
}

// PlaylistTracksLoadedMsg is returned by the playlist tracks fetch command.
// Tracks contains the page of tracks. Total is the playlist's total track count.
// HasNext is true when more pages are available (API next != "").
// Offset mirrors the request offset so the pane can detect stale responses.
type PlaylistTracksLoadedMsg struct {
    PlaylistID string
    Tracks     []domain.Track
    Total      int   // NEW
    HasNext    bool  // NEW: true when next page exists
    Offset     int   // NEW: mirrors request offset
    Err        error
}

// PlaylistTrackViewClosedMsg is emitted by PlaylistsPane when the user presses
// Esc to return to the playlist list. App.go uses it to cancel any in-flight
// playlist track fetch and clear the staleness key.
type PlaylistTrackViewClosedMsg struct{}  // NEW
```

### 3. New debounce types in `playlists_pane.go`

Add at the top of `internal/ui/panes/playlists_pane.go` (alongside struct definition):

```go
// playlistDebounceIntent is a snapshot of the user's desired playlist at the
// moment of pressing Enter. The debounce tick carries this snapshot; if the
// current intent has changed by the time the tick fires, the tick is discarded.
type playlistDebounceIntent struct {
    playlistID string
}

// playlistDebounceMsg is the internal 150ms tick fired by schedulePlaylistDebounce.
// It is never forwarded to app.go — handled entirely within the pane.
type playlistDebounceMsg struct {
    intent playlistDebounceIntent
}
```

### 4. Update `PlaylistsPane` struct fields

```go
type PlaylistsPane struct {
    store   *state.Store
    theme   theme.Theme
    focused bool
    width   int
    height  int
    table      *components.Table
    filter     *components.Filter
    trackTable *components.Table

    // Sub-view identity
    inTrackView  bool
    selectedID   string
    selectedName string
    selectedURI  string  // NEW: needed for PlayContextMsg

    // Sub-view data (pane-owned, NOT in global store)
    loadedTracks []domain.Track  // NEW: all tracks fetched for this playlist
    trackTotal   int             // NEW: total from API response

    // Pagination state (pane-owned)
    trackOffset   int   // NEW: count of tracks loaded so far
    hasMoreTracks bool  // NEW: API returned next != ""
    tracksFetching bool // NEW: request in-flight sentinel

    // Debounce
    playlistIntent playlistDebounceIntent  // NEW: current desired playlist
}
```

### 5. Debounce mechanism in `playlists_pane.go`

```go
// schedulePlaylistDebounce snapshots the current playlist intent and returns
// a 150ms tick. Stale ticks are discarded in handlePlaylistDebounce.
// Used only for the initial fetch (offset 0) triggered by Enter in list view.
func (p *PlaylistsPane) schedulePlaylistDebounce() tea.Cmd {
    snapshot := p.playlistIntent
    return tea.Tick(150*time.Millisecond, func(_ time.Time) tea.Msg {
        return playlistDebounceMsg{intent: snapshot}
    })
}

// handlePlaylistDebounce fires when a 150ms debounce tick arrives.
// It discards stale ticks (user switched to a different playlist) and
// blocks duplicate requests (tracksFetching is already true).
func (p *PlaylistsPane) handlePlaylistDebounce(m playlistDebounceMsg) (tea.Model, tea.Cmd) {
    // Stale: user switched to a different playlist before tick fired.
    if m.intent.playlistID != p.playlistIntent.playlistID {
        return p, nil
    }
    // Already fetching: a request is in-flight for this playlist.
    // This happens when user Enter → Esc → Enter on same playlist quickly.
    if p.tracksFetching {
        return p, nil
    }
    // Fire the initial fetch.
    p.tracksFetching = true
    return p, func() tea.Msg {
        return FetchPlaylistTracksRequestMsg{
            PlaylistID: p.playlistIntent.playlistID,
            Offset:     0,
        }
    }
}
```

### 6. Updated `Update()` in `playlists_pane.go`

The pane's `Update()` must handle the debounce message and `PlaylistTracksLoadedMsg`
regardless of focus (data messages always arrive even when pane is unfocused):

```go
func (p *PlaylistsPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch m := msg.(type) {

    case playlistDebounceMsg:
        return p.handlePlaylistDebounce(m)

    case LibraryLoadedMsg:
        p.refreshPlaylistRows()
        return p, nil

    case PlaylistTracksLoadedMsg:
        // Guard: only process if this matches the currently selected playlist.
        // Discards responses that arrive after user switched playlists.
        if m.PlaylistID != p.selectedID {
            return p, nil
        }
        p.tracksFetching = false
        if m.Err != nil {
            // Error is toasted by app.go. Pane just clears loading state.
            return p, nil
        }
        if m.Offset == 0 {
            // Initial page: replace
            p.loadedTracks = m.Tracks
        } else {
            // Subsequent page: append
            p.loadedTracks = append(p.loadedTracks, m.Tracks...)
        }
        p.trackOffset = len(p.loadedTracks)
        p.trackTotal = m.Total
        p.hasMoreTracks = m.HasNext
        p.refreshTrackRows()  // reads from p.loadedTracks, NOT store
        return p, nil

    }

    if !p.focused {
        return p, nil
    }
    // ... filter handling, key handling unchanged
}
```

### 7. Updated `handleListViewKey` Enter case

```go
case key.Type == tea.KeyEnter:
    playlist := p.filteredPlaylist()
    idx := p.table.SelectedIndex()
    if idx >= 0 && idx < len(playlist) {
        pl := playlist[idx]

        // Update identity
        p.selectedID = pl.ID
        p.selectedName = pl.Name
        p.selectedURI = pl.URI  // NEW

        // Reset sub-view data (new playlist, old data invalid)
        p.loadedTracks = nil
        p.trackOffset = 0
        p.trackTotal = 0
        p.hasMoreTracks = false
        p.tracksFetching = false  // cleared here; set true in debounce handler

        // Update debounce intent
        p.playlistIntent = playlistDebounceIntent{playlistID: pl.ID}

        // Switch to track sub-view immediately (shows empty table while loading)
        p.inTrackView = true
        p.table.SetFocused(false)
        p.trackTable.SetFocused(true)
        p.resizeTable()
        p.refreshTrackRows()  // shows 0 rows initially

        // Schedule 150ms debounce for initial fetch
        return p, p.schedulePlaylistDebounce()
    }
    return p, nil
```

### 8. Updated `handleTrackViewKey` with Enter + Esc + prefetch

```go
func (p *PlaylistsPane) handleTrackViewKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch {
    case key.Type == tea.KeyEsc:
        // Return to playlist list and emit closed message for app.go to cancel
        // any in-flight fetch.
        p.inTrackView = false
        p.trackTable.SetFocused(false)
        p.table.SetFocused(true)
        p.resizeTable()
        return p, func() tea.Msg { return PlaylistTrackViewClosedMsg{} }  // NEW

    case key.Type == tea.KeyEnter:  // NEW: play selected track
        if idx := p.trackTable.SelectedIndex(); idx >= 0 && idx < len(p.loadedTracks) {
            track := p.loadedTracks[idx]
            playlistURI := p.selectedURI
            return p, func() tea.Msg {
                return PlayContextMsg{
                    ContextURI: playlistURI,
                    OffsetURI:  track.URI,
                }
            }
        }
        return p, nil

    }

    // Forward j/k and other navigation to the track table.
    cmd := p.trackTable.Update(key)
    // After navigation, check if we should prefetch the next page.
    prefetchCmd := p.checkPrefetch()
    return p, tea.Batch(cmd, prefetchCmd)
}

// checkPrefetch fires a next-page request when the cursor is within 10 rows
// of the last loaded track and more pages are available.
// Pagination requests bypass the debounce — they fire immediately.
func (p *PlaylistsPane) checkPrefetch() tea.Cmd {
    if !p.hasMoreTracks || p.tracksFetching {
        return nil
    }
    cursor := p.trackTable.SelectedIndex()
    if cursor < len(p.loadedTracks)-10 {
        return nil
    }
    p.tracksFetching = true
    id := p.selectedID
    offset := p.trackOffset
    return func() tea.Msg {
        return FetchPlaylistTracksRequestMsg{PlaylistID: id, Offset: offset}
    }
}
```

### 9. Updated `refreshTrackRows` reads from pane, not store

```go
func (p *PlaylistsPane) refreshTrackRows() {
    rows := make([]map[string]string, len(p.loadedTracks))
    for i, track := range p.loadedTracks {
        artistName := ""
        if len(track.Artists) > 0 {
            artistName = track.Artists[0].Name
        }
        rows[i] = map[string]string{
            "index":    fmt.Sprintf("%d", i+1),
            "track":    track.Name,
            "artist":   artistName,
            "duration": formatDurationMs(track.DurationMs),
        }
    }
    p.trackTable.SetRows(rows)
}
```

### 10. Updated `buildFetchPlaylistTracksCmd` with context + new signature

**`internal/app/commands.go`**:

```go
// buildFetchPlaylistTracksCmd creates a command that fetches one page of playlist
// tracks using Interactive priority. The context is cancellable — app.go cancels
// it when the user switches to a different playlist or presses Esc.
// No Store writes — data is returned in PlaylistTracksLoadedMsg for the pane.
func (a *App) buildFetchPlaylistTracksCmd(ctx context.Context, playlistID string, offset int) tea.Cmd {
    library := a.library
    return func() tea.Msg {
        // Check for cancellation before making the HTTP call.
        if ctx.Err() != nil {
            return nil
        }
        if library == nil {
            return panes.PlaylistTracksLoadedMsg{Err: errNilClient, PlaylistID: playlistID, Offset: offset}
        }
        tracks, total, hasNext, err := library.PlaylistTracks(
            api.WithPriority(ctx, api.Interactive),
            playlistID, 100, offset,
        )
        if err != nil {
            // Check cancellation again — context.Canceled is expected on playlist switch.
            if ctx.Err() != nil {
                return nil  // silently discard; not an error worth toasting
            }
            if retryAfter := parse429RetryAfter(err); retryAfter > 0 {
                return panes.RateLimitedMsg{RetryAfterSecs: retryAfter}
            }
            if isUnauthorizedError(err) {
                return unauthorizedMsg{}
            }
            return panes.PlaylistTracksLoadedMsg{PlaylistID: playlistID, Offset: offset, Err: err}
        }
        return panes.PlaylistTracksLoadedMsg{
            PlaylistID: playlistID,
            Tracks:     tracks,
            Total:      total,
            HasNext:    hasNext,
            Offset:     offset,
        }
    }
}
```

### 11. New app.go fields (alongside searchCancel pattern)

**`internal/app/app.go`** — add to `App` struct:

```go
// Playlist track sub-view: interactive fetch state (mirrors searchCancel/searchQuery)
playlistTracksCancel context.CancelFunc  // cancels the active playlist tracks fetch
playlistTracksID     string              // staleness key: ID of the playlist being fetched
```

In `New()`, initialise cancel to no-op:
```go
playlistTracksCancel: func() {},
```

### 12. New app.go message handlers

**`internal/app/app.go`** — add to the main `Update()` switch:

```go
case panes.FetchPlaylistTracksRequestMsg:
    // Cancel any prior in-flight fetch (user switched playlists or re-entered same one).
    a.playlistTracksCancel()
    ctx, cancel := context.WithCancel(context.Background())
    a.playlistTracksCancel = cancel
    a.playlistTracksID = m.PlaylistID
    return a, a.buildFetchPlaylistTracksCmd(ctx, m.PlaylistID, m.Offset)

case panes.PlaylistTracksLoadedMsg:
    // Staleness gate: discard if user has already switched to a different playlist.
    if m.PlaylistID != a.playlistTracksID {
        return a, nil
    }
    if m.Err != nil {
        return a, tea.Batch(
            a.forwardToPlaylistsPane(m),
            a.alerts.NewAlertCmd("error", fmt.Sprintf("Failed to load playlist tracks: %s", m.Err.Error())),
        )
    }
    // Forward to pane — pane owns the data, not the store.
    return a, a.forwardToPlaylistsPane(m)

case panes.PlaylistTrackViewClosedMsg:
    // User pressed Esc — cancel any in-flight fetch.
    a.playlistTracksCancel()
    a.playlistTracksCancel = func() {}
    a.playlistTracksID = ""
    return a, nil
```

**Helper** — add to `app.go`:
```go
// forwardToPlaylistsPane forwards a message to PlaylistsPane and returns any cmd.
func (a *App) forwardToPlaylistsPane(msg tea.Msg) tea.Cmd {
    pp := a.playlistsPane()
    if pp == nil {
        return nil
    }
    updated, cmd := pp.Update(msg)
    if p, ok := updated.(*panes.PlaylistsPane); ok {
        a.panes[layout.PanePlaylists] = p
    }
    return cmd
}
```

---

## Complete user flow diagrams

### Flow A — Normal: open playlist, browse, play track

```
┌─ Playlists pane (list view) ──────────────────────────────────┐
│  #  Name                  Tracks                              │
│  1  LoFi Chill            42                                  │
│  2  Morning Drive         180  ◄── cursor here                │
│  3  Deep Focus            67                                  │
└───────────────────────────────────────────────────────────────┘
                   │
                   │ user presses Enter
                   ▼
PlaylistsPane.handleListViewKey()
  selectedID   = "MORNING_ID"
  selectedName = "Morning Drive"
  selectedURI  = "spotify:playlist:MORNING_ID"
  loadedTracks = nil    ← reset
  trackOffset  = 0      ← reset
  tracksFetching = false ← reset (set true by debounce handler)
  playlistIntent = {playlistID: "MORNING_ID"}
  inTrackView = true    ← immediate UI switch
  emit schedulePlaylistDebounce() → 150ms tick
                   │
                   │ (immediately)
                   ▼
┌─ Playlists pane (track sub-view, loading) ────────────────────┐
│  Playlists ── Morning Drive (loading...)     Esc back         │
│  #  Track                   Artist          Duration          │
│                                                               │
│  (empty — fetch in progress)                                  │
└───────────────────────────────────────────────────────────────┘
                   │
                   │ 150ms debounce tick fires
                   ▼
handlePlaylistDebounce():
  tick.intent.playlistID == p.playlistIntent.playlistID → match
  tracksFetching == false → proceed
  p.tracksFetching = true
  emit FetchPlaylistTracksRequestMsg{PlaylistID:"MORNING_ID", Offset:0}
                   │
                   ▼
app.go FetchPlaylistTracksRequestMsg handler:
  a.playlistTracksCancel()           ← cancel previous (no-op on first call)
  ctx, cancel = context.WithCancel()
  a.playlistTracksCancel = cancel
  a.playlistTracksID = "MORNING_ID"
  return buildFetchPlaylistTracksCmd(ctx, "MORNING_ID", 0)
                   │
                   ▼
Command closure:
  ctx.Err() == nil → proceed
  library.PlaylistTracks(WithPriority(ctx, Interactive), "MORNING_ID", 100, 0)
  → GET /v1/playlists/MORNING_ID/tracks?limit=100&offset=0
  → {items:[tracks 0-99], total:180, next:"...?offset=100..."}
  → return PlaylistTracksLoadedMsg{ID:"MORNING_ID", Tracks:[100 items],
                                   Total:180, HasNext:true, Offset:0}
                   │
                   ▼
app.go PlaylistTracksLoadedMsg handler:
  m.PlaylistID == a.playlistTracksID → not stale
  no error → forwardToPlaylistsPane(m)
                   │
                   ▼
PlaylistsPane.Update(PlaylistTracksLoadedMsg):
  m.PlaylistID == p.selectedID → match
  p.tracksFetching = false
  m.Offset == 0 → p.loadedTracks = m.Tracks  (100 items)
  p.trackOffset = 100
  p.trackTotal  = 180
  p.hasMoreTracks = true
  p.refreshTrackRows()
                   │
                   ▼
┌─ Playlists pane (track sub-view, loaded) ─────────────────────┐
│  Playlists ── Morning Drive (180 tracks)   Esc back           │
│  #  Track                   Artist          Duration          │
│  1  Sunrise                 Artist A        3:45              │
│  2  Open Road               Artist B        4:12              │
│  3  Highway Blues           Artist C        5:01              │
│  ...                                                          │
│  100 Last Loaded Track      Artist D        3:30              │
│  ↓ more tracks...                                             │
└───────────────────────────────────────────────────────────────┘
                   │
                   │ user navigates to row 91+ with j/k
                   ▼
handleTrackViewKey() → trackTable.Update(j) → checkPrefetch():
  cursor(91) >= len(loadedTracks)(100) - 10 → true
  hasMoreTracks = true, tracksFetching = false → proceed
  p.tracksFetching = true
  emit FetchPlaylistTracksRequestMsg{PlaylistID:"MORNING_ID", Offset:100}
                   │
                   ▼
app.go handles it: same context (same playlist), same cancel function
  return buildFetchPlaylistTracksCmd(ctx, "MORNING_ID", 100)
  → GET /v1/playlists/MORNING_ID/tracks?limit=100&offset=100
  → {items:[tracks 100-179], total:180, next:null}
  → PlaylistTracksLoadedMsg{..., Tracks:[80 items], Total:180, HasNext:false, Offset:100}
                   │
                   ▼
PlaylistsPane.Update(PlaylistTracksLoadedMsg):
  m.Offset > 0 → p.loadedTracks = append(loadedTracks, m.Tracks...)  (180 items)
  p.trackOffset = 180
  p.hasMoreTracks = false  ← no more pages
  p.refreshTrackRows()
                   │
                   ▼
┌─ Playlists pane (all 180 tracks loaded) ──────────────────────┐
│  Playlists ── Morning Drive (180 tracks)   Esc back           │
│  ...                                                          │
│  178 Track 178                Artist        3:00  ◄─ cursor   │
│  179 Track 179                Artist        4:00              │
│  180 Track 180                Artist        2:30              │
└───────────────────────────────────────────────────────────────┘
                   │
                   │ user presses Enter on track 178
                   ▼
handleTrackViewKey() Enter:
  track = p.loadedTracks[177]
  emit PlayContextMsg{
      ContextURI: "spotify:playlist:MORNING_ID",
      OffsetURI:  "spotify:track:TRACK178_URI",
  }
                   │
                   ▼
app.go PlayContextMsg handler:
  buildPlayContextCmd("spotify:playlist:MORNING_ID", "spotify:track:TRACK178_URI")
  → PUT /me/player/play
    { "context_uri": "spotify:playlist:MORNING_ID",
      "offset": { "uri": "spotify:track:TRACK178_URI" } }
                   │
                   ▼
Spotify plays Track 178. Queue fills with: [Track 179, Track 180].
```

---

### Flow B — Rapid playlist switching (debounce + context cancellation)

```
User in list view:
  Enter on "LoFi Chill" (ID: "LOFI")
    → playlistIntent = {id:"LOFI"}
    → inTrackView=true, scheduleDebounce() [tick T1 in 150ms]
    → sub-view shows empty (loading)

  (80ms later) user presses Esc
    → inTrackView=false, emit PlaylistTrackViewClosedMsg
    → app.go: playlistTracksCancel() [no-op, debounce not fired yet]
    → app.go: playlistTracksID = ""

  (10ms later) user presses Enter on "Deep Focus" (ID: "DEEP")
    → playlistIntent = {id:"DEEP"}
    → inTrackView=true, scheduleDebounce() [tick T2 in 150ms]
    → sub-view shows empty (loading)

  T1 fires (150ms after first Enter, so ~60ms after second Enter):
    intent.playlistID="LOFI" != p.playlistIntent.playlistID="DEEP" → STALE, discard ✓

  T2 fires (150ms after second Enter):
    intent.playlistID="DEEP" == p.playlistIntent.playlistID="DEEP" → match
    tracksFetching=false → proceed
    p.tracksFetching=true
    emit FetchPlaylistTracksRequestMsg{ID:"DEEP", Offset:0}
    → only one HTTP call fires ✓
```

---

### Flow C — Enter on same playlist twice rapidly (tracksFetching guard)

```
User in list view, cursor on "Morning Drive":
  Enter #1:
    → inTrackView=true (state changes immediately in this Update() call)
    → scheduleDebounce() → tick T1

  Enter #2 arrives before T1 fires:
    → pane is now in inTrackView=true → goes to handleTrackViewKey
    → no Enter handler outcome (loadedTracks is nil) → nothing emitted ✓
    (OR: if Enter-to-play is handled, loadedTracks is nil so Enter does nothing)

  T1 fires:
    → intent matches → tracksFetching=false → fires FetchPlaylistTracksRequestMsg ✓
    → only one request ✓
```

---

### Flow D — Esc while tracks are loading

```
Track sub-view is visible, fetch is in-flight:
  user presses Esc
    → handleTrackViewKey Esc:
        inTrackView = false
        emit PlaylistTrackViewClosedMsg

    → app.go PlaylistTrackViewClosedMsg:
        a.playlistTracksCancel()   ← cancels HTTP call
        a.playlistTracksID = ""

  (HTTP call returns after cancellation):
    → Command closure: ctx.Err() != nil → return nil  (no Msg sent) ✓

  OR (HTTP already returned, Msg queued):
    → PlaylistTracksLoadedMsg arrives
    → app.go: m.PlaylistID != a.playlistTracksID ("MORNING" != "") → STALE, discard ✓
```

---

### Flow E — Network error fetching tracks

```
  FetchPlaylistTracksRequestMsg fires → HTTP request fails (404 / 403 / 5xx)
    → PlaylistTracksLoadedMsg{Err: err}
    → app.go: not stale → forwardToPlaylistsPane + toast "Failed to load playlist tracks"
    → pane: tracksFetching=false
    → sub-view stays visible with empty table
    → user can press Esc to go back ✓

  Special case — 403 (not owner/collaborator, Feb 2026 API restriction):
    → same flow: toast "Failed to load playlist tracks" (HTTP error message will contain 403 context)
    → NOTE: this is a Spotify API restriction; the app surfaces the error correctly ✓
```

---

### Flow F — Empty playlist (0 tracks)

```
  FetchPlaylistTracksRequestMsg fires for empty playlist
  → GET /v1/playlists/{id}/tracks?limit=100&offset=0
  → {items:[], total:0, next:null}
  → PlaylistTracksLoadedMsg{Tracks:[], Total:0, HasNext:false, Offset:0}
  → pane: loadedTracks=[], trackTotal=0, hasMoreTracks=false
  → refreshTrackRows() → table shows 0 rows
  → Title: "Playlists ── My Empty Playlist (0 tracks)"
  → Enter in track sub-view: idx check fails → nothing played ✓
```

---

## Edge cases summary

| Scenario | Behaviour |
|---|---|
| Empty playlist (0 tracks) | Sub-view shows empty table; Enter does nothing |
| Playlist with exactly 100 tracks | `next` is null → `hasMoreTracks=false` → no extra page request ✓ |
| Playlist with 101 tracks | First page: 100 items, `next`=non-null → prefetch fires → second page: 1 item ✓ |
| Local files in playlist | Skipped in API layer (`is_local=true`) — not shown in track table |
| Unavailable tracks (`"track": null`) | Skipped in API layer — not shown in track table |
| Rapid Enter on same playlist | `inTrackView=true` after first Enter blocks second Enter in list view ✓ |
| Enter → Esc → Enter (same playlist) | `tracksFetching` guard in debounce handler prevents duplicate request ✓ |
| Enter → Esc → Enter (different playlist) | Debounce stale check discards old tick; only new playlist fetches ✓ |
| Esc while loading | `PlaylistTrackViewClosedMsg` cancels context; stale Msg discarded ✓ |
| 403 on tracks fetch | Toast; sub-view stays with empty table; user can Esc ✓ |
| 404 on tracks fetch | Toast; sub-view stays with empty table; user can Esc ✓ |
| API client nil (pre-auth) | `errNilClient` returned; handler silently discards (`errNilClient` case) ✓ |
| Management keys (r, x, Shift+↑/↓, n) | **Out of scope** — keys remain non-functional in this story |

---

## Files changed

| File | Change |
|---|---|
| `internal/api/library_interfaces.go` | Update `PlaylistTracks` signature: returns `([]Track, int, bool, error)` |
| `internal/api/library.go` | Update `PlaylistTracks` implementation: capture `total`, `next`, null track guard |
| `internal/api/apitest/mock.go` | Update `MockLibrary.PlaylistTracks` to match new signature |
| `internal/ui/panes/messages.go` | Update `FetchPlaylistTracksRequestMsg` (add `Offset`), update `PlaylistTracksLoadedMsg` (add `Total`, `HasNext`, `Offset`), add `PlaylistTrackViewClosedMsg` |
| `internal/ui/panes/playlists_pane.go` | New fields, debounce types, `schedulePlaylistDebounce`, `handlePlaylistDebounce`, `checkPrefetch`, updated `handleListViewKey` Enter, updated `handleTrackViewKey` (Enter + Esc), updated `refreshTrackRows` (reads `loadedTracks`) |
| `internal/app/app.go` | New `playlistTracksCancel`/`playlistTracksID` fields; 3 new message handlers (`FetchPlaylistTracksRequestMsg`, `PlaylistTracksLoadedMsg`, `PlaylistTrackViewClosedMsg`); `forwardToPlaylistsPane` helper |
| `internal/app/commands.go` | Update `buildFetchPlaylistTracksCmd`: add `ctx context.Context` param, `offset int`, new return signature call, `api.Interactive` priority |

---

## Acceptance Criteria

- [ ] Pressing Enter on a playlist opens the track sub-view and starts loading tracks
- [ ] Track sub-view shows 0 rows while loading, populates when `PlaylistTracksLoadedMsg` arrives
- [ ] Title shows playlist name and track count: `Playlists ── Morning Drive (180 tracks)`
- [ ] Scrolling to within 10 rows of the bottom triggers the next page fetch (offset-based)
- [ ] Tracks from all pages are appended and visible in the table
- [ ] `hasMoreTracks` is derived from `next != ""` (not from `len(items) == 100`)
- [ ] Pressing Enter on a track emits `PlayContextMsg` with correct `ContextURI` and `OffsetURI`
- [ ] Spotify plays the selected track with the playlist as context; queue fills with subsequent tracks
- [ ] Pressing Esc returns to playlist list and cancels any in-flight fetch
- [ ] Rapid playlist switching: only the last selected playlist's tracks load
- [ ] All errors (403, 404, 5xx) show a toast; sub-view stays navigable
- [ ] Empty playlists show empty track table with correct track count (0)
- [ ] Local files and null tracks are filtered out and never shown
- [ ] `PlaylistTracksLoadedMsg` handler in pane guards against wrong `PlaylistID`
- [ ] Staleness check in app.go discards `PlaylistTracksLoadedMsg` when `playlistTracksID` has changed
- [ ] `context.Canceled` errors are silently discarded (not toasted)

---

## Tasks

- [ ] Update `PlaylistTracks` API method and interface to return `([]Track, int, bool, error)`;
      update null-track guard to use `*Track` pointer; capture `total` and `next` from response.
      Update `MockLibrary.PlaylistTracks` in `apitest/mock.go` to match.
      - test: response with `"next": null` → `hasNext=false`; with `"next": "..."` → `hasNext=true`;
        `"track": null` items are skipped; `is_local: true` items are skipped

- [ ] Add `Offset int` to `FetchPlaylistTracksRequestMsg`; add `Total int`, `HasNext bool`, `Offset int`
      to `PlaylistTracksLoadedMsg`; add `PlaylistTrackViewClosedMsg`.

- [ ] Add `playlistDebounceIntent`, `playlistDebounceMsg` types; add pane fields
      (`selectedURI`, `loadedTracks`, `trackTotal`, `trackOffset`, `hasMoreTracks`,
      `tracksFetching`, `playlistIntent`); implement `schedulePlaylistDebounce` and
      `handlePlaylistDebounce`.
      - test: debounce tick with stale intent is discarded; tick with matching intent fires
        FetchPlaylistTracksRequestMsg; tick with matching intent when tracksFetching=true is discarded

- [ ] Update `handleListViewKey` Enter: reset sub-view state, set `selectedURI`, update
      `playlistIntent`, schedule debounce instead of directly emitting request.
      - test: Enter emits no immediate request (only a debounce cmd); after debounce resolves,
        FetchPlaylistTracksRequestMsg is emitted with Offset:0

- [ ] Update `handleTrackViewKey`: add Enter (play), update Esc (emit `PlaylistTrackViewClosedMsg`).
      Implement `checkPrefetch`. Management keys (x, Shift+↑/↓) are out of scope.
      - test: Enter on row N emits PlayContextMsg{ContextURI:selectedURI, OffsetURI:tracks[N].URI};
        Esc emits PlaylistTrackViewClosedMsg; checkPrefetch fires when cursor >= len-10 and
        hasMoreTracks and not fetching; checkPrefetch does NOT fire when tracksFetching=true

- [ ] Update `refreshTrackRows` to read from `p.loadedTracks` (not `store.PlaylistTracks`).
      - test: table rows match loadedTracks exactly

- [ ] Update `PlaylistTracksLoadedMsg` handler in pane: guard by PlaylistID; handle
      Offset==0 (replace) vs Offset>0 (append); update all pagination fields.
      - test: Offset=0 replaces loadedTracks; Offset>0 appends; wrong PlaylistID is ignored;
        HasNext=true sets hasMoreTracks=true; HasNext=false sets hasMoreTracks=false

- [ ] Update `buildFetchPlaylistTracksCmd`: add `ctx context.Context` param, use
      `api.WithPriority(ctx, api.Interactive)`, call updated `PlaylistTracks` signature,
      return `nil` on `ctx.Err() != nil` (before and after HTTP call).
      - test: cancelled context before HTTP → nil returned; cancelled context after HTTP → nil;
        success → PlaylistTracksLoadedMsg with correct fields; 429 → RateLimitedMsg

- [ ] Add `playlistTracksCancel`/`playlistTracksID` to App; add 3 app.go message handlers
      (`FetchPlaylistTracksRequestMsg`, `PlaylistTracksLoadedMsg`, `PlaylistTrackViewClosedMsg`);
      add `forwardToPlaylistsPane` helper. Management handlers are out of scope.
      - test: FetchPlaylistTracksRequestMsg cancels prior context and sets new ID; stale
        PlaylistTracksLoadedMsg is discarded; PlaylistTrackViewClosedMsg clears ID and
        calls cancel;
        errors trigger toasts
