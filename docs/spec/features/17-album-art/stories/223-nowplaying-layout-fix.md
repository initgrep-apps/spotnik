---
title: "Fix NowPlaying layout: remove overlay background, adaptive width, centering"
feature: 17-album-art
status: open
---

## Background

Stories 221–222 shipped the overlay redesign: InfoBox on the left ~25% of the
visualizer with a solid `OverlayBackground` fill hiding the animation behind it.
Live testing revealed five concrete problems that this story fixes.

| # | Symptom | User observation |
|---|---------|------------------|
| 1 | InfoBox too narrow + phantom dead space | "track info panel is too small" |
| 2 | Solid background adds ANSI noise | "internal background is not required" |
| 3 | Horizontal composition broken | "characters … on right side of now playing" |
| 4 | Compact presets hide controls/volume | "it deserves more width so that all controls are shown" |
| 5 | Solo pane stretches to full terminal height | "it goes full height which doesnt look good" |

## Root-Cause Analysis

### RCA-1: `contentWidth()` double-subtracts border space

`Rect.ContentWidth()` (`internal/ui/layout/pane.go:46`) already returns
`Width - 2` (borderless content area). `app.go:737` passes
`rect.ContentWidth()` directly into `pane.SetSize()`. Therefore `BasePane.width`
**is** the content width.

Yet `NowPlayingPane.contentWidth()` does `paneMax(p.width-4, 10)`, shaving off
4 more columns that do not exist. This leaves permanent dead space and narrows
every sub-component.

**Evidence:**
```go
// pane.go:46
func (r Rect) ContentWidth() int { return r.Width - 2 }

// app.go:737
pane.SetSize(rect.ContentWidth(), rect.ContentHeight())

// nowplaying.go:492
func (p *NowPlayingPane) contentWidth() int { return paneMax(p.width-4, 10) }
```

### RCA-2: `renderSideBySide` only prefixes spaces on line 0

The current code does:
```go
paddedRight := strings.Repeat(" ", infoWidth+npGap) + right
composite = lipgloss.JoinHorizontal(lipgloss.Top, infoView, paddedRight)
```

`right` is a multi-line string starting with `"\n"` (top pad). Prepending spaces
adds them **only to the first line** (the empty top-pad line). On subsequent
lines the visualizer begins at column 0 of the right block.
`lipgloss.JoinHorizontal` concatenates line-by-line, so the visualizer **abuts
the InfoBox border with zero gap** on every content line. The visual width of
line 0 differs from all other lines, creating the inconsistent layout the user
sees as “weird characters.”

### RCA-3: Vertical overflow when height is small

`SetSize` computes:
```go
rightH := p.height - 2*npPadV          // e.g. 3 - 2 = 1
vizHeight := rightH - npSeekRowH       // 1 - 1 = 0 → clamped to rightH = 1
```

Then `renderSideBySide` builds:
```
topPad(1) + viz(1) + seekBar(1) + botPad(1) = 4 lines
```
But `p.height = 3`. The right column is **taller than the pane**. The final
`contentH < p.height` guard only handles **underflow**, not overflow, so the
extra line leaks out and pushes the bottom border down.

### RCA-4: `GradientVolumeBar.Render` reserved-width bug

`GradientVolumeBar.Render()` (`gradient.go:234-238`) reserves 7 chars for
overhead:
```go
reserved := 7
computed := b.width - reserved
```
Actual overhead is `icon(1) + space(1) + pad(2) + percent(4 for "100%") = 8`.
At volume 100 the bar renders **1 character wider than its allocated width**.
`InfoBox.Render` then truncates that line with `…`, contributing to the
“weird characters on the right side” report.

### RCA-5: Compact presets give NowPlaying only 4–6 rows

`PresetDashboard`, `PresetLibrary`, `PresetDiscovery`, and `PresetStats` all set
`MinHeight: 0` for the NowPlaying row. On a 50-row terminal the LayoutManager
allocates ~4–9 rows. InfoBox `innerH = height - 2` becomes 2–7, so at 4 rows
(`innerH = 2`) only track + artist fit; controls and volume are invisible.

## Design

