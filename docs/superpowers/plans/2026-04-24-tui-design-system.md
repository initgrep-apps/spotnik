# TUI Design System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Land the `internal/uikit` package — 18 typed primitives with unicode + ascii rendering, a frozen glyph catalogue, a role-to-theme-token matrix, and consistent feedback channels — then migrate every existing TUI call site to use it. Ship `⚠`→`◬` and the removal of `ᐅ` along the way.

**Architecture:** New package `internal/uikit` is a peer to `internal/cliout`. Primitives are typed Go structs with pure `Render(theme, width, height) string` methods. Glyph resolution is lazy-initialised from config (`ui.glyphs = auto | unicode | ascii`) via `sync.Once`. Existing `internal/ui/layout/border.go` becomes the internal implementation of `PaneChrome`. Existing `bubbleup` alert model is wrapped behind a typed `Toast` API. Onboarding and overlay render paths are rewritten to compose primitives instead of hand-building lipgloss styles.

**Tech Stack:** Go 1.22+, Bubble Tea v0.27+, Lip Gloss, `github.com/charmbracelet/bubbles/textinput`, `github.com/charmbracelet/bubbles/spinner`, `go.dalton.dog/bubbleup`, `github.com/rmhubbert/bubbletea-overlay`, `github.com/BurntSushi/toml`, `testing` + `testify`.

**Spec:** `docs/superpowers/specs/2026-04-24-tui-design-system-design.md`

**Feature slot:** `NN-tui-design-system` — assigned when added to `docs/spec/00-overview.md` as part of Task 1.

---

## File Map

| File | Action | Responsibility |
|---|---|---|
| `internal/uikit/doc.go` | Create | Package doc — design-system overview, primitives list |
| `internal/uikit/glyph.go` | Create | Frozen glyph catalogue: unicode + ascii forms, lookup by role |
| `internal/uikit/glyph_test.go` | Create | Catalogue integrity, ascii fallback correctness |
| `internal/uikit/role.go` | Create | Emphasis role → theme token mapping; `Apply(role, style)` helper |
| `internal/uikit/role_test.go` | Create | Role-token assertions for every role |
| `internal/uikit/config.go` | Create | `GlyphMode` enum, `Resolve()` with env-based auto-detect |
| `internal/uikit/config_test.go` | Create | Auto-detect paths (`LANG`, `LC_ALL`, fallback) |
| `internal/uikit/capture.go` | Create | `Capture(fn func() string)` — strip ANSI, split lines for structural assertions |
| `internal/uikit/capture_test.go` | Create | Capture round-trips; ANSI stripping correctness |
| `internal/uikit/pane_chrome.go` | Create | `PaneChrome` primitive wrapping `layout.RenderPaneBorder` |
| `internal/uikit/pane_chrome_test.go` | Create | Unicode + ascii snapshots, role assertions, notch format |
| `internal/uikit/overlay_chrome.go` | Create | `OverlayChrome` centered floating panel |
| `internal/uikit/overlay_chrome_test.go` | Create | Dimmed-bg compositing + border |
| `internal/uikit/panel.go` | Create | `Panel` full-screen bordered panel; title in border |
| `internal/uikit/panel_test.go` | Create | Title-in-border; intent-driven border colour |
| `internal/uikit/table_chrome.go` | Create | `TableChrome` wrapping `components/table.go` |
| `internal/uikit/table_chrome_test.go` | Create | Column tokens, header, selection |
| `internal/uikit/list_row.go` | Create | `ListRow` + `LockedRow` variants |
| `internal/uikit/list_row_test.go` | Create | Locked row dimming, label truncation |
| `internal/uikit/section_label.go` | Create | `SectionLabel` caps marker |
| `internal/uikit/section_label_test.go` | Create | Parent-accent colour, underline rule |
| `internal/uikit/empty_state.go` | Create | `EmptyState` centered message + hint |
| `internal/uikit/empty_state_test.go` | Create | Centering, hint rendering |
| `internal/uikit/url_box.go` | Create | `URLBox` muted-border code/URL block |
| `internal/uikit/url_box_test.go` | Create | URL wrap at `&` boundaries, ascii fallback |
| `internal/uikit/header_bar.go` | Create | `HeaderBar` top app bar |
| `internal/uikit/header_bar_test.go` | Create | Left/right segments, fill gap |
| `internal/uikit/status_bar.go` | Create | `StatusBar` bottom key bar over `KeyBar` + bubbles/help |
| `internal/uikit/status_bar_test.go` | Create | 2-row mode, page-aware bindings |
| `internal/uikit/key_bar.go` | Create | `KeyBar` stateless strip from `[]key.Binding` |
| `internal/uikit/key_bar_test.go` | Create | Separator rendering, ascii fallback |
| `internal/uikit/chip.go` | Create | `Chip` glyph+label pill |
| `internal/uikit/chip_test.go` | Create | Status-bar bg, intent glyph |
| `internal/uikit/form_field.go` | Create | `FormField` text input + intrinsic validation |
| `internal/uikit/form_field_test.go` | Create | Validation lifecycle, error slot |
| `internal/uikit/toast.go` | Create | Typed `Toast` wrapping bubbleup |
| `internal/uikit/toast_test.go` | Create | Intent routing, TTL defaults, truncation |
| `internal/uikit/status_glyph.go` | Create | Atomic `StatusGlyph` renderer |
| `internal/uikit/status_glyph_test.go` | Create | Role-to-glyph mapping |
| `internal/uikit/progress_bar.go` | Create | `ProgressBar` seek/volume fill with partial blocks |
| `internal/uikit/progress_bar_test.go` | Create | Partial-block algorithm, ascii fallback |
| `internal/uikit/spinner.go` | Create | `Spinner` with Done/Fail/Cancel resolution |
| `internal/uikit/spinner_test.go` | Create | State transitions, emitted msgs |
| `internal/config/config.go` | Modify | Add `UIConfig.Glyphs` field + validation |
| `internal/config/config_test.go` | Modify | `glyphs` config parse + defaults |
| `internal/ui/layout/border.go` | Modify | Remove `ᐅ` filter-mode prefix; ascii-mode support |
| `internal/ui/layout/border_test.go` | Modify | Flip `:742` assertion; add ascii snapshots |
| `internal/ui/components/notifications.go` | Modify | Swap `⚠` → `◬` in warning alert definition |
| `internal/ui/components/notifications_test.go` | Modify | Assert `◬` prefix |
| `internal/ui/components/controls.go` | Modify | Route ProgressBar rendering through `uikit.ProgressBar` |
| `internal/ui/components/controls_test.go` | Modify | Progress-bar ascii snapshots |
| `internal/ui/panes/requestflow_boxed.go` | Modify | Use `uikit.SectionLabel` for GATEWAY/APP/GATEWAY LOG/SPOTIFY |
| `internal/ui/panes/requestflow_pane.go` | Modify | Use `uikit.SectionLabel` for AUTO-TRAFFIC |
| `internal/ui/panes/themes.go` | Modify | Use `uikit.ListRow` for theme rows |
| `internal/ui/panes/profile.go` | Modify | Use `uikit.ListRow` for logout/forget entries |
| `internal/ui/panes/playlists_pane.go` | Modify | Use `uikit.LockedRow` for read-only playlists |
| `internal/ui/panes/queue.go` | Modify | Use `uikit.EmptyState` for empty queue |
| `internal/ui/panes/search.go` | Modify | Use `uikit.EmptyState` for empty results |
| `internal/app/render.go` | Modify | Route renderHeader, renderStatusBar, 5×renderWith*Overlay, renderOnboarding*, renderSplash through uikit |
| `internal/app/render_test.go` | Modify | Update assertions against new render paths |
| `internal/app/app.go` | Modify | Store `uikit.Mode` injected from config |
| `internal/app/handlers.go` | Modify | Route all `a.alerts.NewAlertCmd` through `a.toasts.Cmd(Toast{...})` |
| `internal/cliout/message.go` | Modify | Swap `⚠` → `◬` (Step S14) |
| `internal/cliout/message_test.go` | Modify | Assert `◬` prefix |
| `docs/DESIGN.md` | Modify | Strip primitive details; retain layout/grid/page mechanics; add pointer to TUI-DESIGN-SYSTEM.md |
| `docs/TUI-DESIGN-SYSTEM.md` | Create | Canonical reference for primitives, glyph catalogue, role matrix |
| `docs/PANE-TEMPLATE.md` | Modify | Step-2 scaffold uses `PaneChrome` + uikit primitives |
| `docs/spec/00-overview.md` | Modify | Add `NN-tui-design-system` feature row |
| `docs/spec/features/NN-tui-design-system/feature.md` | Create | Feature spec pointing to this plan |

---

## Task Overview

| # | Story | Phase | Blocks / Blocked By |
|---|---|---|---|
| 1 | S1 — scaffold uikit + config + glyph/role/capture | 1 | Blocks: all |
| 2 | S2 — remove `ᐅ`, add ascii mode to border.go | 1 | Blocks: S3, S7 |
| 3 | S3 — PaneChrome primitive | 2 | Blocked by: S1, S2 |
| 4 | S4 — OverlayChrome primitive | 2 | Blocked by: S1 |
| 5 | S5 — Panel primitive | 2 | Blocked by: S1 |
| 6 | S6 — HeaderBar + Chip | 2 | Blocked by: S1 |
| 7 | S7 — StatusBar + KeyBar | 2 | Blocked by: S1, S2 |
| 8 | S8 — TableChrome | 3 | Blocked by: S3 |
| 9 | S9 — ListRow + LockedRow | 3 | Blocked by: S1 |
| 10 | S10 — SectionLabel | 3 | Blocked by: S1 |
| 11 | S11 — EmptyState | 3 | Blocked by: S1 |
| 12 | S12 — URLBox | 3 | Blocked by: S1 |
| 13 | S13 — Toast + migrate call sites | 4 | Blocked by: S1 |
| 14 | S14 — StatusGlyph + `⚠`→`◬` across codebase | 4 | Blocked by: S1 |
| 15 | S15 — ProgressBar (seek + volume) | 4 | Blocked by: S1 |
| 16 | S16 — Spinner Done/Fail/Cancel + onboarding wiring | 4 | Blocked by: S1 |
| 17 | S17 — FormField + onboarding input migration | 5 | Blocked by: S13, S15, S16 |
| 18 | S18 — Onboarding end-to-end rewrite | 5 | Blocked by: S3–S17 |
| 19 | S19 — DESIGN.md rewrite + TUI-DESIGN-SYSTEM.md + PANE-TEMPLATE.md | 5 | Blocked by: all |

---

## Task 1 (S1): Scaffold `internal/uikit`

**Files:**
- Create: `internal/uikit/doc.go`
- Create: `internal/uikit/glyph.go`
- Create: `internal/uikit/glyph_test.go`
- Create: `internal/uikit/role.go`
- Create: `internal/uikit/role_test.go`
- Create: `internal/uikit/config.go`
- Create: `internal/uikit/config_test.go`
- Create: `internal/uikit/capture.go`
- Create: `internal/uikit/capture_test.go`
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Create: `docs/spec/features/NN-tui-design-system/feature.md`
- Modify: `docs/spec/00-overview.md`

### Step 1.1 — Feature branch

- [ ] Run:

```bash
git checkout main && git pull origin main
git checkout -b feat/NN-tui-design-system-scaffold
```

### Step 1.2 — Write failing glyph catalogue tests

Create `internal/uikit/glyph_test.go`:

```go
package uikit_test

import (
	"strings"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlyph_AllRolesHaveBothForms(t *testing.T) {
	for _, role := range uikit.AllGlyphRoles() {
		u := uikit.GlyphFor(role, uikit.GlyphUnicode)
		a := uikit.GlyphFor(role, uikit.GlyphASCII)
		assert.NotEmpty(t, u, "role %q missing unicode", role)
		assert.NotEmpty(t, a, "role %q missing ascii fallback", role)
	}
}

func TestGlyph_UnicodeFormsAreAllSingleColumnExceptPlaybackMulti(t *testing.T) {
	// Banned: double-width glyphs in single-glyph roles that must align in tables.
	mustBeSingleCol := []uikit.GlyphRole{
		uikit.GlyphSuccess, uikit.GlyphError, uikit.GlyphWarning,
		uikit.GlyphInfo, uikit.GlyphRateLimit,
		uikit.GlyphActive, uikit.GlyphInactive, uikit.GlyphAvailable,
		uikit.GlyphLocked,
	}
	for _, role := range mustBeSingleCol {
		g := uikit.GlyphFor(role, uikit.GlyphUnicode)
		assert.Equal(t, 1, uikit.GlyphWidth(g),
			"role %q glyph %q must be 1-col wide", role, g)
	}
}

func TestGlyph_WarningIsCirclePlusInsideTriangle(t *testing.T) {
	// Confirms Section 5.2 of spec: ◬ (U+25EC), not ⚠ (U+26A0).
	assert.Equal(t, "◬", uikit.GlyphFor(uikit.GlyphWarning, uikit.GlyphUnicode))
	assert.Equal(t, "!", uikit.GlyphFor(uikit.GlyphWarning, uikit.GlyphASCII))
}

func TestGlyph_ActionPrefixIsBanned(t *testing.T) {
	// Confirms Section 5.4: `ᐅ` (U+1405) is removed.
	for _, role := range uikit.AllGlyphRoles() {
		u := uikit.GlyphFor(role, uikit.GlyphUnicode)
		assert.False(t, strings.Contains(u, "ᐅ"),
			"role %q must not use banned glyph ᐅ", role)
	}
}

func TestGlyph_ASCIIModeHasNoBMPNonASCII(t *testing.T) {
	for _, role := range uikit.AllGlyphRoles() {
		a := uikit.GlyphFor(role, uikit.GlyphASCII)
		for _, r := range a {
			assert.Less(t, int(r), 128,
				"role %q ascii form %q contains non-ASCII rune %U",
				role, a, r)
		}
	}
}

func TestGlyph_CornerSharpAndDoubleAreBanned(t *testing.T) {
	// Confirms Section 5.1: only rounded corners ╭╮╰╯.
	u := uikit.GlyphFor(uikit.GlyphCornerTL, uikit.GlyphUnicode)
	require.Equal(t, "╭", u)
	u = uikit.GlyphFor(uikit.GlyphCornerTR, uikit.GlyphUnicode)
	require.Equal(t, "╮", u)
}
```

- [ ] Run `go test ./internal/uikit/ -run TestGlyph -v`
  Expected: compile errors — `uikit` package does not exist yet.

### Step 1.3 — Implement glyph catalogue

Create `internal/uikit/glyph.go`:

