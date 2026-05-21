---
title: "Domain + API: AlbumImage type and player JSON mapping"
feature: 17-album-art
status: open
---

## Background

Spotify's `GET /v1/me/player` response includes an `album.images` array for the
currently playing track:

```json
"item": {
  "album": {
    "id": "...",
    "name": "...",
    "images": [
      {"url": "https://i.scdn.co/image/ab...", "height": 640, "width": 640},
      {"url": "https://i.scdn.co/image/ab...", "height": 300, "width": 300},
      {"url": "https://i.scdn.co/image/ab...", "height": 64,  "width": 64}
    ]
  }
}
```

`domain.Album` currently has only `ID` and `Name` — the images array is silently
discarded on unmarshal. No other change to the API client is needed; Go's
`encoding/json` picks up new struct fields automatically.

## Design

### `internal/domain/types.go`

Add a new `AlbumImage` struct and extend `Album`:

```go
// AlbumImage is a single size variant of an album's cover art from Spotify.
type AlbumImage struct {
    URL    string `json:"url"`
    Width  int    `json:"width"`
    Height int    `json:"height"`
}

// Album — add the Images field alongside existing ID and Name:
type Album struct {
    ID     string       `json:"id"`
    Name   string       `json:"name"`
    Images []AlbumImage `json:"images"`
}
```

Add a `BestImage` helper on `Album`. Spotify always returns images sorted
largest-first. For terminal rendering we want the smallest image that is still
≥ minSize px in both dimensions (avoids thumbnails too small for pixterm):

```go
// BestImage returns the smallest image where both Width and Height are >= minSize,
// falling back to the explicitly largest image. Returns nil if Images is empty.
func (a Album) BestImage(minSize int) *AlbumImage {
    var best *AlbumImage
    for i := range a.Images {
        img := &a.Images[i]
        if img.Width >= minSize && img.Height >= minSize {
            if best == nil || img.Width < best.Width {
                best = img
            }
        }
    }
    if best != nil {
        return best
    }
    // fallback: return explicitly largest
    var largest *AlbumImage
    for i := range a.Images {
        img := &a.Images[i]
        if largest == nil || img.Width > largest.Width {
            largest = img
        }
    }
    return largest
}
```

`BestImage(100)` is the call site default — picks the smallest image ≥ 100 px
in both dimensions, which at terminal scale (16–48 char columns) still has plenty of detail.

### No changes needed elsewhere

`internal/api/player.go` unmarshals the playback response directly into
`domain.PlaybackState` → `domain.Track` → `domain.Album`. Since `Album.Images`
now has a `json:"images"` tag, the array flows through automatically. No
changes needed in `api/models.go` (it re-exports domain types as-is).

## Acceptance Criteria

- [ ] `AlbumImage{URL, Width, Height}` exists in `domain/types.go`
- [ ] `Album.Images []AlbumImage` is populated from Spotify JSON on unmarshal
- [ ] `Album.BestImage(minSize)` returns smallest image where both dims ≥ minSize; falls back to explicitly largest; nil on empty
- [ ] Existing domain tests unaffected; `make ci` passes

## Tasks

- [ ] Add `AlbumImage` struct and `Album.Images` field to `internal/domain/types.go`
      with correct `json:` tags
      - test: unmarshal a fixture JSON containing `album.images` array; assert
        `album.Images` has 3 entries with correct URLs, heights, widths

- [ ] Add `BestImage(minSize int) *AlbumImage` method on `Album` in `internal/domain/types.go`
      - test: table-driven — empty Images → nil; all images < minSize → returns explicitly largest;
        multiple images ≥ minSize → returns the one with smallest Width;
        exactly one image ≥ minSize → returns it;
        image where only Width ≥ minSize (Height too small) → skipped, falls back to largest

- [ ] Add fixture `testdata/fixtures/playback_with_images.json` — copy of a real
      `/me/player` response with a 3-entry `album.images` array
      - test: `TestPlayerClient_GetPlaybackState_ImagesPopulated` — mock server
        returns the fixture, assert `state.Item.Album.Images` has 3 entries

- [ ] `make ci` passes
