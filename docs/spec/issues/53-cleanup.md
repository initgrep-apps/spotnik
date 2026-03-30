# Feature 53 — Cleanup

> **Feature:** Remove dead code from the old layout system, delete `DESIGN_OLD.md`,
> update project documentation, and finalize the feature overview.

## Context

After Features 40-52, the following code and files are dead:
- `internal/ui/panes/library.go` — replaced by PlaylistsPane, AlbumsPane, LikedSongsPane
- `internal/ui/panes/stats.go` — replaced by TopTracksPane, TopArtistsPane, RecentlyPlayedPane
- `internal/ui/panes/playlists.go` — merged into PlaylistsPane
- `internal/ui/panes/player.go` — renamed to nowplaying.go (should already be gone from F45)
- Old test files for deleted panes
- `viewMain`, `viewStats`, `viewPlaylists` enum values (if any remain)
- `focusedPane` enum (if any remains)
- `renderPaneWithBorder()` (if not already removed in F49)
- Old status bar hint methods
- `docs/DESIGN_OLD.md` — archived old three-column design

This feature cleans up all remaining dead code and updates documentation.

**Depends on:** All prior features (40-52) complete.

---

## Task 1: Delete old pane files

**Problem:** Old pane implementations are no longer imported but still exist on disk.

**Fix:**

Delete these files (verify they have no imports first):

```
internal/ui/panes/library.go          ← replaced by playlists_pane.go, albums_pane.go, likedsongs_pane.go
internal/ui/panes/library_test.go
internal/ui/panes/stats.go            ← replaced by toptracks_pane.go, topartists_pane.go, recentlyplayed_pane.go
internal/ui/panes/stats_test.go
internal/ui/panes/playlists.go        ← merged into playlists_pane.go
internal/ui/panes/playlists_test.go
```

Before deleting, verify:
1. `grep -r "library.go\|LibraryPane\|NewLibraryPane" internal/` — no remaining imports
2. `grep -r "stats.go\|StatsView\|NewStatsView" internal/` — no remaining imports
3. `grep -r "playlists.go\|PlaylistManager\|NewPlaylistManager" internal/` — no remaining imports
4. `go build ./...` succeeds after deletion

**Files:**
- Delete: `internal/ui/panes/library.go`
- Delete: `internal/ui/panes/library_test.go`
- Delete: `internal/ui/panes/stats.go`
- Delete: `internal/ui/panes/stats_test.go`
- Delete: `internal/ui/panes/playlists.go`
- Delete: `internal/ui/panes/playlists_test.go`

**Tests:**
- Build: `go build ./...` succeeds
- Test: `make ci` passes

**Commit:** `refactor(ui): delete old LibraryPane, StatsView, PlaylistManager`

---

## Task 2: Delete old components if unused

**Problem:** Old `ProgressBar` and `VolumeBar` may be unused after NowPlaying migration.

**Fix:**

Check if `internal/ui/components/progress.go` and `volume.go` are still imported.
If not, delete them and their tests.

Also check `internal/ui/components/errorview.go` — if error rendering now goes through
toast notifications exclusively, this may be dead code.

```
grep -r "ProgressBar\|NewProgressBar" internal/
grep -r "VolumeBar\|NewVolumeBar" internal/
grep -r "RenderError\|errorview" internal/
```

Delete any that have zero imports.

**Files:**
- Conditionally delete: `internal/ui/components/progress.go` + test
- Conditionally delete: `internal/ui/components/volume.go` + test
- Conditionally delete: `internal/ui/components/errorview.go` + test

**Tests:**
- Build: `go build ./...` succeeds
- Test: `make ci` passes

**Commit:** `refactor(ui): remove unused old components`

---

## Task 3: Clean up old enum values and dead code in app/

**Problem:** Remnants of old view/focus system may exist.

**Fix:**

Search for and remove:
- `viewMain` constant (if still defined)
- `viewStats` constant
- `viewPlaylists` constant
- `focusPlayer`, `focusLibrary`, `focusQueue` constants
- `focusedPane` type definition
- `mainHints()`, `statsHints()`, `playlistsHints()` methods
- Any switch cases referencing deleted constants
- Old field comments referencing deleted panes

**Files:**
- Modify: `internal/app/app.go`
- Modify: `internal/app/render.go`
- Modify: `internal/app/routing.go`

**Tests:**
- Build: `go build ./...` succeeds
- Test: `make ci` passes

**Commit:** `refactor(app): remove old viewMode/focusedPane remnants`

---

## Task 4: Delete DESIGN_OLD.md

**Problem:** The archived old design document is no longer needed now that the new
design is fully implemented.

**Fix:**

1. Delete `docs/DESIGN_OLD.md`
2. Update `docs/DESIGN.md` header — remove reference to DESIGN_OLD.md:
   - Change: `The previous frozen three-column layout is archived in DESIGN_OLD.md.`
   - To: `The previous frozen three-column layout has been fully replaced.`
3. Update the comparison table header if it still references DESIGN_OLD.md

