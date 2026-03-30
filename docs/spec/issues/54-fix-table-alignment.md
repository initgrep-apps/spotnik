# Feature 54 — Fix Table Alignment

> **Bug fix:** All table panes render data and headers right-aligned instead of left-aligned.

## Root Cause

In `internal/ui/components/table.go`, the `rebuild()` method creates columns via
`btable.NewFlexColumn().WithStyle(lipgloss.NewStyle().Foreground(col.Color))` — no alignment
is specified. The bubble-table library (evertras/bubble-table) does not default to left
alignment, so text renders right-aligned within each column width.

The header style at line 94 has the same problem:
`HeaderStyle(lipgloss.NewStyle().Foreground(th.TableHeader()).Bold(false))` — no `Align`.

**Affected panes:** All 7 table-based panes (Playlists, Albums, Liked Songs, Queue,
Recently Played, Top Tracks, Top Artists). None of them set alignment individually —
they all inherit from the shared `Table` component.

---

## Fix

Add `Align(lipgloss.Left)` to both the column style and header style in `rebuild()`.
This is a single-file change that fixes all panes globally.

### `internal/ui/components/table.go`

**Column style (line 88-89):**
```go
// Before:
btCols[i] = btable.NewFlexColumn(col.Key, col.Header, col.FlexFactor).
    WithStyle(lipgloss.NewStyle().Foreground(col.Color))

// After:
btCols[i] = btable.NewFlexColumn(col.Key, col.Header, col.FlexFactor).
    WithStyle(lipgloss.NewStyle().Foreground(col.Color).Align(lipgloss.Left))
```

**Header style (line 94):**
```go
// Before:
HeaderStyle(lipgloss.NewStyle().Foreground(th.TableHeader()).Bold(false))

// After:
HeaderStyle(lipgloss.NewStyle().Foreground(th.TableHeader()).Bold(false).Align(lipgloss.Left))
```

---

## Files

- `internal/ui/components/table.go` — Add `Align(lipgloss.Left)` to column + header styles in `rebuild()`

---

## Acceptance Criteria

- [ ] All table pane columns render text left-aligned
- [ ] All table pane headers render left-aligned
- [ ] No individual pane code changes needed
- [ ] `make ci` passes
