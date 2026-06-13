---
title: "Episode Details overlay (i key)"
feature: 19-player-page-unification
status: open
---

## Background

When an episode is playing, users need to see the full episode description. The
`i` key opens a centered overlay with episode details, description (rendered from
`html_description`), and metadata.

## Design

### New overlay: `EpisodeDetailsOverlay`

A new `EpisodeDetailsOverlay` struct in `internal/ui/panes/episode_details_overlay.go`
implements `tea.Model` (Init, Update, View).

**Layout** — centered overlay (same pattern as Help and Device overlays):

```
╭─ Episode Details ──────────────────────────────── Esc ╮
│                                                        │
│  #497 – Biggest Mysteries in Physics                   │
│  Lex Fridman Podcast · 3h 01m · May 29, 2026           │
│  Published by: Lex Fridman                              │
│                                                        │
│  <html_description rendered as styled plain text>       │
│                                                        │
╰────────────────────────────────────────────────────────╯
```

**Interaction:**
- `i` opens overlay when `currently_playing_type == "episode"`. Silent no-op
  when a track is playing or nothing is playing.
- `Esc` or `q` closes overlay.
- `j`/`k` or `↓`/`↑` scrolls description if it exceeds overlay height.
- Overlay blocks all other input while open (same priority as help/device overlays).

**Data source** — `PlaybackState.Episode` fields:

| Field | Overlay Use | Theme Token |
|-------|-------------|-------------|
| `name` | Title (first line) | `TextPrimary()` bold |
| `show.name` | Metadata line | `TextSecondary()` |
| `show.publisher` | Publisher line | `TextSecondary()` |
| `duration_ms` | Metadata line (formatted) | `TextSecondary()` |
| `release_date` | Metadata line | `TextSecondary()` |
| `html_description` | Primary description | Styled plain text |
| `description` | Fallback plain text | Plain |

**Description rendering priority:**
1. `html_description` — preferred. Stripped of HTML tags with semantic formatting
   preserved: headings → `TextPrimary()` bold, `<b>`/`<strong>` → bold,
   `<a>` → `TextSecondary()`, list markers preserved, `<p>`/`<br>` → line breaks.
2. `description` — plain text fallback when `html_description` is empty.
3. Both empty → `"No description available."` in `TextMuted()`.

Uses existing `htmlToMarkdown()` and `renderMarkdown()` from
`internal/ui/panes/htmlrender.go` (unexported, accessible within `panes` package).

### Message types

```go
type EpisodeDetailsOpenMsg struct{}
type EpisodeDetailsClosedMsg struct{}
```

### App wiring

- `episodeDetailsOpen bool` field on App
- `episodeDetails *panes.EpisodeDetailsOverlay` field on App
- `i` key handler in routing.go: opens overlay only when episode is playing
- Overlay guard in Update loop: routes input to overlay when open
- Overlay rendering in render.go: centered, same pattern as help overlay

### Keybinding updates (3 locations)

This story adds the `i` keybinding. **All 3 locations must be updated in the
same commit** (per AGENTS.md rule #15):

1. `README.md` Keybindings section — add `i` for episode details
2. `docs/system/design.md` §17 — add `i` row
3. `internal/ui/panes/help_overlay.go` `helpContent` — add `i` entry

## Files

### Create

- `internal/ui/panes/episode_details_overlay.go`
- `internal/ui/panes/episode_details_overlay_test.go`

### Modify

- `internal/ui/panes/messages.go` — add `EpisodeDetailsOpenMsg`, `EpisodeDetailsClosedMsg`
- `internal/app/app.go` — add overlay fields and open/close methods
- `internal/app/routing.go` — add `i` key handler with episode check
- `internal/app/handlers.go` — handle `EpisodeDetailsClosedMsg`
- `internal/app/render.go` — add overlay rendering
- `internal/ui/panes/help_overlay.go` — add `i` keybinding entry
- `README.md` — add `i` keybinding
- `docs/system/design.md` §17 — add `i` keybinding row

## Acceptance Criteria

- [ ] `i` key opens Episode Details overlay when episode is playing
- [ ] `i` key is silent no-op when track is playing or nothing is playing
- [ ] `Esc` or `q` closes overlay
- [ ] `j`/`k` and `↓`/`↑` scroll description
- [ ] Overlay blocks all other input while open
- [ ] `html_description` is rendered with semantic formatting preserved
- [ ] `description` is used as fallback when `html_description` is empty
- [ ] `"No description available."` shown when both are empty
- [ ] Keybinding added in all 3 locations (README, design.md, help_overlay.go)
- [ ] `make ci` passes

## Tasks

- [ ] Create `EpisodeDetailsOverlay` struct with `Init()`, `Update()`, `View()` methods
      - Create `internal/ui/panes/episode_details_overlay.go`: centered overlay, scrollable description, `Esc`/`q` close, `j`/`k`/`↑`/`↓` scroll
      - Create `internal/ui/panes/episode_details_overlay_test.go`
      - test: `TestEpisodeDetailsOverlay_View_ShowsEpisodeName`, `TestEpisodeDetailsOverlay_View_ShowsShowName`, `TestEpisodeDetailsOverlay_View_ShowsNoDescription`, `TestEpisodeDetailsOverlay_EscCloses`, `TestEpisodeDetailsOverlay_ScrollUpDown`
- [ ] Add `EpisodeDetailsOpenMsg` and `EpisodeDetailsClosedMsg` message types
      - Modify `internal/ui/panes/messages.go`
      - test: `TestEpisodeDetailsOpenMsg_Type`, `TestEpisodeDetailsClosedMsg_Type`
- [ ] Wire overlay into `App`: fields, open/close, rendering, input guard
      - Modify `internal/app/app.go`: add `episodeDetailsOpen bool`, `episodeDetails *panes.EpisodeDetailsOverlay`
      - Modify `internal/app/render.go`: render overlay when open
      - test: `TestApp_EpisodeDetailsOpen_WhenEpisode`, `TestApp_EpisodeDetailsOpen_WhenTrack_NoOp`
- [ ] Add `i` key handler in `routing.go` with `CurrentlyPlayingType == "episode"` guard
      - Modify `internal/app/routing.go`: `i` key opens overlay only when episode is playing
      - test: `TestRouting_IKey_OpensOverlay_WhenEpisode`, `TestRouting_IKey_NoOp_WhenTrack`
- [ ] Handle `EpisodeDetailsClosedMsg` in `handlers.go`
      - Modify `internal/app/handlers.go`: clear overlay state on close
      - test: `TestHandler_EpisodeDetailsClosed_ClearsState`
- [ ] Update keybindings in all 3 locations (same commit per AGENTS.md rule #15)
      - Modify `README.md` Keybindings section: add `i` entry
      - Modify `docs/system/design.md` §17: add `i` row
      - Modify `internal/ui/panes/help_overlay.go` `helpContent`: add `i` entry (conditionally when episode playing)
      - test: `TestHelpOverlay_ContainsIDetails`
- [ ] Run `make ci` — all lint, tests, and 80% coverage pass