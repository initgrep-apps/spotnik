---
title: "Dead Pane Actions Removal"
feature: 24-controls-cleanup
status: done
---

## Background

Several pane borders and the help overlay advertise keybindings that are either
unimplemented stubs, explicit no-ops, or broken due to global key interception.
All confirmed from code analysis:

| Pane | Key | Problem |
|------|-----|---------|
| Playlists | `n` | `TODO(feature-53)` stub; intercepted globally (routing.go:36) — pane never sees it |
| Playlists | `r` | Same: `TODO(feature-53)` stub; `"r"` is in `isPlaybackKey`, pane never sees it |
| Playlists (track view) | `Shift+↑/↓` | Most terminals don't deliver xterm shift-arrow sequences; effectively never fires |
| Queue | `A` | No `A` handler exists anywhere in `queue.go` |
| LikedSongs | `i` | Handler always emits `LikeTrackRequestMsg{Unlike: true}` but returns 403; feature removed |
| Global (help overlay) | `x` | `playlists_pane.go` has explicit no-op: `// NOTE: 'x' is out of scope` |

After Story 118 removes `"n"` from `isPlaybackKey`, panes will receive `n` again —
but `n` (new playlist) and `r` (rename) are confirmed stubs and should be removed rather
than left as dead handlers.

**Note on `r` still being a global playback key** (Story 118 removed `n` not `r`):
`r` is `ActionCycleRepeat` in `isPlaybackKey` and will remain so after Story 118. The
Playlists pane's `r` rename handler is therefore intercepted globally regardless. Removing
the handler and the Actions entry is the correct fix.

**Depends on:** Story 118 (removes `"n"` from global interception, allowing the Playlists
`n` handler to be reached — we remove it here instead).

## Design

### playlists_pane.go

**`Actions()` list view branch** (lines 137–141): remove `n` and `r` entries:
```go
return []layout.Action{
    {Key: "f", Label: "filter"},
}
```

**`handleListViewKey`** (lines 310–326): remove the `"n"` and `"r"` key cases entirely.
The `TODO(feature-53)` comments go with them.

**`handleTrackViewKey`** (lines 360–393): remove the `tea.KeyShiftUp` and
`tea.KeyShiftDown` cases. The `// NOTE: management operations (x, Shift+↑/↓) are out of
scope` comment on those cases goes with them.

### queue.go

**`Actions()` default branch** (lines 76–79): remove `{Key: "A", Label: "add"}`:
```go
return []layout.Action{
    {Key: "f", Label: "filter"},
}
```

No handler to remove — `A` has no case in `queue.go Update()`.

### likedsongs_pane.go

**`Actions()` default branch** (lines 76–79): remove `{Key: "i", Label: "like"}`:
```go
return []layout.Action{
    {Key: "f", Label: "filter"},
}
```

**`handleKey`** (line ~151): remove the `"i"` case entirely:
```go
// Remove: case key.Type == tea.KeyRunes && string(key.Runes) == "i":
//     return l, func() tea.Msg { return LikeTrackRequestMsg{Unlike: true} }
```

### Keybinding docs — all three locations, same commit

**`docs/keybinding.md`** Pane Actions section: remove these rows:
- `n | new playlist | Playlists pane` (or however it is listed)
- `r | rename playlist | Playlists pane`
- `A | Add to queue | Search overlay, list panes` — remove the "list panes" claim; clarify: `A` only works in the Search overlay (as `Ctrl+A`); remove the row entirely from Pane Actions since `Ctrl+A` is already documented in the Search Overlay section
- `i | Like / unlike track | LikedSongs pane`
- `x | Remove track from playlist | Playlists pane track sub-view`
- `Shift+↑ / Shift+↓ | Reorder track | Playlists pane track sub-view`

**`docs/DESIGN.md §17`**: remove same entries.

**`internal/ui/panes/help_overlay.go` `helpContent`** Pane Actions section:
Remove from the bindings slice:
- `{"A", "add to queue"}`
- `{"i", "like / unlike"}`
- `{"x", "remove track"}`
- `{"Shift+↑/↓", "reorder (playlists)"}`

Playlists `n` and `r` were never in the help overlay's Pane Actions (they only appear in
the pane border via `Actions()`) — no help overlay change for those.

## Acceptance Criteria

- [ ] Pressing `n` in the Playlists pane does nothing (handler removed)
- [ ] Pressing `r` in the Playlists pane does nothing (handler removed; `r` still works as repeat globally)
- [ ] Playlists pane border shows only `f: filter` (no `n`, no `r`)
- [ ] Pressing `Shift+↑/↓` in the Playlists track view does nothing
- [ ] Queue pane border shows only `f: filter` (no `A`)
- [ ] Pressing `i` in the LikedSongs pane does nothing (handler removed)
- [ ] LikedSongs pane border shows only `f: filter` (no `i`)
- [ ] Help overlay Pane Actions does not list `A`, `i`, `x`, or `Shift+↑/↓`
- [ ] `docs/keybinding.md`, `docs/DESIGN.md §17`, and `help_overlay.go` are updated in the same commit
- [ ] `make ci` passes

## Tasks

- [ ] Add `TestPlaylistsPane_NKey_NoOp` to `internal/ui/panes/playlists_pane_test.go` — verify `n` returns nil cmd after handler removal
  - test: `go test ./internal/ui/panes/... -run TestPlaylistsPane_NKey_NoOp -v` → FAIL
- [ ] Add `TestPlaylistsPane_RKey_NoOp` — verify `r` in list view returns nil cmd
  - test: → FAIL
- [ ] Add `TestPlaylistsPane_Actions_ListView_NoNOrR` — verify Actions() list-view branch omits n and r
  - test: → FAIL
- [ ] Update `playlists_pane.go`: remove `n` and `r` cases from `handleListViewKey`; remove from `Actions()`
  - test: all three playlists tests → PASS
- [ ] Remove or update any existing playlists_pane tests that assert on n/r handler behaviour
- [ ] Add `TestPlaylistsPane_ShiftArrows_TrackView_NoOp` — verify ShiftUp/ShiftDown in track view return nil cmd
  - test: → FAIL
- [ ] Remove `tea.KeyShiftUp` and `tea.KeyShiftDown` cases from `handleTrackViewKey`
  - test: shift-arrow test → PASS; all playlists tests → PASS
- [ ] Add `TestQueuePane_Actions_NoAddEntry` — verify `A` is not in queue Actions()
  - test: `go test ./internal/ui/panes/... -run TestQueuePane_Actions_NoAddEntry -v` → FAIL
- [ ] Remove `{Key: "A", Label: "add"}` from `queue.go Actions()`
  - test: queue actions test → PASS
- [ ] Add `TestLikedSongsPane_IKey_NoOp` — verify `i` returns nil cmd
  - test: → FAIL
- [ ] Add `TestLikedSongsPane_Actions_NoLikeEntry` — verify `i` is not in Actions()
  - test: → FAIL
- [ ] Remove `i` handler and Actions entry from `likedsongs_pane.go`
  - test: both likedsongs tests → PASS
- [ ] Update `docs/keybinding.md`, `docs/DESIGN.md §17`, and `help_overlay.go helpContent` — all in a single commit
  - test: `go build ./...` clean; grep confirms `A`, `i`, `x`, `Shift+↑/↓` no longer appear in Pane Actions sections of all three files
- [ ] `make ci` passes
