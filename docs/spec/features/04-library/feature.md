---
title: "Library Browser"
status: done
---

## Description
Provides browsable access to the user's Spotify library -- playlists, saved albums, and liked songs -- with keyboard navigation, playback, filtering, and playlist management, all rendered as independent panes with dense table layouts. Originally built as a monolithic `LibraryPane` with collapsible tree sections, later split into three independent panes (`PlaylistsPane`, `AlbumsPane`, `LikedSongsPane`) each implementing the `layout.Pane` interface with dense table format and in-pane filtering. All panes read data from the central Store and never call the API directly.

## Acceptance Criteria
- [ ] Playlists visible within 2 seconds of app start
- [ ] Pressing Enter on a playlist starts playing it within 500ms
- [ ] `PlaylistsPane`, `AlbumsPane`, `LikedSongsPane` all satisfy `layout.Pane`
- [ ] All 3 panes use bubble-table with correct column widths from DESIGN.md
- [ ] All 3 panes support in-pane filtering with `f` key
- [ ] PlaylistsPane merges PlaylistManager features (create, rename, delete, reorder, track sub-view)
- [ ] Each pane reads from Store, emits request messages (no direct API calls)
- [ ] All API functions and pane Update() handlers tested