### 1. Remove `OverlayBackground` from InfoBox

File: `internal/ui/components/infobox.go`

Delete the `lipgloss.NewStyle().Background(b.th.OverlayBackground())` wrap
inside `Render()`. The interior renders as plain text. `Theme.OverlayBackground()`
stays on the interface (no breaking change).

### 2. Fix `contentWidth()` — no phantom subtraction

```go
func (p *NowPlayingPane) contentWidth() int { return paneMax(p.width, 10) }
```

`p.width` is already the borderless content width; nothing more to subtract.

### 3. Add `effectiveHeight()` with solo-pane cap

```go
const npMaxContentH = 24 // cap content height when pane is oversized

func (p *NowPlayingPane) effectiveHeight() int {
    if p.height > npMaxContentH {
        return npMaxContentH
    }
    return p.height
}
```

### 4. Adaptive InfoBox width constants

Replace the current constants block:

```go
const (
    npPadV         = 1  // vertical padding rows (top + bottom)
    npInfoPctTall  = 3  // tall pane: InfoBox = cw / 3
    npInfoPctShort = 2  // short pane: InfoBox = cw / 2
    npInfoMin      = 28 // minimum InfoBox width for controls + volume
    npGap          = 1  // column gap
    npMinViz       = 10 // below this, drop InfoBox entirely
)
```

Delete `npSeekRowH` and `npCompactMin`.

Adaptive formula (identical in `SetSize` and `renderSideBySide`):
```go
effH := p.effectiveHeight()
cw   := p.contentWidth()

infoWidth := cw / npInfoPctTall
if effH <= 8 {
    infoWidth = cw / npInfoPctShort
}
if infoWidth < npInfoMin {
    infoWidth = npInfoMin
}
vizWidth := cw - infoWidth - npGap
if vizWidth < npMinViz {
    infoWidth = 0
    vizWidth = cw
}
```

Height threshold `<= 8` matches the existing `Title()` compact threshold
(`p.height < 8`) and the `npMaxContentH` cap.

### 5. `SetSize` uses `effectiveHeight()`

```go
func (p *NowPlayingPane) SetSize(width, height int) {
    p.BasePane.SetSize(width, height)

    effH := p.effectiveHeight()
    cw   := p.contentWidth()

    // --- adaptive InfoBox width (same formula as renderSideBySide) ---
    infoWidth := cw / npInfoPctTall
    if effH <= 8 {
        infoWidth = cw / npInfoPctShort
    }
    if infoWidth < npInfoMin {
        infoWidth = npInfoMin
    }
    vizWidth := cw - infoWidth - npGap
    if vizWidth < npMinViz {
        infoWidth = 0
        vizWidth = cw
    }

    // --- visualizer height: reserve 1 row for seek bar ---
    rightH := effH - 2*npPadV
    if rightH < 1 {
        rightH = 1
    }
    vizHeight := rightH - 1
    if vizHeight < 1 {
        vizHeight = rightH
    }

    if infoWidth > 0 {
        p.infoBox.SetSize(infoWidth, effH)
    }
    p.engine.SetSize(vizWidth, vizHeight)
    p.seekBar.SetWidth(vizWidth)
    p.volumeBar.SetWidth(paneMax(infoWidth-4, 1))
}
```

### 6. `buildInfoLines` — drop album when space is tight, keep controls + volume

```go
func (p *NowPlayingPane) buildInfoLines(bodyHeight int) []string {
    ps := p.store.PlaybackState()
    if ps == nil || ps.Item == nil {
        return nil
    }
    t := ps.Item
    primaryStyle   := lipgloss.NewStyle().Foreground(p.theme.TextPrimary()).Bold(true)
    secondaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextSecondary())
    mutedStyle     := lipgloss.NewStyle().Foreground(p.theme.TextMuted())

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

    innerH := bodyHeight - 2 // InfoBox borders consume 2 rows
    if innerH < 1 {
        innerH = 1
    }
    if len(lines) > innerH {
        if innerH >= 4 {
            // Drop album line so controls + volume remain visible
            lines = append(lines[:2], lines[3:]...)
        } else {
            lines = lines[:innerH]
        }
    }
    return lines
}
```

