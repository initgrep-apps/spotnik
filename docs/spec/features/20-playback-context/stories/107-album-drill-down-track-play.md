---
title: "Album drill-down + track play"
feature: 20-playback-context
status: open
---

## Background

The `AlbumsPane` currently plays an album immediately when the user presses Enter:

```go
// albums_pane.go:144 — current Enter handler
case keyMsg.Type == tea.KeyEnter:
    uri := albums[idx].Album.URI
    return a, func() tea.Msg {
        return PlayContextMsg{ContextURI: uri}
    }
```

This means:
- Pressing Enter on an album instantly starts playing that album from track 1
- There is no way to see which tracks are in an album before playing
- There is no way to start playback from a specific track within an album

The user wants the same drill-down pattern that exists in `PlaylistsPane`:
**Enter on album → show tracks → Enter on track → play from that track within album context**.

This is consistent with Story 106 (playlists): album is a Spotify collection context, so
`PlayContextMsg{ContextURI: albumURI, OffsetURI: trackURI}` is the correct play call.

### Why pane-owned data (not store-owned)

Album tracks are interactive/ephemeral data — scoped to a single user session of browsing
one album. The moment the user presses Esc, the data is no longer needed. Storing it in
the global `Store` would pollute shared state with per-session ephemera.

This follows the same architecture established for search overlays and Story 106 playlists:
- `SearchOverlay` owns its own results (`o.results`)
- `PlaylistsPane` (after Story 106) owns its own loaded tracks (`p.loadedTracks`)
- `AlbumsPane` (this story) owns its own loaded tracks (`a.loadedTracks`)

The app's gateway enforces `api.Interactive` priority — bypasses token bucket, waits for
any active rate-limit backoff, gets 100ms transport debounce per unique path, and
deduplicates identical in-flight requests.

---

## Current vs Expected Behaviour

### Current behaviour

```
┌─ Albums pane (list view) ────────────────────────────────┐
│  #   Name                    Artist           Year        │
│  1   Abbey Road               The Beatles      1969        │
│  2   Rumours                  Fleetwood Mac    1977        │
│  3   Kind of Blue             Miles Davis      1959  ◄ Enter
└──────────────────────────────────────────────────────────┘
         │
         ▼  PlayContextMsg{ContextURI: "spotify:album:KindOfBlue"} emitted
         │
         ▼  Spotify immediately starts playing "So What" (track 1)
         │  Queue fills with all album tracks
         │
         NO track sub-view, NO per-track selection
```

### Expected behaviour

```
┌─ Albums pane (list view) ────────────────────────────────┐
│  #   Name                    Artist           Year        │
│  1   Abbey Road               The Beatles      1969        │
│  2   Rumours                  Fleetwood Mac    1977        │
│  3   Kind of Blue             Miles Davis      1959  ◄ Enter
└──────────────────────────────────────────────────────────┘
         │
         ▼  inTrackView = true (immediate)
         ▼  scheduleAlbumDebounce(150ms, intent{albumID:"KOB"})
         │
         ▼  150ms later (if no Esc/re-Enter in that window):
         ▼  FetchAlbumTracksRequestMsg{AlbumID:"KOB", Offset:0}
         │
         ▼  app.go → buildFetchAlbumTracksCmd(ctx, "KOB", 0)
         ▼  GET /albums/KOB/tracks?limit=50&offset=0
         │
         ▼  AlbumTracksLoadedMsg{AlbumID:"KOB", Offset:0, Tracks:[...], HasNext:false}
         │
┌─ Albums pane (track sub-view) ───────────────────────────┐
│  Albums ── Kind of Blue (5 tracks)              Esc back  │
│  #   Track                    Artist          Duration    │
│  1   So What                  Miles Davis     9:22        │
│  2   Freddie Freeloader        Miles Davis     9:46        │
│  3   Blue in Green             Miles Davis     5:37  ◄ Enter
│  4   All Blues                 Miles Davis     11:33       │
│  5   Flamenco Sketches         Miles Davis     9:26        │
└──────────────────────────────────────────────────────────┘
         │
         ▼  PlayContextMsg{
         │      ContextURI: "spotify:album:KindOfBlue",
         │      OffsetURI:  "spotify:track:BlueInGreen"
         │  }
         │
         ▼  Spotify plays "Blue in Green", queue fills:
               [All Blues, Flamenco Sketches]
```

---

## API: Get Album Tracks

### Endpoint

```
GET /v1/albums/{id}/tracks
```

No OAuth scope required (albums are public). Free tier works.

### Query parameters

| Parameter | Type   | Required | Notes                              |
|-----------|--------|----------|------------------------------------|
| `limit`   | int    | No       | Max items per page. Default 20, max 50 |
| `offset`  | int    | No       | Offset into the list. Default 0    |

Use `limit=50` (max) for efficiency. Start at `offset=0`, then paginate via `next`.

### Request example

```
GET https://api.spotify.com/v1/albums/1weenld61qoidwYuZ1GESA/tracks?limit=50&offset=0
Authorization: Bearer <token>
```

### Response (200 OK)

```json
{
  "href": "https://api.spotify.com/v1/albums/1weenld61qoidwYuZ1GESA/tracks?offset=0&limit=50",
  "items": [
    {
      "artists": [
        {
          "id": "0kbYTNQb4Pb1rPbbaF0pT4",
          "name": "Miles Davis",
          "type": "artist",
          "uri": "spotify:artist:0kbYTNQb4Pb1rPbbaF0pT4"
        }
      ],
      "disc_number": 1,
      "duration_ms": 562000,
      "explicit": false,
      "id": "0IzFhSqaHe8d0ANDq26RcS",
      "name": "So What",
      "track_number": 1,
      "type": "track",
      "uri": "spotify:track:0IzFhSqaHe8d0ANDq26RcS"
    }
  ],
  "limit": 50,
  "next": null,
  "offset": 0,
  "previous": null,
  "total": 5
}
```

