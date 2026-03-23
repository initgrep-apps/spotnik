---
name: bubbletea
description: |
  Build terminal UIs with Bubble Tea (Go, Elm Architecture). Use when writing
  or modifying Bubble Tea TUI apps: Model/Update/View patterns, Cmd/Msg flow,
  key handling, tick polling, component composition, focus routing, overlays,
  tables, notifications, navigation. Includes component selection hierarchy
  and v2 migration guide. Triggers on: bubbletea, tea.Model, tea.Cmd, tea.Msg,
  Bubble Tea, TUI, terminal UI, pane, overlay, bubbles, lipgloss.
---

# Bubble Tea Development Guide

## Core Architecture (Elm Architecture)

Every Bubble Tea app has three parts:

```go
type Model interface {
    Init() tea.Cmd                           // initial command (or nil)
    Update(msg tea.Msg) (tea.Model, tea.Cmd) // handle events, return updated model + next command
    View() string                            // pure render — no side effects, no I/O
}
```

**Rules:**
- `View()` must be pure — read state, return string, nothing else
- Side effects only via `tea.Cmd` — never call APIs inside `Update()` directly
- Messages are typed structs — never use strings or constants as message types
- Always return the model from `Update()`, even if unchanged

## Cmd / Msg Flow

```
User Input / Timer / I/O result
        ↓
    tea.Msg (typed struct)
        ↓
    Update(msg) → (Model, tea.Cmd)
                        ↓
                  tea.Cmd executes I/O
                        ↓
                  returns tea.Msg → back to Update
```

**Creating commands:**

```go
// One-shot command returning a message
func fetchData() tea.Msg {
    resp, err := http.Get(url)
    if err != nil { return errMsg{err} }
    return dataMsg{resp}
}

// Use in Update: return m, fetchData  (pass the func, don't call it)

// Tick (periodic polling)
func tickCmd() tea.Cmd {
    return tea.Tick(time.Second, func(t time.Time) tea.Msg {
        return tickMsg(t)
    })
}

// Batch (multiple commands concurrently)
return m, tea.Batch(cmd1, cmd2, cmd3)

// Sequence (commands in order, each waits for previous)
return m, tea.Sequence(cmd1, cmd2)
```

## Key Handling (v1)

```go
case tea.KeyMsg:
    switch msg.String() {
    case "ctrl+c", "q":
        return m, tea.Quit
    case "up", "k":
        m.cursor--
    case "enter", " ":
        // action
    case "tab":
        m.focusNext()
    }
```

Key constants: `"enter"`, `"tab"`, `"shift+tab"`, `"up"`, `"down"`, `"left"`, `"right"`,
`"esc"`, `"backspace"`, `"ctrl+c"`, `"ctrl+z"`, `" "` (space in v1).

## Composing Models (Parent-Child Pattern)

```go
type RootModel struct {
    paneA  PaneAModel
    paneB  PaneBModel
    focus  int
}

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd

    // Route global keys at root level
    if key, ok := msg.(tea.KeyMsg); ok {
        switch key.String() {
        case "tab":
            m.focus = (m.focus + 1) % 2
            return m, nil
        }
    }

    // Forward to focused child only
    switch m.focus {
    case 0:
        updated, cmd := m.paneA.Update(msg)
        m.paneA = updated.(PaneAModel)
        cmds = append(cmds, cmd)
    case 1:
        updated, cmd := m.paneB.Update(msg)
        m.paneB = updated.(PaneBModel)
        cmds = append(cmds, cmd)
    }
    return m, tea.Batch(cmds...)
}
```

**Focus routing rules:**
- Root model owns focus state and key routing
- Children never talk to each other — communicate via messages through root
- Only forward input events to the focused child
- Broadcast non-input messages (window size, tick, data loaded) to all children

## Overlay Pattern

For modal dialogs / overlays on top of a main view:

```go
type RootModel struct {
    main       MainModel
    overlay    OverlayModel
    showOverlay bool
}

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if m.showOverlay {
        // Overlay captures all input when visible
        updated, cmd := m.overlay.Update(msg)
        m.overlay = updated.(OverlayModel)
        return m, cmd
    }
    // Normal routing when no overlay
    updated, cmd := m.main.Update(msg)
    m.main = updated.(MainModel)
    return m, cmd
}
```

## Tick Polling Pattern

```go
type TickMsg time.Time

func (m Model) Init() tea.Cmd {
    return tea.Tick(time.Second, func(t time.Time) tea.Msg {
        return TickMsg(t)
    })
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case TickMsg:
        // Re-arm the tick AND fetch fresh data
        return m, tea.Batch(
            tea.Tick(time.Second, func(t time.Time) tea.Msg { return TickMsg(t) }),
            fetchPlaybackState,
        )
    }
    return m, nil
}
```

## Debounce Pattern (e.g., Search)

```go
type searchTickMsg struct{}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        m.query += msg.String()
        // Reset debounce timer on each keystroke
        return m, tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
            return searchTickMsg{}
        })
    case searchTickMsg:
        // Only fires if no new keystroke arrived in 300ms
        return m, performSearch(m.query)
    }
    return m, nil
}
```

## Window Sizing

```go
case tea.WindowSizeMsg:
    m.width = msg.Width
    m.height = msg.Height
    // Propagate to children
    m.paneA.SetSize(msg.Width/2, msg.Height)
    m.paneB.SetSize(msg.Width/2, msg.Height)
```

## Program Setup

```go
p := tea.NewProgram(
    initialModel(),
    tea.WithAltScreen(),       // full-screen mode
    tea.WithMouseCellMotion(), // mouse support
)
if _, err := p.Run(); err != nil {
    log.Fatal(err)
}
```

## Debugging

```go
if os.Getenv("DEBUG") != "" {
    f, err := tea.LogToFile("debug.log", "debug")
    if err != nil { log.Fatal(err) }
    defer f.Close()
}
// Then: tail -f debug.log in another terminal
```

## Testing Bubble Tea Models

```go
func TestUpdate(t *testing.T) {
    m := initialModel()
    // Simulate a key press
    updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
    model := updated.(Model)
    assert.Equal(t, 1, model.cursor)
    assert.Nil(t, cmd)
}
```

Test commands by checking the returned `tea.Cmd` is non-nil, then execute it
to get the resulting `tea.Msg` and verify the message type/content.

## Component Selection

When you need a UI component, check [references/components.md](references/components.md) for the
priority-ordered library list. Key rule: **always prefer charmbracelet/bubbles first**.
Only reach for third-party components when bubbles doesn't cover the use case.

**Quick decision guide:**
- Spinner, text input, text area, viewport, list, help, progress, paginator, cursor, timer → **bubbles**
- Basic table → **bubbles/table**; advanced (sorting, filtering, flex columns) → **bubble-table**
- Overlays / modals → **bubbletea-overlay** (v1) or **lipgloss v2 compositing** (v2)
- Toast notifications → **bubbleup**
- Navigation stack, menu, breadcrumb → **bubbleo**
- Horizontal list → **bubbles-hlist**

## API Quick Reference

For full type signatures, constants, and Program options, see [references/api-reference.md](references/api-reference.md).

## V2 Migration

If upgrading from v1 to v2, see [references/v2-migration.md](references/v2-migration.md) for the
complete checklist of breaking changes.
