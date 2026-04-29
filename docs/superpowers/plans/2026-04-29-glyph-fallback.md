# Glyph Fallback Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Spotnik render correctly under `ui.glyphs = "ascii"` end-to-end (TUI + CLI) without breaking layout, using one shared glyph catalogue.

**Architecture:** `internal/uikit` is the single source of truth for glyphs (`GlyphFor`), spinner frames (`SpinnerFrames`), and active mode (`ActiveMode()`). Every TUI surface routes through a `uikit` primitive (`PaneChrome`, `OverlayChrome`, `Panel`, `PlaybackControls`, `EmptyState`, `StatusGlyph`, etc.). `internal/cliout` becomes a one-way consumer of `uikit` for the same catalogue. The audio visualizer gains a third renderer (`AsciiBarsRenderer`) selected by `ActiveMode()`.

**Tech Stack:** Go 1.22+, Bubble Tea v2, Lip Gloss, `bubbles`, `bubble-table`, `bubbletea-overlay`, `bubbleup` (toasts), `golangci-lint`, `testify`. No new dependencies.

**Spec:** `docs/superpowers/specs/2026-04-29-glyph-fallback-audit-design.md`

---

## File Structure

### Files created

| Path | Purpose |
|---|---|
| `internal/uikit/spinner_frames.go` | Exports `SpinnerFrames(mode) []string` — single source of braille / ASCII frame arrays |
| `internal/uikit/spinner_frames_test.go` | Tests for `SpinnerFrames` in both modes |
| `internal/uikit/playback_controls.go` | New primitive: `PlaybackControls` — owns transport-glyph rendering with mode-aware fallback |
| `internal/uikit/playback_controls_test.go` | Tests for `PlaybackControls` (state matrix × both modes) |
| `internal/ui/components/viz/ascii_bars.go` | New visualizer renderer for ASCII mode |
| `internal/ui/components/viz/ascii_bars_test.go` | Tests for `AsciiBarsRenderer` |
| `scripts/check-banned-glyphs.sh` | CI guard: greps for banned glyphs, fails on hit |
| `scripts/check-catalogue-leaks.sh` | CI guard: catalogue characters allowed only in `glyph.go` + canonical docs |

### Files modified

| Path | What changes |
|---|---|
| `internal/uikit/glyph.go` | Add new `GlyphRole` constants and `glyphTable` rows (separator, playlist, 4 device, 5 chord, 12 superscript) |
| `internal/uikit/list_row.go` | `:48` ellipsis routed via `GlyphFor(GlyphEllipsis, mode)` |
| `internal/uikit/toast.go` | `:112` ellipsis routed via `GlyphFor(GlyphEllipsis, mode)`; bubbleup-registration moved here |
| `internal/uikit/header_bar.go` | `:52` separator routed via `GlyphFor(GlyphHRule, mode)` |
| `internal/uikit/key_bar.go` | `:31–34` separator routed via `GlyphFor(GlyphSeparator, mode)` |
| `internal/uikit/status_bar.go` | Populate `BorderConfig` glyph fields via `GlyphFor` |
| `internal/uikit/empty_state_test.go`, `url_box_test.go`, `header_bar_test.go`, `form_field_test.go` | Add ASCII snapshot tests |
| `internal/cliout/message.go` | `statusGlyph()` rewritten to call `uikit.GlyphFor`; `Hint.render()` arrow routed via `uikit.GlyphFor` |
| `internal/cliout/spinner.go` | Frames sourced from `uikit.SpinnerFrames(uikit.ActiveMode())` |
| `internal/cliout/tty.go` | Add doc comment clarifying `pinASCII` is colour-only; update `SetTestMode` to call `uikit.SetModeForTest` |
| `internal/cliout/*_test.go` | Parameterise glyph assertions; cover both modes |
| `cmd/root.go` | Single `uikit.Use(cfg.UI.Glyphs)` call already covers cliout — verify wiring is correct |
| `internal/app/render.go` | `renderGrid` migrated to `uikit.PaneChrome.Render`; inline `♪` / `•` / `…` swaps |
| `internal/ui/panes/themes.go` | Direct `RenderPaneBorder` → `uikit.PaneChrome.Render` |
| `internal/ui/panes/profile.go` | Direct `RenderPaneBorder` → `uikit.PaneChrome.Render`; `…` truncation routed via `GlyphFor` |
| `internal/ui/panes/devices.go` | Direct `RenderPaneBorder` → `uikit.OverlayChrome.Render`; status glyphs → `uikit.StatusGlyph`; device-type icons → new `GlyphDevice*` roles; empty message → `uikit.EmptyState` |
| `internal/ui/panes/help_overlay.go` | Direct `RenderPaneBorder` → `uikit.OverlayChrome.Render`; `│` divider → `GlyphFor(GlyphVRule, mode)` |
| `internal/ui/panes/search.go` | 3 direct `RenderPaneBorder` calls → `uikit.OverlayChrome.Render`; `bubbles/spinner.Model` → `uikit.Spinner` |
| `internal/ui/panes/search_delegate.go` | `categorySymbol` glyphs + separators routed through `GlyphFor` |
| `internal/ui/panes/nowplaying.go` | `Title()` `▶`/`⏸`/`─` via `GlyphFor`; "Nothing playing" → `uikit.EmptyState` |
| `internal/ui/panes/recentlyplayed_pane.go` | Empty message → `uikit.EmptyState` |
| `internal/ui/panes/networklog_pane.go` | Priority glyphs `◷`/`⚡` via `GlyphFor` |
| `internal/ui/panes/gateway_health_pane.go` | Custom dot-bar → `uikit.ProgressBar` |
| `internal/ui/components/infobox.go` | Hand-rolled border → `uikit.PaneChrome.Render` |
| `internal/ui/components/controls.go` | Replaced by `uikit.PlaybackControls` wrapper |
| `internal/ui/components/notifications.go` | Bubbleup-registration moved into `uikit.Toast`; this file becomes a thin wrapper or is folded entirely |
| `internal/ui/components/gradient.go` | `♪` icon via `GlyphFor(GlyphMusicNote, mode)` |
| `internal/ui/components/table.go` | `playingSymbol` const lazy-resolved at render time |
| `internal/ui/components/viz/engine.go` | Renderer selection branches on `uikit.ActiveMode()` |
| `internal/ui/components/viz/block.go` | `█` → `GlyphFor(GlyphBarFull, mode)` |
| `docs/TUI-DESIGN-SYSTEM.md` | §3 PlaybackControls primitive added; §4 catalogue rows for new roles |
| `docs/CLI-OUTPUT.md` | Note that cliout now imports uikit for catalogue |
| `Makefile` | Add `make check-glyphs` target invoking the two new scripts |
| `.github/workflows/ci.yml` | CI runs `make check-glyphs`; matrix runs tests under `LANG=en_US.UTF-8` and `LANG=C` |

### Reference files (read but not modified)

- `internal/uikit/pane_chrome.go` — canonical reference for the chrome migration pattern
- `internal/uikit/overlay_chrome.go` — canonical reference for overlays
- `internal/ui/layout/border.go` — `BorderConfig` struct (caller contract)
- `docs/TUI-DESIGN-SYSTEM.md` — catalogue and role matrix

---

## Phase 1 — Catalogue & shared infrastructure

### Task 1.1: Add new domain GlyphRoles to the catalogue

**Files:**
- Modify: `internal/uikit/glyph.go`

Adds the six domain roles surfaced by the audit (separator, playlist, 4 device-type icons).

- [ ] **Step 1: Write the failing test**

