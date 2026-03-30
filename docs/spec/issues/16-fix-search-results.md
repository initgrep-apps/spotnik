# Feature 16 — Fix Search Results

> **Bug fix:** Search overlay accepts typing but no results appear.

## Root Cause (Needs Investigation)

The character-swallowing fix (j/k/a interception) was applied so typing works. But
results still don't appear — still shows "Type to search tracks, artists, albums..." after typing.

The debounce routing fix (commit f3a2a02) may work in tests but not in practice. Need to trace
the full pipeline end-to-end:

1. Is `searchDebounceMsg` reaching the overlay? (verify)
2. Is `SearchRequestMsg` being emitted by the overlay?
3. Is `buildSearchCmd` being called in `app.go`?
4. Is the Spotify search API returning results?
5. Is `SearchResultsMsg` reaching the overlay?
6. Is the store being populated?
7. Is `renderResults()` reading from store correctly?

**Information gap:** The debounce routing fix may work in tests but not in practice with
the actual app. Need to trace the full pipeline with real API calls.

---

## Fix

Investigate and fix the broken link in the chain. Add error states at each step.

1. **Trace the pipeline** — identify where results are lost
2. **Fix the broken link** — could be message routing, store update, or render logic
3. **Add error state** — if search API fails, show error instead of empty hint
4. **Add empty results state** — "No results for '{query}'"

---

## Files

- `internal/ui/panes/search.go` — Result rendering, error state
- `internal/app/app.go` — Message routing for search pipeline
- `internal/api/search.go` — Verify API call works
- Tests for full pipeline from keypress to results display

---

## Acceptance Criteria

- [ ] Typing in search overlay shows results within 400ms of last keypress
- [ ] Results grouped by Tracks, Artists, Albums, Playlists
- [ ] Error state shown if search API fails
- [ ] Empty results show "No results for '{query}'"
- [ ] Tests verify full pipeline from keypress to results display
