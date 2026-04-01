---
title: "Fix: Playlist tab border deformation from empty names"
feature: 18-search-redesign
status: open
---

## Background

The Playlists tab shows unstable borders — some rows have empty Name and Owner
fields (visible as blank rows 3 and 6 in screen 3). When a cell value is an empty
string, `bubble-table` renders a 0-width cell, causing the row to be narrower than
the header. The flex-width column layout shifts row-to-row, making borders appear
to bounce back and forth.

This is not a bug in the data pipeline — `TrackCount` and `Owner` are populated
correctly for playlists that have them. The issue is that some Spotify playlists
genuinely have empty or whitespace-only names (user-created playlists with no title),
and the rendering doesn't handle this gracefully.

## Design

### Fallback for empty cell values

In `refreshPlaylistRows()`, substitute a fallback for empty fields:

```go
func (o *SearchOverlay) refreshPlaylistRows() {
    playlists := o.store.SearchPlaylists()
    rows := make([]map[string]string, len(playlists))
    for i, p := range playlists {
        name := p.Name
        if strings.TrimSpace(name) == "" {
            name = "(untitled)"
        }
        owner := p.Owner
        if strings.TrimSpace(owner) == "" {
            owner = "—"
        }
        rows[i] = map[string]string{
            "index":  fmt.Sprintf("%d", i+1),
            "name":   name,
            "owner":  owner,
            "tracks": fmt.Sprintf("%d", p.TrackCount),
        }
    }
    o.tables[sectionPlaylists].SetRows(rows)
}
```

Apply the same pattern to all 4 `refreshXxxRows()` methods — any primary name
column that is empty gets a placeholder. This prevents 0-width cells from
destabilizing the column layout.

**Tracks:** `name` → "(untitled)", `artist` → "—"
**Artists:** `name` → "(unknown)"
**Albums:** `name` → "(untitled)", `artist` → "—"
**Playlists:** `name` → "(untitled)", `owner` → "—"

### Files Changed

| Action | File | Purpose |
|--------|------|---------|
| Modify | `internal/ui/panes/search.go` | Add empty-field fallbacks in all `refreshXxxRows` |
| Modify | `internal/ui/panes/search_test.go` | Test empty name handling |

## Acceptance Criteria

- [ ] Playlists with empty names render as "(untitled)" instead of blank
- [ ] Playlists with empty owners render as "—" instead of blank
- [ ] Column widths are stable across all rows (no border deformation)
- [ ] Same fallback applied to tracks, artists, and albums for consistency
- [ ] `make ci` passes

## Tasks

- [ ] **Add empty-field fallbacks to `refreshXxxRows`** — in all 4 refresh methods,
      check primary fields for empty/whitespace values and substitute placeholders.
      In `internal/ui/panes/search.go`.
      - test: `TestRefreshPlaylistRows_EmptyName_ShowsUntitled`
      - test: `TestRefreshPlaylistRows_EmptyOwner_ShowsDash`
      - test: `TestRefreshTrackRows_EmptyName_ShowsUntitled`
      - test: `TestRefreshAlbumRows_EmptyName_ShowsUntitled`
