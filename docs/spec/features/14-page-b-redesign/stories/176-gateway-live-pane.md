---
title: "GatewayLivePane"
feature: 14-page-b-redesign
status: done
---

## Background

`GatewayLivePane` (toggle key 4) replaces the event-stream portion of `RequestFlowPane`
with a focused, scrollable, filterable reverse-chronological display. New events are
prepended to a 500-entry buffer on each tick. A committed filter (`activeQuery`) — set by
pressing `Enter` in filter mode — narrows the visible rows; the query appears in the pane
border as `filter(query)`. Pressing `Esc` when no filter input is open first clears the
committed query (if any), then resets scroll on the second press.

**Source:** `docs/superpowers/plans/2026-04-26-page-b-redesign.md` Task 9.
**Design record:** `docs/superpowers/specs/2026-04-26-page-b-redesign-design.md`
§GatewayLivePane.

**Depends on:** Story 175 (GatewayHealthPane and PollingTrafficPane).

---

## Design

### Task 9 — Create GatewayLivePane

**Files to create:** `internal/ui/panes/gateway_live_pane.go`,
`internal/ui/panes/gateway_live_pane_test.go`.

**Struct fields:**
```go
type GatewayLivePane struct {
    store       state.StateReader
    theme       theme.Theme
    focused     bool
    width       int
    height      int
    eventCursor uint64
    buffer      []gatewayLiveRow   // newest-first; capped at 500
    table       *components.Table
    filter      *components.Filter
    activeQuery string             // committed filter (Enter-to-apply)
}
```

**Internal row type:**
```go
type gatewayLiveRow struct {
    glyphRole   uikit.GlyphRole
    intent      uikit.Role
    label       string  // "HH:MM:SS  <event description>"
    matchString string  // pre-built for filter matching
}
```

**Constructor:** Single-column table (`Key: "row"`, no header). `components.Filter` in
Enter-to-apply mode.

**Interface assertions:** Implements both `layout.Pane` and `layout.FilterablePane`.

**Tick handling (`drainEvents`):**
1. `store.ReadEventsFrom(eventCursor)` — advance cursor
2. Build new rows via `buildGatewayLiveRow(event)` for each event
3. Reverse the new rows (events arrive chronologically; newest must be first)
4. Prepend to `buffer`; trim to `maxGatewayLiveRows = 500`
5. Call `buildTableRows()` to repopulate the table

**Event-to-row mapping (`buildGatewayLiveRow`):**

| Kind | GlyphRole | Role | Label template |
|------|-----------|------|----------------|
| `EventRequestEntered` (background) | `GlyphDeadline` | `RoleMuted` | `"HH:MM:SS  METHOD /path"` |
| `EventRequestEntered` (interactive) | `GlyphRunning` | `RolePlain` | `"HH:MM:SS  METHOD /path"` |
| `EventTokenConsumed` | `GlyphWarning` | `RoleWarning` | `"HH:MM:SS  Token consumed → N"` |
| `EventTokenRefilled` | `GlyphRepeatAll` | `RoleSuccess` | `"HH:MM:SS  Tokens refilled → N"` |
| `EventSemaphoreAcquired` | `GlyphFilledSquare` | `RoleInfo` | `"HH:MM:SS  Semaphore acquired  A/M"` |
| `EventSemaphoreReleased` | `GlyphEmptySquare` | `RoleMuted` | `"HH:MM:SS  Semaphore released  A/M"` |
| `EventRequestAllowed` | `GlyphSuccess` | `RoleSuccess` | `"HH:MM:SS  METHOD /path  allowed"` |
| `EventRequestBlocked` | `GlyphError` | `RoleError` | `"HH:MM:SS  METHOD /path  blocked"` |
| `EventDedupJoined` | `GlyphRateLimit` | `RoleInfo` | `"HH:MM:SS  METHOD /path  dedup joined"` |
| `EventDedupResolved` | `GlyphSuccess` | `RoleSuccess` | `"HH:MM:SS  Dedup resolved  N"` |
| `EventBackoffStarted` | `GlyphBlocked` | `RoleError` | `"HH:MM:SS  Backoff started  (retry in X.Xs)"` |
| `EventHttpCompleted` | `GlyphSuccess` | `RoleSuccess` | `"HH:MM:SS  STATUS  Xms"` |
| `EventBackoffExpired` | — | — | Not displayed; `buildGatewayLiveRow` returns `(zero, false)` |

