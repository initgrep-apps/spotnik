# Feature 12 — Fix Queue Overflow

> **Bug fix:** Long queue (19+ tracks) pushes content off-screen. Queue should scroll.

## Root Cause

`QueuePane.View()` renders ALL tracks in a loop without height capping. `SetSize(width, height)`
stores the height but never uses it in rendering. No scroll offset, no viewport window.

**Information gap:** Feature 06 spec did not specify scrolling behavior. DESIGN.md doesn't
specify queue scrolling. This is a missing design spec.

---

## Fix

1. **Add scrolling to QueuePane**
   - Add `scrollOffset` field to QueuePane
   - In `View()`, only render tracks within `[scrollOffset, scrollOffset+visibleLines]`
   - `j`/`k` navigation adjusts `scrollOffset` when cursor moves beyond visible window
   - Show scroll indicators (`▲`/`▼`) when content extends beyond view

2. **Repeat indicator (B9)**
   - When repeat-track mode is on (from store's playback state), show a repeat indicator
     in the queue header: `QUEUE [repeat track]` or similar
   - Read repeat state from store and display in queue header

---

## Files

- `internal/ui/panes/queue.go` — Scroll logic, height-capped rendering, repeat indicator
- `internal/ui/panes/queue_test.go` — Tests for scrolling and repeat label

---

## Acceptance Criteria

- [ ] Queue with 20+ tracks stays within pane height, scrollable with j/k
- [ ] Scroll indicators (▲/▼) shown when content extends beyond view
- [ ] When repeat-track is on, queue shows a repeat indicator in header
- [ ] Tests verify height-capped rendering and scroll behavior
