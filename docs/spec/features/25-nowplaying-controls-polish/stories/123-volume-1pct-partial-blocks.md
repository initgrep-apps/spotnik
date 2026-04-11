---
title: "Volume 1% Steps and Partial-Block Bar"
feature: 25-nowplaying-controls-polish
status: done
---

## Background

Two related problems with volume UX:

**Step size.** `volumeStep` is hardcoded to `5` in `app.go:300`. The Spotify
`PUT /me/player/volume` API accepts any integer 0–100 with no minimum
increment, so 1% is fully supported.

**Bar resolution.** `GradientVolumeBar.Render()` computes fill with integer
truncation:
```go
filled := int(float64(volume) / 100.0 * float64(barWidth))
```
At the default width of 14 chars, one char represents ~7.1%. With 5% steps
this means a press often produces no visual change; at 1% steps it almost
never would. The fix is to use Unicode partial-block characters for the
fractional last cell, giving 8× sub-character resolution (~112 visual steps
across 14 chars).

**Depends on:** nothing — self-contained.

## Design

### app.go — step change

`internal/app/app.go:300`:
```go
// before
volumeStep: 5,
// after
volumeStep: 1,
```

No other logic changes — `buildPlaybackAPICmd` already reads `a.volumeStep`
and clamps to [0, 100].

### messages.go — comment update

`internal/ui/panes/messages.go:49` and `:51`:
```go
// before
// ActionVolumeUp raises volume by 5%.
// ActionVolumeDown lowers volume by 5%.
// after
// ActionVolumeUp raises volume by 1%.
// ActionVolumeDown lowers volume by 1%.
```

### gradient.go — partial-block fill algorithm

Replace the integer-truncation fill in `GradientVolumeBar.Render()` with a
fractional fill that uses Unicode block elements for the last cell.

**Block element set** (8 levels, index 0–7):
```
▏ ▎ ▍ ▌ ▋ ▊ ▉ █
```
Index 0 (`▏`) = 1/8 full. Index 7 (`█`) = fully full. When fraction is 0.0
exactly, no partial block is emitted.

**Algorithm:**
```go
partialChars := []string{"▏", "▎", "▍", "▌", "▋", "▊", "▉", "█"}

filledF    := float64(volume) / 100.0 * float64(barWidth)
fullBlocks := int(filledF)
fraction   := filledF - float64(fullBlocks)
partialIdx := int(fraction * 8) // 0 when fraction == 0

var sb strings.Builder
// full blocks
for i := 0; i < fullBlocks; i++ {
    sb.WriteString(fillStyle.Render("█"))
}
// partial block (only if fraction > 0)
if partialIdx > 0 {
    sb.WriteString(fillStyle.Render(partialChars[partialIdx-1]))
}
// empty
emptyCount := barWidth - fullBlocks
if partialIdx > 0 {
    emptyCount--
}
sb.WriteString(emptyStyle.Render(strings.Repeat("□", emptyCount)))
```

Partial block uses the same `fillStyle` (same band color) as full blocks.

**Edge cases:**
- `volume == 0`: `filledF = 0`, `fullBlocks = 0`, `fraction = 0`, `partialIdx = 0` → all empty. Correct.
- `volume == 100`: `filledF = barWidth` exactly (float), `fullBlocks = barWidth`, `fraction = 0` → all full, no partial. Correct.
- `barWidth == 0`: guard with `if barWidth < 1 { barWidth = 1 }` (already present).

**Visual examples at barWidth=14:**
```
 0%   □□□□□□□□□□□□□□
 1%   ▏□□□□□□□□□□□□□
 7%   █□□□□□□□□□□□□□
14%   ██□□□□□□□□□□□□
31%   ████▎□□□□□□□□□
50%   ███████□□□□□□□
99%   █████████████▊
100%  ██████████████
```

## Acceptance Criteria

- [ ] Pressing `+` sends volume + 1 to the Spotify API (not +5)
- [ ] Pressing `-` sends volume - 1 (not -5); clamped at 0
- [ ] `GradientVolumeBar.Render(0)` → all `□`, no partial block
- [ ] `GradientVolumeBar.Render(100)` → all `█`, no partial block
- [ ] `GradientVolumeBar.Render(1)` → one partial block char as first char, rest `□`
- [ ] `GradientVolumeBar.Render(50)` → exactly `barWidth/2` full blocks, no partial (exact midpoint)
- [ ] `GradientVolumeBar.Render(31)` at width 14 → 4 full blocks + one partial block + 9 `□`
- [ ] Partial block char uses the same gradient band color as adjacent full blocks
- [ ] `make ci` passes

## Tasks

- [ ] Add table-driven tests to `internal/ui/components/gradient_test.go` covering
      boundary volumes: 0, 1, 7, 14, 31, 50, 99, 100 — assert full-block count,
      presence/absence of partial char, and empty count at barWidth=14
  - test: `go test ./internal/ui/components/... -run TestGradientVolumeBar` -v` → FAIL
- [ ] Update any existing `GradientVolumeBar` tests that assert the old integer-fill
      output (block counts will change at some volumes)
- [ ] Replace integer fill logic in `GradientVolumeBar.Render()` with partial-block
      algorithm in `internal/ui/components/gradient.go`
  - test: all gradient volume bar tests → PASS
- [ ] Change `volumeStep: 5` → `volumeStep: 1` in `internal/app/app.go:300`
- [ ] Update volume comments in `internal/ui/panes/messages.go` lines 49 and 51
- [ ] `make ci` passes
