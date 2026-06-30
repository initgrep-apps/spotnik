---
title: "Library Browser & Playlists"
status: in-progress
---

## Description

Three dedicated panes for browsing the user's music library: Playlists, Albums, and LikedSongs. Each is an independent grid pane with dense bubble-table rows and filter support. Selecting a playlist or album drills into its track list. The Playlist Manager extends the playlists pane with full CRUD — create (n), rename (r), delete, and track reordering (Shift+↑/↓). Lazy loading fetches paginated results from Spotify as the user scrolls.

## Acceptance Criteria

- [ ] Playlists, Albums, and LikedSongs each render as independent grid panes
- [ ] Entering a playlist or album shows its track list with album art metadata
- [ ] Playlist create/rename/delete operations reflect immediately (optimistic update)
- [ ] Track reordering with Shift+↑/↓ persists to Spotify API
- [ ] Filter (f key) narrows rows without additional API calls
- [ ] Open: story 10 (library display fixes)
- [x] Done: story 267 (like/unlike core infrastructure)
- [x] Done: story 268 (like/unlike cross-pane wiring)
- [ ] Open: story 269 (fix like/unlike UX — revert heart, border action, 403 error)
