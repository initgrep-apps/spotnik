---
title: "Theme System"
description: "Provides a token-based color theming infrastructure with five built-in themes (True Black, Monokai, Catppuccin, Nord, Light) so every UI component renders with consistent, configurable colors without hardcoding hex values."
status: done
stories: [01, 40]
---

# Theme System

## Background

Spotnik is a terminal Spotify client built for developers who live in the terminal. Visual consistency matters -- every pane, overlay, status bar, and component must draw its colors from a single, centralized theme system rather than scattering hex values across the codebase. The Theme interface is the foundation that every UI component depends on.

The initial theme system (spec 01) established the core infrastructure: a `Theme` interface with 23 color token methods, a registry/loader that maps config IDs to concrete implementations, five built-in themes, and the startup wiring that injects the active theme into every pane constructor. Components call theme methods in `View()` and never store raw hex strings.

As the UI evolved toward a btop-inspired redesign with gradient bars, an audio visualizer, dense table panes, and per-pane colored borders, 16 additional color tokens were needed (spec 40). These tokens -- for gradients, visualizer foreground, table headers, preset indicators, and 10 per-pane border accents -- were added to the Theme interface and implemented across all five themes, bringing the total to 42 methods.

---

## Story: Theme Infrastructure and Five Built-in Themes (spec 01)

### Background
This story built the foundational theme system that every UI component in Spotnik depends on. It defined the `Theme` interface (23 methods covering backgrounds, borders, text hierarchy, selection, semantic colors, status bar, and metadata), created a registry with `Load()` and `Available()` functions, implemented five concrete themes with exact hex values, and wired the theme into the application startup path so panes receive it at construction and never call `Load()` themselves.

### Acceptance Criteria
- [ ] All five themes compile and implement the full `Theme` interface (23 methods each)
- [ ] `Load()` returns the correct concrete theme for every known ID
- [ ] Unknown theme IDs never panic -- `Load()` always falls back to the default
- [ ] `DefaultThemeID` is `"black"` and can never be empty
- [ ] No component file contains a raw hex colour string -- all colour comes from `Theme` methods
- [ ] `theme = "monokai"` in config.toml results in Monokai colours being used throughout the UI
- [ ] Theme is injected at startup and passed to all pane constructors -- panes never call `Load()` themselves
- [ ] 100% test coverage on `theme.go` (registry, load, fallback)

### Tasks

1. **Task 0b.1 -- Theme interface + registry** -- Define the `Theme` interface that every UI component depends on, plus the registry and loader that map config IDs to concrete implementations. This is the foundation -- nothing else in the theme system works without it.
   - Files: `internal/ui/theme/theme.go`, `internal/ui/theme/theme_test.go`
   - Implementation steps:
     - Define `Theme` interface with all 21 colour methods + `ID()` + `Name()`
     - Create `registry` map and `Load(id string) Theme` function
     - Create `Available() []string` returning stable ordered list
     - Create `DefaultThemeID = "black"` constant
     - `Load()` falls back to default on unknown ID, no panic
   - Acceptance criteria:
     - `Theme` interface has exactly 23 methods (21 color tokens + ID + Name)
     - `Load()` returns the correct theme for known IDs
     - `Load()` returns default theme (not panic) for unknown IDs
     - `Available()` returns exactly 5 entries in stable order
     - `DefaultThemeID` is `"black"`
   - Tests:
     - `TestLoad_KnownID` -- Load("monokai") returns MonokaiTheme with correct ID
     - `TestLoad_UnknownID_FallsBackToDefault` -- Load("does-not-exist") returns BlackTheme
     - `TestLoad_DefaultTheme` -- Load("black") returns non-nil with ID "black"
     - `TestAvailable_Returns5Entries` -- returns exactly ["black", "monokai", "catppuccin", "nord", "light"]
     - `TestAvailable_StableOrder` -- multiple calls return same order

