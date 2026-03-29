---
name: project_spotnik_feature62_complete
description: Feature 62 (Request Flow Boxed Layout): three sub-boxes, dual arrows, viewBoxed/viewFlat dispatch, ANSI-safe test helpers
type: project
---

## Feature 62 — Request Flow Boxed Layout

**What was built:**
- `renderSubBox(title, lines, width)` — pure helper drawing rounded-corner bordered box; returns empty string for width < 8; top border constructed as 3 concatenated ANSI segments (╭─ prefix + styled title + suffix+dashes+╮)
- `renderRightArrow(r, colWidth)` — GATEWAY→SPOTIFY arrow reflecting HTTP outcome (2xx=animated/Success, 429=╳/Warning, 5xx=animated/Error, 0=╳/Muted)
- `gatewayStateLines()` — extracted from `renderGatewayState()` to eliminate duplication; `renderGatewayState()` now delegates to it via `strings.Join`
- `buildAppBoxLines / buildGatewayBoxLines / buildSpotifyBoxLines` — content generators returning `[]string` padded to `maxRows`
- `buildLeftArrowLines / buildRightArrowLines` — dual arrow column builders
- `View()` → `viewBoxed()` (width ≥ 60) or `viewFlat()` (width < 60)
- `viewBoxed()`: proportional widths (APP ~25%, arrow ~8%, GATEWAY ~26%, arrow ~8%, SPOTIFY ~20%), arrow blocks padded with blank top+bottom lines to align with box border rows
- `viewFlat()`: exact original View() body, no changes

**Key files:**
- `internal/ui/panes/requestflow_boxed.go` — all new methods (246 lines)
- `internal/ui/panes/requestflow_boxed_test.go` — internal package tests for unexported helpers
- `internal/ui/panes/requestflow_pane.go` — View() restructured, renderGatewayState() now delegates
- `internal/ui/panes/requestflow_pane_test.go` — new boxed layout + flat fallback tests

**Patterns established:**
- Sub-box borders are constructed manually (3 lipgloss.Render calls per top border) since title needs different bold styling from the dashes
- `viewContainsBox(output, title)` test helper checks per-line for `╭` + `title` on same line — required because ANSI codes separate border chars from title text
- Internal test file (`package panes`, not `panes_test`) needed to test unexported methods; use `newInternalTestPane()` helper
- `buildAppBoxLines` uses `strings.TrimRight(renderAppEntry(r, 200), " ")` — pass large colWidth to avoid padding, then trim trailing spaces; box renderer handles final sizing

**Arrow alignment:**
- Arrow block = `blankArrow + "\n" + strings.Join(arrowLines, "\n") + "\n" + blankArrow`
- This creates `innerRows + 2` lines matching box height (top border blank + content rows + bottom border blank)
- `lipgloss.JoinHorizontal(lipgloss.Top, ...)` aligns all elements at their tops

**Gotchas:**
- `╭─ APP` does NOT exist as a literal string in View() output — ANSI codes separate `╭─ ` (styled border) from `APP` (bold title). Tests must check for `╭` and `APP` on the same line, not the combined string.
- `gofmt` needed after implementation — detected by `make fmt-check`
- `statusStripHeight := 1` is not "unused" — it's used in `boxAreaHeight := p.height - statusStripHeight - 1` on the next line

**Testing notes:**
- 61 total tests in panes package, 0 failures
- Coverage: 89.9% for `internal/ui/panes`, 86.4% overall
- Internal tests in `requestflow_boxed_test.go` test all unexported helpers directly
- External tests in `requestflow_pane_test.go` test View() behavior end-to-end
