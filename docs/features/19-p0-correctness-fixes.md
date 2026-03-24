# Feature 19 — P0 Correctness Fixes

> **Refactoring:** Three correctness issues found in architecture review — overlay focus
> restoration, View() purity violation, and dead code removal.

## Context

The architecture review (2026-03-23) identified three P0 issues that violate Elm Architecture
invariants or leave dead code. None break compilation, but they create fragile behavior.

---

## Task 1: Restore prevFocus on overlay close

**Problem:** `closeSearch()` (app.go:297-300) and `closeDeviceOverlay()` (app.go:311-314)
set their respective `Open` flags to false but never restore `a.focus = a.prevFocus`.
`openDeviceOverlay()` correctly saves `a.prevFocus = a.focus` (line 304), and
`openSearch()` does the same. But closing discards the saved focus.

This accidentally works because focus isn't changed while overlays are open, but any
future code that modifies focus during an overlay would silently break.

**Fix:** In both `closeSearch()` and `closeDeviceOverlay()`, add `a.focus = a.prevFocus`
after clearing the open flag.

**Files:** `internal/app/app.go`

**Tests:**
- Unit test: open search overlay with focus on queue → close search → verify focus returns to queue
- Unit test: open device overlay with focus on library → close device overlay → verify focus returns to library

---

## Task 2: Remove View() mutation in LibraryPane

**Problem:** `LibraryPane.View()` at line 400-401 calls `p.tree.UpdateFromStore(p.store)`,
which is a write operation inside what should be a pure render function. The identical
call already exists in `Update()` at line 356, so the View() call is redundant.

Per CLAUDE.md: "`View()` must be pure — no external calls, no heavy computation, just
read state → string."

**Fix:** Remove the `p.tree.UpdateFromStore(p.store)` call from `View()`. The `Update()`
call at line 356 is sufficient — it runs before every render cycle.

**Files:** `internal/ui/panes/library.go`

**Tests:**
- Existing tests should continue to pass — the Update() call covers the data refresh
- Add a test that verifies View() returns valid output without calling Update() first
  (to confirm the mutation was genuinely redundant)

---

## Task 3: Remove dead nextRepeatMode() function

**Problem:** `nextRepeatMode()` in `player.go` (lines 230-240) is declared but never called.
The actual repeat cycling is handled by `PlaybackRequestMsg{Action: ActionCycleRepeat}`
dispatched to the root model at line 216.

**Fix:** Delete the `nextRepeatMode()` function entirely.

**Files:** `internal/ui/panes/player.go`

**Tests:**
- No new tests needed — verify existing tests pass after removal
- Run `go vet ./...` to confirm no references

---

## Acceptance Criteria

- [ ] `closeSearch()` restores `a.focus = a.prevFocus`
- [ ] `closeDeviceOverlay()` restores `a.focus = a.prevFocus`
- [ ] `LibraryPane.View()` contains no calls to `UpdateFromStore`
- [ ] `nextRepeatMode()` function is removed from `player.go`
- [ ] All existing tests pass
- [ ] `make ci` passes