**Why no combined controls+volume line?**
A combined line (`ctrl.Render() + "  " + p.volumeBar.Render()`) does **not fit**
inside `innerW = infoWidth - 2`. Volume bar alone at `SetWidth(infoWidth-4)`
renders `infoWidth - 3` chars (see RCA-4). Adding controls (~11 cols in ASCII
mode) produces `infoWidth + 8`, which overflows `innerW` by 10 cols and triggers
`TruncateOrPad` truncation. Dropping the album line is the correct compact-mode
behaviour.

### 7. Rewrite `renderSideBySide` with explicit line-by-line composition + vertical centering

```go
func (p *NowPlayingPane) renderSideBySide() string {
    ps := p.store.PlaybackState()
    t := ps.Item
    effH := p.effectiveHeight()
    cw   := p.contentWidth()

    infoWidth := cw / npInfoPctTall
    if effH <= 8 {
        infoWidth = cw / npInfoPctShort
    }
    if infoWidth < npInfoMin {
        infoWidth = npInfoMin
    }
    vizWidth := cw - infoWidth - npGap
    if vizWidth < npMinViz {
        infoWidth = 0
        vizWidth = cw
    }

    frame := p.engine.CurrentFrame()
    seekBar := p.seekBar.Render(p.localProgressMs, t.DurationMs)

    // Build right column lines
    rightLines := make([]string, 0, len(frame)+1)
    for _, line := range frame {
        style := lipgloss.NewStyle().Foreground(line.Color)
        rightLines = append(rightLines, style.Render(line.Text))
    }
    rightLines = append(rightLines, seekBar)

    // Fit right column into effective height, clipping visualizer (not seek bar) if needed
    targetH := effH
    contentH := len(rightLines)
    padTotal := targetH - contentH
    if padTotal < 0 {
        keepViz := targetH - 1 // reserve exactly 1 row for seek bar
        if keepViz < 0 {
            keepViz = 0
        }
        if len(rightLines)-1 > keepViz {
            rightLines = append(rightLines[:keepViz], seekBar)
        }
        padTotal = 0
    }
    topPad := padTotal / 2
    botPad := padTotal - topPad
    for i := 0; i < topPad; i++ {
        rightLines = append([]string{""}, rightLines...)
    }
    for i := 0; i < botPad; i++ {
        rightLines = append(rightLines, "")
    }

    // Compose line-by-line
    var lines []string
    if infoWidth > 0 {
        infoLines := p.buildInfoLines(effH)
        infoView := p.infoBox.Render("Track Info", infoLines, p.focused)
        infoSplit := strings.Split(infoView, "\n")

        // Equalise line count
        for len(infoSplit) < len(rightLines) {
            infoSplit = append(infoSplit, strings.Repeat(" ", infoWidth))
        }
        for len(rightLines) < len(infoSplit) {
            rightLines = append(rightLines, strings.Repeat(" ", vizWidth))
        }

        gap := strings.Repeat(" ", npGap)
        for i := range infoSplit {
            lines = append(lines, infoSplit[i]+gap+rightLines[i])
        }
    } else {
        lines = rightLines
    }

    // Centre vertically within oversized pane
    if p.height > effH {
        outerPad := (p.height - effH) / 2
        for i := 0; i < outerPad; i++ {
            lines = append([]string{""}, lines...)
            lines = append(lines, "")
        }
    }

    return strings.Join(lines, "\n")
}
```

Key invariants:
- Every output line width = `infoWidth + npGap + vizWidth` (exactly `cw`).
- No `lipgloss.JoinHorizontal` width surprises.
- Seek bar is never clipped; visualizer is clipped only when height is pathologically small.
- Oversized panes are capped at 24 rows and centred.

### 8. Fix `GradientVolumeBar.Render` reserved width

In `internal/ui/components/gradient.go`, change:
```go
reserved := 7
```
to:
```go
reserved := 8 // icon(1) + space(1) + pad(2) + percent(4 for "100%")
```

### 9. Raise `MinHeight` for compact presets

`internal/ui/layout/presets.go`:

