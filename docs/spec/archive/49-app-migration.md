# Feature 49 — App Migration

> **Feature:** Replace the current layout system (hardcoded 3-column + viewMode enum)
> with the `LayoutManager`-based grid. Wire preset cycling, page toggle, pane toggle,
> and focus rotation. This is the integration feature that connects all prior work.

## Context

The current `app.go` (~41KB) uses:
- `viewMode` enum: `viewMain`, `viewStats`, `viewPlaylists` (+ `viewSplash`, `viewAuth`)
- `focusedPane` enum: `focusPlayer`, `focusLibrary`, `focusQueue`
- Hardcoded 3-pane `JoinHorizontal` in `render.go`
- Individual pane fields: `playerPane`, `libraryPane`, `queuePane`, `statsPane`, `playlistPane`
- Fixed pane widths: 22%/50%/28%

The new design replaces all of this with:
- `layout.Manager` for grid computation, presets, page/pane toggling
- `panes map[PaneID]layout.Pane` for all 10 panes
- `renderGrid()` that assembles panes into the grid using Manager.PaneRect()
- `p` for preset cycling, `0` for page toggle, `1-8` for pane toggle
- `Tab/Shift+Tab` rotates focus through visible panes via Manager.RotateFocus()

**Design reference:** `docs/DESIGN.md` §3 (Layout Grid System), §4 (Pages, Presets, Toggling),
§16 (Focus & Navigation), §22 (Architecture — LayoutManager), §23 (Migration)

**Depends on:** Features 40-48 (all foundation + pane implementations)

---

## Design Diagram

```
Current Architecture:
  app.go
    ├── viewMode: viewMain | viewStats | viewPlaylists | viewSplash | viewAuth
    ├── focus: focusPlayer | focusLibrary | focusQueue
    ├── playerPane, libraryPane, queuePane (3 fixed panes)
    ├── statsPane, playlistPane (alternate views)
    └── render.go: JoinHorizontal(library, player, queue)

New Architecture:
  app.go
    ├── viewMode: viewSplash | viewAuth | viewGrid (replaces viewMain/viewStats/viewPlaylists)
    ├── layout: *layout.Manager (handles focus, presets, pages, toggle)
    ├── panes: map[PaneID]layout.Pane (10 panes)
    │   ├── PaneNowPlaying     (F45)
    │   ├── PaneQueue          (F46)
    │   ├── PanePlaylists      (F47)
    │   ├── PaneAlbums         (F47)
    │   ├── PaneLikedSongs     (F47)
    │   ├── PaneRecentlyPlayed (F48)
    │   ├── PaneTopTracks      (F48)
    │   ├── PaneTopArtists     (F48)
    │   ├── PaneRequestFlow    (F51 — placeholder/nil until then)
    │   └── PaneNetworkLog     (F51 — placeholder/nil until then)
    └── render.go: renderGrid() using layout.Manager.PaneRect()

Key Routing:
  '0'         → layout.TogglePage()
  'p'         → layout.CyclePreset()
  '1'-'8'     → layout.TogglePane(PaneID)
  Tab         → layout.RotateFocus(true)
  Shift+Tab   → layout.RotateFocus(false)
  Space,>,<.. → always route to NowPlaying (playback keys)
  'f'         → route to focused pane (filter)
  '/'         → open search overlay
  'd'         → open device overlay
```

---

## Task 1: Replace viewMode and focus enums

**Problem:** `viewMode` has 5 values; 3 of them (`viewMain`, `viewStats`, `viewPlaylists`)
are replaced by the page/preset system.

**Fix:**

1. Replace `viewMode` values:
   - Keep: `viewSplash`, `viewAuth`
   - Add: `viewGrid` (replaces `viewMain`, `viewStats`, `viewPlaylists`)
   - Delete: `viewMain`, `viewStats`, `viewPlaylists`

2. Remove `focusedPane` enum entirely — replaced by `layout.Manager.FocusedPane()`

3. Add `layout *layout.Manager` field to App struct

4. Add `panes map[layout.PaneID]layout.Pane` field to App struct

5. Remove individual pane fields: `playerPane` → accessed via `panes[PaneNowPlaying]`
   Keep `searchPane` and `devicePane` as separate overlay fields.

**Files:**
- Modify: `internal/app/app.go`

**Tests:**
- Unit: App starts with `viewSplash`, transitions to `viewGrid` (not `viewMain`)
- Unit: `layout.FocusedPane()` returns valid PaneID
- Unit: All 8 Page A panes registered in `panes` map

