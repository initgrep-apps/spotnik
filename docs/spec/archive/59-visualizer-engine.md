# Feature 59 — Visualizer Engine

> **Feature:** Extract visualization logic into a dedicated `viz/` package with per-row
> color gradient, block character rendering mode, and 7 animation patterns.

## Context

The current visualizer (`internal/ui/components/visualizer.go`) renders braille-dot
patterns in a single color (`VisualizerFg()`), hard-caps height at 4 lines, and supports
3 animation patterns. The NowPlaying redesign (design spec:
`docs/superpowers/specs/2026-03-27-nowplaying-redesign.md`) requires:

- Per-row color gradient (green base → yellow mid → red peaks) using existing theme tokens
- Block character rendering mode (`█▓▒░■`) alongside braille
- 7 animation patterns (3 upgraded braille + 4 new block/braille combos)
- No height cap — the engine accepts any height from the pane
- A clean `Renderer` interface for extensible pattern+renderer combinations

This feature creates the new `viz/` package. Feature 60 integrates it into the
NowPlaying pane and deletes the old visualizer.

**Design reference:** `docs/superpowers/specs/2026-03-27-nowplaying-redesign.md` §3
(Visualizer Engine)

**Depends on:** Feature 40 (theme tokens: `Gradient1/2/3`, `VisualizerFg`)

---

## Design Diagram

```
internal/ui/components/viz/
├── engine.go        — Engine struct, pattern registry, frame orchestration
├── pattern.go       — Pattern type, all 7 pattern definitions
├── braille.go       — Braille renderer: column heights → braille strings + per-row color
├── block.go         — Block renderer: column heights → block chars + per-row color
├── frame.go         — Frame and StyledLine types
└── engine_test.go   — Table-driven tests for all patterns and renderers

Per-row color gradient (top → bottom of frame):

  Row 0 (peaks):  Gradient3() — #ff5555 (red/hot)
  Row 1:          Gradient3()
  Row 2 (mid):    Gradient2() — #ffcc00 (yellow/warm)
  Row 3:          Gradient2()
  Row 4 (base):   Gradient1() — #00ff88 (green/cool)
  Row 5:          Gradient1()

Pattern list:
  0. Braille — Dual sine wave (upgraded from old pattern 0)
  1. Braille — Standing wave (upgraded from old pattern 1)
  2. Braille — Pulse/ripple (upgraded from old pattern 2)
  3. Block — Dense equalizer
  4. Block — Waveform/sine
  5. Block — Sparse/low amplitude
  6. Braille — Mid-density organic
```

---

## Task 1: Create Frame and StyledLine types

**Problem:** No shared types exist for colored visualization frames.

**Fix:**

Create `internal/ui/components/viz/frame.go`:

```go
package viz

import "github.com/charmbracelet/lipgloss"

// StyledLine is a single display row with its text content and assigned color.
type StyledLine struct {
    Text  string
    Color lipgloss.Color
}

// Frame is a slice of StyledLines representing one animation frame.
type Frame []StyledLine
```

**Files:**
- Create: `internal/ui/components/viz/frame.go`

**Tests:**
- Unit: `StyledLine` stores text and color correctly
- Unit: `Frame` slice can be created and indexed
- Build: package compiles with no errors

**Commit:** `feat(viz): Frame and StyledLine types`

---

## Task 2: Create Renderer interface and braille renderer

**Problem:** The current visualizer has braille rendering baked into a monolithic file
with no interface for alternative renderers.

**Fix:**

Create `internal/ui/components/viz/braille.go`:

```go
package viz

import "github.com/charmbracelet/lipgloss"

// Renderer produces a Frame from column heights, dimensions, and per-row colors.
type Renderer interface {
    RenderFrame(width, height int, colHeights []int, colors []lipgloss.Color) Frame
}

// BrailleRenderer renders column heights as Unicode braille characters (U+2800 block).
// Each character cell represents a 2×4 dot grid.
type BrailleRenderer struct{}

// RenderFrame converts column heights to braille display lines with per-row coloring.
func (r BrailleRenderer) RenderFrame(width, height int, colHeights []int, colors []lipgloss.Color) Frame
```

**Implementation details:**

1. Port the existing `renderFrame` and `brailleChar` logic from `visualizer.go`
2. Each braille column maps to a height value (0 to `height*4` dot rows)
3. Iterate rows top-to-bottom, for each column compute which dots are filled
4. Assign `colors[rowIndex]` to each output `StyledLine`
5. Handle edge cases: empty colHeights, zero width, zero height

