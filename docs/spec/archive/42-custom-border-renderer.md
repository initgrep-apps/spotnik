# Feature 42 — Custom Border Renderer

> **Feature:** Build a `RenderPaneBorder()` function that draws btop-style pane borders
> with embedded title, toggle key number, and action shortcuts directly in the border line.
> This replaces the current `renderPaneWithBorder()` which uses plain `lipgloss.RoundedBorder()`.

## Context

The current border rendering (`internal/app/render.go:139-149`) uses `lipgloss.RoundedBorder()`
with a single accent color. It has no title, no toggle key indicator, and no action shortcuts.

The new DESIGN.md (§5 "Embedded Shortcut Borders") specifies btop-style borders where:
- The top border line contains the pane title, toggle key superscript, and action shortcuts
- Each pane has a distinct accent color (per-pane border tokens from Feature 40)
- Focused panes show full accent color; unfocused panes use dimmed/faint borders
- Filter mode replaces action shortcuts with the filter query

**Design reference:** `docs/DESIGN.md` §5 (Embedded Shortcut Borders), §10 (Per-Pane Border Colors)

**Depends on:** Feature 40 (theme tokens for per-pane border colors)

---

## Design Diagram

```
Border Anatomy (DESIGN.md §5):

╭─ ¹Playlists ──────────────────── ᐅfilter ─── ᐅnew ╮
│                                                    │
│  (pane content)                                    │
│                                                    │
╰────────────────────────────────────────────────────╯

Top border components:
  1. ╭─    → rounded corner + dash + space
  2. ¹     → superscript toggle key (1-8) in KeyHint() color
  3. Playlists → pane title in accent color (bold when focused)
  4. ─────── → dash fill
  5. ᐅfilter ─ ᐅnew → action shortcuts: ᐅ prefix, key in KeyHint(), label in TextMuted()
  6.  ╮     → space + rounded corner

Filter mode:
╭─ ¹Queue ────────── filtering: "rock" ─── ᐅEsc close ╮

Focused vs Unfocused:
  Focused:   full accent color for all border characters
  Unfocused: lipgloss.NewStyle().Faint(true) on border characters
```

---

## Task 1: Implement RenderPaneBorder function

**Problem:** No function renders btop-style borders with embedded text.

**Fix:**

Create `internal/ui/layout/border.go`:

```go
package layout

import (
    "strings"

    "github.com/charmbracelet/lipgloss"
    "github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// BorderConfig holds all data needed to render a pane border.
type BorderConfig struct {
    Width       int            // Total border width in terminal columns
    Height      int            // Total border height in terminal rows
    Title       string         // Pane title (e.g., "Playlists")
    ToggleKey   int            // Toggle key number (1-8), 0 for none
    Actions     []Action       // Pane-specific shortcuts
    AccentColor lipgloss.Color // Per-pane border color
    Focused     bool           // Whether pane has keyboard focus
    FilterQuery string         // Non-empty when filter is active (replaces actions)
    Theme       theme.Theme    // For KeyHint, TextMuted colors
}

// RenderPaneBorder wraps content in a btop-style border.
// The top border line contains the toggle key, title, and action shortcuts.
// Content is expected to be pre-sized to fit inside the border (Width-2 x Height-2).
func RenderPaneBorder(content string, cfg BorderConfig) string
```

**Implementation details:**

1. **Top border line construction:**
   - Start: `╭─ `
   - Toggle key: superscript digit (`¹²³⁴⁵⁶⁷⁸`) in `KeyHint()` color, skip if ToggleKey=0
   - Title: in accent color, bold if focused
   - Dash fill: `─` characters to fill remaining space
   - Actions (or filter query): right-aligned before the corner
     - Each action: `ᐅ` + key in `KeyHint()` + space + label in `TextMuted()`
     - Separated by ` ─── `
   - End: ` ╮`

2. **Side borders:** `│` + content line padded to Width-2 + `│`

3. **Bottom border:** `╰` + `─` × (Width-2) + `╯`

4. **Color application:**
   - Focused: all border chars (`╭╮╰╯─│`) colored with `AccentColor`
   - Unfocused: all border chars rendered with `lipgloss.NewStyle().Faint(true)`

5. **Filter mode:** When `FilterQuery` is non-empty, replace action shortcuts with:
   `filtering: "query" ─── ᐅEsc close`

6. **Superscript mapping:**
   ```go
   var superscripts = map[int]string{
       1: "¹", 2: "²", 3: "³", 4: "⁴",
       5: "⁵", 6: "⁶", 7: "⁷", 8: "⁸",
   }
   ```

**Files:**
- Create: `internal/ui/layout/border.go`

