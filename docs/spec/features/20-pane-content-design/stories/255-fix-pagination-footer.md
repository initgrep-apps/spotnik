---
title: "Fix pagination footer positioning — use WithMinimumHeight"
feature: 20-pane-content-design
status: open
---

## Background

Pagination footer ("Page X/Y") was static at bottom-right of pane before feature 20. Now footer position varies — glued after last data row instead of fixed at pane bottom. On pages with fewer rows, footer appears higher up.

**Root cause:** Bubble-table's `calculatePadding` fills blank rows to maintain consistent table height. This requires `WithMinimumHeight` to be set. The Table wrapper never calls `WithMinimumHeight`, so `calculatePadding` returns 0 and footer position depends on data row count. Before feature 20 this was less noticeable because `SetSize` used `height - 6` (fewer rows per page = more pages = less variation).

Additionally, the `focused` field added during feature 20 is unnecessary complexity. `SetFocused` already sets focus directly on `t.inner` via `t.inner.Focused(focused)`. The stored field is only used in `rebuild()` which creates a fresh inner model — but `rebuild()` is only called on threshold crossing (rare), and the pane re-applies focus after rebuild anyway via `SetFocused`.

**Solution:** Call `WithMinimumHeight(t.height)` in both `rebuild()` and `SetSize()` to guarantee consistent table height and fixed pagination footer position. Remove the `focused` field from `Table` struct and `SetFocused` storage.

## Design

### Add `WithMinimumHeight` in `rebuild()`

After `inner.WithPageSize(pageSize)`, add:
```go
inner = inner.WithMinimumHeight(t.height)
```

This ensures `calculatePadding` fills blank rows to match the pane's allocated height.

### Add `WithMinimumHeight` in `SetSize()`

After `t.inner.WithPageSize(pageSize)`, add:
```go
t.inner = t.inner.WithMinimumHeight(height)
```

### Remove `focused` field

Remove the `focused bool` field from `Table` struct. Revert `SetFocused` to its pre-feature-20 form:
```go
func (t *Table) SetFocused(focused bool) {
    t.inner = t.inner.Focused(focused)
}
```

Remove `t.focused = focused` assignment. Remove `t.inner = t.inner.Focused(t.focused)` from `rebuild()`.

After `rebuild()`, the pane re-applies focus via `t.SetFocused(focused)` in the constructor (`NewQueuePane`, etc.), so removing the stored field doesn't cause focus loss.

## Files

### Modify

- `internal/ui/components/table.go` — add `WithMinimumHeight` in `rebuild()` + `SetSize()`, remove `focused` field, simplify `SetFocused`

## Acceptance Criteria

- [ ] Pagination footer renders at fixed bottom position regardless of page data count
- [ ] `rebuild()` calls `WithMinimumHeight(t.height)` after `WithPageSize`
- [ ] `SetSize()` calls `WithMinimumHeight(height)` after `WithPageSize`
- [ ] `focused` field removed from `Table` struct
- [ ] `SetFocused` directly sets `t.inner.Focused(focused)` without storing field
- [ ] No `t.inner = t.inner.Focused(t.focused)` in `rebuild()`
- [ ] `go build ./...` compiles without errors
- [ ] `make ci` passes

## Tasks

- [ ] **Task 1: Add WithMinimumHeight and remove focused field**
  In `internal/ui/components/table.go`:
  - Add `inner = inner.WithMinimumHeight(t.height)` after `inner.WithPageSize(pageSize)` in `rebuild()`
  - Add `t.inner = t.inner.WithMinimumHeight(height)` after `t.inner.WithPageSize(pageSize)` in `SetSize()`
  - Remove `focused bool` from `Table` struct
  - Remove `t.inner = t.inner.Focused(t.focused)` from `rebuild()`
  - Remove `t.focused = focused` from `SetFocused`, keep `t.inner = t.inner.Focused(focused)`
  - test: `go build ./internal/ui/components/` — no errors
  - test: `go test ./internal/ui/components/ -v -run "TestTable"` — all pass

- [ ] **Task 2: Run full test suite**
  - test: `make test` — all pass

- [ ] **Task 3: Commit**
  `fix(ui): fix pagination footer positioning with WithMinimumHeight`

- [ ] **Task 4: Create PR**
  `gh pr create` targeting main. Title: `fix(ui): fix pagination footer positioning with WithMinimumHeight`
