---
title: "Visualizer + Gradient Bars"
feature: 12-layout
status: done
---

## Background
The current player uses monochrome bars: ProgressBar with SeekBar() color and VolumeBar with VolumeBar() color. The new DESIGN.md specifies a braille-dot audio visualizer that animates when music plays using Unicode braille characters (U+2800-U+28FF) with a precomputed frame table, a gradient seek bar where fill transitions from Gradient1() to Gradient2() left-to-right, and a volume bar with color bands: green (0-33%), yellow (34-66%), red (67-100%). These components are embedded in the NowPlaying pane.

Design reference: docs/DESIGN.md section 11.

## Design

### Braille-Dot Visualizer
`VisualizerTickMsg time.Time`. Frame table: 40 frames generated deterministically using sine waves with phase offsets. Each frame is []string (one per line of height). Braille chars encode 2x4 dot matrix. Regenerated on SetSize() width change. Color: VisualizerFg() token. Animation: on VisualizerTickMsg, if playing increment frameIndex (wrap at len), re-arm tick at 200ms.

### Gradient Seek Bar
Fill ratio calculation, linear RGB interpolation between Gradient1() and Gradient2() per filled position. Format: `"1:41  ████████████████░░░░░░░░░░░░░░  5:30"`. Helper: `interpolateHex(hex1, hex2 string, t float64) lipgloss.Color`.

### Gradient Volume Bar
Color bands: 0-33% Gradient1() (green/cool), 34-66% Gradient2() (yellow/warm), 67-100% Gradient3() (red/hot). Format: `"VOL  ████████░░░░░░  65%"`. Clamp volume to [0, 100].

## Acceptance Criteria
- [ ] Visualizer animates braille characters on 200ms tick when playing
- [ ] Visualizer shows flat-line when paused
- [ ] Frame table has 40 deterministic patterns
- [ ] Visualizer adapts to width/height via SetSize()
- [ ] Gradient seek bar interpolates color from Gradient1() to Gradient2()
- [ ] Volume bar uses 3 color bands
- [ ] All colors come from Theme interface tokens
- [ ] interpolateHex correctly handles RGB color interpolation
- [ ] No panics on edge cases
- [ ] `make ci` passes

## Tasks
- [ ] Create braille-dot visualizer component in internal/ui/components/visualizer.go
      - test: starts frameIndex=0; playing+tick increments; paused stays; View braille when playing; paused flat-line; frame wraps; SetSize changes output; deterministic; Init returns tick
- [ ] Create gradient seek bar component in internal/ui/components/gradient.go
      - test: 0/50/100% fill; Gradient1/2 colors; time labels; width changes; durationMs=0 safe
- [ ] Create gradient volume bar component in internal/ui/components/gradient.go
      - test: 0/25/50/80/100% volumes; correct color bands; clamping; width changes
- [ ] Integration tests for all visual components
      - test: visualizer lifecycle; play/pause cycle; seek bar gradient visible; volume threshold transitions; width containment; theme tokens only
