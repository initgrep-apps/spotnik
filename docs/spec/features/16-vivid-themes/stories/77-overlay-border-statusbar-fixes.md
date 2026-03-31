---
title: "Fix: Overlay Backgrounds, Border Corner Gap, Network Log Grey Borders, Status Bar Key Pills"
feature: 16-vivid-themes
status: open
---

## Background

After the Vivid Themes fixes (stories 74–75, PR #93), three visual bugs remain:

1. **Overlay non-cursor rows still show visible background rectangles.** Story 75
   changed the non-cursor row background from `Surface()` to `Base()`, but `Base()`
   is still an explicit opaque color (e.g., Gruvbox `#282828`, Catppuccin `#1e1e2e`).
   When the overlay is composited on the dimmed grid via `btoverlay.Composite()`,
   these opaque `Base()` backgrounds create visible colored blocks instead of blending
   with the dimmed content. This affects both ThemeOverlay (`themes.go:170`) and
   DeviceOverlay (`devices.go:208`).

2. **Top-right corner of every pane border has a blank space.** In `border.go:121`,
   `rightSuffix := " "` is unconditionally included in the top border assembly
   (line 167: `borderStyle(rightSuffix) + borderStyle(cornerTR)`). This creates a
   visible gap: `────── ╮` instead of `──────╮`. The space persists even when
   `rightSegment` is empty (panes with no actions), and when actions ARE present the
   last notch's `╭` already provides visual separation from the corner `╮`.

3. **Network Log pane has grey borders.** All 11 TOML theme files define
   `network_log` with desaturated grey values (`#8a8a8a`, `#a89984`, `#6272A4`, etc.)
   while every other pane uses vibrant saturated colors. When unfocused (rendered with
   `Faint(true)`), these grey borders become nearly invisible — defeating the "always
   colorful borders" goal from the feature spec.

4. **Status bar shortcut keys show background "pills."** Story 74 removed
   `Background(StatusBarBg())` from `keyStyle`, but when the styled key text (with
   `Foreground + Bold` only) is nested inside `bgStyle.Render()`, lipgloss ANSI
   sequence nesting can create gaps where the parent's background doesn't carry through
   to inline-styled tokens. This causes subtle but visible background mismatches around
   key characters.

## Design

### Task 1: Remove Explicit Background from Overlay Non-Cursor Rows

**Files:** `internal/ui/panes/themes.go`, `internal/ui/panes/devices.go`

#### Theme Overlay (`themes.go`)

In `renderRow()`, change the non-cursor branch to set NO explicit background. Only
cursor rows get `Background(SelectedBg())`:

```go
// renderRow renders a single theme row with indicator and color swatches.
func (o *ThemeOverlay) renderRow(idx int, th *theme.ConfigTheme, innerWidth int) string {
    isCursor := idx == o.cursor
    isCurrent := th.ID() == o.currentID

    // Indicator style.
    var indicator string
    var indicatorStyle lipgloss.Style
    if isCurrent {
        indicatorStyle = lipgloss.NewStyle().Foreground(o.theme.Success())
        indicator = "◉"
    } else {
        indicatorStyle = lipgloss.NewStyle().Foreground(o.theme.TextMuted())
        indicator = "○"
    }

    // Name style.
    var nameStyle lipgloss.Style
    if isCursor {
        nameStyle = lipgloss.NewStyle().
            Foreground(o.theme.SelectedFg()).
            Background(o.theme.SelectedBg())
    } else {
        nameStyle = lipgloss.NewStyle().Foreground(o.theme.TextPrimary())
    }

    // Apply Background(SelectedBg()) to all elements ONLY on cursor row.
    if isCursor {
        bg := o.theme.SelectedBg()
        indicatorStyle = indicatorStyle.Background(bg)
    }

    swatches := renderSwatches(th)

    row := indicatorStyle.Render(indicator) + " " +
        nameStyle.Render(th.Name()) +
        "  " +
        swatches

    // Pad row to inner width.
    if isCursor {
        bg := o.theme.SelectedBg()
        rowStyle := lipgloss.NewStyle().Background(bg)
        return rowStyle.Render(lipgloss.NewStyle().
            Width(innerWidth).MaxWidth(innerWidth).
            Background(bg).
            Render(row))
    }
    // Non-cursor: no explicit background — row blends with overlay background.
    return lipgloss.NewStyle().
        Width(innerWidth).MaxWidth(innerWidth).
        Render(row)
}
```

Key change: non-cursor rows have **zero** `Background()` calls. Only cursor rows
apply `Background(SelectedBg())` to indicator, name, and row padding.

#### Device Overlay (`devices.go`)

Same pattern in `renderDevice()`:

```go
func (d *DeviceOverlay) renderDevice(idx int, dev DeviceInfo) string {
    isCursor := idx == d.cursor

    var bullet string
    var bulletStyle lipgloss.Style
    var nameStyle lipgloss.Style

    if isCursor {
        bg := d.theme.SelectedBg()
        if dev.IsActive {
            bulletStyle = lipgloss.NewStyle().Foreground(d.theme.DeviceActive()).Background(bg)
            bullet = "◉"
        } else {
            bulletStyle = lipgloss.NewStyle().Foreground(d.theme.InactiveBorder()).Background(bg)
            bullet = "○"
        }
        nameStyle = lipgloss.NewStyle().Foreground(d.theme.TextPrimary()).Background(bg)
    } else {
        // Non-cursor: no Background() at all.
        if dev.IsActive {
            bulletStyle = lipgloss.NewStyle().Foreground(d.theme.DeviceActive())
            bullet = "◉"
        } else {
            bulletStyle = lipgloss.NewStyle().Foreground(d.theme.InactiveBorder())
            bullet = "○"
        }
        nameStyle = lipgloss.NewStyle().Foreground(d.theme.TextPrimary())
    }

    typeIcon := deviceTypeIcon(dev.Type)
    label := ""
    if dev.IsActive {
        if isCursor {
            label = lipgloss.NewStyle().
                Foreground(d.theme.Success()).
                Background(d.theme.SelectedBg()).
                Render(" [active]")
        } else {
            label = lipgloss.NewStyle().
                Foreground(d.theme.Success()).
                Render(" [active]")
        }
    }

    typeStyle := lipgloss.NewStyle().Foreground(d.theme.TextMuted())
    if isCursor {
        typeStyle = typeStyle.Background(d.theme.SelectedBg())
    }

    return bulletStyle.Render(bullet) + " " +
        typeStyle.Render(typeIcon) + " " +
        nameStyle.Render(dev.Name) +
        label
}
```

### Task 2: Fix Border Top-Right Corner Gap

**File:** `internal/ui/layout/border.go`

Make `rightSuffix` conditional — include the space only when `rightSegment` is
non-empty:

```go
// Before (line 121):
rightSuffix := " "

// After:
rightSuffix := ""
if rightSegment != "" {
    rightSuffix = " "
}
```

The rest of the width calculation (`fixedWidth` on lines 127–133) already uses
`lipgloss.Width(rightSuffix)` which will correctly return 0 when empty.

Result: panes with no actions show `──────╮` (flush). Panes with actions show
`╮ key label ╭ ╮` with the space providing breathing room before the corner.

### Task 3: Update Network Log Border Colors in All TOML Files

**Files:** All 11 `.toml` files in `internal/ui/theme/themes/`

Replace the grey `network_log` values with vibrant, saturated colors that fit each
theme's palette and are distinct from existing pane border colors:

| Theme | Old `network_log` | New `network_log` | Rationale |
|-------|-------------------|-------------------|-----------|
| black | `#8a8a8a` | `#ff6ac1` | Hot pink — distinct from all existing cool-toned borders |
| catppuccin | `#6c7086` | `#94e2d5` | Teal — Catppuccin's teal token, distinct from pink/purple |
| dracula | `#6272A4` | `#8BE9FD` | Cyan — Dracula's canonical cyan, complements purple/pink |
| gruvbox | `#a89984` | `#83a598` | Aqua — Gruvbox's blue/aqua, distinct from orange/green |
| light | `#9ca0b0` | `#179299` | Teal — Catppuccin Latte's teal, vibrant on light bg |
| monokai | `#75715e` | `#66d9ef` | Cyan — Monokai's iconic cyan, distinct from purple/pink |
| nord | `#4c566a` | `#88c0d0` | Frost blue — Nord's frost palette, distinct from aurora colors |
| rosepine | `#908caa` | `#9ccfd8` | Foam — Rose Pine's foam token, distinct from iris/gold |
| solarized | `#586e75` | `#2aa198` | Cyan — Solarized's canonical cyan, highly saturated |
| synthwave | `#848bbd` | `#72f1b8` | Neon green — Synthwave's glow green, distinct from pink/yellow |
| tokyonight | `#565f89` | `#7dcfff` | Light blue — Tokyo Night's sky blue, distinct from purple/pink |

Each color was chosen to be:
- From the theme's canonical palette where possible
- Distinct from all existing `[pane_borders]` values in that theme
- Saturated enough to remain visible when dimmed with `Faint(true)`

### Task 4: Fix Status Bar Key Background Pills

**File:** `internal/app/render.go`

Add `Background(StatusBarBg())` back to `keyStyle`, and also apply it to the space
between key and label, ensuring every character in the status bar has a consistent
explicit background:

```go
// Before (lines 327-329):
keyStyle := lipgloss.NewStyle().
    Foreground(a.theme.KeyHint()).
    Bold(true)

// After:
keyStyle := lipgloss.NewStyle().
    Foreground(a.theme.KeyHint()).
    Background(a.theme.StatusBarBg()).
    Bold(true)
```

Also update the parts assembly (line 358) to style the space between key and label:

```go
// Before:
parts = append(parts, keyStyle.Render(h.Key)+" "+bgStyle.Render(h.Label))

// After:
parts = append(parts, keyStyle.Render(h.Key)+bgStyle.Render(" "+h.Label))
```

By joining the space with the label in `bgStyle.Render()`, there is no unstyled gap.
Both `keyStyle` and `bgStyle` now carry the same explicit `Background(StatusBarBg())`,
so every character has a consistent background with no ANSI nesting gaps.

## Acceptance Criteria

- [ ] Theme overlay non-cursor rows have NO explicit `Background()` calls — no visible background rectangles
- [ ] Theme overlay cursor row uses `Background(SelectedBg())` and clearly stands out
- [ ] Device overlay non-cursor rows have NO explicit `Background()` calls — no visible background rectangles
- [ ] Device overlay cursor row uses `Background(SelectedBg())` and clearly stands out
- [ ] Pane borders with no actions show `╮` flush with dashes (no gap): `──────╮`
- [ ] Pane borders with actions retain the space before corner: `╭ ╮`
- [ ] Network Log pane (Page B) shows vibrant colored borders matching each theme's palette
- [ ] Network Log borders remain visible when unfocused (dimmed with `Faint(true)`)
- [ ] Status bar key characters have consistent background with no visible "pills"
- [ ] `make ci` passes (lint + tests + 80% coverage)

## Tasks

- [ ] **Task 1:** Remove all `Background()` calls from non-cursor row elements in `ThemeOverlay.renderRow()` (`themes.go`). Only cursor rows keep `Background(SelectedBg())` on indicator, name, and row padding styles.
      - Update existing test `TestThemeOverlay_NonCursorRow_UsesBaseBackground` → rename to `TestThemeOverlay_NonCursorRow_NoExplicitBackground`. Enable TrueColor profile, render a non-cursor row, and assert the output does NOT contain `"48;2;"` (the ANSI introducer for 24-bit background). This proves no explicit background is set.
      - Keep existing `TestThemeOverlay_CursorRow_UsesSelectedBg` — verify cursor row still contains the `SelectedBg()` ANSI escape.
- [ ] **Task 2:** Remove all `Background()` calls from non-cursor row elements in `DeviceOverlay.renderDevice()` (`devices.go`). Only cursor rows keep `Background(SelectedBg())` on bullet, typeIcon, name, and label styles.
      - Update existing test `TestDeviceOverlay_NonCursorRow_UsesBaseBackground` → rename to `TestDeviceOverlay_NonCursorRow_NoExplicitBackground`. Enable TrueColor profile, render a non-cursor row, and assert output does NOT contain `"48;2;"`.
      - Keep existing `TestDeviceOverlay_CursorRow_UsesSelectedBg` — verify cursor row still contains `SelectedBg()`.
- [ ] **Task 3:** Make `rightSuffix` conditional in `RenderPaneBorder()` (`border.go`): set `rightSuffix = ""` when `rightSegment` is empty, `" "` when non-empty.
      - Add test `TestRenderPaneBorder_NoActions_FlushCorner`: create a `BorderConfig` with zero actions, render it, extract top border line, and assert the line ends with `"╮"` preceded by `"─"` (no space before corner).
      - Add test `TestRenderPaneBorder_WithActions_SpaceBeforeCorner`: create a `BorderConfig` with actions, render it, extract top border line, and assert a space exists before `"╮"` at the end.
      - Existing `TestRenderPaneBorder_WidthMatchesRequested` and `TestRenderPaneBorder_NotchActions_FitsWidth` must still pass (width calculations remain correct).
- [ ] **Task 4:** Update `network_log` color in all 11 TOML files to vibrant saturated values per the table in the Design section.
      - Add test `TestAllThemes_NetworkLogBorderIsVibrant`: load all themes, for each theme get `PaneBorderNetworkLog()` color string. Parse the hex RGB values and assert the color has saturation > 20% (i.e., not a grey — `max(r,g,b) - min(r,g,b) > 50` on a 0-255 scale). This prevents regression to grey values.
- [ ] **Task 5:** Add `Background(StatusBarBg())` to `keyStyle` in `renderStatusBar()` (`render.go`). Change the parts assembly to `keyStyle.Render(h.Key)+bgStyle.Render(" "+h.Label)` so no unstyled gap exists between key and label.
      - Update existing test `TestRenderStatusBar_KeyStyleNoBackground` → rename to `TestRenderStatusBar_KeyStyleHasConsistentBackground`. Enable TrueColor, render keyStyle with `Background(StatusBarBg())`, and assert output DOES contain `"48;2;"` matching StatusBarBg. Also render bgStyle on a label and extract its `"48;2;"` sequence — assert both sequences are identical (same background color on key and label).
      - Existing tests `TestRenderStatusBar_ContainsGlobalShortcuts`, `TestRenderStatusBar_ContainsThemeHint`, `TestRenderStatusBar_PageA_IncludesPresetAndToggle`, `TestRenderStatusBar_PageB_OmitsPresetAndToggle` must still pass.
