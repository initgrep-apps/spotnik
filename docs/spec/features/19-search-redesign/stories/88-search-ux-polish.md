---
title: "Search UX Polish: Enter Behavior, Width, Icons"
feature: 19-search-redesign
status: done
---

## Background

Three simple, independent UX fixes for the search overlay that don't touch the prefix system.

1. **Enter closes overlay**: Pressing Enter plays a song AND closes the overlay via `SearchClosedMsg`. Users may want to browse more results after playing a track — only Esc should close.
2. **Overlay too wide**: At 80% terminal width the overlay feels bloated. 70% is tighter and more focused.
3. **Boring icons**: Current category symbols (`♫ ● ◆ ☰`) are generic. Modern Unicode symbols that are monospace-safe and theme-colorable via lipgloss would give a nerdier feel.

## Design

### Fix 1: Enter Keeps Overlay Open

**File: `internal/ui/panes/search.go` — `handleEnter()`**

Remove the `SearchClosedMsg` from the batched command. Enter should only emit the play command:

```go
func (o *SearchOverlay) handleEnter() (tea.Model, tea.Cmd) {
    selected := o.resultList.SelectedItem()
    if selected == nil { return o, nil }
    si, ok := selected.(SearchListItem)
    if !ok || si.URI == "" { return o, nil }

    if si.IsTrack {
        uri := si.URI
        return o, func() tea.Msg { return PlayTrackMsg{TrackURI: uri} }
    }
    uri := si.URI
    return o, func() tea.Msg { return PlayContextMsg{ContextURI: uri} }
}
```

Only `Esc` (already handled) emits `SearchClosedMsg`.

### Fix 2: Overlay Width → 70%

**File: `internal/ui/panes/search.go` — `overlayWidth()`**

```go
func (o *SearchOverlay) overlayWidth() int {
    w := o.width * 70 / 100  // was 80
    if w < 40 { w = 40 }
    return w
}
```

Height stays at 80%. Update the doc comment on `OverlayWidth()` and the `SetSize` test expectations.

### Fix 3: Better Unicode Icons

**File: `internal/ui/panes/search_delegate.go` — `categorySymbol()`**

Replace the current symbols with more distinctive, monospace-safe Unicode characters. These must be single-width glyphs (NOT emoji) so lipgloss can apply foreground colors via theme tokens:

```go
func categorySymbol(category string) string {
    switch category {
    case "track":    return "♪"  // U+266A Eighth Note — musical, distinct
    case "artist":   return "★"  // U+2605 Black Star — person/fame
    case "album":    return "◎"  // U+25CE Bullseye — disc-like, album
    case "playlist": return "▤"  // U+25A4 Square with horizontal fill — list
    default:         return "·"
    }
}
```

**Why these specific symbols:**
- `♪` (U+266A): Musical note — universally recognized for audio, single-width in all modern terminals
- `★` (U+2605): Star — suggests fame/artists, renders at single width, colors well
- `◎` (U+25CE): Bullseye/double circle — disc-like for albums, always monospace
- `▤` (U+25A4): Horizontal-lined square — suggests a list/playlist, consistent width

All four are in the Basic Multilingual Plane (BMP), supported by every terminal that handles Unicode at all (including iTerm2, Alacritty, Kitty, Windows Terminal, GNOME Terminal, macOS Terminal.app). They are NOT emoji, so:
- lipgloss `Foreground()` coloring works reliably
- No double-width rendering issues
- No skin-tone/variation selector complications

## Acceptance Criteria

- [ ] Enter plays the selected item but does NOT close the overlay
- [ ] Only Esc closes the overlay (existing behavior preserved)
- [ ] Overlay width is 70% of terminal width (minimum 40)
- [ ] Category icons are `♪` (track), `★` (artist), `◎` (album), `▤` (playlist)
- [ ] Icons are colorable via lipgloss theme tokens (not emoji)
- [ ] `make ci` passes

## Tasks

- [ ] Remove `SearchClosedMsg` from `handleEnter()` — only emit play command
      - test: Enter on track emits `PlayTrackMsg` only, no `SearchClosedMsg`; Enter on album emits `PlayContextMsg` only; Esc still emits `SearchClosedMsg`
- [ ] Change `overlayWidth()` from 80% to 70%
      - test: terminal width 200 → overlay 140 (was 160); terminal width 50 → overlay 40 (minimum clamp); update any tests that assert 80% values
- [ ] Replace category symbols in `categorySymbol()`
      - test: `categorySymbol("track")` → "♪"; `categorySymbol("artist")` → "★"; `categorySymbol("album")` → "◎"; `categorySymbol("playlist")` → "▤"; `categorySymbol("unknown")` → "·"
