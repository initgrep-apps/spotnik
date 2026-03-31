---
title: "Six New Vibrant Themes"
feature: 16-vivid-themes
status: done
---

## Background
With the config-driven theme system in place (story 70), adding new themes is just creating TOML files. This story adds 6 new themes curated from popular developer color schemes, each chosen for visual distinctiveness and vibrant color palettes. Combined with the 5 existing themes (converted to TOML in story 70), Spotnik ships with 11 themes total.

## Design

### Theme Lineup (11 Total)

| # | ID | Name | Aesthetic | Source |
|---|---|---|---|---|
| 1 | `black` | True Black | OLED dark, ice blue accent | Existing |
| 2 | `monokai` | Monokai | Classic editor dark | Existing |
| 3 | `catppuccin` | Catppuccin Mocha | Soft pastel dark | Existing |
| 4 | `nord` | Nord | Arctic cool blue | Existing |
| 5 | `light` | Light | Catppuccin Latte light mode | Existing |
| 6 | `dracula` | Dracula | Purple/pink/cyan neon | **NEW** |
| 7 | `gruvbox` | Gruvbox Dark | Warm retro orange/green | **NEW** |
| 8 | `tokyonight` | Tokyo Night | Cool blue-purple pastels | **NEW** |
| 9 | `rosepine` | Rose Pine | Elegant gold/rose/pine | **NEW** |
| 10 | `solarized` | Solarized Dark | Scientific precision contrast | **NEW** |
| 11 | `synthwave` | Synthwave '84 | Neon 80s music aesthetic | **NEW** |

### Color Palettes

Each theme below lists all 50 token values. Colors are sourced from the official theme specifications.

---

#### Dracula (`dracula.toml`)

Official palette: draculatheme.com/contribute

```toml
id = "dracula"
name = "Dracula"

[colors]
base = "#282A36"
surface = "#44475A"
surface_alt = "#6272A4"
active_border = "#BD93F9"
inactive_border = "#44475A"
text_primary = "#F8F8F2"
text_secondary = "#BFBFBF"
text_muted = "#6272A4"
selected_bg = "#44475A"
selected_fg = "#F8F8F2"
section_header = "#BD93F9"
playing_indicator = "#50FA7B"
seek_bar = "#FF79C6"
volume_bar = "#FF79C6"
success = "#50FA7B"
warning = "#F1FA8C"
error = "#FF5555"
device_active = "#8BE9FD"
status_bar_bg = "#1E1F29"
status_bar_fg = "#6272A4"
key_hint = "#BD93F9"
gradient1 = "#50FA7B"
gradient2 = "#F1FA8C"
gradient3 = "#FF5555"
visualizer_fg = "#BD93F9"
table_header = "#6272A4"
preset_indicator = "#BD93F9"
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

---

#### Gruvbox Dark (`gruvbox.toml`)

Official palette: github.com/morhetz/gruvbox (bright variant)

```toml
id = "gruvbox"
name = "Gruvbox Dark"

[colors]
base = "#282828"
surface = "#3c3836"
surface_alt = "#504945"
active_border = "#fe8019"
inactive_border = "#3c3836"
text_primary = "#ebdbb2"
text_secondary = "#a89984"
text_muted = "#665c54"
selected_bg = "#504945"
selected_fg = "#ebdbb2"
section_header = "#fe8019"
playing_indicator = "#b8bb26"
seek_bar = "#fe8019"
volume_bar = "#fe8019"
success = "#b8bb26"
warning = "#fabd2f"
error = "#fb4934"
device_active = "#8ec07c"
status_bar_bg = "#1d2021"
status_bar_fg = "#665c54"
key_hint = "#fe8019"
gradient1 = "#b8bb26"
gradient2 = "#fabd2f"
gradient3 = "#fb4934"
visualizer_fg = "#fe8019"
table_header = "#665c54"
preset_indicator = "#fe8019"
column_index = "#a89984"
column_primary = "#b8bb26"
column_secondary = "#83a598"
column_tertiary = "#fabd2f"

[pane_borders]
now_playing = "#b8bb26"
queue = "#fabd2f"
playlists = "#83a598"
albums = "#8ec07c"
liked_songs = "#b8bb26"
recently_played = "#8ec07c"
top_tracks = "#d3869b"
top_artists = "#fb4934"
request_flow = "#fe8019"
network_log = "#a89984"
```

---

#### Tokyo Night (`tokyonight.toml`)

Official palette: github.com/enkia/tokyo-night-vscode-theme

```toml
id = "tokyonight"
name = "Tokyo Night"