Path strip: `strings.TrimPrefix(e.Path, "/v1/me")` for shorter labels.

**Filter key sequence (Enter-to-apply):**
1. `f` (key rune, filter not active) → `filter.Toggle()`, unfocus table, resize
2. While filter active: characters → `filter.Update(msg)`, `Enter` → commit `activeQuery = filter.Query()`, `filter.Toggle()`, refocus table, rebuild rows; `Esc` → cancel without committing
3. `Esc` (filter not active, `activeQuery != ""`) → clear `activeQuery`, rebuild rows
4. `Esc` (filter not active, `activeQuery == ""`) → `table.GotoTop()`

**`Actions()`** — returns `{Key: "Esc", Label: "cancel"}` when filter is active; `{Key: "f", Label: "filter"}` otherwise.

**`View()`** — `FilterQuery: p.activeQuery` in `uikit.PaneChrome` fields renders `filter(query)` in the border. When filter input is open, prepend `filter.View(width)` above the table content.

**`buildTableRows()`** — applies `strings.Contains(lower(row.matchString), lower(activeQuery))` for committed-query filtering (not `filter.MatchesAny` — that reads the live query which is empty after `filter.Toggle()`). Rows rendered via `uikit.ListRow{Glyph, Label, Intent, Theme}.Render(width - 2)`.

**`SetTheme()`** — rebuilds table via `components.RebuildTableTheme(th, cols, p.table.Rows(), p.focused && !p.filter.IsActive())`.

**White-box test accessors:**
```go
func (p *GatewayLivePane) BufferedEventCount() int { return len(p.buffer) }
func (p *GatewayLivePane) TableCurrentPage() int   { return p.table.CurrentPage() }
```

**Tests:**
- `_ImplementsLayoutPane`, `_ID`, `_Title`, `_ToggleKey` (4), `_View_EmptyBeforeResize`
- `_Update_DrainsCursorOnTick` — RecordEvent then TickMsg, assert `BufferedEventCount() == 1`
- `_Buffer_CapsAt500` — emit 510 events, assert `BufferedEventCount() <= 500`
- `_Esc_ResetsScrollWhenFilterInactive` — 60 events, scroll 8 rows, Esc, assert `TableCurrentPage() == 1`
- `_HasActiveFilter` — assert false initially, send `f` key, assert true

---

## Acceptance Criteria

- [ ] `GatewayLivePane` implements `layout.Pane` and `layout.FilterablePane`
- [ ] `ID()` = `PaneGatewayLive`; `ToggleKey()` = 4
- [ ] Buffer caps at 500 entries; new events prepend (reverse-chronological)
- [ ] `EventBackoffExpired` is silently skipped (returns `false` from `buildGatewayLiveRow`)
- [ ] `f` opens filter; `Enter` commits query; `Esc` cancels without committing
- [ ] `filter(query)` appears in pane border when `activeQuery != ""`
- [ ] `Esc` with committed filter clears query first; second `Esc` resets scroll
- [ ] `Esc` with no committed filter resets scroll directly
- [ ] `SetTheme()` uses `p.focused && !p.filter.IsActive()` focus condition
- [ ] All specified tests pass; `make ci` passes

## Tasks

- [ ] Create `gateway_live_pane.go` with `GatewayLivePane` struct, constructor, and all methods
      - test: compile + `go test ./internal/ui/panes/... -run TestGatewayLive -v` — all pass
- [ ] Create `gateway_live_pane_test.go` with all 9 specified tests
      - test: all 9 tests green
- [ ] Verify full panes suite for regressions
      - test: `go test ./internal/ui/panes/... -v` all green; `make ci` passes