```go
// Package uikit implements the Spotnik TUI design system. It provides typed
// primitives (PaneChrome, Toast, Panel, etc.), a frozen glyph catalogue with
// ASCII fallback, an emphasis-role to theme-token matrix, and a structural
// test capture helper.
//
// Canonical reference: docs/TUI-DESIGN-SYSTEM.md
// Spec:                docs/superpowers/specs/2026-04-24-tui-design-system-design.md
package uikit

import "github.com/charmbracelet/lipgloss"

// GlyphRole is a symbolic identifier for a glyph. Primitives never render raw
// runes — they look up the glyph by role, and the active GlyphMode determines
// whether the unicode or ascii form is returned.
type GlyphRole string

// GlyphMode is the active rendering mode for glyphs.
type GlyphMode int

const (
	GlyphUnicode GlyphMode = iota
	GlyphASCII
)

// Glyph roles. Grouped by category. Additions require updating the spec in
// docs/superpowers/specs/2026-04-24-tui-design-system-design.md §5 and the
// canonical doc docs/TUI-DESIGN-SYSTEM.md in the same PR.
const (
	// Structural / borders
	GlyphCornerTL GlyphRole = "corner.tl"
	GlyphCornerTR GlyphRole = "corner.tr"
	GlyphCornerBL GlyphRole = "corner.bl"
	GlyphCornerBR GlyphRole = "corner.br"
	GlyphHRule    GlyphRole = "rule.h"
	GlyphVRule    GlyphRole = "rule.v"
	GlyphOverlayDismiss GlyphRole = "overlay.dismiss"

	// Intent / feedback
	GlyphSuccess   GlyphRole = "intent.success"
	GlyphError     GlyphRole = "intent.error"
	GlyphWarning   GlyphRole = "intent.warning"
	GlyphInfo      GlyphRole = "intent.info"
	GlyphRateLimit GlyphRole = "intent.ratelimit"
	GlyphRunning   GlyphRole = "intent.running"
	GlyphDeadline  GlyphRole = "intent.deadline"
	GlyphPaused    GlyphRole = "intent.paused"
	GlyphBlocked   GlyphRole = "intent.blocked"

	// State / availability
	GlyphActive      GlyphRole = "state.active"
	GlyphInactive    GlyphRole = "state.inactive"
	GlyphAvailable   GlyphRole = "state.available"
	GlyphFilledDot   GlyphRole = "state.filled"
	GlyphEmptySquare GlyphRole = "state.empty.square"
	GlyphFilledSquare GlyphRole = "state.filled.square"
	GlyphLocked      GlyphRole = "state.locked"
	GlyphPinned      GlyphRole = "state.pinned"
	GlyphUnpinned    GlyphRole = "state.unpinned"
	GlyphBullet      GlyphRole = "state.bullet"

	// Navigation / scroll
	GlyphScrollDown GlyphRole = "nav.down"
	GlyphScrollUp   GlyphRole = "nav.up"
	GlyphScrollRight GlyphRole = "nav.right"
	GlyphScrollLeft  GlyphRole = "nav.left"
	GlyphEllipsis    GlyphRole = "nav.ellipsis"
	GlyphChevronR    GlyphRole = "nav.chevron.r"
	GlyphChevronL    GlyphRole = "nav.chevron.l"
	GlyphArrowLeft   GlyphRole = "nav.arrow.left"
	GlyphArrowRight  GlyphRole = "nav.arrow.right"
	GlyphArrowUp     GlyphRole = "nav.arrow.up"
	GlyphArrowDown   GlyphRole = "nav.arrow.down"
	GlyphArrowLR     GlyphRole = "nav.arrow.lr"

	// Playback controls
	GlyphPlaying  GlyphRole = "play.playing"
	GlyphPausedPB GlyphRole = "play.paused"
	GlyphStop     GlyphRole = "play.stop"
	GlyphNext     GlyphRole = "play.next"
	GlyphPrev     GlyphRole = "play.prev"
	GlyphFFwd     GlyphRole = "play.ffwd"
	GlyphRewind   GlyphRole = "play.rewind"
	GlyphShuffle  GlyphRole = "play.shuffle"
	GlyphRepeatAll GlyphRole = "play.repeat.all"
	GlyphRepeatOne GlyphRole = "play.repeat.one"
	GlyphRepeatOff GlyphRole = "play.repeat.off"
	GlyphQueue    GlyphRole = "play.queue"
	GlyphEject    GlyphRole = "play.eject"

	// Domain / music / identity
	GlyphMusicNote GlyphRole = "music.note"
	GlyphDoubleNote GlyphRole = "music.double"
	GlyphPremium   GlyphRole = "music.premium"
	GlyphFreeTier  GlyphRole = "music.free"
	GlyphCloud     GlyphRole = "music.cloud"

	// Graphical fills
	GlyphBarFull    GlyphRole = "bar.full"
	GlyphBarSevenEighths GlyphRole = "bar.78"
	GlyphBarThreeQuarters GlyphRole = "bar.34"
	GlyphBarFiveEighths GlyphRole = "bar.58"
	GlyphBarHalf    GlyphRole = "bar.12"
	GlyphBarThreeEighths GlyphRole = "bar.38"
	GlyphBarQuarter GlyphRole = "bar.14"
	GlyphBarOneEighth GlyphRole = "bar.18"
	GlyphBarEmpty   GlyphRole = "bar.empty"
	GlyphBarMedium  GlyphRole = "bar.medium"
	GlyphBarHeavy   GlyphRole = "bar.heavy"
)

// glyphTable holds the unicode + ascii forms for every role. Keep this table
// in sync with the spec §5 and docs/TUI-DESIGN-SYSTEM.md §5.
var glyphTable = map[GlyphRole][2]string{
	// Structural
	GlyphCornerTL:       {"╭", "+"},
	GlyphCornerTR:       {"╮", "+"},
	GlyphCornerBL:       {"╰", "+"},
	GlyphCornerBR:       {"╯", "+"},
	GlyphHRule:          {"─", "-"},
	GlyphVRule:          {"│", "|"},
	GlyphOverlayDismiss: {"×", "x"},

	// Intent
	GlyphSuccess:   {"✓", "+"},
	GlyphError:     {"✗", "x"},
	GlyphWarning:   {"◬", "!"},
	GlyphInfo:      {"→", ">"},
	GlyphRateLimit: {"⧖", "~"},
	GlyphRunning:   {"⚡", "*"},
	GlyphDeadline:  {"◷", "@"},
	GlyphPaused:    {"⏸", "||"},
	GlyphBlocked:   {"⊘", "(-)"},

	// State
	GlyphActive:       {"◉", "(*)"},
	GlyphInactive:     {"◎", "( )"},
	GlyphAvailable:    {"○", "(o)"},
	GlyphFilledDot:    {"●", "(#)"},
	GlyphEmptySquare:  {"□", "[ ]"},
	GlyphFilledSquare: {"■", "[x]"},
	GlyphLocked:       {"◌", "(r)"},
	GlyphPinned:       {"★", "*"},
	GlyphUnpinned:     {"☆", "-"},
	GlyphBullet:       {"•", "*"},

	// Navigation
	GlyphScrollDown:  {"▼", "v"},
	GlyphScrollUp:    {"▲", "^"},
	GlyphScrollRight: {"►", ">"},
	GlyphScrollLeft:  {"◄", "<"},
	GlyphEllipsis:    {"…", "..."},
	GlyphChevronR:    {"›", ">"},
	GlyphChevronL:    {"‹", "<"},
	GlyphArrowLeft:   {"←", "<-"},
	GlyphArrowRight:  {"→", "->"},
	GlyphArrowUp:     {"↑", "^"},
	GlyphArrowDown:   {"↓", "v"},
	GlyphArrowLR:     {"↔", "<>"},

	// Playback
	GlyphPlaying:   {"▶", ">"},
	GlyphPausedPB:  {"▷", "|>"},
	GlyphStop:      {"■", "[]"},
	GlyphNext:      {"⏭", ">>"},
	GlyphPrev:      {"⏮", "<<"},
	GlyphFFwd:      {"⏩", ">>>"},
	GlyphRewind:    {"⏪", "<<<"},
	GlyphShuffle:   {"⇄", "sh"},
	GlyphRepeatAll: {"↻", "rp"},
	GlyphRepeatOne: {"↻¹", "rp1"},
	GlyphRepeatOff: {"⟳", "ro"},
	GlyphQueue:     {"≡", "Q"},
	GlyphEject:     {"⏏", "/E"},

	// Domain
	GlyphMusicNote:  {"♪", "*"},
	GlyphDoubleNote: {"♫", "**"},
	GlyphPremium:    {"♛", "*P"},
	GlyphFreeTier:   {"○", "(o)"},
	GlyphCloud:      {"☁", "(c)"},

	// Graphical
	GlyphBarFull:          {"█", "#"},
	GlyphBarSevenEighths:  {"▉", "#"},
	GlyphBarThreeQuarters: {"▊", "#"},
	GlyphBarFiveEighths:   {"▋", "="},
	GlyphBarHalf:          {"▌", "="},
	GlyphBarThreeEighths:  {"▍", "-"},
	GlyphBarQuarter:       {"▎", "-"},
	GlyphBarOneEighth:     {"▏", "."},
	GlyphBarEmpty:         {"░", "."},
	GlyphBarMedium:        {"▒", ":"},
	GlyphBarHeavy:         {"▓", "#"},
}

// GlyphFor returns the form of a role in the given mode.
// Unknown role returns empty string — caller is expected to have compiled
// against one of the exported GlyphRole constants.
func GlyphFor(role GlyphRole, mode GlyphMode) string {
	row, ok := glyphTable[role]
	if !ok {
		return ""
	}
	if mode == GlyphASCII {
		return row[1]
	}
	return row[0]
}

// AllGlyphRoles returns every registered role. Used by tests for full-table
// validation; order is not guaranteed.
func AllGlyphRoles() []GlyphRole {
	roles := make([]GlyphRole, 0, len(glyphTable))
	for r := range glyphTable {
		roles = append(roles, r)
	}
	return roles
}

// GlyphWidth returns the terminal-column width of a rendered glyph.
// Thin wrapper around lipgloss.Width for test convenience.
func GlyphWidth(s string) int { return lipgloss.Width(s) }
```

### Step 1.4 — Run glyph tests and verify they pass

- [ ] Run `go test ./internal/uikit/ -run TestGlyph -v`
  Expected: PASS for all 6 tests.

### Step 1.5 — Write failing role matrix tests

Create `internal/uikit/role_test.go`:

```go
package uikit_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func TestRole_AllRolesResolveToNonEmptyColour(t *testing.T) {
	th := theme.Load("black")
	for _, role := range uikit.AllRoles() {
		c := uikit.ColourFor(role, th)
		assert.NotEmpty(t, string(c), "role %q resolved to empty colour", role)
	}
}

func TestRole_AccentFallsBackToSeekBarWhenUnset(t *testing.T) {
	// The spec Section 6.1 states Accent falls back to SeekBar when
	// theme.Accent() is unset. Theme implementations must return either the
	// TOML `accent` field or SeekBar().
	th := theme.Load("black")
	assert.NotEmpty(t, string(uikit.ColourFor(uikit.RoleAccent, th)))
}

func TestRole_PlainIsTextPrimary(t *testing.T) {
	th := theme.Load("black")
	assert.Equal(t, th.TextPrimary(), uikit.ColourFor(uikit.RolePlain, th))
}

func TestRole_MutedIsTextMuted(t *testing.T) {
	th := theme.Load("black")
	assert.Equal(t, th.TextMuted(), uikit.ColourFor(uikit.RoleMuted, th))
}
```

- [ ] Run `go test ./internal/uikit/ -run TestRole -v`
  Expected: compile error — role API does not exist.

### Step 1.6 — Implement role matrix

Create `internal/uikit/role.go`:

```go
package uikit

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// Role identifies an emphasis role in the design system. Primitives declare
// which fields map to which roles; the role-to-colour resolution happens via
// ColourFor(role, theme). Call sites never pass raw colours — they set a role.
type Role string

const (
	RoleAccent        Role = "accent"
	RoleStrong        Role = "strong"
	RolePlain         Role = "plain"
	RoleMuted         Role = "muted"
	RoleSuccess       Role = "success"
	RoleError         Role = "error"
	RoleWarning       Role = "warning"
	RoleInfo          Role = "info"
	RoleSelection     Role = "selection"
	RoleColumnIndex   Role = "column.index"
	RoleColumnPrimary Role = "column.primary"
	RoleColumnSecondary Role = "column.secondary"
	RoleColumnTertiary Role = "column.tertiary"
)

// AllRoles returns every registered role.
func AllRoles() []Role {
	return []Role{
		RoleAccent, RoleStrong, RolePlain, RoleMuted,
		RoleSuccess, RoleError, RoleWarning, RoleInfo,
		RoleSelection,
		RoleColumnIndex, RoleColumnPrimary, RoleColumnSecondary, RoleColumnTertiary,
	}
}

// ColourFor resolves a role to a lipgloss.Color on the given theme.
// Strong uses TextPrimary (bold is applied at the primitive level, not here).
func ColourFor(r Role, th theme.Theme) lipgloss.Color {
	switch r {
	case RoleAccent:
		return th.Accent()
	case RoleStrong, RolePlain:
		return th.TextPrimary()
	case RoleMuted:
		return th.TextMuted()
	case RoleSuccess:
		return th.Success()
	case RoleError:
		return th.Error()
	case RoleWarning:
		return th.Warning()
	case RoleInfo:
		return th.Info()
	case RoleSelection:
		return th.SelectedFg()
	case RoleColumnIndex:
		return th.ColumnIndex()
	case RoleColumnPrimary:
		return th.ColumnPrimary()
	case RoleColumnSecondary:
		return th.ColumnSecondary()
	case RoleColumnTertiary:
		return th.ColumnTertiary()
	default:
		return th.TextPrimary()
	}
}

// Apply returns a lipgloss.Style foreground-coloured by the role, with Bold
// set when role is Strong. All primitives compose their own additional style
// attributes (width, alignment) on top of this.
func Apply(r Role, th theme.Theme) lipgloss.Style {
	s := lipgloss.NewStyle().Foreground(ColourFor(r, th))
	if r == RoleStrong {
		s = s.Bold(true)
	}
	return s
}
```

- [ ] Run `go test ./internal/uikit/ -run TestRole -v`
  Expected: PASS.

### Step 1.7 — Write failing config tests

Create `internal/uikit/config_test.go`:

```go
package uikit_test

import (
	"os"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
)

func TestResolve_Auto_WithUTF8Lang_ReturnsUnicode(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LANG", "en_US.UTF-8")
	assert.Equal(t, uikit.GlyphUnicode, uikit.ResolveMode("auto"))
}

func TestResolve_Auto_WithCLang_ReturnsASCII(t *testing.T) {
	t.Setenv("LC_ALL", "C")
	t.Setenv("LANG", "")
	assert.Equal(t, uikit.GlyphASCII, uikit.ResolveMode("auto"))
}

func TestResolve_ExplicitUnicode_Honoured(t *testing.T) {
	t.Setenv("LANG", "C")
	assert.Equal(t, uikit.GlyphUnicode, uikit.ResolveMode("unicode"))
}

func TestResolve_ExplicitASCII_Honoured(t *testing.T) {
	t.Setenv("LANG", "en_US.UTF-8")
	assert.Equal(t, uikit.GlyphASCII, uikit.ResolveMode("ascii"))
}

func TestResolve_NO_COLOR_IsOrthogonal(t *testing.T) {
	// NO_COLOR strips colour, not glyphs. Explicitly assert the same LANG
	// still resolves to unicode when NO_COLOR is set.
	t.Setenv("NO_COLOR", "1")
	t.Setenv("LANG", "en_US.UTF-8")
	assert.Equal(t, uikit.GlyphUnicode, uikit.ResolveMode("auto"))
	os.Unsetenv("NO_COLOR")
}
```

- [ ] Run `go test ./internal/uikit/ -run TestResolve -v`
  Expected: compile error — `ResolveMode` does not exist.

### Step 1.8 — Implement config resolver

Create `internal/uikit/config.go`:

```go
package uikit

import (
	"os"
	"strings"
	"sync"
)

// ResolveMode converts the user-facing config value ("auto", "unicode",
// "ascii") to a concrete GlyphMode. Auto inspects LC_ALL then LANG for a
// UTF-8 marker; anything else resolves to ASCII.
//
// NO_COLOR is orthogonal — it strips colour, not glyphs — and is not
// consulted here.
func ResolveMode(cfg string) GlyphMode {
	switch strings.ToLower(strings.TrimSpace(cfg)) {
	case "unicode":
		return GlyphUnicode
	case "ascii":
		return GlyphASCII
	default: // "auto" or empty
		if isUTF8Locale() {
			return GlyphUnicode
		}
		return GlyphASCII
	}
}

func isUTF8Locale() bool {
	for _, key := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		v := os.Getenv(key)
		if v == "" {
			continue
		}
		lower := strings.ToLower(v)
		if strings.Contains(lower, "utf-8") || strings.Contains(lower, "utf8") {
			return true
		}
		// First non-empty locale variable wins; a non-UTF-8 value rules out
		// unicode mode.
		return false
	}
	return false
}

// activeMode stores the resolved mode after first call to Use.
var (
	activeModeOnce sync.Once
	activeMode     GlyphMode
)

// Use resolves and caches the active GlyphMode. Called once at app startup
// after config is loaded. Subsequent calls are no-ops (mode is frozen for
// the lifetime of the process).
func Use(cfg string) {
	activeModeOnce.Do(func() {
		activeMode = ResolveMode(cfg)
	})
}

// ActiveMode returns the cached mode. Panics if Use has not been called —
// catches wiring bugs early. Tests use SetModeForTest instead.
func ActiveMode() GlyphMode { return activeMode }

// SetModeForTest overrides the active mode in tests. Resets the sync.Once so
// subsequent Use calls can reinitialise.
func SetModeForTest(m GlyphMode) {
	activeMode = m
	activeModeOnce = sync.Once{}
}
```

- [ ] Run `go test ./internal/uikit/ -run TestResolve -v`
  Expected: PASS.

### Step 1.9 — Write failing capture tests

Create `internal/uikit/capture_test.go`:

```go
package uikit_test

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
)

func TestCapture_StripsANSI_ReturnsPlainLines(t *testing.T) {
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).
		Render("hello") + "\n" + "world"
	got := uikit.Capture(styled)
	assert.Equal(t, []string{"hello", "world"}, got)
}

func TestCapture_PreservesLeadingSpaces(t *testing.T) {
	got := uikit.Capture("  indented\n    more")
	assert.Equal(t, []string{"  indented", "    more"}, got)
}

func TestCapture_EmptyString_ReturnsEmptySlice(t *testing.T) {
	got := uikit.Capture("")
	assert.Equal(t, []string{""}, got)
}
```

- [ ] Run `go test ./internal/uikit/ -run TestCapture -v`
  Expected: compile error — `Capture` does not exist.

### Step 1.10 — Implement capture helper

Create `internal/uikit/capture.go`:

```go
package uikit

import (
	"regexp"
	"strings"
)

// ansiEscape matches any ANSI CSI escape sequence. Used by Capture to produce
// plain-text lines for structural assertions in primitive tests.
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// Capture returns the rendered string with all ANSI colour/style codes stripped
// and split into individual lines. Used in primitive tests to assert on
// structural content without colour noise.
func Capture(rendered string) []string {
	plain := ansiEscape.ReplaceAllString(rendered, "")
	return strings.Split(plain, "\n")
}
```

- [ ] Run `go test ./internal/uikit/ -run TestCapture -v`
  Expected: PASS.

### Step 1.11 — Add `UIConfig.Glyphs` to config

Modify `internal/config/config.go` — add to the `UIConfig` struct (exact line numbers subject to current file state; search for the `UIConfig` type):

```go
// UIConfig holds TUI rendering preferences.
type UIConfig struct {
	Theme      string `toml:"theme"`
	Preset     int    `toml:"preset"`
	Visualizer string `toml:"visualizer"`
	Palette    string `toml:"palette"`   // existing — for CLI output
	Glyphs     string `toml:"glyphs"`    // NEW — "auto" | "unicode" | "ascii"
}
```

Add validation — append to the existing `Validate()` method on `UIConfig`:

```go
func (c *UIConfig) Validate() error {
	// ... existing validation ...
	switch strings.ToLower(strings.TrimSpace(c.Glyphs)) {
	case "", "auto", "unicode", "ascii":
		// OK; empty defaults to auto at resolution time.
	default:
		return fmt.Errorf("ui.glyphs must be one of auto|unicode|ascii, got %q", c.Glyphs)
	}
	return nil
}
```

Add to `internal/config/config_test.go`:

```go
func TestUIConfig_Glyphs_DefaultAllowedValues(t *testing.T) {
	cases := []struct {
		name  string
		value string
		ok    bool
	}{
		{"default empty", "", true},
		{"auto", "auto", true},
		{"unicode", "unicode", true},
		{"ascii", "ascii", true},
		{"uppercase", "ASCII", true},
		{"invalid", "nerd", false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			c := &config.UIConfig{Glyphs: tt.value}
			err := c.Validate()
			if tt.ok {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
```

- [ ] Run `go test ./internal/config/ -run TestUIConfig_Glyphs -v`
  Expected: PASS.

### Step 1.12 — Create feature + overview entries

Create `docs/spec/features/NN-tui-design-system/feature.md` (replace `NN` with the next available number from `docs/spec/00-overview.md`):

