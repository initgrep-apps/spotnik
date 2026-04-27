---
name: project_spotnik_feature62_complete
description: Feature 62 (Request Flow Boxed Layout): three sub-boxes, dual arrows, viewBoxed/viewFlat dispatch, ANSI-safe test helpers
type: project
---

## Feature 62 — Request Flow Boxed Layout

**What was built:**
- `renderSubBox(title, lines, width)` — pure helper, draws rounded-corner bordered box; returns empty string for width < 8; top border = 3 concatenated ANSI segments (╭─ prefix + styled title + suffix+dashes+╮)
- `renderRightArrow(r, colWidth)` — GATEWAY→SPOTIFY arrow reflects HTTP outcome (2xx=animated/Success, 429=╳/Warning, 5xx=animated/Error, 0=╳/Muted)
- `gatewayStateLines()` — extracted from `renderGatewayState()` to kill duplication; `renderGatewayState()` now delegates via `strings.Join`
- `buildAppBoxLines / buildGatewayBoxLines / buildSpotifyBoxLines` — content generators, return `[]string` padded to `maxRows`
- `buildLeftArrowLines / buildRightArrowLines` — dual arrow column builders
- `View()` → `viewBoxed()` (width ≥ 60) or `viewFlat()` (width < 60)
- `viewBoxed()`: proportional widths (APP ~25%, arrow ~8%, GATEWAY ~26%, arrow ~8%, SPOTIFY ~20%); arrow blocks padded with blank top+bottom lines to align with box border rows
- `viewFlat()`: exact original View() body, no changes

**Key files:**
- `internal/ui/panes/requestflow_boxed.go` — all new methods (246 lines)
- `internal/ui/panes/requestflow_boxed_test.go` — internal package tests for unexported helpers
- `internal/ui/panes/requestflow_pane.go` — View() restructured, renderGatewayState() delegates
- `internal/ui/panes/requestflow_pane_test.go` — new boxed layout + flat fallback tests

**Patterns established:**
- Sub-box borders built manually (3 lipgloss.Render calls per top border) — title needs different bold styling from dashes
- `viewContainsBox(output, title)` test helper checks per-line for `╭` + `title` on same line — required because ANSI codes separate border chars from title text
- Internal test file (`package panes`, not `panes_test`) needed to test unexported methods; use `newInternalTestPane()` helper
- `buildAppBoxLines` uses `strings.TrimRight(renderAppEntry(r, 200), " ")` — pass big colWidth to skip padding, trim trailing spaces; box renderer handles final sizing

**Arrow alignment:**
- Arrow block = `blankArrow + "\n" + strings.Join(arrowLines, "\n") + "\n" + blankArrow`
- Creates `innerRows + 2` lines matching box height (top border blank + content rows + bottom border blank)
- `lipgloss.JoinHorizontal(lipgloss.Top, ...)` aligns all elements at tops

**Gotchas:**
- `╭─ APP` NOT a literal string in View() output — ANSI codes separate `╭─ ` (styled border) from `APP` (bold title). Tests check for `╭` and `APP` on same line, not combined string.
- `gofmt` needed after impl — caught by `make fmt-check`
- `statusStripHeight := 1` not "unused" — used in `boxAreaHeight := p.height - statusStripHeight - 1` next line

**Testing notes:**
- 61 tests in panes package, 0 failures
- Coverage: 89.9% for `internal/ui/panes`, 86.4% overall
- Internal tests in `requestflow_boxed_test.go` cover all unexported helpers directly
- External tests in `requestflow_pane_test.go` cover View() end-to-end