---
title: "NowPlaying: responsive 3-tier layout with album art"
feature: 17-album-art
status: open
---

## Background

With album art now available (story 216), the NowPlaying pane needs a
responsive layout that uses the space well across all preset sizes:

- **Stats page** (≈16 rows after story 215's MinHeight fix) → base tier
- **Dashboard / Library / Discovery** (≈6–10 rows) → still too small; falls
  back to compact title-bar mode (unchanged from today)
- **Listening preset** (≈18–22 rows) → mid tier
- **Only NowPlaying visible / maximised** (≈30+ rows) → full tier

The image must always appear square. Terminal monospace chars are approximately
2:1 (height:width) so `imageCols = imageRows * 2` produces a visually square block.

## Design

### Tier thresholds (bodyHeight = height − 4)

| Tier | bodyHeight | Trigger |
|---|---|---|
| compact | < 10 | title-bar only — existing behaviour, unchanged |
| base | 10 – 18 | 3-col: image · info · viz |
| mid | 19 – 30 | 2-col: [image ┬ info] · viz |
| full | > 30 | 2-col: [larger image ┬ richer info] · viz |

### Image dimensions per tier

```
base:  imageRows = bodyHeight
       imageCols = imageRows * 2

mid:   infoRows  = 5   // name, artist, album, controls, volume
       imageRows = bodyHeight - infoRows
       imageCols = imageRows * 2

full:  infoRows  = 7   // name, artist, album·year, spacer, controls, volume, spacer
       imageRows = min(int(float64(bodyHeight)*0.72), bodyHeight-infoRows)
       imageCols = imageRows * 2
```

### Helper methods on `NowPlayingPane`

```go
func (p *NowPlayingPane) renderTier() renderTier {
    switch {
    case p.bodyHeight() > 30: return tierFull
    case p.bodyHeight() > 18: return tierMid
    case p.bodyHeight() >= 10: return tierBase
    default: return tierCompact
    }
}

func (p *NowPlayingPane) imageRows() int { /* per-tier formula above */ }
func (p *NowPlayingPane) imageCols() int { return p.imageRows() * 2 }
func (p *NowPlayingPane) bodyHeight() int { return max(p.height-4, 0) }
```

### Base tier (3-col)

```
┌─────────────────────────────────────────────────────────────┐
│ Now Playing                                                 │
│ ┌──────────┐  Track Name          ░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░  │
│ │  image   │  Artist · Album      ░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░  │
│ │ (square) │  ⇄  ▷  ↻            ──────────────────────  │
│ └──────────┘  ♪ ■■□□□ 40%        ░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░  │
└─────────────────────────────────────────────────────────────┘
```

Column widths:
- `imageCol`   = `imageCols` (= `bodyHeight * 2`)
- `infoCol`    = `max(contentWidth - imageCol - vizCol - 2, 18)` where gap chars = 2
- `vizCol`     = `contentWidth - imageCol - infoCol - 2`
- Minimum `vizCol = 10`; if terminal too narrow, image column is omitted and pane
  falls back to the pre-feature 2-col layout

Info column content (top-aligned, truncated to `infoCol` width):
```
Track Name
Artist
⇄  ▷  ↻
♪ ■■□□□ 40%
```

Render: `lipgloss.JoinHorizontal(lipgloss.Top, imageBlock, " ", infoBlock, " ", vizBlock)`

### Mid tier (2-col)

```
┌──────────────────────────────────────────────────────────────┐
│ Now Playing                                                  │
│ ┌──────────────────┐  ░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░  │
│ │                  │  ░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░  │
│ │   album art      │  ──────────────────────────────────  │
│ │   (square)       │  ░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░  │
│ └──────────────────┘  ░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░  │
│  Track Name                                                  │
│  Artist · Album                                              │
│  ⇄  ▷  ↻   ♪ ■■■□□ 60%                                   │
└──────────────────────────────────────────────────────────────┘
```

Column widths:
- `col1Width` = `imageCols` (= `imageRows * 2`)
- `col2Width` = `contentWidth - col1Width - 1` (gap = 1)

Col 1: `lipgloss.JoinVertical(lipgloss.Left, imageBlock, infoBlock)`
- `imageBlock` = pixterm rows joined, height = `imageRows`, width = `imageCols`
- `infoBlock` = 5 lines of track info, width = `col1Width`

Col 2: existing viz engine output (full `bodyHeight` rows) + seek bar (unchanged position)

Render: `lipgloss.JoinHorizontal(lipgloss.Top, col1, " ", col2)`

### Full tier (2-col)

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ Now Playing                                                                  │
│ ┌──────────────────────────────────────┐  ░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░  │
│ │                                      │  ░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░  │
│ │           album art                  │  ░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░  │
│ │           (square, large)            │  ░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░  │
│ │                                      │  ░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░  │
│ │                                      │  ──────────────────────────────  │
│ │                                      │  ░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░  │
│ └──────────────────────────────────────┘                                    │
│  Track Name                                                                  │
│  Artist Name                                                                 │
│  Album · 2024                                                                │
│                                                                              │
│  ⇄  ▷  ↻                                                                   │
│  ♪ ■■■■□□□□□□ 40%                                                          │
└──────────────────────────────────────────────────────────────────────────────┘
```

Same 2-col structure as mid. Differences:
- `imageRows` uses the 72% formula → larger square image
- Info block shows 7 lines: name, artist, `album · year`, blank, controls, volume, blank
- `col1Width` grows with the larger image

### Fallback (no image)

When `!artRenderer.HasImage()` and `!artRenderer.IsLoading()`:
- Use pre-feature 2-col layout: InfoBox left, viz+seek bar right
- This is the exact existing behaviour — reuse the existing render helpers

When `artRenderer.IsLoading()`:
- Render a placeholder block in the image column: `bodyHeight` rows of
  `imageCols`-wide spaces styled with `theme.TextMuted()` background
- Rest of layout as normal for the current tier

### Re-triggering fetch on resize

When `SetSize()` is called and the new `imageRows()` differs from the previous
value by more than 2 rows AND an image URL is known, dispatch a new
`FetchAlbumArtCmd` with the updated dimensions. This ensures the image
resolution matches the terminal size after a resize event.

```go
func (p *NowPlayingPane) SetSize(w, h int) {
    prevRows := p.imageRows()
    p.width, p.height = w, h
    // ... existing SetSize logic ...
    if abs(p.imageRows()-prevRows) > 2 && p.artRenderer.HasImage() {
        // return cmd from here or store the pending resize refresh flag
        // and emit it on next Update() tick
    }
}
```

Because `SetSize()` cannot return a `tea.Cmd`, set a `pendingArtRefresh bool`
flag; the next `Update(tea.WindowSizeMsg)` handler checks this flag and
dispatches the re-fetch.

### `internal/ui/panes/nowplaying.go` changes summary

- Add `renderTier` type and constants (`tierCompact`, `tierBase`, `tierMid`, `tierFull`)
- Add `imageRows()`, `imageCols()`, `bodyHeight()`, `renderTier()` helpers
- Add `pendingArtRefresh bool` field for resize re-fetch
- Refactor `View()` to dispatch: `renderCompact()` (existing), `renderBase()`,
  `renderMid()`, `renderFull()`
- `renderCompact()` = existing title-bar-only code, untouched
- `renderBase/Mid/Full()` build their column blocks and join horizontally
- Existing `infoBox`, `engine`, `seekBar`, `volumeBar` sub-components are reused
  in all tiers; their `SetSize()` calls are updated per-tier in `SetSize()`
- `WindowSizeMsg` handler: check `pendingArtRefresh`, dispatch fetch if set

## Acceptance Criteria

- [ ] `renderTier()` returns correct tier for bodyHeight values across the boundary
- [ ] Base tier renders 3 columns; image column width = `bodyHeight * 2`
- [ ] Mid tier renders 2 columns; image above info in col 1; viz in col 2
- [ ] Full tier renders 2 columns; image uses 72% of bodyHeight; 7-line info block
- [ ] Image is always approximately square (cols ≈ rows × 2) in all tiers
- [ ] Compact mode (bodyHeight < 10) unchanged — no regression
- [ ] No image loaded → 2-col fallback (InfoBox + viz), no empty image column
- [ ] Loading state → muted placeholder block in image column position
- [ ] Resize event re-triggers art fetch when image dimensions change by > 2 rows
- [ ] `make ci` passes

## Tasks

- [ ] Add `renderTier` type and constants; implement `bodyHeight()`, `imageRows()`,
      `imageCols()`, `renderTier()` helpers in `internal/ui/panes/nowplaying.go`
      - test: table-driven `TestNowPlayingPane_RenderTier` — assert correct tier
        for bodyHeight 8, 10, 15, 19, 25, 31, 45

- [ ] Implement `renderBase()` — 3-col layout with image, info, viz
      - test: `SetSize(120, 16)` → `View()` contains pixterm ANSI sequences in first
        column position; info text present; viz chars present; no layout gaps wider
        than 1 space between columns

- [ ] Implement `renderMid()` — 2-col: [imageBlock ┬ infoBlock] left, viz right
      - test: `SetSize(120, 25)` → `View()` first section has image rows followed by
        track name / artist / controls on same left edge; right section has viz chars;
        assert col widths via `lipgloss.Width()`

- [ ] Implement `renderFull()` — 2-col: larger image + 7-line info, viz right
      - test: `SetSize(120, 45)` → image rows count ≥ mid-tier image rows;
        info block contains 7 lines including album + year and blank spacers

- [ ] Refactor `View()` to call the correct render helper via `renderTier()`
      - test: `TestNowPlayingPane_View_TierDispatch` — for each of the 4 body heights
        (8, 14, 24, 35) assert the rendered output matches the expected column count

- [ ] Fallback: when `!artRenderer.HasImage()` && `!artRenderer.IsLoading()` in
      base/mid/full tiers, render pre-feature 2-col layout
      - test: no `AlbumArtFetchedMsg` ever sent; all tiers render without empty column

- [ ] Loading placeholder: when `artRenderer.IsLoading()` render muted block
      in image column position
      - test: `artRenderer.SetLoading("id")` then `View()` — assert placeholder
        width = `imageCols` and height = `imageRows`

- [ ] Add `pendingArtRefresh bool` field; `SetSize()` sets it when imageRows changes
      > 2; `WindowSizeMsg` handler dispatches re-fetch and clears flag
      - test: call `SetSize(w, 20)` then `SetSize(w, 40)` — second call sets flag;
        next `Update(tea.WindowSizeMsg{})` returns non-nil cmd

- [ ] Update `SetSize()` to call sub-component `SetSize()` correctly in all tiers
      (viz + seek bar + vol bar all get updated dimensions)
      - test: no sub-component receives a 0 or negative dimension in any tier

- [ ] `make ci` passes
