---
title: "NowPlaying component + playback integration tests"
feature: 21-test-infrastructure
status: done
---

## Background

NowPlayingPane is the most complex pane — track mode, episode mode, adaptive layout (InfoBox
overlay, narrow fallback, compact preset strip), seek bar, volume bar, visualizer animation,
and playback controls. Current tests cover Update() state changes but never assert what the
user sees at each layout tier. Playback control integration (Space play/pause, ←/→ seek,
Shift+←/→ skip, s/r/+/-/v) is tested at the message routing level but not end-to-end with
rendered output.

## Design

### Golden tests: `internal/ui/panes/nowplaying_golden_test.go`

**Track mode (80×24, normal layout):**
- `TestNowPlayingPane_View_TrackPlaying` — `currently_playing_type == "track"`, is_playing=true, InfoBox shows track name/artist/album
- `TestNowPlayingPane_View_TrackPaused` — is_playing=false, visualizer stopped, pause glyph in title
- `TestNowPlayingPane_View_TrackNoData` — empty state

**Episode mode (80×24):**
- `TestNowPlayingPane_View_EpisodePlaying` — episode name, show name, podcast notch on InfoBox border
- `TestNowPlayingPane_View_EpisodePaused`

**Adaptive layout:**
- `TestNowPlayingPane_View_CompactStrip` — height < 8, track info embedded in title, controls visible, no InfoBox
- `TestNowPlayingPane_View_NarrowFallback` — 40 cols, InfoBox dropped, visualizer fills full area
- `TestNowPlayingPane_View_Wide` — 120 cols, InfoBox ~25% left, seek bar right

**Seek bar and volume:**
- `TestNowPlayingPane_View_SeekBar_AtPosition` — seek bar at 30% progress
- `TestNowPlayingPane_View_VolumeBar` — volume bar at 65%

**Edge case types:**
- `TestNowPlayingPane_View_AdType_EmptyState` — `currently_playing_type == "ad"`, renders empty state
- `TestNowPlayingPane_View_UnknownType_EmptyState` — `currently_playing_type == "unknown"`, renders empty state

### Integration test: `internal/app/playback_flow_test.go`

```go
func TestPlaybackFlow_PauseThenResume(t *testing.T) {
    // 1. Create app with mock PlayerAPI, send PlaybackStateFetchedMsg{IsPlaying: true}
    // 2. Send Space key → assert cmd is PlaybackRequestMsg{Action: ActionPause}
    // 3. Send PlaybackCmdSentMsg{} → assert visualizer stopped
    // 4. Send Space key → assert ActionPlay
}

func TestPlaybackFlow_SeekRight(t *testing.T) {
    // Send → key → assert SeekIntentMsg produced with target +5s
    // Multiple rapid → keys → only final seek sent (debounce)
}

func TestPlaybackFlow_ShiftRight_NextTrack(t *testing.T) {
    // Send Shift+→ → assert PlaybackRequestMsg{Action: ActionNext}
}

func TestPlaybackFlow_CycleRepeat(t *testing.T) {
    // Send 'r' three times → off→context→track→off
    // Assert View() shows correct repeat glyph each cycle
}
```

## Files

### Create

- `internal/ui/panes/nowplaying_golden_test.go` — 12 snapshots
- `internal/app/playback_flow_test.go` — playback integration flows
- `internal/ui/panes/testdata/TestNowPlayingPane_View_*.golden` (12 files)

## Acceptance Criteria

- [ ] NowPlaying golden snapshots: track playing, track paused, empty, episode playing, episode paused, compact strip, narrow fallback, wide, seek bar, volume bar, ad_type empty, unknown_type empty
- [ ] Integration: Space → pause → resume cycle with correct cmd types
- [ ] Integration: ←/→ seek with debounce (only final seek sent)
- [ ] Integration: Shift+←/Shift+→ → ActionPrev/ActionNext
- [ ] Integration: 'r' cycles repeat modes, View() shows correct glyph
- [ ] Integration: 's' toggles shuffle
- [ ] Integration: '+/-' volume intent messages with correct sequence numbers
- [ ] `make ci` passes

## Tasks

- [ ] Create NowPlaying golden tests — track mode (3 snapshots)
      - test: `TestNowPlayingPane_View_TrackPlaying`, `TestNowPlayingPane_View_TrackPaused`, `TestNowPlayingPane_View_TrackNoData`
- [ ] Create NowPlaying golden tests — episode mode (2 snapshots)
      - test: `TestNowPlayingPane_View_EpisodePlaying`, `TestNowPlayingPane_View_EpisodePaused`
- [ ] Create NowPlaying golden tests — adaptive layout (3 snapshots)
      - test: `TestNowPlayingPane_View_CompactStrip`, `TestNowPlayingPane_View_NarrowFallback`, `TestNowPlayingPane_View_Wide`
- [ ] Create NowPlaying golden tests — seek + volume (2 snapshots)
      - test: `TestNowPlayingPane_View_SeekBar_AtPosition`, `TestNowPlayingPane_View_VolumeBar`
- [ ] Create NowPlaying golden tests — edge case types (2 snapshots)
      - test: `TestNowPlayingPane_View_AdType_EmptyState`, `TestNowPlayingPane_View_UnknownType_EmptyState`
- [ ] Create playback flow integration tests
      - test: `TestPlaybackFlow_PauseThenResume`, `TestPlaybackFlow_SeekRight`, `TestPlaybackFlow_ShiftRight_NextTrack`, `TestPlaybackFlow_CycleRepeat`
- [ ] Generate golden files and verify all tests pass
