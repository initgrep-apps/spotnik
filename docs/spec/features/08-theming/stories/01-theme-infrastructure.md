---
title: "Theme Infrastructure and Five Built-in Themes"
feature: 01-theme
status: done
---

## Background
This story built the foundational theme system that every UI component in Spotnik depends on. It defined the `Theme` interface (23 methods covering backgrounds, borders, text hierarchy, selection, semantic colors, status bar, and metadata), created a registry with `Load()` and `Available()` functions, implemented five concrete themes with exact hex values, and wired the theme into the application startup path so panes receive it at construction and never call `Load()` themselves.

## Design

### Theme Interface Definition

```go
// internal/ui/theme/theme.go

package theme

import "github.com/charmbracelet/lipgloss"

// Theme defines all colour tokens used across the UI.
// Components call these methods -- they never use raw hex strings.
type Theme interface {
    // Backgrounds
    Base() lipgloss.Color       // App canvas background
    Surface() lipgloss.Color    // Pane interior background
    SurfaceAlt() lipgloss.Color // Overlay backgrounds (search, devices, help)

    // Borders
    ActiveBorder() lipgloss.Color   // Focused pane border
    InactiveBorder() lipgloss.Color // Unfocused pane borders

    // Text hierarchy
    TextPrimary() lipgloss.Color   // Main content -- track names, body text
    TextSecondary() lipgloss.Color // Supporting -- artist names, subtitles
    TextMuted() lipgloss.Color     // Dim -- timestamps, counts, hints

    // Selection
    SelectedBg() lipgloss.Color // Selected list item background
    SelectedFg() lipgloss.Color // Selected list item foreground

    // Semantic colours
    SectionHeader() lipgloss.Color    // Section labels: LIBRARY, QUEUE, NOW PLAYING
    PlayingIndicator() lipgloss.Color // Currently playing symbol
    SeekBar() lipgloss.Color          // Seek bar fill
    VolumeBar() lipgloss.Color        // Volume bar fill
    Success() lipgloss.Color          // Success states
    Warning() lipgloss.Color          // Caution notices
    Error() lipgloss.Color            // Error messages
    DeviceActive() lipgloss.Color     // Active device indicator

    // Status bar
    StatusBarBg() lipgloss.Color  // Status bar background
    StatusBarFg() lipgloss.Color  // Status bar body text
    KeyHint() lipgloss.Color      // Keybinding key labels (Space, Tab, etc.)

    // Metadata
    ID() string   // Config key: "black", "monokai", "catppuccin", "nord", "light"
    Name() string // Display name: "True Black", "Monokai", etc.
}
```

### Theme Registry

```go
// internal/ui/theme/theme.go (continued)

var registry = map[string]func() Theme{
    "black":      func() Theme { return &BlackTheme{} },
    "monokai":    func() Theme { return &MonokaiTheme{} },
    "catppuccin": func() Theme { return &CatppuccinTheme{} },
    "nord":       func() Theme { return &NordTheme{} },
    "light":      func() Theme { return &LightTheme{} },
}

const DefaultThemeID = "black"

// Load returns the theme for the given config ID.
// Falls back to DefaultThemeID if the ID is unknown.
func Load(id string) Theme {
    if constructor, ok := registry[id]; ok {
        return constructor()
    }
    return registry[DefaultThemeID]()
}

// Available returns all registered theme IDs in a stable order.
func Available() []string {
    return []string{"black", "monokai", "catppuccin", "nord", "light"}
}
```

### Theme File Structure

Each theme file follows this exact pattern. Example for Monokai:

