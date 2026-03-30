---
title: "Queue Management"
description: "Displays the upcoming playback queue in a dense table pane, supports add-to-queue from library/search, automatic refresh via tick polling, and in-pane filtering."
status: done
stories: [06, 46]
---

# Queue Management

## Background

The Queue feature gives Spotnik users a persistent view of what is coming up next in their
listening session, without leaving the terminal. The right pane displays the currently playing
track and all queued tracks, updating automatically every second via the root model's tick loop.
Users can add tracks to the queue from the library or search overlay, with status bar feedback
confirming each addition.

The initial implementation (spec 06) built the QueuePane as a manually formatted list with
cursor-based navigation and store-driven data flow. The follow-up migration (spec 46) upgraded
the pane to implement the full `layout.Pane` interface, replaced manual `fmt.Sprintf` rendering
with a bubble-table dense table, and added in-pane filtering support. Together, these stories
deliver a polished, keyboard-driven queue experience consistent with the rest of the Spotnik
pane system.

The queue pane reads all data from `state.Store` and never calls the API directly. Queue
additions are routed through the app-level command pattern, and all errors surface via toast
notifications. The Spotify Web API does not support queue item removal, so no remove
functionality is exposed.

---

## Story: Queue Management (spec 06)

### Background
This story built the foundational queue feature: a QueuePane that renders the current and
upcoming tracks, automatic refresh via the 1-second tick loop, and add-to-queue UX with
status bar confirmation. It established the data flow from the Spotify queue endpoint through
the Store to the pane's view.

### Acceptance Criteria
- [ ] Queue visible on startup within 2 seconds
- [ ] Queue updates within 1 second when track changes (via polling)
- [ ] `a` from library or search adds track and shows status bar confirmation
- [ ] Empty queue shows "Queue is empty" centered message
- [ ] No remove functionality exposed (Spotify API limitation)
- [ ] All pane `Update()` handlers tested

### Implementation Context

**Store fields this feature uses:**
```go
Queue           []api.Track // tracks in the current play queue
// Current track URI: use store.CurrentTrack.URI (no separate field needed)
```

**Message types for this feature:**
```go
type queueLoadedMsg  struct{ tracks []api.Track }
type queueAddedMsg   struct{ uri string }
type queueAddErrMsg  struct{ err error }
```

**Design tokens used:**
`theme.PlayingIndicator()`, `theme.SelectedBg()`, `theme.SelectedFg()`,
`theme.TextPrimary()`, `theme.TextMuted()`

### User Stories

- **As a user**, I see the upcoming queue in the right pane at all times.
- **As a user**, the queue updates automatically as tracks change.
- **As a user**, I press `a` from library or search to add a track to the queue.
- **As a user**, the currently playing track is visually distinct from upcoming tracks.
- **As a user**, I see the count of remaining tracks at the bottom of the queue pane.

### Right Pane Layout

```
|  QUEUE                     |
|  ────────────────────────  |
|                            |
|  > NOW                     |  <- currently playing label
|    Blinding Lights         |  <- track name
|    The Weeknd              |  <- artist (TextSecondary() token)
|                            |
|  ────────────────────────  |
|  NEXT UP                   |
|                            |
|  1  Save Your Tears        |  <- selected item (SelectedBg() token)
|     The Weeknd             |
|  2  Starboy                |
|     The Weeknd             |
|  3  Can't Feel My Face     |
|     The Weeknd             |
|  4  In Your Eyes           |
|     The Weeknd             |
|  5  Repeat After Me        |
|     Post Malone            |
|                            |
|  ────────────────────────  |
|  5 tracks remaining        |
|                            |
```

### API Usage

- **Load queue**: `GET /me/player/queue` -- returns `currently_playing` + `queue` array
- **Add to queue**: `POST /me/player/queue?uri={track_uri}`
- **Remove from queue**: Not supported by Spotify Web API -- no remove-from-queue endpoint

> **Note:** Queue item removal is not supported by the Spotify Web API. There is no `x` key
> in the queue pane. If a user expects removal, the status bar shows:
> "Queue removal not supported by Spotify API"

- **Refresh strategy:** Feature 06 extends the root model's 1-second tick loop to also
  dispatch `fetchQueue` alongside `fetchPlaybackState`. See `docs/ARCHITECTURE.md` ->
  "Polling Ownership" for rules.

### Add to Queue Flow

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

### Keymap (Queue Pane Focus)

