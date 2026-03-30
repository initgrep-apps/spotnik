---
title: "Queue Pane Migration"
feature: 06-queue
status: done
---

## Background
The original QueuePane rendered the queue as a manually formatted list with fixed-width columns using `fmt.Sprintf`. It had `SetSize`, `SetFocused`, `IsFocused` methods but did not implement the full `layout.Pane` interface (missing `ID`, `Title`, `ToggleKey`, `Actions`). This story upgraded the pane to satisfy the `layout.Pane` interface, replaced manual rendering with bubble-table dense table format, and added in-pane filtering with the `f` key.

**Design reference:** `docs/DESIGN.md` section 2 (Pane Definitions -- Queue), section 9 (Dense Table Formatting -- column widths: # 5%, Track 45%, Artist 35%, Duration 15%), section 8 (In-Pane Filtering)

**Depends on:** Feature 41 (Pane interface), Feature 43 (Table + Filter components)

## Design

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

### Pane Interface Implementation
```go
func (q *QueuePane) ID() layout.PaneID       { return layout.PaneQueue }
func (q *QueuePane) Title() string            { return "Queue" }
func (q *QueuePane) ToggleKey() int           { return 2 }
func (q *QueuePane) Actions() []layout.Action {
    if q.filter.IsActive() {
        return []layout.Action{{Key: "Esc", Label: "close"}}
    }
    return []layout.Action{
        {Key: "f", Label: "filter"},
        {Key: "A", Label: "add"},
    }
}
```

### Column Definitions
```go
columns := []components.ColumnDef{
    {Key: "index", Header: "#", FlexFactor: 1, Color: theme.TextMuted()},
    {Key: "track", Header: "Track", FlexFactor: 9, Color: theme.TextPrimary()},
    {Key: "artist", Header: "Artist", FlexFactor: 7, Color: theme.TextSecondary()},
    {Key: "duration", Header: "Duration", FlexFactor: 3, Color: theme.TextMuted()},
}
```

### Notes
- QueuePane still reads data from `state.Store` -- no change to data flow.
- The `A` (Shift+a) key for "add to queue" is handled at the app level, not inside QueuePane.
- bubble-table handles its own keyboard scrolling (j/k, up/down, page up/down).
- When filter is active, key messages go to the filter first. Only Esc and Enter close the filter.

## Acceptance Criteria
- [ ] `QueuePane` satisfies `layout.Pane` interface
- [ ] Dense table format with 4 columns: #, Track, Artist, Duration
- [ ] Column widths approximate 5%/45%/35%/15% ratio
- [ ] Currently playing track shows `>` indicator in index column
- [ ] Per-column colors: TextMuted, TextPrimary, TextSecondary, TextMuted
- [ ] Selected row highlighted with SelectedBg/SelectedFg
- [ ] `f` key toggles in-pane filter
- [ ] Filter matches track name and artist name (case-insensitive)
- [ ] Filter state shown in border label
- [ ] Existing queue functionality preserved (data from Store, refresh on tick)
- [ ] `make ci` passes

## Tasks
- [ ] Implement layout.Pane interface methods (ID, Title, ToggleKey, Actions)
      - test: ID() returns PaneQueue; Title() returns "Queue"; ToggleKey() returns 2; Actions() returns filter + add; compile-time check `var _ layout.Pane = &QueuePane{}`
- [ ] Replace list rendering with bubble-table dense table
      - test: 5 tracks -> 5 rows; playing track shows > indicator; column headers rendered; j/k navigation; SetSize updates dimensions; empty queue no panic; long names truncated
- [ ] Add filter support with `f` key toggle
      - test: `f` activates filter; typing filters by name/artist; Esc closes filter restores list; Actions() changes when filter active; empty filter shows all; no matches shows empty state
- [ ] Comprehensive tests -- Full lifecycle and edge case coverage
      - test: Full lifecycle test; queue updates on new QueueLoadedMsg; playing indicator persists; filter + scroll interaction; interface satisfaction; 200 items scrolling; 0 items clean render; long name truncation