2. **Task 0b.2 -- Implement all five themes** -- Create one file per theme, each implementing the full `Theme` interface with the exact hex values specified in the token tables. Every method must return a non-empty `lipgloss.Color`.
   - Files: `internal/ui/theme/black.go`, `internal/ui/theme/monokai.go`, `internal/ui/theme/catppuccin.go`, `internal/ui/theme/nord.go`, `internal/ui/theme/light.go`, `internal/ui/theme/theme_test.go`
   - Implementation steps:
     - `black.go` -- True Black (all token values from table)
     - `monokai.go` -- Monokai (all token values from table)
     - `catppuccin.go` -- Catppuccin Mocha (all token values from table)
     - `nord.go` -- Nord (all token values from table)
     - `light.go` -- Light/Catppuccin Latte (all token values from table)
   - Acceptance criteria:
     - Each theme struct implements every method of the `Theme` interface (compile-time check)
     - Every color method returns a non-empty `lipgloss.Color` value
     - `ID()` matches the registry key for each theme
     - `Name()` is a non-empty human-readable display name
   - Tests:
     - `TestAllThemes_ImplementInterface` -- iterate Available(), Load each, assert all 23 methods return non-empty values
     - `TestBlackTheme_Base_IsPureBlack` -- Base() returns "#000000"
     - `TestMonokaiTheme_Base` -- Base() returns "#272822"
     - `TestCatppuccinTheme_Base` -- Base() returns "#1e1e2e"
     - `TestNordTheme_Base` -- Base() returns "#2e3440"
     - `TestLightTheme_Base` -- Base() returns "#eff1f5"
     - `TestAllThemes_IDMatchesRegistryKey` -- each theme's ID() equals the key used to Load it

3. **Task 0b.4 -- Wire into app startup** -- Connect the theme system to the application bootstrap so that `cmd/root.go` reads the theme from config, loads it, and passes it through `app.New()` down to every pane constructor.
   - Files: `cmd/root.go` (modify), `internal/app/app.go` (modify)
   - Implementation steps:
     - `cmd/root.go` reads `cfg.UI.Theme` and calls `theme.Load()`
     - Theme passed into `app.New()` and down to all pane constructors
     - Verify unknown theme in config produces a warning log + fallback (not crash)
   - Acceptance criteria:
     - `cmd/root.go` reads `cfg.UI.Theme` and calls `theme.Load()`
     - Theme is passed to `app.New()` and stored as a field
     - Pane constructors receive the theme at construction time
     - Unknown theme in config produces a log warning + fallback to default (never crash)
   - Tests:
     - `TestAppNew_ReceivesTheme` -- verify theme is stored and accessible
     - `TestAppNew_DefaultThemeFallback` -- config with invalid theme ID still creates app with default theme

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

---

## Story: Theme Enhancement -- 16 New Color Tokens (spec 40)

### Background
As Spotnik's UI evolved toward a btop-inspired redesign with gradient seek/volume bars, a braille-dot audio visualizer, dense sortable table panes, and per-pane colored border accents, the original 26-method Theme interface needed expansion. This story added 16 new color tokens -- gradient fills (3), visualizer foreground (1), table header text (1), preset indicator (1), and per-pane border accents (10) -- to the Theme interface and implemented them across all five existing themes with exact hex values from DESIGN.md section 18. No functional changes were made; existing UI rendered identically.

### Acceptance Criteria
- [ ] `Theme` interface has 42 methods (26 original + 16 new)
- [ ] All 5 theme structs compile and satisfy the interface
- [ ] Every new token returns the exact hex value from DESIGN.md section 18
- [ ] No hardcoded hex values in any file outside `internal/ui/theme/`
- [ ] `make ci` passes (lint + tests + coverage)
- [ ] Existing tests still pass (no regressions)
- [ ] No functional changes -- existing UI renders identically

### Design Diagram

