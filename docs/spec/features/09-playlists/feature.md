---
title: "Playlist Manager"
status: done
---

## Description
Full playlist management from the terminal -- create, rename, reorder, and remove tracks without leaving Spotnik, turning it into a music curation tool. The Playlist Manager uses a dedicated view (activated by pressing `3`) with a dual-pane layout: the left pane lists playlists with track counts, and the right pane displays tracks with duration, artist, and reorder/remove capabilities. All mutation operations use optimistic updates that revert on API error, with errors shown via toast notifications.

## Acceptance Criteria
- [ ] `3` opens Playlist Manager with playlists loaded from store
- [ ] New playlist created and visible within 1 second
- [ ] Playlist rename updates immediately (optimistic), reverts on API error
- [ ] Track removed from playlist immediately (optimistic), reverts on error
- [ ] Track reorder moves visually before API confirms, reverts on error
- [ ] On any API error: list reverts to previous state, error shown in status bar
- [ ] All API calls and pane `Update()` handlers tested
