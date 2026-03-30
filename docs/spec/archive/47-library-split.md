# Feature 47 — Library Split

> **Feature:** Split the monolithic `LibraryPane` into 3 independent panes
> (`PlaylistsPane`, `AlbumsPane`, `LikedSongsPane`) and merge `PlaylistManager`
> functionality into `PlaylistsPane`. Each pane implements `layout.Pane` with
> dense table format and filtering.

## Context

The current `LibraryPane` (`internal/ui/panes/library.go`, ~18.5KB) is a collapsible
tree with 4 sections: Playlists, Albums, Liked Songs, Recently Played. It handles all
4 data domains in a single pane with section expand/collapse logic.

The `PlaylistManager` (`internal/ui/panes/playlists.go`, ~24.2KB) is a separate full-screen
view with a dual-pane layout (playlist list + tracks) supporting create, rename, delete,
and reorder operations.

The new DESIGN.md (§2, §23) specifies splitting these into 3 independent panes:
1. **PlaylistsPane** — playlist list + track sub-view + create/rename/delete/reorder (merges PlaylistManager)
2. **AlbumsPane** — album list with artist and year
3. **LikedSongsPane** — liked tracks with like/unlike toggle

RecentlyPlayed moves to Feature 48 (Stats Split) as it was originally part of StatsView.

**Design reference:** `docs/DESIGN.md` §2 (Pane Definitions — Playlists/Albums/LikedSongs),
§9 (Dense Table — column widths per pane), §23 (Migration — LibraryPane split, PlaylistManager merge)

**Depends on:** Feature 41 (Pane interface), Feature 43 (Table + Filter components)

---

## Design Diagram

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

---

## Task 1: Create PlaylistsPane

**Problem:** Playlist functionality is split between LibraryPane (list) and PlaylistManager (management).

**Fix:**

Create `internal/ui/panes/playlists_pane.go` (new file, distinct from old `playlists.go`):

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

**Pane interface:**
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

**Key handling:**
- `Enter` → open track sub-view (emit `FetchPlaylistTracksRequestMsg`)
- `Esc` in track view → return to playlist list
- `n` → emit `PlaylistCreateRequestMsg`
- `r` → emit `PlaylistRenameRequestMsg`
- `x` → emit `PlaylistRemoveRequestMsg` (confirm? — follow existing PlaylistManager pattern)
- `Shift+↑/↓` → emit `PlaylistReorderRequestMsg`
- `f` → toggle filter
- `j/k` → scroll (forwarded to table)

**Data source:** `store.Playlists()` for playlist list, playlist tracks via message.

**Playlist list columns:** `# 5% | Name 70% | Tracks 25%`
**Track sub-view columns:** `# 5% | Track 45% | Artist 35% | Duration 15%`

**Files:**
- Create: `internal/ui/panes/playlists_pane.go`

**Tests:**
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

**Commit:** `feat(ui): create PlaylistsPane with management features`

---

## Task 2: Create AlbumsPane

**Problem:** Album browsing is buried in LibraryPane's tree sections.

**Fix:**

Create `internal/ui/panes/albums_pane.go`:

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

**Pane interface:**
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

**Key handling:**
- `Enter` → emit `PlayContextMsg` with album URI
- `f` → toggle filter
- `j/k` → scroll

**Data source:** `store.Albums()` for album list.

**Columns:** `# 5% | Name 50% | Artist 30% | Year 15%`

**Filter matches:** album name, artist name

**Files:**
- Create: `internal/ui/panes/albums_pane.go`

**Tests:**
- Unit: Interface satisfaction: `var _ layout.Pane = &AlbumsPane{}`
- Unit: Album list renders with correct columns
- Unit: Enter key → emits PlayContextMsg with album URI
- Unit: Filter filters by album name and artist
- Unit: Albums display year correctly
- Unit: Empty albums → clean empty state

**Commit:** `feat(ui): create AlbumsPane with dense table and filter`

---

## Task 3: Create LikedSongsPane

**Problem:** Liked songs browsing is buried in LibraryPane.

**Fix:**

Create `internal/ui/panes/likedsongs_pane.go`:

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

**Pane interface:**
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

**Key handling:**
- `Enter` → emit `PlayTrackMsg` with track URI
- `i` → emit `LikeTrackRequestMsg` (toggle like/unlike for selected track)
- `f` → toggle filter
- `j/k` → scroll