| Key | Action |
|---|---|
| `j` / `down` | Move selection down |
| `k` / `up` | Move selection up |
| `Enter` | Play selected track immediately |
| `Tab` | Move focus to Library pane |
| `Shift+Tab` | Move focus to Player pane |

### Tasks

1. **Task 5.1 -- QueuePane model** -- Implement the QueuePane as a `tea.Model` that reads queue data from the store and renders two sections: "NOW" (the currently playing track) and "NEXT UP" (the rest of the queue). Supports j/k navigation and Enter to play a selected track. Shows item count at the bottom.
   - Files: `internal/ui/panes/queue.go`, `internal/ui/panes/queue_test.go`
   - Implementation steps:
     - [ ] Implement `tea.Model`: `Init()`, `Update()`, `View()`
     - [ ] Read queue data from store (set by playback poll)
     - [ ] Render "NOW" section (current track) + "NEXT UP" section
     - [ ] Show item count at bottom ("{n} tracks remaining")
     - [ ] Render "Queue is empty" centered message when queue is empty
     - [ ] Handle j/k for cursor movement, Enter for play
   - Acceptance criteria:
     - [ ] Empty queue renders "Queue is empty" centered message
     - [ ] NOW section shows current track name and artist
     - [ ] NEXT UP section shows numbered list with artist
     - [ ] j/k moves cursor, Enter returns play command for selected track
     - [ ] Pane ignores input when not focused
   - Tests (unit):
     - `TestQueuePane_View_EmptyQueue` -- renders "Queue is empty" message
     - `TestQueuePane_View_NowPlaying` -- renders NOW section with current track
     - `TestQueuePane_View_NextUp` -- renders numbered NEXT UP items with artist
     - `TestQueuePane_View_ItemCount` -- shows "{n} tracks remaining" at bottom
     - `TestQueuePane_Update_J_MovesDown` -- cursor moves down
     - `TestQueuePane_Update_K_MovesUp` -- cursor moves up
     - `TestQueuePane_Update_Enter_PlaysTrack` -- returns play command for selected track
     - `TestQueuePane_Update_IgnoresWhenNotFocused` -- returns nil when focused=false

2. **Task 5.2 -- Queue refresh** -- Extend the root model's 1-second tick loop to dispatch `fetchQueue` alongside `fetchPlaybackState`. Parse the queue JSON response (which contains `currently_playing` and `queue` array) and update the store so the QueuePane reflects changes.
   - Files: `internal/app/app.go` (modify tick handler)
   - Implementation steps:
     - [ ] Extend playback state poll to also fetch queue (`GET /me/player/queue`)
     - [ ] Store queue in store alongside playback state
     - [ ] Parse queue response: `currently_playing` + `queue` array
   - Acceptance criteria:
     - [ ] Tick handler dispatches both `fetchPlaybackState` and `fetchQueue`
     - [ ] `queueLoadedMsg` updates store and QueuePane re-renders with new items
     - [ ] Queue endpoint is called on every 1-second tick
   - Tests (unit):
     - `TestQueueResponse_Parse` -- correctly parses queue JSON with currently_playing + queue array
   - Tests (integration):
     - `TestApp_TickFetchesQueue` -- tickMsg dispatches both fetchPlaybackState and fetchQueue
     - `TestApp_QueueUpdate_ReflectsInPane` -- queueLoadedMsg updates store, QueuePane renders new items

3. **Task 5.3 -- Add to queue UX** -- Handle `queueAddedMsg` and `queueAddErrMsg` in the root model to show status bar feedback. On success, display "Added to queue: {track name}" for 3 seconds. On error, show the error message in the status bar.
   - Files: `internal/app/app.go` (message handlers), `internal/ui/components/statusbar.go`
   - Implementation steps:
     - [ ] Handle `queueAddedMsg` in root: show status bar "Added to queue: {name}"
     - [ ] Auto-dismiss status after 3s
     - [ ] Handle `queueAddErrMsg`: show error in status bar
   - Acceptance criteria:
     - [ ] Successful add shows "Added to queue: {name}" in status bar
     - [ ] Status message auto-dismisses after 3 seconds
     - [ ] Error shows error message in status bar
     - [ ] `a` key in library pane triggers `addToQueue` command
   - Tests (unit):
     - `TestAddToQueue_Success_ShowsStatusMessage` -- queueAddedMsg sets status bar text
     - `TestAddToQueue_Error_ShowsError` -- queueAddErrMsg shows error in status bar
     - `TestAddToQueue_StatusAutoDismiss` -- status message clears after 3 seconds
   - Tests (integration):
     - `TestApp_AddToQueueFromLibrary` -- press `a` in library -> addToQueue command -> queueAddedMsg -> status bar shows confirmation

