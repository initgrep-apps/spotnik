---
title: "Search List Item Focus Styling & 3-Line Layout"
feature: 19-search-redesign
status: open
---

## Background

Post-implementation testing of the search overlay revealed two UX issues with the list delegate:

1. **Focus visibility** — When navigating search results, only the title text changes color on the selected item. There is no full-item visual indicator. The bubbles `DefaultDelegate` uses a left border bar (`│`) + subtle background on the entire item, which is a much stronger affordance. The user provided a reference screenshot showing this pattern.

2. **Data density** — Items are currently 2 lines (title + subtitle). With all the rich metadata from Story 87, the subtitle line is cramped. A 3-line layout allows bold title, clean subtitle, and a separate description row for metadata — each with distinct styling.

### Bubbles DefaultDelegate Pattern (Reference)

From `bubbles@v1.0.0/list/defaultitem.go`, the focus styling pattern is:

```go
// Selected: left-only │ border + accent color + 1-space padding
SelectedTitle = lipgloss.NewStyle().
    Border(lipgloss.NormalBorder(), false, false, false, true).
    BorderForeground(accentColor).
    Foreground(accentColor).
    Padding(0, 0, 0, 1)

// Normal: no border, 2-space padding (aligns with border + padding)
NormalTitle = lipgloss.NewStyle().
    Padding(0, 0, 0, 2)
```

Both title and description lines receive the same style, so the left border spans the full item height.

### Current vs Desired

| Aspect | Current | Desired |
|---|---|---|
| Item height | 2 lines | 3 lines |
| Title | Normal weight, name only changes on focus | **Bold** always |
| Focus indicator | `SelectedBg`/`SelectedFg` on name text only | Full-width background + left `│` border on all 3 lines |
| Unfocused indent | 2-space `"  "` prefix on line 2 | `Padding(0,0,0,2)` on all lines (aligns with border+pad) |
| Data layout | 2 rows — cramped | 3 rows — title / subtitle / description |

## Design

### Line Wrapping Approach

Instead of styling individual tokens and concatenating, each complete line is wrapped in a full-width style that handles selection state:

**File: `internal/ui/panes/search_delegate.go`**

Add a `wrapLine()` helper:

```go
// wrapLine applies full-width styling to a content line.
// Selected items get a left border bar + background; normal items get left padding.
func (d SearchItemDelegate) wrapLine(content string, width int, selected bool) string {
    if selected {
        return lipgloss.NewStyle().
            Width(width - 2). // account for border + padding
            Border(lipgloss.NormalBorder(), false, false, false, true).
            BorderForeground(d.theme.ActiveBorder()).
            Background(d.theme.SelectedBg()).
            Foreground(d.theme.SelectedFg()).
            Padding(0, 0, 0, 1).
            Render(content)
    }
    return lipgloss.NewStyle().
        Width(width).
        Padding(0, 0, 0, 2).
        Render(content)
}
```

### Height Change

```go
func (d SearchItemDelegate) Height() int { return 3 }
```

### 3-Line Layout Per Category

**Track:**
```
Line 1: ♪ Song Title ..................... [E] 3:42    ← badge + bold name + right-aligned metadata
Line 2: Artist1, Artist2                               ← ColumnSecondary
Line 3: Album Name                                     ← ColumnTertiary
```

**Artist:**
```
Line 1: ★ Artist Name                                  ← badge + bold name
Line 2: pop, r&b, alternative                          ← ColumnSecondary (genres)
Line 3: 12.4M followers · Pop: 97                      ← TextMuted (followers + popularity)
```

**Album:**
```
Line 1: ◎ Album Name ................. Album · 2024    ← badge + bold name + right-aligned type+year
Line 2: Artist1, Artist2                               ← ColumnSecondary
Line 3: 13 tracks                                      ← TextMuted
```

**Playlist:**
```
Line 1: ▤ Playlist Name ................ 245 tracks    ← badge + bold name + right-aligned count
Line 2: by Owner                                       ← ColumnSecondary
Line 3: Description text...                            ← TextMuted + Italic
```

### Render Function Refactor

Each `render*()` function follows this pattern:

