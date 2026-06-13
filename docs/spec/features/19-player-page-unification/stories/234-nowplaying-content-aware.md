---
title: "NowPlaying: content-aware rendering (track + episode)"
feature: 19-player-page-unification
status: open
---

## Background

Currently `NowPlayingPane.View()` checks `ps.Item` (a `*Track`). When an episode
is playing, `PlaybackState.Item` is `nil` and `PlaybackState.Episode` is populated
(because `currently_playing_type` is `"episode"`). When type is `"episode"`,
NowPlaying renders its empty state — showing "Nothing playing" during podcast
playback. This is the critical bug that this story fixes.

The `PodcastPlaybackPane` (deleted in story 233) previously handled episode
rendering. NowPlaying must absorb that logic conditionally.

## Design

### Rendering switch

All rendering in `NowPlayingPane` is gated on `PlaybackState.CurrentlyPlayingType`:

```go
switch ps.CurrentlyPlayingType {
case "track":
    renderTrackInfo(ps.Item)  // existing behavior, unchanged
case "episode":
    renderEpisodeInfo(ps.Episode)
default:
    renderEmptyState()  // "Nothing playing" / "Press / to search"
}
```

### Track mode (unchanged)

- InfoBox title: `"Track Info"`
- Fields: Track name (bold, `TextPrimary()`), Artist (`TextSecondary()`), Album (`TextMuted()`)
- Right side: Visualizer + gradient seek bar
- Border title: `"Now Playing"`
- Controls: ⇄ ▷/⏸ ≡ ↻/↺

### Episode mode (new)

- InfoBox title: `"Episode Info"`
- Fields: Episode name (bold, `TextPrimary()`), Show name (`TextSecondary()`,
  replaces Artist), Release date (`TextMuted()`, replaces Album)
- Right side: Visualizer + gradient seek bar (**same** as track)
- Border title: `"Now Playing"` with `⏵ Podcast` notch in title bar
- Controls: identical to track (⇄ ▷/⏸ ≡ ↻/↺)
- Border notch: `[i:details]` in InfoBox border when episode is playing

### Empty state

- When `CurrentlyPlayingType` is `"ad"`, `"unknown"`, or nothing is playing:
  `"Nothing playing"` with hint `"Press / to search"`

### Stats page strip

The compact NowPlaying strip on the Stats page also adapts:
- Track: `"Now Playing ── Track · Artist ── ▶ 1:41/5:30"`
- Episode: `"Now Playing ── Episode · Show ── ▶ 12:34/45:00"`

### Implementation

Add `renderEpisodeInfo(episode *domain.Episode)` method to `NowPlayingPane`.
Modify `View()` and `Title()` to check `CurrentlyPlayingType`. Modify `Actions()`
to conditionally include `{Key: "i", Label: "details"}` when episode is playing.

## Files

### Modify

- `internal/ui/panes/nowplaying.go` — add `renderEpisodeInfo()`, modify `View()`,
  `Title()`, `Actions()` for content-awareness
- `internal/ui/panes/nowplaying_test.go` — add episode mode tests

## Acceptance Criteria

- [ ] NowPlaying renders track info when `currently_playing_type == "track"` (unchanged behavior)
- [ ] NowPlaying renders episode info (name, show, date) when `currently_playing_type == "episode"`
- [ ] NowPlaying shows `"Nothing playing"` / `"Press / to search"` for unknown/empty types
- [ ] `[i:details]` border notch appears when episode is playing
- [ ] `⏵ Podcast` appears in border title area when episode is playing
- [ ] Stats page compact strip shows episode info format
- [ ] Track mode tests still pass
- [ ] `make ci` passes

## Tasks

- [ ] Add `renderEpisodeInfo(episode *domain.Episode)` method to `NowPlayingPane`
      - Modify `internal/ui/panes/nowplaying.go`: add method rendering episode name (`TextPrimary()` bold), show name (`TextSecondary()`), release date (`TextMuted()`), visualizer + seek bar, `⏵ Podcast` border title
      - test: `TestNowPlayingPane_RenderEpisodeInfo_ShowsEpisodeFields`, `TestNowPlayingPane_RenderEpisodeInfo_ShowsPodcastNotch`
- [ ] Modify `View()` to switch on `CurrentlyPlayingType`
      - Modify `internal/ui/panes/nowplaying.go`: `switch ps.CurrentlyPlayingType` with `case "track"`, `case "episode"`, `default`
      - test: `TestNowPlayingPane_View_EpisodeMode`, `TestNowPlayingPane_View_TrackMode_Unchanged`, `TestNowPlayingPane_View_UnknownType_EmptyState`
- [ ] Modify `Title()` to reflect episode info when playing
      - Modify `internal/ui/panes/nowplaying.go`
      - test: `TestNowPlayingPane_Title_EpisodeMode`
- [ ] Modify `Actions()` to include `{Key: "i", Label: "details"}` when episode is playing
      - Modify `internal/ui/panes/nowplaying.go`
      - test: `TestNowPlayingPane_Actions_TrackMode_NoIDetails`, `TestNowPlayingPane_Actions_EpisodeMode_HasIDetails`
- [ ] Add `[i:details]` border notch in InfoBox when episode is playing
      - Modify episode rendering to include notch
      - test: `TestNowPlayingPane_EpisodeInfoBorderNotch`
- [ ] Update Stats page compact strip to show episode format
      - Modify the strip rendering (wherever the compact NowPlaying strip is built) to show `"Episode · Show"` when `CurrentlyPlayingType == "episode"`
      - test: `TestStatsStrip_EpisodeFormat`, `TestStatsStrip_TrackFormat`
- [ ] Run `make ci` — all lint, tests, and 80% coverage pass