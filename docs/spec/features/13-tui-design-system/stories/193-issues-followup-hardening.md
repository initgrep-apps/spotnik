---
title: "Feature 13 issues follow-up: hardening + content ASCII gaps + test depth"
feature: 13-tui-design-system
status: done
---

## Background

Across stories 183–192, ~25 sub-threshold review observations were filed in
`docs/spec/issues.md` rather than blocking individual merges. The
high-value items fall into three buckets:

- **Production hardening** — invariants that are comment-only today and
  would silently corrupt on misuse
- **Content ASCII fallback gaps** — inline glyphs that bypass `GlyphFor`
  and remain unicode under `ui.glyphs = "ascii"` even though the chrome
  around them is ASCII-correct
- **Test depth** — assertions that would not catch the specific
  regressions the original PR introduced

This story bundles those high-value items into one focused cleanup PR.
Pure code-style polish items (e.g. `max()` helpers, `else`-after-`return`,
constant hoisting) are explicitly out of scope and remain in `issues.md`
for future drive-by cleanup.

## Tasks

### Production hardening

1. **`internal/uikit/spinner_frames.go` `SpinnerFrames` defensive copy or
   typed accessor.** The function returns the package-level
   `spinnerFramesUnicode` / `spinnerFramesASCII` slice directly. Doc says
   "must NOT be mutated by the caller" but the contract is comment-only.
   Both `uikit.Spinner` and `cliout.Spinner` share these slices, so an
   in-place mutation in one consumer corrupts the other.
   Either return `append([]string(nil), spinnerFramesX...)` per call or
   expose `SpinnerFrame(m, i)` / `SpinnerFrameCount(m)` accessors that
   never hand out the slice. (Source: issues.md story 183 item 1)

2. **`internal/uikit/glyph.go GlyphFor` and
   `internal/uikit/spinner_frames.go SpinnerFrames` — convert the
   two-state `if mode == GlyphASCII { … } return unicode` cascade to a
   `switch mode { case GlyphASCII: …; case GlyphUnicode: …; default: … }`.**
   The current cascade silently swallows any future third `GlyphMode`
   (e.g. `GlyphNerdFont`); a `default` clause makes it a compile-time
   missing-case signal. (Source: issues.md story 183 item 2)

3. **`internal/uikit/toast.go` `truncateRunes` runtime guard.** The
   invariant "`max` must be `>= len(ellipsis runes)`" is comment-only.
   Currently safe because `Normalize` is the only caller and uses
   48 / 160. A future caller passing `max < 3` in ASCII mode would slice
   `runes[max-3:max]` and panic. Add a runtime guard at the function
   head: if `max < ellipsisLen`, return the ellipsis truncated to `max`,
   or return the original string unmodified — pick whichever matches
   `Normalize`'s expected fallback behaviour. (Source: issues.md story
   183 item 3)

4. **`internal/cliout/tty.go` `SetTestMode(false)` restores the prior
   `uikit` mode.** Currently only `SetTestMode(true)` pins it to
   `GlyphASCII`; `false` doesn't reverse the side effect. Snapshot the
   prior mode at `SetTestMode(true)` entry (in a package-level var
   guarded by the existing mutex) and restore it on `SetTestMode(false)`.
   Update the doc comment on `SetTestMode` accordingly. (Source:
   issues.md story 184 item 2)

5. **`internal/ui/components/viz/engine.go selectRenderer` mode flip
   honoured at runtime.** Currently called inside `generateFrames`,
   which only runs on `SetSize` / `CyclePattern` / `SetPattern`. If a
   future change toggles `uikit.ActiveMode()` mid-session without resize
   or cycle, the cached frames stay stale. Either (a) hook
   `selectRenderer` into a frame-time read so each frame respects the
   current mode, or (b) leave the regen-time read but add a test that
   flips mode after `SetSize`, calls `CyclePattern`, and verifies the
   new renderer wins. Pick (a) if cheap; document (b) otherwise.
   (Source: issues.md story 189 item 1)

### Content ASCII fallback gaps

6. **`internal/ui/panes/search.go:803` tab-bar `─` separator.** Currently
   `strings.Repeat("─", innerWidth)`. Under `ui.glyphs = "ascii"` this
   still emits `─`. Route through
   `strings.Repeat(uikit.GlyphFor(uikit.GlyphHRule, uikit.ActiveMode()), innerWidth)`.
   Add an ASCII assertion to the search-overlay-results test that the
   `─` is absent under ASCII mode. (Source: issues.md story 186 item 1)

7. **`internal/ui/panes/help_overlay.go:140` `│` content divider.**
   Currently rendered via `lipgloss.NewStyle()...Render("│")`. Route
   through `uikit.GlyphFor(uikit.GlyphVRule, uikit.ActiveMode())`. The
   `TestHelpOverlay_AsciiBorder` carve-out comment for `│` becomes
   obsolete — extend the test to also assert `│` absent in ASCII mode.
   (Source: issues.md story 186 item 3)

