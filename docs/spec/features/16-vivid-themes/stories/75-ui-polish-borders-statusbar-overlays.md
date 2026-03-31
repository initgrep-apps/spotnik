---
title: "Fix: UI Polish — Border Corners, Page-Aware Shortcuts, Header Cleanup, Overlay Backgrounds"
feature: 16-vivid-themes
status: open
---

## Background

After the Vivid Themes feature (stories 70–73) was merged, four interconnected UI
issues were found that all relate to how the status bar, header bar, pane borders,
and overlays render. These require coordinated changes across `render.go` and
`border.go`.

**Root causes:**

1. **Border action arrows use "ᐅ" prefix** — The `buildRightSegment()` function in
   `border.go` renders pane actions as `ᐅkey label` with `───` separators. The user
   wants these to integrate into the border line as corner-character "notches" using
   `╮` and `╭`, making actions look like tabs cut into the border:
   Before: `¹Now Playing ─────────── ᐅs shfl ─── ᐅr rpt ─── ᐅspace play`
   After:  `¹Now Playing ───────────╮s shfl╭─╮r rpt╭─╮space play╭`

2. **Top header duplicates shortcuts from the bottom status bar** — The header
   (`renderHeader()`) shows `ᐅp preset 0`, `ᐅ/ search`, `ᐅd devices` as inline
   shortcut hints. These same keys appear in the bottom status bar. The header should
   show only contextual info: app name, page label, active preset number. Shortcut
   keys belong in the bottom bar only. Also the bottom bar items use 3-space gaps
   (`"   "`) which is too wide — should be tighter.

3. **Status bar shortcuts are static regardless of page** — The hints in
   `renderStatusBar()` never change. Page A supports presets (`p`) and toggle keys
   (`1-8`), but Page B does not (it has a single fixed layout). Showing `p preset`
   and `1-8 toggle` on Page B is misleading.

4. **Overlay backgrounds make focus highlighting unusable** — Both ThemeOverlay and
   DeviceOverlay apply explicit background colors to non-cursor rows (`Surface()` and
   `SurfaceAlt()` respectively). This creates opaque colored rectangles that make the
   cursor/selection highlight hard to distinguish. Non-cursor rows should have no
   explicit background (transparent), letting the dimmed grid show through. Only the
   cursor row should have `SelectedBg()`.

## Design

### Task 1: Replace Arrow Prefix with Corner-Character Notches in Pane Borders

**File:** `internal/ui/layout/border.go`

Redesign `buildRightSegment()` to produce corner-notch styled actions. Each action
becomes a "notch" in the border line using `╮` before the action and `╭` after:

```
Before: ─────── ᐅs shfl ─── ᐅr rpt ─── ᐅspace play ─╮
After:  ──────╮ s shfl ╭─╮ r rpt ╭─╮ space play ╭────╮
```

**Implementation:**

```go
func buildRightSegment(cfg BorderConfig, keyHintStyle, mutedStyle func(string) string) string {
    if cfg.FilterQuery != "" {
        // Filter mode unchanged — keep existing "filtering: ..." format.
        filtering := mutedStyle(`filtering: "` + cfg.FilterQuery + `"`)
        sep := mutedStyle(" ─── ")
        escAction := keyHintStyle("Esc") + " " + mutedStyle("close")
        prefix := mutedStyle("ᐅ")
        return filtering + sep + prefix + escAction
    }
    if len(cfg.Actions) == 0 {
        return ""
    }

    // borderStyle for rendering ╮ and ╭ in accent color.
    borderChar := func(s string) string {
        style := lipgloss.NewStyle().Foreground(cfg.AccentColor)
        if !cfg.Focused {
            style = style.Faint(true)
        }
        return style.Render(s)
    }

    var parts []string
    for _, a := range cfg.Actions {
        notch := borderChar("╮") + " " +
            keyHintStyle(a.Key) + " " + mutedStyle(a.Label) + " " +
            borderChar("╭")
        parts = append(parts, notch)
    }
    return strings.Join(parts, borderChar("─"))
}
```

The `╮` and `╭` characters must be rendered in `borderStyle` (accent color, with
faint if unfocused) so they blend into the border line. The space around key/label
ensures readability.

**Important:** The dash-count calculation in `RenderPaneBorder()` must account for the
changed width of `rightSegment`. The existing `lipgloss.Width()` call handles this
automatically since it measures terminal columns.

Also update the top-right corner logic: the top border currently ends with
`borderStyle(rightSuffix) + borderStyle(cornerTR)` where `rightSuffix = " "`. Since
the last action now ends with `╭`, the trailing pattern should flow naturally into the
top-right corner `╮`. Adjust the trailing dash fill to bridge from the last `╭` to the
`╮` corner:

```
...╮ space play ╭───╮   (dashes fill between last ╭ and the corner ╮)
```

### Task 2: Clean Up Header — Remove Shortcut Duplicates

**File:** `internal/app/render.go`

Simplify `renderHeader()` to show only contextual information:
- Left: `spotnik — Page A — preset 0`  (no `ᐅ` prefix, no `/` search, no `d` devices)
- Right: Device indicator (unchanged)

```go
// Simplified left segment:
left := appName + sep + page + sep + mutedStyle(fmt.Sprintf("preset %d", presetIdx))
```

Remove the `search` and `devices` shortcut hint variables entirely. The preset number
stays because it's contextual info (which preset is active), not a shortcut hint.

On Page B, since there are no user-selectable presets, replace `preset N` with just
the page label:

