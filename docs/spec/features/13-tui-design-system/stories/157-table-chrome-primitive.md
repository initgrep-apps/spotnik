---
title: "TableChrome — thin wrapper over components/table.go"
feature: 13-tui-design-system
status: done
---

## Background

`TableChrome` wraps `internal/ui/components/table.go`. The primitive's job is to
**standardise construction** — column tokens (`ColumnIndex`, `ColumnPrimary`,
`ColumnSecondary`, `ColumnTertiary`), header colour, playing-indicator colour —
so that panes no longer build `components.TableConfig` literals inline.

**Scope note:** this story introduces the primitive but does **not** migrate any
call site. Panes continue to call `components.NewTable` directly; migrations can
opt in story-by-story once the wrapper is available. This matches the design-record
note that S8 is a no-op at call sites.

**Depends on:** S3 (we need the design-system vocabulary settled first). Design
record §7.1 row 4. Full step-by-step: Task 8 (S8) in
`docs/superpowers/plans/2026-04-24-tui-design-system.md`.

## Design

### Struct

```go
type TableChrome struct {
    Columns []components.ColumnDef
    Theme   theme.Theme
    inner   *components.Table
}

func (t *TableChrome) Inner() *components.Table {
    if t.inner == nil {
        t.inner = components.NewTable(components.TableConfig{
            Columns: t.Columns, Theme: t.Theme,
            PlayingIndex: -1, ShowHeader: true,
        })
    }
    return t.inner
}
```

Lazy construction — the wrapped table is created on first `Inner()` call. The
inner `*components.Table` owns all interactive state (scroll position, selection,
etc.). `TableChrome` is effectively stateless from the caller's perspective.

### Roles

| Field | Role |
|---|---|
| TableChrome.Header | `theme.TableHeader()` |
| Cell.Index | Column-Index |
| Cell.Primary | Column-Primary |
| Cell.Secondary | Column-Secondary |
| Cell.Tertiary | Column-Tertiary |
| Cell (selected) | Selection |
| Cell.PlayingIndicator | `theme.PlayingIndicator()` |

## Acceptance Criteria

- [ ] `internal/uikit/table_chrome.go` defines `TableChrome` with `Columns`,
      `Theme`, lazy `Inner()`
- [ ] `table_chrome_test.go` covers:
      - `TestTableChrome_WrapsComponentsTable` — `Inner()` returns a non-nil
        `*components.Table`
- [ ] No call-site migration in this story
- [ ] Uikit coverage remains 100%
- [ ] `make ci` → PASS

## Tasks

Step-by-step: Task 8 (S8) in plan.

- [ ] Branch: `feat/13-uikit-table-chrome`
- [ ] Write failing `table_chrome_test.go` (Step 8.1)
- [ ] Implement `table_chrome.go` with lazy `Inner()` (Step 8.2)
- [ ] Run tests → PASS
- [ ] `make ci` → PASS
- [ ] Commit + push + open PR (Step 8.3)