**Important distinctions vs `/playlists/{id}/tracks`:**

| Aspect | `GET /albums/{id}/tracks` | `GET /playlists/{id}/tracks` |
|--------|---------------------------|------------------------------|
| Item wrapper | Items are `SimplifiedTrackObject` directly | Items are `PlaylistTrackObject{added_at, track: Track}` |
| Album field | **Not present** (we already know the album) | Present in full Track object |
| Max limit | **50** (not 100) | 100 |
| Track number | `track_number` field present | Not present |
| Disc number | `disc_number` field present | Not present |

**Pagination via `next` field:**
- `next: null` → no more pages, `hasNext = false`
- `next: "https://..."` → more pages available, `hasNext = true`

Use `next != ""` (after unmarshaling) to detect more pages — same pattern as playlists.

### Error responses

| Status | Meaning | Spotnik action |
|--------|---------|----------------|
| 400 Bad Request | Malformed album ID | Toast "Failed to load album tracks", stay in sub-view |
| 401 Unauthorized | Expired token | Token refresh (handled by gateway) |
| 403 Forbidden | Not available in market | Toast "Album not available" |
| 404 Not Found | Album ID not found | Toast "Album not found", Esc back to list |
| 429 Too Many Requests | Rate limited | Parse `Retry-After`, toast "Rate limited", retry after delay |

---

## Architecture: Three-Layer Protection

Rapid album selection (Enter→Esc→Enter on different albums) must not result in stale
data loading. Three layers work together:

```
Layer 1: Pane debounce (150ms)
  ↓  only the last intent within 150ms fires a request
Layer 2: App context cancellation
  ↓  new request cancels prior in-flight HTTP
Layer 3: Staleness key
  ↓  loaded msg discarded if albumID doesn't match current selection
```

This is identical to the pattern established in Story 106 for playlists.

---

## Full Message Flow Diagrams

### Flow 1: Normal — Enter on album, browse tracks, play one

```
AlbumsPane.handleListViewKey(Enter)
  ├── a.selectedID   = "ALB123"
  ├── a.selectedURI  = "spotify:album:ALB123"
  ├── a.selectedName = "Kind of Blue"
  ├── a.loadedTracks = nil                    ← clear prior data
  ├── a.trackOffset  = 0
  ├── a.tracksFetching = false
  ├── a.hasMoreTracks  = false
  ├── a.inTrackView  = true                   ← immediate state change
  ├── a.trackTable.SetFocused(true)
  └── scheduleAlbumDebounce(150ms, {albumID:"ALB123", offset:0})

    [150ms tick fires]
    albumDebounceMsg{intent:{albumID:"ALB123", offset:0}} arrives
      └── intent matches current a.albumIntent?
            YES → a.tracksFetching = true
                  emit FetchAlbumTracksRequestMsg{AlbumID:"ALB123", Offset:0}
            NO  → discard (stale)

app.go receives FetchAlbumTracksRequestMsg{AlbumID:"ALB123", Offset:0}
  ├── a.albumTracksCancel()                   ← cancel prior request (if any)
  ├── ctx, cancel = context.WithCancel(Background)
  ├── a.albumTracksCancel = cancel
  ├── a.albumTracksID = "ALB123"              ← staleness key
  └── return buildFetchAlbumTracksCmd(ctx, "ALB123", 0)

buildFetchAlbumTracksCmd closure runs:
  ├── if ctx.Err() != nil → return nil        ← cancelled before HTTP
  ├── tracks, hasNext, err =
  │     library.AlbumTracks(
  │       api.WithPriority(ctx, api.Interactive),  ← gateway Interactive
  │       "ALB123", limit:50, offset:0
  │     )
  ├── if ctx.Err() != nil → return nil        ← cancelled mid-flight
  └── return AlbumTracksLoadedMsg{AlbumID:"ALB123", Offset:0,
                                   Tracks:tracks, HasNext:hasNext, Err:err}

app.go receives AlbumTracksLoadedMsg:
  ├── if m.AlbumID != a.albumTracksID → discard (staleness check)
  ├── if m.Err != nil:
  │     a.alerts.NewAlertCmd(toast.Warning, "Failed to load album tracks")
  │     forward to AlbumsPane (to clear tracksFetching)
  └── else: forward to AlbumsPane

AlbumsPane receives AlbumTracksLoadedMsg:
  ├── if m.AlbumID != a.selectedID → discard (double guard)
  ├── if m.Offset == 0:
  │     a.loadedTracks = m.Tracks             ← replace
  ├── if m.Offset > 0:
  │     a.loadedTracks = append(a.loadedTracks, m.Tracks...)  ← append
  ├── a.tracksFetching = false
  ├── a.trackOffset    = len(a.loadedTracks)
  ├── a.hasMoreTracks  = m.HasNext
  └── refreshTrackRows()  ← reads from a.loadedTracks (NOT store)

  [User navigates with j/k, presses Enter on track 3]
AlbumsPane.handleTrackViewKey(Enter)
  ├── idx = a.trackTable.SelectedIndex()       = 2 (0-based)
  ├── track = a.loadedTracks[2]
  └── emit PlayContextMsg{
            ContextURI: a.selectedURI,          ← "spotify:album:ALB123"
            OffsetURI:  track.URI               ← "spotify:track:BlueInGreen"
        }
```

### Flow 2: Esc from track sub-view → back to album list

