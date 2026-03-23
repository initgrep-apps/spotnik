# Bubble Tea v2 Migration Guide

## Migration Checklist

- [ ] Update import paths
- [ ] Change `View() string` to `View() tea.View`
- [ ] Replace `tea.KeyMsg` with `tea.KeyPressMsg`
- [ ] Update key fields: `msg.Type` / `msg.Runes` / `msg.Alt`
- [ ] Replace `case " ":` with `case "space":`
- [ ] Update mouse message usage
- [ ] Remove old program options → use View fields
- [ ] Remove imperative commands → use View fields
- [ ] Rename `tea.WindowSize()` → `tea.RequestWindowSize`
- [ ] Replace `tea.Sequentially(...)` → `tea.Sequence(...)`

## Import Paths

```go
// v1
import tea "github.com/charmbracelet/bubbletea"
import "github.com/charmbracelet/lipgloss"

// v2
import tea "charm.land/bubbletea/v2"
import "charm.land/lipgloss/v2"
```

## View Returns `tea.View`

The biggest change: `View()` returns a `tea.View` struct, not a string.
Terminal features (alt screen, mouse, cursor) are now declarative View fields.

```go
// v1
func (m model) View() string {
    return "Hello, world!"
}

// v2
func (m model) View() tea.View {
    return tea.NewView("Hello, world!")
}

// v2 with fields
func (m model) View() tea.View {
    var v tea.View
    v.SetContent("Hello, world!")
    v.AltScreen = true
    v.MouseMode = tea.MouseModeCellMotion
    v.WindowTitle = "My App"
    return v
}
```

### View Fields

| Field | What It Does |
|---|---|
| `Content` | The rendered string (via `SetContent()` or `NewView()`) |
| `AltScreen` | Enter/exit alternate screen buffer |
| `MouseMode` | `MouseModeNone`, `MouseModeCellMotion`, `MouseModeAllMotion` |
| `ReportFocus` | Enable focus/blur event reporting |
| `DisableBracketedPasteMode` | Disable bracketed paste |
| `WindowTitle` | Set terminal window title |
| `Cursor` | Cursor position, shape, color, blink |
| `ForegroundColor` | Terminal foreground color |
| `BackgroundColor` | Terminal background color |
| `KeyboardEnhancements` | Request keyboard enhancement features |

## Key Messages

### `tea.KeyMsg` → `tea.KeyPressMsg`

```go
// v1
case tea.KeyMsg:
    switch msg.String() {
    case "q": return m, tea.Quit
    }

// v2
case tea.KeyPressMsg:
    switch msg.String() {
    case "q": return m, tea.Quit
    }
```

`tea.KeyMsg` is now an interface covering both presses and releases.
For most code, use `tea.KeyPressMsg`. For key releases: `tea.KeyReleaseMsg`.

### Key Field Changes

| v1 | v2 | Notes |
|---|---|---|
| `msg.Type` | `msg.Code` | A `rune` — `tea.KeyEnter`, `'a'`, etc. |
| `msg.Runes` | `msg.Text` | Now `string`, not `[]rune` |
| `msg.Alt` | `msg.Mod` | `msg.Mod.Contains(tea.ModAlt)` |
| `tea.KeyRune` | — | Check `len(msg.Text) > 0` |
| `tea.KeyCtrlC` | — | Use `msg.String() == "ctrl+c"` |

### Space Bar

```go
// v1
case " ":

// v2
case "space":
```

`key.Code` is still `' '` and `key.Text` is still `" "`, but `String()` returns `"space"`.

### Ctrl+Key

```go
// v1
case tea.KeyCtrlC:

// v2 (string matching)
case tea.KeyPressMsg:
    switch msg.String() {
    case "ctrl+c":
    }

// v2 (field matching)
case tea.KeyPressMsg:
    if msg.Code == 'c' && msg.Mod.Contains(tea.ModCtrl) { ... }
```

## Removed Program Options → View Fields

These program options no longer exist. Set them as View fields instead:

| Removed Option | View Field Replacement |
|---|---|
| `tea.WithAltScreen()` | `v.AltScreen = true` |
| `tea.WithMouseCellMotion()` | `v.MouseMode = tea.MouseModeCellMotion` |
| `tea.WithMouseAllMotion()` | `v.MouseMode = tea.MouseModeAllMotion` |
| `tea.WithReportFocus()` | `v.ReportFocus = true` |
| `tea.WithoutBracketedPaste()` | `v.DisableBracketedPasteMode = true` |

## Removed Commands → View Fields

| Removed Command | View Field Replacement |
|---|---|
| `tea.EnterAltScreen` | `v.AltScreen = true` |
| `tea.ExitAltScreen` | `v.AltScreen = false` |
| `tea.EnableMouseCellMotion` | `v.MouseMode = tea.MouseModeCellMotion` |
| `tea.EnableMouseAllMotion` | `v.MouseMode = tea.MouseModeAllMotion` |
| `tea.DisableMouse` | `v.MouseMode = tea.MouseModeNone` |
| `tea.EnableBracketedPaste` | `v.DisableBracketedPasteMode = false` |
| `tea.DisableBracketedPaste` | `v.DisableBracketedPasteMode = true` |
| `tea.EnableReportFocus` | `v.ReportFocus = true` |
| `tea.DisableReportFocus` | `v.ReportFocus = false` |
| `tea.SetWindowTitle(s)` | `v.WindowTitle = s` |
| `tea.ShowCursor` | `v.Cursor.Visible = true` |
| `tea.HideCursor` | `v.Cursor.Visible = false` |

## Renamed APIs

| v1 | v2 |
|---|---|
| `tea.WindowSize()` (Cmd) | `tea.RequestWindowSize` |
| `tea.Sequentially(...)` | `tea.Sequence(...)` |

## Mouse Messages

Mouse events use new types in v2:
- `tea.MouseClickMsg` — click events
- `tea.MouseReleaseMsg` — release events
- `tea.MouseWheelMsg` — scroll wheel
- `tea.MouseMotionMsg` — motion events

Button constants renamed:
- `tea.MouseLeft` → `tea.MouseButtonLeft`
- `tea.MouseRight` → `tea.MouseButtonRight`
- `tea.MouseMiddle` → `tea.MouseButtonMiddle`

## Lipgloss v2 Compositing

Lipgloss v2 has built-in compositing, replacing the need for bubbletea-overlay in v2 apps:

```go
import "charm.land/lipgloss/v2"

// Overlay foreground on background
result := lipgloss.Place(bgWidth, bgHeight, lipgloss.Center, lipgloss.Center, fgContent,
    lipgloss.WithWhitespaceBackground(bgContent),
)
```
