---
title: "NowPlaying Controls Polish"
status: done
---

## Description

Two small but visible quality improvements to the NowPlaying pane controls:

1. **Repeat-track icon** — the current `↻1` label uses a plain ASCII `1` that reads
   as a word, not a modifier. Replacing it with the Unicode superscript one (`↻¹`,
   U+00B9) makes the annotation feel integrated into the glyph.

2. **Volume granularity** — volume adjusts in 5% steps and the bar uses integer
   truncation so most 5% presses don't visually move the bar at narrow widths. This
   story drops the step to 1% and adds partial-block rendering (▏▎▍▌▋▊▉█) so the
   bar moves smoothly on every keypress regardless of bar width.

Both changes are fully contained within existing files — no new components, no new
messages, no API contract changes.

## Goals

- Repeat-track mode looks polished and unambiguous in the controls row.
- Volume adjustment feels responsive: every `+`/`-` press visually moves the bar.

## Acceptance Criteria

- [ ] `NowPlayingPane` controls row shows `↻¹` (not `↻1`) when repeat mode is `"track"`
- [ ] Pressing `+` or `-` changes volume by 1% (not 5%)
- [ ] Volume bar advances by one partial-block step on every press at any bar width
- [ ] Volume bar renders exactly 0 partial block chars when volume is an exact
      multiple of the bar-fill ratio (no phantom partial block at 0% or 100%)
- [ ] Existing tests updated; new boundary tests added for the volume bar
- [ ] `make ci` passes (lint + tests + ≥ 80% coverage)

## Stories

| # | Title | Status |
|---|-------|--------|
| 122 | Repeat-track superscript icon | open |
| 123 | Volume 1% steps and partial-block bar | open |

## Files Touched (Summary)

| File | Change |
|---|---|
| `internal/ui/components/controls.go` | `"↻1"` → `"↻¹"` |
| `internal/ui/components/controls_test.go` | update assertion for repeat-track output |
| `internal/app/app.go` | `volumeStep: 5` → `volumeStep: 1` |
| `internal/ui/panes/messages.go` | update comments: `"5%"` → `"1%"` |
| `internal/ui/components/gradient.go` | partial-block fill algorithm in `GradientVolumeBar.Render()` |
| `internal/ui/components/gradient_test.go` | update/add boundary tests for partial-block rendering |
