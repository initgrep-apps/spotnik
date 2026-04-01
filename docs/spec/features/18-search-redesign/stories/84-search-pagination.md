---
title: "Add Pagination to Search Overlay"
feature: 18-search-redesign
status: done
---

## Background

The search overlay shows up to 10 results per section (the Spotify API max per request),
but the API returns totals that can be much higher (e.g., "Tracks 16"). Users see the
total in the tab bar but have no way to access results beyond the first 10. The Spotify
Search API supports `offset` (0–1000) for pagination per `docs/API-CAPABILITY.md` §8.

Other panes in Spotnik use bubble-table's built-in `WithPageSize()` for pagination
(e.g., RecentlyPlayed shows "1/2"). The search overlay uses manual rendering, so it
needs its own pagination logic.

**Spotify Search API pagination support:**
- `offset` parameter: 0–1000
- `limit` parameter: 1–10 (reduced from 50 in Feb 2026)
- Each `type` paginated independently
- Total count available via `tracks.total`, `artists.total`, etc.

## Design

### Per-Section Pagination State

Each section tracks its own page independently. Users can browse Tracks page 2, switch
to Artists page 1, and come back to Tracks page 2.

Add to `SearchOverlay` struct:
```go
sectionOffsets [numSections]int  // per-section offset (multiples of 10)
```

Reset all offsets to 0 when a new query is submitted (on `SearchResultsMsg` where the
query changes).

### Cursor Boundary Navigation

When the cursor moves past the last/first row, load the next/previous page:

**Down past last row:**
```go
func (o *SearchOverlay) moveCursorDown() (tea.Model, tea.Cmd) {
    max := o.maxCursorForActiveSection() - 1
    if o.cursorPos < max {
        o.cursorPos++
        return o, nil
    }
    // At bottom — check if more pages exist
    total := o.totalForSection(o.activeSection)
    offset := o.sectionOffsets[o.activeSection]
    shown := o.maxCursorForActiveSection()
    if offset + shown < total {
        // More results exist — request next page
        return o, o.requestPage(offset + maxResultsPerSection)
    }
    return o, nil
}
```

**Up past first row:**
```go
func (o *SearchOverlay) moveCursorUp() (tea.Model, tea.Cmd) {
    if o.cursorPos > 0 {
        o.cursorPos--
        return o, nil
    }
    // At top — check if previous page exists
    offset := o.sectionOffsets[o.activeSection]
    if offset > 0 {
        return o, o.requestPage(offset - maxResultsPerSection)
    }
    return o, nil
}
```

When navigating to next page, set `cursorPos = 0`. When navigating to previous page,
set `cursorPos = maxCursorForActiveSection() - 1` (bottom of previous page).

### New Message Types

```go
// SearchPageRequestMsg is emitted when the user navigates past the current page boundary.
// The root app handles it by calling buildSearchCmd with the given offset.
type SearchPageRequestMsg struct {
    Query   string
    Offset  int
    Section searchSection  // which section triggered the page request
}
```

The root app handles `SearchPageRequestMsg` by dispatching `buildSearchCmd` with the
offset. The results come back as a normal `SearchResultsMsg`. The overlay updates its
section offset and results for the requesting section only.

### API Changes

**`SearchClient.Search()`** — add `offset int` parameter:

```go
// internal/api/search.go
func (s *SearchClient) Search(ctx context.Context, query string, types []string, limit, offset int) (*SearchResult, error)
```

Add `q.Set("offset", strconv.Itoa(offset))` to the URL params.

**`SearchAPI` interface** — update signature in `internal/api/search_interfaces.go`:
```go
type SearchAPI interface {
    Search(ctx context.Context, query string, types []string, limit, offset int) (*SearchResult, error)
}
```

**`MockSearch`** — update in `internal/api/apitest/mock.go`.

**`buildSearchCmd`** — add offset parameter, update caller in `app.go`:
```go
func (a *App) buildSearchCmd(query string, offset int) tea.Cmd
```

### Page Indicator in Help Bar

When total > `maxResultsPerSection`, show a page indicator in the help bar:

```
Tab next section  ↑↓ navigate  Enter play  Ctrl+A queue  Esc close  1-10 of 16
```

The indicator is right-aligned and shows `"{start}-{end} of {total}"`. When on page 2:
`"11-16 of 16"`.

```go
func (o *SearchOverlay) pageIndicator() string {
    total := o.totalForSection(o.activeSection)
    if total <= maxResultsPerSection {
        return ""
    }
    offset := o.sectionOffsets[o.activeSection]
    start := offset + 1
    end := offset + o.maxCursorForActiveSection()
    if end > total {
        end = total
    }
    return fmt.Sprintf("%d-%d of %d", start, end, total)
}
```

