---
title: "View Height Enforcement"
feature: 12-layout
status: done
---

## Background
Per DESIGN.md: "View() output MUST NOT exceed the height set by SetSize(). Panes with unbounded content must implement viewport scrolling." The architecture review found that LibraryPane, StatsView, and PlaylistManager render unbounded item lists without height capping. QueuePane already implements this correctly via `visibleTrackCount()` -- use it as the reference pattern.

## Design

### Reference Pattern: QueuePane
```go
func (q *QueuePane) visibleTrackCount() int {
    if q.height <= 0 {
        return 10
    }
    available := q.height - 14
    if available <= 0 {
        return 1
    }
    return available / 2
}
```
Tracks scrollOffset, only renders items from scrollOffset to scrollOffset + visibleTrackCount(). Scroll indicators (▲ top, ▼ bottom) show when content extends beyond visible window.

### Apply to LibraryPane
Add scrollOffset, implement visibleItemCount(), render only visible items, scroll follows cursor, ▲/▼ indicators.

### Apply to StatsView
Add scrollOffset (or per-section offsets), implement visibleItemCount(), slice displayed items, scroll indicators.

### Apply to PlaylistManager
Review existing height logic, add scrollOffset for track list, implement visibleTrackCount(), scroll indicators.

## Acceptance Criteria
- [ ] LibraryPane.View() output never exceeds SetSize height
- [ ] StatsView.View() output never exceeds SetSize height
- [ ] PlaylistManager.View() output never exceeds SetSize height
- [ ] All three panes show ▲/▼ scroll indicators when content overflows
- [ ] Cursor navigation scrolls the visible window
- [ ] Small terminal heights render at least 1 item per section
- [ ] All existing tests pass
- [ ] `make ci` passes

## Tasks
- [ ] Add height-capped rendering to LibraryPane in internal/ui/panes/library.go
      - test: View() line count <= SetSize height; scrolling advances scrollOffset; scroll indicators; small height renders at least 1 item
- [ ] Add height-capped rendering to StatsView in internal/ui/panes/stats.go
      - test: View() line count <= SetSize height; scrolling works; scroll indicators
- [ ] Add height-capped rendering to PlaylistManager in internal/ui/panes/playlists.go
      - test: View() line count <= SetSize height; track list scrolling; scroll indicators
