---
title: "Help overlay implementation and keybinding documentation"
feature: 21-help-overlay
status: done
---

## Background

The `?` key is declared in `appKeyMap` (`internal/app/render.go:76`) with
`key.WithHelp("?", "help")` and renders in the status bar, but `routing.go` has no
handler for it and no `HelpOverlay` struct exists anywhere in the codebase.
`docs/DESIGN.md §17` marks it `"Help (planned)"`.

The three existing overlays (ThemeOverlay, DeviceOverlay, SearchOverlay) all follow
the same wiring pattern and are the model for this implementation:

- Struct in `internal/ui/panes/` with `View()` / `Update()` / `SetSize()` / `SetTheme()`
- `bool` flag + `*Overlay` pointer in `app.go`
- Guard at top of `handleKeyMsg()` in `routing.go`
- `open*()` / `close*()` helpers in `app.go`
- `renderWith*Overlay(background string)` in `render.go` using `btoverlay.Composite()`

### Why static keybinding data

`Pane.Actions()` is context-sensitive: when a filter is active it returns
`{Esc, close}` instead of `{f, filter}`; when in a track sub-view it returns
`{Esc, back}`. Calling it at overlay-open time silently gives wrong results.
Playback keys have no structured `key.Binding` definitions at all (defined only in
`isPlaybackKey()` switch). Static data in a `helpContent` var is the right choice
for a stable reference display.

### Three-location keybinding sync rule

After this story, keybindings are documented in three places:
1. `docs/keybinding.md` — human-readable Markdown reference
2. `docs/DESIGN.md §17` — spec-level table
3. `internal/ui/panes/help_overlay.go` `helpContent` var — in-app display

All three must be updated together in any commit that adds, changes, or removes a
keybinding. CLAUDE.md is updated to enforce this.

---

## Design

### New file: `internal/ui/panes/help_overlay.go`

```go
// HelpOverlayClosedMsg is emitted when the user presses Esc in the HelpOverlay.
type HelpOverlayClosedMsg struct{}

// helpBinding is a single key → label pair.
type helpBinding struct{ key, label string }

// helpSection groups related bindings under a titled header.
type helpSection struct {
    title    string
    bindings []helpBinding
}

// helpContent is the static two-column keybinding reference.
// [0] = left column (Global, Navigation), [1] = right column (Playback, Pane Actions).
// NOTE: When changing any keybinding, also update docs/keybinding.md and docs/DESIGN.md §17.
var helpContent = [2][]helpSection{
    {
        {title: "Global", bindings: []helpBinding{
            {"/", "search"}, {"d", "devices"}, {"t", "theme"}, {"?", "help"},
            {"q", "quit"}, {"0", "toggle page"}, {"1-8", "toggle pane"}, {"p", "preset"},
        }},
        {title: "Navigation", bindings: []helpBinding{
            {"Tab", "next pane"}, {"Shift+Tab", "prev pane"},
            {"j / k", "scroll"}, {"Esc", "close overlay"},
        }},
    },
    {
        {title: "Playback", bindings: []helpBinding{
            {"Space", "play / pause"}, {"n", "next track"}, {"← / →", "prev / next"},
            {"+ / -", "volume"}, {"s", "shuffle"}, {"r", "repeat"}, {"v", "visualizer"},
        }},
        {title: "Pane Actions", bindings: []helpBinding{
            {"Enter", "select / play"}, {"f", "filter"}, {"A", "add to queue"},
            {"i", "like / unlike"}, {"x", "remove track"}, {"Shift+↑/↓", "reorder (playlists)"},
        }},
    },
}

// HelpOverlay is the floating keybinding reference overlay model.
type HelpOverlay struct {
    theme  theme.Theme
    width  int
    height int
}

func NewHelpOverlay(th theme.Theme) *HelpOverlay
func (o *HelpOverlay) SetSize(width, height int)
func (o *HelpOverlay) SetTheme(th theme.Theme)
func (o *HelpOverlay) Init() tea.Cmd   // returns nil
func (o *HelpOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd)  // Esc → HelpOverlayClosedMsg; else nil
func (o *HelpOverlay) View() string    // two-column layout inside RenderPaneBorder
```

**View layout** — fixed 78-column overlay, two 37-column content columns separated by a
`│` divider, rendered centered via `btoverlay.Center, Center`:

```
╭─ Help ─────────────────────────────────────────────────────────────────────╮
│  Global                          │  Playback                               │
│  /          search               │  Space      play / pause                │
│  d          devices              │  n          next track                  │
│  ...                             │  ...                                    │
│  Navigation                      │  Pane Actions                           │
│  Tab        next pane            │  Enter      select / play               │
│  ...                             │  ...                                    │
╰─ Esc close ────────────────────────────────────────────────────────────────╯
```

**Color tokens** (no hardcoded hex):

| Element | Token |
|---|---|
| Border + title | `theme.ActiveBorder()` |
| Section headers | `theme.Info()` + bold |
| Key names | `theme.TextPrimary()` |
| Key labels | `theme.TextMuted()` |
| Column divider `│` | `theme.TextMuted()` |

**`renderColumn(sections []helpSection, width int) string`** — renders one side.
Each binding row uses a fixed 12-column key sub-column (`keyColWidth = 12`) wide
enough for "Shift+Tab" and "Shift+↑/↓", with the label taking the remainder.

### Changes to `internal/app/app.go`

**New fields** (add after `showThemeSwitcher bool`):
```go
// helpOpen is true while the help keybinding overlay is visible.
helpOpen    bool
// helpOverlay is the floating help overlay. Populated when open.
helpOverlay *panes.HelpOverlay
```

**New helpers** (add after `closeThemeSwitcher()`):
```go
func (a *App) openHelp() (*App, tea.Cmd) {
    a.helpOpen = true
    a.helpOverlay = panes.NewHelpOverlay(a.theme)
    a.helpOverlay.SetSize(a.width, a.height)
    return a, nil
}

func (a *App) closeHelp() (*App, tea.Cmd) {
    a.helpOpen = false
    a.helpOverlay = nil
    return a, nil
}
```

**handleMsg — new case** (add before `case panes.ThemeOverlayClosedMsg`):
```go
case panes.HelpOverlayClosedMsg:
    return a.closeHelp()
```

**WindowSizeMsg propagation** (add after `themeOverlay.SetSize` block):
```go
if a.helpOverlay != nil {
    a.helpOverlay.SetSize(m.Width, m.Height)
}
```

**Theme switch propagation** (add after `themeOverlay.SetTheme` block):
```go
if a.helpOverlay != nil {
    a.helpOverlay.SetTheme(newTheme)
}
```

### Changes to `internal/app/routing.go`

**Guard** — add after the theme switcher guard, before the device overlay guard:
```go
if a.helpOpen && a.helpOverlay != nil {
    updated, cmd := a.helpOverlay.Update(m)
    if ho, ok := updated.(*panes.HelpOverlay); ok {
        a.helpOverlay = ho
    }
    return a, cmd
}
```

**`?` key handler** — add after the `'t'` theme handler:
```go
if m.Type == tea.KeyRunes && string(m.Runes) == "?" {
    if !a.searchOpen && !a.deviceOverlayOpen && !a.showThemeSwitcher && !a.helpOpen {
        return a.openHelp()
    }
    return a, nil
}
```

**Mouse guard** — update the existing guard in `handleMouseMsg`:
```go
// Before:
if a.deviceOverlayOpen || a.searchOpen {
// After:
if a.deviceOverlayOpen || a.searchOpen || a.helpOpen {
```

### Changes to `internal/app/render.go`

**New method** (add after `renderWithSearchOverlay`):
```go
func (a *App) renderWithHelpOverlay(background string) string {
    fg := a.helpOverlay.View()
    dimmed := lipgloss.NewStyle().Faint(true).Render(background)
    if a.width <= 0 || a.height <= 0 {
        return dimmed + "\n" + fg
    }
    return btoverlay.Composite(fg, dimmed, btoverlay.Center, btoverlay.Center, 0, 0)
}
```

**`buildView()` branch** (add before `return body`):
```go
if a.helpOpen && a.helpOverlay != nil {
    return a.renderWithHelpOverlay(body)
}
```

### New file: `docs/keybinding.md`

Human-readable Markdown keybinding reference. Structure:

```markdown
# Spotnik Keybindings

> Keep in sync with docs/DESIGN.md §17 and helpContent in
> internal/ui/panes/help_overlay.go on every keybinding change.

## Global
| Key | Action |
|-----|--------|
| `/` | Open search overlay |
...

## Playback
## Navigation
## Pane Actions
## Search Overlay
```