```
AlbumsPane.handleTrackViewKey(Esc)
  ├── a.inTrackView    = false
  ├── a.loadedTracks   = nil                   ← free pane memory
  ├── a.trackOffset    = 0
  ├── a.hasMoreTracks  = false
  ├── a.tracksFetching = false
  ├── a.trackTable.SetFocused(false)
  └── emit AlbumTrackViewClosedMsg{}

app.go receives AlbumTrackViewClosedMsg:
  ├── a.albumTracksCancel()                    ← cancel in-flight if any
  └── a.albumTracksID = ""                     ← clear staleness key
```

### Flow 3: Rapid album switching (Enter→Esc→Enter)

```
User: Enter on "Kind of Blue"
  → a.albumIntent = {albumID:"KOB", offset:0}
  → scheduleAlbumDebounce(150ms, {albumID:"KOB"})

User (80ms later): Esc
  → a.inTrackView = false
  → emit AlbumTrackViewClosedMsg
  → app.go: albumTracksCancel()
  → app.go: albumTracksID = ""

User (30ms later): Enter on "Rumours"
  → a.albumIntent = {albumID:"RUM", offset:0}
  → scheduleAlbumDebounce(150ms, {albumID:"RUM"})

[150ms tick fires for "KOB"]
  → albumDebounceMsg{intent:{albumID:"KOB"}}
  → a.albumIntent.albumID == "RUM" (not "KOB") → DISCARD

[150ms tick fires for "RUM"]
  → albumDebounceMsg{intent:{albumID:"RUM"}}
  → a.albumIntent.albumID == "RUM" → MATCH → fire request
```

### Flow 4: Same album double-Enter (Enter while already in track view)

```
User presses Enter on "Kind of Blue" (already in track view)
  → handleTrackViewKey receives Enter
  → Enter in track sub-view plays the selected track → PlayContextMsg
  → NOT re-entering list view logic
  → NOT re-fetching tracks
```

This is not a debounce concern — in track view, Enter plays a track (not re-opens the album).

### Flow 5: Lazy pagination — cursor approaching end

```
User navigates down, cursor at row 45 (of 50 loaded tracks)
  → handleTrackViewKey(j) → trackTable.Update → checkPrefetch()
  → condition: cursor(45) >= len(loadedTracks)(50) - 5
               AND hasMoreTracks == true
               AND tracksFetching == false
  → a.tracksFetching = true
  → emit FetchAlbumTracksRequestMsg{AlbumID:"ALB123", Offset:50}

app.go receives FetchAlbumTracksRequestMsg{AlbumID:"ALB123", Offset:50}
  → a.albumTracksID is still "ALB123"
  → same context still valid (not cancelled)
  → return buildFetchAlbumTracksCmd(ctx, "ALB123", 50)

[Response: 12 more tracks, next=null]
AlbumTracksLoadedMsg{AlbumID:"ALB123", Offset:50, Tracks:[12 tracks], HasNext:false}
  → a.loadedTracks = append(loadedTracks[0:50], tracks[12]...)   ← 62 total
  → a.tracksFetching = false
  → a.trackOffset    = 62
  → a.hasMoreTracks  = false                  ← no more pages
  → refreshTrackRows()  ← table now shows 62 rows
```

Note: The trigger threshold is 5 rows (not 10 as in playlists) because album pages are
max 50 items (not 100). Cursor within 5 of the end is proportionally similar.

### Flow 6: Network error loading tracks

```
AlbumTracksLoadedMsg{AlbumID:"ALB123", Err: "connection refused"}
  → app.go: staleness check passes (albumID matches)
  → app.go: m.Err != nil
      → a.alerts.NewAlertCmd(toast.Warning, "Failed to load album tracks")
      → forward msg to AlbumsPane (to clear tracksFetching)

AlbumsPane receives:
  → a.tracksFetching = false
  → a.loadedTracks stays nil (or previously loaded data if pagination error)
  → inTrackView stays true
  → trackTable shows empty / previous rows
  → User can press Esc to go back
```

### Flow 7: Empty album (0 tracks — rare but possible)

```
GET /albums/{id}/tracks?limit=50&offset=0
→ { "items": [], "total": 0, "next": null }

AlbumTracksLoadedMsg{Offset:0, Tracks:[], HasNext:false}
  → a.loadedTracks = []
  → a.hasMoreTracks = false
  → refreshTrackRows() → trackTable has 0 rows

Track sub-view renders:
  Albums ── Untitled Album (0 tracks)          Esc back
  #   Track   Artist   Duration
  (empty)

Enter in track sub-view:
  → idx = -1 (no selection in empty table)
  → guard: if idx < 0 || idx >= len(a.loadedTracks) → do nothing
```

### Flow 8: Esc while tracks are loading

```
[FetchAlbumTracksRequestMsg fired, HTTP request in-flight]

User: Esc
  → handleTrackViewKey(Esc)
  → a.inTrackView    = false
  → a.loadedTracks   = nil
  → a.tracksFetching = false
  → emit AlbumTrackViewClosedMsg{}

app.go receives AlbumTrackViewClosedMsg:
  → a.albumTracksCancel()          ← context cancelled → HTTP aborted
  → a.albumTracksID = ""

[AlbumTracksLoadedMsg arrives later (before context cancellation took effect)]
  → app.go: m.AlbumID != a.albumTracksID ("" != "ALB123") → DISCARD
```

### Flow 9: Filter active in list view → Enter on album

