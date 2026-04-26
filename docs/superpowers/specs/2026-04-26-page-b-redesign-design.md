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

## Pane Specs

### ²Gateway Health

**File:** `internal/ui/panes/gateway_health_pane.go`
**Data source:** `store.GatewayStateSnapshot()` — refreshed on 1s `TickMsg`
**Scroll:** no · **Filter:** no

```
╭─ ²Gateway Health ────────────────────────────╮
│  Tokens   ●●●●●●●●○○  8/10                   │
│  Slots    ■■■■□□□□□□   3/5                   │
│  Backoff  none                               │
│  Dedup    none                               │
╰──────────────────────────────────────────────╯
```

- `Tokens` — filled dots = available, empty = consumed. `Warning()` color when ≤ 2 remain
- `Slots` — filled = in-flight, empty = free. `Warning()` color when all slots full
- `Backoff` — `none` (`TextMuted()`) when clear; countdown `2.1s` in `Error()` when active
- `Dedup` — `none` (`TextMuted()`) normally; `2 waiters` in `TextSecondary()` when active

---

### ³Polling Traffic

**File:** `internal/ui/panes/polling_traffic_pane.go`
**Data source:** `PollingSnapshotMsg` (tick interval, idle state) + store cache sentinels
**Scroll:** no · **Filter:** no

```
╭─ ³Polling Traffic ────────────────────────────╮
│  ▶  Playback   1s · running                   │
│                                               │
│  Playlists  ◬  2h 55m stale                   │
│  Albums     ◬  2h 55m stale                   │
│  Liked      ○  fresh                          │
│  Recent     ○  fresh                          │
╰───────────────────────────────────────────────╯
```

- Polling row: `▶` running (`Success()`) or `⏸` idle (`Warning()`) · interval · state
- Cache rows: `○` fresh (`TextMuted()`) · `◬` stale with age (`Warning()` < 1h, `Error()` ≥ 1h)
- Sources: `store.PlaylistsFetchedAt()`, `store.AlbumsFetchedAt()`,
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
│  15:52:10  ⚡  GET /v1/me/player                                                      │
│  15:52:10  ⊖   Token consumed → 9/10                                                 │
│  15:52:10  ⊞   Semaphore acquired  3/5                                               │
│  15:52:10  ✓   Request allowed                                                       │
│  15:52:09  ◷   GET /v1/me/player/queue                                               │
│  15:52:09  ⧖   Dedup joined                                                          │
│  15:52:09  ✓   Dedup resolved  200                                                   │
│  15:52:08  ↻   Tokens refilled → 10                                                  │
│  15:52:05  ✗   Request blocked  (backoff active)                                     │
╰──────────────────────────────────────────────────────────────────────────────────────╯
```

When filter is active, border shows `filter(query)`:

```
╭─ ⁴Gateway Live ──────────────────────────────────────── filter(blocked) ─────────────╮
```

#### Event Color Map

| Glyph | Event | Color token |
|-------|-------|-------------|
| `⚡` | Interactive request entered | `TextPrimary()` |
| `◷` | Background request entered | `TextMuted()` |
| `⊖` | Token consumed | `Warning()` |
| `↻` | Tokens refilled | `Success()` |
| `⊞` | Semaphore acquired | `TextSecondary()` |
| `⊟` | Semaphore released | `TextMuted()` |
| `✓` | Request allowed / dedup resolved | `Success()` |
| `✗` | Request blocked | `Error()` |
| `⧖` | Dedup joined | `TextSecondary()` |
| `⏳` | Backoff started | `Error()` |

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
│  15:51:00    GET       /v1/me/player                 429      12ms      Allowed    bg   │
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
| All panes | `Esc` standardized: clears filter / resets scroll |
| `docs/keybinding.md`, `docs/DESIGN.md §17`, `help_overlay.go` | New entries for scroll + Esc reset |

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
| `internal/ui/panes/networklog_pane.go` | Update column headers to Title Case |
| `internal/ui/panes/help_overlay.go` | Add scroll + Esc keybindings |
| `internal/ui/layout/` | Update Page B grid preset |
| `internal/app/app.go` | Reroute `PollingSnapshotMsg`, wire new panes |
| `docs/DESIGN.md §17` | Update Page B spec, add keybindings |
| `docs/keybinding.md` | Add scroll + Esc entries |
