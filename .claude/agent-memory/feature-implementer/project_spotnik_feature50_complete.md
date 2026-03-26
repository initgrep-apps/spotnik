---
name: project_spotnik_feature50_complete
description: Feature 50 (Header + Status Bar + Overlay Restyle): btop header, global status bar, RenderPaneBorder overlays, BottomRight toasts, bubbletea-overlay compositing
type: project
---

## Feature 50 — Header + Status Bar + Overlay Restyle

**What was built:**
- Btop-style header bar: `spotnik ─ Page A ─ ᐅp preset 0 ─ ᐅ/ search ─ ᐅd devices  [gap]  ◉ DeviceName`
- Global-only status bar (removed pane-specific hints — they live in pane borders)
- Search overlay: `RenderPaneBorder()` borders + `btoverlay.Composite(Center, Center)` compositing
- Device overlay: `RenderPaneBorder()` borders + `btoverlay.Composite(Right, Top)` compositing
- Toast notifications repositioned to bottom-right via `bubbleup.BottomRightPosition`
- Added `github.com/rmhubbert/bubbletea-overlay v0.6.6` direct dependency

**Key files:**
- `internal/app/render.go` — renderHeader(), renderStatusBar(), renderWithSearchOverlay(), renderWithDeviceOverlay(), pageLabel(), truncateDeviceName()
- `internal/ui/panes/search.go` — View() now uses layout.RenderPaneBorder(); overlayHeight() method added
- `internal/ui/panes/devices.go` — View() now uses layout.RenderPaneBorder(); overlayWidth() method added
- `internal/ui/components/notifications.go` — WithPosition(BottomRightPosition) pattern

**Patterns established:**
- `btoverlay.Composite(fg, dimmed, xPos, yPos, xOff, yOff)` is string-level compositing — does NOT add borders; use RenderPaneBorder() separately for the border
- `bubbleup.WithPosition()` returns an IMMUTABLE copy (value type) — must take address: `positioned := model.WithPosition(...); return &positioned`
- When `a.width <= 0` (unit test scenario), overlay fallback is `dimmed + "\n" + fg`
- `btoverlay.Right, btoverlay.Top` for device overlay (top-right corner)
- `btoverlay.Center, btoverlay.Center` for search overlay (centered)

**Gotchas:**
- `go mod tidy` promotes bubbletea-overlay from indirect to direct after first import — tidy-check in CI catches this; must commit tidy result separately (or in same commit)
- `gofmt` must be run on render_test.go — CI fmt-check is strict
- `bubbleup.AlertModel` is a value type — `WithPosition()` returns a copy; `return model` (pointer) after `positioned := model.WithPosition(...)` requires `return &positioned`
- DeviceOverlay height: must count lines AFTER lipgloss width-constraining (wrapping) to get correct totalHeight — use `strings.Split(renderedInner, "\n")` after calling `lipgloss.Render()`
- Old `gridHints()` method and hints parameters on renderHeader/renderStatusBar fully removed

**Testing notes:**
- `stripANSI(s)` helper in render_test.go strips ANSI codes for visual width assertions
- `TestRenderHeader_FitsWidth` uses `lipgloss.Width(stripANSI(result))` — direct `lipgloss.Width(result)` works too since lipgloss strips ANSI internally
- Toast bottom-right test: use 20-line content block, check `alertLine >= len(lines)/2`
- btop border tests: check for `╭`, `╰` corners and title/action strings
- CI coverage: 85.7% across 13 packages