```go
// internal/ui/theme/monokai.go

package theme

import "github.com/charmbracelet/lipgloss"

// MonokaiTheme implements Theme using the Monokai colour palette.
type MonokaiTheme struct{}

func (t *MonokaiTheme) ID() string   { return "monokai" }
func (t *MonokaiTheme) Name() string { return "Monokai" }

// Backgrounds
func (t *MonokaiTheme) Base() lipgloss.Color       { return "#272822" }
func (t *MonokaiTheme) Surface() lipgloss.Color    { return "#3e3d32" }
func (t *MonokaiTheme) SurfaceAlt() lipgloss.Color { return "#49483e" }

// Borders
func (t *MonokaiTheme) ActiveBorder() lipgloss.Color   { return "#66d9ef" }
func (t *MonokaiTheme) InactiveBorder() lipgloss.Color { return "#3e3d32" }

// Text
func (t *MonokaiTheme) TextPrimary() lipgloss.Color   { return "#f8f8f2" }
func (t *MonokaiTheme) TextSecondary() lipgloss.Color { return "#cfcfc2" }
func (t *MonokaiTheme) TextMuted() lipgloss.Color     { return "#75715e" }

// Selection
func (t *MonokaiTheme) SelectedBg() lipgloss.Color { return "#49483e" }
func (t *MonokaiTheme) SelectedFg() lipgloss.Color { return "#f8f8f2" }

// Semantic
func (t *MonokaiTheme) SectionHeader() lipgloss.Color    { return "#66d9ef" }
func (t *MonokaiTheme) PlayingIndicator() lipgloss.Color { return "#a6e22e" }
func (t *MonokaiTheme) SeekBar() lipgloss.Color          { return "#fd971f" }
func (t *MonokaiTheme) VolumeBar() lipgloss.Color        { return "#fd971f" }
func (t *MonokaiTheme) Success() lipgloss.Color          { return "#a6e22e" }
func (t *MonokaiTheme) Warning() lipgloss.Color          { return "#e6db74" }
func (t *MonokaiTheme) Error() lipgloss.Color            { return "#f92672" }
func (t *MonokaiTheme) DeviceActive() lipgloss.Color     { return "#66d9ef" }

// Status bar
func (t *MonokaiTheme) StatusBarBg() lipgloss.Color { return "#1e1f1c" }
func (t *MonokaiTheme) StatusBarFg() lipgloss.Color { return "#75715e" }
func (t *MonokaiTheme) KeyHint() lipgloss.Color     { return "#66d9ef" }
```

All five theme files follow this exact same structure -- only the hex values differ.

### All Theme Token Values (Original 23)

#### True Black (`black`)

| Method | Hex |
|---|---|
| `Base()` | `#000000` |
| `Surface()` | `#0f0f0f` |
| `SurfaceAlt()` | `#1a1a1a` |
| `ActiveBorder()` | `#00afff` |
| `InactiveBorder()` | `#1e1e1e` |
| `TextPrimary()` | `#f0f0f0` |
| `TextSecondary()` | `#888888` |
| `TextMuted()` | `#444444` |
| `SelectedBg()` | `#1c3a5e` |
| `SelectedFg()` | `#f0f0f0` |
| `SectionHeader()` | `#00afff` |
| `PlayingIndicator()` | `#00ff88` |
| `SeekBar()` | `#00afff` |
| `VolumeBar()` | `#00afff` |
| `Success()` | `#00ff88` |
| `Warning()` | `#ffcc00` |
| `Error()` | `#ff5555` |
| `DeviceActive()` | `#00e5cc` |
| `StatusBarBg()` | `#000000` |
| `StatusBarFg()` | `#444444` |
| `KeyHint()` | `#00afff` |

#### Monokai (`monokai`)

| Method | Hex |
|---|---|
| `Base()` | `#272822` |
| `Surface()` | `#3e3d32` |
| `SurfaceAlt()` | `#49483e` |
| `ActiveBorder()` | `#66d9ef` |
| `InactiveBorder()` | `#3e3d32` |
| `TextPrimary()` | `#f8f8f2` |
| `TextSecondary()` | `#cfcfc2` |
| `TextMuted()` | `#75715e` |
| `SelectedBg()` | `#49483e` |
| `SelectedFg()` | `#f8f8f2` |
| `SectionHeader()` | `#66d9ef` |
| `PlayingIndicator()` | `#a6e22e` |
| `SeekBar()` | `#fd971f` |
| `VolumeBar()` | `#fd971f` |
| `Success()` | `#a6e22e` |
| `Warning()` | `#e6db74` |
| `Error()` | `#f92672` |
| `DeviceActive()` | `#66d9ef` |
| `StatusBarBg()` | `#1e1f1c` |
| `StatusBarFg()` | `#75715e` |
| `KeyHint()` | `#66d9ef` |