```
User presses f → filter active, typing "miles"
AlbumsPane shows filtered rows (Miles Davis albums only)

User presses Enter on filtered row 1 ("Kind of Blue")
  → handleListViewKey detects Enter while filter is active?
    NO — when filter is active, all keys (including Enter) go to filter.Update()
    User must press Esc to close filter first, THEN Enter to open track view.

  → filter.IsActive() == true → all keys forwarded to filter.Update()
  → Enter in filter confirms filter text (no album open)
```

This is consistent with the existing FilteredAlbums() pattern — filter and track
sub-view are mutually exclusive interactions.

### Flow 10: Album re-selection after Esc (re-enter same album)

```
User: Enter on "Kind of Blue" → tracks load (50 tracks)
User: Esc → back to list, a.loadedTracks = nil
User: Enter on "Kind of Blue" again
  → a.selectedID = "KOB" (same)
  → a.loadedTracks = nil (cleared on Esc)
  → a.albumIntent = {albumID:"KOB", offset:0}
  → scheduleAlbumDebounce → 150ms → FetchAlbumTracksRequestMsg

  [Fresh fetch — data was freed on Esc, must re-fetch]
```

There is no cache — by design. Pane-owned ephemeral data is cleared on Esc.

---

## Edge Cases Table

| Scenario | What happens | Why |
|----------|-------------|-----|
| Enter on album → Esc within 150ms | Debounce tick discarded (intent cleared by Esc handler), no HTTP fired | Layer 1 protection |
| Rapid Enter on 3 different albums | Only last album's tick fires; first two discarded | Layer 1 (debounce stale check) |
| In-flight request when new album selected | Prior HTTP cancelled via context.WithCancel | Layer 2 protection |
| Stale AlbumTracksLoadedMsg arrives | albumID != albumTracksID → discarded in app.go | Layer 3 protection |
| Same message passes app.go but pane changed | albumID != a.selectedID → discarded in pane | Double guard |
| Empty album (0 tracks) | Sub-view opens, table empty, Enter does nothing | idx < 0 guard |
| Album with exactly 50 tracks | hasMoreTracks = true (last page full), triggers prefetch at offset 50 | next != null check |
| Prefetch at offset 50 returns 0 tracks | hasMoreTracks = false, loadedTracks unchanged | len(tracks)==0 check |
| Network error on first page | Toast warning, tracksFetching cleared, inTrackView stays open | Error propagation |
| Network error on pagination | Toast warning, previous tracks still visible, hasMoreTracks=false | Partial load preserved |
| 403 album not in market | Toast "Album not available", user can Esc | 403 handled in command |
| 404 album not found | Toast "Album not found", user can Esc | 404 handled in command |
| 429 rate limited | Toast + Retry-After seconds backoff | Standard error handling |
| Filter active when Enter pressed | Enter goes to filter (not album open) | filter.IsActive() guard |
| Enter in track view (not list view) | Plays selected track via PlayContextMsg | inTrackView gate |
| j/k in track view when 0 tracks | Table handles gracefully (no-op on empty) | Table component guard |

---

## New Types and Messages

### `internal/ui/panes/messages.go`

Add after the playlist messages:

```go
// FetchAlbumTracksRequestMsg is emitted by AlbumsPane when the user opens an album's
// track sub-view. Offset > 0 is used for lazy pagination (triggered by cursor proximity).
type FetchAlbumTracksRequestMsg struct {
    // AlbumID is the Spotify album ID whose tracks are being requested.
    AlbumID string
    // Offset is the 0-based index to start fetching from. 0 = first page.
    Offset int
}

// AlbumTracksLoadedMsg is returned by buildFetchAlbumTracksCmd after the API call.
// AlbumsPane owns the tracks — they are NOT written to the Store.
type AlbumTracksLoadedMsg struct {
    // AlbumID identifies which album's tracks arrived (used for staleness check).
    AlbumID string
    // Offset is the page offset this response corresponds to.
    Offset int
    // Tracks is the loaded slice; nil on error.
    Tracks []domain.Track
    // HasNext is true when the API response had a non-empty "next" URL — more pages exist.
    HasNext bool
    // Err is non-nil if the API call failed.
    Err error
}

// AlbumTrackViewClosedMsg is emitted by AlbumsPane when the user presses Esc to close
// the track sub-view. app.go cancels the in-flight context and clears the staleness key.
type AlbumTrackViewClosedMsg struct{}
```

### `internal/api/library_interfaces.go`

Add to `LibraryClient` interface:

```go
// AlbumTracks fetches a page of tracks for the given album.
// Returns the tracks slice, a hasNext bool (true if more pages exist), and any error.
AlbumTracks(ctx context.Context, albumID string, limit, offset int) ([]domain.Track, bool, error)
```

### New app.go fields

```go
// albumTracksCancel cancels any in-flight album tracks fetch.
// Initialized to a no-op to avoid nil checks.
albumTracksCancel context.CancelFunc

// albumTracksID is the album ID whose tracks are currently being fetched.
// Used as a staleness key to discard out-of-order responses.
albumTracksID string
```

Initialize in `NewApp` (same place `searchCancel` is initialized):
```go
a.albumTracksCancel = func() {}
```

### `internal/ui/panes/albums_pane.go` — new pane fields

```go
// Track sub-view state — pane-owned, not in store.
inTrackView    bool
selectedID     string   // Spotify album ID (e.g. "1weenld61qoidwYuZ1GESA")
selectedURI    string   // Spotify album URI (e.g. "spotify:album:1weenld61qoidwYuZ1GESA")
selectedName   string   // Album display name for sub-view title
loadedTracks   []domain.Track // pane-owned track list; nil when not in sub-view
trackOffset    int       // count of tracks loaded so far (for next page offset)
hasMoreTracks  bool      // true when last API page returned HasNext=true
tracksFetching bool      // true when a fetch is in flight; blocks duplicate prefetch

// trackTable renders the track list in sub-view.
trackTable *components.Table

// albumIntent is the debounce snapshot set on Enter; compared in albumDebounceMsg handler.
albumIntent albumDebounceIntent
```

