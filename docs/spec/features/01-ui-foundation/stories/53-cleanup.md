---
title: "Cleanup"
feature: 12-layout
status: done
---

## Background
After Features 40-52, dead code remains: old pane files (library.go, stats.go, playlists.go, player.go), potentially unused components (ProgressBar, VolumeBar, errorview), old enum values (viewMain, viewStats, viewPlaylists, focusedPane), old status bar hint methods, and docs/DESIGN_OLD.md. This feature cleans up all remaining dead code and updates documentation. This is the final feature in the redesign sequence.

## Design

### Task 1: Delete Old Pane Files
Delete library.go, stats.go, playlists.go and their tests after verifying no imports remain.

### Task 2: Delete Old Components
Check if ProgressBar, VolumeBar, errorview are still imported. Delete any with zero imports.

### Task 3: Clean Up Old Enums
Remove viewMain, viewStats, viewPlaylists, focusedPane, focusPlayer, focusLibrary, focusQueue, mainHints(), statsHints(), playlistsHints().

### Task 4: Delete DESIGN_OLD.md
Remove reference from DESIGN.md.

### Task 5: Update CLAUDE.md
Update design rules for grid layout, add layout/ to project tree, add bubbletea-overlay and bubble-table to tech stack.

### Task 6: Update Feature Overview
Add features 40-53 with dependencies and graph.

## Acceptance Criteria
- [ ] Old LibraryPane, StatsView, PlaylistManager files deleted
- [ ] Old unused components deleted
- [ ] Old viewMode/focusedPane enum remnants removed
- [ ] docs/DESIGN_OLD.md deleted
- [ ] CLAUDE.md updated for new layout system
- [ ] 00-overview.md includes features 40-53
- [ ] No references to deleted code anywhere in repo
- [ ] go build ./... succeeds
- [ ] make ci passes

## Tasks
- [ ] Delete old pane files (library.go, stats.go, playlists.go and tests)
      - test: go build succeeds; make ci passes
- [ ] Delete old components if unused (ProgressBar, VolumeBar, errorview)
      - test: go build succeeds after conditional deletion
- [ ] Clean up old enum values and dead code in app/
      - test: go build succeeds; make ci passes
- [ ] Delete DESIGN_OLD.md and update DESIGN.md reference
      - test: no file references DESIGN_OLD.md
- [ ] Update CLAUDE.md for new grid layout system
      - test: no references to "three-pane"; bubbletea-overlay and bubble-table listed
- [ ] Update feature overview with features 40-53
      - test: all 14 feature specs listed with dependencies