**Files:**
- Delete: `docs/DESIGN_OLD.md`
- Modify: `docs/DESIGN.md`

**Tests:**
- Verify: No file in the repo references `DESIGN_OLD.md`

**Commit:** `docs: remove DESIGN_OLD.md (redesign complete)`

---

## Task 5: Update CLAUDE.md

**Problem:** CLAUDE.md references the old three-pane layout and old design rules.

**Fix:**

Update `CLAUDE.md`:

1. **Design Rules section:**
   - Change: "Three-pane layout is frozen — Library | Player | Queue, never change this"
   - To: "Grid layout managed by LayoutManager — 10 panes across 2 pages, configured via presets"

2. **Project Layout section:**
   - Add `internal/ui/layout/` to the tree
   - Update pane descriptions

3. **Keybinding references:**
   - Update or reference DESIGN.md §17 keybinding table

4. **Dependencies:**
   - Add `bubbletea-overlay` and `bubble-table` to tech stack table

**Files:**
- Modify: `CLAUDE.md`

**Tests:**
- Verify: No references to "three-pane" in CLAUDE.md
- Verify: bubbletea-overlay and bubble-table listed in tech stack

**Commit:** `docs: update CLAUDE.md for new grid layout system`

---

## Task 6: Update feature overview

**Problem:** `docs/features/00-overview.md` doesn't include features 40-53.

**Fix:**

Add to the feature table:

```markdown
## UI Redesign Execution Order

Features from the btop-inspired UI redesign (2026-03-26). See `docs/DESIGN.md` for
the full specification.

| # | Feature | Spec | Status | Depends On | PR |
|---|---------|------|--------|-----------|-----|
| 40 | Theme Enhancement | `40-theme-enhancement.md` | | — | |
| 41 | Layout Infrastructure | `41-layout-infrastructure.md` | | — | |
| 42 | Custom Border Renderer | `42-custom-border-renderer.md` | | 40 | |
| 43 | Reusable Components | `43-reusable-components.md` | | 40 | |
| 44 | Visualizer + Gradient Bars | `44-visualizer-gradient-bars.md` | | 40 | |
| 45 | NowPlaying Pane | `45-nowplaying-pane.md` | | 41,42,44 | |
| 46 | Queue Pane Migration | `46-queue-pane-migration.md` | | 41,43 | |
| 47 | Library Split | `47-library-split.md` | | 41,43 | |
| 48 | Stats Split | `48-stats-split.md` | | 41,43 | |
| 49 | App Migration | `49-app-migration.md` | | 40-48 | |
| 50 | Header + Status Bar + Overlays | `50-header-statusbar-overlays.md` | | 42,49 | |
| 51 | Page B: Nerd Status | `51-page-b-nerd-status.md` | | 41-43,49 | |
| 52 | Mouse Scroll + Responsive | `52-mouse-scroll-responsive.md` | | 41,49 | |
| 53 | Cleanup | `53-cleanup.md` | | 40-52 | |
```

Add dependency graph:

```
F40 (Theme) ──┐
F41 (Layout) ─┤
F42 (Border) ─┼──→ F45 (NowPlaying) ──┐
F43 (Components)┘   F46 (Queue) ──────┤
                     F47 (LibSplit) ───┼──→ F49 (App Migration) ──→ F50 (Header)
F44 (Visualizer) → F45                ┤                             F51 (Page B)
                     F48 (StatsSplit) ─┘                            F52 (Mouse)
                                                                    F53 (Cleanup)
```

Add versioning:
```markdown
| v4.0.0 | Features 40-53 (btop-inspired UI redesign) |
```

**Files:**
- Modify: `docs/features/00-overview.md`

**Tests:**
- Verify: All 14 feature specs listed with correct dependencies
- Verify: Dependency graph matches feature specs

**Commit:** `docs: add features 40-53 to overview (UI redesign)`

---

## Acceptance Criteria

- [ ] Old `LibraryPane`, `StatsView`, `PlaylistManager` files deleted
- [ ] Old unused components deleted (if confirmed unused)
- [ ] Old `viewMode`/`focusedPane` enum remnants removed
- [ ] `docs/DESIGN_OLD.md` deleted
- [ ] `CLAUDE.md` updated for new layout system
- [ ] `00-overview.md` includes features 40-53 with dependencies and graph
- [ ] No references to `DESIGN_OLD.md` anywhere in repo
- [ ] No references to deleted panes/enums anywhere in repo
- [ ] `go build ./...` succeeds
- [ ] `make ci` passes
- [ ] Version table includes v4.0.0

---

## Notes

- This is the final feature in the redesign sequence. After this, Spotnik runs on the
  new btop-inspired grid layout with 10 panes across 2 pages.
- Run `grep -r` searches before each deletion to ensure nothing references the deleted code.
- If `make ci` coverage drops below 80% after deleting old test files, add tests to new
  panes to compensate. The new panes should already have thorough test coverage from F45-F51.
- The old `player.go` file should already be gone (renamed in F45). If it still exists,
  delete it here.
