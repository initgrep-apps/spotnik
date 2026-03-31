---
title: "Theme Switcher Overlay and Runtime Switching"
feature: 16-vivid-themes
status: done
---

## Background
Currently themes can only be changed by editing `config.toml` and restarting the app. This story adds a theme switcher overlay (toggled with `t`) that lets users browse and apply themes at runtime, with the selection persisted to `config.toml`. The overlay follows the same pattern as the existing device switcher overlay.

## Design

### Theme Switcher Overlay Layout

```
╭─ Themes ──────────────────────────────── ᐅEnter select ╮
│                                                         │
│  ◉ True Black        ■ ■ ■ ■ ■                         │
│  ○ Monokai           ■ ■ ■ ■ ■                         │
│  ○ Catppuccin Mocha  ■ ■ ■ ■ ■                         │
│  ○ Nord              ■ ■ ■ ■ ■                         │
│  ○ Light             ■ ■ ■ ■ ■                         │
│  ○ Dracula           ■ ■ ■ ■ ■                         │
│  ○ Gruvbox Dark      ■ ■ ■ ■ ■                         │
│  ○ Tokyo Night       ■ ■ ■ ■ ■                         │
│  ○ Rose Pine         ■ ■ ■ ■ ■                         │
│  ○ Solarized Dark    ■ ■ ■ ■ ■                         │
│  ○ Synthwave '84     ■ ■ ■ ■ ■                         │
│                                                         │
╰─────────────────────────────────────────────────────────╯
```

- Each `■` block is a colored Unicode square (`█`) rendered in one of the theme's signature colors: `ColumnPrimary`, `ColumnSecondary`, `ColumnTertiary`, `PaneBorderNowPlaying`, `ActiveBorder`
- These swatches give users an instant visual preview of each theme's palette
- Current theme marked with `◉` in `Success()` color
- Other themes marked with `○` in `TextMuted()` color
- Selected row highlighted with `SelectedBg()` + `SelectedFg()`

### Overlay Model

```go
// internal/ui/panes/themes.go

package panes

// ThemeOverlay is the model for the theme switcher overlay.
type ThemeOverlay struct {
    themes    []*theme.ConfigTheme  // all available themes
    cursor    int                   // highlighted row index
    currentID string               // currently active theme ID
    theme     theme.Theme           // active theme for rendering
    width     int
    height    int
}

// NewThemeOverlay creates a new ThemeOverlay.
func NewThemeOverlay(themes []*theme.ConfigTheme, currentID string, th theme.Theme) *ThemeOverlay {
    return &ThemeOverlay{
        themes:    themes,
        cursor:    findThemeIndex(themes, currentID),
        currentID: currentID,
        theme:     th,
    }
}
```

### Message Types

```go
// ThemeSwitchMsg is emitted when the user selects a theme.
// The root app handles this by loading the new theme and propagating it.
type ThemeSwitchMsg struct {
    ThemeID string
}
```

### Keymap (Theme Overlay)

| Key | Action |
|---|---|
| `j` / `down` | Move selection down |
| `k` / `up` | Move selection up |
| `Enter` | Apply selected theme |
| `Esc` | Close overlay without change |

### Overlay Rendering

The overlay uses `RenderPaneBorder` with:
- Title: `"Themes"`
- Actions: `[{Key: "Enter", Label: "select"}]`
- AccentColor: `theme.ActiveBorder()`
- Focused: `true` (overlays are always focused)

Width: max theme name length + swatch space + padding (min 40 columns).
Height: number of themes + 4 (border rows + padding).

### Color Swatches

For each theme row, render 5 colored `█` characters using that row's theme colors:

```go
func renderSwatches(t *theme.ConfigTheme) string {
    colors := []lipgloss.Color{
        t.ColumnPrimary(),
        t.ColumnSecondary(),
        t.ColumnTertiary(),
        t.PaneBorderNowPlaying(),
        t.ActiveBorder(),
    }
    var b strings.Builder
    for _, c := range colors {
        b.WriteString(lipgloss.NewStyle().Foreground(c).Render("█"))
        b.WriteString(" ")
    }
    return b.String()
}
```