Append to `internal/uikit/glyph_test.go` (creating the file if it doesn't exist follows the pattern of other primitive tests in the package):

```go
func TestGlyphFor_NewDomainRoles(t *testing.T) {
    cases := []struct {
        role            GlyphRole
        unicode, ascii  string
    }{
        {GlyphSeparator, "·", "|"},
        {GlyphPlaylist, "▤", "[=]"},
        {GlyphDeviceComputer, "⊡", "[c]"},
        {GlyphDevicePhone, "⊞", "[p]"},
        {GlyphDeviceSpeaker, "⊟", "[s]"},
        {GlyphDeviceTV, "⊠", "[tv]"},
    }
    for _, c := range cases {
        if got := GlyphFor(c.role, GlyphUnicode); got != c.unicode {
            t.Errorf("GlyphFor(%s, unicode) = %q, want %q", c.role, got, c.unicode)
        }
        if got := GlyphFor(c.role, GlyphASCII); got != c.ascii {
            t.Errorf("GlyphFor(%s, ascii) = %q, want %q", c.role, got, c.ascii)
        }
    }
}
```

- [ ] **Step 2: Run test, expect fail**

```
go test ./internal/uikit/ -run TestGlyphFor_NewDomainRoles -v
```
Expected: FAIL with "undefined: GlyphSeparator" (and similar for the other roles).

- [ ] **Step 3: Add the role constants**

In `internal/uikit/glyph.go`, in the `// Domain / music / identity` section (around line 84):

```go
    // Domain / music / identity
    GlyphMusicNote  GlyphRole = "music.note"
    GlyphDoubleNote GlyphRole = "music.double"
    GlyphPremium    GlyphRole = "music.premium"
    GlyphFreeTier   GlyphRole = "music.free"
    GlyphCloud      GlyphRole = "music.cloud"
    GlyphPlaylist   GlyphRole = "music.playlist"

    // Generic separators
    GlyphSeparator  GlyphRole = "sep.bullet"

    // Device-type icons (devices pane)
    GlyphDeviceComputer GlyphRole = "device.computer"
    GlyphDevicePhone    GlyphRole = "device.phone"
    GlyphDeviceSpeaker  GlyphRole = "device.speaker"
    GlyphDeviceTV       GlyphRole = "device.tv"
```

- [ ] **Step 4: Add the table entries**

In the same file, in `glyphTable`:

```go
    // Domain
    GlyphMusicNote:  {"♪", "*"},
    GlyphDoubleNote: {"♫", "**"},
    GlyphPremium:    {"♛", "*P"},
    GlyphFreeTier:   {"○", "(o)"},
    GlyphCloud:      {"☁", "(c)"},
    GlyphPlaylist:   {"▤", "[=]"},

    // Separators
    GlyphSeparator: {"·", "|"},

    // Devices
    GlyphDeviceComputer: {"⊡", "[c]"},
    GlyphDevicePhone:    {"⊞", "[p]"},
    GlyphDeviceSpeaker:  {"⊟", "[s]"},
    GlyphDeviceTV:       {"⊠", "[tv]"},
```

- [ ] **Step 5: Run test, expect pass**

```
go test ./internal/uikit/ -run TestGlyphFor_NewDomainRoles -v
```

- [ ] **Step 6: Commit**

```bash
git add internal/uikit/glyph.go internal/uikit/glyph_test.go
git commit -m "$(cat <<'EOF'
feat(uikit): add separator, playlist, and device-type GlyphRoles

Catalogue additions surfaced by the glyph-fallback audit:
- GlyphSeparator (·/|) for KeyBar and prose-list separators
- GlyphPlaylist (▤/[=]) for search-result playlist badges
- GlyphDeviceComputer/Phone/Speaker/TV (⊡⊞⊟⊠) for the devices overlay

Spec: docs/superpowers/specs/2026-04-29-glyph-fallback-audit-design.md §4.1
EOF
)"
```

---

### Task 1.2: Add keyboard-chord and superscript GlyphRoles

**Files:**
- Modify: `internal/uikit/glyph.go`
- Modify: `internal/uikit/glyph_test.go`

Implements the §4.9 (keyboard chords) and §4.10 (superscripts) sections of the design doc that exist on paper but not in the code.

- [ ] **Step 1: Write the failing test**

Append to `internal/uikit/glyph_test.go`:

```go
func TestGlyphFor_KeyboardChords(t *testing.T) {
    cases := []struct {
        role           GlyphRole
        unicode, ascii string
    }{
        {GlyphEnter, "⏎", "Enter"},
        {GlyphEscape, "⎋", "Esc"},
        {GlyphTab, "⇥", "Tab"},
        {GlyphBackspace, "⌫", "BS"},
        {GlyphSpace, "␣", "Space"},
    }
    for _, c := range cases {
        if got := GlyphFor(c.role, GlyphUnicode); got != c.unicode {
            t.Errorf("GlyphFor(%s, unicode) = %q, want %q", c.role, got, c.unicode)
        }
        if got := GlyphFor(c.role, GlyphASCII); got != c.ascii {
            t.Errorf("GlyphFor(%s, ascii) = %q, want %q", c.role, got, c.ascii)
        }
    }
}

func TestGlyphFor_Superscripts(t *testing.T) {
    cases := []struct {
        role           GlyphRole
        unicode, ascii string
    }{
        {GlyphSuperscript0, "⁰", "0"},
        {GlyphSuperscript1, "¹", "1"},
        {GlyphSuperscript2, "²", "2"},
        {GlyphSuperscript3, "³", "3"},
        {GlyphSuperscript4, "⁴", "4"},
        {GlyphSuperscript5, "⁵", "5"},
        {GlyphSuperscript6, "⁶", "6"},
        {GlyphSuperscript7, "⁷", "7"},
        {GlyphSuperscript8, "⁸", "8"},
        {GlyphSuperscript9, "⁹", "9"},
        {GlyphSuperscriptPlus, "⁺", "+"},
        {GlyphSuperscriptMinus, "⁻", "-"},
    }
    for _, c := range cases {
        if got := GlyphFor(c.role, GlyphUnicode); got != c.unicode {
            t.Errorf("GlyphFor(%s, unicode) = %q, want %q", c.role, got, c.unicode)
        }
        if got := GlyphFor(c.role, GlyphASCII); got != c.ascii {
            t.Errorf("GlyphFor(%s, ascii) = %q, want %q", c.role, got, c.ascii)
        }
    }
}
```

- [ ] **Step 2: Run tests, expect fail**

```
go test ./internal/uikit/ -run "TestGlyphFor_KeyboardChords|TestGlyphFor_Superscripts" -v
```

- [ ] **Step 3: Add chord and superscript role constants**

Append to the `const` block in `internal/uikit/glyph.go`:

```go
    // Keyboard chords (text-first; only arrows, Enter, Esc, etc. have glyphs)
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

- [ ] **Step 4: Add the table entries**

Append to `glyphTable`:

```go
    // Keyboard chords
    GlyphEnter:     {"⏎", "Enter"},
    GlyphEscape:    {"⎋", "Esc"},
    GlyphTab:       {"⇥", "Tab"},
    GlyphBackspace: {"⌫", "BS"},
    GlyphSpace:     {"␣", "Space"},

    // Superscripts
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

- [ ] **Step 5: Run tests, expect pass**

```
go test ./internal/uikit/ -run "TestGlyphFor_KeyboardChords|TestGlyphFor_Superscripts" -v
```

- [ ] **Step 6: Commit**

```bash
git add internal/uikit/glyph.go internal/uikit/glyph_test.go
git commit -m "$(cat <<'EOF'
feat(uikit): implement §4.9 keyboard-chord and §4.10 superscript GlyphRoles

These rows existed in docs/TUI-DESIGN-SYSTEM.md §4.9–§4.10 but were never
exposed as GlyphRole constants. Required for upcoming pane-chrome migration
(superscripts) and future chord-rendering paths (chords).

Spec: docs/superpowers/specs/2026-04-29-glyph-fallback-audit-design.md §4.1
EOF
)"
```

---

### Task 1.3: Update TUI-DESIGN-SYSTEM.md with new catalogue rows

**Files:**
- Modify: `docs/TUI-DESIGN-SYSTEM.md`

- [ ] **Step 1: Add the new domain roles to §4.6**

Find the §4.6 (Domain / music / identity) table in `docs/TUI-DESIGN-SYSTEM.md` and append:

```markdown
| playlist badge | `▤` | `[=]` | Search-result row, playlist pane |
```

- [ ] **Step 2: Add the separator row**

Add a new sub-section after §4.6:

```markdown
### 4.6a Generic separators

| Role | Unicode | ASCII | Where used |
|---|---|---|---|
| separator (bullet style) | `·` | `\|` | KeyBar, prose lists, search-delegate row separators |
```

- [ ] **Step 3: Add the device-type icons**

Add a new sub-section:

```markdown
### 4.6b Device-type icons

Used in the devices overlay to indicate device class.

| Role | Unicode | ASCII |
|---|---|---|
| computer | `⊡` | `[c]` |
| phone | `⊞` | `[p]` |
| speaker | `⊟` | `[s]` |
| tv | `⊠` | `[tv]` |
```

- [ ] **Step 4: Mark §4.9 and §4.10 as implemented**

In `docs/TUI-DESIGN-SYSTEM.md` §4.9 and §4.10, no row content changes — but if either section has any "future" or "reserved" qualifier, remove it. The roles are now live.

- [ ] **Step 5: Verify markdown renders**

Run any markdown linter you use locally, or manually inspect — the file should still validate.

- [ ] **Step 6: Commit**

```bash
git add docs/TUI-DESIGN-SYSTEM.md
git commit -m "$(cat <<'EOF'
docs(tui-design-system): add separator, playlist, and device-type catalogue rows

Documents the GlyphRoles introduced in the prior commits. Required by the
"new catalogue entries land alongside doc updates" rule in CLAUDE.md.
EOF
)"
```

---

### Task 1.4: Export uikit.SpinnerFrames(mode) and refactor Spinner to use it

**Files:**
- Create: `internal/uikit/spinner_frames.go`
- Create: `internal/uikit/spinner_frames_test.go`
- Modify: `internal/uikit/spinner.go`

Splits the spinner-frame source out of `spinner.go` so `cliout` can consume it without taking a dependency on the entire spinner primitive.

- [ ] **Step 1: Write the failing test**

Create `internal/uikit/spinner_frames_test.go`:

```go
package uikit

import "testing"

func TestSpinnerFrames_Unicode(t *testing.T) {
    frames := SpinnerFrames(GlyphUnicode)
    if len(frames) == 0 {
        t.Fatal("SpinnerFrames(unicode) returned empty slice")
    }
    // Unicode set must be braille
    want := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
    if len(frames) != len(want) {
        t.Fatalf("len(frames) = %d, want %d", len(frames), len(want))
    }
    for i, f := range frames {
        if f != want[i] {
            t.Errorf("frames[%d] = %q, want %q", i, f, want[i])
        }
    }
}

func TestSpinnerFrames_ASCII(t *testing.T) {
    frames := SpinnerFrames(GlyphASCII)
    want := []string{"|", "/", "-", "\\"}
    if len(frames) != len(want) {
        t.Fatalf("len(frames) = %d, want %d", len(frames), len(want))
    }
    for i, f := range frames {
        if f != want[i] {
            t.Errorf("frames[%d] = %q, want %q", i, f, want[i])
        }
    }
}
```

- [ ] **Step 2: Run test, expect fail**

```
go test ./internal/uikit/ -run TestSpinnerFrames -v
```

- [ ] **Step 3: Create the implementation**

Create `internal/uikit/spinner_frames.go`:

```go
package uikit

// spinnerFramesUnicode is the braille spinner frame set used in unicode mode.
// Shared by uikit.Spinner and cliout.Spinner.
var spinnerFramesUnicode = []string{
    "⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
}

// spinnerFramesASCII is the rotating-bar set used in ascii mode.
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

- [ ] **Step 4: Refactor `internal/uikit/spinner.go` to use the helper**

Locate the existing frame array in `internal/uikit/spinner.go` (it will be a `var spinnerFrames = []string{...}` or similar). Replace the inline initialization with a call to `SpinnerFrames` at construction time inside `NewSpinner`. The exact diff depends on the existing structure — read the file first, then change the source to use `SpinnerFrames(ActiveMode())` at the point where frames are stored on the Spinner struct or used in `View()`.

- [ ] **Step 5: Run all uikit tests**

```
go test ./internal/uikit/ -v
```
Expected: PASS for both new and existing tests.

- [ ] **Step 6: Commit**

```bash
git add internal/uikit/spinner_frames.go internal/uikit/spinner_frames_test.go internal/uikit/spinner.go
git commit -m "$(cat <<'EOF'
feat(uikit): export SpinnerFrames(mode) as the shared frame source

Splits the spinner frame arrays out of spinner.go so internal/cliout can
import them in Phase 2 without dragging in the full Bubble Tea spinner
primitive. Also adds the missing ASCII fallback set (|/-\) that cliout
currently lacks entirely.

Spec: docs/superpowers/specs/2026-04-29-glyph-fallback-audit-design.md §4.4
EOF
)"
```

---

### Task 1.5: Fix uikit ellipsis hardcodes

**Files:**
- Modify: `internal/uikit/list_row.go` (line 48)
- Modify: `internal/uikit/toast.go` (line 112)
- Modify: `internal/uikit/list_row_test.go` and `toast_test.go` (assertions)

- [ ] **Step 1: Write the failing test**

Add to `internal/uikit/list_row_test.go`:

```go
func TestListRow_PadOrTruncate_AsciiEllipsis(t *testing.T) {
    SetModeForTest(GlyphASCII)
    defer SetModeForTest(GlyphUnicode)

    // A label longer than the available width must truncate with the ascii
    // ellipsis form ("...") instead of the unicode "…".
    out := PadOrTruncate("Long Label That Exceeds Width", 8)
    if !strings.Contains(out, "...") {
        t.Errorf("ascii truncation should contain %q, got %q", "...", out)
    }
    if strings.Contains(out, "…") {
        t.Errorf("ascii truncation must not contain unicode ellipsis, got %q", out)
    }
}
```

(Adjust the test's import block to include `strings` if not already present.)

- [ ] **Step 2: Run test, expect fail**

```
go test ./internal/uikit/ -run TestListRow_PadOrTruncate_AsciiEllipsis -v
```
Expected: FAIL because `PadOrTruncate` hardcodes `"…"`.

- [ ] **Step 3: Fix `list_row.go:48`**

Open `internal/uikit/list_row.go` and locate the hardcoded `"…"` (the existing comment in the spec says line 48, in the body of `PadOrTruncate`). Replace the literal with:

```go
ellipsis := GlyphFor(GlyphEllipsis, ActiveMode())
candidate := string(runes) + ellipsis
```

(Where `runes` is the existing variable holding the truncated rune slice. Adjust the local-variable layout to fit the existing function — the goal is to replace the literal `"…"` with the catalogue lookup.)

- [ ] **Step 4: Fix `toast.go:112`**

Open `internal/uikit/toast.go` line 112. The audit notes a raw rune assignment `runes[max-1] = '…'`. Replace with:

```go
// Use the glyph catalogue so ascii mode renders "..." instead of "…".
ell := GlyphFor(GlyphEllipsis, ActiveMode())
// `ell` may be multi-rune in ascii mode, so slice off the last 1-3 runes
// of the truncated buffer and append the ellipsis form instead of mutating
// in place.
runes = append(runes[:max-len([]rune(ell))], []rune(ell)...)
```

Read the surrounding code first; if the existing function signature operates on a string instead of a rune slice, swap to whatever matches the function shape — the goal is "ellipsis comes from the catalogue, not a literal rune."

- [ ] **Step 5: Add a parallel ASCII test for `Toast`**

Add to `internal/uikit/toast_test.go`:

```go
func TestToast_TruncatedTitle_AsciiEllipsis(t *testing.T) {
    SetModeForTest(GlyphASCII)
    defer SetModeForTest(GlyphUnicode)

    long := strings.Repeat("a", 100)
    tt := Toast{Intent: ToastInfo, Title: long}
    out := tt.RenderTitle() // or whatever the existing render entrypoint is
    if !strings.Contains(out, "...") {
        t.Errorf("ascii toast title should truncate with %q, got %q", "...", out)
    }
}
```

(Adjust to match the actual rendering entrypoint in `toast.go` — the test's purpose is "long title in ascii mode contains `...` not `…`".)

- [ ] **Step 6: Run all uikit tests, expect pass**

```
go test ./internal/uikit/ -v
```

- [ ] **Step 7: Commit**

```bash
git add internal/uikit/list_row.go internal/uikit/toast.go internal/uikit/list_row_test.go internal/uikit/toast_test.go
git commit -m "$(cat <<'EOF'
fix(uikit): route ellipsis through GlyphFor in ListRow and Toast

list_row.go:48 and toast.go:112 hardcoded the "…" literal regardless of
mode. Now both call GlyphFor(GlyphEllipsis, ActiveMode()) so ascii mode
renders "..." correctly.

