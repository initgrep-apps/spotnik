---
title: "Always-Colorful Borders and Per-Column Table Colors"
feature: 16-vivid-themes
status: open
---

## Background
Currently, unfocused pane borders use `lipgloss.Faint(true)` which renders them as grey/washed-out regardless of the per-pane accent color. This makes the UI feel monochrome -- only the focused pane shows color. Additionally, all table columns across all panes use the same 3 colors (`TextMuted` for index, `TextPrimary` for main column, `TextSecondary` for supporting columns), making tables look like plain white/grey text.

This story changes border rendering to always show the per-pane accent color (dimmed when unfocused, full when focused) and updates all table panes to use the 4 new column color tokens introduced in story 70.

## Design

### Border Rendering Change

Modify `RenderPaneBorder()` in `internal/ui/layout/border.go`:

**Current behavior:**
```go
// Focused: full accent color
// Unfocused: Faint(true) -- grey regardless of accent
borderStyle := func(s string) string {
    if cfg.Focused {
        return lipgloss.NewStyle().Foreground(cfg.AccentColor).Render(s)
    }
    return lipgloss.NewStyle().Faint(true).Render(s)
}
```

**New behavior:**
```go
// Focused: full accent color
// Unfocused: accent color + Faint(true) -- dimmed but still colored
borderStyle := func(s string) string {
    style := lipgloss.NewStyle().Foreground(cfg.AccentColor)
    if !cfg.Focused {
        style = style.Faint(true)
    }
    return style.Render(s)
}
```

The key difference: `Faint(true)` is applied **on top of** the accent color, not instead of it. This produces a dimmed version of the accent color rather than flat grey. The terminal dims the color channel rather than replacing it.

Apply the same pattern to `titleStyle`:

```go
titleStyle := func(s string) string {
    style := lipgloss.NewStyle().Foreground(cfg.AccentColor)
    if cfg.Focused {
        style = style.Bold(true)
    } else {
        style = style.Faint(true)
    }
    return style.Render(s)
}
```

### Visual Result

```
Page A (before):
╭─ Now Playing ──────╮  ╭─ Queue ───────────╮  ╭─ Playlists ───────╮
│  (green border)     │  │  (grey border)    │  │  (grey border)    │
│  FOCUSED            │  │  unfocused        │  │  unfocused        │
╰────────────────────╯  ╰───────────────────╯  ╰───────────────────╯

Page A (after):
╭─ Now Playing ──────╮  ╭─ Queue ───────────╮  ╭─ Playlists ───────╮
│  (bright green)     │  │  (dim yellow)     │  │  (dim blue)       │
│  FOCUSED            │  │  unfocused        │  │  unfocused        │
╰────────────────────╯  ╰───────────────────╯  ╰───────────────────╯
```

Every pane always shows its identity color. Focus is distinguished by brightness + bold title.

### Per-Column Table Colors

Update every pane constructor that creates table columns to use the new column color tokens instead of `TextMuted`/`TextPrimary`/`TextSecondary`.

**Mapping:**

| Column Semantic | Old Token | New Token | Example |
|---|---|---|---|
| `#` index column | `th.TextMuted()` | `th.ColumnIndex()` | Row numbers |
| Main data column | `th.TextPrimary()` | `th.ColumnPrimary()` | Track name, Playlist name, Artist name |
| Supporting column | `th.TextSecondary()` | `th.ColumnSecondary()` | Artist (in track lists), Genre |
| Metadata column | `th.TextMuted()` | `th.ColumnTertiary()` | Duration, Year, Played time, Pop score, Track count |

### Panes to Update

| Pane | File | Columns → New Colors |
|---|---|---|
| QueuePane | `queue.go` | `#` → ColumnIndex, Track → ColumnPrimary, Artist → ColumnSecondary, Duration → ColumnTertiary |
| AlbumsPane | `albums_pane.go` | `#` → ColumnIndex, Name → ColumnPrimary, Artist → ColumnSecondary, Year → ColumnTertiary |
| LikedSongsPane | `likedsongs_pane.go` | `#` → ColumnIndex, Track → ColumnPrimary, Artist → ColumnSecondary, Duration → ColumnTertiary |
| TopTracksPane | `toptracks_pane.go` | `#` → ColumnIndex, Track → ColumnPrimary, Artist → ColumnSecondary, Pop → ColumnTertiary |
| TopArtistsPane | `topartists_pane.go` | `#` → ColumnIndex, Artist → ColumnPrimary, Genre → ColumnSecondary |
| RecentlyPlayedPane | `recentlyplayed_pane.go` | `#` → ColumnIndex, Track → ColumnPrimary, Artist → ColumnSecondary, Played → ColumnTertiary |
| PlaylistsPane (list) | `playlists_pane.go` | `#` → ColumnIndex, Name → ColumnPrimary, Tracks → ColumnTertiary |
| PlaylistsPane (tracks) | `playlists_pane.go` | `#` → ColumnIndex, Track → ColumnPrimary, Artist → ColumnSecondary, Duration → ColumnTertiary |
| NetworkLogPane | `networklog_pane.go` | TIME → ColumnIndex, METHOD → ColumnSecondary, ENDPOINT → ColumnPrimary, STATUS → ColumnTertiary, LATENCY → ColumnTertiary |
| RequestFlowPane | If it uses table component, same pattern |

