---
title: "Controls Cleanup and Fix"
status: done
---

## Description

A batch of UI/control observations covering broken keybindings, misleading pane
action hints, help overlay clutter, and one missing feature (TopArtists Enter).
All root causes were confirmed from code analysis before the spec was written.

Four areas of work:

1. **Playback key bugs** — Space (play/pause) is delivered by bubbletea v0.27 as
   `tea.KeySpace`, not as a rune, so it never routes to `NowPlayingPane`. Also `n`
   duplicates `→` for next track and prevents pane-level `n` handlers from firing.
2. **`t` key conflict** — `t` is intercepted globally as the theme switcher key, so
   TopTracks and TopArtists panes never see it. Time-range cycling is rebinding to `g`.
   TopArtists gets a long-missing Enter → play-artist action.
3. **Dead pane actions** — Several pane borders and the help overlay advertise actions
   that are stubs, no-ops, or broken due to key interception. Remove all of them.
4. **Help overlay polish** — Capitalize labels, bold key column, drop the implicit
   j/k scroll hint from Navigation and from the networklog pane border.

## Goals

- Every advertised keybinding actually works after this feature.
- Help overlay and pane borders only show actions that are usable.
- Cosmetic polish (label case, bold keys) without any behaviour regression.

## Acceptance Criteria

- [ ] `Space` triggers play/pause regardless of which pane has focus
- [ ] `n` no longer intercepts globally; pressing `n` in non-playback context passes through
- [ ] `g` cycles time range in TopTracks and TopArtists; `t` in those panes opens theme switcher (correct)
- [ ] `Enter` on a TopArtists row emits `PlayContextMsg{ContextURI: artist.URI}`
- [ ] Playlists pane border no longer shows `n` or `r`; key handlers removed
- [ ] Queue pane border no longer shows `A`
- [ ] LikedSongs pane border no longer shows `i`; key handler removed
- [ ] Help overlay does not list `n`, `x`, `Shift+↑/↓`, `i`, `A` in Pane Actions
- [ ] Help overlay Navigation does not list `j / k`
- [ ] Help overlay labels are title-case; key column is bold
- [ ] NetworkLog pane border no longer shows `j/k` hint
- [ ] `docs/keybinding.md`, `docs/DESIGN.md §17`, and `help_overlay.go` are in sync
- [ ] `make ci` passes (lint + tests + ≥ 80% coverage)

## Stories

| # | Title | Status |
|---|-------|--------|
| 118 | Playback key bugs — Space fix and remove n | open |
| 119 | t→g time-range rebind and TopArtists Enter to play | open |
| 120 | Dead pane actions removal | open |
| 121 | Help overlay polish | open |

## Files Touched (Summary)

| File | Change |
|---|---|
| `internal/app/routing.go` | Add `tea.KeySpace` to both key guards; remove `"n"` from both |
| `internal/ui/panes/nowplaying.go` | Add `tea.KeySpace` to Space case; remove `"n"` from next-track case |
| `internal/ui/panes/toptracks_pane.go` | Change `"t"` → `"g"` for time range key |
| `internal/ui/panes/topartists_pane.go` | Change `"t"` → `"g"`; add `Enter` → `PlayContextMsg` |
| `internal/ui/panes/playlists_pane.go` | Remove `n`, `r` from Actions + handlers; remove `ShiftUp/ShiftDown` cases |
| `internal/ui/panes/queue.go` | Remove `A` from Actions |
| `internal/ui/panes/likedsongs_pane.go` | Remove `i` from Actions + handleKey |
| `internal/ui/panes/networklog_pane.go` | Remove `j/k` from Actions |
| `internal/ui/panes/help_overlay.go` | Remove n, j/k, x, Shift+↑/↓, i, A entries; capitalize labels; bold keys; add g |
| `docs/keybinding.md` | Sync all changes |
| `docs/DESIGN.md §17` | Sync all changes |
