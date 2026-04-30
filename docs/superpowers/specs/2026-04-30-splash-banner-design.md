# Splash Banner — Dual-Mode Design

**Date:** 2026-04-30

## Problem

The splash screen uses `go-figure` to render the SPOTNIK banner. The library always
emits unicode box-drawing characters regardless of the active glyph mode, so `ui.glyphs
= "ascii"` terminals still receive unicode art. The library also adds a runtime
dependency for a one-time static string.

## Decision

Drop `go-figure`. Hardcode two banner string constants in `internal/app/splash.go` and
select between them via `uikit.ActiveMode()` at render time.

| Mode | Banner style |
|---|---|
| `GlyphUnicode` | ANSI Shadow — full-block `█` + box-drawing `╗╔╝╚` for filled block letters |
| `GlyphASCII` | Open-outline figlet style — `_`, `/`, `\`, `|` characters only |

## Files

| File | Change |
|---|---|
| `internal/app/splash.go` | Replace `figure.NewFigure(...)` with const-based `bannerUnicode` / `bannerASCII`; remove go-figure import |
| `internal/app/splash_test.go` | Assert `█` present in unicode mode, absent in ascii mode; update stale comment |
| `internal/app/render.go` | Update line-500 comment (mentions go-figure) |
| `go.mod` / `go.sum` | Remove `github.com/common-nighthawk/go-figure` via `go mod tidy` |

## Constraints

- `bannerUnicode` and `bannerASCII` are raw string literals (no backticks inside).
- Selection happens inside `renderSplashView` via `uikit.ActiveMode()` — same pattern
  used by `uikit.RoundedBorder()` throughout the splash and onboarding screens.
- The `╗╔╝╚` characters in `bannerUnicode` are ASCII art, not structural pane borders;
  the tui.md ban on double corners applies to pane chrome only.

## Verification

```bash
make test          # TestRenderSplashView_UnicodeMode / AsciiMode pass
make ci            # full lint + coverage gate
go mod tidy        # no go-figure entry remains in go.mod
```
