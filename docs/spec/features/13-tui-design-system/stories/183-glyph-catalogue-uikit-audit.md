---
title: "Glyph catalogue extensions + uikit self-audit fixes + SpinnerFrames export"
feature: 13-tui-design-system
status: done
---

## Background

The post-merge audit (`docs/superpowers/specs/2026-04-29-glyph-fallback-audit-design.md`)
confirmed that `internal/uikit` is ~97 % compliant — every dynamic primitive routes
through `GlyphFor` — but several closing gaps remain that block the rest of the
glyph-fallback rollout:

1. **Missing catalogue rows.** Six new domain roles surfaced by the audit
   (`GlyphSeparator`, `GlyphPlaylist`, four `GlyphDevice*`), plus the §4.9 keyboard-chord
   roles (`GlyphEnter`, `GlyphEscape`, `GlyphTab`, `GlyphBackspace`, `GlyphSpace`) and
   the §4.10 superscript roles (`GlyphSuperscript0..9`, `GlyphSuperscriptPlus`,
   `GlyphSuperscriptMinus`) exist on paper but not in `glyph.go` / `glyphTable`. Phases
   4–5 of the rollout reference them, so they must land first.
2. **Five hardcoded glyphs inside uikit.** `list_row.go:48` (`"…"`), `toast.go:112`
   (raw `'…'` rune), `header_bar.go:52` (`" ─ "`), `key_bar.go:31–34` (`" · "` / `" | "`
   branch), and `status_bar.go` (calls `layout.RenderPaneBorder` without populating
   `BorderConfig` glyph fields) bypass the catalogue. Each is a test gap as well.
3. **Spinner frames duplicated.** `cliout.spinner.go` carries its own braille frame
   array and has **no** ASCII fallback at all. The fix needs `uikit` to expose a single
   `SpinnerFrames(mode) []string` helper that both packages share — ships in this story
   so Phase 7 (story 184) can consume it without depending on the full Bubble Tea
   primitive.
4. **Missing ASCII snapshot tests.** `empty_state_test.go`, `url_box_test.go`,
   `header_bar_test.go`, `form_field_test.go` lack ASCII-mode assertions. The first
   two render correctly today (no glyphs); the latter two depend on the fixes in
   point 2.

This story is the foundation gate for stories 184–192. It only touches `internal/uikit/`
and `docs/TUI-DESIGN-SYSTEM.md` — no callers change yet.

**CLAUDE.md rule 17.** "Add a new primitive, glyph, or role to `internal/uikit` without
updating `docs/TUI-DESIGN-SYSTEM.md` in the same commit." Each catalogue addition below
ships its doc row in the same commit, not a follow-up.

**Plan tasks:** 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7, 1.8, 1.9 in
`docs/superpowers/plans/2026-04-29-glyph-fallback.md`.

**Files:** `internal/uikit/glyph.go`, `internal/uikit/glyph_test.go`,
`internal/uikit/spinner.go`, `internal/uikit/list_row.go`, `internal/uikit/list_row_test.go`,
`internal/uikit/toast.go`, `internal/uikit/toast_test.go`, `internal/uikit/header_bar.go`,
`internal/uikit/header_bar_test.go`, `internal/uikit/key_bar.go`,
`internal/uikit/key_bar_test.go`, `internal/uikit/status_bar.go`,
`internal/uikit/status_bar_test.go`, `internal/uikit/empty_state_test.go`,
`internal/uikit/url_box_test.go`, `internal/uikit/form_field_test.go`,
`docs/TUI-DESIGN-SYSTEM.md`. **Created:** `internal/uikit/spinner_frames.go`,
`internal/uikit/spinner_frames_test.go`.

## Design

### Catalogue additions

Append to the `const` block in `internal/uikit/glyph.go`:

```go
// Domain
GlyphPlaylist GlyphRole = "music.playlist"

// Generic separators
GlyphSeparator GlyphRole = "sep.bullet"

// Device-type icons (devices pane)
GlyphDeviceComputer GlyphRole = "device.computer"
GlyphDevicePhone    GlyphRole = "device.phone"
GlyphDeviceSpeaker  GlyphRole = "device.speaker"
GlyphDeviceTV       GlyphRole = "device.tv"

// Keyboard chords
GlyphEnter     GlyphRole = "kbd.enter"
GlyphEscape    GlyphRole = "kbd.escape"
GlyphTab       GlyphRole = "kbd.tab"
GlyphBackspace GlyphRole = "kbd.backspace"
GlyphSpace     GlyphRole = "kbd.space"

// Superscripts (used for pane toggle keys)
GlyphSuperscript0     GlyphRole = "sup.0"
GlyphSuperscript1     GlyphRole = "sup.1"
GlyphSuperscript2     GlyphRole = "sup.2"
GlyphSuperscript3     GlyphRole = "sup.3"
GlyphSuperscript4     GlyphRole = "sup.4"
GlyphSuperscript5     GlyphRole = "sup.5"
GlyphSuperscript6     GlyphRole = "sup.6"
GlyphSuperscript7     GlyphRole = "sup.7"
GlyphSuperscript8     GlyphRole = "sup.8"
GlyphSuperscript9     GlyphRole = "sup.9"
GlyphSuperscriptPlus  GlyphRole = "sup.plus"
GlyphSuperscriptMinus GlyphRole = "sup.minus"
```

Append to `glyphTable`:

```go
GlyphPlaylist:         {"▤", "[=]"},
GlyphSeparator:        {"·", "|"},
GlyphDeviceComputer:   {"⊡", "[c]"},
GlyphDevicePhone:      {"⊞", "[p]"},
GlyphDeviceSpeaker:    {"⊟", "[s]"},
GlyphDeviceTV:         {"⊠", "[tv]"},
GlyphEnter:            {"⏎", "Enter"},
GlyphEscape:           {"⎋", "Esc"},
GlyphTab:              {"⇥", "Tab"},
GlyphBackspace:        {"⌫", "BS"},
GlyphSpace:            {"␣", "Space"},
GlyphSuperscript0:     {"⁰", "0"},
GlyphSuperscript1:     {"¹", "1"},
GlyphSuperscript2:     {"²", "2"},
GlyphSuperscript3:     {"³", "3"},
GlyphSuperscript4:     {"⁴", "4"},
GlyphSuperscript5:     {"⁵", "5"},
GlyphSuperscript6:     {"⁶", "6"},
GlyphSuperscript7:     {"⁷", "7"},
GlyphSuperscript8:     {"⁸", "8"},
GlyphSuperscript9:     {"⁹", "9"},
GlyphSuperscriptPlus:  {"⁺", "+"},
GlyphSuperscriptMinus: {"⁻", "-"},
```

Each commit that adds catalogue rows also appends the matching rows to
`docs/TUI-DESIGN-SYSTEM.md` (§4 sub-tables: domain in §4.6, separator in new §4.6a,
device icons in new §4.6b, chords in §4.9, superscripts in §4.10).

### `SpinnerFrames(mode)` extraction

Create `internal/uikit/spinner_frames.go`:

```go
package uikit

var spinnerFramesUnicode = []string{
    "⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
}

var spinnerFramesASCII = []string{"|", "/", "-", "\\"}

// SpinnerFrames returns the animation frames for the given mode.
// The returned slice must NOT be mutated by the caller — it is a stable
// reference shared between uikit.Spinner and cliout.Spinner.
func SpinnerFrames(m GlyphMode) []string {
    if m == GlyphASCII {
        return spinnerFramesASCII
    }
    return spinnerFramesUnicode
}
```

`internal/uikit/spinner.go` source the frames at construction via
`SpinnerFrames(ActiveMode())` instead of an inline array.

### Self-audit fixes

