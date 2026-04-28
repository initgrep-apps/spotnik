---
name: project_spotnik_story182_complete
description: Story 182 (GatewayLive multi-column, drop ListRow): richRows API, two-column refactor, ANSI regression test patterns
type: project
---

## Story 182 — GatewayLive Multi-Column Table (drop uikit.ListRow)

**What was built:**
- `components.Table.SetRichRows([]map[string]any)`: new method accepting mixed plain string / btable.StyledCell per cell; separate `richRows` field from `rows` with mutual exclusion (SetRows clears richRows, SetRichRows clears rows)
- `applyRows()` extended with a rich-rows path: when `richRows != nil`, iterates richRows, assigns each value directly to `btable.RowData{}` (bubble-table accepts both string and StyledCell)
- `GatewayLivePane` refactored to two-column layout: `{Key:"glyph", FlexFactor:1, Color:th.TextPrimary()}` + `{Key:"event", FlexFactor:30, Color:th.ColumnPrimary()}` — no header, no uikit.ListRow anywhere
- `buildTableRows()` now produces `[]map[string]any` with `btable.NewStyledCell(glyph, lipgloss.NewStyle().Foreground(ColourFor(intent, theme)))` for glyph and plain `row.label` string for event

**Key files:**
- `internal/ui/components/table.go` — `richRows []map[string]any` field, two-path `applyRows()`, `SetRichRows()`, `SetRows` clears richRows
- `internal/ui/panes/gateway_live_pane.go` — two-column constructor + SetTheme + buildTableRows; imports lipgloss + btable
- `internal/ui/components/table_test.go` — 3 new tests; per-test TrueColor pattern (not TestMain) to avoid breaking InfoBox centering tests
- `internal/ui/panes/gateway_live_pane_test.go` — `TestGatewayLivePane_View_NoEmbeddedANSIInLabels`, `TestGatewayLivePane_View_ColumnAlignmentMatchesNetworkLog`, `stripANSI()` helper

**Gotchas:**
- **TestMain TrueColor breaks InfoBox centering**: Setting `lipgloss.SetColorProfile(termenv.TrueColor)` in TestMain breaks `TestInfoBox_Render_ContentVerticallycentered` because InfoBox centering math relies on no-color lipgloss width measurements. Fix: use per-test TrueColor with `t.Cleanup(func() { lipgloss.SetColorProfile(prev) })`.
- **ANSI regression test — "no reset in cell" is not directly achievable**: bubble-table appends `\x1b[0m` at the END of every cell. The test must search within the VISIBLE CHARACTERS of the label (from label start to end of last word) — not from label start to end of line. Pattern:
  ```go
  labelStart := strings.Index(view, labelFragment)
  suffixIdx := strings.Index(view[labelStart:], "  allowed")
  labelEnd := labelStart + suffixIdx + len("  allowed")
  labelSpan := view[labelStart:labelEnd]
  if strings.Contains(labelSpan, "\x1b[0m") { t.Errorf(...) }
  ```
  The old ListRow path had resets BETWEEN words (glyph/timestamp/method/path segments), so this test fails on that path and passes on the new two-column plain-string path.
- **Column-count / key test is insufficient alone**: A test that only checks `cols[0].Key == "glyph"` doesn't pin the regression — it passes even if you port the structure but keep ListRow.Render() inside the cell. The ANSI-in-label test is what actually pins the regression.
- **FlexFactor 1:30 not 1:1**: The glyph column needs FlexFactor 1 and event needs FlexFactor 30 — matching the spec. Using 1:1 would allocate half the width to a single-rune glyph.
- **`Rows()` return contract preserved**: `Rows() []map[string]string` returns only the plain rows, not richRows. This is intentional — callers like RebuildTableTheme only need string rows. RichRows callers must re-supply data via SetRichRows after theme rebuild (same as the existing SetRows pattern).

**Testing notes:**
- Coverage: 88.7% total, all packages above 80% threshold
- The `stripANSI()` helper in gateway_live_pane_test.go is a local implementation (not imported) — consistent with the pattern in other pane test files
- Per-test TrueColor pattern (save prev + t.Cleanup restore) first appeared in internal/uikit/list_row_test.go — use that as the canonical reference
