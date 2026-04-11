---
title: "Repeat-Track Superscript Icon"
feature: 25-nowplaying-controls-polish
status: done
---

## Background

`Controls.Render()` in `internal/ui/components/controls.go` renders the
repeat-track mode as `"↻1"` — a plain ASCII `1` appended to the repeat arrow
glyph. In a terminal at small font sizes this reads as a label ("repeat 1
time") rather than as a compact modifier. The Unicode superscript one `¹`
(U+00B9) sits visually above the baseline and integrates into the glyph
cluster naturally.

**Depends on:** nothing — self-contained single-char swap.

## Design

### controls.go change

`internal/ui/components/controls.go:53` — one character substitution:

```go
// before
case "track":
    repeat = c.activeStyle.Render("↻1")
// after
case "track":
    repeat = c.activeStyle.Render("↻¹")
```

No color, layout, width, or routing changes. The rendered string is 2 bytes
longer (U+00B9 is 2 UTF-8 bytes vs 1 for ASCII `1`), but lipgloss measures
display width in runes/cells — both `1` and `¹` occupy one terminal cell, so
the controls row width is unchanged.

## Acceptance Criteria

- [ ] `Controls.Render()` returns a string containing `↻¹` when `repeatMode` is `"track"`
- [ ] `Controls.Render()` returns a string containing `↻` (without superscript) when `repeatMode` is `"context"`
- [ ] `Controls.Render()` returns inactive-styled `↻` when `repeatMode` is `"off"`
- [ ] `make ci` passes

## Tasks

- [ ] Update test `TestControls_Render_RepeatTrack` (or equivalent) in
      `internal/ui/components/controls_test.go` to assert `"↻¹"` instead of `"↻1"`
  - test: `go test ./internal/ui/components/... -run TestControls` -v` → FAIL
- [ ] Change `"↻1"` → `"↻¹"` in `internal/ui/components/controls.go:53`
  - test: controls tests → PASS; `go build ./...` clean
- [ ] `make ci` passes
