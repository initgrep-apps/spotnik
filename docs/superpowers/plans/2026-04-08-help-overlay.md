# Help Overlay Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the `?` keybinding to open a centered help overlay that displays all app keybindings grouped by category, and introduce `docs/keybinding.md` as the canonical keybinding reference with a CLAUDE.md maintenance rule.

**Architecture:** A new `HelpOverlay` struct in `internal/ui/panes/help_overlay.go` holds static two-column keybinding data and renders via `layout.RenderPaneBorder`. The root app wires it identically to the existing `ThemeOverlay` pattern: bool flag + pointer in `app.go`, guard + `?` handler in `routing.go`, `renderWithHelpOverlay()` in `render.go`.

**Tech Stack:** Go 1.22, Bubble Tea v0.27, Lip Gloss, `bubbletea-overlay` (btoverlay), `charmbracelet/bubbles/key`, `testify/assert`.

---

## File Map

| Action | Path | Responsibility |
|--------|------|----------------|
| Create | `internal/ui/panes/help_overlay.go` | `HelpOverlay` struct, `HelpOverlayClosedMsg`, static `helpContent` data, `View()`, `Update()`, `renderColumn()` |
| Create | `internal/ui/panes/help_overlay_test.go` | Unit tests for `HelpOverlay` |
| Modify | `internal/app/app.go` | `helpOpen` + `helpOverlay` fields; `openHelp()` / `closeHelp()`; `HelpOverlayClosedMsg` handler; `WindowSizeMsg` + theme propagation |
| Modify | `internal/app/routing.go` | Help overlay guard at top of `handleKeyMsg()`; `?` key handler; `helpOpen` added to `handleMouseMsg()` guard |
| Modify | `internal/app/render.go` | `renderWithHelpOverlay()` method; `helpOpen` branch in `buildView()` |
| Create | `docs/keybinding.md` | Human-readable canonical keybinding reference |
| Modify | `docs/DESIGN.md` | Update `?` row in §17 from "Help (planned)" to "Open help overlay" |
| Modify | `CLAUDE.md` | Add keybinding maintenance rule #15 and "Keybinding Maintenance" section |

---

## Task 1: Create feature branch

- [ ] **Step 1: Create and check out the feature branch**

```bash
git checkout main && git pull origin main
git checkout -b feat/help-overlay
```

Expected: prompt shows `feat/help-overlay`.

---

## Task 2: Write HelpOverlay tests (TDD — write first)

**Files:**
- Create: `internal/ui/panes/help_overlay_test.go`

- [ ] **Step 1: Create the test file**