```go
func (d SearchItemDelegate) renderTrack(w io.Writer, si SearchListItem, selected bool, width int) {
    badge := d.styledBadge(si.Category)

    // Build line 1 content: badge + bold name + right metadata
    // (token styling uses category/semantic colors, NOT selection colors)
    ...
    line1Content := badge + " " + boldName + padding + rightMeta

    // Build line 2 content: artists
    line2Content := artistStyle.Render(si.ArtistNames)

    // Build line 3 content: album
    line3Content := albumStyle.Render(si.AlbumName)

    // Wrap each line with selection-aware full-width styling
    fmt.Fprintf(w, "%s\n%s\n%s\n",
        d.wrapLine(line1Content, width, selected),
        d.wrapLine(line2Content, width, selected),
        d.wrapLine(line3Content, width, selected))
}
```

### styledName Change

Remove `SelectedBg`/`SelectedFg` from `styledName()` — selection is now handled at the line level. Always apply `Bold(true)`:

```go
func (d SearchItemDelegate) styledName(name string, _ bool, _ int) string {
    return lipgloss.NewStyle().
        Foreground(d.theme.TextPrimary()).
        Bold(true).
        Render(name)
}
```

Note: When selected, `wrapLine()` applies `Foreground(SelectedFg)` to the whole line, which will override the token-level foreground. Badge colors may be overridden by the line-level foreground on selected items — this is intentional (matches bubbles default behavior where selected items use a uniform accent color).

### Token Styling Within Lines

For **unselected** items, individual tokens retain their semantic colors (badge=category color, artists=ColumnSecondary, album=ColumnTertiary, metadata=TextMuted).

For **selected** items, `wrapLine()` sets `Foreground(SelectedFg)` on the whole line. If token colors should be preserved on selection (e.g., badge color stays), use `Inline(true)` on the token style so it isn't overridden. Otherwise, let the line-level foreground win for a clean selected look.

Design decision: **Let line-level foreground win** — selected items use uniform `SelectedFg` text. This matches the bubbles default delegate behavior and provides a clean, obvious selection indicator. The left border bar color (from `ActiveBorder()`) provides enough accent.

## Acceptance Criteria

- [ ] List items render in 3 lines: bold title, subtitle, description
- [ ] Selected item has a left `│` border bar colored with `theme.ActiveBorder()`
- [ ] Selected item has `SelectedBg()` background on all 3 lines
- [ ] Selected item text uses `SelectedFg()` foreground on all 3 lines
- [ ] Unselected items have 2-space left padding (aligning with border+padding)
- [ ] Title line is always bold
- [ ] Track: L1=♪ name + [E] + duration, L2=artists, L3=album
- [ ] Artist: L1=★ name, L2=genres, L3=followers + popularity
- [ ] Album: L1=◎ name + type + year, L2=artists, L3=track count
- [ ] Playlist: L1=▤ name + tracks, L2=by owner, L3=description
- [ ] Theme switching updates focus colors (border, background, foreground)
- [ ] `make ci` passes

## Tasks

- [ ] Add `wrapLine()` helper to `SearchItemDelegate` in `search_delegate.go`; change `Height()` to 3
      - test: `Height()` returns 3; `wrapLine` with `selected=true` output contains `│` border character; `selected=false` output has 2-space left padding
- [ ] Refactor `styledName()` to always use `Bold(true)` and remove `SelectedBg`/`SelectedFg`
      - test: `styledName` output contains bold ANSI sequence; no background styling in styledName regardless of selected param
- [ ] Refactor `renderTrack()` to 3-line layout with `wrapLine()` wrapping
      - test: track render output has 3 lines; line 1 contains badge + name + duration; line 2 contains artists; line 3 contains album name
- [ ] Refactor `renderArtist()` to 3-line layout with `wrapLine()` wrapping
      - test: artist render output has 3 lines; line 1 contains badge + name; line 2 contains genres; line 3 contains followers
- [ ] Refactor `renderAlbum()` to 3-line layout with `wrapLine()` wrapping
      - test: album render output has 3 lines; line 1 contains badge + name + type + year; line 2 contains artists; line 3 contains track count
- [ ] Refactor `renderPlaylist()` to 3-line layout with `wrapLine()` wrapping
      - test: playlist render output has 3 lines; line 1 contains badge + name + tracks; line 2 contains owner; line 3 contains description
- [ ] Update `renderDefault()` to 3-line layout
      - test: default render output has 3 lines
- [ ] Update all existing tests in `search_delegate_test.go` for 3-line output and focus styling
      - test: `make ci` passes with no test failures
