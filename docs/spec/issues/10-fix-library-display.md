# Feature 10 — Fix Library Display

> **Bug fix:** Library pane shows only section headers, no items visible.

## Root Cause

`NewLibraryTree()` creates all sections with `Expanded: false`. Data is fetched on init
(playlists, recently played) and stored, but sections remain collapsed. User must manually
press Enter on each section to see anything. The original spec says "Playlists visible
within 2 seconds of app start" — this was not implemented.

**Information gap:** Agent followed lazy-expand pattern for ALL sections, but the spec
intended Playlists to be visible immediately on load.

---

## Fix

- Auto-expand Playlists section when `LibraryLoadedMsg` arrives and items exist
- Auto-expand Recently Played section when `RecentlyPlayedLoadedMsg` arrives
- Keep Albums and Liked Songs collapsed (lazy load on expand)

---

## Files

- `internal/ui/panes/library.go` — Update handler for load messages to set `Expanded: true`
- `internal/ui/panes/library_test.go` — Tests for auto-expand behavior

---

## Acceptance Criteria

- [ ] Playlists section auto-expands when playlists load on init
- [ ] Recently Played section auto-expands when data arrives
- [ ] Albums and Liked Songs remain collapsed (lazy load)
- [ ] Items are visible within 2 seconds of app start (per original spec)
- [ ] Tests verify auto-expand behavior on LibraryLoadedMsg
