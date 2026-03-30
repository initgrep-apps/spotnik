---
title: "App Migration"
feature: 12-layout
status: done
---

## Background
The current app.go (~41KB) uses a viewMode enum (viewMain, viewStats, viewPlaylists plus viewSplash, viewAuth), a focusedPane enum (focusPlayer, focusLibrary, focusQueue), hardcoded 3-pane JoinHorizontal in render.go, individual pane fields, and fixed pane widths (22%/50%/28%). This story replaces all of this with layout.Manager for grid computation, presets, page/pane toggling; a panes map[PaneID]layout.Pane for all 10 panes; renderGrid() that assembles panes into the grid using Manager.PaneRect(); p for preset cycling, 0 for page toggle, 1-8 for pane toggle; and Tab/Shift+Tab for focus rotation via Manager.RotateFocus().

Design reference: docs/DESIGN.md sections 3, 4, 16, 22, 23.

## Design

### View Mode Consolidation
Keep viewSplash, viewAuth; add viewGrid (replaces viewMain, viewStats, viewPlaylists). Delete old modes and focusedPane enum.

### renderGrid()
Group visible panes by row (using Rect.Y), for each row join cells horizontally via lipgloss.JoinHorizontal, wrap each pane in btop border via layout.RenderPaneBorder(), cap each cell to exact rect dimensions, join rows vertically.

### Key Routing
- `0` -> layout.TogglePage()
- `p` -> layout.CyclePreset()
- `1-8` -> layout.TogglePane() via toggleMap
- `tab` -> layout.RotateFocus(true)
- `shift+tab` -> layout.RotateFocus(false)

### Message Routing Table
PlaybackStateFetchedMsg -> NowPlaying, QueueLoadedMsg -> Queue, LibraryLoadedMsg -> Playlists, AlbumsLoadedMsg -> Albums, LikedTracksLoadedMsg -> LikedSongs, RecentlyPlayedLoadedMsg -> RecentlyPlayed, StatsLoadedMsg -> TopTracks + TopArtists, VisualizerTickMsg -> NowPlaying, TickMsg -> all panes (broadcast).

## Acceptance Criteria
- [ ] viewMode reduced to viewSplash | viewAuth | viewGrid
- [ ] focusedPane enum deleted
- [ ] Individual pane fields replaced by panes map
- [ ] renderGrid() assembles panes using LayoutManager.PaneRect()
- [ ] Key 0 toggles Page A/B, p cycles presets, 1-8 toggle panes
- [ ] Tab/Shift+Tab rotates focus
- [ ] Playback keys always route to NowPlaying
- [ ] All data-loaded messages route to correct panes
- [ ] Overlays still work
- [ ] Minimum terminal size 120x30
- [ ] make ci passes

## Tasks
- [ ] Replace viewMode and focus enums in internal/app/app.go
      - test: App starts with viewSplash, transitions to viewGrid; layout.FocusedPane() returns valid PaneID; all 8 panes registered
- [ ] Implement renderGrid() in internal/app/render.go
      - test: correct total dimensions; each pane border appears; grid respects preset; hidden panes excluded
- [ ] Wire key routing for pages, presets, and toggles in routing.go
      - test: key 0 toggles page; p cycles presets; 1/5 toggle panes; Tab/Shift+Tab focus; playback keys route to NowPlaying
- [ ] Wire WindowSizeMsg propagation
      - test: resize updates layout; visible panes receive SetSize; hidden panes excluded
- [ ] Wire message routing to new panes
      - test: each data message reaches correct pane; TickMsg broadcast to all
- [ ] Update buildView and remove old rendering in render.go
      - test: viewGrid renders grid; minimum size 120x30; splash/auth unchanged; old branches removed
- [ ] Initialize panes in App constructor
      - test: all 8 panes initialized; Init() batches all pane commands; compiles and runs
- [ ] Comprehensive integration tests
      - test: full lifecycle; preset cycling; page toggle; pane toggle; focus rotation; playback keys; search/device overlays; data flow; resize; edge cases
