# Feature 40 — Theme Enhancement

> **Feature:** Add 16 new color tokens to the Theme interface and implement them
> across all 5 themes. These tokens support per-pane border accents, gradient bars,
> audio visualizer, table headers, and preset indicators required by the new
> btop-inspired UI redesign.

## Context

The current `Theme` interface (`internal/ui/theme/theme.go`) defines 26 color methods.
The new DESIGN.md (§18 "Theme Enhancements") requires 16 additional tokens to support:

- **Gradient bars** (3 tokens): Seek bar gradient fill + volume color bands
- **Visualizer** (1 token): Braille-dot audio spectrum foreground
- **Tables** (1 token): Column header text color for dense table panes
- **Status** (1 token): Preset indicator label in the header bar
- **Per-pane borders** (10 tokens): Distinct accent color per pane (btop-style identity)

All 5 existing themes (black, monokai, catppuccin, nord, light) must implement these tokens.

**Design reference:** `docs/DESIGN.md` §2 (Pane Definitions), §10 (Per-Pane Border Colors),
§11 (Visual Components), §18 (Theme Enhancements — full token tables with hex values)

**Depends on:** Nothing — pure infrastructure, no functional changes.

---

## Design Diagram

```
Theme interface (current: 26 methods)
  ├── Backgrounds: Base, Surface, SurfaceAlt
  ├── Borders: ActiveBorder, InactiveBorder
  ├── Text: TextPrimary, TextSecondary, TextMuted
  ├── Selection: SelectedBg, SelectedFg
  ├── Semantic: SectionHeader, PlayingIndicator, SeekBar, VolumeBar, Success, Warning, Error, DeviceActive
  ├── Status bar: StatusBarBg, StatusBarFg, KeyHint
  └── Metadata: ID, Name

  + NEW (16 tokens):
  ├── Gradient: Gradient1, Gradient2, Gradient3
  ├── Visualizer: VisualizerFg
  ├── Tables: TableHeader
  ├── Status: PresetIndicator
  └── Per-pane borders (10):
      ├── PaneBorderNowPlaying (green)
      ├── PaneBorderQueue (yellow)
      ├── PaneBorderPlaylists (blue)
      ├── PaneBorderAlbums (cyan)
      ├── PaneBorderLikedSongs (green)
      ├── PaneBorderRecentlyPlayed (teal)
      ├── PaneBorderTopTracks (purple)
      ├── PaneBorderTopArtists (pink/red)
      ├── PaneBorderRequestFlow (orange/amber)
      └── PaneBorderNetworkLog (warm grey)
```

---

## Task 1: Add new tokens to Theme interface

**Problem:** The Theme interface lacks tokens for gradients, visualizer, tables, presets,
and per-pane border colors.

**Fix:**

1. Add 16 new methods to the `Theme` interface in `internal/ui/theme/theme.go`:

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
   PaneBorderNowPlaying() lipgloss.Color      // green accent
   PaneBorderQueue() lipgloss.Color            // yellow accent
   PaneBorderPlaylists() lipgloss.Color        // blue accent
   PaneBorderAlbums() lipgloss.Color           // cyan accent
   PaneBorderLikedSongs() lipgloss.Color       // green accent
   PaneBorderRecentlyPlayed() lipgloss.Color   // teal accent
   PaneBorderTopTracks() lipgloss.Color        // purple accent
   PaneBorderTopArtists() lipgloss.Color       // pink/red accent
   PaneBorderRequestFlow() lipgloss.Color      // orange/amber accent
   PaneBorderNetworkLog() lipgloss.Color       // warm grey accent
   ```

2. Group the new methods with doc comments matching the existing style.

**Files:**
- Modify: `internal/ui/theme/theme.go`

**Tests:**
- Unit: Verify `Theme` interface has 42 methods (26 existing + 16 new) — compile check
- Unit: Each registered theme satisfies the interface (already covered by registry pattern,
  but add explicit assertion `var _ Theme = &BlackTheme{}` etc. if not present)

**Commit:** `feat(theme): add 16 new tokens to Theme interface`

---

## Task 2: Implement tokens in True Black theme (default)

**Problem:** `BlackTheme` must implement all 16 new methods.

**Fix:**

Add methods to `internal/ui/theme/black.go` with these values (from DESIGN.md §18):

| Token | Hex | Notes |
|-------|-----|-------|
| `Gradient1` | `#00ff88` | Green — seek start, low volume |
| `Gradient2` | `#ffcc00` | Yellow — seek end, mid volume |
| `Gradient3` | `#ff5555` | Red — high volume |
| `VisualizerFg` | `#00afff` | Ice blue — matches accent |
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