Full content mirrors the `helpContent` static data plus the search-overlay-specific
bindings (Tab/Shift+Tab cycle, Ctrl+U clear, PgDn/PgUp pagination, Esc close).

### Changes to `docs/DESIGN.md`

In §17 Keybinding Table, update:
```
| `?` | Help (planned) | Global |
```
→
```
| `?` | Open help overlay | Global |
```

### Changes to `CLAUDE.md`

**Add item 15 to "What Agents Must NEVER Do":**
```
15. Add, change, or remove a keybinding without updating all three locations in the
    same commit: docs/keybinding.md, docs/DESIGN.md §17, and the helpContent var in
    internal/ui/panes/help_overlay.go.
```

**Add new section after "Design Rules":**
```markdown
## Keybinding Maintenance

All keybindings are documented in three places that must stay in sync:
- `docs/keybinding.md` — human-readable reference (canonical for external readers)
- `docs/DESIGN.md §17` — spec-level keybinding table
- `internal/ui/panes/help_overlay.go` `helpContent` var — in-app help overlay display

When adding, changing, or removing any keybinding, update all three in the same commit.
```

---

## Tasks

### Task 1: Feature branch

```bash
git checkout main && git pull origin main
git checkout -b feat/21-help-overlay
```

---

### Task 2: Write HelpOverlay tests (TDD — write before implementation)

**File to create:** `internal/ui/panes/help_overlay_test.go`

Use `package panes` (internal test package, same as `themes_test.go`) to access
unexported fields like `o.theme`, `o.width`, `o.height`.

The `main_test.go` in this package already sets `lipgloss.SetColorProfile(termenv.TrueColor)`
so ANSI codes are emitted in tests — `assert.Contains` checks for text substrings work
correctly even with colour escape codes surrounding them.

```go
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

func TestHelpOverlay_View_HasBorder(t *testing.T) {
    o := newTestHelpOverlay()
    o.SetSize(120, 40)
    view := o.View()
    require.NotEmpty(t, view)
    assert.Contains(t, view, "╭")
    assert.Contains(t, view, "╰")
}

func TestHelpOverlay_View_HasTitle(t *testing.T) {
    o := newTestHelpOverlay()
    o.SetSize(120, 40)
    assert.Contains(t, o.View(), "Help")
}

func TestHelpOverlay_View_ContainsSectionHeaders(t *testing.T) {
    o := newTestHelpOverlay()
    o.SetSize(120, 40)
    view := o.View()
    for _, h := range []string{"Global", "Navigation", "Playback", "Pane Actions"} {
        assert.Contains(t, view, h, "section header %q should appear", h)
    }
}

func TestHelpOverlay_View_ContainsGlobalKeys(t *testing.T) {
    o := newTestHelpOverlay()
    o.SetSize(120, 40)
    view := o.View()
    for _, k := range []string{"/", "d", "t", "?", "q", "0", "1-8", "p"} {
        assert.Contains(t, view, k, "global key %q should appear", k)
    }
}

func TestHelpOverlay_View_ContainsPlaybackKeys(t *testing.T) {
    o := newTestHelpOverlay()
    o.SetSize(120, 40)
    view := o.View()
    for _, k := range []string{"Space", "n", "s", "r", "v"} {
        assert.Contains(t, view, k, "playback key %q should appear", k)
    }
}

func TestHelpOverlay_View_ContainsPaneActionKeys(t *testing.T) {
    o := newTestHelpOverlay()
    o.SetSize(120, 40)
    view := o.View()
    for _, k := range []string{"Enter", "f", "A", "i", "x"} {
        assert.Contains(t, view, k, "pane action key %q should appear", k)
    }
}

func TestHelpOverlay_Update_EscEmitsClosedMsg(t *testing.T) {
    o := newTestHelpOverlay()
    _, cmd := o.Update(tea.KeyMsg{Type: tea.KeyEsc})
    require.NotNil(t, cmd)
    _, ok := cmd().(HelpOverlayClosedMsg)
    assert.True(t, ok)
}

func TestHelpOverlay_Update_OtherKeysConsumed(t *testing.T) {
    o := newTestHelpOverlay()
    for _, k := range []tea.KeyMsg{
        {Type: tea.KeyRunes, Runes: []rune{'j'}},
        {Type: tea.KeyRunes, Runes: []rune{'q'}},
        {Type: tea.KeyEnter},
    } {
        _, cmd := o.Update(k)
        assert.Nil(t, cmd, "key %q should be consumed with nil cmd", k.String())
    }
}

func TestHelpOverlay_Update_NonKeyMsgIgnored(t *testing.T) {
    o := newTestHelpOverlay()
    _, cmd := o.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
    assert.Nil(t, cmd)
}

func TestHelpOverlay_SetTheme(t *testing.T) {
    o := newTestHelpOverlay()
    assert.NotPanics(t, func() { o.SetTheme(theme.Load("monokai")) })
    assert.Equal(t, "monokai", o.theme.ID())
}

func TestHelpOverlay_SetSize(t *testing.T) {
    o := newTestHelpOverlay()
    o.SetSize(100, 30)
    assert.Equal(t, 100, o.width)
    assert.Equal(t, 30, o.height)
}

func TestHelpOverlay_View_NarrowTerminal(t *testing.T) {
    o := newTestHelpOverlay()
    o.SetSize(60, 30) // narrower than fixed 78-col width
    assert.NotPanics(t, func() { _ = o.View() })
}
```