This renders the swatches using the *target* theme's colors, not the current theme, so users can preview the palette before switching.

### Runtime Theme Switching

#### SetTheme on Pane Interface

Add a new method to the `layout.Pane` interface:

```go
// layout/pane.go
type Pane interface {
    // ... existing methods ...

    // SetTheme updates the pane's theme reference for runtime theme switching.
    SetTheme(th theme.Theme)
}
```

Every pane must implement `SetTheme`. For table-based panes, this means:
1. Store the new theme reference
2. Rebuild the table with new column colors
3. The next `View()` call will use the new theme

```go
// Example: QueuePane
func (p *QueuePane) SetTheme(th theme.Theme) {
    p.theme = th
    // Rebuild table config with new column colors
    p.table = components.NewTable(components.TableConfig{
        Columns: []components.ColumnDef{
            {Key: "index", Header: "#", FlexFactor: 1, Color: th.ColumnIndex()},
            {Key: "track", Header: "Track", FlexFactor: 9, Color: th.ColumnPrimary()},
            {Key: "artist", Header: "Artist", FlexFactor: 7, Color: th.ColumnSecondary()},
            {Key: "duration", Header: "Duration", FlexFactor: 3, Color: th.ColumnTertiary()},
        },
        Theme:        th,
        PlayingIndex: -1,
        ShowHeader:   true,
    })
    // Re-apply size and rows
    p.table.SetSize(p.width, p.height)
    p.table.SetRows(p.currentRows)
}
```

#### App-Level Handling

```go
// internal/app/app.go -- in Update()

case panes.ThemeSwitchMsg:
    newTheme := theme.Load(msg.ThemeID)
    a.theme = newTheme

    // Propagate to all panes
    for _, p := range a.allPanes() {
        p.SetTheme(newTheme)
    }

    // Propagate to overlays
    a.searchOverlay.SetTheme(newTheme)
    a.deviceOverlay.SetTheme(newTheme)
    a.themeOverlay.SetTheme(newTheme)

    // Persist to config
    a.persistThemeChoice(msg.ThemeID)

    // Close the overlay
    a.showThemeSwitcher = false

    return a.alerts.NewAlertCmd("success", "Theme: "+newTheme.Name())
```

### Config Persistence

When a theme is selected, persist the choice to `config.toml`:

```go
// internal/config/config.go

// PersistTheme updates the theme field in the config file.
// Creates the file if it doesn't exist. Only modifies the theme key.
func PersistTheme(themeID string) error {
    cfgPath := configFilePath() // ~/.config/spotnik/config.toml

    // Read existing config (or start fresh)
    cfg := loadOrDefault(cfgPath)
    cfg.UI.Theme = themeID

    // Write back
    data, err := toml.Marshal(cfg)
    if err != nil {
        return fmt.Errorf("marshaling config: %w", err)
    }
    return os.WriteFile(cfgPath, data, 0644)
}
```

### Key Routing in App

```go
// internal/app/app.go -- in Update()

// Global key handler:
case "t":
    if !a.showSearchOverlay && !a.showDeviceSwitcher && !a.showThemeSwitcher {
        a.showThemeSwitcher = true
        a.themeOverlay = panes.NewThemeOverlay(
            theme.AllThemes(),
            a.theme.ID(),
            a.theme,
        )
        return nil
    }

// When overlay is active, route all keys to it:
if a.showThemeSwitcher {
    cmd := a.themeOverlay.Update(msg)
    return cmd
}
```

### Overlay Compositing

Same pattern as device overlay -- render the theme overlay above a dimmed main view:

```go
// internal/app/render.go

if a.showThemeSwitcher {
    overlay := a.themeOverlay.View()
    return btoverlay.Composite(mainView, overlay, ...)
}
```

### DESIGN.md Keybinding Update

Add `t` to the keybinding table:

```
| `t` | Open theme switcher overlay | Global |
```

### Files Changed

