---
title: "Config-Driven Theme Loading"
feature: 16-vivid-themes
status: open
---

## Background
The current theme system uses one Go struct file per theme (e.g., `black.go`, `monokai.go`), each implementing the `Theme` interface with ~46 methods returning hardcoded hex values. Adding a new theme requires writing a new Go file, implementing every interface method, recompiling, and updating the registry. This story replaces that architecture with TOML-based theme files that are loaded at runtime, making themes plug-and-play.

## Design

### Theme TOML Schema

Each theme is a single `.toml` file with this structure:

```toml
id = "dracula"
name = "Dracula"

[colors]
# Backgrounds
base = "#282A36"
surface = "#44475A"
surface_alt = "#6272A4"

# Borders
active_border = "#BD93F9"
inactive_border = "#44475A"

# Text hierarchy
text_primary = "#F8F8F2"
text_secondary = "#BFBFBF"
text_muted = "#6272A4"

# Selection
selected_bg = "#44475A"
selected_fg = "#F8F8F2"

# Semantic
section_header = "#BD93F9"
playing_indicator = "#50FA7B"
seek_bar = "#FF79C6"
volume_bar = "#FF79C6"
success = "#50FA7B"
warning = "#F1FA8C"
error = "#FF5555"
device_active = "#8BE9FD"

# Status bar
status_bar_bg = "#1E1F29"
status_bar_fg = "#6272A4"
key_hint = "#BD93F9"

# Gradient bars
gradient1 = "#50FA7B"
gradient2 = "#F1FA8C"
gradient3 = "#FF5555"

# Visualizer
visualizer_fg = "#BD93F9"

# Tables
table_header = "#6272A4"

# Status
preset_indicator = "#BD93F9"

# Column colors (NEW)
column_index = "#6272A4"
column_primary = "#50FA7B"
column_secondary = "#8BE9FD"
column_tertiary = "#FFB86C"

[pane_borders]
now_playing = "#50FA7B"
queue = "#F1FA8C"
playlists = "#BD93F9"
albums = "#8BE9FD"
liked_songs = "#50FA7B"
recently_played = "#8BE9FD"
top_tracks = "#FF79C6"
top_artists = "#FF5555"
request_flow = "#FFB86C"
network_log = "#6272A4"
```

### File Locations

| Source | Path | Priority |
|---|---|---|
| Built-in themes | Embedded via `//go:embed themes/*.toml` in `theme.go` | Base (lowest) |
| User themes | `~/.config/spotnik/themes/*.toml` | Override (highest) |

User themes with the same `id` as a built-in theme override the built-in version entirely.

### ConfigTheme Struct

A single struct replaces all 5 per-theme Go files:

```go
// internal/ui/theme/config_theme.go

package theme

import "github.com/charmbracelet/lipgloss"

// ConfigTheme implements Theme by loading color values from a TOML config file.
// This is the sole concrete implementation of Theme -- all themes (built-in and
// user-provided) use this struct.
type ConfigTheme struct {
    id   string
    name string
    c    themeColors
    pb   paneBorderColors
}

// themeColors holds all color token values parsed from the [colors] section.
type themeColors struct {
    Base             string `toml:"base"`
    Surface          string `toml:"surface"`
    SurfaceAlt       string `toml:"surface_alt"`
    ActiveBorder     string `toml:"active_border"`
    InactiveBorder   string `toml:"inactive_border"`
    TextPrimary      string `toml:"text_primary"`
    TextSecondary    string `toml:"text_secondary"`
    TextMuted        string `toml:"text_muted"`
    SelectedBg       string `toml:"selected_bg"`
    SelectedFg       string `toml:"selected_fg"`
    SectionHeader    string `toml:"section_header"`
    PlayingIndicator string `toml:"playing_indicator"`
    SeekBar          string `toml:"seek_bar"`
    VolumeBar        string `toml:"volume_bar"`
    Success          string `toml:"success"`
    Warning          string `toml:"warning"`
    Error            string `toml:"error"`
    DeviceActive     string `toml:"device_active"`
    StatusBarBg      string `toml:"status_bar_bg"`
    StatusBarFg      string `toml:"status_bar_fg"`
    KeyHint          string `toml:"key_hint"`
    Gradient1        string `toml:"gradient1"`
    Gradient2        string `toml:"gradient2"`
    Gradient3        string `toml:"gradient3"`
    VisualizerFg     string `toml:"visualizer_fg"`
    TableHeader      string `toml:"table_header"`
    PresetIndicator  string `toml:"preset_indicator"`
    ColumnIndex      string `toml:"column_index"`
    ColumnPrimary    string `toml:"column_primary"`
    ColumnSecondary  string `toml:"column_secondary"`
    ColumnTertiary   string `toml:"column_tertiary"`
}

// paneBorderColors holds per-pane border accent values from [pane_borders].
type paneBorderColors struct {
    NowPlaying     string `toml:"now_playing"`
    Queue          string `toml:"queue"`
    Playlists      string `toml:"playlists"`
    Albums         string `toml:"albums"`
    LikedSongs     string `toml:"liked_songs"`
    RecentlyPlayed string `toml:"recently_played"`
    TopTracks      string `toml:"top_tracks"`
    TopArtists     string `toml:"top_artists"`
    RequestFlow    string `toml:"request_flow"`
    NetworkLog     string `toml:"network_log"`
}

// Each Theme interface method reads from the parsed struct fields:
func (t *ConfigTheme) ID() string                          { return t.id }
func (t *ConfigTheme) Name() string                        { return t.name }
func (t *ConfigTheme) Base() lipgloss.Color                { return lipgloss.Color(t.c.Base) }
func (t *ConfigTheme) Surface() lipgloss.Color             { return lipgloss.Color(t.c.Surface) }
// ... all other methods follow the same pattern
```

### themeFile — TOML Deserialization Target

```go
// themeFile is the top-level structure matching the TOML schema.
type themeFile struct {
    ID          string           `toml:"id"`
    Name        string           `toml:"name"`
    Colors      themeColors      `toml:"colors"`
    PaneBorders paneBorderColors `toml:"pane_borders"`
}
```

### Loading Logic

```go
//go:embed themes/*.toml
var builtinThemes embed.FS

// loadAll discovers and loads all theme files.
// Built-in themes are loaded first, then user themes override by ID.
func loadAll() (map[string]*ConfigTheme, error) {
    themes := make(map[string]*ConfigTheme)

    // 1. Load built-in embedded themes
    entries, _ := fs.ReadDir(builtinThemes, "themes")
    for _, e := range entries {
        data, _ := builtinThemes.ReadFile("themes/" + e.Name())
        t, err := parseTheme(data)
        if err != nil {
            continue // skip malformed built-in themes
        }
        themes[t.id] = t
    }

    // 2. Load user themes (override built-in by ID)
    userDir := userThemeDir() // ~/.config/spotnik/themes/
    if entries, err := os.ReadDir(userDir); err == nil {
        for _, e := range entries {
            if !strings.HasSuffix(e.Name(), ".toml") {
                continue
            }
            data, err := os.ReadFile(filepath.Join(userDir, e.Name()))
            if err != nil {
                continue
            }
            t, err := parseTheme(data)
            if err != nil {
                continue // skip malformed user themes
            }
            themes[t.id] = t // override built-in if same ID
        }
    }

    return themes, nil
}

// parseTheme decodes TOML bytes into a ConfigTheme.
func parseTheme(data []byte) (*ConfigTheme, error) {
    var f themeFile
    if err := toml.Unmarshal(data, &f); err != nil {
        return nil, fmt.Errorf("parsing theme: %w", err)
    }
    if f.ID == "" {
        return nil, fmt.Errorf("theme missing id field")
    }
    return &ConfigTheme{id: f.ID, name: f.Name, c: f.Colors, pb: f.PaneBorders}, nil
}

// userThemeDir returns the user theme directory path.
func userThemeDir() string {
    cfgDir, _ := os.UserConfigDir()
    return filepath.Join(cfgDir, "spotnik", "themes")
}
```

### Registry Changes

Replace the current `registry` map and constructor functions:

