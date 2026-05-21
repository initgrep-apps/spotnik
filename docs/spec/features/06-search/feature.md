---
title: "Search"
status: done
---

## Description

Full-screen search overlay opened with `/`. Queries Spotify across four tabs (All / Songs / Artists / Albums / Playlists) with a 300ms debounce universal to all input events. Results render with rich metadata via custom list delegates — album art year, artist genre, playlist track count. Prefix autocomplete (`:songs`, `:artists`, `:albums`, `:playlists`) narrows search to a specific type and is promoted to a prompt tag. Pagination controls (Ctrl+←/→) page through results. Per-page context cancellation prevents stale results from earlier keystrokes overwriting newer ones. Store cleanup and message type refactors keep the data flow Elm-pure.

## Acceptance Criteria

- [x] `/` opens search overlay; Escape closes it and resets state
- [x] Debounce fires 300ms after last keypress — no per-keystroke API calls
- [x] Results display across four tabs with rich metadata per result type
- [x] Prefix autocomplete narrows to a single type and shows prompt tag
- [x] Ctrl+←/→ pages through results for the current tab
- [x] Stale in-flight requests cancelled when a new search supersedes them
- [x] Enter plays selected item; Ctrl+a adds to queue
- [x] Overlay renders as 2 panels (Search + Results) with no bottom keybar
- [x] Prefix hint pills removed; placeholder drives prefix discovery via pill Prompt
- [x] Results panel border shows action notches: ctrl+a queue, tab filter, pgdn next, pgup prev
