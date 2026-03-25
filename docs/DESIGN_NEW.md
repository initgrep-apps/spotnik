# DESIGN_NEW.md — btop-Inspired UI Redesign Specification

> **This document specifies the new UI system for Spotnik.**
> It replaces the frozen three-column layout from `DESIGN.md` with a responsive,
> pane-based grid inspired by btop's celebrated terminal UI design.
> Agents: this is the authoritative design document for all new UI work.

---

## Overview

Spotnik's current UI mimics the Spotify web player: three fixed columns (Library | Player | Queue).
This works poorly in a terminal — text overflows pane boundaries, the layout wastes space,
there's no scroll guidance, and it looks like a web app transplant rather than a native TUI.

The redesign draws from **btop** — a system monitor beloved by terminal enthusiasts for its:
- Pane-based responsive grid that fills every terminal cell
- Preset system for switching between curated layouts
- Embedded shortcuts in pane borders for instant discoverability
- Dense, colorful, information-rich aesthetic

### What Changes

| Aspect | Current (DESIGN.md) | New (DESIGN_NEW.md) |
|--------|---------------------|---------------------|
| Layout | Fixed 3-column (22/50/28%) | 3-row responsive grid, 9 panes across 2 pages |
| Panes | 3 fixed + 2 alternative views | 8 music panes + 1 internals pane, toggleable |
| Pages | None | Page A (Music) + Page B (Nerd Status), toggled by `0` |
| Presets | None (view switching via 1/2/3) | `p` cycles preset layouts within current page |
| Pane toggle | None | Keys `1`-`8` hide/show individual panes (btop-style) |
| Shortcuts | All in status bar | Embedded in pane borders (btop-style) |
| Filtering | None | In-pane `f` key filter on every list |
| Visuals | Monochrome cyan bars | Gradient bars, braille visualizer, multi-color columns |
| Borders | Same color for all panes | Per-pane accent colors |
| Content | Overflows boundaries | Hard-capped with truncation |
| Min terminal | 100x24 | 120x30 (8 music panes + borders need more space) |

### What Stays

- Rounded corners (`╭╮╰╯`) exclusively
- Theme system with token-based colors (no hardcoded hex)
- Elm architecture (messages, commands, Store)
- Overlays for search and device switcher (float above grid)
- Toast notifications (repositioned to bottom-right)
- Splash and Auth screens (render full-screen without the grid, transitional only)
- All existing Spotify API integration
- `tea.WithAltScreen()` for full-screen rendering

---

## 1. Design Philosophy

1. **Information density** — every terminal cell earns its place. No decorative whitespace, no empty panes. If there's room, show data.
2. **Pane independence** — each data category owns its pane. Panes can be shown, hidden, and rearranged via presets without affecting each other.
3. **Space awareness** — when a pane hides, its space redistributes to visible siblings. When an entire row hides, remaining rows expand. No wasted pixels.
4. **Embedded discoverability** — shortcuts are visible in pane borders at all times. Users never need to memorize keys or check a help screen. Like btop's `proc` title bar showing `filter`, `reverse`, `tree`.
5. **Preset-driven layouts** — curated configurations beat user-assembled chaos. Four well-designed presets cover 95% of use cases, with btop-style pane toggling for the rest.
6. **Nerd aesthetic** — braille-dot graphics, gradient-colored bars, dense aligned tables, per-pane border colors. This is a developer tool, not a web app skin.
7. **Content containment** — pane content never, ever overflows its allocated rectangle. Truncation is mandatory, not optional.

---

## 2. Pane Definitions

### Page A — Music (8 panes)

| # | Pane | ID | API Source | Toggle Key | Border Accent |
|---|------|----|-----------|------------|---------------|
| 1 | Now Playing | `PaneNowPlaying` | `GET /me/player` | `1` | `PlayingIndicator()` green |
| 2 | Queue | `PaneQueue` | `GET /me/player/queue` | `2` | `Warning()` yellow |
| 3 | Playlists | `PanePlaylists` | `GET /me/playlists` | `3` | `SectionHeader()` blue |
| 4 | Albums | `PaneAlbums` | `GET /me/albums` | `4` | `SeekBar()` cyan |
| 5 | Liked Songs | `PaneLikedSongs` | `GET /me/tracks` | `5` | `Success()` green |
| 6 | Recently Played | `PaneRecentlyPlayed` | `GET /me/player/recently-played` | `6` | `DeviceActive()` teal |
| 7 | Top Tracks | `PaneTopTracks` | `GET /me/top/tracks` | `7` | `KeyHint()` purple |
| 8 | Top Artists | `PaneTopArtists` | `GET /me/top/artists` | `8` | `Error()` pink/red |

### Page B — Nerd Status (1 pane, full screen)

| Pane | ID | Data | Content |
|------|----|------|---------|
| Internals | `PaneInternals` | App-internal state | Gateway rate limit/backoff, API request log (endpoint+status+latency), polling intervals, store fetching sentinels, cache hit/miss |

### Key Notes

- Keys `1`-`8` **toggle** pane visibility on Page A (btop-style hide/show), not pane-jump
- `0` toggles between Page A and Page B
- Playback keys (`Space`, `>`, `<`, `+`, `-`, `s`, `r`) always route to NowPlaying regardless of focus
- `A` (Shift+a) for "add to queue" in search overlay and list panes
- `i` for "like/unlike track" in Liked Songs and Queue panes

### Pane Interface

Every pane implements:

