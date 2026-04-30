---
title: "Stats & Listening History"
status: done
---

## Description

Three panes displaying the authenticated user's Spotify listening history: TopTracks, TopArtists, and RecentlyPlayed. Each is an independent grid pane backed by the Spotify `/me/top/` and `/me/player/recently-played` endpoints. Time range cycles between past 4 weeks, 6 months, and all time via the `g` key. TopArtists supports Enter to play the artist's top tracks. RecentlyPlayed shows relative timestamps (FormatRelativeTime).

## Acceptance Criteria

- [ ] TopTracks, TopArtists, and RecentlyPlayed render as independent grid panes
- [ ] Time range (g key) cycles short/medium/long correctly for both top tracks and top artists
- [ ] TopArtists Enter plays the selected artist
- [ ] RecentlyPlayed shows human-readable relative timestamps
- [ ] Empty state handled cleanly when no history is available
- [ ] Open: story 55 (recently played empty state fix)