```go
// theme.go (updated)

const DefaultThemeID = "black"

var (
    loaded    map[string]*ConfigTheme
    loadOnce  sync.Once
)

// ensureLoaded lazily loads all themes on first access.
func ensureLoaded() {
    loadOnce.Do(func() {
        var err error
        loaded, err = loadAll()
        if err != nil || len(loaded) == 0 {
            // Fallback: ensure at least the default theme exists
            loaded = make(map[string]*ConfigTheme)
        }
    })
}

// Load returns the theme for the given config ID.
// Falls back to DefaultThemeID if the ID is unknown.
func Load(id string) Theme {
    ensureLoaded()
    if t, ok := loaded[id]; ok {
        return t
    }
    if t, ok := loaded[DefaultThemeID]; ok {
        return t
    }
    // Should never happen if built-in themes are embedded correctly
    return &ConfigTheme{id: "black", name: "True Black"}
}

// Available returns all registered theme IDs in a stable sorted order.
func Available() []string {
    ensureLoaded()
    ids := make([]string, 0, len(loaded))
    for id := range loaded {
        ids = append(ids, id)
    }
    sort.Strings(ids)
    return ids
}

// AllThemes returns all loaded ConfigTheme instances, sorted by ID.
// Used by the theme switcher overlay to display names and preview colors.
func AllThemes() []*ConfigTheme {
    ensureLoaded()
    themes := make([]*ConfigTheme, 0, len(loaded))
    for _, t := range loaded {
        themes = append(themes, t)
    }
    sort.Slice(themes, func(i, j int) bool { return themes[i].id < themes[j].id })
    return themes
}
```

### Migration: Convert Existing Themes to TOML

All 5 existing theme Go files (`black.go`, `monokai.go`, `catppuccin.go`, `nord.go`, `light.go`) are converted to TOML files placed in `internal/ui/theme/themes/`. The Go files are deleted. Every hex value is preserved exactly from the current implementations.

Example -- `internal/ui/theme/themes/black.toml`:

```toml
id = "black"
name = "True Black"

[colors]
base = "#000000"
surface = "#0f0f0f"
surface_alt = "#1a1a1a"
active_border = "#00afff"
inactive_border = "#1e1e1e"
text_primary = "#f0f0f0"
text_secondary = "#888888"
text_muted = "#444444"
selected_bg = "#1c3a5e"
selected_fg = "#f0f0f0"
section_header = "#00afff"
playing_indicator = "#00ff88"
seek_bar = "#00afff"
volume_bar = "#00afff"
success = "#00ff88"
warning = "#ffcc00"
error = "#ff5555"
device_active = "#00e5cc"
status_bar_bg = "#000000"
status_bar_fg = "#444444"
key_hint = "#00afff"
gradient1 = "#00ff88"
gradient2 = "#ffcc00"
gradient3 = "#ff5555"
visualizer_fg = "#00afff"
table_header = "#666666"
preset_indicator = "#00afff"
column_index = "#555555"
column_primary = "#00ff88"
column_secondary = "#00afff"
column_tertiary = "#00e5cc"

[pane_borders]
now_playing = "#00ff88"
queue = "#ffcc00"
playlists = "#00afff"
albums = "#00e5cc"
liked_songs = "#00ff88"
recently_played = "#00ccaa"
top_tracks = "#bd93f9"
top_artists = "#ff79c6"
request_flow = "#ffb86c"
network_log = "#8a8a8a"
```

### Theme Interface Update

Add 4 new methods to the `Theme` interface for column colors:

```go
// Column colors -- distinct foreground for each table column semantic
ColumnIndex() lipgloss.Color      // # column (muted but colorful)
ColumnPrimary() lipgloss.Color    // Main data: track name, playlist name
ColumnSecondary() lipgloss.Color  // Supporting: artist, genre
ColumnTertiary() lipgloss.Color   // Metadata: duration, year, played time
```

Total interface methods: 46 (existing) + 4 (column colors) = 50.

### Files Changed