```go
type Pane interface {
    tea.Model                          // Init, Update, View
    SetSize(width, height int)         // Content area dimensions (inside border)
    SetFocused(focused bool)           // Keyboard focus state
    IsFocused() bool                   // Query focus state
    ID() PaneID                        // Slot identifier
    Title() string                     // Display title for border
    ToggleKey() int                     // Toggle key number (1-8) for border display, 0 if not toggleable
    Actions() []Action                 // Pane-specific shortcuts for border
}

type Action struct {
    Key   string  // e.g., "f"
    Label string  // e.g., "filter"
}
```

---

## 3. Layout Grid System

### Grid Model

The layout is a **row-based grid**. Each row contains cells (panes) with relative width weights. Rows have relative height weights.

```
Grid = []Row
Row  = {HeightWeight int, Cells []Cell}
Cell = {PaneID, WidthWeight int}
```

### Space Distribution Algorithm

1. **Filter hidden cells:** Remove cells whose pane is hidden.
2. **Filter empty rows:** Remove rows where all cells are hidden.
3. **Distribute height:** Divide available height among active rows proportionally by `HeightWeight`.
4. **Distribute width per row:** Divide row width among visible cells proportionally by `WidthWeight`.
5. **Absorb rounding:** Last cell/row absorbs any remainder pixels.

### Reserved Space

```
Total height = terminal rows
  - Header:     1 line  (preset indicator, device, clock)
  - Status bar: 1 line  (global shortcuts)
  - Content:    terminal rows - 2
```

Each pane's content area is `Rect.Width - 2` x `Rect.Height - 2` (borders consume 1 char on each side).

---

## 4. Pages, Pane Toggling, and Preset Layouts

### Page Switching

- `0` toggles between **Page A** (Music) and **Page B** (Nerd Status)
- Each page has its own preset cycle
- Switching pages preserves pane state on both sides

### Pane Toggling (btop-style)

Keys `1`-`8` toggle the corresponding pane's visibility on Page A:
- When a pane hides, siblings in the same row expand to fill its space
- When all panes in a row hide, the row collapses and other rows expand
- When a hidden pane is toggled back, it reappears in its original grid position
- Toggle state is independent of presets — switching preset resets all manual toggles

### Preset Cycling

`p` cycles through preset layouts within the current page:
- Each preset is a hide/show bitmask applied to panes
- After the last preset, wraps to the first (full layout)
- Switching preset resets all manual toggles

### Page A Presets

#### Preset 0 — Full Dashboard (default)

All 8 panes visible across 3 rows. NowPlaying spans full width.

```
╭─ ¹Now Playing ───────────────────────────────────────────────────────╮  Row 1 (weight 2)
│ ⣿⣷⣿⣷⣿  Martbaan · Samar Mehdi     ▶ 1:41 ████████░░░ 5:30           │
│            |<  ||  >|  ~  =>          VOL ████░░ 65%                 │
╰──────────────────────────────────────────────────────────────────────╯
╭─ ³Playlists ─────────╮╭─ ⁴Albums ────────────╮╭─ ⁵Liked Songs ──────╮  Row 2 (weight 3)
│  1  LoFi             ││  1  After Hours      ││  1  Blinding Lights  │
│  2  Soul             ││  2  OK Computer      ││  2  Save Your Tears  │
│  3  Workout          ││  3  In Rainbows      ││  3  Levitating       │
│  4  Best of Coke     ││  4  Blonde           ││  4  Peaches          │
│  ▼ more below        ││  ▼ more below        ││  ▼ more below        │
╰──────────────────────╯╰──────────────────────╯╰──────────────────────╯
╭─ ²Queue ──────╮╭─ ⁶Recent ─────╮╭─ ⁷Top Tracks ──╮╭─ ⁸Top Artists ─╮  Row 3 (weight 3)
│  1  Lil Boo   ││  1  Martbaan  ││  1  Blinding   ││  1  Weeknd     │
│  2  Street F  ││  2  Starboy   ││  2  Martbaan   ││  2  Drake      │
│  3  BIRDS     ││  3  Heat Wav  ││  3  Save Your  ││  3  Dua Lipa   │
│  ▼ more       ││  ▼ more       ││  ▼ more        ││  ▼ more        │
╰───────────────╯╰───────────────╯╰────────────────╯╰────────────────╯
```

**Grid definition:**
```
Row 1 (weight 2): [{NowPlaying, weight=1}]                              ← full width
Row 2 (weight 3): [{Playlists, weight=1}, {Albums, weight=1}, {LikedSongs, weight=1}]
Row 3 (weight 3): [{Queue, weight=1}, {RecentlyPlayed, weight=1}, {TopTracks, weight=1}, {TopArtists, weight=1}]
```

Note: Row 3 has 4 panes. TopTracks and TopArtists share the rightmost region — they can either be side by side (each getting half width) or use a split-pane with a shared border label (`7 Top ── 8 Artists`) and internal tab toggle.

#### Preset 1 — Listening

NowPlaying expanded with large visualizer. Queue and RecentlyPlayed below. All other panes hidden.

```
╭─ ¹Now Playing ──────────────────────────────────────────────────╮  Row 1 (weight 3)
│                                                                │
│  ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿                       │
│                                                                │
│  Martbaan · Samar Mehdi, June, Sarah Mehdi                     │
│  Martbaan (Album)                                              │
│                                                                │
│  ▶  1:41  ████████████████████████░░░░░░░░░░░░░░░░  5:30       │
│  |<   ||   >|        ~   =>           VOL  ████████░░  65%     │
╰────────────────────────────────────────────────────────────────╯
╭─ ²Queue ──────────────────────╮╭─ ⁶Recently Played ────────────╮  Row 2 (weight 2)
│  #   Track          Artist    ││  #  Track          Played      │
│  1   Lil Boo Thang  P.Russell ││  1  Starboy        2m ago      │
│  2   Street Fighter Kamasi W  ││  2  Heat Waves     15m ago     │
│  3   BIRDS OF A     Billie E  ││  3  Levitating     1h ago      │
│  ▼ more below                 ││  ▼ more below                  │
╰───────────────────────────────╯╰────────────────────────────────╯
```