```
Theme interface (current: 26 methods)
  +-- Backgrounds: Base, Surface, SurfaceAlt
  +-- Borders: ActiveBorder, InactiveBorder
  +-- Text: TextPrimary, TextSecondary, TextMuted
  +-- Selection: SelectedBg, SelectedFg
  +-- Semantic: SectionHeader, PlayingIndicator, SeekBar, VolumeBar, Success, Warning, Error, DeviceActive
  +-- Status bar: StatusBarBg, StatusBarFg, KeyHint
  +-- Metadata: ID, Name

  + NEW (16 tokens):
  +-- Gradient: Gradient1, Gradient2, Gradient3
  +-- Visualizer: VisualizerFg
  +-- Tables: TableHeader
  +-- Status: PresetIndicator
  +-- Per-pane borders (10):
      +-- PaneBorderNowPlaying (green)
      +-- PaneBorderQueue (yellow)
      +-- PaneBorderPlaylists (blue)
      +-- PaneBorderAlbums (cyan)
      +-- PaneBorderLikedSongs (green)
      +-- PaneBorderRecentlyPlayed (teal)
      +-- PaneBorderTopTracks (purple)
      +-- PaneBorderTopArtists (pink/red)
      +-- PaneBorderRequestFlow (orange/amber)
      +-- PaneBorderNetworkLog (warm grey)
```

### Tasks

1. **Task 1 -- Add new tokens to Theme interface** -- Add 16 new methods to the `Theme` interface for gradients, visualizer, tables, presets, and per-pane border colors.
   - Files: `internal/ui/theme/theme.go` (modify)
   - New interface methods:
     ```go
     // Gradient bars
     Gradient1() lipgloss.Color     // Seek bar start / low volume
     Gradient2() lipgloss.Color     // Seek bar end / mid volume
     Gradient3() lipgloss.Color     // High volume (hot)

     // Visualizer
     VisualizerFg() lipgloss.Color  // Braille dot foreground

     // Tables
     TableHeader() lipgloss.Color   // Column header text

     // Status
     PresetIndicator() lipgloss.Color  // Preset label in header

     // Per-pane borders
     PaneBorderNowPlaying() lipgloss.Color
     PaneBorderQueue() lipgloss.Color
     PaneBorderPlaylists() lipgloss.Color
     PaneBorderAlbums() lipgloss.Color
     PaneBorderLikedSongs() lipgloss.Color
     PaneBorderRecentlyPlayed() lipgloss.Color
     PaneBorderTopTracks() lipgloss.Color
     PaneBorderTopArtists() lipgloss.Color
     PaneBorderRequestFlow() lipgloss.Color
     PaneBorderNetworkLog() lipgloss.Color
     ```
   - Tests:
     - Unit: Verify `Theme` interface has 42 methods (26 existing + 16 new) -- compile check
     - Unit: Each registered theme satisfies the interface (explicit assertion `var _ Theme = &BlackTheme{}` etc.)

2. **Task 2 -- Implement tokens in True Black theme** -- Add 16 new methods to `BlackTheme`.
   - Files: `internal/ui/theme/black.go` (modify)
   - Token values:

     | Token | Hex | Notes |
     |-------|-----|-------|
     | `Gradient1` | `#00ff88` | Green -- seek start, low volume |
     | `Gradient2` | `#ffcc00` | Yellow -- seek end, mid volume |
     | `Gradient3` | `#ff5555` | Red -- high volume |
     | `VisualizerFg` | `#00afff` | Ice blue -- matches accent |
     | `TableHeader` | `#666666` | Subtle header text |
     | `PresetIndicator` | `#00afff` | Matches accent |
     | `PaneBorderNowPlaying` | `#00ff88` | Green (playing) |
     | `PaneBorderQueue` | `#ffcc00` | Yellow (warning) |
     | `PaneBorderPlaylists` | `#00afff` | Blue (accent) |
     | `PaneBorderAlbums` | `#00e5cc` | Cyan (teal) |
     | `PaneBorderLikedSongs` | `#00ff88` | Green (success) |
     | `PaneBorderRecentlyPlayed` | `#00ccaa` | Teal |
     | `PaneBorderTopTracks` | `#bd93f9` | Purple |
     | `PaneBorderTopArtists` | `#ff79c6` | Pink |
     | `PaneBorderRequestFlow` | `#ffb86c` | Orange/amber |
     | `PaneBorderNetworkLog` | `#8a8a8a` | Warm grey |

   - Tests:
     - Unit: Table-driven test verifying each new token returns the expected hex value
     - Unit: Verify `BlackTheme` still satisfies `Theme` interface

