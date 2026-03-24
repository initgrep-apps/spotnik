# Feature 22 — app.go Decomposition

> **Refactoring:** Split the 1730-line `app.go` into focused files and collapse
> duplicate header/status bar renderers.

## Context

`app.go` is the largest file in the codebase at 1730 lines. The architecture review
identified three extractable sections:

- 18 `build*Cmd` functions (~350 lines) — pure command factories, no routing logic
- `View()` + all `render*` helpers (~500 lines) — pure rendering, no state mutation
- 3 duplicate header renderers + 3 duplicate status bar renderers — ~85% shared code

All extracted code stays in `internal/app/` package. No interface changes. This is a
mechanical move + deduplication.

---

## Task 1: Extract build*Cmd functions to commands.go

**What to move:** All 18 `build*Cmd` functions (lines 926-1730 approximately):
- `buildPlaybackAPICmd`, `buildPlayContextCmd`, `buildPlayTrackCmd`
- `buildFetchPlaylistsCmd`, `buildFetchAlbumsCmd`, `buildFetchLikedTracksCmd`
- `buildFetchRecentlyPlayedCmd`, `buildAddToQueueCmd`, `buildSearchCmd`
- `buildFetchDevicesCmd`, `buildTransferPlaybackCmd`, `buildToggleLikeCmd`
- `buildFetchStatsCmd`, `buildFetchPlaylistTracksCmd`, `buildCreatePlaylistCmd`
- `buildRenamePlaylistCmd`, `buildRemovePlaylistTrackCmd`, `buildReorderPlaylistTracksCmd`

Also move `fetchPlaybackStateCmd` and `fetchQueueCmd` if they exist as standalone functions.

**New file:** `internal/app/commands.go`

All functions are methods on `*App`, so they access the same struct fields. The move
is purely mechanical — cut from `app.go`, paste into `commands.go`, add the same
package declaration and imports.

**Files:**
- `internal/app/app.go` — Remove build*Cmd functions
- `internal/app/commands.go` (new) — All build*Cmd functions

**Tests:**
- All existing tests must pass unchanged (same package, same methods)
- `go build ./...` must compile clean

---

## Task 2: Extract View() and render helpers to render.go

**What to move:** The `View()` method and all `render*` helper functions:
- `View()` (the main render dispatcher)
- `renderHeader`, `renderStatsHeader`, `renderPlaylistsHeader`
- `renderStatusBar`, `renderStatsStatusBar`, `renderPlaylistsStatusBar`
- `renderSplash`, `renderAuthPanel`, `renderTooSmall`
- Any other `render*` private functions

**New file:** `internal/app/render.go`

Same mechanical move — all methods on `*App`, same package.

**Files:**
- `internal/app/app.go` — Remove View() and render* functions
- `internal/app/render.go` (new) — All rendering code

**Tests:**
- All existing tests must pass unchanged
- `go build ./...` must compile clean

---

## Task 3: Collapse duplicate header renderers

**Problem:** `renderHeader`, `renderStatsHeader`, and `renderPlaylistsHeader` share ~85%
of their code (header bar with app name, device indicator, time). The only difference
is the label shown (e.g., "Spotnik", "Spotnik [STATS]", "Spotnik [PLAYLISTS]").

**Fix:** Create a single `renderHeader(label string)` function that accepts the label
as a parameter. Replace all three callers:
- Main view: `renderHeader("")` or `renderHeader("Spotnik")`
- Stats view: `renderHeader("[STATS]")`
- Playlists view: `renderHeader("[PLAYLISTS]")`

Delete the two specialized variants.

**Files:** `internal/app/render.go` (after Task 2)

**Tests:**
- Unit test: renderHeader with different labels produces correct output
- Existing view tests continue to pass

---

## Task 4: Collapse duplicate status bar renderers

**Problem:** `renderStatusBar`, `renderStatsStatusBar`, and `renderPlaylistsStatusBar`
share most of their code. The only difference is the context-specific key hints.

**Fix:** Create a single `renderStatusBar(hints string)` or pass the hint items as
parameters. Replace all three callers with the unified function.

Delete the two specialized variants.

**Files:** `internal/app/render.go` (after Task 2)

**Tests:**
- Unit test: renderStatusBar with different hints produces correct output
- Existing view tests continue to pass

---

## Expected Result

After this feature:
```
internal/app/
├── app.go          (~600 lines) — struct, Init, Update, focus routing
├── commands.go     (~400 lines) — all build*Cmd functions
├── render.go       (~400 lines) — View() + unified render helpers
├── auth.go         (existing, unchanged)
├── splash.go       (existing, unchanged)
```

---

## Acceptance Criteria

- [ ] `app.go` is under 700 lines
- [ ] `commands.go` contains all build*Cmd functions
- [ ] `render.go` contains View() and all render helpers
- [ ] Only ONE `renderHeader` function exists (parameterized)
- [ ] Only ONE `renderStatusBar` function exists (parameterized)
- [ ] Zero duplicate render functions
- [ ] All existing tests pass unchanged
- [ ] `make ci` passes
