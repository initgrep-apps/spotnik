---
title: "NowPlaying: responsive 3-tier layout with album art"
feature: 17-album-art
status: done
---

## Background

With album art now available (story 216), the NowPlaying pane needs a
responsive layout that uses the space well across all preset sizes:

- **Dashboard / Library / Discovery** (≥ 14 rows via MinHeight added in story 215)
  → base tier
- **Stats page** (≈16 rows via existing MinHeight:14) → base tier
- **Listening preset** (≈23 rows at 50-row terminal) → mid tier
- **Only NowPlaying visible / maximised** (42+ rows) → full tier

`MinHeight: 14` is added to the NowPlaying row in Dashboard, Library, and
Discovery presets (in addition to Stats which already has it). This ensures
`bodyH ≥ 10` (base tier) at any terminal size — no pane is left too small for art.

The image must always appear square. Terminal monospace chars are approximately
2:1 (height:width) so `imageCols = imageRows * 2` produces a visually square block.

## Design

### Tier thresholds (bodyHeight = height − 4)

Three tiers. No compact/title-bar-only fallback — base handles all small sizes.

| Tier | bodyHeight | Pane height |
|------|-----------|-------------|
| base | ≤ 18      | ≤ 22 rows   |
| mid  | 19 – 30   | 23 – 34 rows |
| full | > 30      | 35+ rows    |

### Helper methods on `NowPlayingPane`

```go
func (p *NowPlayingPane) renderTier() renderTier {
    switch {
    case p.bodyHeight() > 30: return tierFull
    case p.bodyHeight() > 18: return tierMid
    default:                  return tierBase
    }
}

func (p *NowPlayingPane) imageRows() int { /* per-tier formula — see sections below */ }
func (p *NowPlayingPane) imageCols() int { return p.imageRows() * 2 }
func (p *NowPlayingPane) bodyHeight() int { return max(p.height-4, 0) }
```

### Base tier (bodyH ≤ 18) — 3-col inline

```
╭──────────────────────────────────────────────────────────────╮
│ Now Playing                                                  │
│ ┌──────────┐  ╭─ Track Info ─────╮  ░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░ │
│ │          │  │  Track Name      │  ░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░ │
│ │  album   │  │  Artist          │  ──────────────────────  │
│ │  art     │  │  Album           │  ░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░ │
│ │ (square) │  │  ⇄  ▷  ↻         │  ░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░ │
│ │          │  │  ♪ ■■□□□ 40%    │                          │
│ │          │  │                  │                          │
│ │          │  ╰──────────────────╯                          │
╰──────────────────────────────────────────────────────────────╯
```

Formulas:
```
imageRows = bodyH
imageCols = imageRows × 2
remaining = contentWidth − imageCols − 2   (gap chars)
infoCol   = vizCol = remaining / 2         (equal split, floor)
```

**Fallback:** if `remaining < 28` (18 min info + 10 min viz), image column is
dropped and the pane uses the existing pre-feature 2-col layout (InfoBox left,
viz right).

**InfoBox:** `width = infoCol`, `height = bodyH`, `title = "Track Info"`

InfoBox content — **top-aligned**. InfoBox vertically centers content by
default (`topPad = (innerH − len) / 2`). To force top-alignment, pad the
content slice with trailing blank strings to fill `bodyH − 2` lines:
```
lines = [trackName, artist, album, controls, volume]
for len(lines) < bodyH-2 {
    lines = append(lines, "")
}
```
This ensures `remaining = 0` so InfoBox renders content flush to the top border,
aligned with the image block.

Content (5 meaningful lines + trailing blanks):
```
Track Name
Artist
Album
⇄  ▷  ↻
♪ ■■□□□ 40%
```

Render: `lipgloss.JoinHorizontal(lipgloss.Top, imageBlock, " ", infoBox, " ", vizBlock)`

### Mid tier (bodyH 19–30) — 2-col + compact InfoBox

```
╭──────────────────────────────────────────────────────────────╮
│ Now Playing                                                  │
│ ┌──────────────────────┐  ░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░   │
│ │                      │  ░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░   │
│ │   album art          │  ──────────────────────────────    │
│ │   (square)           │  ░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░   │
│ │                      │  ░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░   │
│ ╭──────────────────────────────────────────────────────────╮ │
│ │  Track Name · Artist · Album      ⇄  ▷  ↻  ♪ ■■■□□ 60% │ │
│ ╰──────────────────────────────────────────────────────────╯ │
╰──────────────────────────────────────────────────────────────╯
```

Formulas:
```
imageRows = min(bodyH − 4, (contentWidth − 11) / 2)
imageCols = imageRows × 2
col1Width = imageCols
col2Width = contentWidth − imageCols − 1   (≥ 10)
```

