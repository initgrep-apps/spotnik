---
title: "Queue Management"
feature: 06-queue
status: done
---

## Background
This story built the foundational queue feature: a QueuePane that renders the current and upcoming tracks, automatic refresh via the 1-second tick loop, and add-to-queue UX with status bar confirmation. It established the data flow from the Spotify queue endpoint through the Store to the pane's view.

## Design

### Store fields
```go
Queue           []api.Track // tracks in the current play queue
// Current track URI: use store.CurrentTrack.URI (no separate field needed)
```

### Message types
```go
type queueLoadedMsg  struct{ tracks []api.Track }
type queueAddedMsg   struct{ uri string }
type queueAddErrMsg  struct{ err error }
```

### Design tokens
`theme.PlayingIndicator()`, `theme.SelectedBg()`, `theme.SelectedFg()`, `theme.TextPrimary()`, `theme.TextMuted()`

### Right Pane Layout

```
|  QUEUE                     |
|  ----------------------    |
|                            |
|  > NOW                     |  <- currently playing label
|    Blinding Lights         |  <- track name
|    The Weeknd              |  <- artist
|                            |
|  ----------------------    |
|  NEXT UP                   |
|                            |
|  1  Save Your Tears        |  <- selected item
|     The Weeknd             |
|  2  Starboy                |
|     The Weeknd             |
|  3  Can't Feel My Face     |
|     The Weeknd             |
|                            |
|  ----------------------    |
|  5 tracks remaining        |
|                            |
```

### API Usage
- **Load queue**: `GET /me/player/queue`
- **Add to queue**: `POST /me/player/queue?uri={track_uri}`
- **Remove from queue**: Not supported by Spotify Web API

### Add to Queue Flow

Queue additions can be triggered from:
1. **Library pane**: press `a` on a track
2. **Search overlay**: press `a` on a track result
3. **Queue pane**: not applicable (already in queue)

```go
func addToQueue(client api.SpotifyClient, uri string) tea.Cmd {
    return func() tea.Msg {
        err := client.AddToQueue(context.Background(), uri)
        if err != nil {
            return queueAddErrMsg{err: err}
        }
        return queueAddedMsg{uri: uri}
    }
}
```

On success: show status bar message "Added to queue: {track name}" for 3 seconds.

### Keymap (Queue Pane Focus)

| Key | Action |
|---|---|
| `j` / `down` | Move selection down |
| `k` / `up` | Move selection up |
| `Enter` | Play selected track immediately |
| `Tab` | Move focus to Library pane |
| `Shift+Tab` | Move focus to Player pane |

### Files Created

| File | Purpose |
|---|---|
| `internal/ui/panes/queue.go` | QueuePane model |
| `internal/ui/panes/queue_test.go` | Unit tests for QueuePane |

## Acceptance Criteria
- [ ] Queue visible on startup within 2 seconds
- [ ] Queue updates within 1 second when track changes (via polling)
- [ ] `a` from library or search adds track and shows status bar confirmation
- [ ] Empty queue shows "Queue is empty" centered message
- [ ] No remove functionality exposed (Spotify API limitation)
- [ ] All pane `Update()` handlers tested

## Tasks
- [ ] QueuePane model -- Implement tea.Model with NOW/NEXT UP sections and j/k navigation
      - test: `TestQueuePane_View_EmptyQueue`, `TestQueuePane_View_NowPlaying`, `TestQueuePane_View_NextUp`, `TestQueuePane_View_ItemCount`, `TestQueuePane_Update_J_MovesDown`, `TestQueuePane_Update_K_MovesUp`, `TestQueuePane_Update_Enter_PlaysTrack`, `TestQueuePane_Update_IgnoresWhenNotFocused`
- [ ] Queue refresh -- Extend root model's tick loop to dispatch fetchQueue alongside fetchPlaybackState
      - test: `TestQueueResponse_Parse`, `TestApp_TickFetchesQueue`, `TestApp_QueueUpdate_ReflectsInPane`
- [ ] Add to queue UX -- Handle queueAddedMsg/queueAddErrMsg with status bar feedback
      - test: `TestAddToQueue_Success_ShowsStatusMessage`, `TestAddToQueue_Error_ShowsError`, `TestAddToQueue_StatusAutoDismiss`, `TestApp_AddToQueueFromLibrary`