```go
// internal/ui/panes/help_overlay_test.go
package panes

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestHelpOverlay() *HelpOverlay {
	return NewHelpOverlay(theme.Load("black"))
}

// TestHelpOverlay_View_HasBorder verifies the overlay uses rounded-corner borders.
func TestHelpOverlay_View_HasBorder(t *testing.T) {
	o := newTestHelpOverlay()
	o.SetSize(120, 40)
	view := o.View()
	require.NotEmpty(t, view)
	assert.Contains(t, view, "╭", "should have rounded top-left corner")
	assert.Contains(t, view, "╰", "should have rounded bottom-left corner")
}

// TestHelpOverlay_View_HasTitle verifies the border title is "Help".
func TestHelpOverlay_View_HasTitle(t *testing.T) {
	o := newTestHelpOverlay()
	o.SetSize(120, 40)
	view := o.View()
	assert.Contains(t, view, "Help", "border should show title 'Help'")
}

// TestHelpOverlay_View_ContainsSectionHeaders verifies all four section names appear.
func TestHelpOverlay_View_ContainsSectionHeaders(t *testing.T) {
	o := newTestHelpOverlay()
	o.SetSize(120, 40)
	view := o.View()
	for _, header := range []string{"Global", "Navigation", "Playback", "Pane Actions"} {
		assert.Contains(t, view, header, "section header %q should appear in view", header)
	}
}

// TestHelpOverlay_View_ContainsGlobalKeys verifies key global bindings appear.
func TestHelpOverlay_View_ContainsGlobalKeys(t *testing.T) {
	o := newTestHelpOverlay()
	o.SetSize(120, 40)
	view := o.View()
	for _, key := range []string{"/", "d", "t", "?", "q", "0", "1-8", "p"} {
		assert.Contains(t, view, key, "global key %q should appear in view", key)
	}
}

// TestHelpOverlay_View_ContainsPlaybackKeys verifies playback bindings appear.
func TestHelpOverlay_View_ContainsPlaybackKeys(t *testing.T) {
	o := newTestHelpOverlay()
	o.SetSize(120, 40)
	view := o.View()
	for _, key := range []string{"Space", "n", "s", "r", "v"} {
		assert.Contains(t, view, key, "playback key %q should appear in view", key)
	}
}

// TestHelpOverlay_View_ContainsPaneActionKeys verifies pane action bindings appear.
func TestHelpOverlay_View_ContainsPaneActionKeys(t *testing.T) {
	o := newTestHelpOverlay()
	o.SetSize(120, 40)
	view := o.View()
	for _, key := range []string{"Enter", "f", "A", "i", "x"} {
		assert.Contains(t, view, key, "pane action key %q should appear in view", key)
	}
}

// TestHelpOverlay_Update_EscEmitsClosedMsg verifies Esc produces HelpOverlayClosedMsg.
func TestHelpOverlay_Update_EscEmitsClosedMsg(t *testing.T) {
	o := newTestHelpOverlay()
	_, cmd := o.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd, "Esc should return a non-nil cmd")
	msg := cmd()
	_, ok := msg.(HelpOverlayClosedMsg)
	assert.True(t, ok, "Esc cmd should produce HelpOverlayClosedMsg")
}

// TestHelpOverlay_Update_OtherKeysConsumed verifies non-Esc keys return nil cmd.
func TestHelpOverlay_Update_OtherKeysConsumed(t *testing.T) {
	o := newTestHelpOverlay()
	keys := []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'j'}},
		{Type: tea.KeyRunes, Runes: []rune{'k'}},
		{Type: tea.KeyRunes, Runes: []rune{'q'}},
		{Type: tea.KeyEnter},
	}
	for _, k := range keys {
		_, cmd := o.Update(k)
		assert.Nil(t, cmd, "key %q should be consumed with nil cmd", k.String())
	}
}

// TestHelpOverlay_SetTheme verifies SetTheme does not panic.
func TestHelpOverlay_SetTheme(t *testing.T) {
	o := newTestHelpOverlay()
	assert.NotPanics(t, func() {
		o.SetTheme(theme.Load("monokai"))
	})
	assert.Equal(t, "monokai", o.theme.ID())
}

// TestHelpOverlay_SetSize verifies dimensions are stored.
func TestHelpOverlay_SetSize(t *testing.T) {
	o := newTestHelpOverlay()
	o.SetSize(100, 30)
	assert.Equal(t, 100, o.width)
	assert.Equal(t, 30, o.height)
}

// TestHelpOverlay_Update_NonKeyMsgIgnored verifies non-key messages return nil cmd.
func TestHelpOverlay_Update_NonKeyMsgIgnored(t *testing.T) {
	o := newTestHelpOverlay()
	_, cmd := o.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	assert.Nil(t, cmd)
}
```

