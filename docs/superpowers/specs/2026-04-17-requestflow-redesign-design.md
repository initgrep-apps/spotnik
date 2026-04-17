# RequestFlow Pane Redesign

**Date:** 2026-04-17
**Status:** draft
**Feature area:** `internal/ui/panes/requestflow_pane.go`, `requestflow_boxed.go`

---

## Problem

The existing RequestFlow pane has three issues:

1. **All sub-boxes share the same border color.** APP, GATEWAY, and SPOTIFY boxes all
   use `PaneBorderRequestFlow()` (orange). There is no visual cue that these are
   distinct zones with different roles.

2. **The GATEWAY box mixes two unrelated concerns.** Token/semaphore metrics and the
   scrolling decision log live in the same box with no separation. A user cannot
   tell at a glance whether the gateway is healthy — they must parse a list.

3. **The status strip at the bottom is opaque.** `POLLING tick:1000ms active STORE
   stale: playlists(1309s)` uses raw technical labels and values. It is not clear
   what these numbers mean or why they belong in this pane.

---

## Design

### Layout overview

```
╭─ GATEWAY banner (full width, 1 content row) ──────────────────────────────────╮
│  TOKENS ●●●●●●●●○○ 8/10 · SLOTS ■■□□□ 2/5 · BACKOFF none · DEDUP 1 waiting    │
╰───────────────────────────────────────────────────────────────────────────────╯

╭─ APP ──────────────╮  ╭─ GATEWAY LOG ──────────────────────────────╮  ╭─ SPOTIFY ──╮
│ ⚡ PUT /player/vol  │  │ ✓ PUT /player/volume  token→8  slot 1/5    │  │ 200  43ms  │
│ ◷  GET /player     │  │ ⧖ GET /player  dedup joined  ×3            │  │            │
│ ◷  GET /player     │  │ ✗ PUT /player  blocked · backoff 8.2s      │  │ 200 322ms  │
│ ◷  GET /queue      │  │ ⊖ token consumed → 7                       │  │            │
│                    │  │ ⊞ semaphore acquired (2/5)                 │  │            │
│                    │  │ ↻ tokens refilled → 10                     │  │            │
│                    │  │ ✓ GET /player/queue  allowed               │  │            │
╰────────────────────╯  ╰────────────────────────────────────────────╯  ╰────────────╯

╭─ AUTO-TRAFFIC (full width, 1 content row) ────────────────────────────────────╮
│  ▶ playback every 1s · running   ·   ⚠ playlists 21m ago   ·   liked fresh   │
╰───────────────────────────────────────────────────────────────────────────────╯
```

Four distinct zones replace the old three-box layout:

| Zone | Role | Border color |
|---|---|---|
| GATEWAY banner | Gateway health at a glance | `PaneBorderRequestFlow()` (orange) |
| APP box | Live requests the app is sending | `ColumnPrimary()` (blue/violet) |
| GATEWAY LOG box | Rolling event stream of gateway decisions | `PaneBorderRequestFlow()` (orange) |
| SPOTIFY box | HTTP responses from Spotify | `Success()` (green) |
| AUTO-TRAFFIC strip | Explains where background requests come from | `ColumnPrimary()` (blue — same as APP) |

No new theme tokens are added. The mapping is semantically correct: blue = app-side
concerns, orange = gateway engine, green = Spotify responses.

---

### Zone 1: GATEWAY banner

Single full-width box. One content row. Renders the live gateway state snapshot as a
scannable status line.

```
TOKENS  ●●●●●●●●○○  8/10  ·  SLOTS  ■■□□□  2/5  ·  BACKOFF  none  ·  DEDUP  1 waiting
```

**Color rules:**