| Action | File | Purpose |
|---|---|---|
| Create | `internal/ui/theme/config_theme.go` | ConfigTheme struct, parseTheme, themeFile |
| Create | `internal/ui/theme/themes/black.toml` | True Black theme TOML |
| Create | `internal/ui/theme/themes/monokai.toml` | Monokai theme TOML |
| Create | `internal/ui/theme/themes/catppuccin.toml` | Catppuccin Mocha theme TOML |
| Create | `internal/ui/theme/themes/nord.toml` | Nord theme TOML |
| Create | `internal/ui/theme/themes/light.toml` | Light (Catppuccin Latte) theme TOML |
| Modify | `internal/ui/theme/theme.go` | Add 4 column methods to interface, replace registry with loadAll/embed, add AllThemes() |
| Delete | `internal/ui/theme/black.go` | Replaced by TOML |
| Delete | `internal/ui/theme/monokai.go` | Replaced by TOML |
| Delete | `internal/ui/theme/catppuccin.go` | Replaced by TOML |
| Delete | `internal/ui/theme/nord.go` | Replaced by TOML |
| Delete | `internal/ui/theme/light.go` | Replaced by TOML |
| Modify | `internal/ui/theme/theme_test.go` | Test TOML loading, parsing, fallback, user override, new column tokens |

### Out of Scope
- Validation of color contrast ratios
- Theme editor or theme preview pane
- Hot-reloading of theme files while app is running (only loaded at startup + overlay switch)

## Acceptance Criteria
- [ ] `Theme` interface has 50 methods (46 existing + 4 column colors)
- [ ] `ConfigTheme` struct implements all 50 methods by reading from parsed TOML fields
- [ ] 5 existing themes are TOML files in `internal/ui/theme/themes/` with identical hex values to the deleted Go files
- [ ] `Load("monokai")` returns a `ConfigTheme` with correct values from `monokai.toml`
- [ ] `Load("unknown")` falls back to `DefaultThemeID` ("black")
- [ ] User TOML files in `~/.config/spotnik/themes/` are discovered and loaded
- [ ] User theme with `id = "black"` overrides the built-in black theme
- [ ] Malformed TOML files are skipped without panic
- [ ] `Available()` returns all theme IDs in sorted order
- [ ] `AllThemes()` returns all loaded ConfigTheme instances
- [ ] All 5 deleted Go theme files have no remaining references
- [ ] `make ci` passes (lint + tests + 80% coverage)

## Tasks
- [ ] Add 4 column color methods (`ColumnIndex`, `ColumnPrimary`, `ColumnSecondary`, `ColumnTertiary`) to the `Theme` interface in `theme.go`
      - test: Compile check -- interface has 50 methods
- [ ] Create `themeFile`, `themeColors`, `paneBorderColors` TOML deserialization structs and `parseTheme()` function in `config_theme.go`
      - test: `TestParseTheme_ValidTOML`, `TestParseTheme_MissingID_ReturnsError`, `TestParseTheme_MalformedTOML_ReturnsError`
- [ ] Implement `ConfigTheme` struct with all 50 `Theme` interface methods reading from parsed fields
      - test: `TestConfigTheme_ImplementsInterface`, `TestConfigTheme_ReturnsCorrectColors` (parse a test TOML and verify each method returns the expected lipgloss.Color)
- [ ] Convert all 5 existing themes to TOML files in `internal/ui/theme/themes/`, preserving every hex value exactly. Include the 4 new column color tokens with appropriate values per theme.
      - test: `TestBuiltinThemes_AllLoad`, `TestBuiltinThemes_HexValuesMatch` (spot-check key values against known hex from the old Go files)
- [ ] Replace the hardcoded `registry` map in `theme.go` with `//go:embed themes/*.toml` loading via `loadAll()`. Update `Load()`, `Available()`, add `AllThemes()`. Use `sync.Once` for lazy loading.
      - test: `TestLoad_KnownID`, `TestLoad_UnknownID_FallsBack`, `TestAvailable_ReturnsSortedIDs`, `TestAllThemes_ReturnsAll`
- [ ] Add user theme directory loading (`~/.config/spotnik/themes/`) with override-by-ID semantics
      - test: `TestUserTheme_OverridesBuiltin` (write a temp TOML to a temp dir, verify it overrides), `TestUserThemeDir_NoDir_NoError`
- [ ] Delete the 5 old Go theme files (`black.go`, `monokai.go`, `catppuccin.go`, `nord.go`, `light.go`). Update `theme_test.go` to test TOML-based loading.
      - test: Existing tests adapted -- `TestLoad_KnownID`, `TestAllThemes_ImplementInterface` still pass with TOML-loaded themes
