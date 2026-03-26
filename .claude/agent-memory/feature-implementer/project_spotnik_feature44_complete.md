---
name: project_spotnik_feature44_complete
description: Feature 44 (Visualizer + Gradient Bars): Visualizer struct, GradientSeekBar, GradientVolumeBar, interpolateHex, divide-by-zero fix, braille encoding
type: project
---

## Feature 44 — Visualizer + Gradient Bars

**What was built:**
- `internal/ui/components/visualizer.go` — Visualizer with 40-frame precomputed table, 200ms tick, sine-wave patterns, VisualizerTickMsg, FrameIndex() accessor
- `internal/ui/components/gradient.go` — GradientSeekBar (Gradient1→Gradient2 interpolation), GradientVolumeBar (3-band colors), interpolateHex helper
- `internal/ui/components/visualizer_test.go` — 12 unit tests
- `internal/ui/components/gradient_test.go` — 23 unit tests
- `internal/ui/components/visualizer_gradient_integration_test.go` — 6 integration tests

**Key files:**
- `internal/ui/components/visualizer.go` — frame table generated once at SetSize(), View() pure (just indexes table)
- `internal/ui/components/gradient.go` — interpolateHex() is a pure function with no Theme dependency; lerp() uses float64 arithmetic to avoid uint8 underflow

**Patterns established:**
- Visualizer follows Bubble Tea component pattern: Init() → Update(msg) tea.Cmd → View() string (not Model.Update returning (Model, Cmd))
- Frame table precomputed once: `generateFrames(width, height)` called in SetSize() when dimensions change; View() just does `frames[frameIndex]`
- FrameIndex() exported for integration tests in external `components_test` package
- GradientSeekBar/VolumeBar are stateless value structs with SetWidth() — no Init/Update needed

**Braille encoding:**
- Code uses ⠀⡀⡠⡰⣰ (U+2800, 2840, 2860, 2870, 28F0) matching the spec's exact height mapping example
- These are NOT pure left-column fills — they mix left/right dots for a visually thicker bar effect
- This is intentional per spec; the comment in brailleChar now correctly documents the codepoints without claiming which "column" they fill

**Gotchas:**
- `lerp(a, b uint8, t float64) uint8` — must use `float64(b) - float64(a)` NOT `float64(b-a)` because uint8 subtraction underflows when b < a
- `Update()` divide-by-zero: `(frameIndex+1) % len(frames)` panics when frames is nil (SetSize never called). Guard with `len(v.frames) > 0` check BEFORE the modulo
- Volume bar test helper `newTestVolumeBar` already existed in volume_test.go (same package). Named the gradient version `newTestGradientVolumeBar` to avoid conflict
- Integration tests are in the external `components_test` package — can only access exported symbols, so FrameIndex() had to be exported

**Testing notes:**
- `interpolateHex("#ff0000", "#0000ff", 0.5)` returns `#800080` (R=128, G=0, B=128) because 255×0.5=127.5 rounds to 128 via math.Round
- Width tests compare `len(out80) > len(out40)` using raw byte length (includes ANSI in colored terminals, but in test env lipgloss renders without ANSI)
- Visualizer SetSize width tests use `len([]rune(lines[0]))` — valid because lipgloss adds no ANSI codes in non-TTY test env
- Coverage: 93.9% components, 84.9% overall