8. **`scripts/check-catalogue-leaks.sh` CHARS includes spinner braille
   frames.** Add `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏` to the CHARS array. By design these
   dispatch via `SpinnerFrames(mode)` not `GlyphFor`, but a leak
   elsewhere should still be caught by the guard. Update
   `scripts/check_guards_test.go` to verify the new entries are
   detected. (Source: issues.md story 192 item 2)

### Test depth (high-value subset)

9. **`internal/uikit/playback_controls_test.go` direct paused-branch
   coverage.** Add a uikit-level test for `Playing=false` asserting
   `▷` (unicode) and `|>` (ASCII) glyphs appear in the rendered output.
   Currently covered only transitively via `controls_test.go`. A
   regression that swapped `GlyphPaused`/`GlyphPausedPB` in the
   primitive itself would not be caught. (Source: issues.md story 187
   item 1)

10. **`internal/uikit/toast_test.go` body-truncation explicit assertion.**
    `truncateRunes` is shared between Title and Body via `Normalize`.
    Existing test `TestToast_TruncatedTitle_AsciiEllipsis` covers Title.
    Add one assertion that a 161-rune Body in ASCII mode ends with
    `...` to make the test surface match the public contract. (Source:
    issues.md story 183 item 4)

11. **`internal/cliout/spinner_test.go` symmetric unicode test.**
    `TestSpinnerFrames_AsciiSet` covers ASCII only. Add a sibling test
    that pins `uikit.SetModeForTest(GlyphUnicode)` and asserts
    `resolveSpinnerFrames()` returns `uikit.SpinnerFrames(GlyphUnicode)`.
    Catches the constant-mode bug `func resolveSpinnerFrames() []string { return uikit.SpinnerFrames(GlyphASCII) }`.
    (Source: issues.md story 184 item 1)

## Out of scope (logged-only)

- Story 183 items 5, 6, 7 (FPS doc comment, no-op defer, GlyphSeparator naming) — pure cleanup
- Story 185 item 1 (`TestRenderGrid_AsciiBorders` brittleness) — only triggers on hypothetical state change
- Story 186 item 2 (bubble-table `│` column separators) — third-party limitation, needs upstream patch or TableChrome glyph injection (separate effort)
- Story 187 items 2, 3 (ASCII-fallback absence assertions, intentional visual diffs) — confidence already high via catalogue tests
- Story 189 item 2 (comment typo) — drive-by fix
- Story 190 items 1, 2, 3 (innerWidth dedup, spinner test isolation, dispatcher comment) — pure polish
- Story 191 items 2, 3 (`*` weak positive, `max()` / hoist / else-after-return) — pure polish
- Story 192 items 1, 3, 4 (dead `-` branch, partial cliout ASCII pin, locale matrix cost) — pure polish or design choice

These items remain in `issues.md` for opportunistic cleanup; they do not warrant their own PR cycle.

## Acceptance criteria

- [ ] `internal/uikit/spinner_frames.go` `SpinnerFrames` no longer hands out the package-level slice (defensive copy or typed accessor)
- [ ] `GlyphFor` and `SpinnerFrames` use `switch` with `default` clause
- [ ] `truncateRunes` has a runtime guard for `max < ellipsisRuneLen`
- [ ] `cliout.SetTestMode(false)` restores the prior `uikit` mode
- [ ] `viz.selectRenderer` either reads mode at frame time OR has a regen-time mode-flip regression test
- [ ] `panes/search.go:803` `─` separator routes through `GlyphFor(GlyphHRule, ActiveMode())`
- [ ] `panes/help_overlay.go:140` `│` divider routes through `GlyphFor(GlyphVRule, ActiveMode())`
- [ ] `scripts/check-catalogue-leaks.sh` CHARS includes the 10 spinner braille frames
- [ ] `playback_controls_test.go` has a direct paused-branch test asserting `▷` (unicode) and `|>` (ASCII)
- [ ] `toast_test.go` has an explicit body-truncation ASCII assertion
- [ ] `cliout/spinner_test.go` has a symmetric unicode-mode `resolveSpinnerFrames` test
- [ ] `make ci` passes
- [ ] `LANG=C make ci` passes
- [ ] After this story merges, the corresponding entries in `docs/spec/issues.md` for stories 183–192 are removed (they're now resolved)

## Notes

This is a drive-by hardening story. Each task is small and largely
independent — the implementer should be able to land them as a single PR
with one commit per task, similar to the cadence of stories 183–192.

The `issues.md` cleanup at the end is part of the story's deliverable so
future readers see only unresolved concerns.
