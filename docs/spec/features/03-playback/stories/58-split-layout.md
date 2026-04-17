---
title: "NowPlaying Split Layout"
feature: 13-nowplaying
status: done
---

## Background
The NowPlayingPane's vertical stack layout was replaced with a btop-inspired horizontal split: an InfoBox sub-pane on the left (~1/4 width) containing track info, controls, and volume bar, and the braille visualizer on the right (~3/4 width), with a gradient seek bar spanning full width at the bottom. The `compact` boolean field and `renderCompact()` method were removed in favor of a height < 8 check in `Title()` for inline track info. Several compact-mode helper functions were also cleaned up.

## Design

### Split Layout
Add `infoBox *components.InfoBox` field initialized via `components.NewInfoBox(t)`. Update SetSize() to compute split dimensions: `infoWidth = paneMax(contentWidth/4, 28)`, `vizWidth = contentWidth - infoWidth - 1`, `bodyHeight = paneMax(height-4, 4) - progressHeight`.

### Removed Code
Remove `compact bool` field, delete `renderCompact()` method, delete `interpolateHexCompact()`, `parseHexParts()`, `lerpByte()` helpers.

### View() Rewrite
Render InfoBox left (track name, artist, album, controls, volume bar) and Visualizer right via `lipgloss.JoinHorizontal`, seek bar at bottom via `lipgloss.JoinVertical`. Update `Title()` to use `height < 8` check instead of compact flag.

## Acceptance Criteria
- [ ] NowPlayingPane has infoBox field initialized in constructor
- [ ] SetSize computes split dimensions for InfoBox, Visualizer, seek bar, volume bar
- [ ] compact field and renderCompact() removed
- [ ] interpolateHexCompact, parseHexParts, lerpByte removed
- [ ] View() renders horizontal split: InfoBox left, Visualizer right, seek bar bottom
- [ ] Title() uses height < 8 check instead of compact flag
- [ ] All old compact-mode tests deleted, new split layout tests added
- [ ] make ci passes

## Tasks
- [ ] Rewrite NowPlayingPane View() with split layout -- add infoBox, compute split dimensions, remove compact code, rewrite View(), update Title()
      - test: DELETE old compact tests; ADD split layout tests: contains InfoBox borders, braille, seek bar, volume, controls; Title at height 6/24; adapts to different sizes
