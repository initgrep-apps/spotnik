---
title: "Visualizer ASCII renderer + engine selects renderer by ActiveMode"
feature: 13-tui-design-system
status: open
---

## Background

The audio visualizer (`internal/ui/components/viz/`) ships two renderers — braille
(`braille.go`, codepoints U+2800–U+28FF) and block (`block.go`, U+2588 `█` and friends)
— and selects between them at engine init via the `viz.style` config knob. Both
renderers are unicode-only by construction; there is no per-render-call mode switch.

Under `ui.glyphs = "ascii"` (or on a non-UTF-8 terminal) the visualizer renders one of:
- mojibake — terminal cannot draw braille codepoints,
- a column of unicode `█` from the block renderer,
- a blank surface.

Audit §4.3 decision: add a third `AsciiBarsRenderer` that draws columns using `#`
(filled), `=` (half), `.` (empty) at 4-level vertical resolution. The visualizer stays
present (degraded resolution) in ASCII mode rather than disappearing or rendering
mojibake. The engine selects renderer based on `uikit.ActiveMode()` plus the existing
`viz.style` knob (which only chooses braille vs. block in unicode mode).

`viz/block.go:45` also hardcodes the unicode `█` literal even though the catalogue has
`GlyphBarFull`. The block renderer's `█` swap to `GlyphFor(GlyphBarFull, mode)` ships in
story 191 (it's a one-line literal swap and pairs with the rest of the inline-glyph
sweep). This story focuses on the new ASCII renderer + engine wiring.

**Depends on:** story 183 (catalogue audit). Uses no new catalogue rows.

**Plan tasks:** 4.5, 4.6 in `docs/superpowers/plans/2026-04-29-glyph-fallback.md`.

**Files created:** `internal/ui/components/viz/ascii_bars.go`,
`internal/ui/components/viz/ascii_bars_test.go`. **Modified:**
`internal/ui/components/viz/engine.go`,
`internal/ui/components/viz/engine_test.go`.

## Design

### `AsciiBarsRenderer`

```go
package viz

import (
    "strings"

    "github.com/charmbracelet/lipgloss"
    "github.com/initgrep-apps/spotnik/internal/ui/theme"
)

type AsciiBarsRenderer struct{}

func NewAsciiBarsRenderer() *AsciiBarsRenderer { return &AsciiBarsRenderer{} }

func (r *AsciiBarsRenderer) MaxHeight() int { return 4 }

func (r *AsciiBarsRenderer) Render(cols []float64, height int, th theme.Theme) string {
    if height < 1 {
        height = 1
    }
    style := lipgloss.NewStyle().Foreground(th.Gradient1())

    rows := make([]string, height)
    for row := 0; row < height; row++ {
        threshold := 1.0 - float64(row)/float64(height)
        var sb strings.Builder
        for _, h := range cols {
            switch {
            case h >= threshold:
                sb.WriteString("#")
            case h >= threshold-0.125:
                sb.WriteString("=")
            case h >= threshold-0.25:
                sb.WriteString(".")
            default:
                sb.WriteString(" ")
            }
        }
        rows[row] = style.Render(sb.String())
    }
    return strings.Join(rows, "\n")
}
```

The renderer must conform to whatever `Pattern` / `Renderer` interface the existing
package expects. The implementer reads `pattern.go` / `block.go` first to copy the
exact signature, then adapts the helper above to fit. The contract is "produces a
multi-line ASCII bar visualization at 4 vertical levels."

### Engine renderer selection

`internal/ui/components/viz/engine.go` — at the renderer-selection point, branch on
`uikit.ActiveMode()`:

```go
import (
    "github.com/initgrep-apps/spotnik/internal/uikit"
)

func (e *Engine) selectRenderer() Renderer {
    if uikit.ActiveMode() == uikit.GlyphASCII {
        return NewAsciiBarsRenderer()
    }
    return e.unicodeRenderer()
}
```

`unicodeRenderer()` retains the existing braille-vs-block selection by `viz.style`.
The exact integration shape depends on whether `Engine` already abstracts renderers
or computes frames inline — implementer adapts.

## Acceptance Criteria

- [ ] `internal/ui/components/viz/ascii_bars.go` defines `AsciiBarsRenderer` matching
      the existing `viz` renderer contract
- [ ] `MaxHeight()` reports `4` (4 vertical levels: empty / `.` / `=` / `#`)
- [ ] `AsciiBarsRenderer.Render` produces a multi-line string composed only of `#`,
      `=`, `.`, ` `, `\n`, and ANSI escape sequences — no braille / block /
      half-block characters
- [ ] `internal/ui/components/viz/engine.go` selects `AsciiBarsRenderer` when
      `uikit.ActiveMode() == uikit.GlyphASCII`; the existing braille-vs-block selection
      remains active in unicode mode
- [ ] `TestAsciiBars_AllAscii` confirms ASCII output contains no banned characters
      (`█`, `▉`, `▊`, `▋`, `▌`, `▍`, `▎`, `▏`, `⠀`, …)
- [ ] `TestAsciiBars_MaxLevels` confirms `MaxHeight() == 4`
- [ ] `TestEngine_SelectsAsciiRendererInAsciiMode` confirms the engine picks the
      ASCII renderer in ASCII mode
- [ ] All existing visualizer tests pass unchanged in unicode mode
- [ ] Manual: launching with `ui.glyphs = "ascii"` shows the visualizer column-bar in
      `# = .` form (verified during Phase 11 smoke test in story 192)
- [ ] `make ci` → PASS

## Tasks

Step-by-step: Tasks 4.5, 4.6 in `docs/superpowers/plans/2026-04-29-glyph-fallback.md`.

- [ ] Branch: `feat/13-viz-ascii-renderer`
- [ ] Read `internal/ui/components/viz/pattern.go` and `block.go` to record the
      `Renderer` interface shape
- [ ] Write failing `TestAsciiBars_AllAscii` and `TestAsciiBars_MaxLevels` → FAIL
- [ ] Create `internal/ui/components/viz/ascii_bars.go` implementing
      `AsciiBarsRenderer` to match the recorded interface → tests PASS
- [ ] Commit: `feat(viz): add AsciiBarsRenderer for ascii-mode visualizer fallback`
- [ ] Write failing `TestEngine_SelectsAsciiRendererInAsciiMode` → FAIL
- [ ] Update `engine.go` renderer-selection point to branch on
      `uikit.ActiveMode()` → PASS
- [ ] Commit: `fix(viz): select AsciiBarsRenderer when uikit.ActiveMode is GlyphASCII`
- [ ] `make ci` → PASS
- [ ] Push branch + open PR