**Visible panes:** NowPlaying, Queue, RecentlyPlayed

#### Preset 2 — Library

NowPlaying compact strip. Playlists, Albums, LikedSongs expanded. All other panes hidden.

```
╭─ ¹Now Playing ── Martbaan · Samar Mehdi ── ▶ 1:41/5:30 ──────────────╮  Row 1 (weight 1)
│  ████████████░░░░░░░  |<  ||  >|  ~  =>   VOL ████░░ 65%             │
╰──────────────────────────────────────────────────────────────────────╯
╭─ ³Playlists ─────────╮╭─ ⁴Albums ────────────╮╭─ ⁵Liked Songs ──────╮  Row 2 (weight 4)
│  1  LoFi             ││  1  After Hours      ││  1  Blinding Lights  │
│  2  Best of Coke     ││  2  OK Computer      ││  2  Save Your Tears  │
│  3  Bosnia           ││  3  In Rainbows      ││  3  Levitating       │
│  4  Soul             ││  4  Blonde           ││  4  Peaches          │
│  5  Our soundtrack   ││  5  Random Access Mem││  5  Mood             │
│  6  Lofi Fruits      ││  6  The Dark Side    ││  6  Watermelon Sugar │
│  7  GT               ││  7  Currents         ││  7  Starboy          │
│  8  Running          ││  8  Rumours          ││  8  Positions        │
│  9  Lizzie Poole     ││  9  Abbey Road       ││  9  Heat Waves       │
│  10 My Playlist #21  ││  10 AM               ││  10 drivers license  │
│  ▼ more below        ││  ▼ more below        ││  ▼ more below        │
╰──────────────────────╯╰──────────────────────╯╰──────────────────────╯
```

**Visible panes:** NowPlaying (compact), Playlists, Albums, LikedSongs

#### Preset 3 — Discovery

NowPlaying compact strip. TopTracks, TopArtists, RecentlyPlayed expanded. All other panes hidden.

```
╭─ ¹Now Playing ── Martbaan · Samar Mehdi ── ▶ 1:41/5:30 ──────────────╮  Row 1 (weight 1)
│  ████████████░░░░░░░  |<  ||  >|  ~  =>   VOL ████░░ 65%             │
╰──────────────────────────────────────────────────────────────────────╯
╭─ ⁷Top Tracks ────────────────╮╭─ ⁸Top Artists ───────────────────────╮  Row 2 (weight 2)
│  #  Track          Pop       ││  #  Artist          Genre            │
│  1  Blinding Ligh  85        ││  1  The Weeknd      pop              │
│  2  Martbaan       72        ││  2  Drake           hip-hop          │
│  3  Save Your Te   80        ││  3  Dua Lipa        dance pop        │
│  ▼ more below                ││  ▼ more below                        │
╰──────────────────────────────╯╰──────────────────────────────────────╯
╭─ ⁶Recently Played ──────────────────────────────────────────────────╮  Row 3 (weight 2)
│  #  Track                    Artist              Played             │
│  1  Starboy                  The Weeknd          2m ago             │
│  2  Heat Waves               Glass Animals       15m ago            │
│  3  Levitating               Dua Lipa            1h ago             │
│  ▼ more below                                                       │
╰──────────────────────────────────────────────────────────────────────╯
```

**Visible panes:** NowPlaying (compact), TopTracks, TopArtists, RecentlyPlayed

### Page A Preset Summary

| Preset | Name | Visible Panes |
|--------|------|---------------|
| 0 | Full Dashboard | All 8 (3 rows) |
| 1 | Listening | NowPlaying, Queue, RecentlyPlayed |
| 2 | Library | NowPlaying (compact), Playlists, Albums, LikedSongs |
| 3 | Discovery | NowPlaying (compact), TopTracks, TopArtists, RecentlyPlayed |

### Page B Layout

Single layout: NowPlaying compact strip (1 row) + Internals full-width (remaining space).

```
╭─ ¹Now Playing ── Martbaan · Samar Mehdi ── ▶ 1:41/5:30 ──────────────╮  Row 1 (weight 1)
│  ████████████░░░░░░░  |<  ||  >|  ~  =>   VOL ████░░ 65%             │
╰──────────────────────────────────────────────────────────────────────╯
╭─ Internals ──────────────────────────────────────────────────────────╮  Row 2 (weight 5)
│  GATEWAY          tokens: 8/10   backoff: —    concurrent: 2/5      │
│  ──────────────────────────────────────────────────────────────────  │
│  API LOG                                                             │
│  GET /me/player              200  45ms   ██                          │
│  GET /me/player/queue        200  62ms   ███                         │
│  GET /me/playlists           200  128ms  ██████                      │
│  GET /me/player              429  12ms   █   (backoff 2s)            │
│  GET /me/top/tracks          200  95ms   ████                        │
│  ──────────────────────────────────────────────────────────────────  │
│  POLLING          tick: 1000ms   state: active   idle: 0s            │
│  STORE            fetching: [playlists]   stale: albums(12s)         │
╰──────────────────────────────────────────────────────────────────────╯
```

