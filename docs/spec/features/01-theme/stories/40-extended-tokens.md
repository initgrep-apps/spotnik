---
title: "Theme Enhancement -- 16 New Color Tokens"
feature: 01-theme
status: done
---

## Background
As Spotnik's UI evolved toward a btop-inspired redesign with gradient seek/volume bars, a braille-dot audio visualizer, dense sortable table panes, and per-pane colored border accents, the original 26-method Theme interface needed expansion. This story added 16 new color tokens -- gradient fills (3), visualizer foreground (1), table header text (1), preset indicator (1), and per-pane border accents (10) -- to the Theme interface and implemented them across all five existing themes with exact hex values from DESIGN.md section 18. No functional changes were made; existing UI rendered identically.

## Design

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

### New Interface Methods

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

### Token Values Per Theme

#### True Black (`black`)

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

#### Monokai (`monokai`)

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

#### Catppuccin Mocha (`catppuccin`)

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

#### Nord (`nord`)

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

#### Light -- Catppuccin Latte (`light`)

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

### Notes

- The `SeekBar()` and `VolumeBar()` tokens remain for backward compatibility. The new `Gradient1/2/3` tokens are used by the new gradient components (Feature 44). Existing `ProgressBar` and `VolumeBar` components continue using the old tokens until migrated.
- `FilterInputBg` was considered but dropped -- use `SurfaceAlt()` instead (DESIGN.md section 18 note).
- The per-pane border tokens are used by `RenderPaneBorder()` in Feature 42. Until that feature is built, these tokens exist but are unused -- that is intentional.

## Acceptance Criteria
- [ ] `Theme` interface has 42 methods (26 original + 16 new)
- [ ] All 5 theme structs compile and satisfy the interface
- [ ] Every new token returns the exact hex value from DESIGN.md section 18
- [ ] No hardcoded hex values in any file outside `internal/ui/theme/`
- [ ] `make ci` passes (lint + tests + coverage)
- [ ] Existing tests still pass (no regressions)
- [ ] No functional changes -- existing UI renders identically

## Tasks
- [ ] Add 16 new methods to the `Theme` interface in `theme.go`
      - test: Verify `Theme` interface has 42 methods (compile check); each registered theme satisfies the interface
- [ ] Implement 16 new tokens in True Black theme (`black.go`)
      - test: Table-driven test verifying each new token returns the expected hex value
- [ ] Implement 16 new tokens in Monokai theme (`monokai.go`)
      - test: Table-driven test verifying each token value
- [ ] Implement 16 new tokens in Catppuccin theme (`catppuccin.go`)
      - test: Table-driven test verifying each token value
- [ ] Implement 16 new tokens in Nord theme (`nord.go`)
      - test: Table-driven test verifying each token value
- [ ] Implement 16 new tokens in Light theme (`light.go`)
      - test: Table-driven test verifying each token value
- [ ] Update theme tests for comprehensive coverage of all 16 new tokens across all 5 themes
      - test: Table-driven test iterating all 5 themes verifying all 16 new tokens return non-empty values; 80 value assertions total; `Load("unknown")` falls back to default with all tokens present