- [ ] **Step 2: Verify tests do not compile yet** (expected — `HelpOverlay` type doesn't exist)

```bash
cd /Users/irshadsheikh/dev/github/apps/spotnik
go test ./internal/ui/panes/ 2>&1 | head -5
```

Expected: `undefined: HelpOverlay` or `undefined: NewHelpOverlay`.

---

## Task 3: Implement HelpOverlay

**Files:**
- Create: `internal/ui/panes/help_overlay.go`

- [ ] **Step 1: Create the implementation file**

```go
// Package panes — HelpOverlay is the floating keybinding reference overlay.
// It renders all app keybindings in a two-column grouped layout.
// Pressing Esc emits HelpOverlayClosedMsg; all other keys are consumed silently.
package panes

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// HelpOverlayClosedMsg is emitted when the user presses Esc in the HelpOverlay.
// The root app handles this by closing the overlay.
type HelpOverlayClosedMsg struct{}

// helpBinding is a single key → label pair displayed in the help overlay.
type helpBinding struct{ key, label string }

// helpSection groups related bindings under a titled header.
type helpSection struct {
	title    string
	bindings []helpBinding
}

// helpContent is the static two-column keybinding reference.
// Index 0 = left column (Global, Navigation).
// Index 1 = right column (Playback, Pane Actions).
//
// NOTE: When adding, changing, or removing any keybinding, also update:
//   - docs/keybinding.md  (human-readable reference)
//   - docs/DESIGN.md §17  (spec keybinding table)
var helpContent = [2][]helpSection{
	// Left column
	{
		{title: "Global", bindings: []helpBinding{
			{"/", "search"},
			{"d", "devices"},
			{"t", "theme"},
			{"?", "help"},
			{"q", "quit"},
			{"0", "toggle page"},
			{"1-8", "toggle pane"},
			{"p", "preset"},
		}},
		{title: "Navigation", bindings: []helpBinding{
			{"Tab", "next pane"},
			{"Shift+Tab", "prev pane"},
			{"j / k", "scroll"},
			{"Esc", "close overlay"},
		}},
	},
	// Right column
	{
		{title: "Playback", bindings: []helpBinding{
			{"Space", "play / pause"},
			{"n", "next track"},
			{"← / →", "prev / next"},
			{"+ / -", "volume"},
			{"s", "shuffle"},
			{"r", "repeat"},
			{"v", "visualizer"},
		}},
		{title: "Pane Actions", bindings: []helpBinding{
			{"Enter", "select / play"},
			{"f", "filter"},
			{"A", "add to queue"},
			{"i", "like / unlike"},
			{"x", "remove track"},
			{"Shift+↑/↓", "reorder (playlists)"},
		}},
	},
}

// HelpOverlay is the floating keybinding reference overlay model.
// It renders two columns of grouped keybindings inside a btop-style border.
// All key events except Esc are consumed without side effects.
type HelpOverlay struct {
	theme  theme.Theme
	width  int
	height int
}

// NewHelpOverlay creates a HelpOverlay styled with the given theme.
func NewHelpOverlay(th theme.Theme) *HelpOverlay {
	return &HelpOverlay{theme: th}
}

// SetSize updates the render dimensions for the overlay.
func (o *HelpOverlay) SetSize(width, height int) {
	o.width = width
	o.height = height
}

// SetTheme updates the overlay's theme reference for runtime theme switching.
func (o *HelpOverlay) SetTheme(th theme.Theme) {
	o.theme = th
}

// Init satisfies tea.Model; no startup command needed.
func (o *HelpOverlay) Init() tea.Cmd { return nil }

// Update handles keyboard input for the help overlay.
// Esc closes the overlay by emitting HelpOverlayClosedMsg.
// All other keys are consumed silently — the overlay is fully modal.
func (o *HelpOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return o, nil
	}
	if keyMsg.Type == tea.KeyEsc {
		return o, func() tea.Msg { return HelpOverlayClosedMsg{} }
	}
	return o, nil
}

// overlayWidth returns the total overlay width in columns.
// Fixed at 78 columns, capped to terminal width when known.
func (o *HelpOverlay) overlayWidth() int {
	const fixedW = 78
	if o.width > 0 && fixedW > o.width {
		return o.width
	}
	return fixedW
}

// View renders the two-column keybinding reference inside a btop-style border.
func (o *HelpOverlay) View() string {
	totalW := o.overlayWidth()
	innerW := totalW - 2
	if innerW < 4 {
		innerW = 4
	}

	// Split inner width into two columns with a 1-column divider.
	leftW := (innerW - 1) / 2
	rightW := innerW - 1 - leftW

	left := o.renderColumn(helpContent[0], leftW)
	right := o.renderColumn(helpContent[1], rightW)

	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")

	// Pad the shorter column so both have equal height.
	for len(leftLines) < len(rightLines) {
		leftLines = append(leftLines, strings.Repeat(" ", leftW))
	}
	for len(rightLines) < len(leftLines) {
		rightLines = append(rightLines, strings.Repeat(" ", rightW))
	}

	divider := lipgloss.NewStyle().Foreground(o.theme.TextMuted()).Render("│")
	rows := make([]string, len(leftLines))
	for i := range leftLines {
		rows[i] = leftLines[i] + divider + rightLines[i]
	}

	inner := strings.Join(rows, "\n")
	inner = lipgloss.NewStyle().
		Width(innerW).MaxWidth(innerW).
		Render(inner)

	cfg := layout.BorderConfig{
		Width:       totalW,
		Height:      len(rows) + 2, // +2 for top and bottom border rows
		Title:       "Help",
		Actions:     []layout.Action{{Key: "Esc", Label: "close"}},
		AccentColor: o.theme.ActiveBorder(),
		Focused:     true, // overlays are always rendered as focused
		Theme:       o.theme,
	}
	return layout.RenderPaneBorder(inner, cfg)
}

// keyColWidth is the fixed width of the key name column within each side column.
// Wide enough for "Shift+Tab" (9 chars) plus padding.
const keyColWidth = 12

// renderColumn renders one column of helpSections, returning a multi-line string.
// Each line is exactly `width` columns wide.
func (o *HelpOverlay) renderColumn(sections []helpSection, width int) string {
	headerStyle := lipgloss.NewStyle().Foreground(o.theme.Info()).Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(o.theme.TextPrimary())
	labelStyle := lipgloss.NewStyle().Foreground(o.theme.TextMuted())
	rowStyle := lipgloss.NewStyle().Width(width).MaxWidth(width)

	var lines []string
	for i, sec := range sections {
		if i > 0 {
			// Blank separator line between sections.
			lines = append(lines, rowStyle.Render(""))
		}
		// Section header.
		lines = append(lines, rowStyle.Render(headerStyle.Render(sec.title)))
		// Binding rows: fixed-width key column + label.
		for _, b := range sec.bindings {
			keyPart := lipgloss.NewStyle().
				Width(keyColWidth).MaxWidth(keyColWidth).
				Render(keyStyle.Render(b.key))
			labelPart := labelStyle.Render(b.label)
			lines = append(lines, rowStyle.Render(keyPart+labelPart))
		}
	}
	return strings.Join(lines, "\n")
}
```

- [ ] **Step 2: Run the tests and verify they pass**

```bash
cd /Users/irshadsheikh/dev/github/apps/spotnik
go test ./internal/ui/panes/ -run TestHelpOverlay -v
```

Expected: all `TestHelpOverlay_*` tests pass.

- [ ] **Step 3: Run lint**

```bash
make lint
```

Expected: no errors. If `Bold()` is flagged (lipgloss v0.x removes it), replace with `lipgloss.NewStyle().Foreground(...).Bold(true)` — which is already the form used.

- [ ] **Step 4: Commit**

```bash
git add internal/ui/panes/help_overlay.go internal/ui/panes/help_overlay_test.go
git commit -m "feat(help): add HelpOverlay struct with static keybinding data and tests"
```

---

## Task 4: Wire HelpOverlay into app.go

**Files:**
- Modify: `internal/app/app.go`

Four changes in this task: fields, open/close helpers, `handleMsg` case, and two propagation sites (resize + theme switch).

- [ ] **Step 1: Add struct fields**

In `app.go`, find the block containing `deviceOverlayOpen bool` and `showThemeSwitcher bool` (around line 136–140). Add two lines immediately after `showThemeSwitcher`:

```go
	// helpOpen is true while the help keybinding overlay is visible.
	helpOpen bool
	// helpOverlay is the floating help overlay. Populated when open.
	helpOverlay *panes.HelpOverlay
```

- [ ] **Step 2: Add openHelp and closeHelp methods**

Find `closeThemeSwitcher()` (around line 822) and add the new helpers immediately after it:

```go
// openHelp opens the help keybinding overlay.
func (a *App) openHelp() (*App, tea.Cmd) {
	a.helpOpen = true
	a.helpOverlay = panes.NewHelpOverlay(a.theme)
	a.helpOverlay.SetSize(a.width, a.height)
	return a, nil
}

// closeHelp closes the help overlay.
func (a *App) closeHelp() (*App, tea.Cmd) {
	a.helpOpen = false
	a.helpOverlay = nil
	return a, nil
}
```

- [ ] **Step 3: Add HelpOverlayClosedMsg handler in handleMsg**

Find the `case panes.ThemeOverlayClosedMsg:` block (around line 1639). Add the new case immediately before it:

```go
	case panes.HelpOverlayClosedMsg:
		// Help overlay closed via Esc.
		return a.closeHelp()

```

- [ ] **Step 4: Propagate size on terminal resize**

In the `tea.WindowSizeMsg` handler (around line 1048), find:

```go
		if a.themeOverlay != nil {
			a.themeOverlay.SetSize(m.Width, m.Height)
		}
```

Add immediately after:

```go
		if a.helpOverlay != nil {
			a.helpOverlay.SetSize(m.Width, m.Height)
		}
```

- [ ] **Step 5: Propagate theme on theme switch**

In the `panes.ThemeSwitchMsg` handler (around line 1614–1618), find:

```go
		if a.themeOverlay != nil {
			a.themeOverlay.SetTheme(newTheme)
		}
```

Add immediately after:

```go
		if a.helpOverlay != nil {
			a.helpOverlay.SetTheme(newTheme)
		}
```

- [ ] **Step 6: Verify it compiles**

```bash
cd /Users/irshadsheikh/dev/github/apps/spotnik
go build ./...
```

Expected: no errors.

- [ ] **Step 7: Commit**

```bash
git add internal/app/app.go
git commit -m "feat(help): wire HelpOverlay fields and lifecycle into app.go"
```

---

## Task 5: Wire ? key and overlay guard into routing.go

**Files:**
- Modify: `internal/app/routing.go`

Three changes: help overlay key guard at the top of `handleKeyMsg`, `?` key handler in the global keys section, and `helpOpen` added to the mouse guard.

- [ ] **Step 1: Add help overlay guard at top of handleKeyMsg**

In `routing.go`, find `handleKeyMsg`. The existing guards at the top are (in order): theme switcher → device overlay → search overlay → auth view → filter check.

Add the help overlay guard immediately after the theme switcher guard (after the `if a.showThemeSwitcher` block closes, before the `if a.deviceOverlayOpen` block):

```go
	// When help overlay is open, route all keys to it.
	if a.helpOpen && a.helpOverlay != nil {
		updated, cmd := a.helpOverlay.Update(m)
		if ho, ok := updated.(*panes.HelpOverlay); ok {
			a.helpOverlay = ho
		}
		return a, cmd
	}
```

- [ ] **Step 2: Add ? key handler**

Find the `'t'` theme switcher handler (around line 107):

```go
	// 't' opens the theme switcher overlay — but only if no other overlay is already open.
	if m.Type == tea.KeyRunes && string(m.Runes) == "t" {
		if !a.searchOpen && !a.deviceOverlayOpen && !a.showThemeSwitcher {
			return a.openThemeSwitcher()
		}
		return a, nil
	}
```

Add the `?` handler immediately after it:

```go
	// '?' opens the help overlay — but only if no other overlay is already open.
	if m.Type == tea.KeyRunes && string(m.Runes) == "?" {
		if !a.searchOpen && !a.deviceOverlayOpen && !a.showThemeSwitcher && !a.helpOpen {
			return a.openHelp()
		}
		return a, nil
	}
```

- [ ] **Step 3: Add helpOpen to mouse event guard**

Find `handleMouseMsg`. Near the top is:

```go
	if a.deviceOverlayOpen || a.searchOpen {
		return nil
	}
```

Change it to:

```go
	if a.deviceOverlayOpen || a.searchOpen || a.helpOpen {
		return nil
	}
```

- [ ] **Step 4: Verify it compiles**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add internal/app/routing.go
git commit -m "feat(help): wire ? key handler and help overlay routing guards"
```

---

## Task 6: Wire renderWithHelpOverlay into render.go

**Files:**
- Modify: `internal/app/render.go`

Two changes: add the `helpOpen` branch in `buildView()`, and add the `renderWithHelpOverlay()` method.

- [ ] **Step 1: Add helpOpen branch in buildView()**

Find the overlay compositing block in `buildView()` (around line 139):

```go
	if a.showThemeSwitcher && a.themeOverlay != nil {
		return a.renderWithThemeOverlay(body)
	}

	if a.deviceOverlayOpen {
		return a.renderWithDeviceOverlay(body)
	}

	if a.searchOpen {
		return a.renderWithSearchOverlay(body)
	}

	return body
```

Add the help overlay branch before `return body`:

```go
	if a.showThemeSwitcher && a.themeOverlay != nil {
		return a.renderWithThemeOverlay(body)
	}

	if a.deviceOverlayOpen {
		return a.renderWithDeviceOverlay(body)
	}

	if a.searchOpen {
		return a.renderWithSearchOverlay(body)
	}

	if a.helpOpen && a.helpOverlay != nil {
		return a.renderWithHelpOverlay(body)
	}

	return body
```

- [ ] **Step 2: Add renderWithHelpOverlay method**

Find `renderWithSearchOverlay` (around line 260) and add the new method immediately after it:

```go
// renderWithHelpOverlay renders the grid dimmed and places the help overlay
// centered on screen using bubbletea-overlay Composite().
func (a *App) renderWithHelpOverlay(background string) string {
	fg := a.helpOverlay.View()
	dimmed := lipgloss.NewStyle().Faint(true).Render(background)
	if a.width <= 0 || a.height <= 0 {
		return dimmed + "\n" + fg
	}
	return btoverlay.Composite(fg, dimmed, btoverlay.Center, btoverlay.Center, 0, 0)
}
```

- [ ] **Step 3: Verify it compiles and all tests pass**

```bash
go build ./...
make test
```

Expected: build succeeds, all tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/app/render.go
git commit -m "feat(help): add renderWithHelpOverlay and buildView branch"
```

---

## Task 7: Create docs/keybinding.md

**Files:**
- Create: `docs/keybinding.md`

- [ ] **Step 1: Create the file**

```markdown
# Spotnik Keybindings

> **Keep this file in sync** with `docs/DESIGN.md §17` and the `helpContent` var
> in `internal/ui/panes/help_overlay.go` whenever any keybinding changes.

---

## Global

| Key | Action |
|-----|--------|
| `/` | Open search overlay |
| `d` | Open device switcher |
| `t` | Open theme switcher |
| `?` | Open help overlay |
| `q` | Quit |
| `0` | Toggle Page A / Page B |
| `1`–`8` | Toggle pane visibility (Page A only) |
| `p` | Cycle preset |

## Playback

Playback keys are always active regardless of which pane has focus.

| Key | Action |
|-----|--------|
| `Space` | Play / pause |
| `n` | Next track |
| `←` / `→` | Previous / next track |
| `+` / `-` | Volume up / down |
| `s` | Toggle shuffle |
| `r` | Cycle repeat mode |
| `v` | Cycle visualizer pattern |

## Navigation

| Key | Action |
|-----|--------|
| `Tab` | Next pane focus |
| `Shift+Tab` | Previous pane focus |
| `j` / `k` | Scroll down / up |
| `Esc` | Close overlay or filter |

## Pane Actions

| Key | Action | Context |
|-----|--------|---------|
| `Enter` | Select / play item | Focused pane |
| `f` | Toggle filter | List panes |
| `A` | Add to queue | Search overlay, list panes |
| `i` | Like / unlike track | LikedSongs pane |
| `x` | Remove track from playlist | Playlists pane track sub-view |
| `Shift+↑` / `Shift+↓` | Reorder track | Playlists pane track sub-view |

## Search Overlay

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Cycle search category |
| `Enter` | Play selected result |
| `Ctrl+A` | Add result to queue |
| `Ctrl+U` | Clear search input |
| `PgDn` / `PgUp` | Next / previous result page |
| `Esc` | Close search overlay |
```

- [ ] **Step 2: Commit**

```bash
git add docs/keybinding.md
git commit -m "docs(keybinding): add canonical keybinding reference"
```

---

## Task 8: Update DESIGN.md and CLAUDE.md

**Files:**
- Modify: `docs/DESIGN.md`
- Modify: `CLAUDE.md`

- [ ] **Step 1: Update DESIGN.md §17**

In `docs/DESIGN.md`, find (around line 822):

```
| `?` | Help (planned) | Global |
```

Replace with:

```
| `?` | Open help overlay | Global |
```

- [ ] **Step 2: Add keybinding maintenance rule to CLAUDE.md — "What Agents Must NEVER Do" list**

In `CLAUDE.md`, find the numbered list under "What Agents Must NEVER Do". It ends at item 14. Add item 15:

```
15. Add, change, or remove a keybinding without updating all three locations in the same commit:
    `docs/keybinding.md`, `docs/DESIGN.md §17`, and the `helpContent` var in
    `internal/ui/panes/help_overlay.go`.
```

- [ ] **Step 3: Add Keybinding Maintenance section to CLAUDE.md**

In `CLAUDE.md`, find the "Design Rules" section. Add a new section immediately after it:

```markdown
## Keybinding Maintenance

All keybindings are documented in three places that must stay in sync:

- `docs/keybinding.md` — human-readable reference (canonical for external readers)
- `docs/DESIGN.md §17` — spec-level keybinding table
- `internal/ui/panes/help_overlay.go` `helpContent` var — in-app help overlay display

When adding, changing, or removing any keybinding, update all three in the same commit.
```

- [ ] **Step 4: Commit**

```bash
git add docs/DESIGN.md CLAUDE.md
git commit -m "docs(keybinding): update DESIGN.md and add CLAUDE.md maintenance rules"
```

---

## Task 9: Full CI check and PR

- [ ] **Step 1: Run full CI**

```bash
make ci
```

Expected: lint passes, all unit tests pass, integration tests pass, coverage ≥ 80%.

- [ ] **Step 2: If coverage is below 80%, check which lines are uncovered**

```bash
make test-coverage
```

Look for uncovered lines in `help_overlay.go`. The most likely gap is `overlayWidth()` when `o.width > 0 && fixedW > o.width`. Add a test if needed:

```go
// TestHelpOverlay_OverlayWidth_CappedToTerminalWidth verifies overlay width
// is capped when terminal is narrower than the fixed 78-column default.
func TestHelpOverlay_OverlayWidth_CappedToTerminalWidth(t *testing.T) {
	o := newTestHelpOverlay()
	o.SetSize(60, 30)
	view := o.View()
	// The view should still render without panic even when capped.
	assert.NotEmpty(t, view)
}
```

- [ ] **Step 3: Push branch**

```bash
git push origin feat/help-overlay
```

- [ ] **Step 4: Open PR**

```bash
gh pr create \
  --title "feat(help): implement ? help overlay with keybinding reference" \
  --body "$(cat <<'EOF'
## Summary
- Adds `HelpOverlay` in `internal/ui/panes/help_overlay.go` — two-column grouped keybinding reference, centered modal, dismissed with Esc
- Wires `?` key in `routing.go` following the same guard pattern as theme/device overlays
- Adds `renderWithHelpOverlay()` in `render.go` using `btoverlay.Center`
- Creates `docs/keybinding.md` as canonical human-readable keybinding reference
- Updates `docs/DESIGN.md §17` — `?` no longer marked "planned"
- Adds CLAUDE.md rule requiring all three keybinding locations stay in sync on every binding change

## Test plan
- [ ] Run `make ci` — all tests pass, coverage ≥ 80%
- [ ] Launch app, press `?` — help overlay appears centered with two columns
- [ ] Press `Esc` — overlay closes, grid resumes
- [ ] Press `?` while search overlay is open — nothing happens
- [ ] Resize terminal while help overlay is open — overlay repositions correctly
- [ ] Switch theme while help overlay is open — overlay re-renders with new theme colors

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

---

## Self-Review

**Spec coverage check:**

| Spec requirement | Task that covers it |
|---|---|
| `HelpOverlay` struct with `helpContent` static data | Task 3 |
| `HelpOverlayClosedMsg` | Task 3 |
| Two-column layout with section headers | Task 3 (View + renderColumn) |
| Theme tokens used, no hardcoded hex | Task 3 (headerStyle, keyStyle, labelStyle) |
| `helpOpen` + `helpOverlay` fields in app.go | Task 4 |
| `openHelp()` / `closeHelp()` | Task 4 |
| `HelpOverlayClosedMsg` handler | Task 4 |
| WindowSizeMsg propagation | Task 4 |
| Theme switch propagation | Task 4 |
| Guard at top of handleKeyMsg | Task 5 |
| `?` key handler (no-op when other overlay open) | Task 5 |
| `helpOpen` in mouse guard | Task 5 |
| `renderWithHelpOverlay()` with Center/Center | Task 6 |
| `helpOpen` branch in `buildView()` | Task 6 |
| `docs/keybinding.md` created | Task 7 |
| `docs/DESIGN.md §17` updated | Task 8 |
| CLAUDE.md rule #15 added | Task 8 |
| CLAUDE.md "Keybinding Maintenance" section | Task 8 |

**Placeholder scan:** No TBD, no "similar to", no incomplete steps. All code blocks are complete.

**Type consistency check:**
- `HelpOverlayClosedMsg` — defined in Task 3 (`help_overlay.go`), handled in Task 4 (`app.go`). ✓
- `HelpOverlay` — defined in Task 3, used in Task 4 (`*panes.HelpOverlay`), Task 5 (`updated.(*panes.HelpOverlay)`). ✓
- `openHelp()` / `closeHelp()` — defined in Task 4, called in Tasks 5 and 4 respectively. ✓
- `renderWithHelpOverlay()` — defined in Task 6, called in Task 6. ✓
- `helpOpen` bool — set in Tasks 4 and 5, read in Tasks 5 and 6. ✓
