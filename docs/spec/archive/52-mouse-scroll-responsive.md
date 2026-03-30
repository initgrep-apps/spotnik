# Feature 52 — Mouse Scroll + Responsive Behavior

> **Feature:** Enable mouse wheel scrolling on any pane without changing focus,
> and update responsive behavior with the new minimum terminal size (120×30).

## Context

The current app does not handle mouse events. Scrolling requires keyboard focus (`j`/`k`)
on the target pane. The minimum terminal size is 100×24.

The new DESIGN.md (§20, §21) specifies:
- Mouse wheel scrolling on any pane without changing focus (btop behavior)
- Hit-test via `LayoutManager.PaneAt(x, y)` to identify pane under cursor
- Minimum terminal size increased to 120×30
- Friendly "needs more space" message when below minimum

**Design reference:** `docs/DESIGN.md` §20 (Mouse Scroll Support), §21 (Responsive Behavior)

**Depends on:** Feature 41 (LayoutManager with PaneAt), Feature 49 (app migration)

---

## Design Diagram

```
Mouse Scroll Flow:
  tea.MouseMsg{Type: MouseWheelUp, X: 45, Y: 12}
    → layout.PaneAt(45, 12) → PanePlaylists
    → panes[PanePlaylists].Update(scrollUpMsg)
    (focus stays on current pane, e.g., NowPlaying)

Minimum Size Check:
  Terminal < 120×30:
  ╭──────────────────────────────────────────╮
  │  Spotnik needs more space                │
  │                                          │
  │  Current:  98 × 25                       │
  │  Required: 120 × 30                      │
  │                                          │
  │  Please resize your terminal and retry.  │
  ╰──────────────────────────────────────────╯
```

---

## Task 1: Enable mouse support at startup

**Problem:** Mouse events are not enabled.

**Fix:**

Add `tea.WithMouseCellMotion()` to the program options in `cmd/root.go`:

```go
p := tea.NewProgram(
    app.New(...),
    tea.WithAltScreen(),
    tea.WithMouseCellMotion(), // NEW
)
```

**Files:**
- Modify: `cmd/root.go`

**Tests:**
- Integration: App starts with mouse support enabled (verify program option)

**Commit:** `feat(app): enable mouse cell motion for scroll support`

---

## Task 2: Handle mouse scroll events

**Problem:** `tea.MouseMsg` events are not handled.

**Fix:**

Add mouse handler in `app.go` Update():

```go
case tea.MouseMsg:
    // Only handle scroll events
    if msg.Action != tea.MouseActionPress {
        break
    }
    switch msg.Button {
    case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
        // Hit-test: which pane is under the cursor?
        targetID := a.layout.PaneAt(msg.X, msg.Y)
        if targetID < 0 {
            break // not over any pane
        }
        target, ok := a.panes[targetID]
        if !ok {
            break
        }

        // Convert mouse scroll to key message for the target pane
        var scrollMsg tea.KeyMsg
        if msg.Button == tea.MouseButtonWheelUp {
            scrollMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
        } else {
            scrollMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
        }

        // Route to target pane WITHOUT changing focus
        updated, cmd := target.Update(scrollMsg)
        a.panes[targetID] = updated.(layout.Pane)
        return a, cmd
    }
```

**Key behavior:**
- Mouse scroll does NOT change focus (matches btop)
- Scroll is converted to j/k key messages for the target pane
- Hit-test uses `PaneAt()` from LayoutManager (Feature 41 Task 6)
- Only handles WheelUp/WheelDown, not clicks or drags

**Files:**
- Modify: `internal/app/app.go`

**Tests:**
- Unit: Mouse wheel up on Playlists pane → Playlists scrolls up, focus unchanged
- Unit: Mouse wheel down on Queue pane → Queue scrolls down, focus unchanged
- Unit: Mouse on header area → no action (PaneAt returns -1)
- Unit: Mouse on status bar → no action
- Unit: Mouse on border between panes → routes to one pane (consistent)
- Unit: Mouse scroll when overlay is open → ignored (overlay captures all input)

**Commit:** `feat(app): mouse wheel scroll on any pane without focus change`

---

## Task 3: Update minimum terminal size

**Problem:** Minimum size check uses old values (100×24).

**Fix:**

Update `renderTooSmall()` in `render.go`:

```go
const (
    minTermWidth  = 120
    minTermHeight = 30
)

func (a *App) buildView() string {
    if a.width > 0 && a.height > 0 && (a.width < minTermWidth || a.height < minTermHeight) {
        return a.renderTooSmall()
    }
    // ...
}

func (a *App) renderTooSmall() string {
    msg := fmt.Sprintf(
        "Spotnik needs more space\n\nCurrent:  %d × %d\nRequired: %d × %d\n\nPlease resize your terminal and retry.",
        a.width, a.height, minTermWidth, minTermHeight,
    )
    // Centered with rounded border
}
```

Also update `cmd/root.go` if there's a startup size check there.

**Files:**
- Modify: `internal/app/render.go`
- Modify: `cmd/root.go` (if startup check exists)

**Tests:**
- Unit: Terminal 119×30 → shows "needs more space" message
- Unit: Terminal 120×29 → shows "needs more space" message
- Unit: Terminal 120×30 → shows normal grid
- Unit: Message shows actual dimensions and required dimensions
- Unit: Message uses rounded border

**Commit:** `feat(app): update minimum terminal size to 120×30`

---

## Task 4: Tests

**Files:**
- Modify: `internal/app/app_test.go`

**Tests:**
- Integration: Mouse scroll lifecycle — scroll pane → verify content scrolled, focus unchanged
- Integration: Mouse scroll during overlay → ignored
- Integration: Resize below minimum → error message, resize above → grid renders
- Integration: Dynamic resize: start at 120×30, shrink to 80×20 → error, grow to 120×30 → grid
- Edge: Mouse at position (0,0) → header area, no action
- Edge: Mouse at last row → status bar area, no action

**Commit:** `test(app): mouse scroll and responsive behavior tests`

---

## Acceptance Criteria

- [ ] `tea.WithMouseCellMotion()` enabled at app startup
- [ ] Mouse wheel up/down scrolls the pane under the cursor
- [ ] Mouse scroll does NOT change keyboard focus
- [ ] `PaneAt()` hit-test correctly identifies pane from mouse coordinates
- [ ] Mouse scroll ignored when overlay is open
- [ ] Minimum terminal size check uses 120×30
- [ ] "Needs more space" message shows current and required dimensions
- [ ] `make ci` passes

---

## Notes

- Mouse click-to-focus is listed as a future enhancement in DESIGN.md §20 and is NOT
  part of this feature. Only scroll is implemented.
- `tea.WithMouseCellMotion()` enables motion tracking within cells. This gives us
  `tea.MouseMsg` for scroll events. It may also produce motion events which we ignore.
- The mouse scroll → j/k conversion is a pragmatic approach. If bubble-table or other
  components handle `tea.MouseMsg` natively, we can forward the raw message instead.
  Check during implementation.
- Future: auto-degrade (hiding low-priority panes when terminal is slightly below optimal)
  is NOT implemented. DESIGN.md §21 explicitly defers this.
