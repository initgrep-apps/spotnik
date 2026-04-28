---
title: "GatewayLivePane multi-column table — drop uikit.ListRow"
feature: 14-page-b-redesign
status: done
---

## Background

`GatewayLivePane` (toggle key 4) currently embeds `*TableBasedPane` with a **single
column** (`Key: "row"`, `Color: ""`) and pre-renders each row's content into a
**styled string** via `uikit.ListRow{...}.Render(width-2)` before stuffing it into
that one cell (`internal/ui/panes/gateway_live_pane.go:307-313`).

Two visual regressions follow from this design:

1. **Selection highlight covers only the leading glyph, not the whole row.** bubble-table
   applies the selected-row background through `WithRowStyleFunc` at
   `internal/ui/components/table.go:103-111`. lipgloss wraps the cell with
   `Background(SelectedBg).Render(...)`, but the cell content already contains
   embedded ANSI from `uikit.ListRow.Render` (each segment ends with `\x1b[0m`,
   which closes the row background mid-cell). Only the first segment shows the
   highlight; the rest of the row reverts to the terminal's default background.

2. **Padding/styling diverges from every other table-based pane.** The other eight
   table panes (`networklog_pane.go`, `likedsongs_pane.go`, `albums_pane.go`,
   `playlists_pane.go`, `queue.go`, `recentlyplayed_pane.go`, `toptracks_pane.go`,
   `topartists_pane.go`) define **multi-column layouts with column-level Color**
   and pass plain strings into cells, so bubble-table handles padding, alignment,
   and row-bg uniformly. GatewayLive uses `Color: ""` and renders its own padding
   inside the cell, producing visibly different spacing.

The ListRow workaround was used because **per-row glyph colour varies by event
kind** (Success, Error, Warning, Info, Muted, Plain) and bubble-table column-level
`Color` is a single value. The proper bubble-table mechanism for per-row styling
is `btable.NewStyledCell(value, fgStyle)` — `bubble-table/table/row.go:103` does
`entry.Style.Copy().Inherit(cellStyle)`, and `Inherit` only fills UNSET fields, so
a foreground-only `StyledCell` correctly preserves the row-level highlight
Background.

The existing `gatewayLiveRow` struct already stores the data in the right shape
(`glyphRole`, `intent`, `label`, `matchString`) — no domain refactor is required.
This story replaces the rendering layer only.

**Source:** post-implementation regression discovered after feature 14 stories
175–181 shipped.

**Depends on:** Stories 175 (GatewayHealthPane/PollingTrafficPane), 176
(GatewayLivePane), 181 (TableBasedPane consolidation, live filter).

---

## Design

### Goal

GatewayLivePane uses a multi-column bubble-table. The glyph column is per-row
coloured via `btable.NewStyledCell`. The event column uses standard column-level
foreground. No `uikit.ListRow` use anywhere in the pane. Selection highlight
covers the full row width. Padding matches NetworkLog.

### Task 1 — Extend `components.Table` to accept rich row data

**Why:** bubble-table's row data API accepts `interface{}` values per key (string
or `StyledCell`). The Spotnik wrapper currently constrains `SetRows` to
`[]map[string]string`. We need to surface the existing per-row styled cell
capability without breaking the eight callers that already use plain strings.

**File to modify:** `internal/ui/components/table.go`.

**Add:**

```go
// SetRichRows updates the table data with rows whose cell values may be either
// plain strings (rendered with the column's foreground colour) or
// btable.StyledCell instances (rendered with a per-cell foreground while still
// inheriting the row-level highlight background). Used by panes that need
// per-row colour variation that single-value column Color cannot express
// (e.g. GatewayLivePane's per-event-kind glyph colours).
//
// Existing SetRows([]map[string]string) callers are unaffected.
func (t *Table) SetRichRows(rows []map[string]any) {
    // Stored separately from t.rows so that Rows() (used by RebuildTableTheme)
    // keeps its existing string-only return contract for callers that do not
    // need rich values.
    t.richRows = rows
    t.rows = nil
    t.applyRows()
}
```

