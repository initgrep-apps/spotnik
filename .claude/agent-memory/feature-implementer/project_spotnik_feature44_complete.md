---
name: project_spotnik_feature44_complete
description: Feature 44 (Visualizer + Gradient Bars): Visualizer struct, GradientSeekBar, GradientVolumeBar, interpolateHex, divide-by-zero fix, braille encoding
type: project
---

## Feature 44 — Visualizer + Gradient Bars

**What was built:**
- `internal/ui/components/visualizer.go` — Visualizer w/ 40-frame precomputed table, 200ms tick, sine-wave patterns, VisualizerTickMsg, FrameIndex() accessor
- `internal/ui/components/gradient.go` — GradientSeekBar (Gradient1→Gradient2 interp), GradientVolumeBar (3-band colors), interpolateHex helper
- `internal/ui/components/visualizer_test.go` — 12 unit tests
- `internal/ui/components/gradient_test.go` — 23 unit tests
- `internal/ui/components/visualizer_gradient_integration_test.go` — 6 integration tests

**Key files:**
- `internal/ui/components/visualizer.go` — frame table gen once at SetSize(), View() pure (indexes table)
- `internal/ui/components/gradient.go` — interpolateHex() pure func, no Theme dep; lerp() uses float64 to avoid uint8 underflow

**Patterns established:**
- Visualizer follows Bubble Tea pattern: Init() → Update(msg) tea.Cmd → View() string (not Model.Update returning (Model, Cmd))
- Frame table precomputed once: `generateFrames(width, height)` called in SetSize() on dim change; View() does `frames[frameIndex]`
- FrameIndex() exported for integration tests in external `components_test` pkg
- GradientSeekBar/VolumeBar stateless value structs w/ SetWidth() — no Init/Update

**Braille encoding:**
- Code uses ⠀⡀⡠⡰⣰ (U+2800, 2840, 2860, 2870, 28F0) matching spec's height mapping example
- NOT pure left-column fills — mix left/right dots for visually thicker bar
- Intentional per spec; comment in brailleChar documents codepoints w/o claiming which "column" fills

**Gotchas:**
- `lerp(a, b uint8, t float64) uint8` — must use `float64(b) - float64(a)` NOT `float64(b-a)` cuz uint8 subtraction underflows when b < a
- `Update()` divide-by-zero: `(frameIndex+1) % len(frames)` panics when frames nil (SetSize never called). Guard w/ `len(v.frames) > 0` BEFORE modulo
- Volume bar helper `newTestVolumeBar` already in volume_test.go (same pkg). Named gradient version `newTestGradientVolumeBar` to avoid conflict
- Integration tests in external `components_test` pkg — only exported symbols, so FrameIndex() exported

**Testing notes:**
- `interpolateHex("#ff0000", "#0000ff", 0.5)` returns `#800080` (R=128, G=0, B=128) cuz 255×0.5=127.5 rounds to 128 via math.Round
- Width tests compare `len(out80) > len(out40)` raw byte length (incl ANSI in colored terms, but test env lipgloss renders w/o ANSI)
- Visualizer SetSize width tests use `len([]rune(lines[0]))` — valid cuz lipgloss adds no ANSI in non-TTY test env
- Coverage: 93.9% components, 84.9% overall