**Commit:** `refactor(app): replace viewMode/focusedPane with LayoutManager`

---

## Task 2: Implement renderGrid()

**Problem:** `buildView()` uses hardcoded 3-pane JoinHorizontal.

**Fix:**

Replace the body of `buildView()` for `viewGrid` mode:

```go
func (a *App) renderGrid() string {
    visiblePanes := a.layout.VisiblePanes()

    // Group panes by row (using Rect.Y to determine row membership)
    rows := groupPanesByRow(visiblePanes, a.layout)

    var rowStrings []string
    for _, row := range rows {
        var cellStrings []string
        for _, paneID := range row {
            rect := a.layout.PaneRect(paneID)
            pane := a.panes[paneID]

            // Get pane content (sized to content area)
            content := pane.View()

            // Wrap in btop-style border
            cfg := layout.BorderConfig{
                Width:       rect.Width,
                Height:      rect.Height,
                Title:       pane.Title(),
                ToggleKey:   pane.ToggleKey(),
                Actions:     pane.Actions(),
                AccentColor: paneBorderColor(paneID, a.theme),
                Focused:     pane.IsFocused(),
                Theme:       a.theme,
            }
            bordered := layout.RenderPaneBorder(content, cfg)

            // Ensure exact width (safety cap)
            capped := lipgloss.NewStyle().
                Width(rect.Width).MaxWidth(rect.Width).
                Height(rect.Height).MaxHeight(rect.Height).
                Render(bordered)
            cellStrings = append(cellStrings, capped)
        }
        rowStr := lipgloss.JoinHorizontal(lipgloss.Top, cellStrings...)
        rowStrings = append(rowStrings, rowStr)
    }

    return lipgloss.JoinVertical(lipgloss.Left, rowStrings...)
}
```

**Helper:**
```go
// paneBorderColor returns the accent color for a pane from the theme.
func paneBorderColor(id layout.PaneID, t theme.Theme) lipgloss.Color {
    switch id {
    case layout.PaneNowPlaying: return t.PaneBorderNowPlaying()
    case layout.PaneQueue: return t.PaneBorderQueue()
    // ... etc for all 10 panes
    }
}
```

**Files:**
- Modify: `internal/app/render.go`

**Tests:**
- Unit: `renderGrid()` produces output with correct total dimensions
- Unit: Each pane's border appears in the output
- Unit: Grid respects preset layout (Dashboard shows 8 panes in 3 rows)
- Unit: Hidden panes don't appear in output

**Commit:** `feat(app): renderGrid with LayoutManager-based pane assembly`

---

## Task 3: Wire key routing for pages, presets, and toggles

**Problem:** Keys `0`, `p`, `1-8` need new handlers.

**Fix:**

Update `routing.go` (or the key handler in app.go):

```go
case tea.KeyMsg:
    // Page toggle
    if msg.String() == "0" {
        a.layout.TogglePage()
        a.propagateSizes()
        a.syncFocus()
        return a, nil
    }

    // Preset cycling
    if msg.String() == "p" {
        a.layout.CyclePreset()
        a.propagateSizes()
        a.syncFocus()
        return a, nil
    }

    // Pane toggle (keys 1-8)
    if r := msg.Runes; len(r) == 1 && r[0] >= '1' && r[0] <= '8' {
        paneID := layout.PaneID(int(r[0]-'1'))  // '1' → 0, '2' → 1, etc.
        // But PaneID mapping: 1→NowPlaying, 2→Queue, etc.
        toggleMap := map[rune]layout.PaneID{
            '1': layout.PaneNowPlaying, '2': layout.PaneQueue,
            '3': layout.PanePlaylists, '4': layout.PaneAlbums,
            '5': layout.PaneLikedSongs, '6': layout.PaneRecentlyPlayed,
            '7': layout.PaneTopTracks, '8': layout.PaneTopArtists,
        }
        if id, ok := toggleMap[r[0]]; ok {
            a.layout.TogglePane(id)
            a.propagateSizes()
            a.syncFocus()
        }
        return a, nil
    }

    // Tab / Shift+Tab focus rotation
    if msg.String() == "tab" {
        a.layout.RotateFocus(true)
        a.syncFocus()
        return a, nil
    }
    if msg.String() == "shift+tab" {
        a.layout.RotateFocus(false)
        a.syncFocus()
        return a, nil
    }
```

