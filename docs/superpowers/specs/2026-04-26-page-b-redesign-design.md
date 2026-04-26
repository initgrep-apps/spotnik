# Page B Redesign — Design Spec

**Date:** 2026-04-26
**Branch:** feat/stats-cols-refine
**Status:** Approved — ready for implementation

---

## Goal

Redesign Page B (Nerd Status) so it is consistent with Page A: pane-based, focusable,
toggleable with number keys, filterable, same PaneChrome chrome. Replace the current
monolithic `RequestFlowPane` with three focused panes. Establish a universal scroll/filter/Esc
reset behavior that applies to all panes across the app.

---

## Layout

### Grid Definition

```
Row 1  weight 1   [{NowPlaying, w=1}]
Row 2  weight 3   [{GatewayHealth, w=1}, {PollingTraffic, w=1}, {GatewayLive, w=2}]
Row 3  weight 2   [{NetworkLog, w=1}]
```

Row 2 column widths: Gateway Health ~25% · Polling Traffic ~25% · Gateway Live ~50%.
Gateway Live gets double weight — event lines need width to avoid truncation.

### Visual

```
╭─ ¹Now Playing ── track · artist ── ▶ time ──────────────────────────────────────────╮
│                                                                                      │
╰──────────────────────────────────────────────────────────────────────────────────────╯
╭─ ²Gateway Health ──────╮ ╭─ ³Polling Traffic ──────╮ ╭─ ⁴Gateway Live ─── f filter ─╮
│                        │ │                         │ │                              │
│    w=1 (~25%)          │ │    w=1 (~25%)           │ │    w=2 (~50%)                │
│                        │ │                         │ │                              │
╰────────────────────────╯ ╰─────────────────────────╯ ╰──────────────────────────────╯
╭─ ⑤Network Log ──────────────────────────────────────────────────── f filter ──────────╮
│                          w=1 (100%)                                                   │
╰───────────────────────────────────────────────────────────────────────────────────────╯
```

### Toggle Keys (Page B)

| Key | Pane |
|-----|------|
| `1` | Now Playing (shared with Page A) |
| `2` | Gateway Health |
| `3` | Polling Traffic |
| `4` | Gateway Live |
| `5` | Network Log |

Focus rotation: `Tab` cycles through visible panes left-to-right, top-to-bottom.
Hidden panes (toggled off) are skipped. Same behavior as Page A.

---

## Primitive Compliance

All new panes follow `docs/PANE-TEMPLATE.md` scaffold:

- `View()` delegates chrome to `uikit.PaneChrome{...}.Render(content)` — no raw `lipgloss.NewStyle()` borders at call sites
- Content rows use `uikit.ListRow` — the same primitive used by profile overlay and search overlay
- Status indicators use `uikit.StatusGlyph` where a single intent-coloured glyph + text suffices
- All glyphs referenced by role constant (`GlyphPlaying`, `GlyphWarning`, etc.) — never raw rune literals in render code

### New Glyph Roles Required

The following gateway-domain glyphs are absent from the current catalogue in `internal/uikit/glyph.go` and `docs/TUI-DESIGN-SYSTEM.md §4`. They **must be added in the same PR** as the new pane files:

| Role constant | Category | Unicode | ASCII | Usage |
|---|---|---|---|---|
| `GlyphGatewayTokenOut` | `gateway.token.out` | `⊖` | `(-)` | Token consumed |
| `GlyphGatewaySemIn` | `gateway.sem.in` | `⊞` | `[+]` | Semaphore acquired |
| `GlyphGatewaySemOut` | `gateway.sem.out` | `⊟` | `[-]` | Semaphore released |
| `GlyphGatewayBackoff` | `gateway.backoff` | `⏳` | `(t)` | Backoff started |

Update both `glyphTable` in `glyph.go` and the catalogue table in `docs/TUI-DESIGN-SYSTEM.md §4`.

---

## Pane Specs

### ²Gateway Health

**File:** `internal/ui/panes/gateway_health_pane.go`
**Data source:** `store.GatewayStateSnapshot()` — refreshed on 1s `TickMsg`
**Scroll:** no · **Filter:** no

```
╭─ ²Gateway Health ────────────────────────────╮
│  ●●●●●●●●○○  Tokens  8/10                    │
│  ■■■■□□□□□□   Slots   3/5                    │
│  ◷  Backoff  none                            │
│  ⧖  Dedup    none                            │
╰──────────────────────────────────────────────╯
```

Each metric is a `uikit.ListRow`:

