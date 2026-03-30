---
title: "Mouse Scroll + Responsive Behavior"
feature: 12-layout
status: done
---

## Background
The current app does not handle mouse events. Scrolling requires keyboard focus (j/k) on the target pane. The minimum terminal size is 100x24. The new DESIGN.md specifies mouse wheel scrolling on any pane without changing focus (btop behavior), hit-test via LayoutManager.PaneAt(x, y) to identify the pane under the cursor, minimum terminal size increased to 120x30, and a friendly "needs more space" message when below minimum.

Design reference: docs/DESIGN.md sections 20, 21.

## Design

### Mouse Support
Add `tea.WithMouseCellMotion()` to program options. On MouseMsg with WheelUp/WheelDown, call layout.PaneAt(x, y), convert scroll to j/k key message, route to target pane WITHOUT changing focus. Only handle WheelUp/WheelDown, not clicks or drags. Ignore when overlay is open.

### Minimum Terminal Size
Constants: minTermWidth = 120, minTermHeight = 30. Message: `"Spotnik needs more space\n\nCurrent:  %d x %d\nRequired: %d x %d\n\nPlease resize your terminal and retry."` centered with rounded border.

## Acceptance Criteria
- [ ] tea.WithMouseCellMotion() enabled at app startup
- [ ] Mouse wheel up/down scrolls the pane under the cursor
- [ ] Mouse scroll does NOT change keyboard focus
- [ ] PaneAt() hit-test correctly identifies pane
- [ ] Mouse scroll ignored when overlay is open
- [ ] Minimum terminal size check uses 120x30
- [ ] "Needs more space" message shows current and required dimensions
- [ ] make ci passes

## Tasks
- [ ] Enable mouse support at startup in cmd/root.go
      - test: app starts with mouse support enabled
- [ ] Handle mouse scroll events in internal/app/app.go
      - test: wheel up on Playlists scrolls up, focus unchanged; wheel down on Queue; header/status bar no action; overlay open ignored
- [ ] Update minimum terminal size in render.go
      - test: 119x30 too small; 120x29 too small; 120x30 shows grid; message shows dimensions
- [ ] Integration and edge case tests
      - test: mouse scroll lifecycle; scroll during overlay; resize below/above minimum; dynamic resize; edge positions
