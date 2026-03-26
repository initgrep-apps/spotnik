---
name: project_spotnik_feature43_complete
description: Feature 43 (Reusable Components): bubble-table wrapper, Filter component, truncation utilities, dependency management pattern
type: project
---

## Feature 43 — Reusable Components

**What was built:**
- `internal/ui/layout/truncate.go` — `Truncate`, `PadRight`, `TruncateOrPad` using `lipgloss.Width()`
- `internal/ui/components/table.go` — `Table` wrapping `github.com/evertras/bubble-table/table`
- `internal/ui/components/filter.go` — `Filter` wrapping `bubbles/textinput`
- Tests: 25 truncation tests, 11 Table tests, 16 Filter tests, 6 integration tests

**Key files:**
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/ui/layout/truncate.go` — rune-aware truncation
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/ui/components/table.go` — Table wrapper (211 lines)
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/ui/components/filter.go` — Filter component (155 lines)

**Patterns established:**
- Table uses `WithRowStyleFunc` for selection + playing indicator (do NOT set HighlightStyle when using WithRowStyleFunc — they conflict per bubble-table docs)
- Playing indicator replaces the first column value with a `btable.NewStyledCell(playingSymbol, style)` keyed to `Columns[0].Key`
- Filter stores `query` field separately from `input.Value()` — query is updated in `Update()` and preserved on Enter but cleared on Esc
- `SetWidth()` is called by the pane from its SetSize — NOT inside `View()` — to keep View() side-effect-free per Elm architecture rules
- `emptyBorder` var with space characters hides bubble-table's built-in border (pane border handles the visible border)
- `GetHighlightedRowIndex()` is on `*Model` (pointer receiver) — call as `(&t.inner).GetHighlightedRowIndex()`

**Dependency management gotcha:**
- `go get` adds the dependency; `go mod tidy` removes it if nothing imports it yet (before code is written)
- The correct workflow: add go.mod entry (indirect), write the code that imports it, then `go mod tidy` promotes it to direct
- Always run `go mod tidy` before the final commit to promote indirect→direct
- The `make ci` tidy-check runs `go mod tidy` then `git diff go.mod go.sum` — so go.mod must already be in its post-tidy state before committing

**bubble-table API notes (v0.19.2):**
- Constructor: `btable.New([]btable.Column{...})` — not `table.NewModel`
- Flex columns: `btable.NewFlexColumn(key, header, flexFactor)`
- Column style: `.WithStyle(lipgloss.NewStyle().Foreground(color))`
- Border: `.Border(customBorder)` — not a method option
- `WithRowStyleFunc(func(RowStyleFuncInput) lipgloss.Style)` — overrides HighlightStyle
- `Focused(bool)` — enables/disables keyboard nav
- `Update(msg)` returns `(Model, tea.Cmd)` not `(tea.Model, tea.Cmd)`
- `GetHighlightedRowIndex()` on `*Model` (pointer receiver)
- `WithHeaderVisibility(bool)` to show/hide header row

**Gotchas:**
- `rebuild()` is called in `NewTable` before any rows are set — OK because `applyRows()` handles nil/empty rows
- `SetSize` does NOT call `rebuild()` or `applyRows()` — it only calls `WithTargetWidth` and `WithPageSize`. Rows stay intact because `WithTargetWidth` returns a new Model with existing rowStyleFunc/rows preserved
- `View()` must be pure — do not set `input.Width` inside View() — use `SetWidth()` method instead
- Lint will flag any unused constants — remove `playingIndicatorKey` if not used

**Testing notes:**
- `table_test.go` and `filter_test.go` share `testTheme()` and `makeColumns()` helpers in `table_test.go` (same `_test` package)
- Integration tests in `integration_test.go` add `newKeyRune()` helper
- Coverage: 94.7% components, 87.8% layout, 84.6% overall