Add private types in `albums_pane.go` (not exported, pane-internal):

```go
// albumDebounceIntent captures the album selection intent at the moment of Enter.
// Compared against current intent when the debounce tick fires.
type albumDebounceIntent struct {
    albumID string
    offset  int
}

// albumDebounceMsg is the tick message returned after the 150ms debounce window.
type albumDebounceMsg struct {
    intent albumDebounceIntent
}
```

---

## New API Method: `LibraryClient.AlbumTracks`

### `internal/api/library.go`

```go
// AlbumTracks fetches a page of tracks for the given album ID via
// GET /albums/{id}/tracks. Returns the tracks, a hasNext bool (true when the
// API's "next" field is non-empty, indicating more pages), and any error.
// The caller controls pagination via limit and offset.
func (l *LibraryClient) AlbumTracks(ctx context.Context, albumID string, limit, offset int) ([]domain.Track, bool, error) {
    path := fmt.Sprintf("/albums/%s/tracks?limit=%d&offset=%d", albumID, limit, offset)
    var resp struct {
        Items []struct {
            ID         string          `json:"id"`
            URI        string          `json:"uri"`
            Name       string          `json:"name"`
            DurationMs int             `json:"duration_ms"`
            Explicit   bool            `json:"explicit"`
            Artists    []domain.Artist `json:"artists"`
        } `json:"items"`
        Next string `json:"next"`
    }
    if err := l.base.doJSON(ctx, http.MethodGet, path, nil, &resp); err != nil {
        return nil, false, fmt.Errorf("fetching album tracks: %w", err)
    }
    tracks := make([]domain.Track, len(resp.Items))
    for i, item := range resp.Items {
        tracks[i] = domain.Track{
            ID:         item.ID,
            URI:        item.URI,
            Name:       item.Name,
            DurationMs: item.DurationMs,
            Explicit:   item.Explicit,
            Artists:    item.Artists,
            // Album field intentionally empty — caller knows the album from context.
        }
    }
    return tracks, resp.Next != "", nil
}
```

**Note:** The album tracks endpoint returns `SimplifiedTrackObject`, not full `Track`
objects. The key difference is:
- No `album` field in the response (the caller already has the album)
- No `popularity` or `external_ids`
- `track_number` and `disc_number` are present but not stored (not needed for display)

We use an inline struct for deserialization to avoid adding a new domain type just for
this response shape. The result is mapped to the existing `domain.Track`.

---

## New App Command: `buildFetchAlbumTracksCmd`

### `internal/app/commands.go`

```go
// buildFetchAlbumTracksCmd fetches a page of tracks for the given album ID.
// Offset 0 = first page (replace); Offset > 0 = subsequent page (append).
// The context is passed in from the caller to support cancellation when the user
// switches albums or presses Esc. api.Interactive priority bypasses the token bucket.
func (a *App) buildFetchAlbumTracksCmd(ctx context.Context, albumID string, offset int) tea.Cmd {
    library := a.library
    return func() tea.Msg {
        if library == nil {
            return panes.AlbumTracksLoadedMsg{Err: errNilClient, AlbumID: albumID}
        }
        if ctx.Err() != nil {
            return nil
        }
        tracks, hasNext, err := library.AlbumTracks(
            api.WithPriority(ctx, api.Interactive),
            albumID, 50, offset,
        )
        if ctx.Err() != nil {
            return nil
        }
        if err != nil {
            if secs := parse429RetryAfter(err); secs > 0 {
                return panes.RateLimitedMsg{RetryAfterSecs: secs}
            }
            if isUnauthorizedError(err) {
                return unauthorizedMsg{}
            }
        }
        return panes.AlbumTracksLoadedMsg{
            AlbumID: albumID,
            Offset:  offset,
            Tracks:  tracks,
            HasNext: hasNext,
            Err:     err,
        }
    }
}
```

---

## App.go Handler Changes

### `internal/app/app.go`

**Add fields to App struct:**
```go
albumTracksCancel context.CancelFunc
albumTracksID     string
```

**Initialize in NewApp (alongside searchCancel):**
```go
albumTracksCancel: func() {},
```

**Add three new cases in the Update switch:**

```go
case panes.FetchAlbumTracksRequestMsg:
    a.albumTracksCancel()
    ctx, cancel := context.WithCancel(context.Background())
    a.albumTracksCancel = cancel
    a.albumTracksID = m.AlbumID
    return a, a.buildFetchAlbumTracksCmd(ctx, m.AlbumID, m.Offset)

case panes.AlbumTracksLoadedMsg:
    if m.AlbumID != a.albumTracksID {
        return a, nil // stale response — discarded
    }
    if m.Err != nil {
        return a, tea.Batch(
            a.alerts.NewAlertCmd(toast.Warning, "Failed to load album tracks"),
            a.forwardToPane(layout.PaneAlbums, m),
        )
    }
    return a, a.forwardToPane(layout.PaneAlbums, m)

case panes.AlbumTrackViewClosedMsg:
    a.albumTracksCancel()
    a.albumTracksID = ""
    return a, nil
```

---

## AlbumsPane Refactor

### `internal/ui/panes/albums_pane.go`

The pane needs significant additions. The existing structure (album list table, filter)
is preserved. A second table (`trackTable`) and sub-view state are added.

#### Track table columns

```
# (5%) | Name (50%) | Artist (30%) | Duration (15%)
Flex factors: 1 : 10 : 6 : 3
```

