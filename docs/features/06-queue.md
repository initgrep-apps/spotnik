# Feature 06 — Queue Management

> **Depends on:** Feature 03 (Playback) complete and committed.

## Implementation Context

### Store fields this feature uses
```go
Queue           []api.Track // tracks in the current play queue
CurrentTrackURI string      // URI of the currently playing track (for ▶ indicator)
```

### Message types for this feature
```go
type queueLoadedMsg      struct{ tracks []api.Track }
type addToQueueMsg       struct{ trackURI string }
type removeFromQueueMsg  struct{ position int }
```

### Design tokens used in this feature
`theme.PlayingIndicator()` · `theme.SelectedBg()` · `theme.SelectedFg()` ·
`theme.TextPrimary()` · `theme.TextMuted()`

---

---

## Goal

The right pane shows what's coming up next. Users can view, remove items, and understand
the shape of their listening session without switching apps.

---

## User Stories

- **As a user**, I see the upcoming queue in the right pane at all times.
- **As a user**, the queue updates automatically as tracks change.
- **As a user**, I press `a` from library or search to add a track to the queue.
- **As a user**, I press `x` on a queue item to remove it (if Spotify supports it — see note).
- **As a user**, the currently playing track is visually distinct from upcoming tracks.
- **As a user**, I see the count of remaining tracks at the bottom of the queue pane.

---

## Right Pane Layout

```
│  QUEUE                     │
│  ────────────────────────  │
│                            │
│  ▶ NOW                     │  ← currently playing label
│    Blinding Lights         │  ← track name
│    The Weeknd              │  ← artist (Subtext1)
│                            │
│  ────────────────────────  │
│  NEXT UP                   │
│                            │
│  1  Save Your Tears        │  ← selected item (Lavender bg)
│     The Weeknd             │
│  2  Starboy                │
│     The Weeknd             │
│  3  Can't Feel My Face     │
│     The Weeknd             │
│  4  In Your Eyes           │
│     The Weeknd             │
│  5  Repeat After Me        │
│     Post Malone            │
│                            │
│  ────────────────────────  │
│  5 tracks remaining        │
│                            │
```

---

## API Usage

- **Load queue**: `GET /me/player/queue` — returns `currently_playing` + `queue` array
- **Add to queue**: `POST /me/player/queue?uri={track_uri}`
- **Remove from queue**: ⚠️ **Not supported by Spotify Web API** — there is no remove-from-queue endpoint
  - Display queue items as read-only; remove button (`x`) is hidden
  - Show info message if user tries: "Spotify doesn't support removing queue items via API"

- **Refresh strategy**: refetch queue on every playback state poll (1s tick)
  - Queue endpoint is fast and lightweight

---

## Add to Queue Flow

Queue additions can be triggered from:
1. **Library pane**: press `a` on a track
2. **Search overlay**: press `a` on a track result
3. **Queue pane**: not applicable (already in queue)

```go
// Command pattern for add-to-queue
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

---

## Keymap (Queue Pane Focus)

| Key | Action |
|---|---|
| `j` / `↓` | Move selection down |
| `k` / `↑` | Move selection up |
| `Enter` | Play selected track immediately |
| `Tab` | Move focus to Library pane |
| `Shift+Tab` | Move focus to Player pane |

---

## Files to Create

| File | Purpose |
|---|---|
| `internal/ui/panes/queue.go` | QueuePane model |
| `internal/ui/panes/queue_test.go` | Update tests |

(API calls for queue are already in `player.go` from Feature 02)

---

## Task Breakdown

### Task 5.1 — QueuePane model
- [ ] Implement `tea.Model`: `Init()`, `Update()`, `View()`
- [ ] Read queue data from store (set by playback poll)
- [ ] Render "NOW" section (current track) + "NEXT UP" section
- [ ] Show item count at bottom
- [ ] Test: empty queue, single track, multiple tracks

### Task 5.2 — Queue refresh
- [ ] Extend playback state poll to also fetch queue (`GET /me/player/queue`)
- [ ] Store queue in store alongside playback state
- [ ] Test: queue updates after track change

### Task 5.3 — Add to queue UX
- [ ] Handle `queueAddedMsg` in root: show status bar "Added to queue: {name}"
- [ ] Auto-dismiss status after 3s
- [ ] Handle `queueAddErrMsg`: show error in status bar
- [ ] Test: add success and error flows

---

## Acceptance Criteria

- [ ] Queue visible on startup within 2 seconds
- [ ] Queue updates within 1 second when track changes
- [ ] `a` from library/search adds track and shows confirmation
- [ ] Empty queue shows "Queue is empty" message
- [ ] All pane update handlers tested

---

*Last updated: 2026-02-21*
