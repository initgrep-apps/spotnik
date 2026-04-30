---
name: project_spotnik_story192_complete
description: Story 192 (Glyph CI Guards, LANG=C matrix, catalogue-leak fixes): guard scripts, Makefile/CI wiring, production leak fixes, engine_test fixes
type: project
---

## Story 192 — Glyph-fallback CI Guards (Final Phase 11)

**What was built:**
- `scripts/check-banned-glyphs.sh` — fails CI on 13 banned glyphs in prod source; excludes `_test.go`
- `scripts/check-catalogue-leaks.sh` — fails CI on catalogue chars outside glyph.go in non-comment prod code; exempt: `layout/border.go`, `layout/truncate.go` (import-cycle blocked); perl filter strips comment-only hits
- `scripts/check-render-pane-border.sh` — fails CI on direct `RenderPaneBorder` callers outside `uikit/`; perl filter strips comment-only hits
- `make check-glyphs` target chained into `make ci`
- `.github/workflows/ci.yml`: `Glyph fallback guards` step + new `tests-locale` job with LANG matrix

**Key files:**
- `scripts/` — three guard scripts, all `chmod +x`
- `internal/ui/panes/search_delegate.go` — `truncateString` fix: accounts for multi-rune ellipsis in ASCII mode
- `internal/ui/panes/help_overlay.go` — `buildHelpContent()` function replaces static `helpContent` var
- Feature 13 `feature.md` and story 192 frontmatter set to `done`

**Production catalogue leaks found and fixed:**
- `gateway_live_pane.go`: raw `→` → `GlyphArrowRight`
- `search.go`: raw `…`, `→`, `←` → `GlyphEllipsis`, `GlyphArrowRight`, `GlyphArrowLeft`
- `search_delegate.go`: raw `…` → `GlyphEllipsis` via `truncateString`
- `themes.go`: raw `██` → `GlyphBarFull × 2`
- `help_overlay.go`: static arrow key labels → `buildHelpContent()` with `GlyphFor`
- `cmd/root.go`: raw `…` → `GlyphEllipsis`
- `theme.go`: doc comment `▶` removed

**engine_test.go fixes:**
- `TestBlockRenderer_OnlyBlockOrSpace` and `TestBlockRenderer_FullHeight_AllFilled`: pin `GlyphUnicode`, use `GlyphFor` for expected value
- `TestBraillePatterns_OnlyBrailleRunes` and `TestBlockPatterns_OnlyBlockOrSpace`: pin `GlyphUnicode` to prevent engine fallthrough to `AsciiBarsRenderer`

**Gotchas:**
- `check-catalogue-leaks.sh` spec script has no comment filter — the real codebase has catalogue chars in doc comments everywhere; need `perl -ne 'print unless m{//.*CHAR}'` filter
- `check-banned-glyphs.sh` spec script doesn't exclude `_test.go` — but tests use `assert.NotContains(t, ..., "ᐅ")` which contains the banned char; must exclude test files
- `truncateString` bug: switching `"…"` (1 rune) to `GlyphEllipsis` which can be `"..."` (3 runes in ASCII) breaks the rune-count contract. Fix: compute `ellipsisLen` and reserve that many slots: `keep = max - ellipsisLen`
- `layout/truncate.go` uses `const ellipsis = "…"` — cannot use `uikit.GlyphFor` because `layout` is imported by `uikit` (import cycle). Exempt this file from the catalogue-leak guard
- `helpContent` static var → `buildHelpContent()` function: needed because `GlyphFor` must be called at render time, not package init (mode may not be set yet at init)
- Tests that reference cancelled `helpContent` var need to call `buildHelpContent()` instead
- `TestTruncateString` had hardcoded `"…"` in `want` fields — must resolve via `uikit.GlyphFor` to be mode-agnostic

**Testing notes:**
- 88.8% coverage maintained
- `LANG=C make test` passes locally (verified)
- Both `LANG=en_US.UTF-8` and `LANG=C` pass full test suite

**PR #244 Review Cycle — Additional fixes:**

**CHARS parity fix:** The original CHARS array had only ~26 chars but glyphTable had 87 unique unicode forms. Added all missing chars (organized by category). Also found 6 new raw-literal leaks triggered by the expanded CHARS:
- `search.go`: raw `"─"` separator line → `GlyphHRule`
- `playlists_pane.go`, `albums_pane.go`: raw `"──"` title decorators → two `GlyphHRule`
- `render.go`: raw `"×"` dimension separator → `GlyphOverlayDismiss` (renders as × in unicode, x in ASCII)
- `search_delegate.go`: raw `"·"` subtitle separator → `subtitleSep()` helper using `GlyphSeparator`
- `polling_traffic_pane.go`: raw `"·"` status separator → `GlyphSeparator`
- `auth.go`: raw `"·"` hint separator → `GlyphSeparator`
- `cliout/message.go`: raw `"·"` caption separator → `GlyphSeparator`

**Golden file update:** `cmd/testdata/golden/auth_status_expiring.txt` had hardcoded `·` which now becomes `|` under the ASCII-pinned cmd test environment. Updated golden file to show `|`.

**engine_test.go Fix 2:** Dropped `SetModeForTest(GlyphUnicode)` pin from `TestBlockRenderer_OnlyBlockOrSpace` and `TestBlockRenderer_FullHeight_AllFilled` (they were no-ops). Added `TestEngine_ASCIIMode_BlockPatterns_OnlyASCIIChars` which pins ASCII and verifies engine routes to AsciiBarsRenderer.

**Guard script tests (Fix 3):** Added `scripts/check_guards_test.go` with 12 e2e tests covering clean/violation/comment-exempt/test-file-exempt for all three guard scripts. Tests use `t.TempDir()` + `makeTree()` to build minimal fixture trees. Script path resolved via `runtime.Caller(0)`.

**truncateString wantPrefix (Fix 4):** Added `wantPrefix` field and `wantPrefixOf()` helper to `TestTruncateString`. Now asserts `strings.HasPrefix(got, wantPrefix)` for truncated cases, preventing wrong-end truncation bugs from passing.

**CI Fix 5:** Changed `tests-locale` matrix to run `LANG=X make ci` instead of `make test`. Added golangci-lint install step to the matrix.

**Gotchas discovered:**
- `cmd/root_test.go` TestMain calls `cliout.SetTestMode(true)` which pins GlyphASCII. Any `cliout` function that previously used raw `·` now outputs `|` in tests. Golden files must reflect ASCII forms.
- `uikit.Use()` is only called from `cmd/root.go` (production code path). In unit tests (non-cmd), `activeMode = 0 = GlyphUnicode` always. So `Title()` tests in `panes/` correctly expect `──` even under LANG=C.
- When adding `uikit` import to a new file, gofmt will reorder imports alphabetically.

**Feature 13 complete.** All 29 stories (150–172, 183–192) done.