**Files:**
- Modify: `internal/ui/theme/black.go`

**Tests:**
- Unit: Table-driven test verifying each new token returns the expected hex value
- Unit: Verify `BlackTheme` still satisfies `Theme` interface

**Commit:** `feat(theme): implement new tokens for True Black theme`

---

## Task 3: Implement tokens in Monokai theme

**Fix:** Add methods to `internal/ui/theme/monokai.go`:

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

**Files:**
- Modify: `internal/ui/theme/monokai.go`

**Tests:**
- Unit: Table-driven test verifying each token value

**Commit:** `feat(theme): implement new tokens for Monokai theme`

---

## Task 4: Implement tokens in Catppuccin theme

**Fix:** Add methods to `internal/ui/theme/catppuccin.go`:

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

**Files:**
- Modify: `internal/ui/theme/catppuccin.go`

**Tests:**
- Unit: Table-driven test verifying each token value

**Commit:** `feat(theme): implement new tokens for Catppuccin theme`

---

## Task 5: Implement tokens in Nord theme

**Fix:** Add methods to `internal/ui/theme/nord.go`:

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

**Files:**
- Modify: `internal/ui/theme/nord.go`

**Tests:**
- Unit: Table-driven test verifying each token value

**Commit:** `feat(theme): implement new tokens for Nord theme`

---

## Task 6: Implement tokens in Light theme

**Fix:** Add methods to `internal/ui/theme/light.go`:

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

**Files:**
- Modify: `internal/ui/theme/light.go`

**Tests:**
- Unit: Table-driven test verifying each token value

**Commit:** `feat(theme): implement new tokens for Light theme`

---

## Task 7: Update theme tests

**Fix:**

1. Update `internal/ui/theme/theme_test.go`:
   - Add table-driven test that iterates all 5 themes and verifies all 16 new tokens return non-empty values
   - Add explicit interface satisfaction assertions: `var _ Theme = &BlackTheme{}` for each theme struct
   - Verify `Available()` still returns all 5 theme IDs
   - Verify `Load()` with each ID returns a theme implementing all 42 methods

**Files:**
- Modify: `internal/ui/theme/theme_test.go`

**Tests:**
- Unit: All 5 themes × 16 tokens = 80 value assertions
- Unit: Interface satisfaction compile-time checks
- Unit: `Load("unknown")` falls back to default with all tokens present

**Commit:** `test(theme): verify all 16 new tokens across 5 themes`

---

## Acceptance Criteria

- [ ] `Theme` interface has 42 methods (26 original + 16 new)
- [ ] All 5 theme structs compile and satisfy the interface
- [ ] Every new token returns the exact hex value from DESIGN.md §18
- [ ] No hardcoded hex values in any file outside `internal/ui/theme/`
- [ ] `make ci` passes (lint + tests + coverage)
- [ ] Existing tests still pass (no regressions)
- [ ] No functional changes — existing UI renders identically

---

## Notes

- The `SeekBar()` and `VolumeBar()` tokens remain for backward compatibility. The new
  `Gradient1/2/3` tokens are used by the new gradient components (Feature 44). Existing
  `ProgressBar` and `VolumeBar` components continue using the old tokens until migrated.
- `FilterInputBg` was considered but dropped — use `SurfaceAlt()` instead (DESIGN.md §18 note).
- The per-pane border tokens are used by `RenderPaneBorder()` in Feature 42. Until that
  feature is built, these tokens exist but are unused — that's intentional.
