---
title: "Remove album art, overlay InfoBox on full-pane visualizer"
feature: 17-album-art
status: open
---

## Background

The pixterm-based album art added in stories 214–220 turned out to be visual
noise: at typical terminal cell sizes (16–48 columns square), the rendered
cover provides no useful information and competes with the visualizer for
attention.

This story removes album art entirely and reallocates its width to the
visualizer, with the InfoBox repositioned as a true overlay on the left ~25%
of the visualizer background. The NowPlaying pane keeps its dimensions,
responsive behaviour, and surrounding chrome — only the internal composition
changes.

Story 221 must land first: it adds the `Theme.OverlayBackground()` token and
the InfoBox solid-background fill that this story relies on to hide the
visualizer behind the InfoBox interior.

## Design

### Layout formula

```text
cw = contentWidth()    = paneMax(width - 4, 10)
x  = vizRows()         = paneMax(height - 2, 4)   // total overlay rows

infoWidth = paneMax(x * npInfoMult, npInfoMin)    // aspect-ratio driven
            then capped at cw / npMaxInfoPct      // ~25% max
vizWidth  = cw - infoWidth - npGap
vizHeight = paneMax(x - 1, 1)                     // -1 for seek bar row
```

If `vizWidth < npMinViz` after subtraction, drop the InfoBox:
`infoWidth = 0`, `vizWidth = cw`.

### Constants (replace the existing block in `nowplaying.go`)

```go
const (
    npPadV       = 1  // rows of vertical padding top + bottom
    npInfoMult   = 1  // infoWidth = vizRows * npInfoMult
    npInfoMin    = 18 // minimum InfoBox width for readability
    npGap        = 1  // gap between InfoBox and visualizer
    npMinViz     = 10 // minimum viz width; below this, drop InfoBox
    npMaxInfoPct = 4  // cap infoWidth at contentWidth / npMaxInfoPct (~25%)
)
```

`npArtAspect` and `artResizeThreshold` are deleted.

### Composition

The visualizer renders once at `vizWidth` columns. It is then padded on the
left with `(infoWidth + npGap)` columns of empty space. The InfoBox is
composited on top of that leading padding via `lipgloss.JoinHorizontal`. The
InfoBox's solid `OverlayBackground` fill (from story 221) hides the padding
behind it, leaving the visualizer visible only to the right of the gap.

```text
┌─ pane border ─────────────────────────────────────┐
│  Now Playing                                       │
│                                                    │  ← 1 row pad top
│  ╭─ Track Info ─╮  visualizer ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓ │
│  │  Despacito    │  visualizer ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓ │
│  │  Luis Fonsi   │  visualizer ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓ │
│  │  Vida         │  visualizer ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓ │
│  │  ◁ ▶ ▷  ↻ ⇄  │  ─── 0:30 ━━━━━━━━ 3:48 ──────  │  ← seek bar
│  │  vol ████░░░  │  visualizer ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓ │
│  ╰───────────────╯  visualizer ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓ │
│                                                    │  ← 1 row pad bot
└────────────────────────────────────────────────────┘
```

The seek bar row sits inside the visualizer column only — it does not extend
under the InfoBox.

### Deletions

`internal/ui/components/albumart.go` — entire file
`internal/ui/components/albumart_test.go` — entire file
`go.mod` — `github.com/eliukblau/pixterm` dependency
`internal/app/` — `case components.AlbumArtFetchedMsg:` routing branch
  (find via `rtk grep -rn "AlbumArtFetchedMsg" internal/app/`)

`internal/ui/panes/nowplaying.go` — remove:
- `artRenderer components.AlbumArtRenderer` field
- `pendingArtRefresh bool` field
- `imageCols()` method
- `renderImageBlock()` method
- `ArtHasImage()` method (test helper only)
- `npArtAspect`, `artResizeThreshold` constants
- All album art fetch in `Init()`, `Update()` (`case components.AlbumArtFetchedMsg:`
  and `case tea.WindowSizeMsg:`), `handlePlaybackFetched()`
- `renderBase()`, `renderFallback()`, helper `buildInfoLinesBase()` (replaced
  by single `renderOverlay()` + `buildInfoLines()`)

### New code in `nowplaying.go`

`Init()` — no album art fetch:

```go
func (p *NowPlayingPane) Init() tea.Cmd {
    return p.engine.Init()
}
```

`vizRows()` (renamed from `imageRows()`):

```go
func (p *NowPlayingPane) vizRows() int {
    return paneMax(p.height-2*npPadV, 4)
}
```

`handlePlaybackFetched()` — no album art dispatch:

```go
func (p *NowPlayingPane) handlePlaybackFetched(msg PlaybackStateFetchedMsg) (*NowPlayingPane, tea.Cmd) {
    ps := p.store.PlaybackState()
    if ps != nil {
        p.localProgressMs = ps.ProgressMs
        p.engine.SetPlaying(ps.IsPlaying)
        if ps.Device != nil {
            p.volumeBar.SetConfirmed(ps.Device.VolumePercent)
        }
    } else {
        p.localProgressMs = 0
        p.engine.SetPlaying(false)
    }
    return p, nil
}
```

