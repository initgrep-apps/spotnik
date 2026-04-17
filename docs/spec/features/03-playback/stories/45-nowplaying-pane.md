---
title: "NowPlaying Pane"
feature: 13-nowplaying
status: done
---

## Background
The original PlayerPane rendered the currently playing track with transport controls, a monochrome seek bar, and volume bar. It did not implement the layout.Pane interface and had no visualizer or gradient effects. This story renamed the pane to NowPlayingPane, implemented the layout.Pane interface, embedded the braille visualizer and gradient bars, and added compact mode for presets where NowPlaying occupies a small row.

## Design

### Rename and Interface
Rename files, struct, constructor, and all references from PlayerPane to NowPlayingPane. Implement layout.Pane: ID() returns layout.PaneNowPlaying, Title() returns "Now Playing", ToggleKey() returns 1, Actions() returns shuffle + repeat.

### Visualizer Embedding
Add `visualizer *components.Visualizer` field, handle VisualizerTickMsg in Update(), set playing state from PlaybackStateFetchedMsg, compute vizHeight based on mode.

### Gradient Bars
Replace ProgressBar with GradientSeekBar, replace VolumeBar with GradientVolumeBar.

### Compact Mode
Detect in SetSize() when height <= 3. Render single content line with controls + volume. Override Title() to include track name and progress when compact. No visualizer in compact mode.

## Acceptance Criteria
- [ ] PlayerPane renamed to NowPlayingPane everywhere
- [ ] NowPlayingPane satisfies layout.Pane interface
- [ ] Braille visualizer animates when playing, stops when paused
- [ ] Gradient seek bar shows color transition
- [ ] Volume bar shows correct color band
- [ ] Compact mode renders single-line strip for small height
- [ ] Dynamic title in compact mode shows track + progress
- [ ] All playback keys still work
- [ ] make ci passes

## Tasks
- [ ] Rename PlayerPane to NowPlayingPane -- files, struct, constructor, all references
      - test: existing player tests pass under new name, build succeeds
- [ ] Implement layout.Pane interface -- ID, Title, ToggleKey, Actions, SetSize, SetFocused, IsFocused
      - test: unit tests for each method; compile-time check var _ layout.Pane = &NowPlayingPane{}
- [ ] Embed visualizer component -- add field, handle VisualizerTickMsg, set playing state
      - test: visualizer ticks after Init(); PlaybackStateFetchedMsg controls animation; braille chars in full mode View()
- [ ] Replace bars with gradient versions -- GradientSeekBar and GradientVolumeBar
      - test: gradient colors; resize correctly; correct API
- [ ] Implement compact mode -- detect height <= 3, single line, title override, no visualizer
      - test: SetSize(80, 3) enables compact; SetSize(80, 10) disables; compact Title includes track info; no visualizer in compact
- [ ] Comprehensive tests -- full lifecycle, mode transitions, playing/paused, shuffle/repeat, nil state
      - test: integration tests for all states and edge cases
