---
title: "Layout: rename PageMusic→PagePlayer, remove PagePodcasts, 6 presets, delete old panes"
feature: 19-player-page-unification
status: open
---

## Background

The 3-page model (Music/Podcasts/Stats) is being unified into a 2-page model
(Player/Stats). This requires renaming `PageMusic` to `PagePlayer`, removing
`PagePodcasts` from the page cycle, deleting the `PodcastPlayback` and
`ShowEpisodes` panes (their functionality is absorbed by NowPlaying and
FollowedShows drill-down respectively in later stories), and defining 6 Player
page presets.

**All changes in this story must compile.** Renaming and deletion are done
together to avoid intermediate broken builds.

## Design

### Page rename

- `PageMusic` → `PagePlayer` in all references
- `PagePodcasts` constant removed from iota; `PageStats` renumbered
- `PagePlayerPresets` replaces `PageMusicPresets` (contains 6 presets)
- `PagePodcastsPresets` removed entirely
- `TogglePage()` cycles Player → Stats → Player (2 states)

### Deleted panes

- `PanePodcastPlayback` removed from PaneID iota
- `PaneShowEpisodes` removed from PaneID iota
- `PodcastPlaybackPane` struct and `podcastplayback.go` file deleted
- `ShowEpisodesPane` struct and `showepisodes.go` file deleted
- `podcastplayback_test.go` and `showepisodes_test.go` deleted

### Deleted theme tokens

- `PaneBorderPodcastPlayback()` removed from Theme interface
- `PaneBorderShowEpisodes()` removed from Theme interface
- `border.go` switch case for `PanePodcastPlayback` removed
- `border.go` switch case for `PaneShowEpisodes` removed

Note: `PaneBorderPodcastPlayback()` was dead code — the border mapping used
`PaneBorderNowPlaying()` for PodcastPlayback. Removing it from the Theme
interface is cleanup, not a behavior change.

### 6 Player presets

| # | Name | Visible Panes |
|---|------|---------------|
| 0 | Dashboard | NP, Queue, Playlists, Albums, LikedSongs, RecentlyPlayed, TopTracks, TopArtists |
| 1 | Listening | NP, Queue, RecentlyPlayed |
| 2 | Podcast | NP, FollowedShows, Queue |
| 3 | Library | NP, Playlists, Albums, LikedSongs |
| 4 | Discovery | NP, TopTracks, TopArtists, RecentlyPlayed |
| 5 | Podcast Dashboard | NP, FollowedShows, SavedEpisodes, Queue |

`PresetDashboard` through `PresetDiscovery` reuse existing grid definitions
(with `PageMusic` references updated to `PagePlayer`). `PresetPodcast` and
`PresetPodcastDashboard` are new.

The old `PresetPodcastListening` and `PresetPodcastDashboard` are removed.

### Removed app wiring

- `podcastPlaybackPane()` accessor removed from `app.go`
- `showEpisodesPane()` accessor removed from `app.go`
- `PanePodcastPlayback` and `PaneShowEpisodes` removed from `panesMap`
- Podcast page toggle key map removed from `routing.go`
- Podcast playback key routing removed (playback keys route to NowPlaying only)
- `0` key handler cycles Player → Stats → Player

### New layout methods

- `SetPreset(index int)` — sets preset directly (for auto-switch, story 239)
- `IsPaneVisible(id PaneID) bool` — checks current preset's Visible map (for polling optimization, story 240)

### FollowedShows/SavedEpisodes remain in preset Visible maps

The panes stay; only `PodcastPlayback` and `ShowEpisodes` are deleted.
`FollowedShowsPane` and `SavedEpisodesPane` continue to exist. Their
drill-down conversion (story 236) and the `i` overlay key are separate stories.

## Files

### Create