`SetSize()` — overlay layout:

```go
func (p *NowPlayingPane) SetSize(width, height int) {
    p.BasePane.SetSize(width, height)

    cw := p.contentWidth()
    x := p.vizRows()

    infoWidth := paneMax(x*npInfoMult, npInfoMin)
    if cap := cw / npMaxInfoPct; infoWidth > cap {
        infoWidth = cap
    }
    vizWidth := cw - infoWidth - npGap
    if vizWidth < npMinViz {
        infoWidth = 0
        vizWidth = cw
    }

    vizHeight := paneMax(x-1, 1)
    if infoWidth > 0 {
        p.infoBox.SetSize(infoWidth, x)
    }
    p.engine.SetSize(vizWidth, vizHeight)
    p.seekBar.SetWidth(vizWidth)
    p.volumeBar.SetWidth(paneMax(infoWidth-4, 1))
}
```

`View()` and `renderOverlay()` — single render path:

```go
func (p *NowPlayingPane) View() string {
    ps := p.store.PlaybackState()
    if ps == nil || ps.Item == nil {
        return p.renderEmpty()
    }
    return p.renderOverlay()
}

func (p *NowPlayingPane) renderOverlay() string {
    ps := p.store.PlaybackState()
    t := ps.Item
    cw := p.contentWidth()
    x := p.vizRows()

    infoWidth := paneMax(x*npInfoMult, npInfoMin)
    if cap := cw / npMaxInfoPct; infoWidth > cap {
        infoWidth = cap
    }
    vizWidth := cw - infoWidth - npGap
    if vizWidth < npMinViz {
        infoWidth = 0
        vizWidth = cw
    }

    frame := p.engine.CurrentFrame()
    topRows, bottomRows := splitFrame(frame)
    seekBar := p.seekBar.Render(p.localProgressMs, t.DurationMs)
    vizPanel := lipgloss.JoinVertical(lipgloss.Left,
        renderStyledLines(topRows), seekBar, renderStyledLines(bottomRows))

    paddedViz := strings.Repeat(" ", infoWidth+npGap) + vizPanel

    var composite string
    if infoWidth > 0 {
        infoLines := p.buildInfoLines(x)
        infoView := p.infoBox.Render("Track Info", infoLines, p.focused)
        composite = lipgloss.JoinHorizontal(lipgloss.Top, infoView, paddedViz)
    } else {
        composite = vizPanel
    }

    // Equal 1-row top + 1-row bottom padding.
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

`buildInfoLines()`:

```go
func (p *NowPlayingPane) buildInfoLines(bodyHeight int) []string {
    ps := p.store.PlaybackState()
    if ps == nil || ps.Item == nil {
        return nil
    }
    t := ps.Item
    primaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextPrimary()).Bold(true)
    secondaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextSecondary())
    mutedStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())

    artistNames := make([]string, len(t.Artists))
    for i, a := range t.Artists {
        artistNames[i] = a.Name
    }

    ctrl := components.NewControls(p.theme, ps.IsPlaying, ps.ShuffleState, ps.RepeatState)

    lines := []string{
        primaryStyle.Render(t.Name),
        secondaryStyle.Render(strings.Join(artistNames, ", ")),
        mutedStyle.Render(t.Album.Name),
        ctrl.Render(),
        p.volumeBar.Render(),
    }

    innerH := bodyHeight - 2
    for len(lines) < innerH {
        lines = append(lines, "")
    }
    return lines
}
```

### Size examples

| `SetSize(w, h)` | x | infoWidth (raw) | cap (cw/4) | infoWidth (final) | vizWidth | Layout |
|-----------------|---|-----------------|------------|-------------------|----------|--------|
| (160, 16) | 14 | 18 (min) | 39 | 18 | 137 | overlay |
| (120, 20) | 18 | 18 | 29 | 18 | 97 | overlay |
| (80, 24)  | 22 | 22 | 19 | 19 | 56 | overlay |
| (60, 16)  | 14 | 18 (min) | 14 | 14 | viz=42 — but 14+1+42=57, cw=56, so info dropped | viz-only |

## Acceptance Criteria

- [ ] `internal/ui/components/albumart.go` deleted
- [ ] `internal/ui/components/albumart_test.go` deleted
- [ ] `github.com/eliukblau/pixterm` removed from `go.mod` and `go.sum`
- [ ] `rtk grep -rn "pixterm\|eliukblau" --include="*.go" .` returns no matches
- [ ] `rtk grep -rn "AlbumArt\|albumart\|FetchAlbumArtCmd\|AlbumArtFetchedMsg\|AlbumArtRenderer" --include="*.go" .` returns no matches
- [ ] `NowPlayingPane` struct has no `artRenderer`, `pendingArtRefresh` fields
- [ ] `imageCols()`, `renderImageBlock()`, `ArtHasImage()`, `renderBase()`, `renderFallback()`, `buildInfoLinesBase()` methods removed
- [ ] `npArtAspect`, `artResizeThreshold` constants removed; `npInfoMin=18`, `npMaxInfoPct=4` added; `npInfoMult` changed to `1`
- [ ] `Init()` returns only `p.engine.Init()` (no album art fetch)
- [ ] `Update()` has no `case components.AlbumArtFetchedMsg:` or `case tea.WindowSizeMsg:` branch
- [ ] `handlePlaybackFetched()` does not dispatch album art fetch
- [ ] `internal/app/` has no `AlbumArtFetchedMsg` routing
- [ ] `vizRows()` returns `paneMax(p.height-2*npPadV, 4)`
- [ ] `SetSize()` uses the overlay formula; `infoBox.SetSize` only called when `infoWidth > 0`
- [ ] `View()` dispatches to `renderEmpty()` or `renderOverlay()` only
- [ ] At `SetSize(120, 20)`: InfoBox visible on left with track name and artist; seek bar present on right
- [ ] At `SetSize(60, 16)`: InfoBox dropped; visualizer fills full content width
- [ ] First and last content rows of `View()` output are blank (equal 1-row padding)
- [ ] Seek bar `▓`/`░` characters appear only to the right of the InfoBox border `│`, not on a row that intersects the InfoBox interior
- [ ] All new overlay tests pass (see Tasks)
- [ ] All old album-art tests removed from `nowplaying_test.go`
- [ ] `make ci` passes — lint + tests + 80% coverage

## Tasks

- [ ] Verify story 221 is merged to main and the branch is rebased on it
      (`Theme.OverlayBackground()` and InfoBox solid background must exist
      before this story builds)

- [ ] Delete `internal/ui/components/albumart.go` and
      `internal/ui/components/albumart_test.go`

- [ ] Run `go mod tidy`; verify `grep -c pixterm go.mod` returns 0; verify
      `go.sum` updated

- [ ] Find and remove `AlbumArtFetchedMsg` routing in `internal/app/`:
      `rtk grep -rn "AlbumArtFetchedMsg" internal/app/` — delete the
      `case components.AlbumArtFetchedMsg:` block and any newly-unused
      `components` import

- [ ] Commit deletions and routing strip:
      `chore(albumart): remove pixterm dep, delete albumart component`
      `refactor(app): drop AlbumArtFetchedMsg routing`

- [ ] Rewrite `internal/ui/panes/nowplaying.go`:
      - Replace constants block with the new constants (see Design)
      - Drop `artRenderer`, `pendingArtRefresh` fields from struct
      - Delete `imageCols()`, `renderImageBlock()`, `ArtHasImage()`,
        `renderBase()`, `renderFallback()`, `buildInfoLinesBase()`
      - Rewrite `Init()`, `handlePlaybackFetched()`, `SetSize()`, `View()`
      - Add `vizRows()`, `renderOverlay()`, `buildInfoLines()` per Design
      - Remove the `case components.AlbumArtFetchedMsg:` and
        `case tea.WindowSizeMsg:` branches in `Update()`

- [ ] Verify build: `rtk go build ./...` — must pass

- [ ] Commit pane rewrite:
      `feat(nowplaying): overlay InfoBox on full-pane visualizer, drop album art`

- [ ] Update `internal/ui/panes/nowplaying_test.go`:
      - Delete every test referencing `AlbumArt`, `ArtHasImage`, `artRenderer`,
        `WindowSizeMsg_RefetchesArt`, `makeArtRows` (and the helper itself
        if it was only used by removed tests)
      - Delete `TestNowPlayingPane_Layout_ThreeCol_AtWideTerminal`,
        `TestNowPlayingPane_Layout_TwoCol_AtNarrowTerminal` (replaced by
        overlay tests below)
      - Add `TestNowPlayingPane_Overlay_InfoBoxOnLeft` — `SetSize(120, 20)`,
        playback state populated; assert `View()` contains `╭`, track name,
        artist name
      - Add `TestNowPlayingPane_Overlay_SeekBarRightOfInfoBox` —
        `SetSize(120, 20)`, find the seek bar line in stripped output, assert
        the InfoBox right border `│` appears before the first `▓`/`░`
      - Add `TestNowPlayingPane_Overlay_NarrowFallback` — `SetSize(60, 16)`,
        assert `View()` does not contain `╭` or `Track Info` (viz fills width)
      - Add `TestNowPlayingPane_Overlay_EqualMargins` — `SetSize(120, 20)`,
        assert first and last lines of `View()` are blank after `ansi.Strip` +
        `strings.TrimSpace`

- [ ] Run `rtk go test ./internal/ui/panes/... -run TestNowPlayingPane -v` —
      all pass

- [ ] Commit tests:
      `test(nowplaying): cover overlay layout, drop album-art tests`

- [ ] Run `make ci` — all gates pass (lint, tests, 80% coverage)

- [ ] If `go.sum` or other files drift after `go mod tidy`, commit:
      `chore(albumart): final cleanup post pixterm removal`