```go
// PresetDashboard
{HeightWeight: 1, MinHeight: 6, Cells: []Cell{{PaneID: PaneNowPlaying, WidthWeight: 1}}}

// PresetLibrary
{HeightWeight: 1, MinHeight: 6, Cells: []Cell{{PaneID: PaneNowPlaying, WidthWeight: 1}}}

// PresetDiscovery
{HeightWeight: 1, MinHeight: 6, Cells: []Cell{{PaneID: PaneNowPlaying, WidthWeight: 1}}}

// PresetStats
{HeightWeight: 1, MinHeight: 6, Cells: []Cell{{PaneID: PaneNowPlaying, WidthWeight: 1}}}
```

`MinHeight: 6` guarantees `innerH >= 4`, which fits track + artist + controls +
volume after the album drop.

`PresetListening` keeps `MinHeight: 0`; its `HeightWeight: 2` already gives it
comfortable height.

### 10. Fix broken tests

- **`nowplaying_test.go:754`**: `splitFrame` does not exist in the current
codebase (leftover from story 222’s abandoned design). Remove
`TestNowPlayingPane_SplitFrame` entirely.
- **`np_inspect_test.go`**: Remove unused imports (`viz`, `layout`, `uikit`) so
it compiles. Keep the file as a debug test.
- Update width-math comments in overlay tests (e.g., `cw = 16-4 = 12` →
`cw = 16`) where they reference the old `contentWidth()` formula.

## Acceptance Criteria

- [ ] `make ci` passes (lint + tests + 80 % coverage)
- [ ] `InfoBox.Render` no longer applies `OverlayBackground` fill
- [ ] `contentWidth()` returns `paneMax(p.width, 10)` (no phantom subtraction)
- [ ] Constants: `npInfoPctTall = 3`, `npInfoPctShort = 2`, `npInfoMin = 28`, `npMaxContentH = 24`
- [ ] `renderSideBySide` uses explicit line arrays and line-by-line concatenation; no `JoinHorizontal` on prefixed raw strings
- [ ] `buildInfoLines` drops album line when `innerH >= 4`, otherwise truncates from bottom
- [ ] `GradientVolumeBar.Render` reserves 8 chars (was 7)
- [ ] `PresetDashboard`, `PresetLibrary`, `PresetDiscovery`, `PresetStats` have `MinHeight: 6`
- [ ] `TestNowPlayingPane_SplitFrame` removed
- [ ] `np_inspect_test.go` compiles (unused imports removed)
- [ ] Listening preset (`SetSize(160, 14)`): InfoBox ~33 % width, no truncation artifacts
- [ ] Library preset (`SetSize(160, 6)`): InfoBox ~50 % width, controls + volume visible, album dropped
- [ ] Solo pane (`SetSize(160, 46)`): content capped at 24 rows, centred vertically
- [ ] Narrow fallback (`SetSize(30, 16)`): InfoBox dropped because `vizWidth < 10`
- [ ] Minimum-size stress test (`SetSize(1, 1)`) does not panic

## Tasks

- [ ] Remove `OverlayBackground` fill from `internal/ui/components/infobox.go`
- [ ] Fix `contentWidth()` in `internal/ui/panes/nowplaying.go`
- [ ] Update constants block (adaptive pct, npInfoMin, npMaxContentH, remove npSeekRowH/npCompactMin)
- [ ] Add `effectiveHeight()` helper to `nowplaying.go`
- [ ] Rewrite `SetSize` to use `effectiveHeight()` and adaptive width formula
- [ ] Rewrite `renderSideBySide` with line-by-line composition + centering
- [ ] Update `buildInfoLines` with album-drop logic for compact mode
- [ ] Fix `GradientVolumeBar.Render` reserved width (`7 → 8`)
- [ ] Update presets (`MinHeight: 6` for Dashboard, Library, Discovery, Stats)
- [ ] Remove `TestNowPlayingPane_SplitFrame` from `nowplaying_test.go`
- [ ] Fix `np_inspect_test.go` unused imports
- [ ] Update width-math comments in overlay tests
- [ ] Run `make ci` and fix any failures
