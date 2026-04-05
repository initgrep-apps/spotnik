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

## Stories

| # | Title | Status |
|---|-------|--------|
| 105 | Context-aware playback for song-list panes | open |
| 106 | Playlist full functionality | open |
| 107 | Album drill-down + track play | open |
