---
title: "SectionLabel — caps label + rule for Page B sub-sections"
feature: 13-tui-design-system
status: open
---

## Background

`SectionLabel` renders a caps label followed by a horizontal rule, used for the
sub-sections in the Request Flow pane (`GATEWAY`, `APP`, `GATEWAY LOG`,
`SPOTIFY`, `AUTO-TRAFFIC`). The label colour inherits the parent pane's accent
token so the sub-section visually belongs to the pane.

Migrates:
- `internal/ui/panes/requestflow_boxed.go` — `GATEWAY`, `APP`, `GATEWAY LOG`,
  `SPOTIFY` labels
- `internal/ui/panes/requestflow_pane.go` — `AUTO-TRAFFIC` label

**Depends on:** S1. Design record §6.2 (SectionLabel role), §7.1 row 7, §7.6
stub. Full step-by-step: Task 10 (S10) in
`docs/superpowers/plans/2026-04-24-tui-design-system.md`.

## Design

### Struct

```go
type SectionLabel struct {
    Label       string
    Width       int
    AccentColor lipgloss.Color // parent pane's border token
    Theme       theme.Theme
}
```

`Render() string` returns two lines: `" <Label> "` (bold, accent colour) + a
horizontal rule of `Width` `─` characters (accent colour). Ascii mode uses `-`.

### Roles

| Field | Role |
|---|---|
| SectionLabel | Parent pane's border token |

## Acceptance Criteria

- [ ] `internal/uikit/section_label.go` defines `SectionLabel` with `Render() string`
- [ ] `section_label_test.go` asserts two-line output: label + rule; rule uses
      `─` in unicode and `-` in ascii
- [ ] `internal/ui/panes/requestflow_boxed.go` uses `SectionLabel` for all four
      sub-section labels
- [ ] `internal/ui/panes/requestflow_pane.go` uses `SectionLabel` for
      `AUTO-TRAFFIC`
- [ ] `make ci` → PASS

## Tasks

Step-by-step: Task 10 (S10) in plan.

- [ ] Branch: `feat/13-uikit-section-label`
- [ ] Write failing `section_label_test.go` (Step 10.1)
- [ ] Implement `section_label.go` (Step 10.2)
- [ ] Migrate `requestflow_boxed.go` call sites (Step 10.3)
- [ ] Migrate `requestflow_pane.go` call site (Step 10.3)
- [ ] `make ci` → PASS
- [ ] Commit + push + open PR (Step 10.4)