| File | Before | After |
|---|---|---|
| `list_row.go:48` `PadOrTruncate` | hardcoded `"…"` | `GlyphFor(GlyphEllipsis, ActiveMode())` |
| `toast.go:112` truncation | raw `runes[max-1] = '…'` | replace last 1–3 runes with `[]rune(GlyphFor(GlyphEllipsis, ActiveMode()))` |
| `header_bar.go:52` separator | `muted.Render(" ─ ")` | `muted.Render(" " + GlyphFor(GlyphHRule, ActiveMode()) + " ")` |
| `key_bar.go:31–34` separator | branch `" · "` / `" \| "` | `sep := " " + GlyphFor(GlyphSeparator, ActiveMode()) + " "` |
| `status_bar.go` `BorderConfig` | calls `layout.RenderPaneBorder` without glyph fields | populate `CornerTL/TR/BL/BR/HRule/VRule` via `GlyphFor` like `pane_chrome.go` |

### ASCII snapshot tests

| File | Test | What it asserts |
|---|---|---|
| `empty_state_test.go` | `TestEmptyState_AsciiMode` | line count matches `Height`; text + hint present (no glyphs to swap) |
| `url_box_test.go` | `TestURLBox_AsciiMode` | no `╭╮╰╯` in output; URL content present |
| `header_bar_test.go` | `TestHeaderBar_AsciiSeparator` | output contains `" - "`; no `" ─ "` |
| `form_field_test.go` | `TestFormField_AsciiValidationError` | validation failure prefix is `x`, not `✗` |

## Acceptance Criteria

- [ ] `uikit.GlyphFor(GlyphPlaylist, GlyphASCII) == "[=]"`;
      `uikit.GlyphFor(GlyphPlaylist, GlyphUnicode) == "▤"`. Same shape for
      `GlyphSeparator`, `GlyphDeviceComputer/Phone/Speaker/TV`, `GlyphEnter`,
      `GlyphEscape`, `GlyphTab`, `GlyphBackspace`, `GlyphSpace`,
      `GlyphSuperscript0..9`, `GlyphSuperscriptPlus`, `GlyphSuperscriptMinus`
- [ ] `docs/TUI-DESIGN-SYSTEM.md` §4.6 lists `playlist badge ▤ [=]`; new §4.6a
      lists `separator (bullet) · |`; new §4.6b lists computer/phone/speaker/tv
      device-type icons; §4.9 keyboard-chord rows are present (not "future");
      §4.10 superscript rows are present (not "reserved")
- [ ] `internal/uikit/spinner_frames.go` exports `SpinnerFrames(mode GlyphMode) []string`;
      unicode mode returns the 10-frame braille set; ASCII mode returns `["|", "/", "-", "\\"]`
- [ ] `internal/uikit/spinner.go` sources frames from `SpinnerFrames(ActiveMode())` and
      no longer carries an inline frame array; existing spinner tests pass unchanged
- [ ] `internal/uikit/list_row.go:PadOrTruncate` uses `GlyphFor(GlyphEllipsis,
      ActiveMode())`; ASCII mode produces `...` (3 chars), unicode produces `…`
- [ ] `internal/uikit/toast.go` truncation uses `GlyphFor(GlyphEllipsis, ActiveMode())`;
      a long title in ASCII mode contains `...`, never `…`
- [ ] `internal/uikit/header_bar.go:52` separator uses `GlyphFor(GlyphHRule,
      ActiveMode())`; ASCII output contains `" - "`, not `" ─ "`
- [ ] `internal/uikit/key_bar.go:31–34` separator uses `GlyphFor(GlyphSeparator,
      ActiveMode())`; the unicode/ascii branch is gone
- [ ] `internal/uikit/status_bar.go` populates `CornerTL/TR/BL/BR/HRule/VRule` via
      `GlyphFor`; ASCII rendering contains `+`/`-`/`|`, not `╭╮╰╯─│`
