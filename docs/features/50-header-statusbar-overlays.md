# Feature 50 — Header + Status Bar + Overlay Restyle

> **Feature:** Restyle the header bar to btop format, make the status bar global-only
> (pane hints moved to borders), update search/device overlays to use btop-style borders
> and `bubbletea-overlay` for compositing, and reposition toasts to bottom-right.

## Context

The current header shows `spotnik` left-aligned and device indicator right-aligned.
The status bar shows context-sensitive keybinding hints (different per focused pane).
Overlays use manual `lipgloss.Place()` for positioning.

The new DESIGN.md (§12, §13, §14, §15) specifies:
- btop-style header with page indicator, preset name, and global action shortcuts
- Status bar with global-only shortcuts (pane hints live in borders now)
- Search/device overlays with `RenderPaneBorder()` borders
- `bubbletea-overlay` for overlay compositing
- Toast notifications repositioned to bottom-right

**Design reference:** `docs/DESIGN.md` §12 (Notifications), §13 (Search Overlay),
§14 (Device Switcher Overlay), §15 (Global Header & Status Bar)

**Depends on:** Feature 42 (border renderer), Feature 49 (app migration)

---

## Design Diagram

```
Header (DESIGN.md §15):
 spotnik ─ Page A ─ ᐅp preset 0 ─ ᐅ/ search ─ ᐅd devices ──────── ◉ iPhone

Status Bar (DESIGN.md §15):
 /search   0 page   p preset   1-8 toggle   Tab pane   d devices   ? help   q quit

Search Overlay (DESIGN.md §13):
╭─ Search ────────────────────────── ᐅEnter play ─ ᐅTab section ╮
│  > blinding lig█                                              │
│  ──────────────────────────────────────────────────────────── │
│  TRACKS                                                       │
│  ▶ Blinding Lights          The Weeknd         3:22           │
│    Blinding Lights (Remix)  Sunday Service     4:15           │
│  ARTISTS                                                      │
│    The Weeknd                                                 │
│  ALBUMS                                                       │
│    After Hours              The Weeknd                        │
╰───────────────────────────────────────────────────────────────╯

Device Overlay (DESIGN.md §14):
╭─ Devices ──────────────────────────── ᐅEnter select ╮
│  ◉  MacBook Pro Speakers     [active]               │
│  ○  iPhone 14                                       │
│  ○  Kitchen Speaker                                 │
╰─────────────────────────────────────────────────────╯

Toast (bottom-right):
                              ╭─────────────────────────────╮
                              │  ✓ Added to queue: Starboy  │
                              ╰─────────────────────────────╯
```

---

## Task 1: Add bubbletea-overlay dependency

**Problem:** `github.com/rmhubbert/bubbletea-overlay` is not in go.mod.

**Fix:**

1. Run `go get github.com/rmhubbert/bubbletea-overlay`
2. Run `go mod tidy`

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

**Tests:**
- Build: `go build ./...` succeeds

**Commit:** `chore(deps): add bubbletea-overlay dependency`

---

## Task 2: Restyle header bar

**Problem:** Header shows minimal info without btop-style formatting.

**Fix:**

Rewrite `renderHeader()` in `render.go`:

```go
func (a *App) renderHeader() string {
    // Left side: app name + page + preset + global actions
    appName := styled("spotnik", bold, TextPrimary)
    page := fmt.Sprintf("Page %s", pageLabel(a.layout.ActivePage()))
    preset := fmt.Sprintf("ᐅp preset %d", a.layout.ActivePresetIndex())
    search := "ᐅ/ search"
    devices := "ᐅd devices"

    left := join(" ─ ", appName, page, preset, search, devices)

    // Right side: device indicator
    device := a.store.ActiveDevice()
    var right string
    if device != nil {
        right = fmt.Sprintf("◉ %s", truncateDeviceName(device.Name))
    } else {
        right = "○ No device"
    }

    // Fill middle with ─
    return fillBetween(left, right, a.width, "─")
}
```

- Key labels (`p`, `/`, `d`) in `KeyHint()` color
- Descriptions in `StatusBarFg()` color
- Page and preset labels in `PresetIndicator()` color
- `ᐅ` prefix for action shortcuts (matching border style)
- Background: `SurfaceAlt()` or `StatusBarBg()`

**Files:**
- Modify: `internal/app/render.go`

**Tests:**
- Unit: Header contains "spotnik"
- Unit: Header shows current page (A or B)
- Unit: Header shows current preset index
- Unit: Header shows device name when active
- Unit: Header shows "○ No device" when no device
- Unit: Header fits exactly terminal width (no overflow, no underflow)

**Commit:** `feat(app): btop-style header with page/preset/actions`

---

## Task 3: Restyle status bar to global-only

**Problem:** Status bar shows context-sensitive hints; pane hints now live in borders.

**Fix:**

Replace all status bar hint methods (`mainHints()`, `statsHints()`, `playlistsHints()`)
with a single global hint set:

```go
func (a *App) renderStatusBar() string {
    hints := []struct{ Key, Label string }{
        {"/", "search"}, {"0", "page"}, {"p", "preset"},
        {"1-8", "toggle"}, {"Tab", "pane"}, {"d", "devices"},
        {"?", "help"}, {"q", "quit"},
    }
    // Render: key in KeyHint(), label in StatusBarFg()
    // Background: StatusBarBg()
}
```