| Metric | Glyph | Label | Caption | Intent |
|--------|-------|-------|---------|--------|
| Tokens | `""` (no glyph) | dot-bar string built from `GlyphFilledDot`/`GlyphAvailable` | `"8/10"` | `RoleWarning` when ≤ 2 remain, else `RolePlain` |
| Slots | `""` (no glyph) | square-bar string built from `GlyphFilledSquare`/`GlyphEmptySquare` | `"3/5"` | `RoleWarning` when all slots full, else `RolePlain` |
| Backoff | `GlyphDeadline` (◷) | `"Backoff"` | `"none"` or `"2.1s"` | `RoleMuted` when clear, `RoleError` when active |
| Dedup | `GlyphRateLimit` (⧖) | `"Dedup"` | `"none"` or `"2 waiters"` | `RoleMuted` when clear, `RoleInfo` when active |

For Tokens and Slots the bar is a string assembled at render time using `uikit.GlyphFor(GlyphFilledDot, mode)` / `GlyphFor(GlyphAvailable, mode)` repeated for capacity. The assembled string becomes the ListRow `Label`; the `n/max` count is `Caption`.

---

### ³Polling Traffic

**File:** `internal/ui/panes/polling_traffic_pane.go`
**Data source:** `PollingSnapshotMsg` (tick interval, idle state) + store cache sentinels
**Scroll:** no · **Filter:** no

```
╭─ ³Polling Traffic ────────────────────────────╮
│  ▶  Playback   1s · running                   │
│                                               │
│  ◬  Playlists  2h 55m stale                   │
│  ◬  Albums     2h 55m stale                   │
│  ○  Liked      fresh                          │
│  ○  Recent     fresh                          │
╰───────────────────────────────────────────────╯
```

Each row is a `uikit.ListRow`:

| Row | Glyph | Label | Caption | Intent |
|-----|-------|-------|---------|--------|
| Playback | `GlyphPlaying` (▶) running / `GlyphPaused` (⏸) idle | `"Playback"` | `"1s · running"` | `RoleSuccess` running, `RoleWarning` idle |
| Playlists | `GlyphWarning` (◬) stale / `GlyphAvailable` (○) fresh | `"Playlists"` | `"2h 55m stale"` or `"fresh"` | `RoleWarning` < 1h stale, `RoleError` ≥ 1h, `RoleMuted` fresh |
| Albums | same pattern | `"Albums"` | staleness age or `"fresh"` | same |
| Liked | same pattern | `"Liked"` | same | same |
| Recent | same pattern | `"Recent"` | same | same |

Sources: `store.PlaylistsFetchedAt()`, `store.AlbumsFetchedAt()`,
`store.LikedTracksFetchedAt()`, `store.RecentPlayedFetchedAt()`

---

### ⁴Gateway Live

**File:** `internal/ui/panes/gateway_live_pane.go`
**Data source:** `store.ReadEventsFrom(cursor)` — 500-entry display buffer, prepend on tick
**Scroll:** `j`/`k` · `↑`/`↓` · **Filter:** `f` → type → `Enter`

Auto-scrolling reverse-chronological event stream. New events prepend at top on each
1s tick. Scrolling lets you read history while new events continue to arrive silently.

```
╭─ ⁴Gateway Live ──────────────────────────────────────────────── f filter ────────────╮
│  ⚡  15:52:10  GET /v1/me/player                                                      │
│  ⊖  15:52:10  Token consumed → 9/10                                                  │
│  ⊞  15:52:10  Semaphore acquired  3/5                                                │
│  ✓  15:52:10  Request allowed                                                        │
│  ◷  15:52:09  GET /v1/me/player/queue                                                │
│  ⧖  15:52:09  Dedup joined                                                           │
│  ✓  15:52:09  Dedup resolved  200                                                    │
│  ↻  15:52:08  Tokens refilled → 10                                                   │
│  ✗  15:52:05  Request blocked  (backoff active)                                      │
╰──────────────────────────────────────────────────────────────────────────────────────╯
```

When filter is active, border shows `filter(query)`:

```
╭─ ⁴Gateway Live ──────────────────────────────────────── filter(blocked) ─────────────╮
```

Each event is a `uikit.ListRow`. Field mapping:

| Field | Value |
|-------|-------|
| `Glyph` | Event-type glyph role from the table below |
| `Label` | `"HH:MM:SS  <event description>"` (timestamp + text combined) |
| `Caption` | `""` (empty) |
| `Intent` | Role from the colour map below |

#### Event Glyph and Color Map