`(contentWidth − 11) / 2` caps `imageRows` so that col2 never falls below 10
chars. The `−4` reserves rows for the InfoBox (2 border + 2 content lines).

Upper section: `lipgloss.JoinHorizontal(lipgloss.Top, imageBlock, " ", vizBlock)`
- `imageBlock` = pixterm rows, height = `imageRows`, width = `imageCols`
- `vizBlock`   = viz engine output, height = `imageRows`, width = `col2Width`

**InfoBox:** `width = contentWidth`, `height = 4`, `title = ""`

InfoBox title is empty — pane header already shows "Now Playing".

InfoBox content (2 lines, innerH = 2):
```
Track Name · Artist · Album
⇄  ▷  ↻   ♪ ■■■□□ 60%
```

Render: `lipgloss.JoinVertical(lipgloss.Left, upperSection, infoBox)`

### Full tier (bodyH > 30) — 2-col + richer InfoBox

```
╭──────────────────────────────────────────────────────────────╮
│ Now Playing                                                  │
│ ┌──────────────────────┐  ░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░   │
│ │                      │  ░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░   │
│ │   album art          │  ░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░   │
│ │   (large square)     │  ──────────────────────────────    │
│ │                      │  ░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░   │
│ │                      │  ░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░▒▒░░   │
│ ╭── Track Info ────────────────────────────────────────────╮ │
│ │  Track Name                                              │ │
│ │  Artist · Album                    ⇄  ▷  ↻  ♪ ■■□□ 40% │ │
│ │                                                          │ │
│ ╰──────────────────────────────────────────────────────────╯ │
╰──────────────────────────────────────────────────────────────╯
```

Formulas:
```
infoRows  = 5  (fixed — 2 border + 3 content rows)
imageRows = min(bodyH − 5, (contentWidth − 11) / 2)
imageCols = imageRows × 2
col1Width = imageCols
col2Width = contentWidth − imageCols − 1   (≥ 10)
```

Same `imageRows` cap as mid. The `−5` reserves rows for the richer InfoBox.

Upper section: `lipgloss.JoinHorizontal(lipgloss.Top, imageBlock, " ", vizBlock)`
- `imageBlock` = pixterm rows, height = `imageRows`, width = `imageCols`
- `vizBlock`   = viz engine output, height = `imageRows`, width = `col2Width`

**InfoBox:** `width = contentWidth`, `height = 5`, `title = "Track Info"`

InfoBox content (3 lines, innerH = 3):
```
Track Name
Artist · Album                    ⇄  ▷  ↻  ♪ ■■□□ 40%
                                                          ← blank (bottom breath)
```

Track name gets its own full-width line. Artist · album and controls share the
second line. Third line is blank — gives the panel breathing room.

Render: `lipgloss.JoinVertical(lipgloss.Left, upperSection, infoBox)`

### Fallback (no image)

When `!artRenderer.HasImage()` and `!artRenderer.IsLoading()`:
- All tiers fall back to the existing pre-feature 2-col layout: InfoBox left,
  viz+seek bar right. No empty image column is shown.

When `artRenderer.IsLoading()`:
- Render a placeholder block in the image column: `imageRows × imageCols` spaces
  styled with `theme.TextMuted()` background. Signals art is incoming.

### Re-triggering fetch on resize

When `SetSize()` is called and the new `imageRows()` differs from the previous
value by more than 2 rows AND an image URL is known, set `pendingArtRefresh = true`.
The next `Update(tea.WindowSizeMsg)` handler dispatches a new `FetchAlbumArtCmd`
with updated dimensions and clears the flag.

```go
func (p *NowPlayingPane) SetSize(w, h int) {
    prevRows := p.imageRows()
    p.width, p.height = w, h
    // ... existing SetSize logic ...
    if abs(p.imageRows()-prevRows) > 2 && p.artRenderer.HasImage() {
        p.pendingArtRefresh = true
    }
}
```

### `internal/ui/panes/nowplaying.go` changes summary

- Add `renderTier` type and constants (`tierBase`, `tierMid`, `tierFull`)
- Add `imageRows()`, `imageCols()`, `bodyHeight()`, `renderTier()` helpers
- Add `pendingArtRefresh bool` field for resize re-fetch
- Refactor `View()` to dispatch: `renderBase()`, `renderMid()`, `renderFull()`
- `renderBase/Mid/Full()` build their column blocks and join horizontally/vertically
- Existing `infoBox`, `engine`, `seekBar`, `volumeBar` sub-components are reused
  in all tiers; their `SetSize()` calls are updated per-tier in `SetSize()`
- `WindowSizeMsg` handler: check `pendingArtRefresh`, dispatch fetch if set
- Add `MinHeight: 14` to Dashboard, Library, Discovery NowPlaying rows in
  `internal/ui/layout/presets.go`

### Info line format