### Preset/Toggle Behavior

When switching presets:
- All pane states (scroll position, selected item, filter text) are preserved
- Focus moves to the first visible pane if the currently focused pane becomes hidden
- The `renderGrid()` function re-assembles the layout immediately
- Manual pane toggles (keys `1`-`8`) are reset when switching presets

---

## 5. Embedded Shortcut Borders (btop-style)

Every pane border shows the pane title and action shortcuts directly in the border line.

### Border Anatomy

```
╭─ ¹Playlists ──────────────────── ᐅfilter ─── ᐅnew ╮
│                                                   │
│  (pane content)                                   │
│                                                   │
╰───────────────────────────────────────────────────╯
```

**Top border components:**
1. `╭─ ` — rounded corner + dash + space
2. `¹` — superscript pane number (toggle key 1-8) in `KeyHint()` color
3. `Playlists` — pane title in `SectionHeader()` color (bold when focused)
4. ` ─────── ` — dash fill
5. `ᐅfilter ─ ᐅnew` — action shortcuts: `ᐅ` prefix, key in `KeyHint()`, label in `TextMuted()`
6. ` ╮` — space + rounded corner

**Border colors:**
- Each pane has its own accent color (see Pane Definitions table)
- Focused pane: full accent color for border characters
- Unfocused pane: dimmed/muted version of accent color

**Custom rendering:** This replaces `lipgloss.RoundedBorder()`. The `RenderPaneBorder()` function builds the border string manually to embed title and action text.

### Filter Mode in Border

When a pane's filter is active, the border shows the filter query:

```
╭─ ¹Queue ────────── filtering: "rock" ─── ᐅEsc close ╮
```

---

## 6. Content Containment

**The #1 rule: pane content never exceeds its allocated rectangle.**

### Width Containment

- Every line of text rendered inside a pane is truncated to `paneWidth` characters
- `Truncate(text, maxWidth)` — rune-aware, uses `lipgloss.Width()` for accurate measurement, appends `…` when truncated
- `lipgloss.NewStyle().MaxWidth(paneWidth).MaxHeight(paneHeight)` wraps every pane's `View()` output as a safety net
- In `renderGrid()`, each cell is wrapped in `lipgloss.NewStyle().Width(rect.Width).MaxWidth(rect.Width)` before `JoinHorizontal` — this prevents any cell from pushing neighbors off-screen

### Vertical Containment

- Each pane computes `visibleItemCount` from its allocated height
- Content beyond the visible window is accessible via `j`/`k` scrolling
- Scroll indicators (`▲` at top, `▼` at bottom) show when content extends
- `lipgloss.NewStyle().Height(paneHeight).MaxHeight(paneHeight)` enforces the vertical cap
- The total `View()` output must equal exactly `terminalHeight` lines — pad if shorter, cap if taller

### Column Truncation (Dense Tables)

- Table columns get fixed proportions of pane width (e.g., `#` 5%, Track 45%, Artist 35%, Duration 15%)
- Each cell value is individually truncated to its column width
- Column widths recalculated on `SetSize()` — never hardcoded

### Truncation Utility

Located in `internal/ui/layout/truncate.go`:

```go
Truncate(s string, maxWidth int) string       // Truncate with "…" if too wide
PadRight(s string, width int) string          // Pad with spaces to exact width
TruncateOrPad(s string, width int) string     // Truncate or pad to exact width
```

All functions use `lipgloss.Width()` for rendered-width measurement, not `len()` or `utf8.RuneCountInString()`. This correctly handles wide characters (CJK), combining marks, and emoji.

---

## 7. Screen Stability

The terminal must never scroll. The entire UI renders within the alternate screen buffer.

- Spotnik uses `tea.WithAltScreen()` — this is correct and must not change
- The `View()` output must be exactly `terminalHeight` lines tall
- If grid content is shorter: pad with empty lines styled with `Base()` background
- If grid content would overflow: the height-capping in content containment prevents this
- Every row in the grid is height-capped to its allocated `Rect.Height`
- The assembled grid + header + status bar must sum to exactly `terminalHeight`

---

## 8. In-Pane Filtering

Every pane with a scrollable list supports real-time filtering, inspired by btop's process filter.

### Behavior

1. Press `f` in a focused pane to toggle filter mode
2. A text input appears at the top of the pane content (below the border)
3. As you type, the list filters in real-time (case-insensitive substring match)
4. `Esc` closes the filter and restores the full list
5. `Enter` selects the first/current filtered result and closes the filter
6. Filter state is per-pane — each pane owns its own filter input and filtered items

### Filterable Fields

| Pane | Filter by |
|------|-----------|
| Playlists | Playlist name |
| Albums | Album name, artist name |
| Liked Songs | Track name, artist name |
| Queue | Track name, artist name |
| Recently Played | Track name, artist name |
| Top Tracks | Track name, artist name |
| Top Artists | Artist name, genre |

### Visual Treatment

- Filter input: `TextPrimary()` text on `SurfaceAlt()` background
- Matching text in results: highlighted with `SelectedBg()` background (optional, future)
- Filter active indicator in border: `filtering: "query"` replaces action shortcuts

### Filter Component

Reusable component in `internal/ui/components/filter.go`:

```go
type Filter struct {
    input    textinput.Model
    active   bool
    query    string
}

func (f *Filter) Toggle()                           // Toggle filter on/off
func (f *Filter) IsActive() bool                    // Check if filtering
func (f *Filter) Query() string                     // Get current query
func (f *Filter) Matches(text string) bool          // Case-insensitive substring match
func (f *Filter) Update(msg tea.Msg) tea.Cmd        // Handle input events
func (f *Filter) View(width int) string             // Render filter bar
```

