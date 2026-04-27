---
name: project_spotnik_feature42_complete
description: Feature 42 (Custom Border Renderer): RenderPaneBorder function, PaneBorderColor helper, edge case handling, TestMain with forced TrueColor, ANSI truncation behavior
type: project
---

## Feature 42 — Custom Border Renderer

**What was built:**
- `internal/ui/layout/border.go` — `RenderPaneBorder()` fn + `PaneBorderColor()` helper
- `internal/ui/layout/border_test.go` — 28 tests cover all 3 spec tasks

**Key files:**
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/ui/layout/border.go` — standalone border render fn (307 lines). No Manager — pure fn so panes test in isolation.
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/ui/layout/border_test.go` — tests + TestMain with forced color profile.

**Patterns established:**
- `RenderPaneBorder(content string, cfg BorderConfig) string` — pure fn sig. Content pre-sized by caller (Width-2 × Height-2). Fn pads/truncates to fit.
- `BorderConfig` struct holds all: Width, Height, Title, ToggleKey, Actions, AccentColor, Focused, FilterQuery, Theme.
- `PaneBorderColor(id PaneID, t theme.Theme) lipgloss.Color` — map PaneID → Theme accent method. Fallback `t.ActiveBorder()` for unknown IDs.
- Degradation order: (1) all actions, (2) drop actions if narrow, (3) truncate title if still narrow.
- `superscripts` var (package-level map) 1-8 → ¹-⁸ Unicode superscripts.

**Dependency change:**
- `github.com/muesli/termenv` indirect → direct dep in go.mod (need `termenv.TrueColor` in TestMain).

**TestMain pattern for color testing:**
```go
func TestMain(m *testing.M) {
    lipgloss.SetColorProfile(termenv.TrueColor)
    os.Exit(m.Run())
}
```
In `border_test.go`, affects ALL tests in `internal/ui/layout` package (external test package). Existing layout/pane/preset tests only check geometry — no side effect.

**Gotchas:**
- No `TestMain` TrueColor → lipgloss emits no ANSI in headless tests (no TTY). Tests checking ANSI or focus diffs both fail.
- `truncateToColumns` called with ANSI-styled strings (leftInner colored). Loop strips runes incl ANSI bytes — terminal reset `\x1b[0m` gets truncated. NOT visible bug: every next border segment applies fresh ANSI. Style doesn't leak visually.
- `lipgloss.Width()` measures terminal cols correctly on ANSI strings — use everywhere, never `len()` or `len([]rune())`.
- `─` in `leftPrefix := "─ "` is Unicode dash (U+2500), NOT ASCII hyphen. `lipgloss.Width("─ ")` = 2.

**Architecture boundary:**
- `internal/ui/layout/` imports: `github.com/charmbracelet/lipgloss`, `github.com/initgrep-apps/spotnik/internal/ui/theme`
- No imports from `app/`, `api/`, `state/`, or `bubbletea`
- Boundary intentional per Feature 41 notes

**Testing notes:**
- 28 new tests + existing 43 = 71 total in layout package
- Coverage: 86.9% layout, 84.4% overall
- `stripANSI` helper in test file: state machine — enter escape mode on `\x1b`, exit on any letter A-Z or a-z. Handles CSI sequences like `\x1b[38;2;0;255;136m`.