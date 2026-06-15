---
title: "Documentation: architecture.md + design.md updates"
feature: 19-player-page-unification
status: done
---

## Background

The Player Page Unification feature changes core architecture (page model,
preset system, pane layout, polling, keybindings). System documentation must be
updated to reflect these changes.

## Design

### `docs/system/architecture.md` updates

1. **Page / Preset / Toggle System**: Replace the 3-page model
   (Music/Podcasts/Stats) with the 2-page model (Player/Stats). Update pane
   lists, page cycling description (`0` cycles Player → Stats), and preset
   tables to include all 6 Player presets + 1 Stats preset.

2. **Polling Architecture**: Add new subsection on **Visibility-Gated Polling**:
   - Update library polling description: iteration skips panes not visible in
     the current preset
   - Add rule: `layout.IsPaneVisible(PaneID)` check at top of each polling entry
   - Add rule: On preset switch, immediately check staleness for newly visible
     panes and dispatch fetch if stale
   - Add rule: Playback state and queue always poll regardless of preset
   - Include the "What Polls When" table from story 240

3. **Pane list**: Update pane count — NowPlaying is content-aware, FollowedShows
   has drill-down, PodcastPlayback and ShowEpisodes are deleted

4. **Toggle key section**: Replace page-specific toggle descriptions with
   context-aware model (toggle keys adapt based on active preset type)

### `docs/system/design.md` updates

1. **§2 Pane Definitions**: Replace 3-page pane tables with unified 2-page
   model:
   - Player page: NowPlaying (1), Queue (2), FollowedShows (2/3), SavedEpisodes
     (3), Playlists (3), Albums (4), LikedSongs (5), RecentlyPlayed (6),
     TopTracks (7), TopArtists (8) — contextual toggle keys
   - Stats page: NowPlaying, GatewayHealth, PollingTraffic, GatewayLive,
     NetworkLog (unchanged)
   - Remove Podcasts page table entirely

2. **§4 Pages, Pane Toggling, and Preset Layouts**: Replace 3-page cycle with
   2-page cycle. Replace Music page presets with 6 Player presets. Remove
   Podcasts page presets. Add auto-switch rules table. Add contextual toggle
   key documentation.

3. **§9 Dense Table Formatting**: Update Queue pane column table:
   - Add `type` column (flex 1, `♪` for track, `◆` for episode)
   - Rename `Track` to `Title` (flex 7)
   - `Artist` shows show name for episodes (flex 4)

4. **§10 Per-Pane Border Colors**: Remove `PaneBorderPodcastPlayback` and
   `PaneBorderShowEpisodes`. Keep `PaneBorderFollowedShows` and
   `PaneBorderSavedEpisodes`.

5. **§16 Focus & Navigation**: Update playback keys to route to NowPlaying only
   (remove PodcastPlayback reference). Add `i` key for Episode Details overlay.

6. **§17 Keybinding Table**: Replace `0` cycle description (Player → Stats only).
   Update toggle keys for contextual behavior. Add `i` keybinding. Remove
   Podcast page toggle keys.

7. **§18 Theme Enhancements**: Remove `PaneBorderPodcastPlayback` and
   `PaneBorderShowEpisodes` tokens. Keep `PaneBorderFollowedShows` and
   `PaneBorderSavedEpisodes`. All 11 themes need updates.

## Files

### Modify

- `docs/system/architecture.md`
- `docs/system/design.md`

## Acceptance Criteria

- [ ] architecture.md reflects 2-page model (Player/Stats)
- [ ] architecture.md documents visibility-gated polling
- [ ] architecture.md lists correct pane count and names
- [ ] design.md §2 reflects 2-page model with contextual toggle keys
- [ ] design.md §4 has 6 Player presets and auto-switch rules
- [ ] design.md §9 has updated Queue columns with type column
- [ ] design.md §10 removes deleted border tokens
- [ ] design.md §16 routes playback to NowPlaying only, adds `i`
- [ ] design.md §17 updated keybinding table
- [ ] design.md §18 removes deleted theme tokens

## Tasks

- [ ] Update `docs/system/architecture.md` — Page/Preset/Toggle system section
      - Replace 3-page model with 2-page (Player/Stats), update pane lists, page cycling (`0` cycles Player ↔ Stats), update preset tables for all 6 Player presets + 1 Stats preset
      - test: manual review (documentation-only)
- [ ] Update `docs/system/architecture.md` — add Visibility-Gated Polling subsection
      - Document `layout.IsPaneVisible(PaneID)` check, per-pane skip rule, preset switch staleness check, "What Polls When" table, playback/queue exemptions
      - test: manual review
- [ ] Update `docs/system/architecture.md` — Pane list and toggle key section
      - Update pane count (NowPlaying content-aware, FollowedShows drill-down, PodcastPlayback/ShowEpisodes deleted), replace page-specific toggle with context-aware model
      - test: manual review
- [ ] Update `docs/system/design.md` §2 — Pane Definitions
      - Replace 3-page tables with unified 2-page model (Player/Stats), remove Podcasts page table, add contextual toggle key descriptions
      - test: manual review
- [ ] Update `docs/system/design.md` §4 — Pages, Pane Toggling, Preset Layouts
      - Replace 3-page cycle with 2-page cycle, replace Music presets with 6 Player presets, remove Podcasts presets, add auto-switch rules table, add contextual toggle key docs
      - test: manual review
- [ ] Update `docs/system/design.md` §9 — Queue column table
      - Add `type` column (flex 1, ♪/◆), rename Track → Title (flex 7), update Artist for episodes
      - test: manual review
- [ ] Update `docs/system/design.md` §10 — Per-Pane Border Colors
      - Remove `PaneBorderPodcastPlayback` and `PaneBorderShowEpisodes`, keep `PaneBorderFollowedShows` and `PaneBorderSavedEpisodes`
      - test: manual review
- [ ] Update `docs/system/design.md` §16 — Focus & Navigation
      - Route playback keys to NowPlaying only (remove PodcastPlayback reference), add `i` key for Episode Details overlay
      - test: manual review
- [ ] Update `docs/system/design.md` §17 — Keybinding Table
      - Replace `0` description, update toggle keys for contextual behavior, add `i` keybinding, remove Podcast page toggle keys
      - test: manual review
- [ ] Update `docs/system/design.md` §18 — Theme Enhancements
      - Remove `PaneBorderPodcastPlayback` and `PaneBorderShowEpisodes` tokens, keep FollowedShows/SavedEpisodes, note ConfigTheme removal
      - test: manual review
- [ ] Run `make ci` — verify no broken doc links or format issues