---

## 9. Dense Table Formatting

List panes (Queue, Playlists, Albums, LikedSongs, RecentlyPlayed, TopTracks, TopArtists) render data in aligned columns with per-column colors.

### Column Layout Example (Queue)

```
 #   Track                    Artist              Duration
 1   Lil Boo Thang            Paul Russell        3:12
 2   Street Fighter           Kamasi Washington   5:44
 3   BIRDS OF A FEATHER       Billie Eilish       3:30
```

### Column Colors

Each column uses a different theme color for visual separation without explicit dividers:

| Column | Color Token | Purpose |
|--------|-------------|---------|
| Index (`#`) | `TextMuted()` | De-emphasized numbering |
| Track name | `TextPrimary()` | Primary data — highest contrast |
| Artist | `TextSecondary()` | Supporting context |
| Duration/metadata | `TextMuted()` | Tertiary information |

**Selected row:** All columns override to `SelectedBg()` + `SelectedFg()`

**Currently playing row:** Index column shows `▶` in `PlayingIndicator()` color

### Column Width Proportions

| Pane | Col 1 | Col 2 | Col 3 | Col 4 |
|------|-------|-------|-------|-------|
| Queue | `#` 5% | Track 45% | Artist 35% | Duration 15% |
| Playlists | `#` 5% | Name 70% | Tracks 25% | — |
| Albums | `#` 5% | Name 50% | Artist 30% | Year 15% |
| Liked Songs | `#` 5% | Track 45% | Artist 35% | Duration 15% |
| Top Tracks | `#` 5% | Track 45% | Artist 35% | Pop 15% |
| Top Artists | `#` 5% | Name 70% | Genre 25% | — |
| Recently Played | `#` 5% | Track 45% | Artist 35% | Played 15% |

Column header row: `TableHeader()` color, not bold.

### Table Component

Reusable component in `internal/ui/components/table.go`:

```go
type Column struct {
    Header     string
    WeightPct  int              // Width as percentage of total
    Color      lipgloss.Color   // Text color for this column
}

type Table struct {
    columns  []Column
    width    int
}

func (t *Table) SetWidth(w int)
func (t *Table) RenderHeader() string
func (t *Table) RenderRow(values []string, selected bool, playing bool) string
```

---

## 10. Per-Pane Border Colors

Like btop, each pane has a distinct border color that provides visual identity without reading the title.

| Pane | Focused Color | Unfocused Color |
|------|--------------|-----------------|
| Now Playing | `PaneBorderNowPlaying()` (green accent) | Dimmed green |
| Queue | `PaneBorderQueue()` (yellow accent) | Dimmed yellow |
| Playlists | `PaneBorderPlaylists()` (blue accent) | Dimmed blue |
| Albums | `PaneBorderAlbums()` (cyan accent) | Dimmed cyan |
| Liked Songs | `PaneBorderLikedSongs()` (green accent) | Dimmed green |
| Recently Played | `PaneBorderRecentlyPlayed()` (teal accent) | Dimmed teal |
| Top Tracks | `PaneBorderTopTracks()` (purple accent) | Dimmed purple |
| Top Artists | `PaneBorderTopArtists()` (pink/red accent) | Dimmed pink |
| Internals | `PaneBorderInternals()` (orange/amber accent) | Dimmed orange |

**Dimming strategy:** Unfocused borders use the same hue at ~40% brightness. This can be achieved by:
- Defining separate unfocused tokens in the theme, OR
- Using `lipgloss.NewStyle().Faint(true)` on the border characters (simpler, theme-independent)

---

## 11. Visual Components

### Braille-Dot Audio Visualizer

Rendered in the Now Playing pane using Unicode braille characters (U+2800-U+28FF).

```
⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿
```

**Behavior:**
- Displays a simulated audio spectrum/waveform pattern
- Animates on `tea.Tick` (e.g., every 200ms) when music is playing
- Static/flat pattern when paused
- Width adapts to available pane width
- Height: 2-4 lines depending on pane height (Preset 1 gets more rows)
- Colors: `VisualizerFg()` token, with optional gradient effect

**Animation strategy:**
- The visualizer component owns a **separate configurable `tea.Tick`**, starting at 200ms interval
- When NowPlaying is focused, arrow keys adjust the tick speed (lower limit 200ms)
- It maintains a `frameIndex int` counter, incremented on each tick
- `View()` uses `frameIndex` to index into a precomputed animation frame table — no randomness in `View()`, ensuring deterministic rendering between ticks
- When paused: `frameIndex` stops incrementing, visualizer shows a static flat-line pattern
- Frame table contains 30-50 pre-generated bar patterns that loop smoothly

**Unicode note:** The `ᐅ` character (U+1405, Canadian Syllabics PA) is used for border action labels. Terminals must support this Unicode block. Fallback: `›` (U+203A, Single Right-Pointing Angle Quotation Mark) for environments without full Unicode support.

**Implementation:** Component in `internal/ui/components/visualizer.go`.

### Gradient-Colored Bars

Replace the current monochrome `SeekBar()` / `VolumeBar()` fills:

**Seek bar:**
```
████████████████░░░░░░░░░░░░░░
```
- Fill gradient: `Gradient1()` → `Gradient2()` (left to right)
- Empty: `Surface()` (unchanged)

