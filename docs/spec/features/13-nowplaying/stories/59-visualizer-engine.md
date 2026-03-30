---
title: "Visualizer Engine"
feature: 13-nowplaying
status: done
---

## Background
The existing visualizer component rendered braille-dot patterns in a single color with a hard 4-line height cap and 3 animation patterns baked into a monolithic file. The NowPlaying redesign required per-row color gradients (green base, yellow mid, red peaks), block character rendering mode alongside braille, 7 animation patterns, no height cap, and an extensible Renderer interface. This story created the viz/ package as a clean extraction, to be integrated into NowPlaying by Feature 60.

## Design

### Frame and StyledLine Types
`StyledLine` struct (Text string, Color lipgloss.Color). `Frame` type (slice of StyledLine).

### Renderer Interface
Two implementations: `BrailleRenderer` (braille dots U+2800-U+28FF, 2x4 dot matrix) and `BlockRenderer` (block chars U+2588/space, 1 char wide per column).

### 7 Animation Patterns
- 0: Braille Dual Sine Wave
- 1: Braille Standing Wave
- 2: Braille Pulse/Ripple
- 3: Block Dense Equalizer
- 4: Block Waveform/Sine
- 5: Block Sparse/Low Amplitude
- 6: Braille Mid-density Organic

All height functions use deterministic math (sine, cosine, Gaussian) -- no math/rand. Frame count is 40 per pattern.

### Engine
Constructor `NewEngine(th theme.Theme)` initializes with Patterns() list, pattern 0, 200ms interval. `SetSize(width, height)` regenerates 40 precomputed frames with per-row colors (top 1/3 Gradient3, middle 1/3 Gradient2, bottom 1/3 Gradient1). `CyclePattern()` advances patternIdx and regenerates. `CurrentFrame()` returns blank when paused.

## Acceptance Criteria
- [ ] viz/ package compiles independently with no imports from components/ or panes/
- [ ] Renderer interface has two implementations
- [ ] 7 patterns registered: 4 braille, 3 block
- [ ] Per-row color gradient uses Gradient1/2/3 theme tokens
- [ ] Engine has no height cap
- [ ] Frame table precomputed (40 frames) on SetSize() and CyclePattern()
- [ ] CurrentFrame() returns blank frame when paused
- [ ] CyclePattern() cycles through all 7 patterns and wraps
- [ ] TickMsg type defined in viz package
- [ ] All height functions deterministic
- [ ] 80%+ test coverage on viz/
- [ ] make ci passes

## Tasks
- [ ] Create Frame and StyledLine types in internal/ui/components/viz/frame.go
      - test: StyledLine stores text/color; Frame can be created and indexed
- [ ] Create Renderer interface and braille renderer in viz/braille.go
      - test: compile-time check; correct StyledLines; braille runes only; colors match; edge cases
- [ ] Create block renderer in viz/block.go
      - test: compile-time check; correct StyledLines; block chars only; colors match; edge cases
- [ ] Create Pattern type and all 7 pattern definitions in viz/pattern.go
      - test: exactly 7 patterns; non-empty names; correct renderer types; HeightFunc returns valid slices; deterministic; different patterns differ
- [ ] Create Engine with frame orchestration in viz/engine.go
      - test: 7 patterns; SetSize produces correct frames; Advance increments; CurrentFrame blank when paused; wrap after 40; CyclePattern regenerates; per-row colors; edge cases
- [ ] Comprehensive engine tests across all 7 patterns and both renderers
      - test: per-pattern frame dimensions, content types, colors, determinism; integration lifecycle; resize mid-animation; edge dimensions
