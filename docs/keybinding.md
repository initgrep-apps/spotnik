# Spotnik Keybindings

> **Keep this file in sync** with `docs/DESIGN.md §17` and the `helpContent` var
> in `internal/ui/panes/help_overlay.go` whenever any keybinding changes.

---

## Global

| Key | Action |
|-----|--------|
| `/` | Open search overlay |
| `d` | Open device switcher |
| `u` | Open user profile overlay |
| `t` | Open theme switcher |
| `?` | Open help overlay |
| `q` | Quit |
| `0` | Toggle Page A / Page B |
| `1`–`8` | Toggle pane visibility (Page A only) |
| `p` | Cycle preset |

## Playback

Playback keys are always active regardless of which pane has focus.

| Key | Action |
|-----|--------|
| `Space` | Play / pause |
| `←` / `→` | Previous / next track |
| `+` / `-` | Volume up / down |
| `s` | Toggle shuffle |
| `r` | Cycle repeat mode |
| `v` | Cycle visualizer pattern |

## Navigation

| Key | Action |
|-----|--------|
| `Tab` | Next pane focus |
| `Shift+Tab` | Previous pane focus |
| `↑` / `k` | Scroll up |
| `↓` / `j` | Scroll down |
| `Esc` | Close overlay · clear filter · scroll top |

## Pane Actions

| Key | Action | Context |
|-----|--------|---------|
| `Enter` | Select / play item | Focused pane |
| `f` | Toggle filter | List panes |
| `g` | Cycle time range | TopTracks / TopArtists |

## Profile Overlay

| Key | Context | Action |
|-----|---------|--------|
| `l` | Profile overlay | Logout — ends session, keeps Client ID. Press twice to confirm. |
| `f` | Profile overlay | Forget — removes session + Client ID. Press twice to confirm. |

## Search Overlay

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Cycle search category |
| `Enter` | Play selected result |
| `Ctrl+A` | Add result to queue |
| `Ctrl+U` | Clear search input |
| `PgDn` / `PgUp` | Next / previous result page |
| `Esc` | Close search overlay |
