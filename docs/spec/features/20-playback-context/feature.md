---
title: "Playback Context & Navigation"
status: open
---

## Overview

When a user plays a track from any pane, Spotify needs a proper context to know
what comes next. Without it, the queue fills with the same song repeated. This
feature fixes playback context across all song-list panes and adds drill-down
navigation to Albums and Playlists so users can browse tracks before playing.

## Goals

- All song-list panes send correct context to Spotify so the queue fills with
  meaningful upcoming tracks
- Playlists pane is fully functional: tracks load on Enter, tracks are playable
- Albums pane has the same drill-down pattern as Playlists

## Out of Scope

- Playlist management operations (rename `r`, remove track `x`, reorder `Shift+↑/↓`,
  create `n`) — addressed in a dedicated story
- Queue pane playback behaviour — existing skip-to-track behaviour is correct and
  unchanged

## Story Implementation Order

**Stories must be implemented in order: 105 → 106 → 107.**

Story 105 introduces `PlayContextMsg.OffsetURI` and the new `PlayTrackListMsg` type.
Both Story 106 and Story 107 depend on these types being present. Implementing 106 or
107 before 105 will cause compile errors.

## Acceptance Criteria

- [ ] Playing a track from Liked Songs fills the queue with subsequent liked songs,
      not repeats of the same track
- [ ] Playing a track from Top Tracks fills the queue with the remaining top tracks
      in order
- [ ] Playing a track from Recently Played fills the queue with the remaining recent
      tracks in order
- [ ] Playing a track from Search results fills the queue with the remaining search
      results in order
- [ ] Pressing Enter on a playlist opens a track sub-view; tracks load from the API;
      pressing Enter on a track plays that track with the playlist as context; queue
      fills with subsequent playlist tracks
- [ ] Pressing Enter on an album opens a track sub-view; tracks load from the API;
      pressing Enter on a track plays that track with the album as context; queue
      fills with subsequent album tracks
- [ ] Pressing Esc from any track sub-view returns to the list without affecting
      current playback
- [ ] The queue pane self-corrects within ~1000ms after any play call (via its
      existing tick-based polling — no additional work required)
- [ ] No regression on existing device switcher, search overlay, or queue pane
      behaviour

## Stories

| # | Title | Status |
|---|-------|--------|
| 105 | Context-aware playback for song-list panes | open |
| 106 | Playlist full functionality | open |
| 107 | Album drill-down + track play | open |
