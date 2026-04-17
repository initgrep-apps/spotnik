---
title: "Fix: Quick Theme Fixes — Sub-Box Borders, Missing Shortcut, Key Styling"
feature: 16-vivid-themes
status: done
---

## Background

After the Vivid Themes feature (stories 70–73) was merged, three small issues were
found in visual testing:

1. **RequestFlow internal sub-boxes still have grey borders.** The `renderSubBox()`
   method in `requestflow_boxed.go` uses `p.theme.TextSecondary()` for the APP,
   GATEWAY, and SPOTIFY sub-panel borders. This was not updated during story 71
   (colorful borders). These internal borders should use the pane's accent color
   (`PaneBorderRequestFlow`) so they match the outer pane border style — dimmed
   accent, not flat grey.

2. **Theme shortcut "t" missing from the status bar.** Story 73 added the `t` key
   to open the theme switcher overlay and updated `docs/DESIGN.md`, but the status
   bar hints in `render.go` were not updated. The `hints` slice has 8 entries and
   does not include `{"t", "theme"}`.

3. **Status bar shortcut keys have visible background "pill" styling.** The `keyStyle`
   in `renderStatusBar()` applies an explicit `Background(a.theme.StatusBarBg())`
   which, combined with `Bold(true)` and `Foreground(KeyHint())`, causes some terminal
   emulators to render visible colored rectangles behind each key character. The keys
   should inherit the parent bar's background naturally — only `Foreground` + `Bold`
   are needed.

All three are isolated single-file changes with no design decisions required.

## Design

### Task 1: RequestFlow Sub-Box Accent Borders

**File:** `internal/ui/panes/requestflow_boxed.go`

Change `renderSubBox()` to use the pane's accent color instead of `TextSecondary()`:

```go
// Before (line 25):
borderColor := p.theme.TextSecondary()

// After:
borderColor := p.theme.PaneBorderRequestFlow()
```

Also update the title style on line 32 to use the accent color:

```go
// Before:
titleStyled := lipgloss.NewStyle().Foreground(p.theme.TextSecondary()).Bold(true).Render(title)

// After:
titleStyled := lipgloss.NewStyle().Foreground(p.theme.PaneBorderRequestFlow()).Bold(true).Render(title)
```

The internal boxes will now show the same orange/amber accent as the outer RequestFlow
border (dimmed since the sub-boxes are always "unfocused" relative to the outer pane).

### Task 2: Add "t" Theme Shortcut to Status Bar

**File:** `internal/app/render.go`

Add `{"t", "theme"}` to the `hints` slice in `renderStatusBar()`. Insert it after
`{"d", "devices"}` since theme switching is a global action similar to device switching:

```go
hints := []struct{ Key, Label string }{
    {"/", "search"},
    {"0", "page"},
    {"p", "preset"},
    {"1-8", "toggle"},
    {"Tab", "pane"},
    {"d", "devices"},
    {"t", "theme"},     // NEW
    {"?", "help"},
    {"q", "quit"},
}
```

### Task 3: Remove Background from Status Bar Key Styles

**File:** `internal/app/render.go`

In `renderStatusBar()`, remove `Background()` from `keyStyle`:

```go
// Before:
keyStyle := lipgloss.NewStyle().
    Background(a.theme.StatusBarBg()).
    Foreground(a.theme.KeyHint()).
    Bold(true)

// After:
keyStyle := lipgloss.NewStyle().
    Foreground(a.theme.KeyHint()).
    Bold(true)
```

The parent `bgStyle.Render()` call on line 341 already wraps the entire bar with
`Background(StatusBarBg())`, so individual keys inherit it without needing their own
explicit background.

## Acceptance Criteria
- [ ] RequestFlow internal sub-boxes (APP, GATEWAY, SPOTIFY) use the pane's accent color for borders and titles, not `TextSecondary`
- [ ] Status bar includes `t theme` shortcut between `d devices` and `? help`
- [ ] Status bar key characters render without visible background rectangles — only foreground color + bold
- [ ] `make ci` passes (lint + tests + 80% coverage)

## Tasks
- [ ] Change `renderSubBox()` in `requestflow_boxed.go` to use `p.theme.PaneBorderRequestFlow()` for `borderColor` (line 25) and title style (line 32)
      - test: Existing `requestflow_boxed_test.go` tests still pass; add `TestRenderSubBox_UsesAccentColor` verifying output contains the accent color ANSI escape, not TextSecondary
- [ ] Add `{"t", "theme"}` to the `hints` slice in `renderStatusBar()` in `render.go` (after `{"d", "devices"}`)
      - test: `TestRenderStatusBar_ContainsThemeHint` — verify output contains "t" and "theme"
- [ ] Remove `Background(a.theme.StatusBarBg())` from `keyStyle` in `renderStatusBar()` in `render.go`
      - test: `TestRenderStatusBar_KeyStyleNoBackground` — verify keyStyle output does not contain background ANSI escape (only foreground + bold)
