---
title: "Improve: Viewport-based Episode Details scrolling"
feature: 19-player-page-unification
status: open
---

## Background

The Episode Details overlay (story 235) uses manual scroll logic — `scrollY`,
`maxScroll`, `clampScroll()`, and `descLines[start:end]` slicing in `View()`.
This has two problems:

1. **Dead-zone text:** `availableInnerHeight()` computes visible space with a
   fixed overhead subtraction (`o.height - 4`). The `visibleDesc` derived from
   this does not account for actual line wrapping in the header area, so some
   description lines become unreachable — they sit in a dead zone between the
   computed visible range and the actual scroll bounds.
2. **Missing pager keybindings:** No pgup/pgdown, home/end, gg/G, or mouse
   wheel support. Only j/k and arrow keys scroll one line at a time, making
   long descriptions tedious to read.

Replace the manual scroll implementation with `viewport.Model` from
`github.com/charmbracelet/bubbles/viewport` (v1.0.0 already in go.mod). The
viewport handles all scrolling logic, bounds, and standard pager keybindings
natively.

## Design

### Layout

The overlay is split into three vertical zones inside the chrome border:

```
╭─ Episode Details ────────────────────────────────────  ╮
│                                                        │
│  Episode Name (TextPrimary bold)     ← static header   │
│  Show · 3h01m · Jun 17                                 │
│  Published by: Publisher                               │
│                                                        │
│  description text renders here        ← viewport       │
│  filling full inner width             scrollable area  │
│  scrollable via j/k/↑/↓/pgup/pgdn    (~78 cols)        │
│  wraps naturally                                       │
│  ...                                                   │
│                                                        │
╰────────────────────────────────────────────────────────╯
```

- **Header** (episode name, metadata, publisher): rendered as styled lines
  above the viewport. Not scrollable.
- **Viewport**: fills remaining height. Width = `overlayWidth - 2` (inner).
  Height = `overlayHeight - headerLines - 1` (keybar row). Upper bounded at 40
  lines so the overlay doesn't exceed the terminal on very tall screens.
- **Keybar**: single line at the bottom with `Esc close` and scroll percentage
  from `viewport.ScrollPercent()`. Replaces the old manual `maxScroll > 0`
  hint.

### Viewport v1 API

Bubbles v1.0.0 viewport (`github.com/charmbracelet/bubbles/viewport`):

```go
vp := viewport.New(width, height)
vp.SetContent(descriptionText)
vp.YOffset               // scroll position (set manually if needed)
vp.KeyMap                // customize keybindings
vp.MouseWheelEnabled = true
vp.Update(msg)           // handles all key/mouse events
vp.View()                // renders the viewport area
vp.ScrollPercent()       // returns 0.0 – 1.0
```

### Struct changes

```go
type EpisodeDetailsOverlay struct {
    store     state.StateReader
    theme     theme.Theme
    width     int
    height    int
    viewport  viewport.Model   // NEW — replaces scrollY + maxScroll
    // REMOVED: scrollY int, maxScroll int, clampScroll()
}
```

### SetSize

`SetSize(width, height)` stores dimensions and resizes the viewport. The
viewport height is computed as:

```go
viewportHeight := min(height - headerLines - 2 - 1, 40)
```

Where `headerLines` = 3–5 (name + optional meta + optional publisher + blank
separator + optional empty line), `-2` for chrome borders, `-1` for keybar.

### View()

Static content (header + viewport output + keybar) is composed without
measuring scroll bounds — the viewport handles its own overflow internally.

```go
func (o *EpisodeDetailsOverlay) View() string {
    // 1. Build header lines (name, meta, publisher)
    // 2. Build keybar with scroll percent
    // 3. Compose: header + viewport.View() + keybar
    // 4. Wrap in OverlayChrome
}
```

### Update()

Esc/q → `EpisodeDetailsClosedMsg`. All other key/mouse events → `viewport.Update`.

```go
func (o *EpisodeDetailsOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    keyMsg, ok := msg.(tea.KeyMsg)
    if ok {
        if keyMsg.Type == tea.KeyEsc || (keyMsg.Type == tea.KeyRunes && string(keyMsg.Runes) == "q") {
            return o, func() tea.Msg { return EpisodeDetailsClosedMsg{} }
        }
    }
    var cmd tea.Cmd
    o.viewport, cmd = o.viewport.Update(msg)
    return o, cmd
}
```

