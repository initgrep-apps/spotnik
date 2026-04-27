---
name: project_spotnik_feature176_complete
description: Story 176 (GatewayLivePane): FilterQueryPane interface, Enter-to-apply filter, render.go hook, width guard in buildTableRows
type: project
---

## Story 176 — GatewayLivePane

**Built:**
- `internal/ui/layout/pane.go`: added `FilterQueryPane` interface (`ActiveFilterQuery() string`) — first consumer of dead `BorderConfig.FilterQuery` path in `layout/border.go`
- `internal/app/render.go`: type-assert `FilterQueryPane`, populate `cfg.FilterQuery` before `RenderPaneBorder` (lines ~421–430)
- `internal/ui/panes/gateway_live_pane.go`: full impl — 500-entry ring buffer, 12 event kind mappings, Enter-to-apply filter, three-mode Esc state machine
- `internal/ui/panes/gateway_live_pane_test.go`: 13 behavioural tests

**Key files:**
- `internal/ui/layout/pane.go` — `FilterQueryPane` interface, bottom of file
- `internal/app/render.go` — FilterQueryPane type-assert ~line 430
- `internal/ui/panes/gateway_live_pane.go` — full pane impl

**Patterns:**
- **FilterQueryPane interface**: panes exposing committed filter query for border impl `ActiveFilterQuery() string`. render.go type-asserts, populates `cfg.FilterQuery`. Border renders `filtering: "query"` (not `filter(query)` — spec informal, actual text differs).
- **Enter-to-apply filter**: handle Enter at pane level (not via `filter.Update(Enter)`). Read `filter.Query()` BEFORE `filter.Toggle()` — Toggle() clears query. Then `buildTableRows()` must use `activeQuery` (committed) not `filter.Query()` (live, empty).
- **Single-column no-header table w/ ANSI content**: use `Color: ""` (empty lipgloss.Color) for column so bubble-table applies no foreground, preserving `ListRow.Render()` ANSI intent colors.
- **Three-mode Esc state machine**: (1) filter active → `filter.Toggle()` cancel; (2) filter inactive + activeQuery != "" → clear activeQuery, rebuild; (3) filter inactive + activeQuery == "" → `table.GotoTop()`

**Gotchas:**
- `buildTableRows()` must guard `if p.width == 0 { return }` — called from SetTheme before SetSize, `ListRow.Render(p.width - 2)` = `Render(-2)` silently renders empty strings
- `matchString` pre-lowercased at build: `strings.ToLower(label)` — avoids per-call allocations; `buildTableRows` query also lowercased, compared via `strings.Contains`
- `RebuildTableTheme` unsuitable for no-header tables (always `ShowHeader: true`) — inline table reconstruction in `SetTheme` instead
- border.go renders `filtering: "query"` not `filter(query)` — spec informal
- `EventBackoffExpired` silently returns `(zero, false)` from `buildGatewayLiveRow`; `default` case also returns false for future events

**Testing:**
- Vacuous assertion smell: `if !cond { _ = view }` does nothing — caught in review, fixed to real `t.Errorf`
- Docstring "tests three modes" when testing one = misleading — fix comment to match scope
- `newTestGatewayLivePane(t)` helper pattern consistent w/ `newTestGatewayHealthPane(t)` in health pane tests