**Volume bar:**
```
VOL  ████████░░░░░░  65%
```
- Low volume (0-33%): `Gradient1()` (green/cool)
- Mid volume (34-66%): `Gradient2()` (yellow/warm)
- High volume (67-100%): `Gradient3()` (red/hot)

**Implementation:** Component in `internal/ui/components/gradient.go`. Uses block characters (`█▓▒░`) with per-character color application for smooth gradients.

---

## 12. Notifications

Toast notifications use `go.dalton.dog/bubbleup` (existing dependency).

**Position change:** Toasts render at **bottom-right** instead of the current top-left default.

**Implementation:** If bubbleup supports position configuration, use it. Otherwise, the `View()` function repositions the toast overlay:
1. Call `alerts.Render(content)` as before
2. If toast is active, use `lipgloss.Place()` to reposition to bottom-right

**Styling:** Toast notifications retain their current look (rounded corners, theme colors, auto-dismiss after 4 seconds). They should feel consistent with the pane aesthetic.

---

## 13. Search Overlay

The search overlay remains a floating modal above the grid. Its styling updates to match the new pane aesthetic.

### Visual Treatment

```
╭─ Search ────────────────────────── ᐅEnter play ─ ᐅTab section ╮
│  > blinding lig█                                              │
│  ──────────────────────────────────────────────────────────── │
│  TRACKS                                                       │
│  ▶ Blinding Lights          The Weeknd         3:22           │
│    Blinding Lights (Remix)  Sunday Service     4:15           │
│  ARTISTS                                                      │
│    The Weeknd                                                 │
│  ALBUMS                                                       │
│    After Hours              The Weeknd                        │
╰───────────────────────────────────────────────────────────────╯
```

- Uses `RenderPaneBorder()` with `Search` title and action shortcuts
- Rounded corners, `ActiveBorder()` color (always focused)
- Centered on screen via `lipgloss.Place()`
- Background dimmed with `.Faint(true)`
- Results use the dense table column format with multi-color columns

---

## 14. Device Switcher Overlay

Same border treatment as search:

```
╭─ Devices ──────────────────────────── ᐅEnter select ╮
│  ◉  MacBook Pro Speakers     [active]               │
│  ○  iPhone 14                                       │
│  ○  Kitchen Speaker                                 │
│  ○  Living Room TV                                  │
╰─────────────────────────────────────────────────────╯
```

- Uses `RenderPaneBorder()` with `Devices` title
- Positioned top-right via `lipgloss.Place()`
- Active device: `◉` in `DeviceActive()`, `[active]` badge

---

## 15. Global Header & Status Bar

### Header (Top Line)

```
 spotnik ─ Page A ─ ᐅp preset 0 ─ ᐅ/ search ─ ᐅd devices ──────── ◉ iPhone
```

btop-style top bar with:
- Left: app name + current page (A/B) + global action shortcuts (`preset`, `search`, `devices`) with highlighted key
- Right: device indicator (`◉ DeviceName` or `○ No device`)

### Status Bar (Bottom Line)

Only **global** shortcuts. Pane-specific shortcuts live in pane borders.

```
 /search   0 page   p preset   1-8 toggle   Tab pane   d devices   ? help   q quit
```

Key labels in `KeyHint()`, descriptions in `StatusBarFg()`.

---

## 16. Focus & Navigation

### Focus Rotation

- `Tab` / `Shift+Tab`: rotate focus among **visible** panes only
- Order: top-left → top-right → second-row-left → ... → bottom-right → wrap
- Invisible panes are skipped in the rotation

### Pane Toggle (replaces Direct Pane Jump)

Keys `1`-`8` toggle pane visibility rather than jumping focus. Use `Tab`/`Shift+Tab` for focus navigation. This follows btop's approach where number keys control what's visible.

### Playback Keys (Always Route to NowPlaying)

| Key | Action |
|-----|--------|
| `Space` | Play/pause |
| `>` / `.` | Next track |
| `<` / `,` | Previous track |
| `+` / `-` | Volume up/down |
| `s` | Toggle shuffle |
| `r` | Cycle repeat |

These keys always route to `PaneNowPlaying` regardless of which pane is focused.

### Overlay Keys

| Key | Action |
|-----|--------|
| `/` | Open search overlay |
| `d` | Open device overlay |
| `Esc` | Close overlay / close filter |

Overlays intercept all keys while open. Focus is saved and restored on close.

---

## 17. Keybinding Table (Complete)

| Key | Action | Scope |
|-----|--------|-------|
| **Pages** | | |
| `0` | Toggle Page A / Page B | Global |
| **Pane Toggle (Page A)** | | |
| `1`-`8` | Toggle pane 1-8 visibility | Page A |
| **Presets** | | |
| `p` | Cycle to next preset | Current page |
| **Playback (always route to NowPlaying)** | | |
| `Space` | Play/pause | Always |
| `>` / `.` | Next track | Always |
| `<` / `,` | Previous track | Always |
| `+` / `-` | Volume up/down | Always |
| `s` | Toggle shuffle | Always |
| `r` | Cycle repeat | Always |
| **Navigation** | | |
| `Tab` | Next pane focus | Visible panes |
| `Shift+Tab` | Previous pane focus | Visible panes |
| `j` / `k` | Scroll down/up | Focused pane |
| `Enter` | Select/play item | Focused pane |
| `Esc` | Close overlay/filter | Context |
| **Pane Actions** | | |
| `f` | Toggle filter in focused pane | List panes |
| `x` | Remove from queue | Queue pane |
| `i` | Like/unlike track | LikedSongs, Queue |
| `n` | New playlist | Playlists pane |
| `A` (Shift+a) | Add to queue | Search overlay, list panes |
| **Playlist Management (Playlists pane)** | | |
| `Enter` | Open playlist tracks (sub-view) | Playlists pane |
| `r` | Rename playlist | Playlists pane (as border action) |
| `x` | Delete playlist | Playlists pane (as border action) |
| `Shift+↑/↓` | Reorder playlist | Playlists pane |
| **Global** | | |
| `/` | Open search overlay | Global |
| `d` | Open device overlay | Global |
| `?` | Help | Global |
| `q` | Quit | Global |