Spec: docs/superpowers/specs/2026-04-29-glyph-fallback-audit-design.md §3.1
EOF
)"
```

---

### Task 1.6: Fix HeaderBar separator hardcode

**Files:**
- Modify: `internal/uikit/header_bar.go` (line 52)
- Modify: `internal/uikit/header_bar_test.go` (add ASCII assertion)

- [ ] **Step 1: Write the failing test**

Add to `internal/uikit/header_bar_test.go`:

```go
func TestHeaderBar_AsciiSeparator(t *testing.T) {
    SetModeForTest(GlyphASCII)
    defer SetModeForTest(GlyphUnicode)

    h := HeaderBar{
        Width:   60,
        AppName: "spotnik",
        Page:    "A",
        Preset:  0,
        Theme:   theme.Load("black"),
    }
    out := stripANSI(h.Render()) // use existing test helper from the package
    if !strings.Contains(out, " - ") {
        t.Errorf("ascii separator should be %q, got: %q", " - ", out)
    }
    if strings.Contains(out, " ─ ") {
        t.Errorf("ascii output must not contain unicode separator, got: %q", out)
    }
}
```

- [ ] **Step 2: Run test, expect fail**

```
go test ./internal/uikit/ -run TestHeaderBar_AsciiSeparator -v
```

- [ ] **Step 3: Fix `header_bar.go:52`**

Find `sep := muted.Render(" ─ ")`. Replace with:

```go
sep := muted.Render(" " + GlyphFor(GlyphHRule, ActiveMode()) + " ")
```

- [ ] **Step 4: Run test, expect pass**

```
go test ./internal/uikit/ -run TestHeaderBar -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/uikit/header_bar.go internal/uikit/header_bar_test.go
git commit -m "fix(uikit): route HeaderBar separator through GlyphFor(GlyphHRule)"
```

---

### Task 1.7: Fix KeyBar separator hardcode

**Files:**
- Modify: `internal/uikit/key_bar.go` (lines 31–34)
- Modify: `internal/uikit/key_bar_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/uikit/key_bar_test.go`:

```go
func TestKeyBar_AsciiSeparator(t *testing.T) {
    SetModeForTest(GlyphASCII)
    defer SetModeForTest(GlyphUnicode)

    bar := KeyBar{
        Bindings: []key.Binding{
            key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "filter")),
            key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
        },
        Theme: theme.Load("black"),
    }
    out := stripANSI(bar.Render())
    if !strings.Contains(out, " | ") {
        t.Errorf("ascii separator should be %q, got: %q", " | ", out)
    }
    if strings.Contains(out, " · ") {
        t.Errorf("ascii output must not contain unicode bullet separator, got: %q", out)
    }
}
```

- [ ] **Step 2: Run test, expect fail**

```
go test ./internal/uikit/ -run TestKeyBar_AsciiSeparator -v
```

- [ ] **Step 3: Replace the hardcoded branching**

Find the existing `if`/`else` that selects `" · "` vs `" | "` (lines 31–34). Replace with:

```go
sep := " " + GlyphFor(GlyphSeparator, ActiveMode()) + " "
```

- [ ] **Step 4: Run test, expect pass**

```
go test ./internal/uikit/ -run TestKeyBar -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/uikit/key_bar.go internal/uikit/key_bar_test.go
git commit -m "fix(uikit): route KeyBar separator through GlyphFor(GlyphSeparator)"
```

---

### Task 1.8: Populate StatusBar BorderConfig glyph fields

**Files:**
- Modify: `internal/uikit/status_bar.go`
- Modify: `internal/uikit/status_bar_test.go`

`status_bar.go` calls `layout.RenderPaneBorder` without setting the glyph fields. The fix is to populate them via `GlyphFor` like `pane_chrome.go` does.

- [ ] **Step 1: Write the failing test**

Add to `internal/uikit/status_bar_test.go`:

```go
func TestStatusBar_AsciiBorder(t *testing.T) {
    SetModeForTest(GlyphASCII)
    defer SetModeForTest(GlyphUnicode)

    bar := StatusBar{
        Width:    60,
        Bindings: testKeyMap{}, // existing test helper, or build a help.KeyMap inline
        Theme:    theme.Load("black"),
    }
    out := stripANSI(bar.Render())
    // ascii mode must use + corners and - rule, NOT ╭╮╰╯─
    if strings.ContainsAny(out, "╭╮╰╯─│") {
        t.Errorf("ascii output must not contain unicode border glyphs, got: %q", out)
    }
    if !strings.Contains(out, "+") {
        t.Errorf("ascii output must contain '+' corner, got: %q", out)
    }
}
```

- [ ] **Step 2: Run test, expect fail**

```
go test ./internal/uikit/ -run TestStatusBar_AsciiBorder -v
```

- [ ] **Step 3: Update `status_bar.go` to populate glyph fields**

Find the `layout.RenderPaneBorder(content, layout.BorderConfig{...})` call and extend the `BorderConfig` literal with the same six fields `pane_chrome.go` populates:

```go
m := ActiveMode()
cfg := layout.BorderConfig{
    Width:       s.Width,
    Height:      s.Height,
    AccentColor: s.Theme.PaneBorder0(), // or whatever existing field
    Theme:       s.Theme,
    CornerTL:    GlyphFor(GlyphCornerTL, m),
    CornerTR:    GlyphFor(GlyphCornerTR, m),
    CornerBL:    GlyphFor(GlyphCornerBL, m),
    CornerBR:    GlyphFor(GlyphCornerBR, m),
    HRule:       GlyphFor(GlyphHRule, m),
    VRule:       GlyphFor(GlyphVRule, m),
}
return layout.RenderPaneBorder(content, cfg)
```

- [ ] **Step 4: Run test, expect pass**

```
go test ./internal/uikit/ -run TestStatusBar -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/uikit/status_bar.go internal/uikit/status_bar_test.go
git commit -m "fix(uikit): populate StatusBar BorderConfig glyph fields via GlyphFor"
```

---

### Task 1.9: Add ASCII tests for EmptyState, URLBox, FormField

**Files:**
- Modify: `internal/uikit/empty_state_test.go`
- Modify: `internal/uikit/url_box_test.go`
- Modify: `internal/uikit/form_field_test.go`

These primitives lack ASCII snapshot tests despite being part of the core surface.

- [ ] **Step 1: Add ASCII test for EmptyState**

Append to `internal/uikit/empty_state_test.go`:

```go
func TestEmptyState_AsciiMode(t *testing.T) {
    SetModeForTest(GlyphASCII)
    defer SetModeForTest(GlyphUnicode)

    es := EmptyState{
        Text:   "Nothing in queue",
        Hint:   "Press / to search",
        Width:  40,
        Height: 6,
        Theme:  theme.Load("black"),
    }
    out := stripANSI(es.Render())
    lines := strings.Split(out, "\n")
    if len(lines) != 6 {
        t.Errorf("EmptyState should produce exactly Height=6 lines, got %d", len(lines))
    }
    if !strings.Contains(out, "Nothing in queue") {
        t.Errorf("output should contain text, got: %q", out)
    }
    if !strings.Contains(out, "Press / to search") {
        t.Errorf("output should contain hint, got: %q", out)
    }
}
```

- [ ] **Step 2: Add ASCII test for URLBox**

Append to `internal/uikit/url_box_test.go`:

```go
func TestURLBox_AsciiMode(t *testing.T) {
    SetModeForTest(GlyphASCII)
    defer SetModeForTest(GlyphUnicode)

    box := URLBox{
        URL:   "http://localhost:8888/callback",
        Width: 40,
        Theme: theme.Load("black"),
    }
    out := stripANSI(box.Render())
    if strings.ContainsAny(out, "╭╮╰╯") {
        t.Errorf("ascii URLBox must not contain unicode corners, got: %q", out)
    }
    if !strings.Contains(out, "http://localhost:8888/callback") {
        t.Errorf("URL content missing, got: %q", out)
    }
}
```

- [ ] **Step 3: Add ASCII test for FormField**

Append to `internal/uikit/form_field_test.go`:

```go
func TestFormField_AsciiValidationError(t *testing.T) {
    SetModeForTest(GlyphASCII)
    defer SetModeForTest(GlyphUnicode)

    f := NewFormField(FormFieldConfig{
        Label:    "Client ID",
        Validate: func(s string) error { return errors.New("must be 32 chars") },
        Theme:    theme.Load("black"),
    })
    f.SetValue("abc")
    out := stripANSI(f.Render())
    // ascii failure glyph is "x" (not ✗)
    if !strings.Contains(out, "x must be 32 chars") {
        t.Errorf("ascii validation error should contain %q, got: %q", "x must be 32 chars", out)
    }
    if strings.Contains(out, "✗") {
        t.Errorf("ascii output must not contain unicode failure glyph, got: %q", out)
    }
}
```

- [ ] **Step 4: Run all three test files**

```
go test ./internal/uikit/ -run "TestEmptyState_AsciiMode|TestURLBox_AsciiMode|TestFormField_AsciiValidationError" -v
```
Expected: PASS for EmptyState and URLBox (they were always mode-correct, just untested). FormField may PASS already since `GlyphFor(GlyphError, mode)` is already used — verify, and only debug if it fails.

- [ ] **Step 5: Commit**

```bash
git add internal/uikit/empty_state_test.go internal/uikit/url_box_test.go internal/uikit/form_field_test.go
git commit -m "$(cat <<'EOF'
test(uikit): add ASCII snapshot tests for EmptyState, URLBox, FormField

Closes the test-coverage gaps identified in §3.1 of the audit. Confirms
these primitives already render correctly in ascii mode.

Spec: docs/superpowers/specs/2026-04-29-glyph-fallback-audit-design.md §3.1
EOF
)"
```

---

## Phase 2 — cliout integration

### Task 2.1: Map Status → GlyphRole and rewrite cliout.statusGlyph

**Files:**
- Modify: `internal/cliout/message.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/cliout/message_test.go`:

```go
func TestStatusGlyph_AsciiMode(t *testing.T) {
    uikit.SetModeForTest(uikit.GlyphASCII)
    defer uikit.SetModeForTest(uikit.GlyphUnicode)

    cases := []struct {
        s    Status
        want string
    }{
        {Active, "(*)"},
        {Inactive, "( )"},
        {StatusSuccess, "+"},
        {StatusFailure, "x"},
        {StatusWarning, "!"},
        {Pending, "(r)"},
    }
    for _, c := range cases {
        if got := statusGlyph(c.s); got != c.want {
            t.Errorf("statusGlyph(%v) ascii = %q, want %q", c.s, got, c.want)
        }
    }
}
```

(Add `import "github.com/initgrep-apps/spotnik/internal/uikit"` to the test file.)

- [ ] **Step 2: Run test, expect fail**

```
go test ./internal/cliout/ -run TestStatusGlyph_AsciiMode -v
```

- [ ] **Step 3: Add the import to message.go**

```go
import (
    "fmt"
    "strings"

    "github.com/charmbracelet/lipgloss"
    "github.com/initgrep-apps/spotnik/internal/uikit"
)
```

- [ ] **Step 4: Replace `statusGlyph()` (lines 36–53)**

```go
// statusGlyphRole maps a cliout Status to a uikit GlyphRole. Single source of
// truth — cliout intents and uikit Toast intents share the same catalogue.
var statusGlyphRole = map[Status]uikit.GlyphRole{
    Active:         uikit.GlyphActive,
    Inactive:       uikit.GlyphInactive,
    StatusSuccess:  uikit.GlyphSuccess,
    StatusFailure:  uikit.GlyphError,
    StatusWarning:  uikit.GlyphWarning,
    Pending:        uikit.GlyphLocked, // ◌ in unicode, (r) in ascii
}

// statusGlyph returns the rendered glyph for a status value, mode-aware.
func statusGlyph(s Status) string {
    role, ok := statusGlyphRole[s]
    if !ok {
        return "?"
    }
    return uikit.GlyphFor(role, uikit.ActiveMode())
}
```

- [ ] **Step 5: Run test, expect pass**

```
go test ./internal/cliout/ -run TestStatusGlyph_AsciiMode -v
```

- [ ] **Step 6: Run the existing unicode tests, expect pass**

```
go test ./internal/cliout/ -run TestStatusGlyph -v
```
Existing tests assert the unicode glyphs (`◉ ◎ ✓ ✗ ◬ ◌`); they should continue to pass because `uikit.SetModeForTest` defaults back to unicode after each test.

- [ ] **Step 7: Commit**

```bash
git add internal/cliout/message.go internal/cliout/message_test.go
git commit -m "$(cat <<'EOF'
refactor(cliout): route statusGlyph through uikit.GlyphFor

cliout was maintaining a parallel hardcoded glyph set and ignoring
ui.glyphs config entirely. Now Status values map to uikit.GlyphRole and
glyphs resolve via uikit.GlyphFor(role, uikit.ActiveMode()) so ascii mode
finally produces ascii output on the CLI side.

Spec: docs/superpowers/specs/2026-04-29-glyph-fallback-audit-design.md §3.4 / §4.4
EOF
)"
```

---

### Task 2.2: Route Hint arrow through GlyphFor

**Files:**
- Modify: `internal/cliout/message.go` (around line 176)

- [ ] **Step 1: Write the failing test**

Append to `internal/cliout/message_test.go`:

```go
func TestHint_AsciiArrow(t *testing.T) {
    uikit.SetModeForTest(uikit.GlyphASCII)
    defer uikit.SetModeForTest(uikit.GlyphUnicode)

    h := Hint{Verb: "Run", Cmd: "spotnik auth login", Tail: "to continue"}
    out := stripANSI(h.render(testPalette()))
    if !strings.HasPrefix(out, "> ") {
        t.Errorf("ascii hint must start with %q, got: %q", "> ", out)
    }
    if strings.Contains(out, "→") {
        t.Errorf("ascii hint must not contain unicode arrow, got: %q", out)
    }
}
```

- [ ] **Step 2: Replace the literal `→` in `Hint.render()`**

In `internal/cliout/message.go` line 176, change:

```go
arrow := lipgloss.NewStyle().Foreground(p.Accent).Bold(true).Render("→")
```

to:

```go
arrow := lipgloss.NewStyle().
    Foreground(p.Accent).Bold(true).
    Render(uikit.GlyphFor(uikit.GlyphInfo, uikit.ActiveMode()))
```

- [ ] **Step 3: Run test, expect pass**

```
go test ./internal/cliout/ -run TestHint_AsciiArrow -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/cliout/message.go internal/cliout/message_test.go
git commit -m "fix(cliout): route Hint arrow through uikit.GlyphFor(GlyphInfo)"
```

---

### Task 2.3: Source spinner frames from uikit.SpinnerFrames

**Files:**
- Modify: `internal/cliout/spinner.go`
- Modify: `internal/cliout/spinner_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/cliout/spinner_test.go`:

```go
func TestSpinnerFrames_AsciiSet(t *testing.T) {
    uikit.SetModeForTest(uikit.GlyphASCII)
    defer uikit.SetModeForTest(uikit.GlyphUnicode)

    // The package-private resolveSpinnerFrames helper (added below) must
    // return the ascii set when ActiveMode is GlyphASCII.
    frames := resolveSpinnerFrames()
    if len(frames) == 0 || frames[0] != "|" {
        t.Errorf("ascii frames must start with '|', got: %#v", frames)
    }
}
```

- [ ] **Step 2: Add the import**

In `internal/cliout/spinner.go`:

```go
import (
    "context"
    "fmt"
    "io"
    "os"
    "os/signal"
    "sync"
    "syscall"
    "time"

    "github.com/charmbracelet/lipgloss"
    "github.com/initgrep-apps/spotnik/internal/uikit"
)
```

- [ ] **Step 3: Replace the inline frame array with a helper**

Remove `var spinnerFrames = []string{...}` (line 16). Add:

```go
// resolveSpinnerFrames returns the active spinner frame set, captured once
// at spinner start. cliout deliberately does not re-resolve mid-animation
// because uikit.ActiveMode is fixed for the lifetime of the process.
func resolveSpinnerFrames() []string {
    return uikit.SpinnerFrames(uikit.ActiveMode())
}
```

- [ ] **Step 4: Update the run goroutine**

In `(h *SpinnerHandle).run()`, replace the reference `spinnerFrames[i%len(spinnerFrames)]` with a local snapshot:

```go
frames := resolveSpinnerFrames()
// ... existing styles ...
render := func() {
    line := padding + frameStyle.Render(frames[i%len(frames)]) + " " + textStyle.Render(h.text)
    _, _ = fmt.Fprint(h.w, "\r\x1b[K"+line)
    i++
}
```

- [ ] **Step 5: Run all spinner tests**

```
go test ./internal/cliout/ -run TestSpinner -v
```

- [ ] **Step 6: Commit**

```bash
git add internal/cliout/spinner.go internal/cliout/spinner_test.go
git commit -m "$(cat <<'EOF'
fix(cliout): source spinner frames from uikit.SpinnerFrames

