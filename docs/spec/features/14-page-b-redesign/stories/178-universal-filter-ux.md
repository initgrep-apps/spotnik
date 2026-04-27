---
title: "Fix: Universal filter UX — border label + Esc priority on all filterable panes"
feature: 14-page-b-redesign
status: done
---

## Background

After Story 173 (Esc scroll-reset) was shipped, three filter UX behaviors are only working
on `GatewayLivePane`. All other filterable panes (Queue, LikedSongs, TopTracks, TopArtists,
Albums, Playlists, RecentlyPlayed, NetworkLog) are broken in the same way.

**Root cause 1 — no filter border label.**
`renderGrid()` in `internal/app/render.go` checks `layout.FilterQueryPane` before populating
`cfg.FilterQuery`. Only `GatewayLivePane` implements `ActiveFilterQuery() string`. The other
8 panes never expose their committed query → the border never shows `filtering: "query"`.

**Root cause 2 — Esc clears scroll instead of filter.**
These panes use `components.Filter` for real-time filtering. After the user presses
`f → type → Enter`, `filter.active = false` but `filter.query != ""`. The Esc handler in
every pane currently calls `p.table.GotoTop()` unconditionally, because `filter.IsActive()`
returns false. The committed query is never cleared by Esc.

**Root cause 3 — no `Filter.ClearQuery()` method.**
`components.Filter.Toggle()` deactivates and clears the query together. There is no way to
clear the query of an already-inactive filter without re-activating it. A new `ClearQuery()`
method is needed so panes can clear the committed query on Esc without opening the filter input.

**Expected behavior (matches `GatewayLivePane`):**
1. `f → type → Enter` → filter input closes; rows narrow; border shows `filtering: "query"`.
2. First `Esc` → committed filter query cleared; rows expand back to full list.
3. Second `Esc` (no filter active, no query) → scroll reset to page 1.

---

## Design

### Task 1 — Add `Filter.ClearQuery()` to `components/filter.go`

**File:** `internal/ui/components/filter.go`

Add after `Toggle()`:

```go
// ClearQuery clears the committed filter query without changing active state.
// Called by panes to implement Esc-to-clear when the filter input is closed but
// a committed query is still narrowing results.
func (f *Filter) ClearQuery() {
    f.input.Reset()
    f.query = ""
}
```

Test in `internal/ui/components/filter_test.go` — `TestFilter_ClearQuery_ResetsQueryWithoutDeactivating`:
```go
func TestFilter_ClearQuery_ResetsQueryWithoutDeactivating(t *testing.T) {
    f := components.NewFilter(theme.Load("black"))
    f.Toggle()                              // activate
    f.Update(tea.KeyMsg{Type: tea.KeyEnter}) // commit (Enter preserves query via input value)
    // Manually set query for test isolation
    // The committed state: active=false, query set
    // Use Update to set a value first
    f2 := components.NewFilter(theme.Load("black"))
    f2.Toggle()
    f2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("rock")})
    f2.Update(tea.KeyMsg{Type: tea.KeyEnter}) // commit "rock"
    require.Equal(t, "rock", f2.Query())
    require.False(t, f2.IsActive())

    f2.ClearQuery()
    assert.Equal(t, "", f2.Query(), "query must be cleared")
    assert.False(t, f2.IsActive(), "active state must be unchanged")
}
```

### Task 2 — Implement `FilterQueryPane` on all 8 filterable panes

For each pane add:
1. A compile-time interface check: `var _ layout.FilterQueryPane = &XxxPane{}`
2. An `ActiveFilterQuery() string` method returning `p.filter.Query()`

**Panes and their receiver names:**

| Pane file | Receiver | Method body |
|---|---|---|
| `queue.go` | `q` | `return q.filter.Query()` |
| `likedsongs_pane.go` | `l` | `return l.filter.Query()` |
| `toptracks_pane.go` | `p` | `return p.filter.Query()` |
| `topartists_pane.go` | `a` | `return a.filter.Query()` |
| `albums_pane.go` | `a` | `return a.filter.Query()` |
| `playlists_pane.go` | `p` | `return p.filter.Query()` |
| `recentlyplayed_pane.go` | `r` | `return r.filter.Query()` |
| `networklog_pane.go` | `p` | `return p.filter.Query()` |

Place the compile-time check near the top of each file (below existing `var _ layout.Pane`
or `var _ layout.FilterablePane` checks).

Test per pane — `TestXxxPane_ActiveFilterQuery_ReturnsCommittedQuery`:
```go
func TestQueuePane_ActiveFilterQuery_ReturnsCommittedQuery(t *testing.T) {
    store := state.New()
    store.SetQueue([]domain.Track{{Name: "Rock Track", URI: "spotify:track:1"}})
    q := panes.NewQueuePane(store, theme.Load("black"), false)
    q.SetSize(80, 20)
    q.SetFocused(true)

    assert.Equal(t, "", q.ActiveFilterQuery(), "empty before filter applied")

    q.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})      // open filter
    q.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})      // type "r"
    q.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})
    q.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
    q.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
    q.Update(tea.KeyMsg{Type: tea.KeyEnter})                           // commit

    assert.Equal(t, "rock", q.ActiveFilterQuery())
}
```