#### Catppuccin Mocha (`catppuccin`)

| Method | Hex |
|---|---|
| `Base()` | `#1e1e2e` |
| `Surface()` | `#313244` |
| `SurfaceAlt()` | `#45475a` |
| `ActiveBorder()` | `#89b4fa` |
| `InactiveBorder()` | `#313244` |
| `TextPrimary()` | `#cdd6f4` |
| `TextSecondary()` | `#bac2de` |
| `TextMuted()` | `#6c7086` |
| `SelectedBg()` | `#b4befe` |
| `SelectedFg()` | `#1e1e2e` |
| `SectionHeader()` | `#cba6f7` |
| `PlayingIndicator()` | `#a6e3a1` |
| `SeekBar()` | `#fab387` |
| `VolumeBar()` | `#fab387` |
| `Success()` | `#a6e3a1` |
| `Warning()` | `#f9e2af` |
| `Error()` | `#f38ba8` |
| `DeviceActive()` | `#94e2d5` |
| `StatusBarBg()` | `#11111b` |
| `StatusBarFg()` | `#a6adc8` |
| `KeyHint()` | `#89dceb` |

#### Nord (`nord`)

| Method | Hex |
|---|---|
| `Base()` | `#2e3440` |
| `Surface()` | `#3b4252` |
| `SurfaceAlt()` | `#434c5e` |
| `ActiveBorder()` | `#88c0d0` |
| `InactiveBorder()` | `#3b4252` |
| `TextPrimary()` | `#eceff4` |
| `TextSecondary()` | `#d8dee9` |
| `TextMuted()` | `#4c566a` |
| `SelectedBg()` | `#4c566a` |
| `SelectedFg()` | `#eceff4` |
| `SectionHeader()` | `#88c0d0` |
| `PlayingIndicator()` | `#a3be8c` |
| `SeekBar()` | `#81a1c1` |
| `VolumeBar()` | `#81a1c1` |
| `Success()` | `#a3be8c` |
| `Warning()` | `#ebcb8b` |
| `Error()` | `#bf616a` |
| `DeviceActive()` | `#8fbcbb` |
| `StatusBarBg()` | `#242831` |
| `StatusBarFg()` | `#4c566a` |
| `KeyHint()` | `#88c0d0` |

#### Light -- Catppuccin Latte (`light`)

| Method | Hex |
|---|---|
| `Base()` | `#eff1f5` |
| `Surface()` | `#e6e9ef` |
| `SurfaceAlt()` | `#dce0e8` |
| `ActiveBorder()` | `#1e66f5` |
| `InactiveBorder()` | `#ccd0da` |
| `TextPrimary()` | `#4c4f69` |
| `TextSecondary()` | `#6c6f85` |
| `TextMuted()` | `#9ca0b0` |
| `SelectedBg()` | `#c6d0f5` |
| `SelectedFg()` | `#4c4f69` |
| `SectionHeader()` | `#1e66f5` |
| `PlayingIndicator()` | `#40a02b` |
| `SeekBar()` | `#fe640b` |
| `VolumeBar()` | `#fe640b` |
| `Success()` | `#40a02b` |
| `Warning()` | `#df8e1d` |
| `Error()` | `#d20f39` |
| `DeviceActive()` | `#179299` |
| `StatusBarBg()` | `#dce0e8` |
| `StatusBarFg()` | `#6c6f85` |
| `KeyHint()` | `#1e66f5` |

### How Theme Is Loaded at Startup

