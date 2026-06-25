---
title: "Podcast component + integration tests"
feature: 21-test-infrastructure
status: open
---

## Background

The Podcast feature (18) provides FollowedShows pane (show list with Enter drill-down to
episodes), SavedEpisodes pane (episode list with Enter-to-play), and Episode Details
overlay (`i` key when episode playing). Current tests cover Update() handlers but not
rendered output or multi-step flows like show→Enter→episodes→Esc→shows.

## Design

### Golden tests: `internal/ui/panes/followedshows_golden_test.go`

- `TestFollowedShowsPane_View_Shows` — 3 shows loaded, 80×24
- `TestFollowedShowsPane_View_EmptyState` — no followed shows
- `TestFollowedShowsPane_View_EpisodeSubView` — Enter on show, episodes shown, title updated
- `TestFollowedShowsPane_View_Narrow` — 40×24

### Golden tests: `internal/ui/panes/savedepisodes_golden_test.go`

- `TestSavedEpisodesPane_View_Episodes` — 5 episodes loaded, 80×24
- `TestSavedEpisodesPane_View_EmptyState` — no episodes
- `TestSavedEpisodesPane_View_Narrow` — 40×24

### Golden tests: `internal/ui/panes/episode_details_golden_test.go`

- `TestEpisodeDetailsOverlay_View_EpisodeInfo` — overlay open, episode name, show name, description, duration
- `TestEpisodeDetailsOverlay_View_Narrow` — 40×24

### Integration test: `internal/app/podcast_flow_test.go`

```go
func TestPodcastFlow_FollowedShowsDrillDown(t *testing.T) {
    // 1. FollowedShowsPane with 3 shows, cursor on show 0
    // 2. Send Enter → assert episode sub-view in View(), show name in title
    // 3. Send Esc → assert show list restored
    // 4. Send Enter on different show → assert different episodes shown
}

func TestPodcastFlow_EpisodeDetailsOverlay(t *testing.T) {
    // 1. NowPlaying episode active: currently_playing_type == "episode"
    // 2. Send 'i' → assert EpisodeDetailsOverlay visible in View()
    // 3. Assert overlay shows episode description, show name, duration
    // 4. Send Esc → overlay closed
}

func TestPodcastFlow_EnterPlaysEpisode(t *testing.T) {
    // SavedEpisodes: cursor on episode, Enter → assert PlayEpisodeMsg cmd
    // FollowedShows episode sub-view: Enter on episode → assert PlayEpisodeMsg cmd
}
```

## Files

### Create

- `internal/ui/panes/followedshows_golden_test.go`
- `internal/ui/panes/savedepisodes_golden_test.go`
- `internal/ui/panes/episode_details_golden_test.go`
- `internal/app/podcast_flow_test.go`
- `internal/ui/panes/testdata/TestFollowedShowsPane_View_*.golden` (4 files)
- `internal/ui/panes/testdata/TestSavedEpisodesPane_View_*.golden` (3 files)
- `internal/ui/panes/testdata/TestEpisodeDetailsOverlay_View_*.golden` (2 files)

## Acceptance Criteria

- [ ] FollowedShowsPane: 4 golden snapshots (shows, empty, episode sub-view, narrow)
- [ ] SavedEpisodesPane: 3 golden snapshots (episodes, empty, narrow)
- [ ] EpisodeDetailsOverlay: 2 golden snapshots (normal, narrow)
- [ ] Integration: drill-down Enter→episodes→Esc→shows flow
- [ ] Integration: 'i' opens Episode Details when episode playing, no-op for tracks
- [ ] Integration: Enter on episode produces PlayEpisodeMsg
- [ ] `make ci` passes

## Tasks

- [ ] Create FollowedShowsPane golden tests (4 snapshots)
      - test: `TestFollowedShowsPane_View_Shows`, `TestFollowedShowsPane_View_EmptyState`, `TestFollowedShowsPane_View_EpisodeSubView`, `TestFollowedShowsPane_View_Narrow`
- [ ] Create SavedEpisodesPane golden tests (3 snapshots)
      - test: `TestSavedEpisodesPane_View_Episodes`, `TestSavedEpisodesPane_View_EmptyState`, `TestSavedEpisodesPane_View_Narrow`
- [ ] Create EpisodeDetailsOverlay golden tests (2 snapshots)
      - test: `TestEpisodeDetailsOverlay_View_EpisodeInfo`, `TestEpisodeDetailsOverlay_View_Narrow`
- [ ] Create podcast integration flow tests
      - test: `TestPodcastFlow_FollowedShowsDrillDown`, `TestPodcastFlow_EpisodeDetailsOverlay`, `TestPodcastFlow_EnterPlaysEpisode`
- [ ] Generate golden files and verify all tests pass
