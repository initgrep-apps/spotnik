---
name: project_spotnik_feature52_complete
description: Feature 52 (Mouse Scroll + Responsive Behavior): mouse option, handleMouseMsg, PaneAt hit-test, minTerm constants
type: project
---

## Feature 52 — Mouse Scroll + Responsive Behavior

**What was built:**
- Added `tea.WithMouseCellMotion()` to `cmd/root.go` program opts (Task 1)
- Added `case tea.MouseMsg:` in `app.go handleMsg()` + `handleMouseMsg()` in `routing.go` (Task 2)
- Extracted `minTermWidth = 120` / `minTermHeight = 30` consts in `render.go` (Task 3)

**Key files:**
- `cmd/root.go` — `tea.WithMouseCellMotion()` added to `tea.NewProgram` opts
- `internal/app/app.go` — `case tea.MouseMsg:` dispatches to `handleMouseMsg`
- `internal/app/routing.go` — `handleMouseMsg(m tea.MouseMsg) tea.Cmd` method
- `internal/app/render.go` — named consts + `renderTooSmall()` uses `minTermWidth`/`minTermHeight`
- `internal/app/app_test.go` — 9 new mouse scroll tests
- `internal/app/render_test.go` — 7 new responsive tests

**Patterns established:**
- `handleMouseMsg` returns only `tea.Cmd` (not `tea.Model, tea.Cmd`) — pane state mutates in-place via pointer receiver. Matches `handleKeyMsg` pattern for pane routing
- Mouse scroll guard order: overlay → action → button → PaneAt → route to pane
- Wheel events from bubbletea v1.3.10 use `Action == MouseActionPress` (not Motion/Release) — per bubbletea source `parseSGRMouseEvent`
- `PaneAt()` hit-test safe before Resize() (returns -1 when height/rects empty)

**Gotchas:**
- `tea.MouseMsg` wheel events DO have `Action == MouseActionPress` in bubbletea v1.3.10 — counter-intuitive, verified from source
- During auth/splash views, `PaneAt` returns -1 for all (safe early return) — layout not Resize()d. No explicit viewMode check needed in handleMouseMsg
- "focus doesn't change" tests pass vacuously pre-implementation — key property is pane routing without focus change. Verified by running pre-impl (focus preservation already implicit)

**Testing notes:**
- 16 new tests: 9 app_test.go (mouse scroll), 7 render_test.go (responsive)
- Mouse scroll tests use `require.True(t, a.SearchOpen())` to assert overlay state before testing suppression
- Coverage: 86.2% overall, 81.9% on internal/app

**Architecture:**
- `handleMouseMsg` placed in `routing.go` next to `handleKeyMsg` — input handlers grouped together
- No new imports — `layout` already imported by routing.go