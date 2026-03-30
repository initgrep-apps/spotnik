---
title: "Fix Table Alignment"
feature: 12-layout
status: done
---

## Background
All table panes render data and headers right-aligned instead of left-aligned. In internal/ui/components/table.go, the rebuild() method creates columns via btable.NewFlexColumn().WithStyle(...) with no alignment specified. The bubble-table library does not default to left alignment. The header style has the same problem. All 7 table-based panes are affected (Playlists, Albums, Liked Songs, Queue, Recently Played, Top Tracks, Top Artists).

## Design
Add `Align(lipgloss.Left)` to both the column style and header style in `rebuild()`. This is a single-file change that fixes all panes globally.

```go
// Column style:
btCols[i] = btable.NewFlexColumn(col.Key, col.Header, col.FlexFactor).
    WithStyle(lipgloss.NewStyle().Foreground(col.Color).Align(lipgloss.Left))

// Header style:
HeaderStyle(lipgloss.NewStyle().Foreground(th.TableHeader()).Bold(false).Align(lipgloss.Left))
```

## Acceptance Criteria
- [ ] All table pane columns render text left-aligned
- [ ] All table pane headers render left-aligned
- [ ] No individual pane code changes needed
- [ ] `make ci` passes

## Tasks
- [ ] Add Align(lipgloss.Left) to column and header styles in internal/ui/components/table.go rebuild()
      - test: table columns left-aligned; headers left-aligned; all panes fixed globally
