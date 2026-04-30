---
title: "Fix Search Results"
feature: 05-search
status: closed
---

## Background
Search overlay accepts typing but no results appear. The character-swallowing fix (j/k/a interception) was applied so typing works. But results still don't appear -- still shows "Type to search tracks, artists, albums..." after typing. The debounce routing fix may work in tests but not in practice. Need to trace the full pipeline end-to-end:

1. Is `searchDebounceMsg` reaching the overlay?
2. Is `SearchRequestMsg` being emitted by the overlay?
3. Is `buildSearchCmd` being called in `app.go`?
4. Is the Spotify search API returning results?
5. Is `SearchResultsMsg` reaching the overlay?
6. Is the store being populated?
7. Is `renderResults()` reading from store correctly?

## Design

Investigate and fix the broken link in the chain. Add error states at each step.

1. **Trace the pipeline** -- identify where results are lost
2. **Fix the broken link** -- could be message routing, store update, or render logic
3. **Add error state** -- if search API fails, show error instead of empty hint
4. **Add empty results state** -- "No results for '{query}'"

### Files
- `internal/ui/panes/search.go` -- Result rendering, error state
- `internal/app/app.go` -- Message routing for search pipeline
- `internal/api/search.go` -- Verify API call works
- Tests for full pipeline from keypress to results display

## Acceptance Criteria
- [ ] Typing in search overlay shows results within 400ms of last keypress
- [ ] Results grouped by Tracks, Artists, Albums, Playlists
- [ ] Error state shown if search API fails
- [ ] Empty results show "No results for '{query}'"
- [ ] Tests verify full pipeline from keypress to results display

## Tasks
- [ ] Trace the search pipeline end-to-end and identify the broken link
      - test: Integration test verifying full pipeline: keypress -> debounce -> request -> API -> results -> render
- [ ] Fix the broken link in the search result chain
      - test: Search results appear after typing; error state shows on API failure
- [ ] Add error state rendering for failed searches
      - test: Search API error shows "Search failed. Try again." in Error() token color
- [ ] Add empty results state rendering
      - test: No results for query shows "No results for '{query}'"