```go
if a.layout.ActivePage() == layout.PageB {
    left = appName + sep + page
} else {
    left = appName + sep + page + sep + mutedStyle(fmt.Sprintf("preset %d", presetIdx))
}
```

### Task 3: Page-Aware Status Bar Shortcuts

**File:** `internal/app/render.go`

Make `renderStatusBar()` aware of the active page. Pass the page to the method (or
read it from `a.layout.ActivePage()`).

**Page A hints:**
```go
{"/", "search"}, {"0", "page"}, {"p", "preset"}, {"1-8", "toggle"},
{"Tab", "pane"}, {"d", "devices"}, {"t", "theme"}, {"?", "help"}, {"q", "quit"}
```

**Page B hints:**
```go
{"/", "search"}, {"0", "page"},
{"Tab", "pane"}, {"d", "devices"}, {"t", "theme"}, {"?", "help"}, {"q", "quit"}
```

The difference: Page B omits `p preset` and `1-8 toggle` since Page B has a single
fixed layout with no presets or toggleable panes.

Also tighten the spacing between hints. Current separator is `"   "` (3 spaces).
Change to `"  "` (2 spaces):

```go
return bgStyle.Render("  " + strings.Join(parts, "  "))
```

### Task 4: Fix Overlay Background Colors

**Files:** `internal/ui/panes/themes.go`, `internal/ui/panes/devices.go`

#### Theme Overlay (`themes.go`)

In `renderRow()`, change non-cursor row background from `Surface()` to `Base()` (the
darkest background, which is effectively transparent against the dimmed grid behind the
overlay):

```go
// Before (line 168):
bg := o.theme.Surface()

// After:
bg := o.theme.Base()
```

This makes non-selected rows visually recede, so the cursor row with `SelectedBg()`
stands out clearly.

Also, ensure the swatches in `renderSwatches()` do NOT apply a background — they should
use only foreground color on the `█` character. Check that `renderSwatches()` doesn't
set any `Background()`. Currently it only sets `Foreground()` which is correct.

#### Device Overlay (`devices.go`)

In `renderDevice()`, change non-cursor row background from `SurfaceAlt()` to `Base()`:

```go
// Before (line 205):
bg := d.theme.SurfaceAlt()

// After:
bg := d.theme.Base()
```

Same rationale — non-cursor rows should recede, cursor row should pop.

## Acceptance Criteria
- [ ] Pane border actions use `╮ key label ╭` corner-notch style instead of `ᐅkey label`
- [ ] Filter mode border rendering is unchanged (still uses `ᐅEsc close`)
- [ ] Top header shows only `spotnik — Page A — preset 0` (no shortcut keys)
- [ ] Top header on Page B shows only `spotnik — Page B` (no preset number)
- [ ] Status bar on Page A shows: `/ search`, `0 page`, `p preset`, `1-8 toggle`, `Tab pane`, `d devices`, `t theme`, `? help`, `q quit`
- [ ] Status bar on Page B omits `p preset` and `1-8 toggle`
- [ ] Status bar spacing is 2 spaces between items (not 3)
- [ ] Theme overlay non-cursor rows use `Base()` background
- [ ] Device overlay non-cursor rows use `Base()` background
- [ ] Cursor row in both overlays clearly stands out with `SelectedBg()` highlight
- [ ] `make ci` passes (lint + tests + 80% coverage)

## Tasks
- [ ] Rewrite `buildRightSegment()` in `border.go` to produce corner-notch format: `╮ key label ╭` with `─` between notches. The `╮` and `╭` characters use `borderStyle` (accent color, faint if unfocused). Filter mode keeps existing format. Update the `rightSuffix` handling to bridge the last `╭` to the `╮` corner with dash fill.
      - test: `TestBuildRightSegment_CornerNotchFormat` — verify output contains `╮` and `╭` and does NOT contain `ᐅ`. `TestBuildRightSegment_FilterMode_Unchanged` — verify filter mode still uses `ᐅEsc close` format. `TestRenderPaneBorder_NotchActions_FitsWidth` — verify total rendered width matches config width exactly.
- [ ] Simplify `renderHeader()` in `render.go`: remove `search`, `devices` shortcut variables and their `ᐅ` prefix rendering. Keep only `appName + sep + page + sep + preset` on Page A, and `appName + sep + page` on Page B.
      - test: `TestRenderHeader_NoShortcutKeys` — verify output does NOT contain "ᐅ/" or "ᐅd". `TestRenderHeader_PageA_ShowsPreset` — verify "preset 0" appears. `TestRenderHeader_PageB_NoPreset` — verify no "preset" text on Page B.
- [ ] Make `renderStatusBar()` in `render.go` page-aware: read `a.layout.ActivePage()` and build hints conditionally — include `p preset` and `1-8 toggle` only on Page A. Change separator from `"   "` to `"  "` (2 spaces).
      - test: `TestRenderStatusBar_PageA_IncludesPresetAndToggle`, `TestRenderStatusBar_PageB_OmitsPresetAndToggle`, `TestRenderStatusBar_TighterSpacing` — verify 2-space separation.
- [ ] Change non-cursor row background in `ThemeOverlay.renderRow()` (`themes.go`) from `o.theme.Surface()` to `o.theme.Base()`. Change non-cursor row background in `DeviceOverlay.renderDevice()` (`devices.go`) from `d.theme.SurfaceAlt()` to `d.theme.Base()`.
      - test: `TestThemeOverlay_NonCursorRow_UsesBaseBackground`, `TestDeviceOverlay_NonCursorRow_UsesBaseBackground` — verify non-cursor rows use `Base()` color and cursor rows use `SelectedBg()`.
