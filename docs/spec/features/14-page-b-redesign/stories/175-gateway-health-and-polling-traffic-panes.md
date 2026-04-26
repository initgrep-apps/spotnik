---
title: "GatewayHealthPane and PollingTrafficPane"
feature: 14-page-b-redesign
status: open
---

## Background

Two static (no scroll, no filter) panes replace the upper section of RequestFlowPane.
`GatewayHealthPane` (toggle key 2) renders a 4-row fixed grid driven by gateway events.
`PollingTrafficPane` (toggle key 3) renders a 5-row fixed grid driven by a
`PollingSnapshotMsg` for the playback row and store TTL sentinel methods for the four
library cache rows.

`PollingSnapshotMsg` is currently defined inside `requestflow_pane.go`. It must be moved
to `messages.go` before RequestFlow is deleted — this migration is Task 6.

**Source:** `docs/superpowers/plans/2026-04-26-page-b-redesign.md` Tasks 6, 7, 8.
**Design record:** `docs/superpowers/specs/2026-04-26-page-b-redesign-design.md`
§GatewayHealthPane, §PollingTrafficPane.

**Depends on:** Story 174 (Page B PaneID Constants and Preset Layout).

---

## Design

### Task 6 — Move PollingSnapshotMsg to messages.go

**Files to modify:** `internal/ui/panes/messages.go`,
`internal/ui/panes/requestflow_pane.go`.

Cut the type block from `requestflow_pane.go` and paste into `messages.go`. Update the
doc comment to reference `PollingTrafficPane` instead of `RequestFlowPane`.

```go
// PollingSnapshotMsg carries app-level polling state to the PollingTrafficPane.
type PollingSnapshotMsg struct {
    TickIntervalMs int
    IsIdle         bool
    IdleSecs       int
}
```

Verify: `go build ./internal/ui/panes/...` compiles with no duplicate type error.

### Task 7 — Create GatewayHealthPane

**Files to create:** `internal/ui/panes/gateway_health_pane.go`,
`internal/ui/panes/gateway_health_pane_test.go`.

Struct fields: `store state.StateReader`, `theme theme.Theme`, `focused bool`,
`width int`, `height int`, `eventCursor uint64`, `snapshot domain.GatewayStateSnapshot`.

Constructor default: `snapshot: domain.GatewayStateSnapshot{TokensMax: 10, ConcurrentMax: 5}`.

`Update()` — on `TickMsg`: call `drainEvents()` which reads `store.ReadEventsFrom(cursor)`,
advances `eventCursor`, and sets `snapshot` to the newest event's `Snapshot` field.

`View()` — renders 4 rows (Tokens, Slots, Backoff, Dedup) via `renderRow(icon, label, data)`.
Each row is `icon + "  " + labelStyle.Render(PadOrTruncate(label, 8)) + "  " + data`.
Wrapped in `uikit.PaneChrome{...}.Render(content)`.

Colour rules:
- **Tokens**: `TextSecondary`; switches to `Warning` when `TokensAvailable <= 2`
- **Slots**: `TextSecondary`; switches to `Warning` when `ConcurrentActive >= ConcurrentMax`
- **Backoff**: `TextMuted` / `"none"` when zero; `Error` / `"X.Xs"` when `BackoffRemaining > 0`
- **Dedup**: `TextMuted` / `"none"` when zero; `TextSecondary` / `"N waiters"` when `> 0`

Dot bars: `renderDotBar(filled, total, filledRole, emptyRole, filledStyle, emptyStyle)` —
iterates 0..total, renders `filledRole` or `emptyRole` glyph with the appropriate style.

Tests: `TestGatewayHealthPane_ImplementsLayoutPane`, `_ID`, `_Title`, `_ToggleKey` (2),
`_View_EmptyBeforeResize`, `_View_ContainsHealthRows` (asserts "Tokens", "Slots", "Backoff",
"Dedup" in view), `_Update_DrainsCursor` (RecordEvent then TickMsg, asserts view not empty).

### Task 8 — Create PollingTrafficPane