| Action | File | Purpose |
|---|---|---|
| Create | `internal/ui/panes/themes.go` | ThemeOverlay model, View, Update, swatches |
| Create | `internal/ui/panes/themes_test.go` | ThemeOverlay tests |
| Modify | `internal/ui/layout/pane.go` | Add `SetTheme(theme.Theme)` to Pane interface |
| Modify | `internal/ui/panes/*.go` | Implement `SetTheme` on every pane |
| Modify | `internal/app/app.go` | Add `showThemeSwitcher`, `themeOverlay` fields; handle `t` key; handle `ThemeSwitchMsg` |
| Modify | `internal/app/render.go` | Composite theme overlay when active |
| Modify | `internal/config/config.go` | Add `PersistTheme()` function |
| Create | `internal/config/config_test.go` | Test PersistTheme |
| Modify | `docs/DESIGN.md` | Add `t` keybinding |

### Out of Scope
- Live preview while browsing (applying the theme before Enter is pressed)
- Theme search/filter within the overlay
- Custom theme creation from within the app

## Acceptance Criteria
- [ ] `t` key opens the theme switcher overlay
- [ ] Overlay shows all 11 themes with names and color swatches
- [ ] Current theme is marked with `◉`
- [ ] `j`/`k`/`up`/`down` navigate the list
- [ ] `Enter` applies the selected theme immediately
- [ ] `Esc` closes the overlay without changing the theme
- [ ] All panes update their colors after a theme switch (borders, table columns, text)
- [ ] Theme selection is persisted to `~/.config/spotnik/config.toml`
- [ ] Next app launch uses the persisted theme
- [ ] Toast notification confirms the switch: "Theme: {name}"
- [ ] `t` does nothing when search or device overlay is already open
- [ ] `docs/DESIGN.md` keybinding table includes `t`
- [ ] `make ci` passes (lint + tests + 80% coverage)

## Tasks
- [ ] Add `SetTheme(theme.Theme)` method to the `layout.Pane` interface in `pane.go`
      - test: Compile check -- all panes implement the updated interface
- [ ] Implement `SetTheme` on all table-based panes: QueuePane, AlbumsPane, LikedSongsPane, TopTracksPane, TopArtistsPane, RecentlyPlayedPane, PlaylistsPane, NetworkLogPane. Each must rebuild its table with new column colors and re-apply size/rows.
      - test: `TestQueuePane_SetTheme_UpdatesColors` (switch theme, verify next View uses new colors)
- [ ] Implement `SetTheme` on non-table panes: NowPlayingPane, RequestFlowPane. Store new theme reference.
      - test: `TestNowPlayingPane_SetTheme`
- [ ] Implement `SetTheme` on overlays: SearchOverlay, DeviceOverlay
      - test: `TestSearchOverlay_SetTheme`, `TestDeviceOverlay_SetTheme`
- [ ] Create `ThemeOverlay` model in `internal/ui/panes/themes.go` with Init, Update, View, SetTheme. Handle j/k/Enter/Esc keys. Render with RenderPaneBorder, color swatches, current theme indicator.
      - test: `TestThemeOverlay_KeyNavigation`, `TestThemeOverlay_Enter_EmitsThemeSwitchMsg`, `TestThemeOverlay_Esc_NoMsg`, `TestThemeOverlay_CurrentThemeMarked`
- [ ] Add `showThemeSwitcher` and `themeOverlay` fields to `App`. Route `t` key to open overlay. Route keys to overlay when active. Handle `ThemeSwitchMsg`: load new theme, propagate via `SetTheme`, close overlay, emit toast.
      - test: `TestApp_TKey_OpensThemeSwitcher`, `TestApp_ThemeSwitchMsg_PropagatesTheme`, `TestApp_ThemeSwitchMsg_ClosesOverlay`
- [ ] Add overlay compositing in `render.go` -- render theme overlay above dimmed main view when `showThemeSwitcher` is true
      - test: `TestRender_ThemeOverlay_Composited`
- [ ] Add `PersistTheme(themeID string) error` to `internal/config/config.go`. Reads existing config.toml, updates `[ui] theme`, writes back. Creates file if missing.
      - test: `TestPersistTheme_WritesFile`, `TestPersistTheme_CreatesFileIfMissing`, `TestPersistTheme_PreservesOtherConfig`
- [ ] Update `docs/DESIGN.md` keybinding table to include `t` → "Open theme switcher overlay" under Global scope
      - test: Manual -- verify table entry exists
