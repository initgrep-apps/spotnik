---
title: "Fix: Border Corner Gap (rightSuffix) & Status Bar Key Pills (nested Render)"
feature: 16-vivid-themes
status: done
---

## Background

Story 77 (PR #94) attempted to fix four visual bugs. The overlay background fix
succeeded, but two bugs persist after three implementation attempts:

1. **Border top-right corner gap** — Every pane with actions shows a visible space
   before the `╮` corner: `╭ ╮` instead of `╭╮`.
2. **Status bar key pills** — Keyboard shortcut keys (`/`, `0`, `p`, etc.) display
   with visible background rectangles that contrast with the surrounding bar.

Both were misdiagnosed in prior attempts. This story addresses the actual root causes.

## Design

### Task 1: Remove rightSuffix Entirely from Border Assembly

**File:** `internal/ui/layout/border.go`

Story 77's fix made `rightSuffix` conditional — empty when no actions, `" "` when
actions exist. But story 77's own description states: *"when actions ARE present the
last notch's `╭` already provides visual separation from the corner `╮`."* The
implementation contradicted the spec.

**Fix:** `rightSuffix` should always be `""`. The last notch `╭` provides separation
from corner `╮` when actions exist; dashes flush against `╮` when no actions exist.

Remove the variable and its conditional (lines 121-128). Inline `""` or remove all
references to `rightSuffix` in the width calculations and top border assembly. Remove
the fallback reset at line 146 (now a no-op). Update the comment at line 117 that
references rightSuffix width.

Width calculations already use `lipgloss.Width(rightSuffix)` which returns 0 for
empty string, so no arithmetic changes are needed.

### Task 2: Flatten Status Bar Rendering — No Nested Render

**File:** `internal/app/render.go`

The current `renderStatusBar()` wraps everything in an outer `bgStyle.Render(...)` that
contains inner `keyStyle.Render()` and `bgStyle.Render()` outputs. The `"  "` separators
in `strings.Join(parts, "  ")` are plain text inside the outer Render. Inner ANSI resets
break the outer style context, causing separators to render with the terminal's default
background instead of `StatusBarBg()`. This creates visible pills around key characters
(which DO have explicit background).

The fix follows `renderHeader()`'s proven pattern (same file, lines 300-308):
concatenate individually-rendered pieces without any outer Render wrapper.

```go
var parts []string
for _, h := range hints {
    parts = append(parts, keyStyle.Render(h.Key)+bgStyle.Render(" "+h.Label))
}
sep := bgStyle.Render("  ")
bar := bgStyle.Render("  ") + strings.Join(parts, sep)
// Pad to terminal width so background covers the full row.
if barW := lipgloss.Width(bar); a.width > barW {
    bar += bgStyle.Render(strings.Repeat(" ", a.width-barW))
}
return bar
```

Every character now has explicit `Background(StatusBarBg())` — no plain text gaps.

## Acceptance Criteria

- [ ] Pane borders with actions show `╭╮` at end (no space before corner)
- [ ] Pane borders without actions show `─╮` (flush, unchanged from story 77)
- [ ] Status bar key characters have NO visible background pills
- [ ] Status bar background is continuous across the full terminal width
- [ ] `make ci` passes (lint + tests + 80% coverage)

## Tasks

- [ ] **Task 1:** Remove `rightSuffix` variable and its conditional from `RenderPaneBorder()` in `border.go`. Inline empty string or remove all references. Remove fallback reset. Update comments.
    - Rename `TestRenderPaneBorder_WithActions_SpaceBeforeCorner` to `TestRenderPaneBorder_WithActions_FlushCorner`. Change assertion from `HasSuffix(topLine, " ╮")` to `HasSuffix(topLine, "╭╮")`.
    - `TestRenderPaneBorder_NoActions_FlushCorner` should still pass unchanged.
    - Existing width tests should still pass.
- [ ] **Task 2:** Flatten `renderStatusBar()` in `render.go` — remove outer `bgStyle.Render()` wrapper, style separators and prefix explicitly with `bgStyle.Render()`, pad to terminal width.
    - Update `TestRenderStatusBar_KeyStyleHasConsistentBackground` to verify flat rendering.
    - Existing content tests must still pass.
