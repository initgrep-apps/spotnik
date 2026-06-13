---
title: "Player Page Unification"
status: open
---

## Description

Unify the Music and Podcasts pages into a single Player page. The NowPlaying pane
becomes content-aware (renders track info or episode info based on
`currently_playing_type`), the PodcastPlayback pane is deleted (its functionality
absorbed into NowPlaying), the ShowEpisodes pane is deleted (absorbed into
FollowedShows as a drill-down sub-view), and 2 podcast-oriented presets are added
to the Player page alongside the 4 existing music-oriented presets.

The page cycle shrinks from 3 (Music/Podcasts/Stats) to 2 (Player/Stats). An
auto-switch mechanism changes the preset when user-initiated playback crosses
content types (track on podcast preset → Listening; episode on music preset →
Podcast). A visibility-gated polling optimization skips API calls for panes not
visible in the current preset. An Episode Details overlay (`i` key) shows full
episode description when an episode is playing. The Queue pane gets a `type`
column to disambiguate tracks from episodes.

## Motivation

- Spotify's player is content-type agnostic — one player surface, display adapts to
  what's playing. The current 3-page model forces a page switch to see podcast
  data even when a track is playing.
- PodcastPlayback duplicated ~350 lines of transport controls and progress bar
  logic from NowPlaying.
- The 3-page cycle (Music/Podcasts/Stats) prevented seamless content-type
  transitions.
- "Podcast Listening" and "Music Listening" are just different presets of the
  same page.

## Stories

| Story | Title | Status |
|-------|-------|--------|
| 233 | Layout: rename PageMusic→PagePlayer, remove PagePodcasts, 6 presets, delete panes | open |
| 234 | NowPlaying: content-aware rendering (track + episode) | open |
| 235 | Episode Details overlay (`i` key) | open |
| 236 | FollowedShows drill-down (absorb ShowEpisodes) | open |
| 238 | Queue: mixed content support (type column, QueueItem) | open |
| 239 | Auto-switch preset on user-initiated playback | open |
| 240 | Polling optimization: skip invisible panes | open |
| 241 | Search integration: auto-switch on show/episode selection | open |
| 242 | Keybinding updates (3 locations: README, design.md, help overlay) | open |
| 243 | Documentation: architecture.md + design.md updates | open |

## Acceptance Criteria

- [ ] `0` key cycles Player → Stats → Player (2-page cycle, no third page)
- [ ] `p` cycles through 6 presets on Player page (Dashboard, Listening, Podcast, Library, Discovery, Podcast Dashboard)
- [ ] NowPlaying renders track info when `currently_playing_type == "track"`
- [ ] NowPlaying renders episode info when `currently_playing_type == "episode"`
- [ ] NowPlaying shows "Nothing playing" / "Press / to search" for unknown type
- [ ] `i` key opens Episode Details overlay when episode is playing; silent no-op for tracks
- [ ] FollowedShows Enter → episode sub-view (drill-down); Esc → show list (Level 1)
- [ ] FollowedShows drill-down shows episodes with pagination
- [ ] PodcastPlayback pane deleted; ShowEpisodes pane deleted
- [ ] `PaneBorderPodcastPlayback` and `PaneBorderShowEpisodes` removed from Theme interface and all 13 themes
- [ ] Queue pane shows `♪` for tracks, `◆` for episodes in new `type` column; `Track` header → `Title`
- [ ] Auto-switch: user plays track on podcast preset → Listening; user plays episode on music preset → Podcast
- [ ] Background playback changes do NOT trigger preset switches
- [ ] Polling skips panes not visible in current preset; newly visible panes fetch stale data on preset switch
- [ ] Search: track/album/artist → stays on music preset or switches to Listening; show/episode → switches to Podcast preset
- [ ] Contextual toggle keys: music presets get keys 1-8, podcast presets get keys 1-4
- [ ] Keybindings updated in all 3 locations (README, design.md §17, help_overlay.go)
- [ ] Stats page compact NowPlaying strip shows episode info when applicable
- [ ] `make ci` passes (lint + tests + 80% coverage)

## Design Reference

See `docs/superpowers/specs/2026-06-13-player-page-unification-design.md` for the
full architectural design including preset grids, auto-switch rules, Queue column
definitions, and Spotify API notes.

## Implementation Reference

See `docs/superpowers/plans/2026-06-13-player-page-unification.md` for the
detailed task-by-task implementation plan.

## Codebase Findings

Key findings from codebase exploration that affect implementation:

1. **`PaneBorderPodcastPlayback()` is dead code**: Defined in Theme interface but
   `border.go` maps PodcastPlayback to `PaneBorderNowPlaying()` (green accent).
   Removing it from the Theme interface is cleanup, not a behavior change.

2. **NowPlaying shows empty when episode plays**: Current `View()` renders empty
   state when `ps.Item == nil` (true during episode playback). This is the
   critical bug that story 234 fixes.

3. **`htmlToMarkdown()` and `renderMarkdown()`** are unexported helpers in
   `internal/ui/panes/htmlrender.go`. The Episode Details overlay (also in the
   `panes` package) can call them directly without a package prefix.

4. **`store.Queue()` returns `[]domain.Track`**: Queue unmarshaling, `QueueLoadedMsg`,
   and store accessor all need updating for mixed content. Story 238 covers this.

5. **`PlayEpisodeMsg.PlaylistURI`** is misleadingly named — it's used for show
   context URI (`spotify:show:XXX`). No rename needed, but implementers should
   be aware.

6. **No `SetPreset()` or `IsPaneVisible()` methods** exist on `layout.Manager`.
   Both must be added. Story 233 covers `SetPreset()`; story 240 covers
   `IsPaneVisible()`.

7. **`SearchResultSelectedMsg` handler** uses `TogglePage()` cycling to reach
   Podcasts. Story 241 replaces this with direct `SetPreset()` calls.