### Tab Switch Behavior

When switching tabs via Tab/Shift+Tab:
- Preserve the target section's offset (don't reset to 0)
- Set cursor to 0 (existing behavior)
- Results for the section at its current offset are already loaded

When submitting a new query:
- Reset all `sectionOffsets` to 0
- Reset `cursorPos` to 0 (existing behavior)

### Files Changed

| Action | File | Purpose |
|--------|------|---------|
| Modify | `internal/api/search.go` | Add `offset` parameter to `Search()` |
| Modify | `internal/api/search_interfaces.go` | Update `SearchAPI` interface |
| Modify | `internal/api/search_test.go` | Test offset parameter |
| Modify | `internal/api/apitest/mock.go` | Update `MockSearch` signature |
| Modify | `internal/ui/panes/messages.go` | Add `SearchPageRequestMsg` |
| Modify | `internal/ui/panes/search.go` | Add `sectionOffsets`, cursor boundary logic, page indicator, `requestPage` helper |
| Modify | `internal/ui/panes/search_test.go` | ~8 new tests |
| Modify | `internal/app/commands.go` | Update `buildSearchCmd` for offset |
| Modify | `internal/app/commands_test.go` | Test offset wiring |
| Modify | `internal/app/app.go` | Handle `SearchPageRequestMsg` in Update() |

## Acceptance Criteria

- [ ] `SearchClient.Search()` accepts an `offset` parameter and passes it to the API
- [ ] Arrow down past last row on a page loads the next page (if more results exist)
- [ ] Arrow up past first row on a page loads the previous page (if offset > 0)
- [ ] Cursor resets to 0 (next page) or last item (previous page) after page load
- [ ] Page indicator shows `"1-10 of N"` in help bar when total > 10
- [ ] Page indicator updates on page change (e.g., `"11-16 of 16"`)
- [ ] No page indicator when total <= 10
- [ ] Each section maintains its own page offset independently
- [ ] Tab switch preserves per-section page position
- [ ] New query resets all section offsets to 0
- [ ] Existing search tests pass (offset=0 for backward compatibility)
- [ ] `make ci` passes

## Tasks

- [ ] **Add offset to Search API** — add `offset int` parameter to `Search()`, update `SearchAPI` interface, update `MockSearch`, set `q.Set("offset", ...)` in the request. In `internal/api/search.go`, `search_interfaces.go`, `apitest/mock.go`.
      - test: `TestSearch_WithOffset` — verify offset query param sent to API
      - test: `TestSearch_ZeroOffset` — verify offset=0 behaves same as before

- [ ] **Update buildSearchCmd for offset** — add offset parameter, update caller in `app.go` Update() handler for `SearchRequestMsg`. Pass offset=0 for initial queries. In `internal/app/commands.go`, `app.go`.
      - test: `TestBuildSearchCmd_WithOffset` — verify offset passed through

- [ ] **Add pagination state to SearchOverlay** — add `sectionOffsets [numSections]int` field. Reset all offsets on new query. Add `requestPage(offset int)` helper that emits `SearchPageRequestMsg`.
      - test: `TestSearchOverlay_NewQuery_ResetsOffsets`

- [ ] **Add SearchPageRequestMsg** — new message type in `messages.go`. Root app handles it by dispatching `buildSearchCmd` with the section's offset. In `internal/ui/panes/messages.go`, `internal/app/app.go`.
      - test: `TestApp_SearchPageRequestMsg_DispatchesSearch`

- [ ] **Cursor boundary navigation** — modify `moveCursorDown` to emit page request when at bottom with more results. Modify `moveCursorUp` to emit page request when at top with offset > 0. Set cursor position after page load.
      - test: `TestMoveCursorDown_LastRow_EmitsPageRequest`
      - test: `TestMoveCursorDown_LastRow_NoMorePages_NoEmit`
      - test: `TestMoveCursorUp_FirstRow_EmitsPreviousPageRequest`
      - test: `TestMoveCursorUp_FirstRow_NoOffset_NoEmit`

- [ ] **Page indicator in help bar** — add `pageIndicator()` method. Append to help bar when total > 10. Right-align in the available space.
      - test: `TestRenderHelpBar_ShowsPageIndicator_WhenTotalExceeds10`
      - test: `TestRenderHelpBar_NoPageIndicator_WhenTotalUnder10`
      - test: `TestPageIndicator_Page2`

- [ ] **Tab switch preserves offset** — verify existing `moveSectionForward/Backward` preserves `sectionOffsets` (they should naturally, since only `cursorPos` is reset).
      - test: `TestSearchOverlay_TabSwitch_PreservesPageOffset`