Verify tests fail to compile (expected — `HelpOverlay` doesn't exist yet):
```bash
go test ./internal/ui/panes/ -run TestHelpOverlay 2>&1 | head -5
```

---

### Task 3: Implement `internal/ui/panes/help_overlay.go`

Create the file with the full implementation. Implement `HelpOverlayClosedMsg`,
`helpBinding`, `helpSection`, `helpContent`, `HelpOverlay`, `NewHelpOverlay`,
`SetSize`, `SetTheme`, `Init`, `Update`, `overlayWidth`, `View`, and `renderColumn`
exactly as specified in the Design section above.

Key implementation notes:
- `overlayWidth()` returns 78, capped to `o.width` when `o.width > 0 && 78 > o.width`
- `View()`: `innerW = totalW - 2`; `leftW = (innerW - 1) / 2`; `rightW = innerW - 1 - leftW`
- Pad the shorter column with `strings.Repeat(" ", colW)` lines to match heights
- `keyColWidth = 12` — const at package level
- `BorderConfig{Title: "Help", Actions: []layout.Action{{Key: "Esc", Label: "close"}}, Focused: true}`
- Height passed to `BorderConfig`: `len(rows) + 2`

Run tests and verify they pass:
```bash
go test ./internal/ui/panes/ -run TestHelpOverlay -v
make lint
```

Commit:
```bash
git add internal/ui/panes/help_overlay.go internal/ui/panes/help_overlay_test.go
git commit -m "feat(help): add HelpOverlay struct with two-column keybinding layout"
```

---

### Task 4: Wire into `internal/app/app.go`

**Four changes — make all in one edit pass:**

1. **Fields** — after `showThemeSwitcher bool` (search for that line):
```go
helpOpen    bool
helpOverlay *panes.HelpOverlay
```

2. **Helpers** — after `closeThemeSwitcher()` function:
```go
func (a *App) openHelp() (*App, tea.Cmd) {
    a.helpOpen = true
    a.helpOverlay = panes.NewHelpOverlay(a.theme)
    a.helpOverlay.SetSize(a.width, a.height)
    return a, nil
}

func (a *App) closeHelp() (*App, tea.Cmd) {
    a.helpOpen = false
    a.helpOverlay = nil
    return a, nil
}
```

3. **handleMsg case** — add before `case panes.ThemeOverlayClosedMsg`:
```go
case panes.HelpOverlayClosedMsg:
    return a.closeHelp()
```

4. **Size propagation** — in `tea.WindowSizeMsg` handler, after the `themeOverlay.SetSize` block:
```go
if a.helpOverlay != nil {
    a.helpOverlay.SetSize(m.Width, m.Height)
}
```

5. **Theme propagation** — in `panes.ThemeSwitchMsg` handler, after the `themeOverlay.SetTheme` block:
```go
if a.helpOverlay != nil {
    a.helpOverlay.SetTheme(newTheme)
}
```

Verify:
```bash
go build ./...
```

Commit:
```bash
git add internal/app/app.go
git commit -m "feat(help): wire HelpOverlay lifecycle into app.go"
```

---

### Task 5: Wire into `internal/app/routing.go`

**Three changes:**

1. **Help overlay guard** — in `handleKeyMsg`, after the theme switcher guard block
   (`if a.showThemeSwitcher && a.themeOverlay != nil { ... }`), before the device overlay guard:
```go
if a.helpOpen && a.helpOverlay != nil {
    updated, cmd := a.helpOverlay.Update(m)
    if ho, ok := updated.(*panes.HelpOverlay); ok {
        a.helpOverlay = ho
    }
    return a, cmd
}
```

2. **`?` key handler** — after the `'t'` theme handler block:
```go
if m.Type == tea.KeyRunes && string(m.Runes) == "?" {
    if !a.searchOpen && !a.deviceOverlayOpen && !a.showThemeSwitcher && !a.helpOpen {
        return a.openHelp()
    }
    return a, nil
}
```

3. **Mouse guard** — in `handleMouseMsg`, update the existing guard:
```go
// Change:
if a.deviceOverlayOpen || a.searchOpen {
// To:
if a.deviceOverlayOpen || a.searchOpen || a.helpOpen {
```

Verify:
```bash
go build ./...
make test
```

Commit:
```bash
git add internal/app/routing.go
git commit -m "feat(help): add ? key handler and routing guards for help overlay"
```

---

### Task 6: Wire into `internal/app/render.go`

**Two changes:**

1. **New method** — add after `renderWithSearchOverlay`:
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

2. **`buildView()` branch** — add before `return body` (after the `searchOpen` check):
```go
if a.helpOpen && a.helpOverlay != nil {
    return a.renderWithHelpOverlay(body)
}
```

Verify:
```bash
go build ./...
make test
```

Commit:
```bash
git add internal/app/render.go
git commit -m "feat(help): add renderWithHelpOverlay and buildView branch"
```

---

### Task 7: Create `docs/keybinding.md`

Create the file with the following content:

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

Commit:
```bash
git add docs/keybinding.md
git commit -m "docs(keybinding): add canonical keybinding reference"
```

---

### Task 8: Update `docs/DESIGN.md` and `CLAUDE.md`

**`docs/DESIGN.md`** — in §17 Keybinding Table, find:
```
| `?` | Help (planned) | Global |
```
Replace with:
```
| `?` | Open help overlay | Global |
```

**`CLAUDE.md`** — add to "What Agents Must NEVER Do" list (item 15):
```
15. Add, change, or remove a keybinding without updating all three locations in the
    same commit: docs/keybinding.md, docs/DESIGN.md §17, and the helpContent var in
    internal/ui/panes/help_overlay.go.
```

**`CLAUDE.md`** — add new section after "Design Rules":
```markdown
## Keybinding Maintenance

All keybindings are documented in three places that must stay in sync:
- `docs/keybinding.md` — human-readable reference (canonical for external readers)
- `docs/DESIGN.md §17` — spec-level keybinding table
- `internal/ui/panes/help_overlay.go` `helpContent` var — in-app help overlay display

When adding, changing, or removing any keybinding, update all three in the same commit.
```

Commit:
```bash
git add docs/DESIGN.md CLAUDE.md
git commit -m "docs(keybinding): update DESIGN.md and add CLAUDE.md maintenance rules"
```

---

### Task 9: Full CI + PR

```bash
make ci
```

If coverage is below 80%, the most likely gap is `overlayWidth()` when terminal is
narrower than 78 cols. The `TestHelpOverlay_View_NarrowTerminal` test already covers
this path (`o.SetSize(60, 30)`).

Open PR:
```bash
git push origin feat/21-help-overlay
gh pr create \
  --title "feat(help): implement ? help overlay with keybinding reference" \
  --body "Closes feature 21. Implements HelpOverlay, wires ? key, creates docs/keybinding.md, adds CLAUDE.md maintenance rule."
```

---

## Acceptance Criteria

- [ ] `go test ./internal/ui/panes/ -run TestHelpOverlay` — all 11 tests pass
- [ ] `make ci` passes (lint + unit + integration + coverage ≥ 80%)
- [ ] Pressing `?` opens the centered help overlay with four grouped sections
- [ ] Pressing `Esc` closes the overlay; pressing any other key consumes it without side effects
- [ ] Pressing `?` while search / device / theme overlay is open does nothing
- [ ] Resizing the terminal while the overlay is open does not crash or misalign
- [ ] Switching theme while the overlay is open updates its colors immediately
- [ ] `docs/keybinding.md` exists with all keybinding sections
- [ ] `docs/DESIGN.md §17` no longer marks `?` as "planned"
- [ ] `CLAUDE.md` contains the "Keybinding Maintenance" section and rule #15
