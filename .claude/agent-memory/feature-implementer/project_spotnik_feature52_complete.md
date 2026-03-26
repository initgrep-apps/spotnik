---
name: project_spotnik_feature52_complete
description: Feature 52 (Mouse Scroll + Responsive Behavior): mouse option, handleMouseMsg, PaneAt hit-test, minTerm constants
type: project
---

## Feature 52 — Mouse Scroll + Responsive Behavior

**What was built:**
- Added `tea.WithMouseCellMotion()` to `cmd/root.go` program options (Task 1)
- Added `case tea.MouseMsg:` in `app.go handleMsg()` + `handleMouseMsg()` in `routing.go` (Task 2)
- Extracted `minTermWidth = 120` / `minTermHeight = 30` constants in `render.go` (Task 3)

**Key files:**
- `cmd/root.go` — `tea.WithMouseCellMotion()` added to `tea.NewProgram` options
- `internal/app/app.go` — `case tea.MouseMsg:` dispatches to `handleMouseMsg`
- `internal/app/routing.go` — `handleMouseMsg(m tea.MouseMsg) tea.Cmd` method
- `internal/app/render.go` — named constants + `renderTooSmall()` uses `minTermWidth`/`minTermHeight`
- `internal/app/app_test.go` — 9 new mouse scroll tests
- `internal/app/render_test.go` — 7 new responsive behavior tests

**Patterns established:**
- `handleMouseMsg` returns only `tea.Cmd` (not `tea.Model, tea.Cmd`) because the pane state mutation happens in-place via pointer receiver — consistent with `handleKeyMsg` pattern for pane routing
- Mouse scroll guard order: overlay check → action check → button check → PaneAt → route to pane
- Wheel events from bubbletea v1.3.10 use `Action == MouseActionPress` (not Motion or Release) — this is per the bubbletea source in parseSGRMouseEvent
- The `PaneAt()` hit-test is safe before Resize() (returns -1 for everything when height/rects are empty)

**Gotchas:**
- `tea.MouseMsg` wheel events DO have `Action == MouseActionPress` in bubbletea v1.3.10 — counter-intuitive but verified from source
- During auth/splash views, `PaneAt` returns -1 for everything (safe early return) since layout hasn't been Resize()d — no explicit viewMode check needed in handleMouseMsg
- Tests for "focus doesn't change" pass vacuously before implementation — the important behavioral property is the pane routing without focus change. Tests verified by running them before implementation (confirmed focus preservation was already implicit)

**Testing notes:**
- 16 new tests: 9 in app_test.go (mouse scroll), 7 in render_test.go (responsive)
- Mouse scroll tests use `require.True(t, a.SearchOpen())` to assert overlay state before testing suppression
- Coverage: 86.2% overall, 81.9% on internal/app

**Architecture:**
- `handleMouseMsg` is placed in `routing.go` alongside `handleKeyMsg` — consistent grouping of input handlers
- No new imports required — `layout` was already imported by routing.go
