# Playback Controls Improvements — Design Spec

**Date:** 2026-04-11  
**Scope:** Two focused improvements to the NowPlaying pane — repeat-track icon rendering and volume bar granularity.

---

## 1. Repeat-Track Superscript Icon

### Problem
When repeat mode is `"track"`, `controls.go` renders `"↻1"` — a plain ASCII `1` appended to the repeat glyph. It reads like a label, not a modifier.

### Solution
Replace with U+00B9 SUPERSCRIPT ONE (`¹`): render `"↻¹"` instead.

**File:** `internal/ui/components/controls.go:53`

```go
// before
repeat = c.activeStyle.Render("↻1")
// after
repeat = c.activeStyle.Render("↻¹")
```

No color, layout, or test-output changes required beyond the literal character swap.

---

## 2. Volume: 1% Steps + Partial-Block Bar

### Problem
- `volumeStep` is hardcoded to `5` in `app.go` — coarse adjustment.
- `GradientVolumeBar.Render()` uses integer truncation: `filled := int(volume/100 * barWidth)`. At default width 14, one char ≈ 7.1%, so pressing `+` six times in a row produces no visible change.

### Spotify API
`PUT /me/player/volume?volume_percent=<n>` accepts any integer 0–100. No minimum increment is enforced.

### Solution

#### 2a. Step change
`internal/app/app.go:300` — change `volumeStep: 5` → `volumeStep: 1`.

Update comment in `internal/ui/panes/messages.go` lines 49 and 51: `"5%"` → `"1%"`.

#### 2b. Partial-block bar rendering
Use Unicode block elements (▏▎▍▌▋▊▉█) to represent the fractional portion of the last cell, giving ~8× sub-character resolution.

**Algorithm** (`internal/ui/components/gradient.go` — `GradientVolumeBar.Render()`):

```
filledF    = float64(volume) / 100.0 * float64(barWidth)
fullBlocks = int(filledF)                          // number of full █ chars
fraction   = filledF - float64(fullBlocks)         // 0.0–1.0
partialIdx = int(fraction * 8)                     // 0–7
partialChar = []string{"▏","▎","▍","▌","▋","▊","▉","█"}[partialIdx]
emptyCount = barWidth - fullBlocks - 1             // remaining □ chars
// if fraction == 0: no partial char, emptyCount = barWidth - fullBlocks
```

Partial char uses the same band color as full blocks (Gradient1/2/3 by volume level). Empty chars stay `□` in Surface() color.

**Visual examples at width=14:**
```
 0%   ♪ □□□□□□□□□□□□□□   0%
 1%   ♪ ▏□□□□□□□□□□□□□   1%
 7%   ♪ █□□□□□□□□□□□□□   7%
31%   ♪ ████▎□□□□□□□□□  31%
50%   ♪ ███████□□□□□□□  50%
100%  ♪ ██████████████ 100%
```

---

## Files Changed

| File | Change |
|---|---|
| `internal/ui/components/controls.go` | `"↻1"` → `"↻¹"` |
| `internal/app/app.go` | `volumeStep: 5` → `volumeStep: 1` |
| `internal/ui/panes/messages.go` | comments: `"5%"` → `"1%"` |
| `internal/ui/components/gradient.go` | partial-block fill algorithm in `GradientVolumeBar.Render()` |
| `internal/ui/components/gradient_test.go` | update/add tests for partial-block rendering |
| `internal/app/commands.go` | no logic change (step comes from `a.volumeStep`) |

---

## Testing

- `GradientVolumeBar.Render()`: table-driven tests for boundary values (0, 1, 7, 31, 50, 99, 100) asserting correct full-block count, partial char, and empty count.
- `Controls.Render()`: assert `"↻¹"` appears for `repeatMode = "track"`.
- Existing tests that assert specific volume bar output strings must be updated.