| Element | Color |
|---|---|
| Filled token dots (●) | `Success()` |
| Empty token dots (○) | `TextMuted()` |
| Filled slot squares (■) | `Warning()` |
| Empty slot squares (□) | `TextMuted()` |
| "BACKOFF none" | `TextMuted()` |
| "BACKOFF 8.2s" (active) | `Error()` |
| "DEDUP N waiting" | `TextSecondary()` |
| Label text (TOKENS, SLOTS, etc.) | `TextSecondary()` |

When backoff is active the entire BACKOFF segment switches to `Error()` so it is
immediately visible without reading the number.

**Height:** border-top + 1 content row + border-bottom = 3 rows.

---

### Zone 2: Three-column area

Occupies the remaining height after the GATEWAY banner (3 rows) and AUTO-TRAFFIC
strip (3 rows) and 2 spacing rows are subtracted.

No per-row arrow columns. The three boxes are placed adjacent with a small gap. The
flow direction (APP → GATEWAY LOG → SPOTIFY) is implied by the left-to-right
reading order and the shared color family (orange gateway = bridge between blue app
and green spotify).

**Column proportions:**

| Column | Width |
|---|---|
| APP | 28% |
| gap | 2% |
| GATEWAY LOG | 42% |
| gap | 3% |
| SPOTIFY | 25% |

Total: 100%. Gaps are plain spaces — no border, no arrow column.
Minimums: APP ≥ 12, GATEWAY LOG ≥ 20, SPOTIFY ≥ 10. Falls back to flat layout
when total minimums exceed pane width.

#### APP box (blue border)

Shows active request animations, newest first. No section header label — the box
title "APP" is sufficient.

**Color rules:**

| Element | Color |
|---|---|
| `⚡` Interactive request (in-flight) | `TextPrimary()` |
| `◷` Background request (in-flight) | `TextMuted()` |
| Any completed request | `TextMuted()` dimmed further |
| Method + path text | inherits from row color |

The `⚡` / `◷` priority marker prefix is preserved. It is the clearest signal for
"user-triggered vs automatic."

#### GATEWAY LOG box (orange border)

Pure event stream — no state metrics. `gatewayStateLines()` is removed from this
box entirely; state is now the GATEWAY banner's job.

Events are rendered newest-first (most recent at the top). Each line is one
`decisionEntry`. Color is determined by event kind:

| Event kind | Color | Rationale |
|---|---|---|
| `EventRequestAllowed` | `Success()` | Good outcome |
| `EventHttpCompleted` (2xx) | `Success()` | Good outcome |
| `EventHttpCompleted` (429) | `Warning()` | Rate limited |
| `EventHttpCompleted` (5xx) | `Error()` | Server error |
| `EventDedupResolved` | `Success()` | Request completed |
| `EventDedupJoined` | `Warning()` | Policy applied, not an error |
| `EventRequestBlocked` | `Error()` | Request rejected |
| `EventBackoffStarted` | `Error()` | Rate limit hit |
| `EventBackoffExpired` | `Success()` | Recovery |
| `EventTokenConsumed` | `TextSecondary()` | Infrastructure detail |
| `EventSemaphoreAcquired` | `TextSecondary()` | Infrastructure detail |
| `EventSemaphoreReleased` | `TextSecondary()` | Infrastructure detail |
| `EventTokenRefilled` | `TextMuted()` | Routine background event |
| `EventRequestEntered` | `TextMuted()` (Background) / `TextPrimary()` (Interactive) | Entry point |

**Label format changes** (cleaner than current):