3. **Task 3 -- Implement tokens in Monokai theme** -- Add 16 new methods to `MonokaiTheme`.
   - Files: `internal/ui/theme/monokai.go` (modify)
   - Token values:

     | Token | Hex | Notes |
     |-------|-----|-------|
     | `Gradient1` | `#a6e22e` | Monokai green |
     | `Gradient2` | `#e6db74` | Monokai yellow |
     | `Gradient3` | `#f92672` | Monokai pink |
     | `VisualizerFg` | `#66d9ef` | Monokai cyan |
     | `TableHeader` | `#75715e` | Monokai comment grey |
     | `PresetIndicator` | `#66d9ef` | Monokai cyan |
     | `PaneBorderNowPlaying` | `#a6e22e` | Green |
     | `PaneBorderQueue` | `#fd971f` | Orange |
     | `PaneBorderPlaylists` | `#66d9ef` | Cyan |
     | `PaneBorderAlbums` | `#e6db74` | Yellow |
     | `PaneBorderLikedSongs` | `#a6e22e` | Green |
     | `PaneBorderRecentlyPlayed` | `#4dc9b0` | Teal |
     | `PaneBorderTopTracks` | `#ae81ff` | Purple |
     | `PaneBorderTopArtists` | `#f92672` | Pink |
     | `PaneBorderRequestFlow` | `#fd971f` | Orange |
     | `PaneBorderNetworkLog` | `#75715e` | Monokai comment grey |

   - Tests:
     - Unit: Table-driven test verifying each token value

4. **Task 4 -- Implement tokens in Catppuccin theme** -- Add 16 new methods to `CatppuccinTheme`.
   - Files: `internal/ui/theme/catppuccin.go` (modify)
   - Token values:

     | Token | Hex | Notes |
     |-------|-----|-------|
     | `Gradient1` | `#a6e3a1` | Green |
     | `Gradient2` | `#f9e2af` | Yellow |
     | `Gradient3` | `#f38ba8` | Red |
     | `VisualizerFg` | `#89b4fa` | Blue |
     | `TableHeader` | `#6c7086` | Overlay0 |
     | `PresetIndicator` | `#89b4fa` | Blue |
     | `PaneBorderNowPlaying` | `#a6e3a1` | Green |
     | `PaneBorderQueue` | `#f9e2af` | Yellow |
     | `PaneBorderPlaylists` | `#89b4fa` | Blue |
     | `PaneBorderAlbums` | `#94e2d5` | Teal |
     | `PaneBorderLikedSongs` | `#a6e3a1` | Green |
     | `PaneBorderRecentlyPlayed` | `#94e2d5` | Teal |
     | `PaneBorderTopTracks` | `#cba6f7` | Mauve |
     | `PaneBorderTopArtists` | `#f38ba8` | Red/pink |
     | `PaneBorderRequestFlow` | `#fab387` | Peach/orange |
     | `PaneBorderNetworkLog` | `#6c7086` | Overlay0 grey |

   - Tests:
     - Unit: Table-driven test verifying each token value