The braille encoding is identical to the existing implementation:
- 0 dots: `⠀` (U+2800)
- 1 dot: `⡀` (U+2840)
- 2 dots: `⡠` (U+2860)
- 3 dots: `⡰` (U+2870)
- 4 dots: `⣰` (U+28F0)

**Files:**
- Create: `internal/ui/components/viz/braille.go`

**Tests:**
- Unit: `BrailleRenderer` implements `Renderer` interface (compile-time check)
- Unit: `RenderFrame(10, 3, colHeights, colors)` → returns Frame with 3 StyledLines
- Unit: Each StyledLine.Text contains only braille runes (U+2800–U+28FF range)
- Unit: Each StyledLine.Color matches the corresponding `colors[i]` input
- Unit: Full column height → top row has filled braille chars
- Unit: Zero column heights → all rows are blank braille (U+2800)
- Unit: Frame width matches input width (each line has `width` runes)
- Edge: Zero width → empty Frame
- Edge: Zero height → empty Frame

**Commit:** `feat(viz): Renderer interface and braille renderer`

---

## Task 3: Create block renderer

**Problem:** No block-character rendering mode exists.

**Fix:**

Create `internal/ui/components/viz/block.go`:

```go
package viz

import "github.com/charmbracelet/lipgloss"

// BlockRenderer renders column heights as block characters (█▓▒░ and ■).
// Visually heavier and coarser than braille.
type BlockRenderer struct{}

// RenderFrame converts column heights to block display lines with per-row coloring.
func (r BlockRenderer) RenderFrame(width, height int, colHeights []int, colors []lipgloss.Color) Frame
```

**Implementation details:**

1. Each column is 1 character wide, 1 row tall per display line
2. Column heights map to filled rows bottom-up:
   - Full: `█` (U+2588)
   - Empty: ` ` (space)
3. Assign `colors[rowIndex]` to each output `StyledLine`
4. Both braille and block characters are treated as 1 column wide per rune.
   Block characters (`█`, `■`) may render as 2 columns in some East Asian terminal
   configurations — this is a known limitation, not something we solve for.

**Files:**
- Create: `internal/ui/components/viz/block.go`

**Tests:**
- Unit: `BlockRenderer` implements `Renderer` interface (compile-time check)
- Unit: `RenderFrame(10, 4, colHeights, colors)` → returns Frame with 4 StyledLines
- Unit: Each StyledLine.Text contains only block chars (`█`) or spaces
- Unit: Each StyledLine.Color matches the corresponding `colors[i]` input
- Unit: Full column height → all rows filled with `█`
- Unit: Zero column heights → all rows are spaces
- Unit: Frame width matches input width
- Edge: Zero width → empty Frame
- Edge: Zero height → empty Frame

**Commit:** `feat(viz): block character renderer`

---

## Task 4: Create Pattern type and all 7 pattern definitions

**Problem:** No extensible pattern registry exists. Patterns are hard-coded switch
cases in the current visualizer.

**Fix:**

Create `internal/ui/components/viz/pattern.go`:

```go
package viz

// HeightFunc computes column heights for a given frame index.
// width: number of columns, maxHeight: maximum dot/block rows,
// frameIdx: current frame (0..numFrames-1).
// Returns a slice of length `width` with values in [0, maxHeight].
type HeightFunc func(width, maxHeight, frameIdx int) []int

// Pattern combines a name, a Renderer, and a HeightFunc.
type Pattern struct {
    Name       string
    Renderer   Renderer
    HeightFunc HeightFunc
}

// Patterns returns the full ordered list of available patterns.
func Patterns() []Pattern
```

**Implementation details:**

1. **Pattern 0 — Braille Dual Sine Wave:** Port existing pattern 0 from `visualizer.go`.
   Two sine waves at different frequencies, flowing organic movement.
   Uses `BrailleRenderer`.

2. **Pattern 1 — Braille Standing Wave:** Port existing pattern 1.
   Counter-propagating sine waves creating stationary nodes/antinodes.
   Uses `BrailleRenderer`.

3. **Pattern 2 — Braille Pulse/Ripple:** Port existing pattern 2.
   Narrow Gaussian peak traveling left-to-right with trailing ripple.
   Uses `BrailleRenderer`.

4. **Pattern 3 — Block Dense Equalizer:** Full-height bars with slight deterministic
   variation per column. Dense, heavy look. Uses `BlockRenderer`.

5. **Pattern 4 — Block Waveform/Sine:** Smooth sine-based heights in block chars.
   Clean, flowing appearance. Uses `BlockRenderer`.

6. **Pattern 5 — Block Sparse/Low Amplitude:** Low overall height with occasional
   deterministic spikes. Ambient, minimal feel. Uses `BlockRenderer`.

