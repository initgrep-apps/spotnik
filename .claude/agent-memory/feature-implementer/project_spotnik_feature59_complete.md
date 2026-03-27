---
name: project_spotnik_feature59_complete
description: Feature 59 (Visualizer Engine): viz package, Renderer interface, BrailleRenderer, BlockRenderer, 7 patterns, Engine with frame precomputation
type: project
---

## Feature 59 — Visualizer Engine

**What was built:**
- `internal/ui/components/viz/frame.go` — StyledLine (Text+Color) and Frame ([]StyledLine) types; package doc lives here
- `internal/ui/components/viz/braille.go` — Renderer interface + BrailleRenderer; brailleChar() ported faithfully from visualizer.go
- `internal/ui/components/viz/block.go` — BlockRenderer using '█' and space; fills from bottom up
- `internal/ui/components/viz/pattern.go` — HeightFunc type, Pattern struct, Patterns() registry with 7 entries, all height functions, clamp01, phaseFor
- `internal/ui/components/viz/engine.go` — Engine struct, NewEngine, SetSize, Advance, CurrentFrame, CyclePattern, TickMsg, Init, Update, generateFrames, buildColors
- `internal/ui/components/viz/engine_test.go` — 91 tests, 98.3% coverage

**Key files:**
- The old `internal/ui/components/visualizer.go` is NOT deleted — Feature 60 does that after migration
- `internal/ui/components/viz/engine.go` imports `internal/ui/theme` directly (no circular import issue)

**Patterns established:**
- `viz/` package is independent — imports only `theme`, `bubbletea`, `lipgloss`; never imports `components/` or `panes/`
- `TickMsg` defined as `type TickMsg time.Time` in `viz` package (replaces `components.VisualizerTickMsg`)
- `Renderer` interface has one method: `RenderFrame(width, height int, colHeights []int, colors []lipgloss.Color) Frame`
- `Engine.Update()` returns `tea.Cmd` (not `(tea.Model, tea.Cmd)`) — component pattern, not top-level model
- Frame precomputation: `generateFrames()` called on `SetSize()` and `CyclePattern()`, resets `frameIdx` to 0
- `buildColors()` divides height into thirds: top→Gradient3, mid→Gradient2, bottom→Gradient1
- `generateFrames()` type-switches on `BrailleRenderer` specifically to set `maxHeight = height*4` vs `height` for block
- `CurrentFrame()` when paused returns a blank Frame (empty Text strings) with correct colors — not nil

**Gotchas:**
- Dead variable `spike := 0.0` in `heightBlockSparse` caught during review — fixed to `spike :=` at assignment point
- `tickCmd` inner closure body is unreachable in tests (standard limitation for all tick-based code) — results in 50% coverage on `tickCmd`, acceptable
- `SetSize` clamps `width < 1` to 1 but allows `height = 0` — asymmetric but intentional per spec ("no height cap")
- Pattern 5 (Block Sparse) has Gaussian spike that only shows for some frame indices — all frames are still deterministic

**Testing notes:**
- Compile-time interface checks: `var _ Renderer = BrailleRenderer{}` and `var _ Renderer = BlockRenderer{}`
- Use `theme.Load("black")` in tests — no mock theme needed since the real theme is pure data
- `TestEngine_Update_NonTickMsg_ReturnsNil` — pass a string as msg to hit the nil-return branch
- `TestEngine_SetSize_SameDimensions_NoReset` — call SetSize twice with same args; verify frameIdx NOT reset
- Coverage achieved: 98.3% on viz package, 86.5% overall
