---
title: "Fix: Page B toggle keys — number keys 1–5 toggle pane visibility on Page B"
feature: 10-developer-tools
status: done
---

## Background

On Page A, pressing `1`–`8` toggles individual pane visibility (btop-style). On Page B,
pressing `1`–`5` does nothing. The user expects the same toggle behavior on both pages.

**Root cause 1 — wrong key map.**
`toggleKeyMap` in `internal/app/routing.go` maps `'1'–'8'` to Page A pane IDs
(`PaneNowPlaying`, `PaneQueue`, `PanePlaylists`, `PaneAlbums`, `PaneLikedSongs`,
`PaneRecentlyPlayed`, `PaneTopTracks`, `PaneTopArtists`). When on Page B, pressing `'2'`
maps to `PaneQueue` — a pane not in the Page B preset. `TogglePane` silently does nothing
for a pane not in the current preset.

**Root cause 2 — `TogglePane` hard-returns on Page B.**
`layout.go` line 277: `if m.activePage == PageB { return }` — all toggle calls are
rejected on Page B regardless of the pane ID.

**Expected behavior:**

| Key | Page A | Page B |
|-----|--------|--------|
| `1` | Toggle NowPlaying | Toggle NowPlaying |
| `2` | Toggle Queue | Toggle GatewayHealth |
| `3` | Toggle Playlists | Toggle PollingTraffic |
| `4` | Toggle Albums | Toggle GatewayLive |
| `5` | Toggle LikedSongs | Toggle NetworkLog |
| `6`–`8` | Toggle their Page A panes | No-op (Page B has ≤5 panes) |

The "cannot hide last pane" guard already in `TogglePane` applies to both pages.

---

## Design

### Task 1 — Add `pageBToggleKeyMap` and make routing page-aware

**File:** `internal/app/routing.go`

Add below the existing `toggleKeyMap`:

```go
// pageBToggleKeyMap maps rune keys '1'-'5' to Page B PaneIDs.
var pageBToggleKeyMap = map[rune]layout.PaneID{
    '1': layout.PaneNowPlaying,
    '2': layout.PaneGatewayHealth,
    '3': layout.PanePollingTraffic,
    '4': layout.PaneGatewayLive,
    '5': layout.PaneNetworkLog,
}
```

Update the toggle key routing block (currently around line 196–204):

Current:
```go
if m.Type == tea.KeyRunes && len(m.Runes) == 1 {
    if id, ok := toggleKeyMap[m.Runes[0]]; ok {
        a.layout.TogglePane(id)
        a.propagateSizes()
        a.syncFocus()
        return a, nil
    }
}
```

Replace with:
```go
if m.Type == tea.KeyRunes && len(m.Runes) == 1 {
    keyMap := toggleKeyMap
    if a.layout.ActivePage() == layout.PageB {
        keyMap = pageBToggleKeyMap
    }
    if id, ok := keyMap[m.Runes[0]]; ok {
        a.layout.TogglePane(id)
        a.propagateSizes()
        a.syncFocus()
        return a, nil
    }
}
```

### Task 2 — Remove Page B early-return from `TogglePane`; tighten page guard

**File:** `internal/ui/layout/layout.go`

Current guards (lines 276–284):
```go
// Page B panes are not individually toggleable
if m.activePage == PageB {
    return
}

// Only Page A panes (0-7) are toggleable
if id >= PaneNetworkLog {
    return
}
```

Replace with a single page-aware guard:
```go
// Each page can only toggle its own panes.
if m.activePage == PageA && id >= PaneNetworkLog {
    return
}
if m.activePage == PageB && id < PaneNetworkLog {
    return
}
```

No other changes — the existing "not in preset" check and "cannot hide last pane" check
handle the rest correctly for both pages.

### Task 3 — Tests

**`internal/ui/layout/layout_test.go`** — add `TestTogglePane_PageB_TogglesPageBPanes`:

```go
func TestTogglePane_PageB_TogglesPageBPanes(t *testing.T) {
    m := layout.NewManager()
    m.Resize(200, 50)
    m.TogglePage() // switch to Page B

    // GatewayHealth (key 2) should be toggleable on Page B
    require.True(t, m.IsPaneVisible(layout.PaneGatewayHealth))
    m.TogglePane(layout.PaneGatewayHealth)
    assert.False(t, m.IsPaneVisible(layout.PaneGatewayHealth), "GatewayHealth must hide after toggle")

    m.TogglePane(layout.PaneGatewayHealth)
    assert.True(t, m.IsPaneVisible(layout.PaneGatewayHealth), "GatewayHealth must show after second toggle")
}

func TestTogglePane_PageB_IgnoresPageAPanes(t *testing.T) {
    m := layout.NewManager()
    m.Resize(200, 50)
    m.TogglePage() // switch to Page B

    // Page A panes must not be toggleable while on Page B
    m.TogglePane(layout.PaneQueue) // PaneQueue < PaneNetworkLog — Page A pane
    assert.True(t, m.IsPaneVisible(layout.PaneNowPlaying), "NowPlaying must still be visible")
}

func TestTogglePane_PageA_IgnoresPageBPanes(t *testing.T) {
    m := layout.NewManager()
    m.Resize(200, 50)
    // Still on Page A — attempting to toggle a Page B pane must be a no-op
    m.TogglePane(layout.PaneGatewayHealth)
    // NowPlaying is a Page A pane and must remain visible (no change to Page A state)
    assert.True(t, m.IsPaneVisible(layout.PaneNowPlaying))
}
```

**`internal/app/routing_test.go`** — add `TestPageBNumberKeys_TogglePageBPanes`:

```go
func TestPageBNumberKeys_TogglePageBPanes(t *testing.T) {
    a := newTestApp(t)
    a.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
    // Switch to Page B
    a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("0")})

    // Pressing '2' on Page B must toggle GatewayHealth
    require.True(t, a.Layout().IsPaneVisible(layout.PaneGatewayHealth))
    a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
    assert.False(t, a.Layout().IsPaneVisible(layout.PaneGatewayHealth))
}
```

> Note: `a.Layout()` must be an exported accessor on `App` (it may already exist; if not,
> add `func (a *App) Layout() *layout.Manager { return a.layout }` to `app.go`).

---

## Acceptance Criteria

- [ ] `pageBToggleKeyMap` exists in `routing.go` mapping `'1'-'5'` to Page B pane IDs
- [ ] On Page B, `'2'` toggles `PaneGatewayHealth`, `'3'` toggles `PanePollingTraffic`, `'4'` toggles `PaneGatewayLive`, `'5'` toggles `PaneNetworkLog`
- [ ] `'1'` toggles `NowPlaying` on both pages
- [ ] `TogglePane` on Page B no longer early-returns — Page B pane IDs are accepted
- [ ] `TogglePane` still rejects Page A pane IDs when on Page B (and vice versa)
- [ ] "Cannot hide last pane" guard still works on Page B
- [ ] All three new layout tests pass
- [ ] `TestPageBNumberKeys_TogglePageBPanes` passes
- [ ] Existing `TestFilterActive_NumberKeys_DoNotTogglePanes` still passes
- [ ] `make ci` passes

## Tasks

- [ ] Add `pageBToggleKeyMap` and update routing toggle block in `internal/app/routing.go`
      — test: `TestPageBNumberKeys_TogglePageBPanes`
- [ ] Replace Page B guard in `TogglePane` with page-aware guard in `internal/ui/layout/layout.go`
      — test: 3 × new `TestTogglePane_PageB_*` tests; existing toggle tests still green
- [ ] `make ci` passes
