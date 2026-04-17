---
title: "P0 Correctness Fixes"
feature: 10-error-resilience
status: done
---

## Background
The architecture review (2026-03-23) identified three P0 issues that violate Elm Architecture invariants or leave dead code. None break compilation, but they create fragile behavior: overlay focus restoration is missing, View() has a purity violation in LibraryPane, and a dead function exists in player.go.

## Design

### 1. Restore prevFocus on overlay close
`closeSearch()` and `closeDeviceOverlay()` set their respective Open flags to false but never restore `a.focus = a.prevFocus`. This accidentally works because focus isn't changed while overlays are open, but any future code that modifies focus during an overlay would silently break.

Fix: In both `closeSearch()` and `closeDeviceOverlay()`, add `a.focus = a.prevFocus` after clearing the open flag.

### 2. Remove View() mutation in LibraryPane
`LibraryPane.View()` at line 400-401 calls `p.tree.UpdateFromStore(p.store)`, which is a write operation inside what should be a pure render function. The identical call already exists in `Update()` at line 356, so the View() call is redundant.

Fix: Remove the `p.tree.UpdateFromStore(p.store)` call from `View()`.

### 3. Remove dead nextRepeatMode() function
`nextRepeatMode()` in `player.go` (lines 230-240) is declared but never called. The actual repeat cycling is handled by `PlaybackRequestMsg{Action: ActionCycleRepeat}`.

Fix: Delete the `nextRepeatMode()` function entirely.

## Acceptance Criteria
- [ ] `closeSearch()` restores `a.focus = a.prevFocus`
- [ ] `closeDeviceOverlay()` restores `a.focus = a.prevFocus`
- [ ] `LibraryPane.View()` contains no calls to `UpdateFromStore`
- [ ] `nextRepeatMode()` function is removed from `player.go`
- [ ] All existing tests pass
- [ ] `make ci` passes

## Tasks
- [ ] Restore prevFocus on overlay close in internal/app/app.go
      - test: open search overlay with focus on queue -> close search -> verify focus returns to queue
      - test: open device overlay with focus on library -> close device overlay -> verify focus returns to library
- [ ] Remove View() mutation in internal/ui/panes/library.go
      - test: existing tests pass; View() returns valid output without calling Update() first
- [ ] Remove dead nextRepeatMode() function from internal/ui/panes/player.go
      - test: existing tests pass after removal; go vet clean
