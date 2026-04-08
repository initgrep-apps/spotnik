# Help Overlay Design Spec
*Date: 2026-04-08*

## Summary

Implement the `?` keybinding to open a centered help overlay showing all app keybindings grouped by category. Currently `?` appears in the status bar as "help" but has no handler — pressing it does nothing. This spec covers the full implementation from overlay struct to routing and render wiring.

---

## Problem

`?` is declared in `appKeyMap` (`internal/app/render.go:76`) and rendered in the status bar, but `routing.go` has no handler for it and no `HelpOverlay` struct exists. `docs/DESIGN.md §17` marks it `"Help (planned)"`.

---

## Decisions

| Question | Decision | Reason |
|---|---|---|
| Content scope | Global + pane-specific keybindings | A help overlay should be a complete reference |
| Data source | Static package-level var | `Actions()` is context-sensitive (changes when filter is active), not suitable for a stable reference |
| Layout | Two-column side-by-side | Maximum density, most scannable |
| Position | Centered (`btoverlay.Center, Center`) | Appropriate for a full-attention reference modal |
| Dismiss | `Esc` only | Consistent with all other overlays; avoids `?`-toggle quirk |
| Keybinding doc | `docs/keybinding.md` created and kept in sync | Single source of truth for keybindings outside the codebase; enforced via CLAUDE.md rule |

---

## Architecture

### New Files

| File | Purpose |
|---|---|
| `internal/ui/panes/help_overlay.go` | `HelpOverlay` struct, static keybinding data, `View()`, `Update()` |
| `internal/ui/panes/help_overlay_test.go` | Unit tests |

### Modified Files

| File | Change |
|---|---|
| `internal/app/app.go` | `helpOpen bool` + `helpOverlay *panes.HelpOverlay` fields; `openHelp()` / `closeHelp()` helpers; `HelpOverlayClosedMsg` handler in `handleMsg()` |
| `internal/app/routing.go` | Guard at top of `handleKeyMsg()` routing all keys to helpOverlay when open; `?` key handler after existing global handlers |
| `internal/app/render.go` | `renderWithHelpOverlay()` method; `helpOpen` branch in `buildView()` |
| `docs/DESIGN.md` | Update `?` row from `"Help (planned)"` to `"Open help overlay"` |

---

## Data Model

```go
// helpBinding is a single key → label pair.
type helpBinding struct{ key, label string }

// helpSection groups related bindings under a titled header.
type helpSection struct {
    title    string
    bindings []helpBinding
}

// helpContent is the static two-column keybinding reference.
// Index 0 = left column, index 1 = right column.
var helpContent = [2][]helpSection{
    // Left column: Global + Navigation
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
    // Right column: Playback + Pane Actions
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
```

---

## HelpOverlay Struct

```go
// HelpOverlayClosedMsg is emitted when the user presses Esc in the HelpOverlay.
type HelpOverlayClosedMsg struct{}

// HelpOverlay is the floating help reference overlay.
// It renders all app keybindings grouped into two columns.
// Pressing Esc emits HelpOverlayClosedMsg; all other keys are consumed silently.
type HelpOverlay struct {
    theme  theme.Theme
    width  int
    height int
}

func NewHelpOverlay(th theme.Theme) *HelpOverlay
func (o *HelpOverlay) SetSize(width, height int)
func (o *HelpOverlay) SetTheme(th theme.Theme)
func (o *HelpOverlay) Init() tea.Cmd
func (o *HelpOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (o *HelpOverlay) View() string
```

`Update()` handles `tea.KeyMsg` only:
- `Esc` → emit `HelpOverlayClosedMsg{}`
- All other keys → consumed silently (no passthrough — overlay is modal)

---

## View Layout

Fixed overlay width: **78 columns** (each content column: 35 cols, divider: 1 col, borders: 2 cols, padding: 4 cols). Height is content-driven (~18 lines).

```
╭─ Help ─────────────────────────────────────────────────────────────────────╮
│  Global                          │  Playback                               │
│  /          search               │  Space      play / pause                │
│  d          devices              │  n          next track                  │
│  t          theme                │  ← / →      prev / next                 │
│  ?          help                 │  + / -      volume                      │
│  q          quit                 │  s          shuffle                     │
│  0          toggle page          │  r          repeat                      │
│  1-8        toggle pane          │  v          visualizer                  │
│  p          preset               │                                         │
│                                  │  Pane Actions                           │
│  Navigation                      │  Enter      select / play               │
│  Tab        next pane            │  f          filter                      │
│  Shift+Tab  prev pane            │  A          add to queue                │
│  j / k      scroll               │  i          like / unlike               │
│  Esc        close overlay        │  x          remove track                │
│                                  │  Shift+↑/↓  reorder (playlists)         │
╰────────────────────────────────────────────────────────────────────────────╯
```

### Color tokens (no hardcoded hex)

| Element | Token |
|---|---|
| Border + title | `theme.ActiveBorder()` |
| Section headers | `theme.Info()` |
| Key names | `theme.TextPrimary()` |
| Key labels | `theme.TextMuted()` |
| Column divider `│` | `theme.TextMuted()` |
| Border actions bar | standard `layout.RenderPaneBorder` conventions |

---

## App Wiring

### app.go fields
```go
helpOpen    bool
helpOverlay *panes.HelpOverlay
```

