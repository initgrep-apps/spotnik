---
title: "Grid Layout Manager"
feature: 12-layout
status: done
---

## Background
The current UI uses a hardcoded 3-column layout assembled with lipgloss.JoinHorizontal() in render.go. There is no concept of pages, presets, pane toggling, or responsive grid reflow. The new DESIGN.md specifies a btop-inspired responsive grid system where 10 panes are organized across 2 pages, Page A has 4 presets, keys 1-8 toggle individual pane visibility, hidden panes redistribute space to visible siblings, and the grid recomputes on terminal resize and preset/toggle changes. This story builds the layout engine only -- no pane implementations, no rendering.

Design reference: docs/DESIGN.md sections 2, 3, 4, 16, 22.

## Design

### Core Types
- `PaneID` (10 values: PaneNowPlaying through PaneNetworkLog)
- `PageID` (PageA, PageB)
- `Rect` (X, Y, Width, Height with ContentWidth/ContentHeight methods)
- `Action` (Key, Label)
- `Pane` interface (tea.Model + SetSize, SetFocused, IsFocused, ID, Title, ToggleKey, Actions)
- `Cell` (PaneID, WidthWeight), `Row` (HeightWeight, Cells), `Preset` (Name, Visible map, Grid []Row)

### Presets
- PresetDashboard (8 panes, 3 rows with weights 2:3:3)
- PresetListening (3 panes, 2 rows with weights 3:2)
- PresetLibrary (4 panes, 2 rows with weights 1:4)
- PresetDiscovery (4 panes, 3 rows with weights 1:2:2)
- PresetNerdStatus (3 panes, 3 rows with weights 1:3:2)

### Space Distribution Algorithm
1. Get current preset's Grid
2. Build activeGrid by filtering hidden cells/empty rows
3. Compute contentHeight = height - headerHeight - statusHeight
4. Distribute height by HeightWeight (last row absorbs remainder)
5. Per row distribute width by WidthWeight (last cell absorbs remainder)
6. Compute Rect for each visible pane
7. Build focusOrder from visible panes in grid order

### PaneAt Hit-Test
Iterates rects map, checks if (x, y) falls within any Rect's bounds accounting for header height offset.

## Acceptance Criteria
- [ ] `internal/ui/layout/` package compiles independently
- [ ] `PaneID` enum has 10 values
- [ ] `Pane` interface defines all methods
- [ ] 4 Page A presets match DESIGN.md exactly
- [ ] 1 Page B preset matches DESIGN.md
- [ ] Space distribution produces rects that tile perfectly (no gaps, no overlap)
- [ ] Pane toggle redistributes space to siblings
- [ ] Row collapse works when all panes in a row are hidden
- [ ] Focus rotation skips hidden panes and wraps correctly
- [ ] Preset switch resets manual toggles
- [ ] PaneAt() correctly identifies pane from coordinates
- [ ] `make ci` passes

## Tasks
- [ ] Define PaneID, PageID enums and Rect struct in internal/ui/layout/pane.go
      - test: Rect.ContentWidth() and ContentHeight() correct; PaneID constants have expected iota values
- [ ] Define grid model and preset data structures in internal/ui/layout/presets.go
      - test: each preset's Visible map matches Grid cells; correct pane counts and row counts
- [ ] Implement LayoutManager with space distribution in internal/ui/layout/layout.go
      - test: NewManager starts on PageA preset 0; Resize computes rects; space distribution no gaps; height/width weights distribute correctly
- [ ] Implement page toggle, preset cycling, and pane toggling
      - test: TogglePage switches; CyclePreset cycles 0->1->2->3->0; TogglePane hides/restores; row collapse; cannot hide last pane
- [ ] Implement focus rotation
      - test: RotateFocus cycles visible panes; wraps; skips hidden; resets on preset change
- [ ] Add PaneAt hit-test for mouse support
      - test: click in pane returns pane; header returns -1; status bar returns -1
- [ ] Comprehensive layout integration tests
      - test: full lifecycle; resize; hide/collapse; preset cycle loop; Page B; focus after hiding; edge cases