- [ ] `empty_state_test.go`, `url_box_test.go`, `header_bar_test.go`,
      `form_field_test.go` each ship a `_AsciiMode` test that uses
      `SetModeForTest(GlyphASCII)` and asserts no banned unicode glyphs in the rendered
      output
- [ ] `internal/uikit` package coverage ≥ existing baseline (no regression)
- [ ] No catalogue characters from new rows appear outside `glyph.go` / canonical doc
      files
- [ ] `make ci` → PASS

## Tasks

Step-by-step: Tasks 1.1–1.9 in `docs/superpowers/plans/2026-04-29-glyph-fallback.md`.

- [ ] Branch: `feat/13-glyph-catalogue-uikit-audit`
- [ ] Write failing `TestGlyphFor_NewDomainRoles` covering `GlyphSeparator`,
      `GlyphPlaylist`, `GlyphDeviceComputer/Phone/Speaker/TV` → FAIL
- [ ] Add the six domain role constants + `glyphTable` rows; append the matching rows
      to `docs/TUI-DESIGN-SYSTEM.md` §4.6 / §4.6a / §4.6b → tests PASS
- [ ] Commit: `feat(uikit): add separator, playlist, and device-type GlyphRoles`
- [ ] Write failing `TestGlyphFor_KeyboardChords` and `TestGlyphFor_Superscripts` → FAIL
- [ ] Add the chord + 12 superscript role constants and table rows; update §4.9 / §4.10
      of `docs/TUI-DESIGN-SYSTEM.md` to remove "future"/"reserved" qualifiers →
      tests PASS
- [ ] Commit: `feat(uikit): implement §4.9 keyboard-chord and §4.10 superscript GlyphRoles`
- [ ] Write failing `TestSpinnerFrames_Unicode` and `TestSpinnerFrames_ASCII` → FAIL
- [ ] Create `spinner_frames.go` exporting `SpinnerFrames(mode)` with both arrays;
      refactor `spinner.go` to call `SpinnerFrames(ActiveMode())` at construction → PASS
- [ ] Commit: `feat(uikit): export SpinnerFrames(mode) as the shared frame source`
- [ ] Write failing `TestListRow_PadOrTruncate_AsciiEllipsis` and
      `TestToast_TruncatedTitle_AsciiEllipsis` → FAIL
- [ ] Replace `list_row.go:48` and `toast.go:112` ellipsis with
      `GlyphFor(GlyphEllipsis, ActiveMode())` → tests PASS
- [ ] Commit: `fix(uikit): route ellipsis through GlyphFor in ListRow and Toast`
- [ ] Write failing `TestHeaderBar_AsciiSeparator` → FAIL
- [ ] Replace `header_bar.go:52` separator with
      `GlyphFor(GlyphHRule, ActiveMode())` → PASS
- [ ] Commit: `fix(uikit): route HeaderBar separator through GlyphFor(GlyphHRule)`
- [ ] Write failing `TestKeyBar_AsciiSeparator` → FAIL
- [ ] Replace the `key_bar.go:31–34` branch with
      `GlyphFor(GlyphSeparator, ActiveMode())` → PASS
- [ ] Commit: `fix(uikit): route KeyBar separator through GlyphFor(GlyphSeparator)`
- [ ] Write failing `TestStatusBar_AsciiBorder` → FAIL
- [ ] Populate `BorderConfig.CornerTL/TR/BL/BR/HRule/VRule` in `status_bar.go` via
      `GlyphFor` → PASS
- [ ] Commit: `fix(uikit): populate StatusBar BorderConfig glyph fields via GlyphFor`
- [ ] Add `TestEmptyState_AsciiMode`, `TestURLBox_AsciiMode`,
      `TestHeaderBar_AsciiSeparator` (already covered above), `TestFormField_AsciiValidationError`
- [ ] Commit: `test(uikit): add ASCII snapshot tests for EmptyState, URLBox, FormField`
- [ ] `make ci` → PASS
- [ ] Push branch + open PR
