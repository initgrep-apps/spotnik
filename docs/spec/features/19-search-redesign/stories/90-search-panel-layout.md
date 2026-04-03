---
title: "Search Panel Layout: Flush Panels, Interior Hints, Per-Panel Border Colors"
feature: 19-search-redesign
status: open
---

## Background

The current search overlay renders three bordered panels (Search, Results, Keys) with a 1-line margin between Search and Results. This creates two problems visible in the live UI:

1. **Dead space**: The margin line between panels wastes vertical real estate and looks unfinished.
2. **Floating hints**: The prefix hint row (`:albums`, `:artists`, etc.) renders in the margin gap *between* the Search and Results borders — it's not inside any panel, making it look like a rendering glitch.
3. **Indistinct panels**: All three panels use similar border colors, so they blend together visually despite being separate bordered boxes.

This story fixes all three: panels sit flush (zero margin), hints move inside the Search panel, and each panel gets a distinct theme-colored border.

## Design

### Target Layout

```
╭─ Search ──────────────────────────────────────────────────────╮  ← ActiveBorder()
│  > :al                                                        │
│    :albums   :artists   :playlists                            │  ← hints INSIDE
╰──────────────────────────────────────────────────────────────╯
╭─ Results ────────────────────────── Enter play  Ctrl+A queue ─╮  ← SeekBar()
│  All  Songs  Artists  [Albums]  Playlists                      │
│  ──────────────────────────────────────────────────────────── │
│  ◎ At World's End (Original Soundtrack)          Album · 2007  │
│    Hans Zimmer · 13 tracks                                     │
│  ◎ Interstellar                                  Album · 2014  │
│    Hans Zimmer · 17 tracks                                     │
│                                                                │
╰──────────────────────────────────────────────────────────────╯
╭──────────────────────────────────────────────────────────────╮  ← TextMuted()
│  enter play • ctrl+a queue • tab filter • shift+tab prev • esc │
╰──────────────────────────────────────────────────────────────╯
```

The Keys panel has **no title** — the keybinding content is self-explanatory. Removing the "Keys" label reduces visual noise and lets the dim border recede further into the background.
│  ◎ Interstellar                                  Album · 2014  │
│    Hans Zimmer · 17 tracks                                     │
│                                                                │
╰──────────────────────────────────────────────────────────────╯
╭─ Keys ────────────────────────────────────────────────────────╮  ← TextMuted()
│  enter play • ctrl+a queue • tab filter • shift+tab prev • esc │
╰──────────────────────────────────────────────────────────────╯
```

All three panels touch — `╰╯` of one panel directly above `╭╮` of the next. Rounded corners on each panel still make them visually distinct. No content floats outside a border.

### Change 1: Remove All Margins

**File: `internal/ui/panes/search.go` — `View()`**

Currently `View()` joins the three panels with an empty margin line between Search and Results:

```go
// Current
margin := ""
return lipgloss.JoinVertical(lipgloss.Left,
    searchPanel,
    margin,        // ← remove this
    resultsPanel,
    helpPanel,
)
```

Change to flush join with no margin:

```go
return lipgloss.JoinVertical(lipgloss.Left,
    searchPanel,
    resultsPanel,
    helpPanel,
)
```

**Height budget update**: The 1 margin line is reclaimed by the results panel. Update the height calculation:

```go
// Current
resultsH := totalH - searchBarH - helpH - 1  // -1 for margin

// New
resultsH := totalH - searchBarH - helpH      // no margin
```

Same change in `SetSize()`:

```go
func (o *SearchOverlay) SetSize(width, height int) {
    // ...
    resultsH := totalH - searchBarH - helpH  // was totalH - searchBarH - helpH - 1
    // ...
}
```

### Change 2: Hints Inside Search Panel

**File: `internal/ui/panes/search.go` — `renderSearchPanel()`**

The prefix hint line (from `renderPrefixHints()` or the new styled pills from Story 89) must render *inside* the Search panel border, below the text input.

The Search panel height is dynamic:
- **3 lines** when no hints visible: border-top + input + border-bottom
- **4 lines** when hints visible: border-top + input + hint-line + border-bottom

The `renderSearchPanel()` method builds inner content as:

```go
func (o *SearchOverlay) renderSearchPanel(width, height int) string {
    innerW := width - 2 // inside border

    // Text input line
    inputView := o.input.View()

    // Hint line (empty string when not applicable)
    hintLine := o.renderPrefixHints(innerW)

    var inner string
    if hintLine != "" {
        inner = lipgloss.JoinVertical(lipgloss.Left, inputView, hintLine)
    } else {
        inner = inputView
    }

    cfg := layout.BorderConfig{
        Width:       width,
        Height:      height,
        Title:       "Search",
        AccentColor: o.theme.ActiveBorder(),  // bright focused border
        Focused:     true,
        Theme:       o.theme,
    }
    return layout.RenderPaneBorder(inner, cfg)
}
```

The key difference from the current implementation: `hintLine` is rendered inside `RenderPaneBorder()`, not between panels.

**Search panel height calculation** (in `View()` and `SetSize()`):

```go
searchBarH := 3
if o.showHintLine() {
    searchBarH = 4
}
```

Where `showHintLine()` centralizes the logic:

```go
func (o *SearchOverlay) showHintLine() bool {
    return o.input.Value() == "" || o.prefixState == PrefixTyping
}
```

This matches the conditions from Story 89 (empty input shows pills, PrefixTyping shows matching hints). When PrefixLocked or typing a normal query, the panel is compact at 3 lines.

### Change 3: Per-Panel Border Colors

**File: `internal/ui/panes/search.go` — `renderSearchPanel()`, `renderResultsPanel()`, `renderHelpPanel()`**

Each panel gets a distinct `AccentColor` in its `layout.BorderConfig`:

| Panel | Current AccentColor | New AccentColor | Token | Rationale |
|-------|-------------------|-----------------|-------|-----------|
| Search | `ActiveBorder()` | `ActiveBorder()` | (unchanged) | Bright — signals "input goes here", always focused |
| Results | `SectionHeader()` | `SeekBar()` | cyan-family | Distinct from Search, prominent enough for main content area |
| Keys | `TextMuted()` | `TextMuted()` | (unchanged) | Dim — reference info, recedes behind the content panels |

**Results panel border update:**

```go
// Current
resultsCfg := layout.BorderConfig{
    // ...
    AccentColor: o.theme.SectionHeader(),  // blue
    Focused:     false,
    // ...
}

