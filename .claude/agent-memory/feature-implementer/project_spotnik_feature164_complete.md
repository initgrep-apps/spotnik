---
name: project_spotnik_feature164_complete
description: Story 164 (ProgressBar primitive + seek/volume migration): import cycle fix, PartialGlyph export, dead zone removal, □→░ migration
type: project
---

## Story 164 — ProgressBar Primitive + Gradient Bar Migration

**What was built:**
- `internal/uikit/progress_bar.go`: `ProgressBar` struct with `Render() string` (partial-block algorithm §5.7)
- `internal/uikit/progress_bar_test.go`: 8 tests covering spec requirements + threshold table + ASCII mode
- `uikit.PartialGlyph(remainder float64, m GlyphMode) string` exported for reuse by gradient bars
- Migrated `GradientSeekBar` and `GradientVolumeBar` in `gradient.go` to use `uikit.GlyphFor` + `uikit.PartialGlyph`
- Moved `TableChrome` from `internal/uikit/` to `internal/ui/components/` to break import cycle

**Import cycle fix:**
- `uikit/table_chrome.go` imported `components` → prevented `components` from importing `uikit`
- `TableChrome` had zero callers outside its own files (stub for future use)
- Solution: rename/move to `internal/ui/components/table_chrome.go` in package `components`
- After move: `components` can freely import `uikit`

**Visual changes from migration:**
- Volume bar empty char: `□` → `░` (GlyphBarEmpty, §5.7 canonical)
- Volume bar dead zone removed: `fraction < 1/8` previously skipped partial; now emits `▏` per §5.7
- Seek bar: now supports partial blocks at fill boundary (previously integer-only fill)
- All `gradient_test.go` partial-block expectations updated accordingly

**Key patterns:**
- `ProgressBar` uses value receiver — clamping `p.Progress` doesn't mutate caller's struct
- `lipgloss.Width(bar) == Width` verified for all cases including partial blocks
- `uikit` coverage must be 100% — pre-existing gap in `StatusGlyph` unknown-role fallback was fixed here

**Gotchas:**
- Import cycle `uikit → components → uikit` will recur if any future file in `uikit` imports `components`. Always check before adding imports.
- ASCII partial char `▏` → `.` collides with empty char `.` in ASCII mode. Not a bug (§5.7 spec), but tests for ASCII mode must use progress values that produce integer fill (no partial) to avoid ambiguous `strings.Count`.
- `TestProgressBar_ASCII_HalfFilled` uses `Progress=0.5` on `Width=20` (→ exactly 10.0 filled, no fractional remainder) to avoid partial-char/empty-char collision in ASCII.
- `make ci` fmt-check uses `git ls-files` which includes deleted-but-not-staged files → shows `lstat` warnings but doesn't fail CI.

**Testing notes:**
- `partialChars := []string{"▏", "▎", "▍", "▌", "▋", "▊", "▉"}` is the right inline slice for negative assertions (replaces old `volumePartialChars[:7]` ref to removed var)
- uikit coverage 100% achieved; overall 88.4%
- Branch: `feat/13-uikit-progress-bar`, PR #210
