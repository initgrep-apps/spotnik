---
name: project_spotnik_feature50_complete
description: Feature 50 (Header + Status Bar + Overlay Restyle): btop header, global status bar, RenderPaneBorder overlays, BottomRight toasts, bubbletea-overlay compositing
type: project
---

## Feature 50 — Header + Status Bar + Overlay Restyle

**Built:**
- Btop header bar: `spotnik ─ Page A ─ ᐅp preset 0 ─ ᐅ/ search ─ ᐅd devices  [gap]  ◉ DeviceName`
- Global-only status bar (pane hints moved to pane borders)
- Search overlay: `RenderPaneBorder()` + `btoverlay.Composite(Center, Center)`
- Device overlay: `RenderPaneBorder()` + `btoverlay.Composite(Right, Top)`
- Toasts moved bottom-right via `bubbleup.BottomRightPosition`
- Added `github.com/rmhubbert/bubbletea-overlay v0.6.6` direct dep

**Key files:**
- `internal/app/render.go` — renderHeader(), renderStatusBar(), renderWithSearchOverlay(), renderWithDeviceOverlay(), pageLabel(), truncateDeviceName()
- `internal/ui/panes/search.go` — View() uses layout.RenderPaneBorder(); added overlayHeight()
- `internal/ui/panes/devices.go` — View() uses layout.RenderPaneBorder(); added overlayWidth()
- `internal/ui/components/notifications.go` — WithPosition(BottomRightPosition) pattern

**Patterns:**
- `btoverlay.Composite(fg, dimmed, xPos, yPos, xOff, yOff)` = string compositing, NO borders; use RenderPaneBorder() for border
- `bubbleup.WithPosition()` returns IMMUTABLE copy (value type) — take address: `positioned := model.WithPosition(...); return &positioned`
- `a.width <= 0` (unit test) → overlay fallback `dimmed + "\n" + fg`
- `btoverlay.Right, btoverlay.Top` → device overlay (top-right)
- `btoverlay.Center, btoverlay.Center` → search overlay (centered)

**Gotchas:**
- `go mod tidy` promotes bubbletea-overlay indirect→direct after first import; CI tidy-check catches; commit tidy result
- Run `gofmt` on render_test.go — CI fmt-check strict
- `bubbleup.AlertModel` value type — `WithPosition()` returns copy; after `positioned := model.WithPosition(...)` need `return &positioned`
- DeviceOverlay height: count lines AFTER lipgloss width-wrap for correct totalHeight — use `strings.Split(renderedInner, "\n")` after `lipgloss.Render()`
- Old `gridHints()` method + hints params on renderHeader/renderStatusBar fully removed

**Testing:**
- `stripANSI(s)` helper in render_test.go strips ANSI for visual width asserts
- `TestRenderHeader_FitsWidth` uses `lipgloss.Width(stripANSI(result))` — direct `lipgloss.Width(result)` works too (lipgloss strips ANSI internally)
- Toast bottom-right test: 20-line content, check `alertLine >= len(lines)/2`
- btop border tests: check `╭`, `╰` corners + title/action strings
- CI coverage: 85.7% across 13 packages