**Files to create:** `internal/ui/panes/polling_traffic_pane.go`,
`internal/ui/panes/polling_traffic_pane_test.go`.

Struct fields: `store state.StateReader`, `theme theme.Theme`, `focused bool`,
`width int`, `height int`, `pollSnapshot PollingSnapshotMsg`.

`Update()` — stores `PollingSnapshotMsg` when received.

`View()` — renders 5 rows. Label width = 10 chars (padded via `uikit.PadOrTruncate`).

**Playback row** (`GlyphMusicNote` icon):
- Not idle: `GlyphPlaying` glyph in `Success` colour + `"Xs · running"` label
- Idle: `GlyphPaused` glyph in `Warning` colour + `"idle · Xs"` label

`pollingHumanInterval(ms int) string` converts milliseconds: `>= 1000` → `"Xs"`, else `"Xms"`.

**Library cache rows** (Playlists/Albums/Liked/Recent) — read `store.PlaylistsFetchedAt()`,
`store.AlbumsFetchedAt()`, `store.LikedTracksFetchedAt()`, `store.RecentPlayedFetchedAt()`
and compare against `state.PlaylistsTTL`, `state.AlbumsTTL`, `state.LikedTracksTTL`,
`state.RecentlyPlayedTTL` via `state.IsStale(fetchedAt, ttl)`:
- Zero time: `TextMuted` + `"never fetched"`
- Fresh: `GlyphAvailable` in `TextMuted` + `"fresh"`
- Stale < 1h: `GlyphWarning` in `Warning` + `"Xm stale"`
- Stale >= 1h: `GlyphWarning` in `Error` + `"Xh Ym stale"`

`cacheAge(t time.Time) string` — returns bare duration string without "ago" suffix
(`"just now"`, `"Xm"`, `"Xh"`, `"Xh Ym"`). Defined here because `humanAge` in
`requestflow_pane.go` appends "ago" and will be deleted in Story 177.

Tests: `_ImplementsLayoutPane`, `_ID`, `_Title`, `_ToggleKey` (3), `_View_EmptyBeforeResize`,
`_View_ContainsAllRows` (asserts "Playback", "Playlists", "Albums", "Liked", "Recent"),
`_Update_PollingSnapshotMsg` (running state shows "running"),
`_Update_IdleSnapshot` (idle state shows "idle").

---

## Acceptance Criteria

- [ ] `PollingSnapshotMsg` defined once in `messages.go`; removed from `requestflow_pane.go`
- [ ] `GatewayHealthPane` implements `layout.Pane`; `ID()` = `PaneGatewayHealth`; toggle key 2
- [ ] `GatewayHealthPane.View()` renders Tokens, Slots, Backoff, Dedup rows
- [ ] Warning colours apply when tokens ≤ 2, slots at capacity, backoff > 0
- [ ] `GatewayHealthPane.Update(TickMsg)` advances the event cursor and updates the snapshot
- [ ] `PollingTrafficPane` implements `layout.Pane`; `ID()` = `PanePollingTraffic`; toggle key 3
- [ ] `PollingTrafficPane.View()` renders all 5 rows with correct labels
- [ ] Running playback shows green `GlyphPlaying` + interval; idle shows amber `GlyphPaused` + "idle"
- [ ] Library cache rows show fresh/stale/never-fetched states with correct colours
- [ ] `make ci` passes

## Tasks

- [ ] Move `PollingSnapshotMsg` from `requestflow_pane.go` to `messages.go`
      - test: `go build ./internal/ui/panes/...` clean; no duplicate type errors
- [ ] Create `gateway_health_pane.go` and `gateway_health_pane_test.go`
      - test: `go test ./internal/ui/panes/... -run TestGatewayHealth -v` — all pass
- [ ] Create `polling_traffic_pane.go` and `polling_traffic_pane_test.go`
      - test: `go test ./internal/ui/panes/... -run TestPollingTraffic -v` — all pass
- [ ] Run full suite to catch regressions
      - test: `go test ./... -v` all green; `make ci` passes