| Event | New label | Old label |
|---|---|---|
| `EventRequestEntered` | `⚡ PUT /player/vol` or `◷ GET /player` | `→ GET /v1/me/player entered [◷]` |
| `EventRequestAllowed` | `✓ GET /player  allowed` | `✓ GET /v1/me/player allowed` |
| `EventRequestBlocked` | `✗ PUT /player  blocked` | `✗ GET /v1/me/player blocked` |
| `EventDedupJoined` | `⧖ GET /player  dedup joined` | `⧖ GET /v1/me/player dedup` |
| `EventDedupResolved` | `✓ dedup resolved  200` | `✓ dedup resolved 200` |
| `EventHttpCompleted` | `✓ 200  43ms` | `✓ 200 43ms` |
| `EventTokenConsumed` | `⊖ token consumed → 8` | `⊖ token consumed → 8` (unchanged) |
| `EventTokenRefilled` | `↻ tokens refilled → 10` | `↻ tokens refilled → 10` (unchanged) |
| `EventSemaphoreAcquired` | `⊞ semaphore acquired (2/5)` | `⊞ semaphore acquired (2/5)` (unchanged) |
| `EventSemaphoreReleased` | `⊟ semaphore released (1/5)` | `⊟ semaphore released (1/5)` (unchanged) |
| `EventBackoffStarted` | `⏳ backoff started  10s` | `⏳ backoff started 10.0s` |
| `EventBackoffExpired` | `✓ backoff cleared` | `✓ backoff cleared` (unchanged) |

Path is truncated to remove the `/v1/me` prefix for brevity — `/v1/me/player` becomes
`/player`. This keeps lines short without losing meaning since all Spotify paths start
with `/v1/me`.

#### SPOTIFY box (green border)

Shows only requests that actually reached Spotify — requests rejected by the gateway
(blocked, dedup joined) are omitted entirely. This makes the box self-contained: you
do not need to cross-reference the APP or GATEWAY LOG columns to understand what it
shows.

Each line: `[status]  [method] [path]  [latency]`

```
200  GET /player         43ms
200  GET /player/queue  322ms
429  PUT /player/vol      8ms
```

Path uses the same `/v1/me` prefix stripping as the GATEWAY LOG. Method is included
because it distinguishes reads (GET) from writes (PUT/POST/DELETE) — two requests to
the same path with different methods mean different things.

In-flight requests (status not yet known) render as a dim placeholder:

```
···  PUT /player/vol      ···
```

**Color rules:**

| Element | Color |
|---|---|
| Status 2xx | `Success()` |
| Status 429 | `Warning()` |
| Status 5xx | `Error()` |
| In-flight placeholder (`···`) | `TextMuted()` |
| Method (GET/PUT/POST/DELETE) | `TextSecondary()` |
| Path | inherits from status color |
| Latency | `TextSecondary()` |

Because blocked and dedup-joined requests are omitted, the SPOTIFY box may have
fewer rows than the APP or GATEWAY LOG boxes. Empty rows are NOT padded to match —
the box simply shows fewer lines, which itself communicates "fewer requests reached
Spotify than the app sent."

---

### Zone 3: AUTO-TRAFFIC strip

Full-width box. One content row. Replaces the old status strip entirely.

Answers: **"why am I seeing background requests in this pane?"**

```
▶ playback  every 1s · running   ·   ⚠ playlists  21m ago   ·   ⚠ albums  21m ago   ·   liked  fresh   ·   recent  fresh
```

**Polling segment:**

| State | Display | Color |
|---|---|---|
| Active (user not idle) | `▶ playback  every 1s · running` | `Success()` |
| Idle | `⏸ playback  every 1s · idle 32s` | `Warning()` |

The interval is rendered as human-readable: intervals ≥ 1000ms are shown in whole
seconds (`1000ms` → `1s`, `3000ms` → `3s`); intervals < 1000ms are shown as-is in
ms (`500ms` stays `500ms`).

**Cache freshness segments** (one per domain: playlists, albums, liked, recent):

| State | Display | Color |
|---|---|---|
| Zero value (never fetched) | domain name only, `never fetched` | `TextMuted()` |
| Fresh (within TTL) | `liked  fresh` | `TextMuted()` |
| Stale (exceeded TTL, < 1h) | `⚠ playlists  21m ago` | `Warning()` |
| Very stale (≥ 1h) | `⚠ playlists  1h 2m ago` | `Error()` |

Age is rendered in human-readable form: `1309s` → `21m ago`, `3672s` → `1h 1m ago`.
This directly answers "how old is the data?" without requiring mental arithmetic.

