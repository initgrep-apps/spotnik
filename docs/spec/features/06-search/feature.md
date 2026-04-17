---
title: "Search"
status: in-progress
---

## Description

Full-screen search overlay opened with `/`. Queries Spotify across four tabs (All / Songs / Artists / Albums / Playlists) with a 300ms debounce universal to all input events. Results render with rich metadata via custom list delegates — album art year, artist genre, playlist track count. Prefix autocomplete (`:songs`, `:artists`, `:albums`, `:playlists`) narrows search to a specific type and is promoted to a prompt tag. Pagination controls (Ctrl+←/→) page through results. Per-page context cancellation prevents stale results from earlier keystrokes overwriting newer ones. Store cleanup and message type refactors keep the data flow Elm-pure.

## Acceptance Criteria

- [ ] `/` opens search overlay; Escape closes it and resets state
- [ ] Debounce fires 300ms after last keypress — no per-keystroke API calls
- [ ] Results display across four tabs with rich metadata per result type
- [ ] Prefix autocomplete narrows to a single type and shows prompt tag
- [ ] Ctrl+←/→ pages through results for the current tab
- [ ] Stale in-flight requests cancelled when a new search supersedes them
- [ ] Enter plays selected item; Ctrl+a adds to queue
- [ ] Open: story 16 (search result fixes)