- `internal/ui/layout/preset_player.go` — 6 Player presets with grid definitions
  (or inline in `presets.go` if that's the existing pattern)

### Modify

- `internal/ui/layout/pane.go` — rename `PageMusic`→`PagePlayer`, remove `PagePodcasts`, remove `PanePodcastPlayback` and `PaneShowEpisodes`
- `internal/ui/layout/presets.go` — remove `PresetPodcastListening`/`PresetPodcastDashboard`/`PagePodcastsPresets`, add `PresetPodcast`/`PresetPodcastDashboard`, rename `PageMusicPresets`→`PagePlayerPresets`
- `internal/ui/layout/layout.go` — `TogglePage()` 2-cycle, add `SetPreset()`, add `IsPaneVisible()`
- `internal/app/app.go` — remove `podcastPlayback`/`showEpisodes` fields and accessors, remove from `panesMap`
- `internal/app/routing.go` — remove podcast toggle key map, remove podcast playback key routing, update `0` for 2-cycle, add `currentToggleKeyMap()` for contextual panes
- `internal/app/handlers.go` — remove PodcastPlayback/ShowEpisodes message handlers, remove `SelectedShowChangedMsg` routing
- `internal/app/commands.go` — remove `podcastClient` field (keep API method references for `buildFetchShowEpisodesCmd` etc., which will be used by FollowedShows drill-down)
- `internal/ui/theme/theme.go` — remove `PaneBorderPodcastPlayback()` and `PaneBorderShowEpisodes()` from Theme interface
- `internal/ui/theme/config_theme.go` — remove `PaneBorderPodcastPlayback()` and `PaneBorderShowEpisodes()` methods and their `paneBorderColors` fields
- `internal/ui/layout/border.go` — remove `PanePodcastPlayback` and `PaneShowEpisodes` cases

### Delete

- `internal/ui/panes/podcastplayback.go`
- `internal/ui/panes/podcastplayback_test.go`
- `internal/ui/panes/showepisodes.go`
- `internal/ui/panes/showepisodes_test.go`

## Acceptance Criteria

- [ ] `PageMusic` no longer exists anywhere in the codebase; `PagePlayer` used instead
- [ ] `PagePodcasts` no longer exists
- [ ] `PanePodcastPlayback` and `PaneShowEpisodes` no longer exist
- [ ] `PodcastPlaybackPane` and `ShowEpisodesPane` files deleted
- [ ] `PaneBorderPodcastPlayback()` and `PaneBorderShowEpisodes()` removed from Theme interface and ConfigTheme implementation
- [ ] `0` key cycles Player → Stats → Player
- [ ] 6 Player presets defined with correct Visible maps and grids
- [ ] `SetPreset(index int)` method on layout Manager
- [ ] `IsPaneVisible(id PaneID) bool` method on layout Manager
- [ ] Contextual toggle keys: music presets 1-8, podcast presets 1-4
- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] `make ci` passes

## Tasks

- [ ] Define `PagePlayer` constant, remove `PagePodcasts`, rename `PageMusic` references
      - Modify `internal/ui/layout/pane.go`: `PagePlayer` replaces `PageMusic`, remove `PagePodcasts` from iota
      - test: `TestPageIDs`, `TestPagePlayer_Value`
- [ ] Define `PagePlayerPresets` with 6 presets, remove `PagePodcastsPresets` and old podcast presets
      - Modify `internal/ui/layout/presets.go`: add `PresetPodcast`, `PresetPodcastDashboard`, `PagePlayerPresets` (6 entries), remove `PagePodcastsPresets`, `PresetPodcastListening`, `PresetPodcastDashboard` (old versions)
      - test: `TestPagePlayerPresets_HasSixEntries`, `TestPresetPodcast_Grid`, `TestPresetPodcastDashboard_Grid`
- [ ] Update `TogglePage()` for 2-cycle (Player ↔ Stats), add `SetPreset()` and `IsPaneVisible()` methods
      - Modify `internal/ui/layout/layout.go`
      - test: `TestTogglePage_PlayerStatsTwoCycle`, `TestSetPreset_DirectSwitch`, `TestIsPaneVisible_CurrentPreset`
- [ ] Remove `PanePodcastPlayback` and `PaneShowEpisodes` from PaneID iota
      - Modify `internal/ui/layout/pane.go`: remove from iota, shift subsequent values
      - test: `TestPaneIDs_NoPodcastPlaybackOrShowEpisodes`
- [ ] Delete `PodcastPlaybackPane` and `ShowEpisodesPane` files
      - Delete `internal/ui/panes/podcastplayback.go`, `internal/ui/panes/podcastplayback_test.go`, `internal/ui/panes/showepisodes.go`, `internal/ui/panes/showepisodes_test.go`
      - test: `go build ./...` compiles
- [ ] Remove `PaneBorderPodcastPlayback()` and `PaneBorderShowEpisodes()` from Theme interface and ConfigTheme
      - Modify `internal/ui/theme/theme.go`: remove 2 method signatures from interface
      - Modify `internal/ui/theme/config_theme.go`: remove method implementations and `paneBorderColors` fields
      - Modify `internal/ui/layout/border.go`: remove 2 switch cases
      - test: `TestTheme_InterfaceCompliance`, `TestBorderPane_NoneForRemovedPanes`
- [ ] Remove podcast pane wiring from `app.go`, `routing.go`, `handlers.go`, `commands.go`
      - Remove `podcastPlaybackPane`/`showEpisodesPane` fields and accessors from `app.go`
      - Remove from `panesMap`, remove podcast toggle key map from `routing.go`, update `0` for 2-cycle
      - Remove `PodcastPlaybackPane`/`ShowEpisodesPane` message handlers from `handlers.go`
      - Remove `podcastClient` field from `commands.go`
      - test: `go build ./...` compiles, `go test ./internal/app/...` passes
- [ ] Add contextual toggle keys: music presets (1–8), podcast presets (1–4)
      - Modify `internal/app/routing.go`: add `currentToggleKeyMap()` that returns keys based on active preset's Visible map
      - test: `TestCurrentToggleKeyMap_MusicPreset`, `TestCurrentToggleKeyMap_PodcastPreset`
- [ ] Create `internal/ui/layout/preset_player.go` with grid definitions for 6 Player presets
      - test: `TestPresetDashboard_PlayerGrid`, `TestPresetPodcast_PlayerGrid`
- [ ] Run `make ci` — all lint, tests, and 80% coverage pass