---
title: "Album Art & Responsive NowPlaying"
status: in-progress
---

## Description

Originally shipped pixelated album cover art inside the NowPlaying pane via
the `eliukblau/pixterm` library, with three responsive layout tiers later
collapsed into a single-formula layout (stories 214–220).

After live testing, the album art column was judged to add visual noise
without conveying meaningful information at terminal cell sizes. Stories 221
and 222 reverse course: album art is removed entirely, the visualizer is
expanded to fill the full pane content area, and the InfoBox is repositioned
as a true overlay on the left ~25% of the visualizer background.

| Phase | Stories | Outcome |
|-------|---------|---------|
| Build  | 214–220 | Album art renderer, responsive tiers, single-formula layout |
| Redesign | 221, 222 | OverlayBackground token + InfoBox fill; remove album art, overlay InfoBox on viz |

The NowPlaying pane keeps its dimensions, responsive behaviour, surrounding
chrome, and InfoBox content (track name, artists, album, controls, volume).
Only the internal composition and the album-art subsystem change.

## Acceptance Criteria

Phase 1 (stories 214–220, shipped):

- [x] `domain.Album` carries `Images []AlbumImage`; Spotify `/me/player` JSON `album.images` mapped into domain types
- [x] `BestImage()` helper on `Album` returns the smallest usable image URL
- [x] LayoutManager honours `Row.MinHeight`
- [x] Album art fetched on `Init()` and on track change
- [x] Single-formula layout replaces 3-tier system

Phase 2 (stories 221–222, this redesign):

- [ ] `Theme.OverlayBackground() lipgloss.Color` exists; all 11 themes return their own `Base()`
- [ ] `InfoBox.Render` applies a solid `OverlayBackground` fill to its interior
- [ ] `internal/ui/components/albumart.go` deleted; `pixterm` removed from `go.mod`
- [ ] `AlbumArtRenderer`, `FetchAlbumArtCmd`, `AlbumArtFetchedMsg` removed from the codebase
- [ ] `NowPlayingPane` has no album-art fields or methods (`artRenderer`, `imageCols`, `renderImageBlock`, `ArtHasImage`)
- [ ] Visualizer fills the full content area
- [ ] InfoBox overlays the left ~25% with a solid background; seek bar lives only on the right
- [ ] Equal 1-row padding top and bottom
- [ ] Narrow-terminal fallback: when `vizWidth < npMinViz`, InfoBox drops and viz fills the full content area
- [ ] All album-art tests removed; overlay layout tests added
- [ ] `make ci` passes
