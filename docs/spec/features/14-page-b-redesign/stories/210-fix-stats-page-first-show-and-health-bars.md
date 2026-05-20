---
title: "Fix: Stats page first-show emptiness + GatewayHealth dot bars"
feature: 14-page-b-redesign
status: done
---

## Background

Two related visual bugs on the Stats page:

**A — First-show emptiness (1-second blank)**
`GatewayLivePane` and `GatewayHealthPane` only drain events on `TickMsg`
(1-second interval). When the user first navigates to the Stats page,
`propagateSizes()` calls `SetSize()` on those panes (transitioning from
`width=0`), but `drainEvents()` has not yet run. The panes render empty
until the next `TickMsg` arrives up to 1 second later. The user sees a brief
blank Stats page on every page switch.

**B — Gateway Health bars use wrong renderer**
`GatewayHealthPane.renderDotBar` was specified (story 175, Task 7) as:

> `renderDotBar(filled, total, filledRole, emptyRole, filledStyle, emptyStyle)`
> — iterates 0..total, renders `filledRole` or `emptyRole` glyph per slot.

The implementation deviates: it uses `uikit.ProgressBar{Width: snap.TokensMax}`
(partial-block bar, 10 or 5 columns wide) instead of per-slot glyphs. Problems:
- Bar is only `TokensMax` (10) or `ConcurrentMax` (5) columns wide — tiny
- Uses fixed `Gradient1()` / `TextMuted()` colors from `ProgressBar` — ignores
  the per-state Warning/TextSecondary colors already computed in the caller
- Renders smooth partial-block characters (████▌░░) not individual slot dots

## Design

### Fix A — `internal/ui/panes/gateway_live_pane.go`

In `SetSize`, drain events eagerly when transitioning from zero to non-zero width:

```go
func (p *GatewayLivePane) SetSize(w, h int) {
    firstShow := p.width == 0 && w > 0
    p.TableBasedPane.SetSize(w, h)
    p.Filter().SetWidth(w)
    p.resizeTable()
    if firstShow {
        p.drainEvents()
        p.buildTableRows()
    }
}
```

### Fix A — `internal/ui/panes/gateway_health_pane.go`

Same pattern in `SetSize`:

```go
func (p *GatewayHealthPane) SetSize(w, h int) {
    firstShow := p.width == 0 && w > 0
    p.width = w
    p.height = h
    if firstShow {
        p.drainEvents()
    }
}
```

### Fix B — `internal/ui/panes/gateway_health_pane.go`

Replace the `renderDotBar` method and its two call sites.

**New signature** (unexported, takes explicit styles + mode):

```go
func renderDotBar(filled, total int, filledRole, emptyRole uikit.GlyphRole,
    filledStyle, emptyStyle lipgloss.Style, mode uikit.GlyphMode) string {
    var sb strings.Builder
    for i := 0; i < total; i++ {
        if i < filled {
            sb.WriteString(filledStyle.Render(uikit.GlyphFor(filledRole, mode)))
        } else {
            sb.WriteString(emptyStyle.Render(uikit.GlyphFor(emptyRole, mode)))
        }
    }
    return sb.String()
}
```

**Token row call site** (inside `View()`):

```go
tokenBar := renderDotBar(
    snap.TokensAvailable, snap.TokensMax,
    uikit.GlyphFilledDot, uikit.GlyphEmptySquare,
    tokenStyle, mutedStyle, mode,
)
```

**Slot row call site**:

```go
slotBar := renderDotBar(
    snap.ConcurrentActive, snap.ConcurrentMax,
    uikit.GlyphFilledSquare, uikit.GlyphEmptySquare,
    slotStyle, mutedStyle, mode,
)
```

Remove the old `renderDotBar` method (the one taking `progress float64, width int, th theme.Theme`).
Remove the `uikit.ProgressBar` import if no longer used in this file.

## Acceptance Criteria

- [ ] Switching to Stats page shows GatewayLive content immediately (no 1-second blank)
- [ ] Switching to Stats page shows GatewayHealth with current snapshot immediately
- [ ] `GatewayHealthPane.View()` renders per-slot dot/square glyphs for Tokens and Slots rows
- [ ] Token bar uses `tokenStyle` (Warning when ≤ 2 tokens) for filled glyphs
- [ ] Slot bar uses `slotStyle` (Warning when at capacity) for filled glyphs
- [ ] Empty glyphs use `mutedStyle` in both bars
- [ ] `make ci` passes

## Tasks

- [ ] Add eager drain to `GatewayLivePane.SetSize` — `internal/ui/panes/gateway_live_pane.go`
      - test: `go test ./internal/ui/panes/ -run TestGatewayLive -v` → PASS

- [ ] Add eager drain to `GatewayHealthPane.SetSize` — `internal/ui/panes/gateway_health_pane.go`
      - test: `go test ./internal/ui/panes/ -run TestGatewayHealth -v` → PASS

- [ ] Replace `renderDotBar` implementation and both call sites in
      `gateway_health_pane.go`; remove `uikit.ProgressBar` usage from this file
      - test: `go test ./internal/ui/panes/ -run TestGatewayHealth_View -v` → PASS
        (assert view contains `●` or `■` character and `□` empty glyph)

- [ ] `make ci` passes