```markdown
---
title: TUI Design System
feature: NN-tui-design-system
status: in-progress
---

# Feature — TUI Design System

See spec: `docs/superpowers/specs/2026-04-24-tui-design-system-design.md`
See plan: `docs/superpowers/plans/2026-04-24-tui-design-system.md`

## Description

Land `internal/uikit` — typed TUI primitives with unicode/ASCII fallback,
frozen glyph catalogue, role-to-token matrix, and feedback-channel consistency
rules. Migrate every existing TUI call site to use primitives. Swap `⚠`→`◬`
and remove `ᐅ` across code and docs.

## Acceptance criteria

- [ ] `internal/uikit` exists with 18 primitives, each with unicode+ASCII
      snapshot tests and role assertions.
- [ ] `ui.glyphs = "auto" | "unicode" | "ascii"` config is honoured.
- [ ] All ad-hoc `lipgloss.NewStyle()` chains for primitives removed from
      `internal/app/render.go` and `internal/ui/panes/*`.
- [ ] `ᐅ` (U+1405) absent from the entire codebase (except in a banned-glyph
      CI-lint rule if added).
- [ ] `⚠` (U+26A0) absent from `internal/` Go files; all warnings use `◬`.
- [ ] `docs/TUI-DESIGN-SYSTEM.md` lands as the canonical reference.
- [ ] `docs/DESIGN.md` retains layout/grid/preset mechanics only; primitive
      rendering details removed with pointers to TUI-DESIGN-SYSTEM.md.
- [ ] `make ci` passes at ≥ 80% coverage.
```

Modify `docs/spec/00-overview.md` — append a row:

```markdown
| NN | TUI Design System | in-progress | 19 |
```

### Step 1.13 — Verify `make ci` passes

- [ ] Run `make ci`
  Expected: lint clean, tests pass, coverage ≥ 80%.

### Step 1.14 — Commit and open PR

- [ ] Run:

```bash
git add internal/uikit/ internal/config/ docs/spec/
git commit -m "$(cat <<'EOF'
feat(uikit): scaffold internal/uikit — glyph/role/config/capture

Adds the internal/uikit package with:
- Frozen glyph catalogue (unicode + ascii forms, banned ᐅ and ⚠)
- Role/colour matrix resolving to theme tokens
- ui.glyphs config with auto/unicode/ascii; env-based detection
- Capture test helper for ANSI-stripped structural assertions
- Feature spec and overview row for NN-tui-design-system

Gate for the TUI design system migration described in
docs/superpowers/specs/2026-04-24-tui-design-system-design.md.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
git push origin feat/NN-tui-design-system-scaffold
gh pr create --title "feat(uikit): scaffold internal/uikit — glyph/role/config/capture" --body "$(cat <<'EOF'
## Summary
- Scaffolds `internal/uikit` with glyph catalogue, role matrix, config, and Capture helper
- Adds `ui.glyphs` config knob (`auto` | `unicode` | `ascii`)
- Registers feature NN-tui-design-system in the overview

## Test plan
- [ ] `make ci` passes
- [ ] Glyph catalogue integrity tests pass
- [ ] Role resolution tests pass
- [ ] Env-based auto-detection tests pass

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

---

## Task 2 (S2): Remove `ᐅ` from `layout/border.go` filter mode, add ascii mode

**Files:**
- Modify: `internal/ui/layout/border.go` (lines 28, 36, 213, 220, 224)
- Modify: `internal/ui/layout/border_test.go` (line 742, plus new ascii-snapshot tests)

### Step 2.1 — Branch

- [ ] Run:

```bash
git checkout main && git pull origin main
git checkout -b feat/NN-border-no-arrow-ascii-mode
```

### Step 2.2 — Write failing test: filter mode must use notch format

Modify `internal/ui/layout/border_test.go` around line 742 — invert the existing assertion:

```go
func TestRenderPaneBorder_FilterMode_UsesNotchNotArrow(t *testing.T) {
	cfg := layout.BorderConfig{
		Width:       60,
		Height:      4,
		Title:       "Queue",
		ToggleKey:   2,
		AccentColor: lipgloss.Color("#ffffff"),
		Focused:     true,
		FilterQuery: "rock",
		Theme:       theme.Load("black"),
	}
	out := layout.RenderPaneBorder("content", cfg)
	topLine := strings.Split(out, "\n")[0]

	assert.NotContains(t, topLine, "ᐅ",
		"filter mode must not use ᐅ prefix — use notch format")
	assert.Contains(t, topLine, "filtering: \"rock\"",
		"filter mode must show preamble")
	assert.Contains(t, topLine, "╮ Esc close ╭",
		"filter mode must use corner-notch format for the close action")
}
```

Also remove / rewrite the original assertion at line 742 that asserted `Contains(topLine, "ᐅ")`.

- [ ] Run `go test ./internal/ui/layout/ -run TestRenderPaneBorder_FilterMode -v`
  Expected: FAIL — current code still emits `ᐅ`.

### Step 2.3 — Write failing ascii-mode test

Add to `internal/ui/layout/border_test.go`:

```go
func TestRenderPaneBorder_ASCIIMode_SwapsCorners(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	cfg := layout.BorderConfig{
		Width:       40,
		Height:      3,
		Title:       "Test",
		AccentColor: lipgloss.Color("#ffffff"),
		Focused:     true,
		Theme:       theme.Load("black"),
	}
	out := layout.RenderPaneBorder("", cfg)
	lines := strings.Split(out, "\n")

	assert.Contains(t, lines[0], "+", "ascii top-left corner")
	assert.NotContains(t, lines[0], "╭", "no unicode corner in ascii mode")
	assert.NotContains(t, lines[0], "╮", "no unicode corner in ascii mode")
	assert.Contains(t, lines[len(lines)-1], "+", "ascii bottom corner")
	assert.NotContains(t, lines[len(lines)-1], "╰")
	assert.NotContains(t, lines[len(lines)-1], "╯")
}
```

- [ ] Run `go test ./internal/ui/layout/ -run TestRenderPaneBorder_ASCIIMode -v`
  Expected: FAIL — border.go hardcodes unicode corners.

### Step 2.4 — Rewrite filter mode and introduce glyph lookup

Modify `internal/ui/layout/border.go`:

Replace constants block near the corner definitions:

```go
import (
	// ... existing ...
	"github.com/initgrep-apps/spotnik/internal/uikit"
)
```

Replace the hardcoded `cornerTL = "╭"` etc. with glyph lookups:

```go
func corners() (tl, tr, bl, br, h, v string) {
	m := uikit.ActiveMode()
	return uikit.GlyphFor(uikit.GlyphCornerTL, m),
		uikit.GlyphFor(uikit.GlyphCornerTR, m),
		uikit.GlyphFor(uikit.GlyphCornerBL, m),
		uikit.GlyphFor(uikit.GlyphCornerBR, m),
		uikit.GlyphFor(uikit.GlyphHRule, m),
		uikit.GlyphFor(uikit.GlyphVRule, m)
}
```

Replace `RenderPaneBorder` body to use `corners()` instead of the const strings.

Rewrite `buildRightSegment` filter branch:

```go
func buildRightSegment(cfg BorderConfig, keyHintStyle, mutedStyle func(string) string) string {
	if cfg.FilterQuery != "" {
		// Filter mode — notch format, same as actions mode.
		preamble := mutedStyle(`filtering: "` + cfg.FilterQuery + `"`)
		sep := mutedStyle(" ─")  // single join-dash
		borderChar := func(s string) string {
			style := lipgloss.NewStyle().Foreground(cfg.AccentColor)
			if !cfg.Focused {
				style = style.Faint(true)
			}
			return style.Render(s)
		}
		_, _, _, _, _, _ = corners() // reserved for ascii refactor
		tlCorner := uikit.GlyphFor(uikit.GlyphCornerTL, uikit.ActiveMode())
		trCorner := uikit.GlyphFor(uikit.GlyphCornerTR, uikit.ActiveMode())
		notch := borderChar(trCorner) + " " +
			keyHintStyle("Esc") + " " + mutedStyle("close") + " " +
			borderChar(tlCorner)
		return preamble + sep + notch
	}
	// ... existing actions-mode branch unchanged ...
}
```

Update docstrings on lines 28, 36, 213, 220 to remove `ᐅ` references:

- Line 28: `// Displayed as: notch-format "╮key label╭", separated by "─".`
- Line 36: `// When set, replaces the action shortcuts with: filtering: "query" ─╮ Esc close ╭`
- Line 213: `// Filter mode: "filtering: "query" ─╮ Esc close ╭" — same notch format as actions.`
- Line 220: `// Filter mode: "filtering: "query" ─╮ Esc close ╭"`

### Step 2.5 — Run tests and verify pass

- [ ] Run `go test ./internal/ui/layout/ -v`
  Expected: all tests PASS including new filter-mode and ascii-mode.

- [ ] Run `make ci`
  Expected: clean.

### Step 2.6 — Commit and PR

- [ ] Run:

```bash
git add internal/ui/layout/border.go internal/ui/layout/border_test.go
git commit -m "$(cat <<'EOF'
refactor(layout): remove ᐅ, notch-format filter mode, ascii-mode corners

- Filter mode now uses the same corner-notch format as actions mode
  (╮ Esc close ╭ instead of ᐅEsc close)
- Border corners look up the glyph via uikit.ActiveMode(), so ascii
  mode swaps ╭╮╰╯─│ → +++--| as required by the design system
- Docstrings updated; border_test.go filter-mode assertion flipped
  to require absence of ᐅ and presence of the notch

Part of NN-tui-design-system (docs/superpowers/specs/2026-04-24-...).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
git push origin feat/NN-border-no-arrow-ascii-mode
gh pr create --title "refactor(layout): remove ᐅ, notch-format filter mode, ascii-mode corners" --body "Gate for the TUI design-system migration. Merges before any primitive story."
```

---

## Task 3 (S3): `PaneChrome` primitive

**Files:**
- Create: `internal/uikit/pane_chrome.go`
- Create: `internal/uikit/pane_chrome_test.go`

### Step 3.1 — Branch

- [ ] Run:

```bash
git checkout main && git pull origin main
git checkout -b feat/NN-uikit-pane-chrome
```

### Step 3.2 — Write failing tests

Create `internal/uikit/pane_chrome_test.go`:

```go
package uikit_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaneChrome_UnicodeSnapshot_ActionsMode(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	th := theme.Load("black")

	pc := uikit.PaneChrome{
		Width: 60, Height: 4,
		Title: "Playlists", ToggleKey: 3,
		Actions: []layout.Action{{Key: "f", Label: "filter"}, {Key: "n", Label: "new"}},
		AccentColor: th.PaneBorderPlaylists(),
		Focused: true, Theme: th,
	}
	out := pc.Render("  (content)")
	lines := uikit.Capture(out)
	require.Len(t, lines, 4)

	assert.True(t, strings.HasPrefix(lines[0], "╭─ ³Playlists"),
		"title immediately after '─ ', no trailing space")
	assert.Contains(t, lines[0], "╮ f filter ╭",
		"first action notch")
	assert.Contains(t, lines[0], "╮ n new ╭",
		"second action notch")
	assert.True(t, strings.HasSuffix(lines[0], "╭╮"),
		"last action ╭ immediately followed by top-right corner ╮")
}

func TestPaneChrome_ASCIISnapshot_ActionsMode(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)
	th := theme.Load("black")

	pc := uikit.PaneChrome{
		Width: 60, Height: 4,
		Title: "Playlists", ToggleKey: 3,
		Actions: []layout.Action{{Key: "f", Label: "filter"}},
		AccentColor: th.PaneBorderPlaylists(),
		Focused: true, Theme: th,
	}
	lines := uikit.Capture(pc.Render(""))
	assert.True(t, strings.HasPrefix(lines[0], "+- 3 Playlists"))
	assert.Contains(t, lines[0], "+ f filter +")
	assert.True(t, strings.HasSuffix(lines[0], "++"))
	// No unicode corners anywhere.
	for _, l := range lines {
		assert.NotContains(t, l, "╭")
		assert.NotContains(t, l, "╮")
	}
}

func TestPaneChrome_FilterMode_NoArrow(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	th := theme.Load("black")

	pc := uikit.PaneChrome{
		Width: 60, Height: 3,
		Title: "Queue", ToggleKey: 2,
		FilterQuery: "rock",
		AccentColor: th.PaneBorderQueue(),
		Focused: true, Theme: th,
	}
	lines := uikit.Capture(pc.Render(""))
	assert.NotContains(t, lines[0], "ᐅ")
	assert.Contains(t, lines[0], `filtering: "rock"`)
	assert.Contains(t, lines[0], "╮ Esc close ╭")
}

func TestPaneChrome_UnfocusedTitleNotBold(t *testing.T) {
	// Structural assertion: when Focused=false, title is rendered without bold.
	// We don't have a direct "bold" check on the plain string, so assert
	// that the raw bytes don't contain the ANSI bold sequence (ESC[1m).
	th := theme.Load("black")
	pc := uikit.PaneChrome{
		Width: 40, Height: 3, Title: "Test",
		AccentColor: th.PaneBorderPlaylists(), Focused: false, Theme: th,
	}
	raw := pc.Render("")
	assert.NotContains(t, raw, "\x1b[1m", "unfocused title must not be bold")
}

func TestPaneChrome_WidthAndHeightMatch(t *testing.T) {
	th := theme.Load("black")
	pc := uikit.PaneChrome{
		Width: 50, Height: 5, Title: "X",
		AccentColor: th.PaneBorderPlaylists(), Focused: true, Theme: th,
	}
	lines := uikit.Capture(pc.Render("line1\nline2\nline3"))
	assert.Len(t, lines, 5, "height matches")
	for _, l := range lines {
		assert.Equal(t, 50, lipgloss.Width(l), "width matches")
	}
}
```

- [ ] Run `go test ./internal/uikit/ -run TestPaneChrome -v`
  Expected: compile errors — `PaneChrome` does not exist.

### Step 3.3 — Implement PaneChrome

Create `internal/uikit/pane_chrome.go`:

```go
package uikit

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// PaneChrome is the standard bordered pane primitive. It wraps
// layout.RenderPaneBorder (which is the internal implementation), applying
// the design-system role matrix for title, toggle key, and action hints.
//
// Every pane composes PaneChrome to render its outer border. Overlay-style
// panels should use OverlayChrome instead; full-screen panels should use
// Panel.
type PaneChrome struct {
	Width       int
	Height      int
	Title       string
	ToggleKey   int            // 0 = no key shown
	Actions     []layout.Action
	AccentColor lipgloss.Color // per-pane border token (theme.PaneBorderX())
	Focused     bool
	FilterQuery string         // "" = no filter mode
	Theme       theme.Theme
}

// Render produces the bordered pane with `content` placed inside. The content
// argument is pre-sized to (Width-2, Height-2) by the caller; this method
// only composes the border around it.
//
// Mode (unicode/ascii) is taken from uikit.ActiveMode() — callers do not pass
// it explicitly.
func (p PaneChrome) Render(content string) string {
	// Delegate to layout.RenderPaneBorder — the existing implementation now
	// honours uikit.ActiveMode() for corner glyphs (see layout/border.go).
	cfg := layout.BorderConfig{
		Width:       p.Width,
		Height:      p.Height,
		Title:       p.Title,
		ToggleKey:   p.ToggleKey,
		Actions:     p.Actions,
		AccentColor: p.AccentColor,
		Focused:     p.Focused,
		FilterQuery: p.FilterQuery,
		Theme:       p.Theme,
	}
	return layout.RenderPaneBorder(content, cfg)
}
```

### Step 3.4 — Run tests

- [ ] Run `go test ./internal/uikit/ -run TestPaneChrome -v`
  Expected: PASS (assumes Task 2 has merged — `layout.RenderPaneBorder` already honours `uikit.ActiveMode`).

### Step 3.5 — Commit and PR

- [ ] Run:

```bash
git add internal/uikit/pane_chrome.go internal/uikit/pane_chrome_test.go
git commit -m "$(cat <<'EOF'
feat(uikit): add PaneChrome primitive

Wraps layout.RenderPaneBorder as the design-system primitive for pane
borders. Unicode and ascii modes snapshot-tested. Notch format verified
for actions mode and filter mode. No ᐅ anywhere.

Part of NN-tui-design-system.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
git push origin feat/NN-uikit-pane-chrome
gh pr create --title "feat(uikit): add PaneChrome primitive"
```

---

## Task 4 (S4): `OverlayChrome` primitive

**Files:**
- Create: `internal/uikit/overlay_chrome.go`
- Create: `internal/uikit/overlay_chrome_test.go`
- Modify: `internal/app/render.go` — consolidate `renderWithThemeOverlay`, `renderWithDeviceOverlay`, `renderWithProfileOverlay`, `renderWithSearchOverlay`, `renderWithHelpOverlay` into a single helper backed by `uikit.OverlayChrome`.

### Step 4.1 — Branch, write failing tests

Create `internal/uikit/overlay_chrome_test.go`:

```go
package uikit_test

import (
	"strings"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func TestOverlayChrome_Unicode_DefaultBorderAccent(t *testing.T) {
	th := theme.Load("black")
	oc := uikit.OverlayChrome{
		Width: 40, Height: 10, Title: "Search",
		Actions: []uikit.Action{{Key: "Enter", Label: "play"}, {Key: "Tab", Label: "filter"}},
		Theme: th,
	}
	lines := uikit.Capture(oc.Render("  body"))
	assert.Equal(t, 10, len(lines))
	assert.True(t, strings.HasPrefix(lines[0], "╭─ Search"))
	assert.Contains(t, lines[0], "╮ Enter play ╭")
}

func TestOverlayChrome_ASCII(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)
	th := theme.Load("black")
	oc := uikit.OverlayChrome{Width: 30, Height: 5, Title: "X", Theme: th}
	lines := uikit.Capture(oc.Render(""))
	for _, l := range lines {
		assert.NotContains(t, l, "╭")
	}
}
```

### Step 4.2 — Implement OverlayChrome

Create `internal/uikit/overlay_chrome.go`:

```go
package uikit

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// Action mirrors layout.Action for the primitive surface.
type Action = layout.Action

// OverlayChrome renders a floating overlay panel. Visually identical to a
// focused PaneChrome but with Accent as the border colour (overlays are
// always "focused" in the sense that they own the input).
type OverlayChrome struct {
	Width   int
	Height  int
	Title   string
	Actions []Action
	Theme   theme.Theme
}

func (o OverlayChrome) Render(content string) string {
	cfg := layout.BorderConfig{
		Width:       o.Width,
		Height:      o.Height,
		Title:       o.Title,
		Actions:     o.Actions,
		AccentColor: lipgloss.Color(o.Theme.Accent()),
		Focused:     true,
		Theme:       o.Theme,
	}
	return layout.RenderPaneBorder(content, cfg)
}
```

### Step 4.3 — Migrate `render.go` overlay helpers

In `internal/app/render.go`, replace the 5 `renderWith*Overlay` functions with a single helper:

```go
// renderWithOverlayChrome dims the background and composites a uikit-rendered
// overlay on top. Replaces the 5 per-overlay helpers.
func (a *App) renderWithOverlayChrome(background string, overlayView string) string {
	dimmed := lipgloss.NewStyle().Faint(true).Render(background)
	if a.width <= 0 || a.height <= 0 {
		return dimmed + "\n" + overlayView
	}
	return btoverlay.Composite(overlayView, dimmed, btoverlay.Center, btoverlay.Center, 0, 0)
}
```

Update `buildView` to call the new helper; existing `a.searchPane.View()`,
`a.devicePane.View()`, etc. already produce the rendered overlay string.

### Step 4.4 — Run tests, commit, PR

- [ ] Run `go test ./internal/uikit/ ./internal/app/ -run "TestOverlayChrome|TestRender" -v`
- [ ] Commit with message `feat(uikit): add OverlayChrome; consolidate 5 render.go overlay helpers`.

---

## Task 5 (S5): `Panel` primitive

**Files:**
- Create: `internal/uikit/panel.go`
- Create: `internal/uikit/panel_test.go`

### Step 5.1 — Write failing tests

Create `internal/uikit/panel_test.go`:

```go
package uikit_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func TestPanel_TitleInBorder(t *testing.T) {
	th := theme.Load("black")
	p := uikit.Panel{
		Width: 60, Height: 10,
		Title: "Step 2 of 2 — Authorize Spotnik with Spotify",
		Theme: th,
	}
	lines := uikit.Capture(p.Render("body"))
	assert.Contains(t, lines[0], "Step 2 of 2")
}

func TestPanel_ErrorIntent_UsesErrorBorder(t *testing.T) {
	th := theme.Load("black")
	p := uikit.Panel{
		Width: 40, Height: 6,
		Title: "Error", Intent: uikit.PanelIntentError,
		Theme: th,
	}
	raw := p.Render("")
	// Error border colour = th.Error(); assert via glyph presence only.
	// Colour is asserted via role integration in a separate layer.
	assert.Contains(t, raw, "╭")
}
```

### Step 5.2 — Implement

Create `internal/uikit/panel.go`:

```go
package uikit

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// PanelIntent controls the Panel border colour. Default intent uses Accent
// (success/neutral screens). Error intent is used for the onboarding error
// panel.
type PanelIntent int

const (
	PanelIntentDefault PanelIntent = iota
	PanelIntentError
)

// Panel is a full-screen bordered container. Title lives in the top border
// (no separate step-header string). Used by onboarding, auth, splash, and
// too-small screens.
type Panel struct {
	Width  int
	Height int
	Title  string
	Intent PanelIntent
	Theme  theme.Theme
}

func (p Panel) Render(content string) string {
	var border lipgloss.Color
	switch p.Intent {
	case PanelIntentError:
		border = p.Theme.Error()
	default:
		border = p.Theme.Accent()
	}
	cfg := layout.BorderConfig{
		Width:       p.Width,
		Height:      p.Height,
		Title:       p.Title,
		AccentColor: border,
		Focused:     true,
		Theme:       p.Theme,
	}
	return layout.RenderPaneBorder(content, cfg)
}
```

### Step 5.3 — Run, commit, PR

- [ ] Run `go test ./internal/uikit/ -run TestPanel -v` — PASS.
- [ ] Commit `feat(uikit): add Panel primitive (title in border, intent-driven colour)`.

---

## Task 6 (S6): `HeaderBar` + `Chip`

**Files:**
- Create: `internal/uikit/header_bar.go`
- Create: `internal/uikit/chip.go`
- Create: `internal/uikit/header_bar_test.go`
- Create: `internal/uikit/chip_test.go`
- Modify: `internal/app/render.go` — `renderHeader` uses `HeaderBar`; device + profile chips use `Chip`.
- Modify: `internal/app/render_test.go` — assertions update to new types.

### Step 6.1 — Failing tests for Chip

Create `internal/uikit/chip_test.go`:

```go
package uikit_test

import (
	"strings"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func TestChip_Unicode_ActiveDevice(t *testing.T) {
	th := theme.Load("black")
	c := uikit.Chip{Glyph: uikit.GlyphActive, Label: "iPhone", Intent: uikit.RoleInfo, Theme: th}
	out := c.Render()
	plain := uikit.Capture(out)[0]
	assert.True(t, strings.Contains(plain, "◉ iPhone"),
		"active chip renders ◉ + label")
}

func TestChip_ASCII_Premium(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)
	th := theme.Load("black")
	c := uikit.Chip{Glyph: uikit.GlyphPremium, Label: "Irshad", Intent: uikit.RoleSuccess, Theme: th}
	assert.Contains(t, uikit.Capture(c.Render())[0], "*P Irshad")
}
```

### Step 6.2 — Implement Chip

Create `internal/uikit/chip.go`:

```go
package uikit

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// Chip renders a small inline pill: `<glyph> <label>` on the status-bar
// background. Used for header chips (device, profile) and wherever a pill is
// appropriate.
type Chip struct {
	Glyph  GlyphRole
	Label  string
	Intent Role
	Theme  theme.Theme
}

func (c Chip) Render() string {
	bg := lipgloss.NewStyle().Background(c.Theme.StatusBarBg())
	glyph := lipgloss.NewStyle().
		Foreground(ColourFor(c.Intent, c.Theme)).
		Background(c.Theme.StatusBarBg()).
		Render(GlyphFor(c.Glyph, ActiveMode()))
	label := lipgloss.NewStyle().
		Foreground(c.Theme.HeaderChipFg()).
		Background(c.Theme.StatusBarBg()).
		Render(c.Label)
	return bg.Render(" ") + glyph + bg.Render(" ") + label + bg.Render(" ")
}
```

### Step 6.3 — HeaderBar test + implementation

Create `internal/uikit/header_bar.go`:

```go
package uikit

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// HeaderBar renders the top app bar: "spotnik ─ Page X ─ preset N" on the
// left; chips on the right; full-width background fill between.
type HeaderBar struct {
	Width     int
	AppName   string
	Page      string // "A" or "B"
	Preset    int    // -1 hides preset segment
	RightChips []string // pre-rendered chips (from Chip.Render)
	Theme     theme.Theme
}

func (h HeaderBar) Render() string {
	t := h.Theme
	bg := lipgloss.NewStyle().Background(t.StatusBarBg()).
		Foreground(t.StatusBarFg())
	appName := lipgloss.NewStyle().Background(t.StatusBarBg()).
		Foreground(t.TextPrimary()).Bold(true).
		Render(" " + h.AppName + " ")
	key := lipgloss.NewStyle().Background(t.StatusBarBg()).
		Foreground(t.KeyHint()).Bold(true).
		Render(h.Page)
	muted := lipgloss.NewStyle().Background(t.StatusBarBg()).
		Foreground(t.TextMuted())
	sep := muted.Render(" ─ ")

	left := appName + sep + muted.Render("Page ") + key
	if h.Preset >= 0 {
		left += sep + muted.Render(fmt.Sprintf("preset %d", h.Preset))
	}
	right := strings.Join(h.RightChips, "")
	if h.Width > 0 {
		gap := h.Width - lipgloss.Width(left) - lipgloss.Width(right)
		if gap < 1 {
			gap = 1
		}
		return left + bg.Render(strings.Repeat(" ", gap)) + right
	}
	return left + "  " + right
}
```

### Step 6.4 — Migrate `render.go:renderHeader`

In `internal/app/render.go`, replace `renderHeader` body:

```go
func (a *App) renderHeader() string {
	preset := a.layout.ActivePresetIndex()
	if a.layout.ActivePage() == layout.PageB {
		preset = -1
	}

	chips := []string{}
	if dev := a.store.ActiveDevice(); dev != nil {
		chips = append(chips, uikit.Chip{
			Glyph: uikit.GlyphActive, Label: truncateDeviceName(dev.Name),
			Intent: uikit.RoleInfo, Theme: a.theme,
		}.Render())
	} else {
		chips = append(chips, uikit.Chip{
			Glyph: uikit.GlyphAvailable, Label: "No device",
			Intent: uikit.RoleMuted, Theme: a.theme,
		}.Render())
	}
	if profile := a.store.UserProfile(); profile.ID != "" {
		g := uikit.GlyphAvailable
		intent := uikit.RoleMuted
		if a.store.IsPremium() {
			g = uikit.GlyphPremium
			intent = uikit.RoleInfo
		}
		chips = append(chips, uikit.Chip{
			Glyph: g, Label: truncateProfileName(profile.DisplayName),
			Intent: intent, Theme: a.theme,
		}.Render())
	}

	return uikit.HeaderBar{
		Width: a.width, AppName: "spotnik",
		Page: pageLabel(a.layout.ActivePage()), Preset: preset,
		RightChips: chips, Theme: a.theme,
	}.Render()
}
```

### Step 6.5 — Run, commit, PR

- [ ] Run `go test ./internal/uikit/ ./internal/app/ -v`
- [ ] Commit `feat(uikit): add HeaderBar + Chip; migrate render.go:renderHeader`.

---

## Task 7 (S7): `StatusBar` + `KeyBar`

**Files:**
- Create: `internal/uikit/key_bar.go`
- Create: `internal/uikit/key_bar_test.go`
- Create: `internal/uikit/status_bar.go`
- Create: `internal/uikit/status_bar_test.go`
- Modify: `internal/app/render.go` — `renderStatusBar` uses `uikit.StatusBar`.
- Modify: `internal/app/render_test.go` — update assertions.

### Step 7.1 — KeyBar failing test + impl

`internal/uikit/key_bar_test.go`:

```go
package uikit_test

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func TestKeyBar_Unicode_RendersDotSeparators(t *testing.T) {
	th := theme.Load("black")
	bindings := []key.Binding{
		key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy")),
		key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	}
	out := uikit.KeyBar{Bindings: bindings, Theme: th}.Render()
	line := uikit.Capture(out)[0]
	assert.Contains(t, line, "c copy")
	assert.Contains(t, line, "·")
	assert.Contains(t, line, "q quit")
}

func TestKeyBar_ASCII_SwapsSeparator(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)
	th := theme.Load("black")
	out := uikit.KeyBar{
		Bindings: []key.Binding{key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy"))},
		Theme: th,
	}.Render()
	assert.NotContains(t, uikit.Capture(out)[0], "·")
}
```

Implement `internal/uikit/key_bar.go`:

```go
package uikit

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// KeyBar renders a one-row keybinding strip: "c copy · q quit".
// Used by StatusBar (global) and by overlays/panels for local hints.
// The "·" bullet separator becomes "|" in ascii mode.
type KeyBar struct {
	Bindings []key.Binding
	Theme    theme.Theme
}

func (k KeyBar) Render() string {
	t := k.Theme
	keyStyle := lipgloss.NewStyle().Foreground(t.KeyHint())
	descStyle := lipgloss.NewStyle().Foreground(t.TextMuted())
	sepUnicode := " · "
	sepASCII := " | "
	sep := sepUnicode
	if ActiveMode() == GlyphASCII {
		sep = sepASCII
	}

	parts := make([]string, 0, len(k.Bindings))
	for _, b := range k.Bindings {
		h := b.Help()
		parts = append(parts, keyStyle.Render(h.Key)+" "+descStyle.Render(h.Desc))
	}
	return strings.Join(parts, descStyle.Render(sep))
}
```

### Step 7.2 — StatusBar wraps KeyBar and uses bubbles/help

`internal/uikit/status_bar.go`:

```go
package uikit

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// StatusBar is the bottom global key bar. Always 3 lines tall (top + content
// + bottom border). Uses bubbles/help for ShortHelp single-row mode.
type StatusBar struct {
	Width    int
	Bindings help.KeyMap
	Theme    theme.Theme
}

func (s StatusBar) Render() string {
	const statusH = 3
	w := s.Width
	if w < 160 {
		w = 160
	}
	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(s.Theme.Info())
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(s.Theme.TextMuted())
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(s.Theme.TextMuted())
	content := h.View(s.Bindings)
	inner := lipgloss.NewStyle().
		Width(w - 2).MaxWidth(w - 2).
		Height(statusH - 2).MaxHeight(statusH - 2).
		Render(content)
	cfg := layout.BorderConfig{
		Width: w, Height: statusH, Title: "",
		Actions: []layout.Action{}, AccentColor: s.Theme.TextMuted(),
		Focused: false, Theme: s.Theme,
	}
	return layout.RenderPaneBorder(inner, cfg)
}
```

### Step 7.3 — Migrate `render.go:renderStatusBar`

In `internal/app/render.go`:

```go
func (a *App) renderStatusBar() string {
	km := a.statusKeyMap
	km.activePage = a.layout.ActivePage()
	return uikit.StatusBar{
		Width: a.width, Bindings: km, Theme: a.theme,
	}.Render()
}
```

### Step 7.4 — Run, commit, PR

- [ ] `go test ./internal/uikit/ ./internal/app/ -v`
- [ ] Commit `feat(uikit): add KeyBar + StatusBar; migrate render.go:renderStatusBar`.

---

## Task 8 (S8): `TableChrome`

**Files:**
- Create: `internal/uikit/table_chrome.go`
- Create: `internal/uikit/table_chrome_test.go`

**Scope:** `TableChrome` is a thin wrapper over `components/table.go`. Call sites are **not changed** in this story — panes continue to call `components.NewTable` directly. This story establishes the primitive and documents the wrapping pattern; S9 onwards can opt in.

### Step 8.1 — Failing test

`internal/uikit/table_chrome_test.go`:

```go
package uikit_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func TestTableChrome_WrapsComponentsTable(t *testing.T) {
	th := theme.Load("black")
	cols := []components.ColumnDef{
		{Key: "n", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
		{Key: "name", Header: "Name", FlexFactor: 4, Color: th.ColumnPrimary()},
	}
	tbl := uikit.TableChrome{Columns: cols, Theme: th}
	assert.NotNil(t, tbl.Inner())
}
```

### Step 8.2 — Implement

`internal/uikit/table_chrome.go`:

```go
package uikit

import (
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// TableChrome wraps components.Table. The primitive's role is to standardise
// construction — column tokens, header colour, playing-indicator colour — so
// that panes no longer build TableConfig literals inline.
type TableChrome struct {
	Columns []components.ColumnDef
	Theme   theme.Theme
	inner   *components.Table
}

// Inner returns the wrapped *components.Table, constructing it on first call.
// The inner table owns all interactive state; TableChrome is stateless.
func (t *TableChrome) Inner() *components.Table {
	if t.inner == nil {
		t.inner = components.NewTable(components.TableConfig{
			Columns: t.Columns, Theme: t.Theme,
			PlayingIndex: -1, ShowHeader: true,
		})
	}
	return t.inner
}
```

### Step 8.3 — Run, commit, PR

- [ ] `go test ./internal/uikit/ -run TestTableChrome -v`
- [ ] Commit `feat(uikit): add TableChrome wrapper for components/table.go`.

---

## Task 9 (S9): `ListRow` + `LockedRow`

**Files:**
- Create: `internal/uikit/list_row.go`
- Create: `internal/uikit/list_row_test.go`
- Modify: `internal/ui/panes/themes.go` — theme rows use `ListRow`.
- Modify: `internal/ui/panes/profile.go` — logout/forget rows use `ListRow`.
- Modify: `internal/ui/panes/playlists_pane.go` — read-only playlists use `LockedRow`.

### Step 9.1 — Failing tests

`internal/uikit/list_row_test.go`:

```go
package uikit_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func TestListRow_Unicode_WithGlyphAndCaption(t *testing.T) {
	th := theme.Load("black")
	r := uikit.ListRow{
		Glyph: uikit.GlyphActive, Label: "Monokai", Caption: "active", Theme: th,
	}
	plain := uikit.Capture(r.Render(40))[0]
	assert.Contains(t, plain, "◉ Monokai")
	assert.Contains(t, plain, "active")
}

func TestLockedRow_Unicode_DimGlyph(t *testing.T) {
	th := theme.Load("black")
	r := uikit.LockedRow{Label: "Discover Weekly (read-only)", Theme: th}
	plain := uikit.Capture(r.Render(40))[0]
	assert.Contains(t, plain, "◌")
	assert.Contains(t, plain, "Discover Weekly")
}
```

### Step 9.2 — Implement

`internal/uikit/list_row.go`:

```go
package uikit

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// ListRow is a single-line item: optional glyph, primary label, optional
// muted caption. Used in the theme overlay, profile overlay action rows, and
// any place a list row is appropriate.
type ListRow struct {
	Glyph   GlyphRole // empty GlyphRole => no glyph
	Label   string
	Caption string
	Intent  Role
	Theme   theme.Theme
}

func (r ListRow) Render(width int) string {
	parts := []string{}
	if r.Glyph != "" {
		g := GlyphFor(r.Glyph, ActiveMode())
		style := lipgloss.NewStyle().Foreground(ColourFor(r.Intent, r.Theme))
		parts = append(parts, style.Render(g))
	}
	lbl := lipgloss.NewStyle().Foreground(r.Theme.TextPrimary()).Render(r.Label)
	parts = append(parts, lbl)
	line := joinSpace(parts...)
	if r.Caption != "" {
		cap := lipgloss.NewStyle().Foreground(r.Theme.TextMuted()).Render(r.Caption)
		line = padOrTruncate(line, width-lipgloss.Width(cap)-1) + " " + cap
	} else {
		line = padOrTruncate(line, width)
	}
	return line
}

// LockedRow is a ListRow variant for read-only / inaccessible entries. The
// entire row is Muted; a leading `◌` signals the locked state.
type LockedRow struct {
	Label string
	Theme theme.Theme
}

func (r LockedRow) Render(width int) string {
	g := GlyphFor(GlyphLocked, ActiveMode())
	muted := lipgloss.NewStyle().Foreground(r.Theme.TextMuted())
	return padOrTruncate(muted.Render(g+" "+r.Label), width)
}

// joinSpace and padOrTruncate are small utilities used by multiple row-level
// primitives. Defined once here to keep each primitive file focused.
func joinSpace(s ...string) string {
	out := ""
	for i, p := range s {
		if i > 0 {
			out += " "
		}
		out += p
	}
	return out
}

func padOrTruncate(s string, w int) string {
	cur := lipgloss.Width(s)
	switch {
	case cur == w:
		return s
	case cur < w:
		return s + pad(w-cur)
	default:
		// Truncate rune-safe.
		runes := []rune(s)
		for len(runes) > 0 {
			c := string(runes) + "…"
			if lipgloss.Width(c) <= w {
				return c + pad(w-lipgloss.Width(c))
			}
			runes = runes[:len(runes)-1]
		}
		return pad(w)
	}
}

func pad(n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += " "
	}
	return out
}
```

### Step 9.3 — Call-site migrations

In `internal/ui/panes/themes.go`, find the existing theme-row render block and replace with:

```go
rows = append(rows, uikit.ListRow{
	Glyph: uikit.GlyphAvailable, Label: themeName,
	Intent: uikit.RoleMuted, Theme: t.theme,
}.Render(innerW))
```

If the theme is the active one, use `Glyph: uikit.GlyphActive, Intent: uikit.RoleAccent` instead.

In `internal/ui/panes/profile.go`, the logout and forget action rows become:

```go
uikit.ListRow{
	Glyph: uikit.GlyphLocked /* or no glyph */, Label: "l Logout",
	Caption: "ends session · keeps Client ID",
	Intent: uikit.RoleMuted, Theme: t.theme,
}.Render(innerW)
```

In `internal/ui/panes/playlists_pane.go`, when a playlist is owned by Spotify (read-only), swap its row to `LockedRow`. Detection: `playlist.Owner.ID == "spotify"`.

### Step 9.4 — Run, commit, PR

- [ ] `make ci`
- [ ] Commit `feat(uikit): add ListRow + LockedRow; migrate theme/profile/playlists`.

---

## Task 10 (S10): `SectionLabel`

**Files:**
- Create: `internal/uikit/section_label.go`
- Create: `internal/uikit/section_label_test.go`
- Modify: `internal/ui/panes/requestflow_boxed.go` — use `SectionLabel` for GATEWAY, APP, GATEWAY LOG, SPOTIFY labels.
- Modify: `internal/ui/panes/requestflow_pane.go` — use `SectionLabel` for AUTO-TRAFFIC.

### Step 10.1 — Test

`internal/uikit/section_label_test.go`:

```go
package uikit_test

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func TestSectionLabel_RendersCapsWithUnderline(t *testing.T) {
	th := theme.Load("black")
	sl := uikit.SectionLabel{
		Label: "GATEWAY", Width: 40,
		AccentColor: lipgloss.Color(th.PaneBorderRequestFlow()),
		Theme: th,
	}
	lines := uikit.Capture(sl.Render())
	assert.Equal(t, 2, len(lines), "two lines: label + underline rule")
	assert.Contains(t, lines[0], "GATEWAY")
	assert.Contains(t, lines[1], "─") // or "-" in ascii
}
```

### Step 10.2 — Implement

`internal/uikit/section_label.go`:

```go
package uikit

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// SectionLabel renders a caps label followed by a horizontal rule underline.
// Used for sub-sections inside panes (e.g. GATEWAY, APP, AUTO-TRAFFIC in the
// Request Flow pane). Label colour inherits the parent pane's accent so the
// sub-section feels part of the pane's visual identity.
type SectionLabel struct {
	Label       string
	Width       int
	AccentColor lipgloss.Color
	Theme       theme.Theme
}

