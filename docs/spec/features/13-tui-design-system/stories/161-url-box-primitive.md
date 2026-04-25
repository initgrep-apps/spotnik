---
title: "URLBox — muted-border block wrapping URL/code content"
feature: 13-tui-design-system
status: done
---

## Background

`URLBox` renders a URL (or short code snippet) inside a muted-border rounded
rectangle with the content coloured in `Accent`. URLs longer than the box width
are broken at `&` boundaries (preferred — matches the current `wrapURL` helper),
falling back to hard-wrapping when no ampersand is available in the first half
of the line.

Primitive is introduced here; the onboarding call site migrates in S18 as part
of the end-to-end onboarding rewrite.

**Depends on:** S1. Design record §7.1 row 9, §7.6 URLBox stub. Full step-by-step:
Task 12 (S12) in `docs/superpowers/plans/2026-04-24-tui-design-system.md`.

## Design

### Struct

```go
type URLBox struct {
    URL   string
    Width int
    Theme theme.Theme
}
```

`Render() string` returns the URL inside a `lipgloss.RoundedBorder()` with
`BorderForeground = theme.TextMuted()` and `Foreground = theme.Accent()`.
Padding `(0, 1)`. Long URLs wrap at `&` (preferred) or at the box width.

### `wrapAtAmpersand(u, width)` helper

- If `u` fits in `width`, return as-is.
- Otherwise: scan back from `width` for the last `&` in the first half; cut
  there. Repeat on the remainder.
- No `&` in the first half → hard-wrap at `width`.

### Roles

| Field | Role |
|---|---|
| URLBox.Border | Muted |
| URLBox.Content | Accent |

## Acceptance Criteria

- [ ] `internal/uikit/url_box.go` defines `URLBox` with `Render() string`
- [ ] `url_box_test.go` covers `TestURLBox_WrapsURLAtAmpersand` — multi-line
      output for a long URL; each line fits within `Width + border`
- [ ] No call-site migration in this story (onboarding migration in S18)
- [ ] `make ci` → PASS

## Tasks

Step-by-step: Task 12 (S12) in plan.

- [ ] Branch: `feat/13-uikit-url-box`
- [ ] Write failing `url_box_test.go` (Step 12.1)
- [ ] Implement `url_box.go` + `wrapAtAmpersand` helper (Step 12.2)
- [ ] Run tests → PASS
- [ ] `make ci` → PASS
- [ ] Commit + push + open PR (Step 12.3)