| Glyph role | Unicode | Event | Intent (Role) |
|---|---|---|---|
| `GlyphRunning` | `⚡` | Interactive request entered | `RolePlain` |
| `GlyphDeadline` | `◷` | Background request entered | `RoleMuted` |
| `GlyphGatewayTokenOut` | `⊖` | Token consumed | `RoleWarning` |
| `GlyphRepeatAll` | `↻` | Tokens refilled | `RoleSuccess` |
| `GlyphGatewaySemIn` | `⊞` | Semaphore acquired | `RoleInfo` |
| `GlyphGatewaySemOut` | `⊟` | Semaphore released | `RoleMuted` |
| `GlyphSuccess` | `✓` | Request allowed / dedup resolved | `RoleSuccess` |
| `GlyphError` | `✗` | Request blocked | `RoleError` |
| `GlyphRateLimit` | `⧖` | Dedup joined | `RoleInfo` |
| `GlyphGatewayBackoff` | `⏳` | Backoff started | `RoleError` |

`GlyphGatewayTokenOut`, `GlyphGatewaySemIn`, `GlyphGatewaySemOut`, `GlyphGatewayBackoff`
are new catalogue entries — see **New Glyph Roles Required** above.

#### Filter Matches On
Endpoint path · event type keyword (allowed, blocked, dedup, backoff, token) · priority (interactive, background)

---

### ⑤Network Log

**File:** `internal/ui/panes/networklog_pane.go` (existing — minimal changes)
**Data source:** `store.ReadEventsFrom(cursor)` — separate cursor from Gateway Live
**Scroll:** `j`/`k` · `↑`/`↓` · **Filter:** `f` → type (real-time) · `Esc` to clear

Unified reverse-chronological HTTP request table. Newest row at top.

```
╭─ ⑤Network Log ──────────────────────────────────────────────────── f filter ──────────╮
│  Time        Method    Endpoint                     Status   Latency   Decision   Pri  │
│  15:52:09    GET       /v1/me/player                 204      40ms      Allowed    bg   │
│  15:52:09    GET       /v1/me/player/queue           200      32ms      Deduped    bg   │
│  15:51:59    GET       /v1/me/player                 204      33ms      Allowed    bg   │
│  15:51:49    GET       /v1/me/player                 204      39ms      Allowed    bg   │
│  15:51:15    PUT       /v1/me/player/play            204      18ms      Allowed    ⚡   │
│  15:51:00    GET       /v1/me/player                 429      12ms      Blocked    bg   │
│  ▼ 1/20                                                                                │
╰────────────────────────────────────────────────────────────────────────────────────────╯
```

#### Columns

| Column | Width | Color token | Notes |
|--------|-------|-------------|-------|
| Time | 10% | `ColumnIndex()` | `HH:MM:SS` |
| Method | 8% | `ColumnSecondary()` | GET / PUT / POST |
| Endpoint | 35% | `ColumnPrimary()` | full path, truncated with `…` |
| Status | 8% | `ColumnTertiary()` | `Success()` 2xx · `Warning()` 429 · `Error()` 5xx |
| Latency | 8% | `ColumnTertiary()` | `ms` suffix |
| Decision | 13% | `ColumnSecondary()` | `Allowed` · `Deduped` · `Blocked` |
| Priority | 8% | `ColumnIndex()` | `bg` (◷) · `⚡` interactive |

Filter matches on: endpoint, status, decision, priority.

#### Decision Column Bug Fix

The current `refreshRows()` rebuilds `decisions := make(map[uint64]domain.EventKind)` on every
tick. If `EventRequestAllowed` / `EventRequestBlocked` / `EventDedupJoined` arrives in tick N
and its paired `EventHttpCompleted` arrives in tick N+1, the decision is lost — every row shows
empty/Allowed regardless.

**Fix:** promote `decisions` to a persistent struct field:

```go
type NetworkLogPane struct {
    // ... existing fields ...
    pendingDecisions map[uint64]domain.EventKind  // persists across tick cycles
}

func NewNetworkLogPane(...) *NetworkLogPane {
    return &NetworkLogPane{
        // ...
        pendingDecisions: make(map[uint64]domain.EventKind),
    }
}
```

In `refreshRows()`:
1. Use `p.pendingDecisions` (the struct field) instead of the local `decisions` variable.
2. After recording a decision for an `EventHttpCompleted` row, call `delete(p.pendingDecisions, e.RequestID)` to avoid unbounded growth.
3. Accumulate new decision events into `p.pendingDecisions` before the HTTP-completed pass.

---

## Universal Pane Behavior (Cross-Cutting)

This behavior applies to **every scrollable or filterable pane across Page A and Page B**.

### Scroll