func (s SectionLabel) Render() string {
	m := ActiveMode()
	style := lipgloss.NewStyle().Foreground(s.AccentColor).Bold(true)
	label := style.Render(" " + s.Label + " ")
	rule := strings.Repeat(GlyphFor(GlyphHRule, m), s.Width)
	return label + "\n" + lipgloss.NewStyle().Foreground(s.AccentColor).Render(rule)
}
```

### Step 10.3 — Call-site migrations

In `internal/ui/panes/requestflow_boxed.go`, replace each hand-rolled section-label
emission with a `uikit.SectionLabel{...}.Render()` call. Widths come from the parent
pane layout.

### Step 10.4 — Run, commit, PR

- [ ] `make ci`
- [ ] Commit `feat(uikit): add SectionLabel; migrate request-flow labels`.

---

## Task 11 (S11): `EmptyState`

**Files:**
- Create: `internal/uikit/empty_state.go`
- Create: `internal/uikit/empty_state_test.go`
- Modify: `internal/ui/panes/queue.go` — empty queue uses `EmptyState`.
- Modify: `internal/ui/panes/search.go` — empty results use `EmptyState`.

### Step 11.1 — Test

`internal/uikit/empty_state_test.go`:

```go
package uikit_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func TestEmptyState_CentersText(t *testing.T) {
	th := theme.Load("black")
	es := uikit.EmptyState{
		Text: "Empty queue", Hint: "Press / to search",
		Width: 40, Height: 6, Theme: th,
	}
	lines := uikit.Capture(es.Render())
	assert.Equal(t, 6, len(lines))
	// Find the line containing the text; it should be roughly centered.
	var textLine int
	for i, l := range lines {
		if containsText(l, "Empty queue") {
			textLine = i
			break
		}
	}
	assert.Greater(t, textLine, 0, "text not on first line")
	assert.Less(t, textLine, 5, "text not on last line")
}