### openHelp / closeHelp
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

### handleMsg addition
```go
case panes.HelpOverlayClosedMsg:
    return a.closeHelp()
```

### WindowSizeMsg propagation
```go
if a.helpOverlay != nil {
    a.helpOverlay.SetSize(m.Width, m.Height)
}
```

### Theme propagation
```go
if a.helpOverlay != nil {
    a.helpOverlay.SetTheme(newTheme)
}
```

---

## Routing (routing.go)

### Guard — first in handleKeyMsg(), before all others except theme switcher
```go
if a.helpOpen && a.helpOverlay != nil {
    updated, cmd := a.helpOverlay.Update(m)
    if ho, ok := updated.(*panes.HelpOverlay); ok {
        a.helpOverlay = ho
    }
    return a, cmd
}
```

### handleMouseMsg guard
Add `a.helpOpen` to the existing mouse event guard in `handleMouseMsg`:
```go
if a.deviceOverlayOpen || a.searchOpen || a.helpOpen {
    return nil
}
```

### `?` key handler — after existing global key handlers
```go
if m.Type == tea.KeyRunes && string(m.Runes) == "?" {
    if !a.searchOpen && !a.deviceOverlayOpen && !a.showThemeSwitcher && !a.helpOpen {
        return a.openHelp()
    }
    return a, nil
}
```

---

## Render (render.go)

### buildView() branch (after theme switcher check, before device overlay check)
```go
if a.helpOpen && a.helpOverlay != nil {
    return a.renderWithHelpOverlay(body)
}
```

### renderWithHelpOverlay
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

---

## Messages

`HelpOverlayClosedMsg` is added to `internal/ui/panes/messages.go` alongside the other overlay closed messages.

---

## Testing

### help_overlay_test.go

| Test | What it verifies |
|---|---|
| `TestHelpOverlay_View_ContainsGlobalKeys` | Output contains `/`, `d`, `t`, `q`, `?` key strings |
| `TestHelpOverlay_View_ContainsPlaybackKeys` | Output contains `Space`, `n`, `s`, `r`, `v` |
| `TestHelpOverlay_View_ContainsSectionHeaders` | Output contains "Global", "Playback", "Navigation", "Pane Actions" |
| `TestHelpOverlay_View_HasBorder` | Output contains `╭` and `╰` (rounded corners) |
| `TestHelpOverlay_Update_EscEmitsClosedMsg` | `Esc` key returns `HelpOverlayClosedMsg{}` as cmd result |
| `TestHelpOverlay_Update_OtherKeysConsumed` | `j`, `k`, `q`, `Enter` return nil cmd (consumed, not passed through) |
| `TestHelpOverlay_SetTheme` | `SetTheme()` updates internal theme reference without panic |
| `TestHelpOverlay_SetSize` | `SetSize()` stores dimensions without panic |

---

## DESIGN.md Update

In §17 Keybinding Table, change:

```
| `?` | Help (planned) | Global |
```

to:

```
| `?` | Open help overlay | Global |
```

---

## docs/keybinding.md — New File

Create `docs/keybinding.md` as a human-readable, always-current reference of every keybinding in the app. This file is the canonical external-facing keybinding reference. It mirrors the `helpContent` static data in `help_overlay.go` and DESIGN.md §17, but in a standalone doc that is easy to link to and share.

### Structure

```markdown
# Spotnik Keybindings

## Global
| Key | Action |
|-----|--------|
| `/` | Open search overlay |
| `d` | Open device switcher |
| `t` | Open theme switcher |
| `?` | Open help overlay |
| `q` | Quit |
| `0` | Toggle Page A / Page B |
| `1`–`8` | Toggle pane visibility (Page A) |
| `p` | Cycle preset |

## Playback
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
| `A` | Add to queue | Search, list panes |
| `i` | Like / unlike track | LikedSongs pane |
| `x` | Remove track | Playlists track sub-view |
| `Shift+↑` / `Shift+↓` | Reorder track | Playlists track sub-view |

## Search Overlay
| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Cycle search category |
| `Enter` | Play selected result |
| `Ctrl+A` | Add result to queue |
| `Ctrl+U` | Clear search input |
| `PgDn` / `PgUp` | Next / previous result page |
| `Esc` | Close search |
```

### Maintenance rule (added to CLAUDE.md)

Any time a keybinding is **added, changed, or removed** — anywhere in the codebase — the implementer must update all three of these in the same commit:

1. `docs/keybinding.md` — the human-readable reference
2. `docs/DESIGN.md §17` — the spec table
3. The static `helpContent` var in `internal/ui/panes/help_overlay.go` — the in-app display

---

## CLAUDE.md Update

Add the following rule to the **"What Agents Must NEVER Do"** list:

```
15. Add, change, or remove a keybinding without updating all three: docs/keybinding.md,
    docs/DESIGN.md §17, and the helpContent var in internal/ui/panes/help_overlay.go.
```

And add a new section near the "Design Rules" block:

```markdown
## Keybinding Maintenance

All keybindings are documented in three places that must stay in sync:
- `docs/keybinding.md` — human-readable reference (canonical for external readers)
- `docs/DESIGN.md §17` — spec-level keybinding table
- `internal/ui/panes/help_overlay.go` `helpContent` — in-app help overlay display

When adding, changing, or removing any keybinding, update all three in the same commit.
```
