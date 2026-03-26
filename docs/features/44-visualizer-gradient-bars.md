# Feature 44 — Visualizer + Gradient Bars

> **Feature:** Build a braille-dot audio visualizer and gradient-colored bars
> (seek bar + volume bar) as reusable components for the NowPlaying pane.

## Context

The current player uses monochrome bars: `ProgressBar` with `SeekBar()` color and
`VolumeBar` with `VolumeBar()` color (both in `internal/ui/components/`).

The new DESIGN.md (§11) specifies:
- A **braille-dot audio visualizer** that animates when music plays, using Unicode
  braille characters (U+2800-U+28FF) with a precomputed frame table
- **Gradient seek bar**: fill transitions from `Gradient1()` to `Gradient2()` left-to-right
- **Volume bar with color bands**: green (0-33%), yellow (34-66%), red (67-100%)

These components are embedded in the NowPlaying pane (Feature 45).

**Design reference:** `docs/DESIGN.md` §11 (Visual Components — Braille-Dot Audio Visualizer,
Gradient-Colored Bars)

**Depends on:** Feature 40 (theme tokens: `VisualizerFg`, `Gradient1/2/3`)

---

## Design Diagram

```
Braille-Dot Visualizer (DESIGN.md §11):

  ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿   ← playing (animated)
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀   ← paused (flat line)

  - Animates on 200ms tick when playing
  - frameIndex counter → precomputed frame table (30-50 patterns)
  - Width adapts to pane width
  - Height: 1-4 lines depending on available space
  - Color: VisualizerFg() token

Gradient Seek Bar:

  ████████████████░░░░░░░░░░░░░░
  ← Gradient1()  →← Gradient2() →     (left-to-right gradient fill)

  Empty portion: Surface() color

Volume Bar with Color Bands:

  VOL  ████████░░░░░░  65%

  0-33%:   Gradient1() (green/cool)    VOL ████░░░░░░░░░░ 25%
  34-66%:  Gradient2() (yellow/warm)   VOL █████████░░░░░ 55%
  67-100%: Gradient3() (red/hot)       VOL ████████████░░ 85%
```

---

## Task 1: Create braille-dot visualizer component

**Problem:** No audio visualizer exists.

**Fix:**

Create `internal/ui/components/visualizer.go`:

```go
package components

import (
    "time"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// VisualizerTickMsg is sent on the visualizer's animation tick.
type VisualizerTickMsg time.Time

// Visualizer renders an animated braille-dot audio spectrum.
type Visualizer struct {
    theme      theme.Theme
    playing    bool
    frameIndex int
    width      int
    height     int       // number of display lines (1-4)
    interval   time.Duration
    frames     [][]string // precomputed frame table [frameIndex][lineIndex] = braille string
}

// NewVisualizer creates a Visualizer with default 200ms interval.
func NewVisualizer(t theme.Theme) *Visualizer

// SetSize updates the visualizer dimensions.
func (v *Visualizer) SetSize(width, height int)

// SetPlaying controls animation state.
// When playing, frameIndex advances on each tick.
// When paused, frameIndex freezes and a flat-line pattern is shown.
func (v *Visualizer) SetPlaying(playing bool)

// Init returns the initial tick command.
func (v *Visualizer) Init() tea.Cmd

// Update handles VisualizerTickMsg to advance animation.
func (v *Visualizer) Update(msg tea.Msg) tea.Cmd

// View renders the current frame. Pure function — reads frameIndex, returns string.
func (v *Visualizer) View() string

// tickCmd returns a tea.Tick for the next animation frame.
func (v *Visualizer) tickCmd() tea.Cmd
```

**Implementation details:**

1. **Frame table generation** (`generateFrames()`):
   - Generate 40 frames of braille bar patterns
   - Each frame is a `[]string` (one string per line of height)
   - Bars simulate audio spectrum: varying heights per column
   - Use deterministic math (sine waves with phase offsets, not random)
   - Braille characters: U+2800 (blank) through U+28FF
   - Each braille character encodes a 2×4 dot matrix — use the lower dots to create
     varying bar heights

