# Feature 46 — Queue Pane Migration

> **Feature:** Upgrade `QueuePane` to implement the `layout.Pane` interface,
> replace manual list rendering with bubble-table dense table format, and add
> in-pane filtering support.

## Context

The current `QueuePane` (`internal/ui/panes/queue.go`, ~6.9KB) renders the queue as
a manually formatted list with fixed-width columns using `fmt.Sprintf`. It has
`SetSize`, `SetFocused`, `IsFocused` methods but doesn't implement the full `layout.Pane`
interface (missing `ID`, `Title`, `ToggleKey`, `Actions`).

The new DESIGN.md (§2, §9) specifies:
- Queue is pane `2` on Page A with toggle key `2`
- Dense table format with columns: #, Track, Artist, Duration
- In-pane filtering with `f` key
- Currently playing track shown with `▶` indicator

**Design reference:** `docs/DESIGN.md` §2 (Pane Definitions — Queue), §9 (Dense Table Formatting —
column widths: # 5%, Track 45%, Artist 35%, Duration 15%), §8 (In-Pane Filtering)

**Depends on:** Feature 41 (Pane interface), Feature 43 (Table + Filter components)

---

## Design Diagram

```
Queue Pane (DESIGN.md §9):

╭─ ²Queue ─────────────────────────────── ᐅf filter ─ ᐅA add ╮
│  #   Track                    Artist              Duration   │
│  1   Lil Boo Thang            Paul Russell        3:12       │
│  2   Street Fighter           Kamasi Washington   5:44       │
│ ▶3   BIRDS OF A FEATHER       Billie Eilish       3:30       │
│  4   Peaches                  Justin Bieber       3:18       │
│  ▼ more below                                                │
╰──────────────────────────────────────────────────────────────╯

Column Widths: # 5% | Track 45% | Artist 35% | Duration 15%

Filter Active:
╭─ ²Queue ────────── filtering: "rock" ─── ᐅEsc close ╮
│  > rock█                                             │
│  3   Rocket Man              Elton John     4:52     │
│  7   Rock and Roll           Led Zeppelin   3:40     │
╰──────────────────────────────────────────────────────╯
```

---

## Task 1: Implement layout.Pane interface methods

**Problem:** QueuePane lacks `ID`, `Title`, `ToggleKey`, `Actions` methods.

**Fix:**

Add to `internal/ui/panes/queue.go`:

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

**Files:**
- Modify: `internal/ui/panes/queue.go`

**Tests:**
- Unit: `ID()` returns `PaneQueue`
- Unit: `Title()` returns "Queue"
- Unit: `ToggleKey()` returns 2
- Unit: `Actions()` returns filter + add
- Unit: Compile-time check: `var _ layout.Pane = &QueuePane{}`

**Commit:** `feat(ui): QueuePane implements layout.Pane interface`

---

## Task 2: Replace list rendering with bubble-table

**Problem:** Queue uses manual `fmt.Sprintf` formatting for each row.

**Fix:**

1. Add `table components.Table` field to `QueuePane`
2. Initialize in constructor with column definitions:
   ```go
   columns := []components.ColumnDef{
       {Key: "index", Header: "#", FlexFactor: 1, Color: theme.TextMuted()},
       {Key: "track", Header: "Track", FlexFactor: 9, Color: theme.TextPrimary()},
       {Key: "artist", Header: "Artist", FlexFactor: 7, Color: theme.TextSecondary()},
       {Key: "duration", Header: "Duration", FlexFactor: 3, Color: theme.TextMuted()},
   }
   ```
   (Flex factors approximate 5%/45%/35%/15% ratios)
3. On `QueueLoadedMsg`: convert `[]domain.Track` to table rows (`[]map[string]string`)
4. Set `PlayingIndex` to the index of the currently playing track
5. In `SetSize()`: call `table.SetSize(contentWidth, contentHeight)`
6. In `View()`: return `table.View()`
7. In `Update()`: forward key messages to `table.Update(msg)` when focused

**Files:**
- Modify: `internal/ui/panes/queue.go`

**Tests:**
- Unit: Queue with 5 tracks → table has 5 rows
- Unit: Currently playing track shows ▶ indicator
- Unit: Column headers rendered: #, Track, Artist, Duration
- Unit: j/k navigation changes selected row
- Unit: SetSize updates table dimensions
- Unit: Empty queue → table shows empty state (no panic)
- Unit: Track names truncated to column width (no overflow)

**Commit:** `feat(ui): QueuePane uses bubble-table for dense rendering`

---

## Task 3: Add filter support

**Problem:** Queue has no in-pane filtering.

**Fix:**

1. Add `filter *components.Filter` field to `QueuePane`
2. Initialize in constructor: `filter: components.NewFilter(theme)`
3. In `Update()`:
   - `f` key when focused → `filter.Toggle()`
   - When filter active, forward key messages to `filter.Update(msg)`
   - On filter query change → filter track data and update table rows
4. In `View()`:
   - If filter active, prepend `filter.View(contentWidth)` above table
   - Reduce table height by 1 to accommodate filter bar
5. Override `Actions()` when filter active to return `filter.BorderLabel()`:
   ```go
   func (q *QueuePane) Actions() []layout.Action {
       if q.filter.IsActive() {
           return []layout.Action{{Key: "Esc", Label: "close"}}
       }
       return []layout.Action{{Key: "f", Label: "filter"}, {Key: "A", Label: "add"}}
   }
   ```
6. Filter matches against track name and artist name (`filter.MatchesAny(track.Name, track.ArtistName)`)

**Files:**
- Modify: `internal/ui/panes/queue.go`

**Tests:**
- Unit: `f` key activates filter
- Unit: Typing in filter → tracks filtered by name/artist
- Unit: Filter "rock" → only tracks with "rock" in name or artist shown
- Unit: Esc closes filter, restores full list
- Unit: Filter active → Actions() changes to show close action
- Unit: Empty filter query → all tracks shown
- Unit: Filter with no matches → table shows empty state

**Commit:** `feat(ui): in-pane filtering for QueuePane`

---

## Task 4: Comprehensive tests

**Files:**
- Modify: `internal/ui/panes/queue_test.go`

**Tests:**
- Integration: Full lifecycle — construct → resize → load queue → filter → navigate → verify View
- Integration: Queue updates on new QueueLoadedMsg → table refreshes
- Integration: Playing track indicator persists across data updates
- Integration: Filter + scroll interaction — filter results, scroll, clear filter, position resets
- Integration: Interface satisfaction: `var _ layout.Pane = &QueuePane{}`
- Edge: Queue with 200 items → scrolling works, no performance issue
- Edge: Queue with 0 items → renders cleanly
- Edge: Track with very long name → truncated in column

**Commit:** `test(ui): comprehensive QueuePane tests with table and filter`

---

## Acceptance Criteria

- [ ] `QueuePane` satisfies `layout.Pane` interface
- [ ] Dense table format with 4 columns: #, Track, Artist, Duration
- [ ] Column widths approximate 5%/45%/35%/15% ratio
- [ ] Currently playing track shows `▶` indicator in index column
- [ ] Per-column colors: TextMuted, TextPrimary, TextSecondary, TextMuted
- [ ] Selected row highlighted with SelectedBg/SelectedFg
- [ ] `f` key toggles in-pane filter
- [ ] Filter matches track name and artist name (case-insensitive)
- [ ] Filter state shown in border label
- [ ] `j`/`k` scrolling works
- [ ] Existing queue functionality preserved (data from Store, refresh on tick)
- [ ] `make ci` passes

---

## Notes

- The QueuePane still reads data from `state.Store` — this doesn't change. The table
  is a presentation layer on top of the same data flow.
- The `A` (Shift+a) key for "add to queue" is listed in Actions but is handled at the
  app level (routing.go), not inside QueuePane. The pane just declares it for border display.
- bubble-table handles its own keyboard scrolling (j/k, up/down, page up/down). The pane
  forwards key messages to `table.Update(msg)` when focused and filter is not active.
- When filter is active, key messages go to the filter first. Only Esc and Enter
  close the filter — all other keys are consumed by the textinput.
