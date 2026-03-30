---
title: "NowPlaying Design Docs Update"
feature: 13-nowplaying
status: done
---

## Background
The design documentation needed updating to reflect the new btop-inspired NowPlaying pane split layout, the 3 animation patterns and `v` key cycling behavior, and the responsive proportions of the InfoBox/Visualizer split. All preset diagrams in DESIGN.md were updated to show the new rendering style.

## Design

### DESIGN.md Updates
- Section 2 (Pane Definitions): add split layout note
- Section 4 (Presets): update all preset diagrams (0-3) to show InfoBox + Visualizer side-by-side
- Section 11 (Visual Components): document 3 animation patterns (Dual Sine Wave, Standing Wave, Pulse/Ripple) with `v` key cycling
- New "NowPlaying Split Layout" subsection in Section 11:
  - Layout proportions: Left ~1/4 width (min 28 chars) InfoBox with rounded border
  - Right ~3/4 width animated braille visualizer
  - Bottom full-width 1-line gradient seek bar with time labels
  - Responsive: `infoWidth = max(contentWidth/4, 28)`, `vizWidth = contentWidth - infoWidth - 1`
  - Height < 8 embeds track info in title bar
  - Pattern state local to pane (not in Store), `v` key routes via `isPlaybackKey()`

## Acceptance Criteria
- [ ] DESIGN.md Section 2 updated with NowPlaying split layout note
- [ ] DESIGN.md Section 4 -- all preset diagrams updated for split layout
- [ ] DESIGN.md Section 11 documents 3 animation patterns and v key cycling
- [ ] DESIGN.md Section 11 new "NowPlaying Split Layout" subsection
- [ ] All preset diagrams (0, 1, 2, 3) show the new NowPlaying rendering style
- [ ] make ci passes

## Tasks
- [ ] Update DESIGN.md with NowPlaying split layout documentation -- sections 2, 4, 11
      - test: docs change, no code tests needed, make ci passes