### Example Change (QueuePane)

```go
// Before:
columns := []components.ColumnDef{
    {Key: "index", Header: "#", FlexFactor: 1, Color: th.TextMuted()},
    {Key: "track", Header: "Track", FlexFactor: 9, Color: th.TextPrimary()},
    {Key: "artist", Header: "Artist", FlexFactor: 7, Color: th.TextSecondary()},
    {Key: "duration", Header: "Duration", FlexFactor: 3, Color: th.TextMuted()},
}

// After:
columns := []components.ColumnDef{
    {Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
    {Key: "track", Header: "Track", FlexFactor: 9, Color: th.ColumnPrimary()},
    {Key: "artist", Header: "Artist", FlexFactor: 7, Color: th.ColumnSecondary()},
    {Key: "duration", Header: "Duration", FlexFactor: 3, Color: th.ColumnTertiary()},
}
```

### Files Changed

| Action | File | Purpose |
|---|---|---|
| Modify | `internal/ui/layout/border.go` | Replace `Faint(true)` fallback with accent+faint for unfocused |
| Modify | `internal/ui/panes/queue.go` | Use column color tokens |
| Modify | `internal/ui/panes/albums_pane.go` | Use column color tokens |
| Modify | `internal/ui/panes/likedsongs_pane.go` | Use column color tokens |
| Modify | `internal/ui/panes/toptracks_pane.go` | Use column color tokens |
| Modify | `internal/ui/panes/topartists_pane.go` | Use column color tokens |
| Modify | `internal/ui/panes/recentlyplayed_pane.go` | Use column color tokens |
| Modify | `internal/ui/panes/playlists_pane.go` | Use column color tokens (both list and track tables) |
| Modify | `internal/ui/panes/networklog_pane.go` | Use column color tokens |
| Modify | `internal/ui/layout/border_test.go` | Update tests for new border behavior |

## Acceptance Criteria
- [ ] Unfocused pane borders show their per-pane accent color (dimmed via Faint on top of Foreground), not flat grey
- [ ] Focused pane borders show full accent color with bold title
- [ ] `Faint(true)` is never used without a `Foreground(AccentColor)` in border rendering
- [ ] All 9 table panes use `ColumnIndex`/`ColumnPrimary`/`ColumnSecondary`/`ColumnTertiary` tokens
- [ ] No pane constructor uses `TextMuted`/`TextPrimary`/`TextSecondary` for table column colors
- [ ] `make ci` passes (lint + tests + 80% coverage)
- [ ] Visual verification: all 10 panes show distinct colored borders on both pages

## Tasks
- [ ] Modify `borderStyle` and `titleStyle` in `RenderPaneBorder()` to apply `Foreground(AccentColor)` for both focused and unfocused states, with `Faint(true)` only added for unfocused
      - test: `TestRenderPaneBorder_Unfocused_UsesAccentColor` (verify output contains ANSI escape for accent color, not just faint), `TestRenderPaneBorder_Focused_NoBoldRegression`
- [ ] Update QueuePane, AlbumsPane, LikedSongsPane, TopTracksPane, RecentlyPlayedPane column definitions to use `ColumnIndex`/`ColumnPrimary`/`ColumnSecondary`/`ColumnTertiary`
      - test: Existing pane tests still pass; new test `TestQueuePane_UsesColumnColors` verifying column defs use the correct token methods
- [ ] Update TopArtistsPane column definitions (2-column variant: index + name + genre)
      - test: `TestTopArtistsPane_UsesColumnColors`
- [ ] Update PlaylistsPane column definitions (both list view and track sub-view tables)
      - test: `TestPlaylistsPane_UsesColumnColors`
- [ ] Update NetworkLogPane column definitions
      - test: `TestNetworkLogPane_UsesColumnColors`
- [ ] Update `border_test.go` to reflect new unfocused border behavior
      - test: All existing border tests adapted to expect accent color in unfocused output