Storage: add a `richRows []map[string]any` field. Update `applyRows()` to handle
both fields — if `richRows != nil`, iterate richRows; otherwise iterate `rows`
as before. Inside the rich loop, assign each value directly to `data[k]`
(bubble-table accepts `string` or `StyledCell` for any cell value).

**Playing-indicator branch:** the existing override at table.go:148-157 only
fires when `len(t.rows) > 0` and replaces the index column. For richRows we
keep the same behaviour — replace the first column key with `NewStyledCell(▶,
PlayingIndicator)`. (GatewayLive sets `PlayingIndex: -1`, so this branch is
inert there, but the symmetry matters for future richRows users.)

**Tests (add to `internal/ui/components/table_test.go`):**

- `TestTable_SetRichRows_PlainStringCellsRender` — pass a `map[string]any`
  with string values for both columns; assert View() renders both column values.
- `TestTable_SetRichRows_StyledCellAppliesForeground` — pass a `StyledCell`
  with `lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000"))`; assert the
  rendered output contains the expected ANSI red SGR (`\x1b[38`).
- `TestTable_SetRichRows_DoesNotAffectExistingSetRows` — call `SetRows` first,
  assert View() renders, then call `SetRichRows`, assert View() now reflects
  rich data and no leftover string row appears.

### Task 2 — Refactor `GatewayLivePane` to multi-column rich-row rendering

**File to modify:** `internal/ui/panes/gateway_live_pane.go`.

**Column layout (no header):**

```go
columns := []components.ColumnDef{
    {Key: "glyph", Header: "", FlexFactor: 1,  Color: th.TextPrimary()},
    {Key: "event", Header: "", FlexFactor: 30, Color: th.ColumnPrimary()},
}
```

