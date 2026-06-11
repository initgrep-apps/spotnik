---
title: "InfoBox content padding & controls centering"
feature: 03-playback
status: done
---

## Background

The NowPlaying InfoBox renders five content lines inside a bordered sub-pane:
track name, artist, album, transport controls, and volume bar. Two layout
issues make the InfoBox feel cramped and visually unbalanced:

1. **No left padding on text lines** — track name, artist, and album text
   starts at column 0 of the InfoBox interior, flush against the left border
   `│`. There is no breathing room between the border glyph and the text.

2. **Controls left-aligned, volume bar full-width** — `PlaybackControls.Render()`
   outputs `⇄  ⏸  ⟳` (~9–11 rendered columns) left-aligned, while
   `GradientVolumeBar.Render()` fills the full available width. The controls
   appear off-center relative to the volume bar beneath them. Centering the
   controls horizontally makes the two rows visually balanced.

Current `buildInfoLines` output (80×24 terminal, infoWidth≈26):

```
│ Blinding Lights             │
│ The Weeknd                  │
│ After Hours                 │
│ ⇄  ⏸  ⟳                    │  ← left-aligned, looks off-center
│ ♪ ████▎░░░░░░░░░ 65%        │  ← full width
```

Desired output:

```
│  Blinding Lights            │  ← 2-col left padding
│  The Weeknd                 │  ← 2-col left padding
│  After Hours                 │  ← 2-col left padding
│       ⇄  ⏸  ⟳              │  ← horizontally centered
│ ♪ ████▎░░░░░░░░░ 65%        │  ← unchanged (full width)
```

---

## Design

### Change 1 — Add left padding constant

**File:** `internal/ui/panes/nowplaying.go`

Add a constant for the left padding applied to InfoBox text lines:

```go
npInfoPadLeft = 2 // left padding columns for InfoBox text lines
```

### Change 2 — Left-pad text lines in `buildInfoLines`

**File:** `internal/ui/panes/nowplaying.go`

Prefix track name, artist, and album lines with
`strings.Repeat(" ", npInfoPadLeft)`. The controls and volume bar lines are
NOT padded — controls will be centered separately (Change 3), and the volume
bar already fills the available width.

```go
pad := strings.Repeat(" ", npInfoPadLeft)
lines := []string{
    pad + primaryStyle.Render(t.Name),
    pad + secondaryStyle.Render(strings.Join(artistNames, ", ")),
    pad + mutedStyle.Render(t.Album.Name),
    ctrlLine,
    p.volumeBar.Render(),
}
```

### Change 3 — Horizontally center the controls row

**File:** `internal/ui/panes/nowplaying.go`

Center the controls string within the InfoBox interior width. This requires
knowing `innerW` (interior width = `infoWidth - 2`). Update `buildInfoLines`
signature to accept `infoWidth`:

```go
func (p *NowPlayingPane) buildInfoLines(bodyHeight int, infoWidth int) []string {
```

Compute `innerW` and center the controls:

```go
innerW := infoWidth - 2 // InfoBox borders consume 2 columns
ctrl := components.NewControls(p.theme, ps.IsPlaying, ps.ShuffleState, ps.RepeatState)
ctrlLine := lipgloss.NewStyle().
    Width(innerW).
    Align(lipgloss.Center).
    Render(ctrl.Render())
```

`lipgloss.Place` with `lipgloss.Center` would also work, but
`Style.Width(innerW).Align(lipgloss.Center)` is simpler and avoids an
extra import. The centered string is exactly `innerW` columns wide, so
`TruncateOrPad` in `InfoBox.Render` is a no-op on this line.

### Change 4 — Update call site in `renderSideBySide`

**File:** `internal/ui/panes/nowplaying.go`

Pass `infoWidth` to `buildInfoLines`:

```go
infoLines := p.buildInfoLines(effH, infoWidth)
```

### Change 5 — Update call site in `buildInfoLines` compact-mode logic

The existing compact-mode branch (`len(lines) > innerH`) drops the album line
when `innerH >= 4`, otherwise truncates. This logic is unchanged — only the
signature and the two content changes (padding, centering) differ.

---

## Acceptance Criteria

- [ ] Track name, artist, and album lines have 2 columns of left padding inside the InfoBox
- [ ] Transport controls (shuffle, play/pause, repeat) are horizontally centered relative to the InfoBox interior width
- [ ] Volume bar rendering and width calculation unchanged
- [ ] Compact mode (height=10): controls still visible and centered
- [ ] Very compact mode (height=8): controls still visible
- [ ] `make ci` passes (lint + tests + 80% coverage)

---

## Tasks

### Task 1 — Add constant and update `buildInfoLines`

**File:** `internal/ui/panes/nowplaying.go`

- [ ] Add `npInfoPadLeft = 2` constant
- [ ] Update `buildInfoLines` signature: add `infoWidth int` parameter
- [ ] Compute `innerW = infoWidth - 2`
- [ ] Prefix text lines with 2-column left padding
- [ ] Center controls line using `lipgloss.NewStyle().Width(innerW).Align(lipgloss.Center).Render(ctrl.Render())`
- [ ] Update call site in `renderSideBySide` to pass `infoWidth`
  - test: `go build ./internal/ui/panes/...` compiles

### Task 2 — Update tests

**File:** `internal/ui/panes/nowplaying_test.go`

- [ ] Add `TestNowPlayingPane_InfoBoxLeftPadding`: verify View() output at 80×24
      contains left-padded track text (strip ANSI, check first content line starts
      with 2 spaces after the left border)
- [ ] Add `TestNowPlayingPane_ControlsCentered`: verify controls row is centered
      by checking the controls line in the InfoBox output has equal or ±1
      leading/trailing space around the controls glyphs
- [ ] Existing compact-mode tests (`CompactShowsControls`, `VeryCompactShowsControls`)
      still pass
  - test: `go test ./internal/ui/panes/ -v` → all PASS

### Task 3 — Full CI verification

- [ ] `make ci` → lint + tests + 80% coverage all PASS