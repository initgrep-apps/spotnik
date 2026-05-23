---
title: "NowPlaying: single-formula layout, remove tier system"
feature: 17-album-art
status: done
---

## Background

The 3-tier layout system (base/mid/full) introduced in story 217 has wrong
internal proportions: the info box takes ~40% of remaining width (too wide) and
the visualizer gets ~60% (too narrow). The image also grows unboundedly with pane
height via per-tier formulas.

The fix is a single layout formula that applies at every pane size:

```
x       = pane height − 2          (visual rows; includes 1-row pad top + bottom)
artCols = x * 2                    (visually square: terminal chars ≈ 2:1 h:w)
info    = x * 2                    (same block width as image)
viz     = contentWidth − artCols − info − 2*gap
```

When `viz` falls below `npMinViz=10`, the info panel is dropped (2-col: image +
viz). When even that is too narrow, only viz is shown (1-col). This replaces
`renderTier()`, `renderFull()`, `buildInfoLinesFull()`, the `renderTier` type,
and all tier constants.

## Design

### Named constants (top of `nowplaying.go`)

```go
const (
    npPadV      = 1  // rows of vertical padding top + bottom
    npArtAspect = 2  // imageCols = imageRows * npArtAspect
    npInfoMult  = 2  // infoWidth = imageRows * npInfoMult
    npGap       = 1  // column gap between components
    npMinViz    = 10 // minimum viz width; below this, drop info panel
)
```

### `imageRows()` — simplified

```go
func (p *NowPlayingPane) imageRows() int {
    return paneMax(p.height-2*npPadV, 4)
}
```

No tier branching. `imageCols = imageRows * npArtAspect`.

### `SetSize()` — single block, no switch

```go
func (p *NowPlayingPane) SetSize(width, height int) {
    prevRows := p.imageRows()
    p.BasePane.SetSize(width, height)

    cw := p.contentWidth()
    x := p.imageRows()
    artCols := x * npArtAspect
    info := x * npInfoMult
    viz := cw - artCols - info - 2*npGap

    if viz < npMinViz {
        info = 0
        viz = paneMax(cw-artCols-npGap, npMinViz)
    }

    if info > 0 {
        p.infoBox.SetSize(info, x)
    }
    p.engine.SetSize(viz, paneMax(x-1, 1))
    p.seekBar.SetWidth(viz)
    p.volumeBar.SetWidth(paneMax(viz-4, 1))

    p.pendingArtRefresh = abs(p.imageRows()-prevRows) > artResizeThreshold
}
```

`infoBox.SetSize` is only called when `info > 0` to avoid a zero-dimension panic
in the Bubbles component.

### `View()` — simplified dispatch

```go
func (p *NowPlayingPane) View() string {
    ps := p.store.PlaybackState()
    if ps == nil || ps.Item == nil {
        return p.renderEmpty()
    }
    if !p.artRenderer.HasImage() && !p.artRenderer.IsLoading() {
        return p.renderFallback()
    }
    return p.renderBase()
}
```

### `renderBase()` — handles 3-col and 2-col

```go
// renderBase renders using the single-formula layout.
// 3-col (image | info | viz) when width allows; 2-col (image | viz) otherwise.
func (p *NowPlayingPane) renderBase() string {
    ps := p.store.PlaybackState()
    if ps == nil || ps.Item == nil {
        return p.renderEmpty()
    }
    t := ps.Item
    bh := p.bodyHeight()
    cw := p.contentWidth()
    x := p.imageRows()
    artCols := x * npArtAspect
    info := x * npInfoMult
    viz := cw - artCols - info - 2*npGap

    imageBlock := p.renderImageBlock(x, artCols)
    frame := p.engine.CurrentFrame()
    topRows, bottomRows := splitFrame(frame)
    seekBar := p.seekBar.Render(p.localProgressMs, t.DurationMs)
    rightPanel := lipgloss.JoinVertical(lipgloss.Left,
        renderStyledLines(topRows), seekBar, renderStyledLines(bottomRows))

    var composite string
    if viz >= npMinViz {
        infoLines := p.buildInfoLinesBase(bh)
        infoView := p.infoBox.Render("Track Info", infoLines, p.focused)
        composite = lipgloss.JoinHorizontal(lipgloss.Top,
            imageBlock, " ", infoView, " ", rightPanel)
    } else {
        composite = lipgloss.JoinHorizontal(lipgloss.Top,
            imageBlock, " ", rightPanel)
    }

    // 1 blank row top, 1 blank row bottom.
    contentH := lipgloss.Height(composite)
    if contentH < p.height {
        pad := p.height - contentH
        topPad := 1
        bottomPad := pad - topPad
        if bottomPad > 1 {
            bottomPad = 1
        }
        if bottomPad < 0 {
            bottomPad = 0
            if topPad > 1 {
                topPad = 1
            }
        }
        composite = strings.Repeat("\n", topPad) + composite
        if bottomPad > 0 {
            composite += strings.Repeat("\n", bottomPad)
        }
    }
    return composite
}
```

### Deletions