**Helper methods:**
```go
// propagateSizes calls SetSize on all visible panes with their computed Rects.
func (a *App) propagateSizes()

// syncFocus calls SetFocused(true/false) on all panes based on layout.FocusedPane().
func (a *App) syncFocus()
```

**Files:**
- Modify: `internal/app/routing.go`
- Modify: `internal/app/app.go`

**Tests:**
- Unit: Key `0` toggles page A↔B
- Unit: Key `p` cycles presets (0→1→2→3→0)
- Unit: Key `1` toggles NowPlaying visibility
- Unit: Key `5` toggles LikedSongs visibility
- Unit: Tab moves focus forward through visible panes
- Unit: Shift+Tab moves focus backward
- Unit: After toggle, pane sizes update correctly
- Unit: Playback keys (Space, >, <, etc.) still route to NowPlaying regardless of focus

**Commit:** `feat(app): wire page toggle, preset cycling, pane toggle, focus rotation`

---

## Task 4: Wire WindowSizeMsg propagation

**Problem:** Terminal resize needs to propagate through LayoutManager to all panes.

**Fix:**

In the `WindowSizeMsg` handler:

```go
case tea.WindowSizeMsg:
    a.width = msg.Width
    a.height = msg.Height
    a.layout.Resize(msg.Width, msg.Height)
    a.propagateSizes()
```

`propagateSizes()` iterates all visible panes and calls `SetSize(contentWidth, contentHeight)` using `layout.PaneRect(id).ContentWidth()` and `.ContentHeight()`.

**Files:**
- Modify: `internal/app/app.go`

**Tests:**
- Unit: WindowSizeMsg updates layout dimensions
- Unit: All visible panes receive SetSize after resize
- Unit: Hidden panes don't receive SetSize (or receive zero)

**Commit:** `feat(app): propagate WindowSizeMsg through LayoutManager to panes`

---

## Task 5: Wire message routing to new panes

**Problem:** Data-loaded messages need to reach the new split panes.

**Fix:**

Update message routing in `app.go` `Update()`:

| Message | Route To |
|---------|----------|
| `PlaybackStateFetchedMsg` | NowPlaying pane |
| `QueueLoadedMsg` | Queue pane |
| `LibraryLoadedMsg` | Playlists pane |
| `AlbumsLoadedMsg` | Albums pane |
| `LikedTracksLoadedMsg` | LikedSongs pane |
| `RecentlyPlayedLoadedMsg` | RecentlyPlayed pane |
| `StatsLoadedMsg` | TopTracks pane + TopArtists pane |
| `FetchStatsMsg` | (from TopTracks/TopArtists — dispatch API call) |
| `PlaylistTracksLoadedMsg` | Playlists pane |
| `Playlist*Msg` (create/rename/remove/reorder) | Playlists pane |
| `VisualizerTickMsg` | NowPlaying pane |
| `TickMsg` | All panes that need periodic refresh |

**Broadcast messages** (sent to ALL panes): `tea.WindowSizeMsg`, `TickMsg`
**Targeted messages** (sent to specific pane): data-loaded messages, request result messages

**Files:**
- Modify: `internal/app/app.go`
- Modify: `internal/app/routing.go`

**Tests:**
- Unit: QueueLoadedMsg reaches QueuePane
- Unit: AlbumsLoadedMsg reaches AlbumsPane
- Unit: StatsLoadedMsg reaches both TopTracks and TopArtists panes
- Unit: PlaylistTracksLoadedMsg reaches PlaylistsPane
- Unit: VisualizerTickMsg reaches NowPlayingPane
- Unit: TickMsg reaches all panes

**Commit:** `feat(app): route messages to new split panes`

---

## Task 6: Update buildView and remove old rendering

**Problem:** `buildView()` still has old viewStats/viewPlaylists branches and old 3-pane rendering.

**Fix:**

1. Replace `buildView()` body:
   - `viewSplash` → renderSplash() (unchanged)
   - `viewAuth` → renderAuthPanel() (unchanged)
   - `viewGrid` → header + renderGrid() + statusBar
   - Remove: `viewStats` branch, `viewPlaylists` branch, old 3-pane JoinHorizontal

2. Update `renderTooSmall()` → change minimum from 100×24 to 120×30

3. Update header to btop style (partially — full header restyle in Feature 50)

4. Remove: `renderPaneWithBorder()` — replaced by `layout.RenderPaneBorder()`

