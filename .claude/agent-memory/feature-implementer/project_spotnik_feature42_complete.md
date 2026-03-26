---
name: project_spotnik_feature42_complete
description: Feature 42 (Custom Border Renderer): RenderPaneBorder function, PaneBorderColor helper, edge case handling, TestMain with forced TrueColor, ANSI truncation behavior
type: project
---

## Feature 42 — Custom Border Renderer

**What was built:**
- `internal/ui/layout/border.go` — `RenderPaneBorder()` function and `PaneBorderColor()` helper
- `internal/ui/layout/border_test.go` — 28 tests covering all 3 tasks from spec

**Key files:**
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/ui/layout/border.go` — standalone border rendering function (307 lines). Does NOT use Manager — intentionally a pure function so panes can test in isolation.
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/ui/layout/border_test.go` — comprehensive tests including TestMain with forced color profile.

**Patterns established:**
- `RenderPaneBorder(content string, cfg BorderConfig) string` — pure function signature. Content is pre-sized by caller (Width-2 × Height-2). Function pads/truncates to fit.
- `BorderConfig` struct holds everything: Width, Height, Title, ToggleKey, Actions, AccentColor, Focused, FilterQuery, Theme.
- `PaneBorderColor(id PaneID, t theme.Theme) lipgloss.Color` — maps PaneID to Theme accent method. Falls back to `t.ActiveBorder()` for unknown IDs.
- Graceful degradation order: (1) try with all actions, (2) drop actions if too narrow, (3) truncate title if still too narrow.
- `superscripts` var (package-level map) maps 1-8 to ¹-⁸ Unicode superscripts.

**Dependency change:**
- `github.com/muesli/termenv` promoted from indirect to direct dep in go.mod (needed for `termenv.TrueColor` in TestMain).

**TestMain pattern for color testing:**
```go
func TestMain(m *testing.M) {
    lipgloss.SetColorProfile(termenv.TrueColor)
    os.Exit(m.Run())
}
```
This is in `border_test.go` and affects ALL tests in `internal/ui/layout` package (external test package). The existing layout/pane/preset tests only check geometry so this has no side effect on them.

**Gotchas:**
- Without `TestMain` forcing TrueColor, lipgloss emits no ANSI codes in headless test environments (no TTY). Tests checking for ANSI codes or focused vs unfocused differences both fail.
- `truncateToColumns` is called with ANSI-styled strings (leftInner has color applied). The loop strips runes including ANSI byte sequences, which means the terminal reset `\x1b[0m` at the end gets truncated. This is NOT a visible bug because every subsequent border segment applies its own fresh ANSI sequence. The style doesn't leak visually.
- `lipgloss.Width()` correctly measures terminal columns even on strings with ANSI escape codes — use it throughout, never `len()` or `len([]rune())`.
- The `─` in `leftPrefix := "─ "` is a Unicode dash (U+2500), NOT ASCII hyphen. `lipgloss.Width("─ ")` = 2.

**Architecture boundary:**
- `internal/ui/layout/` imports: `github.com/charmbracelet/lipgloss`, `github.com/initgrep-apps/spotnik/internal/ui/theme`
- No imports from `app/`, `api/`, `state/`, or `bubbletea`
- This boundary is intentional per Feature 41 notes

**Testing notes:**
- 28 new tests + existing 43 = 71 total in layout package
- Coverage: 86.9% layout, 84.4% overall
- `stripANSI` helper in test file: simple state machine — enters escape mode on `\x1b`, exits on any letter A-Z or a-z. Correctly handles CSI sequences like `\x1b[38;2;0;255;136m`.
