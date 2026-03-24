# Feature 26 — View Height Enforcement

> **Refactoring:** Add height-capped rendering with scroll indicators to LibraryPane,
> StatsView, and PlaylistManager, following the QueuePane pattern.

## Context

Per DESIGN.md: "View() output MUST NOT exceed the height set by SetSize(). Panes with
unbounded content must implement viewport scrolling."

The architecture review found that LibraryPane, StatsView, and PlaylistManager render
unbounded item lists without height capping. QueuePane already implements this correctly
via `visibleTrackCount()` — use it as the reference pattern.

### Reference Pattern: QueuePane (queue.go lines 231-240)

```go
func (q *QueuePane) visibleTrackCount() int {
    if q.height <= 0 {
        return 10 // default for tests
    }
    available := q.height - 14 // header + NOW + dividers + footer
    if available <= 0 {
        return 1
    }
    return available / 2 // 2 lines per track
}
```

The queue pane tracks a `scrollOffset` and only renders items from
`scrollOffset` to `scrollOffset + visibleTrackCount()`. Scroll indicators
(`▲` at top, `▼` at bottom) show when content extends beyond the visible window.

---

## Task 1: Add height-capped rendering to LibraryPane

**Problem:** LibraryPane renders all expanded section items without checking height.
A user with 200+ playlists will overflow the pane.

**Fix:**
1. Add `scrollOffset int` field to `LibraryPane` struct
2. Implement `visibleItemCount()` following the QueuePane pattern — subtract fixed
   UI elements (section headers, dividers, padding) from `p.height`
3. In `View()`, only render items from `scrollOffset` to
   `scrollOffset + visibleItemCount()`
4. In `Update()`, adjust `scrollOffset` on j/k navigation when cursor moves past
   the visible window (scroll follows cursor)
5. Render `▲` indicator at top when `scrollOffset > 0`
6. Render `▼` indicator at bottom when more items exist below the visible window

**Files:**
- `internal/ui/panes/library.go` — Add scroll logic + height capping

**Tests:**
- Unit test: View() output line count does not exceed SetSize height
- Unit test: scrolling down past visible window advances scrollOffset
- Unit test: scroll indicators appear when content overflows
- Unit test: small height still renders at least 1 item

---

## Task 2: Add height-capped rendering to StatsView

**Problem:** StatsView renders top tracks and top artists lists without height capping.
With 50 items per list, the view overflows.

**Fix:**
1. Add `scrollOffset int` field to `StatsView` struct (or per-section offsets)
2. Implement `visibleItemCount()` — subtract headers, time range selector, dividers
3. In `View()`, slice displayed items to the visible window
4. In `Update()`, adjust scrollOffset on j/k navigation
5. Add scroll indicators

**Files:**
- `internal/ui/panes/stats.go` — Add scroll logic + height capping

**Tests:**
- Unit test: View() output line count does not exceed SetSize height
- Unit test: scrolling works within stats sections
- Unit test: scroll indicators appear when content overflows

---

## Task 3: Add height-capped rendering to PlaylistManager

**Problem:** PlaylistManager has some height management (lipgloss Height()) but the
track list within a playlist can still overflow when a playlist has many tracks.

**Fix:**
1. Review existing height logic (lines 645-758) — it uses lipgloss Height() for
   the outer container but may not cap the inner track list
2. Add `scrollOffset` for the track list if not already present
3. Implement `visibleTrackCount()` for the track list panel
4. Only render visible tracks
5. Add scroll indicators

**Files:**
- `internal/ui/panes/playlists.go` — Add/fix scroll logic for track list

**Tests:**
- Unit test: View() output line count does not exceed SetSize height
- Unit test: track list scrolling works correctly
- Unit test: scroll indicators appear when tracks overflow

---

## Acceptance Criteria

- [ ] LibraryPane.View() output never exceeds SetSize height
- [ ] StatsView.View() output never exceeds SetSize height
- [ ] PlaylistManager.View() output never exceeds SetSize height
- [ ] All three panes show `▲`/`▼` scroll indicators when content overflows
- [ ] Cursor navigation scrolls the visible window
- [ ] Small terminal heights render at least 1 item per section
- [ ] All existing tests pass
- [ ] `make ci` passes
