---
title: "Queue: mixed content support (type column, QueueItem)"
feature: 19-player-page-unification
status: open
---

## Background

The Queue pane currently only handles `[]domain.Track`. Spotify's
`GET /me/player/queue` can return both tracks and episodes. This story adds a
`QueueItem` type that wraps both, adds a `type` column to the Queue pane, and
updates the queue data pipeline from API parsing through store to rendering.

## Design

### Domain type: `QueueItem`

```go
type QueueItemType int
const (
    QueueItemTypeTrack   QueueItemType = iota
    QueueItemTypeEpisode
)

type QueueItem struct {
    Type     QueueItemType
    Track    *Track
    Episode  *Episode
}
```

When `Type == QueueItemTypeTrack`, `Track` is non-nil and `Episode` is nil.
When `Type == QueueItemTypeEpisode`, `Episode` is non-nil and `Track` is nil.

### Store changes

- `store.queue` field type changes from `[]domain.Track` to `[]domain.QueueItem`
- `store.Queue()` returns `[]domain.QueueItem`
- `store.SetQueue(items []domain.QueueItem)` accepts `[]domain.QueueItem`
- `store.QueueFetching()` and `store.SetQueueFetching()` unchanged

### API changes

The Queue API response handler must parse both `track` and `episode` objects
from the queue response. Items with `type: "episode"` are wrapped in
`QueueItemTypeEpisode`; items with `type: "track"` (or no type) are wrapped
in `QueueItemTypeTrack`.

### Queue pane columns

| Key | Header | FlexFactor | Color Token | Notes |
|-----|--------|-----------|-------------|-------|
| `index` | `#` | 1 | `ColumnIndex()` | 1-based; `▶` for currently-playing |
| `type` | `""` | 1 | `ColumnSecondary()` | `♪` for track, `◆` for episode |
| `title` | `Title` | 7 | `ColumnPrimary()` | Track name or episode name |
| `artist` | `Artist` | 4 | `ColumnSecondary()` | Artist name (track) or Show name (episode) |
| `duration` | `Duration` | 2 | `ColumnTertiary()` | Formatted `"m:ss"` or `"h:mm:ss"` |
| `icon` | `""` | 1 | `ColumnSecondary()` | Existing icon column |

Total flex factor: 16.

The `type` column replaces what was formerly the `track` header with `Title`.
The `Artist` column header stays `Artist` but displays Show name for episodes.

### Row rendering

- Track: type = `♪`, title = track.Name, artist = track.Artists[0].Name
- Episode: type = `◆`, title = episode.Name, artist = episode.Show.Name

### Queue play behavior

`Enter` on a track row → `PlayTrackMsg{TrackURI: item.Track.URI}` (existing).
`Enter` on an episode row → `PlayEpisodeMsg{EpisodeURI: item.Episode.URI, PlaylistURI: ""}`
(triggers auto-switch to Podcast preset per story 239).

### Message type change

`QueueLoadedMsg` changes from `Tracks []domain.Track` to `Items []domain.QueueItem`.

## Files

### Modify

- `internal/domain/types.go` — add `QueueItemType`, `QueueItem`
- `internal/state/store.go` — change `queue` field and accessors to `[]domain.QueueItem`
- `internal/ui/panes/messages.go` — change `QueueLoadedMsg` to use `[]domain.QueueItem`
- `internal/ui/panes/queue.go` — add `type` column, handle mixed content rendering
- `internal/ui/panes/queue_test.go` — add mixed content tests
- `internal/app/handlers.go` — update queue handler for `QueueItem` type
- `internal/app/commands.go` — update queue fetch command for mixed parsing
- `internal/api/player.go` or relevant client — parse queue response with both types

## Acceptance Criteria

- [ ] `QueueItem` type compiles with `Type`, `Track`, `Episode` fields
- [ ] Store `Queue()` returns `[]domain.QueueItem`
- [ ] `QueueLoadedMsg` carries `[]domain.QueueItem`
- [ ] Queue pane renders `♪` for tracks, `◆` for episodes in new `type` column
- [ ] `Track` header renamed to `Title`
- [ ] `Artist` column shows Show name for episodes
- [ ] Enter on track row plays track
- [ ] Enter on episode row plays episode
- [ ] Existing track-only queue tests still pass
- [ ] `make ci` passes