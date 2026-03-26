---
name: project_spotnik_feature46_complete
description: Feature 46 (Queue Pane Migration): bubble-table integration, filter support, pageSize fix, app test pattern for pre-size view
type: project
---

## Feature 46 — Queue Pane Migration

**What was built:**
- `QueuePane` fully rewritten to use `components.Table` + `components.Filter`
- `layout.Pane` interface implemented: `ID()`, `Title()`, `ToggleKey()`, `Actions()`, `SetSize()`, `SetFocused()`, `IsFocused()`
- `f` key activates filter; Esc closes it; Enter plays selected track from filtered result
- `SetPlayingIndex(index int)` added to drive the `▶` indicator per track
- `Cursor()` kept for backward compatibility (now delegates to `table.SelectedIndex()`)
- Bug fix in `components/table.go`: pageSize overhead was `height-1` → corrected to `height-6` (header visible) or `height-4` (no header)

**Key files:**
- `internal/ui/panes/queue.go` — complete rewrite, 145 additions, 170 deletions
- `internal/ui/panes/queue_test.go` — comprehensive rewrite, 494 additions, 64 deletions
- `internal/ui/components/table.go` — pageSize fix in `rebuild()` and `SetSize()`
- `internal/app/app_test.go` — two tests updated (QUEUE header check → table # check)

**Patterns established:**
- When `filter.IsActive()` is checked at the top of an `if` block, `wasActive` inside that block is always `true` — don't assign a redundant variable; just check `!q.filter.IsActive()` after `filter.Update(msg)`
- `refreshRows()` call after `filter.Update(msg)` handles both the still-active and just-closed cases; `SetFocused(true)` only when the filter just closed
- Table focus must be explicitly set in the constructor: `t.SetFocused(focused)` before returning the pane — not relying on `SetFocused()` being called later
- Filter close → table refocus pattern: `if !q.filter.IsActive() { q.table.SetFocused(true) }`

**App-level test pattern for zero-size view:**
- Calling `app.View()` BEFORE sending any `WindowSizeMsg` gives `a.width=0, a.height=0`
- `render.go` line ~31: when `currentView == viewSplash` AND `width=0`, it falls through to `renderMain()`
- This lets you test the main layout content without triggering the splash or size restriction
- Sending `WindowSizeMsg{Width: 240, Height: 40}` puts you back in `viewSplash` (size is now known) — do NOT do this to test main layout pane content
- Instead: keep `width=0` for basic layout tests; use pane-level unit tests (queue_test.go) for data rendering tests

**pageSize fix for bubble-table emptyBorder:**
- `emptyBorder` always adds: top border (1 line) + bottom border (1 line) + pagination row (1 line) = 3 lines minimum
- With `ShowHeader=true`: header row (1 line) + separator row (1 line) + 1 spare = 6 total overhead
- With `ShowHeader=false`: no header/separator, but still 3 from border/pagination + 1 spare = 4 total overhead
- Formula: `pageSize = height - overhead` (6 or 4); guard with `if pageSize < 1 { pageSize = 1 }`
- This was verified empirically by building a temp test binary and measuring rendered line count vs pageSize

**Gotchas:**
- `filteredQueue()` must be called at Enter-time (not stored): filtered index 0 is the first filtered track, not store index 0
- At width < 60, bubble-table truncates `"Save Your Tears"` to `"Save Your…"` — don't check full track names at narrow widths
- At Width=120, queue pane gets `120*28/100-2 = 31` content chars — too narrow for "Save Your Tears"
- `wasActive` anti-pattern: inside `if q.filter.IsActive() { ... }`, `wasActive = q.filter.IsActive()` is always true

**Testing notes:**
- Final coverage: 87.5% for panes, 85.1% total
- `var _ layout.Pane = &QueuePane{}` compile-time check placed twice in test file (once as `TestQueuePane_ImplementsLayoutPane`, once at package level)
- Filter tests type runes one at a time in a loop: `for _, r := range "rock" { m, _ = pane.Update(tea.KeyMsg{...Runes: []rune{r}}) }`
- `TestQueuePane_LargeQueue` creates 200-item queue and verifies no panic / row count visible
