---
title: "Page B Redesign (Nerd Status)"
status: done
---

## Description

Replace the monolithic `RequestFlowPane` with three focused panes (`GatewayHealthPane`,
`PollingTrafficPane`, `GatewayLivePane`) that each own a distinct slice of gateway
diagnostics. `NetworkLogPane` is fixed in-place (decision cross-tick bug, Title Case
headers). The redesign also establishes a universal Esc scroll-reset behaviour across
every table-based pane in the application.

**Design record:** `docs/superpowers/specs/2026-04-26-page-b-redesign-design.md` — full
rationale, column specs, glyph maps, filter semantics, and layout grid.

**Implementation plan:** `docs/superpowers/plans/2026-04-26-page-b-redesign.md` — step-by-step
TDD guide for every story. Each story cross-references the matching `## Task N` section.

**What changes:**

- New `Table.GotoTop()` and `Table.CurrentPage()` methods on `components.Table`; Esc
  scroll-reset wired into every table-based pane (Queue, TopTracks, TopArtists, LikedSongs,
  RecentlyPlayed, NetworkLog, Albums, Playlists) and all three keybinding doc locations.
- New `PaneID` constants: `PaneGatewayHealth`, `PanePollingTraffic`, `PaneGatewayLive`;
  `PaneRequestFlow` removed; `TogglePane` guard updated to `>= PaneNetworkLog`.
- New `GatewayHealthPane` (toggle key 2): 4-row fixed grid showing token bucket fill,
  concurrent semaphore, backoff countdown, and dedup waiter count; driven by
  `store.ReadEventsFrom(cursor)`.
- New `PollingTrafficPane` (toggle key 3): 5-row fixed grid showing playback poll cadence
  (idle/running) and library cache freshness (playlists, albums, liked songs, recently played).
- New `GatewayLivePane` (toggle key 4): 500-entry reverse-chronological event stream;
  scrollable, filterable (Enter-to-apply filter, `filter(query)` in border), Esc clears
  committed filter first then resets scroll.
- `NetworkLogPane` fixed: `pendingDecisions` promoted to persistent struct field (cross-tick
  decision association bug), column headers changed to Title Case, toggle key updated to 5.
- `PresetNerdStatus` updated to the new 3-row 5-pane grid.
- Six RequestFlow source files deleted after all replacements are wired.

**Supersedes:** `internal/ui/panes/requestflow_pane.go`, `requestflow_boxed.go`,
`requestflow_replay.go` and their tests.

**Depends on:** Feature 13 (TUI Design System) — `uikit.PaneChrome`, `uikit.ListRow`,
`uikit.GlyphFor`, and `uikit.PadOrTruncate` must be available.

## Acceptance Criteria

- [ ] `components.Table` exposes `GotoTop()` and `CurrentPage()` methods
- [ ] Pressing `Esc` with no active filter resets scroll to page 1 on Queue, TopTracks,
      TopArtists, LikedSongs, RecentlyPlayed, NetworkLog, Albums, Playlists, GatewayLive
- [ ] All three keybinding locations (`help_overlay.go`, `docs/keybinding.md`, `docs/DESIGN.md`)
      document `Esc` as "close overlay · clear filter · scroll top"
- [ ] `PaneGatewayHealth`, `PanePollingTraffic`, `PaneGatewayLive` constants exist;
      `PaneRequestFlow` constant does not exist
- [ ] `TogglePane` guard uses `>= PaneNetworkLog` (Page B panes are not toggleable via number keys)
- [ ] `GatewayHealthPane` renders 4 health rows (Tokens, Slots, Backoff, Dedup) with correct
      warning colours; data driven by gateway events via `store.ReadEventsFrom(cursor)`
- [ ] `PollingTrafficPane` renders 5 traffic rows (Playback + 4 library cache rows) with
      fresh/stale status derived from store TTL sentinel methods
- [ ] `GatewayLivePane` maintains a 500-entry reverse-chronological buffer; scrollable;
      `f` opens filter, `Enter` commits, `Esc` clears committed filter or resets scroll;
      `filter(query)` appears in pane border when a committed filter is active
- [ ] `NetworkLogPane.pendingDecisions` persists decision events across ticks so the
      `Decision` column shows the correct value when `HttpCompleted` arrives on a later tick
- [ ] `NetworkLogPane` column headers are Title Case
- [ ] `PresetNerdStatus` contains exactly 5 panes in a 3-row grid (NowPlaying strip, 3-pane
      diagnostic row, NetworkLog full-width row)
- [ ] Six `requestflow_*` files deleted; `humanAge` / `humanInterval` removed with them;
      `cacheAge` and `pollingHumanInterval` live in their respective new pane files
- [ ] `make ci` passes after every story

## Post-implementation fixes

- **PR #229 (2026-04-27):** Story 180 RowSpan rendering bug. `renderGrid()` used
  `lipgloss.JoinHorizontal/JoinVertical` flow composition that ignored `Rect.X`/`Rect.Y`,
  so on Page B the spanner's height bloated its logical row, hiding NowPlaying and
  rendering PollingTraffic in a separate vertical block. Replaced with an absolute-position
  line-by-line compositor and corrected `focusOrder` so spanners are appended after their
  origin-row siblings (Tab now walks left-to-right). Regression tests pin both the
  Page B `VisiblePanes()` order and the rendered output's column alignment.