`FlexFactor 1 : 30` reserves a narrow glyph column (one Unicode column wide for
the glyph plus bubble-table's column padding) and gives the rest to the event
text. `Color` on the `glyph` column is a fallback only — the per-row
`StyledCell.Style` overrides it. The `event` column uses `th.ColumnPrimary()` to
match every other table-based pane's body text colour.

**`gatewayLiveRow` struct:** unchanged — `glyphRole`, `intent`, `label`,
`matchString` already store the data in the correct shape.

**`buildTableRows()` — replacement body:**

```go
func (p *GatewayLivePane) buildTableRows() {
    if p.width == 0 {
        return
    }
    query := strings.ToLower(p.Filter().Query())
    mode := uikit.ActiveMode()
    rows := make([]map[string]any, 0, len(p.buffer))

    for _, row := range p.buffer {
        if query != "" && !strings.Contains(row.matchString, query) {
            continue
        }
        glyphCell := btable.NewStyledCell(
            uikit.GlyphFor(row.glyphRole, mode),
            lipgloss.NewStyle().Foreground(uikit.ColourFor(row.intent, p.theme)),
        )
        rows = append(rows, map[string]any{
            "glyph": glyphCell,
            "event": row.label,
        })
    }
    p.Table().SetRichRows(rows)
}
```

**Imports to add to `gateway_live_pane.go`:**

```go
"github.com/charmbracelet/lipgloss"
btable "github.com/evertras/bubble-table/table"
```

**Imports to remove from `gateway_live_pane.go`:** none of the existing imports
become unused (uikit is still needed for `GlyphFor`, `ActiveMode`, `ColourFor`,
`GlyphRole`, `Role`).

**Constructor (`NewGatewayLivePane`) and `SetTheme`:** update both column
definitions to the two-column layout above. The rest of construction (Table
config, Filter, base init) is unchanged.

### Task 3 — Update tests for the new column shape

**File to modify:** `internal/ui/panes/gateway_live_pane_test.go`.

Existing tests assert that View() contains substrings such as `"player"`,
`"tracks"`, `"Tokens refilled"`, `"200"`, `"e0509"`, `"e0000"`, `"first"`,
`"last"`. These survive the refactor: the substrings are the row labels and
will appear in the event column. Verify by running the suite.

**Add new tests:**

- `TestGatewayLivePane_View_NoEmbeddedANSIInLabels` — call `SetSize(80, 20)`,
  record one EventRequestAllowed, tick, and assert that the event column
  segment (the longer cell on each row) contains no ANSI reset (`\x1b[0m`)
  embedded mid-string. Acceptable form: scan View() for the row containing
  the path, verify there is no `\x1b[0m` between the start of the label and
  the next newline. (Pin the regression: a future revert to ListRow embeds
  resets and would fail this test.)
- `TestGatewayLivePane_View_ColumnAlignmentMatchesNetworkLog` — render
  GatewayLivePane with three events at the same width as a NetworkLogPane
  also rendered with three rows; assert that the **first column right edge
  position** (the column boundary between glyph and label) is at the same
  terminal column in both. Use `lipgloss.Width` on the visible portion of
  one row to confirm the glyph occupies a single Unicode-column-wide slot
  followed by bubble-table's standard column padding.

If either of the existing Enter-to-apply tests
(`TestGatewayLivePane_CommittedFilter_ClearedByEsc`,
`TestGatewayLivePane_EnterOnEmptyPreservesPriorQuery`) is now broken because
of the live-filter contract from story 181, fix them in the same commit so
the suite passes — story 181 already established that pane uses live filter,
not Enter-to-apply.

### Task 4 — Verify regression suite

Run `make ci` and confirm:
- `go test ./internal/ui/components/... -v` passes (Task 1 tests + existing).
- `go test ./internal/ui/panes/... -v` passes (Task 3 tests + existing).
- `make lint` clean.
- `make test-coverage` ≥ 80%.

---

## Acceptance Criteria

- [ ] `components.Table.SetRichRows([]map[string]any)` exists and accepts both
      plain string and `btable.StyledCell` cell values.
- [ ] `GatewayLivePane` uses a two-column layout (`glyph`, `event`) with no
      header (`ShowHeader: false` retained).
- [ ] `gateway_live_pane.go` no longer imports or calls `uikit.ListRow`.
- [ ] Per-row glyph foreground is supplied via
      `btable.NewStyledCell(uikit.GlyphFor(role, mode), Foreground(uikit.ColourFor(intent, theme)))`.
- [ ] `event` column uses `th.ColumnPrimary()` — matches NetworkLog and the
      seven other table-based panes.
- [ ] No embedded `\x1b[0m` resets inside cell content (verified by new test).
- [ ] Selection highlight (`SelectedBg`) paints the **full** width of the
      cursor row when the pane is focused. (Visual confirmation; pinned by the
      "no embedded ANSI" test which is the structural cause.)
- [ ] All existing GatewayLivePane tests pass unchanged in expected substrings.
- [ ] `make ci` passes (lint + tests + coverage ≥ 80%).

## Tasks

- [ ] Extend `internal/ui/components/table.go` with `SetRichRows` and the
      `richRows` storage path; preserve `SetRows` behaviour for existing callers.
      - test: 3 new tests in `table_test.go` green; existing table tests green.
- [ ] Refactor `internal/ui/panes/gateway_live_pane.go` constructor, `SetTheme`,
      and `buildTableRows` to the multi-column rich-row design.
      - test: `go test ./internal/ui/panes/... -run TestGatewayLive -v` green.
- [ ] Add new tests `TestGatewayLivePane_View_NoEmbeddedANSIInLabels` and
      `TestGatewayLivePane_View_ColumnAlignmentMatchesNetworkLog` in
      `gateway_live_pane_test.go`. Reconcile any Enter-to-apply tests that
      contradict story 181's live-filter contract.
      - test: full panes suite green.
- [ ] Run `make ci`; fix any lint/coverage gaps.
      - test: `make ci` green.
