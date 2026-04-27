---
name: project_spotnik_feature164_complete
description: Story 164 (ProgressBar primitive + seek/volume migration): import cycle fix, PartialGlyph export, dead zone removal, □→░ migration
type: project
---

## Story 164 — ProgressBar Primitive + Gradient Bar Migration

**What was built:**
- `internal/uikit/progress_bar.go`: `ProgressBar` struct + `Render() string` (partial-block algo §5.7)
- `internal/uikit/progress_bar_test.go`: 8 tests cover spec + threshold table + ASCII mode
- `uikit.PartialGlyph(remainder float64, m GlyphMode) string` exported, reused by gradient bars
- Migrated `GradientSeekBar` + `GradientVolumeBar` in `gradient.go` use `uikit.GlyphFor` + `uikit.PartialGlyph`
- Moved `TableChrome` from `internal/uikit/` to `internal/ui/components/` — break import cycle

**Import cycle fix:**
- `uikit/table_chrome.go` imported `components` → blocked `components` importing `uikit`
- `TableChrome` zero external callers (future-use stub)
- Fix: move to `internal/ui/components/table_chrome.go` in package `components`
- Post-move: `components` freely imports `uikit`

**Visual changes from migration:**
- Volume bar empty char: `□` → `░` (GlyphBarEmpty, §5.7 canonical)
- Volume bar dead zone removed: `fraction < 1/8` previously skipped partial; now emits `▏` per §5.7
- Seek bar: supports partial blocks at fill boundary (was integer-only)
- All `gradient_test.go` partial-block expectations updated

**Key patterns:**
- `ProgressBar` value receiver — clamping `p.Progress` doesn't mutate caller struct
- `lipgloss.Width(bar) == Width` verified all cases incl. partial blocks
- `uikit` coverage must be 100% — fixed pre-existing gap in `StatusGlyph` unknown-role fallback

**Gotchas:**
- Import cycle `uikit → components → uikit` recurs if any `uikit` file imports `components`. Check before adding imports.
- ASCII partial char `▏` → `.` collides with empty char `.` in ASCII mode. Not bug (§5.7 spec); ASCII tests must use progress values producing integer fill (no partial) to avoid ambiguous `strings.Count`.
- `TestProgressBar_ASCII_HalfFilled` uses `Progress=0.5` on `Width=20` (→ exactly 10.0 filled, no fractional remainder) — avoids partial/empty char collision in ASCII.
- `make ci` fmt-check uses `git ls-files`, includes deleted-but-not-staged files → `lstat` warnings, no CI failure.

**Testing notes:**
- `partialChars := []string{"▏", "▎", "▍", "▌", "▋", "▊", "▉"}` correct inline slice for negative assertions (replaces old `volumePartialChars[:7]` ref to removed var)
- uikit coverage 100%; overall 88.4%
- Branch: `feat/13-uikit-progress-bar`, PR #210