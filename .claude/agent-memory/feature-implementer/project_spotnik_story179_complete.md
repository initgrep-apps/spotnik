---
name: project_spotnik_story179_complete
description: Story 179 (Page B Toggle Keys): preset-membership guard, pageBToggleKeyMap, NowPlaying-on-both-pages gotcha
type: project
---

## Story 179 — Page B Toggle Keys Fix

**What was built:**
- `pageBToggleKeyMap` in `routing.go` mapping `'1'-'5'` to Page B pane IDs
- Toggle routing block in `handleKeyMsg` is now page-aware: selects pageBToggleKeyMap when on Page B
- `TogglePane` blanket Page B early-return removed; preset-membership is the sole authority
- `Layout() *layout.Manager` accessor added to `App` for test access
- Keybinding docs updated in all 3 required locations (keybinding.md, DESIGN.md §17, help_overlay.go)
- 5 new layout tests + 1 routing test

**Key files:**
- `internal/app/routing.go` — `pageBToggleKeyMap` var + page-aware dispatch in `handleKeyMsg`
- `internal/ui/layout/layout.go` — `TogglePane` guard simplified to preset check only
- `internal/app/app.go` — `Layout()` accessor added after `Store()`
- `internal/ui/layout/layout_test.go` — 5 new toggle tests

**Critical gotcha — NowPlaying on both pages:**
The spec's suggested guard code (`id < PaneNetworkLog` rejected on Page B) would block NowPlaying
(id=0) from being toggled on Page B. But `PresetNerdStatus` includes `PaneNowPlaying: true`, and
key '1' is mapped to PaneNowPlaying in both maps. The correct fix is to DROP the page-ID guard
entirely and rely solely on preset-membership check. `PaneNowPlaying` is in both presets so it
passes on both pages; Page A-only panes are not in PresetNerdStatus so they're rejected on Page B.

**Testing pattern:**
- `TestTogglePane_PageB_CannotHideLastPane` hides 4 of 5 Page B panes then verifies the last toggle is rejected
- The `newTestApp()` helper in `staleness_test.go` takes no args; routing_test.go uses `app.New()` directly

**Phantom file gotcha:**
Wrote a temp test file to $TMPDIR, then added it via `git add` before deleting it. The file ended up
in the git index even after disk deletion. `git rm --cached` required to clean up. Always verify
`git status` shows clean before committing.