**Files:**
- Modify: `internal/app/render.go`

**Tests:**
- Unit: buildView() in viewGrid mode renders grid with header + content + statusbar
- Unit: Minimum size check uses 120×30
- Unit: Splash and Auth modes still render correctly
- Unit: Old viewStats/viewPlaylists code paths removed

**Commit:** `refactor(app): replace old view modes with grid rendering`

---

## Task 7: Initialize panes in App constructor

**Problem:** App.New() creates old pane instances.

**Fix:**

Update `New()` to create all 10 panes and register them:

```go
panes := map[layout.PaneID]layout.Pane{
    layout.PaneNowPlaying:     panes.NewNowPlayingPane(store, theme),
    layout.PaneQueue:          panes.NewQueuePane(store, theme),
    layout.PanePlaylists:      panes.NewPlaylistsPane(store, theme),
    layout.PaneAlbums:         panes.NewAlbumsPane(store, theme),
    layout.PaneLikedSongs:     panes.NewLikedSongsPane(store, theme),
    layout.PaneRecentlyPlayed: panes.NewRecentlyPlayedPane(store, theme),
    layout.PaneTopTracks:      panes.NewTopTracksPane(store, theme),
    layout.PaneTopArtists:     panes.NewTopArtistsPane(store, theme),
    // PaneRequestFlow and PaneNetworkLog added in Feature 51
}
```

Remove old field assignments: `playerPane`, `libraryPane`, `queuePane`, `statsPane`, `playlistPane`.

**Files:**
- Modify: `internal/app/app.go`

**Tests:**
- Unit: All 8 panes initialized and registered
- Unit: App.Init() batches all pane Init() commands
- Unit: App compiles and runs with new pane structure

**Commit:** `feat(app): initialize all panes via LayoutManager`

---

## Task 8: Comprehensive integration tests

**Files:**
- Modify: `internal/app/app_test.go`
- Modify: `internal/app/render_test.go`
- Modify: `internal/app/routing_test.go`

**Tests:**
- Integration: Full app lifecycle — init → resize → load data → render → verify grid output
- Integration: Preset cycling — p key → layout changes, all panes resize
- Integration: Page toggle — 0 key → switches to Page B (empty for now), back to Page A
- Integration: Pane toggle — hide pane 3 (Playlists) → row reflows
- Integration: Focus rotation — Tab cycles through all 8 visible panes
- Integration: Playback keys work regardless of focus
- Integration: Search overlay still opens and closes correctly
- Integration: Device overlay still opens and closes correctly
- Integration: Data flow — simulate API responses → panes show data
- Integration: Resize → all panes adjust
- Edge: Very small terminal → "too small" message
- Edge: Toggle all panes except one → still renders

**Commit:** `test(app): comprehensive app migration integration tests`

---

## Acceptance Criteria

- [ ] `viewMode` reduced to `viewSplash | viewAuth | viewGrid`
- [ ] `focusedPane` enum deleted, replaced by `layout.Manager.FocusedPane()`
- [ ] Individual pane fields replaced by `panes map[PaneID]layout.Pane`
- [ ] `renderGrid()` assembles panes using `LayoutManager.PaneRect()`
- [ ] Key `0` toggles Page A/B
- [ ] Key `p` cycles presets within current page
- [ ] Keys `1-8` toggle pane visibility on Page A
- [ ] `Tab/Shift+Tab` rotates focus among visible panes
- [ ] Playback keys always route to NowPlaying
- [ ] All data-loaded messages route to correct new panes
- [ ] Overlays (search, devices) still work
- [ ] Minimum terminal size updated to 120×30
- [ ] Old `renderPaneWithBorder()` deleted
- [ ] `make ci` passes

---

## Notes

- This is the largest feature in the redesign. It touches `app.go`, `render.go`, and `routing.go` extensively.
- Page B panes (`RequestFlow`, `NetworkLog`) are not created yet — they're added in Feature 51.
  Until then, Page B shows only NowPlaying compact strip. This is acceptable for the migration.
- The old `LibraryPane`, `StatsView`, and `PlaylistManager` files still exist after this feature
  but are no longer imported or used. They become dead code, deleted in Feature 53.
- The viewSplash and viewAuth modes are special cases — they render full-screen without the grid.
  They are NOT part of the page system.
- The `searchOpen` and `deviceOverlayOpen` flags remain — overlays float above the grid, not inside it.
