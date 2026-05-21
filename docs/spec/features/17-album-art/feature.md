---
title: "Album Art & Responsive NowPlaying"
status: open
---

## Description

Renders pixelated album cover art inside the NowPlaying pane using the
`eliukblau/pixterm` library. Album images arrive from Spotify's `/me/player`
endpoint — already present in the JSON but previously discarded on unmarshal.

The NowPlaying pane gains three responsive layout tiers that adapt to available
terminal height so the image always appears as a square:

| Tier | bodyHeight | Layout |
|---|---|---|
| Base | < 16 | 3-col: image · info · viz |
| Mid | 16–28 | 2-col: [image above, info below] · viz |
| Full | > 28 | 2-col: [larger image above, richer info below] · viz |

The Stats page NowPlaying row receives a `MinHeight` guarantee via a new
LayoutManager primitive so it always reaches at least the base tier regardless
of terminal size.

## Acceptance Criteria

- [ ] `domain.Album` carries `Images []AlbumImage`; Spotify `/me/player` JSON `album.images` is mapped into domain types
- [ ] `BestImage()` helper on `Album` returns the smallest usable image URL
- [ ] LayoutManager honours `Row.MinHeight`; Stats page NowPlaying gets ≥ 14 rows at any terminal size
- [ ] Album art is fetched asynchronously on `Init()` (if playback already active) and on every track change detected via `PlaybackStateFetchedMsg`
- [ ] Re-poll of the same track does not re-fetch (track ID cache)
- [ ] NowPlaying enters base tier (3-col) when bodyHeight < 16
- [ ] NowPlaying enters mid tier (2-col stacked) when bodyHeight 16–28
- [ ] NowPlaying enters full tier (2-col, larger image + richer info) when bodyHeight > 28
- [ ] Image column is always approximately square: `imageChars ≈ imageRows × 2`
- [ ] No image available (nothing playing, fetch error, nil Images) → pane falls back to the pre-feature 2-col layout without image column
- [ ] Loading placeholder shown in image column while fetch is in flight
- [ ] `make ci` passes
