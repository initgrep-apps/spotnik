---
title: "ListRow + LockedRow — migrate theme, profile, playlists rows"
feature: 13-tui-design-system
status: done
---

## Background

Two row-level primitives ship together because `LockedRow` is a dim variant of
`ListRow`. `ListRow` is a single-line item with an optional leading glyph, a
primary label, and an optional muted trailing caption. `LockedRow` renders the
entire row in `Muted` with a leading `◌` glyph to signal a read-only /
inaccessible state — fills a gap flagged during brainstorming (previously no
primitive covered read-only rows).

Migrates:
- `internal/ui/panes/themes.go` → theme rows use `ListRow` (active theme =
  `GlyphActive` + `RoleAccent`; others = `GlyphAvailable` + `RoleMuted`)
- `internal/ui/panes/profile.go` → logout/forget action rows use `ListRow`
- `internal/ui/panes/playlists_pane.go` → Spotify-owned playlists (read-only)
  render via `LockedRow` (detection: `playlist.Owner.ID == "spotify"`)

**Depends on:** S1. Design record §5.3 (locked glyph `◌`), §6.2
(ListRow/LockedRow role map), §7.1 rows 5–6. Full step-by-step: Task 9 (S9) in
`docs/superpowers/plans/2026-04-24-tui-design-system.md`.

## Design

### Structs

```go
type ListRow struct {
    Glyph   GlyphRole // empty => no glyph
    Label   string
    Caption string
    Intent  Role
    Theme   theme.Theme
}

type LockedRow struct {
    Label string
    Theme theme.Theme
}
```

Both `Render(width int) string`. `ListRow.Render` composes glyph + label +
caption, padding/truncating to width. `LockedRow.Render` prepends `◌` and
colours the entire row in `Muted`.

### Row utilities

`joinSpace(s ...string)` + `padOrTruncate(s, w)` + `pad(n)` — small helpers
defined once in `list_row.go` and reused by other row-level primitives.

### Roles

| Field | Role |
|---|---|
| ListRow.Glyph | matches row intent |
| ListRow.Label | Plain |
| ListRow.Caption | Muted |
| LockedRow.Glyph (`◌`) | Muted |
| LockedRow.Label | Muted (entire row dim) |

## Acceptance Criteria

- [ ] `internal/uikit/list_row.go` defines `ListRow` + `LockedRow` + row utilities
- [ ] `list_row_test.go` covers:
      - `TestListRow_Unicode_WithGlyphAndCaption` — output contains `◉ Monokai`
        and `active`
      - `TestLockedRow_Unicode_DimGlyph` — output contains `◌` and the label
- [ ] `internal/ui/panes/themes.go` theme rows use `ListRow`
- [ ] `internal/ui/panes/profile.go` logout/forget rows use `ListRow`
- [ ] `internal/ui/panes/playlists_pane.go` uses `LockedRow` when
      `playlist.Owner.ID == "spotify"`
- [ ] `make ci` → PASS

## Tasks

Step-by-step: Task 9 (S9) in plan.

- [ ] Branch: `feat/13-uikit-list-row-locked`
- [ ] Write failing `list_row_test.go` (Step 9.1)
- [ ] Implement `list_row.go` with both primitives + row utilities (Step 9.2)
- [ ] Migrate `panes/themes.go` theme rows (Step 9.3)
- [ ] Migrate `panes/profile.go` action rows (Step 9.3)
- [ ] Migrate `panes/playlists_pane.go` to `LockedRow` for Spotify-owned
      playlists (Step 9.3)
- [ ] `make ci` → PASS
- [ ] Commit + push + open PR (Step 9.4)