Repeat the same pattern for the other 7 panes using their appropriate constructors and
store-seeding approach. The exact store-seed values don't matter — the test only checks that
`ActiveFilterQuery()` reflects the committed query after `f → type → Enter`.

### Task 3 — Fix Esc priority in all 8 panes

**Pattern for simple panes** (Queue, LikedSongs, TopTracks, TopArtists, RecentlyPlayed, NetworkLog):

Current Esc handler (wrong — scroll reset unconditional):
```go
if keyMsg.Type == tea.KeyEscape {
    p.table.GotoTop()
    return p, nil
}
```

Replace with:
```go
if keyMsg.Type == tea.KeyEscape {
    if p.filter.Query() != "" {
        p.filter.ClearQuery()
        p.refreshRows()
        return p, nil
    }
    p.table.GotoTop()
    return p, nil
}
```

**Pattern for Albums pane** (`handleListViewKey()`, uses `switch` case):

Current:
```go
case keyMsg.Type == tea.KeyEscape:
    a.table.GotoTop()
    return a, nil
```

Replace with:
```go
case keyMsg.Type == tea.KeyEscape:
    if a.filter.Query() != "" {
        a.filter.ClearQuery()
        a.refreshRows()
        return a, nil
    }
    a.table.GotoTop()
    return a, nil
```

**Pattern for Playlists pane** (`handleListViewKey()`, uses `switch` case):

Current:
```go
case key.Type == tea.KeyEscape:
    p.table.GotoTop()
    return p, nil
```

Replace with:
```go
case key.Type == tea.KeyEscape:
    if p.filter.Query() != "" {
        p.filter.ClearQuery()
        p.refreshPlaylistRows()
        return p, nil
    }
    p.table.GotoTop()
    return p, nil
```

> Note: Playlists uses `refreshPlaylistRows()` (not `refreshRows()`) for the album list.
> Albums uses `refreshRows()` for the album list (not `refreshTrackRows()`).
> NetworkLog's `refreshRows()` re-applies `completedRequests` through the filter — safe to call on Esc.

Test per pane — `TestXxxPane_Esc_ClearsCommittedFilter`:
```go
func TestQueuePane_Esc_ClearsCommittedFilter(t *testing.T) {
    store := state.New()
    tracks := []domain.Track{
        {Name: "Rock Track", URI: "uri:1"},
        {Name: "Jazz Track", URI: "uri:2"},
    }
    store.SetQueue(tracks)
    q := panes.NewQueuePane(store, theme.Load("black"), false)
    q.SetSize(80, 20)
    q.SetFocused(true)

    // Apply filter: f → "rock" → Enter
    q.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
    for _, r := range "rock" {
        q.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
    }
    q.Update(tea.KeyMsg{Type: tea.KeyEnter})
    require.Equal(t, "rock", q.ActiveFilterQuery(), "filter must be committed")

    // Esc → clears filter
    q.Update(tea.KeyMsg{Type: tea.KeyEscape})
    assert.Equal(t, "", q.ActiveFilterQuery(), "Esc must clear committed filter")
}
```

---

## Acceptance Criteria

- [ ] `Filter.ClearQuery()` method exists and passes `TestFilter_ClearQuery_ResetsQueryWithoutDeactivating`
- [ ] All 8 panes implement `layout.FilterQueryPane` (compile-time check + `ActiveFilterQuery()`)
- [ ] `TestXxxPane_ActiveFilterQuery_ReturnsCommittedQuery` passes for all 8 panes
- [ ] After `f → type → Enter` on any pane, the pane border shows `filtering: "query"` (verified via `ActiveFilterQuery()` returning the committed query, which `renderGrid()` picks up)
- [ ] First `Esc` after a committed filter clears the query and restores all rows (not GotoTop)
- [ ] Second `Esc` (no committed filter) resets scroll to page 1 (existing behaviour preserved)
- [ ] `TestXxxPane_Esc_ClearsCommittedFilter` passes for all 8 panes
- [ ] Existing Esc scroll-reset tests (`_Esc_ResetsScrollToPage1`, `_Esc_ResetsScrollInMainListView`) still pass
- [ ] `make ci` passes

## Tasks

- [ ] Add `Filter.ClearQuery()` to `internal/ui/components/filter.go`
      — test: `TestFilter_ClearQuery_ResetsQueryWithoutDeactivating`
- [ ] Implement `ActiveFilterQuery()` on Queue, LikedSongs, TopTracks, TopArtists, RecentlyPlayed, NetworkLog
      — test: 6 × `_ActiveFilterQuery_ReturnsCommittedQuery`
- [ ] Implement `ActiveFilterQuery()` on Albums and Playlists
      — test: 2 × `_ActiveFilterQuery_ReturnsCommittedQuery`
- [ ] Fix Esc priority on Queue, LikedSongs, TopTracks, TopArtists, RecentlyPlayed, NetworkLog
      — test: 6 × `_Esc_ClearsCommittedFilter`; existing `_Esc_ResetsScrollToPage1` still green
- [ ] Fix Esc priority on Albums and Playlists (`handleListViewKey`)
      — test: 2 × `_Esc_ClearsCommittedFilter`; existing `_Esc_ResetsScrollInMainListView` still green
- [ ] `make ci` passes
