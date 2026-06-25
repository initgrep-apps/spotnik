---
title: "Search integration: auto-switch on show/episode selection"
feature: 18-podcasts
status: done
---

## Background

Currently, `SearchResultSelectedMsg` handler uses `TogglePage()` cycling to
reach the Podcasts page. With the 2-page model, search result selection must
use `autoSwitchPreset()` instead, following the same content-type rules as
pane-initiated playback.

## Design

### Current behavior (to be replaced)

```go
case panes.SearchResultSelectedMsg:
    if m.IsShow || m.IsEpisode {
        // Navigate to Podcasts page via TogglePage cycling
        switch a.layout.ActivePage() {
        case layout.PageMusic:
            a.layout.TogglePage() // → Podcasts
        case layout.PageStats:
            a.layout.TogglePage() // → Music
            a.layout.TogglePage() // → Podcasts
        }
        // ...
    }
```

### New behavior

```go
case panes.SearchResultSelectedMsg:
    if m.IsShow || m.IsEpisode {
        a.autoSwitchPreset("episode")  // switches to Podcast (2) if on music preset
    } else {
        a.autoSwitchPreset("track")      // switches to Listening (1) if on podcast preset
    }
```

| Selection in Search | Current Preset Type | Action |
|---------------------|--------------------|--------|
| Track | Music preset | Play. No switch. |
| Track | Podcast preset | Play → auto-switch to Listening (1) |
| Album | Music preset | Open album sub-view. No switch. |
| Album | Podcast preset | Open album sub-view → auto-switch to Listening (1) |
| Artist | Music preset | Open artist detail. No switch. |
| Artist | Podcast preset | Open artist detail → auto-switch to Listening (1) |
| Show | Music preset | Enter show in FollowedShows drill-down → auto-switch to Podcast (2) |
| Show | Podcast preset | Enter show in FollowedShows drill-down. No switch. |
| Episode | Music preset | Play → auto-switch to Podcast (2) |
| Episode | Podcast preset | Play. No switch. |

The search overlay UI doesn't change. Only the post-selection routing changes:
instead of `TogglePage()` cycling, dispatch the appropriate auto-switch command.

### Show selection handling

When a show is selected from search, the handler should:
1. Set `selectedShowID` in store
2. Dispatch `FetchShowEpisodesRequestMsg` for the show
3. Call `autoSwitchPreset("episode")` to switch to Podcast preset if needed
4. Close search overlay

## Files

### Modify

- `internal/app/handlers.go` — replace `TogglePage()` cycling in
  `SearchResultSelectedMsg` handler with `autoSwitchPreset()` calls

## Acceptance Criteria

- [ ] Selecting a track from search while on podcast preset switches to Listening
- [ ] Selecting an episode from search while on music preset switches to Podcast
- [ ] Selecting a show from search while on music preset switches to Podcast and loads episodes
- [ ] Selecting a show from search while on podcast preset — no switch, just loads episodes
- [ ] Track/album/artist selection on music preset — no switch
- [ ] No `TogglePage()` calls remain for podcast navigation
- [ ] `make ci` passes

## Tasks

- [ ] Replace `TogglePage()` cycling in `SearchResultSelectedMsg` handler with `autoSwitchPreset()` calls
      - Modify `internal/app/handlers.go`: `if m.IsShow || m.IsEpisode` → `autoSwitchPreset("episode")`, else → `autoSwitchPreset("track")`
      - test: `TestSearchResult_TrackOnPodcastPreset_SwitchesToListening`, `TestSearchResult_EpisodeOnMusicPreset_SwitchesToPodcast`
- [ ] Add show selection handling: set `selectedShowID`, dispatch `FetchShowEpisodesRequestMsg`, call `autoSwitchPreset("episode")`, close search
      - Modify `internal/app/handlers.go`: when `m.IsShow`, set store state and dispatch commands
      - test: `TestSearchResult_ShowOnMusicPreset_SwitchesToPodcastAndLoadsEpisodes`, `TestSearchResult_ShowOnPodcastPreset_NoSwitchJustLoads`
- [ ] Remove all `TogglePage()` calls used for podcast navigation
      - Grep for `TogglePage()` in search-related handler code and remove
      - test: `TestSearchResult_NoTogglePageCalls`, `go build ./...`
- [ ] Run `make ci` — all lint, tests, and 80% coverage pass