**Tests:**
- Unit: Basic border with title only — correct corner characters
- Unit: Border with toggle key — superscript appears after `╭─ `
- Unit: Border with 2 actions — both rendered right-aligned in top border
- Unit: Border width matches requested width exactly
- Unit: Border height matches requested height exactly
- Unit: Content lines are padded/truncated to fit inside border
- Unit: Filter mode — actions replaced with filter query text
- Unit: ToggleKey=0 — no superscript rendered
- Unit: Empty actions — only title and dashes in top border
- Unit: Focused=true — accent color applied (verify styled output contains color)
- Unit: Focused=false — faint style applied

**Commit:** `feat(layout): btop-style RenderPaneBorder with embedded shortcuts`

---

## Task 2: Handle edge cases and content truncation

**Problem:** Content may not fit the border dimensions, or border may be too narrow for all elements.

**Fix:**

Add to `border.go`:

1. **Narrow border handling:** If border width is too small to fit title + actions:
   - First, drop actions (show only title)
   - If still too narrow, truncate title with `…`
   - Minimum width: 10 characters (corner + 3 dash + truncated title + 3 dash + corner)

2. **Content line handling:**
   - Each content line is truncated to `Width - 2` using `lipgloss.Width()` for accurate measurement
   - Lines shorter than `Width - 2` are padded with spaces
   - If content has fewer lines than `Height - 2`, pad with empty lines
   - If content has more lines than `Height - 2`, truncate (should not happen if caller sized correctly)

3. **Unicode safety:** Use `lipgloss.Width()` for all width measurements (handles CJK, emoji, combining marks). The `ᐅ` character (U+1405) is 1 cell wide. Superscripts (`¹²³⁴⁵⁶⁷⁸`) are also 1 cell wide.

**Files:**
- Modify: `internal/ui/layout/border.go`

**Tests:**
- Unit: Very narrow border (width=15) — title truncated, no actions
- Unit: Minimum border (width=10) — still renders valid border
- Unit: Content shorter than height — padded with empty lines
- Unit: Content wider than width — truncated with `…`
- Unit: Unicode content (CJK characters, emoji) — width measured correctly
- Unit: ᐅ character width measurement

**Commit:** `feat(layout): border edge case handling and content truncation`

---

## Task 3: Border integration tests

**Problem:** Need to verify borders work end-to-end with realistic pane configurations.

**Fix:**

Create comprehensive tests simulating real pane borders:

**Files:**
- Create: `internal/ui/layout/border_test.go`

**Tests:**
- Integration: NowPlaying border — title "Now Playing", key=1, actions=[shuffle, repeat], green accent
- Integration: Playlists border — title "Playlists", key=3, actions=[filter, new, rename, delete], blue accent
- Integration: Queue border with active filter — shows `filtering: "rock"` instead of actions
- Integration: Page B RequestFlow border — title "Request Flow", key=0 (no superscript), orange accent
- Integration: Border output is exactly Width × Height characters (measure each line)
- Integration: Multiple borders side-by-side with `lipgloss.JoinHorizontal` — no overlap
- Integration: Border with all 5 themes — accent colors change correctly

**Commit:** `test(layout): comprehensive border renderer tests`

---

## Acceptance Criteria

- [ ] `RenderPaneBorder()` produces btop-style borders matching DESIGN.md §5 exactly
- [ ] Top border contains: corner, toggle key superscript, title, dashes, action shortcuts, corner
- [ ] Superscript digits (`¹`-`⁸`) render correctly for keys 1-8
- [ ] `ᐅ` prefix used for action labels with key in `KeyHint()` and label in `TextMuted()`
- [ ] Focused borders use full accent color, unfocused use `Faint(true)`
- [ ] Filter mode replaces actions with `filtering: "query"` text
- [ ] Output dimensions exactly match requested Width × Height
- [ ] Content is truncated/padded to fit inside border perfectly
- [ ] Unicode-safe width measurement using `lipgloss.Width()`
- [ ] No hardcoded hex colors — all colors come from Theme interface
- [ ] `make ci` passes

---

## Notes

- This function replaces `renderPaneWithBorder()` in `render.go`. The old function will be
  deleted in Feature 49 (App Migration) when the grid renderer is wired up.
- The `ᐅ` character (U+1405, Canadian Syllabics PA) is chosen for btop-style action prefixes.
  If terminal support is a concern, the fallback `›` (U+203A) can be used. The DESIGN.md
  specifies `ᐅ` as primary.
- Border rendering is intentionally a standalone function (not a method on Manager) so panes
  can test their rendering in isolation without needing a full layout setup.
- The accent color for each pane comes from the Theme's `PaneBorder*()` tokens (Feature 40).
  The caller maps PaneID → accent color. A helper function `PaneBorderColor(id PaneID, t theme.Theme) lipgloss.Color`
  may be added for convenience.