Remove from `nowplaying.go`:
- `renderFull()` function
- `buildInfoLinesFull()` function
- `type renderTier int` and `const ( tierBase ... tierFull )` block
- `func (p *NowPlayingPane) renderTier() renderTier` method

Update `NowPlayingPane` struct doc comment to remove tier references:

```go
// NowPlayingPane is the center pane Bubble Tea model.
// It renders the currently playing track, album art, and visualizer.
// Layout: image (x×2x) | track info (x×2x) | viz (remaining), where x = pane height − 2.
// When width is insufficient for 3-col, the track info panel is dropped.
// Falls back to pre-art layout when no album art is loaded.
```

### Fallback ladder summary

| Condition | Layout |
|-----------|--------|
| `viz ≥ npMinViz` | 3-col: image + info + viz |
| `cw − artCols − npGap ≥ npMinViz` | 2-col: image + viz (info dropped) |
| otherwise | 1-col: viz only |

### Size examples at common terminals

| `SetSize(w, h)` | x | artCols | info | viz | Layout |
|-----------------|---|---------|------|-----|--------|
| (160, 16) | 14 | 28 | 28 | 100 | 3-col |
| (120, 20) | 18 | 36 | 36 | 44 | 3-col |
| (80, 24) | 22 | 44 | 44 | −12 → 33 | 2-col |

## Acceptance Criteria

- [ ] Named constants `npPadV`, `npArtAspect`, `npInfoMult`, `npGap`, `npMinViz` defined
- [ ] `imageRows()` returns `paneMax(height−2, 4)` with no tier branching
- [ ] `SetSize()` uses the single-formula block (no `switch renderTier()`)
- [ ] `infoBox.SetSize` is not called when `info == 0`
- [ ] `View()` dispatches only to `renderBase()` (when art available), `renderFallback()`, or `renderEmpty()`
- [ ] At `SetSize(120, 20)`: 3-col layout — InfoBox borders and track name visible in `View()` output
- [ ] At `SetSize(80, 24)`: 2-col layout — no InfoBox borders in `View()` output; visualizer still present
- [ ] `renderFull()`, `buildInfoLinesFull()`, `renderTier` type and constants, `renderTier()` method all deleted
- [ ] Existing tests updated: `TestNowPlayingPane_RenderTier` and `TestNowPlayingPane_View_FullTier` removed; affected `SetSize(80, 24)` calls changed to `SetSize(120, 20)`
- [ ] `make ci` passes

## Tasks

- [ ] Add new layout tests to `internal/ui/panes/nowplaying_test.go` (tests fail first):
      - `TestNowPlayingPane_Layout_ThreeCol_AtWideTerminal`: `SetSize(120, 20)`, art loaded →
        `View()` contains `"╭"`, `"╰"`, track name, artist name (InfoBox present)
      - `TestNowPlayingPane_Layout_TwoCol_AtNarrowTerminal`: `SetSize(80, 24)`, art loaded →
        `View()` does not contain `"╭"` (InfoBox absent); output contains braille chars (viz present)
      - `TestNowPlayingPane_Layout_SeekBarVisible`: `SetSize(120, 20)`, progress 30s →
        `View()` contains `"0:30"`
      - Add `makeArtRows(cols, rows int) []string` helper (returns slice of `strings.Repeat("█", cols)`)

- [ ] Add named constants block to `internal/ui/panes/nowplaying.go`

- [ ] Simplify `imageRows()` to `paneMax(p.height-2*npPadV, 4)` — remove tier branching

- [ ] Replace `switch p.renderTier()` in `SetSize()` with the single-formula block;
      guard `infoBox.SetSize` behind `if info > 0`

- [ ] Simplify `View()` to dispatch: `renderEmpty()` / `renderFallback()` / `renderBase()`

- [ ] Rewrite `renderBase()` with the single-formula path (3-col when viz≥npMinViz, else 2-col)

- [ ] Delete `renderFull()`, `buildInfoLinesFull()`, `type renderTier int`,
      tier constants, and `renderTier()` method from `nowplaying.go`

- [ ] Update existing tests in `internal/ui/panes/nowplaying_test.go`:
      - Remove `TestNowPlayingPane_RenderTier` and `TestNowPlayingPane_View_FullTier`
      - Change `pane.SetSize(80, 24)` → `pane.SetSize(120, 20)` in
        `TestNowPlayingPane_SplitLayout_ContainsInfoBoxBorders`,
        `TestNowPlayingPane_FullView_ContainsTrackAndAlbum`,
        `TestNowPlayingPane_SplitLayout_ContainsVolumeInInfoBox`,
        `TestNowPlayingPane_SplitLayout_ContainsControls`
      - Rename `TestNowPlayingPane_View_BaseTier` → `TestNowPlayingPane_View_3Col`;
        update comment to remove "base tier" reference
      - Update `TestNowPlayingPane_WindowSizeMsg_RefetchesArt` comment to remove
        "full tier" reference

- [ ] Run `rtk go test ./internal/ui/panes/... -run "TestNowPlayingPane" -v` — all pass

- [ ] `make ci` passes
