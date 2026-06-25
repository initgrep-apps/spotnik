---
title: "Podcasts & Player Unification"
status: done
stories: 227–232, 233–236, 238–244
---

## Description

**Phase 1 — Podcasts Page (stories 227–232):** Added a third page to Spotnik's page cycle for podcast and episode playback with 4 new panes (PodcastPlayback, ShowEpisodes, FollowedShows, SavedEpisodes), 2 presets, a standalone PodcastAPI client, and supporting domain/state types.

**Phase 2 — Player Page Unification (stories 233–244, absorbed from feature 19):** Unified Music and Podcasts into a single Player page. The NowPlaying pane became content-aware (renders track or episode info based on `currently_playing_type`). PodcastPlayback and ShowEpisodes panes were deleted — their functionality absorbed into NowPlaying and FollowedShows drill-down respectively. The page cycle shrank from 3 (Music/Podcasts/Stats) to 2 (Player/Stats). Added auto-switch presets on content-type transitions, visibility-gated polling, Episode Details overlay (`i` key), and mixed-content Queue with type column.

## Stories

| Story | Title | Status |
|-------|-------|--------|
| 227 | Domain types + PodcastAPI client | done |
| 228 | State store + messages | done |
| 229 | Layout, presets, theme border tokens | done |
| 230 | PodcastPlayback pane | done |
| 231 | Table panes (ShowEpisodes + FollowedShows + SavedEpisodes) | done |
| 232 | App wiring (commands, routing, handlers, docs) | done |
| 233 | Layout: rename PageMusic→PagePlayer, remove PagePodcasts, 6 presets, delete panes | done |
| 234 | NowPlaying: content-aware rendering (track + episode) | done |
| 235 | Episode Details overlay (`i` key) | done |
| 236 | FollowedShows drill-down (absorb ShowEpisodes) | done |
| 238 | Queue: mixed content support (type column, QueueItem) | done |
| 239 | Auto-switch preset on user-initiated playback | done |
| 240 | Polling optimization: skip invisible panes | done |
| 241 | Search integration: auto-switch on show/episode selection | done |
| 242 | Keybinding updates (3 locations) | done |
| 243 | Documentation: architecture.md + design.md updates | done |
| 244 | Episode Details: viewport-based scrolling | done |

## Acceptance Criteria

- [ ] `0` key cycles Player → Stats → Player (2-page cycle)
- [ ] NowPlaying renders track info when `currently_playing_type == "track"`
- [ ] NowPlaying renders episode info when `currently_playing_type == "episode"`
- [ ] `i` key opens Episode Details overlay when episode is playing; silent no-op for tracks
- [ ] FollowedShows Enter → episode sub-view (drill-down); Esc → show list
- [ ] PodcastPlayback and ShowEpisodes panes deleted; functionality absorbed
- [ ] Queue pane shows `♪` for tracks, `◆` for episodes in `type` column
- [ ] Auto-switch preset on content-type transitions (user-initiated only)
- [ ] Polling skips panes not visible in current preset
- [ ] Search: track/album/artist → Listening preset; show/episode → Podcast preset
- [ ] 4 border tokens (`BorderPodcastPlayback`, `BorderShowEpisodes`, `BorderFollowedShows`, `BorderSavedEpisodes`) in all 13 themes
- [ ] Keybindings updated in all 3 locations (README, design.md, help_overlay.go)
- [ ] `make ci` passes (lint + tests + 80% coverage)
