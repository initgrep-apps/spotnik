---
title: "t→g Time-Range Rebind and TopArtists Enter to Play"
feature: 24-controls-cleanup
status: done
---

## Background

**Group 2 — `t` key conflict.** Both TopTracksPane and TopArtistsPane use `t` to cycle
the time range (4wk → 6mo → all). However, `routing.go:145–151` intercepts `t` globally
to open the theme switcher before the pane ever sees the key. The time-range cycle is
therefore completely broken.

**Decision (Option B):** keep global keys independent of pane context; rebind the
time-range key to `g` (mnemonic: "go to next range"). `g` is not used anywhere in the
codebase.

**Group 4 — TopArtists Enter.** `topartists_pane.go:165–167` explicitly comments out
Enter with a note that artist play is unsupported. This is incorrect:
`PUT /v1/me/player/play` with `context_uri: spotify:artist:{id}` is a valid Spotify API
operation. `domain.FullArtist.URI` exists (types.go:269). The existing `buildPlayContextCmd`
in commands.go handles this without any changes — it accepts any Spotify context URI.

**Depends on:** nothing — this story is self-contained.

## Design

### toptracks_pane.go

**`Update` key case** (line 162): change `"t"` → `"g"`:
```go
case keyMsg.Type == tea.KeyRunes && string(keyMsg.Runes) == "g":
    return p.cycleTimeRange()
```

**`Actions()`** (line 101): change key label:
```go
{Key: "g", Label: rangeLabel},
```

### topartists_pane.go

Same `"t"` → `"g"` change in `Update` and `Actions()`.

Additionally, add an Enter handler in `Update` before the navigation fall-through:
```go
case keyMsg.Type == tea.KeyEnter:
    artists := a.filteredArtists()
    idx := a.table.SelectedIndex()
    if idx >= 0 && idx < len(artists) {
        uri := artists[idx].URI
        return a, func() tea.Msg { return PlayContextMsg{ContextURI: uri} }
    }
    return a, nil
```

Remove the comment block about Enter having no action.

### NowPlaying Actions display (investigate during implementation)

The design spec notes that only `s` and `r` are visible in the NowPlaying pane border,
even though `Actions()` defines five entries: `s`, `r`, `space`, `+/-`, `v`.
During implementation, check `layout.RenderPaneBorder` for action truncation behaviour.
If the border truncates when the pane is narrow, this is expected behaviour — document
the finding as a comment or `// NOTE:` in the code. If it is a bug, fix it in this story.

### Keybinding docs — all three locations, same commit

**`docs/keybinding.md`** Pane Actions section:
- Add: `| g | Cycle time range | TopTracks / TopArtists |`
- `Enter | select / play` already covers TopArtists play — no change needed there.

**`docs/DESIGN.md §17`**: add `g` to the pane actions keybinding table.

**`internal/ui/panes/help_overlay.go` `helpContent`** Pane Actions section:
- Add `{"g", "Cycle time range"}` to the bindings slice.

## Acceptance Criteria

- [ ] Pressing `g` in TopTracksPane cycles the time range (short_term → medium_term → long_term → short_term)
- [ ] Pressing `g` in TopArtistsPane cycles the time range the same way
- [ ] Pressing `t` in TopTracksPane or TopArtistsPane opens the theme switcher (existing global behaviour — now unobstructed)
- [ ] Pressing `Enter` on a TopArtistsPane row emits `PlayContextMsg{ContextURI: "spotify:artist:{id}"}` for the selected artist
- [ ] Pressing `Enter` on an empty TopArtistsPane (no data) does nothing
- [ ] `g` appears in `help_overlay.go helpContent` Pane Actions, `docs/keybinding.md`, and `docs/DESIGN.md §17`
- [ ] `make ci` passes

## Tasks

- [ ] Add `TestTopTracksPane_GKey_CyclesTimeRange` to `internal/ui/panes/toptracks_pane_test.go` — verify `g` advances time range
  - test: `go test ./internal/ui/panes/... -run TestTopTracksPane_GKey -v` → FAIL
- [ ] Add `TestTopTracksPane_TKey_DoesNotCycle` — verify pressing `t` in TopTracksPane no longer cycles time range
  - test: `go test ./internal/ui/panes/... -run TestTopTracksPane_TKey_DoesNotCycle -v` → FAIL
- [ ] Update `toptracks_pane.go` (`"t"` → `"g"` in Update case and Actions)
  - test: both toptracks key tests → PASS
- [ ] Add `TestTopArtistsPane_GKey_CyclesTimeRange` and `TestTopArtistsPane_TKey_DoesNotCycle`
  - test: → FAIL
- [ ] Add `TestTopArtistsPane_Enter_EmitsPlayContextMsg` — verify Enter on a row emits `PlayContextMsg` with artist URI
  - test: `go test ./internal/ui/panes/... -run TestTopArtistsPane_Enter -v` → FAIL
- [ ] Add `TestTopArtistsPane_Enter_NoData_NoOp` — verify Enter on empty list returns nil cmd
  - test: → FAIL
- [ ] Update `topartists_pane.go` (`"t"` → `"g"`; add Enter handler)
  - test: all topartists tests → PASS
- [ ] Update any existing tests that assert `"t"` is the time-range key (replace with `"g"`)
- [ ] Investigate NowPlaying border action truncation — check `layout.RenderPaneBorder` for truncation on narrow widths; document finding; fix if it is a bug
- [ ] Update `docs/keybinding.md`, `docs/DESIGN.md §17`, and `help_overlay.go helpContent` — all in a single commit
  - test: `go build ./...` clean; grep confirms `g` appears in all three files as time-range key
- [ ] `make ci` passes