### Files Created

| File | Purpose |
|---|---|
| `internal/ui/panes/queue.go` | QueuePane model |
| `internal/ui/panes/queue_test.go` | Unit tests for QueuePane |

(API calls for queue are already in `player.go` from Feature 02)

---

## Story: Queue Pane Migration (spec 46)

### Background
The original QueuePane rendered the queue as a manually formatted list with fixed-width columns
using `fmt.Sprintf`. It had `SetSize`, `SetFocused`, `IsFocused` methods but did not implement
the full `layout.Pane` interface (missing `ID`, `Title`, `ToggleKey`, `Actions`). This story
upgraded the pane to satisfy the `layout.Pane` interface, replaced manual rendering with
bubble-table dense table format, and added in-pane filtering with the `f` key.

**Design reference:** `docs/DESIGN.md` section 2 (Pane Definitions -- Queue), section 9 (Dense Table Formatting --
column widths: # 5%, Track 45%, Artist 35%, Duration 15%), section 8 (In-Pane Filtering)

**Depends on:** Feature 41 (Pane interface), Feature 43 (Table + Filter components)

### Design Diagram

```
Queue Pane (DESIGN.md section 9):

+-- 2Queue ----------------------------------------- >f filter -- >A add -+
|  #   Track                    Artist              Duration               |
|  1   Lil Boo Thang            Paul Russell        3:12                   |
|  2   Street Fighter           Kamasi Washington   5:44                   |
| >3   BIRDS OF A FEATHER       Billie Eilish       3:30                   |
|  4   Peaches                  Justin Bieber       3:18                   |
|  v more below                                                            |
+--------------------------------------------------------------------------+

Column Widths: # 5% | Track 45% | Artist 35% | Duration 15%

Filter Active:
+-- 2Queue -------------- filtering: "rock" --- >Esc close -+
|  > rock                                                   |
|  3   Rocket Man              Elton John     4:52          |
|  7   Rock and Roll           Led Zeppelin   3:40          |
+-----------------------------------------------------------+
```

### Acceptance Criteria
- [ ] `QueuePane` satisfies `layout.Pane` interface
- [ ] Dense table format with 4 columns: #, Track, Artist, Duration
- [ ] Column widths approximate 5%/45%/35%/15% ratio
- [ ] Currently playing track shows `>` indicator in index column
- [ ] Per-column colors: TextMuted, TextPrimary, TextSecondary, TextMuted
- [ ] Selected row highlighted with SelectedBg/SelectedFg
- [ ] `f` key toggles in-pane filter
- [ ] Filter matches track name and artist name (case-insensitive)
- [ ] Filter state shown in border label
- [ ] `j`/`k` scrolling works
- [ ] Existing queue functionality preserved (data from Store, refresh on tick)
- [ ] `make ci` passes

### Tasks

1. **Task 1 -- Implement layout.Pane interface methods** -- QueuePane lacks `ID`, `Title`, `ToggleKey`, `Actions` methods. Add them.
   - Files: `internal/ui/panes/queue.go`
   - Implementation:
     ```go
     func (q *QueuePane) ID() layout.PaneID       { return layout.PaneQueue }
     func (q *QueuePane) Title() string            { return "Queue" }
     func (q *QueuePane) ToggleKey() int           { return 2 }
     func (q *QueuePane) Actions() []layout.Action {
         return []layout.Action{
             {Key: "f", Label: "filter"},
             {Key: "A", Label: "add"},
         }
     }
     ```
   - Tests:
     - Unit: `ID()` returns `PaneQueue`
     - Unit: `Title()` returns "Queue"
     - Unit: `ToggleKey()` returns 2
     - Unit: `Actions()` returns filter + add
     - Unit: Compile-time check: `var _ layout.Pane = &QueuePane{}`
   - Commit: `feat(ui): QueuePane implements layout.Pane interface`

2. **Task 2 -- Replace list rendering with bubble-table** -- Queue uses manual `fmt.Sprintf` formatting for each row. Replace with bubble-table dense table.
   - Files: `internal/ui/panes/queue.go`
   - Implementation steps:
     - Add `table components.Table` field to `QueuePane`
     - Initialize in constructor with column definitions:
       ```go
       columns := []components.ColumnDef{
           {Key: "index", Header: "#", FlexFactor: 1, Color: theme.TextMuted()},
           {Key: "track", Header: "Track", FlexFactor: 9, Color: theme.TextPrimary()},
           {Key: "artist", Header: "Artist", FlexFactor: 7, Color: theme.TextSecondary()},
           {Key: "duration", Header: "Duration", FlexFactor: 3, Color: theme.TextMuted()},
       }
       ```
       (Flex factors approximate 5%/45%/35%/15% ratios)
     - On `QueueLoadedMsg`: convert `[]domain.Track` to table rows (`[]map[string]string`)
     - Set `PlayingIndex` to the index of the currently playing track
     - In `SetSize()`: call `table.SetSize(contentWidth, contentHeight)`
     - In `View()`: return `table.View()`
     - In `Update()`: forward key messages to `table.Update(msg)` when focused
   - Tests:
     - Unit: Queue with 5 tracks -> table has 5 rows
     - Unit: Currently playing track shows > indicator
     - Unit: Column headers rendered: #, Track, Artist, Duration
     - Unit: j/k navigation changes selected row
     - Unit: SetSize updates table dimensions
     - Unit: Empty queue -> table shows empty state (no panic)
     - Unit: Track names truncated to column width (no overflow)
   - Commit: `feat(ui): QueuePane uses bubble-table for dense rendering`

3. **Task 3 -- Add filter support** -- Queue has no in-pane filtering. Add filter toggled with `f` key.
   - Files: `internal/ui/panes/queue.go`
   - Implementation steps:
     - Add `filter *components.Filter` field to `QueuePane`
     - Initialize in constructor: `filter: components.NewFilter(theme)`
     - In `Update()`:
       - `f` key when focused -> `filter.Toggle()`
       - When filter active, forward key messages to `filter.Update(msg)`
       - On filter query change -> filter track data and update table rows
     - In `View()`:
       - If filter active, prepend `filter.View(contentWidth)` above table
       - Reduce table height by 1 to accommodate filter bar
     - Override `Actions()` when filter active to return `filter.BorderLabel()`:
       ```go
       func (q *QueuePane) Actions() []layout.Action {
           if q.filter.IsActive() {
               return []layout.Action{{Key: "Esc", Label: "close"}}
           }
           return []layout.Action{{Key: "f", Label: "filter"}, {Key: "A", Label: "add"}}
       }
       ```
     - Filter matches against track name and artist name (`filter.MatchesAny(track.Name, track.ArtistName)`)
   - Tests:
     - Unit: `f` key activates filter
     - Unit: Typing in filter -> tracks filtered by name/artist
     - Unit: Filter "rock" -> only tracks with "rock" in name or artist shown
     - Unit: Esc closes filter, restores full list
     - Unit: Filter active -> Actions() changes to show close action
     - Unit: Empty filter query -> all tracks shown
     - Unit: Filter with no matches -> table shows empty state
   - Commit: `feat(ui): in-pane filtering for QueuePane`

4. **Task 4 -- Comprehensive tests** -- Full test coverage for the migrated QueuePane.
   - Files: `internal/ui/panes/queue_test.go`
   - Tests (integration):
     - Full lifecycle -- construct -> resize -> load queue -> filter -> navigate -> verify View
     - Queue updates on new QueueLoadedMsg -> table refreshes
     - Playing track indicator persists across data updates
     - Filter + scroll interaction -- filter results, scroll, clear filter, position resets
     - Interface satisfaction: `var _ layout.Pane = &QueuePane{}`
   - Tests (edge cases):
     - Queue with 200 items -> scrolling works, no performance issue
     - Queue with 0 items -> renders cleanly
     - Track with very long name -> truncated in column
   - Commit: `test(ui): comprehensive QueuePane tests with table and filter`

### Notes

- The QueuePane still reads data from `state.Store` -- this does not change. The table
  is a presentation layer on top of the same data flow.
- The `A` (Shift+a) key for "add to queue" is listed in Actions but is handled at the
  app level (routing.go), not inside QueuePane. The pane just declares it for border display.
- bubble-table handles its own keyboard scrolling (j/k, up/down, page up/down). The pane
  forwards key messages to `table.Update(msg)` when focused and filter is not active.
- When filter is active, key messages go to the filter first. Only Esc and Enter
  close the filter -- all other keys are consumed by the textinput.