---

## 18. Theme Enhancements

### New Tokens Required

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
PaneBorderRecentlyPlayed() lipgloss.Color  // teal accent
PaneBorderTopTracks() lipgloss.Color       // purple accent
PaneBorderTopArtists() lipgloss.Color      // pink/red accent
PaneBorderInternals() lipgloss.Color       // orange/amber accent (nerd screen)

// Filter
// FilterInputBg dropped — use SurfaceAlt() instead (same value, no need for separate token)
```

All 5 existing themes (black, monokai, catppuccin, nord, light) must implement these tokens.

### New Token Values — All 5 Themes

#### True Black (`black`) — Default

| Token | Hex | Usage |
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
| `PaneBorderInternals` | `#ffb86c` | Orange/amber |
| ~~FilterInputBg~~ | — | Dropped: use `SurfaceAlt()` instead |

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
| `PaneBorderInternals` | `#fd971f` | Orange |

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
| `PaneBorderInternals` | `#fab387` | Peach/orange |

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
| `PaneBorderInternals` | `#d08770` | Orange |

#### Light — Catppuccin Latte (`light`)

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
| `PaneBorderInternals` | `#fe640b` | Orange |

---

## 19. Page B — Nerd Status Specification

The Internals pane on Page B provides live visibility into Spotnik's internal state. This is a developer-facing diagnostic screen — no Spotify API calls needed.

### Internals Pane Content

| Section | Data | Update Rate |
|---------|------|-------------|
| Gateway | Bucket tokens (X/10), backoff timer, concurrent requests | Every tick |
| API Log | Last 20 requests: `GET /me/player 200 45ms`, color-coded by status (green=2xx, yellow=429, red=5xx) | On each API call |
| Polling | Current tick interval, idle/active state, `lastInteraction` age | Every tick |
| Store | Fetching sentinels (which data in-flight), staleness TTLs remaining | Every tick |

### Data Source

All data is internal — reads from:
- `*Gateway` — rate limit bucket state, backoff timer, concurrent request count
- `*Store` — fetching sentinels, staleness TTLs
- New `APILogger` — ring buffer of last 20 API requests with endpoint, status code, and latency

No new Spotify API calls are made for Page B.

### API Log Color Coding

| Status Range | Color Token | Meaning |
|-------------|-------------|---------|
| 2xx | `Success()` | Successful request |
| 429 | `Warning()` | Rate limited (show backoff duration) |
| 4xx (other) | `TextMuted()` | Client error |
| 5xx | `Error()` | Server error |

### Latency Visualization

Each API log entry includes a small inline bar (1-10 `█` chars) proportional to response time, providing at-a-glance latency distribution.

---

## 20. Mouse Scroll Support

Mouse scroll allows scrolling any pane without changing focus, matching btop's behavior.

### Implementation

- Enable via `tea.EnableMouseCellMotion()` at app startup
- `tea.MouseMsg` with `MouseWheelUp`/`MouseWheelDown` scrolls the pane under the cursor
- Hit-test: check which pane `Rect` contains the mouse position
- Pane doesn't need focus to scroll with mouse (like btop)
- Click on a pane to focus it (optional, future enhancement)

### Architecture

The `LayoutManager` provides a `PaneAt(x, y int) PaneID` method that performs the hit-test against computed `Rect` values. Mouse scroll events are routed to the pane returned by this method, bypassing the focus system.

---

## 21. Responsive Behavior

### Minimum Terminal Size

| Preset | Min Width | Min Height |
|--------|-----------|------------|
| All | 120 columns | 30 rows |

Below minimum, show:
```
╭──────────────────────────────────────────╮
│  Spotnik needs more space                │
│                                          │
│  Current:  98 × 25                       │
│  Required: 120 × 30                      │
│                                          │
│  Please resize your terminal and retry.  │
╰──────────────────────────────────────────╯
```

### Future: Auto-Degrade

Not in initial implementation. Future enhancement: automatically hide lower-priority panes (TopArtists/TopTracks first, then RecentlyPlayed, then library row) when terminal is smaller than optimal but above minimum.

---

## 22. Architecture — LayoutManager

### Package: `internal/ui/layout/`

| File | Purpose |
|------|---------|
| `layout.go` | `Manager` struct, `Resize()`, `recompute()`, `PaneRect()`, `PaneAt()`, `SetPreset()`, `CyclePreset()`, `TogglePage()`, `TogglePane()`, `RotateFocus()`, `FocusedPane()` |
| `pane.go` | `Pane` interface, `PaneID` enum, `PageID` enum, `Action` struct |
| `presets.go` | `PresetDashboard`, `PresetListening`, `PresetLibrary`, `PresetDiscovery` definitions |
| `border.go` | `RenderPaneBorder()` — custom border with btop-style title + actions |
| `truncate.go` | `Truncate()`, `PadRight()`, `TruncateOrPad()` — rune-aware text utilities |
| `*_test.go` | Full table-driven test coverage |

