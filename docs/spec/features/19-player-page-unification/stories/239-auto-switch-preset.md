---
title: "Auto-switch preset on user-initiated playback"
feature: 19-player-page-unification
status: done
---

## Background

When the user initiates playback of content that doesn't match the current preset
type, the app should automatically switch to an appropriate preset. Background
playback changes (from polling) must NOT trigger preset switches.

## Design

### Auto-switch rules

Auto-switch only fires on **user-initiated** playback commands. Background
polling that detects a content-type change updates NowPlaying's display but
**never** changes the preset.

| Trigger | Current Preset Type | Behavior |
|---------|-------------------|----------|
| User plays a track | Music preset (0-4) | No switch |
| User plays a track | Podcast preset (2, 5) | Switch to Listening (1) |
| User plays an episode | Podcast preset (2, 5) | No switch |
| User plays an episode | Music preset (0, 1, 3, 4) | Switch to Podcast (2) |
| Background playback change | Any | Update display only. No preset switch |

The key principle: auto-switch only fires when the **content type changes**
relative to the preset's orientation. If already on a matching preset type, no
switch occurs.

### Implementation

```go
func (a *App) autoSwitchPreset(forContentType string) {
    current := a.layout.ActivePresetIndex()
    isPodcastPreset := a.isCurrentPresetPodcastOriented()
    
    if forContentType == "track" && isPodcastPreset {
        a.layout.SetPreset(1) // Listening
        a.propagateSizes()
        a.syncFocus()
    } else if forContentType == "episode" && !isPodcastPreset {
        a.layout.SetPreset(2) // Podcast
        a.propagateSizes()
        a.syncFocus()
    }
    // Otherwise: content matches preset type, no switch
}

func (a *App) isCurrentPresetPodcastOriented() bool {
    preset := a.layout.ActivePreset()
    return preset.Visible[layout.PaneFollowedShows]
}
```

A preset is "podcast-oriented" if `PaneFollowedShows` is in its Visible map.
Presets 2 (Podcast) and 5 (Podcast Dashboard) contain `PaneFollowedShows`.
Presets 0, 1, 3, 4 do not.

### Call sites

1. `PlayTrackMsg` handler → `autoSwitchPreset("track")`
2. `PlayTrackListMsg` handler → `autoSwitchPreset("track")`
3. `PlayEpisodeMsg` handler → `autoSwitchPreset("episode")`
4. `PlayContextMsg` handler → determine content type from context URI (`spotify:show:`
   prefix = episode, otherwise track) → `autoSwitchPreset()`

**Excluded**: `PlaybackStateFetchedMsg` handler (tick-based polling) must NOT call
`autoSwitchPreset()`.

### SetPreset method

`layout.Manager.SetPreset(index int)` was added in story 233. This story uses it
for the auto-switch mechanism. When `SetPreset` is called, it:
1. Sets `activePreset[activePage]` to the given index
2. Resets `hidden` map
3. Resets `focusIndex` to 0
4. Recomputes layout
5. Returns non-nil tea.Cmd for size propagation (handled by `propagateSizes()`)

### Preset switch side effects

After `SetPreset`, the app must:
1. Propagate sizes to all newly visible panes
2. Sync focus to the first visible pane
3. Check staleness for newly visible panes and dispatch fetch commands
   (covered by the polling optimization in story 240)

## Files

### Modify

- `internal/app/handlers.go` — add `autoSwitchPreset()` and
  `isCurrentPresetPodcastOriented()` methods; call from play message handlers
- `internal/app/routing.go` — no changes needed (auto-switch is handler-driven)

## Acceptance Criteria

- [ ] Playing a track while on a podcast preset switches to Listening (1)
- [ ] Playing an episode while on a music preset switches to Podcast (2)
- [ ] Playing a track while on a music preset — no switch
- [ ] Playing an episode while on a podcast preset — no switch
- [ ] Background `PlaybackStateFetchedMsg` does NOT trigger auto-switch
- [ ] Auto-switch calls `SetPreset()` directly (doesn't cycle)
- [ ] User can override auto-switch with manual `p` key at any time
- [ ] `make ci` passes

## Tasks

- [ ] Implement `autoSwitchPreset(forContentType string)` method on `App`
      - Modify `internal/app/handlers.go`: method that checks `isCurrentPresetPodcastOriented()` and calls `SetPreset(1)` or `SetPreset(2)` based on content type
      - test: `TestAutoSwitchPreset_TrackOnPodcastPreset_SwitchesToListening`, `TestAutoSwitchPreset_EpisodeOnMusicPreset_SwitchesToPodcast`, `TestAutoSwitchPreset_TrackOnMusicPreset_NoSwitch`, `TestAutoSwitchPreset_EpisodeOnPodcastPreset_NoSwitch`
- [ ] Implement `isCurrentPresetPodcastOriented()` helper
      - Modify `internal/app/handlers.go`: checks if `PaneFollowedShows` is in active preset's Visible map
      - test: `TestIsCurrentPresetPodcastOriented_PodcastPreset`, `TestIsCurrentPresetPodcastOriented_MusicPreset`
- [ ] Call `autoSwitchPreset("track")` from `PlayTrackMsg` and `PlayTrackListMsg` handlers
      - Modify `internal/app/handlers.go`: add auto-switch call after existing play logic
      - test: `TestPlayTrackMsg_AutoSwitches_WhenOnPodcastPreset`
- [ ] Call `autoSwitchPreset("episode")` from `PlayEpisodeMsg` handler
      - Modify `internal/app/handlers.go`: add auto-switch call after existing play logic
      - test: `TestPlayEpisodeMsg_AutoSwitches_WhenOnMusicPreset`
- [ ] Call `autoSwitchPreset()` from `PlayContextMsg` handler based on URI prefix
      - Modify `internal/app/handlers.go`: parse `spotify:show:` prefix → `"episode"`, else → `"track"`
      - test: `TestPlayContextMsg_ShowURI_AutoSwitchesToPodcast`, `TestPlayContextMsg_AlbumURI_AutoSwitchesToListening`
- [ ] Verify `PlaybackStateFetchedMsg` handler does NOT call `autoSwitchPreset`
      - test: `TestPlaybackStateFetchedMsg_NoAutoSwitch`
- [ ] Run `make ci` — all lint, tests, and 80% coverage pass