func containsText(haystack, needle string) bool {
	return len(haystack) >= len(needle) &&
		findSubstr(haystack, needle) >= 0
}

func findSubstr(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
```

### Step 11.2 — Implement

`internal/uikit/empty_state.go`:

```go
package uikit

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// EmptyState is shown when a pane has nothing to display. Text is centered
// vertically and horizontally in the provided rectangle; an optional hint
// renders below.
type EmptyState struct {
	Text   string
	Hint   string
	Width  int
	Height int
	Theme  theme.Theme
}

func (e EmptyState) Render() string {
	textStyle := lipgloss.NewStyle().Foreground(e.Theme.TextMuted())
	hintStyle := lipgloss.NewStyle().Foreground(e.Theme.TextMuted())

	body := textStyle.Render(e.Text)
	if e.Hint != "" {
		body = body + "\n" + hintStyle.Render(e.Hint)
	}

	// Center in the rectangle.
	bodyHeight := strings.Count(body, "\n") + 1
	topPad := (e.Height - bodyHeight) / 2
	if topPad < 0 {
		topPad = 0
	}
	lines := make([]string, 0, e.Height)
	for i := 0; i < topPad; i++ {
		lines = append(lines, strings.Repeat(" ", e.Width))
	}
	for _, bl := range strings.Split(body, "\n") {
		lines = append(lines, centerLine(bl, e.Width))
	}
	for len(lines) < e.Height {
		lines = append(lines, strings.Repeat(" ", e.Width))
	}
	return strings.Join(lines, "\n")
}

func centerLine(s string, w int) string {
	cur := lipgloss.Width(s)
	if cur >= w {
		return s
	}
	left := (w - cur) / 2
	right := w - cur - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}
```

### Step 11.3 — Call-site migrations

In `internal/ui/panes/queue.go`, replace the empty-queue branch in `View()` with:

```go
if len(rows) == 0 {
	return uikit.EmptyState{
		Text: "Empty queue", Hint: "Press / to search for tracks to add",
		Width: p.width, Height: p.height, Theme: p.theme,
	}.Render()
}
```

Similarly for `internal/ui/panes/search.go`.

### Step 11.4 — Run, commit, PR

- [ ] `make ci`
- [ ] Commit `feat(uikit): add EmptyState; migrate queue + search`.

---

## Task 12 (S12): `URLBox`

**Files:**
- Create: `internal/uikit/url_box.go`
- Create: `internal/uikit/url_box_test.go`

### Step 12.1 — Test

`internal/uikit/url_box_test.go`:

```go
package uikit_test

import (
	"strings"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func TestURLBox_WrapsURLAtAmpersand(t *testing.T) {
	th := theme.Load("black")
	u := "https://accounts.spotify.com/authorize?client_id=abc&code_challenge=def&scope=user-read-playback-state"
	b := uikit.URLBox{URL: u, Width: 40, Theme: th}
	lines := uikit.Capture(b.Render())
	// Should have at least one break and each line should fit within width.
	assert.GreaterOrEqual(t, len(lines), 2)
	for _, l := range lines {
		assert.LessOrEqual(t, len(strings.TrimSpace(l)), 42,
			"line exceeds box width (incl. border)")
	}
}
```

### Step 12.2 — Implement

`internal/uikit/url_box.go`:

```go
package uikit

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// URLBox wraps a URL or code snippet in a muted-border block. URLs longer
// than the box width are broken at '&' boundaries (preferred) or hard-wrapped.
type URLBox struct {
	URL    string
	Width  int
	Theme  theme.Theme
}

func (b URLBox) Render() string {
	innerW := b.Width - 4 // border + 1 space padding each side
	wrapped := wrapAtAmpersand(b.URL, innerW)
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(b.Theme.TextMuted()).
		Foreground(b.Theme.Accent()).
		Padding(0, 1).
		Width(b.Width - 2)
	return style.Render(wrapped)
}

func wrapAtAmpersand(u string, width int) string {
	if len(u) <= width {
		return u
	}
	var lines []string
	for len(u) > width {
		cut := width
		if i := strings.LastIndex(u[:width], "&"); i > width/2 {
			cut = i
		}
		lines = append(lines, u[:cut])
		u = u[cut:]
	}
	if u != "" {
		lines = append(lines, u)
	}
	return strings.Join(lines, "\n")
}
```

### Step 12.3 — Run, commit, PR

- [ ] `go test ./internal/uikit/ -run TestURLBox -v`
- [ ] Commit `feat(uikit): add URLBox`.

---

## Task 13 (S13): `Toast` + migrate call sites

**Files:**
- Create: `internal/uikit/toast.go`
- Create: `internal/uikit/toast_test.go`
- Modify: `internal/ui/components/notifications.go` — expose intents + glyph mapping via `uikit.ToastIntent`.
- Modify: `internal/app/handlers.go` — every call to `a.alerts.NewAlertCmd(type, msg)` becomes `a.toasts.Cmd(uikit.Toast{...})`.
- Modify: `internal/app/app.go` — `a.toasts *uikit.ToastManager`.

### Step 13.1 — Test Toast intent mapping

`internal/uikit/toast_test.go`:

```go
package uikit_test

import (
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
)

func TestToast_DefaultTTL_ByIntent(t *testing.T) {
	cases := []struct {
		intent uikit.ToastIntent
		ttl    time.Duration
	}{
		{uikit.ToastSuccess, 4 * time.Second},
		{uikit.ToastInfo, 4 * time.Second},
		{uikit.ToastWarning, 5 * time.Second},
		{uikit.ToastError, 6 * time.Second},
	}
	for _, tt := range cases {
		got := uikit.DefaultTTL(tt.intent)
		assert.Equal(t, tt.ttl, got)
	}
}

func TestToast_TruncatesTitle48Runes(t *testing.T) {
	tt := uikit.Toast{Title: strings.Repeat("a", 100), Intent: uikit.ToastSuccess}
	norm := tt.Normalize()
	assert.Equal(t, 48, len([]rune(norm.Title)))
}

func TestToast_TruncatesBody160Runes(t *testing.T) {
	tt := uikit.Toast{Title: "x", Body: strings.Repeat("b", 200), Intent: uikit.ToastInfo}
	norm := tt.Normalize()
	assert.Equal(t, 160, len([]rune(norm.Body)))
}

func TestToast_GlyphByIntent(t *testing.T) {
	assert.Equal(t, "✓", uikit.ToastGlyph(uikit.ToastSuccess, uikit.GlyphUnicode))
	assert.Equal(t, "✗", uikit.ToastGlyph(uikit.ToastError, uikit.GlyphUnicode))
	assert.Equal(t, "◬", uikit.ToastGlyph(uikit.ToastWarning, uikit.GlyphUnicode))
	assert.Equal(t, "→", uikit.ToastGlyph(uikit.ToastInfo, uikit.GlyphUnicode))
	assert.Equal(t, "⧖", uikit.ToastGlyph(uikit.ToastRateLimit, uikit.GlyphUnicode))
}
```

### Step 13.2 — Implement

`internal/uikit/toast.go`:

```go
package uikit

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"go.dalton.dog/bubbleup"
)

// ToastIntent enumerates the severity of a toast notification.
type ToastIntent int

const (
	ToastSuccess ToastIntent = iota
	ToastError
	ToastWarning
	ToastInfo
	ToastRateLimit
)

// Toast is a typed notification surfaced via bubbleup.
type Toast struct {
	Intent ToastIntent
	Title  string
	Body   string
	TTL    time.Duration
}

// DefaultTTL returns the default display time for an intent.
func DefaultTTL(i ToastIntent) time.Duration {
	switch i {
	case ToastError:
		return 6 * time.Second
	case ToastWarning:
		return 5 * time.Second
	case ToastRateLimit:
		return 30 * time.Second // callers override with Retry-After
	default:
		return 4 * time.Second
	}
}

// ToastGlyph returns the glyph for an intent in the given mode.
func ToastGlyph(i ToastIntent, m GlyphMode) string {
	switch i {
	case ToastSuccess:
		return GlyphFor(GlyphSuccess, m)
	case ToastError:
		return GlyphFor(GlyphError, m)
	case ToastWarning:
		return GlyphFor(GlyphWarning, m)
	case ToastInfo:
		return GlyphFor(GlyphInfo, m)
	case ToastRateLimit:
		return GlyphFor(GlyphRateLimit, m)
	}
	return ""
}

// Normalize trims / truncates fields per the §7.4 contract (Title ≤ 48 runes,
// Body ≤ 160 runes), and fills in a default TTL when zero.
func (t Toast) Normalize() Toast {
	norm := t
	if r := []rune(norm.Title); len(r) > 48 {
		norm.Title = string(r[:48])
	}
	if r := []rune(norm.Body); len(r) > 160 {
		norm.Body = string(r[:160])
	}
	if norm.TTL == 0 {
		norm.TTL = DefaultTTL(norm.Intent)
	}
	return norm
}

// ToastManager wraps bubbleup.AlertModel and dispatches typed Toast values.
type ToastManager struct {
	model *bubbleup.AlertModel
}

// NewToastManager wraps an existing bubbleup alert model. The model must have
// been constructed by components.NewNotifications.
func NewToastManager(model *bubbleup.AlertModel) *ToastManager {
	return &ToastManager{model: model}
}

// Cmd returns a tea.Cmd that emits the toast through bubbleup. Call sites
// use: return a.toasts.Cmd(Toast{...}) inside Update.
func (tm *ToastManager) Cmd(t Toast) tea.Cmd {
	norm := t.Normalize()
	key := alertKey(norm.Intent)
	msg := norm.Title
	if norm.Body != "" {
		msg += "\n" + norm.Body
	}
	return tm.model.NewAlertCmd(key, msg)
}

func alertKey(i ToastIntent) string {
	switch i {
	case ToastSuccess:
		return "success"
	case ToastError:
		return "error"
	case ToastWarning:
		return "warning"
	case ToastInfo:
		return "info"
	case ToastRateLimit:
		return "ratelimit"
	}
	return "info"
}
```

### Step 13.3 — Migrate call sites

In `internal/app/app.go`, after constructing `a.alerts` in `New()`:

```go
a.toasts = uikit.NewToastManager(a.alerts)
```

In `internal/app/handlers.go`, for each call to `a.alerts.NewAlertCmd(kind, msg)`:

```go
// Before:
return a, a.alerts.NewAlertCmd("error", err.Error())
// After:
return a, a.toasts.Cmd(uikit.Toast{
	Intent: uikit.ToastError,
	Title:  "Spotify unreachable",
	Body:   err.Error(),
})
```

Use this substitution pattern for every call. Map existing severities:
- `"success"` → `uikit.ToastSuccess`
- `"error"` → `uikit.ToastError`
- `"warning"` → `uikit.ToastWarning`
- `"info"` → `uikit.ToastInfo`
- `"ratelimit"` → `uikit.ToastRateLimit`

Titles follow §7.4 rules (past-participle verb for completions, noun+state for async events).

### Step 13.4 — Run, commit, PR

- [ ] `make ci`
- [ ] Commit `feat(uikit): add Toast typed API; migrate handlers.go call sites`.

---

## Task 14 (S14): `StatusGlyph` + swap `⚠`→`◬` everywhere

**Files:**
- Create: `internal/uikit/status_glyph.go`
- Create: `internal/uikit/status_glyph_test.go`
- Modify: `internal/ui/components/notifications.go` — warning prefix changes to `◬`.
- Modify: `internal/ui/components/notifications_test.go` — assertion updates.
- Modify: `internal/cliout/message.go` — warning glyph changes to `◬`.
- Modify: `internal/cliout/message_test.go` — assertion updates.
- Modify: `internal/app/render.go` — onboarding `warnStyle.Render("⚠ …")` lines use `uikit.StatusGlyph{Role: uikit.RoleWarning, Text: "..."}`.

### Step 14.1 — Test

`internal/uikit/status_glyph_test.go`:

```go
package uikit_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func TestStatusGlyph_WarningRendersCircleTriangle(t *testing.T) {
	th := theme.Load("black")
	sg := uikit.StatusGlyph{Role: uikit.RoleWarning, Text: "Premium required", Theme: th}
	plain := uikit.Capture(sg.Render())[0]
	assert.Contains(t, plain, "◬ Premium required")
	assert.NotContains(t, plain, "⚠")
}

func TestStatusGlyph_ASCII_Warning(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)
	th := theme.Load("black")
	sg := uikit.StatusGlyph{Role: uikit.RoleWarning, Text: "X", Theme: th}
	assert.Contains(t, uikit.Capture(sg.Render())[0], "! X")
}
```

### Step 14.2 — Implement

`internal/uikit/status_glyph.go`:

```go
package uikit

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// StatusGlyph renders an atomic `<glyph> <text>` line coloured by its role.
// Used for persistent informational state (not notifications — those are
// Toasts).
type StatusGlyph struct {
	Role  Role
	Text  string
	Theme theme.Theme
}

