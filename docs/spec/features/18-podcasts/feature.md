---
title: "Podcasts Page"
status: open
---

## Description

Add a third page to Spotnik's page cycle for podcast and episode playback. The
Podcasts page introduces 4 new panes (PodcastPlayback, ShowEpisodes,
FollowedShows, SavedEpisodes), 2 presets (Listening, Dashboard), a standalone
PodcastAPI client, and supporting domain/state types.

Podcasts have a distinct layout from the music page: a 2-row vertical split
instead of the music page's 3-row grid. The top row is always PodcastPlayback
(full width, weight 2). The bottom row holds the table-based panes (weight 3)
with flex widths varying by preset.

No changes to the Music or Stats page NowPlaying pane — PodcastPlayback is the
dedicated episode display surface.

## Stories

| Story | Title | Status |
|-------|-------|--------|
| 227 | Domain types + PodcastAPI client | open |
| 228 | State store + messages | open |
| 229 | Layout, presets, theme border tokens | open |
| 230 | PodcastPlayback pane | open |
| 231 | Table panes (ShowEpisodes + FollowedShows + SavedEpisodes) | open |
| 232 | App wiring (commands, routing, handlers, docs) | open |

## Acceptance Criteria

- [ ] `0` key cycles Music → Podcasts → Stats → Music (3-page cycle)
- [ ] 2 presets: Listening (default) and Dashboard (`D` key)
- [ ] PodcastPlayback pane shows episode info + details + progress bar
- [ ] ShowEpisodes pane shows episodes for the selected show
- [ ] FollowedShows pane lists user's saved shows
- [ ] SavedEpisodes pane lists user's saved episodes
- [ ] `Enter` on a playable episode starts playback with resume position
- [ ] `Enter` on an unplayable episode shows market-restriction toast
- [ ] Search auto-navigation: selecting a show/episode from search switches to Podcasts page
- [ ] `additional_types=episode` query param appended to player state requests
- [ ] `user-read-playback-position` scope added to auth request
- [ ] 4 new border tokens (`BorderPodcastPlayback`, `BorderShowEpisodes`, `BorderFollowedShows`, `BorderSavedEpisodes`) in all 13 themes
- [ ] `make ci` passes (lint + tests + 80% coverage)
