# Feature 14 — Fix Views Rendering (Stats + Playlists)

> **Bug fix:** Stats view shows empty data + duplicate status bar. Playlist view (3) also shows empty.

## Bugs Addressed

| # | Issue | Root Cause |
|---|---|---|
| B12 | Stats shows no data | API errors silently swallowed; no error state |
| B13 | Stats duplicate status bar | StatsView renders helpBar AND app renders statusBar |
| B16 | Playlist view (3) shows empty | Same class — data not loaded or errors swallowed |

---

## Root Cause Analysis

### B12 — Stats Empty Data
`buildFetchStatsCmd()` in `app.go`: if API call fails, returns `StatsLoadedMsg` without writing
to store. StatsView reads from store, finds nothing, shows "No listening data". No error feedback.

### B13 — Duplicate Status Bar
`stats.go` `renderDashboard()` calls `renderHelpBar()` which renders a row with key hints at the
bottom. The root app's `View()` ALSO renders `renderStatsStatusBar()`. Two rows of hints appear.

### B16 — Playlist View Empty
Same class of issue — pressing `3` opens playlist manager but data may not be loaded or API
errors are swallowed. Need to investigate `PlaylistManager` init and data loading path.

---

## Fix

1. **Remove `renderHelpBar()` from StatsView**
   - Only `app.go` owns the status bar (this should be a rule — see DESIGN.md update)

2. **Add error state to StatsView**
   - If fetch fails, show "Failed to load stats. Press f to retry."
   - Store API errors so StatsView can display them

3. **Fix PlaylistManager data loading**
   - Ensure data loads on view open (pressing `3`)
   - Surface errors if API calls fail

4. **Check all panes for duplicate help bars**
   - Remove any pane-level help/hint bars — only `app.go` renders status bar

---

## Files

- `internal/ui/panes/stats.go` — Remove helpBar, add error state
- `internal/ui/panes/playlists.go` — Fix data loading, add error state
- `internal/app/app.go` — Ensure errors propagated in stats/playlist messages
- `internal/state/store.go` — Error fields for stats/playlists if needed
- Tests for error state rendering, single status bar

---

## Acceptance Criteria

- [ ] Stats view has single status bar (no duplicate)
- [ ] Stats API errors shown as "Failed to load stats" with retry hint
- [ ] Playlist view (3) loads and displays playlists
- [ ] All panes that render help bars are removed — only app.go owns status bar
- [ ] Tests verify error state rendering and single status bar
