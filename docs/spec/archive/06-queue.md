# Feature 06 — Queue Management

> **Depends on:** Feature 03 (Playback) complete and committed.

## Goal

The right pane shows what's coming up next. Users can view the queue and understand
the shape of their listening session without switching apps. Tracks can be added to the
queue from library or search.

---

## Feature Acceptance Criteria

- [ ] Queue visible on startup within 2 seconds
- [ ] Queue updates within 1 second when track changes (via polling)
- [ ] `a` from library or search adds track and shows status bar confirmation
- [ ] Empty queue shows "Queue is empty" centered message
- [ ] No remove functionality exposed (Spotify API limitation)
- [ ] All pane `Update()` handlers tested

---

## Implementation Context

### Store fields this feature uses
```go
Queue           []api.Track // tracks in the current play queue
// Current track URI: use store.CurrentTrack.URI (no separate field needed)
```

### Message types for this feature
```go
type queueLoadedMsg  struct{ tracks []api.Track }
type queueAddedMsg   struct{ uri string }
type queueAddErrMsg  struct{ err error }
```

### Design tokens used in this feature
`theme.PlayingIndicator()` · `theme.SelectedBg()` · `theme.SelectedFg()` ·
`theme.TextPrimary()` · `theme.TextMuted()`

---

## User Stories

- **As a user**, I see the upcoming queue in the right pane at all times.
- **As a user**, the queue updates automatically as tracks change.
- **As a user**, I press `a` from library or search to add a track to the queue.
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
│    The Weeknd              │  ← artist (`TextSecondary()` token)
│                            │
│  ────────────────────────  │
│  NEXT UP                   │
│                            │
│  1  Save Your Tears        │  ← selected item (`SelectedBg()` token)
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
- **Remove from queue**: Not supported by Spotify Web API — there is no remove-from-queue endpoint

> **Note:** Queue item removal is not supported by the Spotify Web API. There is no `x` key
> in the queue pane. If a user expects removal, the status bar shows:
> "Queue removal not supported by Spotify API"

- **Refresh strategy:** Feature 06 extends the root model's 1-second tick loop to also
  dispatch `fetchQueue` alongside `fetchPlaybackState`. See `docs/ARCHITECTURE.md` →
  "Polling Ownership" for rules.

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
| `internal/ui/panes/queue_test.go` | Unit tests for QueuePane |

(API calls for queue are already in `player.go` from Feature 02)

---

## Task Breakdown

### Task 5.1 — QueuePane model

**Description:**
Implement the QueuePane as a `tea.Model` that reads queue data from the store and renders
two sections: "NOW" (the currently playing track) and "NEXT UP" (the rest of the queue).
Supports j/k navigation and Enter to play a selected track. Shows item count at the bottom.

**Files:** `internal/ui/panes/queue.go`, `internal/ui/panes/queue_test.go`

**Implementation steps:**
- [ ] Implement `tea.Model`: `Init()`, `Update()`, `View()`
- [ ] Read queue data from store (set by playback poll)
- [ ] Render "NOW" section (current track) + "NEXT UP" section
- [ ] Show item count at bottom ("{n} tracks remaining")
- [ ] Render "Queue is empty" centered message when queue is empty
- [ ] Handle j/k for cursor movement, Enter for play

**Acceptance criteria:**
- [ ] Empty queue renders "Queue is empty" centered message
- [ ] NOW section shows current track name and artist
- [ ] NEXT UP section shows numbered list with artist
- [ ] j/k moves cursor, Enter returns play command for selected track
- [ ] Pane ignores input when not focused

**Tests:**

*Unit tests:*
- `TestQueuePane_View_EmptyQueue` — renders "Queue is empty" message
- `TestQueuePane_View_NowPlaying` — renders NOW section with current track
- `TestQueuePane_View_NextUp` — renders numbered NEXT UP items with artist
- `TestQueuePane_View_ItemCount` — shows "{n} tracks remaining" at bottom
- `TestQueuePane_Update_J_MovesDown` — cursor moves down
- `TestQueuePane_Update_K_MovesUp` — cursor moves up
- `TestQueuePane_Update_Enter_PlaysTrack` — returns play command for selected track
- `TestQueuePane_Update_IgnoresWhenNotFocused` — returns nil when focused=false

---

### Task 5.2 — Queue refresh

**Description:**
Extend the root model's 1-second tick loop to dispatch `fetchQueue` alongside
`fetchPlaybackState`. Parse the queue JSON response (which contains `currently_playing`
and `queue` array) and update the store so the QueuePane reflects changes.

**Files:** `internal/app/app.go` (modify tick handler)

**Implementation steps:**
- [ ] Extend playback state poll to also fetch queue (`GET /me/player/queue`)
- [ ] Store queue in store alongside playback state
- [ ] Parse queue response: `currently_playing` + `queue` array

**Acceptance criteria:**
- [ ] Tick handler dispatches both `fetchPlaybackState` and `fetchQueue`
- [ ] `queueLoadedMsg` updates store and QueuePane re-renders with new items
- [ ] Queue endpoint is called on every 1-second tick

**Tests:**

*Unit tests:*
- `TestQueueResponse_Parse` — correctly parses queue JSON with currently_playing + queue array

*Integration tests:*
- `TestApp_TickFetchesQueue` — tickMsg dispatches both fetchPlaybackState and fetchQueue
- `TestApp_QueueUpdate_ReflectsInPane` — queueLoadedMsg updates store, QueuePane renders new items

---

### Task 5.3 — Add to queue UX

**Description:**
Handle `queueAddedMsg` and `queueAddErrMsg` in the root model to show status bar feedback.
On success, display "Added to queue: {track name}" for 3 seconds. On error, show the error
message in the status bar.

**Files:** `internal/app/app.go` (message handlers), `internal/ui/components/statusbar.go`

**Implementation steps:**
- [ ] Handle `queueAddedMsg` in root: show status bar "Added to queue: {name}"
- [ ] Auto-dismiss status after 3s
- [ ] Handle `queueAddErrMsg`: show error in status bar

**Acceptance criteria:**
- [ ] Successful add shows "Added to queue: {name}" in status bar
- [ ] Status message auto-dismisses after 3 seconds
- [ ] Error shows error message in status bar
- [ ] `a` key in library pane triggers `addToQueue` command

**Tests:**

*Unit tests:*
- `TestAddToQueue_Success_ShowsStatusMessage` — queueAddedMsg sets status bar text
- `TestAddToQueue_Error_ShowsError` — queueAddErrMsg shows error in status bar
- `TestAddToQueue_StatusAutoDismiss` — status message clears after 3 seconds

*Integration tests:*
- `TestApp_AddToQueueFromLibrary` — press `a` in library → addToQueue command → queueAddedMsg → status bar shows confirmation

---

*Last updated: 2026-03-22*