**Data source:** `store.LikedTracks()` for track list.

**Columns:** `# 5% | Track 45% | Artist 35% | Duration 15%`

**Filter matches:** track name, artist name

**Files:**
- Create: `internal/ui/panes/likedsongs_pane.go`

**Tests:**
- Unit: Interface satisfaction: `var _ layout.Pane = &LikedSongsPane{}`
- Unit: Track list renders with correct columns
- Unit: Enter key → emits PlayTrackMsg
- Unit: `i` key → emits LikeTrackRequestMsg
- Unit: Filter filters by track name and artist
- Unit: Duration formatted as M:SS

**Commit:** `feat(ui): create LikedSongsPane with like/unlike and filter`

---

## Task 4: Data loading integration

**Problem:** The new panes need to receive data that previously flowed through LibraryPane.

**Fix:**

Each pane handles the existing message types in its `Update()`:

| Pane | Message | Data |
|------|---------|------|
| PlaylistsPane | `LibraryLoadedMsg` (playlists data) | `store.Playlists()` |
| PlaylistsPane | `PlaylistTracksLoadedMsg` | Track list for selected playlist |
| PlaylistsPane | `PlaylistCreatedMsg`, `PlaylistRenamedMsg`, etc. | Mutation results |
| AlbumsPane | `AlbumsLoadedMsg` | `store.Albums()` |
| LikedSongsPane | `LikedTracksLoadedMsg` | `store.LikedTracks()` |

The panes read from Store on data-loaded messages and update their table rows.
No new message types needed — reuse existing ones from `messages.go`.

**Files:**
- Modify: `internal/ui/panes/playlists_pane.go`
- Modify: `internal/ui/panes/albums_pane.go`
- Modify: `internal/ui/panes/likedsongs_pane.go`

**Tests:**
- Unit: PlaylistsPane handles LibraryLoadedMsg → refreshes table
- Unit: AlbumsPane handles AlbumsLoadedMsg → refreshes table
- Unit: LikedSongsPane handles LikedTracksLoadedMsg → refreshes table
- Unit: PlaylistsPane handles PlaylistCreatedMsg → refreshes list
- Unit: PlaylistsPane handles PlaylistTracksLoadedMsg → shows tracks in sub-view

**Commit:** `feat(ui): wire data loading to split library panes`

---

## Task 5: Comprehensive tests

**Files:**
- Create: `internal/ui/panes/playlists_pane_test.go`
- Create: `internal/ui/panes/albums_pane_test.go`
- Create: `internal/ui/panes/likedsongs_pane_test.go`

**Tests:**
- Integration: PlaylistsPane — load playlists → select → Enter → track view → Esc → back to list
- Integration: PlaylistsPane — create playlist → list refreshes
- Integration: PlaylistsPane — rename playlist → list updates
- Integration: PlaylistsPane — reorder tracks with Shift+↑/↓
- Integration: AlbumsPane — load albums → filter → select → play
- Integration: LikedSongsPane — load tracks → like/unlike → filter
- Integration: All 3 panes handle resize correctly
- Integration: All 3 panes filter independently
- Edge: Large dataset (100+ items) → scrolling works
- Edge: Empty data → clean empty state per pane

**Commit:** `test(ui): comprehensive library split pane tests`

---

## Acceptance Criteria

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

---

## Notes

- **RecentlyPlayed** is NOT part of this feature. It moves to Feature 48 (Stats Split) since
  it was originally a section of StatsView and uses `store.RecentlyPlayed()`.
- The old `LibraryPane` and `PlaylistManager` files remain until Feature 49 (App Migration)
  rewires the app to use the new panes. At that point, the old files become dead code and
  are deleted in Feature 53 (Cleanup).
- PlaylistsPane's track sub-view is internal state — it doesn't change the page or layout.
  The pane renders either the playlist list or the track list based on `inTrackView` flag.
- Playlist mutations (create, rename, delete, reorder) emit request messages. The app's
  `Update()` dispatches the API commands. This flow is unchanged from the current architecture.
- The column flex factors are approximations of the percentage widths. bubble-table distributes
  remaining space after fixed columns, so flex factors 1:14:6:3 ≈ 5%/58%/25%/12%.
  Fine-tune during implementation to match the visual design.
