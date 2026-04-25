---
title: "Docs rewrite — DESIGN.md, TUI-DESIGN-SYSTEM.md, PANE-TEMPLATE.md, CLAUDE.md"
feature: 13-tui-design-system
status: done
---

## Background

Final story in the feature. Doc-only, no code changes. Creates the canonical
reference `docs/TUI-DESIGN-SYSTEM.md`, strips primitive-rendering detail from
`docs/DESIGN.md` (retaining layout/grid/page mechanics), updates
`docs/PANE-TEMPLATE.md` Step 2 scaffold to use `uikit.PaneChrome`, and adds the
Reading-Order + "What Agents Must NEVER Do" entries to `CLAUDE.md`.

By this point every primitive's canonical contract has been written in its own
story; S19 consolidates them into the reference doc.

**Depends on:** S1–S18. Design record §9 (relationship to other docs). Full
step-by-step: Task 19 (S19) in
`docs/superpowers/plans/2026-04-24-tui-design-system.md`.

## Design

### Create `docs/TUI-DESIGN-SYSTEM.md`

Derived from the design record but pitched as operational reference
(imperative, no decision-record framing). Sections:

1. Purpose
2. Hard rules (do / don't list — mirrors §10 of the design record)
3. Primitive catalogue — full 6-block contract (Purpose, Fields, Rendering
   unicode + ascii, Roles, Glyphs, Lifecycle, Tests) for all 18 primitives
4. Glyph catalogue (from §5 of design record, verbatim)
5. Role / colour matrix (from §6, verbatim)
6. Feedback channels (six surfaces: Toast, StatusGlyph, EmptyState, KeyBar,
   StatusBar, PaneChrome filter preamble — each with a single reason to exist)
7. Relationship to other docs (peer of `CLI-OUTPUT.md`; narrower
   `DESIGN.md`; `PANE-TEMPLATE.md` scaffold)

### Strip `docs/DESIGN.md`

Remove:
- §5 "Embedded Shortcut Borders (btop-style)" — border anatomy moves to
  `TUI-DESIGN-SYSTEM.md` under `PaneChrome`
- `ᐅ` Unicode note at line 609 — glyph is banned
- Any prescriptive rendering detail for overlays, toasts, header, status bar
  — replace each deleted section with `See docs/TUI-DESIGN-SYSTEM.md §N`

Retain:
- §1 Overview
- §2 Pane Definitions
- §3 Layout Grid System
- §4 Pages, Pane Toggling, Preset Layouts
- §6 Content Containment
- §17 Keybindings (cross-referenced to `keybinding.md`)
- §21 Min-terminal-size rule
- Visualizer spec

Add a new §0 authority paragraph clarifying the split: layout mechanics in
`DESIGN.md`; primitive rendering in `TUI-DESIGN-SYSTEM.md`.

### Update `docs/PANE-TEMPLATE.md`

Rewrite Step 2 scaffold to use `uikit.PaneChrome`:

```go
func (p *ListenCountPane) View() string {
    if p.width == 0 || p.height == 0 {
        return ""
    }
    content := "  " + strconv.Itoa(p.count) + " listens"
    return uikit.PaneChrome{
        Width: p.width, Height: p.height,
        Title: p.Title(), ToggleKey: p.ToggleKey(),
        Actions: p.Actions(),
        AccentColor: layout.PaneBorderColor(p.ID(), p.theme),
        Focused: p.focused, Theme: p.theme,
    }.Render(content)
}
```

Update the Verification section accordingly.

### Update `CLAUDE.md`

Add to Reading Order:

> When writing or modifying TUI primitives, consult `docs/TUI-DESIGN-SYSTEM.md` —
> the canonical reference for primitives, glyph catalogue, and role matrix.

Add to "What Agents Must NEVER Do":

> 17. Add a new primitive, glyph, or role to `internal/uikit` without updating
>     `docs/TUI-DESIGN-SYSTEM.md` in the same commit.

### Update `docs/spec/00-overview.md`

Mark feature 13 `done` (will happen automatically via feature-implementer's
completion update, but listed here for completeness).

## Acceptance Criteria

- [ ] `docs/TUI-DESIGN-SYSTEM.md` exists with all 7 sections; all 18 primitives
      documented with the 6-block contract template
- [ ] `docs/DESIGN.md` no longer contains §5 "Embedded Shortcut Borders" or any
      primitive-rendering detail; each stripped section is replaced with a
      pointer to `TUI-DESIGN-SYSTEM.md`
- [ ] `docs/DESIGN.md` retains §1, §2, §3, §4, §6, §17, §21, visualizer spec
- [ ] `docs/DESIGN.md` has a new §0 authority paragraph
- [ ] `docs/PANE-TEMPLATE.md` Step 2 scaffold uses `uikit.PaneChrome`
- [ ] `CLAUDE.md` Reading Order references `docs/TUI-DESIGN-SYSTEM.md`
- [ ] `CLAUDE.md` "What Agents Must NEVER Do" has the primitive/glyph/role rule
- [ ] `grep -rn "ᐅ" docs/` → no matches (except in the design record's banned-glyph
      table, which quotes it for reference)
- [ ] `grep -rn "⚠" docs/` → no matches outside design record + plan
- [ ] `docs/spec/00-overview.md` feature 13 status updated to `done`
- [ ] `make ci` → PASS (doc-only changes should not affect lint/tests)

## Tasks

Step-by-step: Task 19 (S19) in plan.

- [ ] Branch: `feat/13-design-system-docs`
- [ ] Create `docs/TUI-DESIGN-SYSTEM.md` with 7 sections + 18 primitive
      contracts (Step 19.1)
- [ ] Strip `docs/DESIGN.md` primitive sections; add §0 authority paragraph;
      retain layout-centric sections (Step 19.2)
- [ ] Rewrite `docs/PANE-TEMPLATE.md` Step 2 scaffold (Step 19.3)
- [ ] Add Reading-Order + NEVER-Do entries to `CLAUDE.md`
- [ ] Run `grep -rn "ᐅ" docs/` and `grep -rn "⚠" docs/` — verify no unexpected
      matches (Step 19.4)
- [ ] Update feature 13 row in `docs/spec/00-overview.md` to `status: done`
- [ ] `make ci` → PASS
- [ ] Commit + push + open PR (Step 19.5)