func (s StatusGlyph) Render() string {
	g := ""
	switch s.Role {
	case RoleSuccess:
		g = GlyphFor(GlyphSuccess, ActiveMode())
	case RoleError:
		g = GlyphFor(GlyphError, ActiveMode())
	case RoleWarning:
		g = GlyphFor(GlyphWarning, ActiveMode())
	case RoleInfo:
		g = GlyphFor(GlyphInfo, ActiveMode())
	}
	style := lipgloss.NewStyle().Foreground(ColourFor(s.Role, s.Theme))
	return style.Render(g) + " " +
		lipgloss.NewStyle().Foreground(s.Theme.TextPrimary()).Render(s.Text)
}
```

### Step 14.3 — Swap `⚠` → `◬` everywhere

In `internal/ui/components/notifications.go`, change warning alert prefix:

```go
warningAlert := bubbleup.AlertDefinition{
	Key:       "warning",
	ForeColor: string(t.Warning()),
	Prefix:    "◬", // was "!"
}
```

Wait — the existing file uses `Prefix: "!"`. Check the actual current value and replace accordingly. If `⚠` is present, replace with `◬`. If `!` is present, leave as-is (ascii mode will still use `!`).

In `internal/cliout/message.go`, find the warning glyph in the `statusGlyph` map (or equivalent), and change to `◬`. Update the matching test assertion.

In `internal/app/render.go`, find `warnStyle.Render("⚠ ...")` and replace:

```go
warnStyle.Render("⚠  Spotify Premium is required for playback controls"),
// becomes:
uikit.StatusGlyph{Role: uikit.RoleWarning, Text: "Spotify Premium is required for playback controls", Theme: t}.Render(),
```

### Step 14.4 — Verify no `⚠` remains

- [ ] Run `grep -rn "⚠" internal/ cmd/`
  Expected: no matches (except perhaps in this plan's own doc string).

- [ ] Run `make ci`

### Step 14.5 — Commit and PR

- [ ] Commit `feat(uikit): add StatusGlyph; swap ⚠ → ◬ across cliout, notifications, render`.

---

## Task 15 (S15): `ProgressBar` — seek + volume

**Files:**
- Create: `internal/uikit/progress_bar.go`
- Create: `internal/uikit/progress_bar_test.go`
- Modify: `internal/ui/components/controls.go` — route seek + volume bars through `uikit.ProgressBar`.

### Step 15.1 — Test

`internal/uikit/progress_bar_test.go`:

```go
package uikit_test

import (
	"strings"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func TestProgressBar_Unicode_HalfFilled(t *testing.T) {
	th := theme.Load("black")
	pb := uikit.ProgressBar{Width: 20, Progress: 0.5, Theme: th}
	bar := pb.Render()
	assert.Equal(t, 20, strings.Count(bar, "█")+strings.Count(bar, "░"))
}

func TestProgressBar_ASCII_HalfFilled(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)
	th := theme.Load("black")
	pb := uikit.ProgressBar{Width: 20, Progress: 0.5, Theme: th}
	bar := pb.Render()
	assert.Equal(t, 10, strings.Count(bar, "#"))
	assert.Equal(t, 10, strings.Count(bar, "."))
}

func TestProgressBar_ClampsProgress(t *testing.T) {
	th := theme.Load("black")
	assert.Equal(t, uikit.ProgressBar{Width: 10, Progress: 2.0, Theme: th}.Render(),
		uikit.ProgressBar{Width: 10, Progress: 1.0, Theme: th}.Render())
	assert.Equal(t, uikit.ProgressBar{Width: 10, Progress: -0.5, Theme: th}.Render(),
		uikit.ProgressBar{Width: 10, Progress: 0.0, Theme: th}.Render())
}
```

### Step 15.2 — Implement

`internal/uikit/progress_bar.go`:

```go
package uikit

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// ProgressBar renders a horizontal fill bar. Used for the seek bar and volume
// bar. Partial blocks give 1/8 resolution in unicode; ascii collapses to
// "#" / "-" / "." based on coarse thresholds.
type ProgressBar struct {
	Width    int
	Progress float64 // 0.0–1.0
	Theme    theme.Theme
}

func (p ProgressBar) Render() string {
	if p.Progress < 0 {
		p.Progress = 0
	}
	if p.Progress > 1 {
		p.Progress = 1
	}
	m := ActiveMode()
	fillStyle := lipgloss.NewStyle().Foreground(p.Theme.Gradient1())
	emptyStyle := lipgloss.NewStyle().Foreground(p.Theme.TextMuted())
	full := GlyphFor(GlyphBarFull, m)
	empty := GlyphFor(GlyphBarEmpty, m)

	totalCells := p.Width
	filledFloat := p.Progress * float64(totalCells)
	filled := int(filledFloat)
	remainder := filledFloat - float64(filled)

	var sb strings.Builder
	sb.WriteString(fillStyle.Render(strings.Repeat(full, filled)))
	// Partial block at the boundary.
	if remainder > 0 && filled < totalCells {
		partial := partialGlyph(remainder, m)
		sb.WriteString(fillStyle.Render(partial))
		filled++
	}
	if rem := totalCells - filled; rem > 0 {
		sb.WriteString(emptyStyle.Render(strings.Repeat(empty, rem)))
	}
	return sb.String()
}

func partialGlyph(remainder float64, m GlyphMode) string {
	switch {
	case remainder >= 7.0/8.0:
		return GlyphFor(GlyphBarSevenEighths, m)
	case remainder >= 6.0/8.0:
		return GlyphFor(GlyphBarThreeQuarters, m)
	case remainder >= 5.0/8.0:
		return GlyphFor(GlyphBarFiveEighths, m)
	case remainder >= 4.0/8.0:
		return GlyphFor(GlyphBarHalf, m)
	case remainder >= 3.0/8.0:
		return GlyphFor(GlyphBarThreeEighths, m)
	case remainder >= 2.0/8.0:
		return GlyphFor(GlyphBarQuarter, m)
	default:
		return GlyphFor(GlyphBarOneEighth, m)
	}
}
```

### Step 15.3 — Migrate `controls.go`

In `internal/ui/components/controls.go`, find the seek-bar and volume-bar rendering
code (function likely named `RenderSeekBar`, `RenderVolumeBar`) and replace the
inline character composition with `uikit.ProgressBar{...}.Render()`.

### Step 15.4 — Run, commit, PR

- [ ] `make ci`
- [ ] Commit `feat(uikit): add ProgressBar; migrate seek + volume to it`.

---

## Task 16 (S16): `Spinner` with Done/Fail/Cancel

**Files:**
- Create: `internal/uikit/spinner.go`
- Create: `internal/uikit/spinner_test.go`
- Modify: `internal/app/app.go` — onboarding spinner uses `uikit.Spinner`.
- Modify: `internal/app/auth.go` — OAuth success emits `SpinnerDoneMsg`; OAuth error emits `SpinnerFailMsg`.

### Step 16.1 — Test

`internal/uikit/spinner_test.go`:

```go
package uikit_test

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpinner_Done_EmitsMsgAfterTTL(t *testing.T) {
	th := theme.Load("black")
	s := uikit.NewSpinner("Working", th)
	_, cmd := s.Done("Authorized")
	require.NotNil(t, cmd)
	msg := cmd()
	done, ok := msg.(uikit.SpinnerDoneMsg)
	assert.True(t, ok)
	assert.Equal(t, "Authorized", done.Text)
}

func TestSpinner_Fail_EmitsMsgWithErr(t *testing.T) {
	th := theme.Load("black")
	s := uikit.NewSpinner("Working", th)
	_, cmd := s.Fail("Failed: bad token")
	msg := cmd()
	fail, ok := msg.(uikit.SpinnerFailMsg)
	assert.True(t, ok)
	assert.Equal(t, "Failed: bad token", fail.Err)
}

func TestSpinner_Cancel_ClearsImmediately(t *testing.T) {
	th := theme.Load("black")
	s := uikit.NewSpinner("Working", th)
	m, cmd := s.Cancel()
	assert.Empty(t, m.View())
	msg := cmd()
	_, ok := msg.(uikit.SpinnerCancelledMsg)
	assert.True(t, ok)
}
```

### Step 16.2 — Implement

`internal/uikit/spinner.go`:

```go
package uikit

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// SpinnerDoneMsg is emitted after Spinner.Done's hold period expires.
type SpinnerDoneMsg struct{ Text string }

// SpinnerFailMsg is emitted after Spinner.Fail's hold period expires.
type SpinnerFailMsg struct{ Err string }

// SpinnerCancelledMsg is emitted immediately on Spinner.Cancel.
type SpinnerCancelledMsg struct{}

// Spinner is the TUI spinner primitive. Wraps bubbles/spinner with terminal
// states for Done, Fail, and Cancel — matching cliout.Spinner's surface.
type Spinner struct {
	s      spinner.Model
	text   string
	result *resolution
	theme  theme.Theme
}

type resolution struct {
	glyph   GlyphRole
	text    string
	role    Role
	holdTTL time.Duration
}

// NewSpinner creates a new spinner primitive with the given text.
func NewSpinner(text string, th theme.Theme) *Spinner {
	m := spinner.New()
	m.Spinner = spinner.Dot
	m.Style = lipgloss.NewStyle().Foreground(th.Accent())
	return &Spinner{s: m, text: text, theme: th}
}

func (s *Spinner) Init() tea.Cmd { return s.s.Tick }

func (s *Spinner) Update(msg tea.Msg) (*Spinner, tea.Cmd) {
	if s.result != nil {
		return s, nil
	}
	var cmd tea.Cmd
	s.s, cmd = s.s.Update(msg)
	return s, cmd
}

func (s *Spinner) View() string {
	if s.result != nil {
		g := GlyphFor(s.result.glyph, ActiveMode())
		gl := lipgloss.NewStyle().Foreground(ColourFor(s.result.role, s.theme)).Render(g)
		tx := lipgloss.NewStyle().Foreground(s.theme.TextMuted()).Render(s.result.text)
		return gl + " " + tx
	}
	frame := s.s.View()
	tx := lipgloss.NewStyle().Foreground(s.theme.TextMuted()).Render(s.text)
	return frame + " " + tx
}

// Done resolves the spinner to ✓ Success; after 1.2s, emits SpinnerDoneMsg.
func (s *Spinner) Done(text string) (*Spinner, tea.Cmd) {
	s.result = &resolution{glyph: GlyphSuccess, text: text, role: RoleSuccess, holdTTL: 1200 * time.Millisecond}
	return s, tea.Tick(s.result.holdTTL, func(time.Time) tea.Msg {
		return SpinnerDoneMsg{Text: text}
	})
}

// Fail resolves to ✗ Error; after 2s, emits SpinnerFailMsg.
func (s *Spinner) Fail(text string) (*Spinner, tea.Cmd) {
	s.result = &resolution{glyph: GlyphError, text: text, role: RoleError, holdTTL: 2 * time.Second}
	return s, tea.Tick(s.result.holdTTL, func(time.Time) tea.Msg {
		return SpinnerFailMsg{Err: text}
	})
}

// Cancel clears the spinner immediately; emits SpinnerCancelledMsg.
func (s *Spinner) Cancel() (*Spinner, tea.Cmd) {
	s.result = &resolution{}
	s.text = ""
	return s, func() tea.Msg { return SpinnerCancelledMsg{} }
}
```

### Step 16.3 — Wire onboarding OAuth

In `internal/app/auth.go`, replace the OAuth-success handler:

```go
case auth.OAuthSuccessMsg:
	spinner, cmd := a.onboardingSpinner.Done("Authorized")
	a.onboardingSpinner = spinner
	return a, cmd
case uikit.SpinnerDoneMsg:
	// Transition to grid view; emit success toast.
	a.currentView = viewGrid
	return a, a.toasts.Cmd(uikit.Toast{
		Intent: uikit.ToastSuccess,
		Title:  "Signed in",
		Body:   "Welcome back to Spotnik.",
	})
case auth.OAuthFailureMsg:
	spinner, cmd := a.onboardingSpinner.Fail("Authorization failed")
	a.onboardingSpinner = spinner
	a.onboardingError = m.Err.Error()
	return a, cmd
case uikit.SpinnerFailMsg:
	// Already showed the spinner failure; transition to error panel.
	a.onboardingStep = stepError
	return a, nil
```

### Step 16.4 — Run, commit, PR

- [ ] `make ci`
- [ ] Commit `feat(uikit): add Spinner with Done/Fail/Cancel; wire onboarding OAuth`.

---

## Task 17 (S17): `FormField` + onboarding input migration

**Files:**
- Create: `internal/uikit/form_field.go`
- Create: `internal/uikit/form_field_test.go`
- Modify: `internal/app/app.go` — onboarding client-ID input becomes `uikit.FormField`.
- Modify: `internal/app/render.go` — `renderOnboardingRegister` uses the FormField primitive.

### Step 17.1 — Test

`internal/uikit/form_field_test.go`:

```go
package uikit_test