2. **Braille encoding for bar heights:**
   ```
   Each column is 1 character wide, up to 4 dots tall per character.
   Multiple lines stack vertically for taller bars.

   Height mapping (single char, 1 line):
     0 dots: ⠀ (U+2800)
     1 dot:  ⡀ (U+2840)
     2 dots: ⡠ (U+2860)
     3 dots: ⡰ (U+2870)
     4 dots: ⣰ (U+28F0)
     ... etc. (use dot positions 1,2,3,7 for left column of each char)
   ```

3. **Animation:** On `VisualizerTickMsg`:
   - If playing: increment `frameIndex`, wrap at len(frames)
   - Re-arm tick: `tea.Tick(v.interval, func(t time.Time) tea.Msg { return VisualizerTickMsg(t) })`
   - If paused: still tick (to detect play resume), but don't increment frameIndex

4. **Paused state:** Show a flat-line pattern (all bars at minimum height).

5. **Color:** All braille characters styled with `VisualizerFg()`.

6. **Width adaptation:** Frame generation uses `v.width` to determine number of braille columns.
   Regenerate frames when `SetSize()` changes width.

**Files:**
- Create: `internal/ui/components/visualizer.go`

**Tests:**
- Unit: `NewVisualizer` starts with frameIndex=0
- Unit: `SetPlaying(true)` + Update with VisualizerTickMsg → frameIndex increments
- Unit: `SetPlaying(false)` + Update with VisualizerTickMsg → frameIndex stays same
- Unit: `View()` when playing returns non-empty braille string
- Unit: `View()` when paused returns flat-line pattern
- Unit: Frame wraps after reaching end of frame table
- Unit: `SetSize(40, 2)` → View output is 2 lines, each 40 chars wide
- Unit: `SetSize(80, 4)` → View output is 4 lines, each 80 chars wide
- Unit: Frame table has deterministic output (same frameIndex → same output)
- Unit: `Init()` returns a tick command

**Commit:** `feat(ui): braille-dot audio visualizer component`

---

## Task 2: Create gradient seek bar component

**Problem:** Current `ProgressBar` uses monochrome `SeekBar()` fill. The redesign
requires a gradient fill from `Gradient1()` to `Gradient2()`.

**Fix:**

Create `internal/ui/components/gradient.go`:

```go
package components

import (
    "github.com/charmbracelet/lipgloss"
    "github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// GradientSeekBar renders a seek bar with gradient fill.
type GradientSeekBar struct {
    theme theme.Theme
    width int
}

// NewGradientSeekBar creates a gradient seek bar.
func NewGradientSeekBar(t theme.Theme) *GradientSeekBar

// SetWidth updates the bar width.
func (b *GradientSeekBar) SetWidth(width int)

// Render returns the seek bar string for the given progress.
// progressMs and durationMs are in milliseconds.
// Format: "1:41  ████████████████░░░░░░░░░░░░░░  5:30"
func (b *GradientSeekBar) Render(progressMs, durationMs int) string
```

**Implementation:**

1. Calculate fill ratio: `fillCount = int(float64(barWidth) * float64(progressMs) / float64(durationMs))`
2. For each filled position, interpolate color between `Gradient1()` and `Gradient2()`:
   - Position 0 → `Gradient1()` (green)
   - Position fillCount-1 → `Gradient2()` (yellow)
   - Linear RGB interpolation between them
3. Render each filled character (`█`) with its interpolated color using `lipgloss.NewStyle().Foreground(color)`
4. Empty positions: `░` in `Surface()` color
5. Prepend time played, append total duration

**Color interpolation helper:**
```go
// interpolateHex interpolates between two hex colors.
// t ranges from 0.0 (color1) to 1.0 (color2).
func interpolateHex(hex1, hex2 string, t float64) lipgloss.Color
```

**Files:**
- Create: `internal/ui/components/gradient.go`

**Tests:**
- Unit: `Render(0, 300000)` → all empty characters
- Unit: `Render(150000, 300000)` → 50% filled
- Unit: `Render(300000, 300000)` → all filled
- Unit: First filled char uses `Gradient1()` color
- Unit: Last filled char uses `Gradient2()` color
- Unit: Time labels correct: `"2:30"` format for 150000ms
- Unit: Width changes → bar length changes proportionally
- Unit: `durationMs=0` → safe handling (no division by zero)

**Commit:** `feat(ui): gradient seek bar component`

---

## Task 3: Create gradient volume bar component

