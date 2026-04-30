---
title: "Fix Library Display"
feature: 04-library
status: closed
---

## Background
Library pane shows only section headers, no items visible. `NewLibraryTree()` creates all sections with `Expanded: false`. Data is fetched on init (playlists, recently played) and stored, but sections remain collapsed. User must manually press Enter on each section to see anything. The original spec says "Playlists visible within 2 seconds of app start" -- this was not implemented. The agent followed the lazy-expand pattern for ALL sections, but the spec intended Playlists to be visible immediately on load.

## Design

- Auto-expand Playlists section when `LibraryLoadedMsg` arrives and items exist
- Auto-expand Recently Played section when `RecentlyPlayedLoadedMsg` arrives
- Keep Albums and Liked Songs collapsed (lazy load on expand)

### Files
- `internal/ui/panes/library.go` -- Update handler for load messages to set `Expanded: true`
- `internal/ui/panes/library_test.go` -- Tests for auto-expand behavior

## Acceptance Criteria
- [ ] Playlists section auto-expands when playlists load on init
- [ ] Recently Played section auto-expands when data arrives
- [ ] Albums and Liked Songs remain collapsed (lazy load)
- [ ] Items are visible within 2 seconds of app start (per original spec)
- [ ] Tests verify auto-expand behavior on LibraryLoadedMsg

## Tasks
- [ ] Auto-expand Playlists section on `LibraryLoadedMsg` when items exist
      - test: LibraryLoadedMsg with playlists sets Playlists section Expanded=true
- [ ] Auto-expand Recently Played section on `RecentlyPlayedLoadedMsg`
      - test: RecentlyPlayedLoadedMsg sets Recently Played section Expanded=true
- [ ] Verify Albums and Liked Songs remain collapsed
      - test: After init, Albums and Liked Songs sections have Expanded=false
