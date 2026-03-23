# Bubble Tea Component Library Reference

## Table of Contents
- [Priority Rules](#priority-rules)
- [Tier 1: charmbracelet/bubbles (always prefer)](#tier-1-charmbracelletbubbles)
- [Tier 2: bubbletea-overlay (overlays/modals)](#tier-2-bubbletea-overlay)
- [Tier 3: bubble-table (advanced tables)](#tier-3-bubble-table)
- [Tier 4: bubbleup (notifications)](#tier-4-bubbleup)
- [Tier 5: bubbleo (navigation)](#tier-5-bubbleo)
- [Tier 6: bubbles-hlist (horizontal list)](#tier-6-bubbles-hlist)

## Priority Rules

1. **Always check charmbracelet/bubbles first.** If it provides the component, use it.
2. Only reach for a lower-tier library when bubbles doesn't cover the use case.
3. If two third-party libs provide similar functionality, prefer the higher-tier one.
4. For v2 of Bubbletea: lipgloss v2 has built-in compositing — prefer it over bubbletea-overlay.

---

## Tier 1: charmbracelet/bubbles

**Import:** `github.com/charmbracelet/bubbles/<component>`
**Stars:** 8k+ | **Status:** Official, production-ready, actively maintained

### Available Components

| Component | Import | Use Case |
|---|---|---|
| **spinner** | `bubbles/spinner` | Loading indicators. Multiple styles: Line, Dot, MiniDot, Jump, Pulse, Points, Globe, Moon, Monkey, Meter, Hamburger |
| **textinput** | `bubbles/textinput` | Single-line text input. Supports placeholder, char limit, width, echo mode (password), validation |
| **textarea** | `bubbles/textarea` | Multi-line text input. Character/line limits, word wrap |
| **table** | `bubbles/table` | Basic data table. Fixed columns, row selection, custom styles |
| **list** | `bubbles/list` | Vertical list with filtering, pagination, custom delegates. Wraps viewport internally |
| **viewport** | `bubbles/viewport` | Scrollable content area. Handles large content, supports mouse wheel |
| **progress** | `bubbles/progress` | Progress bar. Animated or static, gradient support |
| **paginator** | `bubbles/paginator` | Dot-style or Arabic numeral pagination |
| **filepicker** | `bubbles/filepicker` | File system browser |
| **cursor** | `bubbles/cursor` | Cursor blink model |
| **help** | `bubbles/help` | Key binding help view. Short and full modes |
| **key** | `bubbles/key` | Key binding definitions with help text |
| **timer** | `bubbles/timer` | Countdown timer |
| **stopwatch** | `bubbles/stopwatch` | Elapsed time counter |

### Common Patterns

**Spinner:**
```go
import "github.com/charmbracelet/bubbles/spinner"

s := spinner.New()
s.Spinner = spinner.Dot
s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
```

**Text Input:**
```go
import "github.com/charmbracelet/bubbles/textinput"

ti := textinput.New()
ti.Placeholder = "Search..."
ti.Focus()
ti.CharLimit = 100
ti.Width = 30
```

**List with custom delegate:**
```go
import "github.com/charmbracelet/bubbles/list"

items := []list.Item{item1, item2, item3}
l := list.New(items, list.NewDefaultDelegate(), width, height)
l.Title = "My Items"
l.SetFilteringEnabled(true)
```

**Viewport:**
```go
import "github.com/charmbracelet/bubbles/viewport"

vp := viewport.New(width, height)
vp.SetContent(longString)
// In Update: vp, cmd = vp.Update(msg)
```

**Key bindings with help:**
```go
import "github.com/charmbracelet/bubbles/key"
import "github.com/charmbracelet/bubbles/help"

type keyMap struct {
    Up   key.Binding
    Down key.Binding
    Quit key.Binding
}

var keys = keyMap{
    Up:   key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "move up")),
    Down: key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("j/down", "move down")),
    Quit: key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

// keys implements help.KeyMap via ShortHelp() and FullHelp()
h := help.New()
h.View(keys) // renders help text
```

---

## Tier 2: bubbletea-overlay

**Import:** `github.com/rmhubbert/bubbletea-overlay`
**Stars:** 117 | **Status:** Maintained, v1 only (v2 users should use lipgloss v2 compositing)

**Use when:** You need modal dialogs, overlays, or compositing a foreground view on a background.

**Note:** Overlay does NOT call `Update()` on wrapped models — manage updates yourself.

### Usage

```go
import overlay "github.com/rmhubbert/bubbletea-overlay"

// Create overlay model
o := overlay.New(
    foregroundModel,        // the modal/dialog
    backgroundModel,        // the main app behind it
    overlay.Center,         // horizontal position
    overlay.Center,         // vertical position
    0,                      // x offset
    0,                      // y offset
)

// Or use Composite directly for string-level compositing
output := overlay.Composite(fgString, bgString, xPos, yPos, xOffset, yOffset)
```

### Positioning

| Constant | Meaning |
|---|---|
| `overlay.Top` | Top edge |
| `overlay.Bottom` | Bottom edge |
| `overlay.Left` | Left edge |
| `overlay.Right` | Right edge |
| `overlay.Center` | Centered on axis |

Combine: `overlay.Right, overlay.Top` = top-right corner. Add offsets for fine-tuning.

---

## Tier 3: bubble-table

**Import:** `github.com/evertras/bubble-table/table`
**Stars:** 564 | **Status:** Maintained, feature-rich

**Use when:** You need sorting, filtering, flexible columns, horizontal scrolling,
selectable rows, or custom style functions — beyond what `bubbles/table` offers.

### Usage

```go
import "github.com/evertras/bubble-table/table"

columns := []table.Column{
    table.NewColumn("name", "Name", 20),
    table.NewFlexColumn("desc", "Description", 1), // flex!
    table.NewColumn("count", "Count", 10),
}

rows := []table.Row{
    table.NewRow(table.RowData{"name": "Alice", "desc": "Engineer", "count": 42}),
    table.NewRow(table.RowData{"name": "Bob", "desc": "Designer", "count": 17}),
}

t := table.New(columns).
    WithRows(rows).
    WithPageSize(10).
    Focused(true).
    SortByAsc("name").          // sort by column
    WithMaxTotalWidth(80).      // enables horizontal scrolling
    WithBaseStyle(myStyle)
```

### Key Features
- **Flex columns**: `NewFlexColumn` fills remaining space proportionally
- **Sorting**: `.SortByAsc(key)` / `.SortByDesc(key)` — numeric-aware
- **Filtering**: built-in text filter, custom filter functions
- **Selection**: `.WithMultiline(true)` for multi-select, `.SelectedRows()` to fetch
- **Horizontal scrolling**: set `.WithMaxTotalWidth()`, freeze left columns with `.WithFrozenColumnCount()`
- **Style functions**: per-row/cell styling via `.WithStyleFunc()`
- **Events**: check `table.UserEvent` messages for row select, page change

---

## Tier 4: bubbleup

**Import:** `github.com/daltonsw/bubbleup`
**Stars:** ~50 | **Status:** Maintained

**Use when:** You need toast-style notifications/alerts that float over your TUI.

### Usage

```go
import "github.com/daltonsw/bubbleup"

// In your model
type myModel struct {
    alert bubbleup.AlertModel
}

// Init: create alert model
m.alert = bubbleup.NewAlertModel(50, bubbleup.WithNerdFont(), 10*time.Second)
m.alert = m.alert.WithPosition(bubbleup.TopRightPosition)
m.alert = m.alert.WithAllowEscToClose()

// Trigger an alert (in Update, return this cmd)
cmd := bubbleup.NewAlertCmd(bubbleup.InfoAlert, "Data loaded successfully")

// In Update: always pass messages to alert model
m.alert, alertCmd = m.alert.Update(msg)

// In View: render alert on top of your content
return m.alert.ViewWithContent(m.mainView())
```

### Alert Types
- `bubbleup.InfoAlert` — blue/info style
- `bubbleup.WarningAlert` — yellow/warning
- `bubbleup.ErrorAlert` — red/error
- `bubbleup.DebugAlert` — gray/debug

### Positioning
`TopLeftPosition`, `TopCenterPosition`, `TopRightPosition`,
`BottomLeftPosition`, `BottomCenterPosition`, `BottomRightPosition`

### Font Prefixes
- `WithNerdFont()` — NerdFont icons
- `WithUnicodePrefix()` — Unicode symbols (portable)
- Default: ASCII prefixes

---

## Tier 5: bubbleo

**Import:** `github.com/KevM/bubbleo`
**Stars:** 69 | **Status:** Maintained

**Use when:** You need navigation stacks, menus with drill-down, breadcrumbs, or a shell wrapper.

### Components

**Navstack** — push/pop navigation between component models:
```go
import "github.com/KevM/bubbleo/navstack"

// Push a new screen
cmd := utils.Cmdize(navstack.PushNavigation{
    Item: navstack.NavigationItem{Title: "Details", Model: detailModel},
})

// Pop back
cmd := utils.Cmdize(navstack.PopNavigation{})

// Sequence: pop then send result to parent
cmd := tea.Sequence(
    utils.Cmdize(navstack.PopNavigation{}),
    utils.Cmdize(ResultMsg{Value: selected}),
)
```

**Menu** — wraps bubbles/list for selection with navigation:
```go
import "github.com/KevM/bubbleo/menu"
// Each choice has a title, description, and a Model to navigate to
```

**Breadcrumb** — shows navigation trail:
```go
import "github.com/KevM/bubbleo/breadcrumb"
// Auto-updates from navstack titles
```

**Shell** — combines navstack + breadcrumb:
```go
import "github.com/KevM/bubbleo/shell"
s := shell.New()
s.Navstack.Push(navstack.NavigationItem{Model: rootModel, Title: "Home"})
p := tea.NewProgram(s, tea.WithAltScreen())
```

### Key Pattern
`utils.Cmdize(msg)` wraps any value in a `tea.Cmd` (func returning tea.Msg).
Use `tea.Sequence` (not `tea.Batch`) for ordered navigation operations.

---

## Tier 6: bubbles-hlist

**Import:** `github.com/marcelblijleven/bubbles-hlist/hlist`
**Stars:** 3 | **Status:** New, based on bubbles/list

**Use when:** You need a horizontal list layout (side-by-side items instead of top-to-bottom).

### Usage

API mirrors `bubbles/list` with one addition — item cell width via the delegate:
```go
import "github.com/marcelblijleven/bubbles-hlist/hlist"

items := []list.Item{item1, item2, item3}
l := hlist.New(items, hlist.NewDefaultDelegate(), width, height)
// Navigation uses left/right instead of up/down
```

The delegate controls individual item cell width. Supports filtering, pagination,
and key help like the standard list.
