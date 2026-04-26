---
title: "Universal Esc Scroll Reset"
feature: 14-page-b-redesign
status: open
---

## Background

Every table-based pane in Spotnik supports keyboard scrolling but has no way to jump back
to the top without manually pressing `↑` many times. The redesign establishes a
universal `Esc` behaviour: when no filter is active, `Esc` resets the table scroll
position to page 1. When a filter is active, `Esc` already deactivates it — the new
scroll-reset fires only after the filter is cleared.

Two new methods on `components.Table` underpin this:
- `GotoTop()` — calls `PageFirst()` on the inner `evertras/bubble-table` model
- `CurrentPage()` — delegates to `(&t.inner).CurrentPage()` for test assertions

CLAUDE.md rule 15 requires all three keybinding locations to be updated in the same commit.

**Source:** `docs/superpowers/specs/2026-04-26-page-b-redesign-design.md` §Universal Esc,
`docs/superpowers/plans/2026-04-26-page-b-redesign.md` Tasks 1–4.

**Depends on:** Nothing. Implements on a clean branch from `main`.

---

## Design

### Task 1 — Add GotoTop() and CurrentPage() to components.Table

**Files to modify:** `internal/ui/components/table.go`, `internal/ui/components/table_test.go`

Test: `TestTable_GotoTop_ResetsToFirstPage` — creates a table with 20 rows and page size 5,
scrolls 8 rows down, calls `GotoTop()`, asserts `CurrentPage() == 1`.

Implementation — append after `View()` in `table.go`:

```go
func (t *Table) GotoTop() {
    t.inner = t.inner.PageFirst()
}

func (t *Table) CurrentPage() int {
    return (&t.inner).CurrentPage()
}
```

Note: `PageFirst()` is a value receiver method on the inner model; `CurrentPage()` is a
pointer receiver — hence the `&t.inner` indirection.

### Task 2 — Esc scroll-reset on simple table panes

**Files to modify:** `queue.go`, `toptracks_pane.go`, `topartists_pane.go`,
`likedsongs_pane.go`, `recentlyplayed_pane.go`, `networklog_pane.go` and their test files.

**Pattern:** In each pane's key-routing path, after the filter-active branch exits and
before `table.Update(keyMsg)`, add:

```go
if msg.Type == tea.KeyEscape {
    p.table.GotoTop()
    return p, nil
}
```

Add a `TableCurrentPage() int` white-box test accessor to each pane:

```go
func (p *XxxPane) TableCurrentPage() int { return p.table.CurrentPage() }
```

Test per pane: `TestXxxPane_Esc_ResetsScrollToPage1` — fills store with 20 rows, scrolls
8 rows down via `tea.KeyDown`, asserts `TableCurrentPage() > 1`, sends `tea.KeyEscape`,
asserts `TableCurrentPage() == 1`.

### Task 3 — Esc scroll-reset on Albums and Playlists panes

**Files to modify:** `albums_pane.go`, `albums_pane_test.go`, `playlists_pane.go`,
`playlists_pane_test.go`.

These panes have a track sub-view. Esc in sub-view already closes it (unchanged). The new
Esc fires only in the main list view with no active filter — in `handleListViewKey()` before
the `table.Update(keyMsg)` fallthrough.

Tests: `TestAlbumsPane_Esc_ResetsScrollInMainListView`,
`TestPlaylistsPane_Esc_ResetsScrollInMainListView`.

### Task 4 — Update all three keybinding locations (same commit)

> **CLAUDE.md rule 15:** All three locations must update in the same commit.

**Files to modify:**
- `internal/ui/panes/help_overlay.go` — `helpContent` Navigation section
- `docs/keybinding.md` — Navigation table
- `docs/DESIGN.md` — §17 keybinding table

Add `↑ / k Scroll up` and `↓ / j Scroll down` rows to the Navigation section in all three
files. Update the `Esc` row label to: `"Close overlay · clear filter · scroll top"`.

---

## Acceptance Criteria

- [ ] `Table.GotoTop()` and `Table.CurrentPage()` exist and compile
- [ ] `TestTable_GotoTop_ResetsToFirstPage` passes
- [ ] Esc scroll-reset wired into Queue, TopTracks, TopArtists, LikedSongs, RecentlyPlayed,
      NetworkLog (6 panes) — each with a `TableCurrentPage()` accessor and a passing
      `_Esc_ResetsScrollToPage1` test
- [ ] Esc scroll-reset wired into Albums and Playlists main list views — each with a
      `TableCurrentPage()` accessor and a passing `_Esc_ResetsScrollInMainListView` test
- [ ] `help_overlay.go`, `docs/keybinding.md`, and `docs/DESIGN.md` all updated in one commit
- [ ] `make ci` passes

## Tasks

- [ ] Add `GotoTop()` and `CurrentPage()` to `internal/ui/components/table.go`
      - test: `TestTable_GotoTop_ResetsToFirstPage` passes
- [ ] Wire Esc scroll-reset in Queue, TopTracks, TopArtists, LikedSongs, RecentlyPlayed, NetworkLog
      - test: 6 × `_Esc_ResetsScrollToPage1` tests pass; `go test ./internal/ui/panes/... -v` green
- [ ] Wire Esc scroll-reset in Albums and Playlists main list views
      - test: `TestAlbumsPane_Esc_ResetsScrollInMainListView` and `TestPlaylistsPane_Esc_ResetsScrollInMainListView` pass
- [ ] Update keybindings in `help_overlay.go`, `docs/keybinding.md`, `docs/DESIGN.md` in one commit
      - test: `go build ./...` compiles; content review confirms all three locations updated