Same proportions as playlist track sub-view for visual consistency.

#### `NewAlbumsPane` additions

```go
// Track sub-view table — same column shape as playlists track table.
trackCols := []components.ColumnDef{
    {Key: "index",    Header: "#",        FlexFactor: 1,  Color: th.ColumnIndex()},
    {Key: "name",     Header: "Track",    FlexFactor: 10, Color: th.ColumnPrimary()},
    {Key: "artist",   Header: "Artist",   FlexFactor: 6,  Color: th.ColumnSecondary()},
    {Key: "duration", Header: "Duration", FlexFactor: 3,  Color: th.ColumnTertiary()},
}
tt := components.NewTable(components.TableConfig{
    Columns:      trackCols,
    Theme:        th,
    PlayingIndex: -1,
    ShowHeader:   true,
})
```

#### `Actions()` method change

In track sub-view, return different actions:
```go
func (a *AlbumsPane) Actions() []layout.Action {
    if a.inTrackView {
        return []layout.Action{{Key: "Esc", Label: "back"}}
    }
    if a.filter.IsActive() {
        return []layout.Action{{Key: "Esc", Label: "close"}}
    }
    return []layout.Action{{Key: "f", Label: "filter"}}
}
```

#### `Title()` method change

In track sub-view, show album name and track count:
```go
func (a *AlbumsPane) Title() string {
    if a.inTrackView {
        count := len(a.loadedTracks)
        return fmt.Sprintf("Albums ── %s (%d tracks)", a.selectedName, count)
    }
    return "Albums"
}
```

#### `Update()` changes

Add `AlbumTracksLoadedMsg` handler (regardless of focus):
```go
case AlbumTracksLoadedMsg:
    if m.AlbumID != a.selectedID {
        return a, nil
    }
    a.tracksFetching = false
    if m.Err != nil {
        return a, nil
    }
    if m.Offset == 0 {
        a.loadedTracks = m.Tracks
    } else {
        a.loadedTracks = append(a.loadedTracks, m.Tracks...)
    }
    a.trackOffset   = len(a.loadedTracks)
    a.hasMoreTracks = m.HasNext
    a.refreshTrackRows()
    return a, nil

case albumDebounceMsg:
    if m.intent != a.albumIntent {
        return a, nil // stale tick
    }
    a.tracksFetching = true
    return a, func() tea.Msg {
        return FetchAlbumTracksRequestMsg{AlbumID: m.intent.albumID, Offset: m.intent.offset}
    }
```

Route key events through view state:
```go
if a.inTrackView {
    return a.handleTrackViewKey(keyMsg)
}
return a.handleListViewKey(keyMsg)
```

#### `handleListViewKey` changes

Replace current Enter handler:
```go
case keyMsg.Type == tea.KeyEnter:
    albums := a.filteredAlbums()
    idx := a.table.SelectedIndex()
    if idx < 0 || idx >= len(albums) {
        return a, nil
    }
    alb := albums[idx].Album
    a.selectedID   = alb.ID
    a.selectedURI  = alb.URI
    a.selectedName = alb.Name
    a.loadedTracks   = nil
    a.trackOffset    = 0
    a.hasMoreTracks  = false
    a.tracksFetching = false
    a.inTrackView    = true
    a.table.SetFocused(false)
    a.trackTable.SetFocused(true)
    intent := albumDebounceIntent{albumID: alb.ID, offset: 0}
    a.albumIntent = intent
    return a.scheduleAlbumDebounce(intent)
```

#### `handleTrackViewKey` (new method)

```go
func (a *AlbumsPane) handleTrackViewKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch {
    case key.Type == tea.KeyEsc:
        a.inTrackView    = false
        a.loadedTracks   = nil
        a.trackOffset    = 0
        a.hasMoreTracks  = false
        a.tracksFetching = false
        a.trackTable.SetFocused(false)
        a.table.SetFocused(true)
        return a, func() tea.Msg { return AlbumTrackViewClosedMsg{} }

    case key.Type == tea.KeyEnter:
        idx := a.trackTable.SelectedIndex()
        if idx < 0 || idx >= len(a.loadedTracks) {
            return a, nil
        }
        track := a.loadedTracks[idx]
        albumURI := a.selectedURI
        return a, func() tea.Msg {
            return PlayContextMsg{
                ContextURI: albumURI,
                OffsetURI:  track.URI,
            }
        }
    }

    cmd := a.trackTable.Update(key)
    a.checkPrefetch()
    return a, cmd
}
```

#### `checkPrefetch` (new method)

Trigger threshold: 5 rows from end (proportional to page size of 50).

```go
// checkPrefetch fires a lazy pagination request when the cursor is within 5 rows
// of the end of loaded tracks, there are more pages, and no fetch is in-flight.
func (a *AlbumsPane) checkPrefetch() {
    if !a.hasMoreTracks || a.tracksFetching {
        return
    }
    cursor := a.trackTable.SelectedIndex()
    if cursor < len(a.loadedTracks)-5 {
        return
    }
    a.tracksFetching = true
    intent := albumDebounceIntent{albumID: a.selectedID, offset: a.trackOffset}
    // Pagination skips the 150ms debounce — it is a cursor-triggered event, not user
    // typing. Fire immediately.
    _ = FetchAlbumTracksRequestMsg{AlbumID: a.selectedID, Offset: a.trackOffset}
    // Emit via command so app.go handles it through the normal message path.
    a.albumIntent = intent // keep intent in sync so debounce ticks are not accidentally stale
}
```

Wait — pagination requests should be emitted as commands, not debounced. The debounce
is only for the initial Enter on a new album. Pagination from cursor proximity is a
single discrete event — no debounce needed.