Segments are separated by ` · ` in `TextMuted()`. The strip reads left to right:
polling state first (the highest-frequency source of background requests), then
library domains.

**Height:** border-top + 1 content row + border-bottom = 3 rows.

---

### Height budget

```
total pane height
  − 3 rows (GATEWAY banner: top border + content + bottom border)
  − 1 row  (gap between banner and column area)
  − 3 rows (AUTO-TRAFFIC strip)
  − 1 row  (gap between column area and strip)
  = box area height

box area height − 2 (top/bottom border of column boxes) = innerRows
```

Falls back to `viewFlat()` when `innerRows < 1`.

---

## Files changed

| File | Change |
|---|---|
| `internal/ui/panes/requestflow_pane.go` | Refactor `viewBoxed()`: new four-zone layout; add `renderGatewayBanner()`, `renderAutoTrafficStrip()`; remove `renderStatusStrip()`, `renderStoreStatus()`, `renderStalenessStatus()`; update `buildAppBoxLines()` (remove metrics header rows); update `viewFlat()` (remove `renderStatusStrip()` call); update `formatDecisionLabel()` (path truncation `/v1/me` strip, label rewrites) |
| `internal/ui/panes/requestflow_boxed.go` | Update `renderSubBox()` to accept `lipgloss.Color` param instead of hardcoding `PaneBorderRequestFlow()`; remove `gatewayStateLines()` (logic moves to `renderGatewayBanner()` in `requestflow_pane.go`); remove `buildLeftArrowLines()`, `buildRightArrowLines()` |
| `internal/ui/panes/requestflow_pane_test.go` | Update tests: remove status strip tests, add banner and auto-traffic strip tests |
| `internal/ui/panes/requestflow_boxed_test.go` | Update `renderSubBox` call sites (new color param); update label format assertions |

No changes to `requestflow_replay.go`, `domain/`, `state/`, or theme files.

---

## Acceptance criteria

- GATEWAY banner is visible as a single full-width box at the top of the pane.
- Filled token dots render in `Success()`, empty in `TextMuted()`.
- Filled slot squares render in `Warning()`, empty in `TextMuted()`.
- BACKOFF segment renders in `Error()` when throttled, `TextMuted()` when clear.
- APP box border uses `ColumnPrimary()`.
- GATEWAY LOG box border uses `PaneBorderRequestFlow()`.
- SPOTIFY box border uses `Success()`.
- GATEWAY LOG shows only the decision log — no token/semaphore metric bars.
- `EventRequestBlocked` lines render in `Error()`.
- `EventDedupJoined` lines render in `Warning()`.
- `EventRequestAllowed` and `EventHttpCompleted` (2xx) lines render in `Success()`.
- `EventHttpCompleted` (429) renders in `Warning()`, (5xx) in `Error()`.
- `EventTokenConsumed`, `EventSemaphoreAcquired/Released` render in `TextSecondary()`.
- Path strings in the log are truncated to strip the `/v1/me` prefix.
- SPOTIFY box shows `[status]  [method] [path]  [latency]` per request.
- SPOTIFY box omits blocked and dedup-joined requests entirely (no empty placeholder rows).
- SPOTIFY box in-flight requests render as `···  [method] [path]  ···` in `TextMuted()`.
- SPOTIFY box method column renders in `TextSecondary()`, path inherits status color.
- AUTO-TRAFFIC strip is visible as a single full-width box at the bottom.
- Polling state shows `▶ playback every Xs · running` (green) or `⏸ … idle Xs` (yellow).
- Stale cache domains show `⚠ domain Xm ago` in `Warning()`; fresh domains show `fresh` in `TextMuted()`.
- Very stale domains (≥ 1h) render in `Error()`.
- Age values are human-readable (`21m ago`, `1h 2m ago`), not raw seconds.
- No status strip at the bottom of the pane.
- `make ci` passes.
