---
title: "Fix Queue Overflow"
feature: 06-queue
status: open
---

## Background
Long queue (19+ tracks) pushes content off-screen. Queue should scroll. `QueuePane.View()` renders ALL tracks in a loop without height capping. `SetSize(width, height)` stores the height but never uses it in rendering. No scroll offset, no viewport window. Feature 06 spec did not specify scrolling behavior. DESIGN.md doesn't specify queue scrolling. This is a missing design spec.

## Design

### 1. Add scrolling to QueuePane
- Add `scrollOffset` field to QueuePane
- In `View()`, only render tracks within `[scrollOffset, scrollOffset+visibleLines]`
- `j`/`k` navigation adjusts `scrollOffset` when cursor moves beyond visible window
- Show scroll indicators (`up-arrow`/`down-arrow`) when content extends beyond view

### 2. Repeat indicator (B9)
- When repeat-track mode is on (from store's playback state), show a repeat indicator in the queue header: `QUEUE [repeat track]` or similar
- Read repeat state from store and display in queue header

### Files
- `internal/ui/panes/queue.go` -- Scroll logic, height-capped rendering, repeat indicator
- `internal/ui/panes/queue_test.go` -- Tests for scrolling and repeat label

## Acceptance Criteria
- [ ] Queue with 20+ tracks stays within pane height, scrollable with j/k
- [ ] Scroll indicators shown when content extends beyond view
- [ ] When repeat-track is on, queue shows a repeat indicator in header
- [ ] Tests verify height-capped rendering and scroll behavior

## Tasks
- [ ] Add scrolling with scrollOffset and height-capped rendering
      - test: Queue with 20+ tracks stays within pane height; j/k adjusts scrollOffset; scroll indicators shown at boundaries
- [ ] Add repeat-track indicator in queue header
      - test: Repeat-track mode shows indicator in header; repeat off shows no indicator