```go
// internal/app/app.go -- during initialisation

func New(cfg *config.Config, client api.SpotifyClient) *App {
    t := theme.Load(cfg.UI.Theme) // loads "monokai", "black", etc.

    return &App{
        client: client,
        store:  state.NewStore(),
        theme:  t,
        // panes receive the theme at construction -- they never load it themselves
        library: panes.NewLibraryPane(store, t),
        player:  panes.NewPlayerPane(store, t),
        queue:   panes.NewQueuePane(store, t),
    }
}
```

The theme is passed into panes at construction. Panes store a reference to the `Theme` interface and call its methods in `View()`. They never call `theme.Load()` themselves and never store raw hex strings.

### How Components Use the Theme

```go
// internal/ui/panes/player.go -- example usage

type PlayerPane struct {
    store *state.Store
    theme theme.Theme  // stored as interface, not a concrete type
}

func (p PlayerPane) View() string {
    // Use theme tokens -- never raw hex
    titleStyle := lipgloss.NewStyle().
        Bold(true).
        Foreground(p.theme.TextPrimary())

    artistStyle := lipgloss.NewStyle().
        Foreground(p.theme.TextSecondary())

    borderStyle := lipgloss.NewStyle().
        BorderForeground(p.theme.ActiveBorder())

    // render using styles...
}
```

### Theme Storage

| Concern | Where | Notes |
|---|---|---|
| Theme selection | `~/.config/spotnik/config.toml` | `theme = "monokai"` |
| Theme code | `internal/ui/theme/*.go` | One file per theme |
| Active theme instance | In-memory, passed at construction | Not persisted |
| Runtime switching | Not in MVP | Restart required to change theme |

### Files Created

| File | Purpose |
|---|---|
| `internal/ui/theme/theme.go` | Interface, registry, `Load()`, `Available()` |
| `internal/ui/theme/black.go` | True Black theme |
| `internal/ui/theme/monokai.go` | Monokai theme |
| `internal/ui/theme/catppuccin.go` | Catppuccin Mocha theme |
| `internal/ui/theme/nord.go` | Nord theme |
| `internal/ui/theme/light.go` | Light theme (Catppuccin Latte) |
| `internal/ui/theme/theme_test.go` | Interface compliance + registry tests |

### Out of Scope

- Runtime theme switching without restart
- Custom user-defined themes via config
- Per-component theme overrides
- Theme preview in the UI

## Acceptance Criteria
- [ ] All five themes compile and implement the full `Theme` interface (23 methods each)
- [ ] `Load()` returns the correct concrete theme for every known ID
- [ ] Unknown theme IDs never panic -- `Load()` always falls back to the default
- [ ] `DefaultThemeID` is `"black"` and can never be empty
- [ ] No component file contains a raw hex colour string -- all colour comes from `Theme` methods
- [ ] `theme = "monokai"` in config.toml results in Monokai colours being used throughout the UI
- [ ] Theme is injected at startup and passed to all pane constructors -- panes never call `Load()` themselves
- [ ] 100% test coverage on `theme.go` (registry, load, fallback)

## Tasks
- [ ] Define `Theme` interface with all 21 colour methods + `ID()` + `Name()`, create registry, `Load()`, `Available()`, `DefaultThemeID`
      - test: `TestLoad_KnownID`, `TestLoad_UnknownID_FallsBackToDefault`, `TestLoad_DefaultTheme`, `TestAvailable_Returns5Entries`, `TestAvailable_StableOrder`
- [ ] Implement all five themes (`black.go`, `monokai.go`, `catppuccin.go`, `nord.go`, `light.go`) with exact hex values
      - test: `TestAllThemes_ImplementInterface`, `TestBlackTheme_Base_IsPureBlack`, `TestMonokaiTheme_Base`, `TestCatppuccinTheme_Base`, `TestNordTheme_Base`, `TestLightTheme_Base`, `TestAllThemes_IDMatchesRegistryKey`
- [ ] Wire theme into app startup: `cmd/root.go` reads config, calls `theme.Load()`, passes to `app.New()` and pane constructors
      - test: `TestAppNew_ReceivesTheme`, `TestAppNew_DefaultThemeFallback`