Correct `checkPrefetch`:
```go
func (a *AlbumsPane) checkPrefetch() tea.Cmd {
    if !a.hasMoreTracks || a.tracksFetching {
        return nil
    }
    cursor := a.trackTable.SelectedIndex()
    if cursor < len(a.loadedTracks)-5 {
        return nil
    }
    a.tracksFetching = true
    offset := a.trackOffset
    albumID := a.selectedID
    return func() tea.Msg {
        return FetchAlbumTracksRequestMsg{AlbumID: albumID, Offset: offset}
    }
}
```

And in `handleTrackViewKey`, collect both commands:
```go
cmd := a.trackTable.Update(key)
prefetchCmd := a.checkPrefetch()
return a, tea.Batch(cmd, prefetchCmd)
```

#### `scheduleAlbumDebounce` (new method)

```go
func (a *AlbumsPane) scheduleAlbumDebounce(intent albumDebounceIntent) (tea.Model, tea.Cmd) {
    return a, tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg {
        return albumDebounceMsg{intent: intent}
    })
}
```

#### `refreshTrackRows` (new method)

```go
func (a *AlbumsPane) refreshTrackRows() {
    rows := make([]map[string]string, len(a.loadedTracks))
    for i, tr := range a.loadedTracks {
        artistName := ""
        if len(tr.Artists) > 0 {
            artistName = tr.Artists[0].Name
        }
        rows[i] = map[string]string{
            "index":    fmt.Sprintf("%d", i+1),
            "name":     tr.Name,
            "artist":   artistName,
            "duration": components.FormatDuration(tr.DurationMs),
        }
    }
    a.trackTable.SetRows(rows)
}
```

#### `View()` changes

```go
func (a *AlbumsPane) View() string {
    if a.inTrackView {
        return a.trackTable.View()
    }
    var parts []string
    if a.filter.IsActive() {
        parts = append(parts, a.filter.View(a.width))
    }
    parts = append(parts, a.table.View())
    return strings.Join(parts, "\n")
}
```

#### `SetSize()` changes

Propagate size to both tables:
```go
func (a *AlbumsPane) SetSize(width, height int) {
    a.width = width
    a.height = height
    a.filter.SetWidth(width)
    a.trackTable.SetSize(width, height)
    a.resizeTable()
}
```

#### `SetFocused()` changes

```go
func (a *AlbumsPane) SetFocused(focused bool) {
    a.focused = focused
    if a.inTrackView {
        a.trackTable.SetFocused(focused)
        a.table.SetFocused(false)
    } else {
        a.table.SetFocused(focused && !a.filter.IsActive())
        a.trackTable.SetFocused(false)
    }
}
```

#### `SetTheme()` changes

Rebuild `trackTable` alongside `table` when theme changes:
```go
// Rebuild track table with new column colors.
trackCols := []components.ColumnDef{
    {Key: "index",    Header: "#",        FlexFactor: 1,  Color: th.ColumnIndex()},
    {Key: "name",     Header: "Track",    FlexFactor: 10, Color: th.ColumnPrimary()},
    {Key: "artist",   Header: "Artist",   FlexFactor: 6,  Color: th.ColumnSecondary()},
    {Key: "duration", Header: "Duration", FlexFactor: 3,  Color: th.ColumnTertiary()},
}
a.trackTable = components.NewTable(components.TableConfig{
    Columns:      trackCols,
    Theme:        th,
    PlayingIndex: -1,
    ShowHeader:   true,
})
a.trackTable.SetSize(a.width, a.height)
if a.inTrackView {
    a.trackTable.SetFocused(a.focused)
    a.refreshTrackRows()
}
```

---

## Files

| File | Change |
|------|--------|
| `internal/api/library.go` | Add `AlbumTracks` method (GET /albums/{id}/tracks) |
| `internal/api/library_interfaces.go` | Add `AlbumTracks` to `LibraryClient` interface |
| `internal/ui/panes/messages.go` | Add `FetchAlbumTracksRequestMsg`, `AlbumTracksLoadedMsg`, `AlbumTrackViewClosedMsg` |
| `internal/app/app.go` | Add `albumTracksCancel`, `albumTracksID` fields; 3 new message handlers |
| `internal/app/commands.go` | Add `buildFetchAlbumTracksCmd` |
| `internal/ui/panes/albums_pane.go` | Refactor: add track sub-view, debounce, lazy pagination, `handleTrackViewKey`, `checkPrefetch`, `refreshTrackRows` |

---

## Acceptance Criteria

- [ ] Pressing Enter on an album opens a track sub-view showing the album's tracks
- [ ] Track sub-view shows: `#`, `Track`, `Artist`, `Duration` columns
- [ ] Pane title shows `"Albums ── {album name} ({n} tracks)"` in track sub-view
- [ ] Pressing Esc from track sub-view returns to the album list
- [ ] Pressing Enter on a track emits `PlayContextMsg{ContextURI: albumURI, OffsetURI: trackURI}`
- [ ] `PlayContextMsg` with `OffsetURI` causes Spotify to play that track and queue the rest of the album
- [ ] Rapid album switching (Enter→Esc→Enter on different album) results in only the last album loading
- [ ] In-flight requests are cancelled when user switches album or presses Esc
- [ ] Stale `AlbumTracksLoadedMsg` responses are discarded by staleness key check
- [ ] Lazy pagination: scrolling within 5 rows of end triggers next page fetch
- [ ] Lazy pagination: subsequent page tracks are appended to existing list (not replaced)
- [ ] Lazy pagination: `hasMoreTracks = false` stops further prefetch requests
- [ ] Album with exactly 50 tracks correctly fetches page 2 (which returns 0 tracks), then stops
- [ ] Empty album (0 tracks) renders empty track table; Enter does nothing
- [ ] Network error during track load shows toast and keeps sub-view open
- [ ] Filter key `f` and filter state are preserved in album list view (unchanged behavior)
- [ ] Filter cannot be opened while in track sub-view
- [ ] Theme switching rebuilds both tables with correct colors
- [ ] `AlbumTracks` method in `LibraryClient` correctly maps `SimplifiedTrackObject` to `domain.Track`
- [ ] `AlbumTracks` correctly returns `hasNext = true` when API `next != ""`
- [ ] No Store reads or writes for track data — all album track data lives in the pane
- [ ] Existing album list browsing (j/k navigation, filter) is unchanged

