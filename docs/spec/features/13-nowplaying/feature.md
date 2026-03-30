---
title: "NowPlaying Pane"
status: done
---

## Description
A rich terminal music player pane with btop-inspired split layout, braille and block character visualizer engine with per-row color gradients, Unicode transport controls, and adaptive rendering from compact strips to expanded centered layouts.

The NowPlaying pane is the centerpiece of Spotnik's terminal UI, displaying the currently playing track with transport controls, audio visualization, and playback progress. It evolved from a simple PlayerPane with monochrome seek and volume bars into a sophisticated btop-inspired split layout featuring a Track Info sub-pane on the left and an animated braille/block visualizer on the right, with a gradient seek bar embedded between visualization rows in the right panel.

The visualizer system was extracted into a dedicated viz/ package providing a Renderer interface with two implementations (braille and block characters), 7 animation patterns (4 braille, 3 block), per-row color gradients using theme tokens (Gradient1/2/3), and a precomputed frame table for smooth 200ms tick animation.

## Acceptance Criteria
- [ ] PlayerPane renamed to NowPlayingPane everywhere
- [ ] NowPlayingPane satisfies layout.Pane interface
- [ ] viz/ package compiles independently with no imports from components/ or panes/
- [ ] Renderer interface has two implementations: BrailleRenderer and BlockRenderer
- [ ] 7 patterns registered: 4 braille, 3 block
- [ ] Controls row shows Unicode glyphs
- [ ] Volume bar shows discrete block characters with music note icon
- [ ] Seek bar is inside the right panel between top and bottom viz rows
- [ ] Expanded mode: content block vertically centered via lipgloss.Place
- [ ] DESIGN.md updated with split layout documentation
- [ ] make ci passes
