---
title: "EmptyState — centered no-data message with optional hint"
feature: 13-tui-design-system
status: done
---

## Background

`EmptyState` renders a centered `Muted` message with an optional hint line
beneath, used when a pane has no content to display. Replaces hand-rolled
"no data" messages in the queue pane and search results pane. Loading playlist
tracks may also use this primitive in follow-up work (transient empty state
while data streams in).

**Depends on:** S1. Design record §7.1 row 8. Full step-by-step: Task 11 (S11)
in `docs/superpowers/plans/2026-04-24-tui-design-system.md`.

## Design

### Struct

```go
type EmptyState struct {
    Text   string
    Hint   string
    Width  int
    Height int
    Theme  theme.Theme
}
```

`Render() string` returns exactly `Height` lines, centering `Text` (and `Hint`
beneath) both horizontally and vertically.

### Roles

| Field | Role |
|---|---|
| EmptyState.Text | Muted |
| EmptyState.Hint | Muted |

### Call-site migration

```go
// internal/ui/panes/queue.go
if len(rows) == 0 {
    return uikit.EmptyState{
        Text: "Empty queue",
        Hint: "Press / to search for tracks to add",
        Width: p.width, Height: p.height,
        Theme: p.theme,
    }.Render()
}
```

Similarly for `panes/search.go` when a search yields zero results.

## Acceptance Criteria

- [ ] `internal/uikit/empty_state.go` defines `EmptyState` with `Render() string`
- [ ] `empty_state_test.go` asserts the output is `Height` lines and the text is
      vertically centered (neither first nor last line)
- [ ] `internal/ui/panes/queue.go` empty branch uses `EmptyState`
- [ ] `internal/ui/panes/search.go` empty-results branch uses `EmptyState`
- [ ] `make ci` → PASS

## Tasks

Step-by-step: Task 11 (S11) in plan.

- [ ] Branch: `feat/13-uikit-empty-state`
- [ ] Write failing `empty_state_test.go` (Step 11.1)
- [ ] Implement `empty_state.go` with centering algorithm (Step 11.2)
- [ ] Migrate `panes/queue.go` empty branch (Step 11.3)
- [ ] Migrate `panes/search.go` empty-results branch (Step 11.3)
- [ ] `make ci` → PASS
- [ ] Commit + push + open PR (Step 11.4)
