---
title: "Playback Key Bugs — Space Fix and Remove n"
feature: 24-controls-cleanup
status: done
---

## Background

Two bugs in playback key routing, both confirmed from code:

**1A — Space does nothing.** Bubbletea v0.27 delivers Space as `tea.KeySpace` (a
named key type, `Runes` is empty), not as a rune. Both `routing.go:isPlaybackKey`
and `nowplaying.go:handleKey` check for a space rune — neither matches, so Space
falls through to the focused pane and is silently ignored.

**1B — `n` duplicates `→`.** `nowplaying.go:handleKey` has:
```go
case msg.Type == tea.KeyRunes && string(msg.Runes) == "n",
    msg.Type == tea.KeyRight:
    return p, emitPlaybackRequest(ActionNext)
```
Because `"n"` is in `isPlaybackKey` (routing.go:36), it is intercepted globally and
routed to `NowPlayingPane` on every keypress regardless of focus. This also breaks
any pane that defines its own `n` handler (the Playlists pane's stub `n: new` is
dead for this reason). The `→` key alone is sufficient for next track.

**Depends on:** nothing — this story is self-contained.

## Design

### routing.go changes

**`isPlaybackKey`** (line 33–41): add `tea.KeySpace` to the return condition:
```go
func isPlaybackKey(m tea.KeyMsg) bool {
    if m.Type == tea.KeyRunes {
        switch string(m.Runes) {
        case "+", "-", "s", "r", "v": // "n" removed, " " removed (covered by KeySpace below)
            return true
        }
    }
    return m.Type == tea.KeyLeft || m.Type == tea.KeyRight || m.Type == tea.KeySpace
}
```

**`isPremiumOnlyPlaybackKey`** (line 44–53): same treatment — remove `"n"` and `" "`,
add `tea.KeySpace`:
```go
func isPremiumOnlyPlaybackKey(m tea.KeyMsg) bool {
    if m.Type == tea.KeyRunes {
        switch string(m.Runes) {
        case "+", "-", "s", "r":
            return true
        }
    }
    return m.Type == tea.KeyLeft || m.Type == tea.KeyRight || m.Type == tea.KeySpace
}
```

### nowplaying.go changes

**`handleKey` Space case** (line 356): add `tea.KeySpace` as an alternative:
```go
case msg.Type == tea.KeyRunes && string(msg.Runes) == " ",
    msg.Type == tea.KeySpace:
    ps := p.store.PlaybackState()
    if ps != nil && ps.IsPlaying {
        return p, emitPlaybackRequest(ActionPause)
    }
    return p, emitPlaybackRequest(ActionPlay)
```

**`handleKey` next-track case** (line 363–365): remove the `"n"` arm:
```go
case msg.Type == tea.KeyRight:
    return p, emitPlaybackRequest(ActionNext)
```

### Keybinding docs — all three locations, same commit

**`docs/keybinding.md`** Playback section: remove the `n | Next track` row. The `←` / `→`
row already covers next track. Space remains documented (fix is in code only).

**`docs/DESIGN.md §17`**: remove `n` from the playback keybinding table.

**`internal/ui/panes/help_overlay.go` `helpContent`** Playback section:
remove `{"n", "next track"}` from the bindings slice. The `{"← / →", "prev / next"}`
entry already covers next track.

## Acceptance Criteria

- [ ] Pressing `Space` toggles play/pause from any pane (Premium user)
- [ ] Pressing `Space` (free user) emits "Spotify Premium required" toast — no API call
- [ ] `n` no longer intercepts playback globally; pressing `n` when NowPlaying is not focused passes to the focused pane
- [ ] `→` still skips to next track
- [ ] Pane-level `n` handlers (e.g. Playlists) can now receive `n` (verified in Story 120)
- [ ] `docs/keybinding.md`, `docs/DESIGN.md §17`, and `help_overlay.go` no longer list `n` in the Playback section
- [ ] `make ci` passes

## Tasks

- [ ] Add `TestIsPlaybackKey_Space` to `internal/app/routing_test.go` — verify `tea.KeySpace` is a playback key
  - test: `go test ./internal/app/... -run TestIsPlaybackKey_Space -v` → FAIL
- [ ] Add `TestIsPlaybackKey_N_NotPlaybackKey` — verify `"n"` rune is no longer a playback key
  - test: `go test ./internal/app/... -run TestIsPlaybackKey_N_NotPlaybackKey -v` → FAIL
- [ ] Update `isPlaybackKey` and `isPremiumOnlyPlaybackKey` in `routing.go`
  - test: both routing key tests → PASS
- [ ] Add `TestNowPlayingPane_HandleKey_KeySpace_Plays` to `internal/ui/panes/nowplaying_test.go` — verify `tea.KeySpace` triggers play/pause command
  - test: `go test ./internal/ui/panes/... -run TestNowPlayingPane_HandleKey_KeySpace -v` → FAIL
- [ ] Add `TestNowPlayingPane_HandleKey_N_NoOp` — verify `"n"` rune no longer emits a playback command
  - test: `go test ./internal/ui/panes/... -run TestNowPlayingPane_HandleKey_N_NoOp -v` → FAIL
- [ ] Update `handleKey` in `nowplaying.go` (add KeySpace, remove "n" arm)
  - test: both nowplaying handle-key tests → PASS; all nowplaying tests → PASS
- [ ] Remove any existing routing/nowplaying tests that assert the old `"n"` → playback behaviour
- [ ] Update `docs/keybinding.md`, `docs/DESIGN.md §17`, and `help_overlay.go helpContent` — all in a single commit
  - test: `go build ./...` clean; grep confirms `n` does not appear in Playback sections of all three files
- [ ] `make ci` passes
