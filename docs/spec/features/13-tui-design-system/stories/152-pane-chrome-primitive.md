---
title: "PaneChrome primitive — design-system wrapper over layout.RenderPaneBorder"
feature: 13-tui-design-system
status: done
---

## Background

`PaneChrome` is the standard bordered pane primitive. It wraps
`layout.RenderPaneBorder` (which already honours `uikit.ActiveMode` after S2) and
applies the design-system role matrix for title, toggle-key superscript, and
right-side action notches. Every pane in `internal/ui/panes/` will eventually
render through this primitive.

**Depends on:** S1 (uikit scaffold), S2 (border.go knows ascii mode).
Design record §7.3 — full PaneChrome contract with format rules, rendering
snapshots, and role mappings. Full step-by-step: Task 3 (S3) in
`docs/superpowers/plans/2026-04-24-tui-design-system.md`.

## Design

### Struct

```go
type PaneChrome struct {
    Width, Height int
    Title         string
    ToggleKey     int            // 0 = no key shown
    Actions       []layout.Action
    AccentColor   lipgloss.Color // per-pane border token
    Focused       bool
    FilterQuery   string
    Theme         theme.Theme
}
```

### Rendering rules (§7.3)

- Title follows `─ ` with **no trailing space**; dashes flush against it.
- Each action sits in a notch: `╮ <key> <label> ╭`.
- Notches are joined by a single `─`.
- The last notch's `╭` is immediately followed by the top-right corner `╮`,
  producing `╭╮` — intentional.
- Filter mode: muted preamble `filtering: "<query>"` followed by ` ─╮ Esc close ╭`.
- No `ᐅ` anywhere.

### Roles

| Field | Role |
|---|---|
| Border (focused) | PaneBorder-`<ID>` |
| Border (unfocused) | Muted PaneBorder-`<ID>` |
| ToggleKey | Accent |
| Title (focused) | Strong (bold) |
| Title (unfocused) | Plain |
| Action.Key | Accent |
| Action.Label | Muted |
| FilterPreamble label | Muted |
| FilterPreamble query | Accent |

### Implementation

Delegates to `layout.RenderPaneBorder` — see plan Task 3 Step 3.3. The primitive
is stateless; all inputs flow through the struct literal.

No call-site migration in this story — pane migration happens incrementally in
subsequent stories (S7, S19 scaffold update).

## Acceptance Criteria

- [ ] `internal/uikit/pane_chrome.go` defines `PaneChrome` with the fields above
      and a `Render(content string) string` method delegating to
      `layout.RenderPaneBorder`
- [ ] `pane_chrome_test.go` covers:
      - `TestPaneChrome_UnicodeSnapshot_ActionsMode` — title starts with
        `╭─ ³Playlists`; action notches `╮ f filter ╭` present; final `╭╮` present
      - `TestPaneChrome_ASCIISnapshot_ActionsMode` — `+- 3 Playlists` prefix; no
        unicode corners anywhere
      - `TestPaneChrome_FilterMode_NoArrow` — no `ᐅ`; `filtering: "rock"` present;
        `╮ Esc close ╭` present
      - `TestPaneChrome_UnfocusedTitleNotBold` — raw bytes do not contain
        `\x1b[1m`
      - `TestPaneChrome_WidthAndHeightMatch` — line count == Height; every line
        width == Width
- [ ] Uikit package coverage remains 100%
- [ ] `make ci` → PASS

## Tasks

Step-by-step: Task 3 (S3) in plan.

- [ ] Branch: `feat/13-uikit-pane-chrome`
- [ ] Write failing `pane_chrome_test.go` (Step 3.2)
- [ ] Implement `pane_chrome.go` delegating to `layout.RenderPaneBorder` (Step 3.3)
- [ ] Run tests → PASS (Step 3.4)
- [ ] `make ci` → PASS
- [ ] Commit + push + open PR (Step 3.5)