7. **Pattern 6 — Braille Mid-density Organic:** Multi-frequency sine composition
   (no external noise library). Natural, unpredictable movement.
   Uses `BrailleRenderer`.

All height functions use deterministic math (sine, cosine, Gaussian) — no `math/rand`.
Frame count is 40 per pattern.

**Files:**
- Create: `internal/ui/components/viz/pattern.go`

**Tests:**
- Unit: `Patterns()` returns exactly 7 patterns
- Unit: Each pattern has a non-empty Name
- Unit: Each pattern has a non-nil Renderer
- Unit: Each pattern has a non-nil HeightFunc
- Unit: Pattern 0-2 use `BrailleRenderer`
- Unit: Pattern 3-5 use `BlockRenderer`
- Unit: Pattern 6 uses `BrailleRenderer`
- Unit: Each HeightFunc returns slice of length `width`
- Unit: Each HeightFunc returns values in [0, maxHeight]
- Unit: HeightFunc output is deterministic (same inputs → same output)
- Unit: Different patterns produce different height profiles for same frame index

**Commit:** `feat(viz): 7 animation patterns with height functions`

---

## Task 5: Create Engine with frame orchestration

**Problem:** No engine exists to manage pattern registry, frame precomputation,
and animation state.

**Fix:**

Create `internal/ui/components/viz/engine.go`:

```go
package viz

import (
    "time"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// TickMsg is sent on the visualizer's animation tick.
// Replaces components.VisualizerTickMsg.
type TickMsg time.Time

// Engine manages the pattern registry, current pattern, precomputed frame table,
// frame index advancement, and playing/paused state.
type Engine struct {
    theme      theme.Theme
    patterns   []Pattern
    patternIdx int
    frames     []Frame   // precomputed frame table [frameIndex]
    frameIdx   int
    playing    bool
    width      int
    height     int
    interval   time.Duration
}

// NewEngine creates an Engine with all registered patterns.
// The theme is used to resolve gradient colors during frame precomputation.
func NewEngine(th theme.Theme) *Engine

// SetSize updates dimensions and regenerates the precomputed frame table.
// No height cap — the engine accepts any height passed by the pane.
func (e *Engine) SetSize(width, height int)

// SetPlaying controls animation state.
func (e *Engine) SetPlaying(playing bool)

// Advance increments frameIdx when playing. Called by the pane on each tick.
func (e *Engine) Advance()

// CurrentFrame returns the current frame. Returns a blank frame when paused.
func (e *Engine) CurrentFrame() Frame

// CyclePattern advances to the next pattern and regenerates frames.
func (e *Engine) CyclePattern()

// Pattern returns the current pattern index.
func (e *Engine) Pattern() int

// PatternCount returns the total number of registered patterns.
func (e *Engine) PatternCount() int

// FrameIndex returns the current frame index (for testing).
func (e *Engine) FrameIndex() int

// Init returns the initial tick command.
func (e *Engine) Init() tea.Cmd

// Update handles TickMsg to re-arm the tick loop.
func (e *Engine) Update(msg tea.Msg) tea.Cmd

// tickCmd returns a tea.Tick for the next animation frame.
func (e *Engine) tickCmd() tea.Cmd
```

**Implementation details:**

1. **Constructor:** `NewEngine(th)` initializes with `Patterns()` list, pattern 0,
   200ms interval.

2. **Frame precomputation:** On `SetSize()` or `CyclePattern()`, generate 40 frames:
   - For each frame index 0..39:
     - Call `pattern.HeightFunc(width, maxHeight, frameIdx)` → colHeights
     - Build per-row color slice using gradient assignment (see below)
     - Call `pattern.Renderer.RenderFrame(width, height, colHeights, colors)` → Frame
   - Store in `e.frames`

3. **Per-row color gradient:** Colors assigned by row position relative to total height:
   - Top 1/3 rows (peaks): `theme.Gradient3()` — red/hot
   - Middle 1/3 rows: `theme.Gradient2()` — yellow/warm
   - Bottom 1/3 rows (base): `theme.Gradient1()` — green/cool
   - Color assignment happens during frame generation, not during View rendering.

4. **Advance:** Increments `frameIdx` modulo `len(frames)` only when `playing`.

5. **CurrentFrame:** Returns `frames[frameIdx]` when playing. Returns a blank frame
   (all empty StyledLines) when paused.

6. **CyclePattern:** `patternIdx = (patternIdx + 1) % len(patterns)`, then regenerate.

7. **Tick loop:** Same pattern as the old visualizer — `tea.Tick(interval, ...)` with
   `TickMsg` type.

**Files:**
- Create: `internal/ui/components/viz/engine.go`

