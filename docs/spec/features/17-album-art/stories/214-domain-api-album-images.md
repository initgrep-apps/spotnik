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
    Height int    `json:"height"`
    Width  int    `json:"width"`
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
≥ 64 px wide (avoids thumbnails too small for pixterm to work with):

```go
// BestImage returns the smallest image with Width >= minWidth, falling back to
// the last image in the slice (smallest available). Returns nil if Images is empty.
func (a Album) BestImage(minWidth int) *AlbumImage {
    var best *AlbumImage
    for i := range a.Images {
        img := &a.Images[i]
        if img.Width >= minWidth {
            if best == nil || img.Width < best.Width {
                best = img
            }
        }
    }
    if best != nil {
        return best
    }
    if len(a.Images) > 0 {
        return &a.Images[len(a.Images)-1]
    }
    return nil
}
```

`BestImage(100)` is the call site default — picks the smallest image ≥ 100 px
wide, which at terminal scale (16–48 char columns) still has plenty of detail.

### No changes needed elsewhere

`internal/api/player.go` unmarshals the playback response directly into
`domain.PlaybackState` → `domain.Track` → `domain.Album`. Since `Album.Images`
now has a `json:"images"` tag, the array flows through automatically. No
changes needed in `api/models.go` (it re-exports domain types as-is).

## Acceptance Criteria

- [ ] `AlbumImage{URL, Height, Width}` exists in `domain/types.go`
- [ ] `Album.Images []AlbumImage` is populated from Spotify JSON on unmarshal
- [ ] `Album.BestImage(minWidth)` returns the correct variant (smallest ≥ minWidth, fallback to last, nil on empty)
- [ ] Existing domain tests unaffected; `make ci` passes

## Tasks

- [ ] Add `AlbumImage` struct and `Album.Images` field to `internal/domain/types.go`
      with correct `json:` tags
      - test: unmarshal a fixture JSON containing `album.images` array; assert
        `album.Images` has 3 entries with correct URLs, heights, widths

- [ ] Add `BestImage(minWidth int) *AlbumImage` method on `Album` in `internal/domain/types.go`
      - test: table-driven — empty Images → nil; all images < minWidth → returns last;
        multiple images ≥ minWidth → returns the one with smallest Width;
        exactly one image ≥ minWidth → returns it

- [ ] Add fixture `testdata/fixtures/playback_with_images.json` — copy of a real
      `/me/player` response with a 3-entry `album.images` array
      - test: `TestPlayerClient_GetPlaybackState_ImagesPopulated` — mock server
        returns the fixture, assert `state.Item.Album.Images` has 3 entries

- [ ] `make ci` passes
