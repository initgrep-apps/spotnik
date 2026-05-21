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
|------|-----------|--------|
| Base | ≤ 18      | 3-col: image · InfoBox · viz |
| Mid  | 19 – 30   | 2-col: image+viz side-by-side, compact InfoBox below |
| Full | > 30      | 2-col: larger image+viz side-by-side, richer InfoBox below |

`bodyHeight = pane height − 4`. The Stats page and Dashboard/Library/Discovery
NowPlaying rows all receive `MinHeight: 14` via a new LayoutManager primitive
so they always reach at least the base tier regardless of terminal size.

## Acceptance Criteria

- [ ] `domain.Album` carries `Images []AlbumImage`; Spotify `/me/player` JSON `album.images` is mapped into domain types
- [ ] `BestImage()` helper on `Album` returns the smallest usable image URL
- [ ] LayoutManager honours `Row.MinHeight`; Stats page NowPlaying gets ≥ 14 rows at any terminal size
- [ ] Album art is fetched asynchronously on `Init()` (if playback already active) and on every track change detected via `PlaybackStateFetchedMsg`
- [ ] Re-poll of the same track does not re-fetch (track ID cache)
- [ ] NowPlaying enters base tier (3-col) when bodyHeight ≤ 18
- [ ] NowPlaying enters mid tier (2-col + compact InfoBox) when bodyHeight 19–30
- [ ] NowPlaying enters full tier (2-col + richer InfoBox) when bodyHeight > 30
- [ ] Dashboard, Library, Discovery presets have `MinHeight: 14` on NowPlaying row
- [ ] Image column is always approximately square: `imageChars ≈ imageRows × 2`
- [ ] No image available (nothing playing, fetch error, nil Images) → pane falls back to the pre-feature 2-col layout without image column
- [ ] Loading placeholder shown in image column while fetch is in flight
- [ ] `make ci` passes
