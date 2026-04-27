---
name: project_spotnik_feature46_complete
description: Feature 46 (Queue Pane Migration): bubble-table integration, filter support, pageSize fix, app test pattern for pre-size view
type: project
---

## Feature 46 — Queue Pane Migration

**What was built:**
- `QueuePane` rewrite use `components.Table` + `components.Filter`
- `layout.Pane` iface impl: `ID()`, `Title()`, `ToggleKey()`, `Actions()`, `SetSize()`, `SetFocused()`, `IsFocused()`
- `f` key = filter on; Esc close; Enter play selected track from filtered result
- `SetPlayingIndex(index int)` added drives `▶` indicator per track
- `Cursor()` kept backcompat (delegates `table.SelectedIndex()`)
- Bug fix `components/table.go`: pageSize overhead `height-1` → `height-6` (header) or `height-4` (no header)

**Key files:**
- `internal/ui/panes/queue.go` — rewrite, 145 add, 170 del
- `internal/ui/panes/queue_test.go` — rewrite, 494 add, 64 del
- `internal/ui/components/table.go` — pageSize fix in `rebuild()` and `SetSize()`
- `internal/app/app_test.go` — 2 tests updated (QUEUE header check → table # check)

**Patterns established:**
- `filter.IsActive()` checked top of `if` → `wasActive` inside always `true`. Skip redundant var; check `!q.filter.IsActive()` after `filter.Update(msg)`
- `refreshRows()` after `filter.Update(msg)` handles still-active + just-closed; `SetFocused(true)` only when filter just closed
- Table focus must set in ctor: `t.SetFocused(focused)` before return — don't rely on `SetFocused()` later
- Filter close → table refocus: `if !q.filter.IsActive() { q.table.SetFocused(true) }`

**App-level test pattern for zero-size view:**
- `app.View()` BEFORE any `WindowSizeMsg` → `a.width=0, a.height=0`
- `render.go` ~L31: `currentView == viewSplash` AND `width=0` → falls to `renderMain()`
- Lets test main layout content w/o splash or size restriction
- Sending `WindowSizeMsg{Width: 240, Height: 40}` returns to `viewSplash` (size known) — DO NOT to test main layout pane content
- Instead: keep `width=0` for basic layout tests; use pane-level unit tests (queue_test.go) for data rendering

**pageSize fix for bubble-table emptyBorder:**
- `emptyBorder` always adds: top border (1) + bottom border (1) + pagination row (1) = 3 min
- `ShowHeader=true`: header (1) + separator (1) + 1 spare = 6 overhead
- `ShowHeader=false`: no header/separator, 3 from border/pagination + 1 spare = 4 overhead
- Formula: `pageSize = height - overhead` (6 or 4); guard `if pageSize < 1 { pageSize = 1 }`
- Verified empirically: temp test binary, measured rendered line count vs pageSize

**Gotchas:**
- `filteredQueue()` call at Enter-time (not stored): filtered idx 0 = first filtered track, not store idx 0
- Width < 60: bubble-table truncates `"Save Your Tears"` → `"Save Your…"` — skip full track name check at narrow widths
- Width=120: queue pane gets `120*28/100-2 = 31` chars — too narrow for "Save Your Tears"
- `wasActive` antipattern: inside `if q.filter.IsActive() { ... }`, `wasActive = q.filter.IsActive()` always true

**Testing notes:**
- Final coverage: 87.5% panes, 85.1% total
- `var _ layout.Pane = &QueuePane{}` compile-time check 2x in test file (once `TestQueuePane_ImplementsLayoutPane`, once package level)
- Filter tests type runes one-at-a-time in loop: `for _, r := range "rock" { m, _ = pane.Update(tea.KeyMsg{...Runes: []rune{r}}) }`
- `TestQueuePane_LargeQueue` makes 200-item queue, verifies no panic / row count visible