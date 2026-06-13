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

## Tasks

- [ ] Add `QueueItemType` and `QueueItem` types to `internal/domain/types.go`
      - `QueueItemType` iota (`QueueItemTypeTrack`, `QueueItemTypeEpisode`), `QueueItem` struct with `Type`, `Track *Track`, `Episode *Episode`
      - test: `TestQueueItemType_Values`, `TestQueueItem_TrackType`, `TestQueueItem_EpisodeType`
- [ ] Change `store.queue` field and accessors from `[]domain.Track` to `[]domain.QueueItem`
      - Modify `internal/state/store.go`: `queue` field type, `Queue()` return type, `SetQueue()` parameter type
      - test: `TestStore_SetGetQueue_QueueItemType`, `TestStore_Queue_TrackAndEpisode`
- [ ] Change `QueueLoadedMsg` from `Tracks []domain.Track` to `Items []domain.QueueItem`
      - Modify `internal/ui/panes/messages.go`
      - test: `TestQueueLoadedMsg_ItemsField`
- [ ] Update API queue response parsing for mixed content
      - Modify `internal/api/player.go` or relevant client: parse both `track` and `episode` objects from queue response, wrap in `QueueItemTypeTrack`/`QueueItemTypeEpisode`
      - test: `TestParseQueueResponse_TrackOnly`, `TestParseQueueResponse_MixedTrackEpisode`, `TestParseQueueResponse_EpisodeFields`
- [ ] Add `type` column to Queue pane and update column definitions
      - Modify `internal/ui/panes/queue.go`: add `type` column (flex 1, `♪`/`◆`), rename `Track` → `Title` header, show Show name for episode `Artist`
      - test: `TestQueuePane_TypeColumn_TrackSymbol`, `TestQueuePane_TypeColumn_EpisodeSymbol`, `TestQueuePane_TitleHeader`, `TestQueuePane_ArtistColumn_EpisodeShowName`
- [ ] Update Queue pane Enter handler for mixed content playback
      - Modify `internal/ui/panes/queue.go`: Enter on track row `→ PlayTrackMsg`, Enter on episode row `→ PlayEpisodeMsg`
      - test: `TestQueuePane_EnterTrack_PlaysTrack`, `TestQueuePane_EnterEpisode_PlaysEpisode`
- [ ] Update `handlers.go` for `QueueLoadedMsg` type change
      - Modify `internal/app/handlers.go`: handler now reads `msg.Items []domain.QueueItem` instead of `msg.Tracks`
      - test: `TestHandler_QueueLoadedMsg_MixedItems`
- [ ] Update queue fetch command for mixed parsing
      - Modify `internal/app/commands.go`: update `buildFetchQueueCmd` to parse both types
      - test: `TestBuildFetchQueueCmd_MixedContent`
- [ ] Run `make ci` — all lint, tests, and 80% coverage pass