// New
resultsCfg := layout.BorderConfig{
    // ...
    AccentColor: o.theme.SeekBar(),  // cyan — distinct from Search's ActiveBorder
    Focused:     false,
    // ...
}
```

**Why `SeekBar()` for Results**: Looking at the theme tokens, `ActiveBorder()` is typically bright blue/green (for the focused Search panel), `SectionHeader()` is a similar blue that doesn't contrast enough. `SeekBar()` is cyan-family in most themes, providing clear visual distinction from both the bright Search border and the dim Keys border. The three-tone hierarchy is: bright → medium → dim.

**Theme compatibility**: All 11 themes in the codebase define distinct values for `ActiveBorder()`, `SeekBar()`, and `TextMuted()`, so the three panels will always be visually distinguishable regardless of active theme.

### Visual Hierarchy Summary

```
╭─ Search ─╮   ActiveBorder()  ← BRIGHT:  "I'm active, type here"
╰──────────╯
╭─ Results ─╮   SeekBar()      ← MEDIUM:  "I'm the main content"
╰──────────╯
╭──────────╮   TextMuted()    ← DIM:     "I'm reference info, no title needed"
╰──────────╯
```

The eye naturally flows top-to-bottom following the brightness gradient: bright input → medium content → dim reference. The Keys panel drops its title entirely — keybinding content is self-evident.

## Acceptance Criteria

- [ ] All three panels render flush — no margin lines between them
- [ ] The `╰╯` of one panel directly touches the `╭╮` of the next panel
- [ ] Prefix hints/pills render inside the Search panel border, not between panels
- [ ] Search panel is 4 lines tall when hints are visible, 3 lines when not
- [ ] No content renders outside a panel border
- [ ] Search panel border uses `ActiveBorder()` (bright)
- [ ] Results panel border uses `SeekBar()` (medium)
- [ ] Keys panel border uses `TextMuted()` (dim) with no title
- [ ] Keys panel `Title` is empty string — keybinding content is self-explanatory
- [ ] Three-tone hierarchy is visually distinguishable in all 11 themes
- [ ] Removing the margin reclaims 1 line for the results area
- [ ] `SetSize()` propagates correct dimensions with no-margin math
- [ ] `make ci` passes

## Tasks

- [ ] Remove margin line from `View()` — join three panels flush via `lipgloss.JoinVertical`
      - test: View output has no empty line between panel borders; count of `╭` border starts is exactly 3; no blank line between `╰` and `╭`
- [ ] Update height math in `View()` and `SetSize()` — remove the `-1` margin deduction
      - test: SetSize(120, 40) → resultsH is 1 line taller than before; list inner height gains 1 line
- [ ] Move hint line rendering inside `renderSearchPanel()` — build inner content as input + optional hint
      - test: when hints visible, Search panel output contains hint text inside `╭...╯` border; when no hints, panel is 3 lines; hint text never appears outside a border
- [ ] Implement `showHintLine()` helper to centralize hint visibility logic
      - test: empty input → true; PrefixTyping → true; PrefixLocked → false; normal query → false
- [ ] Change Results panel `AccentColor` from `SectionHeader()` to `SeekBar()`
      - test: Results border config uses `theme.SeekBar()`; visually distinct from Search and Keys borders
- [ ] Remove Keys panel title — set `Title: ""` in border config, keep `TextMuted()` accent
      - test: Keys border config has empty Title; output does not contain "Keys" label text; border still renders with `TextMuted()` accent
- [ ] Update existing layout tests that assert margin or panel heights
      - test: any test checking for margin line between panels updated; height assertions reflect no-margin math