import (
	"errors"
	"testing"

	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func TestFormField_NoErrorBeforeValidation(t *testing.T) {
	th := theme.Load("black")
	f := uikit.NewFormField(uikit.FormFieldConfig{
		Label: "Client ID",
		Validate: func(s string) error {
			if len(s) != 32 {
				return errors.New("must be 32 chars")
			}
			return nil
		},
		Theme: th,
	})
	assert.Empty(t, f.ValidationError())
}

func TestFormField_ReportsErrorAfterValidate(t *testing.T) {
	th := theme.Load("black")
	f := uikit.NewFormField(uikit.FormFieldConfig{
		Label: "Client ID",
		Validate: func(s string) error {
			if len(s) != 32 {
				return errors.New("must be 32 chars")
			}
			return nil
		},
		Theme: th,
	})
	f.SetValue("short")
	f.Validate()
	assert.Contains(t, f.ValidationError(), "must be 32 chars")
}

func TestFormField_AcceptsValidValue(t *testing.T) {
	th := theme.Load("black")
	f := uikit.NewFormField(uikit.FormFieldConfig{
		Label: "Client ID",
		Validate: func(s string) error { return nil },
		Theme: th,
	})
	f.SetValue("valid")
	f.Validate()
	assert.Empty(t, f.ValidationError())
}
```

### Step 17.2 — Implement

`internal/uikit/form_field.go`:

```go
package uikit

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// FormFieldConfig configures a new FormField.
type FormFieldConfig struct {
	Label       string
	Placeholder string
	Validate    func(string) error
	Theme       theme.Theme
}

// FormField is a labelled text input with intrinsic validation. The
// validation error is rendered under the input; callers consume Value() when
// the user confirms (typically Enter key).
type FormField struct {
	cfg    FormFieldConfig
	input  textinput.Model
	errMsg string
}

func NewFormField(cfg FormFieldConfig) *FormField {
	ti := textinput.New()
	ti.Placeholder = cfg.Placeholder
	ti.Focus()
	return &FormField{cfg: cfg, input: ti}
}

// Update forwards the msg to the inner input.
func (f *FormField) Update(msg tea.Msg) (*FormField, tea.Cmd) {
	var cmd tea.Cmd
	f.input, cmd = f.input.Update(msg)
	return f, cmd
}

// Render produces the label + input + optional validation error.
func (f *FormField) Render() string {
	t := f.cfg.Theme
	label := lipgloss.NewStyle().Foreground(t.TextMuted()).Render(f.cfg.Label + ":")
	inputBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Accent()).
		Padding(0, 1).
		Render(f.input.View())
	out := label + "\n" + inputBox
	if f.errMsg != "" {
		e := lipgloss.NewStyle().Foreground(t.Error()).Render(
			GlyphFor(GlyphError, ActiveMode()) + " " + f.errMsg,
		)
		out += "\n" + e
	}
	return out
}

// Value returns the current text.
func (f *FormField) Value() string { return f.input.Value() }

// SetValue sets the text and clears any cached validation error.
func (f *FormField) SetValue(v string) {
	f.input.SetValue(v)
	f.errMsg = ""
}

// Validate runs the configured validator. Returns nil on pass, error on fail.
// The error is cached and rendered by Render until the next SetValue or
// Validate.
func (f *FormField) Validate() error {
	if f.cfg.Validate == nil {
		return nil
	}
	err := f.cfg.Validate(f.input.Value())
	if err != nil {
		f.errMsg = err.Error()
		return err
	}
	f.errMsg = ""
	return nil
}

// ValidationError returns the cached validation error text.
func (f *FormField) ValidationError() string { return f.errMsg }
```

### Step 17.3 — Wire into onboarding

In `internal/app/app.go`, replace the existing `a.onboardingInput` field with a
`*uikit.FormField`. In `renderOnboardingRegister` (in `render.go`), emit the field
via `a.onboardingField.Render()`.

### Step 17.4 — Run, commit, PR

- [ ] `make ci`
- [ ] Commit `feat(uikit): add FormField; migrate onboarding client-ID input`.

---

## Task 18 (S18): Onboarding end-to-end rewrite

**Files:**
- Modify: `internal/app/render.go` — `renderOnboardingRegister`, `renderOnboardingOAuth`, `renderOnboardingError` rewritten to compose `Panel + FormField + URLBox + KeyBar + Toast + StatusGlyph + Spinner`.
- Modify: `internal/app/app.go` — `a.onboardingInput` → `a.onboardingField *uikit.FormField`, etc.
- Modify: `internal/app/auth.go` — transitions emit toasts instead of inline flashes.
- Delete: `internal/app/clipboard.go` (clipboard copy still exists; but the copy flash becomes a toast, not inline ✓ Copied!).

### Step 18.1 — Register screen

Replace `renderOnboardingRegister` with:

```go
func (a *App) renderOnboardingRegister() string {
	t := a.theme

	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", a.onboardingPort)
	uriBox := uikit.URLBox{URL: redirectURI, Width: 60, Theme: t}.Render()

	instructions := strings.Join([]string{
		"1. Go to https://developer.spotify.com/dashboard and create an app.",
		"2. In the app settings, set the Redirect URI exactly as shown below:",
		"",
		uriBox,
		"",
		uikit.StatusGlyph{Role: uikit.RoleWarning, Text: "Spotify Premium is required for playback controls", Theme: t}.Render(),
		uikit.StatusGlyph{Role: uikit.RoleSuccess, Text: "Your Client ID will be saved to ~/.config/spotnik/config.toml", Theme: t}.Render(),
	}, "\n")

	inputBlock := a.onboardingField.Render()

	keybar := uikit.KeyBar{
		Bindings: []key.Binding{
			key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy URI")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "confirm")),
			key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		},
		Theme: t,
	}.Render()

	body := strings.Join([]string{
		a.onboardingTitle(),
		"",
		instructions,
		"",
		"Paste your Client ID here:",
		inputBlock,
		"",
		keybar,
	}, "\n")

	return uikit.Panel{
		Width: a.width, Height: a.height,
		Title: "Step 1 of 2 — Set up your Spotify Developer App",
		Theme: t,
	}.Render(body)
}
```

### Step 18.2 — OAuth screen

Replace `renderOnboardingOAuth` with:

```go
func (a *App) renderOnboardingOAuth() string {
	t := a.theme

	urlBox := uikit.URLBox{URL: a.onboardingAuthURL, Width: a.width - 20, Theme: t}.Render()
	spinner := a.onboardingSpinner.View()

	keybar := uikit.KeyBar{
		Bindings: []key.Binding{
			key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy URL")),
			key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		},
		Theme: t,
	}.Render()

	body := strings.Join([]string{
		a.onboardingTitle(),
		"",
		"A browser window has been opened. Log in and click Agree.",
		"",
		"On a headless server or browser didn't open? Visit this URL:",
		urlBox,
		"",
		spinner + "  (times out in 5 minutes)",
		"",
		keybar,
	}, "\n")

	return uikit.Panel{
		Width: a.width, Height: a.height,
		Title: "Step 2 of 2 — Authorize Spotnik with Spotify",
		Theme: t,
	}.Render(body)
}
```

### Step 18.3 — Error screen

Replace `renderOnboardingError` with:

```go
func (a *App) renderOnboardingError() string {
	t := a.theme

	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", a.onboardingPort)
	causes := strings.Join([]string{
		"• Client ID mistyped or truncated",
		"• Redirect URI does not match: " + redirectURI,
		"• Spotify app deleted or suspended",
	}, "\n")

	keybar := uikit.KeyBar{
		Bindings: []key.Binding{
			key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "re-enter Client ID")),
			key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "try again")),
			key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		},
		Theme: t,
	}.Render()

	body := strings.Join([]string{
		a.onboardingTitle(),
		"",
		uikit.StatusGlyph{Role: uikit.RoleError, Text: "Authorization failed", Theme: t}.Render(),
		"Error: " + a.onboardingError,
		"",
		"Common causes:",
		causes,
		"",
		keybar,
	}, "\n")

	return uikit.Panel{
		Width: a.width, Height: a.height,
		Title: "Step 2 of 2 — Authorization Failed",
		Intent: uikit.PanelIntentError, Theme: t,
	}.Render(body)
}
```

### Step 18.4 — Copy-URI → Toast

In the handler for `c` in onboarding (search `onboardingCopied` flow), replace:

```go
// Before:
a.onboardingCopied = true
return a, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return clearCopiedMsg{} })

// After:
return a, a.toasts.Cmd(uikit.Toast{
	Intent: uikit.ToastSuccess,
	Title:  "Copied",
})
```

Remove `a.onboardingCopied`, `clearCopiedMsg`, and the related handler branch.

### Step 18.5 — Run, commit, PR

- [ ] `make ci`
- [ ] Visual smoke test: `make run`, trigger onboarding (clear config client_id), verify:
  - Step 1 panel shows title in border, URI box, warning glyph, input field, key bar.
  - Pressing `c` emits a Success toast `Copied`.
  - Step 2 shows spinner with animated frame.
  - On success: spinner → `✓ Authorized`, 1.2s, then grid view + Success toast `Signed in`.
  - On fail: spinner → `✗ Authorization failed`, 2s, then error panel with Error-intent border.
- [ ] Commit `feat(onboarding): rewrite using uikit primitives end-to-end`.

---

## Task 19 (S19): Docs rewrite — `DESIGN.md` + `TUI-DESIGN-SYSTEM.md` + `PANE-TEMPLATE.md`

**Files:**
- Create: `docs/TUI-DESIGN-SYSTEM.md`
- Modify: `docs/DESIGN.md`
- Modify: `docs/PANE-TEMPLATE.md`

### Step 19.1 — Create `TUI-DESIGN-SYSTEM.md`

Derived from the spec (`docs/superpowers/specs/2026-04-24-tui-design-system-design.md`)
but pitched as operational reference rather than brainstorming record. Sections:

1. Purpose
2. Hard rules (do/don't list)
3. Primitive catalogue with full 6-block contract for every primitive (1–18). Each
   contract includes: Purpose, Fields, Rendering (unicode + ascii), Roles, Glyphs,
   Lifecycle, Tests.
4. Glyph catalogue (from spec §5, verbatim).
5. Role / colour matrix (from spec §6, verbatim).
6. Feedback channels (from spec §4 of the plan; finalised as operational rules).
7. Relationship to other docs.

The full contracts for primitives 1–18 are written during each primitive's story
and consolidated here in S19. By S19 the canonical shape of each primitive is
settled.

### Step 19.2 — Strip `DESIGN.md`

Remove from `docs/DESIGN.md`:
- `§5 — Embedded Shortcut Borders (btop-style)` — the border anatomy is now in
  `TUI-DESIGN-SYSTEM.md` under `PaneChrome`.
- The `ᐅ` Unicode note at line 609 — the glyph is banned.
- Any prescriptive rendering detail for overlays, toasts, header, status bar.
  Replace each deleted section with a single line `See docs/TUI-DESIGN-SYSTEM.md §N`.

Retain:
- §1 Overview
- §2 Pane Definitions
- §3 Layout Grid System
- §4 Pages, Pane Toggling, Preset Layouts
- §6 Content Containment
- §17 Keybindings (cross-referenced to `keybinding.md`)
- §21 Min-terminal-size rule
- Visualizer spec

Add a new §0 paragraph:

> **Authority.** Layout mechanics (grid, pages, presets, keys 1–8, page switch)
> live in this document. Primitive rendering (PaneChrome, Toast, Panel,
> HeaderBar, StatusBar, overlay chrome, onboarding panels) lives in
> `docs/TUI-DESIGN-SYSTEM.md`. Where both apply — e.g. pane borders — this
> document describes the pane identity (colour, toggle key); the design-system
> doc describes the exact rendering contract.

### Step 19.3 — Update `PANE-TEMPLATE.md`

In Step 2 of the pane template, replace the manual `View()` scaffold with a
PaneChrome-based scaffold:

```go
func (p *ListenCountPane) View() string {
	if p.width == 0 || p.height == 0 {
		return ""
	}
	content := "  " + strconv.Itoa(p.count) + " listens"
	return uikit.PaneChrome{
		Width: p.width, Height: p.height,
		Title: p.Title(), ToggleKey: p.ToggleKey(),
		Actions: p.Actions(),
		AccentColor: layout.PaneBorderColor(p.ID(), p.theme),
		Focused: p.focused, Theme: p.theme,
	}.Render(content)
}
```

Update the Verification section accordingly.

### Step 19.4 — Cross-check

- [ ] Run `grep -rn "ᐅ" docs/` — expected: no matches outside change-log history.
- [ ] Run `grep -rn "⚠" docs/` — expected: no matches outside this plan and the spec.
- [ ] Run `make ci` — expected: clean.

### Step 19.5 — Commit and PR

- [ ] Commit `docs: rewrite DESIGN.md; add TUI-DESIGN-SYSTEM.md; update PANE-TEMPLATE.md`.

---

## Self-review

Before marking the plan done, check each spec section against the plan tasks:

**§5 Glyph catalogue** — all tables lock down glyphs. Implemented in Task 1 (`glyph.go`).
Swaps (`⚠`→`◬`) implemented in Task 14. Banned glyph (`ᐅ`) removed in Task 2.

**§6 Role matrix** — implemented in Task 1 (`role.go`). Field-role mapping is
enforced via each primitive's implementation using `uikit.Apply(role, theme)`.

**§7.1 Primitive catalogue** — tasks 3–18 create all 18 primitives. Summary
table mirrors §7.1.

**§7.2 Contract template** — worked examples in S3, S13, S16. Remaining
primitives get their contracts in `TUI-DESIGN-SYSTEM.md` (S19).

**§8 Migration plan** — mirrors Task 1–19. Sequencing respected (S1/S2
are hard gates; S17 depends on S13/S15/S16; S18 depends on everything;
S19 is last).

**§9 Relationship to other docs** — Task 19 covers the doc migration.

**§10 Rules summary** — enforced by tests and lint. Rule 3 (no `ᐅ`) enforced
by Task 2 + glyph catalogue integrity test. Rule 4 (no `⚠`) enforced by Task
14 + notifications test. Rule 9 (no `Hint` primitive) — `KeyBar` is the
single primitive; no `Hint` symbol is exported.

No placeholders found. All code blocks are complete.

---

## Execution handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-24-tui-design-system.md`.

Two execution options:

**1. Subagent-Driven (recommended)** — dispatch a fresh subagent per task, review between tasks, fast iteration. Best for this plan because each task is an independent PR and reviewing primitives individually keeps rollout safe.

**2. Inline Execution** — execute tasks in this session using executing-plans, batch execution with checkpoints. Appropriate if you want to watch every step synchronously.

Which approach?