5. **Task 5 -- Implement tokens in Nord theme** -- Add 16 new methods to `NordTheme`.
   - Files: `internal/ui/theme/nord.go` (modify)
   - Token values:

     | Token | Hex | Notes |
     |-------|-----|-------|
     | `Gradient1` | `#a3be8c` | Nord green |
     | `Gradient2` | `#ebcb8b` | Nord yellow |
     | `Gradient3` | `#bf616a` | Nord red |
     | `VisualizerFg` | `#88c0d0` | Nord frost |
     | `TableHeader` | `#4c566a` | Nord grey |
     | `PresetIndicator` | `#88c0d0` | Nord frost |
     | `PaneBorderNowPlaying` | `#a3be8c` | Green |
     | `PaneBorderQueue` | `#ebcb8b` | Yellow |
     | `PaneBorderPlaylists` | `#88c0d0` | Frost |
     | `PaneBorderAlbums` | `#8fbcbb` | Teal |
     | `PaneBorderLikedSongs` | `#a3be8c` | Green |
     | `PaneBorderRecentlyPlayed` | `#8fbcbb` | Teal |
     | `PaneBorderTopTracks` | `#b48ead` | Purple |
     | `PaneBorderTopArtists` | `#bf616a` | Red |
     | `PaneBorderRequestFlow` | `#d08770` | Orange |
     | `PaneBorderNetworkLog` | `#4c566a` | Nord grey |

   - Tests:
     - Unit: Table-driven test verifying each token value

6. **Task 6 -- Implement tokens in Light theme** -- Add 16 new methods to `LightTheme`.
   - Files: `internal/ui/theme/light.go` (modify)
   - Token values:

     | Token | Hex | Notes |
     |-------|-----|-------|
     | `Gradient1` | `#40a02b` | Latte green |
     | `Gradient2` | `#df8e1d` | Latte yellow |
     | `Gradient3` | `#d20f39` | Latte red |
     | `VisualizerFg` | `#1e66f5` | Latte blue |
     | `TableHeader` | `#9ca0b0` | Latte overlay0 |
     | `PresetIndicator` | `#1e66f5` | Latte blue |
     | `PaneBorderNowPlaying` | `#40a02b` | Green |
     | `PaneBorderQueue` | `#df8e1d` | Yellow |
     | `PaneBorderPlaylists` | `#1e66f5` | Blue |
     | `PaneBorderAlbums` | `#179299` | Teal |
     | `PaneBorderLikedSongs` | `#40a02b` | Green |
     | `PaneBorderRecentlyPlayed` | `#179299` | Teal |
     | `PaneBorderTopTracks` | `#8839ef` | Mauve |
     | `PaneBorderTopArtists` | `#d20f39` | Red |
     | `PaneBorderRequestFlow` | `#fe640b` | Orange |
     | `PaneBorderNetworkLog` | `#9ca0b0` | Latte overlay0 grey |

   - Tests:
     - Unit: Table-driven test verifying each token value

7. **Task 7 -- Update theme tests** -- Comprehensive test coverage for all 16 new tokens across all 5 themes.
   - Files: `internal/ui/theme/theme_test.go` (modify)
   - Tests:
     - Unit: Table-driven test iterating all 5 themes verifying all 16 new tokens return non-empty values
     - Unit: Explicit interface satisfaction assertions: `var _ Theme = &BlackTheme{}` for each theme struct
     - Unit: Verify `Available()` still returns all 5 theme IDs
     - Unit: Verify `Load()` with each ID returns a theme implementing all 42 methods
     - Unit: All 5 themes x 16 tokens = 80 value assertions
     - Unit: `Load("unknown")` falls back to default with all tokens present

### Notes

- The `SeekBar()` and `VolumeBar()` tokens remain for backward compatibility. The new `Gradient1/2/3` tokens are used by the new gradient components (Feature 44). Existing `ProgressBar` and `VolumeBar` components continue using the old tokens until migrated.
- `FilterInputBg` was considered but dropped -- use `SurfaceAlt()` instead (DESIGN.md section 18 note).
- The per-pane border tokens are used by `RenderPaneBorder()` in Feature 42. Until that feature is built, these tokens exist but are unused -- that is intentional.