Removes the duplicated braille frame array. ascii mode now produces the
rotating |/-\ set that piped/non-UTF-8 terminals can render.

Spec: docs/superpowers/specs/2026-04-29-glyph-fallback-audit-design.md §3.4
EOF
)"
```

---

### Task 2.4: Verify cmd/root.go calls uikit.Use to cover cliout

**Files:**
- Modify: `cmd/root.go` (verify, possibly extend)

cliout now reads `uikit.ActiveMode()`. Confirm that the existing `uikit.Use(cfg.UI.Glyphs)` call in `cmd/root.go` runs before any cliout function does.

- [ ] **Step 1: Read `cmd/root.go` and locate the existing `uikit.Use` call**

```
grep -n "uikit.Use" cmd/root.go
```
Note the line number. Confirm it runs before any cliout entrypoint (e.g. before `cliout.Write` could be called from a subcommand).

- [ ] **Step 2: Write a regression test if not already present**

In `cmd/root_test.go`, locate the existing test referenced by the audit (`root_test.go:1060` per memory, which sets `glyphs = "ascii"` and asserts `uikit.ActiveMode() == GlyphASCII`). Verify it is present and passes:

```
go test ./cmd/ -run TestLoadConfigFromPath -v
```

If absent, add a test that loads a config with `glyphs = "ascii"`, runs the bootstrap function, and asserts `uikit.ActiveMode() == uikit.GlyphASCII`.

- [ ] **Step 3: Add a cliout-side smoke test**

Add to `internal/cliout/message_test.go`:

```go
func TestStatusGlyph_HonoursUikitMode(t *testing.T) {
    // Independently of cliout.SetTestMode, switching uikit's mode must change
    // cliout's glyph output. This documents the dependency direction.
    uikit.SetModeForTest(uikit.GlyphUnicode)
    if got := statusGlyph(StatusSuccess); got != "✓" {
        t.Errorf("unicode mode: statusGlyph(Success) = %q, want %q", got, "✓")
    }
    uikit.SetModeForTest(uikit.GlyphASCII)
    if got := statusGlyph(StatusSuccess); got != "+" {
        t.Errorf("ascii mode: statusGlyph(Success) = %q, want %q", got, "+")
    }
    uikit.SetModeForTest(uikit.GlyphUnicode) // restore
}
```

- [ ] **Step 4: Run the test, expect pass**

```
go test ./internal/cliout/ -run TestStatusGlyph_HonoursUikitMode -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/cliout/message_test.go
git commit -m "test(cliout): pin the cliout-honours-uikit-mode contract"
```

---

### Task 2.5: Align cliout.SetTestMode with uikit.SetModeForTest

**Files:**
- Modify: `internal/cliout/tty.go`

- [ ] **Step 1: Update `SetTestMode` to also pin uikit mode**

In `internal/cliout/tty.go`, replace `SetTestMode`:

```go
// SetTestMode enables or disables test mode. In test mode, pinASCII is
// called immediately so colour output is deterministic, AND uikit's glyph
// mode is pinned to GlyphASCII for the same reason. Tests call this in
// TestMain.
func SetTestMode(enabled bool) {
    testModeMu.Lock()
    defer testModeMu.Unlock()
    testMode = enabled
    if enabled {
        pinASCII()
        uikit.SetModeForTest(uikit.GlyphASCII)
    }
}
```

(Add `"github.com/initgrep-apps/spotnik/internal/uikit"` to the import block if not already present.)

- [ ] **Step 2: Document `pinASCII` clearly as colour-only**

Update the existing comment on `pinASCII`:

```go
// pinASCII forces lipgloss to render without ANSI colour escapes. Called
// once when output is non-TTY or NO_COLOR is set, or when SetTestMode(true)
// runs.
//
// NOTE: this controls *colour* only. Glyph mode is sourced independently
// from uikit.ActiveMode() — see internal/uikit/config.go.
func pinASCII() {
    profileOnce.Do(func() {
        lipgloss.SetColorProfile(termenv.Ascii)
    })
}
```

- [ ] **Step 3: Run all cliout tests**

```
go test ./internal/cliout/ -v
```
Expected: existing tests should still pass — `SetTestMode(true)` now pins uikit to ASCII, but cliout tests historically asserted unicode output. Audit each test that fails:
- If the test was relying on unicode output with `SetTestMode(true)`, that test was masking the bug. Update the assertion to expect ASCII.
- If a test sets `SetTestMode(true)` then explicitly switches uikit back to unicode for that case, leave it.

- [ ] **Step 4: Commit**

```bash
git add internal/cliout/tty.go
git commit -m "$(cat <<'EOF'
fix(cliout): align SetTestMode with uikit.SetModeForTest(GlyphASCII)