**Problem:** Current `VolumeBar` uses monochrome `VolumeBar()` fill. The redesign
requires color bands based on volume level.

**Fix:**

Add to `internal/ui/components/gradient.go`:

```go
// GradientVolumeBar renders a volume bar with color bands.
type GradientVolumeBar struct {
    theme theme.Theme
    width int  // total width including "VOL" label and percentage
}

// NewGradientVolumeBar creates a gradient volume bar.
func NewGradientVolumeBar(t theme.Theme) *GradientVolumeBar

// SetWidth updates the bar width.
func (b *GradientVolumeBar) SetWidth(width int)

// Render returns the volume bar string.
// volume is 0-100.
// Format: "VOL  ████████░░░░░░  65%"
func (b *GradientVolumeBar) Render(volume int) string
```

**Implementation:**

1. Calculate fill from volume percentage
2. Color selection based on volume level:
   - 0-33%: all filled chars in `Gradient1()` (green/cool)
   - 34-66%: all filled chars in `Gradient2()` (yellow/warm)
   - 67-100%: all filled chars in `Gradient3()` (red/hot)
3. Format: `VOL  ████████░░░░░░  65%`
4. Clamp volume to [0, 100]

**Files:**
- Modify: `internal/ui/components/gradient.go`

**Tests:**
- Unit: `Render(0)` → no filled chars, "VOL ... 0%"
- Unit: `Render(25)` → green-colored fill (Gradient1)
- Unit: `Render(50)` → yellow-colored fill (Gradient2)
- Unit: `Render(80)` → red-colored fill (Gradient3)
- Unit: `Render(100)` → all filled, red
- Unit: Volume clamped: `Render(150)` → treated as 100
- Unit: Volume clamped: `Render(-5)` → treated as 0
- Unit: Width changes → bar length adjusts

**Commit:** `feat(ui): gradient volume bar with color bands`

---

## Task 4: Integration tests

**Fix:**

**Files:**
- Create: `internal/ui/components/visualizer_test.go`
- Create: `internal/ui/components/gradient_test.go`

**Tests:**
- Integration: Visualizer lifecycle — Init → multiple VisualizerTickMsg updates → View changes each frame
- Integration: Visualizer play/pause cycle — play→tick→frame advances, pause→tick→frame freezes, play→tick→frame resumes
- Integration: Seek bar at various progress points → gradient visible in output
- Integration: Volume bar threshold transitions: 33→34 (green→yellow), 66→67 (yellow→red)
- Integration: All components render within specified width (no overflow)
- Integration: All components use theme tokens (verify no hardcoded hex in output)

**Commit:** `test(ui): visualizer and gradient bar integration tests`

---

## Acceptance Criteria

- [ ] Visualizer animates braille characters on 200ms tick when playing
- [ ] Visualizer shows flat-line when paused
- [ ] Frame table has 40 deterministic patterns (no randomness in View)
- [ ] Visualizer adapts to width/height via `SetSize()`
- [ ] Gradient seek bar interpolates color from `Gradient1()` to `Gradient2()`
- [ ] Volume bar uses 3 color bands: green (0-33%), yellow (34-66%), red (67-100%)
- [ ] All colors come from Theme interface tokens
- [ ] `interpolateHex` correctly handles RGB color interpolation
- [ ] No panics on edge cases (zero width, zero duration, extreme volumes)
- [ ] `make ci` passes

---

## Notes

- The visualizer tick (200ms) is separate from the app tick (1000ms). The NowPlaying pane
  will need to handle both `VisualizerTickMsg` and `TickMsg` in its Update method.
- The frame table is generated once at construction time (or on `SetSize()` when dimensions
  change). It is NOT regenerated on every tick — `View()` just indexes into the table.
- The old `ProgressBar` and `VolumeBar` components remain until Feature 45 (NowPlaying pane)
  replaces them. No deletion in this feature.
- The `interpolateHex` function parses hex strings like `#00ff88` into RGB, interpolates,
  and returns a new `lipgloss.Color`. This is a pure utility function with no Theme dependency.
- Braille dot patterns reference: U+2800 = empty (⠀), dots are numbered 1-8 in a 2×4 grid.
  Each dot adds a power of 2 to the codepoint offset. Left column: dots 1,2,3,7.
  Right column: dots 4,5,6,8.
