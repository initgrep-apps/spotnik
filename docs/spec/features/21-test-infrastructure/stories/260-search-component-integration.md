---
title: "Search component + integration tests"
feature: 21-test-infrastructure
status: open
---

## Background

Search is the most complex overlay — two-panel layout (Search + Results), 5-tab results,
prefix autocomplete with prompt tag, 300ms debounce, pagination (PgDn/PgUp), Enter-to-play,
Ctrl+A add-to-queue, Esc close with full state reset, stale request cancellation. Current
tests cover Update() state transitions but never verify the rendered overlay at each stage.

## Design

### Golden tests: `internal/ui/panes/search_golden_test.go`

- `TestSearchOverlay_View_Idle` — overlay open, no query, placeholder cycling, 2 panels visible
- `TestSearchOverlay_View_WithQuery` — "testing" typed, results panel empty, no results yet
- `TestSearchOverlay_View_Results` — results loaded, all 5 tabs, selected item highlighted
- `TestSearchOverlay_View_PrefixLocked` — `:songs` prefix locked, prompt tag "Search Songs"
- `TestSearchOverlay_View_Page2` — pagination bar shows "page 2", prev arrow active
- `TestSearchOverlay_View_NoResults` — query returned 0 results, empty state
- `TestSearchOverlay_View_Narrow` — 40×24, panel sizing adapted

### Integration test: `internal/app/search_flow_test.go`

```go
func TestSearchFlow_OpenTypeResultsPaginatePlayClose(t *testing.T) {
    // 1. Create app, Send '/' → assert search overlay open
    // 2. Type "test" → debounce → SearchRequestMsg sent
    // 3. Send SearchPageLoadedMsg{Results: [...]} → assert results in View()
    // 4. Send PgDn → page 2, PgUp → page 1
    // 5. Send Enter on track → PlayTrackListMsg produced, overlay stays open
    // 6. Send Esc → overlay closed, state reset
}

func TestSearchFlow_PrefixAutocomplete(t *testing.T) {
    // Type ":songs " → prefix locks, prompt tag changes
    // Type "hello" → search restricted to songs type
    // Backspace on empty → prefix unlocks
}

func TestSearchFlow_CtrlA_AddToQueue(t *testing.T) {
    // Cursor on track, Ctrl+A → AddToQueueMsg produced
}

func TestSearchFlow_TabCycling(t *testing.T) {
    // Tab → All→Songs→Artists→Albums→Playlists→All
    // Shift+Tab → reverse
}
```

## Files

### Create

- `internal/ui/panes/search_golden_test.go`
- `internal/app/search_flow_test.go`
- `internal/ui/panes/testdata/TestSearchOverlay_View_*.golden` (7 files)

## Acceptance Criteria

- [ ] Search overlay: 7 golden snapshots (idle, with query, results, prefix locked, page 2, no results, narrow)
- [ ] Integration: full lifecycle — open → type → debounce → results → paginate → play → close
- [ ] Integration: prefix autocomplete locks/unlocks correctly
- [ ] Integration: Ctrl+A produces AddToQueueMsg
- [ ] Integration: Tab/Shift+Tab cycles tabs
- [ ] Integration: Esc fully resets state (second open is fresh)
- [ ] `make ci` passes

## Tasks

- [ ] Create SearchOverlay golden tests (7 snapshots)
      - test: `TestSearchOverlay_View_Idle`, `TestSearchOverlay_View_WithQuery`, `TestSearchOverlay_View_Results`, `TestSearchOverlay_View_PrefixLocked`, `TestSearchOverlay_View_Page2`, `TestSearchOverlay_View_NoResults`, `TestSearchOverlay_View_Narrow`
- [ ] Create search flow integration test — full lifecycle
      - test: `TestSearchFlow_OpenTypeResultsPaginatePlayClose`
- [ ] Create search flow integration test — prefix + shortcuts
      - test: `TestSearchFlow_PrefixAutocomplete`, `TestSearchFlow_CtrlA_AddToQueue`, `TestSearchFlow_TabCycling`
- [ ] Generate golden files and verify all tests pass