Previously cliout test mode pinned colour to ASCII but glyphs stayed
unicode, so ascii fallback paths in cliout were never tested. Now
test mode pins both. Tests that asserted unicode glyphs under
SetTestMode(true) were masking the bug — those assertions are
updated in this commit.
EOF
)"
```

---

### Task 2.6: Rebaseline cliout tests for both modes

**Files:**
- Modify: every `internal/cliout/*_test.go`

After Task 2.5 the test mode pins uikit to ASCII. Existing assertions that hardcode unicode glyphs (`◉ ◎ ✓ ✗ ◬ ◌ →`) under `SetTestMode(true)` will now fail.

- [ ] **Step 1: Run full cliout suite, list failures**

```
go test ./internal/cliout/ -v 2>&1 | tee /tmp/cliout-fail.log
grep -E "^---? FAIL" /tmp/cliout-fail.log
```

- [ ] **Step 2: For each failing assertion, parameterise via the catalogue**

Replace literal-rune assertions with calls to `uikit.GlyphFor`:

Before:
```go
if !strings.Contains(out, "◉") {
    t.Errorf("expected ◉, got %q", out)
}
```

After:
```go
expected := uikit.GlyphFor(uikit.GlyphActive, uikit.ActiveMode())
if !strings.Contains(out, expected) {
    t.Errorf("expected %q, got %q", expected, out)
}
```

This makes the test assert the *role*, not the codepoint, and survives both modes.

- [ ] **Step 3: Where a test specifically validates a single mode, pin it**

If a test is intentionally checking ASCII output, prefix:

```go
uikit.SetModeForTest(uikit.GlyphASCII)
defer uikit.SetModeForTest(uikit.GlyphUnicode)
```

If unicode-specific:

```go
uikit.SetModeForTest(uikit.GlyphUnicode)
defer uikit.SetModeForTest(uikit.GlyphUnicode)
```

- [ ] **Step 4: Run full cliout suite, expect pass**

```
go test ./internal/cliout/ -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/cliout/
git commit -m "$(cat <<'EOF'
test(cliout): rebaseline tests to assert glyph roles, not codepoints

After SetTestMode pinned uikit to ascii (prior commit), assertions on raw
unicode glyphs began failing — confirming the previous tests were masking
the missing ascii fallback. Each assertion now resolves expected output
through uikit.GlyphFor so it works in either mode.
EOF
)"
```

---

## Phase 3 — Pane chrome migration

### Task 3.1: Migrate renderGrid to uikit.PaneChrome.Render

**Files:**
- Modify: `internal/app/render.go` (lines 417–426 region)
- Modify: `internal/app/render_test.go`

The single largest fix. Replaces the inline `BorderConfig` build in `renderGrid` with a `uikit.PaneChrome.Render` call so every grid pane border honours `ActiveMode()`.

- [ ] **Step 1: Write the failing test**

Append to `internal/app/render_test.go`:

```go
func TestRenderGrid_AsciiBorders(t *testing.T) {
    uikit.SetModeForTest(uikit.GlyphASCII)
    defer uikit.SetModeForTest(uikit.GlyphUnicode)

    a := newTestApp(t) // existing test helper
    a.layout.SetSize(120, 30)
    out := stripANSI(a.renderGrid())

    if strings.ContainsAny(out, "╭╮╰╯") {
        t.Errorf("ascii grid must not contain unicode rounded corners, got snippet: %q",
            out[:min(200, len(out))])
    }
    if !strings.Contains(out, "+") {
        t.Errorf("ascii grid must contain '+' corners, got snippet: %q", out[:min(200, len(out))])
    }
}
```

(`min` and `stripANSI` are existing test helpers.)

- [ ] **Step 2: Run test, expect fail**

```
go test ./internal/app/ -run TestRenderGrid_AsciiBorders -v
```

- [ ] **Step 3: Add the import**

In `internal/app/render.go`:

```go
import (
    // ... existing imports ...
    "github.com/initgrep-apps/spotnik/internal/uikit"
)
```

- [ ] **Step 4: Replace the inline BorderConfig build**

Find the loop in `renderGrid` (lines 411–434 region). Replace the section that builds `cfg` and calls `layout.RenderPaneBorder`:

```go
chrome := uikit.PaneChrome{
    Width:       rect.Width,
    Height:      rect.Height,
    Title:       pane.Title(),
    ToggleKey:   pane.ToggleKey(),
    Actions:     pane.Actions(),
    AccentColor: layout.PaneBorderColor(paneID, a.theme),
    Focused:     pane.IsFocused(),
    Theme:       a.theme,
}
if fqp, ok := pane.(layout.FilterQueryPane); ok {
    chrome.FilterQuery = fqp.ActiveFilterQuery()
}
bordered := chrome.Render(pane.View())
```

- [ ] **Step 5: Run the test, expect pass**

```
go test ./internal/app/ -run TestRenderGrid -v
```

- [ ] **Step 6: Run the full app suite to catch regressions**

```
go test ./internal/app/ -v
```

- [ ] **Step 7: Commit**

```bash
git add internal/app/render.go internal/app/render_test.go
git commit -m "$(cat <<'EOF'
fix(app): route renderGrid through uikit.PaneChrome for ascii fallback

renderGrid was building BorderConfig inline and calling
layout.RenderPaneBorder directly. Layout's resolveGlyphs falls back to
hardcoded unicode when glyph fields are empty, so every grid pane border
rendered as unicode regardless of ui.glyphs config. Now renderGrid calls
uikit.PaneChrome.Render which populates the glyph fields via GlyphFor.

This is the single largest pane-chrome gap — every grid pane on every
page is fixed by this commit.

Spec: docs/superpowers/specs/2026-04-29-glyph-fallback-audit-design.md §3.5
EOF
)"
```

---

### Task 3.2: Migrate themes.go and profile.go to PaneChrome

**Files:**
- Modify: `internal/ui/panes/themes.go` (line 160)
- Modify: `internal/ui/panes/profile.go` (line 183)

Both panes call `layout.RenderPaneBorder` directly without setting glyph fields. Same migration pattern as renderGrid.

- [ ] **Step 1: Write the failing test for themes.go**

Append to `internal/ui/panes/themes_test.go`:

```go
func TestThemesPane_AsciiBorder(t *testing.T) {
    uikit.SetModeForTest(uikit.GlyphASCII)
    defer uikit.SetModeForTest(uikit.GlyphUnicode)

    p := NewThemesPane(theme.Load("black"), nil) // match existing constructor
    p.SetSize(40, 10)
    out := stripANSI(p.View())
    if strings.ContainsAny(out, "╭╮╰╯─│") {
        t.Errorf("ascii themes pane must not contain unicode borders, got: %q", out)
    }
}
```

Same pattern for `profile_test.go`.

- [ ] **Step 2: Run, expect fail**

```
go test ./internal/ui/panes/ -run "TestThemesPane_AsciiBorder|TestProfilePane_AsciiBorder" -v
```

- [ ] **Step 3: Update themes.go**

Open `internal/ui/panes/themes.go` line 160 region. Find the `cfg := layout.BorderConfig{...}` then `layout.RenderPaneBorder(inner, cfg)` pattern. Replace:

```go
chrome := uikit.PaneChrome{
    Width:       p.width,
    Height:      p.height,
    Title:       p.Title(),
    ToggleKey:   p.ToggleKey(),
    Actions:     p.Actions(),
    AccentColor: layout.PaneBorderColor(p.paneID, p.theme),
    Focused:     p.focused,
    Theme:       p.theme,
}
return chrome.Render(inner)
```

(Adjust field references to match what the existing code uses for theme, dimensions, focus.)

- [ ] **Step 4: Update profile.go (line 183)**

Same replacement pattern.

- [ ] **Step 5: Run tests, expect pass**

```
go test ./internal/ui/panes/ -run "TestThemesPane|TestProfilePane" -v
```

- [ ] **Step 6: Commit**

```bash
git add internal/ui/panes/themes.go internal/ui/panes/profile.go internal/ui/panes/themes_test.go internal/ui/panes/profile_test.go
git commit -m "fix(panes): migrate themes and profile panes to uikit.PaneChrome"
```

---

### Task 3.3: Migrate devices.go overlay to OverlayChrome

**Files:**
- Modify: `internal/ui/panes/devices.go` (lines 166–175 region)

The devices pane is a modal overlay, so it should use `uikit.OverlayChrome.Render`, not `PaneChrome`.

- [ ] **Step 1: Write the failing test**

Append to `internal/ui/panes/devices_test.go`:

```go
func TestDevicesOverlay_AsciiBorder(t *testing.T) {
    uikit.SetModeForTest(uikit.GlyphASCII)
    defer uikit.SetModeForTest(uikit.GlyphUnicode)

    o := NewDevicesOverlay(theme.Load("black")) // existing constructor
    o.SetSize(50, 20)
    out := stripANSI(o.View())
    if strings.ContainsAny(out, "╭╮╰╯─│") {
        t.Errorf("ascii devices overlay must not contain unicode borders, got: %q", out)
    }
}
```

- [ ] **Step 2: Replace `layout.RenderPaneBorder` with `uikit.OverlayChrome.Render`**

In `internal/ui/panes/devices.go` lines 166–175, find:

```go
cfg := layout.BorderConfig{...}
return layout.RenderPaneBorder(inner, cfg)
```

Replace with:

```go
chrome := uikit.OverlayChrome{
    Width:   o.width,
    Height:  o.height,
    Title:   "Devices",
    Actions: o.Actions(),
    Theme:   o.theme,
}
return chrome.Render(inner)
```

- [ ] **Step 3: Run tests**

```
go test ./internal/ui/panes/ -run TestDevicesOverlay -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/ui/panes/devices.go internal/ui/panes/devices_test.go
git commit -m "fix(devices): migrate overlay to uikit.OverlayChrome for ascii fallback"
```

---

### Task 3.4: Migrate help_overlay.go to OverlayChrome

**Files:**
- Modify: `internal/ui/panes/help_overlay.go` (lines 152–161)

- [ ] **Step 1: Write the failing test**

Same shape as Task 3.3, in `help_overlay_test.go`. Construct the overlay, call `View()` with mode = ASCII, assert no unicode border characters.

- [ ] **Step 2: Replace the chrome call**

```go
chrome := uikit.OverlayChrome{
    Width:   h.width,
    Height:  h.height,
    Title:   "Help",
    Theme:   h.theme,
}
return chrome.Render(inner)
```

- [ ] **Step 3: Run tests, expect pass**

```
go test ./internal/ui/panes/ -run TestHelpOverlay -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/ui/panes/help_overlay.go internal/ui/panes/help_overlay_test.go
git commit -m "fix(help): migrate help overlay to uikit.OverlayChrome"
```

---

### Task 3.5: Migrate search.go (3 sites) to OverlayChrome

**Files:**
- Modify: `internal/ui/panes/search.go` (lines 778, 880, 943)

The search overlay calls `layout.RenderPaneBorder` from three different render paths (probably one per result-state). All three must migrate.

- [ ] **Step 1: Write the failing test**

Add `TestSearchOverlay_AsciiBorder` to `search_test.go` covering all three render paths the audit identified (likely: idle / loading / results). For each, force `uikit.SetModeForTest(GlyphASCII)`, call the relevant View, assert no unicode border glyphs.

- [ ] **Step 2: Replace all three call sites**

For each of lines 778, 880, 943, find the `layout.RenderPaneBorder(content, cfg)` call and substitute:

```go
chrome := uikit.OverlayChrome{
    Width:   s.width,
    Height:  s.height,
    Title:   "Search",
    Actions: s.Actions(),
    Theme:   s.theme,
}
return chrome.Render(content)
```

(The `Title` may differ between sites — match what the existing border title was.)

- [ ] **Step 3: Run tests**

```
go test ./internal/ui/panes/ -run TestSearch -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/ui/panes/search.go internal/ui/panes/search_test.go
git commit -m "fix(search): migrate three search-overlay render paths to OverlayChrome"
```

---

### Task 3.6: Migrate infobox.go to PaneChrome

**Files:**
- Modify: `internal/ui/components/infobox.go`

`infobox.go` rolls its own border with hardcoded unicode corners. Replace the entire bordering pass with a `uikit.PaneChrome.Render` call.

- [ ] **Step 1: Read `infobox.go` and identify the border-drawing block**

The audit identified hardcoded glyphs at lines 99, 149, 151, 162, 169, 172, 183 — these are the corner and rule literals.

- [ ] **Step 2: Write the failing test**

Append to `internal/ui/components/infobox_test.go`:

```go
func TestInfoBox_AsciiBorder(t *testing.T) {
    uikit.SetModeForTest(uikit.GlyphASCII)
    defer uikit.SetModeForTest(uikit.GlyphUnicode)

    box := NewInfoBox(InfoBoxConfig{
        Width:  40,
        Title:  "Info",
        Theme:  theme.Load("black"),
    })
    out := stripANSI(box.Render("hello"))
    if strings.ContainsAny(out, "╭╮╰╯─│") {
        t.Errorf("ascii infobox must not contain unicode borders, got: %q", out)
    }
}
```

(Adjust to match the existing `InfoBox` constructor / fields.)

- [ ] **Step 3: Rewrite `Render` to delegate to `uikit.PaneChrome`**

Strip the hand-rolled border code. The new `Render` becomes:

```go
func (b InfoBox) Render(content string) string {
    chrome := uikit.PaneChrome{
        Width:       b.Width,
        Height:      b.Height, // or computed from content
        Title:       b.Title,
        AccentColor: b.AccentColor, // existing field
        Theme:       b.Theme,
    }
    return chrome.Render(content)
}
```

(Read the file first to map existing fields to PaneChrome fields. The goal is "InfoBox is a thin wrapper over PaneChrome.")

- [ ] **Step 4: Run tests, expect pass**

```
go test ./internal/ui/components/ -run TestInfoBox -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/ui/components/infobox.go internal/ui/components/infobox_test.go
git commit -m "refactor(components): InfoBox delegates to uikit.PaneChrome instead of rolling its own border"
```

---

### Task 3.7: Add CI guard restricting RenderPaneBorder direct callers

**Files:**
- Create: `scripts/check-render-pane-border.sh`
- Modify: `Makefile`

After this phase, the only legitimate callers of `layout.RenderPaneBorder` are `uikit/pane_chrome.go`, `uikit/overlay_chrome.go`, `uikit/panel.go`, and `uikit/status_bar.go`. New direct callers from outside `uikit/` should fail CI.

- [ ] **Step 1: Create the script**

Create `scripts/check-render-pane-border.sh`:

```bash
#!/usr/bin/env bash
# Guard: layout.RenderPaneBorder may only be called from internal/uikit/.
# Any other caller bypasses the glyph-fallback contract.
set -euo pipefail

OFFENDERS=$(grep -rn --include="*.go" "layout\.RenderPaneBorder\|RenderPaneBorder(" internal/ \
    | grep -v "internal/uikit/" \
    | grep -v "internal/ui/layout/" \
    | grep -v "_test.go" || true)

if [ -n "$OFFENDERS" ]; then
    echo "ERROR: layout.RenderPaneBorder called outside internal/uikit/ — use uikit.PaneChrome / OverlayChrome instead."
    echo "$OFFENDERS"
    exit 1
fi
echo "OK: no direct RenderPaneBorder callers outside uikit"
```

- [ ] **Step 2: Make it executable**

```bash
chmod +x scripts/check-render-pane-border.sh
```

- [ ] **Step 3: Add a Makefile target**

Append to `Makefile`:

```makefile
.PHONY: check-chrome
check-chrome:
	@scripts/check-render-pane-border.sh
```

- [ ] **Step 4: Run the script — expect pass after Phase 3**

```
make check-chrome
```

- [ ] **Step 5: Commit**

```bash
git add scripts/check-render-pane-border.sh Makefile
git commit -m "$(cat <<'EOF'
chore(ci): guard against direct layout.RenderPaneBorder callers

After Phase 3 the only legitimate callers are the four uikit chrome
primitives. Any new direct caller from outside internal/uikit/ now fails
make check-chrome — the guard catches the regression that allowed every
grid pane border to ship as unicode-only.
EOF
)"
```

---

## Phase 4 — Critical content fixes

### Task 4.1: Create uikit.PlaybackControls primitive

**Files:**
- Create: `internal/uikit/playback_controls.go`
- Create: `internal/uikit/playback_controls_test.go`
- Modify: `docs/TUI-DESIGN-SYSTEM.md` (§3 — add §3.19 PlaybackControls)

- [ ] **Step 1: Write the failing test**

Create `internal/uikit/playback_controls_test.go`:

```go
package uikit

import (
    "strings"
    "testing"

    "github.com/initgrep-apps/spotnik/internal/ui/theme"
)

func TestPlaybackControls_RenderUnicode_Playing(t *testing.T) {
    SetModeForTest(GlyphUnicode)
    defer SetModeForTest(GlyphUnicode)

    c := PlaybackControls{
        Playing:    true,
        Shuffle:    false,
        RepeatMode: RepeatOff,
        Theme:      theme.Load("black"),
    }
    out := stripANSI(c.Render())
    // playing → ⏸ active; queue → ≡; shuffle off → ⇄ inactive; repeat off → ↻ inactive
    for _, want := range []string{"⏸", "≡", "⇄", "↻"} {
        if !strings.Contains(out, want) {
            t.Errorf("missing %q in unicode render: %q", want, out)
        }
    }
}

func TestPlaybackControls_RenderASCII_Playing(t *testing.T) {
    SetModeForTest(GlyphASCII)
    defer SetModeForTest(GlyphUnicode)

    c := PlaybackControls{
        Playing:    true,
        Shuffle:    false,
        RepeatMode: RepeatOff,
        Theme:      theme.Load("black"),
    }
    out := stripANSI(c.Render())
    for _, want := range []string{"||", "Q", "sh", "rp"} {
        if !strings.Contains(out, want) {
            t.Errorf("missing %q in ascii render: %q", want, out)
        }
    }
    for _, banned := range []string{"⏸", "≡", "⇄", "↻", "⏷"} {
        if strings.Contains(out, banned) {
            t.Errorf("ascii render must not contain %q, got: %q", banned, out)
        }
    }
}

func TestPlaybackControls_RepeatModes(t *testing.T) {
    SetModeForTest(GlyphUnicode)
    defer SetModeForTest(GlyphUnicode)

    cases := []struct {
        mode RepeatMode
        want string
    }{
        {RepeatOff, "↻"},
        {RepeatAll, "↻"}, // active style colour, same glyph
        {RepeatOne, "↻¹"},
    }
    for _, c := range cases {
        ctrl := PlaybackControls{
            Playing:    false,
            RepeatMode: c.mode,
            Theme:      theme.Load("black"),
        }
        out := stripANSI(ctrl.Render())
        if !strings.Contains(out, c.want) {
            t.Errorf("repeat=%v: missing %q in %q", c.mode, c.want, out)
        }
    }
}
```

(`stripANSI` is the existing helper used by other uikit tests.)

- [ ] **Step 2: Run tests, expect fail**

```
go test ./internal/uikit/ -run TestPlaybackControls -v
```

- [ ] **Step 3: Create the primitive**

Create `internal/uikit/playback_controls.go`:

```go
package uikit

import (
    "github.com/charmbracelet/lipgloss"
    "github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// RepeatMode is the repeat-state of the playback engine.
type RepeatMode int

const (
    RepeatOff     RepeatMode = iota // ↻ rendered in inactive (Muted) colour
    RepeatAll                       // ↻ rendered in active (PlayingIndicator) colour
    RepeatOne                       // ↻¹ rendered in active colour
)

// PlaybackControls renders the transport-controls strip:
//
//   shuffle  play/pause  queue  repeat
//
// Active icons render in theme.PlayingIndicator(); inactive icons render in
// theme.TextSecondary(). All glyphs route through GlyphFor so the strip
// renders correctly in ascii mode.
type PlaybackControls struct {
    Playing    bool
    Shuffle    bool
    RepeatMode RepeatMode
    Theme      theme.Theme
}

// Render produces the controls row. Stateless beyond the input fields.
func (c PlaybackControls) Render() string {
    m := ActiveMode()
    activeStyle := lipgloss.NewStyle().Foreground(c.Theme.PlayingIndicator())
    inactiveStyle := lipgloss.NewStyle().Foreground(c.Theme.TextSecondary())

    pickStyle := func(active bool) lipgloss.Style {
        if active {
            return activeStyle
        }
        return inactiveStyle
    }

    shuffle := pickStyle(c.Shuffle).Render(GlyphFor(GlyphShuffle, m))

    var playPause string
    if c.Playing {
        playPause = activeStyle.Render(GlyphFor(GlyphPaused, m))
    } else {
        playPause = inactiveStyle.Render(GlyphFor(GlyphPausedPB, m))
    }

    queue := inactiveStyle.Render(GlyphFor(GlyphQueue, m))

    var repeat string
    switch c.RepeatMode {
    case RepeatOne:
        repeat = activeStyle.Render(GlyphFor(GlyphRepeatOne, m))
    case RepeatAll:
        repeat = activeStyle.Render(GlyphFor(GlyphRepeatAll, m))
    default:
        repeat = inactiveStyle.Render(GlyphFor(GlyphRepeatOff, m))
    }

    return shuffle + "  " + playPause + "  " + queue + "  " + repeat
}
```

Note: the existing `controls.go` used `↻` for both `RepeatAll` and `RepeatOff` (different colour, same glyph). To match catalogue intent we now use `GlyphRepeatOff` (`⟳` / `ro`) for the off state, `GlyphRepeatAll` (`↻` / `rp`) for all, `GlyphRepeatOne` (`↻¹` / `rp1`) for one. The visual change is intentional.

- [ ] **Step 4: Run tests, expect pass**

```
go test ./internal/uikit/ -run TestPlaybackControls -v
```

- [ ] **Step 5: Document in TUI-DESIGN-SYSTEM.md**

Append a §3.19 to `docs/TUI-DESIGN-SYSTEM.md` after §3.18 (Spinner):

```markdown
### 3.19 PlaybackControls

**Purpose:** Transport-controls strip — shuffle, play/pause, queue, repeat —
with mode-aware glyphs and active/inactive intent colours.

**Fields:**

```go
type PlaybackControls struct {
    Playing    bool
    Shuffle    bool
    RepeatMode RepeatMode  // RepeatOff | RepeatAll | RepeatOne
    Theme      theme.Theme
}
```

**Rendering (unicode, paused + shuffle off + repeat off):**

```
⇄  ▷  ≡  ⟳
```

**Rendering (ascii, same state):**

```
sh  |>  Q  ro
```

**Roles:** active glyph → `theme.PlayingIndicator()`; inactive → `theme.TextSecondary()`.

**Glyphs:** `⇄`/`sh`, `▷`/`|>`, `⏸`/`||`, `≡`/`Q`, `↻`/`rp`, `↻¹`/`rp1`, `⟳`/`ro`.

**Tests:** all four positions render their intent-correct glyph in both modes;
RepeatMode transitions glyph and colour correctly.
```

- [ ] **Step 6: Commit**

```bash
git add internal/uikit/playback_controls.go internal/uikit/playback_controls_test.go docs/TUI-DESIGN-SYSTEM.md
git commit -m "$(cat <<'EOF'
feat(uikit): add PlaybackControls primitive with ascii fallback

Replaces hand-rolled transport rendering in components/controls.go (next
commit) with a uikit primitive that routes all 7 transport glyphs through
GlyphFor.

Spec: docs/superpowers/specs/2026-04-29-glyph-fallback-audit-design.md §4.2
EOF
)"
```

---

### Task 4.2: Migrate controls.go to PlaybackControls

**Files:**
- Modify: `internal/ui/components/controls.go`
- Modify: `internal/ui/components/controls_test.go`

- [ ] **Step 1: Adapt the existing `controls.go` to wrap the primitive**

Replace the entire body of `internal/ui/components/controls.go` with:

```go
package components

import (
    "github.com/initgrep-apps/spotnik/internal/uikit"
    "github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// Controls is a thin compatibility wrapper around uikit.PlaybackControls.
// Existing callers in internal/ui/panes pass repeatMode as a string ("off",
// "context", "track"); we translate to uikit.RepeatMode here so the call
// sites do not need to change in this commit.
type Controls struct {
    inner uikit.PlaybackControls
}

// NewControls constructs a Controls wrapper.
// repeatMode must be one of "off", "context", "track".
func NewControls(t theme.Theme, isPlaying, shuffleOn bool, repeatMode string) Controls {
    var rm uikit.RepeatMode
    switch repeatMode {
    case "track":
        rm = uikit.RepeatOne
    case "context":
        rm = uikit.RepeatAll
    default:
        rm = uikit.RepeatOff
    }
    return Controls{
        inner: uikit.PlaybackControls{
            Playing:    isPlaying,
            Shuffle:    shuffleOn,
            RepeatMode: rm,
            Theme:      t,
        },
    }
}

// Render delegates to the uikit primitive.
func (c Controls) Render() string { return c.inner.Render() }
```

- [ ] **Step 2: Update existing tests**

Open `internal/ui/components/controls_test.go`. Existing tests likely assert on raw unicode glyphs — keep those for the unicode case but remove any "ascii" assertion that relied on the old hand-rolled implementation. The primary contract is now tested by `playback_controls_test.go`; this file should only cover the string→RepeatMode translation:

```go
func TestNewControls_RepeatModeTranslation(t *testing.T) {
    cases := []struct {
        in  string
        want uikit.RepeatMode
    }{
        {"off", uikit.RepeatOff},
        {"context", uikit.RepeatAll},
        {"track", uikit.RepeatOne},
        {"unknown", uikit.RepeatOff},
    }
    for _, c := range cases {
        got := NewControls(theme.Load("black"), false, false, c.in).inner.RepeatMode
        if got != c.want {
            t.Errorf("NewControls(repeatMode=%q).inner.RepeatMode = %v, want %v", c.in, got, c.want)
        }
    }
}
```

- [ ] **Step 3: Run tests**

```
go test ./internal/ui/components/ -run TestNewControls -v
go test ./internal/uikit/ -run TestPlaybackControls -v
```

- [ ] **Step 4: Run the full app suite to catch any caller assumption**

```
go test ./internal/app/ -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/ui/components/controls.go internal/ui/components/controls_test.go
git commit -m "$(cat <<'EOF'
refactor(controls): delegate to uikit.PlaybackControls

components.Controls becomes a thin compatibility wrapper that translates
string repeat modes ("off"/"context"/"track") to uikit.RepeatMode and
delegates rendering. All 7 transport glyphs now route through GlyphFor.

Spec: docs/superpowers/specs/2026-04-29-glyph-fallback-audit-design.md §3.3
EOF
)"
```

---

### Task 4.3: Move bubbleup registration into uikit.Toast

**Files:**
- Modify: `internal/uikit/toast.go`
- Modify: `internal/ui/components/notifications.go`

The five alert prefixes (`✓ ✗ ◬ → ⧖`) are registered with bubbleup at construction time and never re-resolved. Move the registration into `uikit.Toast` so prefixes resolve via `GlyphFor` when `uikit.Use` is called.

- [ ] **Step 1: Read the existing notifications.go to understand the bubbleup integration**

```
cat internal/ui/components/notifications.go
```
Note the `bubbleup.AlertDefinition` struct shape and how `NewNotifications` is constructed.

- [ ] **Step 2: Move alert-definition construction into uikit.Toast**

In `internal/uikit/toast.go`, add:

```go
// RegisterBubbleupAlerts builds the bubbleup alert definitions for the five
// toast intents. Glyph prefixes are resolved via GlyphFor at call time so
// the result honours ActiveMode().
func RegisterBubbleupAlerts(theme theme.Theme) []bubbleup.AlertDefinition {
    m := ActiveMode()
    return []bubbleup.AlertDefinition{
        {
            Type:   "success",
            Prefix: GlyphFor(GlyphSuccess, m),
            Style:  lipgloss.NewStyle().Foreground(theme.Success()),
        },
        {
            Type:   "error",
            Prefix: GlyphFor(GlyphError, m),
            Style:  lipgloss.NewStyle().Foreground(theme.Error()),
        },
        {
            Type:   "warning",
            Prefix: GlyphFor(GlyphWarning, m),
            Style:  lipgloss.NewStyle().Foreground(theme.Warning()),
        },
        {
            Type:   "info",
            Prefix: GlyphFor(GlyphInfo, m),
            Style:  lipgloss.NewStyle().Foreground(theme.Info()),
        },
        {
            Type:   "ratelimit",
            Prefix: GlyphFor(GlyphRateLimit, m),
            Style:  lipgloss.NewStyle().Foreground(theme.Warning()),
        },
    }
}
```

(Adjust `bubbleup.AlertDefinition` field names to match the actual library — read the existing notifications.go first to copy the shape exactly.)

- [ ] **Step 3: Replace notifications.go body with a call to the new helper**

```go
package components

import (
    "github.com/koki-develop/bubbleup"

    "github.com/initgrep-apps/spotnik/internal/uikit"
    "github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// NewNotifications wires up the bubbleup alert manager with mode-aware
// alert definitions sourced from uikit.RegisterBubbleupAlerts.
func NewNotifications(t theme.Theme) *bubbleup.Model {
    defs := uikit.RegisterBubbleupAlerts(t)
    m := bubbleup.New(defs...)
    return m
}
```

- [ ] **Step 4: Add the test**

Create `internal/uikit/toast_test.go` test:

```go
func TestRegisterBubbleupAlerts_AsciiPrefixes(t *testing.T) {
    SetModeForTest(GlyphASCII)
    defer SetModeForTest(GlyphUnicode)

    defs := RegisterBubbleupAlerts(theme.Load("black"))
    wantPrefixes := map[string]string{
        "success":   "+",
        "error":     "x",
        "warning":   "!",
        "info":      ">",
        "ratelimit": "~",
    }
    for _, d := range defs {
        want, ok := wantPrefixes[d.Type]
        if !ok {
            continue
        }
        if d.Prefix != want {
            t.Errorf("alert %q ascii prefix = %q, want %q", d.Type, d.Prefix, want)
        }
    }
}
```

- [ ] **Step 5: Run tests, expect pass**

```
go test ./internal/uikit/ -run TestRegisterBubbleupAlerts -v
go test ./internal/ui/components/ -run TestNotifications -v
```

- [ ] **Step 6: Commit**

```bash
git add internal/uikit/toast.go internal/uikit/toast_test.go internal/ui/components/notifications.go
git commit -m "$(cat <<'EOF'
fix(toast): resolve bubbleup alert prefixes via GlyphFor at registration

Prefixes were hardcoded in notifications.go, so ascii mode rendered
unicode toast glyphs even when uikit was set to GlyphASCII. Moving the
registration into uikit.RegisterBubbleupAlerts lets the prefixes resolve
through the catalogue when ActiveMode() is read.

Spec: docs/superpowers/specs/2026-04-29-glyph-fallback-audit-design.md §3.3
EOF
)"
```

---

### Task 4.4: Fix nowplaying.go Title playback glyphs

**Files:**
- Modify: `internal/ui/panes/nowplaying.go` (lines 85, 87, 91)

- [ ] **Step 1: Write the failing test**

Append to `internal/ui/panes/nowplaying_test.go`:

```go
func TestNowPlaying_AsciiTitle(t *testing.T) {
    uikit.SetModeForTest(uikit.GlyphASCII)
    defer uikit.SetModeForTest(uikit.GlyphUnicode)

    p := NewNowPlayingPane(theme.Load("black")) // existing constructor
    p.SetPlaying(true)
    title := stripANSI(p.Title())
    if strings.ContainsAny(title, "▶⏸─") {
        t.Errorf("ascii Title must not contain unicode playback glyphs, got: %q", title)
    }
    if !strings.Contains(title, "||") {
        t.Errorf("ascii title (playing=true) should contain '||' (paused glyph), got: %q", title)
    }
}
```

- [ ] **Step 2: Replace the literals at lines 85, 87, 91**

```go
m := uikit.ActiveMode()
var stateGlyph string
if p.isPlaying {
    stateGlyph = uikit.GlyphFor(uikit.GlyphPaused, m) // playing → show pause icon
} else {
    stateGlyph = uikit.GlyphFor(uikit.GlyphPlaying, m) // paused → show play icon
}
sep := uikit.GlyphFor(uikit.GlyphHRule, m)
return fmt.Sprintf("%s %s %s %s", stateGlyph, p.trackName, sep, p.artistName)
```

(Adjust to match the existing Title format. Read lines 80–95 first.)

- [ ] **Step 3: Run tests**

```
go test ./internal/ui/panes/ -run TestNowPlaying -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/ui/panes/nowplaying.go internal/ui/panes/nowplaying_test.go
git commit -m "fix(nowplaying): route Title playback glyphs through GlyphFor"
```

---

### Task 4.5: Create viz.AsciiBarsRenderer

**Files:**
- Create: `internal/ui/components/viz/ascii_bars.go`
- Create: `internal/ui/components/viz/ascii_bars_test.go`

A new pattern that draws columns using `#` (filled), `=` (half), `.` (empty), at 4-level vertical resolution.

- [ ] **Step 1: Read the existing `block.go` and `pattern.go` for the Pattern interface**

```
cat internal/ui/components/viz/pattern.go internal/ui/components/viz/block.go
```
Note the `Pattern` interface (or struct with method receivers) the engine expects.

- [ ] **Step 2: Write the failing test**

Create `internal/ui/components/viz/ascii_bars_test.go`:

```go
package viz

import (
    "strings"
    "testing"

    "github.com/initgrep-apps/spotnik/internal/uikit"
    "github.com/initgrep-apps/spotnik/internal/ui/theme"
)

func TestAsciiBars_AllAscii(t *testing.T) {
    uikit.SetModeForTest(uikit.GlyphASCII)
    defer uikit.SetModeForTest(uikit.GlyphUnicode)

    r := NewAsciiBarsRenderer()
    th := theme.Load("black")
    cols := []float64{0.0, 0.25, 0.5, 0.75, 1.0}
    out := r.Render(cols, 4, th)
    // No braille / block characters in ascii output
    for _, banned := range []rune{'█', '▉', '▊', '▋', '▌', '▍', '▎', '▏', '⠀'} {
        if strings.ContainsRune(out, banned) {
            t.Errorf("ascii bars must not contain %q, got: %q", banned, out)
        }
    }
    // Output uses #, =, ., space
    for _, c := range out {
        if c != '#' && c != '=' && c != '.' && c != ' ' && c != '\n' && c != ' ' {
            // Allow ANSI escapes — strip them first if needed.
        }
    }
}

func TestAsciiBars_MaxLevels(t *testing.T) {
    r := NewAsciiBarsRenderer()
    if r.MaxHeight() != 4 {
        t.Errorf("AsciiBarsRenderer.MaxHeight() = %d, want 4", r.MaxHeight())
    }
}
```

- [ ] **Step 3: Implement the renderer**

Create `internal/ui/components/viz/ascii_bars.go`:

```go
package viz

import (
    "strings"

    "github.com/charmbracelet/lipgloss"
    "github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// AsciiBarsRenderer draws bar columns using the # = . set at 4-level vertical
// resolution. Used when uikit.ActiveMode() == GlyphASCII so the visualizer
// stays present (degraded) rather than rendering mojibake.
type AsciiBarsRenderer struct{}

// NewAsciiBarsRenderer constructs an AsciiBarsRenderer.
func NewAsciiBarsRenderer() *AsciiBarsRenderer {
    return &AsciiBarsRenderer{}
}

// MaxHeight reports the maximum vertical resolution this renderer supports.
// 4 levels: empty, low (.), mid (=), full (#).
func (r *AsciiBarsRenderer) MaxHeight() int { return 4 }

// Render produces a multi-line ASCII bar visualization. cols is the column
// heights in [0, 1]; height is the row count.
func (r *AsciiBarsRenderer) Render(cols []float64, height int, th theme.Theme) string {
    if height < 1 {
        height = 1
    }
    style := lipgloss.NewStyle().Foreground(th.Gradient1())

    rows := make([]string, height)
    for row := 0; row < height; row++ {
        // Threshold for this row (top row = highest threshold).
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

(Match the actual `Pattern`/`Renderer` interface in the existing package — the contract may use a different method signature like `Generate(width, height int) Frame`. Read `pattern.go` first and adapt.)

- [ ] **Step 4: Run tests, expect pass**

```
go test ./internal/ui/components/viz/ -run TestAsciiBars -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/ui/components/viz/ascii_bars.go internal/ui/components/viz/ascii_bars_test.go
git commit -m "$(cat <<'EOF'
feat(viz): add AsciiBarsRenderer for ascii-mode visualizer fallback

A 4-level # = . renderer that keeps the visualizer present (at reduced
resolution) when ui.glyphs is "ascii", instead of rendering mojibake from
braille codepoints. Engine integration ships in the next commit.

Spec: docs/superpowers/specs/2026-04-29-glyph-fallback-audit-design.md §4.3
EOF
)"
```

---

### Task 4.6: Engine selects renderer by ActiveMode

**Files:**
- Modify: `internal/ui/components/viz/engine.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/ui/components/viz/engine_test.go`:

```go
func TestEngine_SelectsAsciiRendererInAsciiMode(t *testing.T) {
    uikit.SetModeForTest(uikit.GlyphASCII)
    defer uikit.SetModeForTest(uikit.GlyphUnicode)

    e := NewEngine(theme.Load("black"))
    e.SetSize(20, 4)
    e.SetPlaying(true)
    frame := e.CurrentFrame()
    out := stripANSI(frame.String()) // or whatever the existing String/Render entrypoint is

    // ASCII mode must produce # = . set, not braille/blocks
    for _, banned := range []rune{'█', '▉', '▊', '▋', '⠀', '⣀'} {
        if strings.ContainsRune(out, banned) {
            t.Errorf("ascii engine output must not contain %q, got: %q", banned, out)
        }
    }
}
```

- [ ] **Step 2: Update engine.go to select by mode**

In `internal/ui/components/viz/engine.go`, find the renderer-selection point. Replace whatever currently chooses braille vs block with:

```go
import (
    // ... existing ...
    "github.com/initgrep-apps/spotnik/internal/uikit"
)

func (e *Engine) selectRenderer() Renderer {
    if uikit.ActiveMode() == uikit.GlyphASCII {
        return NewAsciiBarsRenderer()
    }
    // existing unicode selection (braille vs block via config)
    return e.unicodeRenderer()
}
```

(Adjust to match the existing rendering pipeline. The integration shape depends on whether `Engine` already abstracts renderers or computes frames inline.)

- [ ] **Step 3: Run viz tests, expect pass**

```
go test ./internal/ui/components/viz/ -v
```

- [ ] **Step 4: Run app/nowplaying tests for visual regressions**

```
go test ./internal/ui/panes/ -run TestNowPlaying -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/ui/components/viz/engine.go internal/ui/components/viz/engine_test.go
git commit -m "$(cat <<'EOF'
fix(viz): select AsciiBarsRenderer when uikit.ActiveMode is GlyphASCII

Closes the visualizer's fallback gap. Unicode terminals continue to use
braille / block renderers as before; ascii mode degrades to # = . at 4
vertical levels rather than rendering invisible braille codepoints.

Spec: docs/superpowers/specs/2026-04-29-glyph-fallback-audit-design.md §4.3
EOF
)"
```

---

## Phase 5 — Pane content cleanup

Each task in this phase swaps inline glyph leaks for `GlyphFor` calls. The pattern is mechanical: locate the literal, route through the catalogue, add a small ASCII-mode test.

### Task 5.1: devices.go — status glyphs, device icons, empty state

**Files:**
- Modify: `internal/ui/panes/devices.go`
- Modify: `internal/ui/panes/devices_test.go`

- [ ] **Step 1: Replace status glyphs (lines 209, 212, 219, 222)**

Find the four `◉` / `○` literals and substitute `uikit.StatusGlyph`:

```go
m := uikit.ActiveMode()
activeIndicator := uikit.StatusGlyph{
    Role:  RoleSuccess, // or whatever the active row role is
    Text:  "",
    Theme: o.theme,
}.Render()
inactiveIndicator := uikit.StatusGlyph{
    Role:  RoleMuted,
    Text:  "",
    Theme: o.theme,
}.Render()
```

(Read the surrounding code first — `StatusGlyph` may not be the right primitive; the simpler fix is direct `GlyphFor(GlyphActive, m)` / `GlyphFor(GlyphAvailable, m)` lookups inline. Choose whichever keeps the call site smallest.)

- [ ] **Step 2: Replace `deviceTypeIcon` (lines 258–264)**

The existing function returns `⊡⊞⊟⊠` based on device-type strings. Replace with:

```go
func deviceTypeIcon(deviceType string) string {
    m := uikit.ActiveMode()
    switch deviceType {
    case "Computer":
        return uikit.GlyphFor(uikit.GlyphDeviceComputer, m)
    case "Smartphone":
        return uikit.GlyphFor(uikit.GlyphDevicePhone, m)
    case "Speaker":
        return uikit.GlyphFor(uikit.GlyphDeviceSpeaker, m)
    case "TV":
        return uikit.GlyphFor(uikit.GlyphDeviceTV, m)
    default:
        return uikit.GlyphFor(uikit.GlyphInactive, m)
    }
}
```

- [ ] **Step 3: Replace empty-message rendering (lines 145–147)**

Find the "No devices found" custom rendering. Replace with `uikit.EmptyState`:

```go
if len(o.devices) == 0 {
    return uikit.EmptyState{
        Text:   "No devices found",
        Hint:   "Open Spotify on a device to see it here",
        Width:  o.width,
        Height: o.height - 4, // minus chrome
        Theme:  o.theme,
    }.Render()
}
```

- [ ] **Step 4: Add ASCII-mode tests**

Append to `devices_test.go`:

```go
func TestDevicesOverlay_AsciiContent(t *testing.T) {
    uikit.SetModeForTest(uikit.GlyphASCII)
    defer uikit.SetModeForTest(uikit.GlyphUnicode)

    o := NewDevicesOverlay(theme.Load("black"))
    o.SetDevices([]Device{
        {Name: "iPhone 14", Type: "Smartphone", Active: true},
        {Name: "MacBook",   Type: "Computer",   Active: false},
    })
    out := stripANSI(o.View())
    for _, banned := range []string{"◉", "○", "⊡", "⊞", "⊟", "⊠"} {
        if strings.Contains(out, banned) {
            t.Errorf("ascii output must not contain %q, got: %q", banned, out)
        }
    }
    if !strings.Contains(out, "(*)") || !strings.Contains(out, "[p]") {
        t.Errorf("ascii output should contain (*) and [p], got: %q", out)
    }
}
```

- [ ] **Step 5: Run tests**

```
go test ./internal/ui/panes/ -run TestDevicesOverlay -v
```

- [ ] **Step 6: Commit**

```bash
git add internal/ui/panes/devices.go internal/ui/panes/devices_test.go
git commit -m "$(cat <<'EOF'
fix(devices): route status glyphs, device icons, and empty state through uikit

- Active/available indicators now use uikit.GlyphFor(GlyphActive/GlyphAvailable)
- Device-type icons use the new GlyphDeviceComputer/Phone/Speaker/TV roles
- Empty state migrates to uikit.EmptyState

Spec: docs/superpowers/specs/2026-04-29-glyph-fallback-audit-design.md §3.2
EOF
)"
```

---

### Task 5.2: search.go bubbles/spinner → uikit.Spinner

**Files:**
- Modify: `internal/ui/panes/search.go` (line 214 region)

- [ ] **Step 1: Replace `bubbles/spinner.Model` with `uikit.Spinner`**

Find the `spinner.Model` declaration (line 214). Replace import:

```go
// Remove
"github.com/charmbracelet/bubbles/spinner"

// Replace with (already imported elsewhere in the file)
"github.com/initgrep-apps/spotnik/internal/uikit"
```

Replace the field and its initialisation:

```go
// Before
sp spinner.Model

// After
sp *uikit.Spinner
```

In the constructor:

```go
s.sp = uikit.NewSpinner("Loading...", theme)
```

In the Update path, replace `spinner.Model.Update` with `uikit.Spinner.Update`. In View, replace `s.sp.View()` with `s.sp.View()` (same call shape — uikit.Spinner mirrors the bubbles API where possible).

- [ ] **Step 2: Run search tests**

```
go test ./internal/ui/panes/ -run TestSearch -v
```

- [ ] **Step 3: Commit**

```bash
git add internal/ui/panes/search.go
git commit -m "fix(search): swap bubbles/spinner for uikit.Spinner with ascii fallback"
```

---

### Task 5.3: search_delegate.go categorySymbol via GlyphFor

**Files:**
- Modify: `internal/ui/panes/search_delegate.go`
- Modify: `internal/ui/panes/search_delegate_test.go`

- [ ] **Step 1: Rewrite `categorySymbol` (lines 62–77)**

```go
func categorySymbol(category string) string {
    m := uikit.ActiveMode()
    switch category {
    case "track":
        return uikit.GlyphFor(uikit.GlyphMusicNote, m)
    case "artist":
        return uikit.GlyphFor(uikit.GlyphPinned, m)
    case "album":
        return uikit.GlyphFor(uikit.GlyphInactive, m)
    case "playlist":
        return uikit.GlyphFor(uikit.GlyphPlaylist, m)
    default:
        return uikit.GlyphFor(uikit.GlyphSeparator, m)
    }
}
```

- [ ] **Step 2: Replace separators at lines 95 and 335**

Line 95 (left border for selection): replace literal `│` with `uikit.GlyphFor(uikit.GlyphVRule, uikit.ActiveMode())`.

Line 335 (row separator): replace literal `·` with `uikit.GlyphFor(uikit.GlyphSeparator, uikit.ActiveMode())`.

- [ ] **Step 3: Add ASCII test**

```go
func TestSearchDelegate_AsciiCategorySymbols(t *testing.T) {
    uikit.SetModeForTest(uikit.GlyphASCII)
    defer uikit.SetModeForTest(uikit.GlyphUnicode)

    cases := map[string]string{
        "track":    "*",
        "artist":   "*",
        "album":    "( )",
        "playlist": "[=]",
    }
    for in, want := range cases {
        if got := categorySymbol(in); got != want {
            t.Errorf("categorySymbol(%q) ascii = %q, want %q", in, got, want)
        }
    }
}
```

- [ ] **Step 4: Run tests, commit**

```
go test ./internal/ui/panes/ -run TestSearchDelegate -v
```

```bash
git add internal/ui/panes/search_delegate.go internal/ui/panes/search_delegate_test.go
git commit -m "fix(search-delegate): route categorySymbol and separators through GlyphFor"
```

---

### Task 5.4: Empty-state migrations (recentlyplayed, nowplaying)

**Files:**
- Modify: `internal/ui/panes/recentlyplayed_pane.go` (line 134)
- Modify: `internal/ui/panes/nowplaying.go` (lines 314–319)

For each file, replace the custom empty-message string with a `uikit.EmptyState` instance.

- [ ] **Step 1: recentlyplayed_pane.go**

Replace the existing custom render with:

```go
if len(p.tracks) == 0 {
    return uikit.EmptyState{
        Text:   "No recently played tracks",
        Hint:   "Listen to something to populate this list",
        Width:  p.width,
        Height: p.height,
        Theme:  p.theme,
    }.Render()
}
```

- [ ] **Step 2: nowplaying.go**

Replace lines 314–319 (the "Nothing playing" custom render):

```go
if !p.hasTrack() {
    return uikit.EmptyState{
        Text:   "Nothing playing",
        Hint:   "Press / to search and queue a track",
        Width:  p.width,
        Height: p.height,
        Theme:  p.theme,
    }.Render()
}
```

- [ ] **Step 3: Run tests for both panes**

```
go test ./internal/ui/panes/ -run "TestRecentlyPlayed|TestNowPlaying" -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/ui/panes/recentlyplayed_pane.go internal/ui/panes/nowplaying.go
git commit -m "fix(panes): migrate recentlyplayed and nowplaying empty states to uikit.EmptyState"
```

---

### Task 5.5: networklog priorities, profile/help_overlay/gradient/render inline glyphs

**Files:**
- Modify: `internal/ui/panes/networklog_pane.go` (lines 275, 277)
- Modify: `internal/ui/panes/profile.go` (line 228)
- Modify: `internal/ui/panes/help_overlay.go` (line 140)
- Modify: `internal/ui/components/gradient.go` (lines 197, 200)
- Modify: `internal/app/render.go` (lines 132, 320–322, 526, 551)

All mechanical literal-→-`GlyphFor` swaps. One commit covers all five files because the changes are similar and small.

- [ ] **Step 1: networklog_pane.go**

Replace `◷` and `⚡` at lines 275 and 277 with:

```go
m := uikit.ActiveMode()
deadlineGlyph := uikit.GlyphFor(uikit.GlyphDeadline, m)
runningGlyph := uikit.GlyphFor(uikit.GlyphRunning, m)
```

- [ ] **Step 2: profile.go:228**

Replace raw `…` in the truncation helper:

```go
func truncate(s string, max int) string {
    if len(s) <= max {
        return s
    }
    ell := uikit.GlyphFor(uikit.GlyphEllipsis, uikit.ActiveMode())
    return s[:max-len(ell)] + ell
}
```

- [ ] **Step 3: help_overlay.go:140**

Replace `│` divider:

```go
divider := uikit.GlyphFor(uikit.GlyphVRule, uikit.ActiveMode())
```

- [ ] **Step 4: gradient.go:197 & 200**

Replace `♪` icon literals:

```go
note := uikit.GlyphFor(uikit.GlyphMusicNote, uikit.ActiveMode())
```

- [ ] **Step 5: render.go:132 (banner), 320–322 (bullets), 526/551 (ellipsis)**

```go
// Line 132 — banner
m := uikit.ActiveMode()
banner := uikit.GlyphFor(uikit.GlyphMusicNote, m) + "  spotnik"

// Lines 320–322 — bullets
bullet := uikit.GlyphFor(uikit.GlyphBullet, m)
// then use `bullet` in each of the three list items

// Lines 526, 551 — ellipsis truncation
ell := uikit.GlyphFor(uikit.GlyphEllipsis, m)
```

- [ ] **Step 6: Add a single ASCII smoke test for these inline swaps**

In `internal/app/render_test.go`:

```go
func TestRender_AsciiInlineGlyphs(t *testing.T) {
    uikit.SetModeForTest(uikit.GlyphASCII)
    defer uikit.SetModeForTest(uikit.GlyphUnicode)

    a := newTestApp(t)
    out := stripANSI(a.renderAll()) // or whatever the top-level render entry is
    for _, banned := range []string{"♪", "•", "…"} {
        if strings.Contains(out, banned) {
            t.Errorf("ascii output must not contain %q, got snippet around the leak", banned)
        }
    }
}
```

- [ ] **Step 7: Run tests**

```
go test ./internal/app/ ./internal/ui/panes/ ./internal/ui/components/ -v
```

- [ ] **Step 8: Commit**

```bash
git add internal/ui/panes/networklog_pane.go internal/ui/panes/profile.go internal/ui/panes/help_overlay.go internal/ui/components/gradient.go internal/app/render.go internal/app/render_test.go
git commit -m "$(cat <<'EOF'
fix(ui): route remaining inline glyph leaks through GlyphFor

Mechanical sweep covering:
- networklog priority indicators (◷ ⚡)
- profile name truncation ellipsis
- help-overlay divider
- gradient volume-bar music note
- app banner, onboarding bullets, name-truncation ellipsis

After this commit the only remaining unicode-glyph literals are inside
the catalogue itself (uikit/glyph.go) and the canonical doc files.

Spec: docs/superpowers/specs/2026-04-29-glyph-fallback-audit-design.md §3.2 / §3.3
EOF
)"
```

---

### Task 5.6: gateway_health_pane dot-bar → uikit.ProgressBar

**Files:**
- Modify: `internal/ui/panes/gateway_health_pane.go` (lines 169–184)

- [ ] **Step 1: Replace the custom `renderDotBar` helper**

Locate the function (lines 169–184) and replace its body with a `uikit.ProgressBar` call:

```go
func renderDotBar(progress float64, width int, th theme.Theme) string {
    return uikit.ProgressBar{
        Width:    width,
        Progress: progress,
        Theme:    th,
    }.Render()
}
```

If the existing dot-bar took additional parameters (e.g. accent colour), match the existing public signature; the goal is "swap the implementation, keep the call sites stable."

- [ ] **Step 2: Run the gateway tests**

```
go test ./internal/ui/panes/ -run TestGatewayHealth -v
```

- [ ] **Step 3: Commit**

```bash
git add internal/ui/panes/gateway_health_pane.go
git commit -m "refactor(gateway-health): use uikit.ProgressBar for capacity bar"
```

---

### Task 5.7: table.go playingSymbol lazy-resolve, viz/block.go GlyphBarFull

**Files:**
- Modify: `internal/ui/components/table.go` (line 13)
- Modify: `internal/ui/components/viz/block.go` (line 45)

- [ ] **Step 1: table.go — lazy-resolve `playingSymbol`**

Replace the package-level constant:

```go
// Before
const playingSymbol = "▶"

// After — function so it resolves at render time
func playingSymbol() string {
    return uikit.GlyphFor(uikit.GlyphPlaying, uikit.ActiveMode())
}
```

Update every reference from `playingSymbol` to `playingSymbol()`.

- [ ] **Step 2: viz/block.go — replace `█`**

```go
// Line 45
fill := uikit.GlyphFor(uikit.GlyphBarFull, uikit.ActiveMode())
```

- [ ] **Step 3: Run tests**

```
go test ./internal/ui/components/ ./internal/ui/components/viz/ -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/ui/components/table.go internal/ui/components/viz/block.go
git commit -m "fix(components): lazy-resolve playing symbol and viz block fill via GlyphFor"
```

---

## Phase 6 — Test parity & smoke

### Task 6.1: cliout ASCII test sweep

**Files:**
- Modify: `internal/cliout/*_test.go`

Most cliout tests already exercise both modes after Task 2.6, but ensure parity:

- [ ] **Step 1: Identify any cliout test that asserts a unicode glyph without a corresponding ASCII assertion**

```
grep -l "◉\|◎\|✓\|✗\|◬\|◌\|→" internal/cliout/*_test.go
```

- [ ] **Step 2: For each match, add a sibling ASCII test**

Pattern (parameterised via the catalogue):

```go
func Test<Existing>_BothModes(t *testing.T) {
    for _, mode := range []uikit.GlyphMode{uikit.GlyphUnicode, uikit.GlyphASCII} {
        t.Run(modeName(mode), func(t *testing.T) {
            uikit.SetModeForTest(mode)
            defer uikit.SetModeForTest(uikit.GlyphUnicode)
            // ... existing assertions, but use uikit.GlyphFor for the expected value
        })
    }
}
```

- [ ] **Step 3: Run full cliout suite**

```
go test ./internal/cliout/ -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/cliout/
git commit -m "test(cliout): close ASCII test-coverage gaps for every glyph-bearing message type"
```

---

### Task 6.2: CI matrix and grep guards

**Files:**
- Create: `scripts/check-banned-glyphs.sh`
- Create: `scripts/check-catalogue-leaks.sh`
- Modify: `Makefile`
- Modify: `.github/workflows/ci.yml` (or whichever CI workflow file exists)

- [ ] **Step 1: Create banned-glyph guard**

`scripts/check-banned-glyphs.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail
BANNED=( "⚠" "ᐅ" "┌" "┐" "└" "┘" "╔" "╗" "╚" "╝" "✅" "❌" "❗" )
for g in "${BANNED[@]}"; do
    if grep -rn --include="*.go" "$g" internal/ cmd/ 2>/dev/null; then
        echo "ERROR: banned glyph '$g' present in source"
        exit 1
    fi
done
echo "OK: no banned glyphs"
```

`chmod +x scripts/check-banned-glyphs.sh`.

- [ ] **Step 2: Create catalogue-leak guard**

`scripts/check-catalogue-leaks.sh`:

```bash
#!/usr/bin/env bash
# Catalogue characters may only appear in:
#   internal/uikit/glyph.go
#   docs/TUI-DESIGN-SYSTEM.md
#   docs/CLI-OUTPUT.md
# Any other source file containing them is a leak.
set -euo pipefail

# Subset of the catalogue most likely to leak inline. Extend as gaps surface.
CHARS=( "╭" "╮" "╰" "╯" "✓" "✗" "◬" "→" "⧖" "◉" "◎" "○" "●" "◌" "▶" "▷" "⏸" "≡" "↻" "⇄" "♪" "▤" "█" "▒" "•" "…" )

LEAKS=""
for c in "${CHARS[@]}"; do
    found=$(grep -rn --include="*.go" "$c" internal/ cmd/ 2>/dev/null \
        | grep -v "internal/uikit/glyph.go" \
        | grep -v "_test.go" || true)
    if [ -n "$found" ]; then
        LEAKS="$LEAKS\n$found"
    fi
done

if [ -n "$LEAKS" ]; then
    echo "ERROR: catalogue characters leaked outside internal/uikit/glyph.go:"
    printf "%b\n" "$LEAKS"
    exit 1
fi
echo "OK: no catalogue leaks"
```

`chmod +x scripts/check-catalogue-leaks.sh`.

- [ ] **Step 3: Add `make check-glyphs`**

Append to `Makefile`:

```makefile
.PHONY: check-glyphs
check-glyphs:
	@scripts/check-banned-glyphs.sh
	@scripts/check-catalogue-leaks.sh
	@scripts/check-render-pane-border.sh
```

- [ ] **Step 4: Run locally — expect pass**

```
make check-glyphs
```

If it fails, audit the offender. Either it's a legitimate use that needs to be moved into a primitive, or a missed Phase 5 swap.

- [ ] **Step 5: Wire CI**

Open the existing CI workflow (`.github/workflows/ci.yml` or similar). Add a step:

```yaml
      - name: Glyph fallback guards
        run: make check-glyphs
```

Add a matrix entry that runs the test suite with `LANG=C`:

```yaml
strategy:
  matrix:
    locale: [en_US.UTF-8, C]
steps:
  - run: LANG=${{ matrix.locale }} make test
```

(Adapt to the actual workflow shape.)

- [ ] **Step 6: Commit**

```bash
git add scripts/check-banned-glyphs.sh scripts/check-catalogue-leaks.sh Makefile .github/workflows/ci.yml
git commit -m "$(cat <<'EOF'
chore(ci): add banned-glyph and catalogue-leak guards plus LANG=C matrix

- check-banned-glyphs.sh fails on any of the 13 banned glyphs
- check-catalogue-leaks.sh fails when catalogue characters appear outside
  internal/uikit/glyph.go and the canonical docs
- CI runs the full test suite under both LANG=en_US.UTF-8 and LANG=C so
  the ascii fallback path is covered by automation, not just spot-checks

Spec: docs/superpowers/specs/2026-04-29-glyph-fallback-audit-design.md §7
EOF
)"
```

---

### Task 6.3: Manual smoke test under LANG=C

**Files:** none (operational task)

The acceptance gate. Confirms the code paths produce a usable UI on a non-UTF-8 terminal.

- [ ] **Step 1: Build a binary**

```
make build
```

- [ ] **Step 2: Set ascii config**

Edit `~/.config/spotnik/config.toml` (or wherever the user's config lives) and set:

```toml
[ui]
glyphs = "ascii"
```

- [ ] **Step 3: Run with LANG=C**

```
LANG=C ./bin/spotnik
```

- [ ] **Step 4: Walk through every surface**

For each of the following, confirm rendering: no mojibake, no missing borders, no broken width.
- All 10 grid panes on Page A
- All Page B panes (gateway, polling, networklog, etc.)
- Each overlay: devices, help, search, profile, themes
- Splash screen
- Onboarding flow (start with no token)
- Toast notifications: trigger one of each intent (success, error, warning, info, ratelimit)
- Visualizer: confirm `# = .` columns appear in nowplaying

- [ ] **Step 5: Run the same with `LANG=en_US.UTF-8` and `glyphs = "auto"`**

Confirm the unicode UI is unchanged from before the migration. No regressions.

- [ ] **Step 6: Document results**

Add a checklist to the PR description listing each surface tested and the result.

---

## Self-Review Checklist

Before declaring this plan complete, the implementer should confirm:

- [ ] Every spec section in `docs/superpowers/specs/2026-04-29-glyph-fallback-audit-design.md` has at least one task that addresses it.
- [ ] `make check-glyphs` passes.
- [ ] `LANG=C make test` passes.
- [ ] `LANG=en_US.UTF-8 make test` passes.
- [ ] Manual smoke test (Task 6.3) was completed against a real `LANG=C` shell, not a transpiled assertion.
- [ ] All commits follow Conventional Commits format.
- [ ] No new direct callers of `layout.RenderPaneBorder` outside `internal/uikit/`.
- [ ] `cliout` imports `uikit` and contains no hardcoded glyph literals outside test fixtures.
- [ ] `docs/TUI-DESIGN-SYSTEM.md` §3 documents the new `PlaybackControls` primitive and §4 includes every new role.