---

## Tasks

- [ ] Add `AlbumTracks(ctx, albumID, limit, offset) ([]domain.Track, bool, error)` to `LibraryClient`
      in `internal/api/library.go`. Inline struct for response deserialization — do NOT add new
      domain type for SimplifiedTrackObject. Map to `domain.Track` (Album field empty by design).
      - test: `httptest.NewServer` returning fixture with 2 tracks and `next: null` → returns
        `hasNext=false`; fixture with `next: "https://..."` → returns `hasNext=true`
      - test: HTTP 404 → error returned, not panic

- [ ] Add `AlbumTracks` to `LibraryClient` interface in `internal/api/library_interfaces.go`

- [ ] Add `FetchAlbumTracksRequestMsg`, `AlbumTracksLoadedMsg`, `AlbumTrackViewClosedMsg`
      to `internal/ui/panes/messages.go`

- [ ] Add `albumTracksCancel context.CancelFunc` and `albumTracksID string` to App struct;
      initialize `albumTracksCancel = func() {}` in `NewApp` alongside `searchCancel`.
      Add three handlers in `app.go` Update switch:
        - `FetchAlbumTracksRequestMsg`: cancel prior, create new ctx, set staleness key, build cmd
        - `AlbumTracksLoadedMsg`: staleness check, toast on error, forward to pane
        - `AlbumTrackViewClosedMsg`: cancel + clear staleness key
      - test: `FetchAlbumTracksRequestMsg` handler cancels prior cancel and sets new `albumTracksID`
      - test: `AlbumTracksLoadedMsg` with wrong `AlbumID` returns nil (not forwarded)
      - test: `AlbumTracksLoadedMsg` with `Err != nil` returns alert cmd + forward to pane
      - test: `AlbumTrackViewClosedMsg` clears `albumTracksID`

- [ ] Add `buildFetchAlbumTracksCmd(ctx, albumID, offset)` to `internal/app/commands.go`.
      Use `api.Interactive` priority. Return `nil` if ctx cancelled before or after HTTP.
      Handle 429 → `RateLimitedMsg`, 401 → `unauthorizedMsg`, other errors → `AlbumTracksLoadedMsg{Err}`.
      - test: nil library client → `AlbumTracksLoadedMsg{Err: errNilClient}`
      - test: cancelled ctx before call → returns nil
      - test: successful call → `AlbumTracksLoadedMsg{AlbumID, Offset, Tracks, HasNext:false}`
      - test: 429 response → `RateLimitedMsg`

- [ ] Refactor `internal/ui/panes/albums_pane.go`:
      a. Add `albumDebounceIntent` and `albumDebounceMsg` private types
      b. Add pane fields: `inTrackView`, `selectedID`, `selectedURI`, `selectedName`,
         `loadedTracks`, `trackOffset`, `hasMoreTracks`, `tracksFetching`, `trackTable`, `albumIntent`
      c. Add `trackTable` construction in `NewAlbumsPane`
      d. Update `Actions()` — return Esc hint in track sub-view
      e. Update `Title()` — show album name and track count in track sub-view
      f. Update `SetFocused()` — route focus to correct table based on `inTrackView`
      g. Update `SetSize()` — propagate to `trackTable`
      h. Update `Update()` — handle `AlbumTracksLoadedMsg`, `albumDebounceMsg`, route key events
      i. Replace Enter handler in `handleListViewKey` with debounce-based drill-down
      j. Add `handleTrackViewKey(key) (tea.Model, tea.Cmd)`
      k. Add `scheduleAlbumDebounce(intent) (tea.Model, tea.Cmd)`
      l. Add `checkPrefetch() tea.Cmd` (prefetch threshold: 5 rows from end)
      m. Add `refreshTrackRows()`
      n. Update `View()` — render `trackTable` when `inTrackView`
      o. Update `SetTheme()` — rebuild `trackTable`
      - test: Enter on album sets `inTrackView=true`, emits debounce tick
      - test: albumDebounceMsg with matching intent emits `FetchAlbumTracksRequestMsg`
      - test: albumDebounceMsg with stale intent discards
      - test: `AlbumTracksLoadedMsg` Offset=0 → replaces `loadedTracks`
      - test: `AlbumTracksLoadedMsg` Offset=50 → appends to `loadedTracks`
      - test: `AlbumTracksLoadedMsg` wrong AlbumID → no update
      - test: `AlbumTracksLoadedMsg` HasNext=false → `hasMoreTracks=false`, no prefetch
      - test: cursor within 5 of end, `hasMoreTracks=true` → `checkPrefetch` returns cmd
      - test: Esc emits `AlbumTrackViewClosedMsg`, clears `loadedTracks`
      - test: Enter on track row N emits `PlayContextMsg{ContextURI: albumURI, OffsetURI: loadedTracks[N].URI}`
      - test: Enter in empty track list (idx < 0) → no cmd