### Viewport keybindings (provided by bubbles/viewport)

| Key | Action |
|-----|--------|
| j / ↓ | scroll down 1 line |
| k / ↑ | scroll up 1 line |
| PgDn / Ctrl+D | page down |
| PgUp / Ctrl+U | page up |
| Home / g g | jump to top |
| End / G | jump to bottom |
| mouse wheel | scroll |

The default viewport KeyMap also includes `q` and `Esc` → those are intercepted
by the overlay before reaching the viewport.

## Files

### Modify

- `internal/ui/panes/episode_details_overlay.go` — replace manual scroll with viewport
- `internal/ui/panes/episode_details_overlay_test.go` — update tests for viewport behavior

## Acceptance Criteria

- [ ] Long descriptions fully scrollable (no dead zones where text is unreachable)
- [ ] Viewport fills full inner width of overlay (no inner sub-border)
- [ ] Viewport height generously sized up to 40 lines
- [ ] j/k/↑/↓ scroll one line; pgup/pgdn scroll one page; home/end jump to top/bottom
- [ ] Mouse wheel scrolls the description
- [ ] Esc and q still close the overlay
- [ ] Keybar shows scroll percentage from viewport (not manual maxScroll)
- [ ] `scrollY`, `maxScroll`, `clampScroll` removed from struct and code
- [ ] Overlay renders correctly at narrow terminal widths (< 80 cols)
- [ ] `make ci` passes (lint + tests + 80% coverage)

## Tasks

- [ ] Add `viewport` import and `viewport viewport.Model` field to `EpisodeDetailsOverlay`
      - Modify `episode_details_overlay.go`: add `github.com/charmbracelet/bubbles/viewport` import, replace `scrollY int` + `maxScroll int` with `viewport viewport.Model`
      - test: `TestEpisodeDetailsOverlay_ViewportFieldExists`
- [ ] Update `SetSize` to initialize/resize the viewport
      - Modify `episode_details_overlay.go`: compute viewport width = `overlayWidth() - 2`, height = `min(height - headerLines - 2 - 1, 40)`, call `viewport.New(width, height)`. Enable `MouseWheelEnabled = true`.
      - test: `TestEpisodeDetailsOverlay_SetSize_InitializesViewport`, `TestEpisodeDetailsOverlay_SetSize_ResizesViewport`
- [ ] Refactor `View()` to use viewport for description area
      - Modify `episode_details_overlay.go`: build header lines (name, meta, publisher), build keybar with `fmt.Sprintf("%.0f%%", vp.ScrollPercent()*100)`, call `o.viewport.SetContent(desc)`, compose as `header + viewport.View() + keybar`, wrap in `OverlayChrome`
      - test: `TestEpisodeDetailsOverlay_View_UsesViewport`, `TestEpisodeDetailsOverlay_View_ShowsScrollPercent`
- [ ] Refactor `Update()` to delegate to viewport after close-key intercept
      - Modify `episode_details_overlay.go`: intercept Esc/q → close; all other msgs → `o.viewport.Update(msg)`. Remove scrollY/maxScroll/clampScroll logic.
      - test: `TestEpisodeDetailsOverlay_Update_DelegatesToViewport`, `TestEpisodeDetailsOverlay_Update_MouseWheelScrolls`
- [ ] Remove dead code: `clampScroll()`, `overlayWidth()`, `formatDuration` (if unused after refactor)
      - Modify `episode_details_overlay.go`: remove unused methods. Verify `overlayWidth` still needed for chrome width.
      - test: `go build ./...` compiles without removed symbols
- [ ] Update keybar rendering for scroll percentage
      - Modify `episode_details_overlay.go`: replace manual `maxScroll > 0` conditional with viewport percentage
      - test: `TestEpisodeDetailsOverlay_Keybar_ShowsViewportPercent`
- [ ] Update existing tests for new viewport-based rendering
      - Modify `episode_details_overlay_test.go`: update tests that check scroll behavior, remove tests that check scrollY/maxScroll, add viewport-specific assertions
      - test: all existing tests pass after refactor
- [ ] Run `make ci` — all lint, tests, and 80% coverage pass