- `j` / `↓` — scroll down
- `k` / `↑` — scroll up
- No custom scroll position text added to the border. Table component's built-in `▼`/`▲`
  indicators in content area remain — these are from the table component, not custom additions.
- Live panes (Gateway Live): new events continue to prepend at top while scrolled;
  user is reading history
- `Esc` — reset scroll to top (page 1)

### Filter

- `f` — open filter input
- Type query → `Enter` — apply filter; border label changes to `filter(query)`
- `Esc` — clear filter, return to full unfiltered list

### Esc Priority

If both filter and scroll are active, `Esc` clears filter first, then on next `Esc` resets scroll.

### Help Overlay Additions

Add to `helpContent` in `internal/ui/panes/help_overlay.go`:

```
↑ / k     scroll up
↓ / j     scroll down
Esc       reset (clear filter / back to page 1)
```

Same entries added to `docs/keybinding.md` and `docs/DESIGN.md §17` in the same commit.

---

## What's New

| Item | Detail |
|------|--------|
| `GatewayHealthPane` | New file `gateway_health_pane.go` |
| `PollingTrafficPane` | New file `polling_traffic_pane.go`. Receives `PollingSnapshotMsg` (rerouted from `RequestFlowPane`) |
| `GatewayLivePane` | New file `gateway_live_pane.go`. Scrollable + filterable event stream |
| Page B grid | 4 panes, toggle keys `2`–`5` |
| Universal Esc reset | All panes — scroll reset + filter clear |
| Help overlay keybindings | `↑/k`, `↓/j`, `Esc reset` |

## What Changes

| Item | Change |
|------|--------|
| `RequestFlowPane` | **Retired and deleted** — logic split into 3 new panes |
| Page B grid definition | 2 panes → 4 panes |
| `PollingSnapshotMsg` routing in `app.go` | `RequestFlowPane` → `PollingTrafficPane` |
| All uppercase labels in Page B panes | Title Case — matches Page A style |
| `NetworkLogPane` column headers | Uppercase → Title Case |
| `NetworkLogPane` Decision column | `decisions` local map → `pendingDecisions` persistent struct field |
| All panes | `Esc` standardized: clears filter / resets scroll |
| `docs/keybinding.md`, `docs/DESIGN.md §17`, `help_overlay.go` | New entries for scroll + Esc reset |
| `internal/uikit/glyph.go` | Add 4 new gateway glyph roles |
| `docs/TUI-DESIGN-SYSTEM.md §4` | Add 4 new gateway glyph entries to catalogue |

## What Stays Unchanged

- `NetworkLogPane` — structure, columns, cursor logic, filter mechanism unchanged
- All Page A panes — structure and data unchanged (only Esc behavior added)
- `PaneChrome` border design
- `ToggleKey()` = `1` for NowPlaying on both pages
- Toast notifications, overlays, theme system
- No new Spotify API calls — all Page B data is internal

---

## Files Touched

| File | Action |
|------|--------|
| `internal/ui/panes/gateway_health_pane.go` | **Create** |
| `internal/ui/panes/gateway_health_pane_test.go` | **Create** |
| `internal/ui/panes/gateway_live_pane.go` | **Create** |
| `internal/ui/panes/gateway_live_pane_test.go` | **Create** |
| `internal/ui/panes/polling_traffic_pane.go` | **Create** |
| `internal/ui/panes/polling_traffic_pane_test.go` | **Create** |
| `internal/ui/panes/requestflow_pane.go` | **Delete** |
| `internal/ui/panes/requestflow_pane_test.go` | **Delete** |
| `internal/ui/panes/requestflow_boxed.go` | **Delete** |
| `internal/ui/panes/requestflow_boxed_test.go` | **Delete** |
| `internal/ui/panes/requestflow_replay.go` | **Delete** |
| `internal/ui/panes/requestflow_replay_test.go` | **Delete** |
| `internal/ui/panes/networklog_pane.go` | Update column headers to Title Case; fix `pendingDecisions` persistent field |
| `internal/ui/panes/help_overlay.go` | Add scroll + Esc keybindings |
| `internal/ui/layout/` | Update Page B grid preset |
| `internal/app/app.go` | Reroute `PollingSnapshotMsg`, wire new panes |
| `internal/uikit/glyph.go` | Add 4 gateway glyph roles + table entries |
| `docs/TUI-DESIGN-SYSTEM.md §4` | Add 4 gateway glyph entries to catalogue |
| `docs/DESIGN.md §17` | Update Page B spec, add keybindings |
| `docs/keybinding.md` | Add scroll + Esc entries |