| Field | Format |
|-------|--------|
| Track + Artist + Album (mid, 1 line) | `Track Name · Artist · Album` |
| Track name (full/base) | `Track Name` (own line, truncated to width) |
| Artist + Album (full) | `Artist · Album` (truncated, controls fill right) |
| Controls | `⇄  ▷  ↻` |
| Volume | `♪ ■■□□□ 40%` |

Truncation is right-truncated with `…`. Artist before album so artist
(more important) survives truncation first.

## Acceptance Criteria

- [ ] `renderTier()` returns `tierBase` for bodyH ≤ 18, `tierMid` for 19–30, `tierFull` for > 30
- [ ] Base tier renders 3 columns: image · InfoBox · viz; image width = `bodyH * 2`
- [ ] Base tier InfoBox content is top-aligned (trailing blank padding to fill `bodyH − 2`)
- [ ] Mid tier renders image+viz side by side with full-width InfoBox below (2-line, no title)
- [ ] Full tier renders image+viz side by side with full-width InfoBox below (3-line, "Track Info" title)
- [ ] `imageRows` capped so `col2Width ≥ 10` in mid and full tiers
- [ ] Image is always approximately square (`imageCols = imageRows × 2`) in all tiers
- [ ] No image loaded → 2-col fallback (InfoBox + viz), no empty image column
- [ ] Loading state → muted placeholder block (`imageRows × imageCols`) in image column position
- [ ] Resize event re-triggers art fetch when imageRows changes by > 2
- [ ] `MinHeight: 14` added to Dashboard, Library, Discovery NowPlaying rows in presets.go
- [ ] `make ci` passes

## Tasks

- [ ] Add `renderTier` type and constants (`tierBase`, `tierMid`, `tierFull`); implement
      `bodyHeight()`, `imageRows()`, `imageCols()`, `renderTier()` helpers in
      `internal/ui/panes/nowplaying.go`
      - test: table-driven `TestNowPlayingPane_RenderTier` — assert correct tier
        for bodyHeight 10, 15, 18, 19, 25, 30, 31, 45

- [ ] Implement `renderBase()` — 3-col: imageBlock · InfoBox · vizBlock
      - InfoBox top-aligned (content padded with trailing blanks to `bodyH − 2` lines)
      - Fallback to 2-col when `remaining < 28`
      - test: `SetSize(120, 16)` → `View()` contains pixterm ANSI sequences in image
        position; info text present; viz chars present; no layout gaps wider than 1 space;
        InfoBox content starts at top border (no leading blank lines)

- [ ] Implement `renderMid()` — upper: image+viz side-by-side; lower: full-width 2-line InfoBox
      - `imageRows` capped by `(contentWidth − 11) / 2`
      - test: `SetSize(120, 25)` → col2 width ≥ 10; InfoBox spans full contentWidth;
        InfoBox has no title; first content line contains `·` separator

- [ ] Implement `renderFull()` — upper: image+viz side-by-side; lower: full-width 5-row InfoBox
      - `imageRows` capped by `(contentWidth − 11) / 2`
      - test: `SetSize(120, 45)` → InfoBox has title "Track Info"; first content line
        is track name alone; second content line contains `·` and controls; col2 width ≥ 10

- [ ] Refactor `View()` to call the correct render helper via `renderTier()`
      - test: `TestNowPlayingPane_View_TierDispatch` — for bodyHeights 14, 24, 35 assert
        rendered output matches expected tier structure (3-col / 2-col+2-line / 2-col+3-line)

- [ ] Fallback: when `!artRenderer.HasImage()` && `!artRenderer.IsLoading()` in all tiers,
      render pre-feature 2-col layout
      - test: no `AlbumArtFetchedMsg` ever sent; all tiers render without empty column

- [ ] Loading placeholder: when `artRenderer.IsLoading()` render muted block in image
      column position
      - test: `artRenderer.SetLoading("id")` then `View()` — assert placeholder
        width = `imageCols` and height = `imageRows`

- [ ] Add `pendingArtRefresh bool` field; `SetSize()` sets it when imageRows changes > 2;
      `WindowSizeMsg` handler dispatches re-fetch and clears flag
      - test: call `SetSize(w, 20)` then `SetSize(w, 40)` — second call sets flag;
        next `Update(tea.WindowSizeMsg{})` returns non-nil cmd

- [ ] Update `SetSize()` to call sub-component `SetSize()` correctly in all tiers
      (viz + seek bar + vol bar all get updated dimensions)
      - test: no sub-component receives a 0 or negative dimension in any tier

- [ ] Add `MinHeight: 14` to Dashboard, Library, Discovery NowPlaying rows in
      `internal/ui/layout/presets.go`
      - test: verify each preset has MinHeight:14 on the NowPlaying row

- [ ] `make ci` passes