Remove: `mainHints()`, `statsHints()`, `playlistsHints()`, `renderStatusBar(hints)` parameter.

**Files:**
- Modify: `internal/app/render.go`

**Tests:**
- Unit: Status bar contains all global shortcuts
- Unit: Status bar does NOT contain pane-specific hints (filter, etc.)
- Unit: Status bar fits terminal width

**Commit:** `feat(app): global-only status bar (pane hints in borders)`

---

## Task 4: Update search overlay with btop borders + bubbletea-overlay

**Problem:** Search overlay uses plain lipgloss.Place() and has no btop-style border.

**Fix:**

1. Update `SearchOverlay.View()` to use `RenderPaneBorder()`:
   ```go
   cfg := layout.BorderConfig{
       Width:       overlayWidth,
       Height:      overlayHeight,
       Title:       "Search",
       ToggleKey:   0,  // no toggle key
       Actions:     []layout.Action{{Key: "Enter", Label: "play"}, {Key: "Tab", Label: "section"}},
       AccentColor: theme.ActiveBorder(),
       Focused:     true,  // overlays are always focused
       Theme:       theme,
   }
   ```

2. Replace `lipgloss.Place()` compositing with `overlay.Composite()`:
   ```go
   import btoverlay "github.com/rmhubbert/bubbletea-overlay"

   func (a *App) renderWithSearchOverlay(background string) string {
       fg := a.searchPane.View()  // already has btop border
       dimmed := lipgloss.NewStyle().Faint(true).Render(background)
       return btoverlay.Composite(fg, dimmed,
           btoverlay.Center, btoverlay.Center, 0, 0)
   }
   ```

**Files:**
- Modify: `internal/ui/panes/search.go`
- Modify: `internal/app/render.go`

**Tests:**
- Unit: Search overlay has btop-style border with title and actions
- Unit: Search overlay centered on screen
- Unit: Background is dimmed when overlay is open
- Unit: Overlay compositing produces valid output

**Commit:** `feat(ui): search overlay with btop border and bubbletea-overlay`

---

## Task 5: Update device overlay with btop borders + bubbletea-overlay

**Problem:** Device overlay uses plain border and manual lipgloss.Place().

**Fix:**

Same pattern as search:

1. Update `DeviceOverlay.View()` with `RenderPaneBorder()`:
   - Title: "Devices"
   - Actions: `[{Key: "Enter", Label: "select"}]`
   - AccentColor: `DeviceActive()` or `ActiveBorder()`

2. Replace `lipgloss.Place()` with `overlay.Composite()`:
   - Position: `btoverlay.Right, btoverlay.Top`

**Files:**
- Modify: `internal/ui/panes/devices.go`
- Modify: `internal/app/render.go`

**Tests:**
- Unit: Device overlay has btop-style border
- Unit: Device overlay positioned top-right
- Unit: Active device shows `◉`, inactive shows `○`

**Commit:** `feat(ui): device overlay with btop border and bubbletea-overlay`

---

## Task 6: Reposition toast notifications to bottom-right

**Problem:** Toasts render at default position (top-left). DESIGN.md §12 specifies bottom-right.

**Fix:**

Update notification initialization in `components/notifications.go`:

```go
// If bubbleup supports position:
alert = alert.WithPosition(bubbleup.BottomRightPosition)
```

If `bubbleup` doesn't support `BottomRightPosition`, manually reposition in `View()`:
1. Call `alerts.Render(content)` to get the composed view
2. If a toast is active, use `lipgloss.Place()` to reposition to bottom-right

**Files:**
- Modify: `internal/ui/components/notifications.go`
- Modify: `internal/app/render.go` (if manual repositioning needed)

**Tests:**
- Unit: Toast appears in bottom-right area of output
- Unit: Toast doesn't interfere with grid content
- Unit: Toast auto-dismisses after 4 seconds

**Commit:** `feat(ui): reposition toasts to bottom-right`

---

## Acceptance Criteria

- [ ] `bubbletea-overlay` dependency added
- [ ] Header shows: spotnik, page, preset index, global action shortcuts, device
- [ ] Status bar shows global-only shortcuts (no pane-specific hints)
- [ ] Search overlay uses `RenderPaneBorder()` with btop-style border
- [ ] Device overlay uses `RenderPaneBorder()` with btop-style border
- [ ] Both overlays use `bubbletea-overlay` for compositing
- [ ] Search overlay centered, device overlay top-right
- [ ] Toast notifications positioned bottom-right
- [ ] All `ᐅ` action prefixes consistent across header, borders, overlays
- [ ] `make ci` passes

---

## Notes

- The `?` help overlay is mentioned in the keybinding table but is a future enhancement.
  For now, `?` key is reserved but not implemented. The status bar lists it for discoverability.
- bubbletea-overlay's `Composite()` handles ANSI escape sequences correctly during
  string-level compositing. This is more robust than the current manual `lipgloss.Place()`.
- The header rendering uses `ᐅ` prefix for consistency with pane borders. The key character
  is rendered in `KeyHint()` color and the label in dimmed color.
