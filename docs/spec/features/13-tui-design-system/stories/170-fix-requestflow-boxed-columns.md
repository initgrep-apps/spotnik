---
title: "Fix: RequestFlow sub-columns — restore bordered boxes via PaneChrome"
feature: 13-tui-design-system
status: open
---

## Background

The RequestFlow pane (`viewBoxed` layout) renders three side-by-side columns — APP,
GATEWAY LOG, SPOTIFY — each with a titled border. These were originally produced by
`renderSubBox`, a hand-rolled helper that drew `╭─ LABEL ──╮ / │ content │ / ╰───╯`
boxes per column.

Story 159 (SectionLabel migration, commit `f239670`) replaced `renderSubBox` with
`renderSectionColumn`, which uses `uikit.SectionLabel` — a 2-line header primitive
(bold label + horizontal rule). `SectionLabel` is correct for the full-width
GATEWAY banner and AUTO-TRAFFIC strip (1-row content strips with a labelled header),
but the three column areas need full bordered boxes that visually separate them.
After the migration the three columns appear as unseparated line blocks with no
vertical borders.

**Root cause:** Story 159 acceptance criteria said "Page B sub-section labels go
through `uikit.SectionLabel`" — this was over-applied to include the column areas,
which need `PaneChrome` (a fully bordered container), not a header-only primitive.

**Files:** `internal/ui/panes/requestflow_pane.go`,
`internal/ui/panes/requestflow_boxed.go`

## Design

### `renderSectionColumn` — replace internals, keep signature

The function signature is unchanged — callers in `viewBoxed()` need no modification.
Swap the `SectionLabel` header + raw lines for a `PaneChrome` call:

```go
func renderSectionColumn(label string, lines []string, width int, accent lipgloss.Color, th theme.Theme) string {
    innerW := width - 2
    var sb strings.Builder
    for i, line := range lines {
        if i > 0 {
            sb.WriteString("\n")
        }
        sb.WriteString(layout.TruncateOrPad(line, innerW))
    }
    return uikit.PaneChrome{
        Width:       width,
        Height:      len(lines) + 2, // +2 for top/bottom border rows
        Title:       label,
        AccentColor: accent,
        Focused:     false,
        Theme:       th,
    }.Render(sb.String())
}
```

### Height accounting — no changes needed

`viewBoxed()` already computes:

```go
innerRows := boxAreaHeight - 2 // subtract top/bottom border of column boxes
```

`len(lines) + 2 = innerRows + 2 = boxAreaHeight` — the total column height equals
`boxAreaHeight`, which is exactly the space allocated. No arithmetic changes in
`viewBoxed()`.

### GATEWAY banner and AUTO-TRAFFIC strip

These two strips correctly use `uikit.SectionLabel` + single content row. No changes
needed for either.

## Acceptance Criteria

- [ ] `renderSectionColumn` in `requestflow_pane.go` uses `uikit.PaneChrome` instead
      of `uikit.SectionLabel`
- [ ] APP, GATEWAY LOG, and SPOTIFY columns each render with a rounded bordered box
      (`╭─ APP ──╮ / │ lines │ / ╰─────╯`) matching the accent colour for that column
- [ ] GATEWAY banner and AUTO-TRAFFIC strip still use `uikit.SectionLabel` (no change)
- [ ] `viewBoxed()` call sites for `renderSectionColumn` require no argument changes
- [ ] `requestflow_pane_test.go` and `requestflow_boxed_test.go` updated — assertions
      now check for bordered output (top border line contains `╭`, content lines
      contain `│`, bottom border contains `╰`)
- [ ] `make ci` → PASS

## Tasks

- [ ] Branch: `fix/13-requestflow-boxed-columns`
- [ ] Update `renderSectionColumn` in `requestflow_pane.go` to use `PaneChrome`
- [ ] Update test assertions in `requestflow_pane_test.go` and
      `requestflow_boxed_test.go` for bordered output
- [ ] `make ci` → PASS
- [ ] Commit + push + open PR