[colors]
base = "#1a1b26"
surface = "#24283b"
surface_alt = "#414868"
active_border = "#7aa2f7"
inactive_border = "#24283b"
text_primary = "#c0caf5"
text_secondary = "#a9b1d6"
text_muted = "#565f89"
selected_bg = "#414868"
selected_fg = "#c0caf5"
section_header = "#7aa2f7"
playing_indicator = "#9ece6a"
seek_bar = "#ff9e64"
volume_bar = "#ff9e64"
success = "#9ece6a"
warning = "#e0af68"
error = "#f7768e"
device_active = "#73daca"
status_bar_bg = "#16161e"
status_bar_fg = "#565f89"
key_hint = "#7aa2f7"
gradient1 = "#9ece6a"
gradient2 = "#e0af68"
gradient3 = "#f7768e"
visualizer_fg = "#7aa2f7"
table_header = "#565f89"
preset_indicator = "#7aa2f7"
column_index = "#565f89"
column_primary = "#9ece6a"
column_secondary = "#7dcfff"
column_tertiary = "#ff9e64"

[pane_borders]
now_playing = "#9ece6a"
queue = "#e0af68"
playlists = "#7aa2f7"
albums = "#73daca"
liked_songs = "#9ece6a"
recently_played = "#73daca"
top_tracks = "#bb9af7"
top_artists = "#f7768e"
request_flow = "#ff9e64"
network_log = "#565f89"
```

---

#### Rose Pine (`rosepine.toml`)

Official palette: rosepinetheme.com/palette (main variant)

```toml
id = "rosepine"
name = "Rose Pine"

[colors]
base = "#191724"
surface = "#1f1d2e"
surface_alt = "#26233a"
active_border = "#c4a7e7"
inactive_border = "#1f1d2e"
text_primary = "#e0def4"
text_secondary = "#908caa"
text_muted = "#6e6a86"
selected_bg = "#26233a"
selected_fg = "#e0def4"
section_header = "#c4a7e7"
playing_indicator = "#9ccfd8"
seek_bar = "#f6c177"
volume_bar = "#f6c177"
success = "#9ccfd8"
warning = "#f6c177"
error = "#eb6f92"
device_active = "#9ccfd8"
status_bar_bg = "#191724"
status_bar_fg = "#6e6a86"
key_hint = "#c4a7e7"
gradient1 = "#9ccfd8"
gradient2 = "#f6c177"
gradient3 = "#eb6f92"
visualizer_fg = "#c4a7e7"
table_header = "#6e6a86"
preset_indicator = "#c4a7e7"
column_index = "#6e6a86"
column_primary = "#ebbcba"
column_secondary = "#9ccfd8"
column_tertiary = "#f6c177"

[pane_borders]
now_playing = "#9ccfd8"
queue = "#f6c177"
playlists = "#c4a7e7"
albums = "#31748f"
liked_songs = "#ebbcba"
recently_played = "#9ccfd8"
top_tracks = "#c4a7e7"
top_artists = "#eb6f92"
request_flow = "#f6c177"
network_log = "#908caa"
```

---

#### Solarized Dark (`solarized.toml`)

Official palette: ethanschoonover.com/solarized

```toml
id = "solarized"
name = "Solarized Dark"

[colors]
base = "#002b36"
surface = "#073642"
surface_alt = "#586e75"
active_border = "#268bd2"
inactive_border = "#073642"
text_primary = "#839496"
text_secondary = "#657b83"
text_muted = "#586e75"
selected_bg = "#073642"
selected_fg = "#fdf6e3"
section_header = "#268bd2"
playing_indicator = "#859900"
seek_bar = "#cb4b16"
volume_bar = "#cb4b16"
success = "#859900"
warning = "#b58900"
error = "#dc322f"
device_active = "#2aa198"
status_bar_bg = "#002b36"
status_bar_fg = "#586e75"
key_hint = "#268bd2"
gradient1 = "#859900"
gradient2 = "#b58900"
gradient3 = "#dc322f"
visualizer_fg = "#268bd2"
table_header = "#586e75"
preset_indicator = "#268bd2"
column_index = "#586e75"
column_primary = "#859900"
column_secondary = "#2aa198"
column_tertiary = "#b58900"

[pane_borders]
now_playing = "#859900"
queue = "#b58900"
playlists = "#268bd2"
albums = "#2aa198"
liked_songs = "#859900"
recently_played = "#2aa198"
top_tracks = "#6c71c4"
top_artists = "#d33682"
request_flow = "#cb4b16"
network_log = "#586e75"
```

---

#### Synthwave '84 (`synthwave.toml`)

Inspired by Robb Owen's Synthwave '84 VS Code theme — neon 80s aesthetic, perfect for a music player.

```toml
id = "synthwave"
name = "Synthwave '84"

[colors]
base = "#262335"
surface = "#34294f"
surface_alt = "#463465"
active_border = "#ff7edb"
inactive_border = "#34294f"
text_primary = "#ffffff"
text_secondary = "#b6b1b1"
text_muted = "#848bbd"
selected_bg = "#463465"
selected_fg = "#ffffff"
section_header = "#ff7edb"
playing_indicator = "#72f1b8"
seek_bar = "#36f9f6"
volume_bar = "#36f9f6"
success = "#72f1b8"
warning = "#fede5d"
error = "#fe4450"
device_active = "#36f9f6"
status_bar_bg = "#1b172e"
status_bar_fg = "#848bbd"
key_hint = "#ff7edb"
gradient1 = "#72f1b8"
gradient2 = "#fede5d"
gradient3 = "#fe4450"
visualizer_fg = "#ff7edb"
table_header = "#848bbd"
preset_indicator = "#36f9f6"
column_index = "#848bbd"
column_primary = "#72f1b8"
column_secondary = "#36f9f6"
column_tertiary = "#fede5d"