### Manager Struct

```go
type Manager struct {
    activePage   PageID           // PageA (Music) or PageB (Nerd Status)
    presets      map[PageID][]Preset
    activePreset map[PageID]int
    hidden       map[PaneID]bool
    rects        map[PaneID]Rect
    focusOrder   []PaneID        // visible panes in layout order
    focusIndex   int
    width        int
    height       int
    headerHeight int             // 1
    statusHeight int             // 1
}
```

### Integration with App

```go
// App struct changes
type App struct {
    layout *layout.Manager
    panes  map[layout.PaneID]layout.Pane

    // Overlays remain separate
    searchPane  *panes.SearchOverlay
    devicePane  *panes.DeviceOverlay

    // Removed: playerPane, libraryPane, queuePane, statsPane, playlistPane
    // Removed: focus focusedPane, currentView viewMode
    // Added: apiLogger *APILogger (ring buffer for Internals pane)
}
```

---

## 23. Migration from Current Design

### Pane Mapping

| Current | New |
|---------|-----|
| `PlayerPane` | `NowPlayingPane` (renamed, visualizer added) |
| `LibraryPane` | Split into `PlaylistsPane`, `AlbumsPane`, `LikedSongsPane` |
| `QueuePane` | `QueuePane` (add Pane interface, dense table) |
| `StatsView` | Split into `TopTracksPane` + `TopArtistsPane` (separate panes). RecentlyPlayed section → `RecentlyPlayedPane` |
| `PlaylistManager` | Merged into `PlaylistsPane` (Enter=track sub-view, n=new, r=rename, x=delete, Shift+arrow=reorder as border actions) |
| — (new) | `InternalsPane` (Page B, reads from Gateway/Store/APILogger) |

### Pane Interface Migration Checklist

Each existing pane must gain these new methods to satisfy `layout.Pane`:

| Pane | `ID()` | `Title()` | `ToggleKey` | `Actions()` | Notes |
|------|--------|-----------|-------------|-------------|-------|
| `PlayerPane` → `NowPlayingPane` | `PaneNowPlaying` | "Now Playing" | `1` | shuffle, repeat | Rename + add visualizer |
| `QueuePane` | `PaneQueue` | "Queue" | `2` | filter, clear | Add dense table format |
| `LibraryPane` → split | — | — | — | — | Split into 3 below |
| → `PlaylistsPane` | `PanePlaylists` | "Playlists" | `3` | filter, new, rename, delete | Extract from LibraryTree; Enter=track sub-view, Shift+arrow=reorder |
| → `AlbumsPane` | `PaneAlbums` | "Albums" | `4` | filter | Extract from LibraryTree |
| → `LikedSongsPane` | `PaneLikedSongs` | "Liked Songs" | `5` | filter, sort, like | Extract from LibraryTree |
| `StatsView` → split | — | — | — | — | Split into 3 below |
| → `RecentlyPlayedPane` | `PaneRecentlyPlayed` | "Recently Played" | `6` | filter | RecentlyPlayed section extracted |
| → `TopTracksPane` | `PaneTopTracks` | "Top Tracks" | `7` | filter, 4wk/6mo/all | Top tracks extracted |
| → `TopArtistsPane` | `PaneTopArtists` | "Top Artists" | `8` | filter, 4wk/6mo/all | Top artists extracted |
| `PlaylistManager` | — | — | — | — | Merge into PlaylistsPane |
| — (new) | `PaneInternals` | "Internals" | — | — | Page B only, full-screen |

### Code Migration Notes

- **`cmd/root.go`**: Update minimum terminal size check from `100x24` to `120x30`
- **`internal/app/app.go`**: Replace individual pane fields with `layout *Manager` + `panes map[PaneID]Pane`; remove `viewMode` and `focusedPane` enums
- **`internal/app/render.go`**: Replace `buildView()` with `renderGrid()`; remove `renderPaneWithBorder()`
- **`internal/app/routing.go`**: Replace hardcoded 3-pane rotation with `layout.RotateFocus()`

### What Gets Deleted

- `viewMode` enum values `viewStats`, `viewPlaylists` — replaced by page system + presets. `viewSplash` and `viewAuth` remain as special cases (splash and auth screens render full-screen without the grid, transitional only)
- `focusedPane` enum — replaced by `layout.Manager.FocusedPane()`
- `renderPaneWithBorder()` — replaced by `layout.RenderPaneBorder()`
- `libraryPane` tree model — split across 3 independent panes
- Context-sensitive status bar hints — hints move to pane borders
- Key `3` for playlist manager — now pane toggle for Playlists
- Key `2` for stats view — now pane toggle for Queue

---

## 24. Box Drawing Reference

**Unchanged from DESIGN.md:** Rounded corners exclusively.

```
╭─────────────╮   Used for all pane borders and overlays
│             │
╰─────────────╯
```

`─` for horizontal fills, `│` for vertical borders. Never `┌┐└┘`.

---

## 25. Color System Rules

**Unchanged from DESIGN.md:**
- All color values come from `internal/ui/theme/`
- Never hardcode hex values in component code
- Always reference tokens through the `Theme` interface
- New components use new tokens (section 18)

---

## 26. Accessibility

- All state changes visible via color AND text/symbol — never color alone
- Per-pane border colors are supplemented by pane titles (text identification)
- Filter state shown in border text, not just color
- Scroll indicators use text (`▲`/`▼`), not just position
- `?` help always available

---

*Status: DRAFT — supersedes DESIGN.md for new layout work*
*Created: 2026-03-25*