**Tests:**
- Unit: `NewEngine` creates engine with 7 patterns, pattern 0 selected
- Unit: `PatternCount()` returns 7
- Unit: `Pattern()` returns 0 initially
- Unit: `SetSize(40, 6)` → `CurrentFrame()` returns Frame with 6 StyledLines when playing
- Unit: `SetPlaying(true)` + `Advance()` → `FrameIndex()` increments from 0 to 1
- Unit: `SetPlaying(false)` + `Advance()` → `FrameIndex()` stays at 0
- Unit: `CurrentFrame()` when paused returns blank frame (all empty text)
- Unit: `CurrentFrame()` when playing returns non-empty frame
- Unit: Frame wraps after 40 advances (frameIdx returns to 0)
- Unit: `CyclePattern()` → pattern index advances to 1
- Unit: `CyclePattern()` 7 times → wraps back to 0
- Unit: `CyclePattern()` regenerates frames (frame content changes)
- Unit: `Init()` returns a tick command (non-nil)
- Unit: `Update(TickMsg{})` returns a tick command (re-arms loop)
- Unit: Per-row colors: top rows use Gradient3, middle use Gradient2, bottom use Gradient1
- Unit: `SetSize` with height=1 → single-row frame with Gradient1 color
- Unit: `SetSize` with height=0 → empty frames, no panic
- Unit: `SetSize` regenerates frames with current pattern
- Edge: `Advance()` before `SetSize()` → no panic (nil frames handled)
- Edge: `CurrentFrame()` before `SetSize()` → returns empty Frame

**Commit:** `feat(viz): Engine with frame precomputation and pattern cycling`

---

## Task 6: Comprehensive engine tests

**Problem:** Need thorough coverage across all patterns and renderers to reach 80%
minimum.

**Fix:**

Create `internal/ui/components/viz/engine_test.go` with table-driven tests:

**Files:**
- Create: `internal/ui/components/viz/engine_test.go`

**Tests:**
- Table-driven: For each of the 7 patterns:
  - Frame dimensions match requested width × height
  - Non-empty output when playing
  - Blank output when paused
  - Braille patterns (0,1,2,6) produce only braille runes (U+2800–U+28FF)
  - Block patterns (3,4,5) produce only block chars (`█`) or spaces
  - Color assignment follows gradient (top=Gradient3, mid=Gradient2, bottom=Gradient1)
  - Deterministic: same frameIndex → same output
  - Different frame indices → different output (at least some frames differ)
- Integration: Full lifecycle — NewEngine → SetSize → SetPlaying → Advance × N → CurrentFrame
- Integration: Pattern cycling through all 7 → back to 0
- Integration: Resize mid-animation → frames regenerate, frameIdx resets to 0
- Edge: Width=1 → single-column frames for all patterns
- Edge: Height=1 → single-row frames for all patterns
- Edge: Large dimensions (200×20) → no panic, reasonable performance

**Commit:** `test(viz): comprehensive engine and pattern tests`

---

## Acceptance Criteria

- [ ] `viz/` package compiles independently with no imports from `components/` or `panes/`
- [ ] `Renderer` interface has two implementations: `BrailleRenderer` and `BlockRenderer`
- [ ] 7 patterns registered: 4 braille, 3 block
- [ ] Per-row color gradient uses `Gradient1/2/3` theme tokens (no hardcoded hex)
- [ ] Engine has no height cap — accepts any height via `SetSize()`
- [ ] Frame table precomputed (40 frames) on `SetSize()` and `CyclePattern()`
- [ ] `CurrentFrame()` returns blank frame when paused
- [ ] `CyclePattern()` cycles through all 7 patterns and wraps
- [ ] `TickMsg` type defined in `viz` package (replaces `components.VisualizerTickMsg`)
- [ ] All height functions are deterministic (no `math/rand`)
- [ ] 80%+ test coverage on the `viz/` package
- [ ] `make ci` passes

---

## Notes

- The old `internal/ui/components/visualizer.go` is NOT deleted in this feature. It is
  deleted in Feature 60 after the NowPlaying pane migrates to the new engine.
- The `TickMsg` type in `viz` replaces `components.VisualizerTickMsg`. The import update
  in `app.go` happens in Feature 60.
- Block characters (`█`, `■`) may render as 2 columns in some East Asian terminal
  configurations — this is a known limitation, not something we solve for.
- The `VisualizerFg()` theme token becomes a fallback for single-color mode if ever
  needed, but the default rendering uses the gradient tokens.
- Adding new patterns later requires only defining a `HeightFunc` and choosing a
  `Renderer` — no changes to the engine or pane code.
