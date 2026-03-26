---
name: project_spotnik_feature45_complete
description: Feature 45 (NowPlayingPane): rename PlayerPane, layout.Pane interface, Visualizer embed, GradientSeekBar/VolumeBar, compact mode, 50 tests
type: project
---

## Feature 45 — NowPlayingPane

**What was built:**
- `internal/ui/panes/nowplaying.go` — NowPlayingPane replacing PlayerPane (445 lines)
  - Implements `layout.Pane` interface with ID/Title/ToggleKey/Actions
  - Embeds `*components.Visualizer`, `*components.GradientSeekBar`, `*components.GradientVolumeBar`
  - Compact mode: `height <= 3` triggers single-line strip rendering
  - Dynamic `Title()` in compact mode: includes track·artist + elapsed/total
  - Constructor initializes `localProgressMs` and `visualizer.SetPlaying()` from store state
- `internal/ui/panes/nowplaying_test.go` — 50 tests (662 lines)
- Deleted: `player.go`, `player_test.go`
- Modified: `app.go`, `routing.go` — `PlayerPane` → `NowPlayingPane`, `NewPlayerPane` → `NewNowPlayingPane`

**Key files:**
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/ui/panes/nowplaying.go` — full implementation

**Patterns established:**
- Constructor initializes pane-local state from store immediately (no need to send PlaybackStateFetchedMsg first)
- `handlePlaybackFetched()` syncs both `localProgressMs` AND `visualizer.SetPlaying()` atomically
- `SetSize()` determines compact mode: `p.compact = height <= 3`
- Visualizer heights: compact=1, full=2, expanded (height≥10)=4
- `Init()` now returns `p.visualizer.Init()` instead of nil — wires up the tick loop
- `Update()` handles `components.VisualizerTickMsg` and forwards to `visualizer.Update(msg)`

**Code duplication note:**
- `interpolateHexCompact`, `parseHexParts`, `lerpByte` duplicate unexported functions from `components/gradient.go`
- Necessary because compact seek bar needs a label-free fill bar that `GradientSeekBar.Render()` doesn't produce
- Tagged TODO(feature-53) for cleanup

**Rename approach:**
- `sed -i '' 's/panes\.PlayerPane/panes.NowPlayingPane/g; s/panes\.NewPlayerPane/panes.NewNowPlayingPane/g'` worked cleanly for app.go and routing.go
- `paneMax` and `emitPlaybackRequest` were in player.go — must keep them in nowplaying.go since library.go also uses `paneMax` from the same package

**Gotchas:**
- `paneMax` and `emitPlaybackRequest` were private helpers in player.go used by library.go too. Moving to nowplaying.go keeps them in the same `panes` package — no import changes needed.
- Copying player_test.go first then rewriting it entirely was cleaner than editing in place.
- Constructor must call `s.PlaybackState()` to initialize `localProgressMs` — without this, test helpers that set state before constructing a pane would see localProgressMs=0.
- `VisualizerTickMsg` is `type VisualizerTickMsg time.Time` — `components.VisualizerTickMsg{}` is the zero value (valid).

**Testing notes:**
- 50 tests covering: all key bindings, layout.Pane interface methods, visualizer play/pause/tick, compact mode enable/disable thresholds, full/compact transitions, nil state, zero duration, braille char presence
- panes coverage: 87.6%
- Overall coverage: 85.1%
- The `filterNonEmpty` and `splitLines` helpers are defined inline in the test file (same package)