[pane_borders]
now_playing = "#72f1b8"
queue = "#fede5d"
playlists = "#ff7edb"
albums = "#36f9f6"
liked_songs = "#72f1b8"
recently_played = "#36f9f6"
top_tracks = "#ff7edb"
top_artists = "#fe4450"
request_flow = "#fede5d"
network_log = "#848bbd"
```

### Column Color Philosophy

Each theme assigns 4 distinct column colors that create visual rhythm in tables:

| Theme | ColumnIndex | ColumnPrimary | ColumnSecondary | ColumnTertiary |
|---|---|---|---|---|
| True Black | dim grey `#555555` | neon green `#00ff88` | ice blue `#00afff` | teal `#00e5cc` |
| Monokai | comment grey `#75715e` | green `#a6e22e` | cyan `#66d9ef` | orange `#fd971f` |
| Catppuccin | overlay0 `#6c7086` | green `#a6e3a1` | blue `#89b4fa` | peach `#fab387` |
| Nord | polar night `#4c566a` | green `#a3be8c` | frost `#88c0d0` | yellow `#ebcb8b` |
| Light | overlay0 `#9ca0b0` | green `#40a02b` | blue `#1e66f5` | yellow `#df8e1d` |
| Dracula | comment `#6272A4` | green `#50FA7B` | cyan `#8BE9FD` | orange `#FFB86C` |
| Gruvbox | grey `#a89984` | green `#b8bb26` | aqua `#83a598` | yellow `#fabd2f` |
| Tokyo Night | comment `#565f89` | green `#9ece6a` | cyan `#7dcfff` | orange `#ff9e64` |
| Rose Pine | muted `#6e6a86` | rose `#ebbcba` | foam `#9ccfd8` | gold `#f6c177` |
| Solarized | base01 `#586e75` | green `#859900` | cyan `#2aa198` | yellow `#b58900` |
| Synthwave | muted `#848bbd` | neon green `#72f1b8` | neon cyan `#36f9f6` | neon yellow `#fede5d` |

### Files Created

| File | Theme |
|---|---|
| `internal/ui/theme/themes/dracula.toml` | Dracula |
| `internal/ui/theme/themes/gruvbox.toml` | Gruvbox Dark |
| `internal/ui/theme/themes/tokyonight.toml` | Tokyo Night |
| `internal/ui/theme/themes/rosepine.toml` | Rose Pine |
| `internal/ui/theme/themes/solarized.toml` | Solarized Dark |
| `internal/ui/theme/themes/synthwave.toml` | Synthwave '84 |

## Acceptance Criteria
- [ ] All 6 new TOML files are placed in `internal/ui/theme/themes/`
- [ ] Each file has all 50 color tokens (no missing fields)
- [ ] `Available()` returns 11 theme IDs
- [ ] `Load("dracula")` returns a theme with correct Dracula colors
- [ ] `Load("synthwave")` returns a theme with correct Synthwave colors
- [ ] Every theme has 4 distinct, non-grey column color values
- [ ] Every theme has 10 distinct per-pane border accent colors
- [ ] `make ci` passes (lint + tests + 80% coverage)

## Tasks
- [ ] Create `dracula.toml` with all 50 token values from Dracula official palette
      - test: `TestDraculaTheme_Loads`, `TestDraculaTheme_Base` (verify `#282A36`)
- [ ] Create `gruvbox.toml` with all 50 token values from Gruvbox Dark official palette
      - test: `TestGruvboxTheme_Loads`, `TestGruvboxTheme_Base` (verify `#282828`)
- [ ] Create `tokyonight.toml` with all 50 token values from Tokyo Night official palette
      - test: `TestTokyoNightTheme_Loads`, `TestTokyoNightTheme_Base` (verify `#1a1b26`)
- [ ] Create `rosepine.toml` with all 50 token values from Rose Pine official palette
      - test: `TestRosePineTheme_Loads`, `TestRosePineTheme_Base` (verify `#191724`)
- [ ] Create `solarized.toml` with all 50 token values from Solarized Dark official palette
      - test: `TestSolarizedTheme_Loads`, `TestSolarizedTheme_Base` (verify `#002b36`)
- [ ] Create `synthwave.toml` with all 50 token values from Synthwave '84 palette
      - test: `TestSynthwaveTheme_Loads`, `TestSynthwaveTheme_Base` (verify `#262335`)
- [ ] Add comprehensive test: `TestAllThemes_HaveAllTokens` -- iterate all 11 themes, verify no empty color string for any of the 50 tokens
      - test: 11 × 50 = 550 assertions, all non-empty
