---
name: project_spotnik_feature176_complete
description: Story 176 (GatewayLivePane): FilterQueryPane interface, Enter-to-apply filter, render.go hook, width guard in buildTableRows
type: project
---

## Story 176 — GatewayLivePane

**What was built:**
- `internal/ui/layout/pane.go`: Added `FilterQueryPane` interface (`ActiveFilterQuery() string`) — first consumer of the previously-dead `BorderConfig.FilterQuery` code path in `layout/border.go`
- `internal/app/render.go`: Type-assert `FilterQueryPane` and populate `cfg.FilterQuery` before `RenderPaneBorder` call (lines ~421–430)
- `internal/ui/panes/gateway_live_pane.go`: Full implementation — 500-entry ring buffer, 12 event kind mappings, Enter-to-apply filter, three-mode Esc state machine
- `internal/ui/panes/gateway_live_pane_test.go`: 13 behavioural tests

**Key files:**
- `internal/ui/layout/pane.go` — `FilterQueryPane` interface at bottom of file
- `internal/app/render.go` — FilterQueryPane type-assert around line 430
- `internal/ui/panes/gateway_live_pane.go` — full pane implementation

**Patterns established:**
- **FilterQueryPane interface**: Panes that expose committed filter query for border rendering implement `ActiveFilterQuery() string`. render.go type-asserts and populates `cfg.FilterQuery`. Border renders `filtering: "query"` (not `filter(query)` — the spec description is informal, actual text differs).
- **Enter-to-apply filter pattern**: Handle Enter at pane level (not via `filter.Update(Enter)`). Read `filter.Query()` BEFORE calling `filter.Toggle()` — Toggle() clears the query. Then `buildTableRows()` must use `activeQuery` (committed) not `filter.Query()` (live, now empty).
- **Single-column no-header table with ANSI content**: Use `Color: ""` (empty lipgloss.Color) for column so bubble-table applies no foreground, preserving `ListRow.Render()` ANSI intent colors.
- **Three-mode Esc state machine**: (1) filter active → `filter.Toggle()` cancel; (2) filter inactive + activeQuery != "" → clear activeQuery, rebuild; (3) filter inactive + activeQuery == "" → `table.GotoTop()`

**Gotchas:**
- `buildTableRows()` must guard `if p.width == 0 { return }` — called from SetTheme before SetSize, `ListRow.Render(p.width - 2)` = `Render(-2)` silently renders empty strings
- `matchString` pre-lowercased at build time: `strings.ToLower(label)` — avoids per-filter-call allocations; `buildTableRows` query also lowercased and compared with `strings.Contains`
- `RebuildTableTheme` not suitable for no-header tables (it always uses `ShowHeader: true`) — inline table reconstruction in `SetTheme` instead
- border.go renders `filtering: "query"` not `filter(query)` — spec description was informal
- `EventBackoffExpired` silently returns `(zero, false)` from `buildGatewayLiveRow`; `default` case also returns false for future events

**Testing notes:**
- Vacuous assertion pattern is a test smell: `if !cond { _ = view }` does nothing — caught in review, fixed to real `t.Errorf`
- Docstring saying "tests three modes" when only testing one mode is misleading — fix comment to match actual scope
- `newTestGatewayLivePane(t)` helper pattern consistent with `newTestGatewayHealthPane(t)` in health pane tests
